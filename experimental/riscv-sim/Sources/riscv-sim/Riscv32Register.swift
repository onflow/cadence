//
//  Riscv32Register.swift
//  Riscv32
//

extension Riscv32 {
    public enum GPR : Int {
        case x0 = 0
        case x1 = 1
        case x2 = 2
        case x3 = 3
        case x4 = 4
        case x5 = 5
        case x6 = 6
        case x7 = 7
        case x8 = 8
        case x9 = 9
        case x10 = 10
        case x11 = 11
        case x12 = 12
        case x13 = 13
        case x14 = 14
        case x15 = 15
        case x16 = 16
        case x17 = 17
        case x18 = 18
        case x19 = 19
        case x20 = 20
        case x21 = 21
        case x22 = 22
        case x23 = 23
        case x24 = 24
        case x25 = 25
        case x26 = 26
        case x27 = 27
        case x28 = 28
        case x29 = 29
        case x30 = 30
        case x31 = 31
        
        public func name() -> String {
            switch self {
            case .x0: return "x0"
            case .x1: return "x1"
            case .x2: return "x2"
            case .x3: return "x3"
            case .x4: return "x4"
            case .x5: return "x5"
            case .x6: return "x6"
            case .x7: return "x7"
            case .x8: return "x8"
            case .x9: return "x9"
            case .x10: return "x10"
            case .x11: return "x11"
            case .x12: return "x12"
            case .x13: return "x13"
            case .x14: return "x14"
            case .x15: return "x15"
            case .x16: return "x16"
            case .x17: return "x17"
            case .x18: return "x18"
            case .x19: return "x19"
            case .x20: return "x20"
            case .x21: return "x21"
            case .x22: return "x22"
            case .x23: return "x23"
            case .x24: return "x24"
            case .x25: return "x25"
            case .x26: return "x26"
            case .x27: return "x27"
            case .x28: return "x28"
            case .x29: return "x29"
            case .x30: return "x30"
            case .x31: return "x31"
            }
        }
        
        public func preferredName() -> String {
            switch self {
            case .x0: return "zero"
            case .x1: return "ra"
            case .x2: return "sp"
            case .x3: return "gp"
            case .x4: return "tp"
            case .x5: return "t0"
            case .x6: return "t1"
            case .x7: return "t2"
            case .x8: return "s0"
            case .x9: return "s1"
            case .x10: return "a0"
            case .x11: return "a1"
            case .x12: return "a2"
            case .x13: return "a3"
            case .x14: return "a4"
            case .x15: return "a5"
            case .x16: return "a6"
            case .x17: return "a7"
            case .x18: return "s2"
            case .x19: return "s3"
            case .x20: return "s4"
            case .x21: return "s5"
            case .x22: return "s6"
            case .x23: return "s7"
            case .x24: return "s8"
            case .x25: return "s9"
            case .x26: return "s10"
            case .x27: return "s11"
            case .x28: return "t3"
            case .x29: return "t4"
            case .x30: return "t5"
            case .x31: return "t6"
            }
        }
        
        public func encode() -> UInt32 {
            switch self {
            case .x0: return 0
            case .x1: return 1
            case .x2: return 2
            case .x3: return 3
            case .x4: return 4
            case .x5: return 5
            case .x6: return 6
            case .x7: return 7
            case .x8: return 8
            case .x9: return 9
            case .x10: return 10
            case .x11: return 11
            case .x12: return 12
            case .x13: return 13
            case .x14: return 14
            case .x15: return 15
            case .x16: return 16
            case .x17: return 17
            case .x18: return 18
            case .x19: return 19
            case .x20: return 20
            case .x21: return 21
            case .x22: return 22
            case .x23: return 23
            case .x24: return 24
            case .x25: return 25
            case .x26: return 26
            case .x27: return 27
            case .x28: return 28
            case .x29: return 29
            case .x30: return 30
            case .x31: return 31
            }
        }
        
