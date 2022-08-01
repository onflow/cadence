/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2022 Dapper Labs, Inc.
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

package sema_codec

import (
	"bytes"
	"fmt"
	"io"
	"math/big"

	"github.com/onflow/cadence/encoding/custom/common_codec"
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/sema"
)

// A SemaEncoder converts Sema types into custom-encoded bytes.
type SemaEncoder struct {
	w        common_codec.LengthyWriter
	typeDefs map[sema.Type]int
}

// EncodeSema returns the custom-encoded representation of the given sema type.
//
// This function returns an error if the Cadence value cannot be represented in the custom format.
func EncodeSema(t sema.Type) ([]byte, error) {
	var w bytes.Buffer
	enc := NewSemaEncoder(&w)

	err := enc.Encode(t)
	if err != nil {
		return nil, err
	}

	return w.Bytes(), nil
}

// MustEncodeSema returns the custom-encoded representation of the given sema type, or panics
// if the sema type cannot be represented in the custom format.
func MustEncodeSema(value sema.Type) []byte {
	b, err := EncodeSema(value)
	if err != nil {
		panic(err)
	}
	return b
}

// NewSemaEncoder initializes a SemaEncoder that will write custom-encoded bytes to the
// given io.Writer.
func NewSemaEncoder(w io.Writer) *SemaEncoder {
	return &SemaEncoder{
		w:        common_codec.NewLengthyWriter(w),
		typeDefs: map[sema.Type]int{},
	}
}

// TODO include leading byte with version information
//      maybe include other metadata too, like the size the decoder's typeDefs map will be

// Encode writes the custom-encoded representation of the given sema type to this
// encoder's io.Writer.
//
// This function returns an error if the given sema type is not supported by this encoder.
func (e *SemaEncoder) Encode(t sema.Type) (err error) {
	return e.EncodeType(t)
}

// EncodeElaboration serializes the CompositeType and InterfaceType values in the Elaboration.
// The rest of the Elaboration is NOT serialized because they are not needed for encoding external values.
func (e *SemaEncoder) EncodeElaboration(el *sema.Elaboration) (err error) {
	err = EncodeMap(e, el.CompositeTypes, e.EncodeCompositeType)
	if err != nil {
		return
	}

	return EncodeMap(e, el.InterfaceTypes, e.EncodeInterfaceType)
}

// EncodeType encodes any supported sema.Type.
// Includes concrete type identifier because "Type" is an abstract type
// ergo it can't be instantiated on decode.
func (e *SemaEncoder) EncodeType(t sema.Type) (err error) {
	// Non-recursable types
	switch concreteType := t.(type) {
	case *sema.SimpleType:
		return e.EncodeSimpleType(concreteType)
	case *sema.NumericType:
		return e.EncodeNumericType(concreteType)
	case *sema.FixedPointNumericType:
		return e.EncodeFixedPointNumericType(concreteType)
	case *sema.AddressType:
		return e.EncodeTypeIdentifier(EncodedSemaAddressType)
	case nil:
		return e.EncodeTypeIdentifier(EncodedSemaNilType)
	}

	// Recursable types
	if bufferOffset, usePointer := e.typeDefs[t]; usePointer {
		return e.EncodePointer(bufferOffset)
	}
	e.typeDefs[t] = e.w.Len() + 1 // point to the encoded type, not its type identifier

	switch concreteType := t.(type) {
	case *sema.CompositeType:
		err = e.EncodeTypeIdentifier(EncodedSemaCompositeType)
		if err != nil {
			return
		}
		return e.EncodeCompositeType(concreteType)
	case *sema.InterfaceType:
		err = e.EncodeTypeIdentifier(EncodedSemaInterfaceType)
		if err != nil {
			return
		}
		return e.EncodeInterfaceType(concreteType)
	case *sema.GenericType:
		err = e.EncodeTypeIdentifier(EncodedSemaGenericType)
		if err != nil {
			return
		}
		return e.EncodeGenericType(concreteType)
	case *sema.FunctionType:
		err = e.EncodeTypeIdentifier(EncodedSemaFunctionType)
		if err != nil {
			return
		}
		return e.EncodeFunctionType(concreteType)
	case *sema.DictionaryType:
		err = e.EncodeTypeIdentifier(EncodedSemaDictionaryType)
		if err != nil {
			return
		}
		return e.EncodeDictionaryType(concreteType)
	case *sema.TransactionType:
		err = e.EncodeTypeIdentifier(EncodedSemaTransactionType)
		if err != nil {
			return
		}
		return e.EncodeTransactionType(concreteType)
	case *sema.RestrictedType:
		err = e.EncodeTypeIdentifier(EncodedSemaRestrictedType)
		if err != nil {
			return
		}
		return e.EncodeRestrictedType(concreteType)
	case *sema.VariableSizedType:
		err = e.EncodeTypeIdentifier(EncodedSemaVariableSizedType)
		if err != nil {
			return
		}
		return e.EncodeVariableSizedType(concreteType)
	case *sema.ConstantSizedType:
		err = e.EncodeTypeIdentifier(EncodedSemaConstantSizedType)
		if err != nil {
			return
		}
		return e.EncodeConstantSizedType(concreteType)
	case *sema.OptionalType:
		err = e.EncodeTypeIdentifier(EncodedSemaOptionalType)
		if err != nil {
			return
		}
		return e.EncodeOptionalType(concreteType)
	case *sema.ReferenceType:
		err = e.EncodeTypeIdentifier(EncodedSemaReferenceType)
		if err != nil {
			return
		}
		return e.EncodeReferenceType(concreteType)
	case *sema.CapabilityType:
		err = e.EncodeTypeIdentifier(EncodedSemaCapabilityType)
		if err != nil {
			return
		}
		return e.EncodeCapabilityType(concreteType)
	}

	return fmt.Errorf("unexpected type: ${t}")
}

