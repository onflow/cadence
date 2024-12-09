package main

import "fmt"

type decoder struct {
	code   []byte
	offset int
}

func (d *decoder) readUint16() uint16 {
	offset := d.offset
	code := d.code

	lower := uint16(code[offset])
	lower += uint16(code[offset+1]) << 8

	d.offset += 2

	return lower
}

const instHasUpperMask = 0x3

func (d *decoder) readInstruction() (instruction uint32, hasUpper bool) {
	instruction = uint32(d.readUint16())

	hasUpper = (instruction & instHasUpperMask) == instHasUpperMask
	if hasUpper {
		instruction |= uint32(d.readUint16()) << 16
	}

	return
}

func (d *decoder) decodeInstructions() []instruction {
	var instructions []instruction

	for d.offset < len(d.code) {
		encodedInstruction, hasUpper := d.readInstruction()
		decodedInstruction := d.decodeInstruction(encodedInstruction)
		instructions = append(instructions, decodedInstruction)
		if hasUpper {
			instructions = append(instructions, instructionNop{})
		}
	}

	return instructions
}

func (d *decoder) decodeInstruction(instruction uint32) instruction {

	switch instruction & 0x3 {
	case 0x1:
		switch instruction & 0xE003 {
		case 0x1:
			if instruction&0xFFFF == 0x1 {
				// "c.nop	"
				return instructionNop{}
			}
			if instruction&0xE003 == 0x1 {
				// "c.addi	$rd, $imm"
				return decodeCAddi(instruction)
			}
		case 0x4001:
			// "c.li	$rd, $imm"
			return decodeCLi(instruction)
		}
	case 0x2:
		switch instruction & 0xE003 {
		case 0x8002:
			switch instruction & 0xF003 {
			case 0x8002:
				// "c.mv	$rs1, $rs2"
				return decodeCMv(instruction)
			case 0x9002:
				if instruction&0xFFFF == 0x9002 {
					// "c.ebreak	"
					return instructionCEbreak{}
				}
				if instruction&0xF003 == 0x9002 {
					// "c.add	$rs1, $rs2"
					return decodeCAdd(instruction)
				}
			}
		}
	case 0x3:
		switch instruction & 0x7F {
		case 0x63:
			switch instruction & 0x707F {
			case 0x1063:
				// "bne	$rs1, $rs2, $imm12"
				return decodeBne(instruction)

			case 0x5063:
				// "bge	$rs1, $rs2, $imm12"
				return decodeBge(instruction)
			}
		}
	}

	panic(fmt.Sprintf("Unknown instruction: 0x%08X", instruction))
}

func decode_rs1_GPRNoX0_f11t7(instruction uint32) uint32 {
	return (instruction & 0xF80) >> 7
}

func decode_rs2_GPRNoX0_f6t2(instruction uint32) uint32 {
	return (instruction & 0x7C) >> 2
}

func decode_imm_simm6_f12t12f6t2(instruction uint32) uint32 {
	return ((instruction & 0x1000) >> 7) | ((instruction & 0x7C) >> 2)
}

func decode_rd_GPRNoX0_f11t7(instruction uint32) uint32 {
	return (instruction & 0xF80) >> 7
}
func decode_imm12_simm13_lsb0_f31t31f7t7f30t25f11t8(instruction uint32) uint32 {
	return ((instruction & 0x80000000) >> 20) |
		((instruction & 0x80) << 3) |
		((instruction & 0x7E000000) >> 21) |
		((instruction & 0xF00) >> 8)
}

func decode_rs1_GPR_f19t15(instruction uint32) uint32 {
	return (instruction & 0xF8000) >> 15
}

func decode_rs2_GPR_f24t20(instruction uint32) uint32 {
	return (instruction & 0x1F00000) >> 20
}

func decode_imm_simm6nonzero_f12t12f6t2(instruction uint32) uint32 {
	return ((instruction & 0x1000) >> 7) |
		((instruction & 0x7C) >> 2)
}

func decodeInt(bitPattern uint32, bitWidth uint8, shiftCount uint8) int32 {
	var mask = (uint32(1) << bitWidth) - 1
	var signMask = uint32(1) << (bitWidth - 1)
	var maskedValue = bitPattern & mask
	isNegative := (signMask & maskedValue) != 0
	if isNegative {
		maskedValue |= ^mask
	}
	shiftedValue := maskedValue << shiftCount
	return int32(shiftedValue)
}

func decode_simm13_lsb0(value uint32) int32 {
	return decodeInt(value, 12, 1)
}

func decode_simm6nonzero(value uint32) int32 {
	return decodeInt(value, 6, 0)
}
