//go:build external
// +build external

package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"
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

func loadFilesystems() {
	log.Println("Using external storage")
	executable, _ := os.Executable()
	localFS = dirFS(filepath.Join(filepath.Dir(executable), "data"))
	langFS = dirFS(filepath.Join(filepath.Dir(executable), "data", "lang"))
}
