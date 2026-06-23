package main

import (
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
)

const blacklisted_editor_extensions = "exe mp3 mp4 avi mkv mov flv wmv mpeg mpg wav aac ogg flac webm m4a m4v 3gp 3g2 swf iso bin img iso dmg vhd vmdk qcow2 ova"

func getEditor() (string, string) {
	e := os.Getenv("EDITOR")
	if e == "" || strings.Contains(e, "vim") || strings.Contains(e, "nano") {
		return "code", "--wait"
	}

	return e, "--wait"
}

func OpenInEditor(serverPath string, size int64, RRuploadTransfers *map[string]string, RRwindowTransfers *map[string]OngoingTransfer, RRgetTransfers *map[string]string, RRrequestedChunks *[]ReqResRandChunk, RRrequestedChunksResponse *map[ReqResRandChunk]TransferChunk) {
	if strings.Contains(blacklisted_editor_extensions, strings.ToLower(serverPath[strings.LastIndex(serverPath, ".")+1:])) {
		log.Printf("File type is blacklisted for editing: %s\n", serverPath)
		return
	}

	editor, editorArgs := getEditor()
	tempFilePath := "temp_" + GenerateRandomString(5) + "_" + serverPath[strings.LastIndex(serverPath, "/")+1:]

	err := DownloadFile(serverPath, tempFilePath, size, RRgetTransfers, RRrequestedChunks, RRrequestedChunksResponse)
	if err != nil {
		log.Printf("Error downloading file: %v\n", err)
		return
	}

	go func() {
		log.Printf("Opening file with: %s\n", editor)
		cmd := exec.Command(editor, editorArgs, tempFilePath)

		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			log.Printf("Error running editor command: %v\n", err)
			return
		}

		log.Printf("File closed, uploading changes back to server: %s\n", serverPath)

		// Remove existing server file to prevent edits to stack like this [edit 6 bytes][original files last 2 bytes] when original is 8 bytes
		SendCommand(Command{
			Command: "rm",
			Target:  serverPath,
		})
		time.Sleep(200 * time.Millisecond) // temporary fix
		err = UploadFile(tempFilePath, serverPath, RRuploadTransfers, RRwindowTransfers)
		if err != nil {
			log.Printf("Error uploading file: %v\n", err)
			return
		}

		time.Sleep(2 * time.Second) // temporary fix
		err = os.Remove(tempFilePath)
		if err != nil {
			log.Printf("Error removing temporary file: %v\n", err)
			return
		}
	}()
}
