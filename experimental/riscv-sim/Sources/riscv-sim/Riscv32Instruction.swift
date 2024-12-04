//
// Riscv32Instruction.swift
// Riscv32
//

extension Riscv32 {
    
    public enum Opcode : String {
        case dret = "dret"
        case ebreak = "ebreak"
        case ecall = "ecall"
        case fenceI = "fence.i"
        case fenceTso = "fence.tso"
        case mret = "mret"
        case sret = "sret"
        case unimp = "unimp"
        case wfi = "wfi"
        case sfenceVma = "sfence.vma"
        case add = "add"
        case and = "and"
        case div = "div"
        case divu = "divu"
        case mul = "mul"
        case mulh = "mulh"
        case mulhsu = "mulhsu"
        case mulhu = "mulhu"
        case or = "or"
        case rem = "rem"
        case remu = "remu"
        case sll = "sll"
        case slt = "slt"
        case sltu = "sltu"
        case sra = "sra"
        case srl = "srl"
        case sub = "sub"
        case xor = "xor"
        case slli = "slli"
        case srai = "srai"
        case srli = "srli"
        case cEbreak = "c.ebreak"
        case cNop = "c.nop"
        case cUnimp = "c.unimp"
        case cAnd = "c.and"
        case cOr = "c.or"
        case cSub = "c.sub"
        case cXor = "c.xor"
        case cJalr = "c.jalr"
        case cJr = "c.jr"
        case cAdd = "c.add"
        case cMv = "c.mv"
        case cAddi16sp = "c.addi16sp"
        case cAndi = "c.andi"
        case cSrai = "c.srai"
        case cSrli = "c.srli"
        case cAddi = "c.addi"
        case cAddi4spn = "c.addi4spn"
        case cBeqz = "c.beqz"
        case cBnez = "c.bnez"
        case cJ = "c.j"
        case cLi = "c.li"
        case cLui = "c.lui"
        case cLw = "c.lw"
        case cLwsp = "c.lwsp"
        case cSlli = "c.slli"
        case cSw = "c.sw"
        case cSwsp = "c.swsp"
        case addi = "addi"
        case andi = "andi"
        case beq = "beq"
        case bge = "bge"
        case bgeu = "bgeu"
        case blt = "blt"
        case bltu = "bltu"
        case bne = "bne"
        case csrrc = "csrrc"
        case csrrci = "csrrci"
        case csrrs = "csrrs"
        case csrrsi = "csrrsi"
        case csrrw = "csrrw"
        case csrrwi = "csrrwi"
        case jalr = "jalr"
        case lb = "lb"
        case lbu = "lbu"
        case lh = "lh"
        case lhu = "lhu"
        case lw = "lw"
        case ori = "ori"
        case sb = "sb"
        case sh = "sh"
        case slti = "slti"
        case sltiu = "sltiu"
        case sw = "sw"
        case xori = "xori"
        case auipc = "auipc"
        case jal = "jal"
        case lui = "lui"
    }
    
    open class Instruction {
        public let opcode: Opcode
        public let size: Int
        public init( opcode: Opcode, size: Int ) {
            self.opcode = opcode
            self.size = size
        }
    }
    
    public class Dret: Instruction {
        
        public init?() {
            super.init( opcode: .dret, size: 4 )
        }
        
        public func encode() -> UInt32 {
            let encoded: UInt32 = 0x7b200073
            return encoded
        }
    }
    
    public class Ebreak: Instruction {
        
        public init?() {
            super.init( opcode: .ebreak, size: 4 )
        }
        
        public func encode() -> UInt32 {
            let encoded: UInt32 = 0x100073
            return encoded
        }
    }
    
    public class Ecall: Instruction {
        
        public init?() {
            super.init( opcode: .ecall, size: 4 )
        }
        
        public func encode() -> UInt32 {
            let encoded: UInt32 = 0x73
            return encoded
        }
    }
    
    public class FenceI: Instruction {
        
        public init?() {
            super.init( opcode: .fenceI, size: 4 )
        }
        
        public func encode() -> UInt32 {
            let encoded: UInt32 = 0x100f
            return encoded
        }
    }
    
    public class FenceTso: Instruction {
        
        public init?() {
            super.init( opcode: .fenceTso, size: 4 )
        }
        
        public func encode() -> UInt32 {
            let encoded: UInt32 = 0x8330000f
            return encoded
        }
    }
    
    public class Mret: Instruction {
        
        public init?() {
            super.init( opcode: .mret, size: 4 )
        }
        
        public func encode() -> UInt32 {
            let encoded: UInt32 = 0x30200073
            return encoded
        }
    }
    
    public class Sret: Instruction {
        
        public init?() {
            super.init( opcode: .sret, size: 4 )
        }
        
        public func encode() -> UInt32 {
            let encoded: UInt32 = 0x10200073
            return encoded
        }
    }
    
    public class Unimp: Instruction {
        
        public init?() {
            super.init( opcode: .unimp, size: 4 )
        }
        
        public func encode() -> UInt32 {
            let encoded: UInt32 = 0xc0001073
            return encoded
        }
    }
    
    public class Wfi: Instruction {
        
        public init?() {
            super.init( opcode: .wfi, size: 4 )
        }
        
        public func encode() -> UInt32 {
            let encoded: UInt32 = 0x10500073
            return encoded
        }
    }
    
    public class SfenceVma: Instruction {
        public let rs1: GPR
        public let rs2: GPR
        
        public init( rs1: GPR, rs2: GPR ) {
            self.rs1 = rs1
            self.rs2 = rs2
            super.init( opcode: .sfenceVma, size: 4 )
        }
        
        public init?( rs1: UInt32, rs2: UInt32 ) {
            guard let rs1Decoded = Riscv32.GPR.decode( rs1 ) else { return nil }
            self.rs1 = rs1Decoded
            guard let rs2Decoded = Riscv32.GPR.decode( rs2 ) else { return nil }
            self.rs2 = rs2Decoded
            super.init( opcode: .sfenceVma, size: 4 )
        }
        
        public func encode() -> UInt32 {
            var encoded: UInt32 = 0x12000073
            encoded |= ( rs1.encode() & 0x1f ) << 15
            encoded |= ( rs2.encode() & 0x1f ) << 20
            return encoded
        }
    }
    
    public class Add: Instruction {
        public let rd: GPR
        public let rs1: GPR
        public let rs2: GPR
        
        public init( rd: GPR, rs1: GPR, rs2: GPR ) {
            self.rd = rd
            self.rs1 = rs1
            self.rs2 = rs2
            super.init( opcode: .add, size: 4 )
        }
        
        public init?( rd: UInt32, rs1: UInt32, rs2: UInt32 ) {
            guard let rdDecoded = Riscv32.GPR.decode( rd ) else { return nil }
            self.rd = rdDecoded
            guard let rs1Decoded = Riscv32.GPR.decode( rs1 ) else { return nil }
            self.rs1 = rs1Decoded
            guard let rs2Decoded = Riscv32.GPR.decode( rs2 ) else { return nil }
            self.rs2 = rs2Decoded
            super.init( opcode: .add, size: 4 )
        }
        
        public func encode() -> UInt32 {
            var encoded: UInt32 = 0x33
            encoded |= ( rd.encode() & 0x1f ) << 7
            encoded |= ( rs1.encode() & 0x1f ) << 15
            encoded |= ( rs2.encode() & 0x1f ) << 20
            return encoded
        }
    }
    
    public class And: Instruction {
        public let rd: GPR
        public let rs1: GPR
        public let rs2: GPR
        
        public init( rd: GPR, rs1: GPR, rs2: GPR ) {
            self.rd = rd
            self.rs1 = rs1
            self.rs2 = rs2
            super.init( opcode: .and, size: 4 )
        }
        
        public init?( rd: UInt32, rs1: UInt32, rs2: UInt32 ) {
            guard let rdDecoded = Riscv32.GPR.decode( rd ) else { return nil }
            self.rd = rdDecoded
            guard let rs1Decoded = Riscv32.GPR.decode( rs1 ) else { return nil }
            self.rs1 = rs1Decoded
            guard let rs2Decoded = Riscv32.GPR.decode( rs2 ) else { return nil }
            self.rs2 = rs2Decoded
            super.init( opcode: .and, size: 4 )
        }
        
        public func encode() -> UInt32 {
            var encoded: UInt32 = 0x7033
            encoded |= ( rd.encode() & 0x1f ) << 7
            encoded |= ( rs1.encode() & 0x1f ) << 15
            encoded |= ( rs2.encode() & 0x1f ) << 20
            return encoded
        }
    }
    
    public class Div: Instruction {
        public let rd: GPR
        public let rs1: GPR
        public let rs2: GPR
        
        public init( rd: GPR, rs1: GPR, rs2: GPR ) {
            self.rd = rd
            self.rs1 = rs1
            self.rs2 = rs2
            super.init( opcode: .div, size: 4 )
        }
        
        public init?( rd: UInt32, rs1: UInt32, rs2: UInt32 ) {
            guard let rdDecoded = Riscv32.GPR.decode( rd ) else { return nil }
            self.rd = rdDecoded
            guard let rs1Decoded = Riscv32.GPR.decode( rs1 ) else { return nil }
            self.rs1 = rs1Decoded
            guard let rs2Decoded = Riscv32.GPR.decode( rs2 ) else { return nil }
            self.rs2 = rs2Decoded
            super.init( opcode: .div, size: 4 )
        }
        
        public func encode() -> UInt32 {
            var encoded: UInt32 = 0x2004033
            encoded |= ( rd.encode() & 0x1f ) << 7
            encoded |= ( rs1.encode() & 0x1f ) << 15
            encoded |= ( rs2.encode() & 0x1f ) << 20
            return encoded
        }
    }
    
    public class Divu: Instruction {
        public let rd: GPR
        public let rs1: GPR
        public let rs2: GPR
        
        public init( rd: GPR, rs1: GPR, rs2: GPR ) {
            self.rd = rd
            self.rs1 = rs1
            self.rs2 = rs2
            super.init( opcode: .divu, size: 4 )
        }
        
        public init?( rd: UInt32, rs1: UInt32, rs2: UInt32 ) {
            guard let rdDecoded = Riscv32.GPR.decode( rd ) else { return nil }
            self.rd = rdDecoded
            guard let rs1Decoded = Riscv32.GPR.decode( rs1 ) else { return nil }
            self.rs1 = rs1Decoded
            guard let rs2Decoded = Riscv32.GPR.decode( rs2 ) else { return nil }
            self.rs2 = rs2Decoded
            super.init( opcode: .divu, size: 4 )
        }
        
        public func encode() -> UInt32 {
            var encoded: UInt32 = 0x2005033
            encoded |= ( rd.encode() & 0x1f ) << 7
            encoded |= ( rs1.encode() & 0x1f ) << 15
            encoded |= ( rs2.encode() & 0x1f ) << 20
            return encoded
        }
    }
    
    public class Mul: Instruction {
        public let rd: GPR
        public let rs1: GPR
        public let rs2: GPR
        
        public init( rd: GPR, rs1: GPR, rs2: GPR ) {
            self.rd = rd
            self.rs1 = rs1
            self.rs2 = rs2
            super.init( opcode: .mul, size: 4 )
        }
        
        public init?( rd: UInt32, rs1: UInt32, rs2: UInt32 ) {
            guard let rdDecoded = Riscv32.GPR.decode( rd ) else { return nil }
            self.rd = rdDecoded
            guard let rs1Decoded = Riscv32.GPR.decode( rs1 ) else { return nil }
            self.rs1 = rs1Decoded
            guard let rs2Decoded = Riscv32.GPR.decode( rs2 ) else { return nil }
            self.rs2 = rs2Decoded
            super.init( opcode: .mul, size: 4 )
        }
        
