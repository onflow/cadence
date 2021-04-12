/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2020 Dapper Labs, Inc.
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
	IsStaticType()
	Equal(other StaticType) bool
}

// CompositeStaticType

type CompositeStaticType struct {
	Location            common.Location
	QualifiedIdentifier string
}

func (CompositeStaticType) IsStaticType() {}

func (t CompositeStaticType) String() string {
	if t.Location == nil {
		return t.QualifiedIdentifier
	}
	return string(t.Location.TypeID(t.QualifiedIdentifier))
}

func (t CompositeStaticType) Equal(other StaticType) bool {
	otherCompositeType, ok := other.(CompositeStaticType)
	if !ok {
		return false
	}

	return common.LocationsMatch(otherCompositeType.Location, t.Location) &&
		otherCompositeType.QualifiedIdentifier == t.QualifiedIdentifier
}

// InterfaceStaticType

type InterfaceStaticType struct {
	Location            common.Location
	QualifiedIdentifier string
}

func (InterfaceStaticType) IsStaticType() {}

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

// VariableSizedStaticType

type VariableSizedStaticType struct {
	Type StaticType
}

func (VariableSizedStaticType) IsStaticType() {}

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

func (ConstantSizedStaticType) IsStaticType() {}

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

func (DictionaryStaticType) IsStaticType() {}

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

func (OptionalStaticType) IsStaticType() {}

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

// NOTE: must be pointer receiver, as static types get used in type values,
// which are used as keys in maps when exporting.
// Key types in Go maps must be (transitively) hashable types,
// and slices are not, but `Restrictions` is one.
//
func (*RestrictedStaticType) IsStaticType() {}

func (t *RestrictedStaticType) String() string {
	restrictions := make([]string, len(t.Restrictions))

	for i, restriction := range t.Restrictions {
		restrictions[i] = restriction.String()
	}

	return fmt.Sprintf("%s{%s}", t.Type, strings.Join(restrictions, ", "))
}

func (t *RestrictedStaticType) Equal(other StaticType) bool {
	otherRestrictedType, ok := other.(*RestrictedStaticType)
	if !ok {
		return false
	}

	// TODO: restrictions

	return t.Type.Equal(otherRestrictedType.Type)
}

// ReferenceStaticType

type ReferenceStaticType struct {
	Authorized bool
	Type       StaticType
}

func (ReferenceStaticType) IsStaticType() {}

func (t ReferenceStaticType) String() string {
	auth := ""
	if t.Authorized {
		auth = "auth "
	}

	return fmt.Sprintf("%s&%s", auth, t.Type)
}

func (t ReferenceStaticType) Equal(other StaticType) bool {
	otherReferenceType, ok := other.(ReferenceStaticType)
	if !ok {
		return false
	}

	return t.Authorized == otherReferenceType.Authorized &&
		t.Type.Equal(otherReferenceType.Type)
}

// CapabilityStaticType

type CapabilityStaticType struct {
	BorrowType StaticType
}

func (CapabilityStaticType) IsStaticType() {}

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
		}

	case *sema.InterfaceType:
		return convertToInterfaceStaticType(t)

	case *sema.VariableSizedType:
		return VariableSizedStaticType{
			Type: ConvertSemaToStaticType(t.Type),
		}

	case *sema.ConstantSizedType:
		return ConstantSizedStaticType{
			Type: ConvertSemaToStaticType(t.Type),
			Size: t.Size,
		}

	case *sema.DictionaryType:
		return DictionaryStaticType{
			KeyType:   ConvertSemaToStaticType(t.KeyType),
			ValueType: ConvertSemaToStaticType(t.ValueType),
		}

	case *sema.OptionalType:
		return OptionalStaticType{
			Type: ConvertSemaToStaticType(t.Type),
		}

	case *sema.RestrictedType:
		restrictions := make([]InterfaceStaticType, len(t.Restrictions))

		for i, restriction := range t.Restrictions {
			restrictions[i] = convertToInterfaceStaticType(restriction)
		}

		return &RestrictedStaticType{
			Type:         ConvertSemaToStaticType(t.Type),
			Restrictions: restrictions,
		}

	case *sema.ReferenceType:
		return convertSemaReferenceToStaticReferenceType(t)

	case *sema.CapabilityType:
		result := CapabilityStaticType{}
		if t.BorrowType != nil {
			result.BorrowType = ConvertSemaToStaticType(t.BorrowType)
		}
		return result
	}

	primitiveStaticType := ConvertSemaToPrimitiveStaticType(t)
	if primitiveStaticType == PrimitiveStaticTypeUnknown {
		return nil
	}
	return primitiveStaticType
}

func convertSemaReferenceToStaticReferenceType(t *sema.ReferenceType) ReferenceStaticType {
	return ReferenceStaticType{
		Authorized: t.Authorized,
		Type:       ConvertSemaToStaticType(t.Type),
	}
}

func convertToInterfaceStaticType(t *sema.InterfaceType) InterfaceStaticType {
	return InterfaceStaticType{
		Location:            t.Location,
		QualifiedIdentifier: t.QualifiedIdentifier(),
	}
}

func ConvertStaticToSemaType(
	typ StaticType,
	getInterface func(location common.Location, qualifiedIdentifier string) *sema.InterfaceType,
	getComposite func(location common.Location, qualifiedIdentifier string) *sema.CompositeType,
) sema.Type {
	switch t := typ.(type) {
	case CompositeStaticType:
		return getComposite(t.Location, t.QualifiedIdentifier)

	case InterfaceStaticType:
		return getInterface(t.Location, t.QualifiedIdentifier)

	case VariableSizedStaticType:
		return &sema.VariableSizedType{
			Type: ConvertStaticToSemaType(t.Type, getInterface, getComposite),
		}

	case ConstantSizedStaticType:
		return &sema.ConstantSizedType{
			Type: ConvertStaticToSemaType(t.Type, getInterface, getComposite),
			Size: t.Size,
		}

	case DictionaryStaticType:
		return &sema.DictionaryType{
			KeyType:   ConvertStaticToSemaType(t.KeyType, getInterface, getComposite),
			ValueType: ConvertStaticToSemaType(t.ValueType, getInterface, getComposite),
		}

	case OptionalStaticType:
		return &sema.OptionalType{
			Type: ConvertStaticToSemaType(t.Type, getInterface, getComposite),
		}

	case *RestrictedStaticType:
		restrictions := make([]*sema.InterfaceType, len(t.Restrictions))

		for i, restriction := range t.Restrictions {
			restrictions[i] = getInterface(restriction.Location, restriction.QualifiedIdentifier)
		}

		return &sema.RestrictedType{
			Type:         ConvertStaticToSemaType(t.Type, getInterface, getComposite),
			Restrictions: restrictions,
		}

	case ReferenceStaticType:
		return &sema.ReferenceType{
			Authorized: t.Authorized,
			Type:       ConvertStaticToSemaType(t.Type, getInterface, getComposite),
		}

	case CapabilityStaticType:
		var borrowType sema.Type
		if t.BorrowType != nil {
			borrowType = ConvertStaticToSemaType(t.BorrowType, getInterface, getComposite)
		}

		return &sema.CapabilityType{
			BorrowType: borrowType,
		}

	case PrimitiveStaticType:
		return t.SemaType()

	default:
		panic(errors.NewUnreachableError())
	}
}
