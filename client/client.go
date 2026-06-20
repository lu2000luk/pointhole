package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"strings"
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
	Type       string             `json:"type"`
	IsEnd      bool               `json:"e"`
}

type Command struct {
	Target        string             `json:"t"` // path (transferId for getChunk)
	Destination   string             `json:"d"` // path only for mv and copy
	GetChunkRange TransferChunkRange `json:"r"`
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

var copiedPath string = ""
var isCut bool = false

var emulatedFS = make(map[string][]LSResponseEntry)

// browser window

var browserPath = "/"
var sentLSPacketFor = ""
var pathInput = browserPath

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

		genericTemplate := GenericResponse{}
		err = json.Unmarshal(decrypted, &genericTemplate) // Unmarshal to GenericResponse since all responses implement it (to get the type and unmarshal correctly)

		if err != nil {
			log.Println("Error while unmarshaling:", err)
		}

		switch genericTemplate.Type {
		case "ls":
			lsTemplate := LSResponse{}
			_ = json.Unmarshal(decrypted, &lsTemplate)
			fixedPath := lsTemplate.Path
			fixedPath = strings.Replace(fixedPath, "C:\\", "/", -1) // windows compat
			fixedPath = strings.Replace(fixedPath, "\\", "/", -1)
			fmt.Printf("EmulatedFS Path: %s\n", fixedPath)
			emulatedFS[fixedPath] = lsTemplate.Entries

		case "rm", "mkdir", "mv", "copy":
			// refresh
			sendCommand(Command{
				Target:  browserPath,
				Command: "ls",
			})

		default:
			log.Println("Unknown type:", genericTemplate.Type)
		}

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
				setLayout = true
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

		if imgui.Begin("Browser") {
			if browserPath != sentLSPacketFor {
				sendCommand(Command{
					Target:  browserPath,
					Command: "ls",
				})

				sentLSPacketFor = browserPath
			}

			if imgui.InputTextWithHint("##pathInput", "/", &pathInput, imgui.InputTextFlagsEnterReturnsTrue, nil) {
				fixedPath := pathInput
				fixedPath = strings.Replace(fixedPath, "C:\\", "/", -1) // windows compat
				fixedPath = strings.Replace(fixedPath, "\\", "/", -1)
				fixedPath = strings.Trim(fixedPath, "/")
				browserPath = fixedPath
				pathInput = fixedPath
			}

			imgui.SameLine()

			if imgui.Button("Refresh") {
				sendCommand(Command{
					Target:  browserPath,
					Command: "ls",
				})
			}

			imgui.SameLine()

			if imgui.Button("^") {
				if browserPath != "/" {
					browserPath = browserPath[0:strings.LastIndex(browserPath, "/")]
					pathInput = browserPath
				}
			}

			if copiedPath != "" {
				imgui.SameLine()
				if imgui.Button("Paste") {
					if isCut {
						sendCommand(Command{
							Target:      copiedPath,
							Destination: browserPath + "/" + copiedPath[strings.LastIndex(copiedPath, "/")+1:],
							Command:     "mv",
						})
						copiedPath = browserPath + "/" + copiedPath[strings.LastIndex(copiedPath, "/")+1:]
						isCut = false
					} else {
						sendCommand(Command{
							Target:      copiedPath,
							Destination: browserPath + "/" + copiedPath[strings.LastIndex(copiedPath, "/")+1:],
							Command:     "copy",
						})
					}
				}
			}

			imgui.Spacing()

			emulatedEntry := emulatedFS[browserPath]

			for _, entry := range emulatedEntry {
				if entry.Folder == true {
					if imgui.Button(entry.Name + " >") {
						if strings.HasSuffix(browserPath, "/") {
							browserPath = browserPath + entry.Name
							pathInput = browserPath
						} else {
							browserPath = browserPath + "/" + entry.Name
							pathInput = browserPath
						}
					}
				} else {
					imgui.Button(entry.Name)
				}

				if imgui.BeginPopupContextItem() {
					if imgui.MenuItemBool("Copy") {
						copiedPath = browserPath + "/" + entry.Name
						isCut = false
					}
					if imgui.MenuItemBool("Cut") {
						copiedPath = browserPath + "/" + entry.Name
						isCut = true
					}

					if imgui.MenuItemBool("Delete") {
						if copiedPath == browserPath+"/"+entry.Name {
							copiedPath = ""
						}

						sendCommand(Command{
							Target:  browserPath + "/" + entry.Name,
							Command: "rm",
						})
					}

					imgui.EndPopup()
				}

				if entry.Folder == false {
					imgui.SameLine()
					imgui.Text(BytesToReadable(int(entry.Size)))
				}

				imgui.Separator()
			}
		}
		imgui.End()
	}

	if setLayout {
		viewport := imgui.MainViewport().Size()

		imgui.SetWindowSizeStr("Connect", imgui.Vec2{300, 100})
		imgui.SetWindowPosStr("Connect", imgui.Vec2{viewport.X / 2, viewport.Y / 2})

		imgui.SetWindowCollapsedStr("Packet Debugger", true)

		setLayout = false
	}
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
