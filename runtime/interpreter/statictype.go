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

package interpreter

import (
	"fmt"
	"strings"

	"github.com/fxamacker/cbor/v2"
	"github.com/onflow/atree"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/sema"
)

const UnknownElementSize = 0

// StaticType is a shallow representation of a static type (`sema.Type`)
// which doesn't contain the full information, but only refers
// to composite and interface types by ID.
//
// This allows static types to be efficiently serialized and deserialized,
// for example in the world state.
//
type StaticType interface {
	fmt.Stringer
	isStaticType()
	/* this returns the size (in bytes) of the largest inhabitant of this type,
	or UnknownElementSize if the largest inhabitant has arbitrary size */
	elementSize() uint
	Equal(other StaticType) bool
	Encode(e *cbor.StreamEncoder) error
}

// CompositeStaticType

type CompositeStaticType struct {
	Location            common.Location
	QualifiedIdentifier string
	TypeID              common.TypeID
}

var _ StaticType = CompositeStaticType{}

func NewCompositeStaticType(
	memoryGauge common.MemoryGauge,
	location common.Location,
	qualifiedIdentifier string,
	typeID common.TypeID,
) CompositeStaticType {
	common.UseConstantMemory(memoryGauge, common.MemoryKindCompositeStaticType)

	return CompositeStaticType{
		Location:            location,
		QualifiedIdentifier: qualifiedIdentifier,
		TypeID:              typeID,
	}
}

func NewCompositeStaticTypeComputeTypeID(
	memoryGauge common.MemoryGauge,
	location common.Location,
	qualifiedIdentifier string,
) CompositeStaticType {
	// TODO compute memory usage before building typeID string
	typeID := common.NewTypeIDFromQualifiedName(location, qualifiedIdentifier)

	return NewCompositeStaticType(memoryGauge, location, qualifiedIdentifier, typeID)
}

func (CompositeStaticType) isStaticType() {}

func (CompositeStaticType) elementSize() uint {
	return UnknownElementSize
}

func (t CompositeStaticType) String() string {
	if t.Location == nil {
		return t.QualifiedIdentifier
	}
	return string(t.TypeID)
}

func (t CompositeStaticType) Equal(other StaticType) bool {
	otherCompositeType, ok := other.(CompositeStaticType)
	if !ok {
		return false
	}

	return otherCompositeType.TypeID == t.TypeID
}

// InterfaceStaticType

type InterfaceStaticType struct {
	Location            common.Location
	QualifiedIdentifier string
}

var _ StaticType = InterfaceStaticType{}

func NewInterfaceStaticType(
	memoryGauge common.MemoryGauge,
	location common.Location,
	qualifiedIdentifier string,
) InterfaceStaticType {
	common.UseConstantMemory(memoryGauge, common.MemoryKindInterfaceStaticType)

	return InterfaceStaticType{
		Location:            location,
		QualifiedIdentifier: qualifiedIdentifier,
	}
}

func (InterfaceStaticType) isStaticType() {}

func (InterfaceStaticType) elementSize() uint {
	return UnknownElementSize
}

func (t InterfaceStaticType) String() string {
	if t.Location == nil {
		return t.QualifiedIdentifier
	}
	return string(t.Location.TypeID(nil, t.QualifiedIdentifier))
}

func (t InterfaceStaticType) Equal(other StaticType) bool {
	otherInterfaceType, ok := other.(InterfaceStaticType)
	if !ok {
		return false
	}

	return common.LocationsMatch(otherInterfaceType.Location, t.Location) &&
		otherInterfaceType.QualifiedIdentifier == t.QualifiedIdentifier
}

// ArrayStaticType

type ArrayStaticType interface {
	StaticType
	isArrayStaticType()
	ElementType() StaticType
}

// VariableSizedStaticType

type VariableSizedStaticType struct {
	Type StaticType
}

var _ ArrayStaticType = VariableSizedStaticType{}
var _ atree.TypeInfo = VariableSizedStaticType{}

func NewVariableSizedStaticType(
	memoryGauge common.MemoryGauge,
	elementType StaticType,
) VariableSizedStaticType {
	common.UseConstantMemory(memoryGauge, common.MemoryKindVariableSizedStaticType)

	return VariableSizedStaticType{
		Type: elementType,
	}
}

func (VariableSizedStaticType) isStaticType() {}

