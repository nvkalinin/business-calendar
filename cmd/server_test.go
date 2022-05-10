package cmd

import (
	"fmt"
	"net/http"
	"syscall"
	"testing"
	"time"

	"github.com/jessevdk/go-flags"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServerCmd(t *testing.T) {
	_, a, port := newApp(t, nil)
	defer a.shutdown()

	go a.run()
	waitForHTTP(port)

	status, _ := getBody(t, fmt.Sprintf("http://localhost:%d/ping", port))
	assert.Equal(t, 200, status)

	status, _ = getBody(t, fmt.Sprintf("http://localhost:%d/api/cal/2022", port))
	assert.Equal(t, 404, status)
}

func TestServerCmd_syncOnStart(t *testing.T) {
	_, a, port := newApp(t, func(cmd *Server) {
		cmd.SyncOnStart = []string{"2021", "current", "next"}
		cmd.Source.Override = "testdata/override.yml"
	})
	defer a.shutdown()

	go a.run()
	waitForHTTP(port)
	time.Sleep(200 * time.Millisecond) // должно быть достаточно для generic календаря

	// Из generic-календаря.
	status, json := getBody(t, fmt.Sprintf("http://localhost:%d/api/cal/2021/01/01", port))
	expJson := `{
		"weekDay": "fri",
		"working": true,
		"type": "normal"
	}`
	assert.Equal(t, 200, status)
	assert.JSONEq(t, expJson, json)

	// Из override.yml.
	status, json = getBody(t, fmt.Sprintf("http://localhost:%d/api/cal/2021/01/02", port))
	expJson = `{
		"weekDay": "sat",
		"working": true,
		"type": "normal",
		"desc": "работаем"
	}`
	assert.Equal(t, 200, status)
	assert.JSONEq(t, expJson, json)

	y := time.Now().Year()

	status, _ = getBody(t, fmt.Sprintf("http://localhost:%d/api/cal/%d/01/01", port, y))
	assert.Equal(t, 200, status)

	status, _ = getBody(t, fmt.Sprintf("http://localhost:%d/api/cal/%d/01/01", port, y+1))
	assert.Equal(t, 200, status)
}

func TestServerCmd_autoSync(t *testing.T) {
	_, a, port := newApp(t, func(cmd *Server) {
		cmd.SyncAt = time.Now().Add(1 * time.Second).Format("15:04:05")
	})
	defer a.shutdown()

	go a.run()
	waitForHTTP(port)
	time.Sleep(1 * time.Second)

	y := time.Now().Year()
	status, _ := getBody(t, fmt.Sprintf("http://localhost:%d/api/cal/%d/01/01", port, y))
	assert.Equal(t, 200, status)
}

func TestServerCmd_signalsAndShutdown(t *testing.T) {
	cmd, _, port := newApp(t, nil)

	go func() {
		err := cmd.Execute([]string{})
		require.NoError(t, err)
	}()
	waitForHTTP(port)

	status, _ := getBody(t, fmt.Sprintf("http://localhost:%d/ping", port))
	assert.Equal(t, 200, status)

	err := syscall.Kill(syscall.Getpid(), syscall.SIGINT)
	require.NoError(t, err)
	time.Sleep(500 * time.Millisecond) // Должно хватить на звершение.

	cl := &http.Client{
		Timeout: 100 * time.Millisecond,
	}
	_, err = cl.Get(fmt.Sprintf("http://localhost:%d/ping", port))
	require.Error(t, err)
}

func TestServerCmd_fail(t *testing.T) {
	cmd := &Server{}
	_, _ = flags.ParseArgs(cmd, []string{
		"--store.engine=foo",
	})
	_, err := cmd.makeApp()
	assert.ErrorContains(t, err, "unknown store engine")

	cmd = &Server{}
	_, _ = flags.ParseArgs(cmd, []string{
		"--store.engine=memory",
		"--source.parser=foo",
	})
	_, err = cmd.makeApp()
	assert.ErrorContains(t, err, "unknown parser")

	cmd = &Server{}
	_, _ = flags.ParseArgs(cmd, []string{
		"--store.engine=memory",
		"--source.parser=consultant",
		"--sync-at=foo",
	})
	_, err = cmd.makeApp()
	assert.ErrorContains(t, err, "sync at")

	cmd = &Server{}
	_, _ = flags.ParseArgs(cmd, []string{
		"--store.engine=memory",
		"--source.parser=consultant",
		"--sync-at=05:00",
		"--sync-on-start=foo",
	})
	_, err = cmd.makeApp()
	assert.ErrorContains(t, err, "sync on start")

	cmd = &Server{}
	_, _ = flags.ParseArgs(cmd, []string{
		"--store.engine=memory",
		"--source.parser=consultant",
		"--sync-at=05:00",
		"--sync-on-start=2020", "--sync-on-start=current",
	})
	_, err = cmd.makeApp()
	assert.NoError(t, err)
}
