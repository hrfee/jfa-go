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

var localFS fs.FS
var langFS fs.FS

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

func loadFilesystems() {
	log.Println("Using external storage")
	executable, _ := os.Executable()
	localFS = os.DirFS(filepath.Join(filepath.Dir(executable), "data"))
	langFS = os.DirFS(filepath.Join(filepath.Dir(executable), "data", "lang"))
}
