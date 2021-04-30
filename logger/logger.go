// Package logger provides a wrapper around log that adds color support with github.com/fatih/color.
package logger

import (
	"io"
	"log"
	"runtime"
	"strconv"

	c "github.com/fatih/color"
)

type Logger interface {
	Printf(format string, v ...interface{})
	Print(v ...interface{})
	Println(v ...interface{})
	Fatal(v ...interface{})
	Fatalf(format string, v ...interface{})
}

type logger struct {
	logger    *log.Logger
	shortfile bool
	printer   *c.Color
}

func Lshortfile() string {
	// 0 = This function, 1 = Print/Printf/Println, 2 = Caller of Print/Printf/Println
	_, file, line, ok := runtime.Caller(2)
	lineString := strconv.Itoa(line)
	if !ok {
		return ""
	}
	if file == "" {
		return lineString
	}
	for i := len(file) - 1; i > 0; i-- {
		if file[i] == '/' || file[i] == '\\' {
			file = file[i+1:]
			break
		}
	}
	return file + ":" + lineString + ":"
}

func NewLogger(out io.Writer, prefix string, flag int, color c.Attribute) (l logger) {
	// Use reimplemented Lshortfile since wrapping the log functions messes them up
	if flag&log.Lshortfile != 0 {
		flag -= log.Lshortfile
		l.shortfile = true
	}

	l.logger = log.New(out, prefix, flag)
	l.printer = c.New(color)
	return l
}

func (l logger) Printf(format string, v ...interface{}) {
	var out string
	if l.shortfile {
		out = Lshortfile()
	}
	out += " " + l.printer.Sprintf(format, v...)
	l.logger.Print(out)
}

func (l logger) Print(v ...interface{}) {
	var out string
	if l.shortfile {
		out = Lshortfile()
	}
	out += " " + l.printer.Sprint(v...)
	l.logger.Print(out)
}

func (l logger) Println(v ...interface{}) {
	var out string
	if l.shortfile {
		out = Lshortfile()
	}
	out += " " + l.printer.Sprintln(v...)
	l.logger.Print(out)
}

func (l logger) Fatal(v ...interface{}) {
	var out string
	if l.shortfile {
		out = Lshortfile()
	}
	out += " " + l.printer.Sprint(v...)
	l.logger.Fatal(out)
}

func (l logger) Fatalf(format string, v ...interface{}) {
	var out string
	if l.shortfile {
		out = Lshortfile()
	}
	out += " " + l.printer.Sprintf(format, v...)
	l.logger.Fatal(out)
}

type EmptyLogger bool

func (l EmptyLogger) Printf(format string, v ...interface{}) {}
func (l EmptyLogger) Print(v ...interface{})                 {}
func (l EmptyLogger) Println(v ...interface{})               {}
func (l EmptyLogger) Fatal(v ...interface{})                 {}
func (l EmptyLogger) Fatalf(format string, v ...interface{}) {}
