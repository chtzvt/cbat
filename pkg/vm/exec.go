package vm

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"BBS_VM/pkg/store"
)

// execInstruction dispatches non-flow-control instructions.
func (vm *VM) execInstruction(instr *Instruction) {
	switch instr.Op {
	// I/O
	case OpEcho:
		vm.execEcho(instr)
	case OpSetPrompt:
		vm.execSetPrompt(instr)
	case OpPause:
		vm.execPause(instr)
	case OpClearScreen:
		// Home cursor, erase screen, erase scrollback — fills entire viewport with current bg
		fmt.Fprint(vm.IO.Stdout(), "\033[H\033[2J\033[3J")
	case OpTypeFile:
		vm.execTypeFile(instr)
	case OpTypePaged:
		vm.execTypePaged(instr)
	case OpColor:
		vm.execColor(instr)
	case OpTitle:
		title := vm.interpolate(instr.Args[0])
		fmt.Fprintf(vm.IO.Stdout(), "\033]0;%s\a", title)
	case OpSleep:
		ms, _ := strconv.Atoi(vm.interpolate(instr.Args[0]))
		time.Sleep(time.Duration(ms) * time.Millisecond)

	// Variables
	case OpStore:
		vm.setVar(instr.Args[0], vm.interpolate(instr.Args[1]))
	case OpStoreFromFile:
		path := vm.interpolate(instr.Args[1])
		if content, ok := vm.Store.Get(store.Files, path); ok {
			// Match batch "set /p VAR=<file" semantics: read first line only
			if nl := strings.Index(content, "\n"); nl >= 0 {
				vm.setVar(instr.Args[0], content[:nl])
			} else {
				vm.setVar(instr.Args[0], content)
			}
		} else {
			vm.setVar(instr.Args[0], "")
		}
	case OpStoreFromFileIdx:
		path := vm.interpolate(instr.Args[1])
		idx, _ := strconv.Atoi(vm.interpolate(instr.Args[2]))
		content, ok := vm.Store.Get(store.Files, path)
		if ok && idx < len(content) && idx >= 0 {
			vm.setVar(instr.Args[0], string(content[idx]))
		}
	case OpStoreArith:
		expr := vm.interpolate(instr.Args[1])
		result := evalArithmetic(expr)
		vm.setVar(instr.Args[0], fastItoa(result))

	// File operations
	case OpAppendFile:
		content := vm.interpolate(instr.Args[0])
		path := vm.interpolate(instr.Args[1])
		existing, _ := vm.Store.Get(store.Files, path)
		vm.Store.Set(store.Files, path, existing+content+"\n")
	case OpWriteFile:
		content := vm.interpolate(instr.Args[0])
		path := vm.interpolate(instr.Args[1])
		vm.Store.Set(store.Files, path, content)
	case OpWriteFileIdx:
		path := vm.interpolate(instr.Args[0])
		idx, _ := strconv.Atoi(vm.interpolate(instr.Args[1]))
		value := vm.interpolate(instr.Args[2])
		content, _ := vm.Store.Get(store.Files, path)
		if idx >= 0 && idx < len(content) {
			content = content[:idx] + value + content[idx+1:]
		} else if idx == len(content) {
			content = content + value
		}
		vm.Store.Set(store.Files, path, content)
	case OpDeleteFile:
		path := vm.interpolate(instr.Args[0])
		vm.Store.Delete(store.Files, path)
	case OpMakeDir:
		path := vm.interpolate(instr.Args[0])
		vm.Store.Set(store.Files, path, "")

	// Math immediate — inlined to avoid closure allocation
	case OpAddImm:
		name := vm.interpolate(instr.Args[0])
		cur, _ := strconv.Atoi(vm.getVar(name))
		val, _ := strconv.Atoi(vm.interpolate(instr.Args[1]))
		vm.setVar(name, fastItoa(cur+val))
	case OpSubImm:
		name := vm.interpolate(instr.Args[0])
		cur, _ := strconv.Atoi(vm.getVar(name))
		val, _ := strconv.Atoi(vm.interpolate(instr.Args[1]))
		vm.setVar(name, fastItoa(cur-val))
	case OpMulImm:
		name := vm.interpolate(instr.Args[0])
		cur, _ := strconv.Atoi(vm.getVar(name))
		val, _ := strconv.Atoi(vm.interpolate(instr.Args[1]))
		vm.setVar(name, fastItoa(cur*val))
	case OpDivImm:
		name := vm.interpolate(instr.Args[0])
		cur, _ := strconv.Atoi(vm.getVar(name))
		val, _ := strconv.Atoi(vm.interpolate(instr.Args[1]))
		if val == 0 {
			vm.setVar(name, "0")
		} else {
			vm.setVar(name, fastItoa(cur/val))
		}
	case OpModImm:
		name := vm.interpolate(instr.Args[0])
		cur, _ := strconv.Atoi(vm.getVar(name))
		val, _ := strconv.Atoi(vm.interpolate(instr.Args[1]))
		if val == 0 {
			vm.setVar(name, "0")
		} else {
			vm.setVar(name, fastItoa(cur%val))
		}

	// Math register-to-register — inlined
	case OpAdd:
		dest := vm.interpolate(instr.Args[0])
		a, _ := strconv.Atoi(vm.getVar(vm.interpolate(instr.Args[1])))
		b, _ := strconv.Atoi(vm.getVar(vm.interpolate(instr.Args[2])))
		vm.setVar(dest, fastItoa(a+b))
	case OpSub:
		dest := vm.interpolate(instr.Args[0])
		a, _ := strconv.Atoi(vm.getVar(vm.interpolate(instr.Args[1])))
		b, _ := strconv.Atoi(vm.getVar(vm.interpolate(instr.Args[2])))
		vm.setVar(dest, fastItoa(a-b))
	case OpMul:
		dest := vm.interpolate(instr.Args[0])
		a, _ := strconv.Atoi(vm.getVar(vm.interpolate(instr.Args[1])))
		b, _ := strconv.Atoi(vm.getVar(vm.interpolate(instr.Args[2])))
		vm.setVar(dest, fastItoa(a*b))
	case OpDiv:
		dest := vm.interpolate(instr.Args[0])
		a, _ := strconv.Atoi(vm.getVar(vm.interpolate(instr.Args[1])))
		b, _ := strconv.Atoi(vm.getVar(vm.interpolate(instr.Args[2])))
		if b == 0 {
			vm.setVar(dest, "0")
		} else {
			vm.setVar(dest, fastItoa(a/b))
		}
	case OpMod:
		dest := vm.interpolate(instr.Args[0])
		a, _ := strconv.Atoi(vm.getVar(vm.interpolate(instr.Args[1])))
		b, _ := strconv.Atoi(vm.getVar(vm.interpolate(instr.Args[2])))
		if b == 0 {
			vm.setVar(dest, "0")
		} else {
			vm.setVar(dest, fastItoa(a%b))
		}

	// String
	case OpConcat:
		a := vm.getVar(vm.interpolate(instr.Args[1]))
		b := vm.getVar(vm.interpolate(instr.Args[2]))
		vm.setVar(instr.Args[0], a+b)
	case OpLength:
		src := vm.getVar(vm.interpolate(instr.Args[1]))
		vm.setVar(instr.Args[0], fastItoa(len(src)))
	case OpAtoI:
		ch := vm.interpolate(instr.Args[1])
		if len(ch) > 0 {
			vm.setVar(instr.Args[0], fastItoa(int(ch[0])))
		}
	case OpItoA:
		val, _ := strconv.Atoi(vm.interpolate(instr.Args[1]))
		vm.setVar(instr.Args[0], string(rune(val)))

	case OpNop:
		// nothing
	}
}

