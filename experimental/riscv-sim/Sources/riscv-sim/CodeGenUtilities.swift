//
//  CodeGenUtilities.swift
//

public func getCodeGenUtilitiesVersion() -> ( major: Int, minor: Int, string: String ) {
    return ( major: 0, minor: 0, string: "0.0")
}

/// Checks if an unsigned integer fits into the given bitWidth.
public func isUInt( _ value: UInt64, bitWidth: UInt8 ) -> Bool {
    assert( bitWidth > 0 )
    if bitWidth >= 64 {
        return true
    } else {
        return value < ( UInt64(1) << bitWidth )
    }
}

/// Checks if an unsigned integer fits into the given bitWidth.
public func isUInt( _ value: UInt32, bitWidth: UInt8 ) -> Bool {
    assert( bitWidth > 0 )
    if bitWidth >= 32 {
        return true
    } else {
        return value < ( UInt32(1) << bitWidth )
    }
}

/// Checks if a signed integer fits into the given bitWidth.
public func isInt( _ value: UInt64, bitWidth: UInt8 ) -> Bool {
    assert( bitWidth > 0 )
    if bitWidth >= 64 {
        return true
    } else {
        return ( -( Int64( 1 ) << ( bitWidth - 1 ) ) <= value && value < ( Int64( 1 ) << ( bitWidth - 1 ) ) )
    }
}

/// Checks if a signed integer fits into the given bitWidth.
public func isInt( _ value: UInt32, bitWidth: UInt8 ) -> Bool {
    assert( bitWidth > 0 )
    if bitWidth >= 32 {
        return true
    } else {
        return ( -( Int32( 1 ) << ( bitWidth - 1 ) ) <= value && value < ( Int32( 1 ) << ( bitWidth - 1 ) ) )
    }
}

/// Checks if a unsigned integer is a bitWidth number shifted left by shiftCount.
public func isShiftedUInt( _ value: UInt64, bitWidth: UInt8, shiftCount: UInt8 ) -> Bool {
    assert( bitWidth > 0 )
    assert( bitWidth + shiftCount <= 64 )
    return isUInt( value, bitWidth: bitWidth + shiftCount ) && ( value % ( UInt64(1) << shiftCount ) == 0 )
}

/// Checks if a unsigned integer is a bitWidth number shifted left by shiftCount.
public func isShiftedUInt( _ value: UInt32, bitWidth: UInt8, shiftCount: UInt8 ) -> Bool {
    assert( bitWidth > 0 )
    assert( bitWidth + shiftCount <= 32 )
    return isUInt( value, bitWidth: bitWidth + shiftCount ) && ( value % ( UInt32(1) << shiftCount ) == 0 )
}

/// Checks if a ssigned integer is a bitWidth number shifted left by shiftCount.
public func isShiftedInt( _ value: UInt64, bitWidth: UInt8, shiftCount: UInt8 ) -> Bool {
    assert( bitWidth > 0 )
    assert( bitWidth + shiftCount <= 64 )
    return isInt( value, bitWidth: bitWidth + shiftCount ) && ( value % ( UInt64(1) << shiftCount ) == 0 )
}

/// Checks if a signed integer is a bitWidth number shifted left by shiftCount.
public func isShiftedInt( _ value: UInt32, bitWidth: UInt8, shiftCount: UInt8 ) -> Bool {
    assert( bitWidth > 0 )
    assert( bitWidth + shiftCount <= 32 )
    return isUInt( value, bitWidth: bitWidth + shiftCount ) && ( value % ( UInt32(1) << shiftCount ) == 0 )
}

/// Gets the maximum value for a bitWidth unsigned integer.
public func maxUIntN( bitWidth: UInt8 ) -> UInt64 {
    assert( bitWidth > 0 && bitWidth <= 64 )
    return UInt64.max >> ( 64 - bitWidth )
}

/// Gets the minimum value for a bitWidth signed integer.
public func minIntN( bitWidth: UInt8 ) -> Int64 {
    assert( bitWidth > 0 && bitWidth <= 64 )
    let bitPattern = UInt64(1) + ~( UInt64(1) << ( bitWidth - 1 ) )
    return Int64( bitPattern: bitPattern )
}

