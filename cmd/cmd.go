package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

func makeUrl(serverUrl string, path string) string {
	return strings.TrimRight(serverUrl, "/") + path
}

func readJsonError(body []byte) error {
	restErr := &struct{ msg string }{}
	if err := json.Unmarshal(body, restErr); err != nil {
		return fmt.Errorf("cannot read error msg: %w", err)
	}
	return errors.New(restErr.msg)
}
