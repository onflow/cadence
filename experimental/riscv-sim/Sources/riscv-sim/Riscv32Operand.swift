//
//  Riscv32Operand.swift
//  Riscv32
//

extension Riscv32 {
    
    public struct c_lui_imm {
        public let value: Int32
        
        public init?( bitPattern: UInt32 ) {
            guard bitPattern & 0xFFFFFFC0 == 0 else { return nil }
             self.value = Int32( bitPattern )
        }
        
        public init?( value: Int32 ) {
            guard ( value >= minIntN( bitWidth: 6 ) )
                    || ( value <= maxIntN( bitWidth: 6 ) ) else { return nil }
            self.value = value
        }
        
        public func encode() -> UInt32 {
            return encodeInt( value, bitWidth: 6 )!
        }
        
        static public func decode( _ encoding: UInt32 ) -> c_lui_imm? {
            return c_lui_imm( bitPattern: encoding )
        }
    }
    
    public struct simm10_lsb0000nonzero {
        public let value: Int32
        
        public init?( bitPattern: UInt32 ) {
            guard bitPattern & 0x1 == 0 else { return nil }
            guard let value = decodeInt( bitPattern, bitWidth: 10 ) else { return nil }
            self.value = value
        }
        
        public init?( value: Int32 ) {
            guard value & 0x1 == 0 else { return nil }
            guard ( value >= minIntN( bitWidth: 10 ) )
                    || ( value <= maxIntN( bitWidth: 10 ) ) else { return nil }
            self.value = value
        }
        
        public func encode() -> UInt32 {
            return encodeInt( value, bitWidth: 10 )!
        }
        
        static public func decode( _ encoding: UInt32 ) -> simm10_lsb0000nonzero? {
            return simm10_lsb0000nonzero( bitPattern: encoding )
        }
    }
    
    public struct simm12 {
        public let value: Int32
        
        public init?( bitPattern: UInt32 ) {
            guard let value = decodeInt( bitPattern, bitWidth: 12 ) else { return nil }
            self.value = value
        }
        
        public init?( value: Int32 ) {
            guard ( value >= minIntN( bitWidth: 12 ) )
                    || ( value <= maxIntN( bitWidth: 12 ) ) else { return nil }
            self.value = value
        }
        
        public func encode() -> UInt32 {
            return encodeInt( value, bitWidth: 12 )!
        }
        
        static public func decode( _ encoding: UInt32 ) -> simm12? {
            return simm12( bitPattern: encoding )
        }
    }
    
    public struct simm12_lsb0 {
        public let value: Int32
        
        public init?( bitPattern: UInt32 ) {
            guard bitPattern & 0xFFFFF800 == 0 else { return nil }
            guard let value = decodeInt( bitPattern, bitWidth: 11, shiftCount: 1 ) else { return nil }
            self.value = value
        }
        
        public init?( value: Int32 ) {
            guard value & 0x1 == 0 else { return nil }
            guard ( value >= minIntN( bitWidth: 12 ) )
                    || ( value <= maxIntN( bitWidth: 12 ) ) else { return nil }
            self.value = value
        }
        
        public func encode() -> UInt32 {
            return encodeInt( value, bitWidth: 11, shiftCount: 1 )!
        }
        
        static public func decode( _ encoding: UInt32 ) -> simm12_lsb0? {
            return simm12_lsb0( bitPattern: encoding )
        }
    }
    
    public struct simm13_lsb0 {
        public let value: Int32
        
        public init?( bitPattern: UInt32 ) {
            guard bitPattern & 0xFFFFF000 == 0 else { return nil }
            guard let value = decodeInt( bitPattern, bitWidth: 12, shiftCount: 1 ) else { return nil }
            self.value = value
        }
        
        public init?( value: Int32 ) {
            guard value & 0x1 == 0 else { return nil }
            guard ( value >= minIntN( bitWidth: 13 ) )
                    || ( value <= maxIntN( bitWidth: 13 ) ) else { return nil }
            self.value = value
        }
        
        public func encode() -> UInt32 {
            return encodeInt( value, bitWidth: 12, shiftCount: 1 )!
        }
        
        static public func decode( _ encoding: UInt32 ) -> simm13_lsb0? {
            return simm13_lsb0( bitPattern: encoding )
        }
    }
    
    public struct simm21_lsb0_jal {
        public let value: Int32
        
