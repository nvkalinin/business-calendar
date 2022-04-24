package calendar

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/nvkalinin/business-calendar/store"
)

type Source interface {
	// GetYear может вернуть не все месяцы года.
	GetYear(y int) (store.Months, error)
}

type Store interface {
	PutYear(y int, data store.Months) error
}

type ProcOpts struct {
	Src      []Source  // Упорядоченный список источников календарей.
	Store    Store     // Куда сохранять итоговый календарь (необязательно, если нужен только метод MakeCalendar).
	UpdateAt time.Time // Используется только время, остальное игнорируется.
}

type Processor struct {
	ProcOpts
	stopCh  chan struct{}
	stopped bool
}

func NewProcessor(opts ProcOpts) *Processor {
	return &Processor{
		ProcOpts: opts,
		stopCh:   make(chan struct{}),
	}
}

// DoUpdates раз в сутки (UpdateAt) обновляет календари за текущий и следующий год.
func (p *Processor) RunUpdates() {
	t := time.NewTimer(p.untilNextRun())
	for {
		select {
		case <-t.C:
			p.UpdateCurrentYears()
			t.Reset(p.untilNextRun())

		case <-p.stopCh:
			p.stopped = true
			return
		}
	}
}

func (p *Processor) Shutdown(ctx context.Context) error {
	close(p.stopCh)

	for {
		select {
		case <-time.After(10 * time.Millisecond):
			if p.stopped {
				return nil
			}
		case <-ctx.Done():
			log.Printf("[WARN] calendar.Proc shutdown timeout")
			return ctx.Err()
		}
	}
}

func (p *Processor) untilNextRun() time.Duration {
	now := time.Now()

	nextRun := time.Date(
		now.Year(), now.Month(), now.Day(),
		p.UpdateAt.Hour(), p.UpdateAt.Minute(), p.UpdateAt.Second(), p.UpdateAt.Nanosecond(),
		time.Local,
	)

	d := time.Until(nextRun)
	if d < 0 {
		d += 24 * time.Hour
	}
	return d
}

func (p *Processor) UpdateCurrentYears() {
	y := time.Now().Year()

	if err := p.UpdateCalendar(y); err != nil {
		log.Printf("[WARN] calendar/proc cannot update %d: %+v", y, err)
	}

	if err := p.UpdateCalendar(y + 1); err != nil {
		log.Printf("[WARN] calendar/proc cannot update %d: %+v", y+1, err)
	}
}

func (p *Processor) UpdateCalendar(y int) error {
	cal := p.MakeCalendar(y)
	if len(cal) > 0 {
		if err := p.Store.PutYear(y, cal); err != nil {
			return fmt.Errorf("calendar/proc cannot store year %d: %w", y, err)
		}
	}
	return nil
}

// MakeCalendar собирает календарь на один год из источников Src.
// Если два источника возвращают данные на одну дату, данные из последнего заменяют данные из первого.
// Если источник вернет ошибку, он будет пропущен. Если все источники вернут ошибку Src будет пуст, то
// возвращается пустой store.Months (len=0).
func (p *Processor) MakeCalendar(y int) store.Months {
	cal := make(store.Months, 12)

	for i, src := range p.Src {
		months, err := src.GetYear(y)
		if err != nil {
			log.Printf("[WARN] calendar/proc skipping source %d (%T), error: %+v", i, src, err)
			continue
		}

		cal = merge(cal, months)
	}

	return cal
}

func merge(m1 store.Months, m2 store.Months) store.Months {
	res := m1.Copy()
	for mon, days := range m2 {
		_, monExists := res[mon]
		if !monExists {
			res[mon] = make(store.Days, len(days))
		}

		for dayNum, day := range days {
			merged := res[mon][dayNum]
			merged.Working = day.Working

			if day.WeekDay != "" {
				merged.WeekDay = day.WeekDay
			}
			if day.Type != "" {
				merged.Type = day.Type
			}
			if day.Desc != "" {
				merged.Desc = day.Desc
			}

			res[mon][dayNum] = merged
		}
	}
	return res
}
