/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Dapper Labs, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package sema

import (
	"github.com/onflow/cadence/runtime/errors"
)

// TypeTag is a bitmask representation for types.
// Each type has a unique dedicated bit/bit-pattern in the bitmask.
// The mask consist of two sections: `lowerMask` and the `upperMask`.
// Each section can represent 64-types.
type TypeTag struct {
	lowerMask uint64
	upperMask uint64
}

var allTypeTags = map[TypeTag]bool{}
var allLowerMaskedTypeTags []TypeTag
var allUpperMaskedTypeTags []TypeTag

func newTypeTagFromLowerMask(mask uint64) TypeTag {
	typeTag := TypeTag{
		lowerMask: mask,
		upperMask: 0,
	}

	if _, ok := allTypeTags[typeTag]; ok {
		panic(errors.NewUnexpectedError("duplicate type tag: %v", typeTag))
	}

	allTypeTags[typeTag] = true

	allLowerMaskedTypeTags = append(allLowerMaskedTypeTags, typeTag)

	return typeTag
}

func newTypeTagFromUpperMask(mask uint64) TypeTag {
	typeTag := TypeTag{
		lowerMask: 0,
		upperMask: mask,
	}

	if _, ok := allTypeTags[typeTag]; ok {
		panic(errors.NewUnexpectedError("duplicate type tag: %v", typeTag))
	}

	allTypeTags[typeTag] = true

	allUpperMaskedTypeTags = append(allUpperMaskedTypeTags, typeTag)

	return typeTag
}

func (t TypeTag) Equals(tag TypeTag) bool {
	return t.lowerMask == tag.lowerMask &&
		t.upperMask == tag.upperMask
}

func (t TypeTag) And(tag TypeTag) TypeTag {
	return TypeTag{
		lowerMask: t.lowerMask & tag.lowerMask,
		upperMask: t.upperMask & tag.upperMask,
	}
}

func (t TypeTag) Or(tag TypeTag) TypeTag {
	return TypeTag{
		lowerMask: t.lowerMask | tag.lowerMask,
		upperMask: t.upperMask | tag.upperMask,
	}
}

func (t TypeTag) Not() TypeTag {
	return TypeTag{
		lowerMask: ^t.lowerMask,
		upperMask: ^t.upperMask,
	}
}

func (t TypeTag) ContainsAny(typeTags ...TypeTag) bool {
	for _, tag := range typeTags {
		if t.And(tag).Equals(tag) {
			return true
		}
	}

	return false
}

func (t TypeTag) BelongsTo(typeTag TypeTag) bool {
	return t.And(typeTag).Equals(t)
}

/*
 * Following defines the masks used to represent known types in Cadence.
 * Each of the numeric/simple type has a unique dedicated mask. All the derived
 * types (e.g: composites, dictionaries, etc.) have a common bitmask for each
 * category (e.g: one for composites, one for dictionaries, etc.), because
 * tag itself is not sufficient to represent those types, and need more complex
 * analysis. So the bitmask is only used to represent their 'kind'.
 *
 * For simple/numeric types, it is required to have a unique mask. For others, it's
 * optional to have a unique tag (but requires a one for their 'category'). Having
 * a unique mask for a derived type `T` would only give some performance optimization
 * when finding the supertype of a collection of all `T`s. Because it will exit early
 * by checking the corresponding bit of the bitmask, and don't need to fall back on
 * checking type's deep-equality.
 *
 * NOTE: Builtin composite types don't have dedicated masks even though they are
 * pre-known. This is because, though they are pre-known, we might want to treat those
 * as any other composite type (e.g: finding common conformance, etc.) and let them
 * also go on the same execution path as other composites, by NOT optimizing with a
 * dedicated tag.
 */

const noTypeMask = 0