        public init?( bitPattern: UInt32 ) {
            guard bitPattern & 0x1 == 0 else { return nil }
            guard let value = decodeInt( bitPattern, bitWidth: 21 ) else { return nil }
            self.value = value
        }
        
        public init?( value: Int32 ) {
            guard value & 0x1 == 0 else { return nil }
            guard ( value >= minIntN( bitWidth: 21 ) )
                    || ( value <= maxIntN( bitWidth: 21 ) ) else { return nil }
            self.value = value
        }
        
        public func encode() -> UInt32 {
            return encodeInt( value, bitWidth: 21 )!
        }
        
        static public func decode( _ encoding: UInt32 ) -> simm21_lsb0_jal? {
            return simm21_lsb0_jal( bitPattern: encoding )
        }
    }
    
    public struct simm6 {
        public let value: Int32
        
        public init?( bitPattern: UInt32 ) {
            guard let value = decodeInt( bitPattern, bitWidth: 6 ) else { return nil }
            self.value = value
        }
        
        public init?( value: Int32 ) {
            guard ( value >= minIntN( bitWidth: 6 ) )
                    || ( value <= maxIntN( bitWidth: 6 ) ) else { return nil }
            self.value = value
        }
        
        public func encode() -> UInt32 {
            return encodeInt( value, bitWidth: 6 )!
        }
        
        static public func decode( _ encoding: UInt32 ) -> simm6? {
            return simm6( bitPattern: encoding )
        }
    }
    
    public struct simm6nonzero {
        public let value: Int32
        
        public init?( bitPattern: UInt32 ) {
            guard bitPattern != 0 else { return nil }
            guard let value = decodeInt( bitPattern, bitWidth: 6 ) else { return nil }
            self.value = value
        }
        
        public init?( value: Int32 ) {
            guard value != 0 else { return nil }
            guard ( value >= minIntN( bitWidth: 6 ) )
                    || ( value <= maxIntN( bitWidth: 6 ) ) else { return nil }
            self.value = value
        }
        
        public func encode() -> UInt32 {
            return encodeInt( value, bitWidth: 6 )!
        }
        
        static public func decode( _ encoding: UInt32 ) -> simm6nonzero? {
            return simm6nonzero( bitPattern: encoding )
        }
    }
    
    public struct simm9_lsb0 {
        public let value: Int32
        
        public init?( bitPattern: UInt32 ) {
            guard bitPattern & 0x1 == 0 else { return nil }
            guard let value = decodeInt( bitPattern, bitWidth: 9 ) else { return nil }
            self.value = value
        }
        
        public init?( value: Int32 ) {
            guard value & 0x1 == 0 else { return nil }
            guard ( value >= minIntN( bitWidth: 9 ) )
                    || ( value <= maxIntN( bitWidth: 9 ) ) else { return nil }
            self.value = value
        }
        
        public func encode() -> UInt32 {
            return encodeInt( value, bitWidth: 9 )!
        }
        
        static public func decode( _ encoding: UInt32 ) -> simm9_lsb0? {
            return simm9_lsb0( bitPattern: encoding )
        }
    }
    
    public struct uimm10_lsb00nonzero {
        public let value: UInt32
        
        public init?( bitPattern: UInt32 ) {
            guard bitPattern & 0x3 == 0 else { return nil }
            guard let value = decodeUInt( bitPattern, bitWidth: 10 ) else { return nil }
            self.value = value
        }
        
        public init?( value: UInt32 ) {
            guard value & 0x3 == 0 else { return nil }
            guard isUInt( value,  bitWidth: 10 ) else { return nil }
            self.value = value
        }
        
        public func encode() -> UInt32 {
            return encodeUInt( value, bitWidth: 10 )!
        }
        
        static public func decode( _ encoding: UInt32 ) -> uimm10_lsb00nonzero? {
            return uimm10_lsb00nonzero( bitPattern: encoding )
        }
    }
    
    public struct uimm20_auipc {
        public let value: UInt32
        
        public init?( bitPattern: UInt32 ) {
            guard bitPattern & 0xFFF == 0 else { return nil }
            self.value = bitPattern
        }
        
        public init?( value: UInt32 ) {
            guard value & 0xFFF == 0 else { return nil }
            self.value = value
        }
        
        public func encode() -> UInt32 {
            return value
        }
        
