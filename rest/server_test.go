package rest

import (
	"github.com/nvkalinin/business-calendar/store"
	"github.com/nvkalinin/business-calendar/store/engine"
	"github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

var testOpts = Opts{
	LogRequests: false,
	RateLimiter: true,
	ReqLimit:    100,
	LimitWindow: 1 * time.Second,
}

var testStore = engine.NewMemory()

func init() {
	testStore.PutYear(2022, store.Months{
		time.January: {
			1:  store.Day{WeekDay: store.Saturday, Working: false, Type: store.Holiday},
			2:  store.Day{WeekDay: store.Sunday, Working: false, Type: store.Holiday},
			10: store.Day{WeekDay: store.Monday, Working: true, Type: store.Normal},
		},
	})
}

func TestServer_Year(t *testing.T) {
	rest := &Server{Store: testStore, Opts: testOpts}
	srv := httptest.NewServer(rest.routes())
	defer srv.Close()

	// Нормальный случай.
	resp, err := http.Get(srv.URL + "/api/cal/2022")
	assert.NoError(t, err)
	defer resp.Body.Close()
	respJson, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)

	expJson := `{
		"1": {
			"1":  {"weekDay": "sat", "working": false, "type": "holiday"},
			"2":  {"weekDay": "sun", "working": false, "type": "holiday"},
			"10": {"weekDay": "mon", "working": true,  "type": "normal"}
		}
	}`
	assert.JSONEq(t, expJson, string(respJson))

	// Год не найден.
	resp, err = http.Get(srv.URL + "/api/cal/2023")
	assert.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 404, resp.StatusCode)
}

func TestServer_Month(t *testing.T) {
	rest := &Server{Store: testStore, Opts: testOpts}
	srv := httptest.NewServer(rest.routes())
	defer srv.Close()

	// Нормальный случай.
	resp, err := http.Get(srv.URL + "/api/cal/2022/1")
	assert.NoError(t, err)
	defer resp.Body.Close()
	respJson, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)

	expJson := `{
		"1":  {"weekDay": "sat", "working": false, "type": "holiday"},
		"2":  {"weekDay": "sun", "working": false, "type": "holiday"},
		"10": {"weekDay": "mon", "working": true,  "type": "normal"}
	}`
	assert.JSONEq(t, expJson, string(respJson))

	// Месяц не найден.
	resp, err = http.Get(srv.URL + "/api/cal/2022/2")
	assert.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 404, resp.StatusCode)
}

func TestServer_Day(t *testing.T) {
	rest := &Server{Store: testStore, Opts: testOpts}
	srv := httptest.NewServer(rest.routes())
	defer srv.Close()

	// Нормальный случай.
	resp, err := http.Get(srv.URL + "/api/cal/2022/1/10")
	assert.NoError(t, err)
	defer resp.Body.Close()
	respJson, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)

	expJson := `{
		"weekDay": "mon",
		"working": true,
		"type":    "normal"
	}`
	assert.JSONEq(t, expJson, string(respJson))

	// Месяц не найден.
	resp, err = http.Get(srv.URL + "/api/cal/2022/1/31")
	assert.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 404, resp.StatusCode)
}