/// Gets the maximum value for a bitWidth signed integer.
public func maxIntN( bitWidth: UInt8 ) -> Int64 {
    assert( bitWidth > 0 && bitWidth <= 64 )
    let bitPattern = ( UInt64(1) << ( bitWidth - 1 ) ) - 1
    return Int64( bitPattern: bitPattern )
}

/// Checks if an signed integer fits into the given (dynamic) bit width.
public func isIntN( _ value: UInt64, bitWidth: UInt8 ) -> Bool {
    return bitWidth >= 64 || ( minIntN( bitWidth: bitWidth ) <= value && value <= maxIntN( bitWidth: bitWidth ) )
}

/// Return true if the argument is a non-empty sequence of ones starting at the
/// least significant bit with the remainder zero (32 bit version).
/// Ex. isMask_32(0x0000FFFFU) == true.
public func isMask_32( _ value: UInt32 ) -> Bool {
    return value != 0 && ( ( value + 1 ) & value ) == 0
}

/// Return true if the argument is a non-empty sequence of ones starting at the
/// least significant bit with the remainder zero (64 bit version).
public func isMask_64( _ value: UInt64 ) -> Bool {
    return value != 0 && ( ( value + 1 ) & value ) == 0
}

/// Return true if the argument contains a non-empty sequence of ones with the
/// remainder zero (32 bit version.) Ex. isShiftedMask_32(0x0000FF00U) == true.
public func isShiftedMask_32( _ value: UInt32 ) -> Bool {
    return value != 0 && isMask_32( ( value - 1 ) | value )
}

/// Return true if the argument contains a non-empty sequence of ones with the
/// remainder zero (64 bit version.)
public func isShiftedMask_64( _ value: UInt64 ) -> Bool {
    return value != 0 && isMask_64( ( value - 1 ) | value )
}

/// Return true if the argument is a power of two > 0.
/// Ex. isPowerOf2_32(0x00100000U) == true (32 bit edition.)
public func isPowerOf2_32( _ value: UInt32 ) -> Bool {
    return value != 0 && ( ( value & ( value - 1 ) ) == 0 )
}

/// Return true if the argument is a power of two > 0 (64 bit edition.)
public func isPowerOf2_64( _ value: UInt64 ) -> Bool {
    return value != 0 && ( ( value & ( value - 1 ) ) == 0 )
}

/// Count number of 0's from the most significant bit to the least
///   stopping at the first 1.
public func countLeadingZeros( _ value: UInt64 ) -> UInt8 {
    var currentValue = value
    var zeroBits: UInt64 = 0
    var shift: UInt64 = 64 >> 1
    while shift != 0 {
        let tmp = currentValue >> shift
        if tmp != 0 {
            currentValue = tmp
        } else {
            zeroBits |= shift
        }
        
        shift >>= 1
    }
    return UInt8( zeroBits )
}

/// Count number of 0's from the most significant bit to the least
///   stopping at the first 1.
public func countLeadingZeros( _ value: UInt32 ) -> UInt8 {
    var currentValue = value
    var zeroBits: UInt32 = 0
    var shift: UInt32 = 32 >> 1
    while shift != 0 {
        let tmp = currentValue >> shift
        if tmp != 0 {
            currentValue = tmp
        } else {
            zeroBits |= shift
        }
        
        shift >>= 1
    }
    return UInt8( zeroBits )
}

/// Count number of 0's from the most significant bit to the least
///   stopping at the first 1.
public func countLeadingZeros( _ value: UInt16 ) -> UInt8 {
    var currentValue = value
    var zeroBits: UInt16 = 0
    var shift: UInt16 = 16 >> 1
    while shift != 0 {
        let tmp = currentValue >> shift
        if tmp != 0 {
            currentValue = tmp
        } else {
            zeroBits |= shift
        }
        
        shift >>= 1
    }
    return UInt8( zeroBits )
}

/// Count the number of ones from the most significant bit to the first
/// zero bit.
public func countLeadingOnes( _ value: UInt64 ) -> UInt8 {
    return countLeadingZeros( ~value )
}

/// Count the number of ones from the most significant bit to the first
/// zero bit.
public func countLeadingOnes( _ value: UInt32 ) -> UInt8 {
    return countLeadingZeros( ~value )
}

