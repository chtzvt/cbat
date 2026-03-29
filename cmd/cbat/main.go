package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"runtime/pprof"
	"strings"
	"sync/atomic"
	"time"

	"golang.org/x/crypto/ssh"

	"BBS_VM/pkg/bytecode"
	"BBS_VM/pkg/compiler"
	"BBS_VM/pkg/debugger"
	"BBS_VM/pkg/parser"
	"BBS_VM/pkg/store"
	"BBS_VM/pkg/vm"
)

func usage() {
	fmt.Fprintf(os.Stderr, `Usage:
  cbat run   <file.bat> [file2.bat ...]         Compile and run batch files
  cbat exec  <file.cbat>                        Run a compiled .cbat file
  cbat build <file.bat> [file2.bat ...] [-o out] [-seed k=v ...]
                                                 Compile batch to .cbat
  cbat serve <file.cbat> [-listen :2222] [-db bbs.db] [-hostkey key.pem] [-debug]
                                                 Run as SSH server
  cbat bench <file.cbat> [-cpuprof cpu.prof] [-memprof mem.prof]
                                                 Benchmark and profile
`)
	os.Exit(1)
}

func main() {
	if len(os.Args) < 2 {
		usage()
	}

	switch os.Args[1] {
	case "run":
		cmdRun(os.Args[2:])
	case "exec":
		cmdExec(os.Args[2:])
	case "build":
		cmdBuild(os.Args[2:])
	case "serve":
		cmdServe(os.Args[2:])
	case "bench":
		cmdBench(os.Args[2:])
	default:
		// If the argument looks like a file, infer the command
		if strings.HasSuffix(os.Args[1], ".cbatc") {
			cmdExec(os.Args[1:])
		} else if strings.HasSuffix(os.Args[1], ".cbat") {
			cmdExec(os.Args[1:])
		} else if strings.HasSuffix(os.Args[1], ".bat") {
			cmdRun(os.Args[1:])
		} else {
			usage()
		}
	}
}

// --- run: compile .bat and execute immediately ---

func cmdRun(args []string) {
	if len(args) == 0 {
		usage()
	}

	var paths []string
	seeds := map[string]string{}
	for i := 0; i < len(args); i++ {
		if args[i] == "-seed" && i+1 < len(args) {
			if eq := strings.Index(args[i+1], "="); eq > 0 {
				seeds[args[i+1][:eq]] = args[i+1][eq+1:]
			}
			i++
		} else {
			paths = append(paths, args[i])
		}
	}

	// Compile
	var programs []*compiler.CompiledProgram
	for _, path := range paths {
		prog, err := compiler.CompileFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "cbat: compiling %s: %v\n", path, err)
			os.Exit(1)
		}
		programs = append(programs, prog)
	}

	entry := ""
	if len(programs) > 1 {
		entry = programs[0].FileName
	}

	// Emit to temp buffer and parse back
	var buf strings.Builder
	compiler.EmitCBAT(&buf, programs, entry, seeds)

	result, err := parser.ParseString(buf.String())
	if err != nil {
		fmt.Fprintf(os.Stderr, "cbat: %v\n", err)
		os.Exit(1)
	}

	runWithTerminal(result)
}

// --- exec: run a .cbat or .cbatc file ---

func cmdExec(args []string) {
	if len(args) == 0 {
		usage()
	}

	path := args[0]
	result, err := loadWithCache(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cbat: %v\n", err)
		os.Exit(1)
	}

	runWithTerminal(result)
}

// loadWithCache loads a .cbat file, using a cached .cbatc if fresh.
// If the .cbat is newer than the .cbatc (or no cache exists), it parses
// the text source, writes a .cbatc, and returns the result.
// If the input is already a .cbatc, it decodes directly.
func loadWithCache(path string) (*parser.Result, error) {
	// Direct bytecode file
	if strings.HasSuffix(path, ".cbatc") || bytecode.IsBytecode(path) {
		return bytecode.DecodeFile(path)
	}

	cachePath := bytecode.CachedPath(path)

	// Check if cache is fresh
	srcInfo, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if cacheInfo, err := os.Stat(cachePath); err == nil {
		if cacheInfo.ModTime().After(srcInfo.ModTime()) {
			if result, err := bytecode.DecodeFile(cachePath); err == nil {
				return result, nil
			}
			// stale or corrupt cache — fall through to recompile
		}
	}

	// Parse text source
	result, err := parser.ParseFile(path)
	if err != nil {
		return nil, err
	}

	// Write cache (best-effort, don't fail if we can't write)
	bytecode.EncodeFile(cachePath, result)

	return result, nil
}

func runWithTerminal(result *parser.Result) {
	s := store.NewMemoryStore()
	v := vm.New(s, nil)

	tio, err := vm.NewTerminalIO(v)
	if err != nil {
		v.IO = vm.NewStdIO()
	} else {
		v.IO = tio
		defer tio.Close()
	}

	debugger.Attach(v)
	v.LoadResult(result.GlobalEntry, result.Programs, result.Files)

	if err := v.Run(); err != nil {
		if tio != nil {
			tio.Close()
		}
		fmt.Fprintf(os.Stderr, "cbat: %v\n", err)
		os.Exit(1)
	}
}

