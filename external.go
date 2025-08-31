//go:build external
// +build external

package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/hrfee/jfa-go/logger"
)

const binaryType = "external"

func BuildTagsExternal() { buildTags = append(buildTags, "external") }

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

func loadFilesystems(rootDir string, logger *logger.Logger) {
	logger.Println("Using external storage")
	if rootDir == "" {
		executable, _ := os.Executable()
		rootDir = filepath.Dir(executable)
	}
	localFS = dirFS(filepath.Join(rootDir, "data"))
	langFS = dirFS(filepath.Join(rootDir, "data", "lang"))
}
