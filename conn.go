package websocketfs

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"

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

func EchoWebsocketHandler(c echo.Context) error {
	conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return err
	}

	go readLoop(conn)

	return nil
}

type WebSocketMessage struct {
	Type string `json:"type"`
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
}

type ReadRequest struct {
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

			readResponse := &ReadResponse{}
			err = json.NewDecoder(messageReader).Decode(readResponse)
			if err != nil {
				log.Printf("decode error: %v", err)
				break
			}

			data, err := base64.StdEncoding.DecodeString(readResponse.Data)
			if err != nil {
				log.Printf("base64 error %s\n", err)
				break
			}

			f.DataChans[readResponse.Offset] <- data
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
	numParts := 3

	for i := 0; i < numParts; i++ {
		partSize := f.Size / int64(numParts)
		if i+1 == numParts {
			partSize = f.Size - off
		}

		log.Printf("reading at %d-%d size %d\n", off, off+partSize, partSize)
		b := make([]byte, partSize)
		n, err := f.ReadAt(b, off)
		if err != nil {
			log.Printf("error reading at %d %s\n", off, err)
			return err
		}

		log.Printf("read %d bytes\n", n)

		n, err = df.WriteAt(b, off)
		if err != nil {
			log.Printf("error writing at %d %s\n", off, err)
			return err
		}
		log.Printf("wrote %d bytes\n", n)
		off = off + partSize
	}

	return nil

}
