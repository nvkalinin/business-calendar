package cmd

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/nvkalinin/business-calendar/calendar"
	"github.com/nvkalinin/business-calendar/rest"
	"github.com/nvkalinin/business-calendar/source"
	"github.com/nvkalinin/business-calendar/source/parser"
	"github.com/nvkalinin/business-calendar/store"
	"github.com/nvkalinin/business-calendar/store/engine"
	"golang.org/x/sync/errgroup"
)

type EngineType string

var (
	EngineMemory EngineType = "memory"
)

type ParserType string

var (
	ParserNone       ParserType = "none"
	ParserConsultant ParserType = "consultant"
)

type ServerCmd struct {
	Debug       bool     `short:"d" long:"debug" env:"DEBUG" description:"Включить вывод отладочных сообщений в лог."`
	SyncAt      string   `long:"sync-at" env:"SYNC_AT" value-name:"hh:mm[:ss]" description:"В какое время синхронизировать производственный календарь со всеми источниками. Обновление происходит один раз в сутки. Если не указано, то автоматическое обновление отключено."`
	SyncOnStart []string `long:"sync-on-start" env:"SYNC_ON_START" value-name:"year" default:"current" default:"next" description:"За какие годы синхронизировать календарь при запуске программы. Можно указывать числа, 'current' — текущий год, 'next' — следущий год."`

	Web struct {
		Listen      string `long:"listen" env:"LISTEN" value-name:"addr" default:"0.0.0.0" description:"Сетевой адрес для веб-сервера."`
		AccessLog   bool   `long:"access-log" env:"ACCESS_LOG" description:"Логировать все HTTP-запросы."`
		AdminPasswd string `long:"admin-passwd" env:"ADMIN_PASSWD" description:"Пароль пользователя admin для вызова /api/admin/*."`

		ReadTimeout       time.Duration `long:"read-timeout" env:"READ_TIMEOUT" value-name:"duration" default:"5s"`
		ReadHeaderTimeout time.Duration `long:"read-header-timeout" env:"READ_HEADER_TIMEOUT" value-name:"duration" default:"5s"`
		WriteTimeout      time.Duration `long:"write-timeout" env:"WRITE_TIMEOUT" value-name:"duration" default:"5s"`
		IdleTimeout       time.Duration `long:"idle-timeout" env:"IDLE_TIMEOUT" value-name:"duration" default:"30s"`

		RateLimiter struct {
			ReqLimit    int           `long:"reqs" env:"REQS" value-name:"num" default:"100" description:"Количество запросов с одного IP."`
			LimitWindow time.Duration `long:"window" env:"WINDOW" value-name:"duration" default:"1s" description:"Интервал времени, за который разврешено указанное кол-во запросов."`
		} `group:"Rate Limiter" namespace:"ratelim" env-namespace:"RATE_LIM"`
	} `group:"Web" namespace:"web" env-namespace:"WEB"`

	Store struct {
		// TODO: поменять потом default на bolt.
		Engine EngineType `long:"engine" env:"ENGINE" value-name:"type" choice:"memory" default:"memory" description:"Тип хранилища для данных, собранных пармерами."`
	} `group:"Хранилище" namespace:"store" env-namespace:"STORE"`

	Source struct {
		Parser ParserType `long:"parser" env:"PARSER" value-name:"type" choice:"consultant" choice:"none" default:"consultant" description:"Внешний источник производственного календаря, который нужно парсить."`

		Consultant struct {
			Timeout   time.Duration `long:"timeout" env:"TIMEOUT" value-name:"duration" default:"30s" description:"Максимальное время выполнения запроса к сайту."`
			UserAgent string        `long:"user-agent" env:"USER_AGENT" description:"Значение заголовка User-Agent во всех запросах к сайту."`
		} `group:"Парсер consultant.ru" namespace:"consultant" env-namespace:"CONSULTANT"`

		Override string `long:"override" env:"OVERRIDE" value-name:"file.yml" description:"Путь к файлу с локальными изменениями производственного календаря. Если задан, используется всегда, вне зависимости от выбранного парсера."`
	} `group:"Источник данных" namespace:"source" env-namespace:"SOURCE"`
}

func (s *ServerCmd) Execute(args []string) error {
	// TODO s.Debug

	a, err := s.makeApp()
	if err != nil {
		return err
	}

	go a.run()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	a.shutdown()

	return nil
}

type app struct {
	srv             *rest.Server
	proc            *calendar.Processor
	autoSync        bool
	syncYears       []int
	syncYearsFinish chan struct{}
	stopped         bool
}

