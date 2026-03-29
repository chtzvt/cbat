package vm

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand/v2"
	"os"
	"os/user"
	"strconv"
	"strings"
	"time"

	"BBS_VM/pkg/store"
)

type ExecState int

const (
	Idle ExecState = iota
	Running
	Terminated
	EOF
)

// Program represents a single loaded CBAT program.
type Program struct {
	FileName        string
	EntryPoint      int
	Version         string
	Instrs          []Instruction
	Labels          map[string]int
	DebugLog        bool
	DumpStateOnExit bool
}

// IOProvider abstracts stdin/stdout/stderr for embeddability.
type IOProvider interface {
	Stdout() io.Writer
	Stderr() io.Writer
	ReadLine() (string, error)
}

// StdIO is the default IOProvider using os stdin/stdout/stderr.
type StdIO struct {
	reader *io.Reader
}

func NewStdIO() *StdIO {
	r := io.Reader(os.Stdin)
	return &StdIO{reader: &r}
}

func (s *StdIO) Stdout() io.Writer { return os.Stdout }
func (s *StdIO) Stderr() io.Writer { return os.Stderr }
func (s *StdIO) ReadLine() (string, error) {
	var buf []byte
	b := make([]byte, 1)
	for {
		n, err := (*s.reader).Read(b)
		if n > 0 {
			if b[0] == '\n' {
				return string(buf), nil
			}
			buf = append(buf, b[0])
		}
		if err != nil {
			return string(buf), err
		}
	}
}

// StringIO is an IOProvider backed by strings (for testing).
type StringIO struct {
	inputs   []string
	inputIdx int
	out      strings.Builder
	errOut   strings.Builder
}

func NewStringIO(inputs []string) *StringIO {
	return &StringIO{inputs: inputs}
}

func (s *StringIO) Stdout() io.Writer { return &s.out }
func (s *StringIO) Stderr() io.Writer { return &s.errOut }
func (s *StringIO) ReadLine() (string, error) {
	if s.inputIdx < len(s.inputs) {
		line := s.inputs[s.inputIdx]
		s.inputIdx++
		return line, nil
	}
	return "", io.EOF
}
func (s *StringIO) Output() string    { return s.out.String() }
func (s *StringIO) ErrOutput() string { return s.errOut.String() }

// VM is the CBAT execution engine.
// Small integer string cache — avoids strconv.Itoa allocation for common values.
const intCacheSize = 100001

var smallIntStr [intCacheSize]string

func init() {
	for i := range smallIntStr {
		smallIntStr[i] = strconv.Itoa(i)
	}
	// pre-populate lowercase single-char lookup
	for i := 0; i < 26; i++ {
		upperSingleChar[i] = string(rune('a' + i))
	}
}

func fastItoa(n int) string {
	if n >= 0 && n < intCacheSize {
		return smallIntStr[n]
	}
	return strconv.Itoa(n)
}

// Pre-computed lowercase strings for single uppercase characters A-Z.
// Eliminates the #1 allocation hotspot: toLower("I"), toLower("X"), etc.
var upperSingleChar [26]string

// toLower converts a string to lowercase with zero allocation when possible.
func toLower(s string) string {
	// fast path: single uppercase letter (most common CBAT variable names)
	if len(s) == 1 {
		c := s[0]
		if c >= 'A' && c <= 'Z' {
			return upperSingleChar[c-'A']
		}
		return s // already lowercase or non-alpha
	}

	// fast path: check if already all lowercase
	needsLower := false
	for i := 0; i < len(s); i++ {
		if s[i] >= 'A' && s[i] <= 'Z' {
			needsLower = true
			break
		}
	}
	if !needsLower {
		return s
	}

	// slow path: actual conversion needed
	return strings.ToLower(s)
}

type VM struct {
	Store    store.Store
	Programs map[string]*Program
	IO       IOProvider

	pc          int
	state       ExecState
	raStack     []int
	instrCount  int
	currentProg *Program

	cachedUsername string
	interpBuf      strings.Builder

	GlobalEntry  string
	DebugStep    bool
	OnBreakpoint func(vm *VM)
}

func New(s store.Store, ioP IOProvider) *VM {
	u, err := user.Current()
	uname := "unknown"
	if err == nil {
		uname = u.Username
	}
	return &VM{
		Store:          s,
		Programs:       make(map[string]*Program),
		IO:             ioP,
		state:          Idle,
		cachedUsername:  uname,
	}
}

// LoadResult registers parsed programs and files into the VM.
// Programs are deep-copied so each VM instance has independent label maps.
func (vm *VM) LoadResult(globalEntry string, programs []*Program, files map[string]string) {
	vm.GlobalEntry = globalEntry
	for _, p := range programs {
		clone := p.Clone()
		key := strings.ToLower(strings.Trim(clone.FileName, "\""))
		vm.Programs[key] = clone
	}
	for name, content := range files {
		vm.Store.Set(store.Files, name, content)
	}
}

