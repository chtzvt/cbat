package parser

import (
	"bufio"
	"io"
	"os"
	"strconv"
	"strings"

	"BBS_VM/pkg/vm"
)

type Result struct {
	GlobalEntry string
	Programs    []*vm.Program
	Files       map[string]string
}

type Program = vm.Program

func ParseFile(path string) (*Result, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return Parse(f)
}

func ParseString(source string) (*Result, error) {
	return Parse(strings.NewReader(source))
}

func Parse(r io.Reader) (*Result, error) {
	res := &Result{
		Files: make(map[string]string),
	}

	type section int
	const (
		secUnknown section = iota
		secGlobal
		secFiles
		secHeader
		secLabels
		secInstrs
	)

	var cur section
	var prog *vm.Program
	hitHeader := false

	storeProg := func() {
		if prog == nil {
			return
		}
		res.Programs = append(res.Programs, prog)
		prog = nil
	}

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, ";") {
			continue
		}

		// strip inline comments
		if idx := strings.Index(line, ";"); idx > 0 {
			// don't strip semicolons inside quotes
			inQuote := false
			for i := 0; i < len(line); i++ {
				if line[i] == '"' {
					inQuote = !inQuote
				}
				if line[i] == ';' && !inQuote {
					line = strings.TrimSpace(line[:i])
					break
				}
			}
		}
		if line == "" {
			continue
		}

		switch line {
		case ".global":
			cur = secGlobal
			continue
		case ".files":
			cur = secFiles
			continue
		case ".header":
			if hitHeader {
				storeProg()
			}
			hitHeader = true
			prog = &vm.Program{
				Labels: make(map[string]int),
			}
			cur = secHeader
			continue
		case ".labels":
			cur = secLabels
			continue
		case ".instrs":
			cur = secInstrs
			continue
		}

		switch cur {
		case secGlobal:
			parts := strings.SplitN(line, " ", 2)
			if parts[0] == "entry" && len(parts) > 1 {
				res.GlobalEntry = RemoveQuotes(strings.TrimSpace(parts[1]))
			}

		case secFiles:
			// format: "FILENAME","CONTENT"
			parts := strings.SplitN(line, "\",\"", 2)
			if len(parts) == 2 {
				name := strings.TrimPrefix(parts[0], "\"")
				content := strings.TrimSuffix(parts[1], "\"")
				content = strings.ReplaceAll(content, "\\n", "\n")
				res.Files[name] = content
			}

		case secHeader:
			parts := strings.SplitN(line, " ", 2)
			if prog == nil {
				continue
			}
			switch parts[0] {
			case "filename":
				if len(parts) > 1 {
					prog.FileName = RemoveQuotes(strings.TrimSpace(parts[1]))
				}
			case "entry":
				if len(parts) > 1 {
					prog.EntryPoint, _ = strconv.Atoi(strings.TrimSpace(parts[1]))
				}
			case "ver":
				if len(parts) > 1 {
					prog.Version = strings.TrimSpace(parts[1])
				}
			case "debug":
				prog.DebugLog = true
			case "dump_state_on_exit":
				prog.DumpStateOnExit = true
			}

		case secLabels:
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 && prog != nil {
				idx, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
				prog.Labels[strings.ToLower(strings.TrimSpace(parts[0]))] = idx
			}

		case secInstrs:
			parts := strings.SplitN(line, " ", 2)
			op := vm.LookupOpcode(parts[0])
			var args []string
			if len(parts) > 1 {
				args = SplitArgs(parts[1])
			}

			instr := vm.Instruction{Op: op, Args: args}

			// register labels at parse time
			if op == vm.OpLabel && len(args) > 0 && prog != nil {
				prog.Labels[strings.ToLower(args[0])] = len(prog.Instrs)
			}

			if prog != nil {
				prog.Instrs = append(prog.Instrs, instr)
			}
		}
	}

	storeProg()
	return res, scanner.Err()
}
