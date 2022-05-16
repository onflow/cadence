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

package sema

import (
	"fmt"
	"math"
	"math/big"
	"strings"
	"sync"

	"github.com/onflow/cadence/fixedpoint"
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
)

func qualifiedIdentifier(identifier string, containerType Type) string {
	if containerType == nil {
		return identifier
	}

	// Gather all identifiers: this, parent, grand-parent, etc.
	const level = 0
	identifiers, bufSize := containerTypeNames(containerType, level+1)

	identifiers[level] = identifier
	bufSize += len(identifier)

	// Append all identifiers, in reverse order
	var sb strings.Builder

	// Grow the buffer at once.
	//
	// bytes needed for separator '.'
	// i.e: 1 x (length of identifiers - 1)
	bufSize += len(identifiers) - 1
	sb.Grow(bufSize)

	for i := len(identifiers) - 1; i >= 0; i-- {
		sb.WriteString(identifiers[i])
		if i != 0 {
			sb.WriteRune('.')
		}
	}

	return sb.String()
}

func containerTypeNames(typ Type, level int) (typeNames []string, bufSize int) {
	if typ == nil {
		return make([]string, level), 0
	}

	var typeName string
	var containerType Type

	switch typedContainerType := typ.(type) {
	case *InterfaceType:
		typeName = typedContainerType.Identifier
		containerType = typedContainerType.containerType
	case *CompositeType:
		typeName = typedContainerType.Identifier
		containerType = typedContainerType.containerType
	default:
		panic(errors.NewUnreachableError())
	}

	typeNames, bufSize = containerTypeNames(containerType, level+1)

	typeNames[level] = typeName
	bufSize += len(typeName)

	return typeNames, bufSize
}

type TypeID = common.TypeID

type Type interface {
	IsType()
	ID() TypeID
	Tag() TypeTag
	String() string
	QualifiedString() string
	Equal(other Type) bool

	// IsResourceType returns true if the type is itself a resource (a `CompositeType` with resource kind),
	// or it contains a resource type (e.g. for optionals, arrays, dictionaries, etc.)
	IsResourceType() bool

	// IsInvalidType returns true if the type is itself the invalid type (see `InvalidType`),
	// or it contains an invalid type (e.g. for optionals, arrays, dictionaries, etc.)
	IsInvalidType() bool

	// IsStorable returns true if the type is allowed to be a stored,
	// e.g. in a field of a composite type.
	//
	// The check if the type is storable is recursive,
	// the results parameter prevents cycles:
	// it is checked at the start of the recursively called function,
	// and pre-set before a recursive call.
	IsStorable(results map[*Member]bool) bool

	// IsExternallyReturnable returns true if a value of this type can be exported
	//
	// The check if the type is externally returnable is recursive,
	// the results parameter prevents cycles:
	// it is checked at the start of the recursively called function,
	// and pre-set before a recursive call.
	IsExternallyReturnable(results map[*Member]bool) bool

	// IsImportable returns true if values of the type can be imported to a program as arguments
	IsImportable(results map[*Member]bool) bool

	// IsEquatable returns true if values of the type can be equated
	IsEquatable() bool

	TypeAnnotationState() TypeAnnotationState
	RewriteWithRestrictedTypes() (result Type, rewritten bool)

	// Unify attempts to unify the given type with this type, i.e., resolve type parameters
	// in generic types (see `GenericType`) using the given type parameters.
	//
	// For a generic type, unification assigns a given type with a type parameter.
	//
	// If the type parameter has not been previously unified with a type,
	// through an explicitly provided type argument in an invocation
	// or through a previous unification, the type parameter is assigned the given type.
	//
	// If the type parameter has already been previously unified with a type,
	// the type parameter's unified .
	//
	// The boolean return value indicates if a generic type was encountered during unification.
	// For primitives (e.g. `Int`, `String`, etc.) it would be false, as .
	// For types with nested types (e.g. optionals, arrays, and dictionaries)
	// the result is the successful unification of the inner types.
	//
	// The boolean return value does *not* indicate if unification succeeded or not.
	//
	Unify(
		other Type,
		typeParameters *TypeParameterTypeOrderedMap,
		report func(err error),
		outerRange ast.Range,
	) bool

	// Resolve returns a type that is free of generic types (see `GenericType`),
	// i.e. it resolves the type parameters in generic types given the type parameter
	// unifications of `typeParameters`.
	//
	// If resolution fails, it returns `nil`.
	//
	Resolve(typeArguments *TypeParameterTypeOrderedMap) Type

	GetMembers() map[string]MemberResolver
}

// ValueIndexableType is a type which can be indexed into using a value
//
type ValueIndexableType interface {
	Type
	isValueIndexableType() bool
	AllowsValueIndexingAssignment() bool
	ElementType(isAssignment bool) Type
	IndexingType() Type
}

type MemberResolver struct {
	Kind     common.DeclarationKind
	Mutating bool
	Resolve  func(
		memoryGauge common.MemoryGauge,
		identifier string,
		targetRange ast.Range,
		report func(error),
	) *Member
}

// ContainedType is a type which might have a container type
//
type ContainedType interface {
	Type
	GetContainerType() Type
	SetContainerType(containerType Type)
}

// ContainerType is a type which might have nested types
//
type ContainerType interface {
	Type
	IsContainerType() bool
	GetNestedTypes() *StringTypeOrderedMap
}

func VisitThisAndNested(t Type, visit func(ty Type)) {
	visit(t)

	containerType, ok := t.(ContainerType)
	if !ok || !containerType.IsContainerType() {
		return
	}

	containerType.GetNestedTypes().Foreach(func(_ string, nestedType Type) {
		VisitThisAndNested(nestedType, visit)
	})
}

// CompositeKindedType is a type which has a composite kind
//
type CompositeKindedType interface {
	Type
	GetCompositeKind() common.CompositeKind
}

// LocatedType is a type which has a location
//
type LocatedType interface {
	Type
	GetLocation() common.Location
}

// ParameterizedType is a type which might have type parameters
//
type ParameterizedType interface {
	Type
	TypeParameters() []*TypeParameter
	Instantiate(typeArguments []Type, report func(err error)) Type
	BaseType() Type
	TypeArguments() []Type
}

// TypeAnnotation

type TypeAnnotation struct {
	IsResource bool
	Type       Type
}

func (a *TypeAnnotation) TypeAnnotationState() TypeAnnotationState {
	if a.Type.IsInvalidType() {
		return TypeAnnotationStateValid
	}

	innerState := a.Type.TypeAnnotationState()
	if innerState != TypeAnnotationStateValid {
		return innerState
	}

	isResourceType := a.Type.IsResourceType()
	switch {
	case isResourceType && !a.IsResource:
		return TypeAnnotationStateMissingResourceAnnotation
	case !isResourceType && a.IsResource:
		return TypeAnnotationStateInvalidResourceAnnotation
	default:
		return TypeAnnotationStateValid
	}
}

func (a *TypeAnnotation) String() string {
	if a.IsResource {
		return fmt.Sprintf(
			"%s%s",
			common.CompositeKindResource.Annotation(),
			a.Type,
		)
	} else {
		return fmt.Sprint(a.Type)
	}
}

func (a *TypeAnnotation) QualifiedString() string {
	qualifiedString := a.Type.QualifiedString()
	if a.IsResource {
		return fmt.Sprintf(
			"%s%s",
			common.CompositeKindResource.Annotation(),
			qualifiedString,
		)
	} else {
		return fmt.Sprint(qualifiedString)
	}
}

func (a *TypeAnnotation) Equal(other *TypeAnnotation) bool {
	return a.IsResource == other.IsResource &&
		a.Type.Equal(other.Type)
}

func NewTypeAnnotation(ty Type) *TypeAnnotation {
	return &TypeAnnotation{
		IsResource: ty.IsResourceType(),
		Type:       ty,
	}
}

// isInstance

const IsInstanceFunctionName = "isInstance"

var IsInstanceFunctionType = &FunctionType{
	Parameters: []*Parameter{
		{
			Label:      ArgumentLabelNotRequired,
			Identifier: "type",
			TypeAnnotation: NewTypeAnnotation(
				MetaType,
			),
		},
	},
	ReturnTypeAnnotation: NewTypeAnnotation(
		BoolType,
	),
}

const isInstanceFunctionDocString = `
Returns true if the object conforms to the given type at runtime
`

// getType

const GetTypeFunctionName = "getType"

var GetTypeFunctionType = &FunctionType{
	ReturnTypeAnnotation: NewTypeAnnotation(
		MetaType,
	),
}

const getTypeFunctionDocString = `
Returns the type of the value
`

// toString

const ToStringFunctionName = "toString"

var ToStringFunctionType = &FunctionType{
	ReturnTypeAnnotation: NewTypeAnnotation(
		StringType,
	),
}

const toStringFunctionDocString = `
A textual representation of this object
`

// toBigEndianBytes

const ToBigEndianBytesFunctionName = "toBigEndianBytes"

var toBigEndianBytesFunctionType = &FunctionType{
	ReturnTypeAnnotation: NewTypeAnnotation(
		ByteArrayType,
	),
}

const toBigEndianBytesFunctionDocString = `
Returns an array containing the big-endian byte representation of the number
`

func withBuiltinMembers(ty Type, members map[string]MemberResolver) map[string]MemberResolver {
	if members == nil {
		members = map[string]MemberResolver{}
	}

	// All types have a predeclared member `fun isInstance(_ type: Type): Bool`

	members[IsInstanceFunctionName] = MemberResolver{
		Kind: common.DeclarationKindFunction,
		Resolve: func(memoryGauge common.MemoryGauge, identifier string, _ ast.Range, _ func(error)) *Member {
			return NewPublicFunctionMember(
				memoryGauge,
				ty,
				identifier,
				IsInstanceFunctionType,
				isInstanceFunctionDocString,
			)
		},
	}

	// All types have a predeclared member `fun getType(): Type`

	members[GetTypeFunctionName] = MemberResolver{
		Kind: common.DeclarationKindFunction,
		Resolve: func(memoryGauge common.MemoryGauge, identifier string, _ ast.Range, _ func(error)) *Member {
			return NewPublicFunctionMember(
				memoryGauge,
				ty,
				identifier,
				GetTypeFunctionType,
				getTypeFunctionDocString,
			)
		},
	}

	// All number types, addresses, and path types have a `toString` function

	if IsSubType(ty, NumberType) || IsSubType(ty, &AddressType{}) || IsSubType(ty, PathType) {

		members[ToStringFunctionName] = MemberResolver{
			Kind: common.DeclarationKindFunction,
			Resolve: func(memoryGauge common.MemoryGauge, identifier string, _ ast.Range, _ func(error)) *Member {
				return NewPublicFunctionMember(
					memoryGauge,
					ty,
					identifier,
					ToStringFunctionType,
					toStringFunctionDocString,
				)
			},
		}
	}

	// All number types have a `toBigEndianBytes` function

	if IsSubType(ty, NumberType) {

		members[ToBigEndianBytesFunctionName] = MemberResolver{
			Kind: common.DeclarationKindFunction,
			Resolve: func(memoryGauge common.MemoryGauge, identifier string, _ ast.Range, _ func(error)) *Member {
				return NewPublicFunctionMember(
					memoryGauge,
					ty,
					identifier,
					toBigEndianBytesFunctionType,
					toBigEndianBytesFunctionDocString,
				)
			},
		}
	}

	return members
}

// OptionalType represents the optional variant of another type
type OptionalType struct {
	Type Type
}

func NewOptionalType(memoryGauge common.MemoryGauge, typ Type) *OptionalType {
	common.UseMemory(memoryGauge, common.OptionalSemaTypeMemoryUsage)
	return &OptionalType{
		Type: typ,
	}
}

func (*OptionalType) IsType() {}

func (t *OptionalType) Tag() TypeTag {
	if t.Type == NeverType {
		return NilTypeTag
	}

	return t.Type.Tag().Or(NilTypeTag)
}

func (t *OptionalType) String() string {
	if t.Type == nil {
		return "optional"
	}
	return fmt.Sprintf("%s?", t.Type)
}

func (t *OptionalType) QualifiedString() string {
	if t.Type == nil {
		return "optional"
	}
	return fmt.Sprintf("%s?", t.Type.QualifiedString())
}

func (t *OptionalType) ID() TypeID {
	var id string
	if t.Type != nil {
		id = string(t.Type.ID())
	}
	return TypeID(fmt.Sprintf("%s?", id))
}

func (t *OptionalType) Equal(other Type) bool {
	otherOptional, ok := other.(*OptionalType)
	if !ok {
		return false
	}
	return t.Type.Equal(otherOptional.Type)
}

func (t *OptionalType) IsResourceType() bool {
	return t.Type.IsResourceType()
}

func (t *OptionalType) IsInvalidType() bool {
	return t.Type.IsInvalidType()
}

func (t *OptionalType) IsStorable(results map[*Member]bool) bool {
	return t.Type.IsStorable(results)
}

func (t *OptionalType) IsExternallyReturnable(results map[*Member]bool) bool {
	return t.Type.IsExternallyReturnable(results)
}

func (t *OptionalType) IsImportable(results map[*Member]bool) bool {
	return t.Type.IsImportable(results)
}

func (t *OptionalType) IsEquatable() bool {
	return t.Type.IsEquatable()
}

func (t *OptionalType) TypeAnnotationState() TypeAnnotationState {
	return t.Type.TypeAnnotationState()
}

func (t *OptionalType) RewriteWithRestrictedTypes() (Type, bool) {
	rewrittenType, rewritten := t.Type.RewriteWithRestrictedTypes()
	if rewritten {
		return &OptionalType{
			Type: rewrittenType,
		}, true
	} else {
		return t, false
	}
}

func (t *OptionalType) Unify(
	other Type,
	typeParameters *TypeParameterTypeOrderedMap,
	report func(err error),
	outerRange ast.Range,
) bool {

	otherOptional, ok := other.(*OptionalType)
	if !ok {
		return false
	}

	return t.Type.Unify(otherOptional.Type, typeParameters, report, outerRange)
}

func (t *OptionalType) Resolve(typeArguments *TypeParameterTypeOrderedMap) Type {

	newInnerType := t.Type.Resolve(typeArguments)
	if newInnerType == nil {
		return nil
	}

	return &OptionalType{
		Type: newInnerType,
	}
}

const optionalTypeMapFunctionDocString = `
Returns an optional of the result of calling the given function
with the value of this optional when it is not nil.

Returns nil if this optional is nil
`

func (t *OptionalType) GetMembers() map[string]MemberResolver {

	members := map[string]MemberResolver{
		"map": {
			Kind: common.DeclarationKindFunction,
			Resolve: func(memoryGauge common.MemoryGauge, identifier string, targetRange ast.Range, report func(error)) *Member {

				// It invalid for an optional of a resource to have a `map` function

				if t.Type.IsResourceType() {
					report(
						&InvalidResourceOptionalMemberError{
							Name:            identifier,
							DeclarationKind: common.DeclarationKindFunction,
							Range:           targetRange,
						},
					)
				}

				return NewPublicFunctionMember(
					memoryGauge,
					t,
					identifier,
					OptionalTypeMapFunctionType(t.Type),
					optionalTypeMapFunctionDocString,
				)
			},
		},
	}

	return withBuiltinMembers(t, members)
}

func OptionalTypeMapFunctionType(typ Type) *FunctionType {
	typeParameter := &TypeParameter{
		Name: "T",
	}

	resultType := &GenericType{
		TypeParameter: typeParameter,
	}

	return &FunctionType{
		TypeParameters: []*TypeParameter{
			typeParameter,
		},
		Parameters: []*Parameter{
			{
				Label:      ArgumentLabelNotRequired,
				Identifier: "transform",
				TypeAnnotation: NewTypeAnnotation(
					&FunctionType{
						Parameters: []*Parameter{
							{
								Label:          ArgumentLabelNotRequired,
								Identifier:     "value",
								TypeAnnotation: NewTypeAnnotation(typ),
							},
						},
						ReturnTypeAnnotation: NewTypeAnnotation(
							resultType,
						),
					},
				),
			},
		},
		ReturnTypeAnnotation: NewTypeAnnotation(
			&OptionalType{
				Type: resultType,
			},
		),
	}
}

// GenericType
//
type GenericType struct {
	TypeParameter *TypeParameter
}

func (*GenericType) IsType() {}

func (t *GenericType) Tag() TypeTag {
	return GenericTypeTag
}

func (t *GenericType) String() string {
	return t.TypeParameter.Name
}

func (t *GenericType) QualifiedString() string {
	return t.TypeParameter.Name
}

func (t *GenericType) ID() TypeID {
	return TypeID(t.TypeParameter.Name)
}

func (t *GenericType) Equal(other Type) bool {
	otherType, ok := other.(*GenericType)
	if !ok {
		return false
	}
	return t.TypeParameter == otherType.TypeParameter
}

func (*GenericType) IsResourceType() bool {
	return false
}

func (*GenericType) IsInvalidType() bool {
	return false
}

func (*GenericType) IsStorable(_ map[*Member]bool) bool {
	return false
}

func (*GenericType) IsExternallyReturnable(_ map[*Member]bool) bool {
	return false
}

func (t *GenericType) IsImportable(_ map[*Member]bool) bool {
	return false
}

func (*GenericType) IsEquatable() bool {
	return false
}

func (*GenericType) TypeAnnotationState() TypeAnnotationState {
	return TypeAnnotationStateValid
}

func (t *GenericType) RewriteWithRestrictedTypes() (result Type, rewritten bool) {
	return t, false
}

func (t *GenericType) Unify(
	other Type,
	typeParameters *TypeParameterTypeOrderedMap,
	report func(err error),
	outerRange ast.Range,
) bool {

	if unifiedType, ok := typeParameters.Get(t.TypeParameter); ok {

		// If the type parameter is already unified with a type argument
		// (either explicit by a type argument, or implicit through an argument's type),
		// check that this argument's type matches the unified type

		if !other.Equal(unifiedType) {
			report(
				&TypeParameterTypeMismatchError{
					TypeParameter: t.TypeParameter,
					ExpectedType:  unifiedType,
					ActualType:    other,
					Range:         outerRange,
				},
			)
		}

	} else {
		// If the type parameter is not yet unified to a type argument, unify it.

		typeParameters.Set(t.TypeParameter, other)

		// If the type parameter corresponding to the type argument has a type bound,
		// then check that the argument's type is a subtype of the type bound.

		err := t.TypeParameter.checkTypeBound(other, outerRange)
		if err != nil {
			report(err)
		}
	}

	return true
}

func (t *GenericType) Resolve(typeArguments *TypeParameterTypeOrderedMap) Type {
	ty, ok := typeArguments.Get(t.TypeParameter)
	if !ok {
		return nil
	}
	return ty
}

func (t *GenericType) GetMembers() map[string]MemberResolver {
	return withBuiltinMembers(t, nil)
}

// IntegerRangedType

type IntegerRangedType interface {
	Type
	MinInt() *big.Int
	MaxInt() *big.Int
	IsSuperType() bool
}

type FractionalRangedType interface {
	IntegerRangedType
	Scale() uint
	MinFractional() *big.Int
	MaxFractional() *big.Int
}

// SaturatingArithmeticType is a type that supports saturating arithmetic functions
//
type SaturatingArithmeticType interface {
	Type
	SupportsSaturatingAdd() bool
	SupportsSaturatingSubtract() bool
	SupportsSaturatingMultiply() bool
	SupportsSaturatingDivide() bool
}

const NumericTypeSaturatingAddFunctionName = "saturatingAdd"
const numericTypeSaturatingAddFunctionDocString = `
self + other, saturating at the numeric bounds instead of overflowing.
`

const NumericTypeSaturatingSubtractFunctionName = "saturatingSubtract"
const numericTypeSaturatingSubtractFunctionDocString = `
self - other, saturating at the numeric bounds instead of overflowing.
`
const NumericTypeSaturatingMultiplyFunctionName = "saturatingMultiply"
const numericTypeSaturatingMultiplyFunctionDocString = `
self * other, saturating at the numeric bounds instead of overflowing.
`

const NumericTypeSaturatingDivideFunctionName = "saturatingDivide"
const numericTypeSaturatingDivideFunctionDocString = `
self / other, saturating at the numeric bounds instead of overflowing.
`

func addSaturatingArithmeticFunctions(t SaturatingArithmeticType, members map[string]MemberResolver) {

	arithmeticFunctionType := &FunctionType{
		Parameters: []*Parameter{
			{
				Label:          ArgumentLabelNotRequired,
				Identifier:     "other",
				TypeAnnotation: NewTypeAnnotation(t),
			},
		},
		ReturnTypeAnnotation: NewTypeAnnotation(t),
	}

	addArithmeticFunction := func(name string, docString string) {
		members[name] = MemberResolver{
			Kind: common.DeclarationKindFunction,
			Resolve: func(memoryGauge common.MemoryGauge, identifier string, targetRange ast.Range, report func(error)) *Member {
				return NewPublicFunctionMember(
					memoryGauge, t, name, arithmeticFunctionType, docString)
			},
		}
	}

	if t.SupportsSaturatingAdd() {
		addArithmeticFunction(
			NumericTypeSaturatingAddFunctionName,
			numericTypeSaturatingAddFunctionDocString,
		)
	}

	if t.SupportsSaturatingSubtract() {
		addArithmeticFunction(
			NumericTypeSaturatingSubtractFunctionName,
			numericTypeSaturatingSubtractFunctionDocString,
		)
	}

	if t.SupportsSaturatingMultiply() {
		addArithmeticFunction(
			NumericTypeSaturatingMultiplyFunctionName,
			numericTypeSaturatingMultiplyFunctionDocString,
		)
	}

	if t.SupportsSaturatingDivide() {
		addArithmeticFunction(
			NumericTypeSaturatingDivideFunctionName,
			numericTypeSaturatingDivideFunctionDocString,
		)
	}
}

