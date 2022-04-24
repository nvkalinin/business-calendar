package cmd

import (
	"compress/gzip"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBackupCmd(t *testing.T) {
	// 1. Запустить app, подождать SyncOnStart, сделать бекап.

	dir := t.TempDir()
	_, a, port := newApp(t, func(cmd *ServerCmd) {
		cmd.SyncOnStart = []string{"2021"}
		cmd.Store.Engine = "bolt"
		cmd.Store.Bolt.File = dir + "/cal.bolt"
	})

	go a.run()
	defer a.shutdown()
	waitForHTTP(port)
	time.Sleep(200 * time.Millisecond) // должно быть достаточно для generic календаря

	cmd := newBackupCmd(port)

	// Тестируем также генерацию имени файла. Для этого переходим во временную директорию.
	// После теста возвращаемся туда, где были.
	if wd, err := os.Getwd(); err == nil {
		defer os.Chdir(wd)
	}
	require.NoError(t, os.Chdir(dir))

	err := cmd.Execute([]string{})
	require.NoError(t, err)

	files, err := filepath.Glob("cal_*.bolt.gz")
	require.NoError(t, err)
	require.Len(t, files, 1)

	// 2. Запустить новый app из бекапа и проверить наличие календаря.

	dbFile := gzipDecompress(t, files[0])
	_, a, port = newApp(t, func(cmd *ServerCmd) {
		cmd.Store.Engine = "bolt"
		cmd.Store.Bolt.File = dbFile
	})

	go a.run()
	defer a.shutdown()
	waitForHTTP(port)

	status, json := getBody(t, fmt.Sprintf("http://localhost:%d/api/cal/2021/01/01", port))
	expJson := `{
		"weekDay": "fri",
		"working": true,
		"type": "normal"
	}`
	assert.Equal(t, 200, status)
	assert.JSONEq(t, expJson, json)
}

func newBackupCmd(port int) (cmd *BackupCmd) {
	return &BackupCmd{
		ServerUrl:   fmt.Sprintf("http://127.0.0.1:%d", port),
		AdminPasswd: "pass",
		Timeout:     10 * time.Minute,
	}
}

func gzipDecompress(t *testing.T, path string) (decompressedPath string) {
	f, err := os.Open(path)
	require.NoError(t, err)
	defer f.Close()

	decomp := strings.TrimSuffix(path, ".gz")
	if decomp == path {
		decomp = path + "_decomp"
	}

	df, err := os.Create(decomp)
	require.NoError(t, err)
	defer df.Close()

	r, err := gzip.NewReader(f)
	require.NoError(t, err)
	defer r.Close()

	_, err = io.Copy(df, r)
	require.NoError(t, err)

	return decomp
}
