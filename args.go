package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func (app *appContext) loadArgs(firstCall bool) {
	if firstCall {
		flag.Usage = helpFunc
		help := flag.Bool("help", false, "prints this message.")
		flag.BoolVar(help, "h", false, "SHORTHAND")

		DATA = flag.String("data", app.dataPath, "alternate path to data directory.")
		flag.StringVar(DATA, "d", app.dataPath, "SHORTHAND")
		CONFIG = flag.String("config", app.configPath, "alternate path to config file.")
		flag.StringVar(CONFIG, "c", app.configPath, "SHORTHAND")
		HOST = flag.String("host", "", "alternate address to host web ui on.")
		PORT = flag.Int("port", 0, "alternate port to host web ui on.")
		flag.IntVar(PORT, "p", 0, "SHORTHAND")
		_LOADBAK = flag.String("restore", "", "path to database backup to restore.")
		DEBUG = flag.Bool("debug", false, "Enables debug logging.")
		PPROF = flag.Bool("pprof", false, "Exposes pprof profiler on /debug/pprof.")
		SWAGGER = flag.Bool("swagger", false, "Enable swagger at /swagger/index.html")

		flag.Parse()
		if *help {
			flag.Usage()
			os.Exit(0)
		}
		if *SWAGGER {
			os.Setenv("SWAGGER", "1")
		}
		if *DEBUG {
			os.Setenv("DEBUG", "1")
		}
		if *PPROF {
			os.Setenv("PPROF", "1")
		}
		if *_LOADBAK != "" {
			LOADBAK = *_LOADBAK
		}
	}

	if os.Getenv("SWAGGER") == "1" {
		*SWAGGER = true
	}
	if os.Getenv("DEBUG") == "1" {
		*DEBUG = true
	}
	if os.Getenv("PPROF") == "1" {
		*PPROF = true
	}
	// attempt to apply command line flags correctly
	if app.configPath == *CONFIG && app.dataPath != *DATA {
		app.dataPath = *DATA
		app.configPath = filepath.Join(app.dataPath, "config.ini")
	} else if app.configPath != *CONFIG && app.dataPath == *DATA {
		app.configPath = *CONFIG
	} else {
		app.configPath = *CONFIG
		app.dataPath = *DATA
	}

	// Previously used for self-restarts but leaving them here as they might be useful.
	if v := os.Getenv("JFA_CONFIGPATH"); v != "" {
		app.configPath = v
	}
	if v := os.Getenv("JFA_DATAPATH"); v != "" {
		app.dataPath = v
	}

	os.Setenv("JFA_CONFIGPATH", app.configPath)
	os.Setenv("JFA_DATAPATH", app.dataPath)
}

/*
	Adds start/stop/systemd to help message, and

also gets rid of usage for shorthand flags, and merge them with the full-length one.
implementation is 🤢, will clean this up eventually.

	-h SHORTHAND
	-help
		prints this message.

becomes:

	-help, -h
		prints this message.
*/
func helpFunc() {
	fmt.Fprint(stderr, `Usage of jfa-go:
  start
	start jfa-go as a daemon and run in the background.
  stop
	stop a daemonized instance of jfa-go.
  systemd
	generate a systemd .service file.
`)
	shortHands := []string{"-help", "-data", "-config", "-port"}
	var b bytes.Buffer
	// Write defaults into buffer then remove any shorthands
	flag.CommandLine.SetOutput(&b)
	flag.PrintDefaults()
	flag.CommandLine.SetOutput(stderr)
	scanner := bufio.NewScanner(&b)
	out := ""
	line := scanner.Text()
	eof := !scanner.Scan()
	lastLine := false
	for !eof || lastLine {
		nextline := scanner.Text()
		start := 0
		if len(nextline) != 0 {
			for nextline[start] == ' ' && start < len(nextline) {
				start++
			}
		}
		if strings.Contains(line, "SHORTHAND") || (len(nextline) != 0 && strings.Contains(nextline, "SHORTHAND") && nextline[start] != '-') {
			line = nextline
			if lastLine {
				break
			}
			eof := !scanner.Scan()
			if eof {
				lastLine = true
			}
			continue
		}
		// if !strings.Contains(line, "SHORTHAND") && !(strings.Contains(nextline, "SHORTHAND") && !strings.Contains(nextline, "-")) {
		match := false
		for i, c := range line {
			if c != '-' {
				continue
			}
			for _, s := range shortHands {
				if i+len(s) <= len(line) && line[i:i+len(s)] == s {
					out += line[:i+len(s)] + ", " + s[:2] + line[i+len(s):] + "\n"
					match = true
					break
				}
			}
		}
		if !match {
			out += line + "\n"
		}
		line = nextline
		if lastLine {
			break
		}
		eof := !scanner.Scan()
		if eof {
			lastLine = true
		}
	}
	fmt.Fprint(stderr, out)
}
