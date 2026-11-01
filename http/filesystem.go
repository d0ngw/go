package http

import (
	"net/http"
	"os"
)

// NoDirFS is a file system that disables directory listings.
type NoDirFS struct {
	Fs http.FileSystem
}

// Open opens a file and rejects directories without an index.html file with os.ErrPermission.
func (fs NoDirFS) Open(name string) (http.File, error) {
	f, err := fs.Fs.Open(name)
	if err != nil {
		return nil, err
	}
	stat, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if stat.IsDir() {
		index, err := fs.Fs.Open(name + "/index.html")
		if err == nil {
			index.Close()
			return f, nil
		}
		f.Close()
		return nil, os.ErrPermission
	}
	return f, nil
}