        static public func decode( _ encoding: UInt32 ) -> GPR? {
            switch encoding {
            case 0: return .x0
            case 1: return .x1
            case 2: return .x2
            case 3: return .x3
            case 4: return .x4
            case 5: return .x5
            case 6: return .x6
            case 7: return .x7
            case 8: return .x8
            case 9: return .x9
            case 10: return .x10
            case 11: return .x11
            case 12: return .x12
            case 13: return .x13
            case 14: return .x14
            case 15: return .x15
            case 16: return .x16
            case 17: return .x17
            case 18: return .x18
            case 19: return .x19
            case 20: return .x20
            case 21: return .x21
            case 22: return .x22
            case 23: return .x23
            case 24: return .x24
            case 25: return .x25
            case 26: return .x26
            case 27: return .x27
            case 28: return .x28
            case 29: return .x29
            case 30: return .x30
            case 31: return .x31
            default: return nil
            }
        }
        
//        static public func matchGPR( buffer: Buffer ) -> GPR? {
//            return Riscv32.matchGPR( buffer: buffer )
//        }
        
        static public func fromString( stringLiteral value: String ) -> GPR? {
            switch value {
                // Names
            case "x0":
                return .x0
            case "x1":
                return .x1
            case "x2":
                return .x2
            case "x3":
                return .x3
            case "x4":
                return .x4
            case "x5":
                return .x5
            case "x6":
                return .x6
            case "x7":
                return .x7
            case "x8":
                return .x8
            case "x9":
                return .x9
            case "x10":
                return .x10
            case "x11":
                return .x11
            case "x12":
                return .x12
            case "x13":
                return .x13
            case "x14":
                return .x14
            case "x15":
                return .x15
            case "x16":
                return .x16
            case "x17":
                return .x17
            case "x18":
                return .x18
            case "x19":
                return .x19
            case "x20":
                return .x20
            case "x21":
                return .x21
            case "x22":
                return .x22
            case "x23":
                return .x23
            case "x24":
                return .x24
            case "x25":
                return .x25
            case "x26":
                return .x26
            case "x27":
                return .x27
            case "x28":
                return .x28
            case "x29":
                return .x29
            case "x30":
                return .x30
            case "x31":
                return .x31
                // Preferred Names
            case "zero":
                return .x0
            case "ra":
                return .x1
            case "sp":
                return .x2
            case "gp":
                return .x3
            case "tp":
                return .x4
            case "t0":
                return .x5
            case "t1":
                return .x6
            case "t2":
                return .x7
            case "s0":
                return .x8
            case "s1":
                return .x9
            case "a0":
                return .x10
            case "a1":
                return .x11
            case "a2":
                return .x12
            case "a3":
                return .x13
            case "a4":
                return .x14
            case "a5":
                return .x15
            case "a6":
                return .x16
            case "a7":
                return .x17
            case "s2":
                return .x18
            case "s3":
                return .x19
            case "s4":
                return .x20
            case "s5":
                return .x21
            case "s6":
                return .x22
            case "s7":
                return .x23
            case "s8":
                return .x24
            case "s9":
                return .x25
            case "s10":
                return .x26
            case "s11":
                return .x27
            case "t3":
                return .x28
            case "t4":
                return .x29
            case "t5":
                return .x30
            case "t6":
                return .x31
            default:
                return nil
            }
        }
    }
    
    public enum GPRC : Int {
        case x8 = 8
        case x9 = 9
        case x10 = 10
        case x11 = 11
        case x12 = 12
        case x13 = 13
        case x14 = 14
        case x15 = 15
        
        public func name() -> String {
            switch self {
            case .x8: return "x8"
            case .x9: return "x9"
            case .x10: return "x10"
            case .x11: return "x11"
            case .x12: return "x12"
            case .x13: return "x13"
            case .x14: return "x14"
            case .x15: return "x15"
            }
        }
        
        public func preferredName() -> String {
            switch self {
            case .x8: return "s0"
            case .x9: return "s1"
            case .x10: return "a0"
            case .x11: return "a1"
            case .x12: return "a2"
            case .x13: return "a3"
            case .x14: return "a4"
            case .x15: return "a5"
            }
        }
        
