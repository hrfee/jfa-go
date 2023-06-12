package main

import (
	"fmt"
	"html/template"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"time"

	"github.com/robert-nix/ansihtml"
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

// OpenFile attempts to open a given file in the appropriate GUI application.
func OpenFile(fpath string) (err error) {
	switch PLATFORM {
	case "linux":
		err = exec.Command("xdg-open", fpath).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", fpath).Start()
	case "darwin":
		err = exec.Command("open", fpath).Start()
	default:
		err = fmt.Errorf("unknown os")
	}
	return
}

// Exit dumps the last 100 lines of output to a crash file in /tmp (or equivalent), and generates a prettier HTML file containing it that is opened in the browser if possible.
func Exit(err interface{}) {
	tmpl, err2 := template.ParseFS(localFS, "html/crash.html", "html/header.html")
	if err2 != nil {
		log.Fatalf("Failed to load template: %v", err)
	}
	logCache := lineCache.String()
	if err != nil {
		fmt.Println(err)
		logCache += "\n" + fmt.Sprint(err)
	}
	logCache += "\n" + string(debug.Stack())
	sanitized := sanitizeLog(logCache)
	data := map[string]interface{}{
		"Log":          logCache,
		"SanitizedLog": sanitized,
	}
	if err != nil {
		data["Err"] = fmt.Sprintf("%s %v", identifyPanic(), err)
	}
	// Use dashes for time rather than colons for Windows
	fpath := filepath.Join(temp, "jfa-go-crash-"+time.Now().Local().Format("2006-01-02T15-04-05"))
	err2 = os.WriteFile(fpath+".txt", []byte(logCache), 0666)
	if err2 != nil {
		log.Fatalf("Failed to write crash dump file: %v", err2)
	}
	log.Printf("\n------\nA crash report has been saved to \"%s\".\n------", fpath+".txt")

	// Render ANSI colors to HTML
	data["Log"] = template.HTML(string(ansihtml.ConvertToHTML([]byte(data["Log"].(string)))))
	data["SanitizedLog"] = template.HTML(string(ansihtml.ConvertToHTML([]byte(data["SanitizedLog"].(string)))))
	data["Err"] = template.HTML(string(ansihtml.ConvertToHTML([]byte(data["Err"].(string)))))

	f, err2 := os.OpenFile(fpath+".html", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	if err2 != nil {
		log.Fatalf("Failed to open crash dump file: %v", err2)
	}
	defer f.Close()
	err2 = tmpl.Execute(f, data)
	if err2 != nil {
		log.Fatalf("Failed to execute template: %v", err2)
	}
	if err := OpenFile(fpath + ".html"); err != nil {
		log.Printf("Failed to open browser, trying text file...")
		OpenFile(fpath + ".txt")
	}
	if TRAY {
		QuitTray()
	} else {
		os.Exit(1)
	}
}
