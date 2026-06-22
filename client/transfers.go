package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

const chunkSize = 0.5 * 1024 * 1024 // 0.5 MB
const chunkInterval = 50            // milliseconds

func UploadFile(clientPath string, serverPath string, uploadTransfers *map[string]string, windowTransfers *map[string]OngoingTransfer) error {
	data, err := os.Stat(clientPath)
	if err != nil {
		return err
	}

	if data.IsDir() {
		return fmt.Errorf("Uploading directories is not supported")
	}

	log.Printf("Starting upload of %s to %s", clientPath, serverPath)

	ticker := time.NewTicker(chunkInterval * time.Millisecond)
	defer ticker.Stop()

	hasGeneratedId := false
	generatedId := ""
	sentIDRequest := false

	for {
		select {
		case <-ticker.C:
			log.Printf("Uploading %s...\n", serverPath)
			if !hasGeneratedId {
				if GetKeyByValue(*uploadTransfers, serverPath) != "" {
					generatedId = GetKeyByValue(*uploadTransfers, serverPath)
					hasGeneratedId = true

					(*windowTransfers)[generatedId] = OngoingTransfer{
						Path:       serverPath,
						TransferId: generatedId,
						Done:       false,
						TotalBytes: data.Size(),
						DoneBytes:  0,
					}
				} else {
					if !sentIDRequest {
						SendCommand(Command{
							Command: "upload",
							Target:  serverPath,
						})
						sentIDRequest = true
					}
					continue
				}
			}

			lastDoneBytes := (*windowTransfers)[generatedId].DoneBytes

			chunk, err := GetFileContentRange(clientPath, lastDoneBytes, lastDoneBytes+int64(chunkSize)-1)
			if err != nil {
				log.Printf("Error reading file chunk: %v\n", err)
				return err
			}

			resp := Command{
				Command: "uploadChunk",
				Target:  serverPath,
				UploadData: TransferChunk{
					TransferId: generatedId,
					Type:       "uploadChunk",
					Content:    chunk,
					IsEnd:      len(chunk) < chunkSize,
					Chunkrange: TransferChunkRange{
						RangeStart: lastDoneBytes,
						RangeEnd:   lastDoneBytes + int64(len(chunk)) - 1,
					},
				},
			}

			SendCommand(resp)

			(*windowTransfers)[generatedId] = OngoingTransfer{
				Path:       serverPath,
				TransferId: generatedId,
				Done:       len(chunk) < chunkSize,
				TotalBytes: data.Size(),
				DoneBytes:  lastDoneBytes + int64(len(chunk)),
			}

			if len(chunk) < chunkSize {
				go func() {
					time.Sleep(1 * time.Second)
					SendCommand(Command{
						Command: "ls",
						Target:  serverPath[:strings.LastIndex(serverPath, "/")],
					})
				}()
				return nil
			}
		}
	}

	return nil
}

func WaitAndRetryForID(path string, transfers *map[string]string, failures int) error {
	if failures > 40 {
		log.Printf("Failed to get transfer ID for %s after 40 attempts\n", path)
		return fmt.Errorf("failed to get transfer ID for %s", path)
	}
	time.Sleep(200 * time.Millisecond)
	if GetKeyByValue((*transfers), path) == "" {
		return WaitAndRetryForID(path, transfers, failures+1)
	}

	return nil
}

func WaitAndRetryForRandomChunk(req ReqResRandChunk, requestedChunksResponse *map[ReqResRandChunk]TransferChunk, failures int) error {
	if failures > 40 {
		log.Printf("Failed to get transfer ID for %s after 40 attempts\n", req.TransferId)
		return fmt.Errorf("failed to get transfer ID for %s", req.TransferId)
	}
	time.Sleep(200 * time.Millisecond)

	if _, exists := (*requestedChunksResponse)[req]; !exists {
		return WaitAndRetryForRandomChunk(req, requestedChunksResponse, failures+1)
	}

	return nil
}

func RandomRead(serverPath string, start, end int64, getTransfers *map[string]string, requestedChunks *[]ReqResRandChunk, requestedChunksResponse *map[ReqResRandChunk]TransferChunk) ([]byte, error) {
	if start > end {
		return nil, fmt.Errorf("Invalid range: start (%d) is greater than end (%d)", start, end)
	}

	if end-start+1 > 8*1024*1024 { // server limit is 10mb but we set a lower limit to avoid issues
		return nil, fmt.Errorf("Requested range exceeds maximum chunk size of %d bytes", 8*1024*1024)
	}

	id := GetKeyByValue((*getTransfers), serverPath)
	if id == "" {
		SendCommand(Command{
			Command: "get",
			Target:  serverPath,
		})

		err := WaitAndRetryForID(serverPath, getTransfers, 0)
		if err != nil {
			return nil, err
		}
	}

	id = GetKeyByValue((*getTransfers), serverPath)

	command := Command{
		Command: "getChunk",
		Target:  id,
		GetChunkRange: TransferChunkRange{
			RangeStart: start,
			RangeEnd:   end,
		},
	}

	req := ReqResRandChunk{
		TransferId: id,
		Chunkrange: TransferChunkRange{
			RangeStart: start,
			RangeEnd:   end,
		},
	}

	(*requestedChunks) = append((*requestedChunks), req)

	SendCommand(command)

	err := WaitAndRetryForRandomChunk(req, requestedChunksResponse, 0)
	if err != nil {
		return nil, err
	}

	data := (*requestedChunksResponse)[req]
	delete(*requestedChunksResponse, req)

	return data.Content, nil
}

func DownloadFile(serverPath string, localPath string, size int64, getTransfers *map[string]string, requestedChunks *[]ReqResRandChunk, requestedChunksResponse *map[ReqResRandChunk]TransferChunk) error {
	var offset int64 = 0
	for offset < size {
		time.Sleep(80 * time.Millisecond)
		end := offset + chunkSize - 1
		if end >= size {
			end = size - 1
		}

		log.Printf("Requesting chunk %d-%d of %s\n", offset, end, serverPath)
		chunkData, err := RandomRead(serverPath, offset, end, getTransfers, requestedChunks, requestedChunksResponse)
		if err != nil {
			return err
		}

		f, err := os.OpenFile(localPath, os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}

		_, err = f.WriteAt(chunkData, offset)
		if err != nil {
			f.Close()
			return err
		}
		f.Close()

		offset += int64(len(chunkData))
	}

	log.Printf("Download of %s completed successfully\n", serverPath)

	return nil
}
