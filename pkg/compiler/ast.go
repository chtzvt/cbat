package compiler

// Node represents a single batch statement.
type Node interface {
	nodeTag()
}

type EchoOff struct{}
type Echo struct{ Text string }
type EchoBlank struct{}
type EchoToFile struct {
	Text   string
	Path   string
	Append bool // >> vs >
}
type SetVar struct {
	Name  string
	Value string
}
type SetPrompt struct {
	Name   string
	Prompt string
}
type SetArith struct {
	Name string
	Expr string
}
type SetFromFile struct {
	Name string
	Path string
}
type Label struct{ Name string }
type Goto struct{ Target string }
type Call struct{ Path string }
type IfCondition struct {
	Negated  bool
	Operator string // "EQU", "NEQ", "GTR", "LSS", "GEQ", "LEQ", "==", "EXIST"
	Left     string // variable name (for comparisons) or path (for EXIST)
	Right    string // comparison value (empty for EXIST)
	Then     Node   // the action (Goto, Call, Exit, etc.)
}
type Pause struct{ Quiet bool }
type Cls struct{}
type Color struct{ Code string }
type Title struct{ Text string }
type TypeFile struct{ Path string }
type TypeFilePaged struct{ Path string }
type Del struct{ Path string }
type Mkdir struct{ Path string }
type Exit struct{}
type Sleep struct{ Ms int }
type Comment struct{ Text string }
type Noop struct{}

func (EchoOff) nodeTag()      {}
func (Echo) nodeTag()         {}
func (EchoBlank) nodeTag()    {}
func (EchoToFile) nodeTag()   {}
func (SetVar) nodeTag()       {}
func (SetPrompt) nodeTag()    {}
func (SetArith) nodeTag()     {}
func (SetFromFile) nodeTag()  {}
func (Label) nodeTag()        {}
func (Goto) nodeTag()         {}
func (Call) nodeTag()         {}
func (IfCondition) nodeTag()  {}
func (Pause) nodeTag()        {}
func (Cls) nodeTag()          {}
func (Color) nodeTag()        {}
func (Title) nodeTag()        {}
func (TypeFile) nodeTag()     {}
func (TypeFilePaged) nodeTag(){}
func (Del) nodeTag()          {}
func (Mkdir) nodeTag()        {}
func (Exit) nodeTag()         {}
func (Sleep) nodeTag()        {}
func (Comment) nodeTag()      {}
func (Noop) nodeTag()         {}
