package parser

import (
	"github.com/nvkalinin/business-calendar/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestConsultant_ParseYear(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/law/ref/calendar/proizvodstvennye/2021/", func(w http.ResponseWriter, r *http.Request) {
		html, err := ioutil.ReadFile("testdata/consultant_2021.html")
		require.NoError(t, err)

		w.Write(html)
	})

	s := httptest.NewServer(mux)
	defer s.Close()

	consultant := &Consultant{
		Client:  s.Client(),
		baseURL: s.URL,
	}

	year, err := consultant.GetYear(2021)
	assert.NoError(t, err)

	assert.Len(t, year, 12)
	assert.Len(t, year[1], 31)
	assert.Len(t, year[2], 28)
	assert.Len(t, year[3], 31)
	assert.Len(t, year[4], 30)
	assert.Len(t, year[5], 31)
	assert.Len(t, year[6], 30)
	assert.Len(t, year[7], 31)
	assert.Len(t, year[8], 31)
	assert.Len(t, year[9], 30)
	assert.Len(t, year[10], 31)
	assert.Len(t, year[11], 30)
	assert.Len(t, year[12], 31)

	// @formatter:off
	expMay := store.Days{
		1: {WeekDay: store.Saturday, Working: false, Type: store.Weekend},
		2: {WeekDay: store.Sunday,   Working: false, Type: store.Weekend},

		3: {WeekDay: store.Monday,    Working: false, Type: store.Holiday},
		4: {WeekDay: store.Tuesday,   Working: true,  Type: store.NonWorking},
		5: {WeekDay: store.Wednesday, Working: true,  Type: store.NonWorking},
		6: {WeekDay: store.Thursday,  Working: true,  Type: store.NonWorking},
		7: {WeekDay: store.Friday,    Working: true,  Type: store.NonWorking},
		8: {WeekDay: store.Saturday,  Working: false, Type: store.Weekend},
		9: {WeekDay: store.Sunday,    Working: false, Type: store.Weekend},

		10: {WeekDay: store.Monday,    Working: false, Type: store.Holiday},
		11: {WeekDay: store.Tuesday,   Working: true,  Type: store.Normal},
		12: {WeekDay: store.Wednesday, Working: true,  Type: store.Normal},
		13: {WeekDay: store.Thursday,  Working: true,  Type: store.Normal},
		14: {WeekDay: store.Friday,    Working: true,  Type: store.Normal},
		15: {WeekDay: store.Saturday,  Working: false, Type: store.Weekend},
		16: {WeekDay: store.Sunday,    Working: false, Type: store.Weekend},

		17: {WeekDay: store.Monday,    Working: true,  Type: store.Normal},
		18: {WeekDay: store.Tuesday,   Working: true,  Type: store.Normal},
		19: {WeekDay: store.Wednesday, Working: true,  Type: store.Normal},
		20: {WeekDay: store.Thursday,  Working: true,  Type: store.Normal},
		21: {WeekDay: store.Friday,    Working: true,  Type: store.Normal},
		22: {WeekDay: store.Saturday,  Working: false, Type: store.Weekend},
		23: {WeekDay: store.Sunday,    Working: false, Type: store.Weekend},

		24: {WeekDay: store.Monday,    Working: true,  Type: store.Normal},
		25: {WeekDay: store.Tuesday,   Working: true,  Type: store.Normal},
		26: {WeekDay: store.Wednesday, Working: true,  Type: store.Normal},
		27: {WeekDay: store.Thursday,  Working: true,  Type: store.Normal},
		28: {WeekDay: store.Friday,    Working: true,  Type: store.Normal},
		29: {WeekDay: store.Saturday,  Working: false, Type: store.Weekend},
		30: {WeekDay: store.Sunday,    Working: false, Type: store.Weekend},

		31: {WeekDay: store.Monday, Working: true, Type: store.Normal},
	}
	// @formatter:on
	assert.Equal(t, expMay, year[time.May])
}