/// Count the number of ones from the most significant bit to the first
/// zero bit.
public func countLeadingOnes( _ value: UInt16 ) -> UInt8 {
    return countLeadingZeros( ~value )
}

/// Count the number of set bits in a value.
/// Ex. countPopulation(0xF000F000) = 8
/// Returns 0 if the word is zero.
public func countPopulation( _ value: UInt64 ) -> UInt8 {
    var v = value;
    v = v - ( ( v >> 1 ) & 0x5555555555555555 )
    v = ( v & 0x3333333333333333 ) + ( ( v >> 2 ) & 0x3333333333333333 )
    v = ( v + ( v >> 4 ) ) & 0x0F0F0F0F0F0F0F0F
    return UInt8( ( v &* 0x0101010101010101 ) >> 56 )
}

/// Count the number of set bits in a value.
/// Ex. countPopulation(0xF000F000) = 8
/// Returns 0 if the word is zero.
public func countPopulation( _ value: UInt32 ) -> UInt8 {
    var v = value;
    v = v - ( ( v >> 1 ) & 0x55555555 )
    v = ( v & 0x33333333 ) + ( ( v >> 2 ) & 0x33333333 )
    v = ( v + ( v >> 4 ) ) & 0xF0F0F0F
    return UInt8( ( v &* 0x1010101 ) >> 24 )
}

/// Return the greatest common divisor of the values using Euclid's algorithm.
public func greatestCommonDivisor( a: UInt64, b: UInt64 ) -> UInt64 {
    var aValue = a
    var bValue = b
    while ( bValue != 0 ) {
        let tmp = bValue
        bValue = aValue % bValue
        aValue = tmp
    }
    return aValue
}

/// Return the greatest common divisor of the values using Euclid's algorithm.
public func greatestCommonDivisor( a: UInt32, b: UInt32 ) -> UInt32 {
    var aValue = a
    var bValue = b
    while ( bValue != 0 ) {
        let tmp = bValue
        bValue = aValue % bValue
        aValue = tmp
    }
    return aValue
}

/// Return the greatest common divisor of the values using Euclid's algorithm.
public func greatestCommonDivisor( a: Int64, b: Int64 ) -> Int64 {
    var aValue = a
    var bValue = b
    while ( bValue != 0 ) {
        let tmp = bValue
        bValue = aValue % bValue
        aValue = tmp
    }
    return aValue
}

/// Return the greatest common divisor of the values using Euclid's algorithm.
public func greatestCommonDivisor( a: Int32, b: Int32 ) -> Int32 {
    var aValue = a
    var bValue = b
    while ( bValue != 0 ) {
        let tmp = bValue
        bValue = aValue % bValue
        aValue = tmp
    }
    return aValue
}

/// Return the value created by shifting bitPattern bitWidth bits to the left by shiftCount. Return nil if value cannot be represented.
public func decodeUInt( _ bitPattern: UInt64, bitWidth: UInt8, shiftCount: UInt8 = 0 ) -> UInt64? {
    guard bitWidth + shiftCount < 64 else { return nil }
    guard isUInt( bitPattern, bitWidth: bitWidth ) else { return nil }
    let mask: UInt64 = ( UInt64( 1 ) << bitWidth ) - 1
    let value = ( bitPattern & mask ) << shiftCount
    return value
}

/// Return the bit pattern created by shifting value bitWidth bits to the right by shiftCount. Return nil if value cannot be represented.
public func encodeUInt( _ value: UInt64, bitWidth: UInt8, shiftCount: UInt8 = 0 ) -> UInt64? {
    guard bitWidth + shiftCount < 64 else { return nil }
    if shiftCount > 0 {
        let mask: UInt64 = ( UInt64( 1 ) << shiftCount ) - 1
        guard value & mask == 0 else { return nil }
    }
    let shiftedValue = value >> shiftCount
    guard isUInt( shiftedValue, bitWidth: bitWidth ) else { return nil }
    return shiftedValue
}

/// Return the value created by shifting bitPattern bitWidth bits to the left by shiftCount. Return nil if value cannot be represented.
public func decodeUInt( _ bitPattern: UInt32, bitWidth: UInt8, shiftCount: UInt8 = 0 ) -> UInt32? {
    guard bitWidth + shiftCount < 32 else { return nil }
    guard isUInt( bitPattern, bitWidth: bitWidth ) else { return nil }
    let mask: UInt32 = ( UInt32( 1 ) << bitWidth ) - 1
    let value = ( bitPattern & mask ) << shiftCount
    return value
}

