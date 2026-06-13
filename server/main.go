package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand/v2"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

const host = "67worker.lu2000luk.com"
const prefix = "\x1b[35;1m[*]\x1b[0m\x1b[37m "

type TransferChunkRange struct {
	rangeStart int64 `json:"s"`
	rangeEnd   int64 `json:"e"`
}

type TransferChunk struct {
	transferId string             `json:"id"`
	chunkrange TransferChunkRange `json:"r"`
	content    []byte             `json:"c"`
}

type Command struct {
	target      string        `json:"t"` // path
	destination string        `json:"d"` // path only for mv and copy
	uploadData  TransferChunk `json:"u"` // only for uploadChunk
	command     string        `json:"c"` // ls,mv,rm,get,ping,mkdir,copy,upload,uploadChunk
}

type LSResponseEntry struct {
	name   string `json:"n"`
	folder bool   `json:"f"` // false = file / true = folder
	size   int64  `json:"z"` // only for files
}

type UploadResponse struct {
	transferId string `json:"id"`
	success    bool   `json:"s"`
}

type LSResponse struct {
	success bool              `json:"s"`
	path    string            `json:"p"`
	entries []LSResponseEntry `json:"e"`
}

type GETResponse struct {
	success    bool   `json:"s"`
	name       string `json:"n"`
	transferId string `json:"id"`
}

type GenericResponse struct {
	success bool `json:"s"`
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

			if com.command == "ping" {
				log.Printf(prefix+"Received ping from %s", com.target)
				continue
			}

			if com.command == "ls" {
				if com.target == "" {
					log.Printf(prefix + "Missing path for ls")
				}
				log.Printf(prefix+"Received ls command for %s", com.target)

			}

			if com.command == "rm" {
				if com.target == "" {
					log.Printf(prefix + "Missing path for rm")
				}
			}

			if com.command == "mv" {
				if com.target == "" || com.destination == "" {
					log.Printf(prefix + "Missing path for mv")
				}
			}

			if com.command == "mkdir" {
				if com.target == "" {
					log.Printf(prefix + "Missing path for mkdir")
				}
			}

			if com.command == "get" {
				if com.target == "" {
					log.Printf(prefix + "Missing path for get")
				}
			}

			if com.command == "copy" {
				if com.target == "" {
					log.Printf(prefix + "Missing TARGET path for copy")
				}

				if com.destination == "" {
					log.Printf(prefix + "Missing DESTINATION path for copy")
				}
			}

			if com.command == "upload" {
				if com.target == "" {
					log.Printf(prefix + "Missing path for upload")
				}
			}

			if com.command == "uploadChunk" {
				if com.uploadData.transferId == "" {
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
