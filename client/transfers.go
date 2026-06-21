package main

import (
	"fmt"
	"log"
	"os"
	"time"
)

const chunkSize = 1024 * 1024 // 1 MB
const chunkInterval = 500     // milliseconds

func UploadFile(clientPath string, serverPath string, uploadTransfers *map[string]string, windowTransfers *map[string]OngoingTransfer) error {
	data, err := os.Stat(clientPath)
	if err != nil {
		return err
	}

	if data.IsDir() {
		return fmt.Errorf("Uploading directories is not supported")
	}

	log.Printf("Starting upload of %s to %s", clientPath, serverPath)

	go func() {
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
					return
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
					return
				}
			}
		}
	}()
	return nil
}
