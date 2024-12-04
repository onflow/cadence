//
//  Riscv32Simulator.swift
//  Riscv32
//

extension Riscv32 {
    
    public class Registers {
        public var registers: [UInt32]
        
        public init( registerCount: Int ) {
            self.registers = Array<UInt32>( repeating: 0, count: registerCount )
            registers[ 0 ] = 0
        }
        
        public subscript( _ index: Int ) -> UInt32 {
            get {
                if index == 0 {
                    return 0
                } else {
                    return registers[ index ]
                }
            }
            set( newValue ) {
                if index != 0 {
                    registers[ index ] = newValue
                }
            }
        }
    }
    
    public class Simulator {
        public var pc: UInt32
        public var x: Registers
        public let memory: Memory
        public var instructions: [Instruction]
        public let verbose: Bool
        public var count = 0
        
        public init( memory: Memory,
                     instructionSize: Int,
                     verbose: Bool = false ) {
            self.memory = memory
            self.pc = 0
            self.verbose = verbose
            self.x = Registers( registerCount: 32 )
            self.x[ GPR.x2.rawValue ] = UInt32( memory.size )
            
//            print( "instructionSize: \(instructionSize)" )
            
            self.instructions = Riscv32.decodeInstructions( memory: memory,
                                                            instructionSize: instructionSize )
        }
        
        public func executeInstructions() {
            pc = 0
            repeat {
                count += 1
            } while execute() != nil
        }
        
        public func execute() -> UInt32? {
            let address = pc / 2
            let instruction = instructions[ Int( address ) ]
            if verbose {
                let instructionString = Riscv32.printInstruction( address: pc, instruction, comment: "" )
                print( "\(instructionString)" )
            }
            guard execute( instruction: instruction ) else { return nil }
            return pc
        }
        
