package main

import (
	"strings"

	dg "github.com/bwmarrin/discordgo"
	stripmd "github.com/writeas/go-strip-markdown"
)

type Link struct {
	Alt, URL string
}

// StripAltText removes Markdown alt text from links and images and replaces them with just the URL.
// Currently uses the deepest alt text when links/images are nested.
// If links = true, links are completely removed, and a list of URLs and their alt text is also returned.
func StripAltText(md string, links bool) (string, []*dg.MessageEmbed) {
	altTextStart := -1 // Start of alt text (between '[' & ']')
	URLStart := -1     // Start of url (between '(' & ')')
	URLEnd := -1
	previousURLEnd := -2
	out := ""
	embeds := []*dg.MessageEmbed{}
	for i := range md {
		if altTextStart != -1 && URLStart != -1 && md[i] == ')' {
			URLEnd = i - 1
			out += md[previousURLEnd+2 : altTextStart-1]
			if links {
				embed := &dg.MessageEmbed{
					Type:  dg.EmbedTypeLink,
					Title: md[altTextStart : URLStart-2],
				}
				if md[altTextStart-1] == '!' {
					embed.Title = md[altTextStart+1 : URLStart-2]
					embed.Type = dg.EmbedTypeImage
					embed.Image = &dg.MessageEmbedImage{
						URL: md[URLStart : URLEnd+1],
					}
				} else {
					embed.URL = md[URLStart : URLEnd+1]
				}
				embeds = append(embeds, embed)
			} else {
				out += md[URLStart : URLEnd+1]
			}
			previousURLEnd = URLEnd
			// Removing links often leaves a load of extra newlines which look weird, this removes them.
			if links {
				next := 2
				for md[URLEnd+next] == '\n' {
					next++
				}
				if next >= 3 {
					previousURLEnd += next - 2
				}
			}
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
		return md, embeds
	}
	return out, embeds
}

func stripMarkdown(md string) string {
	stripped, _ := StripAltText(md, false)
	return strings.TrimPrefix(strings.TrimSuffix(stripmd.Strip(stripped), "</p>"), "<p>")
}
