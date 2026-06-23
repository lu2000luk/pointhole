package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand/v2"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/gorilla/websocket"
)

func copyFileOrDir(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}

	if info.IsDir() {
		return copyDir(src, dst)
	} else {
		return copyFile(src, dst)
	}
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}

	return out.Sync()
}

func copyDir(src, dst string) error {
	err := os.MkdirAll(dst, 0755)
	if err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			err = copyDir(srcPath, dstPath)
			if err != nil {
				return err
			}
		} else {
			err = copyFile(srcPath, dstPath)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func GetFileContentRange(filePath string, start, end int64) ([]byte, error) {
	if start > end {
		return nil, nil
	}

	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	length := end - start + 1
	content := make([]byte, length)

	_, err = file.ReadAt(content, start)
	if err != nil {
		return nil, err
	}

	return content, nil
}

func WriteFileContentRange(filePath string, start int64, content []byte) error {
	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteAt(content, start)
	if err != nil {
		return err
	}

	return nil
}

func GenerateRandomString(length int) string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	var sb strings.Builder
	sb.Grow(length)
	for i := 0; i < length; i++ {
		randomIndex := rand.IntN(len(charset))
		sb.WriteByte(charset[randomIndex])
	}
	return sb.String()
}

func MarshallAndSend(resp any, c *websocket.Conn, key string) {
	jsonresp, err := json.Marshal(resp)

	if err != nil {
		fmt.Println(prefix+"Error while marshalling response:", err)
	}
	if len(jsonresp) > 1000 {
		fmt.Println(prefix + "Response: " + string(jsonresp)[:1000] + "...")
	} else {
		fmt.Println(prefix + "Response: " + string(jsonresp))
	}

	encrypted, err := Encrypt(jsonresp, key)
	if err != nil {
		fmt.Println(prefix+"Error while encrypting response:", err)
	}

	err = c.WriteMessage(websocket.BinaryMessage, encrypted)
	if err != nil {
		log.Println(prefix+"Error while writing response:", err)
	}
}

func FixPathIfWindows(path string) string {
	if runtime.GOOS == "windows" {
		return "C:\\" + strings.ReplaceAll(path, "/", "\\")
	}
	return path
}
