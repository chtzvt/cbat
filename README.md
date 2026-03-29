# CBAT Batch VM

Welcome to the CBAT (**C**ompiled **Bat**ch) language virtual machine.

CBAT (pronounced "see-bat") is a toy language that's interpreted by a virtual machine. This repository contains the original Ruby implementation, a Go implementation with a batch-to-CBAT compiler, and an SSH server for hosting multi-user BBS sessions.

## Purpose

The intention behind CBAT is to provide a portable, cross-platform language that can be used as a compilation target for old [Batchfile programs](https://github.com/chtzvt/X-DOS-BBS) I wrote when I was a kid.

The Go implementation includes a compiler that automatically translates Windows batch files into CBAT, so the original programs can run in a sandboxed VM with pluggable backing stores (in-memory, SQLite) instead of executing raw shell commands.

CBAT's not intended to be a serious language, but rather a fun project to learn about virtual machines and language design. You can learn more about the CBAT language by reviewing the [instruction set documentation](instruction_set.md) and [examples](programs/) written for fun and testing.

## Quick Start (Go)

Build the `cbat` tool:

```bash
go build -o cbat ./cmd/cbat
```

### Run a batch file directly

Compiles and executes in one step (like `go run`):

```bash
cbat run program.bat
```

Multiple batch files can be compiled together (preserving `call` semantics):

```bash
cbat run main.bat helper.bat another.bat
```

### Run a pre-compiled .cbat file

```bash
cbat exec programs/dms.cbat
```

### Compile batch files to .cbat

```bash
cbat build main.bat helper.bat -o output.cbat
```

Seed files can be provided for initial state (e.g., default passwords, announcements):

```bash
cbat build DMS.bat admin.bat CHAT1.bat -seed 'C:\DSOFT\DMS\ADMINTOOLS\SYSPASSWD.txt=dsoft1' -o dms.cbat
```

### Run as an SSH BBS server

Serves a compiled CBAT program over SSH, with SQLite-backed persistent shared state. Each connection gets its own VM instance with private variables but shared file storage.

```bash
cbat serve programs/dms.cbat -listen :2222 -db bbs.db -hostkey host.pem
```

Connect with any SSH client:

```bash
ssh -p 2222 localhost
```

Use `-debug` to enable the Ctrl+T debug menu for remote sessions:

```bash
cbat serve programs/dms.cbat -listen :2222 -db bbs.db -debug
```

## Pre-compiled Programs

The `programs/` directory contains pre-compiled versions of the original BBS systems:

| Program | Description | Run |
|---------|-------------|-----|
| `dms.cbat` | Dynamic Messaging System — login, mail, chat rooms, admin console | `cbat exec programs/dms.cbat` |
| `xdos.cbat` | X-DOS v5.5 — DOS-style shell with games and utilities | `cbat exec programs/xdos.cbat` |
| `chatserver.cbat` | Anonymous chat server with multiple rooms | `cbat exec programs/chatserver.cbat` |

DMS sysop login: username `system1`, password `dsoft1`. X-DOS: type `newuser` at the login prompt to register.

## Interactive Debug Menu

Press **Ctrl+T** at any prompt to open the debug menu:

```
--- CBAT Debug (Ctrl+T) ---
  d  Enter interactive debugger
  s  Dump state and exit
  t  Toggle step tracing
  r  Resume execution
```

The interactive debugger provides full VM introspection: variable/file/label dumps, instruction stepping, state export as JSON, and more.

## Ruby Implementation

The original Ruby VM is also included. To run a .cbat file with it:

```bash
cd cbat
ruby exec.rb tests/prime.cbat
```

Run the Ruby test suite:

```bash
cd cbat
ruby test/run_all.rb
```

## Features

The CBAT VM and language have been designed specifically to accommodate the needs of the [X-DOS BBS](https://github.com/chtzvt/X-DOS-BBS) and its sibling projects.

At the time, I was writing a lot of Batchfile programs and followed some conventions that I've tried to preserve in CBAT (such as using the filesystem as a global key-value store, and splitting programs into callable batch files with a shared global namespace for variables).

- **Batch-to-CBAT compiler**: Automatically compiles `.bat` files to CBAT IR, handling variable expansion, control flow, file I/O, and the many quirks of CMD.EXE syntax.

- **50+ instruction ISA**: I/O, variables, control flow, conditionals, math (immediate and register-to-register), string operations, file manipulation, and terminal control.

- **Pluggable backing stores**: Variables are per-session (in-memory), files can be backed by SQLite for persistent multi-user state.

- **SSH server**: Host your BBS over SSH with automatic session management and shared persistent state.

- **Integrated debugger**: Invokable via `bp` instruction, `debug` header flag, or Ctrl+T at runtime. Full REPL with variable/file/label inspection, instruction stepping, and JSON state dumps.

- **String interpolation**: Just like in Batch, CBAT supports `%IDENTIFIER%` interpolation in most contexts, including dynamic arguments and path construction.

- **Multi-program compilation**: Multiple `.bat` files compile into a single `.cbat` with preserved `call` semantics, just like a linker.

## Architecture

```
.bat files --> Compiler --> .cbat IR --> VM --> Terminal / SSH
                                         |
                                    Store Interface
                                    /           \
                              MemoryStore    SQLiteStore
```

The Go implementation lives in `pkg/`:

| Package | Description |
|---------|-------------|
| `pkg/compiler` | Batch-to-CBAT compiler (parse, AST, lower, emit) |
| `pkg/parser` | CBAT file format parser |
| `pkg/vm` | Execution engine, instruction dispatch, IO abstraction |
| `pkg/store` | Pluggable store interface, MemoryStore, SQLiteStore, HybridStore |
| `pkg/debugger` | Interactive REPL debugger |

## Further Reading

If you'd like to learn about real-world virtual machines powering serious programming languages, I recommend reading Kevin Newton's [Advent of YARV](https://kddnewton.com/2022/11/30/advent-of-yarv-part-0.html) series about Ruby's YARV virtual machine.