// execEcho prints a string to stdout.
// Appends \033[K (erase to end of line) so the background color fills the full terminal width.
func (vm *VM) execEcho(instr *Instruction) {
	msg := ""
	if len(instr.Args) > 0 {
		msg = vm.interpolate(instr.Args[0])
	}
	fmt.Fprintf(vm.IO.Stdout(), "%s\033[K\n", msg)
}

// execSetPrompt prompts user for input and stores the result.
// The special input "/cbat:dump" triggers a state dump and terminates the VM.
func (vm *VM) execSetPrompt(instr *Instruction) {
	prompt := "Input: "
	if len(instr.Args) > 1 && instr.Args[1] != "" {
		prompt = vm.interpolate(instr.Args[1])
	}
	// \033[J fills everything below the prompt with the current background color
	fmt.Fprintf(vm.IO.Stdout(), "%s\033[J", prompt)
	line, err := vm.IO.ReadLine()
	if err != nil {
		vm.state = Terminated
		return
	}
	if line == "/cbat:dump" {
		vm.Store.Set(store.Variables, "pc", fastItoa(vm.pc))
		vm.DumpState()
		vm.state = Terminated
		return
	}
	vm.setVar(instr.Args[0], line)
}

// execPause waits for user input.
func (vm *VM) execPause(instr *Instruction) {
	quiet := len(instr.Args) > 0 && instr.Args[0] == "0"
	if !quiet {
		fmt.Fprintf(vm.IO.Stdout(), "Press any key to continue...\033[J")
	} else {
		fmt.Fprint(vm.IO.Stdout(), "\033[J")
	}
	if _, err := vm.IO.ReadLine(); err != nil {
		vm.state = Terminated
	}
}