func (s *ServerCmd) makeApp() (*app, error) {
	a := &app{
		syncYearsFinish: make(chan struct{}),
	}

	store, err := s.makeStore()
	if err != nil {
		return nil, err
	}

	src, err := s.makeSources()
	if err != nil {
		return nil, err
	}

	var syncAt time.Time
	if s.SyncAt != "" {
		syncAt, err = parseSyncAt(s.SyncAt)
		if err != nil {
			return nil, fmt.Errorf("sync at: %w", err)
		}
		a.autoSync = true
	}

	syncYears, err := parseYears(s.SyncOnStart)
	if err != nil {
		return nil, fmt.Errorf("sync on start: %w", err)
	}
	a.syncYears = syncYears

	a.proc = calendar.NewProcessor(calendar.ProcOpts{
		Src:      src,
		Store:    calendar.Store(store),
		UpdateAt: syncAt,
	})

	a.srv = &rest.Server{
		Store: store,
		Opts: rest.Opts{
			Listen:      s.Web.Listen,
			LogRequests: s.Web.AccessLog,
			AdminPasswd: s.Web.AdminPasswd,

			ReadTimeout:       s.Web.ReadTimeout,
			ReadHeaderTimeout: s.Web.ReadHeaderTimeout,
			WriteTimeout:      s.Web.WriteTimeout,
			IdleTimeout:       s.Web.IdleTimeout,

			RateLimiter: s.Web.RateLimiter.ReqLimit > 0,
			ReqLimit:    s.Web.RateLimiter.ReqLimit,
			LimitWindow: s.Web.RateLimiter.LimitWindow,
		},
	}

	return a, nil
}

type Store interface {
	FindDay(y int, mon time.Month, d int) (*store.Day, bool)
	FindMonth(y int, mon time.Month) (store.Days, bool)
	FindYear(y int) (store.Months, bool)
	PutYear(y int, data store.Months) error
}

func (s *ServerCmd) makeStore() (Store, error) {
	switch s.Store.Engine {
	case EngineMemory:
		return engine.NewMemory(), nil
	default:
		return nil, fmt.Errorf("unknown store engine %s", s.Store.Engine)
	}
}

func (s *ServerCmd) makeSources() ([]calendar.Source, error) {
	src := make([]calendar.Source, 0, 3)
	src = append(src, source.NewGeneric())

	switch s.Source.Parser {
	case ParserNone:
	case ParserConsultant:
		ua := s.Source.Consultant.UserAgent
		if ua == "" {
			ua = "Go-http-client"
		}

		p := &parser.Consultant{
			Client: &http.Client{
				Timeout: s.Source.Consultant.Timeout,
			},
			UserAgent: ua,
		}

		src = append(src, p)
	default:
		return nil, fmt.Errorf("unknown parser %s", s.Source.Parser)
	}

	// TODO override

	return src, nil
}

func parseSyncAt(val string) (time.Time, error) {
	if t, err := time.Parse("15:04", val); err == nil {
		return t, nil
	}

	t, err := time.Parse("15:04:05", val)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid time '%s', it must match pattern hh:mm[:ss]", val)
	}
	return t, nil
}

func parseYears(vals []string) ([]int, error) {
	years := make(map[int]bool, len(vals))
	for _, val := range vals {
		switch val {
		case "current":
			y := time.Now().Year()
			years[y] = true
		case "next":
			y := time.Now().Year() + 1
			years[y] = true
		default:
			y, err := strconv.Atoi(val)
			if err != nil {
				return nil, fmt.Errorf("invalid year '%s': %w", val, err)
			}
			if y < 0 {
				return nil, fmt.Errorf("invalid year %d", y)
			}
			years[y] = true
		}
	}

	ylist := make([]int, 0, len(years))
	for y := range years {
		ylist = append(ylist, y)
	}

	return ylist, nil
}

func (a *app) run() {
	g, ctx := errgroup.WithContext(context.Background())

	if a.autoSync {
		g.Go(func() error {
			a.proc.RunUpdates()
			return nil
		})
	}

	g.Go(func() error {
		syncOnRun(a.proc, a.syncYears, a.syncYearsFinish)
		return nil
	})

	g.Go(a.srv.Run)

	<-ctx.Done()
	err := ctx.Err()
	if err != nil && err != http.ErrServerClosed {
		a.shutdown()
	}
}

func (a *app) shutdown() {
	if a.stopped {
		return
	}

	log.Printf("[INFO] shutting down...")

	ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)
	g, _ := errgroup.WithContext(ctx)

	if a.autoSync {
		g.Go(func() error {
			return a.proc.Shutdown(ctx)
		})
	}
	g.Go(func() error {
		return a.srv.Shutdown(ctx)
	})
	g.Go(func() error {
		select {
		case <-ctx.Done():
			return fmt.Errorf("sync on run: %w", ctx.Err())
		case <-a.syncYearsFinish:
			return nil
		}
	})

	if err := g.Wait(); err != nil {
		log.Printf("[ERROR] app shutdown: %v", err)
	}
	a.stopped = true
}

func syncOnRun(proc *calendar.Processor, years []int, finished chan<- struct{}) {
	for _, y := range years {
		if err := proc.UpdateCalendar(y); err != nil {
			log.Printf("[WARN] sync on run, year %d: %+v", y, err)
		}
	}
	close(finished)
}
