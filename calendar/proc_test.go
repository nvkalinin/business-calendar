package calendar

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/nvkalinin/business-calendar/store"
	"github.com/stretchr/testify/assert"
)

type SrcMock map[int]store.Months

func (s SrcMock) GetYear(y int) (store.Months, error) {
	months, ok := s[y]
	if !ok {
		return nil, fmt.Errorf("no such year: %d", y)
	}
	return months, nil
}

type StoreMock map[int]store.Months

func (s StoreMock) PutYear(y int, m store.Months) error {
	s[y] = m
	return nil
}

func TestProcessor_MakeCalendar(t *testing.T) {
	src1 := SrcMock{2022: {
		time.February: {
			21: {WeekDay: store.Monday, Working: true, Type: store.Normal},
			22: {WeekDay: store.Tuesday, Working: true, Type: store.Normal},
			23: {WeekDay: store.Wednesday, Working: true, Type: store.Normal},
		},
	}}
	src2 := SrcMock{2022: {
		time.February: {
			22: {WeekDay: store.Tuesday, Working: true, Type: store.PreHoliday},
			23: {Working: false, Type: store.Holiday}, // День недели должен остаться из src1.
			24: {WeekDay: store.Thursday, Working: true, Type: store.Normal},
		},
	}}

	tmpStore := StoreMock{}

	p, _ := makeProcessor(ProcOpts{
		Src:   []Source{src1, src2},
		Store: tmpStore,
	})
	err := p.UpdateCalendar(2022)
	assert.NoError(t, err)

	expStore := StoreMock{2022: {
		time.February: {
			21: {WeekDay: store.Monday, Working: true, Type: store.Normal},
			22: {WeekDay: store.Tuesday, Working: true, Type: store.PreHoliday},
			23: {WeekDay: store.Wednesday, Working: false, Type: store.Holiday},
			24: {WeekDay: store.Thursday, Working: true, Type: store.Normal},
		},
	}}
	assert.Equal(t, expStore, tmpStore)
}

func TestProcessor_DoUpdates(t *testing.T) {
	src := SrcMock{
		2022: {
			time.January: {
				1: {WeekDay: store.Saturday, Working: false, Type: store.Holiday},
			},
		},
		2023: {
			time.January: {
				1: {WeekDay: store.Sunday, Working: false, Type: store.Holiday},
			},
		},
	}
	tmpStore := StoreMock{}

	p, stop := makeProcessor(ProcOpts{
		Src:      []Source{src},
		Store:    tmpStore,
		UpdateAt: time.Now().Add(500 * time.Millisecond),
	})
	defer stop()

	go p.RunUpdates()
	assert.Len(t, tmpStore, 0)

	time.Sleep(1000 * time.Millisecond)
	assert.Len(t, tmpStore, 2)
}

func makeProcessor(opts ProcOpts) (p *Processor, stop func()) {
	p = NewProcessor(opts)
	return p, func() {
		ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
		p.Shutdown(ctx)
	}
}
