package linecache

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func Test(t *testing.T) {
	wr := NewLineCache(10)
	for i := 10; i < 50; i++ {
		fmt.Fprintln(wr, i)
		fmt.Print(strings.ReplaceAll(wr.String(), "\n", " "), "\n")
		time.Sleep(time.Second)
	}
}