        public func encode() -> UInt32 {
            var encoded: UInt32 = 0x2000033
            encoded |= ( rd.encode() & 0x1f ) << 7
            encoded |= ( rs1.encode() & 0x1f ) << 15
            encoded |= ( rs2.encode() & 0x1f ) << 20
            return encoded
        }
    }
    
    public class Mulh: Instruction {
        public let rd: GPR
        public let rs1: GPR
        public let rs2: GPR
        
        public init( rd: GPR, rs1: GPR, rs2: GPR ) {
            self.rd = rd
            self.rs1 = rs1
            self.rs2 = rs2
            super.init( opcode: .mulh, size: 4 )
        }
        
        public init?( rd: UInt32, rs1: UInt32, rs2: UInt32 ) {
            guard let rdDecoded = Riscv32.GPR.decode( rd ) else { return nil }
            self.rd = rdDecoded
            guard let rs1Decoded = Riscv32.GPR.decode( rs1 ) else { return nil }
            self.rs1 = rs1Decoded
            guard let rs2Decoded = Riscv32.GPR.decode( rs2 ) else { return nil }
            self.rs2 = rs2Decoded
            super.init( opcode: .mulh, size: 4 )
        }
        
        public func encode() -> UInt32 {
            var encoded: UInt32 = 0x2001033
            encoded |= ( rd.encode() & 0x1f ) << 7
            encoded |= ( rs1.encode() & 0x1f ) << 15
            encoded |= ( rs2.encode() & 0x1f ) << 20
            return encoded
        }
    }
    
    public class Mulhsu: Instruction {
        public let rd: GPR
        public let rs1: GPR
        public let rs2: GPR
        
        public init( rd: GPR, rs1: GPR, rs2: GPR ) {
            self.rd = rd
            self.rs1 = rs1
            self.rs2 = rs2
            super.init( opcode: .mulhsu, size: 4 )
        }
        
        public init?( rd: UInt32, rs1: UInt32, rs2: UInt32 ) {
            guard let rdDecoded = Riscv32.GPR.decode( rd ) else { return nil }
            self.rd = rdDecoded
            guard let rs1Decoded = Riscv32.GPR.decode( rs1 ) else { return nil }
            self.rs1 = rs1Decoded
            guard let rs2Decoded = Riscv32.GPR.decode( rs2 ) else { return nil }
            self.rs2 = rs2Decoded
            super.init( opcode: .mulhsu, size: 4 )
        }
        
        public func encode() -> UInt32 {
            var encoded: UInt32 = 0x2002033
            encoded |= ( rd.encode() & 0x1f ) << 7
            encoded |= ( rs1.encode() & 0x1f ) << 15
            encoded |= ( rs2.encode() & 0x1f ) << 20
            return encoded
        }
    }
    
    public class Mulhu: Instruction {
        public let rd: GPR
        public let rs1: GPR
        public let rs2: GPR
        
        public init( rd: GPR, rs1: GPR, rs2: GPR ) {
            self.rd = rd
            self.rs1 = rs1
            self.rs2 = rs2
            super.init( opcode: .mulhu, size: 4 )
        }
        
        public init?( rd: UInt32, rs1: UInt32, rs2: UInt32 ) {
            guard let rdDecoded = Riscv32.GPR.decode( rd ) else { return nil }
            self.rd = rdDecoded
            guard let rs1Decoded = Riscv32.GPR.decode( rs1 ) else { return nil }
            self.rs1 = rs1Decoded
            guard let rs2Decoded = Riscv32.GPR.decode( rs2 ) else { return nil }
            self.rs2 = rs2Decoded
            super.init( opcode: .mulhu, size: 4 )
        }
        
        public func encode() -> UInt32 {
            var encoded: UInt32 = 0x2003033
            encoded |= ( rd.encode() & 0x1f ) << 7
            encoded |= ( rs1.encode() & 0x1f ) << 15
            encoded |= ( rs2.encode() & 0x1f ) << 20
            return encoded
        }
    }
    
    public class Or: Instruction {
        public let rd: GPR
        public let rs1: GPR
        public let rs2: GPR
        
        public init( rd: GPR, rs1: GPR, rs2: GPR ) {
            self.rd = rd
            self.rs1 = rs1
            self.rs2 = rs2
            super.init( opcode: .or, size: 4 )
        }
        
        public init?( rd: UInt32, rs1: UInt32, rs2: UInt32 ) {
            guard let rdDecoded = Riscv32.GPR.decode( rd ) else { return nil }
            self.rd = rdDecoded
            guard let rs1Decoded = Riscv32.GPR.decode( rs1 ) else { return nil }
            self.rs1 = rs1Decoded
            guard let rs2Decoded = Riscv32.GPR.decode( rs2 ) else { return nil }
            self.rs2 = rs2Decoded
            super.init( opcode: .or, size: 4 )
        }
        
        public func encode() -> UInt32 {
            var encoded: UInt32 = 0x6033
            encoded |= ( rd.encode() & 0x1f ) << 7
            encoded |= ( rs1.encode() & 0x1f ) << 15
            encoded |= ( rs2.encode() & 0x1f ) << 20
            return encoded
        }
    }
    
    public class Rem: Instruction {
        public let rd: GPR
        public let rs1: GPR
        public let rs2: GPR
        
        public init( rd: GPR, rs1: GPR, rs2: GPR ) {
            self.rd = rd
            self.rs1 = rs1
            self.rs2 = rs2
            super.init( opcode: .rem, size: 4 )
        }
        
        public init?( rd: UInt32, rs1: UInt32, rs2: UInt32 ) {
            guard let rdDecoded = Riscv32.GPR.decode( rd ) else { return nil }
            self.rd = rdDecoded
            guard let rs1Decoded = Riscv32.GPR.decode( rs1 ) else { return nil }
            self.rs1 = rs1Decoded
            guard let rs2Decoded = Riscv32.GPR.decode( rs2 ) else { return nil }
            self.rs2 = rs2Decoded
            super.init( opcode: .rem, size: 4 )
        }
        
        public func encode() -> UInt32 {
            var encoded: UInt32 = 0x2006033
            encoded |= ( rd.encode() & 0x1f ) << 7
            encoded |= ( rs1.encode() & 0x1f ) << 15
            encoded |= ( rs2.encode() & 0x1f ) << 20
            return encoded
        }
    }
    
    public class Remu: Instruction {
        public let rd: GPR
        public let rs1: GPR
        public let rs2: GPR
        
        public init( rd: GPR, rs1: GPR, rs2: GPR ) {
            self.rd = rd
            self.rs1 = rs1
            self.rs2 = rs2
            super.init( opcode: .remu, size: 4 )
        }
        
        public init?( rd: UInt32, rs1: UInt32, rs2: UInt32 ) {
            guard let rdDecoded = Riscv32.GPR.decode( rd ) else { return nil }
            self.rd = rdDecoded
            guard let rs1Decoded = Riscv32.GPR.decode( rs1 ) else { return nil }
            self.rs1 = rs1Decoded
            guard let rs2Decoded = Riscv32.GPR.decode( rs2 ) else { return nil }
            self.rs2 = rs2Decoded
            super.init( opcode: .remu, size: 4 )
        }
        
        public func encode() -> UInt32 {
            var encoded: UInt32 = 0x2007033
            encoded |= ( rd.encode() & 0x1f ) << 7
            encoded |= ( rs1.encode() & 0x1f ) << 15
            encoded |= ( rs2.encode() & 0x1f ) << 20
            return encoded
        }
    }
    
    public class Sll: Instruction {
        public let rd: GPR
        public let rs1: GPR
        public let rs2: GPR
        
        public init( rd: GPR, rs1: GPR, rs2: GPR ) {
            self.rd = rd
            self.rs1 = rs1
            self.rs2 = rs2
            super.init( opcode: .sll, size: 4 )
        }
        
        public init?( rd: UInt32, rs1: UInt32, rs2: UInt32 ) {
            guard let rdDecoded = Riscv32.GPR.decode( rd ) else { return nil }
            self.rd = rdDecoded
            guard let rs1Decoded = Riscv32.GPR.decode( rs1 ) else { return nil }
            self.rs1 = rs1Decoded
            guard let rs2Decoded = Riscv32.GPR.decode( rs2 ) else { return nil }
            self.rs2 = rs2Decoded
            super.init( opcode: .sll, size: 4 )
        }
        
        public func encode() -> UInt32 {
            var encoded: UInt32 = 0x1033
            encoded |= ( rd.encode() & 0x1f ) << 7
            encoded |= ( rs1.encode() & 0x1f ) << 15
            encoded |= ( rs2.encode() & 0x1f ) << 20
            return encoded
        }
    }
    
    public class Slt: Instruction {
        public let rd: GPR
        public let rs1: GPR
        public let rs2: GPR
        
        public init( rd: GPR, rs1: GPR, rs2: GPR ) {
            self.rd = rd
            self.rs1 = rs1
            self.rs2 = rs2
            super.init( opcode: .slt, size: 4 )
        }
        
        public init?( rd: UInt32, rs1: UInt32, rs2: UInt32 ) {
            guard let rdDecoded = Riscv32.GPR.decode( rd ) else { return nil }
            self.rd = rdDecoded
            guard let rs1Decoded = Riscv32.GPR.decode( rs1 ) else { return nil }
            self.rs1 = rs1Decoded
            guard let rs2Decoded = Riscv32.GPR.decode( rs2 ) else { return nil }
            self.rs2 = rs2Decoded
            super.init( opcode: .slt, size: 4 )
        }
        
        public func encode() -> UInt32 {
            var encoded: UInt32 = 0x2033
            encoded |= ( rd.encode() & 0x1f ) << 7
            encoded |= ( rs1.encode() & 0x1f ) << 15
            encoded |= ( rs2.encode() & 0x1f ) << 20
            return encoded
        }
    }
    
    public class Sltu: Instruction {
        public let rd: GPR
        public let rs1: GPR
        public let rs2: GPR
        
        public init( rd: GPR, rs1: GPR, rs2: GPR ) {
            self.rd = rd
            self.rs1 = rs1
            self.rs2 = rs2
            super.init( opcode: .sltu, size: 4 )
        }
        
        public init?( rd: UInt32, rs1: UInt32, rs2: UInt32 ) {
            guard let rdDecoded = Riscv32.GPR.decode( rd ) else { return nil }
            self.rd = rdDecoded
            guard let rs1Decoded = Riscv32.GPR.decode( rs1 ) else { return nil }
            self.rs1 = rs1Decoded
            guard let rs2Decoded = Riscv32.GPR.decode( rs2 ) else { return nil }
            self.rs2 = rs2Decoded
            super.init( opcode: .sltu, size: 4 )
        }
        
        public func encode() -> UInt32 {
            var encoded: UInt32 = 0x3033
            encoded |= ( rd.encode() & 0x1f ) << 7
            encoded |= ( rs1.encode() & 0x1f ) << 15
            encoded |= ( rs2.encode() & 0x1f ) << 20
            return encoded
        }
    }
    
    public class Sra: Instruction {
        public let rd: GPR
        public let rs1: GPR
        public let rs2: GPR
        
        public init( rd: GPR, rs1: GPR, rs2: GPR ) {
            self.rd = rd
            self.rs1 = rs1
            self.rs2 = rs2
            super.init( opcode: .sra, size: 4 )
        }
        
        public init?( rd: UInt32, rs1: UInt32, rs2: UInt32 ) {
            guard let rdDecoded = Riscv32.GPR.decode( rd ) else { return nil }
            self.rd = rdDecoded
            guard let rs1Decoded = Riscv32.GPR.decode( rs1 ) else { return nil }
            self.rs1 = rs1Decoded
            guard let rs2Decoded = Riscv32.GPR.decode( rs2 ) else { return nil }
            self.rs2 = rs2Decoded
            super.init( opcode: .sra, size: 4 )
        }
        