func (VariableSizedStaticType) elementSize() uint {
	return UnknownElementSize
}

func (VariableSizedStaticType) isArrayStaticType() {}

func (t VariableSizedStaticType) ElementType() StaticType {
	return t.Type
}

func (t VariableSizedStaticType) String() string {
	return fmt.Sprintf("[%s]", t.Type)
}

func (t VariableSizedStaticType) Equal(other StaticType) bool {
	otherVariableSizedType, ok := other.(VariableSizedStaticType)
	if !ok {
		return false
	}

	return t.Type.Equal(otherVariableSizedType.Type)
}

// ConstantSizedStaticType

type ConstantSizedStaticType struct {
	Type StaticType
	Size int64
}

var _ ArrayStaticType = ConstantSizedStaticType{}
var _ atree.TypeInfo = ConstantSizedStaticType{}

func NewConstantSizedStaticType(
	memoryGauge common.MemoryGauge,
	elementType StaticType,
	size int64,
) ConstantSizedStaticType {
	common.UseConstantMemory(memoryGauge, common.MemoryKindConstantSizedStaticType)

	return ConstantSizedStaticType{
		Type: elementType,
		Size: size,
	}
}

func (ConstantSizedStaticType) isStaticType() {}

func (ConstantSizedStaticType) elementSize() uint {
	return UnknownElementSize
}

func (ConstantSizedStaticType) isArrayStaticType() {}

func (t ConstantSizedStaticType) ElementType() StaticType {
	return t.Type
}

func (t ConstantSizedStaticType) String() string {
	return fmt.Sprintf("[%s; %d]", t.Type, t.Size)
}

func (t ConstantSizedStaticType) Equal(other StaticType) bool {
	otherConstantSizedType, ok := other.(ConstantSizedStaticType)
	if !ok {
		return false
	}

	return t.Size == otherConstantSizedType.Size &&
		t.Type.Equal(otherConstantSizedType.Type)
}

// DictionaryStaticType

type DictionaryStaticType struct {
	KeyType   StaticType
	ValueType StaticType
}

var _ StaticType = DictionaryStaticType{}
var _ atree.TypeInfo = DictionaryStaticType{}

func NewDictionaryStaticType(
	memoryGauge common.MemoryGauge,
	keyType, valueType StaticType,
) DictionaryStaticType {
	common.UseConstantMemory(memoryGauge, common.MemoryKindDictionaryStaticType)

	return DictionaryStaticType{
		KeyType:   keyType,
		ValueType: valueType,
	}
}

func (DictionaryStaticType) isStaticType() {}

func (DictionaryStaticType) elementSize() uint {
	return UnknownElementSize
}

func (t DictionaryStaticType) String() string {
	return fmt.Sprintf("{%s: %s}", t.KeyType, t.ValueType)
}

func (t DictionaryStaticType) Equal(other StaticType) bool {
	otherDictionaryType, ok := other.(DictionaryStaticType)
	if !ok {
		return false
	}

	return t.KeyType.Equal(otherDictionaryType.KeyType) &&
		t.ValueType.Equal(otherDictionaryType.ValueType)
}

// OptionalStaticType

type OptionalStaticType struct {
	Type StaticType
}

var _ StaticType = OptionalStaticType{}

func NewOptionalStaticType(
	memoryGauge common.MemoryGauge,
	typ StaticType,
) OptionalStaticType {
	common.UseConstantMemory(memoryGauge, common.MemoryKindOptionalStaticType)

	return OptionalStaticType{Type: typ}
}

func (OptionalStaticType) isStaticType() {}

func (OptionalStaticType) elementSize() uint {
	return UnknownElementSize
}

func (t OptionalStaticType) String() string {
	return fmt.Sprintf("%s?", t.Type)
}

func (t OptionalStaticType) Equal(other StaticType) bool {
	otherOptionalType, ok := other.(OptionalStaticType)
	if !ok {
		return false
	}

	return t.Type.Equal(otherOptionalType.Type)
}

// RestrictedStaticType

type RestrictedStaticType struct {
	Type         StaticType
	Restrictions []InterfaceStaticType
}

var _ StaticType = &RestrictedStaticType{}

func NewRestrictedStaticType(
	memoryGauge common.MemoryGauge,
	staticType StaticType,
	restrictions []InterfaceStaticType,
) *RestrictedStaticType {
	common.UseConstantMemory(memoryGauge, common.MemoryKindRestrictedStaticType)

	return &RestrictedStaticType{
		Type:         staticType,
		Restrictions: restrictions,
	}
}

