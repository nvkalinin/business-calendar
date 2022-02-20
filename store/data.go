package store

import "time"

type DayType string

const (
	Normal     DayType = "normal"     // Обычный рабочий день.
	Weekend    DayType = "weekend"    // Выходной.
	PreHoliday DayType = "preHoliday" // Предпраздничный рабочий день.
	Holiday    DayType = "holiday"    // Праздничный, не рабочий.
	NonWorking DayType = "noWork"     // "Нерабочий" рабочий день :-).
)

type Day struct {
	Working bool    `json:"working"`
	Type    DayType `json:"type"`
	Desc    string  `json:"desc,omitempty"`
}

type Days map[int]Day

type Months map[time.Month]Days

func (m Days) Copy() Days {
	mCopy := make(Days, len(m))
	for dayNum, day := range m {
		mCopy[dayNum] = day
	}
	return mCopy
}

func (y Months) Copy() Months {
	yCopy := make(Months, len(y))
	for monNum, month := range y {
		yCopy[monNum] = month.Copy()
	}
	return yCopy
}