        public func encode() -> UInt32 {
            var encoded: UInt32 = 0x40005033
            encoded |= ( rd.encode() & 0x1f ) << 7
            encoded |= ( rs1.encode() & 0x1f ) << 15
            encoded |= ( rs2.encode() & 0x1f ) << 20
            return encoded
        }
    }
    
    public class Srl: Instruction {
        public let rd: GPR
        public let rs1: GPR
        public let rs2: GPR
        
        public init( rd: GPR, rs1: GPR, rs2: GPR ) {
            self.rd = rd
            self.rs1 = rs1
            self.rs2 = rs2
            super.init( opcode: .srl, size: 4 )
        }
        
        public init?( rd: UInt32, rs1: UInt32, rs2: UInt32 ) {
            guard let rdDecoded = Riscv32.GPR.decode( rd ) else { return nil }
            self.rd = rdDecoded
            guard let rs1Decoded = Riscv32.GPR.decode( rs1 ) else { return nil }
            self.rs1 = rs1Decoded
            guard let rs2Decoded = Riscv32.GPR.decode( rs2 ) else { return nil }
            self.rs2 = rs2Decoded
            super.init( opcode: .srl, size: 4 )
        }
        
        public func encode() -> UInt32 {
            var encoded: UInt32 = 0x5033
            encoded |= ( rd.encode() & 0x1f ) << 7
            encoded |= ( rs1.encode() & 0x1f ) << 15
            encoded |= ( rs2.encode() & 0x1f ) << 20
            return encoded
        }
    }
    
    public class Sub: Instruction {
        public let rd: GPR
        public let rs1: GPR
        public let rs2: GPR
        
        public init( rd: GPR, rs1: GPR, rs2: GPR ) {
            self.rd = rd
            self.rs1 = rs1
            self.rs2 = rs2
            super.init( opcode: .sub, size: 4 )
        }
        
        public init?( rd: UInt32, rs1: UInt32, rs2: UInt32 ) {
            guard let rdDecoded = Riscv32.GPR.decode( rd ) else { return nil }
            self.rd = rdDecoded
            guard let rs1Decoded = Riscv32.GPR.decode( rs1 ) else { return nil }
            self.rs1 = rs1Decoded
            guard let rs2Decoded = Riscv32.GPR.decode( rs2 ) else { return nil }
            self.rs2 = rs2Decoded
            super.init( opcode: .sub, size: 4 )
        }
        
        public func encode() -> UInt32 {
            var encoded: UInt32 = 0x40000033
            encoded |= ( rd.encode() & 0x1f ) << 7
            encoded |= ( rs1.encode() & 0x1f ) << 15
            encoded |= ( rs2.encode() & 0x1f ) << 20
            return encoded
        }
    }
    
    public class Xor: Instruction {
        public let rd: GPR
        public let rs1: GPR
        public let rs2: GPR
        
        public init( rd: GPR, rs1: GPR, rs2: GPR ) {
            self.rd = rd
            self.rs1 = rs1
            self.rs2 = rs2
            super.init( opcode: .xor, size: 4 )
        }
        
        public init?( rd: UInt32, rs1: UInt32, rs2: UInt32 ) {
            guard let rdDecoded = Riscv32.GPR.decode( rd ) else { return nil }
            self.rd = rdDecoded
            guard let rs1Decoded = Riscv32.GPR.decode( rs1 ) else { return nil }
            self.rs1 = rs1Decoded
            guard let rs2Decoded = Riscv32.GPR.decode( rs2 ) else { return nil }
            self.rs2 = rs2Decoded
            super.init( opcode: .xor, size: 4 )
        }
        
        public func encode() -> UInt32 {
            var encoded: UInt32 = 0x4033
            encoded |= ( rd.encode() & 0x1f ) << 7
            encoded |= ( rs1.encode() & 0x1f ) << 15
            encoded |= ( rs2.encode() & 0x1f ) << 20
            return encoded
        }
    }
    
    public class Slli: Instruction {
        public let rd: GPR
        public let rs1: GPR
        public let shamt: uimmlog2xlen
        
        public init( rd: GPR, rs1: GPR, shamt: uimmlog2xlen ) {
            self.rd = rd
            self.rs1 = rs1
            self.shamt = shamt
            super.init( opcode: .slli, size: 4 )
        }
        
        public init?( rd: UInt32, rs1: UInt32, shamt: UInt32 ) {
            guard let rdDecoded = Riscv32.GPR.decode( rd ) else { return nil }
            self.rd = rdDecoded
            guard let rs1Decoded = Riscv32.GPR.decode( rs1 ) else { return nil }
            self.rs1 = rs1Decoded
            guard let shamtDecoded = Riscv32.uimmlog2xlen.decode( shamt ) else { return nil }
            self.shamt = shamtDecoded
            super.init( opcode: .slli, size: 4 )
        }
        
        public func encode() -> UInt32 {
            var encoded: UInt32 = 0x1013
            encoded |= ( rd.encode() & 0x1f ) << 7
            encoded |= ( rs1.encode() & 0x1f ) << 15
            encoded |= ( shamt.encode() & 0x3f ) << 20
            return encoded
        }
    }
    
    public class Srai: Instruction {
        public let rd: GPR
        public let rs1: GPR
        public let shamt: uimmlog2xlen
        
        public init( rd: GPR, rs1: GPR, shamt: uimmlog2xlen ) {
            self.rd = rd
            self.rs1 = rs1
            self.shamt = shamt
            super.init( opcode: .srai, size: 4 )
        }
        
        public init?( rd: UInt32, rs1: UInt32, shamt: UInt32 ) {
            guard let rdDecoded = Riscv32.GPR.decode( rd ) else { return nil }
            self.rd = rdDecoded
            guard let rs1Decoded = Riscv32.GPR.decode( rs1 ) else { return nil }
            self.rs1 = rs1Decoded
            guard let shamtDecoded = Riscv32.uimmlog2xlen.decode( shamt ) else { return nil }
            self.shamt = shamtDecoded
            super.init( opcode: .srai, size: 4 )
        }
        
        public func encode() -> UInt32 {
            var encoded: UInt32 = 0x40005013
            encoded |= ( rd.encode() & 0x1f ) << 7
            encoded |= ( rs1.encode() & 0x1f ) << 15
            encoded |= ( shamt.encode() & 0x3f ) << 20
            return encoded
        }
    }
    
    public class Srli: Instruction {
        public let rd: GPR
        public let rs1: GPR
        public let shamt: uimmlog2xlen
        
        public init( rd: GPR, rs1: GPR, shamt: uimmlog2xlen ) {
            self.rd = rd
            self.rs1 = rs1
            self.shamt = shamt
            super.init( opcode: .srli, size: 4 )
        }
        
        public init?( rd: UInt32, rs1: UInt32, shamt: UInt32 ) {
            guard let rdDecoded = Riscv32.GPR.decode( rd ) else { return nil }
            self.rd = rdDecoded
            guard let rs1Decoded = Riscv32.GPR.decode( rs1 ) else { return nil }
            self.rs1 = rs1Decoded
            guard let shamtDecoded = Riscv32.uimmlog2xlen.decode( shamt ) else { return nil }
            self.shamt = shamtDecoded
            super.init( opcode: .srli, size: 4 )
        }
        
        public func encode() -> UInt32 {
            var encoded: UInt32 = 0x5013
            encoded |= ( rd.encode() & 0x1f ) << 7
            encoded |= ( rs1.encode() & 0x1f ) << 15
            encoded |= ( shamt.encode() & 0x3f ) << 20
            return encoded
        }
    }
    
    public class CEbreak: Instruction {
        
        public init?() {
            super.init( opcode: .cEbreak, size: 2 )
        }
        
        public func encode() -> UInt32 {
            let encoded: UInt32 = 0x9002
            return encoded
        }
    }
    
    public class CNop: Instruction {
        
        public init?() {
            super.init( opcode: .cNop, size: 2 )
        }
        
        public func encode() -> UInt32 {
            let encoded: UInt32 = 0x1
            return encoded
        }
    }
    
    public class CUnimp: Instruction {
        
        public init?() {
            super.init( opcode: .cUnimp, size: 2 )
        }
        
        public func encode() -> UInt32 {
            let encoded: UInt32 = 0x0
            return encoded
        }
    }
    
    public class CAnd: Instruction {
        public let rd: GPRC
        public let rs2: GPRC
        
        public init( rd: GPRC, rs2: GPRC ) {
            self.rd = rd
            self.rs2 = rs2
            super.init( opcode: .cAnd, size: 2 )
        }
        
        public init?( rd: UInt32, rs2: UInt32 ) {
            guard let rdDecoded = Riscv32.GPRC.decode( rd ) else { return nil }
            self.rd = rdDecoded
            guard let rs2Decoded = Riscv32.GPRC.decode( rs2 ) else { return nil }
            self.rs2 = rs2Decoded
            super.init( opcode: .cAnd, size: 2 )
        }
        
        public func encode() -> UInt32 {
            var encoded: UInt32 = 0x8c61
            encoded |= ( rd.encode() & 0x7 ) << 7
            encoded |= ( rs2.encode() & 0x7 ) << 2
            return encoded
        }
    }
    
    public class COr: Instruction {
        public let rd: GPRC
        public let rs2: GPRC
        
        public init( rd: GPRC, rs2: GPRC ) {
            self.rd = rd
            self.rs2 = rs2
            super.init( opcode: .cOr, size: 2 )
        }
        
        public init?( rd: UInt32, rs2: UInt32 ) {
            guard let rdDecoded = Riscv32.GPRC.decode( rd ) else { return nil }
            self.rd = rdDecoded
            guard let rs2Decoded = Riscv32.GPRC.decode( rs2 ) else { return nil }
            self.rs2 = rs2Decoded
            super.init( opcode: .cOr, size: 2 )
        }
        
        public func encode() -> UInt32 {
            var encoded: UInt32 = 0x8c41
            encoded |= ( rd.encode() & 0x7 ) << 7
            encoded |= ( rs2.encode() & 0x7 ) << 2
            return encoded
        }
    }
    
    public class CSub: Instruction {
        public let rd: GPRC
        public let rs2: GPRC
        
        public init( rd: GPRC, rs2: GPRC ) {
            self.rd = rd
            self.rs2 = rs2
            super.init( opcode: .cSub, size: 2 )
        }
        
        public init?( rd: UInt32, rs2: UInt32 ) {
            guard let rdDecoded = Riscv32.GPRC.decode( rd ) else { return nil }
            self.rd = rdDecoded
            guard let rs2Decoded = Riscv32.GPRC.decode( rs2 ) else { return nil }
            self.rs2 = rs2Decoded
            super.init( opcode: .cSub, size: 2 )
        }
        
        public func encode() -> UInt32 {
            var encoded: UInt32 = 0x8c01
            encoded |= ( rd.encode() & 0x7 ) << 7
            encoded |= ( rs2.encode() & 0x7 ) << 2
            return encoded
        }
    }
    
    public class CXor: Instruction {
        public let rd: GPRC
        public let rs2: GPRC
        
        public init( rd: GPRC, rs2: GPRC ) {
            self.rd = rd
            self.rs2 = rs2
            super.init( opcode: .cXor, size: 2 )
        }
        
        public init?( rd: UInt32, rs2: UInt32 ) {
            guard let rdDecoded = Riscv32.GPRC.decode( rd ) else { return nil }
            self.rd = rdDecoded
            guard let rs2Decoded = Riscv32.GPRC.decode( rs2 ) else { return nil }
            self.rs2 = rs2Decoded
            super.init( opcode: .cXor, size: 2 )
        }
        
