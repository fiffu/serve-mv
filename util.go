package main

import (
	"fmt"
	"runtime"
	"unicode"
)

func CaseInsensitiveGlobstr(path string) string {
	if runtime.GOOS == "windows" {
		return path
	}

	p := ""
	for _, r := range path {
		if unicode.IsLetter(r) {
			p += fmt.Sprintf("[%c%c]", unicode.ToLower(r), unicode.ToUpper(r))
		} else {
			p += string(r)
		}
	}
	return p
}
