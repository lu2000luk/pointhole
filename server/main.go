package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand/v2"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

const host = "67worker.lu2000luk.com"
const prefix = "\x1b[35;1m[*]\x1b[0m\x1b[37m "

type TransferChunkRange struct {
	RangeStart int64 `json:"s"`
	RangeEnd   int64 `json:"e"`
}

type TransferChunk struct {
	TransferId string             `json:"id"`
	Chunkrange TransferChunkRange `json:"r"`
	Content    []byte             `json:"c"`
	Type       string             `json:"type"`
}

type Command struct {
	Target      string        `json:"t"` // path
	Destination string        `json:"d"` // path only for mv and copy
	UploadData  TransferChunk `json:"u"` // only for uploadChunk
	Command     string        `json:"c"` // ls,mv,rm,get,ping,mkdir,copy,upload,uploadChunk
}

type LSResponseEntry struct {
	Name   string `json:"n"`
	Folder bool   `json:"f"` // false = file / true = folder
	Size   int64  `json:"z"` // only for files
}

type UploadResponse struct {
	TransferId string `json:"id"`
	Success    bool   `json:"s"`
	Type       string `json:"type"`
}

type LSResponse struct {
	Success bool              `json:"s"`
	Path    string            `json:"p"`
	Entries []LSResponseEntry `json:"e"`
	Type    string            `json:"type"`
}

type GETResponse struct {
	Success    bool   `json:"s"`
	Name       string `json:"n"`
	TransferId string `json:"id"`
	Type       string `json:"type"`
}

type GenericResponse struct {
	Success bool   `json:"s"`
	Type    string `json:"type"`
}

func generateRandomString(length int) string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	var sb strings.Builder
	sb.Grow(length)
	for i := 0; i < length; i++ {
		randomIndex := rand.IntN(len(charset))
		sb.WriteByte(charset[randomIndex])
	}
	return sb.String()
}

func connect(u url.URL) (*websocket.Conn, error) {
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	return c, err
}

func ListDirectory(dirPath string) LSResponse {
	absPath, err := filepath.Abs(filepath.Clean(dirPath))
	if err != nil {
		absPath = dirPath // fallback
	}

	resp := LSResponse{
		Success: false,
		Path:    absPath,
		Entries: []LSResponseEntry{},
		Type:    "ls",
	}

	// Check if exists and is directory
	info, err := os.Stat(absPath)
	if err != nil {
		return resp
	}
	if !info.IsDir() {
		return resp
	}

	entries, err := os.ReadDir(absPath)
	if err != nil {
		return resp
	}

	resp.Entries = make([]LSResponseEntry, 0, len(entries))

	for _, entry := range entries {
		name := entry.Name()
		if name == "." || name == ".." {
			continue
		}

		isDir := entry.IsDir()
		var size int64 = 0

		if !isDir {
			if info, err := entry.Info(); err == nil {
				size = info.Size()
			}
		}

		resp.Entries = append(resp.Entries, LSResponseEntry{
			Name:   name,
			Folder: isDir,
			Size:   size,
		})
	}

	// Sort
	sort.Slice(resp.Entries, func(i, j int) bool {
		a, b := resp.Entries[i], resp.Entries[j]

		if a.Folder != b.Folder {
			return a.Folder
		}

		return strings.ToLower(a.Name) < strings.ToLower(b.Name)
	})

	resp.Success = true
	return resp
}

func main() {
	log.SetFlags(0)
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	// Generate key pair [id = 4][key = 8]
	id := generateRandomString(4)
	key := generateRandomString(8)

	fmt.Println(prefix+"Connect your instance: \x1b[30;47;1m", id+key, "\x1b[0m")

	u := url.URL{Scheme: "wss", Host: host, Path: "/", RawQuery: "id=P-" + id}

	done := make(chan struct{})

	go func() {
		defer close(done)

		c, err := connect(u)
		if err != nil {
			log.Println(prefix+"Error while connecting:", err)
			return
		}
		defer c.Close()

		fmt.Println(prefix + "Waiting for client...")

		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println(prefix+"Error while reading:", err)

				select {
				case <-interrupt:
					return
				default:
				}

				c.Close()
				backoff := 3 * time.Second
				for {
					select {
					case <-interrupt:
						return
					default:
					}
					c, err = connect(u)
					if err == nil {
						log.Println(prefix + "Reconnected successfully")
						break
					}
					log.Printf(prefix+"Reconnect failed: %v, retrying in %v", err, backoff)
					time.Sleep(backoff)
					if backoff < 30*time.Second {
						backoff *= 2
					}
				}
				continue
			}

			d, err := Decrypt(message, key)
			if err != nil {
				log.Println(prefix+"Error while decrypting:", err)
			}

			com := Command{}
			if err := json.Unmarshal(d, &com); err != nil {
				log.Println(prefix+"Invalid packet:", err)
			}

			if com.Command == "ping" {
				log.Printf(prefix + "Received ping")
				continue
			}

			if com.Command == "ls" {
				if com.Target == "" {
					log.Printf(prefix + "Missing path for ls")
				}
				log.Printf(prefix+"Received ls command for %s", com.Target)
				resp := ListDirectory(com.Target)
				jsonresp, err := json.Marshal(resp)

				if err != nil {
					fmt.Println(prefix+"Error while marshalling ls response:", err)
				}
				fmt.Println(string(jsonresp))

				encrypted, err := Encrypt(jsonresp, key)
				if err != nil {
					fmt.Println(prefix+"Error while encrypting ls response:", err)
				}

				err = c.WriteMessage(websocket.BinaryMessage, encrypted)
				if err != nil {
					log.Println(prefix+"Error while writing ls response:", err)
				}
			}

			if com.Command == "rm" {
				if com.Target == "" {
					log.Printf(prefix + "Missing path for rm")
				}
			}

			if com.Command == "mv" {
				if com.Target == "" || com.Destination == "" {
					log.Printf(prefix + "Missing path for mv")
				}
			}

			if com.Command == "mkdir" {
				if com.Target == "" {
					log.Printf(prefix + "Missing path for mkdir")
				}
			}

			if com.Command == "get" {
				if com.Target == "" {
					log.Printf(prefix + "Missing path for get")
				}
			}

			if com.Command == "copy" {
				if com.Target == "" {
					log.Printf(prefix + "Missing TARGET path for copy")
				}

				if com.Destination == "" {
					log.Printf(prefix + "Missing DESTINATION path for copy")
				}
			}

			if com.Command == "upload" {
				if com.Target == "" {
					log.Printf(prefix + "Missing path for upload")
				}
			}

			if com.Command == "uploadChunk" {
				if com.UploadData.TransferId == "" {
					log.Printf(prefix + "Missing ID for uploadChunk")
				}
			}

			log.Printf(prefix+"Message: %s", d)
		}
	}()

	select {
	case <-done:
	case <-interrupt:
		log.Println(prefix + "Closing...")
		select {
		case <-done:
		case <-time.After(time.Second):
		}
	}
}
