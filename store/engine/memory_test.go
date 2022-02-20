package engine

import (
	"github.com/nvkalinin/business-calendar/store"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMemory_FindDay(t *testing.T) {
	mem := Memory{store: map[int]store.Months{
		2022: {
			1: {
				2: {Working: false, Type: store.Holiday},
			},
		},
	}}

	// Нормальный сценарий.
	day, ok := mem.FindDay(2022, 1, 2)
	assert.Equal(t, &store.Day{Working: false, Type: store.Holiday}, day)
	assert.True(t, ok)

	// Изменение полей day не должно влиять на mem.store.
	day.Working = true
	assert.False(t, mem.store[2022][1][2].Working)

	// Когда нет искомой даты.
	day, ok = mem.FindDay(2022, 1, 3)
	assert.Nil(t, day)
	assert.False(t, ok)

	day, ok = mem.FindDay(2022, 2, 3)
	assert.Nil(t, day)
	assert.False(t, ok)

	day, ok = mem.FindDay(2023, 2, 3)
	assert.Nil(t, day)
	assert.False(t, ok)
}

func TestMemory_FindMonth(t *testing.T) {
	mem := Memory{store: map[int]store.Months{
		2022: {
			1: {
				2: {Working: false, Type: store.Holiday},
			},
		},
	}}

	// Нормальный сценарий.
	mon, ok := mem.FindMonth(2022, 1)
	expMon := store.Days{
		2: {Working: false, Type: store.Holiday},
	}
	assert.Equal(t, expMon, mon)
	assert.True(t, ok)

	// Изменение полей mon не должно влиять на mem.store.
	mon[2] = store.Day{Working: true}
	assert.False(t, mem.store[2022][1][2].Working)

	// Когда нет искомого месяца.
	mon, ok = mem.FindMonth(2022, 2)
	assert.Nil(t, mon)
	assert.False(t, ok)

	mon, ok = mem.FindMonth(2023, 2)
	assert.Nil(t, mon)
	assert.False(t, ok)
}

func TestMemory_FindYear(t *testing.T) {
	mem := Memory{store: map[int]store.Months{
		2022: {
			1: {
				2: {Working: false, Type: store.Holiday},
			},
		},
	}}

	// Нормальный сценарий.
	year, ok := mem.FindYear(2022)
	expYear := store.Months{
		1: {
			2: {Working: false, Type: store.Holiday},
		},
	}
	assert.Equal(t, expYear, year)
	assert.True(t, ok)

	// Изменение полей mon не должно влиять на mem.store.
	year[1][2] = store.Day{Working: true}
	assert.False(t, mem.store[2022][1][2].Working)

	// Когда нет искомого года.
	year, ok = mem.FindYear(2023)
	assert.Nil(t, year)
	assert.False(t, ok)
}

func TestMemory_PutYear(t *testing.T) {
	mem := NewMemory()

	yearToSave := store.Months{
		1: {
			2: {Working: false, Type: store.Holiday},
		},
	}
	err := mem.PutYear(2022, yearToSave)
	assert.NoError(t, err)

	expMem := &Memory{store: map[int]store.Months{
		2022: {
			1: {
				2: {Working: false, Type: store.Holiday},
			},
		},
	}}
	assert.Equal(t, expMem, mem)

	// Изменение аргумента для PutYear не должно влиять на mem.store.
	yearToSave[1][2] = store.Day{Working: true}
	assert.False(t, mem.store[2022][1][2].Working)
}
