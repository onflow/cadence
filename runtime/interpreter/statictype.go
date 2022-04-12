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

func NewCompositeStaticType(location common.Location, qualifiedIdentifier string) CompositeStaticType {

	var typeID = common.NewTypeIDFromQualifiedName(location, qualifiedIdentifier)

	return CompositeStaticType{
		Location:            location,
		QualifiedIdentifier: qualifiedIdentifier,
		TypeID:              typeID,
	}
}

func (CompositeStaticType) isStaticType() {}

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

func (InterfaceStaticType) isStaticType() {}

func (t InterfaceStaticType) String() string {
	if t.Location == nil {
		return t.QualifiedIdentifier
	}
	return string(t.Location.TypeID(t.QualifiedIdentifier))
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

func (VariableSizedStaticType) isStaticType() {}

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

func (ConstantSizedStaticType) isStaticType() {}

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

func (DictionaryStaticType) isStaticType() {}

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

func (OptionalStaticType) isStaticType() {}

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

// NOTE: must be pointer receiver, as static types get used in type values,
// which are used as keys in maps when exporting.
// Key types in Go maps must be (transitively) hashable types,
// and slices are not, but `Restrictions` is one.
//
func (*RestrictedStaticType) isStaticType() {}

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

func (ReferenceStaticType) isStaticType() {}

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

func (CapabilityStaticType) isStaticType() {}

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

func ConvertSemaToStaticType(t sema.Type) StaticType {
	switch t := t.(type) {
	case *sema.CompositeType:
		return CompositeStaticType{
			Location:            t.Location,
			QualifiedIdentifier: t.QualifiedIdentifier(),
			TypeID:              t.ID(),
		}

	case *sema.InterfaceType:
		return ConvertSemaInterfaceTypeToStaticInterfaceType(t)

	case sema.ArrayType:
		return ConvertSemaArrayTypeToStaticArrayType(t)

	case *sema.DictionaryType:
		return ConvertSemaDictionaryTypeToStaticDictionaryType(t)

	case *sema.OptionalType:
		return OptionalStaticType{
			Type: ConvertSemaToStaticType(t.Type),
		}

	case *sema.RestrictedType:
		restrictions := make([]InterfaceStaticType, len(t.Restrictions))

		for i, restriction := range t.Restrictions {
			restrictions[i] = ConvertSemaInterfaceTypeToStaticInterfaceType(restriction)
		}

		return &RestrictedStaticType{
			Type:         ConvertSemaToStaticType(t.Type),
			Restrictions: restrictions,
		}

	case *sema.ReferenceType:
		return ConvertSemaReferenceTypeToStaticReferenceType(t)

	case *sema.CapabilityType:
		result := CapabilityStaticType{}
		if t.BorrowType != nil {
			result.BorrowType = ConvertSemaToStaticType(t.BorrowType)
		}
		return result

	case *sema.FunctionType:
		return FunctionStaticType{
			Type: t,
		}
	}

	primitiveStaticType := ConvertSemaToPrimitiveStaticType(t)
	if primitiveStaticType == PrimitiveStaticTypeUnknown {
		return nil
	}
	return primitiveStaticType
}

func ConvertSemaArrayTypeToStaticArrayType(t sema.ArrayType) ArrayStaticType {
	switch t := t.(type) {
	case *sema.VariableSizedType:
		return VariableSizedStaticType{
			Type: ConvertSemaToStaticType(t.Type),
		}

	case *sema.ConstantSizedType:
		return ConstantSizedStaticType{
			Type: ConvertSemaToStaticType(t.Type),
			Size: t.Size,
		}

	default:
		panic(errors.NewUnreachableError())
	}
}

func ConvertSemaDictionaryTypeToStaticDictionaryType(t *sema.DictionaryType) DictionaryStaticType {
	return DictionaryStaticType{
		KeyType:   ConvertSemaToStaticType(t.KeyType),
		ValueType: ConvertSemaToStaticType(t.ValueType),
	}
}

func ConvertSemaReferenceTypeToStaticReferenceType(t *sema.ReferenceType) ReferenceStaticType {
	return ReferenceStaticType{
		Authorized:   t.Authorized,
		BorrowedType: ConvertSemaToStaticType(t.Type),
	}
}

func ConvertSemaInterfaceTypeToStaticInterfaceType(t *sema.InterfaceType) InterfaceStaticType {
	return InterfaceStaticType{
		Location:            t.Location,
		QualifiedIdentifier: t.QualifiedIdentifier(),
	}
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

func (t FunctionStaticType) TypeParameters() []*TypeParameter {
	typeParameters := make([]*TypeParameter, len(t.Type.TypeParameters))
	for i, typeParameter := range t.Type.TypeParameters {
		typeParameters[i] = &TypeParameter{
			Name:      typeParameter.Name,
			TypeBound: ConvertSemaToStaticType(typeParameter.TypeBound),
			Optional:  typeParameter.Optional,
		}
	}

	return typeParameters
}

func (t FunctionStaticType) ParameterTypes() []StaticType {
	parameterTypes := make([]StaticType, len(t.Type.Parameters))
	for i, parameter := range t.Type.Parameters {
		parameterTypes[i] = ConvertSemaToStaticType(parameter.TypeAnnotation.Type)
	}

	return parameterTypes
}

func (t FunctionStaticType) ReturnType() StaticType {
	var returnType StaticType
	if t.Type.ReturnTypeAnnotation != nil {
		returnType = ConvertSemaToStaticType(t.Type.ReturnTypeAnnotation.Type)
	}

	return returnType
}

func (FunctionStaticType) isStaticType() {}

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
