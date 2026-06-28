package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/gliderlabs/ssh"
)

type sshStdoutSink struct {
	mu     sync.Mutex
	writer io.Writer
}

func (s *sshStdoutSink) Set(writer io.Writer) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.writer = writer
}

func (s *sshStdoutSink) Clear(writer io.Writer) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.writer == writer {
		s.writer = nil
	}
}

func (s *sshStdoutSink) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.writer == nil {
		return 0, io.ErrClosedPipe
	}
	return s.writer.Write(p)
}

type sshStdinForwarder struct {
	mu   sync.Mutex
	line []byte
}

func (f *sshStdinForwarder) Write(p []byte) (int, error) {
	SendCommand(Command{
		Command: "stdin",
		Stdin:   append([]byte(nil), p...),
	})
	return len(p), nil
}

func (f *sshStdinForwarder) send(p []byte) {
	log.Printf("SSH stdin (%d bytes): %q\n", len(p), string(p))
	SendCommand(Command{
		Command: "stdin",
		Stdin:   append([]byte(nil), p...),
	})
}

var sshOut sshStdoutSink
var sshStarted atomic.Bool
var sshSessionActive atomic.Bool

func GetSSHIDPath() string {
	if runtime.GOOS == "windows" {
		return os.Getenv("USERPROFILE") + "\\.ssh\\id_rsa"
	}
	return os.Getenv("HOME") + "/.ssh/id_rsa"
}

func ServeSSH(key string) {
	if !sshStarted.CompareAndSwap(false, true) {
		log.Println("SSH server is already running.")
		return
	}

	log.Println("Starting SSH server on port 2020...")

	handler := func(s ssh.Session) {
		if !sshSessionActive.CompareAndSwap(false, true) {
			_, _ = s.Write([]byte("Another SSH session is already active.\n"))
			_ = s.Exit(1)
			return
		}

		sshOut.Set(s)
		defer func() {
			sshSessionActive.Store(false)
			sshOut.Clear(s)
		}()

		SendCommand(Command{
			Command:     "stdin",
			Destination: "start",
		})

		done := make(chan struct{}, 1)
		stdinForwarder := &sshStdinForwarder{}
		go func() {
			buf := make([]byte, 4096) // Small buffer for immediate forwarding
			for {
				n, err := s.Read(buf)
				if n > 0 {
					log.Println("n: ", n)
					_, writeErr := stdinForwarder.Write(buf[:n])
					if writeErr != nil {
						break
					}
				}
				if err != nil {
					break
				}
			}
			done <- struct{}{}
		}()

		pty, winCh, ok := s.Pty()
		if ok {
			SendCommand(Command{
				Command:     "stdin",
				Destination: "resize",
				Target:      fmt.Sprintf("%d|%d", pty.Window.Width, pty.Window.Height),
			})

			go func() {
				for win := range winCh {
					SendCommand(Command{
						Command:     "stdin",
						Destination: "resize",
						Target:      fmt.Sprintf("%d|%d", win.Width, win.Height),
					})
				}
			}()
		}

		<-done
		SendCommand(Command{
			Command: "stdin",
			Stdin:   []byte("exit\n"),
		})
	}

	srv := &ssh.Server{
		Addr:    "127.0.0.1:2020",
		Handler: handler,
		PtyCallback: func(ctx ssh.Context, pty ssh.Pty) bool {
			return true
		},
	}

	if err := srv.SetOption(ssh.HostKeyFile(GetSSHIDPath())); err != nil {
		sshStarted.Store(false)
		log.Fatalf("failed to set host key: %v", err)
	}

	log.Printf("Connect with: ssh %s@localhost -p 2020\n", key)
	if err := srv.ListenAndServe(); err != nil {
		sshStarted.Store(false)
		log.Printf("SSH server stopped: %v", err)
	}
}
