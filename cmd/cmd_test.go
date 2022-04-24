package cmd

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"io"
	"net"
	"net/http"
	"testing"
	"time"
)

func newApp(t *testing.T, cmdMod func(*ServerCmd)) (cmd *ServerCmd, app *app, port int) {
	port = unusedPort()

	cmd = &ServerCmd{}
	cmd.SyncOnStart = []string{}
	cmd.Web.Listen = fmt.Sprintf("127.0.0.1:%d", port)
	cmd.Web.AdminPasswd = "pass"
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