func isSimpleType(encodedSema EncodedSema) bool {
	return encodedSema >= EncodedSemaSimpleTypeAnyType &&
		encodedSema <= EncodedSemaSimpleTypeVoidType
}

func isNumericType(encodedSema EncodedSema) bool {
	return encodedSema >= EncodedSemaNumericTypeNumberType &&
		encodedSema <= EncodedSemaNumericTypeSignedFixedPointType
}

func isFixedPointNumericType(encodedSema EncodedSema) bool {
	return encodedSema == EncodedSemaFix64Type || encodedSema == EncodedSemaUFix64Type
}

func (e *SemaEncoder) EncodeSimpleType(t *sema.SimpleType) (err error) {
	var subType EncodedSema

	switch t {
	case sema.AnyType:
		subType = EncodedSemaSimpleTypeAnyType
	case sema.AnyResourceType:
		subType = EncodedSemaSimpleTypeAnyResourceType
	case sema.AnyStructType:
		subType = EncodedSemaSimpleTypeAnyStructType
	case sema.BlockType:
		subType = EncodedSemaSimpleTypeBlockType
	case sema.BoolType:
		subType = EncodedSemaSimpleTypeBoolType
	case sema.CharacterType:
		subType = EncodedSemaSimpleTypeCharacterType
	case sema.DeployedContractType:
		subType = EncodedSemaSimpleTypeDeployedContractType
	case sema.InvalidType:
		subType = EncodedSemaSimpleTypeInvalidType
	case sema.MetaType:
		subType = EncodedSemaSimpleTypeMetaType
	case sema.NeverType:
		subType = EncodedSemaSimpleTypeNeverType
	case sema.PathType:
		subType = EncodedSemaSimpleTypePathType
	case sema.StoragePathType:
		subType = EncodedSemaSimpleTypeStoragePathType
	case sema.CapabilityPathType:
		subType = EncodedSemaSimpleTypeCapabilityPathType
	case sema.PublicPathType:
		subType = EncodedSemaSimpleTypePublicPathType
	case sema.PrivatePathType:
		subType = EncodedSemaSimpleTypePrivatePathType
	case sema.StorableType:
		subType = EncodedSemaSimpleTypeStorableType
	case sema.StringType:
		subType = EncodedSemaSimpleTypeStringType
	case sema.VoidType:
		subType = EncodedSemaSimpleTypeVoidType

	default:
		return fmt.Errorf("unknown simple type: %s", t)
	}

	return e.write([]byte{byte(subType)})
}

