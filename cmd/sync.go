package cmd

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/nvkalinin/business-calendar/log"
)

type Sync struct {
	ServerUrl   string        `long:"server-url" short:"s" env:"SERVER_URL" value-name:"str" default:"http://localhost" description:"URL сервера с REST API календаря."`
	AdminPasswd string        `long:"passwd" short:"p" env:"WEB_ADMIN_PASSWD" value-name:"str" description:"Пароль пользователя admin."`
	Timeout     time.Duration `long:"timeout" short:"t" env:"TIMEOUT" value-name:"duration" default:"60s" description:"Макс. время выполнения запроса."`
	Years       []int         `long:"year" short:"y" env:"YEAR" value-name:"int" required:"true" description:"Год, за который нужно синхронизировать календарь. Можно указывать несколько раз."`
}

func (s *Sync) Execute(args []string) error {
	ystr := make([]string, len(s.Years))
	for i, y := range s.Years {
		ystr[i] = strconv.Itoa(y)
	}

	params := url.Values{"y": ystr}
	body := strings.NewReader(params.Encode())

	url := makeUrl(s.ServerUrl, "/api/admin/sync")
	req, err := http.NewRequest(http.MethodPost, url, body)
	if err != nil {
		log.Fatalf("[ERROR] cannot make request: %v", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth("admin", s.AdminPasswd)
	log.Printf("[DEBUG] sync request: URL=%s, %#v", url, req)

	client := &http.Client{
		Timeout: s.Timeout,
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("[ERROR] cannot make request: %v", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("[WARN] cannot close response: %v", err)
		}
	}()
	log.Printf("[DEBUG] sync response: %#v", resp)

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("[ERROR] cannot read response: %v", err)
	}
	log.Printf("[DEBUG] sync resp body: %v", respBody)

	if resp.StatusCode != 200 {
		err := readJsonError(respBody)
		log.Fatalf("[ERROR] sync error (status %d): %v", resp.StatusCode, err)
	}

	res := map[int]string{}
	if err := json.Unmarshal(respBody, &res); err != nil {
		log.Fatalf("[ERROR] cannot parse response (status %d): %v", resp.StatusCode, err)
	}

	for y, syncRes := range res {
		if syncRes == "ok" {
			log.Printf("[INFO] year %d: ok", y)
		} else {
			log.Printf("[ERROR] year %d: %s", y, syncRes)
		}
	}
	return nil
}
