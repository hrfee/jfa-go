package main

import (
	"fmt"
	"html/template"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"time"

	"github.com/pkg/browser"
)

// https://gist.github.com/swdunlop/9629168
func identifyPanic() string {
	var name, file string
	var line int
	var pc [16]uintptr

	n := runtime.Callers(4, pc[:])
	for _, pc := range pc[:n] {
		fn := runtime.FuncForPC(pc)
		if fn == nil {
			continue
		}
		file, line = fn.FileLine(pc)
		name = fn.Name()
		if !strings.HasPrefix(name, "runtime.") {
			break
		}
	}

	switch {
	case name != "":
		return fmt.Sprintf("%v:%v", name, line)
	case file != "":
		return fmt.Sprintf("%v:%v", file, line)
	}

	return fmt.Sprintf("pc:%x", pc)
}

// Exit dumps the last 100 lines of output to a crash file in /tmp (or equivalent), and generates a prettier HTML file containing it that is opened in the browser if possible.
func Exit(err interface{}) {
	tmpl, err2 := template.ParseFS(localFS, "html/crash.html", "html/header.html")
	if err2 != nil {
		log.Fatalf("Failed to load template: %v", err)
	}
	logCache := lineCache.String()
	logCache += "\n" + string(debug.Stack())
	sanitized := sanitizeLog(logCache)
	data := map[string]interface{}{
		"Log":          logCache,
		"SanitizedLog": sanitized,
	}
	if err != nil {
		data["Err"] = fmt.Sprintf("%s %v", identifyPanic(), err)
	}
	fpath := filepath.Join(temp, "jfa-go-crash-"+time.Now().Local().Format("2006-01-02T15:04:05"))
	err2 = os.WriteFile(fpath+".txt", []byte(logCache), 0666)
	if err2 != nil {
		log.Fatalf("Failed to write crash dump file: %v", err2)
	}
	log.Printf("\n------\nA crash report has been saved to \"%s\".\n------", fpath+".txt")
	f, err2 := os.OpenFile(fpath+".html", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	if err2 != nil {
		log.Fatalf("Failed to open crash dump file: %v", err2)
	}
	defer f.Close()
	err2 = tmpl.Execute(f, data)
	if err2 != nil {
		log.Fatalf("Failed to execute template: %v", err2)
	}
	browser.OpenFile(fpath + ".html")
	if TRAY {
		QuitTray()
	} else {
		os.Exit(1)
	}
}
