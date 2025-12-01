//go:build !external
// +build !external

package main

import (
	"embed"
	"io/fs"

	"github.com/hrfee/jfa-go/logger"
)

const binaryType = "internal"

func BuildTagsExternal() {}

//go:embed build/data build/data/html build/data/web build/data/web/css build/data/web/js
var loFS embed.FS

//go:embed lang/common lang/admin lang/email lang/form lang/setup lang/pwreset lang/telegram
var laFS embed.FS

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

func loadFilesystems(rootDir string, logger *logger.Logger) {
	langFS = rewriteFS{laFS, "lang/"}
	localFS = rewriteFS{loFS, "build/data/"}
	logger.Println("Using internal storage")
}