// execTypeFile prints file contents to stdout.
func (vm *VM) execTypeFile(instr *Instruction) {
	path := vm.interpolate(instr.Args[0])
	if content, ok := vm.Store.Get(store.Files, path); ok {
		fmt.Fprint(vm.IO.Stdout(), content)
	}
}

// execTypePaged prints file contents with pagination.
func (vm *VM) execTypePaged(instr *Instruction) {
	path := vm.interpolate(instr.Args[0])
	linesPerPage := 20
	if len(instr.Args) > 1 {
		linesPerPage, _ = strconv.Atoi(vm.interpolate(instr.Args[1]))
	}
	content, ok := vm.Store.Get(store.Files, path)
	if !ok {
		return
	}
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		fmt.Fprintln(vm.IO.Stdout(), line)
		if (i+1)%linesPerPage == 0 && i+1 < len(lines) {
			fmt.Fprint(vm.IO.Stdout(), "-- MORE --")
			vm.IO.ReadLine()
		}
	}
}

// execColor sets terminal color using ANSI escapes.
// batch color code → ANSI color number mapping
var batchColorMap = map[byte]string{
	'0': "0", '1': "4", '2': "2", '3': "6",
	'4': "1", '5': "5", '6': "3", '7': "7",
	'8': "0;1", '9': "4;1", 'a': "2;1", 'b': "6;1",
	'c': "1;1", 'd': "5;1", 'e': "3;1", 'f': "7;1",
}

func (vm *VM) execColor(instr *Instruction) {
	if len(instr.Args) == 0 {
		return
	}
	code := strings.ToLower(vm.interpolate(instr.Args[0]))
	if len(code) == 2 {
		bg, ok1 := batchColorMap[code[0]]
		fg, ok2 := batchColorMap[code[1]]
		if !ok1 || !ok2 {
			return
		}
		bgBase := strings.Split(bg, ";")[0]
		// Set colors + clear entire screen to fill with new background
		// (matches CMD.EXE behavior where "color" repaints the whole console)
		fmt.Fprintf(vm.IO.Stdout(), "\033[4%sm\033[3%sm\033[H\033[2J", bgBase, fg)
	}
}

// condLeft resolves the left operand of a conditional.
// Simple variable names are looked up with getVar; compound expressions
// containing % (like %TL%+%TM%+%TR%) are interpolated.
func (vm *VM) condLeft(arg string) string {
	if strings.IndexByte(arg, '%') >= 0 {
		return vm.interpolate(arg)
	}
	return vm.getVar(arg)
}

