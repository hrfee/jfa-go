package main

import (
	"strings"

	stripmd "github.com/writeas/go-strip-markdown"
)

func stripMarkdown(md string) string {
	// Search for markdown-formatted urls, and replace them with just the url, then use a library to strip any traces of markdown. You'll need some eyebleach after this.
	foundOpenSquare := false
	openSquare := -1
	openBracket := -1
	closeBracket := -1
	openSquares := []int{}
	closeBrackets := []int{}
	links := []string{}
	foundOpen := false
	for i, c := range md {
		if !foundOpenSquare && !foundOpen && c != '[' && c != ']' {
			continue
		}
		if c == '[' && md[i-1] != '!' {
			foundOpenSquare = true
			openSquare = i
		} else if c == ']' {
			if md[i+1] == '(' {
				foundOpenSquare = false
				foundOpen = true
				openBracket = i + 1
				continue
			}
		} else if c == ')' {
			closeBracket = i
			openSquares = append(openSquares, openSquare)
			closeBrackets = append(closeBrackets, closeBracket)
			links = append(links, md[openBracket+1:closeBracket])
			openBracket = -1
			closeBracket = -1
			openSquare = -1
			foundOpenSquare = false
			foundOpen = false
		}
	}
	fullLinks := make([]string, len(openSquares))
	for i := range openSquares {
		fullLinks[i] = md[openSquares[i] : closeBrackets[i]+1]
	}
	for i, _ := range openSquares {
		md = strings.Replace(md, fullLinks[i], links[i], 1)
	}
	return strings.TrimPrefix(strings.TrimSuffix(stripmd.Strip(md), "</p>"), "<p>")
}
