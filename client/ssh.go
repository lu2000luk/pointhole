package main

import (
	"io"
	"log"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"unicode/utf8"

	"github.com/gliderlabs/ssh"
)

type sshStdoutSink struct {
	mu     sync.Mutex
	writer *io.PipeWriter
}

func (s *sshStdoutSink) Set(writer *io.PipeWriter) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.writer = writer
}

func (s *sshStdoutSink) Clear(writer *io.PipeWriter) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.writer == writer {
		s.writer = nil
	}
}

func (s *sshStdoutSink) Write(p []byte) (int, error) {
	s.mu.Lock()
	writer := s.writer
	s.mu.Unlock()

	if writer == nil {
		return 0, io.ErrClosedPipe
	}

	return writer.Write(p)
}

type sshStdinForwarder struct {
	mu   sync.Mutex
	line []byte
}

func (f *sshStdinForwarder) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	for _, b := range p {
		switch b {
		case 0x03: // Ctrl-C
			if len(f.line) > 0 {
				f.line = f.line[:0]
				_, _ = sshOut.Write([]byte("^C\r\n"))
				f.send([]byte("\n"))
				continue
			}

			_, _ = sshOut.Write([]byte("^C\r\n"))
			f.send([]byte{0x03})
		case '\r', '\n':
			f.line = append(f.line, '\n')
			_, _ = sshOut.Write([]byte("\r\n"))
			f.send(f.line)
			f.line = f.line[:0]
		case 0x7f, 0x08: // DEL and Ctrl-H backspace variants
			if len(f.line) == 0 {
				continue
			}

			_, size := utf8.DecodeLastRune(f.line)
			if size == 0 {
				size = 1
			}
			f.line = f.line[:len(f.line)-size]
			_, _ = sshOut.Write([]byte("\b \b"))
		default:
			f.line = append(f.line, b)
			_, _ = sshOut.Write([]byte{b})
		}
	}

	return len(p), nil
}

func (f *sshStdinForwarder) send(p []byte) {
	log.Printf("SSH stdin (%d bytes): %q\n", len(p), string(p))
	SendCommand(Command{
		Command: "stdin",
		Target:  string(p),
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

	ssh.Handle(func(s ssh.Session) {
		//  if s.User() != key {
		//  	log.Printf("Unauthorized SSH connection attempt from %s with user %s\n", s.RemoteAddr(), s.User())
		//  	s.Write([]byte("Unauthorized\n"))
		//  	s.Exit(1)
		//  	return
		//  }

		if !sshSessionActive.CompareAndSwap(false, true) {
			_, _ = s.Write([]byte("Another SSH session is already active.\n"))
			_ = s.Exit(1)
			return
		}

		stdoutReader, stdoutWriter := io.Pipe()
		sshOut.Set(stdoutWriter)
		defer func() {
			sshSessionActive.Store(false)
			sshOut.Clear(stdoutWriter)
			_ = stdoutWriter.Close()
			_ = stdoutReader.Close()
		}()

		SendCommand(Command{
			Command:     "stdin",
			Destination: "start",
		})

		done := make(chan struct{}, 2)
		stdinForwarder := &sshStdinForwarder{}
		go func() {
			_, _ = io.Copy(stdinForwarder, s)
			done <- struct{}{}
		}()
		go func() {
			_, _ = io.Copy(s, stdoutReader)
			done <- struct{}{}
		}()

		<-done
		SendCommand(Command{
			Command: "stdin",
			Target:  "exit\n",
		})
	})

	log.Printf("Connect with: ssh %s@localhost -p 2020\n", key)
	if err := ssh.ListenAndServe("127.0.0.1:2020", nil, ssh.HostKeyFile(GetSSHIDPath())); err != nil {
		sshStarted.Store(false)
		log.Printf("SSH server stopped: %v", err)
	}
}
