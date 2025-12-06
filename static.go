package main

import (
	"io/fs"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// Since the gin-static middleware uses a version of http.Filesystem with an extra Exists() func, we extend it here.

type httpFS struct {
	hfs http.FileSystem // Created by converting fs.FS using http.FS()
	fs  fs.FS
}

func (f httpFS) Open(name string) (http.File, error) {
	return f.hfs.Open("web" + name)
}

func (f httpFS) Exists(prefix string, filepath string) bool {
	if p := strings.TrimPrefix(filepath, prefix); len(p) < len(filepath) {
		stats, err := fs.Stat(f.fs, "web/"+p)
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

var (
	etag = buildTimeUnix
)

// Use unix build time as the ETag for a request, allowing caching of static files.
// Copied from gin-contrib/static:
// https://github.com/gin-gonic/contrib/blob/2b1292699c15c6bc6ee8f0e801a4d0b4e807f366/static/static.go
func serveTaggedStatic(urlPrefix string, fs httpFS) gin.HandlerFunc {
	fileserver := http.FileServer(fs)
	if urlPrefix != "" {
		fileserver = http.StripPrefix(urlPrefix, fileserver)
	}
	return func(gc *gin.Context) {
		if fs.Exists(urlPrefix, gc.Request.URL.Path) {
			gc.Header("Cache-Control", "no-cache")
			gc.Header("ETag", buildTimeUnix)
			ifNoneMatchTag := gc.Request.Header.Get("If-None-Match")
			if ifNoneMatchTag != "" && ifNoneMatchTag == etag && (gc.Request.Method == http.MethodGet || gc.Request.Method == http.MethodHead) {
				gc.AbortWithStatus(http.StatusNotModified)
			} else {
				fileserver.ServeHTTP(gc.Writer, gc.Request)
				gc.Abort()
			}
		}
	}
}
