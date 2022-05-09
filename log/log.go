package log

import (
	"log"
	"strings"
)

var AllowDebug = false

func Printf(format string, v ...any) {
	if !allowed(format) {
		return
	}
	log.Printf(format, v...)
}

func Fatalf(format string, v ...any) {
	if !allowed(format) {
		return
	}
	log.Fatalf(format, v...)
}

func allowed(s string) bool {
	if AllowDebug {
		return true
	}
	return !strings.HasPrefix(s, "[DEBUG]")
}
