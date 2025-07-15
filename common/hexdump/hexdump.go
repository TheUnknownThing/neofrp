package hexdump

import (
	"fmt"
	"strings"
	"unicode"
)

const (
	grey   = "\033[90m"
	yellow = "\033[93m"
	blue   = "\033[94m"
	reset  = "\033[0m"
)

func Dump(data []byte) string {
	var builder strings.Builder
	const lineLen = 16

	for i := 0; i < len(data); i += lineLen {
		// Address
		builder.WriteString(fmt.Sprintf("%08x: ", i))

		line := data[i:]
		if len(line) > lineLen {
			line = line[:lineLen]
		}

		// Hex dump
		builder.WriteString(grey)
		for j := 0; j < lineLen; j++ {
			if j < len(line) {
				builder.WriteString(fmt.Sprintf("%02x", line[j]))
			} else {
				builder.WriteString("  ") // padding
			}
			if j%2 == 1 {
				builder.WriteString(" ")
			}
		}
		builder.WriteString(reset)

		builder.WriteString(" ")

		// Rendered string
		for _, b := range line {
			if unicode.IsPrint(rune(b)) {
				builder.WriteString(yellow)
				builder.WriteRune(rune(b))
				builder.WriteString(reset)
			} else {
				builder.WriteString(blue)
				builder.WriteString(".")
				builder.WriteString(reset)
			}
		}
		builder.WriteString("\n")
	}

	return builder.String()
}
