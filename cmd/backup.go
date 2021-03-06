package cmd

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"time"

	"github.com/nvkalinin/business-calendar/log"
)

type Backup struct {
	ServerUrl   string        `long:"server-url" short:"s" env:"SERVER_URL" default:"http://localhost" description:"URL сервера с REST API календаря."`
	AdminPasswd string        `long:"passwd" short:"p" env:"WEB_ADMIN_PASSWD" description:"Пароль пользователя admin."`
	OutFile     string        `long:"out" short:"o" env:"OUT" description:"Путь к файлу, куда сохранить бекап. По умолчанию: cal_YYYY-MM-DD.bolt.gz. Значение '-' выводит в стандартный поток вывода."`
	Timeout     time.Duration `long:"timeout" short:"t" env:"TIMEOUT" default:"600s" description:"Макс. время выполнения запроса."`
}

func (b *Backup) Execute(args []string) error {
	url := makeUrl(b.ServerUrl, "/api/admin/backup")
	req, err := http.NewRequest(http.MethodGet, url, http.NoBody)
	if err != nil {
		log.Fatalf("[ERROR] cannot create request: %v", err)
	}
	req.SetBasicAuth("admin", b.AdminPasswd)
	log.Printf("[DEBUG] backup request: URL=%s, %#v", url, req)

	client := &http.Client{Timeout: b.Timeout}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("[ERROR] cannot make request: %v", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("[WARN] cannot close resp body: %v", err)
		}
	}()
	log.Printf("[DEBUG] backup response: %#v", resp)

	if resp.StatusCode != 200 {
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Fatalf("[ERROR] cannot read err response (status %d): %v", resp.StatusCode, err)
		}
		err = readJsonError(respBody)
		log.Fatalf("[ERROR] backup error (status %d): %v", resp.StatusCode, err)
	}

	fname := b.filename(resp)
	log.Printf("[DEBUG] saving backup to file '%s'", fname)
	var f *os.File
	if fname == "-" {
		f = os.Stdout
	} else {
		f, err = os.Create(fname)
		if err != nil {
			log.Fatalf("[ERROR] cannot open %s: %v", fname, err)
		}
		defer func() {
			if err := f.Close(); err != nil {
				log.Printf("[WARN] cannot close %s: %v", fname, err)
			}
		}()
	}

	written, err := io.Copy(f, resp.Body)
	if err != nil {
		log.Fatalf("[ERROR] cannot save backup to %s: %v", fname, err)
	}
	log.Printf("[DEBUG] %d bytes written", written)

	return nil
}

func (b *Backup) filename(resp *http.Response) string {
	if len(b.OutFile) > 0 {
		return b.OutFile
	}

	defName := fmt.Sprintf("cal_%s.bolt.gz", time.Now().Format("2006-01-02"))

	vals, ok := resp.Header["Content-Disposition"]
	if !ok || len(vals) == 0 {
		return defName
	}

	_, params, err := mime.ParseMediaType(vals[0])
	if err != nil {
		return defName
	}

	name, ok := params["filename"]
	if !ok || len(name) == 0 {
		return defName
	}

	return name
}