// NOTE: must be pointer receiver, as static types get used in type values,
// which are used as keys in maps when exporting.
// Key types in Go maps must be (transitively) hashable types,
// and slices are not, but `Restrictions` is one.
//
func (*RestrictedStaticType) isStaticType() {}

func (RestrictedStaticType) elementSize() uint {
	return UnknownElementSize
}

func (t *RestrictedStaticType) String() string {
	restrictions := make([]string, len(t.Restrictions))

	for i, restriction := range t.Restrictions {
		restrictions[i] = restriction.String()
	}

	return fmt.Sprintf("%s{%s}", t.Type, strings.Join(restrictions, ", "))
}

func (t *RestrictedStaticType) Equal(other StaticType) bool {
	otherRestrictedType, ok := other.(*RestrictedStaticType)
	if !ok || len(t.Restrictions) != len(otherRestrictedType.Restrictions) {
		return false
	}

outer:
	for _, restriction := range t.Restrictions {
		for _, otherRestriction := range otherRestrictedType.Restrictions {
			if restriction.Equal(otherRestriction) {
				continue outer
			}
		}

		return false
	}

	return t.Type.Equal(otherRestrictedType.Type)
}

// ReferenceStaticType

type ReferenceStaticType struct {
	Authorized     bool
	BorrowedType   StaticType
	ReferencedType StaticType
}

var _ StaticType = ReferenceStaticType{}

func NewReferenceStaticType(
	memoryGauge common.MemoryGauge,
	authorized bool,
	staticType StaticType,
	referenceType StaticType,
) ReferenceStaticType {
	common.UseConstantMemory(memoryGauge, common.MemoryKindReferenceStaticType)

	return ReferenceStaticType{
		Authorized:     authorized,
		BorrowedType:   staticType,
		ReferencedType: referenceType,
	}
}

func (ReferenceStaticType) isStaticType() {}

func (ReferenceStaticType) elementSize() uint {
	return UnknownElementSize
}

func (t ReferenceStaticType) String() string {
	auth := ""
	if t.Authorized {
		auth = "auth "
	}

	return fmt.Sprintf("%s&%s", auth, t.BorrowedType)
}

func (t ReferenceStaticType) Equal(other StaticType) bool {
	otherReferenceType, ok := other.(ReferenceStaticType)
	if !ok {
		return false
	}

	return t.Authorized == otherReferenceType.Authorized &&
		t.BorrowedType.Equal(otherReferenceType.BorrowedType)
}

// CapabilityStaticType

type CapabilityStaticType struct {
	BorrowType StaticType
}

var _ StaticType = CapabilityStaticType{}

func NewCapabilityStaticType(
	memoryGauge common.MemoryGauge,
	borrowType StaticType,
) CapabilityStaticType {
	common.UseConstantMemory(memoryGauge, common.MemoryKindCapabilityStaticType)

	return CapabilityStaticType{
		BorrowType: borrowType,
	}
}

func (CapabilityStaticType) isStaticType() {}

func (CapabilityStaticType) elementSize() uint {
	return UnknownElementSize
}

func (t CapabilityStaticType) String() string {
	if t.BorrowType != nil {
		return fmt.Sprintf("Capability<%s>", t.BorrowType)
	}
	return "Capability"
}

func (t CapabilityStaticType) Equal(other StaticType) bool {
	otherCapabilityType, ok := other.(CapabilityStaticType)
	if !ok {
		return false
	}

	// The borrow types must either be both nil,
	// or they must be equal

	if t.BorrowType == nil {
		return otherCapabilityType.BorrowType == nil
	}

	return t.BorrowType.Equal(otherCapabilityType.BorrowType)
}

// Conversion

