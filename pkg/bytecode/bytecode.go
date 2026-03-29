// Package bytecode provides a binary encoding for CBAT programs.
//
// File format (.cbatc):
//
//	Header:
//	  [4]  magic:   "CBAT"
//	  [2]  version: uint16 LE (currently 1)
//	  [2]  flags:   uint16 LE (bit 0 = dump_state_on_exit, bit 1 = debug)
//
//	String table:
//	  [4]  count:   uint32 LE — number of interned strings
//	  For each string:
//	    [2]  length: uint16 LE
//	    [N]  data:   UTF-8 bytes (no null terminator)
//
//	File table (seed files from .files section):
//	  [4]  count:   uint32 LE
//	  For each file:
//	    [2]  name:   uint16 LE — string table index
//	    [2]  value:  uint16 LE — string table index
//
//	Program table:
//	  [4]  count:         uint32 LE — number of programs
//	  [2]  global_entry:  uint16 LE — string table index (0xFFFF = none)
//	  For each program:
//	    [2]  filename:    uint16 LE — string table index
//	    [2]  entry_point: uint16 LE
//	    [4]  num_labels:  uint32 LE
//	    For each label:
//	      [2]  name:  uint16 LE — string table index
//	      [2]  addr:  uint16 LE
//	    [4]  num_instrs:  uint32 LE
//	    For each instruction:
//	      [1]  opcode:    uint8
//	      [1]  num_args:  uint8
//	      For each arg:
//	        [2]  arg: uint16 LE — string table index
package bytecode

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strings"

	"BBS_VM/pkg/parser"
	"BBS_VM/pkg/vm"
)

var Magic = [4]byte{'C', 'B', 'A', 'T'}

const Version = 1

const (
	FlagDumpState uint16 = 1 << iota
	FlagDebug
)

// Encode writes a parsed CBAT program to binary bytecode.
func Encode(w io.Writer, result *parser.Result) error {
	// Build string table
	st := newStringTable()
	globalEntryIdx := uint16(0xFFFF)
	if result.GlobalEntry != "" {
		globalEntryIdx = st.intern(result.GlobalEntry)
	}

	// Pre-intern all strings from files, programs, labels, instructions
	type encodedFile struct{ name, value uint16 }
	var files []encodedFile
	for name, content := range result.Files {
		files = append(files, encodedFile{st.intern(name), st.intern(content)})
	}

	type encodedLabel struct {
		name uint16
		addr uint16
	}
	type encodedInstr struct {
		opcode uint8
		args   []uint16
	}
	type encodedProgram struct {
		filename   uint16
		entryPoint uint16
		flags      uint16
		labels     []encodedLabel
		instrs     []encodedInstr
	}

	var programs []encodedProgram
	for _, p := range result.Programs {
		ep := encodedProgram{
			filename:   st.intern(p.FileName),
			entryPoint: uint16(p.EntryPoint),
		}
		if p.DumpStateOnExit {
			ep.flags |= FlagDumpState
		}
		if p.DebugLog {
			ep.flags |= FlagDebug
		}
		for name, addr := range p.Labels {
			ep.labels = append(ep.labels, encodedLabel{st.intern(name), uint16(addr)})
		}
		for _, instr := range p.Instrs {
			ei := encodedInstr{opcode: uint8(instr.Op)}
			for _, arg := range instr.Args {
				ei.args = append(ei.args, st.intern(arg))
			}
			ep.instrs = append(ep.instrs, ei)
		}
		programs = append(programs, ep)
	}

	// Write header
	buf := &bytes.Buffer{}
	buf.Write(Magic[:])
	binary.Write(buf, binary.LittleEndian, uint16(Version))
	binary.Write(buf, binary.LittleEndian, uint16(0)) // reserved flags

	// Write string table
	binary.Write(buf, binary.LittleEndian, uint32(len(st.strings)))
	for _, s := range st.strings {
		binary.Write(buf, binary.LittleEndian, uint16(len(s)))
		buf.WriteString(s)
	}

	// Write file table
	binary.Write(buf, binary.LittleEndian, uint32(len(files)))
	for _, f := range files {
		binary.Write(buf, binary.LittleEndian, f.name)
		binary.Write(buf, binary.LittleEndian, f.value)
	}

	// Write program table
	binary.Write(buf, binary.LittleEndian, uint32(len(programs)))
	binary.Write(buf, binary.LittleEndian, globalEntryIdx)
	for _, p := range programs {
		binary.Write(buf, binary.LittleEndian, p.filename)
		binary.Write(buf, binary.LittleEndian, p.entryPoint)
		binary.Write(buf, binary.LittleEndian, p.flags)
		binary.Write(buf, binary.LittleEndian, uint32(len(p.labels)))
		for _, l := range p.labels {
			binary.Write(buf, binary.LittleEndian, l.name)
			binary.Write(buf, binary.LittleEndian, l.addr)
		}
		binary.Write(buf, binary.LittleEndian, uint32(len(p.instrs)))
		for _, i := range p.instrs {
			buf.WriteByte(i.opcode)
			buf.WriteByte(uint8(len(i.args)))
			for _, a := range i.args {
				binary.Write(buf, binary.LittleEndian, a)
			}
		}
	}

	_, err := w.Write(buf.Bytes())
	return err
}

