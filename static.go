package main

import (
	"fmt"
	"io/fs"
	"net/http"
	"strings"
)

// Since the gin-static middleware uses a version of http.Filesystem with an extra Exists() func, we extend it here.

type httpFS struct {
	hfs http.FileSystem // Created by converting fs.FS using http.FS()
	fs  fs.FS
}

func (f httpFS) Open(name string) (http.File, error) {
	fmt.Println("build/data/web" + name)
	return f.hfs.Open("build/data/web" + name)
}

func (f httpFS) Exists(prefix string, filepath string) bool {
	if p := strings.TrimPrefix(filepath, prefix); len(p) < len(filepath) {
		stats, err := fs.Stat(f.fs, "build/data/web/"+p)
		if err != nil {
			return false
		}
		if stats.IsDir() {
			return false
		}
		return true
	}
	return false
}
