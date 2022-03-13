package rest

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"
	"github.com/nvkalinin/business-calendar/store"
	"log"
	"net"
	"net/http"
	"strconv"
	"time"
)

type Store interface {
	FindDay(y int, mon time.Month, d int) (*store.Day, bool)
	FindMonth(y int, mon time.Month) (store.Days, bool)
	FindYear(y int) (store.Months, bool)
}

type Server struct {
	Store Store
	Opts  Opts
}

type Opts struct {
	Listen      string
	LogRequests bool

	ReadTimeout       time.Duration
	ReadHeaderTimeout time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration

	RateLimiter bool
	ReqLimit    int
	LimitWindow time.Duration
}

func (s *Server) Run(ctx context.Context) error {
	r := s.routes()

	srv := &http.Server{
		Addr:              s.Opts.Listen,
		Handler:           r,
		ReadTimeout:       s.Opts.ReadTimeout,
		ReadHeaderTimeout: s.Opts.ReadHeaderTimeout,
		WriteTimeout:      s.Opts.WriteTimeout,
		IdleTimeout:       s.Opts.IdleTimeout,
		BaseContext: func(net.Listener) context.Context {
			return ctx
		},
	}
	return srv.ListenAndServe()
}

func (s *Server) routes() *chi.Mux {
	r := chi.NewRouter()
	
	if s.Opts.LogRequests {
		r.Use(middleware.Logger)
	}
	r.Use(middleware.Recoverer)

	r.Route("/api", func(r chi.Router) {
		if s.Opts.RateLimiter {
			r.Use(httprate.LimitByIP(s.Opts.ReqLimit, s.Opts.LimitWindow))
		}

		r.Get("/cal/{y}", s.yearCtrl)
		r.Get("/cal/{y}/{m}", s.monthCtrl)
		r.Get("/cal/{y}/{m}/{d}", s.dayCtrl)
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

	restErr := &struct{ msg string }{msg}

	errJson, err := json.Marshal(restErr)
	if err != nil {
		log.Printf("[WARN] cannot marshal rest error: %+v", err)
		return
	}

	if _, err = w.Write(errJson); err != nil {
		log.Printf("[WARN] cannot write rest error: %+v", err)
	}
}
