package main

import (
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
const prefix = "\x1b[35;1m[*]\x1b[0m "

type Command struct {
	target      string `json:"t"` // path
	destination string `json:"d"` // path
	command     string `json:"c"` // ls,mv,rm,get
}

type LSResponseEntry struct {
	success bool   `json:"s"`
	name    string `json:"n"`
	folder  bool   `json:"f"` // false = file / true = folder
	size    int64  `json:"z"` // only for files
}

type LSResponse = []LSResponseEntry

type GETResponse struct {
	success    bool   `json:"s"`
	name       string `json:"n"`
	transferId string `json:"id"`
}

type GenericResponse struct {
	success bool `json:"s"`
}

type TransferChunkRange struct {
	rangeStart int64 `json:"s"`
	rangeEnd   int64 `json:"e"`
}

type TransferChunk struct {
	transferId string             `json:"id"`
	chunkrange TransferChunkRange `json:"r"`
	content    []byte             `json:"c"`
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

func main() {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	// Generate key pair [id = 4][key = 8]
	id := generateRandomString(4)
	key := generateRandomString(8)

	fmt.Println(prefix+"Connect your instance: \x1b[30;47;1m", id+key, "\x1b[0m")

	u := url.URL{Scheme: "wss", Host: host, Path: "/", RawQuery: "id=" + id}

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)

	if err != nil {
		log.Fatal("Error while connecting:", err)
		return
	}

	defer c.Close()

	done := make(chan struct{})

	fmt.Println(prefix + "Waiting for client...")

	go func() {
		defer close(done)
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("Error while reading:", err)
				return
			}

			d, err := Decrypt(message, key)
			if err != nil {
				log.Println("Error while decrypting:", err)
			}

			log.Printf(prefix+"Message: %s", d)
		}
	}()

	for {
		select {
		case <-done:
			return
		case <-interrupt:
			log.Println(prefix + "Closing websocket connection...")

			err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("Error while closing:", err)
				return
			}

			select { // wait for coroutine to close
			case <-done:
			case <-time.After(time.Second):
			}
			return
		}
	}
}
