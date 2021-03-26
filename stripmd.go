package main

import (
	"strings"

	stripmd "github.com/writeas/go-strip-markdown"
)

// StripAltText removes Markdown alt text from links and images and replaces them with just the URL.
// Currently uses the deepest alt text when links/images are nested.
func StripAltText(md string) string {
	altTextStart := -1 // Start of alt text (between '[' & ']')
	URLStart := -1     // Start of url (between '(' & ')')
	URLEnd := -1
	previousURLEnd := -2
	out := ""
	for i := range md {
		if altTextStart != -1 && URLStart != -1 && md[i] == ')' {
			URLEnd = i - 1
			out += md[previousURLEnd+2:altTextStart-1] + md[URLStart:URLEnd+1]
			previousURLEnd = URLEnd
			altTextStart, URLStart, URLEnd = -1, -1, -1
			continue
		}
		if md[i] == '[' && altTextStart == -1 {
			altTextStart = i + 1
			if i > 0 && md[i-1] == '!' {
				altTextStart--
			}
		}
		if i > 0 && md[i-1] == ']' && md[i] == '(' && URLStart == -1 {
			URLStart = i + 1
		}
	}
	if previousURLEnd+1 != len(md)-1 {
		out += md[previousURLEnd+2:]
	}
	if out == "" {
		return md
	}
	return out
}

func stripMarkdown(md string) string {
	return strings.TrimPrefix(strings.TrimSuffix(stripmd.Strip(StripAltText(md)), "</p>"), "<p>")
}