func (e *SemaEncoder) EncodeFunctionType(t *sema.FunctionType) (err error) {
	err = common_codec.EncodeBool(&e.w, t.IsConstructor)
	if err != nil {
		return
	}

	err = EncodeArray(e, t.TypeParameters, e.EncodeTypeParameter)
	if err != nil {
		return
	}

	err = EncodeArray(e, t.Parameters, e.EncodeParameter)
	if err != nil {
		return
	}

	err = e.EncodeTypeAnnotation(t.ReturnTypeAnnotation)
	if err != nil {
		return
	}

	err = e.EncodeIntPointer(t.RequiredArgumentCount)
	if err != nil {
		return
	}

	// TODO Is it OK that ArgumentExpressionCheck is omitted?
	//      I only see it set twice: AddressConversionFunctionType and NumberConversionFunctionType
	//      It is likely that these should be encoded as enums instead, since they are Cadence globals.

	return e.EncodeStringMemberOrderedMap(t.Members)
}

func (e *SemaEncoder) EncodeIntPointer(ptr *int) (err error) {
	if ptr == nil {
		return common_codec.EncodeBool(&e.w, true)
	}

	err = common_codec.EncodeBool(&e.w, false)
	if err != nil {
		return
	}

	return common_codec.EncodeNumber(&e.w, int64(*ptr))
}

func (e *SemaEncoder) EncodeDictionaryType(t *sema.DictionaryType) (err error) {
	err = e.EncodeType(t.KeyType)
	if err != nil {
		return
	}

	return e.EncodeType(t.ValueType)
}

func (e *SemaEncoder) EncodeReferenceType(t *sema.ReferenceType) (err error) {
	err = common_codec.EncodeBool(&e.w, t.Authorized)
	if err != nil {
		return
	}

	return e.EncodeType(t.Type)
}

func (e *SemaEncoder) EncodeTransactionType(t *sema.TransactionType) (err error) {
	err = e.EncodeStringMemberOrderedMap(t.Members)
	if err != nil {
		return
	}

	err = EncodeArray(e, t.Fields, func(s string) error {
		return common_codec.EncodeString(&e.w, s)
	})
	if err != nil {
		return
	}

	err = EncodeArray(e, t.PrepareParameters, e.EncodeParameter)
	if err != nil {
		return
	}

	return EncodeArray(e, t.Parameters, e.EncodeParameter)
}

func (e *SemaEncoder) EncodeRestrictedType(t *sema.RestrictedType) (err error) {
	err = e.EncodeType(t.Type)
	if err != nil {
		return
	}

	return EncodeArray(e, t.Restrictions, e.EncodeInterfaceType)
}

func (e *SemaEncoder) EncodeCapabilityType(t *sema.CapabilityType) (err error) {
	return e.EncodeType(t.BorrowType)
}

func (e *SemaEncoder) EncodeOptionalType(t *sema.OptionalType) (err error) {
	return e.EncodeType(t.Type)
}

func (e *SemaEncoder) EncodeVariableSizedType(t *sema.VariableSizedType) (err error) {
	return e.EncodeType(t.Type)
}

func (e *SemaEncoder) EncodeConstantSizedType(t *sema.ConstantSizedType) (err error) {
	err = e.EncodeType(t.Type)
	if err != nil {
		return
	}

	return common_codec.EncodeNumber(&e.w, t.Size)
}

func (e *SemaEncoder) EncodeGenericType(t *sema.GenericType) (err error) {
	return e.EncodeTypeParameter(t.TypeParameter)
}

