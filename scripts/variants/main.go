// Package variants provides a script to comb through typescript/javascript files and
// find (most) instances of a a17t color (~neutral...) being used without
// an accompanying dark version (dark:~d_neutral...) and insert one.
// Not fully feature-matched with the old bash version, only matching classList.add/remove and class="...",
// but doesn't break on multi line function params, and is probably much faster.
package main

import (
	"bytes"
	"flag"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sync"
)

var VERBOSE = false
var classList = func() *regexp.Regexp {
	classList, err := regexp.Compile(`classList\.(add|remove)(\((?:[^()]*|\((?:[^()]*|\([^()]*\))\))*\))`)
	if err != nil {
		panic(err)
	}
	return classList
}()
var color = func() *regexp.Regexp {
	color, err := regexp.Compile(`\~(neutral|positive|warning|critical|info|urge|gray)`)
	if err != nil {
		panic(err)
	}
	return color
}()
var quotedColor = func() *regexp.Regexp {
	quotedColor, err := regexp.Compile(`("|'|\x60)\~(neutral|positive|warning|critical|info|urge|gray)("|'|\x60)`)
	if err != nil {
		panic(err)
	}
	return quotedColor
}()
var htmlClassList = func() *regexp.Regexp {
	htmlClassList, err := regexp.Compile(`class="[^"]*\~(neutral|positive|warning|critical|info|urge|gray)[^"]*"`)
	if err != nil {
		panic(err)
	}
	return htmlClassList
}()

func ParseDir(in, out string) error {
	err := filepath.WalkDir(in, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		perm := info.Mode()
		rel, err := filepath.Rel(in, path)
		outFile := filepath.Join(out, rel)
		if d.IsDir() {
			return os.MkdirAll(outFile, perm)
		}
		if VERBOSE {
			log.Printf("\"%s\" => \"%s\"\n", path, outFile)
		}
		if err != nil {
			return err
		}
		return ParseFile(path, outFile, &perm)
	})
	return err
}

func ParseDirParallel(in, out string) error {
	var wg sync.WaitGroup
	err := filepath.WalkDir(in, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		perm := info.Mode()
		rel, err := filepath.Rel(in, path)
		outFile := filepath.Join(out, rel)
		if d.IsDir() {
			return os.MkdirAll(outFile, perm)
		}
		if VERBOSE {
			log.Printf("\"%s\" => \"%s\"\n", path, outFile)
		}
		if err != nil {
			return err
		}
		wg.Add(1)
		go func() {
			if err := ParseFile(path, outFile, &perm); err != nil {
				panic(err)
			}
			wg.Done()
		}()
		return nil
	})
	if err != nil {
		return err
	}
	wg.Wait()
	return err
}

func ParseFile(in, out string, perm *fs.FileMode) error {
	file, err := os.ReadFile(in)
	if err != nil {
		return err
	}
	if perm == nil {
		f, err := os.Stat(in)
		if err != nil {
			return err
		}
		p := f.Mode()
		perm = &p
	}

	outText := classList.ReplaceAllFunc(file, func(match []byte) []byte {
		if bytes.Contains(match, []byte("dark:~d_")) {
			if VERBOSE {
				log.Printf("Skipping pre-set dark colour: `%s`\n", string(match))
			}
			return match
		}
		submatches := quotedColor.FindSubmatch(match)
		if len(submatches) == 0 {
			return match
		}
		quote := string(submatches[1])
		color := string(submatches[2])
		if VERBOSE {
			log.Printf("Matched quote %s, color %s\n", quote, color)
		}
		newVal := bytes.Replace(match, []byte(quote+"~"+color+quote), []byte(quote+"~"+color+quote+", "+quote+"dark:~d_"+color+quote), 1)
		if VERBOSE {
			log.Printf("`%s` => `%s`\n", string(match), string(newVal))
		}
		return newVal
	})

	outText = htmlClassList.ReplaceAllFunc(outText, func(match []byte) []byte {
		if bytes.Contains(match, []byte("dark:~d_")) {
			if VERBOSE {
				log.Printf("Skipping pre-set dark colour: `%s`\n", string(match))
			}
			return match
		}
		// Sucks we can't get a submatch from ReplaceAllFunc
		submatches := color.FindSubmatch(match)
		if len(submatches) == 0 {
			return match
		}
		c := submatches[1]
		newVal := bytes.Replace(match, []byte("~"+string(c)), []byte("~"+string(c)+" dark:~d_"+string(c)), 1)
		if VERBOSE {
			log.Printf("`%s` => `%s`\n", string(match), string(newVal))
		}
		return newVal
	})
	err = os.WriteFile(out, outText, *perm)
	return err
}

func main() {
	var inFile, inDir, out string
	var parallel bool
	flag.StringVar(&inFile, "file", "", "Input of an individual file.")
	flag.StringVar(&inDir, "dir", "", "Input of a whole directory.")
	flag.StringVar(&out, "out", "", "Output filepath/directory, depending on if -file or -dir passed.")
	flag.BoolVar(&VERBOSE, "v", false, "Prints information about files and replacements as they are made")
	flag.BoolVar(&parallel, "p", false, "Run a goroutine per file. Probably won't speed things up.")
	flag.Parse()

	if out == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}
	var err error
	if inFile != "" {
		err = ParseFile(inFile, out, nil)
	} else if inDir != "" {
		if parallel {
			err = ParseDirParallel(inDir, out)
		} else {
			err = ParseDir(inDir, out)
		}
	} else {
		flag.PrintDefaults()
		os.Exit(1)
	}
	if err != nil {
		log.Fatalf("failed: %v\n", err)
		os.Exit(1)
	}
}
