# CBAT VM Instruction Set

## Instruction Set

### I/O and Files
| Instruction | Description | Implemented |
| --- | --- | --- |
| `e STRING` | Echo to terminal. Supports `%VAR%` interpolation. | Yes |
| `af STRING,PATH` | Create/append string + newline to file | Yes |
| `wf STRING,PATH` | Write/overwrite file with string | Yes |
| `wfi PATH,INDEX,VALUE` | Write value at character index in file | Yes |
| `df PATH` | Delete file | Yes |
| `t PATH` | Type (print) file contents | Yes |
| `tp PATH,N` | Type file contents with pagination (N lines per page) | Yes |
| `stp IDENT,PROMPT` | Prompt user for input, store in IDENT | Yes |

### Variables
| Instruction | Description | Implemented |
| --- | --- | --- |
| `st IDENT,VALUE` | Set IDENT to VALUE | Yes |
| `stf IDENT,PATH` | Set IDENT from file contents | Yes |
| `stfi IDENT,PATH,INDEX` | Set IDENT from character at INDEX in file | Yes |
| `sta IDENT,EXPR` | Arithmetic expression set (supports +, -, *, /, % with precedence) | Yes |

### Control Flow
| Instruction | Description | Implemented |
| --- | --- | --- |
| `g LABEL` | Goto label | Yes |
| `j LABEL` | Jump to label (alias for `g`) | Yes |
| `ga ADDRESS` | Goto numeric address | Yes |
| `ja ADDRESS` | Jump to address (alias for `ga`) | Yes |
| `l LABEL` | Declare a label | Yes |
| `c NAME` | Call subroutine program (by filename) | Yes |
| `gsub LABEL` | Goto subroutine: push return address, jump to label | Yes |
| `jsub LABEL` | Jump subroutine (alias for `gsub`) | Yes |
| `ret` | Return from subroutine (pop return address stack) | Yes |
| `trm` | Terminate program | Yes |
| `bp` | Breakpoint (enter debugger) | Yes |
| `nop` | No operation | Yes |

### Conditionals

All conditionals support an optional LABEL parameter. If LABEL is specified and the condition is true, execution jumps to LABEL. If LABEL is unspecified, the instruction immediately following will be executed in the true case or skipped in the false case.

| Instruction | Description | Implemented |
| --- | --- | --- |
| `ieq IDENT,VALUE[,LABEL]` | If equal (string comparison) | Yes |
| `inq IDENT,VALUE[,LABEL]` | If not equal (string comparison) | Yes |
| `ieiq IDENT,VALUE[,LABEL]` | If equal (integer comparison) | Yes |
| `iniq IDENT,VALUE[,LABEL]` | If not equal (integer comparison) | Yes |
| `igeqi IDENT,VALUE[,LABEL]` | If greater or equal (integer) | Yes |
| `ileqi IDENT,VALUE[,LABEL]` | If less or equal (integer) | Yes |
| `igti IDENT,VALUE[,LABEL]` | If greater than (integer) | Yes |
| `ilti IDENT,VALUE[,LABEL]` | If less than (integer) | Yes |
| `iex PATH[,LABEL]` | If file exists | Yes |
| `inx PATH[,LABEL]` | If file not exists | Yes |

### Mathematics (immediate)

Operate on a variable in-place with an immediate value.

| Instruction | Description | Implemented |
| --- | --- | --- |
| `adi IDENT,VALUE` | Add immediate | Yes |
| `sbi IDENT,VALUE` | Subtract immediate | Yes |
| `mli IDENT,VALUE` | Multiply immediate | Yes |
| `dvi IDENT,VALUE` | Divide immediate (div by zero = 0) | Yes |
| `mdi IDENT,VALUE` | Modulo immediate | Yes |

### Mathematics (register-to-register)

Store result of operation on two variables into a destination.

| Instruction | Description | Implemented |
| --- | --- | --- |
| `add DEST,OP1,OP2` | Add | Yes |
| `sub DEST,OP1,OP2` | Subtract | Yes |
| `mul DEST,OP1,OP2` | Multiply | Yes |
| `div DEST,OP1,OP2` | Divide (div by zero = 0) | Yes |
| `mod DEST,OP1,OP2` | Modulo | Yes |

### String Operations
| Instruction | Description | Implemented |
| --- | --- | --- |
| `cat DEST,A,B` | Concatenate variables A and B, store in DEST | Yes |
| `len DEST,IDENT` | Store length of variable IDENT in DEST | Yes |
| `atoi DEST,CHAR` | Convert character to ASCII integer | Yes |
| `itoa DEST,IDENT` | Convert integer variable to ASCII character | Yes |

