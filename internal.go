// +build !external

package main

import (
	"embed"
	"io/fs"
	"log"
)

const binaryType = "internal"

//go:embed data data/html data/web data/web/css data/web/js
var loFS embed.FS

//go:embed lang/common lang/admin lang/email lang/form lang/setup lang/pwreset lang/telegram
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
	for _, v := range elem {
		out += v + "/"
	}
	return out[:len(out)-1]
}

func loadFilesystems() {
	langFS = rewriteFS{laFS, "lang/"}
	localFS = rewriteFS{loFS, "data/"}
	log.Println("Using internal storage")
}
