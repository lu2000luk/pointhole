package main

import (
	"io"
	"math/rand/v2"
	"os"
	"strconv"
	"strings"
)

func BytesToReadable(bytes int) string {
	if bytes < 1024 {
		return strconv.Itoa(bytes) + " B"
	} else if bytes < 1024*1024 {
		return strconv.FormatFloat(float64(bytes)/1024, 'f', 2, 64) + " KB"
	} else if bytes < 1024*1024*1024 {
		return strconv.FormatFloat(float64(bytes)/(1024*1024), 'f', 2, 64) + " MB"
	} else {
		return strconv.FormatFloat(float64(bytes)/(1024*1024*1024), 'f', 2, 64) + " GB"
	}
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

	n, err := file.ReadAt(content, start)
	if err != nil && err != io.EOF {
		return nil, err
	}

	return content[:n], nil
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

func GetKeyByValue(m map[string]string, value string) string {
	for k, v := range m {
		if v == value {
			return k
		}
	}
	return ""
}
