#!/usr/bin/python
import sys
import argparse

parser = argparse.ArgumentParser()
parser.add_argument("embed", metavar="<true/false>|<internal/external>|<yes/no>", type=str)
trues = ["true", "internal", "yes", "y"]
falses = ["false", "external", "no", "n"]

EMBED = parser.parse_args().embed

with open("embed.go", "w") as f:
    if EMBED in trues:
        f.write("""package main
import (
    "embed"
    "log"
)

//go:embed data data/html data/web data/web/css data/web/js
var localFS embed.FS

//go:embed lang/common lang/admin lang/email lang/form lang/setup
var langFS embed.FS

func loadLocalFS() {
    log.Println("Using internal storage")
}""")
    elif EMBED in falses:
        f.write("""package main
import (
    "io/fs"
    "os"
    "log"
    "path/filepath"
)

var localFS fs.FS
var langFS fs.FS

func loadLocalFS() {
    log.Println("Using external storage")
    executable, _ := os.Executable()
    localFS = os.DirFS(filepath.Dir(executable))
    langFS = os.DirFS(filepath.Join(filepath.Dir(executable), "data"))
}""")
