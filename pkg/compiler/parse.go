package compiler

import (
	"strconv"
	"strings"
)

// ParseLine parses a single batch line into an AST node.
// Returns nil for blank lines.
func ParseLine(raw string) Node {
	line := strings.TrimSpace(raw)
	if line == "" {
		return nil
	}

	// Strip leading @ (silent mode prefix)
	if strings.HasPrefix(line, "@") {
		line = strings.TrimSpace(line[1:])
	}

	// Comments
	if strings.HasPrefix(line, "::") || strings.HasPrefix(line, "REM ") || strings.EqualFold(line, "REM") {
		return Comment{Text: line}
	}

	// Labels
	if line[0] == ':' && (len(line) < 2 || line[1] != ':') {
		return Label{Name: strings.TrimSpace(line[1:])}
	}

	// Split into command and rest
	lower := strings.ToLower(line)

	// echo off
	if lower == "echo off" || lower == "echo  off" {
		return EchoOff{}
	}

	// echo. (blank line) — also matches "echo."
	if lower == "echo." || lower == "echo ." {
		return EchoBlank{}
	}

	// echo with possible redirect
	if hasPrefix(lower, "echo") && (len(line) <= 4 || line[4] == ' ' || line[4] == '.') {
		return parseEcho(line)
	}

	// set — handle both "set " and "set/p" (no space)
	if hasPrefix(lower, "set ") || hasPrefix(lower, "set/") {
		return parseSet(line)
	}

	// if
	if hasPrefix(lower, "if ") {
		return parseIf(line)
	}

	// goto
	if hasPrefix(lower, "goto ") {
		target := strings.TrimSpace(line[5:])
		return Goto{Target: target}
	}

	// call
	if hasPrefix(lower, "call ") {
		path := strings.TrimSpace(line[5:])
		return Call{Path: unquote(path)}
	}

	// exit
	if lower == "exit" || lower == "exit /b" {
		return Exit{}
	}

	// pause
	if lower == "pause" || strings.TrimSpace(lower) == "pause" {
		return Pause{Quiet: false}
	}
	if strings.HasPrefix(lower, "pause>") || strings.HasPrefix(lower, "pause >") {
		return Pause{Quiet: true}
	}

	// cls
	if lower == "cls" {
		return Cls{}
	}

	// color
	if hasPrefix(lower, "color ") {
		return Color{Code: strings.TrimSpace(line[6:])}
	}

	// title
	if hasPrefix(lower, "title ") {
		text := strings.TrimSpace(line[6:])
		text = strings.ReplaceAll(text, "^&", "&")
		return Title{Text: text}
	}

	// type
	if hasPrefix(lower, "type ") {
		rest := strings.TrimSpace(line[5:])
		if strings.Contains(strings.ToLower(rest), "| more") {
			path := strings.TrimSpace(rest[:strings.Index(strings.ToLower(rest), "|")])
			return TypeFilePaged{Path: unquote(path)}
		}
		return TypeFile{Path: unquote(rest)}
	}

	// del
	if hasPrefix(lower, "del ") {
		return Del{Path: unquote(strings.TrimSpace(line[4:]))}
	}

	// mkdir
	if hasPrefix(lower, "mkdir ") {
		return Mkdir{Path: unquote(strings.TrimSpace(line[6:]))}
	}

	// xcopy — emit as comment (out of scope for sandboxed VM)
	if hasPrefix(lower, "xcopy ") {
		return Comment{Text: "xcopy: " + line}
	}

	// ping — translate to sleep
	if hasPrefix(lower, "ping ") {
		return parsePing(line)
	}

	// taskkill, shutdown — emit as comment (out of scope)
	if hasPrefix(lower, "taskkill ") || hasPrefix(lower, "shutdown ") {
		return Comment{Text: "system: " + line}
	}

	// Unrecognized — treat as comment to avoid data loss
	return Comment{Text: "unrecognized: " + line}
}

func parseEcho(line string) Node {
	// Get the text after "echo" — handle both "echo " and "echo."
	var text string
	lower := strings.ToLower(line)
	if strings.HasPrefix(lower, "echo.") {
		text = strings.TrimSpace(line[5:])
	} else if len(line) > 5 {
		text = line[5:] // preserve leading spaces in echo text
	} else {
		return EchoBlank{}
	}

	if text == "" {
		return EchoBlank{}
	}

	// Check for redirect
	appendIdx, overwriteIdx := findRedirect(text)

	if appendIdx >= 0 {
		echoText := strings.TrimSpace(text[:appendIdx])
		path := strings.TrimSpace(text[appendIdx+2:])
		return EchoToFile{Text: processEscapes(echoText), Path: processEscapes(unquote(path)), Append: true}
	}
	if overwriteIdx >= 0 {
		echoText := strings.TrimSpace(text[:overwriteIdx])
		path := strings.TrimSpace(text[overwriteIdx+1:])
		// skip "> nul" redirects
		if strings.EqualFold(strings.TrimSpace(path), "nul") {
			return Noop{}
		}
		return EchoToFile{Text: processEscapes(echoText), Path: processEscapes(unquote(path)), Append: false}
	}

	return Echo{Text: processEscapes(strings.TrimRight(text, " "))}
}