        public func execute( instruction: Instruction ) -> Bool {
            switch instruction.opcode {
            case .add:
                let add = instruction as! Add
                let rs1 = Int32( bitPattern: x[ add.rs1.rawValue ] )
                let rs2 = Int32( bitPattern: x[ add.rs2.rawValue ] )
                let rd = rs1 + rs2
                x[ add.rd.rawValue ] = UInt32( bitPattern: rd )
                pc += UInt32( add.size )
                return true
            case .sub:
                let sub = instruction as! Sub
                let rs1 = Int32( bitPattern: x[ sub.rs1.rawValue ] )
                let rs2 = Int32( bitPattern: x[ sub.rs2.rawValue ] )
                let rd = rs1 - rs2
                x[ sub.rd.rawValue ] = UInt32( bitPattern: rd )
                pc += UInt32( sub.size )
                return true
            case .xor:
                let xor = instruction as! Xor
                let newValue = x[ xor.rs1.rawValue ] ^ x[ xor.rs2.rawValue ]
                x[ xor.rd.rawValue ] = newValue
                pc += UInt32( xor.size )
                return true
            case .or:
                let or = instruction as! Or
                let newValue = x[ or.rs1.rawValue ] | x[ or.rs2.rawValue ]
                x[ or.rd.rawValue ] = newValue
                pc += UInt32( or.size )
                return true
            case .and:
                let and = instruction as! And
                let newValue = x[ and.rs1.rawValue ] & x[ and.rs2.rawValue ]
                x[ and.rd.rawValue ] = newValue
                pc += UInt32( and.size )
                return true
            case .sll:
                let sll = instruction as! Sll
                let newValue = Int32( bitPattern: x[ sll.rs1.rawValue ] ) << x[ sll.rs2.rawValue ]
                x[ sll.rd.rawValue ] = UInt32( bitPattern: newValue )
                pc += UInt32( sll.size )
                return true
            case .srl:
                let srl = instruction as! Srl
                let newValue = x[ srl.rs1.rawValue ] >> x[ srl.rs2.rawValue ]
                x[ srl.rd.rawValue ] = newValue
                pc += UInt32( srl.size )
                return true
            case .sra:
                let sra = instruction as! Sra
                let newValue = Int32( bitPattern: x[ sra.rs1.rawValue ] ) >> x[ sra.rs2.rawValue ]
                x[ sra.rd.rawValue ] = UInt32( bitPattern: newValue )
                pc += UInt32( sra.size )
                return true
            case .slt:
                let slt = instruction as! Slt
                let lt = Int32( bitPattern: x[ slt.rs1.rawValue ] ) < Int32( bitPattern: x[ slt.rs2.rawValue ] )
                x[ slt.rd.rawValue ] = lt ? 1 : 0
                pc += UInt32( slt.size )
                return true
            case .sltu:
                let sltu = instruction as! Sltu
                let ltu = x[ sltu.rs1.rawValue ] < x[ sltu.rs2.rawValue ]
                x[ sltu.rd.rawValue ] = ltu ? 1 : 0
                pc += UInt32( sltu.size )
                return true
                
            case .addi:
                let addi = instruction as! Addi
                let newValue = Int32( bitPattern: x[ addi.rs1.rawValue ] ) + addi.imm12.value
                x[ addi.rd.rawValue ] = UInt32( bitPattern: newValue )
                pc += UInt32( addi.size )
                return true
            case .xori:
                let xori = instruction as! Xori
                let newValue = x[ xori.rs1.rawValue ] ^ UInt32( bitPattern: xori.imm12.value )
                x[ xori.rd.rawValue ] = newValue
                pc += UInt32( xori.size )
                return true
            case .ori:
                let ori = instruction as! Ori
                let newValue = x[ ori.rs1.rawValue ] | UInt32( bitPattern: ori.imm12.value )
                x[ ori.rd.rawValue ] = newValue
                pc += UInt32( ori.size )
                return true
            case .andi:
                let andi = instruction as! Andi
                let newValue = Int32( bitPattern: x[ andi.rs1.rawValue ] ) & andi.imm12.value
                x[ andi.rd.rawValue ] = UInt32( bitPattern: newValue )
                pc += UInt32( andi.size )
                return true
            case .slli:
                let slli = instruction as! Slli
                x[ slli.rd.rawValue ] = x[ slli.rs1.rawValue ] << slli.shamt.value
                pc += UInt32( slli.size )
                return true
            case .srli:
                let srli = instruction as! Srli
                let newValue = x[ srli.rs1.rawValue ] >> srli.shamt.value
                x[ srli.rd.rawValue ] = newValue
                pc += UInt32( srli.size )
                return true
            case .srai:
                let srai = instruction as! Srai
                let newValue = Int32( bitPattern: x[ srai.rs1.rawValue ] ) >> srai.shamt.value
                x[ srai.rd.rawValue ] = UInt32( bitPattern: newValue )
                pc += UInt32( srai.size )
                return true
            case .slti:
                let slti = instruction as! Slti
                let lti = Int32( bitPattern: x[ slti.rs1.rawValue ] ) < slti.imm12.value
                x[ slti.rd.rawValue ] = lti ? 1 : 0
                pc += UInt32( slti.size )
                return true
            case .sltiu:
                let sltiu = instruction as! Sltiu
                let ltiu = x[ sltiu.rs1.rawValue ] < UInt32( bitPattern: sltiu.imm12.value )
                x[ sltiu.rd.rawValue ] = ltiu ? 1 : 0
                pc += UInt32( sltiu.size )
                return true
                
            case .lb:
                let lb = instruction as! Lb
                let address = UInt32( bitPattern: Int32( x[ lb.rs1.rawValue ] ) + Int32( lb.imm12.value ) )
                guard let value: Int8 = memory[ address ] else { return false }
                x[ lb.rd.rawValue ] = UInt32( bitPattern: Int32( value ) )
                pc += UInt32( lb.size )
                return true
            case .lh:
                let lh = instruction as! Lh
                let address = UInt32( bitPattern: Int32( x[ lh.rs1.rawValue ] ) + Int32( lh.imm12.value ) )
                guard let value: Int16 = memory[ address ] else { return false }
                x[ lh.rd.rawValue ] = UInt32( bitPattern: Int32( value ) )
                pc += UInt32( lh.size )
                return true
            case .lw:
                let lw = instruction as! Lw
                let address = UInt32( bitPattern: Int32( x[ lw.rs1.rawValue ] ) + Int32( lw.imm12.value ) )
                guard let value: UInt32 = memory[ address ] else { return false }
                x[ lw.rd.rawValue ] = value
                pc += UInt32( lw.size )
                return true
            case .lbu:
                let lbu = instruction as! Lbu
                let address = UInt32( bitPattern: Int32( x[ lbu.rs1.rawValue ] ) + Int32( lbu.imm12.value ) )
                guard let value: UInt8 = memory[ address ] else { return false }
                x[ lbu.rd.rawValue ] = UInt32( value )
                pc += UInt32( lbu.size )
                return true
            case .lhu:
                let lhu = instruction as! Lhu
                let address = UInt32( bitPattern: Int32( x[ lhu.rs1.rawValue ] ) + Int32( lhu.imm12.value ) )
                guard let value: UInt16 = memory[ address ] else { return false }
                x[ lhu.rd.rawValue ] = UInt32( value )
                pc += UInt32( lhu.size )
                return true
                
            case .sb:
                let sb = instruction as! Sb
                let address = UInt32( bitPattern: Int32( x[ sb.rs1.rawValue ] ) + Int32( sb.imm12.value ) )
                memory[ address ] = UInt8( truncatingIfNeeded: x[ sb.rs2.rawValue ] )
                pc += UInt32( sb.size )
                return true
            case .sh:
                let sh = instruction as! Sh
                let address = UInt32( bitPattern: Int32( x[ sh.rs1.rawValue ] ) + Int32( sh.imm12.value ) )
                memory[ address ] = UInt16( truncatingIfNeeded: x[ sh.rs2.rawValue ] )
                pc += UInt32( sh.size )
                return true
            case .sw:
                let sw = instruction as! Sw
                let address = UInt32( bitPattern: Int32( x[ sw.rs1.rawValue ] ) + Int32( sw.imm12.value ) )
                memory[ address ] = x[ sw.rs2.rawValue ]
                pc += UInt32( sw.size )
                return true
                
            case .beq:
                let beq = instruction as! Beq

                let eq = x[ beq.rs1.rawValue ] == x[ beq.rs2.rawValue ]
                if eq {
                    pc = UInt32( bitPattern: Int32( bitPattern: pc ) + beq.imm12.value )
                } else {
                    pc += UInt32( beq.size )
                }
                return true
            case .bne:
                let bne = instruction as! Bne

                let ne = x[ bne.rs1.rawValue ] != x[ bne.rs2.rawValue ]
                if ne {
                    pc = UInt32( bitPattern: Int32( bitPattern: pc ) + bne.imm12.value )
                } else {
                    pc += UInt32( bne.size )
                }
                return true
            case .blt:
                let blt = instruction as! Blt

                let lt = Int32( bitPattern: x[ blt.rs1.rawValue ] ) < Int32( bitPattern: x[ blt.rs2.rawValue ] )
                if lt {
                    pc = UInt32( bitPattern: Int32( bitPattern: pc ) + blt.imm12.value )
                } else {
                    pc += UInt32( blt.size )
                }
                return true
            case .bge:
                let bge = instruction as! Bge

                let ge = Int32( bitPattern: x[ bge.rs1.rawValue ] ) >= Int32( bitPattern: x[ bge.rs2.rawValue ] )
                if ge {
                    pc = UInt32( bitPattern: Int32( bitPattern: pc ) + bge.imm12.value )
                } else {
                    pc += UInt32( bge.size )
                }
                return true
            case .bltu:
                let bltu = instruction as! Bltu

                let ltu = x[ bltu.rs1.rawValue ] < x[ bltu.rs2.rawValue ]
                if ltu {
                    pc = UInt32( bitPattern: Int32( bitPattern: pc ) + bltu.imm12.value )
                } else {
                    pc += UInt32( bltu.size )
                }
                return true
            case .bgeu:
                let bgeu = instruction as! Bgeu

                let geu = x[ bgeu.rs1.rawValue ] >= x[ bgeu.rs2.rawValue ]
                if geu {
                    pc = UInt32( bitPattern: Int32( bitPattern: pc ) + bgeu.imm12.value )
                } else {
                    pc += UInt32( bgeu.size )
                }
                return true
                
            case .jal:
                let jal = instruction as! Jal
                x[ jal.rd.rawValue ] = pc + 4
                let newPc = Int32( bitPattern: pc ) + jal.imm20.value
                pc = UInt32( bitPattern: newPc )
                return true
            case .jalr:
                let jalr = instruction as! Jalr
                let t = pc + 4
                pc = UInt32( bitPattern: Int32( bitPattern: x[ jalr.rs1.rawValue ] ) + jalr.imm12.value ) & 0xFFFFFFFE
                x[ jalr.rd.rawValue ] = t
                return true
                
            case .lui:
                let lui = instruction as! Lui
                let newValue = UInt32( truncatingIfNeeded: lui.imm20.value )
                x[ lui.rd.rawValue ] = newValue
                pc += UInt32( lui.size )
                return true
            case .auipc:
                let auipc = instruction as! Auipc
                let newValue = Int32( bitPattern: auipc.imm20.value ) + Int32( bitPattern: pc )
                x[ auipc.rd.rawValue ] = UInt32( bitPattern: newValue )
                pc += UInt32( auipc.size )
                return true
                
            case .ecall:
                let ecall = instruction as! Ecall
                pc += UInt32( ecall.size )
                return true
                
            case .ebreak:
                let ebreak = instruction as! Ebreak
                pc += UInt32( ebreak.size )
                return true
                
            case .mul:
                let mul = instruction as! Mul
                let result = Int32( bitPattern: x[ mul.rs1.rawValue ] ) &* Int32( bitPattern: x[ mul.rs2.rawValue ] )
                x[ mul.rd.rawValue ] = UInt32( bitPattern: result )
                pc += UInt32( mul.size )
                return true
            case .mulh:
                let mulh = instruction as! Mulh
                let result: Int64 = ( Int64( Int32( bitPattern: x[ mulh.rs1.rawValue ] ) ) &* Int64( Int32( bitPattern: x[ mulh.rs2.rawValue ] ) ) ) >> 32
                x[ mulh.rd.rawValue ] = UInt32( truncatingIfNeeded: result )
                pc += UInt32( mulh.size )
                return true
            case .mulhsu:
                let mulhsu = instruction as! Mulhsu
                let result: Int64 = ( Int64( Int32( bitPattern: x[ mulhsu.rs1.rawValue ] ) ) &* Int64( bitPattern: UInt64( x[ mulhsu.rs2.rawValue ] ) ) ) >> 32
                x[ mulhsu.rd.rawValue ] = UInt32( truncatingIfNeeded: result )
                pc += UInt32( mulhsu.size )
                return true
            case .mulhu:
                let mulhu = instruction as! Mulhu
                let result: Int64 = ( Int64( bitPattern: UInt64( x[ mulhu.rs1.rawValue ] ) ) &* Int64( bitPattern: UInt64( x[ mulhu.rs2.rawValue ] ) ) ) >> 32
                x[ mulhu.rd.rawValue ] = UInt32( truncatingIfNeeded: result )
                pc += UInt32( mulhu.size )
                return true
            case .div:
                let div = instruction as! Div
                let result = Int32( bitPattern: x[ div.rs1.rawValue ] ) / Int32( bitPattern: x[ div.rs2.rawValue ] )
                x[ div.rd.rawValue ] = UInt32( bitPattern: result )
                pc += UInt32( div.size )
                return true
            case .divu:
                let divu = instruction as! Divu
                let result = x[ divu.rs1.rawValue ] / x[ divu.rs2.rawValue ]
                x[ divu.rd.rawValue ] = result
                pc += UInt32( divu.size )
                return true
            case .rem:
                let rem = instruction as! Rem
                let result = Int32( bitPattern: x[ rem.rs1.rawValue ] ) % Int32( bitPattern: x[ rem.rs2.rawValue ] )
                x[ rem.rd.rawValue ] = UInt32( bitPattern: result )
                pc += UInt32( rem.size )
                return true
            case .remu:
                let remu = instruction as! Remu
                let result = x[ remu.rs1.rawValue ] % x[ remu.rs2.rawValue ]
                x[ remu.rd.rawValue ] = result
                pc += UInt32( remu.size )
                return true
                
            case .fenceI:
                let fenceI = instruction as! FenceI
                pc += UInt32( fenceI.size )
                return true
            case .unimp:
                let unimp = instruction as! Unimp
                pc += UInt32( unimp.size )
                return false
            case .wfi:
                let wfi = instruction as! Wfi
                pc += UInt32( wfi.size )
                return true
                
            case .cLwsp:
                let cLwsp = instruction as! CLwsp
                let address = UInt32( bitPattern: Int32( bitPattern: x[ GPR.x2.rawValue ] ) + Int32( bitPattern: cLwsp.imm.value ) )
                guard let value: Int32 = memory[ address ] else { return false }
                x[ cLwsp.rd.rawValue ] = UInt32( bitPattern: value )
                pc += UInt32( cLwsp.size )
                return true
            case .cSwsp:
                let cSwsp = instruction as! CSwsp
                let address = UInt32( bitPattern: Int32( bitPattern: x[ GPR.x2.rawValue ] ) + Int32( bitPattern: cSwsp.imm.value ) )
                memory[ address ] = x[ cSwsp.rs2.rawValue ]
                pc += UInt32( cSwsp.size )
                return true
            case .cLw:
                let cLw = instruction as! CLw
                let address = UInt32( bitPattern: Int32( x[ cLw.rs1.rawValue ] ) + Int32( cLw.imm.value ) )
                guard let value: UInt32 = memory[ address ] else { return false }
                x[ cLw.rd.rawValue ] = value
                pc += UInt32( cLw.size )
                return true
            case .cSw:
                let cSw = instruction as! CSw
                let rs1Value = Int32( x[ cSw.rs1.rawValue ] )
                let immValue = Int32( bitPattern: cSw.imm.value )
                let address = UInt32( bitPattern: rs1Value + immValue )
                memory[ address ] = x[ cSw.rs2.rawValue ]
                pc += UInt32( cSw.size )
                return true
            case .cJ:
                let cJ = instruction as! CJ
                pc = UInt32( bitPattern: Int32( bitPattern: pc ) + cJ.offset.value )
                return true
//            case .cJal:
//                let cJal = instruction as! CJal
//                x[ GPR.x1.rawValue ] = UInt32( bitPattern: Int32( bitPattern: pc ) + 2 )
//                pc = UInt32( bitPattern: Int32( bitPattern: pc ) + cJal.offset.value )
//                return true
            case .cJr:
                let cJr = instruction as! CJr
                pc = x[ cJr.rs1.rawValue ] 
                return true
            case .cJalr:
                let cJalr = instruction as! CJalr
                let t = UInt32( bitPattern: Int32( bitPattern: pc ) + 2 )
                pc = x[ cJalr.rs1.rawValue ]
                x[ GPR.x1.rawValue ] = t
                return true
            case .cBeqz:
                let cBeqz = instruction as! CBeqz
                if x[ cBeqz.rs1.rawValue ] == 0 {
                    pc = UInt32( bitPattern: Int32( bitPattern: pc ) + cBeqz.imm.value )
                } else {
                    pc += UInt32( cBeqz.size )
                }
                return true
            case .cBnez:
                let cBnez = instruction as! CBnez
                if x[ cBnez.rs1.rawValue ] != 0 {
                    pc = UInt32( bitPattern: Int32( bitPattern: pc ) + cBnez.imm.value )
                } else {
                    pc += UInt32( cBnez.size )
                }
                return true
            case .cLi:
                let cLi = instruction as! CLi
                x[ cLi.rd.rawValue ] = UInt32( bitPattern: cLi.imm.value )
                pc += UInt32( cLi.size )
                return true
            case .cLui:
                let cLui = instruction as! CLui
                x[ cLui.rd.rawValue ] = UInt32( bitPattern: cLui.imm.value )
                pc += UInt32( cLui.size )
                return true
            case .cAddi:
                let cAddi = instruction as! CAddi
                let newValue = Int32( bitPattern: x[ cAddi.rd.rawValue ] ) + cAddi.imm.value
                x[ cAddi.rd.rawValue ] = UInt32( bitPattern: newValue )
                pc += UInt32( cAddi.size )
                return true
            case .cAddi16sp:
                let cAddi16sp = instruction as! CAddi16sp
                x[ GPR.x2.rawValue ] = UInt32( bitPattern: Int32( bitPattern: x[ GPR.x2.rawValue ] ) + cAddi16sp.imm.value )
                pc += UInt32( cAddi16sp.size )
                return true
            case .cAddi4spn:
                let cAddi4spn = instruction as! CAddi4spn
                x[ cAddi4spn.rd.rawValue ] = UInt32( bitPattern: Int32( bitPattern: x[ GPR.x2.rawValue ] ) + Int32( bitPattern: cAddi4spn.imm.value ) )
                pc += UInt32( cAddi4spn.size )
                return true
            case .cSlli:
                let cSlli = instruction as! CSlli
                x[ cSlli.rd.rawValue ] = x[ cSlli.rd.rawValue ] &<< cSlli.imm.value
                pc += UInt32( cSlli.size )
                return true
            case .cSrli:
                let cSrli = instruction as! CSrli
                x[ cSrli.rs1.rawValue ] = x[ cSrli.rs1.rawValue ] >> cSrli.imm.value
                pc += UInt32( cSrli.size )
                return true
            case .cSrai:
                let cSrai = instruction as! CSrai
                let newValue = Int32( bitPattern: x[ cSrai.rs1.rawValue ] ) >> cSrai.imm.value
                x[ cSrai.rs1.rawValue ] = UInt32( bitPattern: newValue )
                pc += UInt32( cSrai.size )
                return true
            case .cAndi:
                let cAndi = instruction as! CAndi
                x[ cAndi.rs1.rawValue ] = x[ cAndi.rs1.rawValue ] & UInt32( bitPattern: cAndi.imm.value )
                pc += UInt32( cAndi.size )
                return true
            case .cMv:
                let cMv = instruction as! CMv
                x[ cMv.rs1.rawValue ] = x[ cMv.rs2.rawValue ]
                pc += UInt32( cMv.size )
                return true
            case .cAdd:
                let cAdd = instruction as! CAdd
                let rs1Value = Int32( bitPattern: x[ cAdd.rs1.rawValue ] )
                let rs2Value = Int32( bitPattern: x[ cAdd.rs2.rawValue ] )
                let newValue = rs1Value + rs2Value
                x[ cAdd.rs1.rawValue ] = UInt32( bitPattern: newValue )
                pc += UInt32( cAdd.size )
                return true
            case .cAnd:
                let cAnd = instruction as! CAnd
                x[ cAnd.rd.rawValue ] = x[ cAnd.rd.rawValue ] & x[ cAnd.rs2.rawValue ]
                pc += UInt32( cAnd.size )
                return true
            case .cOr:
                let cOr = instruction as! COr
                x[ cOr.rd.rawValue ] = x[ cOr.rd.rawValue ] | x[ cOr.rs2.rawValue ]
                pc += UInt32( cOr.size )
                return true
            case .cXor:
                let cXor = instruction as! CXor
                x[ cXor.rd.rawValue ] = x[ cXor.rd.rawValue ] ^ x[ cXor.rs2.rawValue ]
                pc += UInt32( cXor.size )
                return true
            case .cSub:
                let cSub = instruction as! CSub
                let newValue = Int32( bitPattern: x[ cSub.rd.rawValue ] ) - Int32( bitPattern: x[ cSub.rs2.rawValue ] )
                x[ cSub.rd.rawValue ] = UInt32( bitPattern: newValue )
                pc += UInt32( cSub.size )
                return true
            case .cNop:
                let cNop = instruction as! CNop
                pc += UInt32( cNop.size )
                return true
            case .cEbreak:
//                let cEbreak = instruction as! CEbreak
//                pc += UInt32( cEbreak.size )
//                return true
                return false
                
            case .cUnimp:
                let cUnimp = instruction as! CUnimp
                pc += UInt32( cUnimp.size )
                return true
                
            case .csrrc:
                fatalError( "'\(instruction.opcode)' instruction not implemented!" )
            case .csrrci:
                fatalError( "'\(instruction.opcode)' instruction not implemented!" )
            case .csrrs:
                fatalError( "'\(instruction.opcode)' instruction not implemented!" )
            case .csrrsi:
                fatalError( "'\(instruction.opcode)' instruction not implemented!" )
            case .csrrw:
                fatalError( "'\(instruction.opcode)' instruction not implemented!" )
            case .csrrwi:
                fatalError( "'\(instruction.opcode)' instruction not implemented!" )
            case .dret:
                fatalError( "'\(instruction.opcode)' instruction not implemented!" )
            case .fenceTso:
                fatalError( "'\(instruction.opcode)' instruction not implemented!" )
            case .mret:
                fatalError( "'\(instruction.opcode)' instruction not implemented!" )
            case .sfenceVma:
                fatalError( "'\(instruction.opcode)' instruction not implemented!" )
            case .sret:
                fatalError( "'\(instruction.opcode)' instruction not implemented!" )
            }
        }
    }
}
