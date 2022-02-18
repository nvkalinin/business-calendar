package engine

import "github.com/nvkalinin/business-calendar/store"

type Memory struct {
	store map[int]store.Year
}

func NewMemory() *Memory {
	return &Memory{
		store: make(map[int]store.Year, 3),
	}
}

func (m *Memory) FindDay(y, mon, d int) (*store.Day, bool) {
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

func (m *Memory) FindMonth(y, mon int) (store.Month, bool) {
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

func (m *Memory) FindYear(y int) (store.Year, bool) {
	year, ok := m.store[y]
	if !ok {
		return nil, false
	}

	return year.Copy(), true
}

func (m *Memory) PutYear(y int, data store.Year) error {
	m.store[y] = data.Copy()
	return nil
}
