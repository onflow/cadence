package cbf_codec

type EncodedValue byte

const (
	EncodedValueUnknown EncodedValue = iota

	EncodedValueVoid
	EncodedValueOptional
	EncodedValueBool
	EncodedValueString
	EncodedValueBytes // NOTE: only used in tests so this might be removable
	EncodedValueCharacter
	EncodedValueAddress
	EncodedValueInt
	EncodedValueInt8
	EncodedValueInt16
	EncodedValueInt32
	EncodedValueInt64
	EncodedValueInt128
	EncodedValueInt256
	EncodedValueUInt
	EncodedValueUInt8
	EncodedValueUInt16
	EncodedValueUInt32
	EncodedValueUInt64
	EncodedValueUInt128
	EncodedValueUInt256
	EncodedValueWord8
	EncodedValueWord16
	EncodedValueWord32
	EncodedValueWord64
	EncodedValueFix64
	EncodedValueUFix64
	EncodedValueUntypedArray
	EncodedValueVariableArray
	EncodedValueConstantArray
	EncodedValueDictionary
	EncodedValueStruct
	EncodedValueResource
	EncodedValueEvent
	EncodedValueContract
	EncodedValueLink
	EncodedValuePath
	EncodedValueCapability
	EncodedValueEnum

	// TODO dont do this here. goes as first byte of custom codec
	EncodedValueReservedForJsonCodec = byte('{')
)

type EncodedType byte

const (
	EncodedTypeUnknown EncodedType = iota

	// TODO classify these types, probably as simple, complex, or abstract

	// Concrete Types

	EncodedTypeVoid
	EncodedTypeBool
	EncodedTypeOptional
	EncodedTypeString
	EncodedTypeCharacter
	EncodedTypeBytes
	EncodedTypeAddress
	EncodedTypeInt
	EncodedTypeInt8
	EncodedTypeInt16
	EncodedTypeInt32
	EncodedTypeInt64
	EncodedTypeInt128
	EncodedTypeInt256
	EncodedTypeUInt
	EncodedTypeUInt8
	EncodedTypeUInt16
	EncodedTypeUInt32
	EncodedTypeUInt64
	EncodedTypeUInt128
	EncodedTypeUInt256
	EncodedTypeWord8
	EncodedTypeWord16
	EncodedTypeWord32
	EncodedTypeWord64
	EncodedTypeFix64
	EncodedTypeUFix64
	EncodedTypeVariableSizedArray
	EncodedTypeConstantSizedArray
	EncodedTypeDictionary
	EncodedTypeStruct
	EncodedTypeResource
	EncodedTypeEvent
	EncodedTypeContract
	EncodedTypeStructInterface
	EncodedTypeResourceInterface
	EncodedTypeContractInterface
	EncodedTypeFunction
	EncodedTypeReference
	EncodedTypeRestricted
	EncodedTypeBlock
	EncodedTypeCapabilityPath
	EncodedTypeStoragePath
	EncodedTypePublicPath
	EncodedTypePrivatePath
	EncodedTypeCapability
	EncodedTypeEnum
	EncodedTypeAuthAccount
	EncodedTypePublicAccount
	EncodedTypeDeployedContract
	EncodedTypeAuthAccountContracts
	EncodedTypePublicAccountContracts
	EncodedTypeAuthAccountKeys
	EncodedTypePublicAccountKeys

	// Abstract Types

	EncodedTypeNever
	EncodedTypeNumber
	EncodedTypeSignedNumber
	EncodedTypeInteger
	EncodedTypeSignedInteger
	EncodedTypeFixedPoint
	EncodedTypeSignedFixedPoint
	EncodedTypeAnyType
	EncodedTypeAnyStructType
	EncodedTypeAnyResourceType
	EncodedTypePath

	EncodedTypeComposite // TODO is this necessary?
	EncodedTypeInterface // TODO is this necessary?

	// TODO - classify

	EncodedTypeMetaType

	// TODO dont do this here. goes as first byte of custom codec
	EncodedTypeReservedForJsonCodec = byte('{')
)