// NumericType represent all the types in the integer range
// and non-fractional ranged types.
//
type NumericType struct {
	name                       string
	tag                        TypeTag
	minInt                     *big.Int
	maxInt                     *big.Int
	supportsSaturatingAdd      bool
	supportsSaturatingSubtract bool
	supportsSaturatingMultiply bool
	supportsSaturatingDivide   bool
	memberResolvers            map[string]MemberResolver
	memberResolversOnce        sync.Once
	isSuperType                bool
}

var _ IntegerRangedType = &NumericType{}

func NewNumericType(typeName string) *NumericType {
	return &NumericType{name: typeName}
}

func (t *NumericType) Tag() TypeTag {
	return t.tag
}

func (t *NumericType) WithTag(tag TypeTag) *NumericType {
	t.tag = tag
	return t
}

func (t *NumericType) WithIntRange(min *big.Int, max *big.Int) *NumericType {
	t.minInt = min
	t.maxInt = max
	return t
}

func (t *NumericType) WithSaturatingAdd() *NumericType {
	t.supportsSaturatingAdd = true
	return t
}

func (t *NumericType) WithSaturatingSubtract() *NumericType {
	t.supportsSaturatingSubtract = true
	return t
}

func (t *NumericType) WithSaturatingMultiply() *NumericType {
	t.supportsSaturatingMultiply = true
	return t
}

func (t *NumericType) WithSaturatingDivide() *NumericType {
	t.supportsSaturatingDivide = true
	return t
}

func (t *NumericType) SupportsSaturatingAdd() bool {
	return t.supportsSaturatingAdd
}

func (t *NumericType) SupportsSaturatingSubtract() bool {
	return t.supportsSaturatingSubtract
}

func (t *NumericType) SupportsSaturatingMultiply() bool {
	return t.supportsSaturatingMultiply
}

func (t *NumericType) SupportsSaturatingDivide() bool {
	return t.supportsSaturatingDivide
}

func (*NumericType) IsType() {}

func (t *NumericType) String() string {
	return t.name
}

func (t *NumericType) QualifiedString() string {
	return t.name
}

func (t *NumericType) ID() TypeID {
	return TypeID(t.name)
}

func (t *NumericType) Equal(other Type) bool {
	// Numeric types are singletons. Hence their pointers should be equal.
	if t == other {
		return true
	}

	// Check for the value equality as well, as a backup strategy.
	otherNumericType, ok := other.(*NumericType)
	return ok && t.ID() == otherNumericType.ID()
}

func (*NumericType) IsResourceType() bool {
	return false
}

func (*NumericType) IsInvalidType() bool {
	return false
}

func (*NumericType) IsStorable(_ map[*Member]bool) bool {
	return true
}

func (*NumericType) IsExternallyReturnable(_ map[*Member]bool) bool {
	return true
}

func (t *NumericType) IsImportable(_ map[*Member]bool) bool {
	return true
}

func (*NumericType) IsEquatable() bool {
	return true
}

func (*NumericType) TypeAnnotationState() TypeAnnotationState {
	return TypeAnnotationStateValid
}

func (t *NumericType) RewriteWithRestrictedTypes() (result Type, rewritten bool) {
	return t, false
}

func (t *NumericType) MinInt() *big.Int {
	return t.minInt
}

func (t *NumericType) MaxInt() *big.Int {
	return t.maxInt
}

func (*NumericType) Unify(_ Type, _ *TypeParameterTypeOrderedMap, _ func(err error), _ ast.Range) bool {
	return false
}

func (t *NumericType) Resolve(_ *TypeParameterTypeOrderedMap) Type {
	return t
}

func (t *NumericType) GetMembers() map[string]MemberResolver {
	t.initializeMemberResolvers()
	return t.memberResolvers
}

func (t *NumericType) initializeMemberResolvers() {
	t.memberResolversOnce.Do(func() {
		members := map[string]MemberResolver{}

		addSaturatingArithmeticFunctions(t, members)

		t.memberResolvers = withBuiltinMembers(t, members)
	})
}

func (t *NumericType) AsSuperType() *NumericType {
	t.isSuperType = true
	return t
}

func (t *NumericType) IsSuperType() bool {
	return t.isSuperType
}

// FixedPointNumericType represents all the types in the fixed-point range.
//
type FixedPointNumericType struct {
	name                       string
	tag                        TypeTag
	scale                      uint
	minInt                     *big.Int
	maxInt                     *big.Int
	minFractional              *big.Int
	maxFractional              *big.Int
	supportsSaturatingAdd      bool
	supportsSaturatingSubtract bool
	supportsSaturatingMultiply bool
	supportsSaturatingDivide   bool
	memberResolvers            map[string]MemberResolver
	memberResolversOnce        sync.Once
	isSuperType                bool
}

var _ FractionalRangedType = &FixedPointNumericType{}

func NewFixedPointNumericType(typeName string) *FixedPointNumericType {
	return &FixedPointNumericType{
		name: typeName,
	}
}

func (t *FixedPointNumericType) Tag() TypeTag {
	return t.tag
}

func (t *FixedPointNumericType) WithTag(tag TypeTag) *FixedPointNumericType {
	t.tag = tag
	return t
}

func (t *FixedPointNumericType) WithIntRange(minInt *big.Int, maxInt *big.Int) *FixedPointNumericType {
	t.minInt = minInt
	t.maxInt = maxInt
	return t
}

func (t *FixedPointNumericType) WithFractionalRange(
	minFractional *big.Int,
	maxFractional *big.Int,
) *FixedPointNumericType {

	t.minFractional = minFractional
	t.maxFractional = maxFractional
	return t
}

func (t *FixedPointNumericType) WithScale(scale uint) *FixedPointNumericType {
	t.scale = scale
	return t
}

func (t *FixedPointNumericType) WithSaturatingAdd() *FixedPointNumericType {
	t.supportsSaturatingAdd = true
	return t
}

func (t *FixedPointNumericType) WithSaturatingSubtract() *FixedPointNumericType {
	t.supportsSaturatingSubtract = true
	return t
}

func (t *FixedPointNumericType) WithSaturatingMultiply() *FixedPointNumericType {
	t.supportsSaturatingMultiply = true
	return t
}

func (t *FixedPointNumericType) WithSaturatingDivide() *FixedPointNumericType {
	t.supportsSaturatingDivide = true
	return t
}

func (t *FixedPointNumericType) SupportsSaturatingAdd() bool {
	return t.supportsSaturatingAdd
}

func (t *FixedPointNumericType) SupportsSaturatingSubtract() bool {
	return t.supportsSaturatingSubtract
}

func (t *FixedPointNumericType) SupportsSaturatingMultiply() bool {
	return t.supportsSaturatingMultiply
}

func (t *FixedPointNumericType) SupportsSaturatingDivide() bool {
	return t.supportsSaturatingDivide
}

func (*FixedPointNumericType) IsType() {}

func (t *FixedPointNumericType) String() string {
	return t.name
}

func (t *FixedPointNumericType) QualifiedString() string {
	return t.name
}

func (t *FixedPointNumericType) ID() TypeID {
	return TypeID(t.name)
}

func (t *FixedPointNumericType) Equal(other Type) bool {
	// Numeric types are singletons. Hence their pointers should be equal.
	if t == other {
		return true
	}

	// Check for the value equality as well, as a backup strategy.
	otherNumericType, ok := other.(*FixedPointNumericType)
	return ok && t.ID() == otherNumericType.ID()
}

func (*FixedPointNumericType) IsResourceType() bool {
	return false
}

func (*FixedPointNumericType) IsInvalidType() bool {
	return false
}

func (*FixedPointNumericType) IsStorable(_ map[*Member]bool) bool {
	return true
}

func (*FixedPointNumericType) IsExternallyReturnable(_ map[*Member]bool) bool {
	return true
}

func (t *FixedPointNumericType) IsImportable(_ map[*Member]bool) bool {
	return true
}

func (*FixedPointNumericType) IsEquatable() bool {
	return true
}

func (*FixedPointNumericType) TypeAnnotationState() TypeAnnotationState {
	return TypeAnnotationStateValid
}

func (t *FixedPointNumericType) RewriteWithRestrictedTypes() (result Type, rewritten bool) {
	return t, false
}

func (t *FixedPointNumericType) MinInt() *big.Int {
	return t.minInt
}

func (t *FixedPointNumericType) MaxInt() *big.Int {
	return t.maxInt
}

func (t *FixedPointNumericType) MinFractional() *big.Int {
	return t.minFractional
}

func (t *FixedPointNumericType) MaxFractional() *big.Int {
	return t.maxFractional
}

func (t *FixedPointNumericType) Scale() uint {
	return t.scale
}

func (*FixedPointNumericType) Unify(_ Type, _ *TypeParameterTypeOrderedMap, _ func(err error), _ ast.Range) bool {
	return false
}

func (t *FixedPointNumericType) Resolve(_ *TypeParameterTypeOrderedMap) Type {
	return t
}

func (t *FixedPointNumericType) GetMembers() map[string]MemberResolver {
	t.initializeMemberResolvers()
	return t.memberResolvers
}

func (t *FixedPointNumericType) initializeMemberResolvers() {
	t.memberResolversOnce.Do(func() {
		members := map[string]MemberResolver{}

		addSaturatingArithmeticFunctions(t, members)

		t.memberResolvers = withBuiltinMembers(t, members)
	})
}

func (t *FixedPointNumericType) AsSuperType() *FixedPointNumericType {
	t.isSuperType = true
	return t
}

func (t *FixedPointNumericType) IsSuperType() bool {
	return t.isSuperType
}

// Numeric types

var (

	// NumberType represents the super-type of all number types
	NumberType = NewNumericType(NumberTypeName).
			WithTag(NumberTypeTag).
			AsSuperType()

	// SignedNumberType represents the super-type of all signed number types
	SignedNumberType = NewNumericType(SignedNumberTypeName).
				WithTag(SignedNumberTypeTag).
				AsSuperType()

	// IntegerType represents the super-type of all integer types
	IntegerType = NewNumericType(IntegerTypeName).
			WithTag(IntegerTypeTag).
			AsSuperType()

	// SignedIntegerType represents the super-type of all signed integer types
	SignedIntegerType = NewNumericType(SignedIntegerTypeName).
				WithTag(SignedIntegerTypeTag).
				AsSuperType()

	// IntType represents the arbitrary-precision integer type `Int`
	IntType = NewNumericType(IntTypeName).
		WithTag(IntTypeTag)

	// Int8Type represents the 8-bit signed integer type `Int8`
	Int8Type = NewNumericType(Int8TypeName).
			WithTag(Int8TypeTag).
			WithIntRange(Int8TypeMinInt, Int8TypeMaxInt).
			WithSaturatingAdd().
			WithSaturatingSubtract().
			WithSaturatingMultiply().
			WithSaturatingDivide()

	// Int16Type represents the 16-bit signed integer type `Int16`
	Int16Type = NewNumericType(Int16TypeName).
			WithTag(Int16TypeTag).
			WithIntRange(Int16TypeMinInt, Int16TypeMaxInt).
			WithSaturatingAdd().
			WithSaturatingSubtract().
			WithSaturatingMultiply().
			WithSaturatingDivide()

	// Int32Type represents the 32-bit signed integer type `Int32`
	Int32Type = NewNumericType(Int32TypeName).
			WithTag(Int32TypeTag).
			WithIntRange(Int32TypeMinInt, Int32TypeMaxInt).
			WithSaturatingAdd().
			WithSaturatingSubtract().
			WithSaturatingMultiply().
			WithSaturatingDivide()

	// Int64Type represents the 64-bit signed integer type `Int64`
	Int64Type = NewNumericType(Int64TypeName).
			WithTag(Int64TypeTag).
			WithIntRange(Int64TypeMinInt, Int64TypeMaxInt).
			WithSaturatingAdd().
			WithSaturatingSubtract().
			WithSaturatingMultiply().
			WithSaturatingDivide()

	// Int128Type represents the 128-bit signed integer type `Int128`
	Int128Type = NewNumericType(Int128TypeName).
			WithTag(Int128TypeTag).
			WithIntRange(Int128TypeMinIntBig, Int128TypeMaxIntBig).
			WithSaturatingAdd().
			WithSaturatingSubtract().
			WithSaturatingMultiply().
			WithSaturatingDivide()

	// Int256Type represents the 256-bit signed integer type `Int256`
	Int256Type = NewNumericType(Int256TypeName).
			WithTag(Int256TypeTag).
			WithIntRange(Int256TypeMinIntBig, Int256TypeMaxIntBig).
			WithSaturatingAdd().
			WithSaturatingSubtract().
			WithSaturatingMultiply().
			WithSaturatingDivide()

	// UIntType represents the arbitrary-precision unsigned integer type `UInt`
	UIntType = NewNumericType(UIntTypeName).
			WithTag(UIntTypeTag).
			WithIntRange(UIntTypeMin, nil).
			WithSaturatingSubtract()

	// UInt8Type represents the 8-bit unsigned integer type `UInt8`
	// which checks for overflow and underflow
	UInt8Type = NewNumericType(UInt8TypeName).
			WithTag(UInt8TypeTag).
			WithIntRange(UInt8TypeMinInt, UInt8TypeMaxInt).
			WithSaturatingAdd().
			WithSaturatingSubtract().
			WithSaturatingMultiply()

	// UInt16Type represents the 16-bit unsigned integer type `UInt16`
	// which checks for overflow and underflow
	UInt16Type = NewNumericType(UInt16TypeName).
			WithTag(UInt16TypeTag).
			WithIntRange(UInt16TypeMinInt, UInt16TypeMaxInt).
			WithSaturatingAdd().
			WithSaturatingSubtract().
			WithSaturatingMultiply()

	// UInt32Type represents the 32-bit unsigned integer type `UInt32`
	// which checks for overflow and underflow
	UInt32Type = NewNumericType(UInt32TypeName).
			WithTag(UInt32TypeTag).
			WithIntRange(UInt32TypeMinInt, UInt32TypeMaxInt).
			WithSaturatingAdd().
			WithSaturatingSubtract().
			WithSaturatingMultiply()

	// UInt64Type represents the 64-bit unsigned integer type `UInt64`
	// which checks for overflow and underflow
	UInt64Type = NewNumericType(UInt64TypeName).
			WithTag(UInt64TypeTag).
			WithIntRange(UInt64TypeMinInt, UInt64TypeMaxInt).
			WithSaturatingAdd().
			WithSaturatingSubtract().
			WithSaturatingMultiply()

	// UInt128Type represents the 128-bit unsigned integer type `UInt128`
	// which checks for overflow and underflow
	UInt128Type = NewNumericType(UInt128TypeName).
			WithTag(UInt128TypeTag).
			WithIntRange(UInt128TypeMinIntBig, UInt128TypeMaxIntBig).
			WithSaturatingAdd().
			WithSaturatingSubtract().
			WithSaturatingMultiply()

	// UInt256Type represents the 256-bit unsigned integer type `UInt256`
	// which checks for overflow and underflow
	UInt256Type = NewNumericType(UInt256TypeName).
			WithTag(UInt256TypeTag).
			WithIntRange(UInt256TypeMinIntBig, UInt256TypeMaxIntBig).
			WithSaturatingAdd().
			WithSaturatingSubtract().
			WithSaturatingMultiply()

	// Word8Type represents the 8-bit unsigned integer type `Word8`
	// which does NOT check for overflow and underflow
	Word8Type = NewNumericType(Word8TypeName).
			WithTag(Word8TypeTag).
			WithIntRange(Word8TypeMinInt, Word8TypeMaxInt)

	// Word16Type represents the 16-bit unsigned integer type `Word16`
	// which does NOT check for overflow and underflow
	Word16Type = NewNumericType(Word16TypeName).
			WithTag(Word16TypeTag).
			WithIntRange(Word16TypeMinInt, Word16TypeMaxInt)

	// Word32Type represents the 32-bit unsigned integer type `Word32`
	// which does NOT check for overflow and underflow
	Word32Type = NewNumericType(Word32TypeName).
			WithTag(Word32TypeTag).
			WithIntRange(Word32TypeMinInt, Word32TypeMaxInt)

	// Word64Type represents the 64-bit unsigned integer type `Word64`
	// which does NOT check for overflow and underflow
	Word64Type = NewNumericType(Word64TypeName).
			WithTag(Word64TypeTag).
			WithIntRange(Word64TypeMinInt, Word64TypeMaxInt)

	// FixedPointType represents the super-type of all fixed-point types
	FixedPointType = NewNumericType(FixedPointTypeName).
			WithTag(FixedPointTypeTag).
			AsSuperType()

	// SignedFixedPointType represents the super-type of all signed fixed-point types
	SignedFixedPointType = NewNumericType(SignedFixedPointTypeName).
				WithTag(SignedFixedPointTypeTag).
				AsSuperType()

	// Fix64Type represents the 64-bit signed decimal fixed-point type `Fix64`
	// which has a scale of Fix64Scale, and checks for overflow and underflow
	Fix64Type = NewFixedPointNumericType(Fix64TypeName).
			WithTag(Fix64TypeTag).
			WithIntRange(Fix64TypeMinIntBig, Fix64TypeMaxIntBig).
			WithFractionalRange(Fix64TypeMinFractionalBig, Fix64TypeMaxFractionalBig).
			WithScale(Fix64Scale).
			WithSaturatingAdd().
			WithSaturatingSubtract().
			WithSaturatingMultiply().
			WithSaturatingDivide()

	// UFix64Type represents the 64-bit unsigned decimal fixed-point type `UFix64`
	// which has a scale of 1E9, and checks for overflow and underflow
	UFix64Type = NewFixedPointNumericType(UFix64TypeName).
			WithTag(UFix64TypeTag).
			WithIntRange(UFix64TypeMinIntBig, UFix64TypeMaxIntBig).
			WithFractionalRange(UFix64TypeMinFractionalBig, UFix64TypeMaxFractionalBig).
			WithScale(Fix64Scale).
			WithSaturatingAdd().
			WithSaturatingSubtract().
			WithSaturatingMultiply()
)

// Numeric type ranges
var (
	Int8TypeMinInt = new(big.Int).SetInt64(math.MinInt8)
	Int8TypeMaxInt = new(big.Int).SetInt64(math.MaxInt8)

	Int16TypeMinInt = new(big.Int).SetInt64(math.MinInt16)
	Int16TypeMaxInt = new(big.Int).SetInt64(math.MaxInt16)

	Int32TypeMinInt = new(big.Int).SetInt64(math.MinInt32)
	Int32TypeMaxInt = new(big.Int).SetInt64(math.MaxInt32)

	Int64TypeMinInt = new(big.Int).SetInt64(math.MinInt64)
	Int64TypeMaxInt = new(big.Int).SetInt64(math.MaxInt64)

	Int128TypeMinIntBig = func() *big.Int {
		int128TypeMin := big.NewInt(-1)
		int128TypeMin.Lsh(int128TypeMin, 127)
		return int128TypeMin
	}()

	Int128TypeMaxIntBig = func() *big.Int {
		int128TypeMax := big.NewInt(1)
		int128TypeMax.Lsh(int128TypeMax, 127)
		int128TypeMax.Sub(int128TypeMax, big.NewInt(1))
		return int128TypeMax
	}()

	Int256TypeMinIntBig = func() *big.Int {
		int256TypeMin := big.NewInt(-1)
		int256TypeMin.Lsh(int256TypeMin, 255)
		return int256TypeMin
	}()

	Int256TypeMaxIntBig = func() *big.Int {
		int256TypeMax := big.NewInt(1)
		int256TypeMax.Lsh(int256TypeMax, 255)
		int256TypeMax.Sub(int256TypeMax, big.NewInt(1))
		return int256TypeMax
	}()

	UIntTypeMin = new(big.Int)

	UInt8TypeMinInt = new(big.Int)
	UInt8TypeMaxInt = new(big.Int).SetUint64(math.MaxUint8)

	UInt16TypeMinInt = new(big.Int)
	UInt16TypeMaxInt = new(big.Int).SetUint64(math.MaxUint16)

	UInt32TypeMinInt = new(big.Int)
	UInt32TypeMaxInt = new(big.Int).SetUint64(math.MaxUint32)

	UInt64TypeMinInt = new(big.Int)
	UInt64TypeMaxInt = new(big.Int).SetUint64(math.MaxUint64)

	UInt128TypeMinIntBig = new(big.Int)

	UInt128TypeMaxIntBig = func() *big.Int {
		uInt128TypeMax := big.NewInt(1)
		uInt128TypeMax.Lsh(uInt128TypeMax, 128)
		uInt128TypeMax.Sub(uInt128TypeMax, big.NewInt(1))
		return uInt128TypeMax

	}()

	UInt256TypeMinIntBig = new(big.Int)

	UInt256TypeMaxIntBig = func() *big.Int {
		uInt256TypeMax := big.NewInt(1)
		uInt256TypeMax.Lsh(uInt256TypeMax, 256)
		uInt256TypeMax.Sub(uInt256TypeMax, big.NewInt(1))
		return uInt256TypeMax
	}()

	Word8TypeMinInt = new(big.Int)
	Word8TypeMaxInt = new(big.Int).SetUint64(math.MaxUint8)

	Word16TypeMinInt = new(big.Int)
	Word16TypeMaxInt = new(big.Int).SetUint64(math.MaxUint16)

	Word32TypeMinInt = new(big.Int)
	Word32TypeMaxInt = new(big.Int).SetUint64(math.MaxUint32)

	Word64TypeMinInt = new(big.Int)
	Word64TypeMaxInt = new(big.Int).SetUint64(math.MaxUint64)

	Fix64FactorBig = new(big.Int).SetUint64(uint64(Fix64Factor))

	Fix64TypeMinIntBig = fixedpoint.Fix64TypeMinIntBig
	Fix64TypeMaxIntBig = fixedpoint.Fix64TypeMaxIntBig

	Fix64TypeMinFractionalBig = fixedpoint.Fix64TypeMinFractionalBig
	Fix64TypeMaxFractionalBig = fixedpoint.Fix64TypeMaxFractionalBig

	UFix64TypeMinIntBig = fixedpoint.UFix64TypeMinIntBig
	UFix64TypeMaxIntBig = fixedpoint.UFix64TypeMaxIntBig

	UFix64TypeMinFractionalBig = fixedpoint.UFix64TypeMinFractionalBig
	UFix64TypeMaxFractionalBig = fixedpoint.UFix64TypeMaxFractionalBig
)

