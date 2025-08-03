package main

import (
	"fmt"
	"slices"
)

func truthy(val interface{}) bool {
	switch v := val.(type) {
	case string:
		return v != ""
	case bool:
		return v
	case int:
		return v != 0
	}
	return false
}

// Templater for custom emails.
// Variables should be written as {varName}.
// If statements should be written as {if (!)varName}...{endif}.
// Strings are true if != "", ints are true if != 0.
// Errors returned are likely warnings only.
func templateEmail(content string, variables []string, conditionals []string, values map[string]interface{}) (string, error) {
	// minimum length for templatable content (albeit just "{}" -> "")
	if len(content) < 2 {
		return content, nil
	}
	ifStart, ifEnd := -1, -1
	ifTrue := false
	invalidIf := false
	previousEnd := -2
	blockRawStart := -1
	blockContentStart, blockContentEnd := -1, -1
	varStart, varEnd := -1, -1
	varName := ""
	out := ""
	var err error = nil

	oob := func(i int) bool { return i < 0 || i >= len(content) }

	for i, c := range content {
		if c == '{' {
			blockContentStart = i + 1
			blockRawStart = i
			if content[i+1] == '{' {
				err = fmt.Errorf(`double braces ("{{") at position %d, use single brace only`, i)
				blockContentStart++
			}
			for !oob(blockContentStart) && content[blockContentStart] == ' ' {
				blockContentStart++
			}
			if oob(blockContentStart) {
				continue
			}
			if !oob(blockContentStart+3) && content[blockContentStart:blockContentStart+3] == "if " {
				varStart = blockContentStart + 3
				for content[varStart] == ' ' {
					varStart++
				}
			}
			if ifStart == -1 && (oob(i-1) || content[i-1] != '{') {
				out += content[previousEnd+2 : i]
			}
			if invalidIf || oob(blockContentStart+5) || content[blockContentStart:blockContentStart+5] != "endif" {
				continue
			}
			ifEnd = i - 1
			if ifTrue {
				toAppend, subErr := templateEmail(content[ifStart:ifEnd+1], variables, conditionals, values)
				out += toAppend
				if subErr != nil {
					err = subErr
				}
				ifTrue = false
			}
		} else if c == '}' {
			doubleBraced := !oob(i+1) && content[i+1] == '}'
			if doubleBraced {
				err = fmt.Errorf(`double braces ("}}") at position %d, use single brace only`, i)
			}
			if !oob(i-1) && content[i-1] == '}' {
				continue
			}
			if varStart != -1 {
				ifStart = i + 1
				varEnd = i - 1
				for !oob(varEnd) && content[varEnd] == ' ' {
					varEnd--
				}
				varName = content[varStart : varEnd+1]
				positive := true
				if varName[0] == '!' {
					positive = false
					varName = varName[1:]
				}
				wrappedVarName := "{" + varName + "}"
				validVar := slices.Contains(conditionals, wrappedVarName)
				if validVar {
					ifTrue = positive == truthy(values[varName])
				} else {
					invalidIf = true
					ifStart, ifEnd = -1, -1
				}
				varStart, varEnd = -1, -1
			}
			blockContentEnd = i - 1
			for content[blockContentEnd] == ' ' {
				blockContentEnd--
			}
			previousEnd = i - 1
			// Skip the extra brace
			if doubleBraced {
				previousEnd++
			}
			if !oob(blockContentEnd-4) && !oob(blockContentEnd+1) && content[blockContentEnd-4:blockContentEnd+1] == "endif" && !invalidIf {
				continue
			}
			varName = content[blockContentStart : blockContentEnd+1]
			blockContentStart, blockContentEnd = -1, -1
			blockRawStart = -1
			if ifStart != -1 {
				continue
			}
			wrappedVarName := "{" + varName + "}"
			validVar := slices.Contains(variables, wrappedVarName)
			if !validVar {
				out += wrappedVarName
				continue
			}
			out += fmt.Sprint(values[varName])
		}
	}
	if blockContentStart != -1 && blockContentEnd == -1 {
		err = fmt.Errorf(`incomplete block (single "{") near position %d`, blockContentStart)
		// Include the brace, maybe the user wants it.
		previousEnd = blockRawStart - 2
	}
	if previousEnd+1 != len(content)-1 {
		out += content[previousEnd+2:]

	}
	if out == "" {
		return content, err
	}
	return out, err
}
