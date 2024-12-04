//
//  Memory.swift
//  Riscv32
//

public class Memory {
    public var bytes: UnsafeMutablePointer<UInt8>?
    public var size: Int
    public var beginPosition: UnsafeMutablePointer<UInt8>!  // start of allocated data
    public var endPosition: UnsafeMutablePointer<UInt8>     // end of allocated data
    
    public init( bytes: [UInt8] ) {
        let memorySize = 16 * 4096
        self.size = bytes.count + memorySize
        self.bytes = UnsafeMutablePointer<UInt8>.allocate( capacity: self.size )
        for index in 0 ..< bytes.count {
            self.bytes![ index ] = bytes[ index ]
        }
        self.beginPosition = self.bytes
        self.endPosition = beginPosition + self.size
    }
    
    public init( size: Int ) {
        self.bytes = UnsafeMutablePointer<UInt8>.allocate( capacity: size )
        self.beginPosition = self.bytes
        self.endPosition = beginPosition + size
        self.size = size
    }
    
    public subscript( _ address: UInt32 ) -> UInt8? {
        get {
            return getUInt8( address: address )
        }
        set( newValue ) {
            let result = put( address: address, uInt8: newValue! )
            assert( result )
        }
    }
    
    public subscript( _ address: UInt32 ) -> UInt16? {
        get {
            return getUInt16( address: address )
        }
        set( newValue ) {
            let result = put( address: address, uInt16: newValue! )
            assert( result )
        }
    }
    
    public subscript( _ address: UInt32 ) -> UInt32? {
        get {
            return getUInt32( address: address )
        }
        set( newValue ) {
            let result = put( address: address, uInt32: newValue! )
            assert( result )
        }
    }
    
    public subscript( _ address: UInt32 ) -> UInt64? {
        get {
            return getUInt64( address: address )
        }
        set( newValue ) {
            let result = put( address: address, uInt64: newValue! )
            assert( result )
        }
    }
    
    public subscript( _ address: UInt32 ) -> Int8? {
        get {
            return getInt8( address: address )
        }
        set( newValue ) {
            let result = put( address: address, int8: newValue! )
            assert( result )
        }
    }
    
    public subscript( _ address: UInt32 ) -> Int16? {
        get {
            return getInt16( address: address )
        }
        set( newValue ) {
            let result = put( address: address, int16: newValue! )
            assert( result )
        }
    }
    
    public subscript( _ address: UInt32 ) -> Int32? {
        get {
            return getInt32( address: address )
        }
        set( newValue ) {
            let result = put( address: address, int32: newValue! )
            assert( result )
        }
    }
    
    public subscript( _ address: UInt32 ) -> Int64? {
        get {
            return getInt64( address: address )
        }
        set( newValue ) {
            let result = put( address: address, int64: newValue! )
            assert( result )
        }
    }
    
    public subscript( _ address: UInt32 ) -> Float? {
        get {
            return getFloat( address: address )
        }
        set( newValue ) {
            let result = put( address: address, float: newValue! )
            assert( result )
        }
    }
    
    public subscript( _ address: UInt32 ) -> Double? {
        get {
            return getDouble( address: address )
        }
        set( newValue ) {
            let result = put( address: address, double: newValue! )
            assert( result )
        }
    }
    
    public func getDouble( address: UInt32, isLittleEndian: Bool = true ) -> Double? {
        if let integer = getUInt64( address: address, isLittleEndian: isLittleEndian ) {
            return Double( bitPattern: integer )
        } else {
            return nil
        }
    }
    
    public func getFloat( address: UInt32, isLittleEndian: Bool = true ) -> Float? {
        if let integer = getUInt32( address: address, isLittleEndian: isLittleEndian ) {
            return Float( bitPattern: integer )
        } else {
            return nil
        }
    }
    
    public func getInt16( address: UInt32, isLittleEndian: Bool = true ) -> Int16? {
        let readPosition = beginPosition + Int( address )
        
        if readPosition + 1 < endPosition {
            if isLittleEndian {
                var int16: UInt16 = 0
                
                int16 += UInt16( readPosition[ 0 ] )
                int16 += UInt16( readPosition[ 1 ] ) << 8
                
                return Int16( bitPattern: int16 )
            } else {
                var int16: UInt16 = 0
                
                int16 += UInt16( readPosition[ 0 ] ) << 8
                int16 += UInt16( readPosition[ 1 ] )
                
                return Int16( bitPattern: int16 )
            }
        } else {
            return nil
        }
    }
    
