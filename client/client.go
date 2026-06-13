package main

import (
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

var id string = ""
var connecting atomic.Bool
var setLayout bool = true
var interrupt = make(chan os.Signal, 1)
var c unsafe.Pointer
var u url.URL
var reconnectStop = make(chan struct{})

func loadConn() *websocket.Conn {
	return (*websocket.Conn)(atomic.LoadPointer(&c))
}

func storeConn(conn *websocket.Conn) {
	atomic.StorePointer(&c, unsafe.Pointer(conn))
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

	u = url.URL{Scheme: "wss", Host: host, Path: "/", RawQuery: "id=p-" + id[0:4]}
	addr := u.String()
	fmt.Printf("Connecting to %s\n", addr)

	backoff := 1 * time.Second
	const maxBackoff = 30 * time.Second

	for attempt := 0; ; attempt++ {
		conn, _, err := websocket.DefaultDialer.Dial(addr, nil)
		if err == nil {
			storeConn(conn)
			fmt.Printf("Connected successfully to %s", addr)
			readLoop(conn)

			select {
			case <-reconnectStop:
				return
			default:
			}

			fmt.Printf("Connection lost, reconnecting to %s...", addr)
			backoff = 1 * time.Second
			continue
		}

		fmt.Printf("Connection failed: %v (attempt %d)\n", err, attempt+1)

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

		log.Printf("Message: %s", message)
	}
}

func loop() {
	imgui.ClearSizeCallbackPool()

	if imgui.BeginV("Connect", nil, imgui.WindowFlagsNoResize|imgui.WindowFlagsNoCollapse) {
		if imgui.InputTextWithHint("##id", "Your code...", &id, imgui.InputTextFlagsEnterReturnsTrue, nil) {
			go connect(id)
		}

		if imgui.Button("Connect") {
			go connect(id)
		}
	}
	imgui.End()

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
