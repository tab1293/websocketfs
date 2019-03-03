package websocketfs

import (
	"fmt"
	"log"
	"os"
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

type readJob struct {
	F      *File
	DF     *os.File
	Offset int64
	Length int64
}

func copyWorker(id int, jobs <-chan *readJob, results chan<- error) {
	for j := range jobs {
		log.Printf("reading at %d-%d size %d\n", j.Offset, j.Offset+j.Length, j.Length)
		b := make([]byte, j.Length)
		n, err := j.F.ReadAt(b, j.Offset)
		if err != nil {
			log.Printf("error reading at %d %s\n", j.Offset, err)
			results <- err
			continue
		}

		n, err = j.DF.WriteAt(b, j.Offset)
		if err != nil {
			log.Printf("error writing at %d %s\n", j.Offset, err)
			results <- err
			continue
		}

		log.Printf("wrote %d-%d size %d\n", j.Offset, j.Offset+j.Length, n)

		results <- nil
	}
}

func (fs *FileSystem) CopyFileToDisk(f *File) error {
	log.Printf("creating file %s\n", f.Name)
	df, err := os.Create(f.Name)
	if err != nil {
		return err
	}
	defer df.Close()

	var partSize int64 = 1024 * 1024 * 20
	var off int64 = 0
	jobs := make([]*readJob, 0)
	for off < f.Size {
		if off+partSize > f.Size {
			partSize = f.Size - off
		}

		j := &readJob{
			F:      f,
			DF:     df,
			Offset: off,
			Length: partSize,
		}
		jobs = append(jobs, j)
		off = off + partSize
	}

	jobChan := make(chan *readJob, len(jobs))
	results := make(chan error, len(jobs))

	start := time.Now()

	for w := 0; w < 3; w++ {
		go copyWorker(w, jobChan, results)
	}

	for _, j := range jobs {
		jobChan <- j
	}

	close(jobChan)

	for r := 0; r < len(jobs); r++ {
		<-results
	}

	elapsed := time.Since(start)
	log.Printf("copied file %s in %s\n", f.Name, elapsed)

	return nil
}
