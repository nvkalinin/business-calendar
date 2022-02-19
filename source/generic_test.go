package source

import (
	"encoding/csv"
	"github.com/nvkalinin/business-calendar/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"strings"
	"testing"
)

func TestGeneric_GetYear(t *testing.T) {
	generic := NewGeneric()
	year := generic.GetYear(2022)
	expYear := yearFromCsv(t, "testdata/generic_2022.csv")
	assert.Equal(t, expYear, year)
}

func yearFromCsv(t *testing.T, filename string) store.Year {
	f, err := os.Open(filename)
	require.NoError(t, err)

	r := csv.NewReader(f)
	r.Comma = '|'
	r.Comment = '#'

	rows, err := r.ReadAll()
	require.NoError(t, err)

	cal := make(store.Year, 12)
	for rowNum, row := range rows[1:] {
		mon := rowNum + 1

		cal[mon] = make(store.Month, 31)
		for colNum, rawVal := range row[1:] {
			day := colNum + 1

			dayData, exists := dayFromCsv(rawVal)
			if !exists {
				continue
			}

			cal[mon][day] = dayData
		}
	}

	return cal
}

func dayFromCsv(rawVal string) (store.Day, bool) {
	val := strings.TrimSpace(rawVal)
	if val == "-" {
		// Дня нет в месяце
		return store.Day{}, false
	}

	dayType := store.Normal
	if val == "X" {
		dayType = store.Weekend
	}

	return store.Day{
		Working: dayType == store.Normal,
		Type:    dayType,
	}, true
}
