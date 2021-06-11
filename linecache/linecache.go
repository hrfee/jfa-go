// Package linecache provides a writer that stores n lines of text at once, overwriting old content as it reaches its capacity. Its contents can be read from with a String() method.
package linecache

import (
	"strings"
	"sync"
)

// LineCache provides an io.Writer that stores a fixed number of lines of text.
type LineCache struct {
	count   int
	lines   [][]byte
	current int
	lock    *sync.Mutex
}

// NewLineCache returns a new line cache of capacity (n) lines.
func NewLineCache(n int) *LineCache {
	return &LineCache{
		current: 0,
		count:   n,
		lines:   make([][]byte, n),
		lock:    &sync.Mutex{},
	}
}

// Write writes a given byte array to the cache.
func (l *LineCache) Write(p []byte) (n int, err error) {
	l.lock.Lock()
	defer l.lock.Unlock()
	lines := strings.Split(string(p), "\n")
	for _, line := range lines {
		if string(line) == "" {
			continue
		}
		if l.current == l.count {
			l.current = 0
		}
		l.lines[l.current] = []byte(line)
		l.current++
	}
	n = len(p)
	return
}

// String returns a string representation of the cache contents.
func (l *LineCache) String() string {
	i := 0
	if l.lines[l.count-1] != nil && l.current != l.count {
		i = l.current
	}
	out := ""
	for {
		if l.lines[i] == nil {
			return out
		}
		out += string(l.lines[i]) + "\n"
		i++
		if i == l.current {
			return out
		}
		if i == l.count {
			i = 0
		}
	}
}
