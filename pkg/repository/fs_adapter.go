package repository

import (
	"io/fs"

	"github.com/go-git/go-billy/v5"
)

var _ fs.FS = &fsAdapter{}

// fsAdapter adapts billy.Filesystem to fs.FS
type fsAdapter struct{ fs billy.Filesystem }

func (fsa fsAdapter) Open(path string) (fs.File, error) {
	file, err := fsa.fs.Open(path)
	if err != nil {
		return nil, err
	}
	return &fileAdapter{f: file, fs: fsa.fs}, err
}

type fileAdapter struct {
	f  billy.File
	fs billy.Filesystem
}

func (fa fileAdapter) Stat() (fs.FileInfo, error) {
	return fa.fs.Stat(fa.f.Name())
}

func (fa fileAdapter) Read(p []byte) (int, error) {
	return fa.f.Read(p)
}

func (fa fileAdapter) Close() error {
	return fa.f.Close()
}