        public func encode() -> UInt32 {
            switch self {
            case .x8: return 0
            case .x9: return 1
            case .x10: return 2
            case .x11: return 3
            case .x12: return 4
            case .x13: return 5
            case .x14: return 6
            case .x15: return 7
            }
        }
        
        static public func decode( _ encoding: UInt32 ) -> GPRC? {
            switch encoding {
            case 0: return .x8
            case 1: return .x9
            case 2: return .x10
            case 3: return .x11
            case 4: return .x12
            case 5: return .x13
            case 6: return .x14
            case 7: return .x15
            default: return nil
            }
        }
        
//        static public func matchGPRC( buffer: Buffer ) -> GPRC? {
//            return Riscv32.matchGPRC( buffer: buffer )
//        }
        
        static public func fromString( stringLiteral value: String ) -> GPRC? {
            switch value {
                // Names
            case "x8":
                return .x8
            case "x9":
                return .x9
            case "x10":
                return .x10
            case "x11":
                return .x11
            case "x12":
                return .x12
            case "x13":
                return .x13
            case "x14":
                return .x14
            case "x15":
                return .x15
                // Preferred Names
            case "s0":
                return .x8
            case "s1":
                return .x9
            case "a0":
                return .x10
            case "a1":
                return .x11
            case "a2":
                return .x12
            case "a3":
                return .x13
            case "a4":
                return .x14
            case "a5":
                return .x15
            default:
                return nil
            }
        }
    }
    
    public enum GPRNoX0 : Int {
        case x1 = 1
        case x2 = 2
        case x3 = 3
        case x4 = 4
        case x5 = 5
        case x6 = 6
        case x7 = 7
        case x8 = 8
        case x9 = 9
        case x10 = 10
        case x11 = 11
        case x12 = 12
        case x13 = 13
        case x14 = 14
        case x15 = 15
        case x16 = 16
        case x17 = 17
        case x18 = 18
        case x19 = 19
        case x20 = 20
        case x21 = 21
        case x22 = 22
        case x23 = 23
        case x24 = 24
        case x25 = 25
        case x26 = 26
        case x27 = 27
        case x28 = 28
        case x29 = 29
        case x30 = 30
        case x31 = 31
        
        public func name() -> String {
            switch self {
            case .x1: return "x1"
            case .x2: return "x2"
            case .x3: return "x3"
            case .x4: return "x4"
            case .x5: return "x5"
            case .x6: return "x6"
            case .x7: return "x7"
            case .x8: return "x8"
            case .x9: return "x9"
            case .x10: return "x10"
            case .x11: return "x11"
            case .x12: return "x12"
            case .x13: return "x13"
            case .x14: return "x14"
            case .x15: return "x15"
            case .x16: return "x16"
            case .x17: return "x17"
            case .x18: return "x18"
            case .x19: return "x19"
            case .x20: return "x20"
            case .x21: return "x21"
            case .x22: return "x22"
            case .x23: return "x23"
            case .x24: return "x24"
            case .x25: return "x25"
            case .x26: return "x26"
            case .x27: return "x27"
            case .x28: return "x28"
            case .x29: return "x29"
            case .x30: return "x30"
            case .x31: return "x31"
            }
        }
        
        public func preferredName() -> String {
            switch self {
            case .x1: return "ra"
            case .x2: return "sp"
            case .x3: return "gp"
            case .x4: return "tp"
            case .x5: return "t0"
            case .x6: return "t1"
            case .x7: return "t2"
            case .x8: return "s0"
            case .x9: return "s1"
            case .x10: return "a0"
            case .x11: return "a1"
            case .x12: return "a2"
            case .x13: return "a3"
            case .x14: return "a4"
            case .x15: return "a5"
            case .x16: return "a6"
            case .x17: return "a7"
            case .x18: return "s2"
            case .x19: return "s3"
            case .x20: return "s4"
            case .x21: return "s5"
            case .x22: return "s6"
            case .x23: return "s7"
            case .x24: return "s8"
            case .x25: return "s9"
            case .x26: return "s10"
            case .x27: return "s11"
            case .x28: return "t3"
            case .x29: return "t4"
            case .x30: return "t5"
            case .x31: return "t6"
            }
        }
        
