package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(t *testing.T) {

	os.Args = []string{"test", "server", "--secret=123456", "--store.bolt.path=/tmp/xyz", "--backup=/tmp",
		"--avatar.fs.path=/tmp", "--port=18202", "--url=https://demo.remark42.com", "--dbg", "--notify.type=none"}

	go func() {
		time.Sleep(500 * time.Millisecond)
		err := syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
		require.Nil(t, err)
	}()

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		st := time.Now()
		main()
		assert.True(t, time.Since(st).Seconds() < 1, "should take about 500msec")
		wg.Done()
	}()

	time.Sleep(200 * time.Millisecond) // let server start

	// send ping
	resp, err := http.Get("http://localhost:18202/api/v1/ping")
	require.Nil(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 200, resp.StatusCode)
	body, err := ioutil.ReadAll(resp.Body)
	assert.Nil(t, err)
	assert.Equal(t, "pong", string(body))

	wg.Wait()
}

func TestGetDump(t *testing.T) {
	dump := getDump()
	assert.True(t, strings.Contains(dump, "goroutine"))
	assert.True(t, strings.Contains(dump, "[running]"))
	assert.True(t, strings.Contains(dump, "backend/app/main.go"))
	log.Print("\n dump:" + dump)
}

func TestCrasher(t *testing.T) {
	if os.Getenv("BE_CRASHER") == "1" {
		main()
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestCrasher")
	cmd.Env = append(os.Environ(), "BE_CRASHER=1")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stdout
	err := cmd.Run()
	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		return
	}
	t.Fatalf("process ran with err %v, want exit status 1", err)
}
