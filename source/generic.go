package source

import (
	"github.com/nvkalinin/business-calendar/store"
	"time"
)

// Generic генерирует календарь на год, в котором
// дни недели Weekend являются выходными, остальные — рабочие.
type Generic struct {
	Weekend []time.Weekday
}

func NewGeneric() *Generic {
	return &Generic{
		Weekend: []time.Weekday{time.Saturday, time.Sunday},
	}
}

func (g *Generic) GetYear(targetYear int) (store.Months, error) {
	cal := makeEmptyYear()

	date := time.Date(targetYear, time.January, 1, 0, 0, 0, 0, time.UTC)
	for date.Year() == targetYear {
		month := date.Month()
		day := date.Day()

		isWeekend := g.isWeekend(date.Weekday())
		dayType := store.Normal
		if isWeekend {
			dayType = store.Weekend
		}

		weekDay, _ := store.NewWeekDay(date.Weekday())

		cal[month][day] = store.Day{
			WeekDay: weekDay,
			Working: !isWeekend,
			Type:    dayType,
		}

		date = date.AddDate(0, 0, 1)
	}

	return cal, nil
}

func makeEmptyYear() store.Months {
	cal := make(store.Months, 12)
	for mon := 1; mon <= 12; mon++ {
		cal[time.Month(mon)] = make(store.Days, 31)
	}
	return cal
}

func (g *Generic) isWeekend(w time.Weekday) bool {
	for _, weekday := range g.Weekend {
		if w == weekday {
			return true
		}
	}
	return false
}
