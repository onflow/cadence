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
type StaticType interface {
	fmt.Stringer
	isStaticType()
	/* this returns the size (in bytes) of the largest inhabitant of this type,
	or UnknownElementSize if the largest inhabitant has arbitrary size */
	elementSize() uint
	Equal(other StaticType) bool
	Encode(e *cbor.StreamEncoder) error
	MeteredString(memoryGauge common.MemoryGauge) string
	ID() TypeID
}

type TypeID = common.TypeID

// CompositeStaticType

type CompositeStaticType struct {
	Location            common.Location
	QualifiedIdentifier string
	TypeID              TypeID
}

var _ StaticType = CompositeStaticType{}

func NewCompositeStaticType(
	memoryGauge common.MemoryGauge,
	location common.Location,
	qualifiedIdentifier string,
	typeID TypeID,
) CompositeStaticType {
	common.UseMemory(memoryGauge, common.CompositeStaticTypeMemoryUsage)

	if typeID == "" {
		panic(errors.NewUnreachableError())
	}

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
	typeID := common.NewTypeIDFromQualifiedName(
		memoryGauge,
		location,
		qualifiedIdentifier,
	)

	return NewCompositeStaticType(
		memoryGauge,
		location,
		qualifiedIdentifier,
		typeID,
	)
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

func (t CompositeStaticType) MeteredString(memoryGauge common.MemoryGauge) string {
	var amount int
	if t.Location == nil {
		amount = len(t.QualifiedIdentifier)
	} else {
		amount = len(t.TypeID)
	}

	common.UseMemory(memoryGauge, common.NewRawStringMemoryUsage(amount))
	return t.String()
}

func (t CompositeStaticType) Equal(other StaticType) bool {
	otherCompositeType, ok := other.(CompositeStaticType)
	if !ok {
		return false
	}

	return otherCompositeType.TypeID == t.TypeID
}

func (t CompositeStaticType) ID() TypeID {
	return t.TypeID
}

// InterfaceStaticType

type InterfaceStaticType struct {
	Location            common.Location
	QualifiedIdentifier string
	TypeID              common.TypeID
}

var _ StaticType = InterfaceStaticType{}

func NewInterfaceStaticType(
	memoryGauge common.MemoryGauge,
	location common.Location,
	qualifiedIdentifier string,
	typeID common.TypeID,
) InterfaceStaticType {
	common.UseMemory(memoryGauge, common.InterfaceStaticTypeMemoryUsage)

	if typeID == "" {
		panic(errors.NewUnreachableError())
	}

	return InterfaceStaticType{
		Location:            location,
		QualifiedIdentifier: qualifiedIdentifier,
		TypeID:              typeID,
	}
}

func NewInterfaceStaticTypeComputeTypeID(
	memoryGauge common.MemoryGauge,
	location common.Location,
	qualifiedIdentifier string,
) InterfaceStaticType {
	typeID := common.NewTypeIDFromQualifiedName(
		memoryGauge,
		location,
		qualifiedIdentifier,
	)

	return NewInterfaceStaticType(
		memoryGauge,
		location,
		qualifiedIdentifier,
		typeID,
	)
}

func (InterfaceStaticType) isStaticType() {}

func (InterfaceStaticType) elementSize() uint {
	return UnknownElementSize
}

func (t InterfaceStaticType) String() string {
	if t.Location == nil {
		return t.QualifiedIdentifier
	}
	return string(t.TypeID)
}

func (t InterfaceStaticType) MeteredString(memoryGauge common.MemoryGauge) string {
	var amount int
	if t.Location == nil {
		amount = len(t.QualifiedIdentifier)
	} else {
		amount = len(t.TypeID)
	}

	common.UseMemory(memoryGauge, common.NewRawStringMemoryUsage(amount))
	return t.String()
}

func (t InterfaceStaticType) Equal(other StaticType) bool {
	otherInterfaceType, ok := other.(InterfaceStaticType)
	if !ok {
		return false
	}

	return otherInterfaceType.TypeID == t.TypeID
}

func (t InterfaceStaticType) ID() TypeID {
	if t.Location == nil {
		return TypeID(t.QualifiedIdentifier)
	}
	return t.Location.TypeID(nil, t.QualifiedIdentifier)
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
	common.UseMemory(memoryGauge, common.VariableSizedStaticTypeMemoryUsage)

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

func (t VariableSizedStaticType) MeteredString(memoryGauge common.MemoryGauge) string {
	common.UseMemory(memoryGauge, common.VariableSizedStaticTypeStringMemoryUsage)

	typeStr := t.Type.MeteredString(memoryGauge)
	return fmt.Sprintf("[%s]", typeStr)
}

func (t VariableSizedStaticType) Equal(other StaticType) bool {
	otherVariableSizedType, ok := other.(VariableSizedStaticType)
	if !ok {
		return false
	}

	return t.Type.Equal(otherVariableSizedType.Type)
}

func (t VariableSizedStaticType) ID() TypeID {
	return sema.VariableSizedTypeID(t.Type.ID())
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
	common.UseMemory(memoryGauge, common.ConstantSizedStaticTypeMemoryUsage)

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

func (t ConstantSizedStaticType) MeteredString(memoryGauge common.MemoryGauge) string {
	// n - for size
	// 2 - for open and close bracket.
	// 1 - for space
	// 1 - for semicolon
	// Nested type is separately metered.
	strLen := OverEstimateIntStringLength(int(t.Size)) + 4
	common.UseMemory(memoryGauge, common.NewRawStringMemoryUsage(strLen))

	typeStr := t.Type.MeteredString(memoryGauge)

	return fmt.Sprintf("[%s; %d]", typeStr, t.Size)
}

func (t ConstantSizedStaticType) Equal(other StaticType) bool {
	otherConstantSizedType, ok := other.(ConstantSizedStaticType)
	if !ok {
		return false
	}

	return t.Size == otherConstantSizedType.Size &&
		t.Type.Equal(otherConstantSizedType.Type)
}

func (t ConstantSizedStaticType) ID() TypeID {
	return sema.ConstantSizedTypeID(t.Type.ID(), t.Size)
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
	common.UseMemory(memoryGauge, common.DictionaryStaticTypeMemoryUsage)

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

func (t DictionaryStaticType) MeteredString(memoryGauge common.MemoryGauge) string {
	common.UseMemory(memoryGauge, common.DictionaryStaticTypeStringMemoryUsage)

	keyStr := t.KeyType.MeteredString(memoryGauge)
	valueStr := t.ValueType.MeteredString(memoryGauge)

	return fmt.Sprintf("{%s: %s}", keyStr, valueStr)
}

func (t DictionaryStaticType) Equal(other StaticType) bool {
	otherDictionaryType, ok := other.(DictionaryStaticType)
	if !ok {
		return false
	}

	return t.KeyType.Equal(otherDictionaryType.KeyType) &&
		t.ValueType.Equal(otherDictionaryType.ValueType)
}

func (t DictionaryStaticType) ID() TypeID {
	return sema.DictionaryTypeID(
		t.KeyType.ID(),
		t.ValueType.ID(),
	)
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
	common.UseMemory(memoryGauge, common.OptionalStaticTypeMemoryUsage)

	return OptionalStaticType{Type: typ}
}

func (OptionalStaticType) isStaticType() {}

func (OptionalStaticType) elementSize() uint {
	return UnknownElementSize
}

func (t OptionalStaticType) String() string {
	return fmt.Sprintf("%s?", t.Type)
}

func (t OptionalStaticType) MeteredString(memoryGauge common.MemoryGauge) string {
	common.UseMemory(memoryGauge, common.OptionalStaticTypeStringMemoryUsage)

	typeStr := t.Type.MeteredString(memoryGauge)
	return fmt.Sprintf("%s?", typeStr)
}

func (t OptionalStaticType) Equal(other StaticType) bool {
	otherOptionalType, ok := other.(OptionalStaticType)
	if !ok {
		return false
	}

	return t.Type.Equal(otherOptionalType.Type)
}

func (t OptionalStaticType) ID() TypeID {
	return sema.OptionalTypeID(t.Type.ID())
}

var NilStaticType = OptionalStaticType{
	Type: PrimitiveStaticTypeNever,
}

// RestrictedStaticType

type RestrictedStaticType struct {
	Type         StaticType
	Restrictions []InterfaceStaticType
	typeID       TypeID
}

var _ StaticType = &RestrictedStaticType{}

func NewRestrictedStaticType(
	memoryGauge common.MemoryGauge,
	staticType StaticType,
	restrictions []InterfaceStaticType,
) *RestrictedStaticType {
	common.UseMemory(memoryGauge, common.RestrictedStaticTypeMemoryUsage)

	return &RestrictedStaticType{
		Type:         staticType,
		Restrictions: restrictions,
	}
}

// NOTE: must be pointer receiver, as static types get used in type values,
// which are used as keys in maps when exporting.
// Key types in Go maps must be (transitively) hashable types,
// and slices are not, but `Restrictions` is one.
func (*RestrictedStaticType) isStaticType() {}

func (*RestrictedStaticType) elementSize() uint {
	return UnknownElementSize
}

func (t *RestrictedStaticType) String() string {
	var restrictions []string

	count := len(t.Restrictions)
	if count > 0 {
		restrictions = make([]string, count)

		for i, restriction := range t.Restrictions {
			restrictions[i] = restriction.String()
		}
	}

	return fmt.Sprintf("%s{%s}", t.Type, strings.Join(restrictions, ", "))
}

func (t *RestrictedStaticType) MeteredString(memoryGauge common.MemoryGauge) string {
	restrictions := make([]string, len(t.Restrictions))

	for i, restriction := range t.Restrictions {
		restrictions[i] = restriction.MeteredString(memoryGauge)
	}

	// len = (comma + space) x (n - 1)
	// To handle n == 0:
	// 		len = (comma + space) x n
	//
	l := len(restrictions)*2 + 2
	common.UseMemory(memoryGauge, common.NewRawStringMemoryUsage(l))

	typeStr := t.Type.MeteredString(memoryGauge)

	return fmt.Sprintf("%s{%s}", typeStr, strings.Join(restrictions, ", "))
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

func (t *RestrictedStaticType) ID() TypeID {
	if t.typeID == "" {
		var restrictionStrings []string
		restrictionCount := len(t.Restrictions)
		if restrictionCount > 0 {
			restrictionStrings = make([]string, 0, restrictionCount)
			for _, restriction := range t.Restrictions {
				restrictionStrings = append(restrictionStrings, string(restriction.ID()))
			}
		}
		var typeString string
		if t.Type != nil {
			typeString = string(t.Type.ID())
		}
		t.typeID = TypeID(sema.FormatRestrictedTypeID(typeString, restrictionStrings))
	}
	return t.typeID
}

// ReferenceStaticType

type ReferenceStaticType struct {
	// BorrowedType is the type of the usage (T in &T)
	BorrowedType StaticType
	// ReferencedType is type of the referenced value (the type of the target)
	ReferencedType StaticType
	Authorized     bool
	typeID         TypeID
}

var _ StaticType = ReferenceStaticType{}

func NewReferenceStaticType(
	memoryGauge common.MemoryGauge,
	authorized bool,
	borrowedType StaticType,
	referencedType StaticType,
) ReferenceStaticType {
	common.UseMemory(memoryGauge, common.ReferenceStaticTypeMemoryUsage)

	return ReferenceStaticType{
		Authorized:     authorized,
		BorrowedType:   borrowedType,
		ReferencedType: referencedType,
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

func (t ReferenceStaticType) MeteredString(memoryGauge common.MemoryGauge) string {
	if t.Authorized {
		common.UseMemory(memoryGauge, common.AuthReferenceStaticTypeStringMemoryUsage)
	} else {
		common.UseMemory(memoryGauge, common.ReferenceStaticTypeStringMemoryUsage)
	}

	typeStr := t.BorrowedType.MeteredString(memoryGauge)

	auth := ""
	if t.Authorized {
		auth = "auth "
	}

	return fmt.Sprintf("%s&%s", auth, typeStr)
}

func (t ReferenceStaticType) Equal(other StaticType) bool {
	otherReferenceType, ok := other.(ReferenceStaticType)
	if !ok {
		return false
	}

	return t.Authorized == otherReferenceType.Authorized &&
		t.BorrowedType.Equal(otherReferenceType.BorrowedType)
}

func (t ReferenceStaticType) ID() TypeID {
	if t.typeID == "" {
		t.typeID = TypeID(sema.FormatReferenceTypeID(t.Authorized, string(t.BorrowedType.ID())))
	}
	return t.typeID
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
	common.UseMemory(memoryGauge, common.CapabilityStaticTypeMemoryUsage)

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

func (t CapabilityStaticType) MeteredString(memoryGauge common.MemoryGauge) string {
	common.UseMemory(memoryGauge, common.CapabilityStaticTypeStringMemoryUsage)

	if t.BorrowType != nil {
		typeStr := t.BorrowType.MeteredString(memoryGauge)
		return fmt.Sprintf("Capability<%s>", typeStr)
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

func (t CapabilityStaticType) ID() TypeID {
	var borrowTypeString string
	borrowType := t.BorrowType
	if borrowType != nil {
		borrowTypeString = string(borrowType.ID())
	}
	return TypeID(sema.FormatCapabilityTypeID(borrowTypeString))
}

// Conversion

func ConvertSemaToStaticType(memoryGauge common.MemoryGauge, t sema.Type) StaticType {

	primitiveStaticType := ConvertSemaToPrimitiveStaticType(memoryGauge, t)
	if primitiveStaticType != PrimitiveStaticTypeUnknown {
		return primitiveStaticType
	}

	switch t := t.(type) {
	case *sema.CompositeType:
		return ConvertSemaCompositeTypeToStaticCompositeType(memoryGauge, t)

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
		var restrictions []InterfaceStaticType
		restrictionCount := len(t.Restrictions)
		if restrictionCount > 0 {
			restrictions = make([]InterfaceStaticType, restrictionCount)

			for i, restriction := range t.Restrictions {
				restrictions[i] = ConvertSemaInterfaceTypeToStaticInterfaceType(memoryGauge, restriction)
			}
		}

		return NewRestrictedStaticType(
			memoryGauge,
			ConvertSemaToStaticType(memoryGauge, t.Type),
			restrictions,
		)

	case *sema.ReferenceType:
		return ConvertSemaReferenceTypeToStaticReferenceType(memoryGauge, t)

	case *sema.CapabilityType:
		if t.BorrowType == nil {
			// Unparameterized Capability type should have been
			// converted to primitive static type earlier
			panic(errors.NewUnreachableError())
		}
		borrowType := ConvertSemaToStaticType(memoryGauge, t.BorrowType)
		return NewCapabilityStaticType(memoryGauge, borrowType)

	case *sema.FunctionType:
		return NewFunctionStaticType(memoryGauge, t)
	}

	return nil
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

func ConvertSemaCompositeTypeToStaticCompositeType(
	memoryGauge common.MemoryGauge,
	t *sema.CompositeType,
) CompositeStaticType {
	return NewCompositeStaticType(
		memoryGauge,
		t.Location,
		t.QualifiedIdentifier(),
		t.ID(),
	)
}

func ConvertSemaInterfaceTypeToStaticInterfaceType(
	memoryGauge common.MemoryGauge,
	t *sema.InterfaceType,
) InterfaceStaticType {
	return NewInterfaceStaticType(
		memoryGauge,
		t.Location,
		t.QualifiedIdentifier(),
		t.ID(),
	)
}

func ConvertStaticToSemaType(
	memoryGauge common.MemoryGauge,
	typ StaticType,
	getInterface func(location common.Location, qualifiedIdentifier string) (*sema.InterfaceType, error),
	getComposite func(location common.Location, qualifiedIdentifier string, typeID TypeID) (*sema.CompositeType, error),
) (_ sema.Type, err error) {
	switch t := typ.(type) {
	case CompositeStaticType:
		return getComposite(t.Location, t.QualifiedIdentifier, t.TypeID)

	case InterfaceStaticType:
		return getInterface(t.Location, t.QualifiedIdentifier)

	case VariableSizedStaticType:
		ty, err := ConvertStaticToSemaType(memoryGauge, t.Type, getInterface, getComposite)
		if err != nil {
			return nil, err
		}
		return sema.NewVariableSizedType(memoryGauge, ty), nil

	case ConstantSizedStaticType:
		ty, err := ConvertStaticToSemaType(memoryGauge, t.Type, getInterface, getComposite)
		if err != nil {
			return nil, err
		}

		return sema.NewConstantSizedType(
			memoryGauge,
			ty,
			t.Size,
		), nil

	case DictionaryStaticType:
		keyType, err := ConvertStaticToSemaType(memoryGauge, t.KeyType, getInterface, getComposite)
		if err != nil {
			return nil, err
		}

		valueType, err := ConvertStaticToSemaType(memoryGauge, t.ValueType, getInterface, getComposite)
		if err != nil {
			return nil, err
		}

		return sema.NewDictionaryType(
			memoryGauge,
			keyType,
			valueType,
		), nil

	case OptionalStaticType:
		ty, err := ConvertStaticToSemaType(memoryGauge, t.Type, getInterface, getComposite)
		if err != nil {
			return nil, err
		}
		return sema.NewOptionalType(memoryGauge, ty), err

	case *RestrictedStaticType:
		var restrictions []*sema.InterfaceType

		restrictionCount := len(t.Restrictions)
		if restrictionCount > 0 {
			restrictions = make([]*sema.InterfaceType, restrictionCount)

			for i, restriction := range t.Restrictions {
				restrictions[i], err = getInterface(restriction.Location, restriction.QualifiedIdentifier)
				if err != nil {
					return nil, err
				}
			}
		}

		ty, err := ConvertStaticToSemaType(memoryGauge, t.Type, getInterface, getComposite)
		if err != nil {
			return nil, err
		}

		return sema.NewRestrictedType(
			memoryGauge,
			ty,
			restrictions,
		), nil

	case ReferenceStaticType:
		ty, err := ConvertStaticToSemaType(memoryGauge, t.BorrowedType, getInterface, getComposite)
		if err != nil {
			return nil, err
		}

		return sema.NewReferenceType(
			memoryGauge,
			ty,
			t.Authorized,
		), nil

	case CapabilityStaticType:
		var borrowType sema.Type
		if t.BorrowType != nil {
			borrowType, err = ConvertStaticToSemaType(memoryGauge, t.BorrowType, getInterface, getComposite)
			if err != nil {
				return nil, err
			}
		}

		return sema.NewCapabilityType(memoryGauge, borrowType), nil

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
	common.UseMemory(memoryGauge, common.FunctionStaticTypeMemoryUsage)

	return FunctionStaticType{
		Type: functionType,
	}
}

func (t FunctionStaticType) TypeParameters(interpreter *Interpreter) []*TypeParameter {
	var typeParameters []*TypeParameter

	count := len(t.Type.TypeParameters)
	if count > 0 {
		typeParameters = make([]*TypeParameter, count)
		for i, typeParameter := range t.Type.TypeParameters {
			typeParameters[i] = &TypeParameter{
				Name:      typeParameter.Name,
				TypeBound: ConvertSemaToStaticType(interpreter, typeParameter.TypeBound),
				Optional:  typeParameter.Optional,
			}
		}
	}

	return typeParameters
}

func (t FunctionStaticType) ParameterTypes(interpreter *Interpreter) []StaticType {
	var parameterTypes []StaticType

	count := len(t.Type.Parameters)
	if count > 0 {
		parameterTypes = make([]StaticType, count)
		for i, parameter := range t.Type.Parameters {
			parameterTypes[i] = ConvertSemaToStaticType(interpreter, parameter.TypeAnnotation.Type)
		}
	}

	return parameterTypes
}

func (t FunctionStaticType) ReturnType(interpreter *Interpreter) StaticType {
	var returnType StaticType
	if t.Type.ReturnTypeAnnotation.Type != nil {
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

func (t FunctionStaticType) MeteredString(memoryGauge common.MemoryGauge) string {
	// TODO: Meter sema.Type string conversion
	typeStr := t.String()
	common.UseMemory(memoryGauge, common.NewRawStringMemoryUsage(len(typeStr)))
	return typeStr
}

func (t FunctionStaticType) Equal(other StaticType) bool {
	otherFunction, ok := other.(FunctionStaticType)
	if !ok {
		return false
	}

	return t.Type.Equal(otherFunction.Type)
}

func (t FunctionStaticType) ID() TypeID {
	return t.Type.ID()
}

type TypeParameter struct {
	TypeBound StaticType
	Name      string
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
