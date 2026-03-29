package vm_test

import (
	"encoding/json"
	"strings"
	"testing"

	"BBS_VM/pkg/parser"
	"BBS_VM/pkg/store"
	"BBS_VM/pkg/vm"
)

type vmResult struct {
	stdout string
	state  map[string]interface{}
}

func runCBAT(t *testing.T, source string, inputs ...string) vmResult {
	t.Helper()
	// inject dump_state_on_exit
	if !strings.Contains(source, "dump_state_on_exit") {
		source = strings.Replace(source, ".header", ".header\n    dump_state_on_exit", 1)
	}

	res, err := parser.ParseString(source)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	s := store.NewMemoryStore()
	io := vm.NewStringIO(inputs)
	v := vm.New(s, io)
	v.LoadResult(res.GlobalEntry, res.Programs, res.Files)
	if err := v.Run(); err != nil {
		t.Fatalf("run error: %v", err)
	}

	// parse state from stderr
	errOut := io.ErrOutput()
	var state map[string]interface{}
	if idx := strings.Index(errOut, "---CBAT_STATE_BEGIN---"); idx >= 0 {
		jsonStart := idx + len("---CBAT_STATE_BEGIN---\n")
		jsonEnd := strings.Index(errOut, "---CBAT_STATE_END---")
		if jsonEnd > jsonStart {
			json.Unmarshal([]byte(errOut[jsonStart:jsonEnd]), &state)
		}
	}

	return vmResult{stdout: io.Output(), state: state}
}

func getVar(r vmResult, name string) string {
	if vars, ok := r.state["variables"].(map[string]interface{}); ok {
		if v, ok := vars[name]; ok {
			return v.(string)
		}
	}
	return ""
}

func getFile(r vmResult, name string) string {
	if files, ok := r.state["files"].(map[string]interface{}); ok {
		if v, ok := files[name]; ok {
			return v.(string)
		}
	}
	return ""
}

// --- Instructions ---

func TestEcho(t *testing.T) {
	r := runCBAT(t, `.header
    filename "t.bat"
.instrs
    e "hello world"
    trm`)
	if !strings.Contains(r.stdout, "hello world") {
		t.Errorf("expected 'hello world', got %q", r.stdout)
	}
}

func TestEchoInterpolation(t *testing.T) {
	r := runCBAT(t, `.header
    filename "t.bat"
.instrs
    st NAME,"alice"
    e "hi %NAME%"
    trm`)
	if !strings.Contains(r.stdout, "hi alice") {
		t.Errorf("expected 'hi alice', got %q", r.stdout)
	}
}

func TestStore(t *testing.T) {
	r := runCBAT(t, `.header
    filename "t.bat"
.instrs
    st X,42
    trm`)
	if v := getVar(r, "x"); v != "42" {
		t.Errorf("expected x=42, got %q", v)
	}
}

func TestStf(t *testing.T) {
	r := runCBAT(t, `.header
    filename "t.bat"
.files
    "DATA.txt","contents here"
.instrs
    stf VAR,"DATA.txt"
    trm`)
	if v := getVar(r, "var"); v != "contents here" {
		t.Errorf("expected 'contents here', got %q", v)
	}
}

func TestStp(t *testing.T) {
	r := runCBAT(t, `.header
    filename "t.bat"
.instrs
    stp NAME,"Enter: "
    trm`, "bob")
	if v := getVar(r, "name"); v != "bob" {
		t.Errorf("expected 'bob', got %q", v)
	}
}

func TestAppendFile(t *testing.T) {
	r := runCBAT(t, `.header
    filename "t.bat"
.instrs
    af "line1","OUT.txt"
    af "line2","OUT.txt"
    trm`)
	if f := getFile(r, "OUT.txt"); f != "line1\nline2\n" {
		t.Errorf("expected 'line1\\nline2\\n', got %q", f)
	}
}

func TestWriteFile(t *testing.T) {
	r := runCBAT(t, `.header
    filename "t.bat"
.instrs
    wf "first","OUT.txt"
    wf "second","OUT.txt"
    trm`)
	if f := getFile(r, "OUT.txt"); f != "second" {
		t.Errorf("expected 'second', got %q", f)
	}
}

func TestDeleteFile(t *testing.T) {
	r := runCBAT(t, `.header
    filename "t.bat"
.files
    "TEMP.txt","data"
.instrs
    df "TEMP.txt"
    trm`)
	if f := getFile(r, "TEMP.txt"); f != "" {
		t.Errorf("file should be deleted, got %q", f)
	}
}

func TestMathImmediate(t *testing.T) {
	r := runCBAT(t, `.header
    filename "t.bat"
.instrs
    st X,10
    adi X,5
    sbi X,3
    mli X,7
    dvi X,2
    mdi X,13
    trm`)
	// (10+5-3)*7 = 84 / 2 = 42 % 13 = 3
	if v := getVar(r, "x"); v != "3" {
		t.Errorf("expected 3, got %q", v)
	}
}

func TestMathRegister(t *testing.T) {
	r := runCBAT(t, `.header
    filename "t.bat"
.instrs
    st A,10
    st B,20
    add C,A,B
    trm`)
	if v := getVar(r, "c"); v != "30" {
		t.Errorf("expected 30, got %q", v)
	}
}

func TestSta(t *testing.T) {
	r := runCBAT(t, `.header
    filename "t.bat"
.instrs
    sta R,"2+3*4"
    trm`)
	if v := getVar(r, "r"); v != "14" {
		t.Errorf("expected 14, got %q", v)
	}
}