    public func getInt32( address: UInt32, isLittleEndian: Bool = true ) -> Int32? {
        let readPosition = beginPosition + Int( address )
        
        if readPosition + 3 < endPosition {
            if isLittleEndian {
                var int32: UInt32 = 0
                
                int32 += UInt32( readPosition[ 0 ] )
                int32 += UInt32( readPosition[ 1 ] ) << 8
                int32 += UInt32( readPosition[ 2 ] ) << 16
                int32 += UInt32( readPosition[ 3 ] ) << 24
                
                return Int32( bitPattern: int32 )
            } else {
                var int32: UInt32 = 0
                
                int32 += UInt32( readPosition[ 0 ] ) << 24
                int32 += UInt32( readPosition[ 1 ] ) << 16
                int32 += UInt32( readPosition[ 2 ] ) << 8
                int32 += UInt32( readPosition[ 3 ] )
                
                return Int32( bitPattern: int32 )
            }
        } else {
            return nil
        }
    }
    
    public func getInt64( address: UInt32, isLittleEndian: Bool = true ) -> Int64? {
        let readPosition = beginPosition + Int( address )
        
        if readPosition + 7 < endPosition {
            if isLittleEndian {
                var int64: UInt64 = 0
                
                int64 += UInt64( readPosition[ 0 ] )
                int64 += UInt64( readPosition[ 1 ] ) << 8
                int64 += UInt64( readPosition[ 2 ] ) << 16
                int64 += UInt64( readPosition[ 3 ] ) << 24
                int64 += UInt64( readPosition[ 4 ] ) << 32
                int64 += UInt64( readPosition[ 5 ] ) << 40
                int64 += UInt64( readPosition[ 6 ] ) << 48
                int64 += UInt64( readPosition[ 7 ] ) << 56
                
                return Int64( bitPattern: int64 )
            } else {
                var int64: UInt64 = 0
                
                int64 += UInt64( readPosition[ 0 ] ) << 56
                int64 += UInt64( readPosition[ 1 ] ) << 48
                int64 += UInt64( readPosition[ 2 ] ) << 40
                int64 += UInt64( readPosition[ 3 ] ) << 32
                int64 += UInt64( readPosition[ 4 ] ) << 24
                int64 += UInt64( readPosition[ 5 ] ) << 16
                int64 += UInt64( readPosition[ 6 ] ) << 8
                int64 += UInt64( readPosition[ 7 ] )
                
                return Int64( bitPattern: int64 )
            }
        } else {
            return nil
        }
    }
    
    public func getInt8( address: UInt32 ) -> Int8? {
        let readPosition = beginPosition + Int( address )
        
        if readPosition < endPosition {
            let int8 = readPosition[ 0 ]
            return Int8( bitPattern:  int8 )
        } else {
            return nil
        }
    }
    
    public func getUInt16( address: UInt32, isLittleEndian: Bool = true ) -> UInt16? {
        let readPosition = beginPosition + Int( address )
        
        if readPosition + 1 < endPosition {
            if isLittleEndian {
                var uInt16: UInt16 = 0
                
                uInt16 += UInt16( readPosition[ 0 ] )
                uInt16 += UInt16( readPosition[ 1 ] ) << 8
                
                return uInt16
            } else {
                var uInt16: UInt16 = 0
                
                uInt16 += UInt16( readPosition[ 0 ] ) << 8
                uInt16 += UInt16( readPosition[ 1 ] )
                
                return uInt16
            }
        } else {
            return nil
        }
    }
    
    public func getUInt32( address: UInt32, isLittleEndian: Bool = true ) -> UInt32? {
        let readPosition = beginPosition + Int( address )
        
        if readPosition + 3 < endPosition {
            if isLittleEndian {
                var uInt32: UInt32 = 0
                
                uInt32 += UInt32( readPosition[ 0 ] )
                uInt32 += UInt32( readPosition[ 1 ] ) << 8
                uInt32 += UInt32( readPosition[ 2 ] ) << 16
                uInt32 += UInt32( readPosition[ 3 ] ) << 24
                
                return uInt32
            } else {
                var uInt32: UInt32 = 0
                
                uInt32 += UInt32( readPosition[ 0 ] ) << 24
                uInt32 += UInt32( readPosition[ 1 ] ) << 16
                uInt32 += UInt32( readPosition[ 2 ] ) << 8
                uInt32 += UInt32( readPosition[ 3 ] )
                
                return uInt32
            }
        } else {
            return nil
        }
    }
    
