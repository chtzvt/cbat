package compiler

import (
	"fmt"
)

// LowerProgram converts a list of AST nodes into CBAT instruction lines.
func LowerProgram(nodes []Node) []string {
	var lines []string
	for _, node := range nodes {
		lines = append(lines, lowerNode(node)...)
	}
	return lines
}

func lowerNode(node Node) []string {
	switch n := node.(type) {
	case EchoOff:
		return nil // no-op in CBAT

	case Echo:
		return []string{fmt.Sprintf(`e "%s"`, escapeForCbat(n.Text))}

	case EchoBlank:
		return []string{`e ""`}

	case EchoToFile:
		text := escapeForCbat(n.Text)
		path := escapeForCbat(n.Path)
		if n.Append {
			return []string{fmt.Sprintf(`af "%s","%s"`, text, path)}
		}
		return []string{fmt.Sprintf(`wf "%s","%s"`, text, path)}

	case SetVar:
		return []string{fmt.Sprintf(`st %s,"%s"`, n.Name, escapeForCbat(n.Value))}

	case SetPrompt:
		return []string{fmt.Sprintf(`stp %s,"%s"`, n.Name, escapeForCbat(n.Prompt))}

	case SetArith:
		return []string{fmt.Sprintf(`sta %s,"%s"`, n.Name, n.Expr)}

	case SetFromFile:
		return []string{fmt.Sprintf(`stf %s,"%s"`, n.Name, escapeForCbat(n.Path))}

	case Label:
		return []string{fmt.Sprintf("l %s", n.Name)}

	case Goto:
		return []string{fmt.Sprintf("g %s", n.Target)}

	case Call:
		return []string{fmt.Sprintf(`c "%s"`, escapeForCbat(n.Path))}

	case Exit:
		return []string{"trm"}

	case Pause:
		if n.Quiet {
			return []string{"p 0"}
		}
		return []string{"p"}

	case Cls:
		return []string{"cls"}

	case Color:
		return []string{fmt.Sprintf("clr %s", n.Code)}

	case Title:
		return []string{fmt.Sprintf(`ttl "%s"`, escapeForCbat(n.Text))}

	case TypeFile:
		return []string{fmt.Sprintf(`t "%s"`, escapeForCbat(n.Path))}

	case TypeFilePaged:
		return []string{fmt.Sprintf(`tp "%s",20`, escapeForCbat(n.Path))}

	case Del:
		return []string{fmt.Sprintf(`df "%s"`, escapeForCbat(n.Path))}

	case Mkdir:
		return []string{fmt.Sprintf(`mkd "%s"`, escapeForCbat(n.Path))}

	case Sleep:
		return []string{fmt.Sprintf("slp %d", n.Ms)}

	case IfCondition:
		return lowerIf(n)

	case Comment:
		return []string{"; " + n.Text}

	case Noop:
		return nil
	}

	return nil
}

func lowerIf(n IfCondition) []string {
	// Determine the CBAT conditional opcode
	var opcode string
	switch n.Operator {
	case "EXIST":
		opcode = "iex"
	case "NOT EXIST":
		opcode = "inx"
	case "EQU", "==":
		if n.Negated {
			opcode = "inq"
		} else {
			opcode = "ieq"
		}
	case "NEQ":
		if n.Negated {
			opcode = "ieq"
		} else {
			opcode = "inq"
		}
	case "GTR":
		opcode = "igti"
	case "LSS":
		opcode = "ilti"
	case "GEQ":
		opcode = "igeqi"
	case "LEQ":
		opcode = "ileqi"
	default:
		opcode = "ieq"
	}

	// Get the target label from the action
	var targetLabel string

	// For inline actions (not goto), emit as label-less conditional + action instruction.
	// The conditional executes the next instruction on true, skips it on false.
	emitInline := func(actionLines []string) []string {
		var condLine string
		if opcode == "iex" || opcode == "inx" {
			condLine = fmt.Sprintf(`%s "%s"`, opcode, escapeForCbat(n.Left))
		} else {
			condLine = fmt.Sprintf(`%s %s,"%s"`, opcode, n.Left, escapeForCbat(n.Right))
		}
		return append([]string{condLine}, actionLines...)
	}

	switch t := n.Then.(type) {
	case Goto:
		targetLabel = t.Target
	case Exit:
		return emitInline([]string{"trm"})
	case Call:
		return emitInline([]string{fmt.Sprintf(`c "%s"`, escapeForCbat(t.Path))})
	case SetVar:
		return emitInline(lowerNode(t))
	case SetPrompt:
		return emitInline(lowerNode(t))
	case SetFromFile:
		return emitInline(lowerNode(t))
	case SetArith:
		return emitInline(lowerNode(t))
	}

	// Standard: conditional with label
	if opcode == "iex" || opcode == "inx" {
		return []string{fmt.Sprintf(`%s "%s",%s`, opcode, escapeForCbat(n.Left), targetLabel)}
	}

	return []string{fmt.Sprintf(`%s %s,"%s",%s`, opcode, n.Left, escapeForCbat(n.Right), targetLabel)}
}

func escapeForCbat(s string) string {
	return s
}