// execConditional evaluates a conditional instruction and returns the next PC.
func (vm *VM) execConditional(instr *Instruction) int {
	var condTrue bool

	switch instr.Op {
	case OpIfEq:
		condTrue = vm.condLeft(instr.Args[0]) == vm.interpolate(instr.Args[1])
	case OpIfNotEq:
		condTrue = vm.condLeft(instr.Args[0]) != vm.interpolate(instr.Args[1])
	case OpIfEqInt:
		a, _ := strconv.Atoi(vm.condLeft(instr.Args[0]))
		b, _ := strconv.Atoi(vm.interpolate(instr.Args[1]))
		condTrue = a == b
	case OpIfNotEqInt:
		a, _ := strconv.Atoi(vm.condLeft(instr.Args[0]))
		b, _ := strconv.Atoi(vm.interpolate(instr.Args[1]))
		condTrue = a != b
	case OpIfGteInt:
		a, _ := strconv.Atoi(vm.condLeft(instr.Args[0]))
		b, _ := strconv.Atoi(vm.interpolate(instr.Args[1]))
		condTrue = a >= b
	case OpIfLteInt:
		a, _ := strconv.Atoi(vm.condLeft(instr.Args[0]))
		b, _ := strconv.Atoi(vm.interpolate(instr.Args[1]))
		condTrue = a <= b
	case OpIfGtInt:
		a, _ := strconv.Atoi(vm.condLeft(instr.Args[0]))
		b, _ := strconv.Atoi(vm.interpolate(instr.Args[1]))
		condTrue = a > b
	case OpIfLtInt:
		a, _ := strconv.Atoi(vm.condLeft(instr.Args[0]))
		b, _ := strconv.Atoi(vm.interpolate(instr.Args[1]))
		condTrue = a < b
	case OpIfFileExists:
		path := vm.interpolate(instr.Args[0])
		condTrue = vm.Store.Exists(store.Files, path)
	case OpIfNotFileExists:
		path := vm.interpolate(instr.Args[0])
		condTrue = !vm.Store.Exists(store.Files, path)
	}

	// determine label arg (last arg for 2-arg conditionals, 3rd arg for 3-arg)
	var label string
	switch instr.Op {
	case OpIfFileExists, OpIfNotFileExists:
		if len(instr.Args) > 1 {
			label = instr.Args[1]
		}
	default:
		if len(instr.Args) > 2 {
			label = instr.Args[2]
		}
	}

	if condTrue {
		if label != "" {
			return vm.resolveLabel(label)
		}
		return vm.pc + 1
	}
	if label != "" {
		return vm.pc + 1
	}
	return vm.pc + 2
}

// evalArithmetic parses and evaluates a simple arithmetic expression.
func evalArithmetic(expr string) int {
	tokens := tokenize(expr)
	if len(tokens) == 0 {
		return 0
	}

	// First pass: * / %
	for i := 1; i < len(tokens)-1; i += 2 {
		if tokens[i] == "*" || tokens[i] == "/" || tokens[i] == "%" {
			a, _ := strconv.Atoi(tokens[i-1])
			b, _ := strconv.Atoi(tokens[i+1])
			var result int
			switch tokens[i] {
			case "*":
				result = a * b
			case "/":
				if b != 0 {
					result = a / b
				}
			case "%":
				if b != 0 {
					result = a % b
				}
			}
			tokens = append(tokens[:i-1], append([]string{strconv.Itoa(result)}, tokens[i+2:]...)...)
			i -= 2
		}
	}

	// Second pass: + -
	result, _ := strconv.Atoi(tokens[0])
	for i := 1; i < len(tokens)-1; i += 2 {
		val, _ := strconv.Atoi(tokens[i+1])
		switch tokens[i] {
		case "+":
			result += val
		case "-":
			result -= val
		}
	}
	return result
}

func tokenize(expr string) []string {
	var tokens []string
	var num strings.Builder
	for _, ch := range expr {
		if ch >= '0' && ch <= '9' || (ch == '-' && num.Len() == 0 && (len(tokens) == 0 || isOp(tokens[len(tokens)-1]))) {
			num.WriteRune(ch)
		} else if ch == '+' || ch == '-' || ch == '*' || ch == '/' || ch == '%' {
			if num.Len() > 0 {
				tokens = append(tokens, num.String())
				num.Reset()
			}
			tokens = append(tokens, string(ch))
		}
	}
	if num.Len() > 0 {
		tokens = append(tokens, num.String())
	}
	return tokens
}

func isOp(s string) bool {
	return s == "+" || s == "-" || s == "*" || s == "/" || s == "%"
}