    public func getUInt64( address: UInt32, isLittleEndian: Bool = true ) -> UInt64? {
        let readPosition = beginPosition + Int( address )
        
        if readPosition + 7 < endPosition {
            if isLittleEndian {
                var uInt64: UInt64 = 0
                
                uInt64 += UInt64( readPosition[ 0 ] )
                uInt64 += UInt64( readPosition[ 1 ] ) << 8
                uInt64 += UInt64( readPosition[ 2 ] ) << 16
                uInt64 += UInt64( readPosition[ 3 ] ) << 24
                uInt64 += UInt64( readPosition[ 4 ] ) << 32
                uInt64 += UInt64( readPosition[ 5 ] ) << 40
                uInt64 += UInt64( readPosition[ 6 ] ) << 48
                uInt64 += UInt64( readPosition[ 7 ] ) << 56
                
                return uInt64
            } else {
                var uInt64: UInt64 = 0
                
                uInt64 += UInt64( readPosition[ 0 ] ) << 56
                uInt64 += UInt64( readPosition[ 1 ] ) << 48
                uInt64 += UInt64( readPosition[ 2 ] ) << 40
                uInt64 += UInt64( readPosition[ 3 ] ) << 32
                uInt64 += UInt64( readPosition[ 4 ] ) << 24
                uInt64 += UInt64( readPosition[ 5 ] ) << 16
                uInt64 += UInt64( readPosition[ 6 ] ) << 8
                uInt64 += UInt64( readPosition[ 7 ] )
                
                return uInt64
            }
        } else {
            return nil
        }
    }
    
    public func getUInt8( address: UInt32 ) -> UInt8? {
        let readPosition = beginPosition + Int( address )
        
        if readPosition < endPosition {
            let byte = readPosition[ 0 ]
            return byte
        } else {
            return nil
        }
    }
    
    public func put( address: UInt32, double: Double, isLittleEndian: Bool = true ) -> Bool {
        put( address: address, uInt64: UInt64( bitPattern: Int64( double.bitPattern ) ), isLittleEndian: isLittleEndian )
    }
    
    public func put( address: UInt32, float: Float, isLittleEndian: Bool = true ) -> Bool {
        put( address: address, uInt32: UInt32( bitPattern: Int32( bitPattern: float.bitPattern ) ), isLittleEndian: isLittleEndian )
    }
    
    public func put( address: UInt32, int16: Int16, isLittleEndian: Bool = true ) -> Bool {
        let writePosition = beginPosition + Int( address )
        guard endPosition - writePosition >= 2 else { return false }
        
        if isLittleEndian {
            writePosition[ 0 ] = UInt8( truncatingIfNeeded: int16 )
            writePosition[ 1 ] = UInt8( truncatingIfNeeded: int16 >> 8 )
        } else {
            writePosition[ 0 ] = UInt8( truncatingIfNeeded: int16 >> 8 )
            writePosition[ 1 ] = UInt8( truncatingIfNeeded: int16 )
        }
        
        return true
    }
    
    public func put( address: UInt32, int32: Int32, isLittleEndian: Bool = true ) -> Bool {
        let writePosition = beginPosition + Int( address )
        guard endPosition - writePosition >= 4 else { return false }
        
        if isLittleEndian {
            writePosition[ 0 ] = UInt8( truncatingIfNeeded: int32 )
            writePosition[ 1 ] = UInt8( truncatingIfNeeded: int32 >> 8 )
            writePosition[ 2 ] = UInt8( truncatingIfNeeded: int32 >> 16 )
            writePosition[ 3 ] = UInt8( truncatingIfNeeded: int32 >> 24 )
        } else {
            writePosition[ 0 ] = UInt8( truncatingIfNeeded: int32 >> 24 )
            writePosition[ 1 ] = UInt8( truncatingIfNeeded: int32 >> 16 )
            writePosition[ 2 ] = UInt8( truncatingIfNeeded: int32 >> 8 )
            writePosition[ 3 ] = UInt8( truncatingIfNeeded: int32 )
        }
        
        return true
    }
    
    public func put( address: UInt32, int64: Int64, isLittleEndian: Bool = true ) -> Bool {
        let writePosition = beginPosition + Int( address )
        guard endPosition - writePosition >= 8 else { return false }
        
        if isLittleEndian {
            writePosition[ 0 ] = UInt8( truncatingIfNeeded: int64 )
            writePosition[ 1 ] = UInt8( truncatingIfNeeded: int64 >> 8 )
            writePosition[ 2 ] = UInt8( truncatingIfNeeded: int64 >> 16 )
            writePosition[ 3 ] = UInt8( truncatingIfNeeded: int64 >> 24 )
            writePosition[ 4 ] = UInt8( truncatingIfNeeded: int64 >> 32 )
            writePosition[ 5 ] = UInt8( truncatingIfNeeded: int64 >> 40 )
            writePosition[ 6 ] = UInt8( truncatingIfNeeded: int64 >> 48 )
            writePosition[ 7 ] = UInt8( truncatingIfNeeded: int64 >> 56 )
        } else {
            writePosition[ 0 ] = UInt8( truncatingIfNeeded: int64 >> 56 )
            writePosition[ 1 ] = UInt8( truncatingIfNeeded: int64 >> 48 )
            writePosition[ 2 ] = UInt8( truncatingIfNeeded: int64 >> 40 )
            writePosition[ 3 ] = UInt8( truncatingIfNeeded: int64 >> 32 )
            writePosition[ 4 ] = UInt8( truncatingIfNeeded: int64 >> 24 )
            writePosition[ 5 ] = UInt8( truncatingIfNeeded: int64 >> 16 )
            writePosition[ 6 ] = UInt8( truncatingIfNeeded: int64 >> 8 )
            writePosition[ 7 ] = UInt8( truncatingIfNeeded: int64 )
        }
        
        return true
    }
    