/// Return the bit pattern created by shifting value bitWidth bits to the right by shiftCount. Return nil if value cannot be represented.
public func encodeUInt( _ value: UInt32, bitWidth: UInt8, shiftCount: UInt8 = 0 ) -> UInt32? {
    guard bitWidth + shiftCount < 32 else { return nil }
    if shiftCount > 0 {
        let mask: UInt32 = ( UInt32( 1 ) << shiftCount ) - 1
        guard value & mask == 0 else { return nil }
    }
    let shiftedValue = value >> shiftCount
    guard isUInt( shiftedValue, bitWidth: bitWidth ) else { return nil }
    return shiftedValue
}

/// Return the signed integer represented by the bitWidth least significant bits shifted to left shiftCount.
public func decodeInt( _ bitPattern: UInt64, bitWidth: UInt8, shiftCount: UInt8 = 0 ) -> Int64? {
    guard bitWidth > 1 else { return nil }
    guard bitWidth + shiftCount < 64 else { return nil }
    let mask: UInt64 = ( UInt64( 1 ) << bitWidth ) - 1
    let signMask: UInt64 = UInt64( 1 ) << ( bitWidth - 1 )
    var maskedValue = bitPattern & mask
    let isNegative = ( signMask & maskedValue ) != 0
    if isNegative {
        maskedValue |= ~mask
    }
    let shiftedValue = maskedValue << shiftCount
    return Int64( bitPattern: shiftedValue )
}

public func encodeInt( _ value: Int64, bitWidth: UInt8, shiftCount: UInt8 = 0 ) -> UInt64? {
    guard bitWidth > 1 else { return nil }
    guard bitWidth + shiftCount < 64 else { return nil }
    var bitPattern = UInt64( bitPattern: value )
    if shiftCount > 0 {
        let mask: UInt64 = ( UInt64( 1 ) << shiftCount ) - 1
        guard bitPattern & mask == 0 else { return nil }
        bitPattern >>= shiftCount
    }
    let mask: UInt64 = ( UInt64( 1 ) << bitWidth ) - 1
    let maskedBitPattern = bitPattern & mask
    return maskedBitPattern
}

/// Return the signed integer represented by the bitWidth least significant bits shifted to left shiftCount.
public func decodeInt( _ bitPattern: UInt32, bitWidth: UInt8, shiftCount: UInt8 = 0 ) -> Int32? {
    guard bitWidth > 1 else { return nil }
    guard bitWidth + shiftCount < 32 else { return nil }
    let mask: UInt32 = ( UInt32( 1 ) << bitWidth ) - 1
    let signMask: UInt32 = UInt32( 1 ) << ( bitWidth - 1 )
    var maskedValue = bitPattern & mask
    let isNegative = ( signMask & maskedValue ) != 0
    if isNegative {
        maskedValue |= ~mask
    }
    let shiftedValue = maskedValue << shiftCount
    return Int32( bitPattern: shiftedValue )
}

public func encodeInt( _ value: Int32, bitWidth: UInt8, shiftCount: UInt8 = 0 ) -> UInt32? {
    guard bitWidth > 1 else { return nil }
    guard bitWidth + shiftCount < 32 else { return nil }
    var bitPattern = UInt32( bitPattern: value )
    if shiftCount > 0 {
        let mask: UInt32 = ( UInt32( 1 ) << shiftCount ) - 1
        guard bitPattern & mask == 0 else { return nil }
        bitPattern >>= shiftCount
    }
    let mask: UInt32 = ( UInt32( 1 ) << bitWidth ) - 1
    let maskedBitPattern = bitPattern & mask
    return maskedBitPattern
}

public func parseInt32( text: String ) -> Int32? {
    guard text.hasPrefix( "0x" ) else { return nil }
    let start = text.index( text.startIndex, offsetBy: 2 )
    let range = start ..< text.endIndex
    let trimmedText = text[ range ]
    guard let value = Int32( trimmedText, radix: 16 ) else { return nil }
    let bitPattern = UInt32( bitPattern: value )
    let parsed = Int32( bitPattern: bitPattern )
    return parsed
}
