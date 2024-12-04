//
//  Riscv32Decoder.swift
//  Riscv32
//

extension Riscv32 {
    
    public class Fetcher {
        public let memory: Memory
        public var address: UInt32
        
        public init( memory: Memory, address: UInt32) {
            self.memory = memory
            self.address = address
        }
        
        public func fetchInstruction() -> UInt32? {
            let address = self.address
            let bitPattern: UInt32
            guard let lowerBytes = memory.getUInt16( address: address ) else { return nil }
            self.address += 2
            if ( lowerBytes & 0x3 ) == 0x3 {
                guard let upperBytes = memory.getUInt16( address: self.address ) else { return nil }
                self.address += 2
                bitPattern = ( UInt32( upperBytes ) << 16 ) | UInt32( lowerBytes )
            } else {
                bitPattern = UInt32( lowerBytes )
            }
            return bitPattern
        }
    }
    
    public class Decoder {
        public var instruction: UInt32 = 0
        
        public var imm: UInt32 = 0
        public var imm12: UInt32 = 0
        public var imm20: UInt32 = 0
        public var offset: UInt32 = 0
        public var rd: UInt32 = 0
        public var rs1: UInt32 = 0
        public var rs2: UInt32 = 0
        public var shamt: UInt32 = 0
        
        public init() {
        }
        
        public func decode_imm12_csr_sysreg_f31t20() {
            var value: UInt32 = 0
            value |= ( self.instruction & 0xFFF00000 ) >> 20
            imm12 = value
        }
        
        public func decode_imm12_simm12_f31t20() {
            var value: UInt32 = 0
            value |= ( self.instruction & 0xFFF00000 ) >> 20
            imm12 = value
        }
        
        public func decode_imm12_simm12_f31t25f11t7() {
            var value: UInt32 = 0
            value |= ( self.instruction & 0xFE000000 ) >> 20
            value |= ( self.instruction & 0xF80 ) >> 7
            imm12 = value
        }
        
        public func decode_imm12_simm13_lsb0_f31t31f7t7f30t25f11t8() {
            var value: UInt32 = 0
            value |= ( self.instruction & 0x80000000 ) >> 20
            value |= ( self.instruction & 0x80 ) << 3
            value |= ( self.instruction & 0x7E000000 ) >> 21
            value |= ( self.instruction & 0xF00 ) >> 8
            imm12 = value
        }
        
        public func decode_imm20_simm21_lsb0_jal_f31t31f19t12f20t20f30t21() {
            var value: UInt32 = 0
            value |= ( self.instruction & 0x80000000 ) >> 12
            value |= ( self.instruction & 0xFF000 ) >> 1
            value |= ( self.instruction & 0x100000 ) >> 10
            value |= ( self.instruction & 0x7FE00000 ) >> 21
            imm20 = value
        }
        
        public func decode_imm20_uimm20_auipc_f31t12() {
            var value: UInt32 = 0
            value |= ( self.instruction & 0xFFFFF000 ) >> 12
            imm20 = value
        }
        
        public func decode_imm20_uimm20_lui_f31t12() {
            var value: UInt32 = 0
            value |= ( self.instruction & 0xFFFFF000 ) >> 12
            imm20 = value
        }
        
        public func decode_imm_c_lui_imm_f12t12f6t2() {
            var value: UInt32 = 0
            value |= ( self.instruction & 0x1000 ) >> 7
            value |= ( self.instruction & 0x7C ) >> 2
            imm = value
        }
        
        public func decode_imm_simm10_lsb0000nonzero_f12t12f4t3f5t5f2t2f6t6_0000() {
            var value: UInt32 = 0
            value |= ( self.instruction & 0x1000 ) >> 3
            value |= ( self.instruction & 0x18 ) << 4
            value |= ( self.instruction & 0x20 ) << 1
            value |= ( self.instruction & 0x4 ) << 3
            value |= ( self.instruction & 0x40 ) >> 2
            imm = value
        }
        
        public func decode_imm_simm6_f12t12f6t2() {
            var value: UInt32 = 0
            value |= ( self.instruction & 0x1000 ) >> 7
            value |= ( self.instruction & 0x7C ) >> 2
            imm = value
        }
        
        public func decode_imm_simm6nonzero_f12t12f6t2() {
            var value: UInt32 = 0
            value |= ( self.instruction & 0x1000 ) >> 7
            value |= ( self.instruction & 0x7C ) >> 2
            imm = value
        }
        
        public func decode_imm_simm9_lsb0_f12t12f6t5f2t2f11t10f4t3() {
            var value: UInt32 = 0
            value |= ( self.instruction & 0x1000 ) >> 5
            value |= ( self.instruction & 0x60 ) >> 0
            value |= ( self.instruction & 0x4 ) << 2
            value |= ( self.instruction & 0xC00 ) >> 8
            value |= ( self.instruction & 0x18 ) >> 3
            imm = value
        }
        
        public func decode_imm_uimm10_lsb00nonzero_f10t7f12t11f5t5f6t6_00() {
            var value: UInt32 = 0
            value |= ( self.instruction & 0x780 ) >> 1
            value |= ( self.instruction & 0x1800 ) >> 7
            value |= ( self.instruction & 0x20 ) >> 2
            value |= ( self.instruction & 0x40 ) >> 4
            imm = value
        }
        
