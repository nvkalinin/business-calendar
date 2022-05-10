package parser

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/nvkalinin/business-calendar/log"
	"github.com/nvkalinin/business-calendar/store"
	"golang.org/x/text/unicode/norm"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type SuperJob struct {
	Client    *http.Client // Должен быть настроен Cookie Jar.
	UserAgent string
	baseURL   string // Только для тестирования.
}

func (s *SuperJob) GetYear(y int) (store.Months, error) {
	dom, err := s.getCalendarPage(y)
	if err != nil {
		return nil, err
	}

	months := make(store.Months, 12)
	for mon, monNode := range s.findMonths(dom) {
		days := s.findDays(monNode, mon, y)

		// На SuperJob в календарной сетке выходные дни отмечены как праздничные.
		// Поэтому нужно дополнительно парсить список праздников, чтобы отделить настоящие праздники от обычных выходных.
		months[mon] = s.parseRealHolidays(mon, dom, days)
	}

	return months, nil
}

func (s *SuperJob) getBaseURL() string {
	if s.baseURL != "" {
		return s.baseURL
	}
	return "https://www.superjob.ru"
}

// getCalendarPage делает запрос к странице календаря за год <y> и возвращает
// DOM-дерево этой страницы.
func (s *SuperJob) getCalendarPage(y int) (*goquery.Document, error) {
	url := fmt.Sprintf("%s/proizvodstvennyj_kalendar/%d/", s.getBaseURL(), y)
	req, _ := http.NewRequest("GET", url, nil)

	if s.UserAgent != "" {
		req.Header.Set("User-Agent", s.UserAgent)
	}
	log.Printf("[DEBUG] parser/superjob year %d request: URL=%s %#v", y, url, req)

	resp, err := s.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("parser/superjob cannot GET calendar page: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("[WARN] parser/superjob cannot close response: %+v", err)
		}
	}()
	log.Printf("[DEBUG] parser/superjob year %d response: %#v", y, resp)

	dom, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("parser/superjob cannot parse html: %w", err)
	}
	return dom, nil
}

// findMonths находит в DOM страницы календари за все месяцы и возвращает их DOM-поддеревья
// для дальнейшего парсинга. В этих поддеревьях не содержится информация о праздничных днях.
func (*SuperJob) findMonths(doc *goquery.Document) map[time.Month]*goquery.Selection {
	grids := doc.Find("div.MonthsList_grid")
	log.Printf("[DEBUG] parser/superjob found %d month nodes", grids.Length())

	byMonth := make(map[time.Month]*goquery.Selection, 12)
	for i := range grids.Nodes {
		grid := grids.Eq(i)

		nameNode := grid.Find("div.sj_h2")
		if nameNode.Length() == 0 {
			log.Printf("[WARN] parser/superjob skipping month at index %d: name node missing", i)
			continue
		}

		monthName := nameNode.Text()

		month, mapped := mapMonthName(monthName)
		if !mapped {
			log.Printf("[WARN] parser/superjob skipping month at index %d: unknown name '%s'", i, monthName)
			continue
		}

		if _, exists := byMonth[month]; exists {
			log.Printf("[WARN] parser/superjob skipping month at index %d: month with the same name '%s' was already found", i, monthName)
			continue
		}
		byMonth[month] = grid
	}

	if len(byMonth) != 12 {
		log.Printf("[WARN] parser/superjob returns incomplete calendar: expected 12 months, found %d", len(byMonth))
	}
	return byMonth
}

// findDays возвращает описания дней месяца n.
// Выходные дни будут иметь тип store.Holiday. Поэтому далее требуется парсинг праздников, чтобы определить реальные выходные.
func (s *SuperJob) findDays(n *goquery.Selection, m time.Month, y int) store.Days {
	maxDays := daysInMonth(y, m)
	dayNodes := s.dayNodes(n)

	days := make(store.Days, maxDays)
	for d := range dayNodes {
		num, err := s.parseDayNum(d.node)
		if err != nil {
			log.Printf("[WARN] parser/superjob %s, skipping day: %+v", m, err)
			continue
		}
		if num < 1 || num > maxDays {
			log.Printf("[WARN] parser/superjob %s: skipping day %d: out of bounds", m, num)
		}

		expWeekday := weekdayOf(y, m, num)
		if d.weekDay != expWeekday {
			log.Printf("[WARN] parser/superjob %s: skipping day %d: expected weekday %s, parsed %s", m, num, expWeekday, d.weekDay)
			continue
		}
		storedWeekday, _ := store.NewWeekDay(d.weekDay)

		day := store.Day{
			WeekDay: storedWeekday,
			Working: true,
			Type:    store.Normal,
		}

		if d.node.HasClass("MonthsList_preholiday") {
			day.Type = store.PreHoliday
		}
		if d.node.HasClass("MonthsList_holiday") {
			day.Type = store.Holiday
			day.Working = false
		}
		// В SuperJob выходные тоже отмечены как holiday.

		days[num] = day
	}

	if len(days) != maxDays {
		log.Printf("[WARN] parser/superjob %s: expected %d days, found %d", m, maxDays, len(days))
	}

	return days
}