// findRedirect finds >> or > outside of quotes, returns index or -1.
func findRedirect(s string) (appendIdx, overwriteIdx int) {
	inQuote := false
	for i := 0; i < len(s); i++ {
		if s[i] == '"' {
			inQuote = !inQuote
		}
		// skip escaped >
		if i > 0 && s[i-1] == '^' {
			continue
		}
		if !inQuote && s[i] == '>' {
			if i+1 < len(s) && s[i+1] == '>' {
				return i, -1
			}
			return -1, i
		}
	}
	return -1, -1
}

func parseSet(line string) Node {
	// Handle both "set " and "set/" (no space before /p)
	lower := strings.ToLower(line)
	var rest string
	if hasPrefix(lower, "set/") {
		rest = strings.TrimSpace(line[3:]) // keep the /p
	} else {
		rest = strings.TrimSpace(line[4:])
	}
	lower = strings.ToLower(rest)

	// set /p VAR=prompt or set /p VAR=<file
	if strings.HasPrefix(lower, "/p ") || strings.HasPrefix(lower, "/p\"") {
		rest = strings.TrimSpace(rest[2:])
		// Handle quoted form: /p "VAR=prompt"
		rest = strings.TrimSpace(rest)
		if len(rest) > 0 && rest[0] == '"' {
			rest = unquote(rest)
		}
		eqIdx := strings.Index(rest, "=")
		if eqIdx < 0 {
			return Comment{Text: "malformed set /p: " + line}
		}
		name := strings.TrimSpace(rest[:eqIdx])
		value := rest[eqIdx+1:]

		// Check for <"file" pattern (read from file)
		trimmed := strings.TrimSpace(value)
		if strings.HasPrefix(trimmed, "<") {
			path := strings.TrimSpace(trimmed[1:])
			return SetFromFile{Name: name, Path: unquote(path)}
		}

		return SetPrompt{Name: name, Prompt: processEscapes(value)}
	}

	// set /a VAR=expr
	if strings.HasPrefix(lower, "/a ") {
		rest = strings.TrimSpace(rest[3:])
		eqIdx := strings.Index(rest, "=")
		if eqIdx < 0 {
			return Comment{Text: "malformed set /a: " + line}
		}
		name := strings.TrimSpace(rest[:eqIdx])
		expr := strings.TrimSpace(rest[eqIdx+1:])
		return SetArith{Name: name, Expr: expr}
	}

	// set VAR=value
	eqIdx := strings.Index(rest, "=")
	if eqIdx < 0 {
		return Comment{Text: "malformed set: " + line}
	}
	name := strings.TrimSpace(rest[:eqIdx])
	value := rest[eqIdx+1:]

	// Check for <file redirect in plain set too
	trimmed := strings.TrimSpace(value)
	if strings.HasPrefix(trimmed, "<") {
		path := strings.TrimSpace(trimmed[1:])
		return SetFromFile{Name: name, Path: unquote(path)}
	}

	return SetVar{Name: name, Value: unquote(value)}
}

func parseIf(line string) Node {
	rest := strings.TrimSpace(line[3:])

	negated := false
	if hasPrefix(strings.ToLower(rest), "not ") {
		negated = true
		rest = strings.TrimSpace(rest[4:])
	}

	// IF EXIST "path" goto LABEL
	if hasPrefix(strings.ToLower(rest), "exist ") {
		rest = strings.TrimSpace(rest[6:])
		// Find goto/call/exit in the rest
		path, then := splitIfAction(rest)
		op := "EXIST"
		if negated {
			op = "NOT EXIST"
		}
		return IfCondition{
			Negated:  negated,
			Operator: op,
			Left:     unquote(path),
			Then:     then,
		}
	}

	// IF "%VAR%" EQU "value" goto LABEL
	// IF %VAR%==value goto LABEL
	// IF "%VAR%" "value" goto LABEL (malformed — treat as EQU)

	// Extract left operand
	left, rest := extractOperand(rest)

	// Try to find operator
	operator, rest := extractOperator(rest)

	// Extract right operand
	right, rest := extractOperand(rest)

	// Rest should be the action (goto/call/exit)
	then := parseAction(strings.TrimSpace(rest))

	return IfCondition{
		Negated:  negated,
		Operator: operator,
		Left:     smartStripPercents(unquote(left)),
		Right:    unquote(right),
		Then:     then,
	}
}

func parsePing(line string) Node {
	// Extract -n and -w values for sleep calculation
	lower := strings.ToLower(line)
	parts := strings.Fields(lower)

	n, w := 1, 1000
	for i, p := range parts {
		if p == "-n" && i+1 < len(parts) {
			n, _ = strconv.Atoi(parts[i+1])
		}
		if p == "-w" && i+1 < len(parts) {
			// strip any trailing > or >nul
			val := strings.TrimRight(parts[i+1], ">")
			w, _ = strconv.Atoi(val)
		}
	}
	return Sleep{Ms: n * w}
}

