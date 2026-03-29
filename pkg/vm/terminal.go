package vm

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"golang.org/x/term"
)

const (
	ctrlT     = 0x14
	ctrlC     = 0x03
	ctrlD     = 0x04
	backspace = 0x7f
	enterCR   = 0x0d
	enterLF   = 0x0a
)

// TerminalIO provides a raw-mode terminal with Ctrl+T debug menu
// and Ctrl+C interrupt support.
type TerminalIO struct {
	fd       int
	oldState *term.State
	vm       *VM

	// Input channel — a background goroutine reads raw bytes from stdin
	// and sends them here. This means Ctrl+C and Ctrl+T are caught even
	// when the VM is executing (not blocked on ReadLine).
	inputCh chan byte
	stopCh  chan struct{}

	// When the debug menu or Ctrl+C is detected during execution,
	// these flags are set and checked by the VM loop.
	mu            sync.Mutex
	wantInterrupt bool // Ctrl+C
	wantDebug     bool // Ctrl+T

	skipNextLF bool

	outWriter io.Writer
	errWriter io.Writer
}

func NewTerminalIO(v *VM) (*TerminalIO, error) {
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		return nil, fmt.Errorf("not a terminal")
	}

	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return nil, fmt.Errorf("failed to set raw mode: %w", err)
	}

	t := &TerminalIO{
		fd:        fd,
		oldState:  oldState,
		vm:        v,
		inputCh:   make(chan byte, 256),
		stopCh:    make(chan struct{}),
		outWriter: &termWriter{w: os.Stdout},
		errWriter: &termWriter{w: os.Stderr},
	}

	// Background reader — pumps raw bytes from stdin into inputCh.
	// Ctrl+C and Ctrl+T are intercepted here so they work even when
	// the VM is executing instructions (not waiting for input).
	go func() {
		b := make([]byte, 1)
		for {
			n, err := os.Stdin.Read(b)
			if n > 0 {
				ch := b[0]
				switch ch {
				case ctrlC, ctrlD:
					t.mu.Lock()
					t.wantInterrupt = true
					t.mu.Unlock()
					// Also push to inputCh so ReadLine unblocks
					select {
					case t.inputCh <- ch:
					default:
					}
				case ctrlT:
					t.mu.Lock()
					t.wantDebug = true
					t.mu.Unlock()
					select {
					case t.inputCh <- ch:
					default:
					}
				default:
					select {
					case t.inputCh <- ch:
					default:
					}
				}
			}
			if err != nil {
				close(t.stopCh)
				return
			}
		}
	}()

	return t, nil
}

func (t *TerminalIO) Close() {
	if t.oldState != nil {
		term.Restore(t.fd, t.oldState)
		t.oldState = nil
	}
}

func (t *TerminalIO) Stdout() io.Writer { return t.outWriter }
func (t *TerminalIO) Stderr() io.Writer { return t.errWriter }

func (t *TerminalIO) ReadLine() (string, error) {
	var buf []byte

	for {
		var ch byte
		select {
		case ch = <-t.inputCh:
		case <-t.stopCh:
			return string(buf), io.EOF
		}

		// CRLF normalization
		if t.skipNextLF {
			t.skipNextLF = false
			if ch == enterLF {
				continue
			}
		}

		switch {
		case ch == ctrlT:
			t.mu.Lock()
			t.wantDebug = false
			t.mu.Unlock()
			t.showDebugMenu()
			fmt.Fprintf(t.outWriter, "\r\n")
			return string(buf), nil

		case ch == ctrlC || ch == ctrlD:
			t.mu.Lock()
			t.wantInterrupt = false
			t.mu.Unlock()
			fmt.Fprintf(t.outWriter, "\r\n")
			return "", io.EOF

		case ch == enterCR || ch == enterLF:
			fmt.Fprintf(t.outWriter, "\r\n")
			if ch == enterCR {
				t.skipNextLF = true
			}
			return string(buf), nil

		case ch == backspace || ch == 0x08:
			if len(buf) > 0 {
				buf = buf[:len(buf)-1]
				fmt.Fprintf(t.outWriter, "\b \b")
			}

		case ch >= 0x20 && ch < 0x7f:
			buf = append(buf, ch)
			fmt.Fprintf(t.outWriter, "%c", ch)
		}
	}
}

// CheckInterrupt returns true if Ctrl+C was pressed during execution.
func (t *TerminalIO) CheckInterrupt() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.wantInterrupt {
		t.wantInterrupt = false
		return true
	}
	return false
}

// CheckDebug returns true if Ctrl+T was pressed during execution.
func (t *TerminalIO) CheckDebug() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.wantDebug {
		t.wantDebug = false
		return true
	}
	return false
}

func (t *TerminalIO) showDebugMenu() {
	fmt.Fprintf(t.outWriter, "\r\n\r\n--- CBAT Debug (Ctrl+T) ---\r\n")
	fmt.Fprintf(t.outWriter, "  d  Enter interactive debugger\r\n")
	fmt.Fprintf(t.outWriter, "  s  Dump state and exit\r\n")
	fmt.Fprintf(t.outWriter, "  t  Toggle step tracing\r\n")
	fmt.Fprintf(t.outWriter, "  r  Resume execution\r\n")
	fmt.Fprintf(t.outWriter, "> ")

	for {
		var ch byte
		select {
		case ch = <-t.inputCh:
		case <-t.stopCh:
			return
		}

		// Ignore control characters, whitespace, and extra Ctrl+T presses
		if ch < 0x20 || ch == 0x7f {
			continue
		}

		fmt.Fprintf(t.outWriter, "%c\r\n", ch)

		switch ch {
		case 'd':
			fmt.Fprintf(t.outWriter, "Entering debugger...\r\n")
			t.vm.enterDebugger()
			return
		case 's':
			fmt.Fprintf(t.outWriter, "Dumping state and exiting...\r\n")
			t.vm.DumpState()
			t.Close()
			os.Exit(0)
		case 't':
			t.vm.DebugStep = !t.vm.DebugStep
			if t.vm.DebugStep {
				fmt.Fprintf(t.outWriter, "Step tracing: ON\r\n")
			} else {
				fmt.Fprintf(t.outWriter, "Step tracing: OFF\r\n")
			}
			return
		case 'r':
			fmt.Fprintf(t.outWriter, "Resuming...\r\n")
			return
		default:
			fmt.Fprintf(t.outWriter, "> ")
		}
	}
}

// termWriter translates \n to \r\n for raw terminal mode.
type termWriter struct {
	w io.Writer
}

func (t *termWriter) Write(p []byte) (int, error) {
	s := strings.ReplaceAll(string(p), "\n", "\r\n")
	_, err := t.w.Write([]byte(s))
	return len(p), err
}
