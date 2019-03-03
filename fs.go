package websocketfs

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

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

func (fs *FileSystem) CopyFileToDisk(f *File) error {
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

			n, err = df.WriteAt(b, offset)
			if err != nil {
				log.Printf("error writing at %d %s\n", offset, err)
				return err
			}
			log.Printf("wrote %d-%d size %d\n", offset, offset+length, n)
			return nil
		}(off, partSize)

		off = off + partSize
	}

	wg.Wait()
	elapsed := time.Since(start)
	log.Printf("copied file %s in %s\n", f.Name, elapsed)

	return nil
}