        public func encode() -> UInt32 {
            var encoded: UInt32 = 0x8c21
            encoded |= ( rd.encode() & 0x7 ) << 7
            encoded |= ( rs2.encode() & 0x7 ) << 2
            return encoded
        }
    }
    
    public class CJalr: Instruction {
        public let rs1: GPRNoX0
        
        public init( rs1: GPRNoX0 ) {
            self.rs1 = rs1
            super.init( opcode: .cJalr, size: 2 )
        }
        
        public init?( rs1: UInt32 ) {
            guard let rs1Decoded = Riscv32.GPRNoX0.decode( rs1 ) else { return nil }
            self.rs1 = rs1Decoded
            super.init( opcode: .cJalr, size: 2 )
        }
        
        public func encode() -> UInt32 {
            var encoded: UInt32 = 0x9002
            encoded |= ( rs1.encode() & 0x1f ) << 7
            return encoded
        }
    }
    
    public class CJr: Instruction {
        public let rs1: GPRNoX0
        
        public init( rs1: GPRNoX0 ) {
            self.rs1 = rs1
            super.init( opcode: .cJr, size: 2 )
        }
        
        public init?( rs1: UInt32 ) {
            guard let rs1Decoded = Riscv32.GPRNoX0.decode( rs1 ) else { return nil }
            self.rs1 = rs1Decoded
            super.init( opcode: .cJr, size: 2 )
        }
        
        public func encode() -> UInt32 {
            var encoded: UInt32 = 0x8002
            encoded |= ( rs1.encode() & 0x1f ) << 7
            return encoded
        }
    }
    
    public class CAdd: Instruction {
        public let rs1: GPRNoX0
        public let rs2: GPRNoX0
        
        public init( rs1: GPRNoX0, rs2: GPRNoX0 ) {
            self.rs1 = rs1
            self.rs2 = rs2
            super.init( opcode: .cAdd, size: 2 )
        }
        
        public init?( rs1: UInt32, rs2: UInt32 ) {
            guard let rs1Decoded = Riscv32.GPRNoX0.decode( rs1 ) else { return nil }
            self.rs1 = rs1Decoded
            guard let rs2Decoded = Riscv32.GPRNoX0.decode( rs2 ) else { return nil }
            self.rs2 = rs2Decoded
            super.init( opcode: .cAdd, size: 2 )
        }
        
        public func encode() -> UInt32 {
            var encoded: UInt32 = 0x9002
            encoded |= ( rs1.encode() & 0x1f ) << 7
            encoded |= ( rs2.encode() & 0x1f ) << 2
            return encoded
        }
    }
    
    public class CMv: Instruction {
        public let rs1: GPRNoX0
        public let rs2: GPRNoX0
        
        public init( rs1: GPRNoX0, rs2: GPRNoX0 ) {
            self.rs1 = rs1
            self.rs2 = rs2
            super.init( opcode: .cMv, size: 2 )
        }
        
        public init?( rs1: UInt32, rs2: UInt32 ) {
            guard let rs1Decoded = Riscv32.GPRNoX0.decode( rs1 ) else { return nil }
            self.rs1 = rs1Decoded
            guard let rs2Decoded = Riscv32.GPRNoX0.decode( rs2 ) else { return nil }
            self.rs2 = rs2Decoded
            super.init( opcode: .cMv, size: 2 )
        }
        
        public func encode() -> UInt32 {
            var encoded: UInt32 = 0x8002
            encoded |= ( rs1.encode() & 0x1f ) << 7
            encoded |= ( rs2.encode() & 0x1f ) << 2
            return encoded
        }
    }
    
    public class CAddi16sp: Instruction {
        public let imm: simm10_lsb0000nonzero
        public let rd: SP
        
        public init( imm: simm10_lsb0000nonzero, rd: SP ) {
            self.imm = imm
            self.rd = rd
            super.init( opcode: .cAddi16sp, size: 2 )
        }
        
        public init?( imm: UInt32, rd: UInt32 ) {
            guard let immDecoded = Riscv32.simm10_lsb0000nonzero.decode( imm ) else { return nil }
            self.imm = immDecoded
            guard let rdDecoded = Riscv32.SP.decode( rd ) else { return nil }
            self.rd = rdDecoded
            super.init( opcode: .cAddi16sp, size: 2 )
        }
        
        public func encode() -> UInt32 {
            var encoded: UInt32 = 0x6101
            encoded |= ( imm.encode() & 0x200 ) << 3
            encoded |= ( imm.encode() & 0x180 ) >> 4
            encoded |= ( imm.encode() & 0x40 ) >> 1
            encoded |= ( imm.encode() & 0x20 ) >> 3
            encoded |= ( imm.encode() & 0x10 ) << 2
            return encoded
        }
    }
    
    public class CAndi: Instruction {
        public let imm: simm6
        public let rs1: GPRC
        
        public init( imm: simm6, rs1: GPRC ) {
            self.imm = imm
            self.rs1 = rs1
            super.init( opcode: .cAndi, size: 2 )
        }
        
        public init?( imm: UInt32, rs1: UInt32 ) {
            guard let immDecoded = Riscv32.simm6.decode( imm ) else { return nil }
            self.imm = immDecoded
            guard let rs1Decoded = Riscv32.GPRC.decode( rs1 ) else { return nil }
            self.rs1 = rs1Decoded
            super.init( opcode: .cAndi, size: 2 )
        }
        
        public func encode() -> UInt32 {
            var encoded: UInt32 = 0x8801
            encoded |= ( imm.encode() & 0x20 ) << 7
            encoded |= ( imm.encode() & 0x1f ) << 2
            encoded |= ( rs1.encode() & 0x7 ) << 7
            return encoded
        }
    }
    
    public class CSrai: Instruction {
        public let imm: uimmlog2xlennonzero
        public let rs1: GPRC
        
        public init( imm: uimmlog2xlennonzero, rs1: GPRC ) {
            self.imm = imm
            self.rs1 = rs1
            super.init( opcode: .cSrai, size: 2 )
        }
        
        public init?( imm: UInt32, rs1: UInt32 ) {
            guard let immDecoded = Riscv32.uimmlog2xlennonzero.decode( imm ) else { return nil }
            self.imm = immDecoded
            guard let rs1Decoded = Riscv32.GPRC.decode( rs1 ) else { return nil }
            self.rs1 = rs1Decoded
            super.init( opcode: .cSrai, size: 2 )
        }
        
        public func encode() -> UInt32 {
            var encoded: UInt32 = 0x8401
            encoded |= ( imm.encode() & 0x20 ) << 7
            encoded |= ( imm.encode() & 0x1f ) << 2
            encoded |= ( rs1.encode() & 0x7 ) << 7
            return encoded
        }
    }
    
    public class CSrli: Instruction {
        public let imm: uimmlog2xlennonzero
        public let rs1: GPRC
        
        public init( imm: uimmlog2xlennonzero, rs1: GPRC ) {
            self.imm = imm
            self.rs1 = rs1
            super.init( opcode: .cSrli, size: 2 )
        }
        
        public init?( imm: UInt32, rs1: UInt32 ) {
            guard let immDecoded = Riscv32.uimmlog2xlennonzero.decode( imm ) else { return nil }
            self.imm = immDecoded
            guard let rs1Decoded = Riscv32.GPRC.decode( rs1 ) else { return nil }
            self.rs1 = rs1Decoded
            super.init( opcode: .cSrli, size: 2 )
        }
        
        public func encode() -> UInt32 {
            var encoded: UInt32 = 0x8001
            encoded |= ( imm.encode() & 0x20 ) << 7
            encoded |= ( imm.encode() & 0x1f ) << 2
            encoded |= ( rs1.encode() & 0x7 ) << 7
            return encoded
        }
    }
    
    public class CAddi: Instruction {
        public let imm: simm6nonzero
        public let rd: GPRNoX0
        
        public init( imm: simm6nonzero, rd: GPRNoX0 ) {
            self.imm = imm
            self.rd = rd
            super.init( opcode: .cAddi, size: 2 )
        }
        
        public init?( imm: UInt32, rd: UInt32 ) {
            guard let immDecoded = Riscv32.simm6nonzero.decode( imm ) else { return nil }
            self.imm = immDecoded
            guard let rdDecoded = Riscv32.GPRNoX0.decode( rd ) else { return nil }
            self.rd = rdDecoded
            super.init( opcode: .cAddi, size: 2 )
        }
        
        public func encode() -> UInt32 {
            var encoded: UInt32 = 0x1
            encoded |= ( imm.encode() & 0x20 ) << 7
            encoded |= ( imm.encode() & 0x1f ) << 2
            encoded |= ( rd.encode() & 0x1f ) << 7
            return encoded
        }
    }
    
    public class CAddi4spn: Instruction {
        public let imm: uimm10_lsb00nonzero
        public let rd: GPRC
        public let rs1: SP
        
        public init( imm: uimm10_lsb00nonzero, rd: GPRC, rs1: SP ) {
            self.imm = imm
            self.rd = rd
            self.rs1 = rs1
            super.init( opcode: .cAddi4spn, size: 2 )
        }
        
        public init?( imm: UInt32, rd: UInt32, rs1: UInt32 ) {
            guard let immDecoded = Riscv32.uimm10_lsb00nonzero.decode( imm ) else { return nil }
            self.imm = immDecoded
            guard let rdDecoded = Riscv32.GPRC.decode( rd ) else { return nil }
            self.rd = rdDecoded
            guard let rs1Decoded = Riscv32.SP.decode( rs1 ) else { return nil }
            self.rs1 = rs1Decoded
            super.init( opcode: .cAddi4spn, size: 2 )
        }
        
        public func encode() -> UInt32 {
            var encoded: UInt32 = 0x0
            encoded |= ( imm.encode() & 0x3c0 ) << 1
            encoded |= ( imm.encode() & 0x30 ) << 7
            encoded |= ( imm.encode() & 0x8 ) << 2
            encoded |= ( imm.encode() & 0x4 ) << 4
            encoded |= ( rd.encode() & 0x7 ) << 2
            return encoded
        }
    }
    
    public class CBeqz: Instruction {
        public let imm: simm9_lsb0
        public let rs1: GPRC
        
        public init( imm: simm9_lsb0, rs1: GPRC ) {
            self.imm = imm
            self.rs1 = rs1
            super.init( opcode: .cBeqz, size: 2 )
        }
        
        public init?( imm: UInt32, rs1: UInt32 ) {
            guard let immDecoded = Riscv32.simm9_lsb0.decode( imm ) else { return nil }
            self.imm = immDecoded
            guard let rs1Decoded = Riscv32.GPRC.decode( rs1 ) else { return nil }
            self.rs1 = rs1Decoded
            super.init( opcode: .cBeqz, size: 2 )
        }
        
        public func encode() -> UInt32 {
            var encoded: UInt32 = 0xc001
            encoded |= ( imm.encode() & 0x80 ) << 5
            encoded |= imm.encode() & 0x60
            encoded |= ( imm.encode() & 0x10 ) >> 2
            encoded |= ( imm.encode() & 0xc ) << 8
            encoded |= ( imm.encode() & 0x3 ) << 3
            encoded |= ( rs1.encode() & 0x7 ) << 7
            return encoded
        }
    }
    
    public class CBnez: Instruction {
        public let imm: simm9_lsb0
        public let rs1: GPRC
        
        public init( imm: simm9_lsb0, rs1: GPRC ) {
            self.imm = imm
            self.rs1 = rs1
            super.init( opcode: .cBnez, size: 2 )
        }
        
        public init?( imm: UInt32, rs1: UInt32 ) {
            guard let immDecoded = Riscv32.simm9_lsb0.decode( imm ) else { return nil }
            self.imm = immDecoded
            guard let rs1Decoded = Riscv32.GPRC.decode( rs1 ) else { return nil }
            self.rs1 = rs1Decoded
            super.init( opcode: .cBnez, size: 2 )
        }
        
