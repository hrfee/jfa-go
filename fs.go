package main

import (
	"io/fs"
	"os"
)

type genericFS interface {
	fs.FS
	fs.ReadDirFS
	fs.ReadFileFS
}

var localFS genericFS
var langFS genericFS

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
