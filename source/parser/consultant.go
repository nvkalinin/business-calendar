package parser

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/nvkalinin/business-calendar/store"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Consultant struct {
	Client    *http.Client
	UserAgent string
	baseURL   string // Только для тестирования.
}

func (c *Consultant) GetYear(y int) (store.Months, error) {
	dom, err := c.getCalendarPage(y)
	if err != nil {
		return nil, err
	}

	months := make(store.Months, 12)
	for mon, monNode := range c.findMonths(dom) {
		months[mon] = c.findDays(monNode, mon, y)
	}

	return months, nil
}

func (c *Consultant) getBaseURL() string {
	if c.baseURL != "" {
		return c.baseURL
	}
	return "https://www.consultant.ru"
}

func (c *Consultant) getCalendarPage(y int) (*goquery.Document, error) {
	url := fmt.Sprintf("%s/law/ref/calendar/proizvodstvennye/%d/", c.getBaseURL(), y)
	req, _ := http.NewRequest("GET", url, nil)

	if c.UserAgent != "" {
		req.Header.Set("User-Agent", c.UserAgent)
	}

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("parser/consultant cannot GET calendar page: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("[WARN] parser/consultant cannot close response: %+v", err)
		}
	}()

	dom, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("parser/consultant cannot parse html: %w", err)
	}
	return dom, nil
}

func (*Consultant) findMonths(doc *goquery.Document) map[time.Month]*goquery.Selection {
	tables := doc.Find("table.cal")
	if tables.Length() != 12 {
		log.Printf("[WARN] parser/consultant expected 12 months, found %d", tables.Length())
	}

	byMonth := make(map[time.Month]*goquery.Selection, 12)
	for i := range tables.Nodes {
		tab := tables.Eq(i)

		nameNode := tab.Find("th.month")
		if nameNode.Length() == 0 {
			log.Printf("[WARN] parser/consultant skipping month at index %d: name node missing", i)
			continue
		}

		monthName := nameNode.Text()

		month, mapped := mapMonthName(monthName)
		if !mapped {
			log.Printf("[WARN] parser/consultant skipping month at index %d: unknown name '%s'", i, monthName)
			continue
		}

		if _, exists := byMonth[month]; exists {
			log.Printf("[WARN] parser/consultant skipping month at index %d: month with the same name '%s' was already found", i, monthName)
			continue
		}
		byMonth[month] = tab
	}

	if len(byMonth) != 12 {
		log.Printf("[WARN] parser/consultant returns incomplete calendar: expected 12 months, found %d", len(byMonth))
	}
	return byMonth
}

func (c *Consultant) findDays(n *goquery.Selection, m time.Month, y int) store.Days {
	maxDays := daysInMonth(y, m)

	dayNodes := n.Find("td:not(.inactively)")
	if dayNodes.Length() != maxDays {
		log.Printf("[WARN] parser/consultant %s: expected %d days, found %d", m, maxDays, dayNodes.Length())
	}

	days := make(store.Days, maxDays)
	for i := range dayNodes.Nodes {
		dayNode := dayNodes.Eq(i)

		sNum, num, err := c.parseDayNum(dayNode)
		if err != nil {
			log.Printf("[WARN] parser/consultant %s: skipping day '%s': %+v", m, sNum, err)
			continue
		}
		if num < 1 || num > maxDays {
			log.Printf("[WARN] parser/consultant %s: skipping day %d: out of bounds", m, num)
		}

		weekday, mapped := mapWeekday(dayNode.Index())
		if !mapped {
			log.Printf("[WARN] parser/consultant %s: skipping day %d: unknown weekday %d", m, num, dayNode.Index())
			continue
		}
		expWeekday := weekdayOf(y, m, num)
		if weekday != expWeekday {
			log.Printf("[WARN] parser/consultant %s: skipping day %d: expected weekday %s, parsed %s", m, num, expWeekday, weekday)
			continue
		}
		storedWeekday, _ := store.NewWeekDay(weekday)

		day := store.Day{
			WeekDay: storedWeekday,
			Working: true,
			Type:    store.Normal,
		}

		if dayNode.HasClass("preholiday") {
			day.Type = store.PreHoliday
		}
		if dayNode.HasClass("weekend") {
			day.Working = false
			day.Type = store.Weekend
		}
		if dayNode.HasClass("holiday") {
			day.Type = store.Holiday
		}
		if dayNode.HasClass("nowork") {
			day.Type = store.NonWorking
		}

		days[num] = day
	}

	return days
}

func (c *Consultant) parseDayNum(node *goquery.Selection) (raw string, num int, err error) {
	sNum := node.Text()
	sNum = strings.TrimSpace(sNum)
	sNum = strings.Trim(sNum, "*")
	num, err = strconv.Atoi(sNum)
	return sNum, num, err
}
