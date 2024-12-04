//
//  File.swift
//  Riscv32
//

extension Riscv32 {
    
    public static func printInstruction( address: UInt32? = nil, _ instruction: Instruction, comment: String ) -> String {
        let addressString: String
        if address != nil {
            addressString = "\(address!): "
        } else {
            addressString = ""
        }
        
        switch instruction.opcode {
        case .add:
            let add = instruction as! Add
            return "\(addressString)\(add.opcode.rawValue)\t\(add.rd.preferredName()), \(add.rs1.preferredName()), \(add.rs2.preferredName())\(comment)"
        case .sub:
            let sub = instruction as! Sub
            return "\(addressString)\(sub.opcode.rawValue)\t\(sub.rd.preferredName()), \(sub.rs1.preferredName()), \(sub.rs2.preferredName())\(comment)"
        case .xor:
            let xor = instruction as! Xor
            return "\(addressString)\(xor.opcode.rawValue)\t\(xor.rd.preferredName()), \(xor.rs1.preferredName()), \(xor.rs2.preferredName())\(comment)"
        case .or:
            let or = instruction as! Or
            return "\(addressString)\(or.opcode.rawValue)\t\(or.rd.preferredName()), \(or.rs1.preferredName()), \(or.rs2.preferredName())\(comment)"
        case .and:
            let and = instruction as! And
            return "\(addressString)\(and.opcode.rawValue)\t\(and.rd.preferredName()), \(and.rs1.preferredName()), \(and.rs2.preferredName())\(comment)"
        case .sll:
            let sll = instruction as! Sll
            return "\(addressString)\(sll.opcode.rawValue)\t\(sll.rd.preferredName()), \(sll.rs1.preferredName()), \(sll.rs2.preferredName())\(comment)"
        case .srl:
            let srl = instruction as! Srl
            return "\(addressString)\(srl.opcode.rawValue)\t\(srl.rd.preferredName()), \(srl.rs1.preferredName()), \(srl.rs2.preferredName())\(comment)"
        case .sra:
            let sra = instruction as! Sra
            return "\(addressString)\(sra.opcode.rawValue)\t\(sra.rd.preferredName()), \(sra.rs1.preferredName()), \(sra.rs2.preferredName())\(comment)"
        case .slt:
            let slt = instruction as! Slt
            return "\(addressString)\(slt.opcode.rawValue)\t\(slt.rd.preferredName()), \(slt.rs1.preferredName()), \(slt.rs2.preferredName())\(comment)"
        case .sltu:
            let sltu = instruction as! Sltu
            return "\(addressString)\(sltu.opcode.rawValue)\t\(sltu.rd.preferredName()), \(sltu.rs1.preferredName()), \(sltu.rs2.preferredName())\(comment)"
            
        case .addi:
            let addi = instruction as! Addi
            return "\(addressString)\(addi.opcode.rawValue)\t\(addi.rd.preferredName()), \(addi.rs1.preferredName()), \(addi.imm12.value)\(comment)"
        case .xori:
            let xori = instruction as! Xori
            return "\(addressString)\(xori.opcode.rawValue)\t\(xori.rd.preferredName()), \(xori.rs1.preferredName()), \(xori.imm12.value)\(comment)"
        case .ori:
            let ori = instruction as! Ori
            return "\(addressString)\(ori.opcode.rawValue)\t\(ori.rd.preferredName()), \(ori.rs1.preferredName()), \(ori.imm12.value)\(comment)"
        case .andi:
            let andi = instruction as! Andi
            return "\(addressString)\(andi.opcode.rawValue)\t\(andi.rd.preferredName()), \(andi.rs1.preferredName()), \(andi.imm12.value)\(comment)"
        case .slli:
            let slli = instruction as! Slli
            return "\(addressString)\(slli.opcode.rawValue)\t\(slli.rd.preferredName()), \(slli.rs1.preferredName()), \(slli.shamt.value)\(comment)"
        case .srli:
            let srli = instruction as! Srli
            return "\(addressString)\(srli.opcode.rawValue)\t\(srli.rd.preferredName()), \(srli.rs1.preferredName()), \(srli.shamt.value)\(comment)"
        case .srai:
            let srai = instruction as! Srai
            return "\(addressString)\(srai.opcode.rawValue)\t\(srai.rd.preferredName()), \(srai.rs1.preferredName()), \(srai.shamt.value)\(comment)"
        case .slti:
            let slti = instruction as! Slti
            return "\(addressString)\(slti.opcode.rawValue)\t\(slti.rd.preferredName()), \(slti.rs1.preferredName()), \(slti.imm12.value)\(comment)"
        case .sltiu:
            let sltiu = instruction as! Sltiu
            return "\(addressString)\(sltiu.opcode.rawValue)\t\(sltiu.rd.preferredName()), \(sltiu.rs1.preferredName()), \(sltiu.imm12.value)\(comment)"
            
        case .lb:
            let lb = instruction as! Lb
            return "\(addressString)\(lb.opcode.rawValue)\t\(lb.rd.preferredName()), \(lb.imm12.value)(\(lb.rs1.preferredName()))\(comment)"
        case .lh:
            let lh = instruction as! Lh
            return "\(addressString)\(lh.opcode.rawValue)\t\(lh.rd.preferredName()), \(lh.imm12.value)(\(lh.rs1.preferredName()))\(comment)"
        case .lw:
            let lw = instruction as! Lw
            return "\(addressString)\(lw.opcode.rawValue)\t\(lw.rd.preferredName()), \(lw.imm12.value)(\(lw.rs1.preferredName()))\(comment)"
        case .lbu:
            let lbu = instruction as! Lbu
            return "\(addressString)\(lbu.opcode.rawValue)\t\(lbu.rd.preferredName()), \(lbu.imm12.value)(\(lbu.rs1.preferredName()))\(comment)"
        case .lhu:
            let lhu = instruction as! Lhu
            return "\(addressString)\(lhu.opcode.rawValue)\t\(lhu.rd.preferredName()), \(lhu.imm12.value)(\(lhu.rs1.preferredName()))\(comment)"
            
        case .sb:
            let sb = instruction as! Sb
            return "\(addressString)\(sb.opcode.rawValue)\t\(sb.rs2.preferredName()), \(sb.imm12.value)(\(sb.rs1.preferredName()))\(comment)"
        case .sh:
            let sh = instruction as! Sh
            return "\(addressString)\(sh.opcode.rawValue)\t\(sh.rs2.preferredName()), \(sh.imm12.value)(\(sh.rs1.preferredName()))\(comment)"
        case .sw:
            let sw = instruction as! Sw
            return "\(addressString)\(sw.opcode.rawValue)\t\(sw.rs2.preferredName()), \(sw.imm12.value)(\(sw.rs1.preferredName()))\(comment)"
            
        case .beq:
            let beq = instruction as! Beq
            return "\(addressString)\(beq.opcode.rawValue)\t\(beq.rs1.preferredName()), \(beq.imm12.value)\(comment)"
        case .bne:
            let bne = instruction as! Bne
            return "\(addressString)\(bne.opcode.rawValue)\t\(bne.rs1.preferredName()), \(bne.rs2.preferredName()), \(bne.imm12.value)\(comment)"
        case .blt:
            let blt = instruction as! Blt
            return "\(addressString)\(blt.opcode.rawValue)\t\(blt.rs1.preferredName()), \(blt.rs2.preferredName()), \(blt.imm12.value)\(comment)"
        case .bge:
            let bge = instruction as! Bge
            return "\(addressString)\(bge.opcode.rawValue)\t\(bge.rs1.preferredName()), \(bge.rs2.preferredName()), \(bge.imm12.value)\(comment)"
        case .bltu:
            let bltu = instruction as! Bltu
            return "\(addressString)\(bltu.opcode.rawValue)\t\(bltu.rs1.preferredName()), \(bltu.rs2.preferredName()), \(bltu.imm12.value)\(comment)"
        case .bgeu:
            let bgeu = instruction as! Bgeu
            return "\(addressString)\(bgeu.opcode.rawValue)\t\(bgeu.rs1.preferredName()), \(bgeu.rs2.preferredName()), \(bgeu.imm12.value)\(comment)"
            
        case .jal:
            let jal = instruction as! Jal
            return "\(addressString)\(jal.opcode.rawValue)\t\(jal.rd.preferredName()), \(jal.imm20.value)\(comment)"
        case .jalr:
            let jalr = instruction as! Jalr
            return "\(addressString)\(jalr.opcode.rawValue)\t\(jalr.rd.preferredName()), \(jalr.imm12.value)(\(jalr.rs1.preferredName()))\(comment)"
            
        case .lui:
            let lui = instruction as! Lui
            return "\(addressString)\(lui.opcode.rawValue)\t\(lui.rd.preferredName()), \(lui.imm20.value)\(comment)"
        case .auipc:
            let auipc = instruction as! Auipc
            return "\(addressString)\(auipc.opcode.rawValue)\t\(auipc.rd.preferredName()), \(auipc.imm20.value)\(comment)"
            
        case .ecall:
            let ecall = instruction as! Ecall
            return "\(addressString)\(ecall.opcode.rawValue)\(comment)"
            
        case .ebreak:
            let ebreak = instruction as! Ebreak
            return "\(addressString)\(ebreak.opcode.rawValue)\(comment)"
            
        case .mul:
            let mul = instruction as! Mul
            return "\(addressString)\(mul.opcode.rawValue)\t\(mul.rd.preferredName()), \(mul.rs1.preferredName()), \(mul.rs2.preferredName())\(comment)"
        case .mulh:
            let mulh = instruction as! Mulh
            return "\(addressString)\(mulh.opcode.rawValue)\t\(mulh.rd.preferredName()), \(mulh.rs1.preferredName()), \(mulh.rs2.preferredName())\(comment)"
        case .mulhsu:
            let mulhsu = instruction as! Mulhsu
            return "\(addressString)\(mulhsu.opcode.rawValue)\t\(mulhsu.rd.preferredName()), \(mulhsu.rs1.preferredName()), \(mulhsu.rs2.preferredName())\(comment)"
        case .mulhu:
            let mulhu = instruction as! Mulhu
            return "\(addressString)\(mulhu.opcode.rawValue)\t\(mulhu.rd.preferredName()), \(mulhu.rs1.preferredName()), \(mulhu.rs2.preferredName())\(comment)"
        case .div:
            let div = instruction as! Div
            return "\(addressString)\(div.opcode.rawValue)\t\(div.rd.preferredName()), \(div.rs1.preferredName()), \(div.rs2.preferredName())\(comment)"
        case .divu:
            let divu = instruction as! Divu
            return "\(addressString)\(divu.opcode.rawValue)\t\(divu.rd.preferredName()), \(divu.rs1.preferredName()), \(divu.rs2.preferredName())\(comment)"
        case .rem:
            let rem = instruction as! Rem
            return "\(addressString)\(rem.opcode.rawValue)\t\(rem.rd.preferredName()), \(rem.rs1.preferredName()), \(rem.rs2.preferredName())\(comment)"
        case .remu:
            let remu = instruction as! Remu
            return "\(addressString)\(remu.opcode.rawValue)\t\(remu.rd.preferredName()), \(remu.rs1.preferredName()), \(remu.rs2.preferredName())\(comment)"
            
        case .fenceI:
            let fenceI = instruction as! FenceI
            return "\(addressString)\(fenceI.opcode.rawValue)\(comment)"
        case .unimp:
            let unimp = instruction as! Unimp
            return "\(addressString)\(unimp.opcode.rawValue)\(comment)"
        case .wfi:
            let wfi = instruction as! Wfi
            return "\(addressString)\(wfi.opcode.rawValue)\(comment)"
            
        case .cLwsp:
            let cLwsp = instruction as! CLwsp
            return "\(addressString)\(cLwsp.opcode.rawValue)\t\(cLwsp.rd.preferredName()), \(cLwsp.imm.value)(SP)\(comment)"
        case .cSwsp:
            let cSwsp = instruction as! CSwsp
            return "\(addressString)\(cSwsp.opcode.rawValue)\t\(cSwsp.rs2.preferredName()), \(cSwsp.imm.value)(SP)\(comment)"
        case .cLw:
            let cLw = instruction as! CLw
            return "\(addressString)\(cLw.opcode.rawValue)\t\(cLw.imm.value)(\(cLw.rs1.preferredName()))\(comment)"
        case .cSw:
            let cSw = instruction as! CSw
            return "\(addressString)\(cSw.opcode.rawValue)\t\(cSw.rs2.preferredName()), \(cSw.imm.value)(\(cSw.rs1.preferredName()))\(comment)"
        case .cJ:
            let cJ = instruction as! CJ
            return "\(addressString)\(cJ.opcode.rawValue)\t\(cJ.offset.value)\(comment)"
//        case .cJal:
//            let cJal = instruction as! CJal
//            return "\(addressString)\(cJal.opcode.rawValue)\t\(cJal.offset.value)\(comment)"
        case .cJr:
            let cJr = instruction as! CJr
            return "\(addressString)\(cJr.opcode.rawValue)\t\(cJr.rs1.preferredName())\(comment)"
        case .cJalr:
            let cJalr = instruction as! CJalr
            return "\(addressString)\(cJalr.opcode.rawValue)\t\(cJalr.rs1.preferredName())\(comment)"
        case .cBeqz:
            let cBeqz = instruction as! CBeqz
            return "\(addressString)\(cBeqz.opcode.rawValue)\t\(cBeqz.rs1.preferredName()), \(cBeqz.imm.value)\(comment)"
        case .cBnez:
            let cBnez = instruction as! CBnez
            return "\(addressString)\(cBnez.opcode.rawValue)\t\(cBnez.rs1.preferredName()), \(cBnez.imm.value)\(comment)"
        case .cLi:
            let cLi = instruction as! CLi
            return "\(addressString)\(cLi.opcode.rawValue)\t\(cLi.rd.preferredName()), \(cLi.imm.value)\(comment)"
        case .cLui:
            let cLui = instruction as! CLui
            return "\(addressString)\(cLui.opcode.rawValue)\t\(cLui.rd.preferredName()), \(cLui.imm.value)\(comment)"
        case .cAddi:
            let cAddi = instruction as! CAddi
            return "\(addressString)\(cAddi.opcode.rawValue)\t\(cAddi.rd.preferredName()), \(cAddi.imm.value)\(comment)"
        case .cAddi16sp:
            let cAddi16sp = instruction as! CAddi16sp
            return "\(addressString)\(cAddi16sp.opcode.rawValue)\t\(cAddi16sp.imm.value), SP\(comment)"
        case .cAddi4spn:
            let cAddi4spn = instruction as! CAddi4spn
            return "\(addressString)\(cAddi4spn.opcode.rawValue)\t\(cAddi4spn.rd.preferredName()), SP, \(cAddi4spn.imm.value)\(comment)"
        case .cSlli:
            let cSlli = instruction as! CSlli
            return "\(addressString)\(cSlli.opcode.rawValue)\t\(cSlli.rd.preferredName()), \(cSlli.imm.value)\(comment)"
        case .cSrli:
            let cSrli = instruction as! CSrli
            return "\(addressString)\(cSrli.opcode.rawValue)\t\(cSrli.rs1.preferredName()), \(cSrli.imm.value)\(comment)"
        case .cSrai:
            let cSrai = instruction as! CSrai
            return "\(addressString)\(cSrai.opcode.rawValue)\t\(cSrai.rs1.preferredName()), \(cSrai.imm.value)\(comment)"
        case .cAndi:
            let cAndi = instruction as! CAndi
            return "\(addressString)\(cAndi.opcode.rawValue)\t\(cAndi.rs1.preferredName()), \(cAndi.imm.value)\(comment)"
        case .cMv:
            let cMv = instruction as! CMv
            return "\(addressString)\(cMv.opcode.rawValue)\t\(cMv.rs1.preferredName()), \(cMv.rs2.preferredName())\(comment)"
        case .cAdd:
            let cAdd = instruction as! CAdd
            return "\(addressString)\(cAdd.opcode.rawValue)\t\(cAdd.rs1.preferredName()), \(cAdd.rs2.preferredName())\(comment)"
        case .cAnd:
            let cAnd = instruction as! CAnd
            return "\(addressString)\(cAnd.opcode.rawValue)\t\(cAnd.rd.preferredName()), \(cAnd.rs2.preferredName())\(comment)"
        case .cOr:
            let cOr = instruction as! COr
            return "\(addressString)\(cOr.opcode.rawValue)\t\(cOr.rd.preferredName()), \(cOr.rs2.rawValue)\(comment)"
        case .cXor:
            let cXor = instruction as! CXor
            return "\(addressString)\(cXor.opcode.rawValue)\t\(cXor.rd.preferredName()), \(cXor.rs2.preferredName())\(comment)"
        case .cSub:
            let cSub = instruction as! CSub
            return "\(addressString)\(cSub.opcode.rawValue)\t\(cSub.rd.preferredName()), \(cSub.rs2.preferredName())\(comment)"
        case .cNop:
            let cNop = instruction as! CNop
            return "\(addressString)\(cNop.opcode.rawValue)\(comment)"
        case .cEbreak:
            let cEbreak = instruction as! CEbreak
            return "\(addressString)\(cEbreak.opcode.rawValue)\(comment)"
            
        case .cUnimp:
            let cUnimp = instruction as! CUnimp
            return "\(addressString)\(cUnimp.opcode.rawValue)\(comment)"
            
        case .csrrc:
            return "'\(instruction.opcode)' instruction not implemented!\(comment)"
        case .csrrci:
            return "'\(instruction.opcode)' instruction not implemented!\(comment)"
        case .csrrs:
            return "'\(instruction.opcode)' instruction not implemented!\(comment)"
        case .csrrsi:
            return "'\(instruction.opcode)' instruction not implemented!\(comment)"
        case .csrrw:
            return "'\(instruction.opcode)' instruction not implemented!\(comment)"
        case .csrrwi:
            return "'\(instruction.opcode)' instruction not implemented!\(comment)"
        case .dret:
            return "'\(instruction.opcode)' instruction not implemented!\(comment)"
        case .fenceTso:
            return "'\(instruction.opcode)' instruction not implemented!\(comment)"
        case .mret:
            return "'\(instruction.opcode)' instruction not implemented!\(comment)"
        case .sfenceVma:
            return "'\(instruction.opcode)' instruction not implemented!\(comment)"
        case .sret:
            return "'\(instruction.opcode)' instruction not implemented!\(comment)"
        }
    }
    
    public static func instructionText( address: UInt32? = nil, _ instruction: Instruction, comment: String = "" ) -> String {
        let instructionString = printInstruction( address: address, instruction, comment: comment )
        return instructionString
    }
}

