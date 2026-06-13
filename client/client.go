package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/AllenDang/cimgui-go/backend"
	"github.com/AllenDang/cimgui-go/backend/glfwbackend"
	"github.com/AllenDang/cimgui-go/imgui"
	_ "github.com/AllenDang/cimgui-go/impl/glfw"
	"github.com/gorilla/websocket"
)

var host = "67worker.lu2000luk.com"
var currentBackend backend.Backend[glfwbackend.GLFWWindowFlags]

// TYPES (SAME AS IN server/main.go)

type TransferChunkRange struct {
	RangeStart int64 `json:"s"`
	RangeEnd   int64 `json:"e"`
}

type TransferChunk struct {
	TransferId string             `json:"id"`
	Chunkrange TransferChunkRange `json:"r"`
	Content    []byte             `json:"c"`
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
}

type LSResponse struct {
	Success bool              `json:"s"`
	Path    string            `json:"p"`
	Entries []LSResponseEntry `json:"e"`
}

type GETResponse struct {
	Success    bool   `json:"s"`
	Name       string `json:"n"`
	TransferId string `json:"id"`
}

type GenericResponse struct {
	Success bool `json:"s"`
}

// -------------------------------------

var id string = ""
var connecting atomic.Bool
var setLayout bool = true
var interrupt = make(chan os.Signal, 1)
var c unsafe.Pointer
var u url.URL
var reconnectStop = make(chan struct{})
var connected bool = false
var reconnecting bool = false

// packet debugger window
var commandName = ""

var commandTarget = ""

var commandDestination = ""

func loadConn() *websocket.Conn {
	return (*websocket.Conn)(atomic.LoadPointer(&c))
}

func storeConn(conn *websocket.Conn) {
	atomic.StorePointer(&c, unsafe.Pointer(conn))
}

func sendCommand(command Command) {
	conn := loadConn()
	if conn == nil {
		fmt.Println("No connection available")
		return
	}

	if command.Command == "" {
		println("Command name cannot be empty")
		return
	}

	pass := id[4:]
	payload, err := json.Marshal(command)

	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Sending payload: %+v\n", command)
	payloadEncrypted, err := Encrypt(payload, pass)
	fmt.Printf("Encrypted payload: %x\n", payloadEncrypted)

	if err != nil {
		log.Fatal(err)
	}

	err = conn.WriteMessage(websocket.BinaryMessage, payloadEncrypted)
	if err != nil {
		log.Fatal(err)
		return
	}
}

func connect(id string) {
	if connecting.Load() {
		return
	}
	connecting.Store(true)
	defer connecting.Store(false)

	if len(id) != 12 {
		fmt.Println("Invalid ID")
		return
	}

	close(reconnectStop)
	reconnectStop = make(chan struct{})

	if old := loadConn(); old != nil {
		old.Close()
	}

	u = url.URL{Scheme: "wss", Host: host, Path: "/", RawQuery: "id=P-" + id[0:4]}
	addr := u.String()
	fmt.Printf("Connecting to %s\n", addr)

	backoff := 1 * time.Second
	const maxBackoff = 30 * time.Second

	for attempt := 0; ; attempt++ {
		conn, _, err := websocket.DefaultDialer.Dial(addr, nil)
		if err == nil {
			storeConn(conn)
			fmt.Printf("Connected successfully to %s\n", addr)
			connected = true
			reconnecting = false
			readLoop(conn)

			select {
			case <-reconnectStop:
				return
			default:
			}

			fmt.Printf("Connection lost, reconnecting to %s...\n", addr)
			reconnecting = true
			backoff = 1 * time.Second
			continue
		}

		fmt.Printf("Connection failed: %v (attempt %d)\n", err, attempt+1)
		connected = false

		select {
		case <-time.After(backoff):
		case <-reconnectStop:
			fmt.Println("Reconnection cancelled")
			return
		}

		backoff *= 2
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
	}
}

func readLoop(conn *websocket.Conn) {
	defer conn.Close()

	for {
		select {
		case <-reconnectStop:
			return
		default:
		}

		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Println("Error while reading:", err)
			storeConn(nil)
			return
		}

		pass := id[4:]
		decrypted, err := Decrypt(message, pass)
		if err != nil {
			log.Println("Error while decrypting:", err)
		}

		log.Printf("Message: %s", decrypted)
	}
}

func loop() {
	imgui.ClearSizeCallbackPool()

	if !connected {
		if imgui.BeginV("Connect", nil, imgui.WindowFlagsNoResize|imgui.WindowFlagsNoCollapse) {
			if imgui.InputTextWithHint("##id", "Your code...", &id, imgui.InputTextFlagsEnterReturnsTrue, nil) {
				go connect(id)
			}

			if imgui.Button("Connect") {
				go connect(id)
			}
		}
		imgui.End()
	}

	if connected {
		if imgui.Begin("Packet Debugger") {
			imgui.InputTextWithHint("##command", "Command name...", &commandName, 0, nil)
			imgui.InputTextWithHint("##target", "Command target...", &commandTarget, 0, nil)
			imgui.InputTextWithHint("##destination", "Command destination...", &commandDestination, 0, nil)
			if imgui.Button("Send") {
				sendCommand(Command{
					Target:      commandTarget,
					Command:     commandName,
					Destination: commandDestination,
				})
				commandName = ""
				commandTarget = ""
				commandDestination = ""
			}
		}
		imgui.End()
	}

	if setLayout {
		viewport := imgui.MainViewport().Size()

		imgui.SetWindowSizeStr("Connect", imgui.Vec2{300, 100})
		imgui.SetWindowPosStr("Connect", imgui.Vec2{viewport.X / 2, viewport.Y / 2})
	}

	setLayout = false
}

func main() {
	log.SetFlags(0)
	signal.Notify(interrupt, os.Interrupt)

	niceGLFWBackend := glfwbackend.NewGLFWBackend()
	currentBackend, _ = backend.CreateBackend(niceGLFWBackend)

	// Flags
	currentBackend.SetWindowFlags(glfwbackend.GLFWWindowFlagsResizable, 0)
	currentBackend.SetWindowFlags(glfwbackend.GLFWWindowFlagsTransparent, 0)
	currentBackend.SetWindowFlags(glfwbackend.GLFWWindowFlagsDecorated, 1)
	currentBackend.SetWindowFlags(glfwbackend.GLFWWindowFlagsFloating, 0)
	currentBackend.SetWindowFlags(glfwbackend.GLFWWindowFlagsMaximized, 0)

	currentBackend.CreateWindow("PointHole", 920, 560)

	currentBackend.SetCloseCallback(func() {
		fmt.Println("Bye!")
	})

	go func() {
		<-interrupt
		log.Println("Interrupted!")
		close(reconnectStop)
		conn := loadConn()
		if conn != nil {
			conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			conn.Close()
		}
	}()

	currentBackend.Run(loop)
}