// --- build: compile .bat to .cbat ---

func cmdBuild(args []string) {
	if len(args) == 0 {
		usage()
	}

	var paths []string
	output := os.Stdout
	seeds := map[string]string{}

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-o":
			if i+1 < len(args) {
				f, err := os.Create(args[i+1])
				if err != nil {
					fmt.Fprintf(os.Stderr, "cbat: %v\n", err)
					os.Exit(1)
				}
				defer f.Close()
				output = f
				i++
			}
		case "-seed":
			if i+1 < len(args) {
				if eq := strings.Index(args[i+1], "="); eq > 0 {
					seeds[args[i+1][:eq]] = args[i+1][eq+1:]
				}
				i++
			}
		default:
			paths = append(paths, args[i])
		}
	}

	var programs []*compiler.CompiledProgram
	for _, path := range paths {
		prog, err := compiler.CompileFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "cbat: compiling %s: %v\n", path, err)
			os.Exit(1)
		}
		programs = append(programs, prog)
	}

	entry := ""
	if len(programs) > 1 {
		entry = programs[0].FileName
	}

	compiler.EmitCBAT(output, programs, entry, seeds)
}

// --- bench: benchmark and profile ---

func cmdBench(args []string) {
	if len(args) == 0 {
		usage()
	}

	cbatFile := args[0]
	cpuProf := ""
	memProf := ""

	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "-cpuprof":
			if i+1 < len(args) {
				cpuProf = args[i+1]
				i++
			}
		case "-memprof":
			if i+1 < len(args) {
				memProf = args[i+1]
				i++
			}
		}
	}

	result, err := loadWithCache(cbatFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cbat: %v\n", err)
		os.Exit(1)
	}

	if cpuProf != "" {
		f, err := os.Create(cpuProf)
		if err != nil {
			fmt.Fprintf(os.Stderr, "cbat: %v\n", err)
			os.Exit(1)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	s := store.NewMemoryStore()
	sio := vm.NewStringIO(nil)
	v := vm.New(s, sio)
	v.LoadResult(result.GlobalEntry, result.Programs, result.Files)

	start := time.Now()
	v.Run()
	elapsed := time.Since(start)

	fmt.Fprintf(os.Stderr, "Elapsed: %v\n", elapsed)
	fmt.Fprintf(os.Stderr, "Output:  %q\n", sio.Output())

	if memProf != "" {
		f, err := os.Create(memProf)
		if err != nil {
			fmt.Fprintf(os.Stderr, "cbat: %v\n", err)
			os.Exit(1)
		}
		pprof.WriteHeapProfile(f)
		f.Close()
	}
}

// --- serve: SSH BBS server ---

var sessionCount atomic.Int64

func cmdServe(args []string) {
	if len(args) == 0 {
		usage()
	}

	cbatFile := args[0]
	listen := ":2222"
	dbPath := "bbs.db"
	hostKeyPath := ""
	enableDebug := false

	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "-listen":
			if i+1 < len(args) {
				listen = args[i+1]
				i++
			}
		case "-db":
			if i+1 < len(args) {
				dbPath = args[i+1]
				i++
			}
		case "-hostkey":
			if i+1 < len(args) {
				hostKeyPath = args[i+1]
				i++
			}
		case "-debug":
			enableDebug = true
		}
	}

	result, err := loadWithCache(cbatFile)
	if err != nil {
		log.Fatalf("Failed to load %s: %v", cbatFile, err)
	}

	sqlStore, err := store.NewSQLiteStore(dbPath)
	if err != nil {
		log.Fatalf("Failed to open database %s: %v", dbPath, err)
	}
	defer sqlStore.Close()

	for name, content := range result.Files {
		if !sqlStore.Exists(store.Files, name) {
			sqlStore.Set(store.Files, name, content)
		}
	}

	config := &ssh.ServerConfig{NoClientAuth: true}
	config.AddHostKey(loadOrGenerateHostKey(hostKeyPath))

	listener, err := net.Listen("tcp", listen)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", listen, err)
	}
	log.Printf("CBAT BBS listening on %s (program: %s, db: %s)", listen, cbatFile, dbPath)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Accept error: %v", err)
			continue
		}
		go handleConnection(conn, config, result, sqlStore, enableDebug)
	}
}

func handleConnection(conn net.Conn, config *ssh.ServerConfig, program *parser.Result, sqlStore *store.SQLiteStore, enableDebug bool) {
	sessionID := sessionCount.Add(1)
	log.Printf("[session %d] connection from %s", sessionID, conn.RemoteAddr())

	sshConn, chans, reqs, err := ssh.NewServerConn(conn, config)
	if err != nil {
		log.Printf("[session %d] SSH handshake failed: %v", sessionID, err)
		return
	}
	defer sshConn.Close()
	go ssh.DiscardRequests(reqs)

	for newChan := range chans {
		if newChan.ChannelType() != "session" {
			newChan.Reject(ssh.UnknownChannelType, "unsupported")
			continue
		}
		channel, requests, err := newChan.Accept()
		if err != nil {
			return
		}
		go handleSession(channel, requests, sessionID, program, sqlStore, enableDebug)
	}
	log.Printf("[session %d] disconnected", sessionID)
}

