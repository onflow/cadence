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
	"github.com/onflow/cadence/runtime/common/orderedmap"
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
	IsDeprecated() bool
}

type TypeID = common.TypeID

// CompositeStaticType

type CompositeStaticType struct {
	Location            common.Location
	QualifiedIdentifier string
	TypeID              TypeID
}

var _ StaticType = &CompositeStaticType{}

func NewCompositeStaticType(
	memoryGauge common.MemoryGauge,
	location common.Location,
	qualifiedIdentifier string,
	typeID TypeID,
) *CompositeStaticType {
	common.UseMemory(memoryGauge, common.CompositeStaticTypeMemoryUsage)

	if typeID == "" {
		panic(errors.NewUnreachableError())
	}

	return &CompositeStaticType{
		Location:            location,
		QualifiedIdentifier: qualifiedIdentifier,
		TypeID:              typeID,
	}
}

func NewCompositeStaticTypeComputeTypeID(
	memoryGauge common.MemoryGauge,
	location common.Location,
	qualifiedIdentifier string,
) *CompositeStaticType {
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

func (*CompositeStaticType) isStaticType() {}

func (*CompositeStaticType) elementSize() uint {
	return UnknownElementSize
}

func (t *CompositeStaticType) String() string {
	return t.MeteredString(nil)
}

func (t *CompositeStaticType) MeteredString(memoryGauge common.MemoryGauge) string {
	common.UseMemory(memoryGauge, common.NewRawStringMemoryUsage(len(t.TypeID)))
	return string(t.TypeID)
}

func (t *CompositeStaticType) Equal(other StaticType) bool {
	otherCompositeType, ok := other.(*CompositeStaticType)
	if !ok {
		return false
	}

	return otherCompositeType.TypeID == t.TypeID
}

func (t *CompositeStaticType) ID() TypeID {
	return t.TypeID
}

func (*CompositeStaticType) IsDeprecated() bool {
	return false
}

// InterfaceStaticType

type InterfaceStaticType struct {
	Location            common.Location
	QualifiedIdentifier string
	TypeID              common.TypeID
}

var _ StaticType = &InterfaceStaticType{}

func NewInterfaceStaticType(
	memoryGauge common.MemoryGauge,
	location common.Location,
	qualifiedIdentifier string,
	typeID common.TypeID,
) *InterfaceStaticType {
	common.UseMemory(memoryGauge, common.InterfaceStaticTypeMemoryUsage)

	if typeID == "" {
		panic(errors.NewUnreachableError())
	}

	return &InterfaceStaticType{
		Location:            location,
		QualifiedIdentifier: qualifiedIdentifier,
		TypeID:              typeID,
	}
}

func NewInterfaceStaticTypeComputeTypeID(
	memoryGauge common.MemoryGauge,
	location common.Location,
	qualifiedIdentifier string,
) *InterfaceStaticType {
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

func (*InterfaceStaticType) isStaticType() {}

func (*InterfaceStaticType) elementSize() uint {
	return UnknownElementSize
}

func (t *InterfaceStaticType) String() string {
	return t.MeteredString(nil)
}

func (t *InterfaceStaticType) MeteredString(memoryGauge common.MemoryGauge) string {
	common.UseMemory(memoryGauge, common.NewRawStringMemoryUsage(len(t.TypeID)))
	return string(t.TypeID)
}

func (t *InterfaceStaticType) Equal(other StaticType) bool {
	otherInterfaceType, ok := other.(*InterfaceStaticType)
	if !ok {
		return false
	}

	return otherInterfaceType.TypeID == t.TypeID
}

func (t *InterfaceStaticType) ID() TypeID {
	return t.TypeID
}

func (*InterfaceStaticType) IsDeprecated() bool {
	return false
}

// ArrayStaticType

type ArrayStaticType interface {
	StaticType
	isArrayStaticType()
	ElementType() StaticType
	atree.TypeInfo
}

// VariableSizedStaticType

type VariableSizedStaticType struct {
	Type StaticType
}

var _ ArrayStaticType = &VariableSizedStaticType{}
var _ atree.TypeInfo = &VariableSizedStaticType{}

func NewVariableSizedStaticType(
	memoryGauge common.MemoryGauge,
	elementType StaticType,
) *VariableSizedStaticType {
	common.UseMemory(memoryGauge, common.VariableSizedStaticTypeMemoryUsage)

	return &VariableSizedStaticType{
		Type: elementType,
	}
}

func (*VariableSizedStaticType) IsComposite() bool {
	return false
}

func (t *VariableSizedStaticType) Copy() atree.TypeInfo {
	// VariableSizedStaticType is never mutated, return a shallow copy
	return t
}

func (t *VariableSizedStaticType) Identifier() string {
	return string(t.ID())
}

func (*VariableSizedStaticType) isStaticType() {}

func (*VariableSizedStaticType) elementSize() uint {
	return UnknownElementSize
}

func (*VariableSizedStaticType) isArrayStaticType() {}

func (t *VariableSizedStaticType) ElementType() StaticType {
	return t.Type
}

func (t *VariableSizedStaticType) String() string {
	return t.MeteredString(nil)
}

func (t *VariableSizedStaticType) MeteredString(memoryGauge common.MemoryGauge) string {
	typeStr := t.Type.MeteredString(memoryGauge)

	common.UseMemory(memoryGauge, common.VariableSizedStaticTypeStringMemoryUsage)
	return fmt.Sprintf("[%s]", typeStr)
}

func (t *VariableSizedStaticType) Equal(other StaticType) bool {
	otherVariableSizedType, ok := other.(*VariableSizedStaticType)
	if !ok {
		return false
	}

	return t.Type.Equal(otherVariableSizedType.Type)
}

func (t *VariableSizedStaticType) ID() TypeID {
	return sema.FormatVariableSizedTypeID(t.Type.ID())
}

func (t *VariableSizedStaticType) IsDeprecated() bool {
	return t.Type.IsDeprecated()
}

// InclusiveRangeStaticType

type InclusiveRangeStaticType struct {
	ElementType StaticType
}

var _ StaticType = InclusiveRangeStaticType{}
var _ atree.TypeInfo = InclusiveRangeStaticType{}

func NewInclusiveRangeStaticType(
	memoryGauge common.MemoryGauge,
	elementType StaticType,
) InclusiveRangeStaticType {
	common.UseMemory(memoryGauge, common.InclusiveRangeStaticTypeMemoryUsage)

	return InclusiveRangeStaticType{
		ElementType: elementType,
	}
}

func (t InclusiveRangeStaticType) IsComposite() bool {
	return false
}

func (t InclusiveRangeStaticType) Identifier() string {
	return string(t.ID())
}

func (t InclusiveRangeStaticType) Copy() atree.TypeInfo {
	return t
}

func (InclusiveRangeStaticType) isStaticType() {}

func (InclusiveRangeStaticType) elementSize() uint {
	return UnknownElementSize
}

func (t InclusiveRangeStaticType) String() string {
	return t.MeteredString(nil)
}

func (t InclusiveRangeStaticType) MeteredString(memoryGauge common.MemoryGauge) string {
	common.UseMemory(memoryGauge, common.InclusiveRangeStaticTypeStringMemoryUsage)

	elementStr := t.ElementType.MeteredString(memoryGauge)

	return fmt.Sprintf("InclusiveRange<%s>", elementStr)
}

func (t InclusiveRangeStaticType) Equal(other StaticType) bool {
	otherRangeType, ok := other.(InclusiveRangeStaticType)
	if !ok {
		return false
	}

	return t.ElementType.Equal(otherRangeType.ElementType)
}

func (t InclusiveRangeStaticType) ID() TypeID {
	return sema.InclusiveRangeTypeID(string(t.ElementType.ID()))
}

func (t InclusiveRangeStaticType) IsDeprecated() bool {
	return t.ElementType.IsDeprecated()
}

// ConstantSizedStaticType

type ConstantSizedStaticType struct {
	Type StaticType
	Size int64
}

var _ ArrayStaticType = &ConstantSizedStaticType{}
var _ atree.TypeInfo = &ConstantSizedStaticType{}

func NewConstantSizedStaticType(
	memoryGauge common.MemoryGauge,
	elementType StaticType,
	size int64,
) *ConstantSizedStaticType {
	common.UseMemory(memoryGauge, common.ConstantSizedStaticTypeMemoryUsage)

	return &ConstantSizedStaticType{
		Type: elementType,
		Size: size,
	}
}

func (*ConstantSizedStaticType) IsComposite() bool {
	return false
}

func (t *ConstantSizedStaticType) Copy() atree.TypeInfo {
	// ConstantSizedStaticType is never mutated, return a shallow copy\
	return t
}

func (t *ConstantSizedStaticType) Identifier() string {
	return string(t.ID())
}

func (*ConstantSizedStaticType) isStaticType() {}

func (*ConstantSizedStaticType) elementSize() uint {
	return UnknownElementSize
}

func (*ConstantSizedStaticType) isArrayStaticType() {}

func (t *ConstantSizedStaticType) ElementType() StaticType {
	return t.Type
}

func (t *ConstantSizedStaticType) String() string {
	return t.MeteredString(nil)
}

func (t *ConstantSizedStaticType) MeteredString(memoryGauge common.MemoryGauge) string {
	typeStr := t.Type.MeteredString(memoryGauge)

	// n - for size
	// 2 - for open and close bracket.
	// 1 - for space
	// 1 - for semicolon
	// Nested type is separately metered.
	strLen := OverEstimateIntStringLength(int(t.Size)) + 4
	common.UseMemory(memoryGauge, common.NewRawStringMemoryUsage(strLen))
	return fmt.Sprintf("[%s; %d]", typeStr, t.Size)
}

func (t *ConstantSizedStaticType) Equal(other StaticType) bool {
	otherConstantSizedType, ok := other.(*ConstantSizedStaticType)
	if !ok {
		return false
	}

	return t.Size == otherConstantSizedType.Size &&
		t.Type.Equal(otherConstantSizedType.Type)
}

func (t *ConstantSizedStaticType) ID() TypeID {
	return sema.FormatConstantSizedTypeID(t.Type.ID(), t.Size)
}

func (t *ConstantSizedStaticType) IsDeprecated() bool {
	return t.Type.IsDeprecated()
}

// DictionaryStaticType

type DictionaryStaticType struct {
	KeyType   StaticType
	ValueType StaticType
}

var _ StaticType = &DictionaryStaticType{}
var _ atree.TypeInfo = &DictionaryStaticType{}

func NewDictionaryStaticType(
	memoryGauge common.MemoryGauge,
	keyType, valueType StaticType,
) *DictionaryStaticType {
	common.UseMemory(memoryGauge, common.DictionaryStaticTypeMemoryUsage)

	return &DictionaryStaticType{
		KeyType:   keyType,
		ValueType: valueType,
	}
}

func (*DictionaryStaticType) IsComposite() bool {
	return false
}

func (t *DictionaryStaticType) Copy() atree.TypeInfo {
	// DictionaryStaticType is never mutated, return a shallow copy
	return t
}

func (t *DictionaryStaticType) Identifier() string {
	return string(t.ID())
}

func (*DictionaryStaticType) isStaticType() {}

func (*DictionaryStaticType) elementSize() uint {
	return UnknownElementSize
}

func (t *DictionaryStaticType) String() string {
	return t.MeteredString(nil)
}

func (t *DictionaryStaticType) MeteredString(memoryGauge common.MemoryGauge) string {
	keyStr := t.KeyType.MeteredString(memoryGauge)
	valueStr := t.ValueType.MeteredString(memoryGauge)

	common.UseMemory(memoryGauge, common.DictionaryStaticTypeStringMemoryUsage)
	return fmt.Sprintf("{%s: %s}", keyStr, valueStr)
}

func (t *DictionaryStaticType) Equal(other StaticType) bool {
	otherDictionaryType, ok := other.(*DictionaryStaticType)
	if !ok {
		return false
	}

	return t.KeyType.Equal(otherDictionaryType.KeyType) &&
		t.ValueType.Equal(otherDictionaryType.ValueType)
}

func (t *DictionaryStaticType) ID() TypeID {
	return sema.FormatDictionaryTypeID(
		t.KeyType.ID(),
		t.ValueType.ID(),
	)
}

func (t *DictionaryStaticType) IsDeprecated() bool {
	return t.KeyType.IsDeprecated() ||
		t.ValueType.IsDeprecated()
}

// OptionalStaticType

type OptionalStaticType struct {
	Type StaticType
}

var _ StaticType = &OptionalStaticType{}

func NewOptionalStaticType(
	memoryGauge common.MemoryGauge,
	typ StaticType,
) *OptionalStaticType {
	common.UseMemory(memoryGauge, common.OptionalStaticTypeMemoryUsage)

	return &OptionalStaticType{Type: typ}
}

func (*OptionalStaticType) isStaticType() {}

func (*OptionalStaticType) elementSize() uint {
	return UnknownElementSize
}

func (t *OptionalStaticType) String() string {
	return t.MeteredString(nil)
}

func (t *OptionalStaticType) MeteredString(memoryGauge common.MemoryGauge) string {
	typeStr := t.Type.MeteredString(memoryGauge)

	common.UseMemory(memoryGauge, common.OptionalStaticTypeStringMemoryUsage)
	return fmt.Sprintf("%s?", typeStr)
}

func (t *OptionalStaticType) Equal(other StaticType) bool {
	otherOptionalType, ok := other.(*OptionalStaticType)
	if !ok {
		return false
	}

	return t.Type.Equal(otherOptionalType.Type)
}

func (t *OptionalStaticType) ID() TypeID {
	return sema.FormatOptionalTypeID(t.Type.ID())
}

func (t *OptionalStaticType) IsDeprecated() bool {
	return t.Type.IsDeprecated()
}

var NilStaticType = &OptionalStaticType{
	Type: PrimitiveStaticTypeNever,
}

// IntersectionStaticType

type IntersectionStaticType struct {
	Types      []*InterfaceStaticType
	LegacyType StaticType
}

var _ StaticType = &IntersectionStaticType{}

func NewIntersectionStaticType(
	memoryGauge common.MemoryGauge,
	types []*InterfaceStaticType,
) *IntersectionStaticType {
	common.UseMemory(memoryGauge, common.IntersectionStaticTypeMemoryUsage)

	return &IntersectionStaticType{
		Types: types,
	}
}

// NOTE: must be pointer receiver, as static types get used in type values,
// which are used as keys in maps when exporting.
// Key types in Go maps must be (transitively) hashable types,
// and slices are not, but `Types` is one.
func (*IntersectionStaticType) isStaticType() {}

func (*IntersectionStaticType) elementSize() uint {
	return UnknownElementSize
}

func (t *IntersectionStaticType) String() string {
	return t.MeteredString(nil)
}

func (t *IntersectionStaticType) MeteredString(memoryGauge common.MemoryGauge) string {
	common.UseMemory(memoryGauge, common.IntersectionStaticTypeStringMemoryUsage)

	var builder strings.Builder
	builder.WriteString("{")

	for i, typ := range t.Types {
		if i > 0 {
			common.UseMemory(memoryGauge, common.IntersectionStaticTypeSeparatorStringMemoryUsage)
			builder.WriteString(", ")
		}

		typeString := typ.MeteredString(memoryGauge)
		common.UseMemory(memoryGauge, common.NewRawStringMemoryUsage(len(typeString)))
		builder.WriteString(typeString)
	}

	builder.WriteString("}")

	return builder.String()
}

func (t *IntersectionStaticType) Equal(other StaticType) bool {
	otherIntersectionType, ok := other.(*IntersectionStaticType)
	if !ok || len(t.Types) != len(otherIntersectionType.Types) {
		return false
	}

outer:
	for _, typ := range t.Types {
		for _, otherType := range otherIntersectionType.Types {
			if typ.Equal(otherType) {
				continue outer
			}
		}

		return false
	}

	return true
}

func (t *IntersectionStaticType) ID() TypeID {
	var interfaceTypeIDs []TypeID
	typeCount := len(t.Types)
	if typeCount > 0 {
		interfaceTypeIDs = make([]TypeID, 0, typeCount)
		for _, ty := range t.Types {
			interfaceTypeIDs = append(interfaceTypeIDs, ty.ID())
		}
	}
	// FormatIntersectionTypeID sorts
	return sema.FormatIntersectionTypeID(interfaceTypeIDs)
}

func (t *IntersectionStaticType) IsDeprecated() bool {
	for _, typ := range t.Types {
		if typ.IsDeprecated() {
			return true
		}
	}

	return false
}

// Authorization

type Authorization interface {
	isAuthorization()
	String() string
	MeteredString(common.MemoryGauge) string
	Equal(auth Authorization) bool
	Encode(e *cbor.StreamEncoder) error
	ID() TypeID
}

type Unauthorized struct{}

var UnauthorizedAccess Authorization = Unauthorized{}

var FullyEntitledAccountAccess = ConvertSemaAccessToStaticAuthorization(nil, sema.FullyEntitledAccountAccess)

func (Unauthorized) isAuthorization() {}

func (Unauthorized) String() string {
	return ""
}

func (Unauthorized) MeteredString(_ common.MemoryGauge) string {
	return ""
}

func (Unauthorized) ID() TypeID {
	panic(errors.NewUnreachableError())
}

func (Unauthorized) Equal(auth Authorization) bool {
	_, ok := auth.(Unauthorized)
	return ok
}

type EntitlementSetAuthorization struct {
	Entitlements *sema.TypeIDOrderedSet
	SetKind      sema.EntitlementSetKind
}

var _ Authorization = EntitlementSetAuthorization{}

func NewEntitlementSetAuthorization(
	memoryGauge common.MemoryGauge,
	entitlementListConstructor func() []common.TypeID,
	entitlementListSize int,
	kind sema.EntitlementSetKind,
) EntitlementSetAuthorization {
	common.UseMemory(memoryGauge, common.MemoryUsage{
		Kind:   common.MemoryKindEntitlementSetStaticAccess,
		Amount: uint64(entitlementListSize),
	})

	entitlementList := entitlementListConstructor()
	if len(entitlementList) > entitlementListSize {
		// it should not be possible to reach this point unless something is implemented wrong
		panic(errors.NewUnreachableError())
	}

	entitlements := orderedmap.New[sema.TypeIDOrderedSet](len(entitlementList))
	for _, entitlement := range entitlementList {
		entitlements.Set(entitlement, struct{}{})
	}

	return EntitlementSetAuthorization{Entitlements: entitlements, SetKind: kind}
}

func (EntitlementSetAuthorization) isAuthorization() {}

func (a EntitlementSetAuthorization) ID() TypeID {
	entitlementTypeIDs := make([]TypeID, 0, a.Entitlements.Len())
	a.Entitlements.Foreach(func(typeID TypeID, _ struct{}) {
		entitlementTypeIDs = append(
			entitlementTypeIDs,
			typeID,
		)
	})

	return sema.FormatEntitlementSetTypeID(entitlementTypeIDs, a.SetKind)
}

func (a EntitlementSetAuthorization) String() string {
	return a.MeteredString(nil)
}

func (a EntitlementSetAuthorization) MeteredString(memoryGauge common.MemoryGauge) string {
	common.UseMemory(memoryGauge, common.AuthStringMemoryUsage)

	var builder strings.Builder
	builder.WriteString("auth(")
	var separator string

	switch a.SetKind {
	case sema.Conjunction:
		separator = ", "
	case sema.Disjunction:
		separator = " | "
	default:
		panic(errors.NewUnreachableError())
	}

	var i int
	a.Entitlements.Foreach(func(typeID common.TypeID, _ struct{}) {
		if i > 0 {
			common.UseMemory(memoryGauge, common.NewRawStringMemoryUsage(len(separator)))
			builder.WriteString(separator)
		}

		common.UseMemory(memoryGauge, common.NewRawStringMemoryUsage(len(typeID)))
		builder.WriteString(string(typeID))

		i++
	})

	builder.WriteString(") ")
	return builder.String()
}

func (a EntitlementSetAuthorization) Equal(auth Authorization) bool {
	// sets are equivalent if they contain the same elements, regardless of order
	if auth, ok := auth.(EntitlementSetAuthorization); ok {
		if a.SetKind != auth.SetKind {
			return false
		}
		if auth.Entitlements.Len() != a.Entitlements.Len() {
			return false
		}
		return auth.Entitlements.ForAllKeys(func(entitlement common.TypeID) bool {
			return a.Entitlements.Contains(entitlement)
		})
	}
	return false
}

type EntitlementMapAuthorization struct {
	TypeID common.TypeID
}

var _ Authorization = EntitlementMapAuthorization{}

func NewEntitlementMapAuthorization(memoryGauge common.MemoryGauge, id common.TypeID) EntitlementMapAuthorization {
	common.UseMemory(memoryGauge, common.EntitlementMapStaticTypeMemoryUsage)

	return EntitlementMapAuthorization{TypeID: id}
}

func (EntitlementMapAuthorization) isAuthorization() {}

func (a EntitlementMapAuthorization) String() string {
	return a.MeteredString(nil)
}

func (a EntitlementMapAuthorization) MeteredString(memoryGauge common.MemoryGauge) string {
	common.UseMemory(memoryGauge, common.AuthStringMemoryUsage)
	common.UseMemory(memoryGauge, common.NewRawStringMemoryUsage(len(a.TypeID)))
	return fmt.Sprintf("auth(%s) ", a.TypeID)
}

func (a EntitlementMapAuthorization) ID() TypeID {
	return a.TypeID
}

func (a EntitlementMapAuthorization) Equal(other Authorization) bool {
	auth, ok := other.(EntitlementMapAuthorization)
	if !ok {
		return false
	}
	return a.TypeID == auth.TypeID
}

// ReferenceStaticType

type ReferenceStaticType struct {
	Authorization Authorization
	// ReferencedType is type of the referenced value (the type of the target)
	ReferencedType     StaticType
	LegacyIsAuthorized bool
}

var _ StaticType = &ReferenceStaticType{}

func NewReferenceStaticType(
	memoryGauge common.MemoryGauge,
	authorization Authorization,
	referencedType StaticType,
) *ReferenceStaticType {
	common.UseMemory(memoryGauge, common.ReferenceStaticTypeMemoryUsage)

	return &ReferenceStaticType{
		Authorization:  authorization,
		ReferencedType: referencedType,
	}
}

func (*ReferenceStaticType) isStaticType() {}

func (*ReferenceStaticType) elementSize() uint {
	return UnknownElementSize
}

func (t *ReferenceStaticType) String() string {
	return t.MeteredString(nil)
}

func (t *ReferenceStaticType) MeteredString(memoryGauge common.MemoryGauge) string {
	typeStr := t.ReferencedType.MeteredString(memoryGauge)
	authString := t.Authorization.MeteredString(memoryGauge)

	common.UseMemory(memoryGauge, common.NewRawStringMemoryUsage(len(typeStr)+1+len(authString)))
	return fmt.Sprintf("%s&%s", authString, typeStr)
}

func (t *ReferenceStaticType) Equal(other StaticType) bool {
	otherReferenceType, ok := other.(*ReferenceStaticType)
	if !ok {
		return false
	}

	return t.Authorization.Equal(otherReferenceType.Authorization) &&
		t.ReferencedType.Equal(otherReferenceType.ReferencedType)
}

func (t *ReferenceStaticType) ID() TypeID {
	var authorization TypeID
	if t.Authorization != UnauthorizedAccess {
		authorization = t.Authorization.ID()
	}
	return sema.FormatReferenceTypeID(
		authorization,
		t.ReferencedType.ID(),
	)
}

func (t *ReferenceStaticType) IsDeprecated() bool {
	return t.ReferencedType.IsDeprecated()
}

// CapabilityStaticType

type CapabilityStaticType struct {
	BorrowType StaticType
}

var _ StaticType = &CapabilityStaticType{}

func NewCapabilityStaticType(
	memoryGauge common.MemoryGauge,
	borrowType StaticType,
) *CapabilityStaticType {
	common.UseMemory(memoryGauge, common.CapabilityStaticTypeMemoryUsage)

	return &CapabilityStaticType{
		BorrowType: borrowType,
	}
}

func (*CapabilityStaticType) isStaticType() {}

func (*CapabilityStaticType) elementSize() uint {
	return UnknownElementSize
}

func (t *CapabilityStaticType) String() string {
	return t.MeteredString(nil)
}

func (t *CapabilityStaticType) MeteredString(memoryGauge common.MemoryGauge) string {
	if t.BorrowType != nil {
		typeStr := t.BorrowType.MeteredString(memoryGauge)

		common.UseMemory(memoryGauge, common.CapabilityStaticTypeStringMemoryUsage)
		return fmt.Sprintf("Capability<%s>", typeStr)
	}

	return "Capability"
}

func (t *CapabilityStaticType) Equal(other StaticType) bool {
	otherCapabilityType, ok := other.(*CapabilityStaticType)
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

func (t *CapabilityStaticType) ID() TypeID {
	var borrowTypeID TypeID
	borrowType := t.BorrowType
	if borrowType != nil {
		borrowTypeID = borrowType.ID()
	}
	return sema.FormatCapabilityTypeID(borrowTypeID)
}

func (t *CapabilityStaticType) IsDeprecated() bool {
	if t.BorrowType == nil {
		return false
	}
	return t.BorrowType.IsDeprecated()
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

	case *sema.IntersectionType:
		var intersectedTypes []*InterfaceStaticType
		typeCount := len(t.Types)
		if typeCount > 0 {
			intersectedTypes = make([]*InterfaceStaticType, typeCount)

			for i, typ := range t.Types {
				intersectedTypes[i] = ConvertSemaInterfaceTypeToStaticInterfaceType(memoryGauge, typ)
			}
		}

		return NewIntersectionStaticType(
			memoryGauge,
			intersectedTypes,
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

	case *sema.InclusiveRangeType:
		memberType := ConvertSemaToStaticType(memoryGauge, t.MemberType)
		return NewInclusiveRangeStaticType(memoryGauge, memberType)

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
		return NewVariableSizedStaticType(
			memoryGauge,
			ConvertSemaToStaticType(memoryGauge, t.Type),
		)

	case *sema.ConstantSizedType:
		return NewConstantSizedStaticType(
			memoryGauge,
			ConvertSemaToStaticType(memoryGauge, t.Type),
			t.Size,
		)

	default:
		panic(errors.NewUnreachableError())
	}
}

func ConvertSemaDictionaryTypeToStaticDictionaryType(
	memoryGauge common.MemoryGauge,
	t *sema.DictionaryType,
) *DictionaryStaticType {
	return NewDictionaryStaticType(
		memoryGauge,
		ConvertSemaToStaticType(memoryGauge, t.KeyType),
		ConvertSemaToStaticType(memoryGauge, t.ValueType),
	)
}

func ConvertSemaAccessToStaticAuthorization(
	memoryGauge common.MemoryGauge,
	access sema.Access,
) Authorization {
	switch access := access.(type) {
	case sema.PrimitiveAccess:
		if access.Equal(sema.UnauthorizedAccess) {
			return UnauthorizedAccess
		}

	case sema.EntitlementSetAccess:
		var entitlements []common.TypeID
		access.Entitlements.Foreach(func(key *sema.EntitlementType, _ struct{}) {
			typeId := key.ID()
			entitlements = append(entitlements, typeId)
		})
		return NewEntitlementSetAuthorization(
			memoryGauge,
			func() (entitlements []common.TypeID) {
				access.Entitlements.Foreach(func(key *sema.EntitlementType, _ struct{}) {
					typeId := key.ID()
					entitlements = append(entitlements, typeId)
				})
				return
			},
			access.Entitlements.Len(),
			access.SetKind,
		)

	case *sema.EntitlementMapAccess:
		typeId := access.Type.ID()
		return NewEntitlementMapAuthorization(memoryGauge, typeId)
	}
	panic(errors.NewUnreachableError())
}

func ConvertSemaReferenceTypeToStaticReferenceType(
	memoryGauge common.MemoryGauge,
	t *sema.ReferenceType,
) *ReferenceStaticType {
	return NewReferenceStaticType(
		memoryGauge,
		ConvertSemaAccessToStaticAuthorization(memoryGauge, t.Authorization),
		ConvertSemaToStaticType(memoryGauge, t.Type),
	)
}

func ConvertSemaCompositeTypeToStaticCompositeType(
	memoryGauge common.MemoryGauge,
	t *sema.CompositeType,
) *CompositeStaticType {
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
) *InterfaceStaticType {
	return NewInterfaceStaticType(
		memoryGauge,
		t.Location,
		t.QualifiedIdentifier(),
		t.ID(),
	)
}

func ConvertStaticAuthorizationToSemaAccess(
	memoryGauge common.MemoryGauge,
	auth Authorization,
	getEntitlement func(typeID common.TypeID) (*sema.EntitlementType, error),
	getEntitlementMapType func(typeID common.TypeID) (*sema.EntitlementMapType, error),
) (sema.Access, error) {
	switch auth := auth.(type) {
	case Unauthorized:
		return sema.UnauthorizedAccess, nil
	case EntitlementMapAuthorization:
		entitlement, err := getEntitlementMapType(auth.TypeID)
		if err != nil {
			return nil, err
		}
		return sema.NewEntitlementMapAccess(entitlement), nil

	case EntitlementSetAuthorization:
		var entitlements []*sema.EntitlementType
		err := auth.Entitlements.ForeachWithError(func(id common.TypeID, value struct{}) error {
			entitlement, err := getEntitlement(id)
			if err != nil {
				return err
			}
			entitlements = append(entitlements, entitlement)
			return nil
		})
		if err != nil {
			return nil, err
		}
		return sema.NewEntitlementSetAccess(entitlements, auth.SetKind), nil
	}

	panic(errors.NewUnreachableError())
}

func ConvertStaticToSemaType(
	memoryGauge common.MemoryGauge,
	typ StaticType,
	getInterface func(location common.Location, qualifiedIdentifier string, typeID TypeID) (*sema.InterfaceType, error),
	getComposite func(location common.Location, qualifiedIdentifier string, typeID TypeID) (*sema.CompositeType, error),
	getEntitlement func(typeID TypeID) (*sema.EntitlementType, error),
	getEntitlementMapType func(typeID TypeID) (*sema.EntitlementMapType, error),
) (_ sema.Type, err error) {
	switch t := typ.(type) {
	case *CompositeStaticType:
		return getComposite(t.Location, t.QualifiedIdentifier, t.TypeID)

	case *InterfaceStaticType:
		return getInterface(t.Location, t.QualifiedIdentifier, t.TypeID)

	case *VariableSizedStaticType:
		ty, err := ConvertStaticToSemaType(
			memoryGauge,
			t.Type,
			getInterface,
			getComposite,
			getEntitlement,
			getEntitlementMapType,
		)
		if err != nil {
			return nil, err
		}
		return sema.NewVariableSizedType(memoryGauge, ty), nil

	case *ConstantSizedStaticType:
		ty, err := ConvertStaticToSemaType(
			memoryGauge,
			t.Type,
			getInterface,
			getComposite,
			getEntitlement,
			getEntitlementMapType,
		)
		if err != nil {
			return nil, err
		}

		return sema.NewConstantSizedType(
			memoryGauge,
			ty,
			t.Size,
		), nil

	case *DictionaryStaticType:
		keyType, err := ConvertStaticToSemaType(
			memoryGauge,
			t.KeyType,
			getInterface,
			getComposite,
			getEntitlement,
			getEntitlementMapType,
		)
		if err != nil {
			return nil, err
		}

		valueType, err := ConvertStaticToSemaType(
			memoryGauge,
			t.ValueType,
			getInterface,
			getComposite,
			getEntitlement,
			getEntitlementMapType,
		)
		if err != nil {
			return nil, err
		}

		return sema.NewDictionaryType(
			memoryGauge,
			keyType,
			valueType,
		), nil

	case InclusiveRangeStaticType:
		elementType, err := ConvertStaticToSemaType(
			memoryGauge,
			t.ElementType,
			getInterface,
			getComposite,
			getEntitlement,
			getEntitlementMapType,
		)
		if err != nil {
			return nil, err
		}

		return sema.NewInclusiveRangeType(
			memoryGauge,
			elementType,
		), nil

	case *OptionalStaticType:
		ty, err := ConvertStaticToSemaType(
			memoryGauge,
			t.Type,
			getInterface,
			getComposite,
			getEntitlement,
			getEntitlementMapType,
		)
		if err != nil {
			return nil, err
		}
		return sema.NewOptionalType(memoryGauge, ty), err

	case *IntersectionStaticType:
		var intersectedTypes []*sema.InterfaceType

		typeCount := len(t.Types)
		if typeCount > 0 {
			intersectedTypes = make([]*sema.InterfaceType, typeCount)

			for i, typ := range t.Types {
				intersectedTypes[i], err = getInterface(typ.Location, typ.QualifiedIdentifier, typ.TypeID)
				if err != nil {
					return nil, err
				}
			}
		}

		return sema.NewIntersectionType(
			memoryGauge,
			intersectedTypes,
		), nil

	case *ReferenceStaticType:
		ty, err := ConvertStaticToSemaType(
			memoryGauge,
			t.ReferencedType,
			getInterface,
			getComposite,
			getEntitlement,
			getEntitlementMapType,
		)
		if err != nil {
			return nil, err
		}

		access, err := ConvertStaticAuthorizationToSemaAccess(
			memoryGauge,
			t.Authorization,
			getEntitlement,
			getEntitlementMapType,
		)

		if err != nil {
			return nil, err
		}

		return sema.NewReferenceType(memoryGauge, access, ty), nil

	case *CapabilityStaticType:
		var borrowType sema.Type
		if t.BorrowType != nil {
			borrowType, err = ConvertStaticToSemaType(
				memoryGauge,
				t.BorrowType,
				getInterface,
				getComposite,
				getEntitlement,
				getEntitlementMapType,
			)
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

func (t FunctionStaticType) ReturnType(gauge common.MemoryGauge) StaticType {
	var returnType StaticType
	if t.Type.ReturnTypeAnnotation.Type != nil {
		returnType = ConvertSemaToStaticType(gauge, t.Type.ReturnTypeAnnotation.Type)
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

func (FunctionStaticType) IsDeprecated() bool {
	return false
}

// TypeParameter

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
