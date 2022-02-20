package engine

import (
	"github.com/nvkalinin/business-calendar/store"
	"time"
)

type Memory struct {
	store map[int]store.Months
}

func NewMemory() *Memory {
	return &Memory{
		store: make(map[int]store.Months, 3),
	}
}

func (m *Memory) FindDay(y int, mon time.Month, d int) (*store.Day, bool) {
	month, ok := m.FindMonth(y, mon)
	if !ok {
		return nil, false
	}

	day, ok := month[d]
	if !ok {
		return nil, false
	}

	return &day, true
}

func (m *Memory) FindMonth(y int, mon time.Month) (store.Days, bool) {
	year, ok := m.FindYear(y)
	if !ok {
		return nil, false
	}

	month, ok := year[mon]
	if !ok {
		return nil, false
	}

	return month, true
}

func (m *Memory) FindYear(y int) (store.Months, bool) {
	year, ok := m.store[y]
	if !ok {
		return nil, false
	}

	return year.Copy(), true
}

func (m *Memory) PutYear(y int, data store.Months) error {
	m.store[y] = data.Copy()
	return nil
}
