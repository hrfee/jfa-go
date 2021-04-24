package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"os"
	"strings"
)

/* Adds start/stop/systemd to help message, and
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
	fmt.Fprint(os.Stderr, `Usage of jfa-go:
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
	flag.CommandLine.SetOutput(os.Stderr)
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
	fmt.Fprint(os.Stderr, out)
}
