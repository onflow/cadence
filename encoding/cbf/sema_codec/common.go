package sema_codec

// TODO add protections against regressions from changes to enum
// TODO consider putting simple and numeric types in a specific ranges (128+, 64-127)
//      that turns certain bits into flags for the presence of those types, which can be calculated very fast
//      (check the leftmost bit first, then the next bit, in that order, or there's overlap)
type EncodedSema byte

const (
	EncodedSemaUnknown EncodedSema = iota // lacking type information; should not be encoded

	// Simple Types

	EncodedSemaSimpleTypeAnyType
	EncodedSemaSimpleTypeAnyResourceType
	EncodedSemaSimpleTypeAnyStructType
	EncodedSemaSimpleTypeBlockType
	EncodedSemaSimpleTypeBoolType
	EncodedSemaSimpleTypeCharacterType
	EncodedSemaSimpleTypeDeployedContractType
	EncodedSemaSimpleTypeInvalidType
	EncodedSemaSimpleTypeMetaType
	EncodedSemaSimpleTypeNeverType
	EncodedSemaSimpleTypePathType
	EncodedSemaSimpleTypeStoragePathType
	EncodedSemaSimpleTypeCapabilityPathType
	EncodedSemaSimpleTypePublicPathType
	EncodedSemaSimpleTypePrivatePathType
	EncodedSemaSimpleTypeStorableType
	EncodedSemaSimpleTypeStringType
	EncodedSemaSimpleTypeVoidType

	// Numeric Types

	EncodedSemaNumericTypeNumberType
	EncodedSemaNumericTypeSignedNumberType
	EncodedSemaNumericTypeIntegerType
	EncodedSemaNumericTypeSignedIntegerType
	EncodedSemaNumericTypeIntType
	EncodedSemaNumericTypeInt8Type
	EncodedSemaNumericTypeInt16Type
	EncodedSemaNumericTypeInt32Type
	EncodedSemaNumericTypeInt64Type
	EncodedSemaNumericTypeInt128Type
	EncodedSemaNumericTypeInt256Type
	EncodedSemaNumericTypeUIntType
	EncodedSemaNumericTypeUInt8Type
	EncodedSemaNumericTypeUInt16Type
	EncodedSemaNumericTypeUInt32Type
	EncodedSemaNumericTypeUInt64Type
	EncodedSemaNumericTypeUInt128Type
	EncodedSemaNumericTypeUInt256Type
	EncodedSemaNumericTypeWord8Type
	EncodedSemaNumericTypeWord16Type
	EncodedSemaNumericTypeWord32Type
	EncodedSemaNumericTypeWord64Type
	EncodedSemaNumericTypeFixedPointType
	EncodedSemaNumericTypeSignedFixedPointType

	// Fixed Point Numeric Types

	EncodedSemaFix64Type
	EncodedSemaUFix64Type

	// Pointable Types

	EncodedSemaCompositeType
	EncodedSemaInterfaceType
	EncodedSemaGenericType
	EncodedSemaTransactionType
	EncodedSemaRestrictedType
	EncodedSemaVariableSizedType
	EncodedSemaConstantSizedType
	EncodedSemaFunctionType
	EncodedSemaDictionaryType

	// Other Types

	EncodedSemaNilType // no type is specified
	EncodedSemaOptionalType

	EncodedSemaReferenceType
	EncodedSemaAddressType
	EncodedSemaCapabilityType
	EncodedSemaPointerType
)
