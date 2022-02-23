package engine

import (
	"github.com/nvkalinin/business-calendar/store"
	"sync"
	"time"
)

type Memory struct {
	store map[int]store.Months
	mu    sync.RWMutex
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

	// month — отдельная копия, блокировка не нужна.
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

	// year — отдельная копия, блокировка не нужна.
	month, ok := year[mon]
	if !ok {
		return nil, false
	}

	return month, true
}

func (m *Memory) FindYear(y int) (store.Months, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	year, ok := m.store[y]
	if !ok {
		return nil, false
	}

	return year.Copy(), true
}

func (m *Memory) PutYear(y int, data store.Months) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.store[y] = data.Copy()
	return nil
}
