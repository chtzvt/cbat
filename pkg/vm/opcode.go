package vm

type Opcode int

const (
	OpNop Opcode = iota
	OpEcho
	OpSetPrompt
	OpStore
	OpStoreFromFile
	OpStoreFromFileIdx
	OpStoreArith
	OpAppendFile
	OpWriteFile
	OpWriteFileIdx
	OpTypeFile
	OpTypePaged
	OpDeleteFile
	OpMakeDir
	OpGoto
	OpGotoAddr
	OpGotoSub
	OpReturn
	OpCall
	OpLabel
	OpTerminate
	OpBreakpoint
	OpPause
	OpClearScreen
	OpColor
	OpTitle
	OpSleep
	OpIfEq
	OpIfNotEq
	OpIfEqInt
	OpIfNotEqInt
	OpIfGteInt
	OpIfLteInt
	OpIfGtInt
	OpIfLtInt
	OpIfFileExists
	OpIfNotFileExists
	OpAddImm
	OpSubImm
	OpMulImm
	OpDivImm
	OpModImm
	OpAdd
	OpSub
	OpMul
	OpDiv
	OpMod
	OpAtoI
	OpItoA
	OpConcat
	OpLength
)

var mnemonicToOpcode = map[string]Opcode{
	"nop":  OpNop,
	"e":    OpEcho,
	"stp":  OpSetPrompt,
	"st":   OpStore,
	"stf":  OpStoreFromFile,
	"stfi": OpStoreFromFileIdx,
	"sta":  OpStoreArith,
	"af":   OpAppendFile,
	"wf":   OpWriteFile,
	"wfi":  OpWriteFileIdx,
	"t":    OpTypeFile,
	"tp":   OpTypePaged,
	"df":   OpDeleteFile,
	"mkd":  OpMakeDir,
	"g":    OpGoto,
	"j":    OpGoto,
	"ga":   OpGotoAddr,
	"ja":   OpGotoAddr,
	"gsub": OpGotoSub,
	"jsub": OpGotoSub,
	"ret":  OpReturn,
	"c":    OpCall,
	"l":    OpLabel,
	"trm":  OpTerminate,
	"bp":   OpBreakpoint,
	"p":    OpPause,
	"cls":  OpClearScreen,
	"clr":  OpColor,
	"ttl":  OpTitle,
	"slp":  OpSleep,
	"ieq":  OpIfEq,
	"inq":  OpIfNotEq,
	"ieiq": OpIfEqInt,
	"iniq": OpIfNotEqInt,
	"igeqi": OpIfGteInt,
	"ileqi": OpIfLteInt,
	"igti": OpIfGtInt,
	"ilti": OpIfLtInt,
	"iex":  OpIfFileExists,
	"inx":  OpIfNotFileExists,
	"adi":  OpAddImm,
	"sbi":  OpSubImm,
	"mli":  OpMulImm,
	"dvi":  OpDivImm,
	"mdi":  OpModImm,
	"add":  OpAdd,
	"sub":  OpSub,
	"mul":  OpMul,
	"div":  OpDiv,
	"mod":  OpMod,
	"atoi": OpAtoI,
	"itoa": OpItoA,
	"cat":  OpConcat,
	"len":  OpLength,
}

func LookupOpcode(mnemonic string) Opcode {
	if op, ok := mnemonicToOpcode[mnemonic]; ok {
		return op
	}
	return OpNop
}
