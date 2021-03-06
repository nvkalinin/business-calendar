package rest

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"
	"github.com/nvkalinin/business-calendar/log"
	"github.com/nvkalinin/business-calendar/store"
	"github.com/nvkalinin/business-calendar/store/engine"
)

type Store interface {
	FindDay(y int, mon time.Month, d int) (*store.Day, bool)
	FindMonth(y int, mon time.Month) (store.Days, bool)
	FindYear(y int) (store.Months, bool)
}

type Updater interface {
	UpdateCalendar(y int) error
}

type Server struct {
	Store   Store
	Updater Updater
	Opts    Opts
	srv     *http.Server
}

type Opts struct {
	Listen      string
	LogRequests bool
	AdminPasswd string

	ReadTimeout       time.Duration
	ReadHeaderTimeout time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration

	RateLimiter bool
	ReqLimit    int
	LimitWindow time.Duration
}

func (s *Server) Run() error {
	r := s.routes()

	s.srv = &http.Server{
		Addr:              s.Opts.Listen,
		Handler:           r,
		ReadTimeout:       s.Opts.ReadTimeout,
		ReadHeaderTimeout: s.Opts.ReadHeaderTimeout,
		WriteTimeout:      s.Opts.WriteTimeout,
		IdleTimeout:       s.Opts.IdleTimeout,
	}

	log.Printf("[INFO] starting web server at %s", s.Opts.Listen)
	if err := s.srv.ListenAndServe(); err != http.ErrServerClosed {
		return fmt.Errorf("cannot run rest server: %w", err)
	}
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	if err := s.srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("cannot shutdown rest server: %w", err)
	}
	return nil
}

func (s *Server) routes() *chi.Mux {
	r := chi.NewRouter()

	if s.Opts.LogRequests {
		r.Use(middleware.Logger)
	}
	r.Use(middleware.Recoverer)

	r.Get("/ping", pingCtrl)
	r.Route("/api", func(r chi.Router) {
		if s.Opts.RateLimiter {
			r.Use(httprate.LimitByIP(s.Opts.ReqLimit, s.Opts.LimitWindow))
		}

		r.Get("/cal/{y}", s.yearCtrl)
		r.Get("/cal/{y}/{m}", s.monthCtrl)
		r.Get("/cal/{y}/{m}/{d}", s.dayCtrl)

		r.Route("/admin", func(r chi.Router) {
			r.Use(middleware.BasicAuth("business-calendar", map[string]string{"admin": s.Opts.AdminPasswd}))
			r.Use(middleware.NoCache)

			r.Get("/backup", s.backupCtrl)
			r.Post("/sync", s.syncCtrl)
		})
	})

	return r
}

func (s *Server) yearCtrl(w http.ResponseWriter, r *http.Request) {
	y, err := yearParam(r)
	if err != nil {
		sendErrorJson(w, 400, "invalid year")
		return
	}

	year, found := s.Store.FindYear(y)
	if !found {
		sendErrorJson(w, 404, "year not found")
		return
	}

	sendJsonResponse(w, year)
}

func (s *Server) monthCtrl(w http.ResponseWriter, r *http.Request) {
	y, err1 := yearParam(r)
	m, err2 := monthParam(r)
	err := combineErrors(err1, err2)
	if err != nil {
		sendErrorJson(w, 400, "invalid date")
		return
	}

	month, found := s.Store.FindMonth(y, m)
	if !found {
		sendErrorJson(w, 404, "month not found")
		return
	}

	sendJsonResponse(w, month)
}

func (s *Server) dayCtrl(w http.ResponseWriter, r *http.Request) {
	y, err1 := yearParam(r)
	m, err2 := monthParam(r)
	d, err3 := dayParam(r)
	err := combineErrors(err1, err2, err3)
	if err != nil {
		sendErrorJson(w, 400, "invalid date")
		return
	}

	day, found := s.Store.FindDay(y, m, d)
	if !found {
		sendErrorJson(w, 404, "date not found")
		return
	}

	sendJsonResponse(w, day)
}

func pingCtrl(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
}

func intParam(r *http.Request, param string) (int, error) {
	strVal := chi.URLParam(r, param)
	return strconv.Atoi(strVal)
}

