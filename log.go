package main

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"

	"github.com/gin-gonic/gin"
	"github.com/hrfee/jfa-go/linecache"
)

var logPath string = filepath.Join(temp, "jfa-go.log")
var lineCache = linecache.NewLineCache(100)

var stderr = os.Stderr

func logOutput() (closeFunc func(), err error) {
	old := os.Stdout
	writers := []io.Writer{old, colorStripper{lineCache}}
	wExit := make(chan bool)
	r, w, _ := os.Pipe()
	var f *os.File
	if TRAY {
		log.Printf("Logging to \"%s\"", logPath)
		f, err = os.OpenFile(logPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
		if err != nil {
			closeFunc = func() {}
			return
		}
		if PLATFORM == "windows" {
			writers = []io.Writer{colorStripper{lineCache}, colorStripper{f}}
		} else {
			writers = append(writers, colorStripper{f})
		}
		closeFunc = func() {
			w.Close()
			<-wExit
			f.Close()
		}
	} else {
		closeFunc = func() {
			w.Close()
			<-wExit
		}
	}
	writer := io.MultiWriter(writers...)
	// FIXME: Potential cause if last log line doesn't get printed sometimes.
	os.Stdout, os.Stderr = w, w
	log.SetOutput(writer)
	gin.DefaultWriter, gin.DefaultErrorWriter = writer, writer
	go func() {
		io.Copy(writer, r)
		wExit <- true
	}()
	return
}

// Regex that removes ANSI color escape sequences. Used for outputting to log file and log cache.
var stripColors = func() *regexp.Regexp {
	r, err := regexp.Compile("\\x1b\\[[0-9;]*m")
	if err != nil {
		log.Fatalf("Failed to compile color escape regexp: %v", err)
	}
	return r
}()

type colorStripper struct {
	file io.Writer
}

func (c colorStripper) Write(p []byte) (n int, err error) {
	_, err = c.file.Write(stripColors.ReplaceAll(p, []byte("")))
	n = len(p)
	return
}

func sanitizeLog(l string) string {
	quoteCensor, err := regexp.Compile("\"([^\"]*)\"")
	if err != nil {
		log.Fatalf("Failed to compile sanitizing regexp: %v", err)
	}
	return string(quoteCensor.ReplaceAll([]byte(l), []byte("\"CENSORED\"")))
}
