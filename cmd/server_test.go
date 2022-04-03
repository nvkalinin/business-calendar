package cmd

import (
	"fmt"
	"io"
	"net"
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
	_, a, port := newApp(t, func(cmd *ServerCmd) {
		cmd.SyncOnStart = []string{"2021", "current", "next"}
	})
	defer a.shutdown()

	go a.run()
	waitForHTTP(port)
	time.Sleep(200 * time.Millisecond) // должно быть достаточно для generic календаря

	status, json := getBody(t, fmt.Sprintf("http://localhost:%d/api/cal/2021/01/01", port))
	expJson := `{
		"weekDay": "fri",
		"working": true,
		"type": "normal"
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
	_, a, port := newApp(t, func(cmd *ServerCmd) {
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
	cmd := &ServerCmd{}
	flags.ParseArgs(cmd, []string{
		"--store.engine=foo",
	})
	_, err := cmd.makeApp()
	assert.ErrorContains(t, err, "unknown store engine")

	cmd = &ServerCmd{}
	flags.ParseArgs(cmd, []string{
		"--store.engine=memory",
		"--source.parser=foo",
	})
	_, err = cmd.makeApp()
	assert.ErrorContains(t, err, "unknown parser")

	cmd = &ServerCmd{}
	flags.ParseArgs(cmd, []string{
		"--store.engine=memory",
		"--source.parser=consultant",
		"--sync-at=foo",
	})
	_, err = cmd.makeApp()
	assert.ErrorContains(t, err, "sync at")

	cmd = &ServerCmd{}
	flags.ParseArgs(cmd, []string{
		"--store.engine=memory",
		"--source.parser=consultant",
		"--sync-at=05:00",
		"--sync-on-start=foo",
	})
	_, err = cmd.makeApp()
	assert.ErrorContains(t, err, "sync on start")

	cmd = &ServerCmd{}
	flags.ParseArgs(cmd, []string{
		"--store.engine=memory",
		"--source.parser=consultant",
		"--sync-at=05:00",
		"--sync-on-start=2020", "--sync-on-start=current",
	})
	_, err = cmd.makeApp()
	assert.NoError(t, err)
}

func newApp(t *testing.T, cmdMod func(*ServerCmd)) (cmd *ServerCmd, app *app, port int) {
	port = unusedPort()

	cmd = &ServerCmd{}
	cmd.SyncOnStart = []string{}
	cmd.Web.Listen = fmt.Sprintf("127.0.0.1:%d", port)
	cmd.Web.ReadTimeout = 5 * time.Second
	cmd.Web.ReadHeaderTimeout = 5 * time.Second
	cmd.Web.WriteTimeout = 5 * time.Second
	cmd.Web.IdleTimeout = 30 * time.Second
	cmd.Web.RateLimiter.ReqLimit = 100
	cmd.Web.RateLimiter.LimitWindow = 1 * time.Second
	cmd.Store.Engine = EngineType("memory")
	cmd.Source.Parser = ParserType("none")
	if cmdMod != nil {
		cmdMod(cmd)
	}

	var err error
	app, err = cmd.makeApp()
	require.NoError(t, err)

	return cmd, app, port
}

func unusedPort() int {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := l.Close(); err != nil {
			panic(err)
		}
	}()

	return l.Addr().(*net.TCPAddr).Port
}

func waitForHTTP(port int) {
	for i := 0; i < 100; i++ {
		time.Sleep(50 * time.Millisecond)
		if resp, err := http.Get(fmt.Sprintf("http://localhost:%d", port)); err == nil {
			resp.Body.Close()
			return
		}
	}
	panic(fmt.Sprintf("cannot connect to localhost:%d", port))
}

func getBody(t *testing.T, url string) (int, string) {
	resp, err := http.Get(url)
	require.NoError(t, err)
	defer resp.Body.Close()
	json, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return resp.StatusCode, string(json)
}
