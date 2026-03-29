package parser

import "strings"

// SplitArgs splits a raw argument string on commas, respecting quoted strings.
func SplitArgs(raw string) []string {
	if raw == "" {
		return nil
	}

	var args []string
	var current strings.Builder
	inQuote := false

	for i := 0; i < len(raw); i++ {
		ch := raw[i]
		switch {
		case ch == '"':
			inQuote = !inQuote
			current.WriteByte(ch)
		case ch == ',' && !inQuote:
			args = append(args, RemoveQuotes(current.String()))
			current.Reset()
		default:
			current.WriteByte(ch)
		}
	}
	args = append(args, RemoveQuotes(current.String()))

	return args
}

// RemoveQuotes strips leading/trailing double quotes from a string.
func RemoveQuotes(s string) string {
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}
