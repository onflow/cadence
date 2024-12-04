package main

import (
	"fmt"
	"unsafe"

	"golang.org/x/exp/constraints"
)

type vm struct {
	instructions []instruction
	pc           uint32
	registers    [32]uint32
}

func reinterpret[T constraints.Integer, U constraints.Integer](value T) U {
	return *(*U)(unsafe.Pointer(&value))
}

func (vm *vm) run(verbose bool) {

	regs := vm.registers

	if verbose {
		fmt.Printf("# regs: %v\n\n", regs)
	}

loop:
	for {
		if verbose {
			fmt.Printf("# pc:\t%d", vm.pc)
		}

		ins := vm.instructions[vm.pc/2]
		if verbose {
			fmt.Printf(", ins: %#+v\n", ins)
		}

		switch ins := ins.(type) {
		case instructionNop:
			// no-op
			vm.pc += 2

		case instructionCMv:
			if verbose {
				fmt.Printf("c.mv	%s, %s\n", GPRNoX0[ins.rs1], GPRNoX0[ins.rs2])
			}
			regs[ins.rs1] = regs[ins.rs2]
			vm.pc += 2

		case instructionCLi:
			if verbose {
				fmt.Printf("c.li	%s, %d\n", GPRNoX0[ins.rd], ins.imm)
			}
			regs[ins.rd] = ins.imm
			vm.pc += 2

		case instructionBge:
			if verbose {
				fmt.Printf("bge	%s, %s, %d\n", GPRNoX0[ins.rs1], GPRNoX0[ins.rs2], ins.imm12)
			}

			a := reinterpret[uint32, int32](regs[ins.rs1])
			b := reinterpret[uint32, int32](regs[ins.rs2])
			if a >= b {
				vm.pc = reinterpret[int32, uint32](
					reinterpret[uint32, int32](vm.pc) + ins.imm12,
				)
			} else {
				vm.pc += 4
			}

		case instructionBne:
			if verbose {
				fmt.Printf("bne	%s, %s, %d\n", GPRNoX0[ins.rs1], GPRNoX0[ins.rs2], ins.imm12)
			}

			a := reinterpret[uint32, int32](regs[ins.rs1])
			b := reinterpret[uint32, int32](regs[ins.rs2])
			if a != b {
				vm.pc = reinterpret[int32, uint32](
					reinterpret[uint32, int32](vm.pc) + ins.imm12,
				)
			} else {
				vm.pc += 4
			}

		case instructionCAddi:
			if verbose {
				fmt.Printf("c.addi	%s, %d\n", GPRNoX0[ins.rd], ins.imm)
			}
			a := reinterpret[uint32, int32](regs[ins.rd])
			regs[ins.rd] = reinterpret[int32, uint32](a + ins.imm)
			vm.pc += 2

		case instructionCAdd:
			if verbose {
				fmt.Printf("c.add	%s, %s\n", GPRNoX0[ins.rs1], GPRNoX0[ins.rs2])
			}
			a := reinterpret[uint32, int32](regs[ins.rs1])
			b := reinterpret[uint32, int32](regs[ins.rs2])
			regs[ins.rs1] = reinterpret[int32, uint32](a + b)
			vm.pc += 2

		case instructionCEbreak:
			if verbose {
				fmt.Printf("c.ebreak\n")
			}
			break loop

		default:
			panic(fmt.Sprintf("unsupported instruction: %#+v", ins))
		}

		if verbose {
			fmt.Printf("# regs: %v\n\n", regs)
		}
	}

	vm.registers = regs
}