func yearParam(r *http.Request) (int, error) {
	y, err := intParam(r, "y")
	if err != nil {
		return 0, err
	}

	if y <= 0 {
		return 0, fmt.Errorf("invalid year")
	}
	return y, nil
}

func monthParam(r *http.Request) (time.Month, error) {
	m, err := intParam(r, "m")
	if err != nil {
		return 0, err
	}

	if m < int(time.January) || m > int(time.December) {
		return 0, fmt.Errorf("invalid month number")
	}
	return time.Month(m), nil
}

func dayParam(r *http.Request) (int, error) {
	d, err := intParam(r, "d")
	if err != nil {
		return 0, err
	}

	if d < 1 || d > 31 {
		return 0, fmt.Errorf("invalid day number")
	}
	return d, nil
}

func (s *Server) backupCtrl(w http.ResponseWriter, r *http.Request) {
	// ???????????????????????????? ???????????? ?????????????????? ?????????????????????? bolt.
	// ?????? ?????????????????? ???????????? ???????????????????????? ???????????????? ?????????? ?????????????? ??????????????/???????????? ?????????? ?????????????????? ????????????.
	// ???????????????????????? ?????????????????????????? ?? ???????? ??????.

	boltStore, isBolt := s.Store.(*engine.Bolt)
	if !isBolt {
		sendErrorJson(w, 500, "only bolt supports backup")
		return
	}

	fileName := fmt.Sprintf("cal_%s.bolt.gz", time.Now().Format("2006-01-02"))

	w.Header().Set("Content-Type", "application/gzip")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, fileName))
	w.WriteHeader(200)

	gzw := gzip.NewWriter(w)
	defer func() {
		if err := gzw.Close(); err != nil {
			log.Printf("[WARN] cannot close gzip writer: %v", err)
		}
	}()

	if err := boltStore.Backup(gzw); err != nil {
		log.Printf("[WARN] cannot make backup: %v", err)
	}
}

func (s *Server) syncCtrl(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		sendErrorJson(w, 400, "cannot parse request")
		return
	}

	yStr, ok := r.Form["y"]
	if !ok {
		sendErrorJson(w, 400, "'y' param is required")
		return
	}
	log.Printf("[DEBUG] requested years to sync: %v", yStr)

	years := make([]int, len(yStr))
	for i, v := range yStr {
		y, err := strconv.Atoi(v)
		if err != nil {
			sendErrorJson(w, 400, fmt.Sprintf("invalid year '%s': %v", v, err))
			return
		}
		years[i] = y
	}
	log.Printf("[DEBUG] requested years to sync (after parsing): %v", years)

	res := make(map[int]string)
	for _, y := range years {
		log.Printf("[INFO] syncing year %d...", y)
		err := s.Updater.UpdateCalendar(y)
		if err != nil {
			res[y] = fmt.Sprintf("error: %v", err)
		} else {
			res[y] = "ok"
		}
	}
	log.Printf("[DEBUG] sync result: %+v", res)

	sendJsonResponse(w, res)
}

func combineErrors(err ...error) error {
	nonNil := make([]error, 0, len(err))
	for _, e := range err {
		if e != nil {
			nonNil = append(nonNil, e)
		}
	}

	if len(nonNil) == 0 {
		return nil
	}
	return fmt.Errorf("%+v", nonNil)
}

func sendJsonResponse(w http.ResponseWriter, data interface{}) {
	respJson, err := json.Marshal(data)
	if err != nil {
		log.Printf("[WARN] cannot marshal response data: %+v", err)
		sendErrorJson(w, 500, "cannot marshal response data")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)

	if _, err = w.Write(respJson); err != nil {
		log.Printf("[WARN] cannot write response data: %+v", err)
	}
}

func sendErrorJson(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	restErr := &struct {
		Msg string `json:"msg"`
	}{msg}

	errJson, err := json.Marshal(restErr)
	if err != nil {
		log.Printf("[WARN] cannot marshal rest error: %+v", err)
		return
	}

	if _, err = w.Write(errJson); err != nil {
		log.Printf("[WARN] cannot write rest error: %+v", err)
	}
}
