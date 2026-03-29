package compiler

import (
	"strings"
	"testing"
)

func TestParseEcho(t *testing.T) {
	tests := []struct {
		input string
		want  Node
	}{
		{"echo hello world", Echo{Text: "hello world"}},
		{"echo.", EchoBlank{}},
		{"echo .", EchoBlank{}},
		{"ECHO Hello", Echo{Text: "Hello"}},
		{`echo text >>"file.txt"`, EchoToFile{Text: "text", Path: "file.txt", Append: true}},
		{`echo text >"file.txt"`, EchoToFile{Text: "text", Path: "file.txt", Append: false}},
		{"echo off", EchoOff{}},
	}

	for _, tt := range tests {
		got := ParseLine(tt.input)
		if got == nil {
			t.Errorf("ParseLine(%q) = nil", tt.input)
			continue
		}
		switch g := got.(type) {
		case Echo:
			if w, ok := tt.want.(Echo); ok && g.Text != w.Text {
				t.Errorf("ParseLine(%q) = Echo{%q}, want Echo{%q}", tt.input, g.Text, w.Text)
			}
		case EchoToFile:
			if w, ok := tt.want.(EchoToFile); ok && (g.Path != w.Path || g.Append != w.Append) {
				t.Errorf("ParseLine(%q) = EchoToFile{path:%q,append:%v}, want {path:%q,append:%v}", tt.input, g.Path, g.Append, w.Path, w.Append)
			}
		}
	}
}

func TestParseSet(t *testing.T) {
	n := ParseLine("set NAME=hello")
	if s, ok := n.(SetVar); !ok || s.Name != "NAME" || s.Value != "hello" {
		t.Errorf("got %#v", n)
	}

	n = ParseLine("set /p user=Username:")
	if s, ok := n.(SetPrompt); !ok || s.Name != "user" || s.Prompt != "Username:" {
		t.Errorf("got %#v", n)
	}

	n = ParseLine(`set /p PASSWD=<"C:\DSOFT\DMS\USERS\test.txt"`)
	if s, ok := n.(SetFromFile); !ok || s.Name != "PASSWD" || !strings.Contains(s.Path, "test.txt") {
		t.Errorf("got %#v", n)
	}

	n = ParseLine("set /a tries=%TRY%+1")
	if s, ok := n.(SetArith); !ok || s.Name != "tries" {
		t.Errorf("got %#v", n)
	}
}

func TestParseIf(t *testing.T) {
	n := ParseLine(`if "%user%" EQU "exit" goto DONE`)
	if c, ok := n.(IfCondition); !ok || c.Left != "user" || c.Right != "exit" {
		t.Errorf("got %#v", n)
	} else if g, ok := c.Then.(Goto); !ok || g.Target != "DONE" {
		t.Errorf("then = %#v", c.Then)
	}

	n = ParseLine(`IF EXIST "C:\DSOFT\DMS\test.txt" goto found`)
	if c, ok := n.(IfCondition); !ok || c.Operator != "EXIST" {
		t.Errorf("got %#v", n)
	} else if _, ok := c.Then.(Goto); !ok {
		t.Errorf("then = %#v", c.Then)
	}

	n = ParseLine(`IF NOT EXIST "C:\DSOFT\DMS\test.txt" goto notfound`)
	if c, ok := n.(IfCondition); !ok || c.Operator != "NOT EXIST" {
		t.Errorf("got %#v", n)
	}

	// Unquoted == comparison
	n = ParseLine("if %input%==1 goto MENU")
	if c, ok := n.(IfCondition); !ok || c.Left != "input" || c.Right != "1" {
		t.Errorf("got %#v", n)
	}
}

func TestParseLabel(t *testing.T) {
	n := ParseLine(":DMSLOGIN")
	if l, ok := n.(Label); !ok || l.Name != "DMSLOGIN" {
		t.Errorf("got %#v", n)
	}
}

func TestParseGoto(t *testing.T) {
	n := ParseLine("goto DMSLOGIN")
	if g, ok := n.(Goto); !ok || g.Target != "DMSLOGIN" {
		t.Errorf("got %#v", n)
	}

	n = ParseLine("GOTO main")
	if g, ok := n.(Goto); !ok || g.Target != "main" {
		t.Errorf("got %#v", n)
	}
}

func TestParsePause(t *testing.T) {
	n := ParseLine("pause")
	if p, ok := n.(Pause); !ok || p.Quiet {
		t.Errorf("got %#v", n)
	}

	n = ParseLine("pause>nul")
	if p, ok := n.(Pause); !ok || !p.Quiet {
		t.Errorf("got %#v", n)
	}
}

func TestParsePing(t *testing.T) {
	n := ParseLine("@ping 127.0.0.1 -n 2 -w 1000 > nul")
	if s, ok := n.(Sleep); !ok || s.Ms != 2000 {
		t.Errorf("got %#v, want Sleep{Ms:2000}", n)
	}
}

func TestCompileAndLower(t *testing.T) {
	source := `@echo off
color 17
cls
:LOGIN
echo Welcome to DMS
set /p user=Username:
if "%user%" EQU "exit" exit
if "%user%" EQU "admin" goto ADMIN
echo Hello, %user%!
pause
goto LOGIN
:ADMIN
echo Admin panel
exit
`
	prog, err := CompileString(source, "TEST.BAT")
	if err != nil {
		t.Fatal(err)
	}

	// Check that key lines are present
	joined := strings.Join(prog.Lines, "\n")
	for _, want := range []string{
		"clr 17",
		"cls",
		"l LOGIN",
		`e "Welcome to DMS"`,
		`stp user,"Username:"`,
		`ieq user,"exit"`,
		"trm",
		`ieq user,"admin",ADMIN`,
		`e "Hello, %user%!"`,
		"p",
		"g LOGIN",
		"l ADMIN",
		`e "Admin panel"`,
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("output missing %q\n\nFull output:\n%s", want, joined)
		}
	}
}
