// Package logger provides a wrapper around log that adds color support with github.com/fatih/color.
package logger

import (
	"errors"
	"io"
	"log"
	"runtime"
	"strconv"

	c "github.com/fatih/color"
)

// type Logger interface {
// 	Printf(format string, v ...interface{})
// 	Print(v ...interface{})
// 	Println(v ...interface{})
// 	Fatal(v ...interface{})
// 	Fatalf(format string, v ...interface{})
// 	SetFatalFunc(f func(err interface{}))
// }

type Logger struct {
	empty     bool
	logger    *log.Logger
	shortfile bool
	printer   *c.Color
	fatalFunc func(err interface{})
}

// Lshortfile is a re-implemented log.Lshortfile with a modifiable call level.
func Lshortfile(level int) string {
	// 0 = This function, 1 = Print/Printf/Println, 2 = Caller of Print/Printf/Println
	_, file, line, ok := runtime.Caller(level)
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

func lshortfile() string {
	return Lshortfile(3)
}

func NewLogger(out io.Writer, prefix string, flag int, color c.Attribute) (l *Logger) {
	l = &Logger{}
	// Use reimplemented Lshortfile since wrapping the log functions messes them up
	if flag&log.Lshortfile != 0 {
		flag -= log.Lshortfile
		l.shortfile = true
	}

	l.logger = log.New(out, prefix, flag)
	l.printer = c.New(color)
	return l
}

func NewEmptyLogger() (l *Logger) {
	l = &Logger{
		empty: true,
	}
	return
}

func (l *Logger) Printf(format string, v ...interface{}) {
	if l.empty {
		return
	}
	var out string
	if l.shortfile {
		out = lshortfile()
	}
	out += " " + l.printer.Sprintf(format, v...)
	l.logger.Print(out)
}

func (l *Logger) Print(v ...interface{}) {
	if l.empty {
		return
	}
	var out string
	if l.shortfile {
		out = lshortfile()
	}
	out += " " + l.printer.Sprint(v...)
	l.logger.Print(out)
}

func (l *Logger) Println(v ...interface{}) {
	if l.empty {
		return
	}
	var out string
	if l.shortfile {
		out = lshortfile()
	}
	out += " " + l.printer.Sprintln(v...)
	l.logger.Print(out)
}

func (l *Logger) Fatal(v ...interface{}) {
	if l.empty {
		return
	}
	var out string
	if l.shortfile {
		out = lshortfile()
	}
	out += " " + l.printer.Sprint(v...)
	l.logger.Fatal(out)
}

func (l *Logger) Fatalf(format string, v ...interface{}) {
	if l.empty {
		return
	}
	var out string
	if l.shortfile {
		out = lshortfile()
	}
	out += " " + l.printer.Sprintf(format, v...)
	if l.fatalFunc != nil {
		l.fatalFunc(errors.New(out))
	} else {
		l.logger.Fatal(out)
	}
}

func (l *Logger) SetFatalFunc(f func(err interface{})) {
	l.fatalFunc = f
}