// Lower mask types
const (
	numberTypeMask uint64 = 1 << iota
	signedNumberTypeMask
	integerTypeMask
	signedIntegerTypeMask
	unsignedIntegerTypeMask
	fixedPointTypeMask
	signedFixedPointTypeMask
	unsignedFixedPointTypeMask

	intTypeMask
	int8TypeMask
	int16TypeMask
	int32TypeMask
	int64TypeMask
	int128TypeMask
	int256TypeMask

	uintTypeMask
	uint8TypeMask
	uint16TypeMask
	uint32TypeMask
	uint64TypeMask
	uint128TypeMask
	uint256TypeMask

	word8TypeMask
	word16TypeMask
	word32TypeMask
	word64TypeMask

	_ // future: Fix8
	_ // future: Fix16
	_ // future: Fix32
	fix64TypeMask
	_ // future: Fix128
	_ // future: Fix256

	_ // future: UFix8
	_ // future: UFix16
	_ // future: UFix32
	ufix64TypeMask
	_ // future: UFix128
	_ // future: UFix256

	stringTypeMask
	characterTypeMask
	boolTypeMask
	nilTypeMask
	voidTypeMask
	addressTypeMask
	metaTypeMask
	blockTypeMask
	anyStructTypeMask
	anyResourceTypeMask
	anyTypeMask
	deployedContractMask
	neverTypeMask

	pathTypeMask
	storagePathTypeMask
	capabilityPathTypeMask
	publicPathTypeMask
	privatePathTypeMask

	constantSizedTypeMask
	variableSizedTypeMask
	dictionaryTypeMask
	compositeTypeMask
	referenceTypeMask
	genericTypeMask
	functionTypeMask
	interfaceTypeMask

	// ~~ NOTE: End of limit for lower mask type. Any new type should go to upper mask. ~~
)

// Upper mask types
const (
	capabilityTypeMask uint64 = 1 << iota
	restrictedTypeMask
	transactionTypeMask
	anyResourceAttachmentMask
	anyStructAttachmentMask

	invalidTypeMask
)