// Clone creates a deep copy of a Program (independent label map, shared instruction slice).
func (p *Program) Clone() *Program {
	labels := make(map[string]int, len(p.Labels))
	for k, v := range p.Labels {
		labels[k] = v
	}
	return &Program{
		FileName:        p.FileName,
		EntryPoint:      p.EntryPoint,
		Version:         p.Version,
		Instrs:          p.Instrs, // instructions are read-only, safe to share
		Labels:          labels,
		DebugLog:        p.DebugLog,
		DumpStateOnExit: p.DumpStateOnExit,
	}
}

func (vm *VM) Run() error {
	prog := vm.resolveEntryProgram()
	if prog == nil {
		return fmt.Errorf("no entry program found")
	}
	return vm.runProgram(prog)
}

func (vm *VM) resolveEntryProgram() *Program {
	if vm.GlobalEntry != "" {
		key := strings.ToLower(strings.Trim(vm.GlobalEntry, "\""))
		if p, ok := vm.Programs[key]; ok {
			return p
		}
	}
	if p, ok := vm.Programs["main"]; ok {
		return p
	}
	for _, p := range vm.Programs {
		return p
	}
	return nil
}

func (vm *VM) runProgram(prog *Program) error {
	prev := vm.currentProg
	prevPC := vm.pc
	prevState := vm.state
	prevRA := vm.raStack

	vm.currentProg = prog
	vm.pc = prog.EntryPoint
	vm.state = Running
	vm.raStack = vm.raStack[:0] // reuse backing array

	instrs := prog.Instrs
	nInstrs := len(instrs)

	for vm.state == Running {
		if vm.pc >= nInstrs {
			vm.state = EOF
			break
		}

		vm.instrCount++

		// Check for Ctrl+C / Ctrl+T every 1024 instructions
		if vm.instrCount&0x3FF == 0 {
			if tio, ok := vm.IO.(*TerminalIO); ok {
				if tio.CheckInterrupt() {
					vm.state = Terminated
					break
				}
				if tio.CheckDebug() {
					tio.showDebugMenu()
				}
			}
		}

		instr := &instrs[vm.pc]

		if vm.DebugStep {
			fmt.Fprintf(vm.IO.Stderr(), "[step] %d | %s\n", vm.pc, vm.instrToString(*instr))
		}

		switch instr.Op {
		case OpTerminate:
			vm.state = Terminated

		case OpLabel:
			// labels are pre-resolved at parse time; runtime is a no-op
			vm.pc++

		case OpGoto:
			vm.pc = vm.resolveLabel(vm.interpolate(instr.Args[0]))

		case OpGotoAddr:
			vm.pc = vm.interpolateInt(instr.Args[0])

		case OpGotoSub:
			vm.raStack = append(vm.raStack, vm.pc+1)
			vm.pc = vm.resolveLabel(vm.interpolate(instr.Args[0]))

		case OpReturn:
			n := len(vm.raStack)
			if n > 0 {
				vm.pc = vm.raStack[n-1]
				vm.raStack = vm.raStack[:n-1]
			} else {
				ra, _ := vm.Store.Get(store.Variables, "ra")
				vm.pc, _ = strconv.Atoi(ra)
			}

		case OpBreakpoint:
			if vm.OnBreakpoint != nil {
				vm.OnBreakpoint(vm)
			}
			vm.pc++

		case OpIfEq, OpIfNotEq, OpIfEqInt, OpIfNotEqInt,
			OpIfGteInt, OpIfLteInt, OpIfGtInt, OpIfLtInt,
			OpIfFileExists, OpIfNotFileExists:
			vm.pc = vm.execConditional(instr)

		case OpCall:
			name := vm.interpolate(instr.Args[0])
			sub := vm.resolveProgram(name)
			if sub == nil {
				fmt.Fprintf(vm.IO.Stderr(), "cbat: subroutine '%s' not found\n", name)
				vm.pc++
				continue
			}
			vm.runProgram(sub)
			vm.pc++

		default:
			vm.execInstruction(instr)
			vm.pc++
		}
	}

	// Write PC and RA to store only when needed (state dump / debugger)
	if prog.DumpStateOnExit {
		vm.Store.Set(store.Variables, "pc", strconv.Itoa(vm.pc))
		vm.DumpState()
	}

	vm.currentProg = prev
	vm.pc = prevPC
	vm.state = prevState
	vm.raStack = prevRA

	return nil
}

// resolveProgram finds a program by name, trying full path first then basename.
func (vm *VM) resolveProgram(name string) *Program {
	key := strings.ToLower(strings.Trim(name, "\""))
	if p, ok := vm.Programs[key]; ok {
		return p
	}
	// Extract basename: last component after \ or /
	base := key
	if idx := strings.LastIndexAny(key, "\\/"); idx >= 0 {
		base = key[idx+1:]
	}
	if p, ok := vm.Programs[base]; ok {
		return p
	}
	return nil
}

func (vm *VM) resolveLabel(name string) int {
	// fast path: labels are already lowercased at parse time
	if idx, ok := vm.currentProg.Labels[name]; ok {
		return idx
	}
	// try lowercase (for interpolated labels)
	lower := toLower(name)
	if idx, ok := vm.currentProg.Labels[lower]; ok {
		return idx
	}
	if addr, err := strconv.Atoi(name); err == nil {
		return addr
	}
	return 0
}

