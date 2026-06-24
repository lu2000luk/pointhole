package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"sort"
	"strings"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/AllenDang/cimgui-go/backend"
	"github.com/AllenDang/cimgui-go/backend/glfwbackend"
	"github.com/AllenDang/cimgui-go/imgui"
	_ "github.com/AllenDang/cimgui-go/impl/glfw"
	"github.com/gorilla/websocket"
	"github.com/sqweek/dialog"
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

var showUI bool = true

var copiedPath string = ""
var isCut bool = false

var emulatedFS = make(map[string][]LSResponseEntry)
var get_transfers = make(map[string]string)    // [id]:[path]
var upload_transfers = make(map[string]string) // [id]:[path]

type ReqResRandChunk struct {
	TransferId string             `json:"id"`
	Chunkrange TransferChunkRange `json:"r"`
}

var requested_random_chunk = []ReqResRandChunk{}
var requested_random_chunk_response = make(map[ReqResRandChunk]TransferChunk)

// transfers window
type OngoingTransfer struct {
	TransferId string
	Path       string
	Done       bool
	DoneBytes  int64
	TotalBytes int64
}

var ratransfers = []string{}                            // [path] (random access transfers, only added to the list when the transfer is getting an access, IDs arent used here as the access can be either get or upload)
var ongoingTransfers = make(map[string]OngoingTransfer) // [id]:[ongoingTransfer]
var showTransfersWindow bool = false
var showAboutWindow bool = false

// browser window

var calcH float32 = 0

var browserPath = "/"
var sentLSPacketFor = ""
var pathInput = browserPath

var isRenaming bool = false
var renamingPath string = ""
var renamingNewName string = ""

var isCreatingDir bool = false
var creatingDirPath string = ""
var creatingDirName string = ""

// packet debugger window
var commandName = ""
var commandTarget = ""
var commandDestination = ""
var showPacketDebugger bool = false
var showInfoMenu bool = false
var showTransferDebugger bool = false
var transferClientPath = ""
var transferServerPath = ""
var showRandomReadWindow bool = false
var randomReadPath = ""
var randomReadStart int32 = 0
var randomReadEnd int32 = 0
var randomReadContent atomic.Value // stores []byte

func loadConn() *websocket.Conn {
	return (*websocket.Conn)(atomic.LoadPointer(&c))
}

func storeConn(conn *websocket.Conn) {
	atomic.StorePointer(&c, unsafe.Pointer(conn))
}

func SendCommand(command Command) {
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

	if len(payload) < 512*1024 {
		fmt.Printf("Sending payload: %+v\n", command)
	} else {
		fmt.Printf("[LIMITED LOGGING] Sending payload of size %d bytes\n", len(payload))
	}

	payloadEncrypted, err := Encrypt(payload, pass)

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
			SendCommand(Command{
				Target:  browserPath,
				Command: "ls",
			})
		case "get":
			getTemplate := GETResponse{}
			_ = json.Unmarshal(decrypted, &getTemplate)
			get_transfers[getTemplate.TransferId] = getTemplate.Path

		case "getChunk":
			chunkData := TransferChunk{}
			_ = json.Unmarshal(decrypted, &chunkData)

			path := get_transfers[chunkData.TransferId]
			if path == "" {
				log.Println("No transfer found for id:", chunkData.TransferId)
				continue
			}

			matchedReq := FindMatchingRequest(chunkData.TransferId, chunkData.Chunkrange, requested_random_chunk)
			if matchedReq != nil {
				requested_random_chunk_response[*matchedReq] = chunkData

				for i, req := range requested_random_chunk {
					if req == *matchedReq {
						requested_random_chunk = append(requested_random_chunk[:i], requested_random_chunk[i+1:]...)
						break
					}
				}
			} else {
				log.Println("Random chunk not requested for id:", chunkData.TransferId)
			}

		case "upload":
			uploadTemplate := UploadResponse{}
			_ = json.Unmarshal(decrypted, &uploadTemplate)
			upload_transfers[uploadTemplate.TransferId] = uploadTemplate.Path

		case "uploadChunk":
			uploadChunkTemplate := UploadGetChunkResponse{}
			_ = json.Unmarshal(decrypted, &uploadChunkTemplate)
			log.Printf("Upload chunk response: %+v\n", uploadChunkTemplate)

		default:
			log.Println("Unknown type:", genericTemplate.Type)
		}

	}
}