var (
	// NoTypeTag is a special tag to represent mask with no types included
	NoTypeTag = newTypeTagFromLowerMask(noTypeMask)

	SignedIntegerTypeTag = newTypeTagFromLowerMask(signedIntegerTypeMask).
				Or(IntTypeTag).
				Or(Int8TypeTag).
				Or(Int16TypeTag).
				Or(Int32TypeTag).
				Or(Int64TypeTag).
				Or(Int128TypeTag).
				Or(Int256TypeTag)

	UnsignedIntegerTypeTag = newTypeTagFromLowerMask(unsignedIntegerTypeMask).
				Or(UIntTypeTag).
				Or(UInt8TypeTag).
				Or(UInt16TypeTag).
				Or(UInt32TypeTag).
				Or(UInt64TypeTag).
				Or(UInt128TypeTag).
				Or(UInt256TypeTag).
				Or(Word8TypeTag).
				Or(Word16TypeTag).
				Or(Word32TypeTag).
				Or(Word64TypeTag)

	IntegerTypeTag = newTypeTagFromLowerMask(integerTypeMask).
			Or(SignedIntegerTypeTag).
			Or(UnsignedIntegerTypeTag)

	SignedFixedPointTypeTag = newTypeTagFromLowerMask(signedFixedPointTypeMask).
				Or(Fix64TypeTag)

	UnsignedFixedPointTypeTag = newTypeTagFromLowerMask(unsignedFixedPointTypeMask).
					Or(UFix64TypeTag)

	FixedPointTypeTag = newTypeTagFromLowerMask(fixedPointTypeMask).
				Or(SignedFixedPointTypeTag).
				Or(UnsignedFixedPointTypeTag)

	SignedNumberTypeTag = newTypeTagFromLowerMask(signedNumberTypeMask).
				Or(SignedIntegerTypeTag).
				Or(SignedFixedPointTypeTag)

	NumberTypeTag = newTypeTagFromLowerMask(numberTypeMask).
			Or(IntegerTypeTag).
			Or(FixedPointTypeTag).
			Or(SignedNumberTypeTag)

	UIntTypeTag    = newTypeTagFromLowerMask(uintTypeMask)
	UInt8TypeTag   = newTypeTagFromLowerMask(uint8TypeMask)
	UInt16TypeTag  = newTypeTagFromLowerMask(uint16TypeMask)
	UInt32TypeTag  = newTypeTagFromLowerMask(uint32TypeMask)
	UInt64TypeTag  = newTypeTagFromLowerMask(uint64TypeMask)
	UInt128TypeTag = newTypeTagFromLowerMask(uint128TypeMask)
	UInt256TypeTag = newTypeTagFromLowerMask(uint256TypeMask)

	IntTypeTag    = newTypeTagFromLowerMask(intTypeMask)
	Int8TypeTag   = newTypeTagFromLowerMask(int8TypeMask)
	Int16TypeTag  = newTypeTagFromLowerMask(int16TypeMask)
	Int32TypeTag  = newTypeTagFromLowerMask(int32TypeMask)
	Int64TypeTag  = newTypeTagFromLowerMask(int64TypeMask)
	Int128TypeTag = newTypeTagFromLowerMask(int128TypeMask)
	Int256TypeTag = newTypeTagFromLowerMask(int256TypeMask)

	Word8TypeTag  = newTypeTagFromLowerMask(word8TypeMask)
	Word16TypeTag = newTypeTagFromLowerMask(word16TypeMask)
	Word32TypeTag = newTypeTagFromLowerMask(word32TypeMask)
	Word64TypeTag = newTypeTagFromLowerMask(word64TypeMask)

	Fix64TypeTag  = newTypeTagFromLowerMask(fix64TypeMask)
	UFix64TypeTag = newTypeTagFromLowerMask(ufix64TypeMask)

	StringTypeTag           = newTypeTagFromLowerMask(stringTypeMask)
	CharacterTypeTag        = newTypeTagFromLowerMask(characterTypeMask)
	BoolTypeTag             = newTypeTagFromLowerMask(boolTypeMask)
	NilTypeTag              = newTypeTagFromLowerMask(nilTypeMask)
	VoidTypeTag             = newTypeTagFromLowerMask(voidTypeMask)
	AddressTypeTag          = newTypeTagFromLowerMask(addressTypeMask)
	MetaTypeTag             = newTypeTagFromLowerMask(metaTypeMask)
	NeverTypeTag            = newTypeTagFromLowerMask(neverTypeMask)
	BlockTypeTag            = newTypeTagFromLowerMask(blockTypeMask)
	DeployedContractTypeTag = newTypeTagFromLowerMask(deployedContractMask)

	StoragePathTypeTag = newTypeTagFromLowerMask(storagePathTypeMask)
	PublicPathTypeTag  = newTypeTagFromLowerMask(publicPathTypeMask)
	PrivatePathTypeTag = newTypeTagFromLowerMask(privatePathTypeMask)

	CapabilityPathTypeTag = newTypeTagFromLowerMask(capabilityPathTypeMask).
				Or(PublicPathTypeTag).
				Or(PrivatePathTypeTag)

	PathTypeTag = newTypeTagFromLowerMask(pathTypeMask).
			Or(CapabilityPathTypeTag).
			Or(StoragePathTypeTag)

	ConstantSizedTypeTag = newTypeTagFromLowerMask(constantSizedTypeMask)
	VariableSizedTypeTag = newTypeTagFromLowerMask(variableSizedTypeMask)
	DictionaryTypeTag    = newTypeTagFromLowerMask(dictionaryTypeMask)
	CompositeTypeTag     = newTypeTagFromLowerMask(compositeTypeMask)
	ReferenceTypeTag     = newTypeTagFromLowerMask(referenceTypeMask)
	GenericTypeTag       = newTypeTagFromLowerMask(genericTypeMask)
	FunctionTypeTag      = newTypeTagFromLowerMask(functionTypeMask)
	InterfaceTypeTag     = newTypeTagFromLowerMask(interfaceTypeMask)

	RestrictedTypeTag            = newTypeTagFromUpperMask(restrictedTypeMask)
	CapabilityTypeTag            = newTypeTagFromUpperMask(capabilityTypeMask)
	InvalidTypeTag               = newTypeTagFromUpperMask(invalidTypeMask)
	TransactionTypeTag           = newTypeTagFromUpperMask(transactionTypeMask)
	AnyResourceAttachmentTypeTag = newTypeTagFromUpperMask(anyResourceAttachmentMask)
	AnyStructAttachmentTypeTag   = newTypeTagFromUpperMask(anyStructAttachmentMask)

	// AnyStructTypeTag only includes the types that are pre-known
	// to belong to AnyStruct type. This is more of an optimization.
	// Other types (derived types such as collections, etc.) are not possible
	// to be included in the mask without knowing their member types.
	// Hence, they are checked on demand in `getSuperTypeOfDerivedTypes()`.
	AnyStructTypeTag = newTypeTagFromLowerMask(anyStructTypeMask).
				Or(AnyStructAttachmentTypeTag).
				Or(NeverTypeTag).
				Or(NumberTypeTag).
				Or(StringTypeTag).
				Or(ReferenceTypeTag).
				Or(NilTypeTag).
				Or(BoolTypeTag).
				Or(CharacterTypeTag).
				Or(VoidTypeTag).
				Or(MetaTypeTag).
				Or(PathTypeTag).
				Or(AddressTypeTag).
				Or(BlockTypeTag).
				Or(DeployedContractTypeTag).
				Or(CapabilityTypeTag).
				Or(FunctionTypeTag)

	AnyResourceTypeTag = newTypeTagFromLowerMask(anyResourceTypeMask).
				Or(AnyResourceAttachmentTypeTag)

	AnyTypeTag = newTypeTagFromLowerMask(anyTypeMask).
			Or(AnyStructTypeTag).
			Or(AnyResourceTypeTag).
			Or(ConstantSizedTypeTag).
			Or(VariableSizedTypeTag).
			Or(DictionaryTypeTag).
			Or(GenericTypeTag).
			Or(InterfaceTypeTag).
			Or(TransactionTypeTag).
			Or(RestrictedTypeTag)
)