func ConvertSemaToStaticType(memoryGauge common.MemoryGauge, t sema.Type) StaticType {
	switch t := t.(type) {
	case *sema.CompositeType:
		return NewCompositeStaticType(memoryGauge, t.Location, t.QualifiedIdentifier(), t.ID())

	case *sema.InterfaceType:
		return ConvertSemaInterfaceTypeToStaticInterfaceType(memoryGauge, t)

	case sema.ArrayType:
		return ConvertSemaArrayTypeToStaticArrayType(memoryGauge, t)

	case *sema.DictionaryType:
		return ConvertSemaDictionaryTypeToStaticDictionaryType(memoryGauge, t)

	case *sema.OptionalType:
		return NewOptionalStaticType(
			memoryGauge,
			ConvertSemaToStaticType(memoryGauge, t.Type),
		)

	case *sema.RestrictedType:
		restrictions := make([]InterfaceStaticType, len(t.Restrictions))

		for i, restriction := range t.Restrictions {
			restrictions[i] = ConvertSemaInterfaceTypeToStaticInterfaceType(memoryGauge, restriction)
		}

		return NewRestrictedStaticType(
			memoryGauge,
			ConvertSemaToStaticType(memoryGauge, t.Type),
			restrictions,
		)

	case *sema.ReferenceType:
		return ConvertSemaReferenceTypeToStaticReferenceType(memoryGauge, t)

	case *sema.CapabilityType:
		var borrowType StaticType
		if t.BorrowType != nil {
			borrowType = ConvertSemaToStaticType(memoryGauge, t.BorrowType)
		}
		return NewCapabilityStaticType(memoryGauge, borrowType)

	case *sema.FunctionType:
		return NewFunctionStaticType(memoryGauge, t)
	}

	primitiveStaticType := ConvertSemaToPrimitiveStaticType(memoryGauge, t)
	if primitiveStaticType == PrimitiveStaticTypeUnknown {
		return nil
	}
	return primitiveStaticType
}

func ConvertSemaArrayTypeToStaticArrayType(
	memoryGauge common.MemoryGauge,
	t sema.ArrayType,
) ArrayStaticType {
	switch t := t.(type) {
	case *sema.VariableSizedType:
		return VariableSizedStaticType{
			Type: ConvertSemaToStaticType(memoryGauge, t.Type),
		}

	case *sema.ConstantSizedType:
		return ConstantSizedStaticType{
			Type: ConvertSemaToStaticType(memoryGauge, t.Type),
			Size: t.Size,
		}

	default:
		panic(errors.NewUnreachableError())
	}
}

func ConvertSemaDictionaryTypeToStaticDictionaryType(
	memoryGauge common.MemoryGauge,
	t *sema.DictionaryType,
) DictionaryStaticType {
	return NewDictionaryStaticType(
		memoryGauge,
		ConvertSemaToStaticType(memoryGauge, t.KeyType),
		ConvertSemaToStaticType(memoryGauge, t.ValueType),
	)
}

func ConvertSemaReferenceTypeToStaticReferenceType(
	memoryGauge common.MemoryGauge,
	t *sema.ReferenceType,
) ReferenceStaticType {
	return NewReferenceStaticType(
		memoryGauge,
		t.Authorized,
		ConvertSemaToStaticType(memoryGauge, t.Type),
		nil,
	)
}

func ConvertSemaInterfaceTypeToStaticInterfaceType(
	memoryGauge common.MemoryGauge,
	t *sema.InterfaceType,
) InterfaceStaticType {
	return NewInterfaceStaticType(memoryGauge, t.Location, t.QualifiedIdentifier())
}

