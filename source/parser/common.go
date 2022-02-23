package parser

import (
	"strings"
	"time"
)

func mapMonthName(name string) (time.Month, bool) {
	// @formatter:off
	cleanName := strings.ToLower(strings.TrimSpace(name))
	switch cleanName {
	case "январь":   return time.January,   true
	case "февраль":  return time.February,  true
	case "март":     return time.March,     true
	case "апрель":   return time.April,     true
	case "май":      return time.May,       true
	case "июнь":     return time.June,      true
	case "июль":     return time.July,      true
	case "август":   return time.August,    true
	case "сентябрь": return time.September, true
	case "октябрь":  return time.October,   true
	case "ноябрь":   return time.November,  true
	case "декабрь":  return time.December,  true
	default:         return 0,              false
	}
	// @formatter:on
}

// WeekDayByIndex использует порядок, принятый в России (неделя начинается с понедельника).
func mapWeekday(i int) (time.Weekday, bool) {
	// @formatter:off
	switch i {
	case 0:  return time.Monday,    true
	case 1:  return time.Tuesday,   true
	case 2:  return time.Wednesday, true
	case 3:  return time.Thursday,  true
	case 4:  return time.Friday,    true
	case 5:  return time.Saturday,  true
	case 6:  return time.Sunday,    true
	default: return -1,             false
	}
	// @formatter:on
}

func daysInMonth(y int, m time.Month) int {
	// day=0 нормализуется: будет выбран последний день месяца.
	lastDay := time.Date(y, nextMonth(m), 0, 0, 0, 0, 0, time.UTC)
	return lastDay.Day()
}

func nextMonth(m time.Month) time.Month {
	next := m + 1
	if next > time.December {
		next = time.January
	}
	return next
}

func weekdayOf(y int, m time.Month, d int) time.Weekday {
	date := time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
	return date.Weekday()
}