// Methods

func LeastCommonSuperType(types ...Type) Type {

	superType := leastCommonSuperType(types...)
	if superType == InvalidType {
		return superType
	}

	// Do a sanity check to see if all types are
	// actually subtypes of the inferred supertype.
	for _, typ := range types {
		if !IsSubType(typ, superType) {
			return InvalidType
		}

	}

	return superType
}

func leastCommonSuperType(types ...Type) Type {
	join := NoTypeTag

	for _, typ := range types {
		join = join.Or(typ.Tag())
	}

	supertype := findCommonSuperType(join, types...)

	// NOTE: Important:
	// 'Any' is a valid inferred type, since it's the supertype of all types.
	// However, in the context of checker, 'Any' is not a possible type for expressions,
	// as it mixes value-kinded and resource-kinded types, leading to undefined behaviour.
	// Hence, return 'InvalidType'.
	if supertype == AnyType {
		return InvalidType
	}

	return supertype
}

var notNeverType = NeverTypeTag.Not()
var notNilType = NilTypeTag.Not()

func findCommonSuperType(joinedTypeTag TypeTag, types ...Type) Type {
	var superType Type

	if joinedTypeTag == NeverTypeTag {
		return NeverType
	}

	// Remove 'Never' type out of the way.
	// Because 'Never' is a subtype of any other type. So
	// finding super type for the rest of the types is sufficient.
	joinedTypeTag = joinedTypeTag.And(notNeverType)

	if joinedTypeTag == NoTypeTag {
		return InvalidType
	}

	// If both masks are on, then the types are heterogeneous.
	// So skip the optimization and find the supertype in the hard way.
	if joinedTypeTag.lowerMask != 0 && joinedTypeTag.upperMask == 0 {
		superType = findSuperTypeFromLowerMask(joinedTypeTag, types)
	} else if joinedTypeTag.lowerMask == 0 && joinedTypeTag.upperMask != 0 {
		superType = findSuperTypeFromUpperMask(joinedTypeTag, types)
	}

	if superType != nil {
		return superType
	}

	// Optional types.
	if joinedTypeTag.ContainsAny(NilTypeTag) {

		// Get the typeTag without the optional flag
		joinedTypeTag = joinedTypeTag.And(notNilType)

		// Get the types without the optionals
		unwrappedTypes, levels := unwrapOptionals(types)

		superType = findCommonSuperType(joinedTypeTag, unwrappedTypes...)

		// If the common supertype of the rest of types contain nil (e.g: AnyStruct),
		// then do not wrap with optional again.
		// NOTE: At this point, the `superType` cannot be an optional. Can only be AnyStruct, AnyResource, etc.
		// Hence, no need of re-wrapping.
		if superType.Tag().ContainsAny(NilTypeTag) {
			return superType
		}

		// Re-wrap the optionals to the same amount of levels.
		// Because supertype of `T`, `T?`, `T??` is `T??`.
		return wrapOptionals(superType, levels)
	}

	// NOTE: Below order is important!

	switch {
	case joinedTypeTag.ContainsAny(InvalidTypeTag):
		return InvalidType
	case joinedTypeTag.BelongsTo(SignedIntegerTypeTag):
		return SignedIntegerType
	case joinedTypeTag.BelongsTo(IntegerTypeTag):
		return IntegerType
	case joinedTypeTag.BelongsTo(SignedFixedPointTypeTag):
		return SignedFixedPointType
	case joinedTypeTag.BelongsTo(FixedPointTypeTag):
		return FixedPointType
	case joinedTypeTag.BelongsTo(SignedNumberTypeTag):
		return SignedNumberType
	case joinedTypeTag.BelongsTo(NumberTypeTag):
		return NumberType
	case joinedTypeTag.BelongsTo(CapabilityPathTypeTag):
		return CapabilityPathType
	case joinedTypeTag.BelongsTo(PathTypeTag):
		return PathType
	}

	// At this point, all the types are heterogeneous.
	// So the common supertype could only be one of:
	//    - AnyStruct
	//    - AnyResource
	//    - None (if there are both structs and resources)

	return commonSuperTypeOfHeterogeneousTypes(types)
}

