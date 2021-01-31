package main

import (
	"io/fs"
	"net/http"
	"strings"
)

type httpFS struct {
	hfs http.FileSystem
	fs  fs.FS
}

func (f httpFS) Open(name string) (http.File, error) {
	return f.hfs.Open(name)
}

func (f httpFS) Exists(prefix string, filepath string) bool {
	if p := strings.TrimPrefix(filepath, prefix); len(p) < len(filepath) {
		stats, err := fs.Stat(f.fs, p)
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