        public func encode() -> UInt32 {
            switch self {
            case .x1: return 1
            case .x2: return 2
            case .x3: return 3
            case .x4: return 4
            case .x5: return 5
            case .x6: return 6
            case .x7: return 7
            case .x8: return 8
            case .x9: return 9
            case .x10: return 10
            case .x11: return 11
            case .x12: return 12
            case .x13: return 13
            case .x14: return 14
            case .x15: return 15
            case .x16: return 16
            case .x17: return 17
            case .x18: return 18
            case .x19: return 19
            case .x20: return 20
            case .x21: return 21
            case .x22: return 22
            case .x23: return 23
            case .x24: return 24
            case .x25: return 25
            case .x26: return 26
            case .x27: return 27
            case .x28: return 28
            case .x29: return 29
            case .x30: return 30
            case .x31: return 31
            }
        }
        
        static public func decode( _ encoding: UInt32 ) -> GPRNoX0? {
            switch encoding {
            case 1: return .x1
            case 2: return .x2
            case 3: return .x3
            case 4: return .x4
            case 5: return .x5
            case 6: return .x6
            case 7: return .x7
            case 8: return .x8
            case 9: return .x9
            case 10: return .x10
            case 11: return .x11
            case 12: return .x12
            case 13: return .x13
            case 14: return .x14
            case 15: return .x15
            case 16: return .x16
            case 17: return .x17
            case 18: return .x18
            case 19: return .x19
            case 20: return .x20
            case 21: return .x21
            case 22: return .x22
            case 23: return .x23
            case 24: return .x24
            case 25: return .x25
            case 26: return .x26
            case 27: return .x27
            case 28: return .x28
            case 29: return .x29
            case 30: return .x30
            case 31: return .x31
            default: return nil
            }
        }
        
//        static public func matchGPRNoX0( buffer: Buffer ) -> GPRNoX0? {
//            return Riscv32.matchGPRNoX0( buffer: buffer )
//        }
        
        static public func fromString( stringLiteral value: String ) -> GPRNoX0? {
            switch value {
                // Names
            case "x1":
                return .x1
            case "x2":
                return .x2
            case "x3":
                return .x3
            case "x4":
                return .x4
            case "x5":
                return .x5
            case "x6":
                return .x6
            case "x7":
                return .x7
            case "x8":
                return .x8
            case "x9":
                return .x9
            case "x10":
                return .x10
            case "x11":
                return .x11
            case "x12":
                return .x12
            case "x13":
                return .x13
            case "x14":
                return .x14
            case "x15":
                return .x15
            case "x16":
                return .x16
            case "x17":
                return .x17
            case "x18":
                return .x18
            case "x19":
                return .x19
            case "x20":
                return .x20
            case "x21":
                return .x21
            case "x22":
                return .x22
            case "x23":
                return .x23
            case "x24":
                return .x24
            case "x25":
                return .x25
            case "x26":
                return .x26
            case "x27":
                return .x27
            case "x28":
                return .x28
            case "x29":
                return .x29
            case "x30":
                return .x30
            case "x31":
                return .x31
                // Preferred Names
            case "ra":
                return .x1
            case "sp":
                return .x2
            case "gp":
                return .x3
            case "tp":
                return .x4
            case "t0":
                return .x5
            case "t1":
                return .x6
            case "t2":
                return .x7
            case "s0":
                return .x8
            case "s1":
                return .x9
            case "a0":
                return .x10
            case "a1":
                return .x11
            case "a2":
                return .x12
            case "a3":
                return .x13
            case "a4":
                return .x14
            case "a5":
                return .x15
            case "a6":
                return .x16
            case "a7":
                return .x17
            case "s2":
                return .x18
            case "s3":
                return .x19
            case "s4":
                return .x20
            case "s5":
                return .x21
            case "s6":
                return .x22
            case "s7":
                return .x23
            case "s8":
                return .x24
            case "s9":
                return .x25
            case "s10":
                return .x26
            case "s11":
                return .x27
            case "t3":
                return .x28
            case "t4":
                return .x29
            case "t5":
                return .x30
            case "t6":
                return .x31
            default:
                return nil
            }
        }
    }
    