func findSuperTypeFromLowerMask(joinedTypeTag TypeTag, types []Type) Type {
	switch joinedTypeTag.lowerMask {

	case numberTypeMask:
		return NumberType
	case signedNumberTypeMask:
		return SignedNumberType
	case integerTypeMask:
		return IntegerType
	case signedIntegerTypeMask:
		return SignedIntegerType
	case fixedPointTypeMask:
		return FixedPointType
	case signedFixedPointTypeMask:
		return SignedFixedPointType

	case intTypeMask:
		return IntType
	case int8TypeMask:
		return Int8Type
	case int16TypeMask:
		return Int16Type
	case int32TypeMask:
		return Int32Type
	case int64TypeMask:
		return Int64Type
	case int128TypeMask:
		return Int128Type
	case int256TypeMask:
		return Int256Type

	case uintTypeMask:
		return UIntType
	case uint8TypeMask:
		return UInt8Type
	case uint16TypeMask:
		return UInt16Type
	case uint32TypeMask:
		return UInt32Type
	case uint64TypeMask:
		return UInt64Type
	case uint128TypeMask:
		return UInt128Type
	case uint256TypeMask:
		return UInt256Type

	case word8TypeMask:
		return Word8Type
	case word16TypeMask:
		return Word16Type
	case word32TypeMask:
		return Word32Type
	case word64TypeMask:
		return Word64Type

	case fix64TypeMask:
		return Fix64Type
	case ufix64TypeMask:
		return UFix64Type

	case stringTypeMask:
		return StringType
	case nilTypeMask:
		return &OptionalType{
			Type: NeverType,
		}
	case neverTypeMask:
		return NeverType
	case characterTypeMask:
		return CharacterType
	case boolTypeMask:
		return BoolType
	case voidTypeMask:
		return VoidType
	case addressTypeMask:
		return TheAddressType
	case metaTypeMask:
		return MetaType
	case blockTypeMask:
		return BlockType
	case deployedContractMask:
		return DeployedContractType
	case pathTypeMask:
		return PathType
	case privatePathTypeMask:
		return PrivatePathType
	case publicPathTypeMask:
		return PublicPathType
	case storagePathTypeMask:
		return StoragePathType
	case capabilityPathTypeMask:
		return CapabilityPathType
	case anyStructTypeMask:
		return AnyStructType
	case anyResourceTypeMask:
		return AnyResourceType
	case anyTypeMask:
		return AnyType
	case noTypeMask:
		return InvalidType

	case compositeTypeMask:
		// We reach here if all are composite types.
		// Therefore, check for member types, and decide the
		// common supertype based on the member types.
		var prevType Type
		for _, typ := range types {
			// Ignore 'Never' type as it doesn't affect the supertype.
			if typ == NeverType {
				continue
			}

			if prevType == nil {
				prevType = typ
				continue
			}

			if !typ.Equal(prevType) {
				return commonSuperTypeOfComposites(types)
			}
		}

		return prevType

	// All derived types goes here.
	case constantSizedTypeMask:
		return commonSuperTypeOfConstantSizedArrays(types)
	case variableSizedTypeMask:
		return commonSuperTypeOfVariableSizedArrays(types)
	case dictionaryTypeMask:
		return commonSuperTypeOfDictionaries(types)
	case referenceTypeMask,
		genericTypeMask,
		functionTypeMask,
		interfaceTypeMask:

		return getSuperTypeOfDerivedTypes(types)
	default:
		// not homogenous. Return nil and continue on advanced checks.
		return nil
	}
}