func ConvertStaticToSemaType(
	typ StaticType,
	getInterface func(location common.Location, qualifiedIdentifier string) (*sema.InterfaceType, error),
	getComposite func(location common.Location, qualifiedIdentifier string, typeID common.TypeID) (*sema.CompositeType, error),
) (_ sema.Type, err error) {
	switch t := typ.(type) {
	case CompositeStaticType:
		return getComposite(t.Location, t.QualifiedIdentifier, t.TypeID)

	case InterfaceStaticType:
		return getInterface(t.Location, t.QualifiedIdentifier)

	case VariableSizedStaticType:
		ty, err := ConvertStaticToSemaType(t.Type, getInterface, getComposite)
		return &sema.VariableSizedType{
			Type: ty,
		}, err

	case ConstantSizedStaticType:
		ty, err := ConvertStaticToSemaType(t.Type, getInterface, getComposite)
		return &sema.ConstantSizedType{
			Type: ty,
			Size: t.Size,
		}, err

	case DictionaryStaticType:
		keyType, err := ConvertStaticToSemaType(t.KeyType, getInterface, getComposite)
		if err != nil {
			return nil, err
		}
		valueType, err := ConvertStaticToSemaType(t.ValueType, getInterface, getComposite)
		return &sema.DictionaryType{
			KeyType:   keyType,
			ValueType: valueType,
		}, err

	case OptionalStaticType:
		ty, err := ConvertStaticToSemaType(t.Type, getInterface, getComposite)
		return &sema.OptionalType{
			Type: ty,
		}, err

	case *RestrictedStaticType:
		restrictions := make([]*sema.InterfaceType, len(t.Restrictions))

		for i, restriction := range t.Restrictions {
			restrictions[i], err = getInterface(restriction.Location, restriction.QualifiedIdentifier)
			if err != nil {
				return nil, err
			}
		}

		ty, err := ConvertStaticToSemaType(t.Type, getInterface, getComposite)
		return &sema.RestrictedType{
			Type:         ty,
			Restrictions: restrictions,
		}, err

	case ReferenceStaticType:
		ty, err := ConvertStaticToSemaType(t.BorrowedType, getInterface, getComposite)
		return &sema.ReferenceType{
			Authorized: t.Authorized,
			Type:       ty,
		}, err

	case CapabilityStaticType:
		var borrowType sema.Type
		if t.BorrowType != nil {
			borrowType, err = ConvertStaticToSemaType(t.BorrowType, getInterface, getComposite)
			if err != nil {
				return nil, err
			}
		}

		return &sema.CapabilityType{
			BorrowType: borrowType,
		}, nil

	case FunctionStaticType:
		return t.Type, nil

	case PrimitiveStaticType:
		return t.SemaType(), nil

	default:
		panic(errors.NewUnreachableError())
	}
}

// FunctionStaticType

type FunctionStaticType struct {
	Type *sema.FunctionType
}

var _ StaticType = FunctionStaticType{}

func NewFunctionStaticType(
	memoryGauge common.MemoryGauge,
	functionType *sema.FunctionType,
) FunctionStaticType {
	common.UseConstantMemory(memoryGauge, common.MemoryKindFunctionStaticType)

	return FunctionStaticType{
		Type: functionType,
	}
}

func (t FunctionStaticType) TypeParameters(interpreter *Interpreter) []*TypeParameter {
	typeParameters := make([]*TypeParameter, len(t.Type.TypeParameters))
	for i, typeParameter := range t.Type.TypeParameters {
		typeParameters[i] = &TypeParameter{
			Name:      typeParameter.Name,
			TypeBound: ConvertSemaToStaticType(interpreter, typeParameter.TypeBound),
			Optional:  typeParameter.Optional,
		}
	}

	return typeParameters
}

func (t FunctionStaticType) ParameterTypes(interpreter *Interpreter) []StaticType {
	parameterTypes := make([]StaticType, len(t.Type.Parameters))
	for i, parameter := range t.Type.Parameters {
		parameterTypes[i] = ConvertSemaToStaticType(interpreter, parameter.TypeAnnotation.Type)
	}

	return parameterTypes
}

func (t FunctionStaticType) ReturnType(interpreter *Interpreter) StaticType {
	var returnType StaticType
	if t.Type.ReturnTypeAnnotation != nil {
		returnType = ConvertSemaToStaticType(interpreter, t.Type.ReturnTypeAnnotation.Type)
	}

	return returnType
}

func (FunctionStaticType) isStaticType() {}

func (FunctionStaticType) elementSize() uint {
	return UnknownElementSize
}

func (t FunctionStaticType) String() string {
	return t.Type.String()
}

func (t FunctionStaticType) Equal(other StaticType) bool {
	otherFunction, ok := other.(FunctionStaticType)
	if !ok {
		return false
	}

	return t.Type.Equal(otherFunction.Type)
}

type TypeParameter struct {
	Name      string
	TypeBound StaticType
	Optional  bool
}

func (p TypeParameter) Equal(other *TypeParameter) bool {
	if p.TypeBound == nil {
		if other.TypeBound != nil {
			return false
		}
	} else {
		if other.TypeBound == nil ||
			!p.TypeBound.Equal(other.TypeBound) {

			return false
		}
	}

	return p.Optional == other.Optional
}

func (p TypeParameter) String() string {
	var builder strings.Builder
	builder.WriteString(p.Name)
	if p.TypeBound != nil {
		builder.WriteString(": ")
		builder.WriteString(p.TypeBound.String())
	}
	return builder.String()
}
