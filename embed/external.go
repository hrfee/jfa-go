package main

import (
	"io/fs"
	"log"
	"os"
	"path/filepath"
)

var localFS fs.FS
var langFS fs.FS

func FSJoin(elem ...string) string { return filepath.Join(elem...) }

func loadFilesystems() {
	log.Println("Using external storage")
	executable, _ := os.Executable()
	localFS = os.DirFS(filepath.Join(filepath.Dir(executable), "data"))
	langFS = os.DirFS(filepath.Join(filepath.Dir(executable), "data", "lang"))
}
