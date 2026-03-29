package compiler

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// CompiledProgram represents a single compiled batch file.
type CompiledProgram struct {
	FileName string
	Lines    []string // CBAT instruction lines
}

// CompileFile compiles a single .bat file to CBAT instructions.
func CompileFile(path string) (*CompiledProgram, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return Compile(f, filepath.Base(path))
}

// Compile compiles batch source from a reader.
func Compile(r io.Reader, filename string) (*CompiledProgram, error) {
	var nodes []Node
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		node := ParseLine(scanner.Text())
		if node != nil {
			nodes = append(nodes, node)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	lines := LowerProgram(nodes)
	return &CompiledProgram{
		FileName: filename,
		Lines:    lines,
	}, nil
}

// EmitCBAT writes one or more compiled programs as a .cbat file.
func EmitCBAT(w io.Writer, programs []*CompiledProgram, entryPoint string, seedFiles map[string]string) {
	if entryPoint != "" {
		fmt.Fprintf(w, ".global\n")
		fmt.Fprintf(w, "    entry \"%s\"\n\n", entryPoint)
	}

	if len(seedFiles) > 0 {
		fmt.Fprintf(w, ".files\n")
		for name, content := range seedFiles {
			escaped := strings.ReplaceAll(content, "\n", "\\n")
			fmt.Fprintf(w, "    \"%s\",\"%s\"\n", name, escaped)
		}
		fmt.Fprintln(w)
	}

	for _, prog := range programs {
		fmt.Fprintf(w, ".header\n")
		fmt.Fprintf(w, "    filename \"%s\"\n", prog.FileName)
		fmt.Fprintf(w, "    entry 0\n")
		fmt.Fprintf(w, "    ver 1.0\n")
		fmt.Fprintf(w, ".instrs\n")

		for _, line := range prog.Lines {
			fmt.Fprintf(w, "    %s\n", line)
		}
		fmt.Fprintln(w)
	}
}

// CompileFiles compiles multiple .bat files and emits a single .cbat.
func CompileFiles(paths []string, output io.Writer) error {
	var programs []*CompiledProgram

	for _, path := range paths {
		prog, err := CompileFile(path)
		if err != nil {
			return fmt.Errorf("compiling %s: %w", path, err)
		}
		programs = append(programs, prog)
	}

	entry := ""
	if len(programs) > 1 {
		entry = programs[0].FileName
	}

	EmitCBAT(output, programs, entry, nil)
	return nil
}

// CompileString compiles a batch source string. Convenience for testing.
func CompileString(source string, filename string) (*CompiledProgram, error) {
	return Compile(strings.NewReader(source), filename)
}