func (e *SemaEncoder) EncodeNumericType(t *sema.NumericType) (err error) {
	var numericType EncodedSema

	switch t {
	case sema.NumberType:
		numericType = EncodedSemaNumericTypeNumberType
	case sema.SignedNumberType:
		numericType = EncodedSemaNumericTypeSignedNumberType
	case sema.IntegerType:
		numericType = EncodedSemaNumericTypeIntegerType
	case sema.SignedIntegerType:
		numericType = EncodedSemaNumericTypeSignedIntegerType
	case sema.IntType:
		numericType = EncodedSemaNumericTypeIntType
	case sema.Int8Type:
		numericType = EncodedSemaNumericTypeInt8Type
	case sema.Int16Type:
		numericType = EncodedSemaNumericTypeInt16Type
	case sema.Int32Type:
		numericType = EncodedSemaNumericTypeInt32Type
	case sema.Int64Type:
		numericType = EncodedSemaNumericTypeInt64Type
	case sema.Int128Type:
		numericType = EncodedSemaNumericTypeInt128Type
	case sema.Int256Type:
		numericType = EncodedSemaNumericTypeInt256Type
	case sema.UIntType:
		numericType = EncodedSemaNumericTypeUIntType
	case sema.UInt8Type:
		numericType = EncodedSemaNumericTypeUInt8Type
	case sema.UInt16Type:
		numericType = EncodedSemaNumericTypeUInt16Type
	case sema.UInt32Type:
		numericType = EncodedSemaNumericTypeUInt32Type
	case sema.UInt64Type:
		numericType = EncodedSemaNumericTypeUInt64Type
	case sema.UInt128Type:
		numericType = EncodedSemaNumericTypeUInt128Type
	case sema.UInt256Type:
		numericType = EncodedSemaNumericTypeUInt256Type
	case sema.Word8Type:
		numericType = EncodedSemaNumericTypeWord8Type
	case sema.Word16Type:
		numericType = EncodedSemaNumericTypeWord16Type
	case sema.Word32Type:
		numericType = EncodedSemaNumericTypeWord32Type
	case sema.Word64Type:
		numericType = EncodedSemaNumericTypeWord64Type
	case sema.FixedPointType:
		numericType = EncodedSemaNumericTypeFixedPointType
	case sema.SignedFixedPointType:
		numericType = EncodedSemaNumericTypeSignedFixedPointType
	default:
		return fmt.Errorf("unexpected numeric type: %s", t)
	}

	return e.EncodeTypeIdentifier(numericType)
}

func (e *SemaEncoder) EncodeFixedPointNumericType(t *sema.FixedPointNumericType) (err error) {
	var fixedPointNumericType EncodedSema

	switch t {
	case sema.Fix64Type:
		fixedPointNumericType = EncodedSemaFix64Type
	case sema.UFix64Type:
		fixedPointNumericType = EncodedSemaUFix64Type
	default:
		return fmt.Errorf("unexpected fixed point numeric type: %s", t)
	}

	return e.write([]byte{byte(fixedPointNumericType)})
}

func (e *SemaEncoder) EncodeBigInt(bi *big.Int) (err error) {
	sign := bi.Sign()
	neg := sign == -1
	err = common_codec.EncodeBool(&e.w, neg)
	if err != nil {
		return
	}

	return common_codec.EncodeBytes(&e.w, bi.Bytes())
}

func (e *SemaEncoder) EncodeTypeIdentifier(id EncodedSema) (err error) {
	return e.write([]byte{byte(id)})
}

// EncodeCompositeKind expects the CompositeKind to fit within a single byte.
func (e *SemaEncoder) EncodeCompositeKind(kind common.CompositeKind) (err error) {
	return e.write([]byte{byte(kind)})
}

func (e *SemaEncoder) EncodePointer(bufferOffset int) (err error) {
	err = e.write([]byte{byte(EncodedSemaPointerType)})
	if err != nil {
		return
	}

	return common_codec.EncodeLength(&e.w, bufferOffset)
}

// TODO encode built-in CompositeTypes as enums
// TODO are composite types encodable is CompositeType.IsStorable() is false?
// TODO if IsImportable is false then do we want to skip for execution state storage?
func (e *SemaEncoder) EncodeCompositeType(compositeType *sema.CompositeType) (err error) {
	// Location -> common.Location
	err = common_codec.EncodeLocation(&e.w, compositeType.Location)
	if err != nil {
		return
	}

	// Identifier -> string
	err = common_codec.EncodeString(&e.w, compositeType.Identifier)
	if err != nil {
		return
	}

	// Kind -> common.CompositeKind
	err = e.EncodeCompositeKind(compositeType.Kind)
	if err != nil {
		return
	}

	// TODO does this handle recursive types correctly?
	// ExplicitInterfaceConformances -> []*InterfaceType
	err = EncodeArray(e, compositeType.ExplicitInterfaceConformances, e.EncodeInterfaceType)
	if err != nil {
		return
	}

	// TODO does this handle recursive types correctly?
	// ImplicitTypeRequirementConformances -> []*CompositeType
	err = EncodeArray(e, compositeType.ImplicitTypeRequirementConformances, e.EncodeCompositeType)
	if err != nil {
		return
	}

	// Members -> *StringMemberOrderedMap
	err = e.EncodeStringMemberOrderedMap(compositeType.Members)
	if err != nil {
		return
	}

	// Fields -> []string
	err = EncodeArray(e, compositeType.Fields, func(s string) error {
		return common_codec.EncodeString(&e.w, s)
	})
	if err != nil {
		return
	}

	// ConstructorParameters -> []*Parameter
	err = EncodeArray(e, compositeType.ConstructorParameters, e.EncodeParameter)
	if err != nil {
		return
	}

	// nestedTypes -> *StringTypeOrderedMap
	err = e.EncodeStringTypeOrderedMap(compositeType.GetNestedTypes())
	if err != nil {
		return
	}

	// containerType -> Type
	err = e.EncodeType(compositeType.GetContainerType())
	if err != nil {
		return
	}

	// EnumRawType -> Type
	err = e.EncodeType(compositeType.EnumRawType)
	if err != nil {
		return
	}

	// hasComputedMembers -> bool
	err = common_codec.EncodeBool(&e.w, compositeType.HasComputedMembers())
	if err != nil {
		return
	}
	// ImportableWithoutLocation -> bool
	return common_codec.EncodeBool(&e.w, compositeType.ImportableWithoutLocation)

}