type sjDay struct {
	node    *goquery.Selection
	weekDay time.Weekday
}

// dayNodes возвращает канал, в который записываются дни месяца n.
// С каждым днем возвращается день недели, как он указан в сетке календаря.
func (s *SuperJob) dayNodes(n *goquery.Selection) <-chan sjDay {
	wds := s.weekDays(n)

	dayNodes := n.Find("div.MonthsList_date")

	c := make(chan sjDay)
	go func() {
		dayNodes.Each(func(_ int, n *goquery.Selection) {
			wd := <-wds

			// Пропускаем дни соседних месяцев.
			if n.HasClass("h_color_gray") || n.HasClass("m_outshortday") || n.HasClass("m_outholiday") {
				return
			}

			c <- sjDay{n, wd}
		})
		close(c)
	}()
	return c
}

// weekDays возвращает канал, в который бесконечно записываются дни недели,
// как они были расположены в сетке календаря n.
func (s *SuperJob) weekDays(n *goquery.Selection) <-chan time.Weekday {
	nameToWeekDay := map[string]time.Weekday{
		"пн": time.Monday,
		"вт": time.Tuesday,
		"ср": time.Wednesday,
		"чт": time.Thursday,
		"пт": time.Friday,
		"сб": time.Saturday,
		"вс": time.Sunday,
	}

	wdNodes := n.Find(".MonthsList_weekdays .MonthsList_weekday")
	wds := make([]time.Weekday, wdNodes.Length())
	wdNodes.Each(func(i int, n *goquery.Selection) {
		wdName := n.Text()
		wdName = strings.TrimSpace(wdName)
		wdName = strings.ToLower(wdName)
		wds[i] = nameToWeekDay[wdName]
	})

	c := make(chan time.Weekday)
	go func() {
		for {
			for _, wd := range wds {
				c <- wd
			}
		}
	}()
	return c
}

func (s *SuperJob) parseDayNum(n *goquery.Selection) (int, error) {
	numNode := n.Find(".MonthsList_day")
	if numNode.Length() == 0 {
		return 0, fmt.Errorf("cannot parse day num from node '%s'", n.Text())
	}

	sNum := strings.TrimSpace(numNode.Text())
	num, err := strconv.Atoi(sNum)
	if err != nil {
		return 0, fmt.Errorf("cannot parse day num from text '%s'", sNum)
	}

	return num, nil
}

// parseRealHolidays находит на странице описание праздников указанного месяца и дополняет days:
// отмечает предпраздничные, праздничные дни, задает названия праздников, а все дни, что в календарной сетке
// были отмечены праздничными, но отсутствуют в списке праздников, меняет на обычные выходные.
func (s *SuperJob) parseRealHolidays(m time.Month, doc *goquery.Document, days store.Days) store.Days {
	summaryByType := doc.Find(fmt.Sprintf(".MonthsList_summary.m_%d", m))

	preHolidays := s.parseSummary(m, summaryByType.Find(".MonthsList_summary_preholiday"))
	log.Printf("[DEBUG] parser/superjob %s: found pre-holidays: %#v", m, preHolidays)

	holidays := make(map[int]string, 10)
	summaryByType.Find(".MonthsList_summary_holiday").Each(func(_ int, n *goquery.Selection) {
		for num, desc := range s.parseSummary(m, n) {
			holidays[num] = desc
		}
	})
	log.Printf("[DEBUG] parser/superjob %s: found holidays: %#v", m, preHolidays)

	resDays := make(store.Days, len(days))
	for num, day := range days {
		if desc, isPreHoliday := preHolidays[num]; isPreHoliday {
			day.Type = store.PreHoliday
			day.Desc = desc
		}
		if desc, isHoliday := holidays[num]; isHoliday {
			day.Type = store.Holiday
			day.Working = false
			day.Desc = desc
		} else if day.Type == store.Holiday {
			day.Type = store.Weekend
		}
		resDays[num] = day
	}

	return resDays
}

// parseSummary парсит описание праздника, на SuperJob для каждого праздника может быть указано несколько дней.
// Ключ возвращаемого map — номер дня, значение — название праздника.
func (s *SuperJob) parseSummary(m time.Month, n *goquery.Selection) map[int]string {
	if n.Length() == 0 {
		return nil
	}

	nameNode := n.Find(".MonthsList_summary_name")
	name := strings.TrimSpace(nameNode.Text())

	// Устраняем всякие неожиданные символы, вроде неразрывного пробела.
	// Конечно, эту задачу надо решать не так, но кому какое дело, если здесь только названия праздников и они предсказуемы.
	name = norm.NFKC.String(name)

	if len(name) == 0 {
		log.Printf("[WARN] parser/superjob %s: empty summary name", m)
	}

	dayNodes := n.Find(".MonthsList_summary_days span")
	days := make(map[int]string, dayNodes.Length())
	dayNodes.Each(func(i int, n *goquery.Selection) {
		sNum := strings.TrimSpace(n.Text())
		num, err := strconv.Atoi(sNum)
		if err != nil {
			log.Printf("[WARN] parser/superjob %s: skipping summary for day %s: %v", m, sNum, err)
			return
		}
		days[num] = name
	})

	return days
}