        public func decode_imm_uimm7_lsb00_f5t5f12t10f6t6_00() {
            var value: UInt32 = 0
            value |= ( self.instruction & 0x20 ) << 1
            value |= ( self.instruction & 0x1C00 ) >> 7
            value |= ( self.instruction & 0x40 ) >> 4
            imm = value
        }
        
        public func decode_imm_uimm8_lsb00_f3t2f12t12f6t4_00() {
            var value: UInt32 = 0
            value |= ( self.instruction & 0xC ) << 4
            value |= ( self.instruction & 0x1000 ) >> 7
            value |= ( self.instruction & 0x70 ) >> 2
            imm = value
        }
        
        public func decode_imm_uimm8_lsb00_f8t7f12t9_00() {
            var value: UInt32 = 0
            value |= ( self.instruction & 0x180 ) >> 1
            value |= ( self.instruction & 0x1E00 ) >> 7
            imm = value
        }
        
        public func decode_imm_uimmlog2xlennonzero_f12t12f6t2() {
            var value: UInt32 = 0
            value |= ( self.instruction & 0x1000 ) >> 7
            value |= ( self.instruction & 0x7C ) >> 2
            imm = value
        }
        
        public func decode_offset_simm12_lsb0_f12t12f8t8f10t9f6t6f7t7f2t2f11t11f5t3() {
            var value: UInt32 = 0
            value |= ( self.instruction & 0x1000 ) >> 2
            value |= ( self.instruction & 0x100 ) << 1
            value |= ( self.instruction & 0x600 ) >> 2
            value |= ( self.instruction & 0x40 ) >> 0
            value |= ( self.instruction & 0x80 ) >> 2
            value |= ( self.instruction & 0x4 ) << 2
            value |= ( self.instruction & 0x800 ) >> 8
            value |= ( self.instruction & 0x38 ) >> 3
            offset = value
        }
        
        public func decode_rd_GPRC_f4t2() {
            var value: UInt32 = 0
            value |= ( self.instruction & 0x1C ) >> 2
            rd = value
        }
        
        public func decode_rd_GPRC_f9t7() {
            var value: UInt32 = 0
            value |= ( self.instruction & 0x380 ) >> 7
            rd = value
        }
        
        public func decode_rd_GPRNoX0X2_f11t7() {
            var value: UInt32 = 0
            value |= ( self.instruction & 0xF80 ) >> 7
            rd = value
        }
        
        public func decode_rd_GPRNoX0_f11t7() {
            var value: UInt32 = 0
            value |= ( self.instruction & 0xF80 ) >> 7
            rd = value
        }
        
        public func decode_rd_GPR_f11t7() {
            var value: UInt32 = 0
            value |= ( self.instruction & 0xF80 ) >> 7
            rd = value
        }
        
        public func decode_rd_SP__0() {
            var value: UInt32 = 0
            value = 2
            rd = value
        }
        
        public func decode_rs1_GPRCMem_f9t7() {
            var value: UInt32 = 0
            value |= ( self.instruction & 0x380 ) >> 7
            rs1 = value
        }
        
        public func decode_rs1_GPRC_f9t7() {
            var value: UInt32 = 0
            value |= ( self.instruction & 0x380 ) >> 7
            rs1 = value
        }
        
        public func decode_rs1_GPRMem_f19t15() {
            var value: UInt32 = 0
            value |= ( self.instruction & 0xF8000 ) >> 15
            rs1 = value
        }
        
        public func decode_rs1_GPRNoX0_f11t7() {
            var value: UInt32 = 0
            value |= ( self.instruction & 0xF80 ) >> 7
            rs1 = value
        }
        
        public func decode_rs1_GPR_f19t15() {
            var value: UInt32 = 0
            value |= ( self.instruction & 0xF8000 ) >> 15
            rs1 = value
        }
        
        public func decode_rs1_SPMem__0() {
            var value: UInt32 = 0
            value = 2
            rs1 = value
        }
        
        public func decode_rs1_SP__0() {
            var value: UInt32 = 0
            value = 2
            rs1 = value
        }
        
        public func decode_rs1_uimm5_f19t15() {
            var value: UInt32 = 0
            value |= ( self.instruction & 0xF8000 ) >> 15
            rs1 = value
        }
        
        public func decode_rs2_GPRC_f4t2() {
            var value: UInt32 = 0
            value |= ( self.instruction & 0x1C ) >> 2
            rs2 = value
        }
        
        public func decode_rs2_GPRNoX0_f6t2() {
            var value: UInt32 = 0
            value |= ( self.instruction & 0x7C ) >> 2
            rs2 = value
        }
        
        public func decode_rs2_GPR_f24t20() {
            var value: UInt32 = 0
            value |= ( self.instruction & 0x1F00000 ) >> 20
            rs2 = value
        }
        
        public func decode_rs2_GPR_f6t2() {
            var value: UInt32 = 0
            value |= ( self.instruction & 0x7C ) >> 2
            rs2 = value
        }
        
        public func decode_shamt_uimmlog2xlen_f25t20() {
            var value: UInt32 = 0
            value |= ( self.instruction & 0x3F00000 ) >> 20
            shamt = value
        }
        