// Decode reads binary bytecode and returns the equivalent of a parser.Result.
func Decode(r io.Reader) (*parser.Result, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	buf := bytes.NewReader(data)

	// Header
	var magic [4]byte
	binary.Read(buf, binary.LittleEndian, &magic)
	if magic != Magic {
		return nil, fmt.Errorf("not a CBAT bytecode file")
	}
	var version, flags uint16
	binary.Read(buf, binary.LittleEndian, &version)
	binary.Read(buf, binary.LittleEndian, &flags)
	if version != Version {
		return nil, fmt.Errorf("unsupported bytecode version %d", version)
	}

	// String table
	var strCount uint32
	binary.Read(buf, binary.LittleEndian, &strCount)
	strs := make([]string, strCount)
	for i := uint32(0); i < strCount; i++ {
		var slen uint16
		binary.Read(buf, binary.LittleEndian, &slen)
		sb := make([]byte, slen)
		buf.Read(sb)
		strs[i] = string(sb)
	}

	str := func(idx uint16) string {
		if int(idx) < len(strs) {
			return strs[idx]
		}
		return ""
	}

	// File table
	var fileCount uint32
	binary.Read(buf, binary.LittleEndian, &fileCount)
	files := make(map[string]string, fileCount)
	for i := uint32(0); i < fileCount; i++ {
		var nameIdx, valIdx uint16
		binary.Read(buf, binary.LittleEndian, &nameIdx)
		binary.Read(buf, binary.LittleEndian, &valIdx)
		files[str(nameIdx)] = str(valIdx)
	}

	// Program table
	var progCount uint32
	binary.Read(buf, binary.LittleEndian, &progCount)
	var globalEntryIdx uint16
	binary.Read(buf, binary.LittleEndian, &globalEntryIdx)

	globalEntry := ""
	if globalEntryIdx != 0xFFFF {
		globalEntry = str(globalEntryIdx)
	}

	programs := make([]*vm.Program, progCount)
	for p := uint32(0); p < progCount; p++ {
		var filenameIdx, entryPoint, pflags uint16
		binary.Read(buf, binary.LittleEndian, &filenameIdx)
		binary.Read(buf, binary.LittleEndian, &entryPoint)
		binary.Read(buf, binary.LittleEndian, &pflags)

		var labelCount uint32
		binary.Read(buf, binary.LittleEndian, &labelCount)
		labels := make(map[string]int, labelCount)
		for l := uint32(0); l < labelCount; l++ {
			var nameIdx, addr uint16
			binary.Read(buf, binary.LittleEndian, &nameIdx)
			binary.Read(buf, binary.LittleEndian, &addr)
			labels[str(nameIdx)] = int(addr)
		}

		var instrCount uint32
		binary.Read(buf, binary.LittleEndian, &instrCount)
		instrs := make([]vm.Instruction, instrCount)
		for i := uint32(0); i < instrCount; i++ {
			var opcode, numArgs uint8
			binary.Read(buf, binary.LittleEndian, &opcode)
			binary.Read(buf, binary.LittleEndian, &numArgs)
			args := make([]string, numArgs)
			for a := uint8(0); a < numArgs; a++ {
				var argIdx uint16
				binary.Read(buf, binary.LittleEndian, &argIdx)
				args[a] = str(argIdx)
			}
			instrs[i] = vm.Instruction{Op: vm.Opcode(opcode), Args: args}
		}

		programs[p] = &vm.Program{
			FileName:        str(filenameIdx),
			EntryPoint:      int(entryPoint),
			Labels:          labels,
			Instrs:          instrs,
			DumpStateOnExit: pflags&FlagDumpState != 0,
			DebugLog:        pflags&FlagDebug != 0,
		}
	}

	return &parser.Result{
		GlobalEntry: globalEntry,
		Programs:    programs,
		Files:       files,
	}, nil
}

// EncodeFile writes bytecode to a file.
func EncodeFile(path string, result *parser.Result) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return Encode(f, result)
}

// DecodeFile reads bytecode from a file.
func DecodeFile(path string) (*parser.Result, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return Decode(f)
}

// IsBytecode checks if the first 4 bytes of a file are the CBAT magic.
func IsBytecode(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	var magic [4]byte
	n, _ := f.Read(magic[:])
	return n == 4 && magic == Magic
}

// CachedPath returns the .cbatc path for a given .cbat source file.
func CachedPath(sourcePath string) string {
	return strings.TrimSuffix(sourcePath, ".cbat") + ".cbatc"
}

// --- string table ---

type stringTable struct {
	strings []string
	index   map[string]uint16
}

func newStringTable() *stringTable {
	return &stringTable{index: make(map[string]uint16)}
}

func (st *stringTable) intern(s string) uint16 {
	if idx, ok := st.index[s]; ok {
		return idx
	}
	idx := uint16(len(st.strings))
	st.strings = append(st.strings, s)
	st.index[s] = idx
	return idx
}
