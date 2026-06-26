//go:generate go-winres make --product-version=git-tag --file-version=git-tag
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

var host = "67worker.lu2000luk.com"

const prefix = "\x1b[35;1m[+]\x1b[0m\x1b[37m "

var transfers = make(map[string]string) // [id]:[path]
var demo = false

type TransferChunkRange struct {
	RangeStart int64 `json:"s"`
	RangeEnd   int64 `json:"e"`
}

type TransferChunk struct {
	TransferId string             `json:"id"`
	Chunkrange TransferChunkRange `json:"r"`
	Content    []byte             `json:"c"`
	Type       string             `json:"type"`
	IsEnd      bool               `json:"e"`
}

type Command struct {
	Target        string             `json:"t"` // path (transferId for getChunk)
	Destination   string             `json:"d"` // path only for mv and copy
	GetChunkRange TransferChunkRange `json:"r"` // only for getChunk
	UploadData    TransferChunk      `json:"u"` // only for uploadChunk
	Command       string             `json:"c"` // ls,mv,rm,get,ping,mkdir,copy,upload,uploadChunk,getChunk
}

type LSResponseEntry struct {
	Name   string `json:"n"`
	Folder bool   `json:"f"` // false = file / true = folder
	Size   int64  `json:"z"` // only for files
}

type UploadResponse struct {
	TransferId string `json:"id"`
	Path       string `json:"p"`
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
	Path       string `json:"p"`
	TransferId string `json:"id"`
	Type       string `json:"type"`
}

type GETChunkResponse struct {
	Success    bool          `json:"s"`
	Name       string        `json:"n"`
	TransferId string        `json:"id"`
	Chunk      TransferChunk `json:"c"`
	Type       string        `json:"type"`
}

type UploadGetChunkResponse struct {
	Success bool               `json:"s"` // no operation has to be done on the server
	Type    string             `json:"type"`
	Range   TransferChunkRange `json:"r"`
}

