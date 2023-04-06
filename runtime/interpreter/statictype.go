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
	"golang.org/x/exp/slices"

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
	common.UseMemory(memoryGauge, common.CompositeStaticTypeMemoryUsage)

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
	typeID := common.NewTypeIDFromQualifiedName(memoryGauge, location, qualifiedIdentifier)

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
	common.UseMemory(memoryGauge, common.InterfaceStaticTypeMemoryUsage)

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

func (t InterfaceStaticType) MeteredString(memoryGauge common.MemoryGauge) string {
	if t.Location == nil {
		return t.QualifiedIdentifier
	}
	return string(t.Location.TypeID(memoryGauge, t.QualifiedIdentifier))
}

func (t InterfaceStaticType) Equal(other StaticType) bool {
	otherInterfaceType, ok := other.(InterfaceStaticType)
	if !ok {
		return false
	}

	return otherInterfaceType.Location == t.Location &&
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

func (RestrictedStaticType) elementSize() uint {
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

// Authorization

type Authorization interface {
	isAuthorization()
	String() string
	Equal(auth Authorization) bool
	Encode(e *cbor.StreamEncoder) error
}

type Unauthorized struct{}

var UnauthorizedAccess Unauthorized = Unauthorized{}

func (Unauthorized) isAuthorization() {}

func (Unauthorized) String() string {
	return ""
}

func (Unauthorized) Equal(auth Authorization) bool {
	switch auth.(type) {
	case Unauthorized:
		return true
	}
	return false
}

type EntitlementSetAuthorization struct {
	Entitlements []common.TypeID
}

func NewEntitlementSetAuthorization(memoryGauge common.MemoryGauge, entitlements []common.TypeID) EntitlementSetAuthorization {
	common.UseMemory(memoryGauge, common.MemoryUsage{
		Kind:   common.MemoryKindEntitlementSetStaticAccess,
		Amount: uint64(len(entitlements)),
	})

	return EntitlementSetAuthorization{Entitlements: entitlements}
}

func (EntitlementSetAuthorization) isAuthorization() {}

func (e EntitlementSetAuthorization) String() string {
	var builder strings.Builder
	builder.WriteString("auth(")

	for i, entitlement := range e.Entitlements {
		builder.WriteString(string(entitlement))
		if i < len(e.Entitlements) {
			builder.WriteString(", ")
		}
	}
	builder.WriteString(") ")
	return builder.String()
}

func (e EntitlementSetAuthorization) Equal(auth Authorization) bool {
	// sets are equivalent if they contain the same elements, regardless of order
	switch auth := auth.(type) {
	case EntitlementSetAuthorization:
		for _, entitlement := range e.Entitlements {
			if !slices.Contains(auth.Entitlements, entitlement) {
				return false
			}
		}
		for _, entitlement := range auth.Entitlements {
			if !slices.Contains(e.Entitlements, entitlement) {
				return false
			}
		}
		return true
	}
	return false
}

type EntitlementMapAuthorization struct {
	TypeID common.TypeID
}

func NewEntitlementMapAuthorization(memoryGauge common.MemoryGauge, id common.TypeID) EntitlementMapAuthorization {
	common.UseMemory(memoryGauge, common.NewConstantMemoryUsage(common.MemoryKindEntitlementMapStaticAccess))

	return EntitlementMapAuthorization{TypeID: id}
}

func (EntitlementMapAuthorization) isAuthorization() {}

func (e EntitlementMapAuthorization) String() string {
	return fmt.Sprintf("auth(%s) ", e.TypeID)
}

func (e EntitlementMapAuthorization) Equal(auth Authorization) bool {
	switch auth := auth.(type) {
	case EntitlementMapAuthorization:
		return e.TypeID == auth.TypeID
	}
	return false
}

// ReferenceStaticType

type ReferenceStaticType struct {
	// ReferencedType is type of the referenced value (the type of the target)
	ReferencedType StaticType
	Authorization  Authorization
}

var _ StaticType = ReferenceStaticType{}

func NewReferenceStaticType(
	memoryGauge common.MemoryGauge,
	authorization Authorization,
	referencedType StaticType,
) ReferenceStaticType {
	common.UseMemory(memoryGauge, common.ReferenceStaticTypeMemoryUsage)

	return ReferenceStaticType{
		Authorization:  authorization,
		ReferencedType: referencedType,
	}
}

func (ReferenceStaticType) isStaticType() {}

func (ReferenceStaticType) elementSize() uint {
	return UnknownElementSize
}

func (t ReferenceStaticType) String() string {
	auth := t.Authorization.String()
	return fmt.Sprintf("%s&%s", auth, t.ReferencedType)
}

func (t ReferenceStaticType) MeteredString(memoryGauge common.MemoryGauge) string {

	typeStr := t.ReferencedType.MeteredString(memoryGauge)
	authString := t.Authorization.String()
	memoryGauge.MeterMemory(common.NewRawStringMemoryUsage(len(authString)))
	return fmt.Sprintf("%s&%s", authString, typeStr)
}

func (t ReferenceStaticType) Equal(other StaticType) bool {
	otherReferenceType, ok := other.(ReferenceStaticType)
	if !ok {
		return false
	}

	return t.Authorization.Equal(otherReferenceType.Authorization) &&
		t.ReferencedType.Equal(otherReferenceType.ReferencedType)
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

func ConvertSemaAccesstoStaticAuthorization(
	memoryGauge common.MemoryGauge,
	access sema.Access,
) Authorization {
	switch access := access.(type) {
	case sema.PrimitiveAccess:
		if access.Equal(sema.UnauthorizedAccess) {
			return UnauthorizedAccess
		}

	case sema.EntitlementSetAccess:
		if access.SetKind != sema.Conjunction {
			// disjoint entitlement sets cannot exist at runtime; this should be unreachable
			panic(InvalidDisjointRuntimeEntitlementSetCreationError{
				Authorization: access,
			})
		}
		var entitlements []common.TypeID
		access.Entitlements.Foreach(func(key *sema.EntitlementType, _ struct{}) {
			entitlements = append(entitlements, key.Location.TypeID(memoryGauge, key.QualifiedIdentifier()))
		})
		return NewEntitlementSetAuthorization(memoryGauge, entitlements)

	case sema.EntitlementMapAccess:
		return NewEntitlementMapAuthorization(memoryGauge, access.Type.Location.TypeID(memoryGauge, access.Type.QualifiedIdentifier()))
	}
	panic(errors.NewUnreachableError())
}

func ConvertSemaReferenceTypeToStaticReferenceType(
	memoryGauge common.MemoryGauge,
	t *sema.ReferenceType,
) ReferenceStaticType {
	return NewReferenceStaticType(
		memoryGauge,
		ConvertSemaAccesstoStaticAuthorization(memoryGauge, t.Authorization),
		ConvertSemaToStaticType(memoryGauge, t.Type),
	)
}

func ConvertSemaInterfaceTypeToStaticInterfaceType(
	memoryGauge common.MemoryGauge,
	t *sema.InterfaceType,
) InterfaceStaticType {
	return NewInterfaceStaticType(memoryGauge, t.Location, t.QualifiedIdentifier())
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
		for _, id := range auth.Entitlements {
			entitlement, err := getEntitlement(id)
			if err != nil {
				return nil, err
			}
			entitlements = append(entitlements, entitlement)
		}
		// only conjunction sets can actually exist at runtime
		return sema.NewEntitlementSetAccess(entitlements, sema.Conjunction), nil
	}
	panic(errors.NewUnreachableError())
}

func ConvertStaticToSemaType(
	memoryGauge common.MemoryGauge,
	typ StaticType,
	getInterface func(location common.Location, qualifiedIdentifier string) (*sema.InterfaceType, error),
	getComposite func(location common.Location, qualifiedIdentifier string, typeID common.TypeID) (*sema.CompositeType, error),
	getEntitlement func(typeID common.TypeID) (*sema.EntitlementType, error),
	getEntitlementMapType func(typeID common.TypeID) (*sema.EntitlementMapType, error),
) (_ sema.Type, err error) {
	switch t := typ.(type) {
	case CompositeStaticType:
		return getComposite(t.Location, t.QualifiedIdentifier, t.TypeID)

	case InterfaceStaticType:
		return getInterface(t.Location, t.QualifiedIdentifier)

	case VariableSizedStaticType:
		ty, err := ConvertStaticToSemaType(memoryGauge, t.Type, getInterface, getComposite, getEntitlement, getEntitlementMapType)
		if err != nil {
			return nil, err
		}
		return sema.NewVariableSizedType(memoryGauge, ty), nil

	case ConstantSizedStaticType:
		ty, err := ConvertStaticToSemaType(memoryGauge, t.Type, getInterface, getComposite, getEntitlement, getEntitlementMapType)
		if err != nil {
			return nil, err
		}

		return sema.NewConstantSizedType(
			memoryGauge,
			ty,
			t.Size,
		), nil

	case DictionaryStaticType:
		keyType, err := ConvertStaticToSemaType(memoryGauge, t.KeyType, getInterface, getComposite, getEntitlement, getEntitlementMapType)
		if err != nil {
			return nil, err
		}

		valueType, err := ConvertStaticToSemaType(memoryGauge, t.ValueType, getInterface, getComposite, getEntitlement, getEntitlementMapType)
		if err != nil {
			return nil, err
		}

		return sema.NewDictionaryType(
			memoryGauge,
			keyType,
			valueType,
		), nil

	case OptionalStaticType:
		ty, err := ConvertStaticToSemaType(memoryGauge, t.Type, getInterface, getComposite, getEntitlement, getEntitlementMapType)
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

		ty, err := ConvertStaticToSemaType(memoryGauge, t.Type, getInterface, getComposite, getEntitlement, getEntitlementMapType)
		if err != nil {
			return nil, err
		}

		return sema.NewRestrictedType(
			memoryGauge,
			ty,
			restrictions,
		), nil

	case ReferenceStaticType:
		ty, err := ConvertStaticToSemaType(memoryGauge, t.ReferencedType, getInterface, getComposite, getEntitlement, getEntitlementMapType)
		if err != nil {
			return nil, err
		}

		access, err := ConvertStaticAuthorizationToSemaAccess(memoryGauge, t.Authorization, getEntitlement, getEntitlementMapType)

		if err != nil {
			return nil, err
		}

		return sema.NewReferenceType(
			memoryGauge,
			ty,
			access,
		), nil

	case CapabilityStaticType:
		var borrowType sema.Type
		if t.BorrowType != nil {
			borrowType, err = ConvertStaticToSemaType(memoryGauge, t.BorrowType, getInterface, getComposite, getEntitlement, getEntitlementMapType)
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