func findSuperTypeFromUpperMask(joinedTypeTag TypeTag, types []Type) Type {
	switch joinedTypeTag.upperMask {

	case invalidTypeMask:
		return InvalidType

	// All derived types goes here.
	case capabilityTypeMask,
		restrictedTypeMask,
		transactionTypeMask:
		return getSuperTypeOfDerivedTypes(types)
	case anyResourceAttachmentMask:
		return AnyResourceAttachmentType
	case anyStructAttachmentMask:
		return AnyStructAttachmentType
	default:
		return nil
	}
}

func getSuperTypeOfDerivedTypes(types []Type) Type {
	// We reach here if all types belongs to same kind.
	// e.g: All are arrays, all are dictionaries, etc.
	// Therefore, check for member types, and decide the
	// common supertype based on the member types.
	var prevType Type
	for _, typ := range types {
		// 'Never' type doesn't affect the supertype.
		// Hence, ignore them
		if typ == NeverType {
			continue
		}

		if prevType == nil {
			prevType = typ
			continue
		}

		if !typ.Equal(prevType) {
			return commonSuperTypeOfHeterogeneousTypes(types)
		}
	}

	if prevType == nil {
		return InvalidType
	}

	return prevType
}

func commonSuperTypeOfVariableSizedArrays(types []Type) Type {
	// We reach here if all types are variable-sized arrays.
	// Therefore, decide the common supertype based on the element types.

	var elementTypes []Type

	for _, typ := range types {
		// 'Never' type doesn't affect the supertype.
		// Hence, ignore them
		if typ == NeverType {
			continue
		}

		arrayType, ok := typ.(*VariableSizedType)
		if !ok {
			panic(errors.NewUnexpectedError("expected variable-sized array type, found %s", typ))
		}

		elementTypes = append(elementTypes, arrayType.ElementType(false))
	}

	elementSuperType := leastCommonSuperType(elementTypes...)

	if elementSuperType == InvalidType {
		return InvalidType
	}

	return &VariableSizedType{
		Type: elementSuperType,
	}
}

func commonSuperTypeOfConstantSizedArrays(types []Type) Type {
	// We reach here if all types are constant-sized arrays.
	// Therefore, decide the common supertype based on the element types.

	var elementTypes []Type
	var prevType *ConstantSizedType

	for _, typ := range types {
		// 'Never' type doesn't affect the supertype.
		// Hence, ignore them
		if typ == NeverType {
			continue
		}

		arrayType, ok := typ.(*ConstantSizedType)
		if !ok {
			panic(errors.NewUnexpectedError("expected constant-sized array type, found %s", typ))
		}

		elementTypes = append(elementTypes, arrayType.ElementType(false))

		if prevType == nil {
			prevType = arrayType
			continue
		}

		// Arrays with different sizes are not covariant
		if arrayType.Size != prevType.Size {
			return commonSuperTypeOfHeterogeneousTypes(types)
		}
	}

	elementSuperType := leastCommonSuperType(elementTypes...)

	if elementSuperType == InvalidType {
		return InvalidType
	}

	return &ConstantSizedType{
		Type: elementSuperType,
		Size: prevType.Size,
	}
}

