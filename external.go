// +build external

package main

import (
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
)

const binaryType = "external"

var localFS dirFS
var langFS dirFS

// When using os.DirFS, even on Windows the separator seems to be '/'.
// func FSJoin(elem ...string) string { return filepath.Join(elem...) }
func FSJoin(elem ...string) string {
	sep := "/"
	if strings.Contains(elem[0], "\\") {
		sep = "\\"
	}
	path := ""
	for _, el := range elem {
		path += el + sep
	}
	return strings.TrimSuffix(path, sep)
}

type dirFS string

func (dir dirFS) Open(name string) (fs.File, error) {
	return os.Open(string(dir) + "/" + name)
}

func (dir dirFS) ReadFile(name string) ([]byte, error) {
	return os.ReadFile(string(dir) + "/" + name)
}

func (dir dirFS) ReadDir(name string) ([]fs.DirEntry, error) {
	return os.ReadDir(string(dir) + "/" + name)
}

func loadFilesystems() {
	log.Println("Using external storage")
	executable, _ := os.Executable()
	localFS = dirFS(filepath.Join(filepath.Dir(executable), "data"))
	langFS = dirFS(filepath.Join(filepath.Dir(executable), "data", "lang"))
}