    public enum GPRNoX0X2 : Int {
        case x1 = 1
        case x3 = 3
        case x4 = 4
        case x5 = 5
        case x6 = 6
        case x7 = 7
        case x8 = 8
        case x9 = 9
        case x10 = 10
        case x11 = 11
        case x12 = 12
        case x13 = 13
        case x14 = 14
        case x15 = 15
        case x16 = 16
        case x17 = 17
        case x18 = 18
        case x19 = 19
        case x20 = 20
        case x21 = 21
        case x22 = 22
        case x23 = 23
        case x24 = 24
        case x25 = 25
        case x26 = 26
        case x27 = 27
        case x28 = 28
        case x29 = 29
        case x30 = 30
        case x31 = 31
        
        public func name() -> String {
            switch self {
            case .x1: return "x1"
            case .x3: return "x3"
            case .x4: return "x4"
            case .x5: return "x5"
            case .x6: return "x6"
            case .x7: return "x7"
            case .x8: return "x8"
            case .x9: return "x9"
            case .x10: return "x10"
            case .x11: return "x11"
            case .x12: return "x12"
            case .x13: return "x13"
            case .x14: return "x14"
            case .x15: return "x15"
            case .x16: return "x16"
            case .x17: return "x17"
            case .x18: return "x18"
            case .x19: return "x19"
            case .x20: return "x20"
            case .x21: return "x21"
            case .x22: return "x22"
            case .x23: return "x23"
            case .x24: return "x24"
            case .x25: return "x25"
            case .x26: return "x26"
            case .x27: return "x27"
            case .x28: return "x28"
            case .x29: return "x29"
            case .x30: return "x30"
            case .x31: return "x31"
            }
        }
        
        public func preferredName() -> String {
            switch self {
            case .x1: return "ra"
            case .x3: return "gp"
            case .x4: return "tp"
            case .x5: return "t0"
            case .x6: return "t1"
            case .x7: return "t2"
            case .x8: return "s0"
            case .x9: return "s1"
            case .x10: return "a0"
            case .x11: return "a1"
            case .x12: return "a2"
            case .x13: return "a3"
            case .x14: return "a4"
            case .x15: return "a5"
            case .x16: return "a6"
            case .x17: return "a7"
            case .x18: return "s2"
            case .x19: return "s3"
            case .x20: return "s4"
            case .x21: return "s5"
            case .x22: return "s6"
            case .x23: return "s7"
            case .x24: return "s8"
            case .x25: return "s9"
            case .x26: return "s10"
            case .x27: return "s11"
            case .x28: return "t3"
            case .x29: return "t4"
            case .x30: return "t5"
            case .x31: return "t6"
            }
        }
        
        public func encode() -> UInt32 {
            switch self {
            case .x1: return 1
            case .x3: return 3
            case .x4: return 4
            case .x5: return 5
            case .x6: return 6
            case .x7: return 7
            case .x8: return 8
            case .x9: return 9
            case .x10: return 10
            case .x11: return 11
            case .x12: return 12
            case .x13: return 13
            case .x14: return 14
            case .x15: return 15
            case .x16: return 16
            case .x17: return 17
            case .x18: return 18
            case .x19: return 19
            case .x20: return 20
            case .x21: return 21
            case .x22: return 22
            case .x23: return 23
            case .x24: return 24
            case .x25: return 25
            case .x26: return 26
            case .x27: return 27
            case .x28: return 28
            case .x29: return 29
            case .x30: return 30
            case .x31: return 31
            }
        }
        
