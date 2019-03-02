package websocketfs

import (
	"fmt"
	"io"

	"github.com/gorilla/websocket"
	uuid "github.com/satori/go.uuid"
)

type File struct {
	ID           string
	Name         string
	Size         int64
	Mime         string
	LastModified uint64
	DataChans    map[int64]chan []byte
	conn         *websocket.Conn
	off          int64
}

func (f *File) Read(p []byte) (n int, err error) {
	if f.off >= f.Size {
		return 0, io.EOF
	}

	req := &ReadRequest{
		Type:   WS_MESSAGE_TYPE_READ_REQUEST,
		Length: int64(len(p)),
		Offset: f.off,
	}

	err = f.conn.WriteJSON(req)
	if err != nil {
		return n, err
	}

	f.DataChans[f.off] = make(chan []byte)
	data := <-f.DataChans[f.off]
	n = copy(p, data)
	f.off = f.off + int64(n)
	return n, err
}

func (f *File) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
		f.off = offset
	case io.SeekCurrent:
		f.off = f.off + offset
	case io.SeekEnd:
		f.off = f.Size - offset
	}

	if f.off < 0 {
		return 0, fmt.Errorf("Can't seek before beginning of file")
	}

	return f.off, nil
}

func (f *File) ReadAt(p []byte, off int64) (n int, err error) {
	if f.off >= f.Size {
		return 0, io.EOF
	}

	f.off = off
	req := &ReadRequest{
		Type:   WS_MESSAGE_TYPE_READ_REQUEST,
		Length: int64(len(p)),
		Offset: f.off,
	}

	err = f.conn.WriteJSON(req)
	if err != nil {
		return n, err
	}

	f.DataChans[off] = make(chan []byte)
	data := <-f.DataChans[off]
	n = copy(p, data)
	f.off = f.off + int64(n)
	return n, err

}

func NewFile(name string, size int64, conn *websocket.Conn) *File {
	return &File{
		ID:        uuid.Must(uuid.NewV4()).String(),
		Name:      name,
		Size:      size,
		conn:      conn,
		off:       0,
		DataChans: make(map[int64](chan []byte)),
	}
}