func (e *SemaEncoder) EncodeTypeParameter(p *sema.TypeParameter) (err error) {
	err = common_codec.EncodeString(&e.w, p.Name)
	if err != nil {
		return
	}

	err = e.EncodeType(p.TypeBound)
	if err != nil {
		return
	}

	return common_codec.EncodeBool(&e.w, p.Optional)
}

func (e *SemaEncoder) EncodeParameter(parameter *sema.Parameter) (err error) {
	err = common_codec.EncodeString(&e.w, parameter.Label)
	if err != nil {
		return
	}

	err = common_codec.EncodeString(&e.w, parameter.Identifier)
	if err != nil {
		return
	}

	return e.EncodeTypeAnnotation(parameter.TypeAnnotation)
}

func (e *SemaEncoder) EncodeStringMemberOrderedMap(om *sema.StringMemberOrderedMap) (err error) {
	// TODO save a bit in the length for nil check?
	err = common_codec.EncodeBool(&e.w, om == nil)
	if om == nil || err != nil {
		return
	}

	type StringMemberTuple struct {
		String string
		Member *sema.Member
	}

	serializables := make([]StringMemberTuple, 0, om.Len())

	om.Foreach(func(key string, value *sema.Member) {
		if value.IsStorable(make(map[*sema.Member]bool)) {
			serializables = append(serializables, StringMemberTuple{
				String: key,
				Member: value,
			})
		}
	})
	err = common_codec.EncodeLength(&e.w, len(serializables))
	if err != nil {
		return
	}

	for _, tuple := range serializables {
		err = common_codec.EncodeString(&e.w, tuple.String)
		if err != nil {
			return err
		}

		err = e.EncodeMember(tuple.Member)
		if err != nil {
			return err
		}
	}

	return
}

func (e *SemaEncoder) EncodeStringTypeOrderedMap(om *sema.StringTypeOrderedMap) (err error) {
	// TODO save a bit in the length for nil check?
	err = common_codec.EncodeBool(&e.w, om == nil)
	if om == nil || err != nil {
		return
	}

	type StringTypeTuple struct {
		String string
		Type   sema.Type
	}

	serializables := make([]StringTypeTuple, 0, om.Len())

	om.Foreach(func(key string, value sema.Type) {
		if value.IsStorable(make(map[*sema.Member]bool)) {
			serializables = append(serializables, StringTypeTuple{
				String: key,
				Type:   value,
			})
		}
	})
	err = common_codec.EncodeLength(&e.w, len(serializables))
	if err != nil {
		return
	}

	for _, tuple := range serializables {
		err = common_codec.EncodeString(&e.w, tuple.String)
		if err != nil {
			return err
		}

		err = e.EncodeType(tuple.Type)
		if err != nil {
			return err
		}
	}

	return
}

func (e *SemaEncoder) EncodeMember(member *sema.Member) (err error) {
	err = common_codec.EncodeNumber(&e.w, uint64(member.Access))
	if err != nil {
		return
	}

	err = e.EncodeAstIdentifier(member.Identifier)
	if err != nil {
		return
	}

	err = e.EncodeTypeAnnotation(member.TypeAnnotation)
	if err != nil {
		return
	}

	err = common_codec.EncodeNumber(&e.w, uint64(member.DeclarationKind))
	if err != nil {
		return
	}

	err = common_codec.EncodeNumber(&e.w, uint64(member.VariableKind))
	if err != nil {
		return
	}

	err = EncodeArray(e, member.ArgumentLabels, func(s string) error {
		return common_codec.EncodeString(&e.w, s)
	})
	if err != nil {
		return
	}

	err = common_codec.EncodeBool(&e.w, member.Predeclared)
	if err != nil {
		return
	}

	return common_codec.EncodeString(&e.w, member.DocString)
}