const Fix64Scale = fixedpoint.Fix64Scale
const Fix64Factor = fixedpoint.Fix64Factor

const Fix64TypeMinInt = fixedpoint.Fix64TypeMinInt
const Fix64TypeMaxInt = fixedpoint.Fix64TypeMaxInt

const Fix64TypeMinFractional = fixedpoint.Fix64TypeMinFractional
const Fix64TypeMaxFractional = fixedpoint.Fix64TypeMaxFractional

const UFix64TypeMinInt = fixedpoint.UFix64TypeMinInt
const UFix64TypeMaxInt = fixedpoint.UFix64TypeMaxInt

const UFix64TypeMinFractional = fixedpoint.UFix64TypeMinFractional
const UFix64TypeMaxFractional = fixedpoint.UFix64TypeMaxFractional

// ArrayType

type ArrayType interface {
	ValueIndexableType
	isArrayType()
}

const arrayTypeFirstIndexFunctionDocString = `
Returns the index of the first element matching the given object in the array, nil if no match.
Available if the array element type is not resource-kinded and equatable.
`

const arrayTypeContainsFunctionDocString = `
Returns true if the given object is in the array
`

const arrayTypeLengthFieldDocString = `
Returns the number of elements in the array
`

const arrayTypeAppendFunctionDocString = `
Adds the given element to the end of the array
`

const arrayTypeAppendAllFunctionDocString = `
Adds all the elements from the given array to the end of the array
`

const arrayTypeConcatFunctionDocString = `
Returns a new array which contains the given array concatenated to the end of the original array, but does not modify the original array
`

const arrayTypeInsertFunctionDocString = `
Inserts the given element at the given index of the array.

The index must be within the bounds of the array.
If the index is outside the bounds, the program aborts.

The existing element at the supplied index is not overwritten.

All the elements after the new inserted element are shifted to the right by one
`

const arrayTypeRemoveFunctionDocString = `
Removes the element at the given index from the array and returns it.

The index must be within the bounds of the array.
If the index is outside the bounds, the program aborts
`

const arrayTypeRemoveFirstFunctionDocString = `
Removes the first element from the array and returns it.

The array must not be empty. If the array is empty, the program aborts
`

const arrayTypeRemoveLastFunctionDocString = `
Removes the last element from the array and returns it.

The array must not be empty. If the array is empty, the program aborts
`

const arrayTypeSliceFunctionDocString = `
Returns a new variable-sized array containing the slice of the elements in the given array from start index ` + "`from`" + ` up to, but not including, the end index ` + "`upTo`" + `.

This function creates a new array whose length is ` + "`upTo - from`" + `.
It does not modify the original array.
If either of the parameters are out of the bounds of the array, or the indices are invalid (` + "`from > upTo`" + `), then the function will fail.
`

func getArrayMembers(arrayType ArrayType) map[string]MemberResolver {

	members := map[string]MemberResolver{
		"contains": {
			Kind: common.DeclarationKindFunction,
			Resolve: func(memoryGauge common.MemoryGauge, identifier string, targetRange ast.Range, report func(error)) *Member {

				elementType := arrayType.ElementType(false)

				// It is impossible for an array of resources to have a `contains` function:
				// if the resource is passed as an argument, it cannot be inside the array

				if elementType.IsResourceType() {
					report(
						&InvalidResourceArrayMemberError{
							Name:            identifier,
							DeclarationKind: common.DeclarationKindFunction,
							Range:           targetRange,
						},
					)
				}

				// TODO: implement Equatable interface: https://github.com/dapperlabs/bamboo-node/issues/78

				if !elementType.IsEquatable() {
					report(
						&NotEquatableTypeError{
							Type:  elementType,
							Range: targetRange,
						},
					)
				}

				return NewPublicFunctionMember(
					memoryGauge,
					arrayType,
					identifier,
					ArrayContainsFunctionType(elementType),
					arrayTypeContainsFunctionDocString,
				)
			},
		},
		"length": {
			Kind: common.DeclarationKindField,
			Resolve: func(memoryGauge common.MemoryGauge, identifier string, _ ast.Range, _ func(error)) *Member {
				return NewPublicConstantFieldMember(
					memoryGauge,
					arrayType,
					identifier,
					IntType,
					arrayTypeLengthFieldDocString,
				)
			},
		},
		"firstIndex": {
			Kind: common.DeclarationKindFunction,
			Resolve: func(memoryGauge common.MemoryGauge, identifier string, targetRange ast.Range, report func(error)) *Member {

				elementType := arrayType.ElementType(false)

				// It is impossible for an array of resources to have a `firstIndex` function:
				// if the resource is passed as an argument, it cannot be inside the array

				if elementType.IsResourceType() {
					report(
						&InvalidResourceArrayMemberError{
							Name:            identifier,
							DeclarationKind: common.DeclarationKindFunction,
							Range:           targetRange,
						},
					)
				}

				// TODO: implement Equatable interface

				if !elementType.IsEquatable() {
					report(
						&NotEquatableTypeError{
							Type:  elementType,
							Range: targetRange,
						},
					)
				}

				return NewPublicFunctionMember(
					memoryGauge,
					arrayType,
					identifier,
					ArrayFirstIndexFunctionType(elementType),
					arrayTypeFirstIndexFunctionDocString,
				)
			},
		},
	}

	// TODO: maybe still return members but report a helpful error?

	if _, ok := arrayType.(*VariableSizedType); ok {

		members["append"] = MemberResolver{
			Kind:     common.DeclarationKindFunction,
			Mutating: true,
			Resolve: func(memoryGauge common.MemoryGauge, identifier string, targetRange ast.Range, report func(error)) *Member {
				elementType := arrayType.ElementType(false)
				return NewPublicFunctionMember(
					memoryGauge,
					arrayType,
					identifier,
					ArrayAppendFunctionType(elementType),
					arrayTypeAppendFunctionDocString,
				)
			},
		}

		members["appendAll"] = MemberResolver{
			Kind:     common.DeclarationKindFunction,
			Mutating: true,
			Resolve: func(memoryGauge common.MemoryGauge, identifier string, targetRange ast.Range, report func(error)) *Member {

				elementType := arrayType.ElementType(false)

				if elementType.IsResourceType() {
					report(
						&InvalidResourceArrayMemberError{
							Name:            identifier,
							DeclarationKind: common.DeclarationKindFunction,
							Range:           targetRange,
						},
					)
				}

				return NewPublicFunctionMember(
					memoryGauge,
					arrayType,
					identifier,
					ArrayAppendAllFunctionType(arrayType),
					arrayTypeAppendAllFunctionDocString,
				)
			},
		}

		members["concat"] = MemberResolver{
			Kind: common.DeclarationKindFunction,
			Resolve: func(memoryGauge common.MemoryGauge, identifier string, targetRange ast.Range, report func(error)) *Member {

				// TODO: maybe allow for resource element type

				elementType := arrayType.ElementType(false)

				if elementType.IsResourceType() {
					report(
						&InvalidResourceArrayMemberError{
							Name:            identifier,
							DeclarationKind: common.DeclarationKindFunction,
							Range:           targetRange,
						},
					)
				}

				return NewPublicFunctionMember(
					memoryGauge,
					arrayType,
					identifier,
					ArrayConcatFunctionType(arrayType),
					arrayTypeConcatFunctionDocString,
				)
			},
		}

		members["slice"] = MemberResolver{
			Kind: common.DeclarationKindFunction,
			Resolve: func(memoryGauge common.MemoryGauge, identifier string, targetRange ast.Range, report func(error)) *Member {

				elementType := arrayType.ElementType(false)

				if elementType.IsResourceType() {
					report(
						&InvalidResourceArrayMemberError{
							Name:            identifier,
							DeclarationKind: common.DeclarationKindFunction,
							Range:           targetRange,
						},
					)
				}

				return NewPublicFunctionMember(
					memoryGauge,
					arrayType,
					identifier,
					ArraySliceFunctionType(elementType),
					arrayTypeSliceFunctionDocString,
				)
			},
		}

		members["insert"] = MemberResolver{
			Kind:     common.DeclarationKindFunction,
			Mutating: true,
			Resolve: func(memoryGauge common.MemoryGauge, identifier string, _ ast.Range, _ func(error)) *Member {

				elementType := arrayType.ElementType(false)

				return NewPublicFunctionMember(
					memoryGauge,
					arrayType,
					identifier,
					ArrayInsertFunctionType(elementType),
					arrayTypeInsertFunctionDocString,
				)
			},
		}

		members["remove"] = MemberResolver{
			Kind:     common.DeclarationKindFunction,
			Mutating: true,
			Resolve: func(memoryGauge common.MemoryGauge, identifier string, _ ast.Range, _ func(error)) *Member {

				elementType := arrayType.ElementType(false)

				return NewPublicFunctionMember(
					memoryGauge,
					arrayType,
					identifier,
					ArrayRemoveFunctionType(elementType),
					arrayTypeRemoveFunctionDocString,
				)
			},
		}

		members["removeFirst"] = MemberResolver{
			Kind:     common.DeclarationKindFunction,
			Mutating: true,
			Resolve: func(memoryGauge common.MemoryGauge, identifier string, _ ast.Range, _ func(error)) *Member {

				elementType := arrayType.ElementType(false)

				return NewPublicFunctionMember(
					memoryGauge,
					arrayType,
					identifier,
					ArrayRemoveFirstFunctionType(elementType),

					arrayTypeRemoveFirstFunctionDocString,
				)
			},
		}

		members["removeLast"] = MemberResolver{
			Kind:     common.DeclarationKindFunction,
			Mutating: true,
			Resolve: func(memoryGauge common.MemoryGauge, identifier string, _ ast.Range, _ func(error)) *Member {

				elementType := arrayType.ElementType(false)

				return NewPublicFunctionMember(
					memoryGauge,
					arrayType,
					identifier,
					ArrayRemoveLastFunctionType(elementType),
					arrayTypeRemoveLastFunctionDocString,
				)
			},
		}
	}

	return withBuiltinMembers(arrayType, members)
}

func ArrayRemoveLastFunctionType(elementType Type) *FunctionType {
	return &FunctionType{
		ReturnTypeAnnotation: NewTypeAnnotation(
			elementType,
		),
	}
}

func ArrayRemoveFirstFunctionType(elementType Type) *FunctionType {
	return &FunctionType{
		ReturnTypeAnnotation: NewTypeAnnotation(
			elementType,
		),
	}
}

func ArrayRemoveFunctionType(elementType Type) *FunctionType {
	return &FunctionType{
		Parameters: []*Parameter{
			{
				Identifier:     "at",
				TypeAnnotation: NewTypeAnnotation(IntegerType),
			},
		},
		ReturnTypeAnnotation: NewTypeAnnotation(
			elementType,
		),
	}
}

func ArrayInsertFunctionType(elementType Type) *FunctionType {
	return &FunctionType{
		Parameters: []*Parameter{
			{
				Identifier:     "at",
				TypeAnnotation: NewTypeAnnotation(IntegerType),
			},
			{
				Label:          ArgumentLabelNotRequired,
				Identifier:     "element",
				TypeAnnotation: NewTypeAnnotation(elementType),
			},
		},
		ReturnTypeAnnotation: NewTypeAnnotation(
			VoidType,
		),
	}
}

func ArrayConcatFunctionType(arrayType Type) *FunctionType {
	typeAnnotation := NewTypeAnnotation(arrayType)
	return &FunctionType{
		Parameters: []*Parameter{
			{
				Label:          ArgumentLabelNotRequired,
				Identifier:     "other",
				TypeAnnotation: typeAnnotation,
			},
		},
		ReturnTypeAnnotation: typeAnnotation,
	}
}

func ArrayFirstIndexFunctionType(elementType Type) *FunctionType {
	return &FunctionType{
		Parameters: []*Parameter{
			{
				Identifier:     "of",
				TypeAnnotation: NewTypeAnnotation(elementType),
			},
		},
		ReturnTypeAnnotation: NewTypeAnnotation(
			&OptionalType{Type: IntType},
		),
	}
}
func ArrayContainsFunctionType(elementType Type) *FunctionType {
	return &FunctionType{
		Parameters: []*Parameter{
			{
				Label:          ArgumentLabelNotRequired,
				Identifier:     "element",
				TypeAnnotation: NewTypeAnnotation(elementType),
			},
		},
		ReturnTypeAnnotation: NewTypeAnnotation(
			BoolType,
		),
	}
}

func ArrayAppendAllFunctionType(arrayType Type) *FunctionType {
	return &FunctionType{
		Parameters: []*Parameter{
			{
				Label:          ArgumentLabelNotRequired,
				Identifier:     "other",
				TypeAnnotation: NewTypeAnnotation(arrayType),
			},
		},
		ReturnTypeAnnotation: NewTypeAnnotation(VoidType),
	}
}

func ArrayAppendFunctionType(elementType Type) *FunctionType {
	return &FunctionType{
		Parameters: []*Parameter{
			{
				Label:          ArgumentLabelNotRequired,
				Identifier:     "element",
				TypeAnnotation: NewTypeAnnotation(elementType),
			},
		},
		ReturnTypeAnnotation: NewTypeAnnotation(
			VoidType,
		),
	}
}

func ArraySliceFunctionType(elementType Type) *FunctionType {
	return &FunctionType{
		Parameters: []*Parameter{
			{
				Identifier:     "from",
				TypeAnnotation: NewTypeAnnotation(IntType),
			},
			{
				Identifier:     "upTo",
				TypeAnnotation: NewTypeAnnotation(IntType),
			},
		},
		ReturnTypeAnnotation: NewTypeAnnotation(&VariableSizedType{
			Type: elementType,
		}),
	}
}

// VariableSizedType is a variable sized array type
type VariableSizedType struct {
	Type                Type
	memberResolvers     map[string]MemberResolver
	memberResolversOnce sync.Once
}

func NewVariableSizedType(memoryGauge common.MemoryGauge, typ Type) *VariableSizedType {
	common.UseMemory(memoryGauge, common.VariableSizedSemaTypeMemoryUsage)
	return &VariableSizedType{
		Type: typ,
	}
}

func (*VariableSizedType) IsType() {}

func (*VariableSizedType) isArrayType() {}

func (t *VariableSizedType) Tag() TypeTag {
	return VariableSizedTypeTag
}

func (t *VariableSizedType) String() string {
	return fmt.Sprintf("[%s]", t.Type)
}

func (t *VariableSizedType) QualifiedString() string {
	return fmt.Sprintf("[%s]", t.Type.QualifiedString())
}

func (t *VariableSizedType) ID() TypeID {
	return TypeID(fmt.Sprintf("[%s]", t.Type.ID()))
}

func (t *VariableSizedType) Equal(other Type) bool {
	otherArray, ok := other.(*VariableSizedType)
	if !ok {
		return false
	}

	return t.Type.Equal(otherArray.Type)
}

func (t *VariableSizedType) GetMembers() map[string]MemberResolver {
	t.initializeMemberResolvers()
	return t.memberResolvers
}

func (t *VariableSizedType) initializeMemberResolvers() {
	t.memberResolversOnce.Do(func() {
		t.memberResolvers = getArrayMembers(t)
	})
}

func (t *VariableSizedType) IsResourceType() bool {
	return t.Type.IsResourceType()
}

func (t *VariableSizedType) IsInvalidType() bool {
	return t.Type.IsInvalidType()
}

func (t *VariableSizedType) IsStorable(results map[*Member]bool) bool {
	return t.Type.IsStorable(results)
}

func (t *VariableSizedType) IsExternallyReturnable(results map[*Member]bool) bool {
	return t.Type.IsExternallyReturnable(results)
}

func (t *VariableSizedType) IsImportable(results map[*Member]bool) bool {
	return t.Type.IsImportable(results)
}

func (*VariableSizedType) IsEquatable() bool {
	// TODO:
	return false
}

func (t *VariableSizedType) TypeAnnotationState() TypeAnnotationState {
	return t.Type.TypeAnnotationState()
}

func (t *VariableSizedType) RewriteWithRestrictedTypes() (Type, bool) {
	rewrittenType, rewritten := t.Type.RewriteWithRestrictedTypes()
	if rewritten {
		return &VariableSizedType{
			Type: rewrittenType,
		}, true
	} else {
		return t, false
	}
}

func (*VariableSizedType) isValueIndexableType() bool {
	return true
}

func (*VariableSizedType) AllowsValueIndexingAssignment() bool {
	return true
}

func (t *VariableSizedType) ElementType(_ bool) Type {
	return t.Type
}

func (t *VariableSizedType) IndexingType() Type {
	return IntegerType
}

func (t *VariableSizedType) Unify(
	other Type,
	typeParameters *TypeParameterTypeOrderedMap,
	report func(err error),
	outerRange ast.Range,
) bool {

	otherArray, ok := other.(*VariableSizedType)
	if !ok {
		return false
	}

	return t.Type.Unify(otherArray.Type, typeParameters, report, outerRange)
}

func (t *VariableSizedType) Resolve(typeArguments *TypeParameterTypeOrderedMap) Type {
	newInnerType := t.Type.Resolve(typeArguments)
	if newInnerType == nil {
		return nil
	}

	return &VariableSizedType{
		Type: newInnerType,
	}
}

// ConstantSizedType is a constant sized array type
type ConstantSizedType struct {
	Type                Type
	Size                int64
	memberResolvers     map[string]MemberResolver
	memberResolversOnce sync.Once
}

func NewConstantSizedType(memoryGauge common.MemoryGauge, typ Type, size int64) *ConstantSizedType {
	common.UseMemory(memoryGauge, common.ConstantSizedSemaTypeMemoryUsage)
	return &ConstantSizedType{
		Type: typ,
		Size: size,
	}
}

func (*ConstantSizedType) IsType() {}

func (*ConstantSizedType) isArrayType() {}

func (t *ConstantSizedType) Tag() TypeTag {
	return ConstantSizedTypeTag
}

func (t *ConstantSizedType) String() string {
	return fmt.Sprintf("[%s; %d]", t.Type, t.Size)
}

func (t *ConstantSizedType) QualifiedString() string {
	return fmt.Sprintf("[%s; %d]", t.Type.QualifiedString(), t.Size)
}

func (t *ConstantSizedType) ID() TypeID {
	return TypeID(fmt.Sprintf("[%s;%d]", t.Type.ID(), t.Size))
}

func (t *ConstantSizedType) Equal(other Type) bool {
	otherArray, ok := other.(*ConstantSizedType)
	if !ok {
		return false
	}

	return t.Type.Equal(otherArray.Type) &&
		t.Size == otherArray.Size
}

func (t *ConstantSizedType) GetMembers() map[string]MemberResolver {
	t.initializeMemberResolvers()
	return t.memberResolvers
}

func (t *ConstantSizedType) initializeMemberResolvers() {
	t.memberResolversOnce.Do(func() {
		t.memberResolvers = getArrayMembers(t)
	})
}

func (t *ConstantSizedType) IsResourceType() bool {
	return t.Type.IsResourceType()
}

func (t *ConstantSizedType) IsInvalidType() bool {
	return t.Type.IsInvalidType()
}

func (t *ConstantSizedType) IsStorable(results map[*Member]bool) bool {
	return t.Type.IsStorable(results)
}

func (t *ConstantSizedType) IsExternallyReturnable(results map[*Member]bool) bool {
	return t.Type.IsStorable(results)
}

func (t *ConstantSizedType) IsImportable(results map[*Member]bool) bool {
	return t.Type.IsImportable(results)
}

func (*ConstantSizedType) IsEquatable() bool {
	// TODO:
	return false
}

func (t *ConstantSizedType) TypeAnnotationState() TypeAnnotationState {
	return t.Type.TypeAnnotationState()
}

func (t *ConstantSizedType) RewriteWithRestrictedTypes() (Type, bool) {
	rewrittenType, rewritten := t.Type.RewriteWithRestrictedTypes()
	if rewritten {
		return &ConstantSizedType{
			Type: rewrittenType,
			Size: t.Size,
		}, true
	} else {
		return t, false
	}
}

func (*ConstantSizedType) isValueIndexableType() bool {
	return true
}

func (*ConstantSizedType) AllowsValueIndexingAssignment() bool {
	return true
}

func (t *ConstantSizedType) ElementType(_ bool) Type {
	return t.Type
}

func (t *ConstantSizedType) IndexingType() Type {
	return IntegerType
}

func (t *ConstantSizedType) Unify(
	other Type,
	typeParameters *TypeParameterTypeOrderedMap,
	report func(err error),
	outerRange ast.Range,
) bool {

	otherArray, ok := other.(*ConstantSizedType)
	if !ok {
		return false
	}

	if t.Size != otherArray.Size {
		return false
	}

	return t.Type.Unify(otherArray.Type, typeParameters, report, outerRange)
}

func (t *ConstantSizedType) Resolve(typeArguments *TypeParameterTypeOrderedMap) Type {
	newInnerType := t.Type.Resolve(typeArguments)
	if newInnerType == nil {
		return nil
	}

	return &ConstantSizedType{
		Type: newInnerType,
		Size: t.Size,
	}
}

// Parameter

func formatParameter(spaces bool, label, identifier, typeAnnotation string) string {
	var builder strings.Builder

	if label != "" {
		builder.WriteString(label)
		if spaces {
			builder.WriteRune(' ')
		}
	}

	if identifier != "" {
		builder.WriteString(identifier)
		builder.WriteRune(':')
		if spaces {
			builder.WriteRune(' ')
		}
	}

	builder.WriteString(typeAnnotation)

	return builder.String()
}

type Parameter struct {
	Label          string
	Identifier     string
	TypeAnnotation *TypeAnnotation
}

func (p *Parameter) String() string {
	return formatParameter(
		true,
		p.Label,
		p.Identifier,
		p.TypeAnnotation.String(),
	)
}