    public func put( address: UInt32, int8: Int8 ) -> Bool {
        let writePosition = beginPosition + Int( address )
        guard endPosition - writePosition >= 1 else { return false }
        
        writePosition[ 0 ] = UInt8( bitPattern:  int8 )
        
        return true
    }
    
    public func put( address: UInt32, uInt16: UInt16, isLittleEndian: Bool = true ) -> Bool {
        let writePosition = beginPosition + Int( address )
        guard endPosition - writePosition >= 2 else { return false }
        
        if isLittleEndian {
            writePosition[ 0 ] = UInt8( truncatingIfNeeded: uInt16 )
            writePosition[ 1 ] = UInt8( truncatingIfNeeded: uInt16 >> 8 )
        } else {
            writePosition[ 0 ] = UInt8( truncatingIfNeeded: uInt16 >> 8 )
            writePosition[ 1 ] = UInt8( truncatingIfNeeded: uInt16 )
        }
        
        return true
    }
    
    public func put( address: UInt32, uInt32: UInt32, isLittleEndian: Bool = true ) -> Bool {
        let writePosition = beginPosition + Int( address )
        guard endPosition - writePosition >= 4 else { return false }
        
        if isLittleEndian {
            writePosition[ 0 ] = UInt8( truncatingIfNeeded: uInt32 )
            writePosition[ 1 ] = UInt8( truncatingIfNeeded: uInt32 >> 8 )
            writePosition[ 2 ] = UInt8( truncatingIfNeeded: uInt32 >> 16 )
            writePosition[ 3 ] = UInt8( truncatingIfNeeded: uInt32 >> 24 )
        } else {
            writePosition[ 0 ] = UInt8( truncatingIfNeeded: uInt32 >> 24 )
            writePosition[ 1 ] = UInt8( truncatingIfNeeded: uInt32 >> 16 )
            writePosition[ 2 ] = UInt8( truncatingIfNeeded: uInt32 >> 8 )
            writePosition[ 3 ] = UInt8( truncatingIfNeeded: uInt32 )
        }
        
        return true
    }
    
    public func put( address: UInt32, uInt64: UInt64, isLittleEndian: Bool = true ) -> Bool {
        let writePosition = beginPosition + Int( address )
        guard endPosition - writePosition >= 8 else { return false }
        
        if isLittleEndian {
            writePosition[ 0 ] = UInt8( truncatingIfNeeded: uInt64 )
            writePosition[ 1 ] = UInt8( truncatingIfNeeded: uInt64 >> 8 )
            writePosition[ 2 ] = UInt8( truncatingIfNeeded: uInt64 >> 16 )
            writePosition[ 3 ] = UInt8( truncatingIfNeeded: uInt64 >> 24 )
            writePosition[ 4 ] = UInt8( truncatingIfNeeded: uInt64 >> 32 )
            writePosition[ 5 ] = UInt8( truncatingIfNeeded: uInt64 >> 40 )
            writePosition[ 6 ] = UInt8( truncatingIfNeeded: uInt64 >> 48 )
            writePosition[ 7 ] = UInt8( truncatingIfNeeded: uInt64 >> 56 )
        } else {
            writePosition[ 0 ] = UInt8( truncatingIfNeeded: uInt64 >> 56 )
            writePosition[ 1 ] = UInt8( truncatingIfNeeded: uInt64 >> 48 )
            writePosition[ 2 ] = UInt8( truncatingIfNeeded: uInt64 >> 40 )
            writePosition[ 3 ] = UInt8( truncatingIfNeeded: uInt64 >> 32 )
            writePosition[ 4 ] = UInt8( truncatingIfNeeded: uInt64 >> 24 )
            writePosition[ 5 ] = UInt8( truncatingIfNeeded: uInt64 >> 16 )
            writePosition[ 6 ] = UInt8( truncatingIfNeeded: uInt64 >> 8 )
            writePosition[ 7 ] = UInt8( truncatingIfNeeded: uInt64 )
        }
        
        return true
    }
    
    public func put( address: UInt32, uInt8: UInt8 ) -> Bool {
        let writePosition = beginPosition + Int( address )
        guard endPosition - writePosition >= 1 else { return false }
        
        writePosition[ 0 ] = uInt8
        
        return true
    }
}
