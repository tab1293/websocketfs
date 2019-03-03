package websocketfs

import "fmt"

type FileSystem struct {
	files map[string]*File
}

func NewFileSystem() *FileSystem {
	return &FileSystem{
		files: make(map[string]*File),
	}
}

func (fs *FileSystem) AddFile(f *File) {
	fs.files[f.ID] = f
}

func (fs *FileSystem) GetFile(ID string) (*File, error) {
	f := fs.files[ID]
	if f == nil {
		return nil, fmt.Errorf("File not found by ID %s", ID)
	}

	return f, nil
}