        static public func decode( _ encoding: UInt32 ) -> GPRNoX0X2? {
            switch encoding {
            case 1: return .x1
            case 3: return .x3
            case 4: return .x4
            case 5: return .x5
            case 6: return .x6
            case 7: return .x7
            case 8: return .x8
            case 9: return .x9
            case 10: return .x10
            case 11: return .x11
            case 12: return .x12
            case 13: return .x13
            case 14: return .x14
            case 15: return .x15
            case 16: return .x16
            case 17: return .x17
            case 18: return .x18
            case 19: return .x19
            case 20: return .x20
            case 21: return .x21
            case 22: return .x22
            case 23: return .x23
            case 24: return .x24
            case 25: return .x25
            case 26: return .x26
            case 27: return .x27
            case 28: return .x28
            case 29: return .x29
            case 30: return .x30
            case 31: return .x31
            default: return nil
            }
        }
        
//        static public func matchGPRNoX0X2( buffer: Buffer ) -> GPRNoX0X2? {
//            return Riscv32.matchGPRNoX0X2( buffer: buffer )
//        }
        
        static public func fromString( stringLiteral value: String ) -> GPRNoX0X2? {
            switch value {
                // Names
            case "x1":
                return .x1
            case "x3":
                return .x3
            case "x4":
                return .x4
            case "x5":
                return .x5
            case "x6":
                return .x6
            case "x7":
                return .x7
            case "x8":
                return .x8
            case "x9":
                return .x9
            case "x10":
                return .x10
            case "x11":
                return .x11
            case "x12":
                return .x12
            case "x13":
                return .x13
            case "x14":
                return .x14
            case "x15":
                return .x15
            case "x16":
                return .x16
            case "x17":
                return .x17
            case "x18":
                return .x18
            case "x19":
                return .x19
            case "x20":
                return .x20
            case "x21":
                return .x21
            case "x22":
                return .x22
            case "x23":
                return .x23
            case "x24":
                return .x24
            case "x25":
                return .x25
            case "x26":
                return .x26
            case "x27":
                return .x27
            case "x28":
                return .x28
            case "x29":
                return .x29
            case "x30":
                return .x30
            case "x31":
                return .x31
                // Preferred Names
            case "ra":
                return .x1
            case "gp":
                return .x3
            case "tp":
                return .x4
            case "t0":
                return .x5
            case "t1":
                return .x6
            case "t2":
                return .x7
            case "s0":
                return .x8
            case "s1":
                return .x9
            case "a0":
                return .x10
            case "a1":
                return .x11
            case "a2":
                return .x12
            case "a3":
                return .x13
            case "a4":
                return .x14
            case "a5":
                return .x15
            case "a6":
                return .x16
            case "a7":
                return .x17
            case "s2":
                return .x18
            case "s3":
                return .x19
            case "s4":
                return .x20
            case "s5":
                return .x21
            case "s6":
                return .x22
            case "s7":
                return .x23
            case "s8":
                return .x24
            case "s9":
                return .x25
            case "s10":
                return .x26
            case "s11":
                return .x27
            case "t3":
                return .x28
            case "t4":
                return .x29
            case "t5":
                return .x30
            case "t6":
                return .x31
            default:
                return nil
            }
        }
    }
    
    public enum SP : Int {
        case x2 = 2
        
        public func name() -> String {
            switch self {
            case .x2: return "x2"
            }
        }
        
        public func preferredName() -> String {
            switch self {
            case .x2: return "sp"
            }
        }
        
        public func encode() -> UInt32 {
            switch self {
            case .x2: return 2
            }
        }
        
        static public func decode( _ encoding: UInt32 ) -> SP? {
            switch encoding {
            case 2: return .x2
            default: return nil
            }
        }
        
//        static public func matchSP( buffer: Buffer ) -> SP? {
//            return Riscv32.matchSP( buffer: buffer )
//        }
        
        static public func fromString( stringLiteral value: String ) -> SP? {
            switch value {
                // Names
            case "x2":
                return .x2
                // Preferred Names
            case "sp":
                return .x2
            default:
                return nil
            }
        }
    }
    
    public typealias GPRCMem = GPRC
    public typealias GPRMem = GPR
    public typealias SPMem = SP
}