func (e *SemaEncoder) EncodeTypeAnnotation(anno *sema.TypeAnnotation) (err error) {
	err = common_codec.EncodeBool(&e.w, anno == nil)
	if anno == nil || err != nil {
		return
	}

	err = common_codec.EncodeBool(&e.w, anno.IsResource)
	if err != nil {
		return
	}

	return e.EncodeType(anno.Type)
}

func (e *SemaEncoder) EncodeAstIdentifier(id ast.Identifier) (err error) {
	err = common_codec.EncodeString(&e.w, id.Identifier)
	if err != nil {
		return
	}

	return e.EncodeAstPosition(id.Pos)
}

func (e *SemaEncoder) EncodeAstPosition(pos ast.Position) (err error) {
	err = common_codec.EncodeNumber(&e.w, int64(pos.Offset))
	if err != nil {
		return
	}

	err = common_codec.EncodeNumber(&e.w, int64(pos.Line))
	if err != nil {
		return
	}

	return common_codec.EncodeNumber(&e.w, int64(pos.Column))
}

func (e *SemaEncoder) EncodeInterfaceType(interfaceType *sema.InterfaceType) (err error) {
	err = common_codec.EncodeLocation(&e.w, interfaceType.Location)
	if err != nil {
		return
	}

	err = common_codec.EncodeString(&e.w, interfaceType.Identifier)
	if err != nil {
		return
	}

	err = e.EncodeCompositeKind(interfaceType.CompositeKind)
	if err != nil {
		return
	}

	err = e.EncodeStringMemberOrderedMap(interfaceType.Members)
	if err != nil {
		return
	}

	err = EncodeArray(e, interfaceType.Fields, func(s string) error {
		return common_codec.EncodeString(&e.w, s)
	})
	if err != nil {
		return
	}

	err = EncodeArray(e, interfaceType.InitializerParameters, e.EncodeParameter)
	if err != nil {
		return
	}

	err = e.EncodeType(interfaceType.GetContainerType())
	if err != nil {
		return
	}

	// TODO can I drop nested types if I encode built-in composite types as enums?
	return e.EncodeStringTypeOrderedMap(interfaceType.GetNestedTypes())
}

func (e *SemaEncoder) write(b []byte) (err error) {
	_, err = e.w.Write(b)
	return
}

func EncodeArray[T any](e *SemaEncoder, arr []T, encodeFn func(T) error) (err error) {
	// TODO save a bit in the array length for nil check?
	err = common_codec.EncodeBool(&e.w, arr == nil)
	if arr == nil || err != nil {
		return
	}

	err = common_codec.EncodeLength(&e.w, len(arr))
	if err != nil {
		return
	}

	for _, element := range arr {
		// TODO does this need to include pointer logic for recursive types in arrays to be handled correctly?
		err = encodeFn(element)
		if err != nil {
			return
		}
	}

	return
}

// EncodeMap serializes a map from TypeID to sema.Type.
func EncodeMap[V sema.Type](e *SemaEncoder, m map[common.TypeID]V, encodeFn func(V) error) (err error) {
	// ASSUMPTION: map is never `nil`. WHY: EncodeMap only used for Elaboration, which is always fully instantiated.

	err = common_codec.EncodeLength(&e.w, len(m))
	if err != nil {
		return
	}

	// The order of encoded key-value pairs does not matter so long as we don't rely on hashing encoded Elaborations.
	for k, v := range m { //nolint:maprangecheck
		err = common_codec.EncodeString(&e.w, string(k))
		if err != nil {
			return
		}

		if bufferOffset, usePointer := e.typeDefs[v]; usePointer {
			err = e.EncodePointer(bufferOffset)
			if err != nil {
				return
			}
			continue
		}
		e.typeDefs[v] = e.w.Len()

		err = encodeFn(v)
		if err != nil {
			return
		}
	}

	return
}
