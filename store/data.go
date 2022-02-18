package store

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

type Month map[int]Day

type Year map[int]Month

func (m Month) Copy() Month {
	mCopy := make(Month, len(m))
	for dayNum, day := range m {
		mCopy[dayNum] = day
	}
	return mCopy
}

func (y Year) Copy() Year {
	yCopy := make(Year, len(y))
	for monNum, month := range y {
		yCopy[monNum] = month.Copy()
	}
	return yCopy
}
