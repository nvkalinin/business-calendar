package cmd

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestSyncCmd(t *testing.T) {
	_, a, port := newApp(t, nil)
	go a.run()
	defer a.shutdown()
	waitForHTTP(port)

	cmd := newSyncCmd(port, []int{2021, 2022})
	err := cmd.Execute([]string{})
	require.NoError(t, err)

	// После синхронизации должен быть доступен календарь.

	status, json := getBody(t, fmt.Sprintf("http://localhost:%d/api/cal/2021/01/01", port))
	expJson := `{
		"weekDay": "fri",
		"working": true,
		"type": "normal"
	}`
	assert.Equal(t, 200, status)
	assert.JSONEq(t, expJson, json)

	status, json = getBody(t, fmt.Sprintf("http://localhost:%d/api/cal/2022/04/20", port))
	expJson = `{
		"weekDay": "wed",
		"working": true,
		"type": "normal"
	}`
	assert.Equal(t, 200, status)
	assert.JSONEq(t, expJson, json)
}

func newSyncCmd(port int, y []int) *SyncCmd {
	return &SyncCmd{
		ServerUrl:   fmt.Sprintf("http://localhost:%d", port),
		AdminPasswd: "pass",
		Timeout:     60 * time.Second,
		Years:       y,
	}
}