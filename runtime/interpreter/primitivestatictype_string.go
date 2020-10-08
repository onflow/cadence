// Code generated by "stringer -type=PrimitiveStaticType -trimprefix=PrimitiveStaticType"; DO NOT EDIT.

package interpreter

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[PrimitiveStaticTypeUnknown-0]
	_ = x[PrimitiveStaticTypeVoid-1]
	_ = x[PrimitiveStaticTypeAny-2]
	_ = x[PrimitiveStaticTypeNever-3]
	_ = x[PrimitiveStaticTypeAnyStruct-4]
	_ = x[PrimitiveStaticTypeAnyResource-5]
	_ = x[PrimitiveStaticTypeBool-6]
	_ = x[PrimitiveStaticTypeAddress-7]
	_ = x[PrimitiveStaticTypeString-8]
	_ = x[PrimitiveStaticTypeCharacter-9]
	_ = x[PrimitiveStaticTypeNumber-18]
	_ = x[PrimitiveStaticTypeSignedNumber-19]
	_ = x[PrimitiveStaticTypeInteger-24]
	_ = x[PrimitiveStaticTypeSignedInteger-25]
	_ = x[PrimitiveStaticTypeFixedPoint-30]
	_ = x[PrimitiveStaticTypeSignedFixedPoint-31]
	_ = x[PrimitiveStaticTypeInt-36]
	_ = x[PrimitiveStaticTypeInt8-37]
	_ = x[PrimitiveStaticTypeInt16-38]
	_ = x[PrimitiveStaticTypeInt32-39]
	_ = x[PrimitiveStaticTypeInt64-40]
	_ = x[PrimitiveStaticTypeInt128-41]
	_ = x[PrimitiveStaticTypeInt256-42]
	_ = x[PrimitiveStaticTypeUInt-44]
	_ = x[PrimitiveStaticTypeUInt8-45]
	_ = x[PrimitiveStaticTypeUInt16-46]
	_ = x[PrimitiveStaticTypeUInt32-47]
	_ = x[PrimitiveStaticTypeUInt64-48]
	_ = x[PrimitiveStaticTypeUInt128-49]
	_ = x[PrimitiveStaticTypeUInt256-50]
	_ = x[PrimitiveStaticTypeWord8-53]
	_ = x[PrimitiveStaticTypeWord16-54]
	_ = x[PrimitiveStaticTypeWord32-55]
	_ = x[PrimitiveStaticTypeWord64-56]
	_ = x[PrimitiveStaticTypeFix64-64]
	_ = x[PrimitiveStaticTypeUFix64-72]
	_ = x[PrimitiveStaticTypePath-76]
	_ = x[PrimitiveStaticTypeCapability-77]
	_ = x[PrimitiveStaticTypeStoragePath-78]
	_ = x[PrimitiveStaticTypeCapabilityPath-79]
	_ = x[PrimitiveStaticTypePublicPath-80]
	_ = x[PrimitiveStaticTypePrivatePath-81]
}

const (
	_PrimitiveStaticType_name_0 = "UnknownVoidAnyNeverAnyStructAnyResourceBoolAddressStringCharacter"
	_PrimitiveStaticType_name_1 = "NumberSignedNumber"
	_PrimitiveStaticType_name_2 = "IntegerSignedInteger"
	_PrimitiveStaticType_name_3 = "FixedPointSignedFixedPoint"
	_PrimitiveStaticType_name_4 = "IntInt8Int16Int32Int64Int128Int256"
	_PrimitiveStaticType_name_5 = "UIntUInt8UInt16UInt32UInt64UInt128UInt256"
	_PrimitiveStaticType_name_6 = "Word8Word16Word32Word64"
	_PrimitiveStaticType_name_7 = "Fix64"
	_PrimitiveStaticType_name_8 = "UFix64"
	_PrimitiveStaticType_name_9 = "PathCapabilityStoragePathCapabilityPathPublicPathPrivatePath"
)

var (
	_PrimitiveStaticType_index_0 = [...]uint8{0, 7, 11, 14, 19, 28, 39, 43, 50, 56, 65}
	_PrimitiveStaticType_index_1 = [...]uint8{0, 6, 18}
	_PrimitiveStaticType_index_2 = [...]uint8{0, 7, 20}
	_PrimitiveStaticType_index_3 = [...]uint8{0, 10, 26}
	_PrimitiveStaticType_index_4 = [...]uint8{0, 3, 7, 12, 17, 22, 28, 34}
	_PrimitiveStaticType_index_5 = [...]uint8{0, 4, 9, 15, 21, 27, 34, 41}
	_PrimitiveStaticType_index_6 = [...]uint8{0, 5, 11, 17, 23}
	_PrimitiveStaticType_index_9 = [...]uint8{0, 4, 14, 25, 39, 49, 60}
)

func (i PrimitiveStaticType) String() string {
	switch {
	case i <= 9:
		return _PrimitiveStaticType_name_0[_PrimitiveStaticType_index_0[i]:_PrimitiveStaticType_index_0[i+1]]
	case 18 <= i && i <= 19:
		i -= 18
		return _PrimitiveStaticType_name_1[_PrimitiveStaticType_index_1[i]:_PrimitiveStaticType_index_1[i+1]]
	case 24 <= i && i <= 25:
		i -= 24
		return _PrimitiveStaticType_name_2[_PrimitiveStaticType_index_2[i]:_PrimitiveStaticType_index_2[i+1]]
	case 30 <= i && i <= 31:
		i -= 30
		return _PrimitiveStaticType_name_3[_PrimitiveStaticType_index_3[i]:_PrimitiveStaticType_index_3[i+1]]
	case 36 <= i && i <= 42:
		i -= 36
		return _PrimitiveStaticType_name_4[_PrimitiveStaticType_index_4[i]:_PrimitiveStaticType_index_4[i+1]]
	case 44 <= i && i <= 50:
		i -= 44
		return _PrimitiveStaticType_name_5[_PrimitiveStaticType_index_5[i]:_PrimitiveStaticType_index_5[i+1]]
	case 53 <= i && i <= 56:
		i -= 53
		return _PrimitiveStaticType_name_6[_PrimitiveStaticType_index_6[i]:_PrimitiveStaticType_index_6[i+1]]
	case i == 64:
		return _PrimitiveStaticType_name_7
	case i == 72:
		return _PrimitiveStaticType_name_8
	case 76 <= i && i <= 81:
		i -= 76
		return _PrimitiveStaticType_name_9[_PrimitiveStaticType_index_9[i]:_PrimitiveStaticType_index_9[i+1]]
	default:
		return "PrimitiveStaticType(" + strconv.FormatInt(int64(i), 10) + ")"
	}
}