        public func encode() -> UInt32 {
            var encoded: UInt32 = 0xe001
            encoded |= ( imm.encode() & 0x80 ) << 5
            encoded |= imm.encode() & 0x60
            encoded |= ( imm.encode() & 0x10 ) >> 2
            encoded |= ( imm.encode() & 0xc ) << 8
            encoded |= ( imm.encode() & 0x3 ) << 3
            encoded |= ( rs1.encode() & 0x7 ) << 7
            return encoded
        }
    }
    
    public class CJ: Instruction {
        public let offset: simm12_lsb0
        
        public init( offset: simm12_lsb0 ) {
            self.offset = offset
            super.init( opcode: .cJ, size: 2 )
        }
        
        public init?( offset: UInt32 ) {
            guard let offsetDecoded = Riscv32.simm12_lsb0.decode( offset ) else { return nil }
            self.offset = offsetDecoded
            super.init( opcode: .cJ, size: 2 )
        }
        
        public func encode() -> UInt32 {
            var encoded: UInt32 = 0xa001
            encoded |= ( offset.encode() & 0x400 ) << 2
            encoded |= ( offset.encode() & 0x200 ) >> 1
            encoded |= ( offset.encode() & 0x180 ) << 2
            encoded |= offset.encode() & 0x40
            encoded |= ( offset.encode() & 0x20 ) << 2
            encoded |= ( offset.encode() & 0x10 ) >> 2
            encoded |= ( offset.encode() & 0x8 ) << 8
            encoded |= ( offset.encode() & 0x7 ) << 3
            return encoded
        }
    }
    
    public class CLi: Instruction {
        public let imm: simm6
        public let rd: GPRNoX0
        
        public init( imm: simm6, rd: GPRNoX0 ) {
            self.imm = imm
            self.rd = rd
            super.init( opcode: .cLi, size: 2 )
        }
        
        public init?( imm: UInt32, rd: UInt32 ) {
            guard let immDecoded = Riscv32.simm6.decode( imm ) else { return nil }
            self.imm = immDecoded
            guard let rdDecoded = Riscv32.GPRNoX0.decode( rd ) else { return nil }
            self.rd = rdDecoded
            super.init( opcode: .cLi, size: 2 )
        }
        
        public func encode() -> UInt32 {
            var encoded: UInt32 = 0x4001
            encoded |= ( imm.encode() & 0x20 ) << 7
            encoded |= ( imm.encode() & 0x1f ) << 2
            encoded |= ( rd.encode() & 0x1f ) << 7
            return encoded
        }
    }
    
    public class CLui: Instruction {
        public let imm: c_lui_imm
        public let rd: GPRNoX0X2
        
        public init( imm: c_lui_imm, rd: GPRNoX0X2 ) {
            self.imm = imm
            self.rd = rd
            super.init( opcode: .cLui, size: 2 )
        }
        
        public init?( imm: UInt32, rd: UInt32 ) {
            guard let immDecoded = Riscv32.c_lui_imm.decode( imm ) else { return nil }
            self.imm = immDecoded
            guard let rdDecoded = Riscv32.GPRNoX0X2.decode( rd ) else { return nil }
            self.rd = rdDecoded
            super.init( opcode: .cLui, size: 2 )
        }
        
        public func encode() -> UInt32 {
            var encoded: UInt32 = 0x6001
            encoded |= ( imm.encode() & 0x20 ) << 7
            encoded |= ( imm.encode() & 0x1f ) << 2
            encoded |= ( rd.encode() & 0x1f ) << 7
            return encoded
        }
    }
    
    public class CLw: Instruction {
        public let imm: uimm7_lsb00
        public let rd: GPRC
        public let rs1: GPRCMem
        
        public init( imm: uimm7_lsb00, rd: GPRC, rs1: GPRCMem ) {
            self.imm = imm
            self.rd = rd
            self.rs1 = rs1
            super.init( opcode: .cLw, size: 2 )
        }
        
        public init?( imm: UInt32, rd: UInt32, rs1: UInt32 ) {
            guard let immDecoded = Riscv32.uimm7_lsb00.decode( imm ) else { return nil }
            self.imm = immDecoded
            guard let rdDecoded = Riscv32.GPRC.decode( rd ) else { return nil }
            self.rd = rdDecoded
            guard let rs1Decoded = Riscv32.GPRCMem.decode( rs1 ) else { return nil }
            self.rs1 = rs1Decoded
            super.init( opcode: .cLw, size: 2 )
        }
        
        public func encode() -> UInt32 {
            var encoded: UInt32 = 0x4000
            encoded |= ( imm.encode() & 0x40 ) >> 1
            encoded |= ( imm.encode() & 0x38 ) << 7
            encoded |= ( imm.encode() & 0x4 ) << 4
            encoded |= ( rd.encode() & 0x7 ) << 2
            encoded |= ( rs1.encode() & 0x7 ) << 7
            return encoded
        }
    }
    
    public class CLwsp: Instruction {
        public let imm: uimm8_lsb00
        public let rd: GPRNoX0
        public let rs1: SPMem
        
        public init( imm: uimm8_lsb00, rd: GPRNoX0, rs1: SPMem ) {
            self.imm = imm
            self.rd = rd
            self.rs1 = rs1
            super.init( opcode: .cLwsp, size: 2 )
        }
        
        public init?( imm: UInt32, rd: UInt32, rs1: UInt32 ) {
            guard let immDecoded = Riscv32.uimm8_lsb00.decode( imm ) else { return nil }
            self.imm = immDecoded
            guard let rdDecoded = Riscv32.GPRNoX0.decode( rd ) else { return nil }
            self.rd = rdDecoded
            guard let rs1Decoded = Riscv32.SPMem.decode( rs1 ) else { return nil }
            self.rs1 = rs1Decoded
            super.init( opcode: .cLwsp, size: 2 )
        }
        
        public func encode() -> UInt32 {
            var encoded: UInt32 = 0x4002
            encoded |= ( imm.encode() & 0xc0 ) >> 4
            encoded |= ( imm.encode() & 0x20 ) << 7
            encoded |= ( imm.encode() & 0x1c ) << 2
            encoded |= ( rd.encode() & 0x1f ) << 7
            return encoded
        }
    }
    
    public class CSlli: Instruction {
        public let imm: uimmlog2xlennonzero
        public let rd: GPRNoX0
        
        public init( imm: uimmlog2xlennonzero, rd: GPRNoX0 ) {
            self.imm = imm
            self.rd = rd
            super.init( opcode: .cSlli, size: 2 )
        }
        
        public init?( imm: UInt32, rd: UInt32 ) {
            guard let immDecoded = Riscv32.uimmlog2xlennonzero.decode( imm ) else { return nil }
            self.imm = immDecoded
            guard let rdDecoded = Riscv32.GPRNoX0.decode( rd ) else { return nil }
            self.rd = rdDecoded
            super.init( opcode: .cSlli, size: 2 )
        }
        
        public func encode() -> UInt32 {
            var encoded: UInt32 = 0x2
            encoded |= ( imm.encode() & 0x20 ) << 7
            encoded |= ( imm.encode() & 0x1f ) << 2
            encoded |= ( rd.encode() & 0x1f ) << 7
            return encoded
        }
    }
    
    public class CSw: Instruction {
        public let imm: uimm7_lsb00
        public let rs1: GPRCMem
        public let rs2: GPRC
        
        public init( imm: uimm7_lsb00, rs1: GPRCMem, rs2: GPRC ) {
            self.imm = imm
            self.rs1 = rs1
            self.rs2 = rs2
            super.init( opcode: .cSw, size: 2 )
        }
        
        public init?( imm: UInt32, rs1: UInt32, rs2: UInt32 ) {
            guard let immDecoded = Riscv32.uimm7_lsb00.decode( imm ) else { return nil }
            self.imm = immDecoded
            guard let rs1Decoded = Riscv32.GPRCMem.decode( rs1 ) else { return nil }
            self.rs1 = rs1Decoded
            guard let rs2Decoded = Riscv32.GPRC.decode( rs2 ) else { return nil }
            self.rs2 = rs2Decoded
            super.init( opcode: .cSw, size: 2 )
        }
        
        public func encode() -> UInt32 {
            var encoded: UInt32 = 0xc000
            encoded |= ( imm.encode() & 0x40 ) >> 1
            encoded |= ( imm.encode() & 0x38 ) << 7
            encoded |= ( imm.encode() & 0x4 ) << 4
            encoded |= ( rs1.encode() & 0x7 ) << 7
            encoded |= ( rs2.encode() & 0x7 ) << 2
            return encoded
        }
    }
    
    public class CSwsp: Instruction {
        public let imm: uimm8_lsb00
        public let rs2: GPR
        public let rs1: SPMem
        
        public init( imm: uimm8_lsb00, rs2: GPR, rs1: SPMem ) {
            self.imm = imm
            self.rs2 = rs2
            self.rs1 = rs1
            super.init( opcode: .cSwsp, size: 2 )
        }
        
        public init?( imm: UInt32, rs2: UInt32, rs1: UInt32 ) {
            guard let immDecoded = Riscv32.uimm8_lsb00.decode( imm ) else { return nil }
            self.imm = immDecoded
            guard let rs2Decoded = Riscv32.GPR.decode( rs2 ) else { return nil }
            self.rs2 = rs2Decoded
            guard let rs1Decoded = Riscv32.SPMem.decode( rs1 ) else { return nil }
            self.rs1 = rs1Decoded
            super.init( opcode: .cSwsp, size: 2 )
        }
        
        public func encode() -> UInt32 {
            var encoded: UInt32 = 0xc002
            encoded |= ( imm.encode() & 0xc0 ) << 1
            encoded |= ( imm.encode() & 0x3c ) << 7
            encoded |= ( rs2.encode() & 0x1f ) << 2
            return encoded
        }
    }
    
    public class Addi: Instruction {
        public let imm12: simm12
        public let rd: GPR
        public let rs1: GPR
        
        public init( imm12: simm12, rd: GPR, rs1: GPR ) {
            self.imm12 = imm12
            self.rd = rd
            self.rs1 = rs1
            super.init( opcode: .addi, size: 4 )
        }
        
        public init?( imm12: UInt32, rd: UInt32, rs1: UInt32 ) {
            guard let imm12Decoded = Riscv32.simm12.decode( imm12 ) else { return nil }
            self.imm12 = imm12Decoded
            guard let rdDecoded = Riscv32.GPR.decode( rd ) else { return nil }
            self.rd = rdDecoded
            guard let rs1Decoded = Riscv32.GPR.decode( rs1 ) else { return nil }
            self.rs1 = rs1Decoded
            super.init( opcode: .addi, size: 4 )
        }
        
        public func encode() -> UInt32 {
            var encoded: UInt32 = 0x13
            encoded |= ( imm12.encode() & 0xfff ) << 20
            encoded |= ( rd.encode() & 0x1f ) << 7
            encoded |= ( rs1.encode() & 0x1f ) << 15
            return encoded
        }
    }
    
    public class Andi: Instruction {
        public let imm12: simm12
        public let rd: GPR
        public let rs1: GPR
        
        public init( imm12: simm12, rd: GPR, rs1: GPR ) {
            self.imm12 = imm12
            self.rd = rd
            self.rs1 = rs1
            super.init( opcode: .andi, size: 4 )
        }
        
        public init?( imm12: UInt32, rd: UInt32, rs1: UInt32 ) {
            guard let imm12Decoded = Riscv32.simm12.decode( imm12 ) else { return nil }
            self.imm12 = imm12Decoded
            guard let rdDecoded = Riscv32.GPR.decode( rd ) else { return nil }
            self.rd = rdDecoded
            guard let rs1Decoded = Riscv32.GPR.decode( rs1 ) else { return nil }
            self.rs1 = rs1Decoded
            super.init( opcode: .andi, size: 4 )
        }
        
