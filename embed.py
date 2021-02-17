#!/usr/bin/python
import sys
import argparse

parser = argparse.ArgumentParser()
parser.add_argument("embed", metavar="<true/false>|<internal/external>|<yes/no>", type=str)
trues = ["true", "internal", "yes", "y"]
falses = ["false", "external", "no", "n"]

EMBED = parser.parse_args().embed

with open("embed.go", "w") as f:
    if EMBED in trues:
        f.write("""package main
import (
	"embed"
	"io/fs"
	"log"
)

//go:embed data data/html data/web data/web/css data/web/js
var loFS embed.FS

//go:embed lang/common lang/admin lang/email lang/form lang/setup
var laFS embed.FS

var langFS rewriteFS
var localFS rewriteFS

type rewriteFS struct {
	fs     embed.FS
	prefix string
}

func (l rewriteFS) Open(name string) (fs.File, error)          { return l.fs.Open(l.prefix + name) }
func (l rewriteFS) ReadDir(name string) ([]fs.DirEntry, error) { return l.fs.ReadDir(l.prefix + name) }
func (l rewriteFS) ReadFile(name string) ([]byte, error)       { return l.fs.ReadFile(l.prefix + name) }
func FSJoin(elem ...string) string {
    out := ""
    for _, v := range elem { out += v + "/" }
    return out[:len(out)-1]
}

func loadFilesystems() {
	langFS = rewriteFS{laFS, "lang/"}
	localFS = rewriteFS{loFS, "data/"}
	log.Println("Using internal storage")
}""")
    elif EMBED in falses:
        f.write("""package main
import (
    "io/fs"
    "os"
    "log"
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
}""")
