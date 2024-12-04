package main

type instruction interface {
	isInstruction()
}

// Nop

type instructionNop struct{}

func (instructionNop) isInstruction() {}

// CLi

type instructionCLi struct {
	rd  uint32
	imm uint32
}

func decodeCLi(instruction uint32) instructionCLi {
	// "c.li	$rd, $imm"
	return instructionCLi{
		rd:  decode_rd_GPRNoX0_f11t7(instruction),
		imm: decode_imm_simm6_f12t12f6t2(instruction),
	}
}

func (instructionCLi) isInstruction() {}

// CMv

type instructionCMv struct {
	rs1 uint32
	rs2 uint32
}

func (instructionCMv) isInstruction() {}

func decodeCMv(instruction uint32) instructionCMv {
	// "c.mv	$rs1, $rs2"
	return instructionCMv{
		rs1: decode_rs1_GPRNoX0_f11t7(instruction),
		rs2: decode_rs2_GPRNoX0_f6t2(instruction),
	}
}

// Bge

type instructionBge struct {
	rs1   uint32
	rs2   uint32
	imm12 int32
}

func (instructionBge) isInstruction() {}

func decodeBge(instruction uint32) instructionBge {
	// "bge	$rs1, $rs2, $imm12"
	return instructionBge{
		rs1:   decode_rs1_GPR_f19t15(instruction),
		rs2:   decode_rs2_GPR_f24t20(instruction),
		imm12: decode_simm13_lsb0(decode_imm12_simm13_lsb0_f31t31f7t7f30t25f11t8(instruction)),
	}
}

// CAddi

type instructionCAddi struct {
	rd  uint32
	imm int32
}

func (instructionCAddi) isInstruction() {}

func decodeCAddi(instruction uint32) instructionCAddi {
	// "c.addi	$rd, $imm"
	return instructionCAddi{
		rd:  decode_rd_GPRNoX0_f11t7(instruction),
		imm: decode_simm6nonzero(decode_imm_simm6nonzero_f12t12f6t2(instruction)),
	}
}

// CAdd

type instructionCAdd struct {
	rs1 uint32
	rs2 uint32
}

func (instructionCAdd) isInstruction() {}

func decodeCAdd(instruction uint32) instructionCAdd {
	// "c.add	$rs1, $rs2"
	return instructionCAdd{
		rs1: decode_rs1_GPRNoX0_f11t7(instruction),
		rs2: decode_rs2_GPRNoX0_f6t2(instruction),
	}
}

// Bne

type instructionBne struct {
	rs1   uint32
	rs2   uint32
	imm12 int32
}

func (instructionBne) isInstruction() {}

func decodeBne(instruction uint32) instructionBne {
	return instructionBne{
		rs1:   decode_rs1_GPR_f19t15(instruction),
		rs2:   decode_rs2_GPR_f24t20(instruction),
		imm12: decode_simm13_lsb0(decode_imm12_simm13_lsb0_f31t31f7t7f30t25f11t8(instruction)),
	}
}

// CEbreak

type instructionCEbreak struct{}

func (instructionCEbreak) isInstruction() {}
