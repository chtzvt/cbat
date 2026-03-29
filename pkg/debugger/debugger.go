package debugger

import (
	"encoding/json"
	"fmt"
	"strconv"

	"BBS_VM/pkg/store"
	"BBS_VM/pkg/vm"
)

// Attach registers the debugger as the VM's breakpoint handler.
func Attach(v *vm.VM) {
	v.OnBreakpoint = func(v *vm.VM) {
		REPL(v)
	}
}

// REPL runs the interactive debugger loop.
func REPL(v *vm.VM) {
	fmt.Fprintln(v.IO.Stdout(), ">>> Debugger enters. Hello!")
	for {
		line, err := v.IO.ReadLine()
		if err != nil {
			break
		}
		if line == "q" {
			break
		}

		switch line {
		case "pc":
			fmt.Fprintf(v.IO.Stdout(), "Current instruction: %d\n", v.CurrentPC())

		case "vdmp":
			fmt.Fprintln(v.IO.Stdout(), "Variable dump:")
			for k, val := range v.Store.Snapshot(store.Variables) {
				fmt.Fprintf(v.IO.Stdout(), " %s | %s\n", k, val)
			}

		case "fdmp":
			fmt.Fprintln(v.IO.Stdout(), "File dump:")
			for k, val := range v.Store.Snapshot(store.Files) {
				fmt.Fprintf(v.IO.Stdout(), " %s | %s\n", k, val)
			}

		case "lbdmp":
			fmt.Fprintln(v.IO.Stdout(), "Label dump:")
			for k, val := range v.CurrentLabels() {
				fmt.Fprintf(v.IO.Stdout(), " %s | %d\n", k, val)
			}

		case "sdmp":
			data, _ := json.MarshalIndent(v.StateHash(), "", "  ")
			fmt.Fprintln(v.IO.Stdout(), string(data))

		case "step":
			v.DebugStep = !v.DebugStep
			if v.DebugStep {
				fmt.Fprintln(v.IO.Stdout(), "step log enabled")
			} else {
				fmt.Fprintln(v.IO.Stdout(), "step log disabled")
			}

		case "trm":
			v.ForceTerminate()
			fmt.Fprintln(v.IO.Stdout(), ">>> Debugger exits. Bye!")
			return

		default:
			// Try vset, vdel, etc. with arguments
			if len(line) > 4 && line[:4] == "vset" {
				fmt.Fprint(v.IO.Stdout(), "Variable name: ")
				name, _ := v.IO.ReadLine()
				fmt.Fprint(v.IO.Stdout(), "Value: ")
				val, _ := v.IO.ReadLine()
				v.Store.Set(store.Variables, name, val)
			} else if len(line) > 3 && line[:3] == "lbc" {
				fmt.Fprint(v.IO.Stdout(), "Label name: ")
				name, _ := v.IO.ReadLine()
				fmt.Fprint(v.IO.Stdout(), "Destination: ")
				dest, _ := v.IO.ReadLine()
				idx, _ := strconv.Atoi(dest)
				v.SetLabel(name, idx)
			} else if line == "j" {
				fmt.Fprint(v.IO.Stdout(), "Destination address: ")
				dest, _ := v.IO.ReadLine()
				idx, _ := strconv.Atoi(dest)
				v.SetPC(idx)
			} else {
				fmt.Fprintln(v.IO.Stdout(), `
--- CBAT DEBUGGER ---

pc    | current PC value
j     | jump to address
vset  | set variable
vdmp  | dump variables
fdmp  | dump files
lbc   | create label
lbdmp | dump labels
sdmp  | dump full VM state as JSON
step  | toggle step log
trm   | terminate execution
q     | exit debugger`)
			}
		}
	}
	fmt.Fprintln(v.IO.Stdout(), ">>> Debugger exits. Bye!")
}