        public func encode() -> UInt32 {
            var encoded: UInt32 = 0x7013
            encoded |= ( imm12.encode() & 0xfff ) << 20
            encoded |= ( rd.encode() & 0x1f ) << 7
            encoded |= ( rs1.encode() & 0x1f ) << 15
            return encoded
        }
    }
    
    public class Beq: Instruction {
        public let imm12: simm13_lsb0
        public let rs1: GPR
        public let rs2: GPR
        
        public init( imm12: simm13_lsb0, rs1: GPR, rs2: GPR ) {
            self.imm12 = imm12
            self.rs1 = rs1
            self.rs2 = rs2
            super.init( opcode: .beq, size: 4 )
        }
        
        public init?( imm12: UInt32, rs1: UInt32, rs2: UInt32 ) {
            guard let imm12Decoded = Riscv32.simm13_lsb0.decode( imm12 ) else { return nil }
            self.imm12 = imm12Decoded
            guard let rs1Decoded = Riscv32.GPR.decode( rs1 ) else { return nil }
            self.rs1 = rs1Decoded
            guard let rs2Decoded = Riscv32.GPR.decode( rs2 ) else { return nil }
            self.rs2 = rs2Decoded
            super.init( opcode: .beq, size: 4 )
        }
        
        public func encode() -> UInt32 {
            var encoded: UInt32 = 0x63
            encoded |= ( imm12.encode() & 0x800 ) << 20
            encoded |= ( imm12.encode() & 0x400 ) >> 3
            encoded |= ( imm12.encode() & 0x3f0 ) << 21
            encoded |= ( imm12.encode() & 0xf ) << 8
            encoded |= ( rs1.encode() & 0x1f ) << 15
            encoded |= ( rs2.encode() & 0x1f ) << 20
            return encoded
        }
    }
    
    public class Bge: Instruction {
        public let imm12: simm13_lsb0
        public let rs1: GPR
        public let rs2: GPR
        
        public init( imm12: simm13_lsb0, rs1: GPR, rs2: GPR ) {
            self.imm12 = imm12
            self.rs1 = rs1
            self.rs2 = rs2
            super.init( opcode: .bge, size: 4 )
        }
        
        public init?( imm12: UInt32, rs1: UInt32, rs2: UInt32 ) {
            guard let imm12Decoded = Riscv32.simm13_lsb0.decode( imm12 ) else { return nil }
            self.imm12 = imm12Decoded
            guard let rs1Decoded = Riscv32.GPR.decode( rs1 ) else { return nil }
            self.rs1 = rs1Decoded
            guard let rs2Decoded = Riscv32.GPR.decode( rs2 ) else { return nil }
            self.rs2 = rs2Decoded
            super.init( opcode: .bge, size: 4 )
        }
        
        public func encode() -> UInt32 {
            var encoded: UInt32 = 0x5063
            encoded |= ( imm12.encode() & 0x800 ) << 20
            encoded |= ( imm12.encode() & 0x400 ) >> 3
            encoded |= ( imm12.encode() & 0x3f0 ) << 21
            encoded |= ( imm12.encode() & 0xf ) << 8
            encoded |= ( rs1.encode() & 0x1f ) << 15
            encoded |= ( rs2.encode() & 0x1f ) << 20
            return encoded
        }
    }
    
    public class Bgeu: Instruction {
        public let imm12: simm13_lsb0
        public let rs1: GPR
        public let rs2: GPR
        
        public init( imm12: simm13_lsb0, rs1: GPR, rs2: GPR ) {
            self.imm12 = imm12
            self.rs1 = rs1
            self.rs2 = rs2
            super.init( opcode: .bgeu, size: 4 )
        }
        
        public init?( imm12: UInt32, rs1: UInt32, rs2: UInt32 ) {
            guard let imm12Decoded = Riscv32.simm13_lsb0.decode( imm12 ) else { return nil }
            self.imm12 = imm12Decoded
            guard let rs1Decoded = Riscv32.GPR.decode( rs1 ) else { return nil }
            self.rs1 = rs1Decoded
            guard let rs2Decoded = Riscv32.GPR.decode( rs2 ) else { return nil }
            self.rs2 = rs2Decoded
            super.init( opcode: .bgeu, size: 4 )
        }
        
        public func encode() -> UInt32 {
            var encoded: UInt32 = 0x7063
            encoded |= ( imm12.encode() & 0x800 ) << 20
            encoded |= ( imm12.encode() & 0x400 ) >> 3
            encoded |= ( imm12.encode() & 0x3f0 ) << 21
            encoded |= ( imm12.encode() & 0xf ) << 8
            encoded |= ( rs1.encode() & 0x1f ) << 15
            encoded |= ( rs2.encode() & 0x1f ) << 20
            return encoded
        }
    }
    
    public class Blt: Instruction {
        public let imm12: simm13_lsb0
        public let rs1: GPR
        public let rs2: GPR
        
        public init( imm12: simm13_lsb0, rs1: GPR, rs2: GPR ) {
            self.imm12 = imm12
            self.rs1 = rs1
            self.rs2 = rs2
            super.init( opcode: .blt, size: 4 )
        }
        
        public init?( imm12: UInt32, rs1: UInt32, rs2: UInt32 ) {
            guard let imm12Decoded = Riscv32.simm13_lsb0.decode( imm12 ) else { return nil }
            self.imm12 = imm12Decoded
            guard let rs1Decoded = Riscv32.GPR.decode( rs1 ) else { return nil }
            self.rs1 = rs1Decoded
            guard let rs2Decoded = Riscv32.GPR.decode( rs2 ) else { return nil }
            self.rs2 = rs2Decoded
            super.init( opcode: .blt, size: 4 )
        }
        
        public func encode() -> UInt32 {
            var encoded: UInt32 = 0x4063
            encoded |= ( imm12.encode() & 0x800 ) << 20
            encoded |= ( imm12.encode() & 0x400 ) >> 3
            encoded |= ( imm12.encode() & 0x3f0 ) << 21
            encoded |= ( imm12.encode() & 0xf ) << 8
            encoded |= ( rs1.encode() & 0x1f ) << 15
            encoded |= ( rs2.encode() & 0x1f ) << 20
            return encoded
        }
    }
    
    public class Bltu: Instruction {
        public let imm12: simm13_lsb0
        public let rs1: GPR
        public let rs2: GPR
        
        public init( imm12: simm13_lsb0, rs1: GPR, rs2: GPR ) {
            self.imm12 = imm12
            self.rs1 = rs1
            self.rs2 = rs2
            super.init( opcode: .bltu, size: 4 )
        }
        
        public init?( imm12: UInt32, rs1: UInt32, rs2: UInt32 ) {
            guard let imm12Decoded = Riscv32.simm13_lsb0.decode( imm12 ) else { return nil }
            self.imm12 = imm12Decoded
            guard let rs1Decoded = Riscv32.GPR.decode( rs1 ) else { return nil }
            self.rs1 = rs1Decoded
            guard let rs2Decoded = Riscv32.GPR.decode( rs2 ) else { return nil }
            self.rs2 = rs2Decoded
            super.init( opcode: .bltu, size: 4 )
        }
        
        public func encode() -> UInt32 {
            var encoded: UInt32 = 0x6063
            encoded |= ( imm12.encode() & 0x800 ) << 20
            encoded |= ( imm12.encode() & 0x400 ) >> 3
            encoded |= ( imm12.encode() & 0x3f0 ) << 21
            encoded |= ( imm12.encode() & 0xf ) << 8
            encoded |= ( rs1.encode() & 0x1f ) << 15
            encoded |= ( rs2.encode() & 0x1f ) << 20
            return encoded
        }
    }
    
    public class Bne: Instruction {
        public let imm12: simm13_lsb0
        public let rs1: GPR
        public let rs2: GPR
        
        public init( imm12: simm13_lsb0, rs1: GPR, rs2: GPR ) {
            self.imm12 = imm12
            self.rs1 = rs1
            self.rs2 = rs2
            super.init( opcode: .bne, size: 4 )
        }
        
        public init?( imm12: UInt32, rs1: UInt32, rs2: UInt32 ) {
            guard let imm12Decoded = Riscv32.simm13_lsb0.decode( imm12 ) else { return nil }
            self.imm12 = imm12Decoded
            guard let rs1Decoded = Riscv32.GPR.decode( rs1 ) else { return nil }
            self.rs1 = rs1Decoded
            guard let rs2Decoded = Riscv32.GPR.decode( rs2 ) else { return nil }
            self.rs2 = rs2Decoded
            super.init( opcode: .bne, size: 4 )
        }
        
        public func encode() -> UInt32 {
            var encoded: UInt32 = 0x1063
            encoded |= ( imm12.encode() & 0x800 ) << 20
            encoded |= ( imm12.encode() & 0x400 ) >> 3
            encoded |= ( imm12.encode() & 0x3f0 ) << 21
            encoded |= ( imm12.encode() & 0xf ) << 8
            encoded |= ( rs1.encode() & 0x1f ) << 15
            encoded |= ( rs2.encode() & 0x1f ) << 20
            return encoded
        }
    }
    
    public class Csrrc: Instruction {
        public let imm12: csr_sysreg
        public let rd: GPR
        public let rs1: GPR
        
        public init( imm12: csr_sysreg, rd: GPR, rs1: GPR ) {
            self.imm12 = imm12
            self.rd = rd
            self.rs1 = rs1
            super.init( opcode: .csrrc, size: 4 )
        }
        
        public init?( imm12: UInt32, rd: UInt32, rs1: UInt32 ) {
            guard let imm12Decoded = Riscv32.csr_sysreg.decode( imm12 ) else { return nil }
            self.imm12 = imm12Decoded
            guard let rdDecoded = Riscv32.GPR.decode( rd ) else { return nil }
            self.rd = rdDecoded
            guard let rs1Decoded = Riscv32.GPR.decode( rs1 ) else { return nil }
            self.rs1 = rs1Decoded
            super.init( opcode: .csrrc, size: 4 )
        }
        
        public func encode() -> UInt32 {
            var encoded: UInt32 = 0x3073
            encoded |= ( imm12.encode() & 0xfff ) << 20
            encoded |= ( rd.encode() & 0x1f ) << 7
            encoded |= ( rs1.encode() & 0x1f ) << 15
            return encoded
        }
    }
    
    public class Csrrci: Instruction {
        public let imm12: csr_sysreg
        public let rd: GPR
        public let rs1: uimm5
        
        public init( imm12: csr_sysreg, rd: GPR, rs1: uimm5 ) {
            self.imm12 = imm12
            self.rd = rd
            self.rs1 = rs1
            super.init( opcode: .csrrci, size: 4 )
        }
        
        public init?( imm12: UInt32, rd: UInt32, rs1: UInt32 ) {
            guard let imm12Decoded = Riscv32.csr_sysreg.decode( imm12 ) else { return nil }
            self.imm12 = imm12Decoded
            guard let rdDecoded = Riscv32.GPR.decode( rd ) else { return nil }
            self.rd = rdDecoded
            guard let rs1Decoded = Riscv32.uimm5.decode( rs1 ) else { return nil }
            self.rs1 = rs1Decoded
            super.init( opcode: .csrrci, size: 4 )
        }
        
        public func encode() -> UInt32 {
            var encoded: UInt32 = 0x7073
            encoded |= ( imm12.encode() & 0xfff ) << 20
            encoded |= ( rd.encode() & 0x1f ) << 7
            encoded |= ( rs1.encode() & 0x1f ) << 15
            return encoded
        }
    }
    
    public class Csrrs: Instruction {
        public let imm12: csr_sysreg
        public let rd: GPR
        public let rs1: GPR
        
