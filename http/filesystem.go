package http

import (
	"net/http"
	"os"
)

// NoDirFS 不输出目录列表的FS
type NoDirFS struct {
	Fs http.FileSystem
}

// Open 取得指定的文件,如果name指向的是个目录,且目录下没有index.html,返回os.ErrPermission
func (fs NoDirFS) Open(name string) (http.File, error) {
	f, err := fs.Fs.Open(name)
	if err != nil {
		return nil, err
	}

	stat, err := f.Stat()
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
