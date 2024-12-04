//
//  Riscv32Utilities.swift
//  Riscv32
//

public class Riscv32Utilities {
    public static func operands( mnemonic: String ) -> [(String, String)]? {
        switch mnemonic {
        case "dret":
            return []
        case "ebreak":
            return []
        case "ecall":
            return []
        case "fence.i":
            return []
        case "fence.tso":
            return []
        case "mret":
            return []
        case "sret":
            return []
        case "unimp":
            return []
        case "wfi":
            return []
        case "sfence.vma":
            return [ ( "rs1", "GPR" ), ( "rs2", "GPR" ) ]
        case "add":
            return [ ( "rd", "GPR" ), ( "rs1", "GPR" ), ( "rs2", "GPR" ) ]
        case "and":
            return [ ( "rd", "GPR" ), ( "rs1", "GPR" ), ( "rs2", "GPR" ) ]
        case "div":
            return [ ( "rd", "GPR" ), ( "rs1", "GPR" ), ( "rs2", "GPR" ) ]
        case "divu":
            return [ ( "rd", "GPR" ), ( "rs1", "GPR" ), ( "rs2", "GPR" ) ]
        case "mul":
            return [ ( "rd", "GPR" ), ( "rs1", "GPR" ), ( "rs2", "GPR" ) ]
        case "mulh":
            return [ ( "rd", "GPR" ), ( "rs1", "GPR" ), ( "rs2", "GPR" ) ]
        case "mulhsu":
            return [ ( "rd", "GPR" ), ( "rs1", "GPR" ), ( "rs2", "GPR" ) ]
        case "mulhu":
            return [ ( "rd", "GPR" ), ( "rs1", "GPR" ), ( "rs2", "GPR" ) ]
        case "or":
            return [ ( "rd", "GPR" ), ( "rs1", "GPR" ), ( "rs2", "GPR" ) ]
        case "rem":
            return [ ( "rd", "GPR" ), ( "rs1", "GPR" ), ( "rs2", "GPR" ) ]
        case "remu":
            return [ ( "rd", "GPR" ), ( "rs1", "GPR" ), ( "rs2", "GPR" ) ]
        case "sll":
            return [ ( "rd", "GPR" ), ( "rs1", "GPR" ), ( "rs2", "GPR" ) ]
        case "slt":
            return [ ( "rd", "GPR" ), ( "rs1", "GPR" ), ( "rs2", "GPR" ) ]
        case "sltu":
            return [ ( "rd", "GPR" ), ( "rs1", "GPR" ), ( "rs2", "GPR" ) ]
        case "sra":
            return [ ( "rd", "GPR" ), ( "rs1", "GPR" ), ( "rs2", "GPR" ) ]
        case "srl":
            return [ ( "rd", "GPR" ), ( "rs1", "GPR" ), ( "rs2", "GPR" ) ]
        case "sub":
            return [ ( "rd", "GPR" ), ( "rs1", "GPR" ), ( "rs2", "GPR" ) ]
        case "xor":
            return [ ( "rd", "GPR" ), ( "rs1", "GPR" ), ( "rs2", "GPR" ) ]
        case "slli":
            return [ ( "rd", "GPR" ), ( "rs1", "GPR" ), ( "shamt", "uimmlog2xlen" ) ]
        case "srai":
            return [ ( "rd", "GPR" ), ( "rs1", "GPR" ), ( "shamt", "uimmlog2xlen" ) ]
        case "srli":
            return [ ( "rd", "GPR" ), ( "rs1", "GPR" ), ( "shamt", "uimmlog2xlen" ) ]
        case "c.ebreak":
            return []
        case "c.nop":
            return []
        case "c.unimp":
            return []
        case "c.and":
            return [ ( "rd", "GPRC" ), ( "rs2", "GPRC" ) ]
        case "c.or":
            return [ ( "rd", "GPRC" ), ( "rs2", "GPRC" ) ]
        case "c.sub":
            return [ ( "rd", "GPRC" ), ( "rs2", "GPRC" ) ]
        case "c.xor":
            return [ ( "rd", "GPRC" ), ( "rs2", "GPRC" ) ]
        case "c.jalr":
            return [ ( "rs1", "GPRNoX0" ) ]
        case "c.jr":
            return [ ( "rs1", "GPRNoX0" ) ]
        case "c.add":
            return [ ( "rs1", "GPRNoX0" ), ( "rs2", "GPRNoX0" ) ]
        case "c.mv":
            return [ ( "rs1", "GPRNoX0" ), ( "rs2", "GPRNoX0" ) ]
        case "c.addi16sp":
            return [ ( "imm", "simm10_lsb0000nonzero" ), ( "rd", "SP" ) ]
        case "c.andi":
            return [ ( "imm", "simm6" ), ( "rs1", "GPRC" ) ]
        case "c.srai":
            return [ ( "imm", "uimmlog2xlennonzero" ), ( "rs1", "GPRC" ) ]
        case "c.srli":
            return [ ( "imm", "uimmlog2xlennonzero" ), ( "rs1", "GPRC" ) ]
        case "c.addi":
            return [ ( "imm", "simm6nonzero" ), ( "rd", "GPRNoX0" ) ]
        case "c.addi4spn":
            return [ ( "imm", "uimm10_lsb00nonzero" ), ( "rd", "GPRC" ), ( "rs1", "SP" ) ]
        case "c.beqz":
            return [ ( "imm", "simm9_lsb0" ), ( "rs1", "GPRC" ) ]
        case "c.bnez":
            return [ ( "imm", "simm9_lsb0" ), ( "rs1", "GPRC" ) ]
        case "c.j":
            return [ ( "offset", "simm12_lsb0" ) ]
        case "c.li":
            return [ ( "imm", "simm6" ), ( "rd", "GPRNoX0" ) ]
        case "c.lui":
            return [ ( "imm", "c_lui_imm" ), ( "rd", "GPRNoX0X2" ) ]
        case "c.lw":
            return [ ( "imm", "uimm7_lsb00" ), ( "rd", "GPRC" ), ( "rs1", "GPRCMem" ) ]
        case "c.lwsp":
            return [ ( "imm", "uimm8_lsb00" ), ( "rd", "GPRNoX0" ), ( "rs1", "SPMem" ) ]
        case "c.slli":
            return [ ( "imm", "uimmlog2xlennonzero" ), ( "rd", "GPRNoX0" ) ]
        case "c.sw":
            return [ ( "imm", "uimm7_lsb00" ), ( "rs1", "GPRCMem" ), ( "rs2", "GPRC" ) ]
        case "c.swsp":
            return [ ( "imm", "uimm8_lsb00" ), ( "rs2", "GPR" ), ( "rs1", "SPMem" ) ]
        case "addi":
            return [ ( "imm12", "simm12" ), ( "rd", "GPR" ), ( "rs1", "GPR" ) ]
        case "andi":
            return [ ( "imm12", "simm12" ), ( "rd", "GPR" ), ( "rs1", "GPR" ) ]
        case "beq":
            return [ ( "imm12", "simm13_lsb0" ), ( "rs1", "GPR" ), ( "rs2", "GPR" ) ]
        case "bge":
            return [ ( "imm12", "simm13_lsb0" ), ( "rs1", "GPR" ), ( "rs2", "GPR" ) ]
        case "bgeu":
            return [ ( "imm12", "simm13_lsb0" ), ( "rs1", "GPR" ), ( "rs2", "GPR" ) ]
        case "blt":
            return [ ( "imm12", "simm13_lsb0" ), ( "rs1", "GPR" ), ( "rs2", "GPR" ) ]
        case "bltu":
            return [ ( "imm12", "simm13_lsb0" ), ( "rs1", "GPR" ), ( "rs2", "GPR" ) ]
        case "bne":
            return [ ( "imm12", "simm13_lsb0" ), ( "rs1", "GPR" ), ( "rs2", "GPR" ) ]
        case "csrrc":
            return [ ( "imm12", "csr_sysreg" ), ( "rd", "GPR" ), ( "rs1", "GPR" ) ]
        case "csrrci":
            return [ ( "imm12", "csr_sysreg" ), ( "rd", "GPR" ), ( "rs1", "uimm5" ) ]
        case "csrrs":
            return [ ( "imm12", "csr_sysreg" ), ( "rd", "GPR" ), ( "rs1", "GPR" ) ]
        case "csrrsi":
            return [ ( "imm12", "csr_sysreg" ), ( "rd", "GPR" ), ( "rs1", "uimm5" ) ]
        case "csrrw":
            return [ ( "imm12", "csr_sysreg" ), ( "rd", "GPR" ), ( "rs1", "GPR" ) ]
        case "csrrwi":
            return [ ( "imm12", "csr_sysreg" ), ( "rd", "GPR" ), ( "rs1", "uimm5" ) ]
        case "jalr":
            return [ ( "imm12", "simm12" ), ( "rd", "GPR" ), ( "rs1", "GPR" ) ]
        case "lb":
            return [ ( "imm12", "simm12" ), ( "rd", "GPR" ), ( "rs1", "GPRMem" ) ]
        case "lbu":
            return [ ( "imm12", "simm12" ), ( "rd", "GPR" ), ( "rs1", "GPRMem" ) ]
        case "lh":
            return [ ( "imm12", "simm12" ), ( "rd", "GPR" ), ( "rs1", "GPRMem" ) ]
        case "lhu":
            return [ ( "imm12", "simm12" ), ( "rd", "GPR" ), ( "rs1", "GPRMem" ) ]
        case "lw":
            return [ ( "imm12", "simm12" ), ( "rd", "GPR" ), ( "rs1", "GPRMem" ) ]
        case "ori":
            return [ ( "imm12", "simm12" ), ( "rd", "GPR" ), ( "rs1", "GPR" ) ]
        case "sb":
            return [ ( "imm12", "simm12" ), ( "rs1", "GPRMem" ), ( "rs2", "GPR" ) ]
        case "sh":
            return [ ( "imm12", "simm12" ), ( "rs1", "GPRMem" ), ( "rs2", "GPR" ) ]
        case "slti":
            return [ ( "imm12", "simm12" ), ( "rd", "GPR" ), ( "rs1", "GPR" ) ]
        case "sltiu":
            return [ ( "imm12", "simm12" ), ( "rd", "GPR" ), ( "rs1", "GPR" ) ]
        case "sw":
            return [ ( "imm12", "simm12" ), ( "rs1", "GPRMem" ), ( "rs2", "GPR" ) ]
        case "xori":
            return [ ( "imm12", "simm12" ), ( "rd", "GPR" ), ( "rs1", "GPR" ) ]
        case "auipc":
            return [ ( "imm20", "uimm20_auipc" ), ( "rd", "GPR" ) ]
        case "jal":
            return [ ( "imm20", "simm21_lsb0_jal" ), ( "rd", "GPR" ) ]
        case "lui":
            return [ ( "imm20", "uimm20_lui" ), ( "rd", "GPR" ) ]
        default:
            return nil
        }
    }
}