func (p *Parameter) QualifiedString() string {
	return formatParameter(
		true,
		p.Label,
		p.Identifier,
		p.TypeAnnotation.QualifiedString(),
	)
}

// EffectiveArgumentLabel returns the effective argument label that
// an argument in a call must use:
// If no argument label is declared for parameter,
// the parameter name is used as the argument label
//
func (p *Parameter) EffectiveArgumentLabel() string {
	if p.Label != "" {
		return p.Label
	}
	return p.Identifier
}

// TypeParameter

type TypeParameter struct {
	Name      string
	TypeBound Type
	Optional  bool
}

func (p TypeParameter) string(typeFormatter func(Type) string) string {
	var builder strings.Builder
	builder.WriteString(p.Name)
	if p.TypeBound != nil {
		builder.WriteString(": ")
		builder.WriteString(typeFormatter(p.TypeBound))
	}
	return builder.String()
}

func (p TypeParameter) String() string {
	return p.string(func(t Type) string {
		return t.String()
	})
}

func (p TypeParameter) QualifiedString() string {
	return p.string(func(t Type) string {
		return t.QualifiedString()
	})
}

func (p TypeParameter) Equal(other *TypeParameter) bool {
	if p.Name != other.Name {
		return false
	}

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

func (p TypeParameter) checkTypeBound(ty Type, typeRange ast.Range) error {
	if p.TypeBound == nil ||
		p.TypeBound.IsInvalidType() ||
		ty.IsInvalidType() {

		return nil
	}

	if !IsSubType(ty, p.TypeBound) {
		return &TypeMismatchError{
			ExpectedType: p.TypeBound,
			ActualType:   ty,
			Range:        typeRange,
		}
	}

	return nil
}

// Function types

func formatFunctionType(
	spaces bool,
	typeParameters []string,
	parameters []string,
	returnTypeAnnotation string,
) string {

	var builder strings.Builder
	builder.WriteRune('(')

	if len(typeParameters) > 0 {
		builder.WriteRune('<')
		for i, typeParameter := range typeParameters {
			if i > 0 {
				builder.WriteRune(',')
				if spaces {
					builder.WriteRune(' ')
				}
			}
			builder.WriteString(typeParameter)
		}
		builder.WriteRune('>')
	}
	builder.WriteRune('(')
	for i, parameter := range parameters {
		if i > 0 {
			builder.WriteRune(',')
			if spaces {
				builder.WriteRune(' ')
			}
		}
		builder.WriteString(parameter)
	}
	builder.WriteString("):")
	if spaces {
		builder.WriteRune(' ')
	}
	builder.WriteString(returnTypeAnnotation)
	builder.WriteRune(')')
	return builder.String()
}

// FunctionType
//
type FunctionType struct {
	IsConstructor            bool
	TypeParameters           []*TypeParameter
	Parameters               []*Parameter
	ReturnTypeAnnotation     *TypeAnnotation
	RequiredArgumentCount    *int
	ArgumentExpressionsCheck ArgumentExpressionsCheck
	Members                  *StringMemberOrderedMap
}

func RequiredArgumentCount(count int) *int {
	return &count
}

func (*FunctionType) IsType() {}

func (t *FunctionType) Tag() TypeTag {
	return FunctionTypeTag
}

func (t *FunctionType) CheckArgumentExpressions(
	checker *Checker,
	argumentExpressions []ast.Expression,
	invocationRange ast.Range,
) {
	if t.ArgumentExpressionsCheck == nil {
		return
	}
	t.ArgumentExpressionsCheck(checker, argumentExpressions, invocationRange)
}

func (t *FunctionType) String() string {

	typeParameters := make([]string, len(t.TypeParameters))

	for i, typeParameter := range t.TypeParameters {
		typeParameters[i] = typeParameter.String()
	}

	parameters := make([]string, len(t.Parameters))

	for i, parameter := range t.Parameters {
		parameters[i] = parameter.String()
	}

	returnTypeAnnotation := t.ReturnTypeAnnotation.String()

	return formatFunctionType(
		true,
		typeParameters,
		parameters,
		returnTypeAnnotation,
	)
}

func (t *FunctionType) QualifiedString() string {

	typeParameters := make([]string, len(t.TypeParameters))

	for i, typeParameter := range t.TypeParameters {
		typeParameters[i] = typeParameter.QualifiedString()
	}

	parameters := make([]string, len(t.Parameters))

	for i, parameter := range t.Parameters {
		parameters[i] = parameter.QualifiedString()
	}

	returnTypeAnnotation := t.ReturnTypeAnnotation.QualifiedString()

	return formatFunctionType(
		true,
		typeParameters,
		parameters,
		returnTypeAnnotation,
	)
}

// NOTE: parameter names and argument labels are *not* part of the ID!
func (t *FunctionType) ID() TypeID {

	typeParameters := make([]string, len(t.TypeParameters))

	for i, typeParameter := range t.TypeParameters {
		typeParameters[i] = string(typeParameter.TypeBound.ID())
	}

	parameters := make([]string, len(t.Parameters))

	for i, parameter := range t.Parameters {
		parameters[i] = string(parameter.TypeAnnotation.Type.ID())
	}

	returnTypeAnnotation := string(t.ReturnTypeAnnotation.Type.ID())

	return TypeID(
		formatFunctionType(
			false,
			typeParameters,
			parameters,
			returnTypeAnnotation,
		),
	)
}

// NOTE: parameter names and argument labels are intentionally *not* considered!
func (t *FunctionType) Equal(other Type) bool {
	otherFunction, ok := other.(*FunctionType)
	if !ok {
		return false
	}

	// type parameters

	if len(t.TypeParameters) != len(otherFunction.TypeParameters) {
		return false
	}

	for i, typeParameter := range t.TypeParameters {
		otherTypeParameter := otherFunction.TypeParameters[i]
		if !typeParameter.Equal(otherTypeParameter) {
			return false
		}
	}

	// parameters

	if len(t.Parameters) != len(otherFunction.Parameters) {
		return false
	}

	for i, parameter := range t.Parameters {
		otherParameter := otherFunction.Parameters[i]
		if !parameter.TypeAnnotation.Equal(otherParameter.TypeAnnotation) {
			return false
		}
	}

	// Ensures that a constructor function type is
	// NOT equal to a function type with the same parameters, return type, etc.

	if t.IsConstructor != otherFunction.IsConstructor {
		return false
	}

	// return type

	if !t.ReturnTypeAnnotation.Type.
		Equal(otherFunction.ReturnTypeAnnotation.Type) {
		return false
	}

	// constructors

	if t.IsConstructor != otherFunction.IsConstructor {
		return false
	}

	return true
}

func (t *FunctionType) HasSameArgumentLabels(other *FunctionType) bool {
	if len(t.Parameters) != len(other.Parameters) {
		return false
	}

	for i, parameter := range t.Parameters {
		otherParameter := other.Parameters[i]
		if parameter.EffectiveArgumentLabel() != otherParameter.EffectiveArgumentLabel() {
			return false
		}
	}

	return true
}

func (*FunctionType) IsResourceType() bool {
	return false
}

func (t *FunctionType) IsInvalidType() bool {

	for _, typeParameter := range t.TypeParameters {

		if typeParameter.TypeBound != nil &&
			typeParameter.TypeBound.IsInvalidType() {

			return true
		}
	}

	for _, parameter := range t.Parameters {
		if parameter.TypeAnnotation.Type.IsInvalidType() {
			return true
		}
	}

	return t.ReturnTypeAnnotation.Type.IsInvalidType()
}

func (t *FunctionType) IsStorable(_ map[*Member]bool) bool {
	// Functions cannot be stored, as they cannot be serialized
	return false
}

func (t *FunctionType) IsExternallyReturnable(_ map[*Member]bool) bool {
	// Functions cannot be exported, as they cannot be serialized
	return false
}

func (t *FunctionType) IsImportable(_ map[*Member]bool) bool {
	return false
}

func (*FunctionType) IsEquatable() bool {
	return false
}

func (t *FunctionType) TypeAnnotationState() TypeAnnotationState {

	for _, typeParameter := range t.TypeParameters {
		typeParameterTypeAnnotationState := typeParameter.TypeBound.TypeAnnotationState()
		if typeParameterTypeAnnotationState != TypeAnnotationStateValid {
			return typeParameterTypeAnnotationState
		}
	}

	for _, parameter := range t.Parameters {
		parameterTypeAnnotationState := parameter.TypeAnnotation.TypeAnnotationState()
		if parameterTypeAnnotationState != TypeAnnotationStateValid {
			return parameterTypeAnnotationState
		}
	}

	returnTypeAnnotationState := t.ReturnTypeAnnotation.TypeAnnotationState()
	if returnTypeAnnotationState != TypeAnnotationStateValid {
		return returnTypeAnnotationState
	}

	return TypeAnnotationStateValid
}

func (t *FunctionType) RewriteWithRestrictedTypes() (Type, bool) {
	anyRewritten := false

	rewrittenTypeParameterTypeBounds := map[*TypeParameter]Type{}

	for _, typeParameter := range t.TypeParameters {
		if typeParameter.TypeBound == nil {
			continue
		}

		rewrittenType, rewritten := typeParameter.TypeBound.RewriteWithRestrictedTypes()
		if rewritten {
			anyRewritten = true
			rewrittenTypeParameterTypeBounds[typeParameter] = rewrittenType
		}
	}

	rewrittenParameterTypes := map[*Parameter]Type{}

	for _, parameter := range t.Parameters {
		rewrittenType, rewritten := parameter.TypeAnnotation.Type.RewriteWithRestrictedTypes()
		if rewritten {
			anyRewritten = true
			rewrittenParameterTypes[parameter] = rewrittenType
		}
	}

	rewrittenReturnType, rewritten := t.ReturnTypeAnnotation.Type.RewriteWithRestrictedTypes()
	if rewritten {
		anyRewritten = true
	}

	if anyRewritten {
		var rewrittenTypeParameters []*TypeParameter
		if len(t.TypeParameters) > 0 {
			rewrittenTypeParameters = make([]*TypeParameter, len(t.TypeParameters))
			for i, typeParameter := range t.TypeParameters {
				rewrittenTypeBound, ok := rewrittenTypeParameterTypeBounds[typeParameter]
				if ok {
					rewrittenTypeParameters[i] = &TypeParameter{
						Name:      typeParameter.Name,
						TypeBound: rewrittenTypeBound,
						Optional:  typeParameter.Optional,
					}
				} else {
					rewrittenTypeParameters[i] = typeParameter
				}
			}
		}

		var rewrittenParameters []*Parameter
		if len(t.Parameters) > 0 {
			rewrittenParameters = make([]*Parameter, len(t.Parameters))
			for i, parameter := range t.Parameters {
				rewrittenParameterType, ok := rewrittenParameterTypes[parameter]
				if ok {
					rewrittenParameters[i] = &Parameter{
						Label:          parameter.Label,
						Identifier:     parameter.Identifier,
						TypeAnnotation: NewTypeAnnotation(rewrittenParameterType),
					}
				} else {
					rewrittenParameters[i] = parameter
				}
			}
		}

		return &FunctionType{
			TypeParameters:        rewrittenTypeParameters,
			Parameters:            rewrittenParameters,
			ReturnTypeAnnotation:  NewTypeAnnotation(rewrittenReturnType),
			RequiredArgumentCount: t.RequiredArgumentCount,
		}, true
	} else {
		return t, false
	}
}

func (t *FunctionType) ArgumentLabels() (argumentLabels []string) {

	for _, parameter := range t.Parameters {

		argumentLabel := ArgumentLabelNotRequired
		if parameter.Label != "" {
			argumentLabel = parameter.Label
		} else if parameter.Identifier != "" {
			argumentLabel = parameter.Identifier
		}

		argumentLabels = append(argumentLabels, argumentLabel)
	}

	return
}

func (t *FunctionType) Unify(
	other Type,
	typeParameters *TypeParameterTypeOrderedMap,
	report func(err error),
	outerRange ast.Range,
) (
	result bool,
) {

	otherFunction, ok := other.(*FunctionType)
	if !ok {
		return false
	}

	// TODO: type parameters ?

	if len(t.TypeParameters) > 0 ||
		len(otherFunction.TypeParameters) > 0 {

		return false
	}

	// parameters

	if len(t.Parameters) != len(otherFunction.Parameters) {
		return false
	}

	for i, parameter := range t.Parameters {
		otherParameter := otherFunction.Parameters[i]
		parameterUnified := parameter.TypeAnnotation.Type.Unify(
			otherParameter.TypeAnnotation.Type,
			typeParameters,
			report,
			outerRange,
		)
		result = result || parameterUnified
	}

	// return type

	returnTypeUnified := t.ReturnTypeAnnotation.Type.Unify(
		otherFunction.ReturnTypeAnnotation.Type,
		typeParameters,
		report,
		outerRange,
	)

	result = result || returnTypeUnified

	return
}

func (t *FunctionType) Resolve(typeArguments *TypeParameterTypeOrderedMap) Type {

	// TODO: type parameters ?

	// parameters

	var newParameters []*Parameter

	for _, parameter := range t.Parameters {
		newParameterType := parameter.TypeAnnotation.Type.Resolve(typeArguments)
		if newParameterType == nil {
			return nil
		}

		newParameters = append(newParameters,
			&Parameter{
				Label:          parameter.Label,
				Identifier:     parameter.Identifier,
				TypeAnnotation: NewTypeAnnotation(newParameterType),
			},
		)
	}

	// return type

	newReturnType := t.ReturnTypeAnnotation.Type.Resolve(typeArguments)
	if newReturnType == nil {
		return nil
	}

	return &FunctionType{
		Parameters:            newParameters,
		ReturnTypeAnnotation:  NewTypeAnnotation(newReturnType),
		RequiredArgumentCount: t.RequiredArgumentCount,
	}

}

func (t *FunctionType) GetMembers() map[string]MemberResolver {
	// TODO: optimize
	var members map[string]MemberResolver
	if t.Members != nil {
		members = make(map[string]MemberResolver, t.Members.Len())
		t.Members.Foreach(func(name string, loopMember *Member) {
			// NOTE: don't capture loop variable
			member := loopMember
			members[name] = MemberResolver{
				Kind: member.DeclarationKind,
				Resolve: func(_ common.MemoryGauge, _ string, _ ast.Range, _ func(error)) *Member {
					return member
				},
			}
		})
	}
	return withBuiltinMembers(t, members)
}

type ArgumentExpressionsCheck func(
	checker *Checker,
	argumentExpressions []ast.Expression,
	invocationRange ast.Range,
)

// BaseTypeActivation is the base activation that contains
// the types available in programs
//
var BaseTypeActivation = NewVariableActivation(nil)

func init() {

	types := AllNumberTypes[:]

	types = append(
		types,
		MetaType,
		VoidType,
		AnyStructType,
		AnyResourceType,
		NeverType,
		BoolType,
		CharacterType,
		StringType,
		&AddressType{},
		AuthAccountType,
		PublicAccountType,
		PathType,
		StoragePathType,
		CapabilityPathType,
		PrivatePathType,
		PublicPathType,
		&CapabilityType{},
		DeployedContractType,
		BlockType,
		AccountKeyType,
		PublicKeyType,
		SignatureAlgorithmType,
		HashAlgorithmType,
	)

	for _, ty := range types {
		typeName := ty.String()

		// Check that the type is not accidentally redeclared

		if BaseTypeActivation.Find(typeName) != nil {
			panic(errors.NewUnreachableError())
		}

		BaseTypeActivation.Set(
			typeName,
			baseTypeVariable(typeName, ty),
		)
	}

	// The AST contains empty type annotations, resolve them to Void

	BaseTypeActivation.Set(
		"",
		BaseTypeActivation.Find("Void"),
	)
}

func baseTypeVariable(name string, ty Type) *Variable {
	return &Variable{
		Identifier:      name,
		Type:            ty,
		DeclarationKind: common.DeclarationKindType,
		IsConstant:      true,
		IsBaseValue:     true,
		Access:          ast.AccessPublic,
	}
}

// BaseValueActivation is the base activation that contains
// the values available in programs
//
var BaseValueActivation = NewVariableActivation(nil)

var AllSignedFixedPointTypes = []Type{
	Fix64Type,
}

var AllUnsignedFixedPointTypes = []Type{
	UFix64Type,
}

var AllFixedPointTypes = append(
	append(
		AllUnsignedFixedPointTypes[:],
		AllSignedFixedPointTypes...,
	),
	FixedPointType,
	SignedFixedPointType,
)

var AllSignedIntegerTypes = []Type{
	IntType,
	Int8Type,
	Int16Type,
	Int32Type,
	Int64Type,
	Int128Type,
	Int256Type,
}

var AllUnsignedIntegerTypes = []Type{
	// UInt*
	UIntType,
	UInt8Type,
	UInt16Type,
	UInt32Type,
	UInt64Type,
	UInt128Type,
	UInt256Type,
	// Word*
	Word8Type,
	Word16Type,
	Word32Type,
	Word64Type,
}

var AllIntegerTypes = append(
	append(
		AllUnsignedIntegerTypes[:],
		AllSignedIntegerTypes...,
	),
	IntegerType,
	SignedIntegerType,
)

var AllNumberTypes = append(
	append(
		AllIntegerTypes[:],
		AllFixedPointTypes...,
	),
	NumberType,
	SignedNumberType,
)

const NumberTypeMinFieldName = "min"
const NumberTypeMaxFieldName = "max"

const numberTypeMinFieldDocString = `The minimum integer of this type`
const numberTypeMaxFieldDocString = `The maximum integer of this type`

const fixedPointNumberTypeMinFieldDocString = `The minimum fixed-point value of this type`
const fixedPointNumberTypeMaxFieldDocString = `The maximum fixed-point value of this type`

const numberConversionFunctionDocStringSuffix = `
The value must be within the bounds of this type.
If a value is passed that is outside the bounds, the program aborts.`

func init() {

	// Declare a conversion function for all (leaf) number types

	for _, numberType := range AllNumberTypes {

		switch numberType {
		case NumberType, SignedNumberType,
			IntegerType, SignedIntegerType,
			FixedPointType, SignedFixedPointType:
			continue

		default:
			typeName := numberType.String()

			// Check that the function is not accidentally redeclared

			if BaseValueActivation.Find(typeName) != nil {
				panic(errors.NewUnreachableError())
			}

			functionType := NumberConversionFunctionType(numberType)

			addMember := func(member *Member) {
				if functionType.Members == nil {
					functionType.Members = NewStringMemberOrderedMap()
				}
				name := member.Identifier.Identifier
				_, exists := functionType.Members.Get(name)
				if exists {
					panic(errors.NewUnreachableError())
				}
				functionType.Members.Set(name, member)
			}

			switch numberType := numberType.(type) {
			case *NumericType:
				if numberType.minInt != nil {
					addMember(NewUnmeteredPublicConstantFieldMember(
						functionType,
						NumberTypeMinFieldName,
						numberType,
						numberTypeMinFieldDocString,
					))
				}

				if numberType.maxInt != nil {
					addMember(NewUnmeteredPublicConstantFieldMember(
						functionType,
						NumberTypeMaxFieldName,
						numberType,
						numberTypeMaxFieldDocString,
					))
				}

			case *FixedPointNumericType:
				if numberType.minInt != nil {
					// If a minimum integer is set, a minimum fractional must be set
					if numberType.minFractional == nil {
						panic(errors.NewUnreachableError())
					}

					addMember(NewUnmeteredPublicConstantFieldMember(
						functionType,
						NumberTypeMinFieldName,
						numberType,
						fixedPointNumberTypeMinFieldDocString,
					))
				}

				if numberType.maxInt != nil {
					// If a maximum integer is set, a maximum fractional must be set
					if numberType.maxFractional == nil {
						panic(errors.NewUnreachableError())
					}

					addMember(NewUnmeteredPublicConstantFieldMember(
						functionType,
						NumberTypeMaxFieldName,
						numberType,
						fixedPointNumberTypeMaxFieldDocString,
					))
				}
			}

			BaseValueActivation.Set(
				typeName,
				baseFunctionVariable(
					typeName,
					functionType,
					numberConversionDocString(
						fmt.Sprintf("the type %s", numberType.String()),
					),
				),
			)
		}
	}
}

func NumberConversionFunctionType(numberType Type) *FunctionType {
	return &FunctionType{
		Parameters: []*Parameter{
			{
				Label:          ArgumentLabelNotRequired,
				Identifier:     "value",
				TypeAnnotation: NewTypeAnnotation(NumberType),
			},
		},
		ReturnTypeAnnotation:     NewTypeAnnotation(numberType),
		ArgumentExpressionsCheck: numberFunctionArgumentExpressionsChecker(numberType),
	}
}

func numberConversionDocString(targetDescription string) string {
	return fmt.Sprintf(
		"Converts the given number to %s. %s",
		targetDescription,
		numberConversionFunctionDocStringSuffix,
	)
}

func baseFunctionVariable(name string, ty *FunctionType, docString string) *Variable {
	return &Variable{
		Identifier:      name,
		DeclarationKind: common.DeclarationKindFunction,
		ArgumentLabels:  ty.ArgumentLabels(),
		IsConstant:      true,
		IsBaseValue:     true,
		Type:            ty,
		Access:          ast.AccessPublic,
		DocString:       docString,
	}
}

var AddressConversionFunctionType = &FunctionType{
	Parameters: []*Parameter{
		{
			Label:          ArgumentLabelNotRequired,
			Identifier:     "value",
			TypeAnnotation: NewTypeAnnotation(IntegerType),
		},
	},
	ReturnTypeAnnotation: NewTypeAnnotation(&AddressType{}),
	ArgumentExpressionsCheck: func(checker *Checker, argumentExpressions []ast.Expression, _ ast.Range) {
		if len(argumentExpressions) < 1 {
			return
		}

		intExpression, ok := argumentExpressions[0].(*ast.IntegerExpression)
		if !ok {
			return
		}

		// No need to meter. This is only checked once.
		CheckAddressLiteral(nil, intExpression, checker.report)
	},
}

func init() {
	// Declare a conversion function for the address type

	// Check that the function is not accidentally redeclared

	typeName := AddressTypeName

	if BaseValueActivation.Find(typeName) != nil {
		panic(errors.NewUnreachableError())
	}

	BaseValueActivation.Set(
		typeName,
		baseFunctionVariable(
			typeName,
			AddressConversionFunctionType,
			numberConversionDocString("an address"),
		),
	)
}

func numberFunctionArgumentExpressionsChecker(targetType Type) ArgumentExpressionsCheck {
	return func(checker *Checker, arguments []ast.Expression, invocationRange ast.Range) {
		if len(arguments) < 1 {
			return
		}

		argument := arguments[0]

		switch argument := argument.(type) {
		case *ast.IntegerExpression:
			if CheckIntegerLiteral(nil, argument, targetType, checker.report) {
				if checker.lintEnabled {
					suggestIntegerLiteralConversionReplacement(checker, argument, targetType, invocationRange)
				}
			}

		case *ast.FixedPointExpression:
			if CheckFixedPointLiteral(nil, argument, targetType, checker.report) {
				if checker.lintEnabled {
					suggestFixedPointLiteralConversionReplacement(checker, targetType, argument, invocationRange)
				}
			}
		}
	}
}

func init() {

	// Declare a function for the string type.
	// For now it has no parameters and creates an empty string

	typeName := StringType.String()

	// Check that the function is not accidentally redeclared

	if BaseValueActivation.Find(typeName) != nil {
		panic(errors.NewUnreachableError())
	}

	functionType := &FunctionType{
		ReturnTypeAnnotation: NewTypeAnnotation(StringType),
	}

	addMember := func(member *Member) {
		if functionType.Members == nil {
			functionType.Members = NewStringMemberOrderedMap()
		}
		name := member.Identifier.Identifier
		_, exists := functionType.Members.Get(name)
		if exists {
			panic(errors.NewUnreachableError())
		}
		functionType.Members.Set(name, member)
	}

	addMember(NewUnmeteredPublicFunctionMember(
		functionType,
		StringTypeEncodeHexFunctionName,
		StringTypeEncodeHexFunctionType,
		StringTypeEncodeHexFunctionDocString,
	))

	BaseValueActivation.Set(
		typeName,
		baseFunctionVariable(
			typeName,
			functionType,
			"Creates an empty string",
		),
	)
}

var StringTypeEncodeHexFunctionType = &FunctionType{
	Parameters: []*Parameter{
		{
			Label:      ArgumentLabelNotRequired,
			Identifier: "data",
			TypeAnnotation: NewTypeAnnotation(
				ByteArrayType,
			),
		},
	},
	ReturnTypeAnnotation: NewTypeAnnotation(
		StringType,
	),
}

func suggestIntegerLiteralConversionReplacement(
	checker *Checker,
	argument *ast.IntegerExpression,
	targetType Type,
	invocationRange ast.Range,
) {
	negative := argument.Value.Sign() < 0

	if IsSameTypeKind(targetType, FixedPointType) {

		// If the integer literal is converted to a fixed-point type,
		// suggest replacing it with a fixed-point literal

		signed := IsSameTypeKind(targetType, SignedFixedPointType)

		var hintExpression ast.Expression = ast.NewFixedPointExpression(
			checker.memoryGauge,
			"",
			negative,
			new(big.Int).Abs(argument.Value),
			new(big.Int),
			1,
			argument.Range,
		)

		// If the fixed-point literal is positive
		// and the the target fixed-point type is signed,
		// then a static cast is required

		if !negative && signed {
			hintExpression = ast.NewCastingExpression(
				checker.memoryGauge,
				hintExpression,
				ast.OperationCast,
				ast.NewTypeAnnotation(
					checker.memoryGauge,
					false,
					ast.NewNominalType(
						checker.memoryGauge,
						ast.NewIdentifier(
							checker.memoryGauge,
							targetType.String(),
							ast.EmptyPosition,
						),
						nil,
					),
					ast.EmptyPosition,
				),
				nil,
			)
		}

		checker.hint(
			&ReplacementHint{
				Expression: hintExpression,
				Range:      invocationRange,
			},
		)

	} else if IsSameTypeKind(targetType, IntegerType) {

		// If the integer literal is converted to an integer type,
		// suggest replacing it with a fixed-point literal

		var hintExpression ast.Expression = argument

		// If the target type is not `Int`,
		// then a static cast is required,
		// as all integer literals (positive and negative)
		// are inferred to be of type `Int`

		if !IsSameTypeKind(targetType, IntType) {
			hintExpression = ast.NewCastingExpression(
				checker.memoryGauge,
				hintExpression,
				ast.OperationCast,
				ast.NewTypeAnnotation(
					checker.memoryGauge,
					false,
					ast.NewNominalType(
						checker.memoryGauge,
						ast.NewIdentifier(
							checker.memoryGauge,
							targetType.String(),
							ast.EmptyPosition,
						),
						nil,
					),
					ast.EmptyPosition,
				),
				nil,
			)
		}

		checker.hint(
			&ReplacementHint{
				Expression: hintExpression,
				Range:      invocationRange,
			},
		)
	}
}

func suggestFixedPointLiteralConversionReplacement(
	checker *Checker,
	targetType Type,
	argument *ast.FixedPointExpression,
	invocationRange ast.Range,
) {
	// If the fixed-point literal is converted to a fixed-point type,
	// suggest replacing it with a fixed-point literal

	if !IsSameTypeKind(targetType, FixedPointType) {
		return
	}

	negative := argument.Negative
	signed := IsSameTypeKind(targetType, SignedFixedPointType)

	if (!negative && !signed) || (negative && signed) {
		checker.hint(
			&ReplacementHint{
				Expression: argument,
				Range:      invocationRange,
			},
		)
	}
}

func pathConversionFunctionType(pathType Type) *FunctionType {
	return &FunctionType{
		Parameters: []*Parameter{
			{
				Identifier:     "identifier",
				TypeAnnotation: NewTypeAnnotation(StringType),
			},
		},
		ReturnTypeAnnotation: NewTypeAnnotation(
			&OptionalType{
				Type: pathType,
			},
		),
	}
}

var PublicPathConversionFunctionType = pathConversionFunctionType(PublicPathType)
var PrivatePathConversionFunctionType = pathConversionFunctionType(PrivatePathType)
var StoragePathConversionFunctionType = pathConversionFunctionType(StoragePathType)

func init() {

	// Declare the run-time type construction function

	typeName := MetaTypeName

	// Check that the function is not accidentally redeclared

	if BaseValueActivation.Find(typeName) != nil {
		panic(errors.NewUnreachableError())
	}

	BaseValueActivation.Set(
		typeName,
		baseFunctionVariable(
			typeName,
			&FunctionType{
				TypeParameters:       []*TypeParameter{{Name: "T"}},
				ReturnTypeAnnotation: NewTypeAnnotation(MetaType),
			},
			"Creates a run-time type representing the given static type as a value",
		),
	)

	BaseValueActivation.Set(
		PublicPathType.String(),
		baseFunctionVariable(
			PublicPathType.String(),
			PublicPathConversionFunctionType,
			"Converts the given string into a public path. Returns nil if the string does not specify a public path",
		),
	)

	BaseValueActivation.Set(
		PrivatePathType.String(),
		baseFunctionVariable(
			PrivatePathType.String(),
			PrivatePathConversionFunctionType,
			"Converts the given string into a private path. Returns nil if the string does not specify a private path",
		),
	)

	BaseValueActivation.Set(
		StoragePathType.String(),
		baseFunctionVariable(
			StoragePathType.String(),
			StoragePathConversionFunctionType,
			"Converts the given string into a storage path. Returns nil if the string does not specify a storage path",
		),
	)

	for _, v := range runtimeTypeConstructors {
		BaseValueActivation.Set(
			v.Name,
			baseFunctionVariable(
				v.Name,
				v.Value,
				v.DocString,
			))
	}
}

// CompositeType

type EnumInfo struct {
	RawType Type
	Cases   []string
}

type CompositeType struct {
	Location   common.Location
	Identifier string
	Kind       common.CompositeKind
	// an internal set of field `ExplicitInterfaceConformances`
	explicitInterfaceConformanceSet     *InterfaceSet
	explicitInterfaceConformanceSetOnce sync.Once
	ExplicitInterfaceConformances       []*InterfaceType
	ImplicitTypeRequirementConformances []*CompositeType
	Members                             *StringMemberOrderedMap
	memberResolvers                     map[string]MemberResolver
	memberResolversOnce                 sync.Once
	Fields                              []string
	// TODO: add support for overloaded initializers
	ConstructorParameters []*Parameter
	nestedTypes           *StringTypeOrderedMap
	containerType         Type
	EnumRawType           Type
	hasComputedMembers    bool

	// Only applicable for native composite types.
	importable bool

	cachedIdentifiers *struct {
		TypeID              TypeID
		QualifiedIdentifier string
	}
	cachedIdentifiersLock sync.RWMutex
}

func (t *CompositeType) Tag() TypeTag {
	return CompositeTypeTag
}

func (t *CompositeType) ExplicitInterfaceConformanceSet() *InterfaceSet {
	t.initializeExplicitInterfaceConformanceSet()
	return t.explicitInterfaceConformanceSet
}

func (t *CompositeType) initializeExplicitInterfaceConformanceSet() {
	t.explicitInterfaceConformanceSetOnce.Do(func() {
		// TODO: also include conformances' conformances recursively
		//   once interface can have conformances

		t.explicitInterfaceConformanceSet = NewInterfaceSet()
		for _, conformance := range t.ExplicitInterfaceConformances {
			t.explicitInterfaceConformanceSet.Add(conformance)
		}
	})
}

func (t *CompositeType) addImplicitTypeRequirementConformance(typeRequirement *CompositeType) {
	t.ImplicitTypeRequirementConformances =
		append(t.ImplicitTypeRequirementConformances, typeRequirement)
}

func (*CompositeType) IsType() {}

func (t *CompositeType) String() string {
	return t.Identifier
}

func (t *CompositeType) QualifiedString() string {
	return t.QualifiedIdentifier()
}

func (t *CompositeType) GetContainerType() Type {
	return t.containerType
}

func (t *CompositeType) SetContainerType(containerType Type) {
	t.checkIdentifiersCached()
	t.containerType = containerType
}

func (t *CompositeType) checkIdentifiersCached() {
	t.cachedIdentifiersLock.Lock()
	defer t.cachedIdentifiersLock.Unlock()

	if t.cachedIdentifiers != nil {
		panic(errors.NewUnreachableError())
	}

	if t.nestedTypes != nil {
		t.nestedTypes.Foreach(checkIdentifiersCached)
	}
}

func checkIdentifiersCached(_ string, typ Type) {
	switch semaType := typ.(type) {
	case *CompositeType:
		semaType.checkIdentifiersCached()
	case *InterfaceType:
		semaType.checkIdentifiersCached()
	}
}

func (t *CompositeType) GetCompositeKind() common.CompositeKind {
	return t.Kind
}

func (t *CompositeType) GetLocation() common.Location {
	return t.Location
}

func (t *CompositeType) QualifiedIdentifier() string {
	t.initializeIdentifiers()
	return t.cachedIdentifiers.QualifiedIdentifier
}

func (t *CompositeType) ID() TypeID {
	t.initializeIdentifiers()
	return t.cachedIdentifiers.TypeID
}

func (t *CompositeType) initializeIdentifiers() {
	t.cachedIdentifiersLock.Lock()
	defer t.cachedIdentifiersLock.Unlock()

	if t.cachedIdentifiers != nil {
		return
	}

	identifier := qualifiedIdentifier(t.Identifier, t.containerType)

	var typeID TypeID
	if t.Location == nil {
		typeID = TypeID(identifier)
	} else {
		typeID = t.Location.TypeID(nil, identifier)
	}

	t.cachedIdentifiers = &struct {
		TypeID              TypeID
		QualifiedIdentifier string
	}{
		TypeID:              typeID,
		QualifiedIdentifier: identifier,
	}
}

func (t *CompositeType) Equal(other Type) bool {
	otherStructure, ok := other.(*CompositeType)
	if !ok {
		return false
	}

	return otherStructure.Kind == t.Kind &&
		otherStructure.ID() == t.ID()
}

func (t *CompositeType) GetMembers() map[string]MemberResolver {
	t.initializeMemberResolvers()
	return t.memberResolvers
}

func (t *CompositeType) IsResourceType() bool {
	return t.Kind == common.CompositeKindResource
}

func (*CompositeType) IsInvalidType() bool {
	return false
}

func (t *CompositeType) IsStorable(results map[*Member]bool) bool {
	if t.hasComputedMembers {
		return false
	}

	// Only structures, resources, and enums can be stored

	switch t.Kind {
	case common.CompositeKindStructure,
		common.CompositeKindResource,
		common.CompositeKindEnum:
		break
	default:
		return false
	}

	// Native/built-in types are not storable for now
	if t.Location == nil {
		return false
	}

	// If this composite type has a member which is non-storable,
	// then the composite type is not storable.

	for pair := t.Members.Oldest(); pair != nil; pair = pair.Next() {
		if !pair.Value.IsStorable(results) {
			return false
		}
	}

	return true
}

func (t *CompositeType) IsImportable(results map[*Member]bool) bool {
	// Use the pre-determined flag for native types
	if t.Location == nil {
		return t.importable
	}

	// Only structures and enums can be imported

	switch t.Kind {
	case common.CompositeKindStructure,
		common.CompositeKindEnum:
		break
	default:
		return false
	}

	// If this composite type has a member which is not importable,
	// then the composite type is not importable.

	for pair := t.Members.Oldest(); pair != nil; pair = pair.Next() {
		if !pair.Value.IsImportable(results) {
			return false
		}
	}

	return true
}

func (t *CompositeType) IsExternallyReturnable(results map[*Member]bool) bool {
	// Only structures, resources, and enums can be stored

	switch t.Kind {
	case common.CompositeKindStructure,
		common.CompositeKindResource,
		common.CompositeKindEnum:
		break
	default:
		return false
	}

	// If this composite type has a member which is not externally returnable,
	// then the composite type is not externally returnable.

	for p := t.Members.Oldest(); p != nil; p = p.Next() {
		if !p.Value.IsExternallyReturnable(results) {
			return false
		}
	}

	return true
}

func (t *CompositeType) IsEquatable() bool {
	// TODO: add support for more composite kinds
	return t.Kind == common.CompositeKindEnum
}

func (*CompositeType) TypeAnnotationState() TypeAnnotationState {
	return TypeAnnotationStateValid
}

func (t *CompositeType) RewriteWithRestrictedTypes() (result Type, rewritten bool) {
	return t, false
}

func (t *CompositeType) InterfaceType() *InterfaceType {
	return &InterfaceType{
		Location:              t.Location,
		Identifier:            t.Identifier,
		CompositeKind:         t.Kind,
		Members:               t.Members,
		Fields:                t.Fields,
		InitializerParameters: t.ConstructorParameters,
		containerType:         t.containerType,
		nestedTypes:           t.nestedTypes,
	}
}

func (t *CompositeType) TypeRequirements() []*CompositeType {

	var typeRequirements []*CompositeType

	if containerComposite, ok := t.containerType.(*CompositeType); ok {
		for _, conformance := range containerComposite.ExplicitInterfaceConformances {
			ty, ok := conformance.nestedTypes.Get(t.Identifier)
			if !ok {
				continue
			}

			typeRequirement, ok := ty.(*CompositeType)
			if !ok {
				continue
			}

			typeRequirements = append(typeRequirements, typeRequirement)
		}
	}

	return typeRequirements
}

func (*CompositeType) Unify(_ Type, _ *TypeParameterTypeOrderedMap, _ func(err error), _ ast.Range) bool {
	// TODO:
	return false
}

func (t *CompositeType) Resolve(_ *TypeParameterTypeOrderedMap) Type {
	return t
}

func (t *CompositeType) IsContainerType() bool {
	return t.nestedTypes != nil
}

func (t *CompositeType) GetNestedTypes() *StringTypeOrderedMap {
	return t.nestedTypes
}

func (t *CompositeType) initializeMemberResolvers() {
	t.memberResolversOnce.Do(func() {
		members := make(map[string]MemberResolver, t.Members.Len())

		t.Members.Foreach(func(name string, loopMember *Member) {
			// NOTE: don't capture loop variable
			member := loopMember
			members[name] = MemberResolver{
				Kind: member.DeclarationKind,
				Resolve: func(_ common.MemoryGauge, _ string, _ ast.Range, _ func(error)) *Member {
					return member
				},
			}
		})

		// Check conformances.
		// If this composite type results from a normal composite declaration,
		// it must have members declared for all interfaces it conforms to.
		// However, if this composite type is a type requirement,
		// it acts like an interface and does not have to declare members.

		t.ExplicitInterfaceConformanceSet().
			ForEach(func(conformance *InterfaceType) {
				for name, resolver := range conformance.GetMembers() { //nolint:maprangecheck
					if _, ok := members[name]; !ok {
						members[name] = resolver
					}
				}
			})

		t.memberResolvers = withBuiltinMembers(t, members)
	})
}

func (t *CompositeType) FieldPosition(name string, declaration *ast.CompositeDeclaration) ast.Position {
	var pos ast.Position
	if t.Kind == common.CompositeKindEnum &&
		name == EnumRawValueFieldName {

		if len(declaration.Conformances) > 0 {
			pos = declaration.Conformances[0].StartPosition()
		}
	} else {
		pos = declaration.Members.FieldPosition(name, declaration.CompositeKind)
	}
	return pos
}

// Member

type Member struct {
	ContainerType  Type
	Access         ast.Access
	Identifier     ast.Identifier
	TypeAnnotation *TypeAnnotation
	// TODO: replace with dedicated MemberKind enum
	DeclarationKind common.DeclarationKind
	VariableKind    ast.VariableKind
	ArgumentLabels  []string
	// Predeclared fields can be considered initialized
	Predeclared bool
	// IgnoreInSerialization fields are ignored in serialization
	IgnoreInSerialization bool
	DocString             string
}

func NewUnmeteredPublicFunctionMember(
	containerType Type,
	identifier string,
	functionType *FunctionType,
	docString string,
) *Member {
	return NewPublicFunctionMember(
		nil,
		containerType,
		identifier,
		functionType,
		docString,
	)
}

func NewPublicFunctionMember(
	memoryGauge common.MemoryGauge,
	containerType Type,
	identifier string,
	functionType *FunctionType,
	docString string,
) *Member {

	return &Member{
		ContainerType: containerType,
		Access:        ast.AccessPublic,
		Identifier: ast.NewIdentifier(
			memoryGauge,
			identifier,
			ast.EmptyPosition,
		),
		DeclarationKind: common.DeclarationKindFunction,
		VariableKind:    ast.VariableKindConstant,
		TypeAnnotation:  NewTypeAnnotation(functionType),
		ArgumentLabels:  functionType.ArgumentLabels(),
		DocString:       docString,
	}
}

func NewUnmeteredPublicConstantFieldMember(
	containerType Type,
	identifier string,
	fieldType Type,
	docString string,
) *Member {
	return NewPublicConstantFieldMember(
		nil,
		containerType,
		identifier,
		fieldType,
		docString,
	)
}

func NewPublicConstantFieldMember(
	memoryGauge common.MemoryGauge,
	containerType Type,
	identifier string,
	fieldType Type,
	docString string,
) *Member {
	return &Member{
		ContainerType: containerType,
		Access:        ast.AccessPublic,
		Identifier: ast.NewIdentifier(
			memoryGauge,
			identifier,
			ast.EmptyPosition,
		),
		DeclarationKind: common.DeclarationKindField,
		VariableKind:    ast.VariableKindConstant,
		TypeAnnotation:  NewTypeAnnotation(fieldType),
		DocString:       docString,
	}
}

// IsStorable returns whether a member is a storable field
func (m *Member) IsStorable(results map[*Member]bool) (result bool) {
	test := func(t Type) bool {
		return t.IsStorable(results)
	}
	return m.testType(test, results)
}

// IsExternallyReturnable returns whether a member is externally returnable
func (m *Member) IsExternallyReturnable(results map[*Member]bool) (result bool) {
	test := func(t Type) bool {
		return t.IsExternallyReturnable(results)
	}
	return m.testType(test, results)
}

// IsImportable returns whether a member can be imported to a program
func (m *Member) IsImportable(results map[*Member]bool) (result bool) {
	test := func(t Type) bool {
		return t.IsImportable(results)
	}
	return m.testType(test, results)
}

// IsValidEventParameterType returns whether has a valid event parameter type
func (m *Member) IsValidEventParameterType(results map[*Member]bool) bool {
	test := func(t Type) bool {
		return IsValidEventParameterType(t, results)
	}
	return m.testType(test, results)
}

func (m *Member) testType(test func(Type) bool, results map[*Member]bool) (result bool) {

	// Prevent a potential stack overflow due to cyclic declarations
	// by keeping track of the result for each member

	// If a result for the member is available, return it,
	// instead of checking the type

	var ok bool
	if result, ok = results[m]; ok {
		return result
	}

	// Temporarily assume the member passes the test while it's type is tested.
	// If a recursive call occurs, the check for an existing result will prevent infinite recursion

	results[m] = true

	result = func() bool {
		// Skip checking predeclared members

		if m.Predeclared {
			return true
		}

		if m.DeclarationKind == common.DeclarationKindField {

			fieldType := m.TypeAnnotation.Type

			if !fieldType.IsInvalidType() && !test(fieldType) {
				return false
			}
		}

		return true
	}()

	results[m] = result
	return result
}

// InterfaceType

type InterfaceType struct {
	Location            common.Location
	Identifier          string
	CompositeKind       common.CompositeKind
	Members             *StringMemberOrderedMap
	memberResolvers     map[string]MemberResolver
	memberResolversOnce sync.Once
	Fields              []string
	// TODO: add support for overloaded initializers
	InitializerParameters []*Parameter
	containerType         Type
	nestedTypes           *StringTypeOrderedMap
	cachedIdentifiers     *struct {
		TypeID              TypeID
		QualifiedIdentifier string
	}
	cachedIdentifiersLock sync.RWMutex
}

func (*InterfaceType) IsType() {}

func (t *InterfaceType) Tag() TypeTag {
	return InterfaceTypeTag
}

func (t *InterfaceType) String() string {
	return t.Identifier
}

func (t *InterfaceType) QualifiedString() string {
	return t.QualifiedIdentifier()
}

func (t *InterfaceType) GetContainerType() Type {
	return t.containerType
}

func (t *InterfaceType) SetContainerType(containerType Type) {
	t.checkIdentifiersCached()
	t.containerType = containerType
}

func (t *InterfaceType) checkIdentifiersCached() {
	t.cachedIdentifiersLock.Lock()
	defer t.cachedIdentifiersLock.Unlock()

	if t.cachedIdentifiers != nil {
		panic(errors.NewUnreachableError())
	}

	if t.nestedTypes != nil {
		t.nestedTypes.Foreach(checkIdentifiersCached)
	}
}

func (t *InterfaceType) GetCompositeKind() common.CompositeKind {
	return t.CompositeKind
}

func (t *InterfaceType) GetLocation() common.Location {
	return t.Location
}

func (t *InterfaceType) QualifiedIdentifier() string {
	t.initializeIdentifiers()
	return t.cachedIdentifiers.QualifiedIdentifier
}

func (t *InterfaceType) ID() TypeID {
	t.initializeIdentifiers()
	return t.cachedIdentifiers.TypeID
}

func (t *InterfaceType) initializeIdentifiers() {
	t.cachedIdentifiersLock.Lock()
	defer t.cachedIdentifiersLock.Unlock()

	if t.cachedIdentifiers != nil {
		return
	}

	identifier := qualifiedIdentifier(t.Identifier, t.containerType)

	var typeID TypeID
	if t.Location == nil {
		typeID = TypeID(identifier)
	} else {
		typeID = t.Location.TypeID(nil, identifier)
	}

	t.cachedIdentifiers = &struct {
		TypeID              TypeID
		QualifiedIdentifier string
	}{
		TypeID:              typeID,
		QualifiedIdentifier: identifier,
	}
}

func (t *InterfaceType) Equal(other Type) bool {
	otherInterface, ok := other.(*InterfaceType)
	if !ok {
		return false
	}

	return otherInterface.CompositeKind == t.CompositeKind &&
		otherInterface.ID() == t.ID()
}

func (t *InterfaceType) GetMembers() map[string]MemberResolver {
	t.initializeMemberResolvers()
	return t.memberResolvers
}

func (t *InterfaceType) initializeMemberResolvers() {
	t.memberResolversOnce.Do(func() {
		members := make(map[string]MemberResolver, t.Members.Len())
		t.Members.Foreach(func(name string, loopMember *Member) {
			// NOTE: don't capture loop variable
			member := loopMember
			members[name] = MemberResolver{
				Kind: member.DeclarationKind,
				Resolve: func(_ common.MemoryGauge, _ string, _ ast.Range, _ func(error)) *Member {
					return member
				},
			}
		})

		t.memberResolvers = withBuiltinMembers(t, members)
	})
}

func (t *InterfaceType) IsResourceType() bool {
	return t.CompositeKind == common.CompositeKindResource
}

func (t *InterfaceType) IsInvalidType() bool {
	return false
}

func (t *InterfaceType) IsStorable(results map[*Member]bool) bool {

	// If this interface type has a member which is non-storable,
	// then the interface type is not storable.

	for pair := t.Members.Oldest(); pair != nil; pair = pair.Next() {
		if !pair.Value.IsStorable(results) {
			return false
		}
	}

	return true
}

func (t *InterfaceType) IsExternallyReturnable(results map[*Member]bool) bool {

	if t.CompositeKind != common.CompositeKindStructure {
		return false
	}

	// If this interface type has a member which is not externally returnable,
	// then the interface type is not externally returnable.

	for pair := t.Members.Oldest(); pair != nil; pair = pair.Next() {
		if !pair.Value.IsExternallyReturnable(results) {
			return false
		}
	}

	return true
}

func (t *InterfaceType) IsImportable(results map[*Member]bool) bool {
	if t.CompositeKind != common.CompositeKindStructure {
		return false
	}

	// If this interface type has a member which is not importable,
	// then the interface type is not importable.

	for pair := t.Members.Oldest(); pair != nil; pair = pair.Next() {
		if !pair.Value.IsImportable(results) {
			return false
		}
	}

	return true
}

func (*InterfaceType) IsEquatable() bool {
	// TODO:
	return false
}

func (*InterfaceType) TypeAnnotationState() TypeAnnotationState {
	return TypeAnnotationStateValid
}

func (t *InterfaceType) RewriteWithRestrictedTypes() (Type, bool) {
	switch t.CompositeKind {
	case common.CompositeKindResource:
		return &RestrictedType{
			Type:         AnyResourceType,
			Restrictions: []*InterfaceType{t},
		}, true

	case common.CompositeKindStructure:
		return &RestrictedType{
			Type:         AnyStructType,
			Restrictions: []*InterfaceType{t},
		}, true

	default:
		return t, false
	}
}

func (*InterfaceType) Unify(_ Type, _ *TypeParameterTypeOrderedMap, _ func(err error), _ ast.Range) bool {
	// TODO:
	return false
}

func (t *InterfaceType) Resolve(_ *TypeParameterTypeOrderedMap) Type {
	return t
}

func (t *InterfaceType) IsContainerType() bool {
	return t.nestedTypes != nil
}

func (t *InterfaceType) GetNestedTypes() *StringTypeOrderedMap {
	return t.nestedTypes
}

func (t *InterfaceType) FieldPosition(name string, declaration *ast.InterfaceDeclaration) ast.Position {
	return declaration.Members.FieldPosition(name, declaration.CompositeKind)
}

// DictionaryType consists of the key and value type
// for all key-value pairs in the dictionary:
// All keys have to be a subtype of the key type,
// and all values have to be a subtype of the value type.

type DictionaryType struct {
	KeyType             Type
	ValueType           Type
	memberResolvers     map[string]MemberResolver
	memberResolversOnce sync.Once
}

func NewDictionaryType(memoryGauge common.MemoryGauge, keyType, valueType Type) *DictionaryType {
	common.UseMemory(memoryGauge, common.DictionarySemaTypeMemoryUsage)
	return &DictionaryType{
		KeyType:   keyType,
		ValueType: valueType,
	}
}

func (*DictionaryType) IsType() {}

func (t *DictionaryType) Tag() TypeTag {
	return DictionaryTypeTag
}

func (t *DictionaryType) String() string {
	return fmt.Sprintf(
		"{%s: %s}",
		t.KeyType,
		t.ValueType,
	)
}

func (t *DictionaryType) QualifiedString() string {
	return fmt.Sprintf(
		"{%s: %s}",
		t.KeyType.QualifiedString(),
		t.ValueType.QualifiedString(),
	)
}

func (t *DictionaryType) ID() TypeID {
	return TypeID(fmt.Sprintf(
		"{%s:%s}",
		t.KeyType.ID(),
		t.ValueType.ID(),
	))
}

func (t *DictionaryType) Equal(other Type) bool {
	otherDictionary, ok := other.(*DictionaryType)
	if !ok {
		return false
	}

	return otherDictionary.KeyType.Equal(t.KeyType) &&
		otherDictionary.ValueType.Equal(t.ValueType)
}

func (t *DictionaryType) IsResourceType() bool {
	return t.KeyType.IsResourceType() ||
		t.ValueType.IsResourceType()
}

func (t *DictionaryType) IsInvalidType() bool {
	return t.KeyType.IsInvalidType() ||
		t.ValueType.IsInvalidType()
}

func (t *DictionaryType) IsStorable(results map[*Member]bool) bool {
	return t.KeyType.IsStorable(results) &&
		t.ValueType.IsStorable(results)
}

func (t *DictionaryType) IsExternallyReturnable(results map[*Member]bool) bool {
	return t.KeyType.IsExternallyReturnable(results) &&
		t.ValueType.IsExternallyReturnable(results)
}

func (t *DictionaryType) IsImportable(results map[*Member]bool) bool {
	return t.KeyType.IsImportable(results) &&
		t.ValueType.IsImportable(results)
}

func (*DictionaryType) IsEquatable() bool {
	// TODO:
	return false
}

func (t *DictionaryType) TypeAnnotationState() TypeAnnotationState {
	keyTypeAnnotationState := t.KeyType.TypeAnnotationState()
	if keyTypeAnnotationState != TypeAnnotationStateValid {
		return keyTypeAnnotationState
	}

	valueTypeAnnotationState := t.ValueType.TypeAnnotationState()
	if valueTypeAnnotationState != TypeAnnotationStateValid {
		return valueTypeAnnotationState
	}

	return TypeAnnotationStateValid
}

func (t *DictionaryType) RewriteWithRestrictedTypes() (Type, bool) {
	rewrittenKeyType, keyTypeRewritten := t.KeyType.RewriteWithRestrictedTypes()
	rewrittenValueType, valueTypeRewritten := t.ValueType.RewriteWithRestrictedTypes()
	rewritten := keyTypeRewritten || valueTypeRewritten
	if rewritten {
		return &DictionaryType{
			KeyType:   rewrittenKeyType,
			ValueType: rewrittenValueType,
		}, true
	} else {
		return t, false
	}
}

const dictionaryTypeContainsKeyFunctionDocString = `
Returns true if the given key is in the dictionary
`

const dictionaryTypeLengthFieldDocString = `
The number of entries in the dictionary
`

const dictionaryTypeKeysFieldDocString = `
An array containing all keys of the dictionary
`

const dictionaryTypeValuesFieldDocString = `
An array containing all values of the dictionary
`

const dictionaryTypeInsertFunctionDocString = `
Inserts the given value into the dictionary under the given key.

Returns the previous value as an optional if the dictionary contained the key, or nil if the dictionary did not contain the key
`

const dictionaryTypeRemoveFunctionDocString = `
Removes the value for the given key from the dictionary.

Returns the value as an optional if the dictionary contained the key, or nil if the dictionary did not contain the key
`

func (t *DictionaryType) GetMembers() map[string]MemberResolver {
	t.initializeMemberResolvers()
	return t.memberResolvers
}

func (t *DictionaryType) initializeMemberResolvers() {
	t.memberResolversOnce.Do(func() {

		t.memberResolvers = withBuiltinMembers(t, map[string]MemberResolver{
			"containsKey": {
				Kind: common.DeclarationKindFunction,
				Resolve: func(memoryGauge common.MemoryGauge, identifier string, targetRange ast.Range, report func(error)) *Member {

					return NewPublicFunctionMember(
						memoryGauge,
						t,
						identifier,
						DictionaryContainsKeyFunctionType(t),
						dictionaryTypeContainsKeyFunctionDocString,
					)
				},
			},
			"length": {
				Kind: common.DeclarationKindField,
				Resolve: func(memoryGauge common.MemoryGauge, identifier string, _ ast.Range, _ func(error)) *Member {
					return NewPublicConstantFieldMember(
						memoryGauge,
						t,
						identifier,
						IntType,
						dictionaryTypeLengthFieldDocString,
					)
				},
			},
			"keys": {
				Kind: common.DeclarationKindField,
				Resolve: func(memoryGauge common.MemoryGauge, identifier string, targetRange ast.Range, report func(error)) *Member {
					// TODO: maybe allow for resource key type

					if t.KeyType.IsResourceType() {
						report(
							&InvalidResourceDictionaryMemberError{
								Name:            identifier,
								DeclarationKind: common.DeclarationKindField,
								Range:           targetRange,
							},
						)
					}

					return NewPublicConstantFieldMember(
						memoryGauge,
						t,
						identifier,
						&VariableSizedType{Type: t.KeyType},
						dictionaryTypeKeysFieldDocString,
					)
				},
			},
			"values": {
				Kind: common.DeclarationKindField,
				Resolve: func(memoryGauge common.MemoryGauge, identifier string, targetRange ast.Range, report func(error)) *Member {
					// TODO: maybe allow for resource value type

					if t.ValueType.IsResourceType() {
						report(
							&InvalidResourceDictionaryMemberError{
								Name:            identifier,
								DeclarationKind: common.DeclarationKindField,
								Range:           targetRange,
							},
						)
					}

					return NewPublicConstantFieldMember(
						memoryGauge,
						t,
						identifier,
						&VariableSizedType{Type: t.ValueType},
						dictionaryTypeValuesFieldDocString,
					)
				},
			},
			"insert": {
				Kind:     common.DeclarationKindFunction,
				Mutating: true,
				Resolve: func(memoryGauge common.MemoryGauge, identifier string, _ ast.Range, _ func(error)) *Member {
					return NewPublicFunctionMember(
						memoryGauge,
						t,
						identifier,
						DictionaryInsertFunctionType(t),
						dictionaryTypeInsertFunctionDocString,
					)
				},
			},
			"remove": {
				Kind:     common.DeclarationKindFunction,
				Mutating: true,
				Resolve: func(memoryGauge common.MemoryGauge, identifier string, _ ast.Range, _ func(error)) *Member {
					return NewPublicFunctionMember(
						memoryGauge,
						t,
						identifier,
						DictionaryRemoveFunctionType(t),
						dictionaryTypeRemoveFunctionDocString,
					)
				},
			},
		})
	})
}

func DictionaryContainsKeyFunctionType(t *DictionaryType) *FunctionType {
	return &FunctionType{
		Parameters: []*Parameter{
			{
				Label:          ArgumentLabelNotRequired,
				Identifier:     "key",
				TypeAnnotation: NewTypeAnnotation(t.KeyType),
			},
		},
		ReturnTypeAnnotation: NewTypeAnnotation(
			BoolType,
		),
	}
}

func DictionaryInsertFunctionType(t *DictionaryType) *FunctionType {
	return &FunctionType{
		Parameters: []*Parameter{
			{
				Identifier:     "key",
				TypeAnnotation: NewTypeAnnotation(t.KeyType),
			},
			{
				Label:          ArgumentLabelNotRequired,
				Identifier:     "value",
				TypeAnnotation: NewTypeAnnotation(t.ValueType),
			},
		},
		ReturnTypeAnnotation: NewTypeAnnotation(
			&OptionalType{
				Type: t.ValueType,
			},
		),
	}
}

func DictionaryRemoveFunctionType(t *DictionaryType) *FunctionType {
	return &FunctionType{
		Parameters: []*Parameter{
			{
				Identifier:     "key",
				TypeAnnotation: NewTypeAnnotation(t.KeyType),
			},
		},
		ReturnTypeAnnotation: NewTypeAnnotation(
			&OptionalType{
				Type: t.ValueType,
			},
		),
	}
}

func (*DictionaryType) isValueIndexableType() bool {
	return true
}

func (t *DictionaryType) ElementType(_ bool) Type {
	return &OptionalType{Type: t.ValueType}
}

func (*DictionaryType) AllowsValueIndexingAssignment() bool {
	return true
}

func (t *DictionaryType) IndexingType() Type {
	return t.KeyType
}

type DictionaryEntryType struct {
	KeyType   Type
	ValueType Type
}

func (t *DictionaryType) Unify(
	other Type,
	typeParameters *TypeParameterTypeOrderedMap,
	report func(err error),
	outerRange ast.Range,
) bool {

	otherDictionary, ok := other.(*DictionaryType)
	if !ok {
		return false
	}

	keyUnified := t.KeyType.Unify(otherDictionary.KeyType, typeParameters, report, outerRange)
	valueUnified := t.ValueType.Unify(otherDictionary.ValueType, typeParameters, report, outerRange)
	return keyUnified || valueUnified
}

func (t *DictionaryType) Resolve(typeArguments *TypeParameterTypeOrderedMap) Type {
	newKeyType := t.KeyType.Resolve(typeArguments)
	if newKeyType == nil {
		return nil
	}

	newValueType := t.ValueType.Resolve(typeArguments)
	if newValueType == nil {
		return nil
	}

	return &DictionaryType{
		KeyType:   newKeyType,
		ValueType: newValueType,
	}
}

// ReferenceType represents the reference to a value
type ReferenceType struct {
	Authorized bool
	Type       Type
}

func NewReferenceType(memoryGauge common.MemoryGauge, typ Type, authorized bool) *ReferenceType {
	common.UseMemory(memoryGauge, common.ReferenceSemaTypeMemoryUsage)
	return &ReferenceType{
		Type:       typ,
		Authorized: authorized,
	}
}

func (*ReferenceType) IsType() {}

func (t *ReferenceType) Tag() TypeTag {
	return ReferenceTypeTag
}

func (t *ReferenceType) string(typeFormatter func(Type) string) string {
	if t.Type == nil {
		return "reference"
	}
	var builder strings.Builder
	if t.Authorized {
		builder.WriteString("auth ")
	}
	builder.WriteRune('&')
	builder.WriteString(typeFormatter(t.Type))
	return builder.String()
}

func (t *ReferenceType) String() string {
	return t.string(func(ty Type) string {
		return ty.String()
	})
}

func (t *ReferenceType) QualifiedString() string {
	return t.string(func(ty Type) string {
		return ty.QualifiedString()
	})
}

func (t *ReferenceType) ID() TypeID {
	return TypeID(
		t.string(func(ty Type) string {
			return string(ty.ID())
		}),
	)
}

func (t *ReferenceType) Equal(other Type) bool {
	otherReference, ok := other.(*ReferenceType)
	if !ok {
		return false
	}

	if t.Authorized != otherReference.Authorized {
		return false
	}

	return t.Type.Equal(otherReference.Type)
}

func (t *ReferenceType) IsResourceType() bool {
	return false
}

func (t *ReferenceType) IsInvalidType() bool {
	return t.Type.IsInvalidType()
}

func (t *ReferenceType) IsStorable(_ map[*Member]bool) bool {
	return false
}

func (t *ReferenceType) IsExternallyReturnable(_ map[*Member]bool) bool {
	return true
}

func (t *ReferenceType) IsImportable(_ map[*Member]bool) bool {
	return false
}

func (*ReferenceType) IsEquatable() bool {
	return true
}

func (*ReferenceType) TypeAnnotationState() TypeAnnotationState {
	return TypeAnnotationStateValid
}

func (t *ReferenceType) RewriteWithRestrictedTypes() (Type, bool) {
	rewrittenType, rewritten := t.Type.RewriteWithRestrictedTypes()
	if rewritten {
		return &ReferenceType{
			Authorized: t.Authorized,
			Type:       rewrittenType,
		}, true
	} else {
		return t, false
	}
}

func (t *ReferenceType) GetMembers() map[string]MemberResolver {
	return t.Type.GetMembers()
}

func (t *ReferenceType) isValueIndexableType() bool {
	referencedType, ok := t.Type.(ValueIndexableType)
	if !ok {
		return false
	}
	return referencedType.isValueIndexableType()
}

func (t *ReferenceType) AllowsValueIndexingAssignment() bool {
	referencedType, ok := t.Type.(ValueIndexableType)
	if !ok {
		return false
	}
	return referencedType.AllowsValueIndexingAssignment()
}

func (t *ReferenceType) ElementType(isAssignment bool) Type {
	referencedType, ok := t.Type.(ValueIndexableType)
	if !ok {
		return nil
	}
	return referencedType.ElementType(isAssignment)
}

func (t *ReferenceType) IndexingType() Type {
	referencedType, ok := t.Type.(ValueIndexableType)
	if !ok {
		return nil
	}
	return referencedType.IndexingType()
}

func (*ReferenceType) Unify(_ Type, _ *TypeParameterTypeOrderedMap, _ func(err error), _ ast.Range) bool {
	// TODO:
	return false
}

func (t *ReferenceType) Resolve(_ *TypeParameterTypeOrderedMap) Type {
	// TODO:
	return t
}

const AddressTypeName = "Address"

// AddressType represents the address type
type AddressType struct{}

var _ IntegerRangedType = &AddressType{}

func (*AddressType) IsType() {}

func (t *AddressType) Tag() TypeTag {
	return AddressTypeTag
}

func (*AddressType) String() string {
	return AddressTypeName
}

func (*AddressType) QualifiedString() string {
	return AddressTypeName
}

func (*AddressType) ID() TypeID {
	return AddressTypeName
}

func (*AddressType) Equal(other Type) bool {
	_, ok := other.(*AddressType)
	return ok
}

func (*AddressType) IsResourceType() bool {
	return false
}

func (*AddressType) IsInvalidType() bool {
	return false
}

func (*AddressType) IsStorable(_ map[*Member]bool) bool {
	return true
}

func (*AddressType) IsExternallyReturnable(_ map[*Member]bool) bool {
	return true
}

func (t *AddressType) IsImportable(_ map[*Member]bool) bool {
	return true
}

func (*AddressType) IsEquatable() bool {
	return true
}

func (*AddressType) TypeAnnotationState() TypeAnnotationState {
	return TypeAnnotationStateValid
}

func (t *AddressType) RewriteWithRestrictedTypes() (Type, bool) {
	return t, false
}

var AddressTypeMinIntBig = new(big.Int)
var AddressTypeMaxIntBig = new(big.Int).SetUint64(math.MaxUint64)

func (*AddressType) MinInt() *big.Int {
	return AddressTypeMinIntBig
}

func (*AddressType) MaxInt() *big.Int {
	return AddressTypeMaxIntBig
}

func (*AddressType) IsSuperType() bool {
	return false
}

func (*AddressType) Unify(_ Type, _ *TypeParameterTypeOrderedMap, _ func(err error), _ ast.Range) bool {
	return false
}

func (t *AddressType) Resolve(_ *TypeParameterTypeOrderedMap) Type {
	return t
}

const AddressTypeToBytesFunctionName = `toBytes`

var AddressTypeToBytesFunctionType = &FunctionType{
	ReturnTypeAnnotation: NewTypeAnnotation(
		ByteArrayType,
	),
}

const addressTypeToBytesFunctionDocString = `
Returns an array containing the byte representation of the address
`

func (t *AddressType) GetMembers() map[string]MemberResolver {
	return withBuiltinMembers(t, map[string]MemberResolver{
		AddressTypeToBytesFunctionName: {
			Resolve: func(memoryGauge common.MemoryGauge, identifier string, _ ast.Range, _ func(error)) *Member {
				return NewPublicFunctionMember(
					memoryGauge,
					t,
					identifier,
					AddressTypeToBytesFunctionType,
					addressTypeToBytesFunctionDocString,
				)
			},
		},
	})
}

// IsSubType determines if the given subtype is a subtype
// of the given supertype.
//
// Types are subtypes of themselves.
//
// NOTE: This method can be used to check the assignability of `subType` to `superType`.
// However, to check if a type *strictly* belongs to a certain category, then consider
// using `IsSameTypeKind` method. e.g: "Is type `T` an Integer type?". Using this method
// for the later use-case may produce incorrect results.
//   * IsSubType()      - To check the assignability. e.g: Is argument type T is a sub-type
//                        of parameter type R. This is the more frequent use-case.
//   * IsSameTypeKind() - To check if a type strictly belongs to a certain category. e.g: Is the
//                        expression type T is any of the integer types, but nothing else.
//                        Another way to check is, asking the question of "if the subType is Never,
//                        should the check still pass?". A common code-smell for potential incorrect
//                        usage is, using IsSubType() method with a constant/pre-defined superType.
//                        e.g: IsSubType(<<someType>>, FixedPointType)
//
func IsSubType(subType Type, superType Type) bool {

	if subType == nil {
		return false
	}

	if subType.Equal(superType) {
		return true
	}

	return checkSubTypeWithoutEquality(subType, superType)
}

// IsSameTypeKind determines if the given subtype belongs to the
// same kind as the supertype.
//
// e.g: 'Never' type is a subtype of 'Integer', but not of the
// same kind as 'Integer'. Whereas, 'Int8' is both a subtype
// and also of same kind as 'Integer'.
//
func IsSameTypeKind(subType Type, superType Type) bool {

	if subType == NeverType {
		return false
	}

	return IsSubType(subType, superType)
}

// IsProperSubType is similar to IsSubType,
// i.e. it determines if the given subtype is a subtype
// of the given supertype, but returns false
// if the subtype and supertype refer to the same type.
//
func IsProperSubType(subType Type, superType Type) bool {

	if subType.Equal(superType) {
		return false
	}

	return checkSubTypeWithoutEquality(subType, superType)
}

// checkSubTypeWithoutEquality determines if the given subtype
// is a subtype of the given supertype, BUT it does NOT check
// the equality of the two types, so does NOT return a specific
// value when the two types are equal or are not.
//
// Consider using IsSubType or IsProperSubType
//
func checkSubTypeWithoutEquality(subType Type, superType Type) bool {

	if subType == NeverType {
		return true
	}

	switch superType {
	case AnyType:
		return true

	case AnyStructType:
		if subType.IsResourceType() {
			return false
		}
		return subType != AnyType

	case AnyResourceType:
		return subType.IsResourceType()

	case NumberType:
		switch subType {
		case NumberType, SignedNumberType:
			return true
		}

		return IsSubType(subType, IntegerType) ||
			IsSubType(subType, FixedPointType)

	case SignedNumberType:
		if subType == SignedNumberType {
			return true
		}

		return IsSubType(subType, SignedIntegerType) ||
			IsSubType(subType, SignedFixedPointType)

	case IntegerType:
		switch subType {
		case IntegerType, SignedIntegerType,
			UIntType,
			UInt8Type, UInt16Type, UInt32Type, UInt64Type, UInt128Type, UInt256Type,
			Word8Type, Word16Type, Word32Type, Word64Type:

			return true

		default:
			return IsSubType(subType, SignedIntegerType)
		}

	case SignedIntegerType:
		switch subType {
		case SignedIntegerType,
			IntType,
			Int8Type, Int16Type, Int32Type, Int64Type, Int128Type, Int256Type:

			return true

		default:
			return false
		}

	case FixedPointType:
		switch subType {
		case FixedPointType, SignedFixedPointType,
			UFix64Type:

			return true

		default:
			return IsSubType(subType, SignedFixedPointType)
		}

	case SignedFixedPointType:
		switch subType {
		case SignedFixedPointType, Fix64Type:
			return true

		default:
			return false
		}
	}

	switch typedSuperType := superType.(type) {
	case *OptionalType:
		optionalSubType, ok := subType.(*OptionalType)
		if !ok {
			// T <: U? if T <: U
			return IsSubType(subType, typedSuperType.Type)
		}
		// Optionals are covariant: T? <: U? if T <: U
		return IsSubType(optionalSubType.Type, typedSuperType.Type)

	case *DictionaryType:
		typedSubType, ok := subType.(*DictionaryType)
		if !ok {
			return false
		}

		return IsSubType(typedSubType.KeyType, typedSuperType.KeyType) &&
			IsSubType(typedSubType.ValueType, typedSuperType.ValueType)

	case *VariableSizedType:
		typedSubType, ok := subType.(*VariableSizedType)
		if !ok {
			return false
		}

		return IsSubType(
			typedSubType.ElementType(false),
			typedSuperType.ElementType(false),
		)

	case *ConstantSizedType:
		typedSubType, ok := subType.(*ConstantSizedType)
		if !ok {
			return false
		}

		if typedSubType.Size != typedSuperType.Size {
			return false
		}

		return IsSubType(
			typedSubType.ElementType(false),
			typedSuperType.ElementType(false),
		)

	case *ReferenceType:
		// References types are only subtypes of reference types

		typedSubType, ok := subType.(*ReferenceType)
		if !ok {
			return false
		}

		// An authorized reference type `auth &T`
		// is a subtype of a reference type `&U` (authorized or non-authorized),
		// if `T` is a subtype of `U`

		if typedSubType.Authorized {
			return IsSubType(typedSubType.Type, typedSuperType.Type)
		}

		// An unauthorized reference type is not a subtype of an authorized reference type.
		// Not even dynamically.
		//
		// The holder of the reference may not gain more permissions.

		if typedSuperType.Authorized {
			return false
		}

		switch typedInnerSuperType := typedSuperType.Type.(type) {
		case *RestrictedType:

			restrictedSuperType := typedInnerSuperType.Type
			switch restrictedSuperType {
			case AnyResourceType, AnyStructType, AnyType:

				switch typedInnerSubType := typedSubType.Type.(type) {
				case *RestrictedType:
					// An unauthorized reference to a restricted type `&T{Us}`
					// is a subtype of a reference to a restricted type
					// `&AnyResource{Vs}` / `&AnyStruct{Vs}` / `&Any{Vs}`:
					// if the `T` is a subset of the supertype's restricted type,
					// and `Vs` is a subset of `Us`.
					//
					// The holder of the reference may only further restrict the reference.
					//
					// The requirement for `T` to conform to `Vs` is implied by the subset requirement.

					return IsSubType(typedInnerSubType.Type, restrictedSuperType) &&
						typedInnerSuperType.RestrictionSet().
							IsSubsetOf(typedInnerSubType.RestrictionSet())

				case *CompositeType:
					// An unauthorized reference to an unrestricted type `&T`
					// is a subtype of a reference to a restricted type
					// `&AnyResource{Us}` / `&AnyStruct{Us}` / `&Any{Us}`:
					// When `T != AnyResource && T != AnyStruct && T != Any`:
					// if `T` conforms to `Us`.
					//
					// The holder of the reference may only restrict the reference.

					// TODO: once interfaces can conform to interfaces, include
					return IsSubType(typedInnerSubType, restrictedSuperType) &&
						typedInnerSuperType.RestrictionSet().
							IsSubsetOf(typedInnerSubType.ExplicitInterfaceConformanceSet())
				}

				switch typedSubType.Type {
				case AnyResourceType, AnyStructType, AnyType:
					// An unauthorized reference to an unrestricted type `&T`
					// is a subtype of a reference to a restricted type
					// `&AnyResource{Us}` / `&AnyStruct{Us}` / `&Any{Us}`:
					// When `T == AnyResource || T == AnyStruct || T == Any`: never.
					//
					// The holder of the reference may not gain more permissions or knowledge.

					return false
				}

			default:

				switch typedInnerSubType := typedSubType.Type.(type) {
				case *RestrictedType:

					// An unauthorized reference to a restricted type `&T{Us}`
					// is a subtype of a reference to a restricted type `&V{Ws}:`

					if _, ok := typedInnerSubType.Type.(*CompositeType); ok {
						// When `T != AnyResource && T != AnyStruct && T != Any`:
						// if `T == V` and `Ws` is a subset of `Us`.
						//
						// The holder of the reference may not gain more permissions or knowledge
						// and may only further restrict the reference to the composite.

						return typedInnerSubType.Type == typedInnerSuperType.Type &&
							typedInnerSuperType.RestrictionSet().
								IsSubsetOf(typedInnerSubType.RestrictionSet())
					}

					switch typedInnerSubType.Type {
					case AnyResourceType, AnyStructType, AnyType:
						// When `T == AnyResource || T == AnyStruct || T == Any`: never.

						return false
					}

				case *CompositeType:
					// An unauthorized reference to an unrestricted type `&T`
					// is a subtype of a reference to a restricted type `&U{Vs}`:
					// When `T != AnyResource && T != AnyStruct && T != Any`: if `T == U`.
					//
					// The holder of the reference may only further restrict the reference.

					return typedInnerSubType == typedInnerSuperType.Type

				}

				switch typedSubType.Type {
				case AnyResourceType, AnyStructType, AnyType:
					// An unauthorized reference to an unrestricted type `&T`
					// is a subtype of a reference to a restricted type `&U{Vs}`:
					// When `T == AnyResource || T == AnyStruct || T == Any`: never.
					//
					// The holder of the reference may not gain more permissions or knowledge.

					return false
				}
			}

		case *CompositeType:

			// The supertype composite type might be a type requirement.
			// Check if the subtype composite type implicitly conforms to it.

			if typedInnerSubType, ok := typedSubType.Type.(*CompositeType); ok {

				for _, conformance := range typedInnerSubType.ImplicitTypeRequirementConformances {
					if conformance == typedInnerSuperType {
						return true
					}
				}
			}

			// An unauthorized reference is not a subtype of a reference to a composite type `&V`
			// (e.g. reference to a restricted type `&T{Us}`, or reference to an interface type `&T`)
			//
			// The holder of the reference may not gain more permissions or knowledge.

			return false
		}

		switch typedSuperType.Type {

		case AnyType:

			// An unauthorized reference to a restricted type `&T{Us}`
			// or to a unrestricted type `&T`
			// is a subtype of the type `&Any`: always.

			return true

		case AnyResourceType:

			// An unauthorized reference to a restricted type `&T{Us}`
			// or to a unrestricted type `&T`
			// is a subtype of the type `&AnyResource`:
			// if `T == AnyResource` or `T` is a resource-kinded composite.

			switch typedInnerSubType := typedSubType.Type.(type) {
			case *RestrictedType:
				if typedInnerInnerSubType, ok := typedInnerSubType.Type.(*CompositeType); ok {
					return typedInnerInnerSubType.Kind == common.CompositeKindResource
				}

				return typedInnerSubType.Type == AnyResourceType

			case *CompositeType:
				return typedInnerSubType.Kind == common.CompositeKindResource
			}

		case AnyStructType:
			// `&T <: &AnyStruct` iff `T <: AnyStruct`
			return IsSubType(typedSubType.Type, typedSuperType.Type)
		}

	case *FunctionType:
		typedSubType, ok := subType.(*FunctionType)
		if !ok {
			return false
		}

		if len(typedSubType.Parameters) != len(typedSuperType.Parameters) {
			return false
		}

		// Functions are contravariant in their parameter types

		for i, subParameter := range typedSubType.Parameters {
			superParameter := typedSuperType.Parameters[i]
			if !IsSubType(
				superParameter.TypeAnnotation.Type,
				subParameter.TypeAnnotation.Type,
			) {
				return false
			}
		}

		// Functions are covariant in their return type

		if typedSubType.ReturnTypeAnnotation != nil {
			if typedSuperType.ReturnTypeAnnotation == nil {
				return false
			}

			if !IsSubType(
				typedSubType.ReturnTypeAnnotation.Type,
				typedSuperType.ReturnTypeAnnotation.Type,
			) {
				return false
			}
		} else if typedSuperType.ReturnTypeAnnotation != nil {
			return false
		}

		// Receiver type wouldn't matter for sub-typing.
		// i.e: In a bound function pointer `x.foo`, `x` is a closure,
		// and is not part of the function pointer's inputs/outputs.

		// Constructors?

		if typedSubType.IsConstructor != typedSuperType.IsConstructor {
			return false
		}

		return true

	case *RestrictedType:

		restrictedSuperType := typedSuperType.Type
		switch restrictedSuperType {
		case AnyResourceType, AnyStructType, AnyType:

			switch subType {
			case AnyResourceType:
				// `AnyResource` is a subtype of a restricted type
				// - `AnyResource{Us}`: not statically;
				// - `AnyStruct{Us}`: never.
				// - `Any{Us}`: not statically;

				return false

			case AnyStructType:
				// `AnyStruct` is a subtype of a restricted type
				// - `AnyStruct{Us}`: not statically.
				// - `AnyResource{Us}`: never;
				// - `Any{Us}`: not statically.

				return false

			case AnyType:
				// `Any` is a subtype of a restricted type
				// - `Any{Us}: not statically.`
				// - `AnyStruct{Us}`: never;
				// - `AnyResource{Us}`: never;

				return false
			}

			switch typedSubType := subType.(type) {
			case *RestrictedType:

				// A restricted type `T{Us}`
				// is a subtype of a restricted type `AnyResource{Vs}` / `AnyStruct{Vs}` / `Any{Vs}`:

				restrictedSubtype := typedSubType.Type
				switch restrictedSubtype {
				case AnyResourceType, AnyStructType, AnyType:
					// When `T == AnyResource || T == AnyStruct || T == Any`:
					// if the restricted type of the subtype
					// is a subtype of the restricted supertype,
					// and `Vs` is a subset of `Us`.

					return IsSubType(restrictedSubtype, restrictedSuperType) &&
						typedSuperType.RestrictionSet().
							IsSubsetOf(typedSubType.RestrictionSet())
				}

				if restrictedSubtype, ok := restrictedSubtype.(*CompositeType); ok {
					// When `T != AnyResource && T != AnyStruct && T != Any`:
					// if the restricted type of the subtype
					// is a subtype of the restricted supertype,
					// and `T` conforms to `Vs`.
					// `Us` and `Vs` do *not* have to be subsets.

					// TODO: once interfaces can conform to interfaces, include
					return IsSubType(restrictedSubtype, restrictedSuperType) &&
						typedSuperType.RestrictionSet().
							IsSubsetOf(restrictedSubtype.ExplicitInterfaceConformanceSet())
				}

			case *CompositeType:
				// An unrestricted type `T`
				// is a subtype of a restricted type `AnyResource{Us}` / `AnyStruct{Us}` / `Any{Us}`:
				// if `T` is a subtype of the restricted supertype,
				// and `T` conforms to `Us`.

				return IsSubType(typedSubType, typedSuperType.Type) &&
					typedSuperType.RestrictionSet().
						IsSubsetOf(typedSubType.ExplicitInterfaceConformanceSet())
			}

		default:

			switch typedSubType := subType.(type) {
			case *RestrictedType:

				// A restricted type `T{Us}`
				// is a subtype of a restricted type `V{Ws}`:

				switch typedSubType.Type {
				case AnyResourceType, AnyStructType, AnyType:
					// When `T == AnyResource || T == AnyStruct || T == Any`:
					// not statically.
					return false
				}

				if restrictedSubType, ok := typedSubType.Type.(*CompositeType); ok {
					// When `T != AnyResource && T != AnyStructType && T != Any`: if `T == V`.
					//
					// `Us` and `Ws` do *not* have to be subsets:
					// The owner may freely restrict and unrestrict.

					return restrictedSubType == typedSuperType.Type
				}

			case *CompositeType:
				// An unrestricted type `T`
				// is a subtype of a restricted type `U{Vs}`: if `T <: U`.
				//
				// The owner may freely restrict.

				return IsSubType(typedSubType, typedSuperType.Type)
			}

			switch subType {
			case AnyResourceType, AnyStructType, AnyType:
				// An unrestricted type `T`
				// is a subtype of a restricted type `AnyResource{Vs}` / `AnyStruct{Vs}` / `Any{Vs}`:
				// not statically.

				return false
			}
		}

	case *CompositeType:

		// NOTE: type equality case (composite type `T` is subtype of composite type `U`)
		// is already handled at beginning of function

		switch typedSubType := subType.(type) {
		case *RestrictedType:

			// A restricted type `T{Us}`
			// is a subtype of an unrestricted type `V`:

			switch typedSubType.Type {
			case AnyResourceType, AnyStructType, AnyType:
				// When `T == AnyResource || T == AnyStruct || T == Any`: not statically.
				return false
			}

			if restrictedSubType, ok := typedSubType.Type.(*CompositeType); ok {
				// When `T != AnyResource && T != AnyStruct`: if `T == V`.
				//
				// The owner may freely unrestrict.

				return restrictedSubType == typedSuperType
			}

		case *CompositeType:
			// The supertype composite type might be a type requirement.
			// Check if the subtype composite type implicitly conforms to it.

			for _, conformance := range typedSubType.ImplicitTypeRequirementConformances {
				if conformance == typedSuperType {
					return true
				}
			}
		}

	case *InterfaceType:

		switch typedSubType := subType.(type) {
		case *CompositeType:

			// A composite type `T` is a subtype of a interface type `V`:
			// if `T` conforms to `V`, and `V` and `T` are of the same kind

			if typedSubType.Kind != typedSuperType.CompositeKind {
				return false
			}

			// TODO: once interfaces can conform to interfaces, include
			return typedSubType.ExplicitInterfaceConformanceSet().
				Includes(typedSuperType)

		case *InterfaceType:
			// TODO: Once interfaces can conform to interfaces, check conformances here
			return false
		}

	case ParameterizedType:
		if superTypeBaseType := typedSuperType.BaseType(); superTypeBaseType != nil {

			// T<Us> <: V<Ws>
			// if T <: V  && |Us| == |Ws| && U_i <: W_i

			if typedSubType, ok := subType.(ParameterizedType); ok {
				if subTypeBaseType := typedSubType.BaseType(); subTypeBaseType != nil {

					if !IsSubType(subTypeBaseType, superTypeBaseType) {
						return false
					}

					subTypeTypeArguments := typedSubType.TypeArguments()
					superTypeTypeArguments := typedSuperType.TypeArguments()

					if len(subTypeTypeArguments) != len(superTypeTypeArguments) {
						return false
					}

					for i, superTypeTypeArgument := range superTypeTypeArguments {
						subTypeTypeArgument := subTypeTypeArguments[i]
						if !IsSubType(subTypeTypeArgument, superTypeTypeArgument) {
							return false
						}
					}

					return true
				}
			}
		}

	case *SimpleType:
		if typedSuperType.IsSuperTypeOf == nil {
			return false
		}
		return typedSuperType.IsSuperTypeOf(subType)
	}

	// TODO: enforce type arguments, remove this rule

	// T<Us> <: V
	// if T <: V

	if typedSubType, ok := subType.(ParameterizedType); ok {
		if baseType := typedSubType.BaseType(); baseType != nil {
			return IsSubType(baseType, superType)
		}
	}

	return false
}

// UnwrapOptionalType returns the type if it is not an optional type,
// or the inner-most type if it is (optional types are repeatedly unwrapped)
//
func UnwrapOptionalType(ty Type) Type {
	for {
		optionalType, ok := ty.(*OptionalType)
		if !ok {
			return ty
		}
		ty = optionalType.Type
	}
}

func AreCompatibleEquatableTypes(leftType, rightType Type) bool {
	unwrappedLeftType := UnwrapOptionalType(leftType)
	unwrappedRightType := UnwrapOptionalType(rightType)

	leftIsEquatable := unwrappedLeftType.IsEquatable()
	rightIsEquatable := unwrappedRightType.IsEquatable()

	if unwrappedLeftType.Equal(unwrappedRightType) &&
		leftIsEquatable && rightIsEquatable {

		return true
	}

	// The types are equatable if this is a comparison with `nil`,
	// which has type `Never?`

	if IsNilType(leftType) || IsNilType(rightType) {
		return true
	}

	return false
}

// IsNilType returns true if the given type is the type of `nil`, i.e. `Never?`.
//
func IsNilType(ty Type) bool {
	optionalType, ok := ty.(*OptionalType)
	if !ok {
		return false
	}

	if optionalType.Type != NeverType {
		return false
	}

	return true
}

type TransactionType struct {
	Members           *StringMemberOrderedMap
	Fields            []string
	PrepareParameters []*Parameter
	Parameters        []*Parameter
}

func (t *TransactionType) EntryPointFunctionType() *FunctionType {
	return &FunctionType{
		Parameters:           append(t.Parameters, t.PrepareParameters...),
		ReturnTypeAnnotation: NewTypeAnnotation(VoidType),
	}
}

func (t *TransactionType) PrepareFunctionType() *FunctionType {
	return &FunctionType{
		IsConstructor:        true,
		Parameters:           t.PrepareParameters,
		ReturnTypeAnnotation: NewTypeAnnotation(VoidType),
	}
}

func (*TransactionType) ExecuteFunctionType() *FunctionType {
	return &FunctionType{
		IsConstructor:        true,
		Parameters:           []*Parameter{},
		ReturnTypeAnnotation: NewTypeAnnotation(VoidType),
	}
}

func (*TransactionType) IsType() {}

func (t *TransactionType) Tag() TypeTag {
	return TransactionTypeTag
}

func (*TransactionType) String() string {
	return "Transaction"
}

func (*TransactionType) QualifiedString() string {
	return "Transaction"
}

func (*TransactionType) ID() TypeID {
	return "Transaction"
}

func (*TransactionType) Equal(other Type) bool {
	_, ok := other.(*TransactionType)
	return ok
}

func (*TransactionType) IsResourceType() bool {
	return false
}

func (*TransactionType) IsInvalidType() bool {
	return false
}

func (*TransactionType) IsStorable(_ map[*Member]bool) bool {
	return false
}

func (*TransactionType) IsExternallyReturnable(_ map[*Member]bool) bool {
	return false
}

func (t *TransactionType) IsImportable(_ map[*Member]bool) bool {
	return false
}

func (*TransactionType) IsEquatable() bool {
	return false
}

func (*TransactionType) TypeAnnotationState() TypeAnnotationState {
	return TypeAnnotationStateValid
}

func (t *TransactionType) RewriteWithRestrictedTypes() (Type, bool) {
	return t, false
}

func (t *TransactionType) GetMembers() map[string]MemberResolver {
	// TODO: optimize
	var members map[string]MemberResolver
	if t.Members != nil {
		members = make(map[string]MemberResolver, t.Members.Len())
		t.Members.Foreach(func(name string, loopMember *Member) {
			// NOTE: don't capture loop variable
			member := loopMember
			members[name] = MemberResolver{
				Kind: member.DeclarationKind,
				Resolve: func(memoryGauge common.MemoryGauge, identifier string, _ ast.Range, _ func(error)) *Member {
					return member
				},
			}
		})
	}
	return withBuiltinMembers(t, members)
}

func (*TransactionType) Unify(_ Type, _ *TypeParameterTypeOrderedMap, _ func(err error), _ ast.Range) bool {
	return false
}

func (t *TransactionType) Resolve(_ *TypeParameterTypeOrderedMap) Type {
	return t
}

// RestrictedType
//
// No restrictions implies the type is fully restricted,
// i.e. no members of the underlying resource type are available.
//
type RestrictedType struct {
	Type         Type
	Restrictions []*InterfaceType
	// an internal set of field `Restrictions`
	restrictionSet     *InterfaceSet
	restrictionSetOnce sync.Once
}

func NewRestrictedType(memoryGauge common.MemoryGauge, typ Type, restrictions []*InterfaceType) *RestrictedType {
	common.UseMemory(memoryGauge, common.RestrictedSemaTypeMemoryUsage)

	// Also meter the cost for the `restrictionSet` here, since ordered maps are not separately metered.
	wrapperUsage, entryListUsage, entriesUsage := common.NewOrderedMapMemoryUsages(uint64(len(restrictions)))
	common.UseMemory(memoryGauge, wrapperUsage)
	common.UseMemory(memoryGauge, entryListUsage)
	common.UseMemory(memoryGauge, entriesUsage)

	return &RestrictedType{
		Type:         typ,
		Restrictions: restrictions,
	}
}

func (t *RestrictedType) RestrictionSet() *InterfaceSet {
	t.initializeRestrictionSet()
	return t.restrictionSet
}

func (t *RestrictedType) initializeRestrictionSet() {
	t.restrictionSetOnce.Do(func() {
		t.restrictionSet = NewInterfaceSet()
		for _, restriction := range t.Restrictions {
			t.restrictionSet.Add(restriction)
		}
	})
}

func (*RestrictedType) IsType() {}

func (t *RestrictedType) Tag() TypeTag {
	return RestrictedTypeTag
}

func (t *RestrictedType) string(separator string, typeFormatter func(Type) string) string {
	var result strings.Builder
	result.WriteString(typeFormatter(t.Type))
	result.WriteRune('{')
	for i, restriction := range t.Restrictions {
		if i > 0 {
			result.WriteRune(',')
			result.WriteString(separator)
		}
		result.WriteString(typeFormatter(restriction))
	}
	result.WriteRune('}')
	return result.String()
}

func (t *RestrictedType) String() string {
	return t.string(" ", func(ty Type) string {
		return ty.String()
	})
}

func (t *RestrictedType) QualifiedString() string {
	return t.string(" ", func(ty Type) string {
		return ty.QualifiedString()
	})
}

func (t *RestrictedType) ID() TypeID {
	return TypeID(
		t.string("", func(ty Type) string {
			return string(ty.ID())
		}),
	)
}

func (t *RestrictedType) Equal(other Type) bool {
	otherRestrictedType, ok := other.(*RestrictedType)
	if !ok {
		return false
	}

	if !otherRestrictedType.Type.Equal(t.Type) {
		return false
	}

	// Check that the set of restrictions are equal; order does not matter

	restrictionSet := t.RestrictionSet()
	otherRestrictionSet := otherRestrictedType.RestrictionSet()

	if restrictionSet.Len() != otherRestrictionSet.Len() {
		return false
	}

	return restrictionSet.IsSubsetOf(otherRestrictionSet)
}

func (t *RestrictedType) IsResourceType() bool {
	if t.Type == nil {
		return false
	}
	return t.Type.IsResourceType()
}

func (t *RestrictedType) IsInvalidType() bool {
	if t.Type != nil && t.Type.IsInvalidType() {
		return true
	}

	for _, restriction := range t.Restrictions {
		if restriction.IsInvalidType() {
			return true
		}
	}

	return false
}

func (t *RestrictedType) IsStorable(results map[*Member]bool) bool {
	if t.Type != nil && !t.Type.IsStorable(results) {
		return false
	}

	for _, restriction := range t.Restrictions {
		if !restriction.IsStorable(results) {
			return false
		}
	}

	return true
}

func (t *RestrictedType) IsExternallyReturnable(results map[*Member]bool) bool {
	if t.Type != nil && !t.Type.IsExternallyReturnable(results) {
		return false
	}

	for _, restriction := range t.Restrictions {
		if !restriction.IsExternallyReturnable(results) {
			return false
		}
	}

	return true
}

func (t *RestrictedType) IsImportable(results map[*Member]bool) bool {
	if t.Type != nil && !t.Type.IsImportable(results) {
		return false
	}

	for _, restriction := range t.Restrictions {
		if !restriction.IsImportable(results) {
			return false
		}
	}

	return true
}

func (*RestrictedType) IsEquatable() bool {
	// TODO:
	return false
}

func (*RestrictedType) TypeAnnotationState() TypeAnnotationState {
	return TypeAnnotationStateValid
}

func (t *RestrictedType) RewriteWithRestrictedTypes() (Type, bool) {
	// Even though the restrictions should be resource interfaces,
	// they are not on the "first level", i.e. not the restricted type
	return t, false
}

func (t *RestrictedType) GetMembers() map[string]MemberResolver {

	members := map[string]MemberResolver{}

	// Return the members of all restrictions.
	// The invariant that restrictions may not have overlapping members is not checked here,
	// but implicitly when the resource declaration's conformances are checked.

	for _, restriction := range t.Restrictions {
		for name, resolver := range restriction.GetMembers() { //nolint:maprangecheck
			if _, ok := members[name]; !ok {
				members[name] = resolver
			}
		}
	}

	// Also include members of the restricted type for convenience,
	// to help check the rest of the program and improve the developer experience,
	// *but* also report an error that this access is invalid when the entry is resolved.
	//
	// The restricted type may be `AnyResource`, in which case there are no members.

	for name, loopResolver := range t.Type.GetMembers() { //nolint:maprangecheck

		if _, ok := members[name]; ok {
			continue
		}

		// NOTE: don't capture loop variable
		resolver := loopResolver

		members[name] = MemberResolver{
			Kind: resolver.Kind,
			Resolve: func(memoryGauge common.MemoryGauge, identifier string, targetRange ast.Range, report func(error)) *Member {
				member := resolver.Resolve(memoryGauge, identifier, targetRange, report)

				report(
					&InvalidRestrictedTypeMemberAccessError{
						Name:  identifier,
						Range: targetRange,
					},
				)

				return member
			},
		}
	}

	return members
}

func (*RestrictedType) Unify(_ Type, _ *TypeParameterTypeOrderedMap, _ func(err error), _ ast.Range) bool {
	// TODO: how do we unify the restriction sets?
	return false
}

func (t *RestrictedType) Resolve(_ *TypeParameterTypeOrderedMap) Type {
	// TODO:
	return t
}

// CapabilityType

type CapabilityType struct {
	BorrowType          Type
	memberResolvers     map[string]MemberResolver
	memberResolversOnce sync.Once
}

func NewCapabilityType(memoryGauge common.MemoryGauge, borrowType Type) *CapabilityType {
	common.UseMemory(memoryGauge, common.CapabilitySemaTypeMemoryUsage)
	return &CapabilityType{
		BorrowType: borrowType,
	}
}

func (*CapabilityType) IsType() {}

func (t *CapabilityType) Tag() TypeTag {
	return CapabilityTypeTag
}

func (t *CapabilityType) string(typeFormatter func(Type) string) string {
	var builder strings.Builder
	builder.WriteString("Capability")
	if t.BorrowType != nil {
		builder.WriteRune('<')
		builder.WriteString(typeFormatter(t.BorrowType))
		builder.WriteRune('>')
	}
	return builder.String()
}

func (t *CapabilityType) String() string {
	return t.string(func(t Type) string {
		return t.String()
	})
}

func (t *CapabilityType) QualifiedString() string {
	return t.string(func(t Type) string {
		return t.QualifiedString()
	})
}

func (t *CapabilityType) ID() TypeID {
	return TypeID(t.string(func(t Type) string {
		return string(t.ID())
	}))
}

func (t *CapabilityType) Equal(other Type) bool {
	otherCapability, ok := other.(*CapabilityType)
	if !ok {
		return false
	}
	if otherCapability.BorrowType == nil {
		return t.BorrowType == nil
	}
	return otherCapability.BorrowType.Equal(t.BorrowType)
}

func (*CapabilityType) IsResourceType() bool {
	return false
}

func (t *CapabilityType) IsInvalidType() bool {
	if t.BorrowType == nil {
		return false
	}
	return t.BorrowType.IsInvalidType()
}

func (t *CapabilityType) TypeAnnotationState() TypeAnnotationState {
	if t.BorrowType == nil {
		return TypeAnnotationStateValid
	}
	return t.BorrowType.TypeAnnotationState()
}

func (*CapabilityType) IsStorable(_ map[*Member]bool) bool {
	return true
}

func (*CapabilityType) IsExternallyReturnable(_ map[*Member]bool) bool {
	return true
}

func (t *CapabilityType) IsImportable(_ map[*Member]bool) bool {
	return true
}

func (*CapabilityType) IsEquatable() bool {
	// TODO:
	return false
}

func (t *CapabilityType) RewriteWithRestrictedTypes() (Type, bool) {
	if t.BorrowType == nil {
		return t, false
	}
	rewrittenType, rewritten := t.BorrowType.RewriteWithRestrictedTypes()
	if rewritten {
		return &CapabilityType{
			BorrowType: rewrittenType,
		}, true
	} else {
		return t, false
	}
}

func (t *CapabilityType) Unify(
	other Type,
	typeParameters *TypeParameterTypeOrderedMap,
	report func(err error),
	outerRange ast.Range,
) bool {
	otherCap, ok := other.(*CapabilityType)
	if !ok {
		return false
	}

	if t.BorrowType == nil {
		return false
	}

	return t.BorrowType.Unify(otherCap.BorrowType, typeParameters, report, outerRange)
}

func (t *CapabilityType) Resolve(typeArguments *TypeParameterTypeOrderedMap) Type {
	var resolvedBorrowType Type
	if t.BorrowType != nil {
		resolvedBorrowType = t.BorrowType.Resolve(typeArguments)
	}

	return &CapabilityType{
		BorrowType: resolvedBorrowType,
	}
}

var capabilityTypeParameter = &TypeParameter{
	Name: "T",
	TypeBound: &ReferenceType{
		Type: AnyType,
	},
}

func (t *CapabilityType) TypeParameters() []*TypeParameter {
	return []*TypeParameter{
		capabilityTypeParameter,
	}
}

func (t *CapabilityType) Instantiate(typeArguments []Type, _ func(err error)) Type {
	borrowType := typeArguments[0]
	return &CapabilityType{
		BorrowType: borrowType,
	}
}

func (t *CapabilityType) BaseType() Type {
	if t.BorrowType == nil {
		return nil
	}
	return &CapabilityType{}
}

func (t *CapabilityType) TypeArguments() []Type {
	borrowType := t.BorrowType
	if borrowType == nil {
		borrowType = &ReferenceType{
			Type: AnyType,
		}
	}
	return []Type{
		borrowType,
	}
}

func CapabilityTypeBorrowFunctionType(borrowType Type) *FunctionType {

	var typeParameters []*TypeParameter

	if borrowType == nil {
		typeParameter := capabilityTypeParameter

		typeParameters = []*TypeParameter{
			typeParameter,
		}

		borrowType = &GenericType{
			TypeParameter: typeParameter,
		}
	}

	return &FunctionType{
		TypeParameters: typeParameters,
		ReturnTypeAnnotation: NewTypeAnnotation(
			&OptionalType{
				Type: borrowType,
			},
		),
	}
}

func CapabilityTypeCheckFunctionType(borrowType Type) *FunctionType {

	var typeParameters []*TypeParameter

	if borrowType == nil {
		typeParameters = []*TypeParameter{
			capabilityTypeParameter,
		}
	}

	return &FunctionType{
		TypeParameters:       typeParameters,
		ReturnTypeAnnotation: NewTypeAnnotation(BoolType),
	}
}

const capabilityTypeBorrowFunctionDocString = `
Returns a reference to the object targeted by the capability, provided it can be borrowed using the given type
`

const capabilityTypeCheckFunctionDocString = `
Returns true if the capability currently targets an object that satisfies the given type, i.e. could be borrowed using the given type
`

const addressTypeCheckFunctionDocString = `
The address of the capability
`

func (t *CapabilityType) GetMembers() map[string]MemberResolver {
	t.initializeMemberResolvers()
	return t.memberResolvers
}

func (t *CapabilityType) initializeMemberResolvers() {
	t.memberResolversOnce.Do(func() {
		t.memberResolvers = withBuiltinMembers(t, map[string]MemberResolver{
			"borrow": {
				Kind: common.DeclarationKindFunction,
				Resolve: func(memoryGauge common.MemoryGauge, identifier string, _ ast.Range, _ func(error)) *Member {
					return NewPublicFunctionMember(
						memoryGauge,
						t,
						identifier,
						CapabilityTypeBorrowFunctionType(t.BorrowType),
						capabilityTypeBorrowFunctionDocString,
					)
				},
			},
			"check": {
				Kind: common.DeclarationKindFunction,
				Resolve: func(memoryGauge common.MemoryGauge, identifier string, _ ast.Range, _ func(error)) *Member {
					return NewPublicFunctionMember(
						memoryGauge,
						t,
						identifier,
						CapabilityTypeCheckFunctionType(t.BorrowType),
						capabilityTypeCheckFunctionDocString,
					)
				},
			},
			"address": {
				Kind: common.DeclarationKindField,
				Resolve: func(memoryGauge common.MemoryGauge, identifier string, _ ast.Range, _ func(error)) *Member {
					return NewPublicConstantFieldMember(
						memoryGauge,
						t,
						identifier,
						&AddressType{},
						addressTypeCheckFunctionDocString,
					)
				},
			},
		})
	})
}

var NativeCompositeTypes = map[string]*CompositeType{}

func init() {
	types := []*CompositeType{
		AccountKeyType,
		PublicKeyType,
		HashAlgorithmType,
		SignatureAlgorithmType,
		AuthAccountType,
		AuthAccountKeysType,
		AuthAccountContractsType,
		PublicAccountType,
		PublicAccountKeysType,
		PublicAccountContractsType,
	}

	for _, semaType := range types {
		NativeCompositeTypes[semaType.QualifiedIdentifier()] = semaType
	}
}

const AccountKeyTypeName = "AccountKey"
const AccountKeyKeyIndexField = "keyIndex"
const AccountKeyPublicKeyField = "publicKey"
const AccountKeyHashAlgoField = "hashAlgorithm"
const AccountKeyWeightField = "weight"
const AccountKeyIsRevokedField = "isRevoked"

// AccountKeyType represents the key associated with an account.
var AccountKeyType = func() *CompositeType {

	accountKeyType := &CompositeType{
		Identifier: AccountKeyTypeName,
		Kind:       common.CompositeKindStructure,
		importable: false,
	}

	const accountKeyIndexFieldDocString = `The index of the account key`
	const accountKeyPublicKeyFieldDocString = `The public key of the account`
	const accountKeyHashAlgorithmFieldDocString = `The hash algorithm used by the public key`
	const accountKeyWeightFieldDocString = `The weight assigned to the public key`
	const accountKeyIsRevokedFieldDocString = `Flag indicating whether the key is revoked`

	var members = []*Member{
		NewUnmeteredPublicConstantFieldMember(
			accountKeyType,
			AccountKeyKeyIndexField,
			IntType,
			accountKeyIndexFieldDocString,
		),
		NewUnmeteredPublicConstantFieldMember(
			accountKeyType,
			AccountKeyPublicKeyField,
			PublicKeyType,
			accountKeyPublicKeyFieldDocString,
		),
		NewUnmeteredPublicConstantFieldMember(
			accountKeyType,
			AccountKeyHashAlgoField,
			HashAlgorithmType,
			accountKeyHashAlgorithmFieldDocString,
		),
		NewUnmeteredPublicConstantFieldMember(
			accountKeyType,
			AccountKeyWeightField,
			UFix64Type,
			accountKeyWeightFieldDocString,
		),
		NewUnmeteredPublicConstantFieldMember(
			accountKeyType,
			AccountKeyIsRevokedField,
			BoolType,
			accountKeyIsRevokedFieldDocString,
		),
	}

	accountKeyType.Members = GetMembersAsMap(members)
	accountKeyType.Fields = getFieldNames(members)
	return accountKeyType
}()

const PublicKeyTypeName = "PublicKey"
const PublicKeyPublicKeyField = "publicKey"
const PublicKeySignAlgoField = "signatureAlgorithm"
const PublicKeyVerifyFunction = "verify"
const PublicKeyVerifyPoPFunction = "verifyPoP"

const publicKeyKeyFieldDocString = `
The public key
`

const publicKeySignAlgoFieldDocString = `
The signature algorithm to be used with the key
`

const publicKeyVerifyFunctionDocString = `
Verifies a signature. Checks whether the signature was produced by signing
the given tag and data, using this public key and the given hash algorithm
`

const publicKeyVerifyPoPFunctionDocString = `
Verifies the proof of possession of the private key.
This function is only implemented if the signature algorithm
of the public key is BLS (BLS_BLS12_381).
If called with any other signature algorithm, the program aborts
`

// PublicKeyType represents the public key associated with an account key.
var PublicKeyType = func() *CompositeType {

	publicKeyType := &CompositeType{
		Identifier:         PublicKeyTypeName,
		Kind:               common.CompositeKindStructure,
		hasComputedMembers: true,
		importable:         true,
	}

	var members = []*Member{
		NewUnmeteredPublicConstantFieldMember(
			publicKeyType,
			PublicKeyPublicKeyField,
			ByteArrayType,
			publicKeyKeyFieldDocString,
		),
		NewUnmeteredPublicConstantFieldMember(
			publicKeyType,
			PublicKeySignAlgoField,
			SignatureAlgorithmType,
			publicKeySignAlgoFieldDocString,
		),
		NewUnmeteredPublicFunctionMember(
			publicKeyType,
			PublicKeyVerifyFunction,
			PublicKeyVerifyFunctionType,
			publicKeyVerifyFunctionDocString,
		),
		NewUnmeteredPublicFunctionMember(
			publicKeyType,
			PublicKeyVerifyPoPFunction,
			PublicKeyVerifyPoPFunctionType,
			publicKeyVerifyPoPFunctionDocString,
		),
	}

	publicKeyType.Members = GetMembersAsMap(members)
	publicKeyType.Fields = getFieldNames(members)

	return publicKeyType
}()

var PublicKeyArrayType = &VariableSizedType{
	Type: PublicKeyType,
}

var PublicKeyVerifyFunctionType = &FunctionType{
	TypeParameters: []*TypeParameter{},
	Parameters: []*Parameter{
		{
			Identifier: "signature",
			TypeAnnotation: NewTypeAnnotation(
				ByteArrayType,
			),
		},
		{
			Identifier: "signedData",
			TypeAnnotation: NewTypeAnnotation(
				ByteArrayType,
			),
		},
		{
			Identifier:     "domainSeparationTag",
			TypeAnnotation: NewTypeAnnotation(StringType),
		},
		{
			Identifier:     "hashAlgorithm",
			TypeAnnotation: NewTypeAnnotation(HashAlgorithmType),
		},
	},
	ReturnTypeAnnotation: NewTypeAnnotation(BoolType),
}

var PublicKeyVerifyPoPFunctionType = &FunctionType{
	TypeParameters: []*TypeParameter{},
	Parameters: []*Parameter{
		{
			Label:      ArgumentLabelNotRequired,
			Identifier: "proof",
			TypeAnnotation: NewTypeAnnotation(
				ByteArrayType,
			),
		},
	},
	ReturnTypeAnnotation: NewTypeAnnotation(BoolType),
}

type CryptoAlgorithm interface {
	RawValue() uint8
	Name() string
	DocString() string
}

func GetMembersAsMap(members []*Member) *StringMemberOrderedMap {
	membersMap := NewStringMemberOrderedMap()
	for _, member := range members {
		name := member.Identifier.Identifier
		_, ok := membersMap.Get(name)
		if ok {
			panic(fmt.Errorf("invalid duplicate member: %s", name))
		}
		membersMap.Set(name, member)
	}

	return membersMap
}

func getFieldNames(members []*Member) []string {
	fields := make([]string, 0)
	for _, member := range members {
		if member.DeclarationKind == common.DeclarationKindField {
			fields = append(fields, member.Identifier.Identifier)
		}
	}

	return fields
}

func isNumericSuperType(typ Type) bool {
	if numberType, ok := typ.(IntegerRangedType); ok {
		return numberType.IsSuperType()
	}

	return false
}