        public init( imm12: csr_sysreg, rd: GPR, rs1: GPR ) {
            self.imm12 = imm12
            self.rd = rd
            self.rs1 = rs1
            super.init( opcode: .csrrs, size: 4 )
        }
        
        public init?( imm12: UInt32, rd: UInt32, rs1: UInt32 ) {
            guard let imm12Decoded = Riscv32.csr_sysreg.decode( imm12 ) else { return nil }
            self.imm12 = imm12Decoded
            guard let rdDecoded = Riscv32.GPR.decode( rd ) else { return nil }
            self.rd = rdDecoded
            guard let rs1Decoded = Riscv32.GPR.decode( rs1 ) else { return nil }
            self.rs1 = rs1Decoded
            super.init( opcode: .csrrs, size: 4 )
        }
        
        public func encode() -> UInt32 {
            var encoded: UInt32 = 0x2073
            encoded |= ( imm12.encode() & 0xfff ) << 20
            encoded |= ( rd.encode() & 0x1f ) << 7
            encoded |= ( rs1.encode() & 0x1f ) << 15
            return encoded
        }
    }
    
    public class Csrrsi: Instruction {
        public let imm12: csr_sysreg
        public let rd: GPR
        public let rs1: uimm5
        
        public init( imm12: csr_sysreg, rd: GPR, rs1: uimm5 ) {
            self.imm12 = imm12
            self.rd = rd
            self.rs1 = rs1
            super.init( opcode: .csrrsi, size: 4 )
        }
        
        public init?( imm12: UInt32, rd: UInt32, rs1: UInt32 ) {
            guard let imm12Decoded = Riscv32.csr_sysreg.decode( imm12 ) else { return nil }
            self.imm12 = imm12Decoded
            guard let rdDecoded = Riscv32.GPR.decode( rd ) else { return nil }
            self.rd = rdDecoded
            guard let rs1Decoded = Riscv32.uimm5.decode( rs1 ) else { return nil }
            self.rs1 = rs1Decoded
            super.init( opcode: .csrrsi, size: 4 )
        }
        
        public func encode() -> UInt32 {
            var encoded: UInt32 = 0x6073
            encoded |= ( imm12.encode() & 0xfff ) << 20
            encoded |= ( rd.encode() & 0x1f ) << 7
            encoded |= ( rs1.encode() & 0x1f ) << 15
            return encoded
        }
    }
    
    public class Csrrw: Instruction {
        public let imm12: csr_sysreg
        public let rd: GPR
        public let rs1: GPR
        
        public init( imm12: csr_sysreg, rd: GPR, rs1: GPR ) {
            self.imm12 = imm12
            self.rd = rd
            self.rs1 = rs1
            super.init( opcode: .csrrw, size: 4 )
        }
        
        public init?( imm12: UInt32, rd: UInt32, rs1: UInt32 ) {
            guard let imm12Decoded = Riscv32.csr_sysreg.decode( imm12 ) else { return nil }
            self.imm12 = imm12Decoded
            guard let rdDecoded = Riscv32.GPR.decode( rd ) else { return nil }
            self.rd = rdDecoded
            guard let rs1Decoded = Riscv32.GPR.decode( rs1 ) else { return nil }
            self.rs1 = rs1Decoded
            super.init( opcode: .csrrw, size: 4 )
        }
        
        public func encode() -> UInt32 {
            var encoded: UInt32 = 0x1073
            encoded |= ( imm12.encode() & 0xfff ) << 20
            encoded |= ( rd.encode() & 0x1f ) << 7
            encoded |= ( rs1.encode() & 0x1f ) << 15
            return encoded
        }
    }
    
    public class Csrrwi: Instruction {
        public let imm12: csr_sysreg
        public let rd: GPR
        public let rs1: uimm5
        
        public init( imm12: csr_sysreg, rd: GPR, rs1: uimm5 ) {
            self.imm12 = imm12
            self.rd = rd
            self.rs1 = rs1
            super.init( opcode: .csrrwi, size: 4 )
        }
        
        public init?( imm12: UInt32, rd: UInt32, rs1: UInt32 ) {
            guard let imm12Decoded = Riscv32.csr_sysreg.decode( imm12 ) else { return nil }
            self.imm12 = imm12Decoded
            guard let rdDecoded = Riscv32.GPR.decode( rd ) else { return nil }
            self.rd = rdDecoded
            guard let rs1Decoded = Riscv32.uimm5.decode( rs1 ) else { return nil }
            self.rs1 = rs1Decoded
            super.init( opcode: .csrrwi, size: 4 )
        }
        
        public func encode() -> UInt32 {
            var encoded: UInt32 = 0x5073
            encoded |= ( imm12.encode() & 0xfff ) << 20
            encoded |= ( rd.encode() & 0x1f ) << 7
            encoded |= ( rs1.encode() & 0x1f ) << 15
            return encoded
        }
    }
    
    public class Jalr: Instruction {
        public let imm12: simm12
        public let rd: GPR
        public let rs1: GPR
        
        public init( imm12: simm12, rd: GPR, rs1: GPR ) {
            self.imm12 = imm12
            self.rd = rd
            self.rs1 = rs1
            super.init( opcode: .jalr, size: 4 )
        }
        
        public init?( imm12: UInt32, rd: UInt32, rs1: UInt32 ) {
            guard let imm12Decoded = Riscv32.simm12.decode( imm12 ) else { return nil }
            self.imm12 = imm12Decoded
            guard let rdDecoded = Riscv32.GPR.decode( rd ) else { return nil }
            self.rd = rdDecoded
            guard let rs1Decoded = Riscv32.GPR.decode( rs1 ) else { return nil }
            self.rs1 = rs1Decoded
            super.init( opcode: .jalr, size: 4 )
        }
        
        public func encode() -> UInt32 {
            var encoded: UInt32 = 0x67
            encoded |= ( imm12.encode() & 0xfff ) << 20
            encoded |= ( rd.encode() & 0x1f ) << 7
            encoded |= ( rs1.encode() & 0x1f ) << 15
            return encoded
        }
    }
    
    public class Lb: Instruction {
        public let imm12: simm12
        public let rd: GPR
        public let rs1: GPRMem
        
        public init( imm12: simm12, rd: GPR, rs1: GPRMem ) {
            self.imm12 = imm12
            self.rd = rd
            self.rs1 = rs1
            super.init( opcode: .lb, size: 4 )
        }
        
        public init?( imm12: UInt32, rd: UInt32, rs1: UInt32 ) {
            guard let imm12Decoded = Riscv32.simm12.decode( imm12 ) else { return nil }
            self.imm12 = imm12Decoded
            guard let rdDecoded = Riscv32.GPR.decode( rd ) else { return nil }
            self.rd = rdDecoded
            guard let rs1Decoded = Riscv32.GPRMem.decode( rs1 ) else { return nil }
            self.rs1 = rs1Decoded
            super.init( opcode: .lb, size: 4 )
        }
        
        public func encode() -> UInt32 {
            var encoded: UInt32 = 0x3
            encoded |= ( imm12.encode() & 0xfff ) << 20
            encoded |= ( rd.encode() & 0x1f ) << 7
            encoded |= ( rs1.encode() & 0x1f ) << 15
            return encoded
        }
    }
    
    public class Lbu: Instruction {
        public let imm12: simm12
        public let rd: GPR
        public let rs1: GPRMem
        
        public init( imm12: simm12, rd: GPR, rs1: GPRMem ) {
            self.imm12 = imm12
            self.rd = rd
            self.rs1 = rs1
            super.init( opcode: .lbu, size: 4 )
        }
        
        public init?( imm12: UInt32, rd: UInt32, rs1: UInt32 ) {
            guard let imm12Decoded = Riscv32.simm12.decode( imm12 ) else { return nil }
            self.imm12 = imm12Decoded
            guard let rdDecoded = Riscv32.GPR.decode( rd ) else { return nil }
            self.rd = rdDecoded
            guard let rs1Decoded = Riscv32.GPRMem.decode( rs1 ) else { return nil }
            self.rs1 = rs1Decoded
            super.init( opcode: .lbu, size: 4 )
        }
        
        public func encode() -> UInt32 {
            var encoded: UInt32 = 0x4003
            encoded |= ( imm12.encode() & 0xfff ) << 20
            encoded |= ( rd.encode() & 0x1f ) << 7
            encoded |= ( rs1.encode() & 0x1f ) << 15
            return encoded
        }
    }
    
    public class Lh: Instruction {
        public let imm12: simm12
        public let rd: GPR
        public let rs1: GPRMem
        
        public init( imm12: simm12, rd: GPR, rs1: GPRMem ) {
            self.imm12 = imm12
            self.rd = rd
            self.rs1 = rs1
            super.init( opcode: .lh, size: 4 )
        }
        
        public init?( imm12: UInt32, rd: UInt32, rs1: UInt32 ) {
            guard let imm12Decoded = Riscv32.simm12.decode( imm12 ) else { return nil }
            self.imm12 = imm12Decoded
            guard let rdDecoded = Riscv32.GPR.decode( rd ) else { return nil }
            self.rd = rdDecoded
            guard let rs1Decoded = Riscv32.GPRMem.decode( rs1 ) else { return nil }
            self.rs1 = rs1Decoded
            super.init( opcode: .lh, size: 4 )
        }
        
        public func encode() -> UInt32 {
            var encoded: UInt32 = 0x1003
            encoded |= ( imm12.encode() & 0xfff ) << 20
            encoded |= ( rd.encode() & 0x1f ) << 7
            encoded |= ( rs1.encode() & 0x1f ) << 15
            return encoded
        }
    }
    
    public class Lhu: Instruction {
        public let imm12: simm12
        public let rd: GPR
        public let rs1: GPRMem
        
        public init( imm12: simm12, rd: GPR, rs1: GPRMem ) {
            self.imm12 = imm12
            self.rd = rd
            self.rs1 = rs1
            super.init( opcode: .lhu, size: 4 )
        }
        
        public init?( imm12: UInt32, rd: UInt32, rs1: UInt32 ) {
            guard let imm12Decoded = Riscv32.simm12.decode( imm12 ) else { return nil }
            self.imm12 = imm12Decoded
            guard let rdDecoded = Riscv32.GPR.decode( rd ) else { return nil }
            self.rd = rdDecoded
            guard let rs1Decoded = Riscv32.GPRMem.decode( rs1 ) else { return nil }
            self.rs1 = rs1Decoded
            super.init( opcode: .lhu, size: 4 )
        }
        
        public func encode() -> UInt32 {
            var encoded: UInt32 = 0x5003
            encoded |= ( imm12.encode() & 0xfff ) << 20
            encoded |= ( rd.encode() & 0x1f ) << 7
            encoded |= ( rs1.encode() & 0x1f ) << 15
            return encoded
        }
    }
    
    public class Lw: Instruction {
        public let imm12: simm12
        public let rd: GPR
        public let rs1: GPRMem
        
        public init( imm12: simm12, rd: GPR, rs1: GPRMem ) {
            self.imm12 = imm12
            self.rd = rd
            self.rs1 = rs1
            super.init( opcode: .lw, size: 4 )
        }
        
        public init?( imm12: UInt32, rd: UInt32, rs1: UInt32 ) {
            guard let imm12Decoded = Riscv32.simm12.decode( imm12 ) else { return nil }
            self.imm12 = imm12Decoded
            guard let rdDecoded = Riscv32.GPR.decode( rd ) else { return nil }
            self.rd = rdDecoded
            guard let rs1Decoded = Riscv32.GPRMem.decode( rs1 ) else { return nil }
            self.rs1 = rs1Decoded
            super.init( opcode: .lw, size: 4 )
        }
        