// splitIfAction splits "path" goto LABEL or "path" set VAR=val into path and action node.
func splitIfAction(s string) (string, Node) {
	lower := strings.ToLower(s)
	for _, kw := range []string{" goto ", " call ", " exit", " set "} {
		idx := strings.Index(lower, kw)
		if idx >= 0 {
			path := strings.TrimSpace(s[:idx])
			action := parseAction(strings.TrimSpace(s[idx+1:]))
			return path, action
		}
	}
	return s, Noop{}
}

func extractOperand(s string) (string, string) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", ""
	}

	// Double-quoted: "%VAR%" or "value"
	if s[0] == '"' {
		end := strings.Index(s[1:], "\"")
		if end >= 0 {
			return s[:end+2], strings.TrimSpace(s[end+2:])
		}
	}

	// Single-quoted: '%VAR%' or 'value'
	if s[0] == '\'' {
		end := strings.Index(s[1:], "'")
		if end >= 0 {
			return s[:end+2], strings.TrimSpace(s[end+2:])
		}
	}

	// Check for == operator (unquoted: %VAR%==value)
	eqIdx := strings.Index(s, "==")
	if eqIdx >= 0 {
		return s[:eqIdx], s[eqIdx:]
	}

	// Unquoted: up to next space
	spIdx := strings.Index(s, " ")
	if spIdx >= 0 {
		return s[:spIdx], strings.TrimSpace(s[spIdx:])
	}

	return s, ""
}

func extractOperator(s string) (string, string) {
	s = strings.TrimSpace(s)
	lower := strings.ToLower(s)

	for _, op := range []string{"equ", "neq", "gtr", "lss", "geq", "leq"} {
		if strings.HasPrefix(lower, op+" ") || strings.HasPrefix(lower, op+"\t") {
			return strings.ToUpper(op), strings.TrimSpace(s[len(op):])
		}
	}

	if strings.HasPrefix(s, "==") {
		return "==", strings.TrimSpace(s[2:])
	}

	// Malformed: missing operator — assume EQU
	return "EQU", s
}

func parseAction(s string) Node {
	lower := strings.ToLower(s)
	if hasPrefix(lower, "goto ") {
		return Goto{Target: strings.TrimSpace(s[5:])}
	}
	if hasPrefix(lower, "call ") {
		return Call{Path: unquote(strings.TrimSpace(s[5:]))}
	}
	if lower == "exit" {
		return Exit{}
	}
	if hasPrefix(lower, "set ") {
		return parseSet(s)
	}
	return Noop{}
}

// Helpers

func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && strings.EqualFold(s[:len(prefix)], prefix)
}

func unquote(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	// batch also uses single quotes in some comparisons
	if len(s) >= 2 && s[0] == '\'' && s[len(s)-1] == '\'' {
		return s[1 : len(s)-1]
	}
	return s
}

func stripPercents(s string) string {
	s = strings.TrimPrefix(s, "%")
	s = strings.TrimSuffix(s, "%")
	return s
}

// smartStripPercents strips outer %% from simple variable references like %user%,
// but preserves compound expressions like %TL%+%TM%+%TR% which need VM interpolation.
func smartStripPercents(s string) string {
	// Count % signs — a simple %VAR% has exactly 2
	count := strings.Count(s, "%")
	if count == 2 && strings.HasPrefix(s, "%") && strings.HasSuffix(s, "%") {
		return s[1 : len(s)-1]
	}
	// Compound expression or no percents — keep as-is for VM interpolation
	return s
}

func processEscapes(s string) string {
	s = strings.ReplaceAll(s, "^>", ">")
	s = strings.ReplaceAll(s, "^&", "&")
	s = strings.ReplaceAll(s, "^|", "|")
	s = strings.ReplaceAll(s, "^%", "%")
	return s
}

// preprocessBatchPercents handles batch %% escaping in echo text.
// In batch, %VAR% is a variable reference and %% is a literal %.
// We need to distinguish these before the text reaches the CBAT emitter.
// Strategy: replace %% with a placeholder \x00PCT\x00 that can't appear
// in normal text, then after interpolation references are preserved,
// convert the placeholder to a raw % in the final CBAT string.
const pctPlaceholder = "\x00PCT\x00"

func preprocessBatchPercents(s string) string {
	// Process from left to right, distinguishing %VAR% from %%
	var out strings.Builder
	i := 0
	for i < len(s) {
		if s[i] != '%' {
			out.WriteByte(s[i])
			i++
			continue
		}
		// %% → literal percent placeholder
		if i+1 < len(s) && s[i+1] == '%' {
			out.WriteString(pctPlaceholder)
			i += 2
			continue
		}
		// %VAR% → keep as-is for VM interpolation
		end := strings.IndexByte(s[i+1:], '%')
		if end >= 0 && !strings.ContainsAny(s[i+1:i+1+end], " \t") {
			out.WriteString(s[i : i+2+end])
			i = i + 2 + end
		} else {
			out.WriteByte(s[i])
			i++
		}
	}
	return out.String()
}

func finalizeBatchPercents(s string) string {
	return strings.ReplaceAll(s, pctPlaceholder, "%")
}