func TestCat(t *testing.T) {
	r := runCBAT(t, `.header
    filename "t.bat"
.instrs
    st A,"foo"
    st B,"bar"
    cat C,A,B
    trm`)
	if v := getVar(r, "c"); v != "foobar" {
		t.Errorf("expected 'foobar', got %q", v)
	}
}

func TestLen(t *testing.T) {
	r := runCBAT(t, `.header
    filename "t.bat"
.instrs
    st S,"hello"
    len L,S
    trm`)
	if v := getVar(r, "l"); v != "5" {
		t.Errorf("expected 5, got %q", v)
	}
}

func TestAtoI(t *testing.T) {
	r := runCBAT(t, `.header
    filename "t.bat"
.instrs
    atoi CODE,"A"
    trm`)
	if v := getVar(r, "code"); v != "65" {
		t.Errorf("expected 65, got %q", v)
	}
}

func TestItoA(t *testing.T) {
	r := runCBAT(t, `.header
    filename "t.bat"
.instrs
    st N,65
    itoa CH,"%N%"
    trm`)
	if v := getVar(r, "ch"); v != "A" {
		t.Errorf("expected 'A', got %q", v)
	}
}

// --- Control Flow ---

func TestGotoLabel(t *testing.T) {
	r := runCBAT(t, `.header
    filename "t.bat"
.instrs
    g skip
    st X,"wrong"
    trm
    l skip
    st X,"right"
    trm`)
	if v := getVar(r, "x"); v != "right" {
		t.Errorf("expected 'right', got %q", v)
	}
}

func TestJsubRet(t *testing.T) {
	r := runCBAT(t, `.header
    filename "t.bat"
.instrs
    st R,"before"
    jsub mysub
    st R,"after"
    trm
    l mysub
    st INSIDE,"yes"
    ret`)
	if v := getVar(r, "r"); v != "after" {
		t.Errorf("expected 'after', got %q", v)
	}
	if v := getVar(r, "inside"); v != "yes" {
		t.Errorf("expected 'yes', got %q", v)
	}
}

func TestJsubNested(t *testing.T) {
	r := runCBAT(t, `.header
    filename "t.bat"
.instrs
    jsub outer
    st R,"done"
    trm
    l outer
    st A,"outer"
    jsub inner
    st B,"back"
    ret
    l inner
    st C,"inner"
    ret`)
	if v := getVar(r, "r"); v != "done" {
		t.Errorf("expected 'done', got %q", v)
	}
	if v := getVar(r, "c"); v != "inner" {
		t.Errorf("expected 'inner', got %q", v)
	}
}

func TestCallSubroutine(t *testing.T) {
	r := runCBAT(t, `.global
    entry "MAIN.BAT"
.header
    filename "MAIN.BAT"
.instrs
    st X,"before"
    c "SUB.BAT"
    st X,"after"
    trm
.header
    filename "SUB.BAT"
.instrs
    st Y,"from sub"
    trm`)
	if v := getVar(r, "x"); v != "after" {
		t.Errorf("expected 'after', got %q", v)
	}
	if v := getVar(r, "y"); v != "from sub" {
		t.Errorf("expected 'from sub', got %q", v)
	}
}

// --- Conditionals ---

func TestIeqTaken(t *testing.T) {
	r := runCBAT(t, `.header
    filename "t.bat"
.instrs
    st V,"yes"
    ieq V,"yes",match
    st R,"no"
    trm
    l match
    st R,"yes"
    trm`)
	if v := getVar(r, "r"); v != "yes" {
		t.Errorf("expected 'yes', got %q", v)
	}
}

func TestIeqNotTaken(t *testing.T) {
	r := runCBAT(t, `.header
    filename "t.bat"
.instrs
    st V,"no"
    ieq V,"yes",match
    st R,"not taken"
    trm
    l match
    st R,"taken"
    trm`)
	if v := getVar(r, "r"); v != "not taken" {
		t.Errorf("expected 'not taken', got %q", v)
	}
}

func TestIeqWithoutLabelSkip(t *testing.T) {
	r := runCBAT(t, `.header
    filename "t.bat"
.instrs
    st V,"no"
    ieq V,"yes"
    st R,"should skip"
    st R,"fell through"
    trm`)
	if v := getVar(r, "r"); v != "fell through" {
		t.Errorf("expected 'fell through', got %q", v)
	}
}

func TestIexTaken(t *testing.T) {
	r := runCBAT(t, `.header
    filename "t.bat"
.files
    "F.txt","data"
.instrs
    iex "F.txt",found
    st R,"no"
    trm
    l found
    st R,"yes"
    trm`)
	if v := getVar(r, "r"); v != "yes" {
		t.Errorf("expected 'yes', got %q", v)
	}
}

func TestCountingLoop(t *testing.T) {
	r := runCBAT(t, `.header
    filename "t.bat"
.instrs
    st I,0
    l loop
    adi I,1
    ilti I,5,loop
    trm`)
	if v := getVar(r, "i"); v != "5" {
		t.Errorf("expected 5, got %q", v)
	}
}

func TestTerminate(t *testing.T) {
	r := runCBAT(t, `.header
    filename "t.bat"
.instrs
    st X,"alive"
    trm
    st X,"dead"`)
	if v := getVar(r, "x"); v != "alive" {
		t.Errorf("expected 'alive', got %q", v)
	}
}

func TestEOF(t *testing.T) {
	r := runCBAT(t, `.header
    filename "t.bat"
.instrs
    st X,"alive"`)
	if v := getVar(r, "x"); v != "alive" {
		t.Errorf("expected 'alive', got %q", v)
	}
	if r.state["exit_reason"] != "eof" {
		t.Errorf("expected eof exit, got %v", r.state["exit_reason"])
	}
}