func handleSession(channel ssh.Channel, requests <-chan *ssh.Request, sessionID int64, program *parser.Result, sqlStore *store.SQLiteStore, enableDebug bool) {
	defer channel.Close()

	go func() {
		for req := range requests {
			switch req.Type {
			case "pty-req", "shell", "window-change":
				req.Reply(true, nil)
			default:
				req.Reply(false, nil)
			}
		}
	}()

	hybridStore := store.NewHybridStore(sqlStore)
	sshIO := &sshIOProvider{channel: channel, enableDebug: enableDebug}
	v := vm.New(hybridStore, sshIO)
	sshIO.vm = v
	if enableDebug {
		debugger.Attach(v)
	}
	v.LoadResult(program.GlobalEntry, program.Programs, nil)

	log.Printf("[session %d] VM started", sessionID)
	if err := v.Run(); err != nil {
		fmt.Fprintf(channel, "\r\ncbat: %v\r\n", err)
	}
	log.Printf("[session %d] VM exited", sessionID)
}

// --- SSH IO ---

type sshIOProvider struct {
	channel     ssh.Channel
	enableDebug bool
	vm          *vm.VM
}

func (s *sshIOProvider) Stdout() io.Writer { return &crlfWriter{w: s.channel} }
func (s *sshIOProvider) Stderr() io.Writer { return &crlfWriter{w: s.channel} }

func (s *sshIOProvider) ReadLine() (string, error) {
	var buf []byte
	b := make([]byte, 1)
	for {
		n, err := s.channel.Read(b)
		if n > 0 {
			switch ch := b[0]; {
			case ch == '\r' || ch == '\n':
				fmt.Fprint(s.channel, "\r\n")
				return string(buf), nil
			case ch == 0x7f || ch == 0x08:
				if len(buf) > 0 {
					buf = buf[:len(buf)-1]
					fmt.Fprint(s.channel, "\b \b")
				}
			case ch == 0x03 || ch == 0x04:
				fmt.Fprint(s.channel, "\r\n")
				return "", io.EOF
			case ch == 0x14 && s.enableDebug:
				s.showDebugMenu()
				return string(buf), nil
			case ch >= 0x20 && ch < 0x7f:
				buf = append(buf, ch)
				fmt.Fprintf(s.channel, "%c", ch)
			}
		}
		if err != nil {
			return string(buf), err
		}
	}
}

func (s *sshIOProvider) showDebugMenu() {
	w := s.channel
	fmt.Fprintf(w, "\r\n\r\n--- CBAT Debug (Ctrl+T) ---\r\n")
	fmt.Fprintf(w, "  d  Enter interactive debugger\r\n")
	fmt.Fprintf(w, "  s  Dump state and exit\r\n")
	fmt.Fprintf(w, "  t  Toggle step tracing\r\n")
	fmt.Fprintf(w, "  r  Resume execution\r\n")
	fmt.Fprintf(w, "> ")
	b := make([]byte, 1)
	for {
		n, _ := s.channel.Read(b)
		if n == 0 {
			continue
		}
		fmt.Fprintf(w, "%c\r\n", b[0])
		switch b[0] {
		case 'd':
			fmt.Fprintf(w, "Entering debugger...\r\n")
			if s.vm.OnBreakpoint != nil {
				s.vm.OnBreakpoint(s.vm)
			}
			return
		case 's':
			fmt.Fprintf(w, "Dumping state and exiting...\r\n")
			s.vm.DumpState()
			s.vm.ForceTerminate()
			return
		case 't':
			s.vm.DebugStep = !s.vm.DebugStep
			if s.vm.DebugStep {
				fmt.Fprintf(w, "Step tracing: ON\r\n")
			} else {
				fmt.Fprintf(w, "Step tracing: OFF\r\n")
			}
			return
		default:
			fmt.Fprintf(w, "Resuming...\r\n")
			return
		}
	}
}

type crlfWriter struct{ w io.Writer }

func (c *crlfWriter) Write(p []byte) (int, error) {
	s := strings.ReplaceAll(string(p), "\n", "\r\n")
	_, err := c.w.Write([]byte(s))
	return len(p), err
}

// --- Host key ---

func loadOrGenerateHostKey(path string) ssh.Signer {
	if path != "" {
		if data, err := os.ReadFile(path); err == nil {
			if signer, err := ssh.ParsePrivateKey(data); err == nil {
				return signer
			}
		}
	}
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	signer, _ := ssh.NewSignerFromKey(priv)
	if path != "" {
		der, _ := x509.MarshalPKCS8PrivateKey(priv)
		pemBlock := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})
		os.WriteFile(path, pemBlock, 0600)
		log.Printf("Generated host key: %s", path)
	}
	return signer
}
