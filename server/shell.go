package main

import (
	"bytes"
	"errors"
	"io"
	"log"
	"os"
	"runtime"
	"sync"

	"github.com/aymanbagabas/go-pty"
)

var inpipe io.WriteCloser
var outpipe io.ReadCloser
var errpipe io.ReadCloser

var started = false
var shellMu sync.Mutex
var shellCmd *pty.Cmd
var shellConn *SafeWebSocket
var shellKey string
var restartShell bool

func GetShell() string {
	if runtime.GOOS == "windows" {
		return "cmd"
	} else {
		return "sh"
	}
}

type ChunkInterceptor struct {
	Buffer  bytes.Buffer
	OnChunk func(chunk []byte)
}

func (ci *ChunkInterceptor) Write(p []byte) (n int, err error) {
	ci.OnChunk(p)

	return ci.Buffer.Write(p)
}

func HandlePipe(c *SafeWebSocket, key string) {
	if err := startShell(c, key); err != nil {
		log.Println("Error starting shell:", err)
	}
}

func startShell(c *SafeWebSocket, key string) error {
	shellMu.Lock()
	if started {
		shellConn = c
		shellKey = key
		shellMu.Unlock()
		return nil
	}
	started = true
	shellConn = c
	shellKey = key
	shellMu.Unlock()

	ptm, err := pty.New()
	if err != nil {
		log.Fatalf("failed to open pty: %s", err)
	}

	cmd := ptm.Command(GetShell())

	if err := cmd.Start(); err != nil {
		markShellStartFailed()
		ptm.Close()
		return err
	}

	shellMu.Lock()
	inpipe = ptm
	outpipe = ptm
	errpipe = nil
	shellCmd = cmd
	shellMu.Unlock()

	out_interceptor := &ChunkInterceptor{
		OnChunk: func(chunk []byte) {
			log.Printf("OutChunk (%d bytes): %s\n", len(chunk), string(chunk))

			MarshallAndSend(Stdout{
				Type:    "stdout",
				Content: chunk,
				Error:   false,
			}, c, key)
		},
	}

	go func() {
		defer ptm.Close()
		if _, err := io.Copy(out_interceptor, ptm); err != nil {
			log.Println("Error copying stdout:", err)
		}
	}()

	go func() {
		err := cmd.Wait()
		restart := markShellStopped(cmd)
		if err != nil {
			log.Println("Shell stopped:", err)
		}
		if restart {
			if err := startShell(c, key); err != nil {
				log.Println("Error restarting shell:", err)
			}
		}
	}()

	return nil
}

func markShellStartFailed() {
	shellMu.Lock()
	defer shellMu.Unlock()

	started = false
	restartShell = false
	shellCmd = nil
	inpipe = nil
	outpipe = nil
	errpipe = nil
}

func markShellStopped(cmd *pty.Cmd) bool {
	shellMu.Lock()
	defer shellMu.Unlock()

	if shellCmd != cmd {
		return false
	}

	shouldRestart := restartShell
	started = false
	restartShell = false
	shellCmd = nil
	inpipe = nil
	outpipe = nil
	errpipe = nil
	return shouldRestart
}

func InterruptShell() error {
	shellMu.Lock()
	cmd := shellCmd
	stdinPipe := inpipe
	shellMu.Unlock()

	if cmd == nil || cmd.Process == nil {
		return errors.New("shell is not running")
	}

	if stdinPipe != nil {
		_, _ = stdinPipe.Write([]byte{0x03})
	}

	if err := cmd.Process.Signal(os.Interrupt); err == nil {
		return nil
	}

	shellMu.Lock()
	if shellCmd == cmd {
		restartShell = true
	}
	shellMu.Unlock()

	return cmd.Process.Kill()
}