type GenericResponse struct {
	Success bool   `json:"s"`
	Type    string `json:"type"`
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
	id := GenerateRandomString(4)
	key := GenerateRandomString(8)

	if len(os.Args) > 1 && os.Args[1] == "--DO_NOT_USE_THIS_OR_YOU_WILL_BE_HACKED__DEMO" {
		log.Println(prefix + "WARNING: Demo mode enabled")
		id = "DEMO"
		key = "67676767"
		demo = true
	}

	if len(os.Args) > 2 && os.Args[1] == "--server" {
		host = os.Args[2]
	}

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

			log.Printf(prefix+"Message: %s", d)

			if com.Command == "ping" {
				log.Printf(prefix + "Received ping")
				resp := GenericResponse{
					Success: true,
					Type:    "ping",
				}

				MarshallAndSend(resp, c, key)
				continue
			}

			if com.Command == "ls" {
				if com.Target == "" {
					log.Printf(prefix + "Missing path for ls")
				}
				log.Printf(prefix+"Received ls command for %s", FixPathIfWindows(com.Target))
				resp := ListDirectory(FixPathIfWindows(com.Target))

				MarshallAndSend(resp, c, key)
			}

			if com.Command == "rm" {
				if com.Target == "" {
					log.Printf(prefix + "Missing path for rm")
				}

				if demo {
					if strings.Contains(com.Target, "demo_no_delete") {
						log.Println(prefix + "[DEMO] Skipping deletion of " + com.Target)
					}
				}

				log.Printf(prefix+"Received rm command for %s", FixPathIfWindows(com.Target))
				err := os.RemoveAll(FixPathIfWindows(com.Target))
				resp := GenericResponse{
					Success: err == nil,
					Type:    "rm",
				}

				MarshallAndSend(resp, c, key)
			}

			if com.Command == "mv" {
				if com.Target == "" || com.Destination == "" {
					log.Printf(prefix + "Missing path for mv")
				}

				log.Printf(prefix+"Received mv command from %s to %s", FixPathIfWindows(com.Target), FixPathIfWindows(com.Destination))

				if demo {
					if strings.Contains(com.Target, "demo_no_delete") {
						log.Println(prefix + "[DEMO] Skipping mv for " + com.Target)
					}
				}

				err := os.Rename(FixPathIfWindows(com.Target), FixPathIfWindows(com.Destination))
				resp := GenericResponse{
					Success: err == nil,
					Type:    "mv",
				}

				MarshallAndSend(resp, c, key)
			}

			if com.Command == "mkdir" {
				if com.Target == "" {
					log.Printf(prefix + "Missing path for mkdir")
				}

				log.Printf(prefix+"Received mkdir command for %s", FixPathIfWindows(com.Target))

				err := os.MkdirAll(FixPathIfWindows(com.Target), 0755)
				resp := GenericResponse{
					Success: err == nil,
					Type:    "mkdir",
				}

				MarshallAndSend(resp, c, key)
			}

			if com.Command == "get" {
				if com.Target == "" {
					log.Printf(prefix + "Missing path for get")
				}

				log.Printf(prefix+"Received get command for %s", com.Target)

				absPath, err := filepath.Abs(filepath.Clean(com.Target))
				if err != nil {
					absPath = com.Target // fallback
				}

				info, err := os.Stat(FixPathIfWindows(absPath))
				if err != nil || info.IsDir() {
					log.Printf(prefix+"Invalid path for get: %s", FixPathIfWindows(absPath))
					continue
				}

				transferId := GenerateRandomString(12)
				transfers[transferId] = FixPathIfWindows(absPath)

				resp := GETResponse{
					Success:    true,
					Path:       com.Target,
					TransferId: transferId,
					Type:       "get",
				}

				MarshallAndSend(resp, c, key)
			}

			if com.Command == "copy" {
				if com.Target == "" {
					log.Printf(prefix + "Missing TARGET path for copy")
				}

				if com.Destination == "" {
					log.Printf(prefix + "Missing DESTINATION path for copy")
				}

				log.Printf(prefix+"Received copy command from %s to %s", FixPathIfWindows(com.Target), FixPathIfWindows(com.Destination))

				if demo {
					if strings.Contains(com.Target, "demo_no_delete") {
						log.Println(prefix + "[DEMO] Skipping copy of " + com.Target)
					}
				}

				err := copyFileOrDir(FixPathIfWindows(com.Target), FixPathIfWindows(com.Destination))
				resp := GenericResponse{
					Success: err == nil,
					Type:    "copy",
				}
				MarshallAndSend(resp, c, key)
			}

			if com.Command == "upload" {
				if com.Target == "" {
					log.Printf(prefix + "Missing path for upload")
				}

				log.Printf(prefix+"Received upload command for %s", FixPathIfWindows(com.Target))

				absPath, err := filepath.Abs(filepath.Clean(FixPathIfWindows(com.Target)))
				if err != nil {
					absPath = FixPathIfWindows(com.Target) // fallback
				}

				info, err := os.Stat(absPath) // dont check for error since the file might not exist yet (for new files)

				if err == nil && info.IsDir() {
				}

				transferId := GenerateRandomString(12)
				transfers[transferId] = absPath

				resp := UploadResponse{
					Success:    true,
					TransferId: transferId,
					Path:       com.Target,
					Type:       "upload",
				}
				MarshallAndSend(resp, c, key)
			}

			if com.Command == "uploadChunk" {
				if com.UploadData.TransferId == "" {
					log.Printf(prefix + "Missing ID for uploadChunk")
				}

				log.Printf(prefix+"Received uploadChunk command for %s, range: %d-%d", com.UploadData.TransferId, com.UploadData.Chunkrange.RangeStart, com.UploadData.Chunkrange.RangeEnd)

				transferId := com.UploadData.TransferId
				filePath, exists := transfers[transferId]
				if !exists {
					log.Printf(prefix+"Invalid transfer ID for uploadChunk: %s", transferId)
					continue
				}

				err := WriteFileContentRange(filePath, com.UploadData.Chunkrange.RangeStart, com.UploadData.Content)
				if err != nil {
					log.Printf(prefix+"Error while writing file chunk: %v", err)
					continue
				}

				resp := UploadGetChunkResponse{
					Success: true,
					Type:    "uploadChunk",
					Range: TransferChunkRange{
						RangeStart: com.UploadData.Chunkrange.RangeStart,
						RangeEnd:   com.UploadData.Chunkrange.RangeEnd,
					},
				}

				MarshallAndSend(resp, c, key)
			}

			if com.Command == "getChunk" {
				if com.Target == "" {
					log.Printf(prefix + "Missing ID for getChunk")
				}

				transferId := com.Target
				filePath, exists := transfers[transferId]
				if !exists {
					log.Printf(prefix+"Invalid transfer ID for getChunk: %s", transferId)
					continue
				}

				if com.GetChunkRange.RangeEnd-com.GetChunkRange.RangeStart > 10*1024*1024 {
					log.Printf(prefix+"Requested chunk size exceeds 10MB for transfer ID: %s", transferId)
					continue
				}

				isEnd := false
				rangeEnd := com.GetChunkRange.RangeEnd
				info, err := os.Stat(filePath)
				if err == nil && rangeEnd >= info.Size()-1 {
					isEnd = true
					rangeEnd = info.Size() - 1
				}

				content, err := GetFileContentRange(filePath, com.GetChunkRange.RangeStart, rangeEnd)
				if err != nil {
					log.Printf(prefix+"Error while reading file chunk: %v", err)
					continue
				}

				chunkResp := TransferChunk{
					TransferId: transferId,
					Chunkrange: TransferChunkRange{
						RangeStart: com.GetChunkRange.RangeStart,
						RangeEnd:   rangeEnd,
					},
					Content: content,
					Type:    "getChunk",
					IsEnd:   isEnd,
				}

				MarshallAndSend(chunkResp, c, key)
			}

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
