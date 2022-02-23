package source

import (
	"github.com/nvkalinin/business-calendar/store"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestGeneric_GetYear(t *testing.T) {
	generic := NewGeneric()
	year, _ := generic.GetYear(2022)

	expJanuary := store.Days{
		1: store.Day{WeekDay: "sat", Working: false, Type: store.Weekend},
		2: store.Day{WeekDay: "sun", Working: false, Type: store.Weekend},

		3: store.Day{WeekDay: "mon", Working: true, Type: store.Normal},
		4: store.Day{WeekDay: "tue", Working: true, Type: store.Normal},
		5: store.Day{WeekDay: "wed", Working: true, Type: store.Normal},
		6: store.Day{WeekDay: "thu", Working: true, Type: store.Normal},
		7: store.Day{WeekDay: "fri", Working: true, Type: store.Normal},
		8: store.Day{WeekDay: "sat", Working: false, Type: store.Weekend},
		9: store.Day{WeekDay: "sun", Working: false, Type: store.Weekend},

		10: store.Day{WeekDay: "mon", Working: true, Type: store.Normal},
		11: store.Day{WeekDay: "tue", Working: true, Type: store.Normal},
		12: store.Day{WeekDay: "wed", Working: true, Type: store.Normal},
		13: store.Day{WeekDay: "thu", Working: true, Type: store.Normal},
		14: store.Day{WeekDay: "fri", Working: true, Type: store.Normal},
		15: store.Day{WeekDay: "sat", Working: false, Type: store.Weekend},
		16: store.Day{WeekDay: "sun", Working: false, Type: store.Weekend},

		17: store.Day{WeekDay: "mon", Working: true, Type: store.Normal},
		18: store.Day{WeekDay: "tue", Working: true, Type: store.Normal},
		19: store.Day{WeekDay: "wed", Working: true, Type: store.Normal},
		20: store.Day{WeekDay: "thu", Working: true, Type: store.Normal},
		21: store.Day{WeekDay: "fri", Working: true, Type: store.Normal},
		22: store.Day{WeekDay: "sat", Working: false, Type: store.Weekend},
		23: store.Day{WeekDay: "sun", Working: false, Type: store.Weekend},

		24: store.Day{WeekDay: "mon", Working: true, Type: store.Normal},
		25: store.Day{WeekDay: "tue", Working: true, Type: store.Normal},
		26: store.Day{WeekDay: "wed", Working: true, Type: store.Normal},
		27: store.Day{WeekDay: "thu", Working: true, Type: store.Normal},
		28: store.Day{WeekDay: "fri", Working: true, Type: store.Normal},
		29: store.Day{WeekDay: "sat", Working: false, Type: store.Weekend},
		30: store.Day{WeekDay: "sun", Working: false, Type: store.Weekend},

		31: store.Day{WeekDay: "mon", Working: true, Type: store.Normal},
	}
	expDecember := store.Days{
		1: store.Day{WeekDay: "thu", Working: true, Type: store.Normal},
		2: store.Day{WeekDay: "fri", Working: true, Type: store.Normal},
		3: store.Day{WeekDay: "sat", Working: false, Type: store.Weekend},
		4: store.Day{WeekDay: "sun", Working: false, Type: store.Weekend},

		5:  store.Day{WeekDay: "mon", Working: true, Type: store.Normal},
		6:  store.Day{WeekDay: "tue", Working: true, Type: store.Normal},
		7:  store.Day{WeekDay: "wed", Working: true, Type: store.Normal},
		8:  store.Day{WeekDay: "thu", Working: true, Type: store.Normal},
		9:  store.Day{WeekDay: "fri", Working: true, Type: store.Normal},
		10: store.Day{WeekDay: "sat", Working: false, Type: store.Weekend},
		11: store.Day{WeekDay: "sun", Working: false, Type: store.Weekend},

		12: store.Day{WeekDay: "mon", Working: true, Type: store.Normal},
		13: store.Day{WeekDay: "tue", Working: true, Type: store.Normal},
		14: store.Day{WeekDay: "wed", Working: true, Type: store.Normal},
		15: store.Day{WeekDay: "thu", Working: true, Type: store.Normal},
		16: store.Day{WeekDay: "fri", Working: true, Type: store.Normal},
		17: store.Day{WeekDay: "sat", Working: false, Type: store.Weekend},
		18: store.Day{WeekDay: "sun", Working: false, Type: store.Weekend},

		19: store.Day{WeekDay: "mon", Working: true, Type: store.Normal},
		20: store.Day{WeekDay: "tue", Working: true, Type: store.Normal},
		21: store.Day{WeekDay: "wed", Working: true, Type: store.Normal},
		22: store.Day{WeekDay: "thu", Working: true, Type: store.Normal},
		23: store.Day{WeekDay: "fri", Working: true, Type: store.Normal},
		24: store.Day{WeekDay: "sat", Working: false, Type: store.Weekend},
		25: store.Day{WeekDay: "sun", Working: false, Type: store.Weekend},

		26: store.Day{WeekDay: "mon", Working: true, Type: store.Normal},
		27: store.Day{WeekDay: "tue", Working: true, Type: store.Normal},
		28: store.Day{WeekDay: "wed", Working: true, Type: store.Normal},
		29: store.Day{WeekDay: "thu", Working: true, Type: store.Normal},
		30: store.Day{WeekDay: "fri", Working: true, Type: store.Normal},
		31: store.Day{WeekDay: "sat", Working: false, Type: store.Weekend},
	}

	assert.Equal(t, 12, len(year))
	assert.Equal(t, expJanuary, year[time.January])
	assert.Equal(t, expDecember, year[time.December])
}
