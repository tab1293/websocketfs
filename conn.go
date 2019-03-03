package websocketfs

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  0,
	WriteBufferSize: 0,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type WebSocketMessage struct {
	Type string `json:"type"`
}

func FileAnnounceHandler(c echo.Context) error {
	conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return err
	}

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("websocket error: %v", err)
			break
		}

		messageReader := bytes.NewReader(message)
		fileAnnounce := &FileAnnounce{}
		err = json.NewDecoder(messageReader).Decode(fileAnnounce)
		if err != nil {
			log.Printf("decode error: %v", err)
			break
		}

		log.Printf("fileAnnounce %s %d", fileAnnounce.Name, fileAnnounce.Size)
		f := NewFile(fileAnnounce.Name, fileAnnounce.Size, conn)
		fs := c.Get("fs").(*FileSystem)
		fs.AddFile(f)
		go CopyFileToDisk(f)
	}

	return nil
}

func ReadResponseHandler(c echo.Context) error {
	conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return err
	}

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("websocket error: %v", err)
			break
		}

		messageReader := bytes.NewReader(message)
		log.Printf("decoding read response\n")
		readResponse := &ReadResponse{}
		err = json.NewDecoder(messageReader).Decode(readResponse)
		if err != nil {
			log.Printf("decode error: %v", err)
			break
		}

		fs := c.Get("fs").(*FileSystem)
		f, err := fs.GetFile(readResponse.FileID)
		if err != nil {
			log.Printf("%s\n", err)
			continue
		}

		log.Printf("decoding read string\n")
		data, err := base64.StdEncoding.DecodeString(readResponse.Data)
		if err != nil {
			log.Printf("base64 error %s\n", err)
			break
		}

		log.Printf("reading data in to channel\n")
		go func() {
			f.DataChans[readResponse.Offset] <- data
		}()

	}

	return nil
}

func EchoWebsocketHandler(c echo.Context) error {
	conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return err
	}

	go readLoop(conn)

	return nil
}

const WS_MESSAGE_TYPE_FILE_ANNOUNCE = "fileAnnounce"
const WS_MESSAGE_TYPE_CLIENT_ANNOUNCE = "clientAnnounce"
const WS_MESSAGE_TYPE_READ_RESPONSE = "readResponse"
const WS_MESSAGE_TYPE_READ_REQUEST = "readRequest"

type FileAnnounce struct {
	Size         int64  `json:"size"`
	Name         string `json:"name"`
	Mime         string `json:"mime"`
	LastModified int64  `json:"lastModified"`
}

type ReadResponse struct {
	Offset int64  `json:"offset"`
	Data   string `json:"data"`
	FileID string `json:"file_id"`
}

type ReadRequest struct {
	FileID string `json:"file_id"`
	Type   string `json:"type"`
	Length int64  `json:"length"`
	Offset int64  `json:"offset"`
}

func readLoop(conn *websocket.Conn) {
	var f *File

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("websocket error: %v", err)
			break
		}

		messageReader := bytes.NewReader(message)
		wsMessage := &WebSocketMessage{}
		err = json.NewDecoder(messageReader).Decode(wsMessage)
		if err != nil {
			log.Printf("decode error: %v", err)
			break
		}

		messageReader.Seek(0, io.SeekStart)
		switch wsMessage.Type {
		case WS_MESSAGE_TYPE_FILE_ANNOUNCE:
			fileAnnounce := &FileAnnounce{}
			err = json.NewDecoder(messageReader).Decode(fileAnnounce)
			if err != nil {
				log.Printf("decode error: %v", err)
				break
			}

			log.Printf("fileAnnounce %s %d", fileAnnounce.Name, fileAnnounce.Size)
			f = NewFile(fileAnnounce.Name, fileAnnounce.Size, conn)
			go CopyFileToDisk(f)

		case WS_MESSAGE_TYPE_READ_RESPONSE:
			if f == nil {
				continue
			}

			log.Printf("decoding read response\n")
			readResponse := &ReadResponse{}
			err = json.NewDecoder(messageReader).Decode(readResponse)
			if err != nil {
				log.Printf("decode error: %v", err)
				break
			}

			log.Printf("decoding read string\n")
			data, err := base64.StdEncoding.DecodeString(readResponse.Data)
			if err != nil {
				log.Printf("base64 error %s\n", err)
				break
			}

			log.Printf("reading data in to channel\n")
			go func() {
				f.DataChans[readResponse.Offset] <- data
			}()

		}
	}
}

func CopyFileToDisk(f *File) error {
	log.Printf("creating file %s\n", f.Name)
	df, err := os.Create(f.Name)
	if err != nil {
		return err
	}
	defer df.Close()

	var off int64 = 0
	start := time.Now()
	numParts := 20

	var wg sync.WaitGroup
	for i := 0; i < numParts; i++ {
		partSize := f.Size / int64(numParts)
		if i+1 == numParts {
			partSize = f.Size - off
		}

		wg.Add(1)
		go func(offset int64, length int64) error {
			defer wg.Done()

			log.Printf("reading at %d-%d size %d\n", offset, offset+length, length)
			b := make([]byte, length)
			n, err := f.ReadAt(b, offset)
			if err != nil {
				log.Printf("error reading at %d %s\n", offset, err)
				return err
			}

			log.Printf("read %d bytes\n", n)

			n, err = df.WriteAt(b, offset)
			if err != nil {
				log.Printf("error writing at %d %s\n", offset, err)
				return err
			}
			log.Printf("wrote %d bytes\n", n)
			return nil
		}(off, partSize)

		off = off + partSize
	}

	wg.Wait()
	elapsed := time.Since(start)
	log.Printf("copied file %s in %s\n", f.Name, elapsed)

	return nil

}