### Utility
| Instruction | Description | Implemented |
| --- | --- | --- |
| `cls` | Clear terminal (ANSI escape) | Yes |
| `clr CODE` | Set terminal color (batch color code, e.g. `17`) | Yes |
| `ttl STRING` | Set terminal title (ANSI escape) | Yes |
| `p [INT]` | Pause. Quiet pause if INT = 0. | Yes |
| `slp MILLISECONDS` | Sleep for specified milliseconds | Yes |
| `mkd PATH` | Create directory/file namespace | Yes |

## Built-in Identifiers

| Identifier | Description |
| --- | --- |
| `%username%` | Current OS user |
| `%date%` | Current date (YYYY-MM-DD) |
| `%time%` | Current time (HH:MM:SS) |
| `%random%` | Random number 0-32767 |
| `%PC%` | Current program counter |
| `%RA%` | Return address (set by gsub/jsub) |

## CBAT File Format (.cbat)

### Sections

| Section | Description | Optional |
| --- | --- | --- |
| `.global` | Global configuration | Yes |
| `.files` | Pre-defined global file table | Yes |
| `.header` | Program file header | No |
| `.labels` | Program file label table | Yes |
| `.instrs` | Instructions | No |

Multiple programs can be defined in a single `.cbat` file by repeating `.header` through `.instrs` sections. This enables batch's `call` semantics — a set of `.bat` files can be compiled into a single `.cbat` object.

### Comments

Comments are denoted by a `;` character. They may appear on their own line or at the end of an instruction line.

### `.global` Section

| Key | Description |
| --- | --- |
| `entry NAME` | Entry point program name |

### `.header` Section

| Key | Description |
| --- | --- |
| `filename NAME` | Program/subroutine name |
| `entry N` | Entry point instruction index (default: 0) |
| `ver VERSION` | Version string |
| `debug` | Enable debug logging |
| `dump_state_on_exit` | Emit VM state as JSON to stderr on termination |

### `.files` Section

Pre-loaded in-memory files. Format: `"FILENAME","CONTENT"`. Supports `\n` for newlines.

### `.labels` Section

Pre-defined label mappings. Format: `LABEL:INDEX`.

### `.instrs` Section

One instruction per line. Arguments are comma-separated. String arguments are quoted with `"`. Supports `%VAR%` interpolation in string arguments.

## CBAT VM Debugger

The debugger is entered via a `bp` instruction or by setting `debug` in the header.

### Commands

#### Program Counter

| Command | Description |
| --- | --- |
| `pc` | Current PC value |
| `j` | Jump to address |

#### Execution

| Command | Description |
| --- | --- |
| `d` | Dump instructions (internal repr) |
| `dc` | Dump instructions (cbat repr) |
| `insi` | Insert instruction at address |
| `deli` | Delete instruction at address |

#### Variables

| Command | Description |
| --- | --- |
| `vset` | Set variable |
| `vdel` | Delete variable |
| `vdmp` | Dump variables |

#### Labels

| Command | Description |
| --- | --- |
| `lbc` | Create label |
| `lbd` | Delete label |
| `lbdmp` | Dump labels |

#### Files

| Command | Description |
| --- | --- |
| `fcp` | File copy |
| `fdel` | File delete |
| `fr` | Read file content |
| `fw` | Write file content |
| `fdmp` | Dump files |

#### State & Output

| Command | Description |
| --- | --- |
| `sdmp` | Dump full VM state as JSON |
| `foutc` | Dump program as CBAT |
| `foutb` | Dump program as batch |
| `trm` | Terminate execution |
| `step` | Toggle step log |
| `dlog` | Toggle debug log |

#### Exit

| Command | Description |
| --- | --- |
| `q` | Exit debugger and resume execution |

## Example

```batchfile
TEST.BAT

echo "Hello there!"
echo "Welcome to the test program."
set /p NAME=Please enter your name:
echo %NAME% >>"USERS.TXT"
if %NAME% EQU "Charlton" goto WAZZUP
echo "Goodbye, %NAME%"
exit

:WAZZUP
echo "Wazzup dad"
```

```
.header
    filename "TEST.BAT"
    entry 0
    ver 0.1
.labels
    WAZZUP:7
.instrs
    e "Hello there!"
    e "Welcome to the test program."
    stp NAME,"Please enter your name:"
    af "%NAME%","USERS.TXT"
    ieq NAME,"Charlton",WAZZUP
    e "Goodbye, %NAME%"
    trm
    l WAZZUP
    e "Wazzup dad"
```