func commonSuperTypeOfDictionaries(types []Type) Type {
	// We reach here if all types are dictionary types.
	// Therefore, decide the common supertype based on the key types and value types.

	var keyTypes []Type
	var valueTypes []Type

	for _, typ := range types {
		// 'Never' type doesn't affect the supertype.
		// Hence, ignore them
		if typ == NeverType {
			continue
		}

		dictionaryType, ok := typ.(*DictionaryType)
		if !ok {
			panic(errors.NewUnexpectedError("expected dictionary type, found %s", typ))
		}

		valueTypes = append(valueTypes, dictionaryType.ValueType)
		keyTypes = append(keyTypes, dictionaryType.KeyType)
	}

	keySuperType := leastCommonSuperType(keyTypes...)
	valueSuperType := leastCommonSuperType(valueTypes...)

	if keySuperType == InvalidType ||
		valueSuperType == InvalidType {
		return InvalidType
	}

	if !IsValidDictionaryKeyType(keySuperType) {
		return commonSuperTypeOfHeterogeneousTypes(types)
	}

	return &DictionaryType{
		KeyType:   keySuperType,
		ValueType: valueSuperType,
	}
}

func commonSuperTypeOfHeterogeneousTypes(types []Type) Type {
	var hasStructs, hasResources bool
	for _, typ := range types {
		isResource := typ.IsResourceType()
		hasResources = hasResources || isResource
		hasStructs = hasStructs || !isResource

		if hasResources && hasStructs {
			return AnyType
		}
	}

	if hasResources {
		return AnyResourceType
	}

	return AnyStructType
}

func commonSuperTypeOfComposites(types []Type) Type {
	var hasStructs, hasResources bool

	commonInterfaces := map[string]bool{}
	commonInterfacesList := make([]*InterfaceType, 0)

	hasCommonInterface := true

	firstType := true

	for _, typ := range types {

		// Ignore 'Never' type as it doesn't affect the supertype.
		if typ == NeverType {
			continue
		}

		isResource := typ.IsResourceType()
		hasResources = hasResources || isResource
		hasStructs = hasStructs || !isResource

		if hasResources && hasStructs {
			// If the types has both structs and resources,
			// then there's no common super type.
			return AnyType
		}

		if !hasCommonInterface {
			break
		}

		compositeType, ok := typ.(*CompositeType)
		// this function is only called when all types are composites, so this cannot fail
		if !ok {
			panic(errors.NewUnreachableError())
		}

		// NOTE: index 0 may not always be the first type, since there can be 'Never' types.
		if firstType {
			for _, interfaceType := range compositeType.ExplicitInterfaceConformances {
				commonInterfaces[interfaceType.QualifiedIdentifier()] = true
				commonInterfacesList = append(commonInterfacesList, interfaceType)
			}
			firstType = false
		} else {
			intersection := map[string]bool{}
			commonInterfacesList = make([]*InterfaceType, 0)

			for _, interfaceType := range compositeType.ExplicitInterfaceConformances {
				if _, ok := commonInterfaces[interfaceType.QualifiedIdentifier()]; ok {
					intersection[interfaceType.QualifiedIdentifier()] = true
					commonInterfacesList = append(commonInterfacesList, interfaceType)
				}
			}

			commonInterfaces = intersection
		}

		if len(commonInterfaces) == 0 {
			hasCommonInterface = false
		}
	}

	var superType Type
	if hasResources {
		superType = AnyResourceType
	} else {
		superType = AnyStructType
	}

	if hasCommonInterface {
		return &RestrictedType{
			Type:         superType,
			Restrictions: commonInterfacesList,
		}
	}

	return superType
}

func unwrapOptionals(types []Type) ([]Type, int) {
	unwrappedTypes := make([]Type, 0, len(types))

	maxLevels := 0
	for _, typ := range types {
		levels := 0

		// Unwrap optionals
		for {
			optionalType, ok := typ.(*OptionalType)
			if !ok {
				break
			}

			typ = optionalType.Type
			levels++
		}

		maxLevels = max(maxLevels, levels)

		unwrappedTypes = append(unwrappedTypes, typ)
	}

	return unwrappedTypes, maxLevels
}

func wrapOptionals(typ Type, levels int) Type {
	for i := 0; i < levels; i++ {
		typ = &OptionalType{
			Type: typ,
		}
	}

	return typ
}

func max(a, b int) int {
	if a >= b {
		return a
	}

	return b
}
