package main

import "fmt"

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
func templateEmail(content string, variables []string, conditionals []string, values map[string]interface{}) string {
	ifStart, ifEnd := -1, -1
	ifTrue := false
	invalidIf := false
	previousEnd := -2
	cStart, cEnd := -1, -1
	varStart, varEnd := -1, -1
	varName := ""
	out := ""
	for i, c := range content {
		if c == '{' {
			cStart = i + 1
			for content[cStart] == ' ' {
				cStart++
			}
			if content[cStart:cStart+3] == "if " {
				varStart = cStart + 3
				for content[varStart] == ' ' {
					varStart++
				}
			}
			if ifStart == -1 {
				out += content[previousEnd+2 : i]
			}
			if content[cStart:cStart+5] != "endif" || invalidIf {
				continue
			}
			ifEnd = i - 1
			if ifTrue {
				out += templateEmail(content[ifStart:ifEnd+1], variables, conditionals, values)
				ifTrue = false
			}
		} else if c == '}' {
			if varStart != -1 {
				ifStart = i + 1
				varEnd = i - 1
				for content[varEnd] == ' ' {
					varEnd--
				}
				varName = content[varStart : varEnd+1]
				positive := true
				if varName[0] == '!' {
					positive = false
					varName = varName[1:]
				}
				validVar := false
				wrappedVarName := "{" + varName + "}"
				for _, v := range conditionals {
					if v == wrappedVarName {
						validVar = true
						break
					}
				}
				if validVar {
					ifTrue = positive == truthy(values[varName])
				} else {
					invalidIf = true
					ifStart, ifEnd = -1, -1
				}
				varStart, varEnd = -1, -1
			}
			cEnd = i - 1
			for content[cEnd] == ' ' {
				cEnd--
			}
			previousEnd = i - 1
			if content[cEnd-4:cEnd+1] == "endif" && !invalidIf {
				continue
			}
			validVar := false
			varName = content[cStart : cEnd+1]
			cStart, cEnd = -1, -1
			if ifStart != -1 {
				continue
			}
			wrappedVarName := "{" + varName + "}"
			for _, v := range variables {
				if v == wrappedVarName {
					validVar = true
					break
				}
			}
			if !validVar {
				out += wrappedVarName
				continue
			}
			out += fmt.Sprint(values[varName])
		}
	}
	if previousEnd+1 != len(content)-1 {
		out += content[previousEnd+2:]
	}
	if out == "" {
		return content
	}
	return out
}
