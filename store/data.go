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

type WeekDay string

const (
	Monday    WeekDay = "mon"
	Tuesday   WeekDay = "tue"
	Wednesday WeekDay = "wed"
	Thursday  WeekDay = "thu"
	Friday    WeekDay = "fri"
	Saturday  WeekDay = "sat"
	Sunday    WeekDay = "sun"
)

func NewWeekDay(wd time.Weekday) (WeekDay, bool) {
	// @formatter:off
	switch wd {
	case time.Monday:    return Monday,    true
	case time.Tuesday:   return Tuesday,   true
	case time.Wednesday: return Wednesday, true
	case time.Thursday:  return Thursday,  true
	case time.Friday:    return Friday,    true
	case time.Saturday:  return Saturday,  true
	case time.Sunday:    return Sunday,    true
	default:             return "",        false
	}
	// @formatter:on
}

type Day struct {
	WeekDay WeekDay `json:"weekDay,omitempty"`
	Working bool    `json:"working"`
	Type    DayType `json:"type,omitempty"`
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