        static public func decode( _ encoding: UInt32 ) -> uimm20_auipc? {
            return uimm20_auipc( bitPattern: encoding )
        }
    }
    
    public struct uimm20_lui {
        public let value: UInt32
        
        public init?( bitPattern: UInt32 ) {
            guard let value = decodeUInt( bitPattern, bitWidth: 20 ) else { return nil }
            self.value = value
        }
        
        public init?( value: UInt32 ) {
            guard isUInt( value,  bitWidth: 20 ) else { return nil }
            self.value = value
        }
        
        public func encode() -> UInt32 {
            return encodeUInt( value, bitWidth: 20 )!
        }
        
        static public func decode( _ encoding: UInt32 ) -> uimm20_lui? {
            return uimm20_lui( bitPattern: encoding )
        }
    }
    
    public struct uimm5 {
        public let value: UInt32
        
        public init?( bitPattern: UInt32 ) {
            guard let value = decodeUInt( bitPattern, bitWidth: 5 ) else { return nil }
            self.value = value
        }
        
        public init?( value: UInt32 ) {
            guard isUInt( value,  bitWidth: 5 ) else { return nil }
            self.value = value
        }
        
        public func encode() -> UInt32 {
            return encodeUInt( value, bitWidth: 5 )!
        }
        
        static public func decode( _ encoding: UInt32 ) -> uimm5? {
            return uimm5( bitPattern: encoding )
        }
    }
    
    public struct uimm7_lsb00 {
        public let value: UInt32
        
        public init?( bitPattern: UInt32 ) {
            guard bitPattern & 0x3 == 0 else { return nil }
            guard let value = decodeUInt( bitPattern, bitWidth: 7 ) else { return nil }
            self.value = value
        }
        
        public init?( value: UInt32 ) {
            guard value & 0x3 == 0 else { return nil }
            guard isUInt( value,  bitWidth: 7 ) else { return nil }
            self.value = value
        }
        
        public func encode() -> UInt32 {
            return encodeUInt( value, bitWidth: 7 )!
        }
        
        static public func decode( _ encoding: UInt32 ) -> uimm7_lsb00? {
            return uimm7_lsb00( bitPattern: encoding )
        }
    }
    
    public struct uimm8_lsb00 {
        public let value: UInt32
        
        public init?( bitPattern: UInt32 ) {
            guard bitPattern & 0x3 == 0 else { return nil }
            guard let value = decodeUInt( bitPattern, bitWidth: 8 ) else { return nil }
            self.value = value
        }
        
        public init?( value: UInt32 ) {
            guard value & 0x3 == 0 else { return nil }
            guard isUInt( value,  bitWidth: 8 ) else { return nil }
            self.value = value
        }
        
        public func encode() -> UInt32 {
            return encodeUInt( value, bitWidth: 8 )!
        }
        
        static public func decode( _ encoding: UInt32 ) -> uimm8_lsb00? {
            return uimm8_lsb00( bitPattern: encoding )
        }
    }
    
    public struct uimmlog2xlen {
        public let value: UInt32
        
        public init?( bitPattern: UInt32 ) {
            guard let value = decodeUInt( bitPattern, bitWidth: 5 ) else { return nil }
            self.value = value
        }
        
        public init?( value: UInt32 ) {
            guard isUInt( value,  bitWidth: 5 ) else { return nil }
            self.value = value
        }
        
        public func encode() -> UInt32 {
            return encodeUInt( value, bitWidth: 5 )!
        }
        
        static public func decode( _ encoding: UInt32 ) -> uimmlog2xlen? {
            return uimmlog2xlen( bitPattern: encoding )
        }
    }
    
    public struct uimmlog2xlennonzero {
        public let value: UInt32
        
        public init?( bitPattern: UInt32 ) {
            guard bitPattern != 0 else { return nil }
            guard let value = decodeUInt( bitPattern, bitWidth: 5 ) else { return nil }
            self.value = value
        }
        
        public init?( value: UInt32 ) {
            guard value != 0 else { return nil }
            guard isUInt( value,  bitWidth: 5 ) else { return nil }
            self.value = value
        }
        
        public func encode() -> UInt32 {
            return encodeUInt( value, bitWidth: 5 )!
        }
        
        static public func decode( _ encoding: UInt32 ) -> uimmlog2xlennonzero? {
            return uimmlog2xlennonzero( bitPattern: encoding )
        }
    }
}