        public func decodeInstruction( instruction: UInt32 ) -> Instruction? {
            self.instruction = instruction
            
            let bitPattern = self.instruction & 0x3
            switch bitPattern {
            case 0x0:
                let bitPattern = self.instruction & 0xE003
                switch bitPattern {
                case 0x0:
                    if self.instruction & 0xFFFF == 0x0 {
                        // "c.unimp	"
                        return CUnimp()
                    }
                    if self.instruction & 0xE003 == 0x0 {
                        // "c.addi4spn	$rd, $rs1, $imm"
                        decode_imm_uimm10_lsb00nonzero_f10t7f12t11f5t5f6t6_00()
                        decode_rd_GPRC_f4t2()
                        decode_rs1_SP__0()
                        return CAddi4spn( imm: imm, rd: rd, rs1: rs1 )
                    }
                case 0x4000:
                    if self.instruction & 0xE003 == 0x4000 {
                        // "c.lw	$rd, ${imm}(${rs1})"
                        decode_imm_uimm7_lsb00_f5t5f12t10f6t6_00()
                        decode_rd_GPRC_f4t2()
                        decode_rs1_GPRCMem_f9t7()
                        return CLw( imm: imm, rd: rd, rs1: rs1 )
                    }
                case 0xC000:
                    if self.instruction & 0xE003 == 0xC000 {
                        // "c.sw	$rs2, ${imm}(${rs1})"
                        decode_imm_uimm7_lsb00_f5t5f12t10f6t6_00()
                        decode_rs1_GPRCMem_f9t7()
                        decode_rs2_GPRC_f4t2()
                        return CSw( imm: imm, rs1: rs1, rs2: rs2 )
                    }
                default:
                    return nil
                }
            case 0x1:
                let bitPattern = self.instruction & 0xE003
                switch bitPattern {
                case 0x1:
                    if self.instruction & 0xFFFF == 0x1 {
                        // "c.nop	"
                        return CNop()
                    }
                    if self.instruction & 0xE003 == 0x1 {
                        // "c.addi	$rd, $imm"
                        decode_imm_simm6nonzero_f12t12f6t2()
                        decode_rd_GPRNoX0_f11t7()
                        return CAddi( imm: imm, rd: rd )
                    }
                case 0x4001:
                    if self.instruction & 0xE003 == 0x4001 {
                        // "c.li	$rd, $imm"
                        decode_imm_simm6_f12t12f6t2()
                        decode_rd_GPRNoX0_f11t7()
                        return CLi( imm: imm, rd: rd )
                    }
                case 0x6001:
                    if self.instruction & 0xEF83 == 0x6101 {
                        // "c.addi16sp	$rd, $imm"
                        decode_imm_simm10_lsb0000nonzero_f12t12f4t3f5t5f2t2f6t6_0000()
                        decode_rd_SP__0()
                        return CAddi16sp( imm: imm, rd: rd )
                    }
                    if self.instruction & 0xE003 == 0x6001 {
                        // "c.lui	$rd, $imm"
                        decode_imm_c_lui_imm_f12t12f6t2()
                        decode_rd_GPRNoX0X2_f11t7()
                        return CLui( imm: imm, rd: rd )
                    }
                case 0x8001:
                    let bitPattern = self.instruction & 0xEC03
                    switch bitPattern {
                    case 0x8001:
                        if self.instruction & 0xEC03 == 0x8001 {
                            // "c.srli	$rs1, $imm"
                            decode_imm_uimmlog2xlennonzero_f12t12f6t2()
                            decode_rs1_GPRC_f9t7()
                            return CSrli( imm: imm, rs1: rs1 )
                        }
                    case 0x8401:
                        if self.instruction & 0xEC03 == 0x8401 {
                            // "c.srai	$rs1, $imm"
                            decode_imm_uimmlog2xlennonzero_f12t12f6t2()
                            decode_rs1_GPRC_f9t7()
                            return CSrai( imm: imm, rs1: rs1 )
                        }
                    case 0x8801:
                        if self.instruction & 0xEC03 == 0x8801 {
                            // "c.andi	$rs1, $imm"
                            decode_imm_simm6_f12t12f6t2()
                            decode_rs1_GPRC_f9t7()
                            return CAndi( imm: imm, rs1: rs1 )
                        }
                    case 0x8C01:
                        let bitPattern = self.instruction & 0xFC63
                        switch bitPattern {
                        case 0x8C01:
                            if self.instruction & 0xFC63 == 0x8C01 {
                                // "c.sub	$rd, $rs2"
                                decode_rd_GPRC_f9t7()
                                decode_rs2_GPRC_f4t2()
                                return CSub( rd: rd, rs2: rs2 )
                            }
                        case 0x8C21:
                            if self.instruction & 0xFC63 == 0x8C21 {
                                // "c.xor	$rd, $rs2"
                                decode_rd_GPRC_f9t7()
                                decode_rs2_GPRC_f4t2()
                                return CXor( rd: rd, rs2: rs2 )
                            }
                        case 0x8C41:
                            if self.instruction & 0xFC63 == 0x8C41 {
                                // "c.or	$rd, $rs2"
                                decode_rd_GPRC_f9t7()
                                decode_rs2_GPRC_f4t2()
                                return COr( rd: rd, rs2: rs2 )
                            }
                        case 0x8C61:
                            if self.instruction & 0xFC63 == 0x8C61 {
                                // "c.and	$rd, $rs2"
                                decode_rd_GPRC_f9t7()
                                decode_rs2_GPRC_f4t2()
                                return CAnd( rd: rd, rs2: rs2 )
                            }
                        default:
                            return nil
                        }
                    default:
                        return nil
                    }
                case 0xA001:
                    if self.instruction & 0xE003 == 0xA001 {
                        // "c.j	$offset"
                        decode_offset_simm12_lsb0_f12t12f8t8f10t9f6t6f7t7f2t2f11t11f5t3()
                        return CJ( offset: offset )
                    }
                case 0xC001:
                    if self.instruction & 0xE003 == 0xC001 {
                        // "c.beqz	$rs1, $imm"
                        decode_imm_simm9_lsb0_f12t12f6t5f2t2f11t10f4t3()
                        decode_rs1_GPRC_f9t7()
                        return CBeqz( imm: imm, rs1: rs1 )
                    }
                case 0xE001:
                    if self.instruction & 0xE003 == 0xE001 {
                        // "c.bnez	$rs1, $imm"
                        decode_imm_simm9_lsb0_f12t12f6t5f2t2f11t10f4t3()
                        decode_rs1_GPRC_f9t7()
                        return CBnez( imm: imm, rs1: rs1 )
                    }
                default:
                    return nil
                }
            case 0x2:
                let bitPattern = self.instruction & 0xE003
                switch bitPattern {
                case 0x2:
                    if self.instruction & 0xE003 == 0x2 {
                        // "c.slli	$rd, $imm"
                        decode_imm_uimmlog2xlennonzero_f12t12f6t2()
                        decode_rd_GPRNoX0_f11t7()
                        return CSlli( imm: imm, rd: rd )
                    }
                case 0x4002:
                    if self.instruction & 0xE003 == 0x4002 {
                        // "c.lwsp	$rd, ${imm}(${rs1})"
                        decode_imm_uimm8_lsb00_f3t2f12t12f6t4_00()
                        decode_rd_GPRNoX0_f11t7()
                        decode_rs1_SPMem__0()
                        return CLwsp( imm: imm, rd: rd, rs1: rs1 )
                    }
                case 0x8002:
                    let bitPattern = self.instruction & 0xF003
                    switch bitPattern {
                    case 0x8002:
                        if self.instruction & 0xF07F == 0x8002 {
                            // "c.jr	$rs1"
                            decode_rs1_GPRNoX0_f11t7()
                            return CJr( rs1: rs1 )
                        }
                        if self.instruction & 0xF003 == 0x8002 {
                            // "c.mv	$rs1, $rs2"
                            decode_rs1_GPRNoX0_f11t7()
                            decode_rs2_GPRNoX0_f6t2()
                            return CMv( rs1: rs1, rs2: rs2 )
                        }
                    case 0x9002:
                        if self.instruction & 0xFFFF == 0x9002 {
                            // "c.ebreak	"
                            return CEbreak()
                        }
                        if self.instruction & 0xF07F == 0x9002 {
                            // "c.jalr	$rs1"
                            decode_rs1_GPRNoX0_f11t7()
                            return CJalr( rs1: rs1 )
                        }
                        if self.instruction & 0xF003 == 0x9002 {
                            // "c.add	$rs1, $rs2"
                            decode_rs1_GPRNoX0_f11t7()
                            decode_rs2_GPRNoX0_f6t2()
                            return CAdd( rs1: rs1, rs2: rs2 )
                        }
                    default:
                        return nil
                    }
                case 0xC002:
                    if self.instruction & 0xE003 == 0xC002 {
                        // "c.swsp	$rs2, ${imm}(${rs1})"
                        decode_imm_uimm8_lsb00_f8t7f12t9_00()
                        decode_rs2_GPR_f6t2()
                        decode_rs1_SPMem__0()
                        return CSwsp( imm: imm, rs2: rs2, rs1: rs1 )
                    }
                default:
                    return nil
                }
            case 0x3:
                let bitPattern = self.instruction & 0x7F
                switch bitPattern {
                case 0x3:
                    let bitPattern = self.instruction & 0x707F
                    switch bitPattern {
                    case 0x3:
                        if self.instruction & 0x707F == 0x3 {
                            // "lb	$rd, ${imm12}(${rs1})"
                            decode_imm12_simm12_f31t20()
                            decode_rd_GPR_f11t7()
                            decode_rs1_GPRMem_f19t15()
                            return Lb( imm12: imm12, rd: rd, rs1: rs1 )
                        }
                    case 0x1003:
                        if self.instruction & 0x707F == 0x1003 {
                            // "lh	$rd, ${imm12}(${rs1})"
                            decode_imm12_simm12_f31t20()
                            decode_rd_GPR_f11t7()
                            decode_rs1_GPRMem_f19t15()
                            return Lh( imm12: imm12, rd: rd, rs1: rs1 )
                        }
                    case 0x2003:
                        if self.instruction & 0x707F == 0x2003 {
                            // "lw	$rd, ${imm12}(${rs1})"
                            decode_imm12_simm12_f31t20()
                            decode_rd_GPR_f11t7()
                            decode_rs1_GPRMem_f19t15()
                            return Lw( imm12: imm12, rd: rd, rs1: rs1 )
                        }
                    case 0x4003:
                        if self.instruction & 0x707F == 0x4003 {
                            // "lbu	$rd, ${imm12}(${rs1})"
                            decode_imm12_simm12_f31t20()
                            decode_rd_GPR_f11t7()
                            decode_rs1_GPRMem_f19t15()
                            return Lbu( imm12: imm12, rd: rd, rs1: rs1 )
                        }
                    case 0x5003:
                        if self.instruction & 0x707F == 0x5003 {
                            // "lhu	$rd, ${imm12}(${rs1})"
                            decode_imm12_simm12_f31t20()
                            decode_rd_GPR_f11t7()
                            decode_rs1_GPRMem_f19t15()
                            return Lhu( imm12: imm12, rd: rd, rs1: rs1 )
                        }
                    default:
                        return nil
                    }
                case 0xF:
                    let bitPattern = self.instruction & 0xFFFFFFFF
                    switch bitPattern {
                    case 0x100F:
                        if self.instruction & 0xFFFFFFFF == 0x100F {
                            // "fence.i	"
                            return FenceI()
                        }
                    case 0x8330000F:
                        if self.instruction & 0xFFFFFFFF == 0x8330000F {
                            // "fence.tso	"
                            return FenceTso()
                        }
                    default:
                        return nil
                    }
                case 0x13:
                    let bitPattern = self.instruction & 0x707F
                    switch bitPattern {
                    case 0x13:
                        if self.instruction & 0x707F == 0x13 {
                            // "addi	$rd, $rs1, $imm12"
                            decode_imm12_simm12_f31t20()
                            decode_rd_GPR_f11t7()
                            decode_rs1_GPR_f19t15()
                            return Addi( imm12: imm12, rd: rd, rs1: rs1 )
                        }
                    case 0x1013:
                        if self.instruction & 0xFC00707F == 0x1013 {
                            // "slli	$rd, $rs1, $shamt"
                            decode_rd_GPR_f11t7()
                            decode_rs1_GPR_f19t15()
                            decode_shamt_uimmlog2xlen_f25t20()
                            return Slli( rd: rd, rs1: rs1, shamt: shamt )
                        }
                    case 0x2013:
                        if self.instruction & 0x707F == 0x2013 {
                            // "slti	$rd, $rs1, $imm12"
                            decode_imm12_simm12_f31t20()
                            decode_rd_GPR_f11t7()
                            decode_rs1_GPR_f19t15()
                            return Slti( imm12: imm12, rd: rd, rs1: rs1 )
                        }
                    case 0x3013:
                        if self.instruction & 0x707F == 0x3013 {
                            // "sltiu	$rd, $rs1, $imm12"
                            decode_imm12_simm12_f31t20()
                            decode_rd_GPR_f11t7()
                            decode_rs1_GPR_f19t15()
                            return Sltiu( imm12: imm12, rd: rd, rs1: rs1 )
                        }
                    case 0x4013:
                        if self.instruction & 0x707F == 0x4013 {
                            // "xori	$rd, $rs1, $imm12"
                            decode_imm12_simm12_f31t20()
                            decode_rd_GPR_f11t7()
                            decode_rs1_GPR_f19t15()
                            return Xori( imm12: imm12, rd: rd, rs1: rs1 )
                        }
                    case 0x5013:
                        let bitPattern = self.instruction & 0xFC00707F
                        switch bitPattern {
                        case 0x5013:
                            if self.instruction & 0xFC00707F == 0x5013 {
                                // "srli	$rd, $rs1, $shamt"
                                decode_rd_GPR_f11t7()
                                decode_rs1_GPR_f19t15()
                                decode_shamt_uimmlog2xlen_f25t20()
                                return Srli( rd: rd, rs1: rs1, shamt: shamt )
                            }
                        case 0x40005013:
                            if self.instruction & 0xFC00707F == 0x40005013 {
                                // "srai	$rd, $rs1, $shamt"
                                decode_rd_GPR_f11t7()
                                decode_rs1_GPR_f19t15()
                                decode_shamt_uimmlog2xlen_f25t20()
                                return Srai( rd: rd, rs1: rs1, shamt: shamt )
                            }
                        default:
                            return nil
                        }
                    case 0x6013:
                        if self.instruction & 0x707F == 0x6013 {
                            // "ori	$rd, $rs1, $imm12"
                            decode_imm12_simm12_f31t20()
                            decode_rd_GPR_f11t7()
                            decode_rs1_GPR_f19t15()
                            return Ori( imm12: imm12, rd: rd, rs1: rs1 )
                        }
                    case 0x7013:
                        if self.instruction & 0x707F == 0x7013 {
                            // "andi	$rd, $rs1, $imm12"
                            decode_imm12_simm12_f31t20()
                            decode_rd_GPR_f11t7()
                            decode_rs1_GPR_f19t15()
                            return Andi( imm12: imm12, rd: rd, rs1: rs1 )
                        }
                    default:
                        return nil
                    }
                case 0x17:
                    if self.instruction & 0x7F == 0x17 {
                        // "auipc	$rd, $imm20"
                        decode_imm20_uimm20_auipc_f31t12()
                        decode_rd_GPR_f11t7()
                        return Auipc( imm20: imm20, rd: rd )
                    }
                case 0x23:
                    let bitPattern = self.instruction & 0x707F
                    switch bitPattern {
                    case 0x23:
                        if self.instruction & 0x707F == 0x23 {
                            // "sb	$rs2, ${imm12}(${rs1})"
                            decode_imm12_simm12_f31t25f11t7()
                            decode_rs1_GPRMem_f19t15()
                            decode_rs2_GPR_f24t20()
                            return Sb( imm12: imm12, rs1: rs1, rs2: rs2 )
                        }
                    case 0x1023:
                        if self.instruction & 0x707F == 0x1023 {
                            // "sh	$rs2, ${imm12}(${rs1})"
                            decode_imm12_simm12_f31t25f11t7()
                            decode_rs1_GPRMem_f19t15()
                            decode_rs2_GPR_f24t20()
                            return Sh( imm12: imm12, rs1: rs1, rs2: rs2 )
                        }
                    case 0x2023:
                        if self.instruction & 0x707F == 0x2023 {
                            // "sw	$rs2, ${imm12}(${rs1})"
                            decode_imm12_simm12_f31t25f11t7()
                            decode_rs1_GPRMem_f19t15()
                            decode_rs2_GPR_f24t20()
                            return Sw( imm12: imm12, rs1: rs1, rs2: rs2 )
                        }
                    default:
                        return nil
                    }
                case 0x33:
                    let bitPattern = self.instruction & 0xFE00707F
                    switch bitPattern {
                    case 0x33:
                        if self.instruction & 0xFE00707F == 0x33 {
                            // "add	$rd, $rs1, $rs2"
                            decode_rd_GPR_f11t7()
                            decode_rs1_GPR_f19t15()
                            decode_rs2_GPR_f24t20()
                            return Add( rd: rd, rs1: rs1, rs2: rs2 )
                        }
                    case 0x1033:
                        if self.instruction & 0xFE00707F == 0x1033 {
                            // "sll	$rd, $rs1, $rs2"
                            decode_rd_GPR_f11t7()
                            decode_rs1_GPR_f19t15()
                            decode_rs2_GPR_f24t20()
                            return Sll( rd: rd, rs1: rs1, rs2: rs2 )
                        }
                    case 0x2033:
                        if self.instruction & 0xFE00707F == 0x2033 {
                            // "slt	$rd, $rs1, $rs2"
                            decode_rd_GPR_f11t7()
                            decode_rs1_GPR_f19t15()
                            decode_rs2_GPR_f24t20()
                            return Slt( rd: rd, rs1: rs1, rs2: rs2 )
                        }
                    case 0x3033:
                        if self.instruction & 0xFE00707F == 0x3033 {
                            // "sltu	$rd, $rs1, $rs2"
                            decode_rd_GPR_f11t7()
                            decode_rs1_GPR_f19t15()
                            decode_rs2_GPR_f24t20()
                            return Sltu( rd: rd, rs1: rs1, rs2: rs2 )
                        }
                    case 0x4033:
                        if self.instruction & 0xFE00707F == 0x4033 {
                            // "xor	$rd, $rs1, $rs2"
                            decode_rd_GPR_f11t7()
                            decode_rs1_GPR_f19t15()
                            decode_rs2_GPR_f24t20()
                            return Xor( rd: rd, rs1: rs1, rs2: rs2 )
                        }
                    case 0x5033:
                        if self.instruction & 0xFE00707F == 0x5033 {
                            // "srl	$rd, $rs1, $rs2"
                            decode_rd_GPR_f11t7()
                            decode_rs1_GPR_f19t15()
                            decode_rs2_GPR_f24t20()
                            return Srl( rd: rd, rs1: rs1, rs2: rs2 )
                        }
                    case 0x6033:
                        if self.instruction & 0xFE00707F == 0x6033 {
                            // "or	$rd, $rs1, $rs2"
                            decode_rd_GPR_f11t7()
                            decode_rs1_GPR_f19t15()
                            decode_rs2_GPR_f24t20()
                            return Or( rd: rd, rs1: rs1, rs2: rs2 )
                        }
                    case 0x7033:
                        if self.instruction & 0xFE00707F == 0x7033 {
                            // "and	$rd, $rs1, $rs2"
                            decode_rd_GPR_f11t7()
                            decode_rs1_GPR_f19t15()
                            decode_rs2_GPR_f24t20()
                            return And( rd: rd, rs1: rs1, rs2: rs2 )
                        }
                    case 0x2000033:
                        if self.instruction & 0xFE00707F == 0x2000033 {
                            // "mul	$rd, $rs1, $rs2"
                            decode_rd_GPR_f11t7()
                            decode_rs1_GPR_f19t15()
                            decode_rs2_GPR_f24t20()
                            return Mul( rd: rd, rs1: rs1, rs2: rs2 )
                        }
                    case 0x2001033:
                        if self.instruction & 0xFE00707F == 0x2001033 {
                            // "mulh	$rd, $rs1, $rs2"
                            decode_rd_GPR_f11t7()
                            decode_rs1_GPR_f19t15()
                            decode_rs2_GPR_f24t20()
                            return Mulh( rd: rd, rs1: rs1, rs2: rs2 )
                        }
                    case 0x2002033:
                        if self.instruction & 0xFE00707F == 0x2002033 {
                            // "mulhsu	$rd, $rs1, $rs2"
                            decode_rd_GPR_f11t7()
                            decode_rs1_GPR_f19t15()
                            decode_rs2_GPR_f24t20()
                            return Mulhsu( rd: rd, rs1: rs1, rs2: rs2 )
                        }
                    case 0x2003033:
                        if self.instruction & 0xFE00707F == 0x2003033 {
                            // "mulhu	$rd, $rs1, $rs2"
                            decode_rd_GPR_f11t7()
                            decode_rs1_GPR_f19t15()
                            decode_rs2_GPR_f24t20()
                            return Mulhu( rd: rd, rs1: rs1, rs2: rs2 )
                        }
                    case 0x2004033:
                        if self.instruction & 0xFE00707F == 0x2004033 {
                            // "div	$rd, $rs1, $rs2"
                            decode_rd_GPR_f11t7()
                            decode_rs1_GPR_f19t15()
                            decode_rs2_GPR_f24t20()
                            return Div( rd: rd, rs1: rs1, rs2: rs2 )
                        }
                    case 0x2005033:
                        if self.instruction & 0xFE00707F == 0x2005033 {
                            // "divu	$rd, $rs1, $rs2"
                            decode_rd_GPR_f11t7()
                            decode_rs1_GPR_f19t15()
                            decode_rs2_GPR_f24t20()
                            return Divu( rd: rd, rs1: rs1, rs2: rs2 )
                        }
                    case 0x2006033:
                        if self.instruction & 0xFE00707F == 0x2006033 {
                            // "rem	$rd, $rs1, $rs2"
                            decode_rd_GPR_f11t7()
                            decode_rs1_GPR_f19t15()
                            decode_rs2_GPR_f24t20()
                            return Rem( rd: rd, rs1: rs1, rs2: rs2 )
                        }
                    case 0x2007033:
                        if self.instruction & 0xFE00707F == 0x2007033 {
                            // "remu	$rd, $rs1, $rs2"
                            decode_rd_GPR_f11t7()
                            decode_rs1_GPR_f19t15()
                            decode_rs2_GPR_f24t20()
                            return Remu( rd: rd, rs1: rs1, rs2: rs2 )
                        }
                    case 0x40000033:
                        if self.instruction & 0xFE00707F == 0x40000033 {
                            // "sub	$rd, $rs1, $rs2"
                            decode_rd_GPR_f11t7()
                            decode_rs1_GPR_f19t15()
                            decode_rs2_GPR_f24t20()
                            return Sub( rd: rd, rs1: rs1, rs2: rs2 )
                        }
                    case 0x40005033:
                        if self.instruction & 0xFE00707F == 0x40005033 {
                            // "sra	$rd, $rs1, $rs2"
                            decode_rd_GPR_f11t7()
                            decode_rs1_GPR_f19t15()
                            decode_rs2_GPR_f24t20()
                            return Sra( rd: rd, rs1: rs1, rs2: rs2 )
                        }
                    default:
                        return nil
                    }
                case 0x37:
                    if self.instruction & 0x7F == 0x37 {
                        // "lui	$rd, $imm20"
                        decode_imm20_uimm20_lui_f31t12()
                        decode_rd_GPR_f11t7()
                        return Lui( imm20: imm20, rd: rd )
                    }
                case 0x63:
                    let bitPattern = self.instruction & 0x707F
                    switch bitPattern {
                    case 0x63:
                        if self.instruction & 0x707F == 0x63 {
                            // "beq	$rs1, $rs2, $imm12"
                            decode_imm12_simm13_lsb0_f31t31f7t7f30t25f11t8()
                            decode_rs1_GPR_f19t15()
                            decode_rs2_GPR_f24t20()
                            return Beq( imm12: imm12, rs1: rs1, rs2: rs2 )
                        }
                    case 0x1063:
                        if self.instruction & 0x707F == 0x1063 {
                            // "bne	$rs1, $rs2, $imm12"
                            decode_imm12_simm13_lsb0_f31t31f7t7f30t25f11t8()
                            decode_rs1_GPR_f19t15()
                            decode_rs2_GPR_f24t20()
                            return Bne( imm12: imm12, rs1: rs1, rs2: rs2 )
                        }
                    case 0x4063:
                        if self.instruction & 0x707F == 0x4063 {
                            // "blt	$rs1, $rs2, $imm12"
                            decode_imm12_simm13_lsb0_f31t31f7t7f30t25f11t8()
                            decode_rs1_GPR_f19t15()
                            decode_rs2_GPR_f24t20()
                            return Blt( imm12: imm12, rs1: rs1, rs2: rs2 )
                        }
                    case 0x5063:
                        if self.instruction & 0x707F == 0x5063 {
                            // "bge	$rs1, $rs2, $imm12"
                            decode_imm12_simm13_lsb0_f31t31f7t7f30t25f11t8()
                            decode_rs1_GPR_f19t15()
                            decode_rs2_GPR_f24t20()
                            return Bge( imm12: imm12, rs1: rs1, rs2: rs2 )
                        }
                    case 0x6063:
                        if self.instruction & 0x707F == 0x6063 {
                            // "bltu	$rs1, $rs2, $imm12"
                            decode_imm12_simm13_lsb0_f31t31f7t7f30t25f11t8()
                            decode_rs1_GPR_f19t15()
                            decode_rs2_GPR_f24t20()
                            return Bltu( imm12: imm12, rs1: rs1, rs2: rs2 )
                        }
                    case 0x7063:
                        if self.instruction & 0x707F == 0x7063 {
                            // "bgeu	$rs1, $rs2, $imm12"
                            decode_imm12_simm13_lsb0_f31t31f7t7f30t25f11t8()
                            decode_rs1_GPR_f19t15()
                            decode_rs2_GPR_f24t20()
                            return Bgeu( imm12: imm12, rs1: rs1, rs2: rs2 )
                        }
                    default:
                        return nil
                    }
                case 0x67:
                    if self.instruction & 0x707F == 0x67 {
                        // "jalr	$rd, ${imm12}(${rs1})"
                        decode_imm12_simm12_f31t20()
                        decode_rd_GPR_f11t7()
                        decode_rs1_GPR_f19t15()
                        return Jalr( imm12: imm12, rd: rd, rs1: rs1 )
                    }
                case 0x6F:
                    if self.instruction & 0x7F == 0x6F {
                        // "jal	$rd, $imm20"
                        decode_imm20_simm21_lsb0_jal_f31t31f19t12f20t20f30t21()
                        decode_rd_GPR_f11t7()
                        return Jal( imm20: imm20, rd: rd )
                    }
                case 0x73:
                    let bitPattern = self.instruction & 0x707F
                    switch bitPattern {
                    case 0x73:
                        let bitPattern = self.instruction & 0xFE007FFF
                        switch bitPattern {
                        case 0x73:
                            let bitPattern = self.instruction & 0xFFFFFFFF
                            switch bitPattern {
                            case 0x73:
                                if self.instruction & 0xFFFFFFFF == 0x73 {
                                    // "ecall	"
                                    return Ecall()
                                }
                            case 0x100073:
                                if self.instruction & 0xFFFFFFFF == 0x100073 {
                                    // "ebreak	"
                                    return Ebreak()
                                }
                            default:
                                return nil
                            }
                        case 0x10000073:
                            let bitPattern = self.instruction & 0xFFFFFFFF
                            switch bitPattern {
                            case 0x10200073:
                                if self.instruction & 0xFFFFFFFF == 0x10200073 {
                                    // "sret	"
                                    return Sret()
                                }
                            case 0x10500073:
                                if self.instruction & 0xFFFFFFFF == 0x10500073 {
                                    // "wfi	"
                                    return Wfi()
                                }
                            default:
                                return nil
                            }
                        case 0x12000073:
                            if self.instruction & 0xFE007FFF == 0x12000073 {
                                // "sfence.vma	$rs1, $rs2"
                                decode_rs1_GPR_f19t15()
                                decode_rs2_GPR_f24t20()
                                return SfenceVma( rs1: rs1, rs2: rs2 )
                            }
                        case 0x30000073:
                            if self.instruction & 0xFFFFFFFF == 0x30200073 {
                                // "mret	"
                                return Mret()
                            }
                        case 0x7A000073:
                            if self.instruction & 0xFFFFFFFF == 0x7B200073 {
                                // "dret	"
                                return Dret()
                            }
                        default:
                            return nil
                        }
                    case 0x1073:
                        if self.instruction & 0xFFFFFFFF == 0xC0001073 {
                            // "unimp	"
                            return Unimp()
                        }
                        if self.instruction & 0x707F == 0x1073 {
                            // "csrrw	$rd, $imm12, $rs1"
                            decode_imm12_csr_sysreg_f31t20()
                            decode_rd_GPR_f11t7()
                            decode_rs1_GPR_f19t15()
                            return Csrrw( imm12: imm12, rd: rd, rs1: rs1 )
                        }
                    case 0x2073:
                        if self.instruction & 0x707F == 0x2073 {
                            // "csrrs	$rd, $imm12, $rs1"
                            decode_imm12_csr_sysreg_f31t20()
                            decode_rd_GPR_f11t7()
                            decode_rs1_GPR_f19t15()
                            return Csrrs( imm12: imm12, rd: rd, rs1: rs1 )
                        }
                    case 0x3073:
                        if self.instruction & 0x707F == 0x3073 {
                            // "csrrc	$rd, $imm12, $rs1"
                            decode_imm12_csr_sysreg_f31t20()
                            decode_rd_GPR_f11t7()
                            decode_rs1_GPR_f19t15()
                            return Csrrc( imm12: imm12, rd: rd, rs1: rs1 )
                        }
                    case 0x5073:
                        if self.instruction & 0x707F == 0x5073 {
                            // "csrrwi	$rd, $imm12, $rs1"
                            decode_imm12_csr_sysreg_f31t20()
                            decode_rd_GPR_f11t7()
                            decode_rs1_uimm5_f19t15()
                            return Csrrwi( imm12: imm12, rd: rd, rs1: rs1 )
                        }
                    case 0x6073:
                        if self.instruction & 0x707F == 0x6073 {
                            // "csrrsi	$rd, $imm12, $rs1"
                            decode_imm12_csr_sysreg_f31t20()
                            decode_rd_GPR_f11t7()
                            decode_rs1_uimm5_f19t15()
                            return Csrrsi( imm12: imm12, rd: rd, rs1: rs1 )
                        }
                    case 0x7073:
                        if self.instruction & 0x707F == 0x7073 {
                            // "csrrci	$rd, $imm12, $rs1"
                            decode_imm12_csr_sysreg_f31t20()
                            decode_rd_GPR_f11t7()
                            decode_rs1_uimm5_f19t15()
                            return Csrrci( imm12: imm12, rd: rd, rs1: rs1 )
                        }
                    default:
                        return nil
                    }
                default:
                    return nil
                }
            default:
                return nil
            }
            
            return nil
        }
    }
}