func openFolder(entry LSResponseEntry) {
	if strings.HasSuffix(browserPath, "/") {
		browserPath = browserPath + entry.Name
		pathInput = browserPath
	} else {
		browserPath = browserPath + "/" + entry.Name
		pathInput = browserPath
	}
}

func loop() {
	imgui.ClearSizeCallbackPool()

	io := imgui.CurrentIO()
	if io.ConfigFlags()&imgui.ConfigFlagsViewportsEnable != 0 {
		io.SetConfigFlags(io.ConfigFlags() &^ imgui.ConfigFlagsViewportsEnable &^ imgui.ConfigFlagsDockingEnable)
	}

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

	imgui.PushStyleColorVec4(imgui.ColWindowBg, imgui.NewVec4(0, 0, 0, 0))
	imgui.PushStyleVarFloat(imgui.StyleVarWindowBorderSize, 0)
	if imgui.BeginV("Title Bar", nil, imgui.WindowFlagsMenuBar|imgui.WindowFlagsNoTitleBar|imgui.WindowFlagsNoResize|imgui.WindowFlagsNoMove) {
		if imgui.BeginMenuBar() {
			if imgui.BeginMenu("Pointhole") {
				if imgui.MenuItemBool("Show packet debugger") {
					showPacketDebugger = true
				}
				if imgui.MenuItemBool("Show info menu") {
					showInfoMenu = true
				}
				if imgui.MenuItemBool("Show transfer debugger") {
					showTransferDebugger = true
				}
				if imgui.MenuItemBool("Show random read window") {
					showRandomReadWindow = true
				}

				if imgui.MenuItemBool("About") {
					showAboutWindow = true
				}

				imgui.EndMenu()
			}

			if imgui.BeginMenu("Transfers") {
				if imgui.MenuItemBool("Toggle transfers window") {
					showTransfersWindow = !showTransfersWindow
				}

				imgui.EndMenu()
			}
			imgui.EndMenuBar()
		}
	}
	imgui.End()
	imgui.PopStyleColorV(1)
	imgui.PopStyleVarV(1)

	if isRenaming {
		if imgui.BeginV("Rename", nil, imgui.WindowFlagsNoResize|imgui.WindowFlagsNoCollapse|imgui.WindowFlagsAlwaysAutoResize|imgui.WindowFlagsNoMove) {
			imgui.InputTextWithHint("##newname", "New name...", &renamingNewName, imgui.InputTextFlagsEnterReturnsTrue, nil)
			if imgui.Button("Rename") {
				SendCommand(Command{
					Target:      renamingPath,
					Destination: browserPath + "/" + renamingNewName,
					Command:     "mv",
				})
				isRenaming = false
				showUI = true
			}
		}
		imgui.End()
	}

	if showAboutWindow {
		if imgui.BeginV("About", &showAboutWindow, imgui.WindowFlagsAlwaysAutoResize) {
			imgui.Text("Pointhole")
			imgui.Spacing()
			imgui.Text("Made with love by @lu2000luk")
			imgui.Spacing()
			if imgui.Button("Source code") {
				var cmd *exec.Cmd
				switch runtime.GOOS {
				case "windows":
					cmd = exec.Command("cmd.exe", "/c", "start", "https://git.lu2000luk.com/lu2000luk/pointhole")
				case "darwin":
					cmd = exec.Command("open", "https://git.lu2000luk.com/lu2000luk/pointhole")
				default:
					cmd = exec.Command("xdg-open", "https://git.lu2000luk.com/lu2000luk/pointhole")
				}
				if err := cmd.Start(); err != nil {
					log.Printf("Failed to open URL: %v", err)
				}
			}
		}
		imgui.End()
	}

	if isCreatingDir {
		if imgui.BeginV("Create Directory", nil, imgui.WindowFlagsNoResize|imgui.WindowFlagsNoCollapse|imgui.WindowFlagsAlwaysAutoResize|imgui.WindowFlagsNoMove) {
			imgui.InputTextWithHint("##dirname", "Directory name...", &creatingDirName, imgui.InputTextFlagsEnterReturnsTrue, nil)
			if imgui.Button("Create") {
				SendCommand(Command{
					Target:  browserPath + "/" + creatingDirName,
					Command: "mkdir",
				})
				isCreatingDir = false
				showUI = true
			}
			imgui.SameLine()
			if imgui.Button("Cancel") {
				isCreatingDir = false
				showUI = true
			}
		}
		imgui.End()
	}

	if connected && showUI {
		if showTransfersWindow {
			imgui.SetNextWindowSizeV(imgui.Vec2{200, 100}, imgui.CondAppearing)
			if imgui.BeginV("Transfers", &showTransfersWindow, imgui.WindowFlagsNone) {
				transferIDs := make([]string, 0, len(ongoingTransfers))
				for id := range ongoingTransfers {
					transferIDs = append(transferIDs, id)
				}
				sort.Strings(transferIDs)
				for _, id := range transferIDs {
					transfer := ongoingTransfers[id]
					imgui.PushIDStr(id)
					if transfer.Done {
						imgui.Spacing()
						imgui.Text(fmt.Sprintf("%s | DONE", id))
					} else {
						imgui.Spacing()
						imgui.ProgressBarV(float32(transfer.DoneBytes)/float32(transfer.TotalBytes), imgui.Vec2{0, 0}, fmt.Sprintf("%s/%s", BytesToReadable(int(transfer.DoneBytes)), BytesToReadable(int(transfer.TotalBytes))))
					}
					imgui.PopID()
				}

				if len(ongoingTransfers) == 0 {
					imgui.Text("No ongoing transfers")
				}
			}
			imgui.End()
		}

		if showRandomReadWindow {
			if imgui.BeginV("Random Read Window", &showRandomReadWindow, imgui.WindowFlagsNone) {
				imgui.InputTextWithHint("##randomReadPath", "Path...", &randomReadPath, 0, nil)
				imgui.InputInt("Start", &randomReadStart)
				imgui.InputInt("End", &randomReadEnd)

				if imgui.Button("Request Random Read") {
					go func() {
						log.Printf("Requesting random read for %s from %d to %d\n", randomReadPath, randomReadStart, randomReadEnd)
						res, err := RandomRead(randomReadPath, int64(randomReadStart), int64(randomReadEnd), &get_transfers, &requested_random_chunk, &requested_random_chunk_response)
						if err != nil {
							log.Printf("Error requesting random read: %v\n", err)
							return
						}
						randomReadContent.Store(res)
					}()
				}

				if val := randomReadContent.Load(); val != nil {
					content := val.([]byte)
					imgui.Text(fmt.Sprintf("Received content (%d bytes):", len(content)))
					imgui.Text(string(content))
				}
			}
			imgui.End()
		}

		if showTransferDebugger {
			if imgui.BeginV("TransferDebugger", &showTransferDebugger, imgui.WindowFlagsNone) {
				imgui.InputTextWithHint("##debuggerclientpath", "Client path...", &transferClientPath, 0, nil)
				imgui.InputTextWithHint("##debuggerserverpath", "Server path...", &transferServerPath, 0, nil)
				if imgui.Button("Upload") {
					go func() {
						err := UploadFile(transferClientPath, transferServerPath, &upload_transfers, &ongoingTransfers)
						if err != nil {
							fmt.Printf("Error uploading file: %v\n", err)
						}
					}()
				}

				debugIDs := make([]string, 0, len(ongoingTransfers))
				for id := range ongoingTransfers {
					debugIDs = append(debugIDs, id)
				}
				sort.Strings(debugIDs)
				for _, id := range debugIDs {
					transfer := ongoingTransfers[id]
					imgui.Separator()
					imgui.Text(fmt.Sprintf("%s | Done: %v | %d/%d", id, transfer.Done, transfer.DoneBytes, transfer.TotalBytes))
				}
			}
			imgui.End()
		}

		if showPacketDebugger {
			if imgui.BeginV("Packet Debugger", &showPacketDebugger, imgui.WindowFlagsNone) {
				imgui.InputTextWithHint("##debuggercommand", "Command name...", &commandName, 0, nil)
				imgui.InputTextWithHint("##debuggertarget", "Command target...", &commandTarget, 0, nil)
				imgui.InputTextWithHint("##debuggerdestination", "Command destination...", &commandDestination, 0, nil)
				if imgui.Button("Send") {
					SendCommand(Command{
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

		if showInfoMenu {
			if imgui.BeginV("Info Menu", &showInfoMenu, imgui.WindowFlagsNone) {
				imgui.Text("ID: " + id)
				if reconnecting {
					imgui.Text("Status: Reconnecting...")
				} else if connected {
					imgui.Text("Status: Connected")
				} else {
					imgui.Text("Status: Disconnected")
				}

				imgui.Text(fmt.Sprintf("ShowInfoMenu: %v", showInfoMenu))
				imgui.Text(fmt.Sprintf("ShowPacketDebugger: %v", showPacketDebugger))

				RenderIconButton("##info_refreshicon", DrawIconRefresh, 32, 30)
				imgui.SameLine()
				RenderIconButton("##info_uploadicon", DrawIconUpload, 32, 30)

				RenderIconButton("##info_lefticon", DrawIconChevronLeft, 32, 30)
				imgui.SameLine()
				RenderIconButton("##info_createfoldericon", DrawIconFolderPlus, 32, 30)
			}
			imgui.End()
		}

		imgui.SetNextWindowSizeV(imgui.Vec2{800, 400}, imgui.CondAppearing)
		if imgui.Begin("Browser") {

			if browserPath != sentLSPacketFor {
				SendCommand(Command{
					Target:  browserPath,
					Command: "ls",
				})

				sentLSPacketFor = browserPath
			}

			if RenderIconButton("##backicon", DrawIconChevronLeft, calcH, calcH-2) {
				browserPath = browserPath[0:strings.LastIndex(browserPath, "/")]
				if browserPath == "" {
					browserPath = "/"
				}

				pathInput = browserPath
			}

			imgui.SameLine()

			if imgui.InputTextWithHint("##pathInput", "/", &pathInput, imgui.InputTextFlagsEnterReturnsTrue, nil) {
				if pathInput == "" {
					pathInput = "/"
				}

				fixedPath := pathInput
				fixedPath = strings.Replace(fixedPath, "C:\\", "/", -1) // windows compat
				fixedPath = strings.Replace(fixedPath, "\\", "/", -1)
				if len(fixedPath) > 1 {
					fixedPath = strings.TrimRight(fixedPath, "/")
				}
				browserPath = fixedPath
				pathInput = fixedPath
			}

			imgui.SameLine()

			if RenderIconButton("##refreshicon", DrawIconRefresh, calcH, calcH-4) {
				SendCommand(Command{
					Target:  browserPath,
					Command: "ls",
				})
			}

			imgui.SameLine()

			if RenderIconButton("##uploadicon", DrawIconUpload, calcH, calcH-4) {
				showTransfersWindow = true
				filename, err := dialog.File().Title("Select file to upload").Load()
				if err != nil {
					log.Printf("Error occurred while selecting file: %v", err)
				} else {
					info, err := os.Stat(filename)
					if err != nil {
						log.Printf("Error occurred while getting file info: %v", err)
						return
					}
					if info.IsDir() {
						log.Printf("Selected path is a directory, only files are allowed: %s", filename)
						return
					}

					if info.Size() > 500*1024*1024 { // 500MB
						dialog.Message("No.").Title("File too large").Info()
						log.Printf("Selected file is too large (%s), max allowed size is 500 MB", BytesToReadable(int(info.Size())))
						return
					}

					serverPath := strings.ReplaceAll(browserPath, "\\", "/") + "/" + filename[strings.LastIndex(strings.ReplaceAll(filename, "\\", "/"), "/")+1:]
					go func() {
						err = UploadFile(filename, serverPath, &upload_transfers, &ongoingTransfers)
						if err != nil {
							log.Printf("Error uploading file: %v", err)
						}
					}()
				}
			}

			imgui.SameLine()

			if RenderIconButton("##createfoldericon", DrawIconFolderPlus, calcH, calcH-2) {
				isCreatingDir = true
				creatingDirPath = browserPath
				creatingDirName = ""
				showUI = false
			}

			if copiedPath != "" {
				imgui.SameLine()
				if imgui.Button("Paste") {
					if isCut {
						SendCommand(Command{
							Target:      copiedPath,
							Destination: browserPath + "/" + copiedPath[strings.LastIndex(copiedPath, "/")+1:],
							Command:     "mv",
						})
						copiedPath = browserPath + "/" + copiedPath[strings.LastIndex(copiedPath, "/")+1:]
						isCut = false
					} else {
						SendCommand(Command{
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
					if RenderIconButton("##foldericon_"+entry.Name, DrawIconFolder, calcH, calcH-3) {
						openFolder(entry)
					}
					imgui.SameLine()
					if imgui.Button(entry.Name) {
						openFolder(entry)
					}
				} else {
					imgui.Button(entry.Name)
				}

				if imgui.BeginPopupContextItem() {
					if imgui.MenuItemBool("Copy") {
						copiedPath = strings.ReplaceAll(browserPath, "\\", "/") + "/" + entry.Name
						isCut = false
					}
					if imgui.MenuItemBool("Cut") {
						copiedPath = strings.ReplaceAll(browserPath, "\\", "/") + "/" + entry.Name
						isCut = true
					}
					if imgui.MenuItemBool("Delete") {
						if copiedPath == strings.ReplaceAll(browserPath, "\\", "/")+"/"+entry.Name {
							copiedPath = ""
						}

						SendCommand(Command{
							Target:  strings.ReplaceAll(browserPath, "\\", "/") + "/" + entry.Name,
							Command: "rm",
						})
					}

					if imgui.MenuItemBool("Rename") {
						isRenaming = true
						renamingPath = strings.ReplaceAll(browserPath, "\\", "/") + "/" + entry.Name
						renamingNewName = entry.Name
						showUI = false
					}

					if entry.Folder == false {
						if imgui.MenuItemBool("Edit") {
							serverPath := strings.ReplaceAll(browserPath, "\\", "/") + "/" + entry.Name
							size := entry.Size
							go func() {
								log.Printf("Opening file in editor: %s\n", serverPath)
								OpenInEditor(serverPath, size, &upload_transfers, &ongoingTransfers, &get_transfers, &requested_random_chunk, &requested_random_chunk_response)
							}()
						}

						if imgui.MenuItemBool("Download") {
							showTransfersWindow = true
							serverPath := strings.ReplaceAll(browserPath, "\\", "/") + "/" + entry.Name
							size := entry.Size

							filename, err := dialog.File().Title("Select download location").Save()
							if err != nil {
								log.Printf("Error occurred while selecting download location: %v", err)
							}

							go func() {
								log.Printf("Downloading file: %s to %s\n", serverPath, filename)
								err := DownloadFile(serverPath, filename, size, &get_transfers, &requested_random_chunk, &requested_random_chunk_response)
								if err != nil {
									log.Printf("Error downloading file: %v\n", err)
								}
							}()
						}
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

		imgui.SetWindowSizeStr("Title Bar", imgui.Vec2{viewport.X, 0})

		calcH = imgui.GetFontBaked().Size() + (imgui.CurrentStyle().FramePadding().Y * 2)

		pushStyles()

		setLayout = false
	}

	imgui.SetWindowPosStr("Title Bar", imgui.Vec2{0, 0})
}

func pushStyles() {
	style := imgui.CurrentStyle()

	style.SetAlpha(1.0)
	style.SetDisabledAlpha(0.6)
	style.SetWindowPadding(imgui.NewVec2(8, 8))
	style.SetWindowRounding(0.0)
	style.SetWindowBorderSize(1.0)
	style.SetWindowMinSize(imgui.NewVec2(32, 32))
	style.SetWindowTitleAlign(imgui.NewVec2(0.0, 0.5))
	style.SetWindowMenuButtonPosition(imgui.DirLeft)
	style.SetChildRounding(0.0)
	style.SetChildBorderSize(1.0)
	style.SetPopupRounding(0.0)
	style.SetPopupBorderSize(1.0)
	style.SetFramePadding(imgui.NewVec2(4, 3))
	style.SetFrameRounding(0.0)
	style.SetFrameBorderSize(0.0)
	style.SetItemSpacing(imgui.NewVec2(8, 4))
	style.SetItemInnerSpacing(imgui.NewVec2(4, 4))
	style.SetCellPadding(imgui.NewVec2(4, 2))
	style.SetIndentSpacing(21.0)
	style.SetColumnsMinSpacing(6.0)
	style.SetScrollbarSize(14.0)
	style.SetScrollbarRounding(0.0)
	style.SetGrabMinSize(10.0)
	style.SetGrabRounding(0.0)
	style.SetTabRounding(0.0)
	style.SetTabBorderSize(0.0)
	style.SetTabCloseButtonMinWidthSelected(0.0)
	style.SetTabCloseButtonMinWidthUnselected(0.0)
	style.SetColorButtonPosition(imgui.DirRight)
	style.SetButtonTextAlign(imgui.NewVec2(0.5, 0.5))
	style.SetSelectableTextAlign(imgui.NewVec2(0.0, 0.0))

	colors := style.Colors()
	colors[imgui.ColText] = imgui.NewVec4(1.000, 1.000, 1.000, 1.000)
	colors[imgui.ColTextDisabled] = imgui.NewVec4(0.592, 0.592, 0.592, 1.000)
	colors[imgui.ColWindowBg] = imgui.NewVec4(0.145, 0.145, 0.149, 1.000)
	colors[imgui.ColChildBg] = imgui.NewVec4(0.145, 0.145, 0.149, 1.000)
	colors[imgui.ColPopupBg] = imgui.NewVec4(0.145, 0.145, 0.149, 1.000)
	colors[imgui.ColBorder] = imgui.NewVec4(0.306, 0.306, 0.306, 1.000)
	colors[imgui.ColBorderShadow] = imgui.NewVec4(0.306, 0.306, 0.306, 1.000)
	colors[imgui.ColFrameBg] = imgui.NewVec4(0.200, 0.200, 0.216, 1.000)
	colors[imgui.ColFrameBgHovered] = imgui.NewVec4(0.114, 0.592, 0.925, 1.000)
	colors[imgui.ColFrameBgActive] = imgui.NewVec4(0.000, 0.467, 0.784, 1.000)
	colors[imgui.ColTitleBg] = imgui.NewVec4(0.145, 0.145, 0.149, 1.000)
	colors[imgui.ColTitleBgActive] = imgui.NewVec4(0.145, 0.145, 0.149, 1.000)
	colors[imgui.ColTitleBgCollapsed] = imgui.NewVec4(0.145, 0.145, 0.149, 1.000)
	colors[imgui.ColMenuBarBg] = imgui.NewVec4(0.200, 0.200, 0.216, 1.000)
	colors[imgui.ColScrollbarBg] = imgui.NewVec4(0.200, 0.200, 0.216, 1.000)
	colors[imgui.ColScrollbarGrab] = imgui.NewVec4(0.322, 0.322, 0.333, 1.000)
	colors[imgui.ColScrollbarGrabHovered] = imgui.NewVec4(0.353, 0.353, 0.373, 1.000)
	colors[imgui.ColScrollbarGrabActive] = imgui.NewVec4(0.353, 0.353, 0.373, 1.000)
	colors[imgui.ColCheckMark] = imgui.NewVec4(0.000, 0.467, 0.784, 1.000)
	colors[imgui.ColSliderGrab] = imgui.NewVec4(0.114, 0.592, 0.925, 1.000)
	colors[imgui.ColSliderGrabActive] = imgui.NewVec4(0.000, 0.467, 0.784, 1.000)
	colors[imgui.ColButton] = imgui.NewVec4(0.200, 0.200, 0.216, 1.000)
	colors[imgui.ColButtonHovered] = imgui.NewVec4(0.114, 0.592, 0.925, 1.000)
	colors[imgui.ColButtonActive] = imgui.NewVec4(0.114, 0.592, 0.925, 1.000)
	colors[imgui.ColHeader] = imgui.NewVec4(0.200, 0.200, 0.216, 1.000)
	colors[imgui.ColHeaderHovered] = imgui.NewVec4(0.114, 0.592, 0.925, 1.000)
	colors[imgui.ColHeaderActive] = imgui.NewVec4(0.000, 0.467, 0.784, 1.000)
	colors[imgui.ColSeparator] = imgui.NewVec4(0.306, 0.306, 0.306, 1.000)
	colors[imgui.ColSeparatorHovered] = imgui.NewVec4(0.306, 0.306, 0.306, 1.000)
	colors[imgui.ColSeparatorActive] = imgui.NewVec4(0.306, 0.306, 0.306, 1.000)
	colors[imgui.ColResizeGrip] = imgui.NewVec4(0.145, 0.145, 0.149, 1.000)
	colors[imgui.ColResizeGripHovered] = imgui.NewVec4(0.200, 0.200, 0.216, 1.000)
	colors[imgui.ColResizeGripActive] = imgui.NewVec4(0.322, 0.322, 0.333, 1.000)
	colors[imgui.ColTab] = imgui.NewVec4(0.145, 0.145, 0.149, 1.000)
	colors[imgui.ColTabHovered] = imgui.NewVec4(0.114, 0.592, 0.925, 1.000)
	colors[imgui.ColPlotLines] = imgui.NewVec4(0.000, 0.467, 0.784, 1.000)
	colors[imgui.ColPlotLinesHovered] = imgui.NewVec4(0.114, 0.592, 0.925, 1.000)
	colors[imgui.ColPlotHistogram] = imgui.NewVec4(0.000, 0.467, 0.784, 1.000)
	colors[imgui.ColPlotHistogramHovered] = imgui.NewVec4(0.114, 0.592, 0.925, 1.000)
	colors[imgui.ColTableHeaderBg] = imgui.NewVec4(0.188, 0.188, 0.200, 1.000)
	colors[imgui.ColTableBorderStrong] = imgui.NewVec4(0.310, 0.310, 0.349, 1.000)
	colors[imgui.ColTableBorderLight] = imgui.NewVec4(0.227, 0.227, 0.247, 1.000)
	colors[imgui.ColTableRowBg] = imgui.NewVec4(0.000, 0.000, 0.000, 0.000)
	colors[imgui.ColTableRowBgAlt] = imgui.NewVec4(1.000, 1.000, 1.000, 0.060)
	colors[imgui.ColTextSelectedBg] = imgui.NewVec4(0.000, 0.467, 0.784, 1.000)
	colors[imgui.ColDragDropTarget] = imgui.NewVec4(0.145, 0.145, 0.149, 1.000)
	colors[imgui.ColNavCursor] = imgui.NewVec4(0.145, 0.145, 0.149, 1.000)
	colors[imgui.ColNavWindowingHighlight] = imgui.NewVec4(1.000, 1.000, 1.000, 0.700)
	colors[imgui.ColNavWindowingDimBg] = imgui.NewVec4(0.800, 0.800, 0.800, 0.200)
	colors[imgui.ColModalWindowDimBg] = imgui.NewVec4(0.145, 0.145, 0.149, 1.000)
	style.SetColors(&colors)
}

func init() { // this runs before main??? go magic...
	runtime.LockOSThread()
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

	currentBackend.SetBgColor(imgui.NewVec4(0, 0, 0, 1))
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

	currentBackend.SetTargetFPS(60)
	currentBackend.Run(loop)
}