// interpolate replaces %VAR% references in a string.
// Fast path: if no '%' is present, returns the string unchanged (zero alloc).
// Handles %% as a literal % (batch escape convention).
func (vm *VM) interpolate(s string) string {
	// fast path: no interpolation needed
	if strings.IndexByte(s, '%') < 0 {
		return s
	}

	vm.interpBuf.Reset()
	i := 0
	for i < len(s) {
		pct := strings.IndexByte(s[i:], '%')
		if pct < 0 {
			vm.interpBuf.WriteString(s[i:])
			break
		}
		pct += i
		vm.interpBuf.WriteString(s[i:pct])

		// Try to find a closing % for a variable reference
		end := strings.IndexByte(s[pct+1:], '%')
		if end < 0 {
			// No closing % — emit literal % and continue
			vm.interpBuf.WriteByte('%')
			i = pct + 1
			continue
		}
		end += pct + 1

		name := s[pct+1 : end]

		// Empty name (%%) or name with spaces — literal %
		if name == "" || strings.ContainsAny(name, " \t") {
			vm.interpBuf.WriteByte('%')
			i = pct + 1
			continue
		}

		// Valid variable name — resolve it
		vm.interpBuf.WriteString(vm.getVar(name))
		i = end + 1
	}
	return vm.interpBuf.String()
}

func isVarChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_'
}

// interpolateInt is a fast path for when we know the result should be an int.
func (vm *VM) interpolateInt(s string) int {
	v, _ := strconv.Atoi(vm.interpolate(s))
	return v
}

// getVar retrieves a variable value. The name is assumed to be
// the raw content between %...% delimiters (not yet lowercased).
func (vm *VM) getVar(name string) string {
	lower := toLower(name)
	switch lower {
	case "date":
		return time.Now().Format("2006-01-02")
	case "time":
		return time.Now().Format("15:04:05")
	case "random":
		return strconv.Itoa(rand.IntN(32768))
	case "username":
		return vm.cachedUsername
	case "pc":
		return fastItoa(vm.pc)
	case "ra":
		n := len(vm.raStack)
		if n > 0 {
			return strconv.Itoa(vm.raStack[n-1])
		}
		if v, ok := vm.Store.Get(store.Variables, "ra"); ok {
			return v
		}
		return "undefined"
	default:
		if v, ok := vm.Store.Get(store.Variables, lower); ok {
			return v
		}
		return "undefined"
	}
}

func (vm *VM) setVar(name string, value string) {
	vm.Store.Set(store.Variables, toLower(name), value)
}

func (vm *VM) StateHash() map[string]interface{} {
	// write PC to store for snapshot
	vm.Store.Set(store.Variables, "pc", strconv.Itoa(vm.pc))
	return map[string]interface{}{
		"exit_reason":           vm.stateString(),
		"pc":                    vm.pc,
		"instructions_executed": vm.instrCount,
		"variables":             vm.Store.Snapshot(store.Variables),
		"labels":                vm.currentProg.Labels,
		"files":                 vm.Store.Snapshot(store.Files),
	}
}

func (vm *VM) stateString() string {
	switch vm.state {
	case Terminated:
		return "terminated"
	case EOF:
		return "eof"
	default:
		return "running"
	}
}

func (vm *VM) DumpState() {
	fmt.Fprintln(vm.IO.Stderr(), "---CBAT_STATE_BEGIN---")
	data, _ := json.Marshal(vm.StateHash())
	fmt.Fprintln(vm.IO.Stderr(), string(data))
	fmt.Fprintln(vm.IO.Stderr(), "---CBAT_STATE_END---")
}

// CurrentPC returns the current program counter (for debugger).
func (vm *VM) CurrentPC() int { return vm.pc }

// SetPC sets the program counter (for debugger).
func (vm *VM) SetPC(addr int) { vm.pc = addr }

// CurrentLabels returns the label map of the current program.
func (vm *VM) CurrentLabels() map[string]int {
	if vm.currentProg != nil {
		return vm.currentProg.Labels
	}
	return nil
}

// SetLabel sets a label in the current program.
func (vm *VM) SetLabel(name string, addr int) {
	if vm.currentProg != nil {
		vm.currentProg.Labels[strings.ToLower(name)] = addr
	}
}

// ForceTerminate stops the VM from the debugger.
func (vm *VM) ForceTerminate() { vm.state = Terminated }

// enterDebugger invokes the breakpoint handler (debugger REPL).
func (vm *VM) enterDebugger() {
	if vm.OnBreakpoint != nil {
		vm.OnBreakpoint(vm)
	}
}


func (vm *VM) instrToString(i Instruction) string {
	for mnem, op := range mnemonicToOpcode {
		if op == i.Op {
			if len(i.Args) > 0 {
				return mnem + " " + strings.Join(i.Args, ",")
			}
			return mnem
		}
	}
	return "nop"
}
