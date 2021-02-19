package main

import (
	"io"
	"log"

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
	logger  *log.Logger
	printer *c.Color
}

func NewLogger(out io.Writer, prefix string, flag int, color c.Attribute) (l logger) {
	l.logger = log.New(out, prefix, flag)
	l.printer = c.New(color)
	return l
}

func (l logger) Printf(format string, v ...interface{}) {
	l.logger.Print(l.printer.Sprintf(format, v...))
}
func (l logger) Print(v ...interface{})   { l.logger.Print(l.printer.Sprint(v...)) }
func (l logger) Println(v ...interface{}) { l.logger.Print(l.printer.Sprintln(v...)) }
func (l logger) Fatal(v ...interface{})   { l.logger.Fatal(l.printer.Sprint(v...)) }
func (l logger) Fatalf(format string, v ...interface{}) {
	l.logger.Fatal(l.printer.Sprintf(format, v...))
}

type emptyLogger bool

func (l emptyLogger) Printf(format string, v ...interface{}) {}
func (l emptyLogger) Print(v ...interface{})                 {}
func (l emptyLogger) Println(v ...interface{})               {}
func (l emptyLogger) Fatal(v ...interface{})                 {}
func (l emptyLogger) Fatalf(format string, v ...interface{}) {}
