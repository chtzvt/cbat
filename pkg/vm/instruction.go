package vm

// Instruction is a single parsed CBAT instruction.
type Instruction struct {
	Op   Opcode
	Args []string // raw arguments as parsed (before interpolation)
}