        public func encode() -> UInt32 {
            var encoded: UInt32 = 0x2003
            encoded |= ( imm12.encode() & 0xfff ) << 20
            encoded |= ( rd.encode() & 0x1f ) << 7
            encoded |= ( rs1.encode() & 0x1f ) << 15
            return encoded
        }
    }
    
    public class Ori: Instruction {
        public let imm12: simm12
        public let rd: GPR
        public let rs1: GPR
        
        public init( imm12: simm12, rd: GPR, rs1: GPR ) {
            self.imm12 = imm12
            self.rd = rd
            self.rs1 = rs1
            super.init( opcode: .ori, size: 4 )
        }
        
        public init?( imm12: UInt32, rd: UInt32, rs1: UInt32 ) {
            guard let imm12Decoded = Riscv32.simm12.decode( imm12 ) else { return nil }
            self.imm12 = imm12Decoded
            guard let rdDecoded = Riscv32.GPR.decode( rd ) else { return nil }
            self.rd = rdDecoded
            guard let rs1Decoded = Riscv32.GPR.decode( rs1 ) else { return nil }
            self.rs1 = rs1Decoded
            super.init( opcode: .ori, size: 4 )
        }
        
        public func encode() -> UInt32 {
            var encoded: UInt32 = 0x6013
            encoded |= ( imm12.encode() & 0xfff ) << 20
            encoded |= ( rd.encode() & 0x1f ) << 7
            encoded |= ( rs1.encode() & 0x1f ) << 15
            return encoded
        }
    }
    
    public class Sb: Instruction {
        public let imm12: simm12
        public let rs1: GPRMem
        public let rs2: GPR
        
        public init( imm12: simm12, rs1: GPRMem, rs2: GPR ) {
            self.imm12 = imm12
            self.rs1 = rs1
            self.rs2 = rs2
            super.init( opcode: .sb, size: 4 )
        }
        
        public init?( imm12: UInt32, rs1: UInt32, rs2: UInt32 ) {
            guard let imm12Decoded = Riscv32.simm12.decode( imm12 ) else { return nil }
            self.imm12 = imm12Decoded
            guard let rs1Decoded = Riscv32.GPRMem.decode( rs1 ) else { return nil }
            self.rs1 = rs1Decoded
            guard let rs2Decoded = Riscv32.GPR.decode( rs2 ) else { return nil }
            self.rs2 = rs2Decoded
            super.init( opcode: .sb, size: 4 )
        }
        
        public func encode() -> UInt32 {
            var encoded: UInt32 = 0x23
            encoded |= ( imm12.encode() & 0xfe0 ) << 20
            encoded |= ( imm12.encode() & 0x1f ) << 7
            encoded |= ( rs1.encode() & 0x1f ) << 15
            encoded |= ( rs2.encode() & 0x1f ) << 20
            return encoded
        }
    }
    
    public class Sh: Instruction {
        public let imm12: simm12
        public let rs1: GPRMem
        public let rs2: GPR
        
        public init( imm12: simm12, rs1: GPRMem, rs2: GPR ) {
            self.imm12 = imm12
            self.rs1 = rs1
            self.rs2 = rs2
            super.init( opcode: .sh, size: 4 )
        }
        
        public init?( imm12: UInt32, rs1: UInt32, rs2: UInt32 ) {
            guard let imm12Decoded = Riscv32.simm12.decode( imm12 ) else { return nil }
            self.imm12 = imm12Decoded
            guard let rs1Decoded = Riscv32.GPRMem.decode( rs1 ) else { return nil }
            self.rs1 = rs1Decoded
            guard let rs2Decoded = Riscv32.GPR.decode( rs2 ) else { return nil }
            self.rs2 = rs2Decoded
            super.init( opcode: .sh, size: 4 )
        }
        
        public func encode() -> UInt32 {
            var encoded: UInt32 = 0x1023
            encoded |= ( imm12.encode() & 0xfe0 ) << 20
            encoded |= ( imm12.encode() & 0x1f ) << 7
            encoded |= ( rs1.encode() & 0x1f ) << 15
            encoded |= ( rs2.encode() & 0x1f ) << 20
            return encoded
        }
    }
    
    public class Slti: Instruction {
        public let imm12: simm12
        public let rd: GPR
        public let rs1: GPR
        
        public init( imm12: simm12, rd: GPR, rs1: GPR ) {
            self.imm12 = imm12
            self.rd = rd
            self.rs1 = rs1
            super.init( opcode: .slti, size: 4 )
        }
        
        public init?( imm12: UInt32, rd: UInt32, rs1: UInt32 ) {
            guard let imm12Decoded = Riscv32.simm12.decode( imm12 ) else { return nil }
            self.imm12 = imm12Decoded
            guard let rdDecoded = Riscv32.GPR.decode( rd ) else { return nil }
            self.rd = rdDecoded
            guard let rs1Decoded = Riscv32.GPR.decode( rs1 ) else { return nil }
            self.rs1 = rs1Decoded
            super.init( opcode: .slti, size: 4 )
        }
        
        public func encode() -> UInt32 {
            var encoded: UInt32 = 0x2013
            encoded |= ( imm12.encode() & 0xfff ) << 20
            encoded |= ( rd.encode() & 0x1f ) << 7
            encoded |= ( rs1.encode() & 0x1f ) << 15
            return encoded
        }
    }
    
    public class Sltiu: Instruction {
        public let imm12: simm12
        public let rd: GPR
        public let rs1: GPR
        
        public init( imm12: simm12, rd: GPR, rs1: GPR ) {
            self.imm12 = imm12
            self.rd = rd
            self.rs1 = rs1
            super.init( opcode: .sltiu, size: 4 )
        }
        
        public init?( imm12: UInt32, rd: UInt32, rs1: UInt32 ) {
            guard let imm12Decoded = Riscv32.simm12.decode( imm12 ) else { return nil }
            self.imm12 = imm12Decoded
            guard let rdDecoded = Riscv32.GPR.decode( rd ) else { return nil }
            self.rd = rdDecoded
            guard let rs1Decoded = Riscv32.GPR.decode( rs1 ) else { return nil }
            self.rs1 = rs1Decoded
            super.init( opcode: .sltiu, size: 4 )
        }
        
        public func encode() -> UInt32 {
            var encoded: UInt32 = 0x3013
            encoded |= ( imm12.encode() & 0xfff ) << 20
            encoded |= ( rd.encode() & 0x1f ) << 7
            encoded |= ( rs1.encode() & 0x1f ) << 15
            return encoded
        }
    }
    
    public class Sw: Instruction {
        public let imm12: simm12
        public let rs1: GPRMem
        public let rs2: GPR
        
        public init( imm12: simm12, rs1: GPRMem, rs2: GPR ) {
            self.imm12 = imm12
            self.rs1 = rs1
            self.rs2 = rs2
            super.init( opcode: .sw, size: 4 )
        }
        
        public init?( imm12: UInt32, rs1: UInt32, rs2: UInt32 ) {
            guard let imm12Decoded = Riscv32.simm12.decode( imm12 ) else { return nil }
            self.imm12 = imm12Decoded
            guard let rs1Decoded = Riscv32.GPRMem.decode( rs1 ) else { return nil }
            self.rs1 = rs1Decoded
            guard let rs2Decoded = Riscv32.GPR.decode( rs2 ) else { return nil }
            self.rs2 = rs2Decoded
            super.init( opcode: .sw, size: 4 )
        }
        
        public func encode() -> UInt32 {
            var encoded: UInt32 = 0x2023
            encoded |= ( imm12.encode() & 0xfe0 ) << 20
            encoded |= ( imm12.encode() & 0x1f ) << 7
            encoded |= ( rs1.encode() & 0x1f ) << 15
            encoded |= ( rs2.encode() & 0x1f ) << 20
            return encoded
        }
    }
    
    public class Xori: Instruction {
        public let imm12: simm12
        public let rd: GPR
        public let rs1: GPR
        
        public init( imm12: simm12, rd: GPR, rs1: GPR ) {
            self.imm12 = imm12
            self.rd = rd
            self.rs1 = rs1
            super.init( opcode: .xori, size: 4 )
        }
        
        public init?( imm12: UInt32, rd: UInt32, rs1: UInt32 ) {
            guard let imm12Decoded = Riscv32.simm12.decode( imm12 ) else { return nil }
            self.imm12 = imm12Decoded
            guard let rdDecoded = Riscv32.GPR.decode( rd ) else { return nil }
            self.rd = rdDecoded
            guard let rs1Decoded = Riscv32.GPR.decode( rs1 ) else { return nil }
            self.rs1 = rs1Decoded
            super.init( opcode: .xori, size: 4 )
        }
        
        public func encode() -> UInt32 {
            var encoded: UInt32 = 0x4013
            encoded |= ( imm12.encode() & 0xfff ) << 20
            encoded |= ( rd.encode() & 0x1f ) << 7
            encoded |= ( rs1.encode() & 0x1f ) << 15
            return encoded
        }
    }
    
    public class Auipc: Instruction {
        public let imm20: uimm20_auipc
        public let rd: GPR
        
        public init( imm20: uimm20_auipc, rd: GPR ) {
            self.imm20 = imm20
            self.rd = rd
            super.init( opcode: .auipc, size: 4 )
        }
        
        public init?( imm20: UInt32, rd: UInt32 ) {
            guard let imm20Decoded = Riscv32.uimm20_auipc.decode( imm20 ) else { return nil }
            self.imm20 = imm20Decoded
            guard let rdDecoded = Riscv32.GPR.decode( rd ) else { return nil }
            self.rd = rdDecoded
            super.init( opcode: .auipc, size: 4 )
        }
        
        public func encode() -> UInt32 {
            var encoded: UInt32 = 0x17
            encoded |= ( imm20.encode() & 0xfffff ) << 12
            encoded |= ( rd.encode() & 0x1f ) << 7
            return encoded
        }
    }
    
    public class Jal: Instruction {
        public let imm20: simm21_lsb0_jal
        public let rd: GPR
        
        public init( imm20: simm21_lsb0_jal, rd: GPR ) {
            self.imm20 = imm20
            self.rd = rd
            super.init( opcode: .jal, size: 4 )
        }
        
        public init?( imm20: UInt32, rd: UInt32 ) {
            guard let imm20Decoded = Riscv32.simm21_lsb0_jal.decode( imm20 ) else { return nil }
            self.imm20 = imm20Decoded
            guard let rdDecoded = Riscv32.GPR.decode( rd ) else { return nil }
            self.rd = rdDecoded
            super.init( opcode: .jal, size: 4 )
        }
        
        public func encode() -> UInt32 {
            var encoded: UInt32 = 0x6f
            encoded |= ( imm20.encode() & 0x80000 ) << 12
            encoded |= ( imm20.encode() & 0x7f800 ) << 1
            encoded |= ( imm20.encode() & 0x400 ) << 10
            encoded |= ( imm20.encode() & 0x3ff ) << 21
            encoded |= ( rd.encode() & 0x1f ) << 7
            return encoded
        }
    }
    
    public class Lui: Instruction {
        public let imm20: uimm20_lui
        public let rd: GPR
        
        public init( imm20: uimm20_lui, rd: GPR ) {
            self.imm20 = imm20
            self.rd = rd
            super.init( opcode: .lui, size: 4 )
        }
        
        public init?( imm20: UInt32, rd: UInt32 ) {
            guard let imm20Decoded = Riscv32.uimm20_lui.decode( imm20 ) else { return nil }
            self.imm20 = imm20Decoded
            guard let rdDecoded = Riscv32.GPR.decode( rd ) else { return nil }
            self.rd = rdDecoded
            super.init( opcode: .lui, size: 4 )
        }
        
        public func encode() -> UInt32 {
            var encoded: UInt32 = 0x37
            encoded |= ( imm20.encode() & 0xfffff ) << 12
            encoded |= ( rd.encode() & 0x1f ) << 7
            return encoded
        }
    }
}
