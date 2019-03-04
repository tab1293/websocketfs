package websocketfs

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo"
)

const WS_MESSAGE_TYPE_FILE_ANNOUNCE = "fileAnnounce"
const WS_MESSAGE_TYPE_READ_RESPONSE = "readResponse"
const WS_MESSAGE_TYPE_READ_REQUEST = "readRequest"

var upgrader = websocket.Upgrader{
	ReadBufferSize:  0,
	WriteBufferSize: 0,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type FileAnnounce struct {
	Size         int64  `json:"size"`
	Name         string `json:"name"`
	Mime         string `json:"mime"`
	LastModified int64  `json:"lastModified"`
}

type ReadResponse struct {
	Offset int64  `json:"offset"`
	Data   []byte `json:"data"`
	FileID string `json:"file_id"`
}

type ReadRequest struct {
	FileID string `json:"file_id"`
	Type   string `json:"type"`
	Length int64  `json:"length"`
	Offset int64  `json:"offset"`
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
		go fs.CopyFileToDisk(f)
	}

	return nil
}

func ReadResponseHandler(c echo.Context) error {
	conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return err
	}

	re := regexp.MustCompile("/readResponse/(.*)_(.*)$")
	matches := re.FindAllStringSubmatch(c.Request().URL.Path, -1)
	var fileID string
	var offset int64

	for _, match := range matches {
		fmt.Printf("key=%s, value=%s\n", match[1], match[2])
		fileID = match[1]
		offset, err = strconv.ParseInt(match[2], 10, 64)
		if err != nil {
			return err
		}
	}
	// matches := pat.FindAllStringSubmatch(data, -1) // matches is [][]string
	// var re = regexp.MustCompile(`(?m)/readResponse/(.*)_(\d*)$`)
	// for i, match := range re.FindAllStringSubmatch(c.Request().URL.Path, -1) {
	// log.Printf("found at index %s %d\n", match, i)
	// }

	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			log.Printf("websocket error: %v", err)
			break
		}

		// messageReader := bytes.NewReader(message)
		// readResponse := &ReadResponse{}
		// err = json.NewDecoder(messageReader).Decode(readResponse)
		// if err != nil {
		// 	log.Printf("decode error: %v", err)
		// 	break
		// }

		fs := c.Get("fs").(*FileSystem)
		f, err := fs.GetFile(fileID)
		if err != nil {
			log.Printf("%s\n", err)
			continue
		}

		// data, err := base64.StdEncoding.DecodeString(readResponse.Data)
		// if err != nil {
		// 	log.Printf("base64 error %s\n", err)
		// 	break
		// }

		log.Printf("reading %d data in to channel\n", offset)
		go func() {
			f.DataChans[offset] <- data
		}()

	}

	return nil
}
