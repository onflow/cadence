/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Flow Foundation
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

	"golang.org/x/exp/slices"

	"github.com/onflow/cadence/fixedpoint"
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
)

const TypeIDSeparator = '.'

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
			sb.WriteByte(TypeIDSeparator)
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

	// IsPrimitiveType returns true if the type is itself a primitive,
	// Note that the container of a primitive type (e.g. optionals, arrays, dictionaries, etc.)
	// are not a primitive.
	IsPrimitiveType() bool

	// IsResourceType returns true if the type is itself a resource (a `CompositeType` with resource kind),
	// or it contains a resource type (e.g. for optionals, arrays, dictionaries, etc.)
	IsResourceType() bool

	// IsInvalidType returns true if the type is itself the invalid type (see `InvalidType`),
	// or it contains an invalid type (e.g. for optionals, arrays, dictionaries, etc.)
	IsInvalidType() bool

	// IsOrContainsReferenceType returns true if the type is itself a reference type,
	// or it contains a reference type (e.g. for optionals, arrays, dictionaries, etc.)
	IsOrContainsReferenceType() bool

	// IsStorable returns true if the type is allowed to be a stored,
	// e.g. in a field of a composite type.
	//
	// The check if the type is storable is recursive,
	// the results parameter prevents cycles:
	// it is checked at the start of the recursively called function,
	// and pre-set before a recursive call.
	IsStorable(results map[*Member]bool) bool

	// IsExportable returns true if a value of this type can be exported.
	//
	// The check if the type is exportable is recursive,
	// the results parameter prevents cycles:
	// it is checked at the start of the recursively called function,
	// and pre-set before a recursive call.
	IsExportable(results map[*Member]bool) bool

	// IsImportable returns true if values of the type can be imported to a program as arguments
	IsImportable(results map[*Member]bool) bool

	// IsEquatable returns true if values of the type can be equated
	IsEquatable() bool

	// IsComparable returns true if values of the type can be compared
	IsComparable() bool

	// ContainFieldsOrElements returns true if value of the type can have nested values (fields or elements).
	// This notion is to indicate that a type can be used to access its nested values using
	// either index-expression or member-expression. e.g. `foo.bar` or `foo[bar]`.
	// This is used to determine if a field/element of this type should be returning a reference or not.
	//
	// Only a subset of types has this characteristic. e.g:
	//  - Composites
	//  - Interfaces
	//  - Arrays (Variable/Constant sized)
	//  - Dictionaries
	//  - Restricted types
	//  - Optionals of the above.
	//  - Then there are also built-in simple types, like StorageCapabilityControllerType, BlockType, etc.
	//    where the type is implemented as a simple type, but they also have fields.
	//
	// This is different from the existing  `ValueIndexableType` in the sense that it is also implemented by simple types
	// but not all simple types are indexable.
	// On the other-hand, some indexable types (e.g. String) shouldn't be treated/returned as references.
	//
	ContainFieldsOrElements() bool

	TypeAnnotationState() TypeAnnotationState
	RewriteWithIntersectionTypes() (result Type, rewritten bool)

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
		memoryGauge common.MemoryGauge,
		outerRange ast.HasPosition,
	) bool

	// Resolve returns a type that is free of generic types (see `GenericType`),
	// i.e. it resolves the type parameters in generic types given the type parameter
	// unifications of `typeParameters`.
	//
	// If resolution fails, it returns `nil`.
	//
	Resolve(typeArguments *TypeParameterTypeOrderedMap) Type

	GetMembers() map[string]MemberResolver

	// applies `f` to all the types syntactically comprising this type.
	// i.e. `[T]` would map to `f([f(T)])`, but the internals of composite types are not
	// inspected, as they appear simply as nominal types in annotations
	Map(memoryGauge common.MemoryGauge, typeParamMap map[*TypeParameter]*TypeParameter, f func(Type) Type) Type

	CheckInstantiated(pos ast.HasPosition, memoryGauge common.MemoryGauge, report func(err error))
}

// ValueIndexableType is a type which can be indexed into using a value
type ValueIndexableType interface {
	Type
	isValueIndexableType() bool
	AllowsValueIndexingAssignment() bool
	ElementType(isAssignment bool) Type
	IndexingType() Type
}

// TypeIndexableType is a type which can be indexed into using a type
type TypeIndexableType interface {
	Type
	isTypeIndexableType() bool
	IsValidIndexingType(indexingType Type) bool
	TypeIndexingElementType(indexingType Type, astRange func() ast.Range) (Type, error)
}

type MemberResolver struct {
	Resolve func(
		memoryGauge common.MemoryGauge,
		identifier string,
		targetRange ast.HasPosition,
		report func(error),
	) *Member
	Kind common.DeclarationKind
}

// supertype of interfaces and composites
type NominalType interface {
	Type
	MemberMap() *StringMemberOrderedMap
}

// entitlement supporting types
type EntitlementSupportingType interface {
	Type
	SupportedEntitlements() *EntitlementSet
}

// ContainedType is a type which might have a container type
type ContainedType interface {
	Type
	GetContainerType() Type
	SetContainerType(containerType Type)
}

// ContainerType is a type which might have nested types
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

func TypeActivationNestedType(typeActivation *VariableActivation, qualifiedIdentifier string) Type {

	typeIDComponents := strings.Split(qualifiedIdentifier, string(TypeIDSeparator))

	rootTypeName := typeIDComponents[0]
	variable := typeActivation.Find(rootTypeName)
	if variable == nil {
		return nil
	}
	ty := variable.Type

	// Traverse nested types until the leaf type

	for i := 1; i < len(typeIDComponents); i++ {
		containerType, ok := ty.(ContainerType)
		if !ok || !containerType.IsContainerType() {
			return nil
		}

		typeIDComponent := typeIDComponents[i]

		ty, ok = containerType.GetNestedTypes().Get(typeIDComponent)
		if !ok {
			return nil
		}
	}

	return ty
}

// allow all types to specify interface conformances
type ConformingType interface {
	Type
	EffectiveInterfaceConformanceSet() *InterfaceSet
}

// CompositeKindedType is a type which has a composite kind
type CompositeKindedType interface {
	Type
	EntitlementSupportingType
	GetCompositeKind() common.CompositeKind
}

// LocatedType is a type which has a location
type LocatedType interface {
	Type
	GetLocation() common.Location
}

// ParameterizedType is a type which might have type parameters
type ParameterizedType interface {
	Type
	TypeParameters() []*TypeParameter
	Instantiate(
		memoryGauge common.MemoryGauge,
		typeArguments []Type,
		astTypeArguments []*ast.TypeAnnotation,
		report func(err error),
	) Type
	BaseType() Type
	TypeArguments() []Type
}

func MustInstantiate(t ParameterizedType, typeArguments ...Type) Type {
	return t.Instantiate(
		nil, /* memoryGauge */
		typeArguments,
		nil, /* astTypeArguments */
		func(err error) {
			panic(errors.NewUnexpectedErrorFromCause(err))
		},
	)
}

func CheckParameterizedTypeInstantiated(
	t ParameterizedType,
	pos ast.HasPosition,
	memoryGauge common.MemoryGauge,
	report func(err error),
) {
	typeArgs := t.TypeArguments()
	typeParameters := t.TypeParameters()

	// The check for the argument and parameter count already happens in the checker, so we skip that here.

	// Ensure that each non-optional typeparameter is non-nil.
	for index, typeParam := range typeParameters {
		if !typeParam.Optional && typeArgs[index] == nil {
			report(
				&MissingTypeArgumentError{
					TypeArgumentName: typeParam.Name,
					Range: ast.NewRange(
						memoryGauge,
						pos.StartPosition(),
						pos.EndPosition(memoryGauge),
					),
				},
			)
		}
	}
}

// TypeAnnotation

type TypeAnnotation struct {
	Type       Type
	IsResource bool
}

func (a TypeAnnotation) TypeAnnotationState() TypeAnnotationState {
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

func (a TypeAnnotation) String() string {
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

func (a TypeAnnotation) QualifiedString() string {
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

func (a TypeAnnotation) Equal(other TypeAnnotation) bool {
	return a.IsResource == other.IsResource &&
		a.Type.Equal(other.Type)
}

func NewTypeAnnotation(ty Type) TypeAnnotation {
	return TypeAnnotation{
		IsResource: ty.IsResourceType(),
		Type:       ty,
	}
}

func (a TypeAnnotation) Map(gauge common.MemoryGauge, typeParamMap map[*TypeParameter]*TypeParameter, f func(Type) Type) TypeAnnotation {
	return NewTypeAnnotation(a.Type.Map(gauge, typeParamMap, f))
}

// isInstance

const IsInstanceFunctionName = "isInstance"

var IsInstanceFunctionType = NewSimpleFunctionType(
	FunctionPurityView,
	[]Parameter{
		{
			Label:          ArgumentLabelNotRequired,
			Identifier:     "type",
			TypeAnnotation: MetaTypeAnnotation,
		},
	},
	BoolTypeAnnotation,
)

const isInstanceFunctionDocString = `
Returns true if the object conforms to the given type at runtime
`

// getType

const GetTypeFunctionName = "getType"

var GetTypeFunctionType = NewSimpleFunctionType(
	FunctionPurityView,
	nil,
	MetaTypeAnnotation,
)

const getTypeFunctionDocString = `
Returns the type of the value
`

// toString

const ToStringFunctionName = "toString"

var ToStringFunctionType = NewSimpleFunctionType(
	FunctionPurityView,
	nil,
	StringTypeAnnotation,
)

const toStringFunctionDocString = `
A textual representation of this object
`

// fromString
const FromStringFunctionName = "fromString"

func FromStringFunctionDocstring(ty Type) string {

	builder := new(strings.Builder)
	builder.WriteString(
		fmt.Sprintf(
			"Attempts to parse %s from a string. Returns `nil` on overflow or invalid input. Whitespace or invalid digits will return a nil value.\n",
			ty.String(),
		))

	if IsSameTypeKind(ty, FixedPointType) {
		builder.WriteString(
			`Both decimal and fractional components must be supplied. For instance, both "0." and ".1" are invalid string representations, but "0.1" is accepted.\n`,
		)
	}
	if IsSameTypeKind(ty, SignedIntegerType) || IsSameTypeKind(ty, SignedFixedPointType) {
		builder.WriteString(
			"The string may optionally begin with a sign prefix of '-' or '+'.\n",
		)
	}

	return builder.String()
}

func FromStringFunctionType(ty Type) *FunctionType {
	return NewSimpleFunctionType(
		FunctionPurityView,
		[]Parameter{
			{
				Label:          ArgumentLabelNotRequired,
				Identifier:     "input",
				TypeAnnotation: StringTypeAnnotation,
			},
		},
		NewTypeAnnotation(
			&OptionalType{
				Type: ty,
			},
		),
	)
}

// fromBigEndianBytes

const FromBigEndianBytesFunctionName = "fromBigEndianBytes"

func FromBigEndianBytesFunctionDocstring(ty Type) string {
	return fmt.Sprintf(
		"Attempts to parse %s from a big-endian byte representation. Returns `nil` on invalid input.",
		ty.String(),
	)
}

func FromBigEndianBytesFunctionType(ty Type) *FunctionType {
	return &FunctionType{
		Purity: FunctionPurityView,
		Parameters: []Parameter{
			{
				Label:          ArgumentLabelNotRequired,
				Identifier:     "bytes",
				TypeAnnotation: NewTypeAnnotation(ByteArrayType),
			},
		},
		ReturnTypeAnnotation: NewTypeAnnotation(
			&OptionalType{
				Type: ty,
			},
		),
	}
}

// toBigEndianBytes

const ToBigEndianBytesFunctionName = "toBigEndianBytes"

var ToBigEndianBytesFunctionType = NewSimpleFunctionType(
	FunctionPurityView,
	nil,
	ByteArrayTypeAnnotation,
)

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
		Resolve: func(memoryGauge common.MemoryGauge, identifier string, _ ast.HasPosition, _ func(error)) *Member {
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
		Resolve: func(memoryGauge common.MemoryGauge, identifier string, _ ast.HasPosition, _ func(error)) *Member {
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

	if IsSubType(ty, NumberType) || IsSubType(ty, TheAddressType) || IsSubType(ty, PathType) {

		members[ToStringFunctionName] = MemberResolver{
			Kind: common.DeclarationKindFunction,
			Resolve: func(memoryGauge common.MemoryGauge, identifier string, _ ast.HasPosition, _ func(error)) *Member {
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
			Resolve: func(memoryGauge common.MemoryGauge, identifier string, _ ast.HasPosition, _ func(error)) *Member {
				return NewPublicFunctionMember(
					memoryGauge,
					ty,
					identifier,
					ToBigEndianBytesFunctionType,
					toBigEndianBytesFunctionDocString,
				)
			},
		}
	}

	return members
}

// OptionalType represents the optional variant of another type
type OptionalType struct {
	Type                Type
	memberResolvers     map[string]MemberResolver
	memberResolversOnce sync.Once
}

var _ Type = &OptionalType{}

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

func FormatOptionalTypeID[T ~string](elementTypeID T) T {
	return T(fmt.Sprintf("(%s)?", elementTypeID))
}

func (t *OptionalType) ID() TypeID {
	return FormatOptionalTypeID(t.Type.ID())
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

func (t *OptionalType) IsPrimitiveType() bool {
	return t.Type.IsPrimitiveType()
}

func (t *OptionalType) IsInvalidType() bool {
	return t.Type.IsInvalidType()
}

func (t *OptionalType) IsOrContainsReferenceType() bool {
	return t.Type.IsOrContainsReferenceType()
}

func (t *OptionalType) IsStorable(results map[*Member]bool) bool {
	return t.Type.IsStorable(results)
}

func (t *OptionalType) IsExportable(results map[*Member]bool) bool {
	return t.Type.IsExportable(results)
}

func (t *OptionalType) IsImportable(results map[*Member]bool) bool {
	return t.Type.IsImportable(results)
}

func (t *OptionalType) IsEquatable() bool {
	return t.Type.IsEquatable()
}

func (*OptionalType) IsComparable() bool {
	return false
}

func (t *OptionalType) ContainFieldsOrElements() bool {
	return t.Type.ContainFieldsOrElements()
}

func (t *OptionalType) TypeAnnotationState() TypeAnnotationState {
	return t.Type.TypeAnnotationState()
}

func (t *OptionalType) RewriteWithIntersectionTypes() (Type, bool) {
	rewrittenType, rewritten := t.Type.RewriteWithIntersectionTypes()
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
	memoryGauge common.MemoryGauge,
	outerRange ast.HasPosition,
) bool {

	otherOptional, ok := other.(*OptionalType)
	if !ok {
		return false
	}

	return t.Type.Unify(
		otherOptional.Type,
		typeParameters,
		report,
		memoryGauge,
		outerRange,
	)
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

func (t *OptionalType) SupportedEntitlements() *EntitlementSet {
	if entitlementSupportingType, ok := t.Type.(EntitlementSupportingType); ok {
		return entitlementSupportingType.SupportedEntitlements()
	}
	return nil
}

func (t *OptionalType) CheckInstantiated(pos ast.HasPosition, memoryGauge common.MemoryGauge, report func(err error)) {
	t.Type.CheckInstantiated(pos, memoryGauge, report)
}

const optionalTypeMapFunctionDocString = `
Returns an optional of the result of calling the given function
with the value of this optional when it is not nil.

Returns nil if this optional is nil
`

const OptionalTypeMapFunctionName = "map"

func (t *OptionalType) Map(memoryGauge common.MemoryGauge, typeParamMap map[*TypeParameter]*TypeParameter, f func(Type) Type) Type {
	return f(NewOptionalType(memoryGauge, t.Type.Map(memoryGauge, typeParamMap, f)))
}

func (t *OptionalType) GetMembers() map[string]MemberResolver {
	t.initializeMembers()
	return t.memberResolvers
}

func (t *OptionalType) initializeMembers() {
	t.memberResolversOnce.Do(func() {
		t.memberResolvers = withBuiltinMembers(
			t,
			map[string]MemberResolver{
				OptionalTypeMapFunctionName: {
					Kind: common.DeclarationKindFunction,
					Resolve: func(
						memoryGauge common.MemoryGauge,
						identifier string,
						targetRange ast.HasPosition,
						report func(error),
					) *Member {

						// It's invalid for an optional of a resource to have a `map` function

						if t.Type.IsResourceType() {
							report(
								&InvalidResourceOptionalMemberError{
									Name:            identifier,
									DeclarationKind: common.DeclarationKindFunction,
									Range:           ast.NewRangeFromPositioned(memoryGauge, targetRange),
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
			},
		)
	})
}

func OptionalTypeMapFunctionType(typ Type) *FunctionType {
	typeParameter := &TypeParameter{
		Name: "T",
	}

	resultType := &GenericType{
		TypeParameter: typeParameter,
	}

	const functionPurity = FunctionPurityImpure

	return &FunctionType{
		Purity: functionPurity,
		TypeParameters: []*TypeParameter{
			typeParameter,
		},
		Parameters: []Parameter{
			{
				Label:      ArgumentLabelNotRequired,
				Identifier: "transform",
				TypeAnnotation: NewTypeAnnotation(
					&FunctionType{
						Purity: functionPurity,
						Parameters: []Parameter{
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
type GenericType struct {
	TypeParameter *TypeParameter
}

var _ Type = &GenericType{}

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

func (*GenericType) IsPrimitiveType() bool {
	return false
}

func (*GenericType) IsInvalidType() bool {
	return false
}

func (*GenericType) IsOrContainsReferenceType() bool {
	return false
}

func (*GenericType) IsStorable(_ map[*Member]bool) bool {
	return false
}

func (*GenericType) IsExportable(_ map[*Member]bool) bool {
	return false
}

func (t *GenericType) IsImportable(_ map[*Member]bool) bool {
	return false
}

func (*GenericType) IsEquatable() bool {
	return false
}

func (*GenericType) IsComparable() bool {
	return false
}

func (t *GenericType) ContainFieldsOrElements() bool {
	return false
}

func (*GenericType) TypeAnnotationState() TypeAnnotationState {
	return TypeAnnotationStateValid
}

func (t *GenericType) RewriteWithIntersectionTypes() (result Type, rewritten bool) {
	return t, false
}

func (t *GenericType) Unify(
	other Type,
	typeParameters *TypeParameterTypeOrderedMap,
	report func(err error),
	memoryGauge common.MemoryGauge,
	outerRange ast.HasPosition,
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
					Range:         ast.NewRangeFromPositioned(memoryGauge, outerRange),
				},
			)
		}

	} else {
		// If the type parameter is not yet unified to a type argument, unify it.

		typeParameters.Set(t.TypeParameter, other)

		// If the type parameter corresponding to the type argument has a type bound,
		// then check that the argument's type is a subtype of the type bound.

		err := t.TypeParameter.checkTypeBound(other, memoryGauge, outerRange)
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

func (t *GenericType) Map(_ common.MemoryGauge, typeParamMap map[*TypeParameter]*TypeParameter, f func(Type) Type) Type {
	if param, ok := typeParamMap[t.TypeParameter]; ok {
		return f(&GenericType{
			TypeParameter: param,
		})
	}
	panic(errors.NewUnreachableError())
}

func (t *GenericType) GetMembers() map[string]MemberResolver {
	return withBuiltinMembers(t, nil)
}

func (t *GenericType) CheckInstantiated(pos ast.HasPosition, memoryGauge common.MemoryGauge, report func(err error)) {
	if t.TypeParameter.TypeBound != nil {
		t.TypeParameter.TypeBound.CheckInstantiated(pos, memoryGauge, report)
	}
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

var SaturatingArithmeticTypeFunctionTypes = map[Type]*FunctionType{}

func registerSaturatingArithmeticType(t Type) {
	SaturatingArithmeticTypeFunctionTypes[t] = NewSimpleFunctionType(
		FunctionPurityView,
		[]Parameter{
			{
				Label:          ArgumentLabelNotRequired,
				Identifier:     "other",
				TypeAnnotation: NewTypeAnnotation(t),
			},
		},
		NewTypeAnnotation(t),
	)
}

func addSaturatingArithmeticFunctions(t SaturatingArithmeticType, members map[string]MemberResolver) {

	addArithmeticFunction := func(name string, docString string) {
		members[name] = MemberResolver{
			Kind: common.DeclarationKindFunction,
			Resolve: func(memoryGauge common.MemoryGauge, _ string, _ ast.HasPosition, _ func(error)) *Member {
				return NewPublicFunctionMember(
					memoryGauge,
					t,
					name,
					SaturatingArithmeticTypeFunctionTypes[t],
					docString,
				)
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

type SaturatingArithmeticSupport struct {
	Add      bool
	Subtract bool
	Multiply bool
	Divide   bool
}

// NumericType represent all the types in the integer range
// and non-fractional ranged types.
type NumericType struct {
	minInt               *big.Int
	maxInt               *big.Int
	byteSize             int
	memberResolvers      map[string]MemberResolver
	name                 string
	tag                  TypeTag
	memberResolversOnce  sync.Once
	saturatingArithmetic SaturatingArithmeticSupport
	isSuperType          bool

	// allow numeric types to conform to interfaces
	conformances                         []*InterfaceType
	effectiveInterfaceConformanceSet     *InterfaceSet
	effectiveInterfaceConformanceSetOnce sync.Once
}

var _ Type = &NumericType{}
var _ IntegerRangedType = &NumericType{}
var _ SaturatingArithmeticType = &NumericType{}

func NewNumericType(typeName string) *NumericType {
	return &NumericType{
		name: typeName,
		conformances: []*InterfaceType{
			StructStringerType,
		},
	}
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

func (t *NumericType) WithByteSize(size int) *NumericType {
	t.byteSize = size
	return t
}

func (t *NumericType) WithSaturatingFunctions(saturatingArithmetic SaturatingArithmeticSupport) *NumericType {
	t.saturatingArithmetic = saturatingArithmetic

	registerSaturatingArithmeticType(t)

	return t
}

func (t *NumericType) SupportsSaturatingAdd() bool {
	return t.saturatingArithmetic.Add
}

func (t *NumericType) SupportsSaturatingSubtract() bool {
	return t.saturatingArithmetic.Subtract
}

func (t *NumericType) SupportsSaturatingMultiply() bool {
	return t.saturatingArithmetic.Multiply
}

func (t *NumericType) SupportsSaturatingDivide() bool {
	return t.saturatingArithmetic.Divide
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

func (*NumericType) IsPrimitiveType() bool {
	return true
}

func (*NumericType) IsInvalidType() bool {
	return false
}

func (*NumericType) IsOrContainsReferenceType() bool {
	return false
}

func (*NumericType) IsStorable(_ map[*Member]bool) bool {
	return true
}

func (*NumericType) IsExportable(_ map[*Member]bool) bool {
	return true
}

func (t *NumericType) IsImportable(_ map[*Member]bool) bool {
	return true
}

func (*NumericType) IsEquatable() bool {
	return true
}

func (t *NumericType) IsComparable() bool {
	return !t.IsSuperType()
}

func (t *NumericType) ContainFieldsOrElements() bool {
	return false
}

func (*NumericType) TypeAnnotationState() TypeAnnotationState {
	return TypeAnnotationStateValid
}

func (t *NumericType) RewriteWithIntersectionTypes() (result Type, rewritten bool) {
	return t, false
}

func (t *NumericType) MinInt() *big.Int {
	return t.minInt
}

func (t *NumericType) MaxInt() *big.Int {
	return t.maxInt
}

func (t *NumericType) ByteSize() int {
	return t.byteSize
}

func (*NumericType) Unify(
	_ Type,
	_ *TypeParameterTypeOrderedMap,
	_ func(err error),
	_ common.MemoryGauge,
	_ ast.HasPosition,
) bool {
	return false
}

func (t *NumericType) Resolve(_ *TypeParameterTypeOrderedMap) Type {
	return t
}

func (t *NumericType) Map(_ common.MemoryGauge, _ map[*TypeParameter]*TypeParameter, f func(Type) Type) Type {
	return f(t)
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

func (*NumericType) CheckInstantiated(_ ast.HasPosition, _ common.MemoryGauge, _ func(err error)) {
	// NO-OP
}

func (t *NumericType) EffectiveInterfaceConformanceSet() *InterfaceSet {
	t.initializeEffectiveInterfaceConformanceSet()
	return t.effectiveInterfaceConformanceSet
}

func (t *NumericType) initializeEffectiveInterfaceConformanceSet() {
	t.effectiveInterfaceConformanceSetOnce.Do(func() {
		t.effectiveInterfaceConformanceSet = NewInterfaceSet()

		for _, conformance := range t.conformances {
			t.effectiveInterfaceConformanceSet.Add(conformance)
		}
	})
}

// FixedPointNumericType represents all the types in the fixed-point range.
type FixedPointNumericType struct {
	maxFractional        *big.Int
	minFractional        *big.Int
	memberResolvers      map[string]MemberResolver
	minInt               *big.Int
	maxInt               *big.Int
	name                 string
	tag                  TypeTag
	scale                uint
	memberResolversOnce  sync.Once
	saturatingArithmetic SaturatingArithmeticSupport
	isSuperType          bool
}

var _ Type = &FixedPointNumericType{}
var _ IntegerRangedType = &FixedPointNumericType{}
var _ FractionalRangedType = &FixedPointNumericType{}
var _ SaturatingArithmeticType = &FixedPointNumericType{}

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

func (t *FixedPointNumericType) WithSaturatingFunctions(saturatingArithmetic SaturatingArithmeticSupport) *FixedPointNumericType {
	t.saturatingArithmetic = saturatingArithmetic

	registerSaturatingArithmeticType(t)

	return t
}

func (t *FixedPointNumericType) SupportsSaturatingAdd() bool {
	return t.saturatingArithmetic.Add
}

func (t *FixedPointNumericType) SupportsSaturatingSubtract() bool {
	return t.saturatingArithmetic.Subtract
}

func (t *FixedPointNumericType) SupportsSaturatingMultiply() bool {
	return t.saturatingArithmetic.Multiply
}

func (t *FixedPointNumericType) SupportsSaturatingDivide() bool {
	return t.saturatingArithmetic.Divide
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

func (*FixedPointNumericType) IsPrimitiveType() bool {
	return true
}

func (*FixedPointNumericType) IsInvalidType() bool {
	return false
}

func (*FixedPointNumericType) IsOrContainsReferenceType() bool {
	return false
}

func (*FixedPointNumericType) IsStorable(_ map[*Member]bool) bool {
	return true
}

func (*FixedPointNumericType) IsExportable(_ map[*Member]bool) bool {
	return true
}

func (t *FixedPointNumericType) IsImportable(_ map[*Member]bool) bool {
	return true
}

func (*FixedPointNumericType) IsEquatable() bool {
	return true
}

func (t *FixedPointNumericType) IsComparable() bool {
	return !t.IsSuperType()
}

func (t *FixedPointNumericType) ContainFieldsOrElements() bool {
	return false
}

func (*FixedPointNumericType) TypeAnnotationState() TypeAnnotationState {
	return TypeAnnotationStateValid
}

func (t *FixedPointNumericType) RewriteWithIntersectionTypes() (result Type, rewritten bool) {
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

func (*FixedPointNumericType) Unify(
	_ Type,
	_ *TypeParameterTypeOrderedMap,
	_ func(err error),
	_ common.MemoryGauge,
	_ ast.HasPosition,
) bool {
	return false
}

func (t *FixedPointNumericType) Resolve(_ *TypeParameterTypeOrderedMap) Type {
	return t
}

func (t *FixedPointNumericType) Map(_ common.MemoryGauge, _ map[*TypeParameter]*TypeParameter, f func(Type) Type) Type {
	return f(t)
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

func (*FixedPointNumericType) CheckInstantiated(_ ast.HasPosition, _ common.MemoryGauge, _ func(err error)) {
	// NO-OP
}

// Numeric types

var (

	// NumberType represents the super-type of all number types
	NumberType = NewNumericType(NumberTypeName).
			WithTag(NumberTypeTag).
			AsSuperType()

	NumberTypeAnnotation = NewTypeAnnotation(NumberType)

	// SignedNumberType represents the super-type of all signed number types
	SignedNumberType = NewNumericType(SignedNumberTypeName).
				WithTag(SignedNumberTypeTag).
				AsSuperType()

	SignedNumberTypeAnnotation = NewTypeAnnotation(SignedNumberType)

	// IntegerType represents the super-type of all integer types
	IntegerType = NewNumericType(IntegerTypeName).
			WithTag(IntegerTypeTag).
			AsSuperType()

	IntegerTypeAnnotation = NewTypeAnnotation(IntegerType)

	// SignedIntegerType represents the super-type of all signed integer types
	SignedIntegerType = NewNumericType(SignedIntegerTypeName).
				WithTag(SignedIntegerTypeTag).
				AsSuperType()

	SignedIntegerTypeAnnotation = NewTypeAnnotation(SignedIntegerType)

	// FixedSizeUnsignedIntegerType represents the super-type of all unsigned integer types which have a fixed size.
	FixedSizeUnsignedIntegerType = NewNumericType(FixedSizeUnsignedIntegerTypeName).
					WithTag(FixedSizeUnsignedIntegerTypeTag).
					AsSuperType()

	// IntType represents the arbitrary-precision integer type `Int`
	IntType = NewNumericType(IntTypeName).
		WithTag(IntTypeTag)

	IntTypeAnnotation = NewTypeAnnotation(IntType)

	// Int8Type represents the 8-bit signed integer type `Int8`
	Int8Type = NewNumericType(Int8TypeName).
			WithTag(Int8TypeTag).
			WithIntRange(Int8TypeMinInt, Int8TypeMaxInt).
			WithByteSize(1).
			WithSaturatingFunctions(SaturatingArithmeticSupport{
			Add:      true,
			Subtract: true,
			Multiply: true,
			Divide:   true,
		})

	Int8TypeAnnotation = NewTypeAnnotation(Int8Type)

	// Int16Type represents the 16-bit signed integer type `Int16`
	Int16Type = NewNumericType(Int16TypeName).
			WithTag(Int16TypeTag).
			WithIntRange(Int16TypeMinInt, Int16TypeMaxInt).
			WithByteSize(2).
			WithSaturatingFunctions(SaturatingArithmeticSupport{
			Add:      true,
			Subtract: true,
			Multiply: true,
			Divide:   true,
		})

	Int16TypeAnnotation = NewTypeAnnotation(Int16Type)

	// Int32Type represents the 32-bit signed integer type `Int32`
	Int32Type = NewNumericType(Int32TypeName).
			WithTag(Int32TypeTag).
			WithIntRange(Int32TypeMinInt, Int32TypeMaxInt).
			WithByteSize(4).
			WithSaturatingFunctions(SaturatingArithmeticSupport{
			Add:      true,
			Subtract: true,
			Multiply: true,
			Divide:   true,
		})

	Int32TypeAnnotation = NewTypeAnnotation(Int32Type)

	// Int64Type represents the 64-bit signed integer type `Int64`
	Int64Type = NewNumericType(Int64TypeName).
			WithTag(Int64TypeTag).
			WithIntRange(Int64TypeMinInt, Int64TypeMaxInt).
			WithByteSize(8).
			WithSaturatingFunctions(SaturatingArithmeticSupport{
			Add:      true,
			Subtract: true,
			Multiply: true,
			Divide:   true,
		})

	Int64TypeAnnotation = NewTypeAnnotation(Int64Type)

	// Int128Type represents the 128-bit signed integer type `Int128`
	Int128Type = NewNumericType(Int128TypeName).
			WithTag(Int128TypeTag).
			WithIntRange(Int128TypeMinIntBig, Int128TypeMaxIntBig).
			WithByteSize(16).
			WithSaturatingFunctions(SaturatingArithmeticSupport{
			Add:      true,
			Subtract: true,
			Multiply: true,
			Divide:   true,
		})

	Int128TypeAnnotation = NewTypeAnnotation(Int128Type)

	// Int256Type represents the 256-bit signed integer type `Int256`
	Int256Type = NewNumericType(Int256TypeName).
			WithTag(Int256TypeTag).
			WithIntRange(Int256TypeMinIntBig, Int256TypeMaxIntBig).
			WithByteSize(32).
			WithSaturatingFunctions(SaturatingArithmeticSupport{
			Add:      true,
			Subtract: true,
			Multiply: true,
			Divide:   true,
		})

	Int256TypeAnnotation = NewTypeAnnotation(Int256Type)

	// UIntType represents the arbitrary-precision unsigned integer type `UInt`
	UIntType = NewNumericType(UIntTypeName).
			WithTag(UIntTypeTag).
			WithIntRange(UIntTypeMin, nil).
			WithSaturatingFunctions(SaturatingArithmeticSupport{
			Subtract: true,
		})

	UIntTypeAnnotation = NewTypeAnnotation(UIntType)

	// UInt8Type represents the 8-bit unsigned integer type `UInt8`
	// which checks for overflow and underflow
	UInt8Type = NewNumericType(UInt8TypeName).
			WithTag(UInt8TypeTag).
			WithIntRange(UInt8TypeMinInt, UInt8TypeMaxInt).
			WithByteSize(1).
			WithSaturatingFunctions(SaturatingArithmeticSupport{
			Add:      true,
			Subtract: true,
			Multiply: true,
		})

	UInt8TypeAnnotation = NewTypeAnnotation(UInt8Type)

	// UInt16Type represents the 16-bit unsigned integer type `UInt16`
	// which checks for overflow and underflow
	UInt16Type = NewNumericType(UInt16TypeName).
			WithTag(UInt16TypeTag).
			WithIntRange(UInt16TypeMinInt, UInt16TypeMaxInt).
			WithByteSize(2).
			WithSaturatingFunctions(SaturatingArithmeticSupport{
			Add:      true,
			Subtract: true,
			Multiply: true,
		})

	UInt16TypeAnnotation = NewTypeAnnotation(UInt16Type)

	// UInt32Type represents the 32-bit unsigned integer type `UInt32`
	// which checks for overflow and underflow
	UInt32Type = NewNumericType(UInt32TypeName).
			WithTag(UInt32TypeTag).
			WithIntRange(UInt32TypeMinInt, UInt32TypeMaxInt).
			WithByteSize(4).
			WithSaturatingFunctions(SaturatingArithmeticSupport{
			Add:      true,
			Subtract: true,
			Multiply: true,
		})

	UInt32TypeAnnotation = NewTypeAnnotation(UInt32Type)

	// UInt64Type represents the 64-bit unsigned integer type `UInt64`
	// which checks for overflow and underflow
	UInt64Type = NewNumericType(UInt64TypeName).
			WithTag(UInt64TypeTag).
			WithIntRange(UInt64TypeMinInt, UInt64TypeMaxInt).
			WithByteSize(8).
			WithSaturatingFunctions(SaturatingArithmeticSupport{
			Add:      true,
			Subtract: true,
			Multiply: true,
		})

	UInt64TypeAnnotation = NewTypeAnnotation(UInt64Type)

	// UInt128Type represents the 128-bit unsigned integer type `UInt128`
	// which checks for overflow and underflow
	UInt128Type = NewNumericType(UInt128TypeName).
			WithTag(UInt128TypeTag).
			WithIntRange(UInt128TypeMinIntBig, UInt128TypeMaxIntBig).
			WithByteSize(16).
			WithSaturatingFunctions(SaturatingArithmeticSupport{
			Add:      true,
			Subtract: true,
			Multiply: true,
		})

	UInt128TypeAnnotation = NewTypeAnnotation(UInt128Type)

	// UInt256Type represents the 256-bit unsigned integer type `UInt256`
	// which checks for overflow and underflow
	UInt256Type = NewNumericType(UInt256TypeName).
			WithTag(UInt256TypeTag).
			WithIntRange(UInt256TypeMinIntBig, UInt256TypeMaxIntBig).
			WithByteSize(32).
			WithSaturatingFunctions(SaturatingArithmeticSupport{
			Add:      true,
			Subtract: true,
			Multiply: true,
		})

	UInt256TypeAnnotation = NewTypeAnnotation(UInt256Type)

	// Word8Type represents the 8-bit unsigned integer type `Word8`
	// which does NOT check for overflow and underflow
	Word8Type = NewNumericType(Word8TypeName).
			WithTag(Word8TypeTag).
			WithByteSize(1).
			WithIntRange(Word8TypeMinInt, Word8TypeMaxInt)

	Word8TypeAnnotation = NewTypeAnnotation(Word8Type)

	// Word16Type represents the 16-bit unsigned integer type `Word16`
	// which does NOT check for overflow and underflow
	Word16Type = NewNumericType(Word16TypeName).
			WithTag(Word16TypeTag).
			WithByteSize(2).
			WithIntRange(Word16TypeMinInt, Word16TypeMaxInt)

	Word16TypeAnnotation = NewTypeAnnotation(Word16Type)

	// Word32Type represents the 32-bit unsigned integer type `Word32`
	// which does NOT check for overflow and underflow
	Word32Type = NewNumericType(Word32TypeName).
			WithTag(Word32TypeTag).
			WithByteSize(4).
			WithIntRange(Word32TypeMinInt, Word32TypeMaxInt)

	Word32TypeAnnotation = NewTypeAnnotation(Word32Type)

	// Word64Type represents the 64-bit unsigned integer type `Word64`
	// which does NOT check for overflow and underflow
	Word64Type = NewNumericType(Word64TypeName).
			WithTag(Word64TypeTag).
			WithByteSize(8).
			WithIntRange(Word64TypeMinInt, Word64TypeMaxInt)

	Word64TypeAnnotation = NewTypeAnnotation(Word64Type)

	// Word128Type represents the 128-bit unsigned integer type `Word128`
	// which does NOT check for overflow and underflow
	Word128Type = NewNumericType(Word128TypeName).
			WithTag(Word128TypeTag).
			WithByteSize(16).
			WithIntRange(Word128TypeMinIntBig, Word128TypeMaxIntBig)

	Word128TypeAnnotation = NewTypeAnnotation(Word128Type)

	// Word256Type represents the 256-bit unsigned integer type `Word256`
	// which does NOT check for overflow and underflow
	Word256Type = NewNumericType(Word256TypeName).
			WithTag(Word256TypeTag).
			WithByteSize(32).
			WithIntRange(Word256TypeMinIntBig, Word256TypeMaxIntBig)

	Word256TypeAnnotation = NewTypeAnnotation(Word256Type)

	// FixedPointType represents the super-type of all fixed-point types
	FixedPointType = NewNumericType(FixedPointTypeName).
			WithTag(FixedPointTypeTag).
			AsSuperType()

	FixedPointTypeAnnotation = NewTypeAnnotation(FixedPointType)

	// SignedFixedPointType represents the super-type of all signed fixed-point types
	SignedFixedPointType = NewNumericType(SignedFixedPointTypeName).
				WithTag(SignedFixedPointTypeTag).
				AsSuperType()

	SignedFixedPointTypeAnnotation = NewTypeAnnotation(SignedFixedPointType)

	// Fix64Type represents the 64-bit signed decimal fixed-point type `Fix64`
	// which has a scale of Fix64Scale, and checks for overflow and underflow
	Fix64Type = NewFixedPointNumericType(Fix64TypeName).
			WithTag(Fix64TypeTag).
			WithIntRange(Fix64TypeMinIntBig, Fix64TypeMaxIntBig).
			WithFractionalRange(Fix64TypeMinFractionalBig, Fix64TypeMaxFractionalBig).
			WithScale(Fix64Scale).
			WithSaturatingFunctions(SaturatingArithmeticSupport{
			Add:      true,
			Subtract: true,
			Multiply: true,
			Divide:   true,
		})

	Fix64TypeAnnotation = NewTypeAnnotation(Fix64Type)

	// UFix64Type represents the 64-bit unsigned decimal fixed-point type `UFix64`
	// which has a scale of 1E9, and checks for overflow and underflow
	UFix64Type = NewFixedPointNumericType(UFix64TypeName).
			WithTag(UFix64TypeTag).
			WithIntRange(UFix64TypeMinIntBig, UFix64TypeMaxIntBig).
			WithFractionalRange(UFix64TypeMinFractionalBig, UFix64TypeMaxFractionalBig).
			WithScale(Fix64Scale).
			WithSaturatingFunctions(SaturatingArithmeticSupport{
			Add:      true,
			Subtract: true,
			Multiply: true,
		})

	UFix64TypeAnnotation = NewTypeAnnotation(UFix64Type)
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

	// 1 << 128
	Word128TypeMaxIntPlusOneBig = func() *big.Int {
		word128TypeMaxPlusOne := big.NewInt(1)
		word128TypeMaxPlusOne.Lsh(word128TypeMaxPlusOne, 128)
		return word128TypeMaxPlusOne
	}()
	Word128TypeMinIntBig = new(big.Int)
	Word128TypeMaxIntBig = func() *big.Int {
		word128TypeMax := new(big.Int)
		word128TypeMax.Sub(Word128TypeMaxIntPlusOneBig, big.NewInt(1))
		return word128TypeMax
	}()

	// 1 << 256
	Word256TypeMaxIntPlusOneBig = func() *big.Int {
		word256TypeMaxPlusOne := big.NewInt(1)
		word256TypeMaxPlusOne.Lsh(word256TypeMaxPlusOne, 256)
		return word256TypeMaxPlusOne
	}()
	Word256TypeMinIntBig = new(big.Int)
	Word256TypeMaxIntBig = func() *big.Int {
		word256TypeMax := new(big.Int)
		word256TypeMax.Sub(Word256TypeMaxIntPlusOneBig, big.NewInt(1))
		return word256TypeMax
	}()

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

// size constants (in bytes) for fixed-width numeric types
const (
	Int8TypeSize    uint = 1
	UInt8TypeSize   uint = 1
	Word8TypeSize   uint = 1
	Int16TypeSize   uint = 2
	UInt16TypeSize  uint = 2
	Word16TypeSize  uint = 2
	Int32TypeSize   uint = 4
	UInt32TypeSize  uint = 4
	Word32TypeSize  uint = 4
	Int64TypeSize   uint = 8
	UInt64TypeSize  uint = 8
	Word64TypeSize  uint = 8
	Fix64TypeSize   uint = 8
	UFix64TypeSize  uint = 8
	Int128TypeSize  uint = 16
	UInt128TypeSize uint = 16
	Int256TypeSize  uint = 32
	UInt256TypeSize uint = 32
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
	EntitlementSupportingType
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

const ArrayTypeReverseFunctionName = "reverse"

const arrayTypeReverseFunctionDocString = `
Returns a new array with contents in the reversed order.
Available if the array element type is not resource-kinded.
`

const ArrayTypeToVariableSizedFunctionName = "toVariableSized"

const arrayTypeToVariableSizedFunctionDocString = `
Returns a new variable-sized array with the copy of the contents of the given array.
Available if the array is constant sized and the element type is not resource-kinded.
`

const ArrayTypeToConstantSizedFunctionName = "toConstantSized"

const arrayTypeToConstantSizedFunctionDocString = `
Returns a new constant-sized array with the copy of the contents of the given array.
Available if the array is variable-sized and the element type is not resource-kinded.
`

var insertMutateEntitledAccess = NewEntitlementSetAccess(
	[]*EntitlementType{
		InsertType,
		MutateType,
	},
	Disjunction,
)

var removeMutateEntitledAccess = NewEntitlementSetAccess(
	[]*EntitlementType{
		RemoveType,
		MutateType,
	},
	Disjunction,
)

const ArrayTypeFilterFunctionName = "filter"

const arrayTypeFilterFunctionDocString = `
Returns a new array whose elements are filtered by applying the filter function on each element of the original array.
Available if the array element type is not resource-kinded.
`

const ArrayTypeMapFunctionName = "map"

const arrayTypeMapFunctionDocString = `
Returns a new array whose elements are produced by applying the mapper function on each element of the original array.
`

func getArrayMembers(arrayType ArrayType) map[string]MemberResolver {

	members := map[string]MemberResolver{
		"contains": {
			Kind: common.DeclarationKindFunction,
			Resolve: func(
				memoryGauge common.MemoryGauge,
				identifier string,
				targetRange ast.HasPosition,
				report func(error),
			) *Member {

				elementType := arrayType.ElementType(false)

				// It is impossible for an array of resources to have a `contains` function:
				// if the resource is passed as an argument, it cannot be inside the array

				if elementType.IsResourceType() {
					report(
						&InvalidResourceArrayMemberError{
							Name:            identifier,
							DeclarationKind: common.DeclarationKindFunction,
							Range:           ast.NewRangeFromPositioned(memoryGauge, targetRange),
						},
					)
				}

				// TODO: implement Equatable interface: https://github.com/dapperlabs/bamboo-node/issues/78

				if !elementType.IsEquatable() {
					report(
						&NotEquatableTypeError{
							Type:  elementType,
							Range: ast.NewRangeFromPositioned(memoryGauge, targetRange),
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
			Resolve: func(memoryGauge common.MemoryGauge, identifier string, _ ast.HasPosition, _ func(error)) *Member {
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
			Resolve: func(
				memoryGauge common.MemoryGauge,
				identifier string,
				targetRange ast.HasPosition,
				report func(error),
			) *Member {

				elementType := arrayType.ElementType(false)

				// It is impossible for an array of resources to have a `firstIndex` function:
				// if the resource is passed as an argument, it cannot be inside the array

				if elementType.IsResourceType() {
					report(
						&InvalidResourceArrayMemberError{
							Name:            identifier,
							DeclarationKind: common.DeclarationKindFunction,
							Range:           ast.NewRangeFromPositioned(memoryGauge, targetRange),
						},
					)
				}

				// TODO: implement Equatable interface

				if !elementType.IsEquatable() {
					report(
						&NotEquatableTypeError{
							Type:  elementType,
							Range: ast.NewRangeFromPositioned(memoryGauge, targetRange),
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
		ArrayTypeReverseFunctionName: {
			Kind: common.DeclarationKindFunction,
			Resolve: func(
				memoryGauge common.MemoryGauge,
				identifier string,
				targetRange ast.HasPosition,
				report func(error),
			) *Member {
				elementType := arrayType.ElementType(false)

				// It is impossible for a resource to be present in two arrays.
				if elementType.IsResourceType() {
					report(
						&InvalidResourceArrayMemberError{
							Name:            identifier,
							DeclarationKind: common.DeclarationKindFunction,
							Range:           ast.NewRangeFromPositioned(memoryGauge, targetRange),
						},
					)
				}

				return NewPublicFunctionMember(
					memoryGauge,
					arrayType,
					identifier,
					ArrayReverseFunctionType(arrayType),
					arrayTypeReverseFunctionDocString,
				)
			},
		},
		ArrayTypeFilterFunctionName: {
			Kind: common.DeclarationKindFunction,
			Resolve: func(
				memoryGauge common.MemoryGauge,
				identifier string,
				targetRange ast.HasPosition,
				report func(error),
			) *Member {

				elementType := arrayType.ElementType(false)

				if elementType.IsResourceType() {
					report(
						&InvalidResourceArrayMemberError{
							Name:            identifier,
							DeclarationKind: common.DeclarationKindFunction,
							Range:           ast.NewRangeFromPositioned(memoryGauge, targetRange),
						},
					)
				}

				return NewPublicFunctionMember(
					memoryGauge,
					arrayType,
					identifier,
					ArrayFilterFunctionType(memoryGauge, elementType),
					arrayTypeFilterFunctionDocString,
				)
			},
		},
		ArrayTypeMapFunctionName: {
			Kind: common.DeclarationKindFunction,
			Resolve: func(
				memoryGauge common.MemoryGauge,
				identifier string,
				targetRange ast.HasPosition,
				report func(error),
			) *Member {
				elementType := arrayType.ElementType(false)

				// TODO: maybe allow for resource element type as a reference.
				if elementType.IsResourceType() {
					report(
						&InvalidResourceArrayMemberError{
							Name:            identifier,
							DeclarationKind: common.DeclarationKindFunction,
							Range:           ast.NewRangeFromPositioned(memoryGauge, targetRange),
						},
					)
				}

				return NewPublicFunctionMember(
					memoryGauge,
					arrayType,
					identifier,
					ArrayMapFunctionType(memoryGauge, arrayType),
					arrayTypeMapFunctionDocString,
				)
			},
		},
	}

	// TODO: maybe still return members but report a helpful error?

	if _, ok := arrayType.(*VariableSizedType); ok {

		members["append"] = MemberResolver{
			Kind: common.DeclarationKindFunction,
			Resolve: func(
				memoryGauge common.MemoryGauge,
				identifier string,
				targetRange ast.HasPosition,
				report func(error),
			) *Member {
				elementType := arrayType.ElementType(false)
				return NewFunctionMember(
					memoryGauge,
					arrayType,
					insertMutateEntitledAccess,
					identifier,
					ArrayAppendFunctionType(elementType),
					arrayTypeAppendFunctionDocString,
				)
			},
		}

		members["appendAll"] = MemberResolver{
			Kind: common.DeclarationKindFunction,
			Resolve: func(
				memoryGauge common.MemoryGauge,
				identifier string,
				targetRange ast.HasPosition,
				report func(error),
			) *Member {

				elementType := arrayType.ElementType(false)

				if elementType.IsResourceType() {
					report(
						&InvalidResourceArrayMemberError{
							Name:            identifier,
							DeclarationKind: common.DeclarationKindFunction,
							Range:           ast.NewRangeFromPositioned(memoryGauge, targetRange),
						},
					)
				}

				return NewFunctionMember(
					memoryGauge,
					arrayType,
					insertMutateEntitledAccess,
					identifier,
					ArrayAppendAllFunctionType(arrayType),
					arrayTypeAppendAllFunctionDocString,
				)
			},
		}

		members["concat"] = MemberResolver{
			Kind: common.DeclarationKindFunction,
			Resolve: func(
				memoryGauge common.MemoryGauge,
				identifier string,
				targetRange ast.HasPosition,
				report func(error),
			) *Member {

				// TODO: maybe allow for resource element type

				elementType := arrayType.ElementType(false)

				if elementType.IsResourceType() {
					report(
						&InvalidResourceArrayMemberError{
							Name:            identifier,
							DeclarationKind: common.DeclarationKindFunction,
							Range:           ast.NewRangeFromPositioned(memoryGauge, targetRange),
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
			Resolve: func(
				memoryGauge common.MemoryGauge,
				identifier string,
				targetRange ast.HasPosition,
				report func(error),
			) *Member {

				elementType := arrayType.ElementType(false)

				if elementType.IsResourceType() {
					report(
						&InvalidResourceArrayMemberError{
							Name:            identifier,
							DeclarationKind: common.DeclarationKindFunction,
							Range:           ast.NewRangeFromPositioned(memoryGauge, targetRange),
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
			Kind: common.DeclarationKindFunction,
			Resolve: func(
				memoryGauge common.MemoryGauge,
				identifier string,
				_ ast.HasPosition,
				_ func(error),
			) *Member {

				elementType := arrayType.ElementType(false)

				return NewFunctionMember(
					memoryGauge,
					arrayType,
					insertMutateEntitledAccess,
					identifier,
					ArrayInsertFunctionType(elementType),
					arrayTypeInsertFunctionDocString,
				)
			},
		}

		members["remove"] = MemberResolver{
			Kind: common.DeclarationKindFunction,
			Resolve: func(
				memoryGauge common.MemoryGauge,
				identifier string,
				_ ast.HasPosition,
				_ func(error),
			) *Member {

				elementType := arrayType.ElementType(false)

				return NewFunctionMember(
					memoryGauge,
					arrayType,
					removeMutateEntitledAccess,
					identifier,
					ArrayRemoveFunctionType(elementType),
					arrayTypeRemoveFunctionDocString,
				)
			},
		}

		members["removeFirst"] = MemberResolver{
			Kind: common.DeclarationKindFunction,
			Resolve: func(
				memoryGauge common.MemoryGauge,
				identifier string,
				_ ast.HasPosition,
				_ func(error),
			) *Member {

				elementType := arrayType.ElementType(false)

				return NewFunctionMember(
					memoryGauge,
					arrayType,
					removeMutateEntitledAccess,
					identifier,
					ArrayRemoveFirstFunctionType(elementType),
					arrayTypeRemoveFirstFunctionDocString,
				)
			},
		}

		members["removeLast"] = MemberResolver{
			Kind: common.DeclarationKindFunction,
			Resolve: func(
				memoryGauge common.MemoryGauge,
				identifier string,
				_ ast.HasPosition,
				_ func(error),
			) *Member {

				elementType := arrayType.ElementType(false)

				return NewFunctionMember(
					memoryGauge,
					arrayType,
					removeMutateEntitledAccess,
					identifier,
					ArrayRemoveLastFunctionType(elementType),
					arrayTypeRemoveLastFunctionDocString,
				)
			},
		}

		members[ArrayTypeToConstantSizedFunctionName] = MemberResolver{
			Kind: common.DeclarationKindFunction,
			Resolve: func(
				memoryGauge common.MemoryGauge,
				identifier string,
				targetRange ast.HasPosition,
				report func(error),
			) *Member {
				elementType := arrayType.ElementType(false)

				if elementType.IsResourceType() {
					report(
						&InvalidResourceArrayMemberError{
							Name:            identifier,
							DeclarationKind: common.DeclarationKindFunction,
							Range:           ast.NewRangeFromPositioned(memoryGauge, targetRange),
						},
					)
				}

				return NewPublicFunctionMember(
					memoryGauge,
					arrayType,
					identifier,
					ArrayToConstantSizedFunctionType(elementType),
					arrayTypeToConstantSizedFunctionDocString,
				)
			},
		}
	}

	if _, ok := arrayType.(*ConstantSizedType); ok {

		members[ArrayTypeToVariableSizedFunctionName] = MemberResolver{
			Kind: common.DeclarationKindFunction,
			Resolve: func(
				memoryGauge common.MemoryGauge,
				identifier string,
				targetRange ast.HasPosition,
				report func(error),
			) *Member {
				elementType := arrayType.ElementType(false)

				if elementType.IsResourceType() {
					report(
						&InvalidResourceArrayMemberError{
							Name:            identifier,
							DeclarationKind: common.DeclarationKindFunction,
							Range:           ast.NewRangeFromPositioned(memoryGauge, targetRange),
						},
					)
				}

				return NewPublicFunctionMember(
					memoryGauge,
					arrayType,
					identifier,
					ArrayToVariableSizedFunctionType(elementType),
					arrayTypeToVariableSizedFunctionDocString,
				)
			},
		}
	}

	return withBuiltinMembers(arrayType, members)
}

func ArrayRemoveLastFunctionType(elementType Type) *FunctionType {
	return NewSimpleFunctionType(
		FunctionPurityImpure,
		nil,
		NewTypeAnnotation(elementType),
	)
}

func ArrayRemoveFirstFunctionType(elementType Type) *FunctionType {
	return NewSimpleFunctionType(
		FunctionPurityImpure,
		nil,
		NewTypeAnnotation(elementType),
	)
}

func ArrayRemoveFunctionType(elementType Type) *FunctionType {
	return NewSimpleFunctionType(
		FunctionPurityImpure,
		[]Parameter{
			{
				Identifier:     "at",
				TypeAnnotation: IntegerTypeAnnotation,
			},
		},
		NewTypeAnnotation(elementType),
	)
}

func ArrayInsertFunctionType(elementType Type) *FunctionType {
	return NewSimpleFunctionType(
		FunctionPurityImpure,
		[]Parameter{
			{
				Identifier:     "at",
				TypeAnnotation: IntegerTypeAnnotation,
			},
			{
				Label:          ArgumentLabelNotRequired,
				Identifier:     "element",
				TypeAnnotation: NewTypeAnnotation(elementType),
			},
		},
		VoidTypeAnnotation,
	)
}

func ArrayConcatFunctionType(arrayType Type) *FunctionType {
	typeAnnotation := NewTypeAnnotation(arrayType)
	return NewSimpleFunctionType(
		FunctionPurityView,
		[]Parameter{
			{
				Label:          ArgumentLabelNotRequired,
				Identifier:     "other",
				TypeAnnotation: typeAnnotation,
			},
		},
		typeAnnotation,
	)
}

func ArrayFirstIndexFunctionType(elementType Type) *FunctionType {
	return NewSimpleFunctionType(
		FunctionPurityView,
		[]Parameter{
			{
				Identifier:     "of",
				TypeAnnotation: NewTypeAnnotation(elementType),
			},
		},
		NewTypeAnnotation(
			&OptionalType{Type: IntType},
		),
	)
}
func ArrayContainsFunctionType(elementType Type) *FunctionType {
	return NewSimpleFunctionType(
		FunctionPurityView,
		[]Parameter{
			{
				Label:          ArgumentLabelNotRequired,
				Identifier:     "element",
				TypeAnnotation: NewTypeAnnotation(elementType),
			},
		},
		BoolTypeAnnotation,
	)
}

func ArrayAppendAllFunctionType(arrayType Type) *FunctionType {
	return NewSimpleFunctionType(
		FunctionPurityImpure,
		[]Parameter{
			{
				Label:          ArgumentLabelNotRequired,
				Identifier:     "other",
				TypeAnnotation: NewTypeAnnotation(arrayType),
			},
		},
		VoidTypeAnnotation,
	)
}

func ArrayAppendFunctionType(elementType Type) *FunctionType {
	return NewSimpleFunctionType(
		FunctionPurityImpure,
		[]Parameter{
			{
				Label:          ArgumentLabelNotRequired,
				Identifier:     "element",
				TypeAnnotation: NewTypeAnnotation(elementType),
			},
		},
		VoidTypeAnnotation,
	)
}

func ArraySliceFunctionType(elementType Type) *FunctionType {
	return NewSimpleFunctionType(
		FunctionPurityView,
		[]Parameter{
			{
				Identifier:     "from",
				TypeAnnotation: IntTypeAnnotation,
			},
			{
				Identifier:     "upTo",
				TypeAnnotation: IntTypeAnnotation,
			},
		},
		NewTypeAnnotation(&VariableSizedType{
			Type: elementType,
		}),
	)
}

func ArrayToVariableSizedFunctionType(elementType Type) *FunctionType {
	return NewSimpleFunctionType(
		FunctionPurityView,
		[]Parameter{},
		NewTypeAnnotation(&VariableSizedType{
			Type: elementType,
		}),
	)
}

func ArrayToConstantSizedFunctionType(elementType Type) *FunctionType {
	// Ideally this should have a typebound of [T; _] but since we don't know
	// the size of the ConstantSizedArray, we omit specifying the bound.
	typeParameter := &TypeParameter{
		Name: "T",
	}

	typeAnnotation := NewTypeAnnotation(
		&GenericType{
			TypeParameter: typeParameter,
		},
	)

	return &FunctionType{
		Purity: FunctionPurityView,
		TypeParameters: []*TypeParameter{
			typeParameter,
		},
		ReturnTypeAnnotation: NewTypeAnnotation(
			&OptionalType{
				Type: typeAnnotation.Type,
			},
		),
		TypeArgumentsCheck: func(
			memoryGauge common.MemoryGauge,
			typeArguments *TypeParameterTypeOrderedMap,
			astTypeArguments []*ast.TypeAnnotation,
			invocationRange ast.HasPosition,
			report func(error),
		) {
			typeArg, ok := typeArguments.Get(typeParameter)
			if !ok || typeArg == nil {
				// Invalid, already reported by checker
				return
			}

			constArrayType, ok := typeArg.(*ConstantSizedType)
			if !ok || constArrayType.Type != elementType {
				errorRange := invocationRange
				if len(astTypeArguments) > 0 {
					errorRange = astTypeArguments[0]
				}

				report(&InvalidTypeArgumentError{
					TypeArgumentName: typeParameter.Name,
					Range:            ast.NewRangeFromPositioned(memoryGauge, errorRange),
					Details: fmt.Sprintf(
						"Type argument for %s must be [%s; _]",
						ArrayTypeToConstantSizedFunctionName,
						elementType,
					),
				})
			}
		},
	}
}

func ArrayReverseFunctionType(arrayType ArrayType) *FunctionType {
	return &FunctionType{
		Parameters:           []Parameter{},
		ReturnTypeAnnotation: NewTypeAnnotation(arrayType),
		Purity:               FunctionPurityView,
	}
}

func ArrayFilterFunctionType(memoryGauge common.MemoryGauge, elementType Type) *FunctionType {
	// fun filter(_ function: ((T): Bool)): [T]
	// funcType: elementType -> Bool
	funcType := &FunctionType{
		Parameters: []Parameter{
			{
				Identifier:     "element",
				TypeAnnotation: NewTypeAnnotation(elementType),
			},
		},
		ReturnTypeAnnotation: NewTypeAnnotation(BoolType),
		Purity:               FunctionPurityView,
	}

	return &FunctionType{
		Parameters: []Parameter{
			{
				Label:          ArgumentLabelNotRequired,
				Identifier:     "f",
				TypeAnnotation: NewTypeAnnotation(funcType),
			},
		},
		ReturnTypeAnnotation: NewTypeAnnotation(NewVariableSizedType(memoryGauge, elementType)),
		Purity:               FunctionPurityView,
	}
}

func ArrayMapFunctionType(memoryGauge common.MemoryGauge, arrayType ArrayType) *FunctionType {
	// For [T] or [T; N]
	// fun map(_ function: ((T): U)): [U]
	//               or
	// fun map(_ function: ((T): U)): [U; N]

	typeParameter := &TypeParameter{
		Name: "U",
	}

	typeU := &GenericType{
		TypeParameter: typeParameter,
	}

	var returnArrayType Type
	switch arrayType := arrayType.(type) {
	case *VariableSizedType:
		returnArrayType = NewVariableSizedType(memoryGauge, typeU)
	case *ConstantSizedType:
		returnArrayType = NewConstantSizedType(memoryGauge, typeU, arrayType.Size)
	default:
		panic(errors.NewUnreachableError())
	}

	// transformFuncType: elementType -> U
	transformFuncType := &FunctionType{
		Parameters: []Parameter{
			{
				Identifier:     "element",
				TypeAnnotation: NewTypeAnnotation(arrayType.ElementType(false)),
			},
		},
		ReturnTypeAnnotation: NewTypeAnnotation(typeU),
	}

	return &FunctionType{
		TypeParameters: []*TypeParameter{
			typeParameter,
		},
		Parameters: []Parameter{
			{
				Label:          ArgumentLabelNotRequired,
				Identifier:     "transform",
				TypeAnnotation: NewTypeAnnotation(transformFuncType),
			},
		},
		ReturnTypeAnnotation: NewTypeAnnotation(returnArrayType),
	}
}

// VariableSizedType is a variable sized array type
type VariableSizedType struct {
	Type                Type
	memberResolvers     map[string]MemberResolver
	memberResolversOnce sync.Once
}

var _ Type = &VariableSizedType{}
var _ ArrayType = &VariableSizedType{}
var _ ValueIndexableType = &VariableSizedType{}
var _ EntitlementSupportingType = &VariableSizedType{}

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

func FormatVariableSizedTypeID[T ~string](elementTypeID T) T {
	return T(fmt.Sprintf("[%s]", elementTypeID))
}

func (t *VariableSizedType) ID() TypeID {
	return FormatVariableSizedTypeID(t.Type.ID())
}

func (t *VariableSizedType) Equal(other Type) bool {
	otherArray, ok := other.(*VariableSizedType)
	if !ok {
		return false
	}

	return t.Type.Equal(otherArray.Type)
}

func (t *VariableSizedType) Map(gauge common.MemoryGauge, typeParamMap map[*TypeParameter]*TypeParameter, f func(Type) Type) Type {
	return f(NewVariableSizedType(gauge, t.Type.Map(gauge, typeParamMap, f)))
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

func (t *VariableSizedType) IsPrimitiveType() bool {
	return false
}

func (t *VariableSizedType) IsInvalidType() bool {
	return t.Type.IsInvalidType()
}

func (t *VariableSizedType) IsOrContainsReferenceType() bool {
	return t.Type.IsOrContainsReferenceType()
}

func (t *VariableSizedType) IsStorable(results map[*Member]bool) bool {
	return t.Type.IsStorable(results)
}

func (t *VariableSizedType) IsExportable(results map[*Member]bool) bool {
	return t.Type.IsExportable(results)
}

func (t *VariableSizedType) IsImportable(results map[*Member]bool) bool {
	return t.Type.IsImportable(results)
}

func (t *VariableSizedType) IsEquatable() bool {
	return t.Type.IsEquatable()
}

func (t *VariableSizedType) IsComparable() bool {
	return t.Type.IsComparable()
}

func (t *VariableSizedType) ContainFieldsOrElements() bool {
	return true
}

func (t *VariableSizedType) TypeAnnotationState() TypeAnnotationState {
	return t.Type.TypeAnnotationState()
}

func (t *VariableSizedType) RewriteWithIntersectionTypes() (Type, bool) {
	rewrittenType, rewritten := t.Type.RewriteWithIntersectionTypes()
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
	memoryGauge common.MemoryGauge,
	outerRange ast.HasPosition,
) bool {

	otherArray, ok := other.(*VariableSizedType)
	if !ok {
		return false
	}

	return t.Type.Unify(
		otherArray.Type,
		typeParameters,
		report,
		memoryGauge,
		outerRange,
	)
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

func (t *VariableSizedType) SupportedEntitlements() *EntitlementSet {
	return arrayDictionaryEntitlements
}

var arrayDictionaryEntitlements = func() *EntitlementSet {
	set := &EntitlementSet{}
	set.Add(MutateType)
	set.Add(InsertType)
	set.Add(RemoveType)
	return set
}()

func (t *VariableSizedType) CheckInstantiated(pos ast.HasPosition, memoryGauge common.MemoryGauge, report func(err error)) {
	t.ElementType(false).CheckInstantiated(pos, memoryGauge, report)
}

// ConstantSizedType is a constant sized array type
type ConstantSizedType struct {
	Type                Type
	memberResolvers     map[string]MemberResolver
	Size                int64
	memberResolversOnce sync.Once
}

var _ Type = &ConstantSizedType{}
var _ ArrayType = &ConstantSizedType{}
var _ ValueIndexableType = &ConstantSizedType{}
var _ EntitlementSupportingType = &ConstantSizedType{}

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

func FormatConstantSizedTypeID[T ~string](elementTypeID T, size int64) T {
	return T(fmt.Sprintf("[%s;%d]", elementTypeID, size))
}

func (t *ConstantSizedType) ID() TypeID {
	return FormatConstantSizedTypeID(t.Type.ID(), t.Size)
}

func (t *ConstantSizedType) Equal(other Type) bool {
	otherArray, ok := other.(*ConstantSizedType)
	if !ok {
		return false
	}

	return t.Type.Equal(otherArray.Type) &&
		t.Size == otherArray.Size
}

func (t *ConstantSizedType) Map(gauge common.MemoryGauge, typeParamMap map[*TypeParameter]*TypeParameter, f func(Type) Type) Type {
	return f(NewConstantSizedType(gauge, t.Type.Map(gauge, typeParamMap, f), t.Size))
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

func (t *ConstantSizedType) IsPrimitiveType() bool {
	return false
}

func (t *ConstantSizedType) IsInvalidType() bool {
	return t.Type.IsInvalidType()
}

func (t *ConstantSizedType) IsOrContainsReferenceType() bool {
	return t.Type.IsOrContainsReferenceType()
}

func (t *ConstantSizedType) IsStorable(results map[*Member]bool) bool {
	return t.Type.IsStorable(results)
}

func (t *ConstantSizedType) IsExportable(results map[*Member]bool) bool {
	return t.Type.IsStorable(results)
}

func (t *ConstantSizedType) IsImportable(results map[*Member]bool) bool {
	return t.Type.IsImportable(results)
}

func (t *ConstantSizedType) IsEquatable() bool {
	return t.Type.IsEquatable()
}

func (t *ConstantSizedType) IsComparable() bool {
	return t.Type.IsComparable()
}

func (t *ConstantSizedType) ContainFieldsOrElements() bool {
	return true
}

func (t *ConstantSizedType) TypeAnnotationState() TypeAnnotationState {
	return t.Type.TypeAnnotationState()
}

func (t *ConstantSizedType) RewriteWithIntersectionTypes() (Type, bool) {
	rewrittenType, rewritten := t.Type.RewriteWithIntersectionTypes()
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
	memoryGauge common.MemoryGauge,
	outerRange ast.HasPosition,
) bool {

	otherArray, ok := other.(*ConstantSizedType)
	if !ok {
		return false
	}

	if t.Size != otherArray.Size {
		return false
	}

	return t.Type.Unify(
		otherArray.Type,
		typeParameters,
		report,
		memoryGauge,
		outerRange,
	)
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

func (t *ConstantSizedType) SupportedEntitlements() *EntitlementSet {
	return arrayDictionaryEntitlements
}

func (t *ConstantSizedType) CheckInstantiated(pos ast.HasPosition, memoryGauge common.MemoryGauge, report func(err error)) {
	t.ElementType(false).CheckInstantiated(pos, memoryGauge, report)
}

// Parameter

func formatParameter(spaces bool, label, identifier, typeAnnotation string) string {
	var builder strings.Builder

	if label != "" {
		builder.WriteString(label)
		if spaces {
			builder.WriteByte(' ')
		}
	}

	if identifier != "" {
		builder.WriteString(identifier)
		builder.WriteByte(':')
		if spaces {
			builder.WriteByte(' ')
		}
	}

	builder.WriteString(typeAnnotation)

	return builder.String()
}

type Parameter struct {
	TypeAnnotation  TypeAnnotation
	DefaultArgument Type
	Label           string
	Identifier      string
}

func (p Parameter) String() string {
	return formatParameter(
		true,
		p.Label,
		p.Identifier,
		p.TypeAnnotation.String(),
	)
}

func (p Parameter) QualifiedString() string {
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
func (p Parameter) EffectiveArgumentLabel() string {
	if p.Label != "" {
		return p.Label
	}
	return p.Identifier
}

// TypeParameter

type TypeParameter struct {
	TypeBound Type
	Name      string
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

func (p TypeParameter) checkTypeBound(ty Type, memoryGauge common.MemoryGauge, typeRange ast.HasPosition) error {
	if p.TypeBound == nil ||
		p.TypeBound.IsInvalidType() ||
		ty.IsInvalidType() {

		return nil
	}

	if !IsSubType(ty, p.TypeBound) {
		return &TypeMismatchError{
			ExpectedType: p.TypeBound,
			ActualType:   ty,
			Range:        ast.NewRangeFromPositioned(memoryGauge, typeRange),
		}
	}

	return nil
}

// Function types

func formatFunctionType(
	separator string,
	purity string,
	functionName string,
	typeParameters []string,
	parameters []string,
	returnTypeAnnotation string,
) string {

	var builder strings.Builder

	if len(purity) > 0 {
		builder.WriteString(purity)
		builder.WriteByte(' ')
	}

	builder.WriteString("fun")

	if functionName != "" {
		builder.WriteByte(' ')
		builder.WriteString(functionName)
	}

	if len(typeParameters) > 0 {
		builder.WriteByte('<')
		for i, typeParameter := range typeParameters {
			if i > 0 {
				builder.WriteByte(',')
				builder.WriteString(separator)
			}
			builder.WriteString(typeParameter)
		}
		builder.WriteByte('>')
	}
	builder.WriteByte('(')
	for i, parameter := range parameters {
		if i > 0 {
			builder.WriteByte(',')
			builder.WriteString(separator)
		}
		builder.WriteString(parameter)
	}
	builder.WriteString("):")
	builder.WriteString(separator)
	builder.WriteString(returnTypeAnnotation)
	return builder.String()
}

// Arity

type Arity struct {
	Min int
	Max int
}

func (arity *Arity) MinCount(parameterCount int) int {
	minCount := parameterCount
	if arity != nil {
		minCount = arity.Min
	}

	return minCount
}

func (arity *Arity) MaxCount(parameterCount int) *int {
	maxCount := parameterCount
	if arity != nil {
		if arity.Max < parameterCount {
			return nil
		}
		maxCount = arity.Max
	}

	return &maxCount
}

type FunctionPurity int

const (
	FunctionPurityImpure = iota
	FunctionPurityView
)

func (p FunctionPurity) String() string {
	if p == FunctionPurityImpure {
		return ""
	}
	return "view"
}

// FunctionType

type FunctionType struct {
	Purity                   FunctionPurity
	ReturnTypeAnnotation     TypeAnnotation
	Arity                    *Arity
	ArgumentExpressionsCheck ArgumentExpressionsCheck
	TypeArgumentsCheck       TypeArgumentsCheck
	Members                  *StringMemberOrderedMap
	TypeParameters           []*TypeParameter
	Parameters               []Parameter
	memberResolvers          map[string]MemberResolver
	memberResolversOnce      sync.Once
	IsConstructor            bool
}

func NewSimpleFunctionType(
	purity FunctionPurity,
	parameters []Parameter,
	returnTypeAnnotation TypeAnnotation,
) *FunctionType {
	return &FunctionType{
		Purity:               purity,
		Parameters:           parameters,
		ReturnTypeAnnotation: returnTypeAnnotation,
	}
}

var _ Type = &FunctionType{}

func (*FunctionType) IsType() {}

func (t *FunctionType) Tag() TypeTag {
	return FunctionTypeTag
}

func (t *FunctionType) string(
	typeParameterFormatter func(*TypeParameter) string,
	functionName string,
	parameterFormatter func(Parameter) string,
	returnTypeAnnotationFormatter func(TypeAnnotation) string,
) string {

	purity := t.Purity.String()

	var typeParameters []string
	typeParameterCount := len(t.TypeParameters)
	if typeParameterCount > 0 {
		typeParameters = make([]string, typeParameterCount)
		for i, typeParameter := range t.TypeParameters {
			typeParameters[i] = typeParameterFormatter(typeParameter)
		}
	}

	var parameters []string
	parameterCount := len(t.Parameters)
	if parameterCount > 0 {
		parameters = make([]string, parameterCount)
		for i, parameter := range t.Parameters {
			parameters[i] = parameterFormatter(parameter)
		}
	}

	returnTypeAnnotation := returnTypeAnnotationFormatter(t.ReturnTypeAnnotation)

	return formatFunctionType(
		" ",
		purity,
		functionName,
		typeParameters,
		parameters,
		returnTypeAnnotation,
	)
}

func FormatFunctionTypeID(
	purity string,
	typeParameters []string,
	parameters []string,
	returnTypeAnnotation string,
) string {
	return formatFunctionType(
		"",
		purity,
		"",
		typeParameters,
		parameters,
		returnTypeAnnotation,
	)
}

func (t *FunctionType) String() string {
	return t.string(
		func(parameter *TypeParameter) string {
			return parameter.String()
		},
		"",
		func(parameter Parameter) string {
			return parameter.String()
		},
		func(typeAnnotation TypeAnnotation) string {
			return typeAnnotation.String()
		},
	)
}

func (t *FunctionType) QualifiedString() string {
	return t.NamedQualifiedString("")
}

func (t *FunctionType) NamedQualifiedString(functionName string) string {
	return t.string(
		func(parameter *TypeParameter) string {
			return parameter.QualifiedString()
		},
		functionName,
		func(parameter Parameter) string {
			return parameter.QualifiedString()
		},
		func(typeAnnotation TypeAnnotation) string {
			return typeAnnotation.QualifiedString()
		},
	)
}

// NOTE: parameter names and argument labels are *not* part of the ID!
func (t *FunctionType) ID() TypeID {

	purity := t.Purity.String()

	typeParameterCount := len(t.TypeParameters)
	var typeParameters []string
	if typeParameterCount > 0 {
		typeParameters = make([]string, typeParameterCount)
		for i, typeParameter := range t.TypeParameters {
			typeParameters[i] = typeParameter.Name
		}
	}

	parameterCount := len(t.Parameters)
	var parameters []string
	if parameterCount > 0 {
		parameters = make([]string, parameterCount)
		for i, parameter := range t.Parameters {
			parameters[i] = string(parameter.TypeAnnotation.Type.ID())
		}
	}

	returnTypeAnnotation := string(t.ReturnTypeAnnotation.Type.ID())

	return TypeID(
		FormatFunctionTypeID(
			purity,
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

	if t.Purity != otherFunction.Purity {
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

func (t *FunctionType) IsPrimitiveType() bool {
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

func (*FunctionType) IsOrContainsReferenceType() bool {
	return false
}

func (t *FunctionType) IsStorable(_ map[*Member]bool) bool {
	// Functions cannot be stored, as they cannot be serialized
	return false
}

func (t *FunctionType) IsExportable(_ map[*Member]bool) bool {
	// Even though functions cannot be serialized,
	// they are still treated as exportable,
	// as values are simply omitted.
	return true
}

func (t *FunctionType) IsImportable(_ map[*Member]bool) bool {
	return false
}

func (*FunctionType) IsEquatable() bool {
	return false
}

func (*FunctionType) IsComparable() bool {
	return false
}

func (*FunctionType) ContainFieldsOrElements() bool {
	return false
}

func (t *FunctionType) TypeAnnotationState() TypeAnnotationState {

	for _, typeParameter := range t.TypeParameters {
		TypeParameterTypeAnnotationState := typeParameter.TypeBound.TypeAnnotationState()
		if TypeParameterTypeAnnotationState != TypeAnnotationStateValid {
			return TypeParameterTypeAnnotationState
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

func (t *FunctionType) RewriteWithIntersectionTypes() (Type, bool) {
	anyRewritten := false

	rewrittenTypeParameterTypeBounds := map[*TypeParameter]Type{}

	for _, typeParameter := range t.TypeParameters {
		if typeParameter.TypeBound == nil {
			continue
		}

		rewrittenType, rewritten := typeParameter.TypeBound.RewriteWithIntersectionTypes()
		if rewritten {
			anyRewritten = true
			rewrittenTypeParameterTypeBounds[typeParameter] = rewrittenType
		}
	}

	rewrittenParameterTypes := map[*Parameter]Type{}

	for i := range t.Parameters {
		parameter := &t.Parameters[i]
		rewrittenType, rewritten := parameter.TypeAnnotation.Type.RewriteWithIntersectionTypes()
		if rewritten {
			anyRewritten = true
			rewrittenParameterTypes[parameter] = rewrittenType
		}
	}

	rewrittenReturnType, rewritten := t.ReturnTypeAnnotation.Type.RewriteWithIntersectionTypes()
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

		var rewrittenParameters []Parameter
		if len(t.Parameters) > 0 {
			rewrittenParameters = make([]Parameter, len(t.Parameters))
			for i := range t.Parameters {
				parameter := &t.Parameters[i]
				rewrittenParameterType, ok := rewrittenParameterTypes[parameter]
				if ok {
					rewrittenParameters[i] = Parameter{
						Label:          parameter.Label,
						Identifier:     parameter.Identifier,
						TypeAnnotation: NewTypeAnnotation(rewrittenParameterType),
					}
				} else {
					rewrittenParameters[i] = *parameter
				}
			}
		}

		return &FunctionType{
			Purity:               t.Purity,
			TypeParameters:       rewrittenTypeParameters,
			Parameters:           rewrittenParameters,
			ReturnTypeAnnotation: NewTypeAnnotation(rewrittenReturnType),
			Arity:                t.Arity,
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
	memoryGauge common.MemoryGauge,
	outerRange ast.HasPosition,
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
			memoryGauge,
			outerRange,
		)
		result = result || parameterUnified
	}

	// return type

	returnTypeUnified := t.ReturnTypeAnnotation.Type.Unify(
		otherFunction.ReturnTypeAnnotation.Type,
		typeParameters,
		report,
		memoryGauge,
		outerRange,
	)

	result = result || returnTypeUnified

	return
}

func (t *FunctionType) Resolve(typeArguments *TypeParameterTypeOrderedMap) Type {

	// TODO: type parameters ?

	// parameters

	var newParameters []Parameter

	if len(t.Parameters) > 0 {
		newParameters = make([]Parameter, 0, len(t.Parameters))

		for _, parameter := range t.Parameters {
			newParameterType := parameter.TypeAnnotation.Type.Resolve(typeArguments)
			if newParameterType == nil {
				return nil
			}

			newParameters = append(
				newParameters,
				Parameter{
					Label:          parameter.Label,
					Identifier:     parameter.Identifier,
					TypeAnnotation: NewTypeAnnotation(newParameterType),
				},
			)
		}
	}

	// return type

	newReturnType := t.ReturnTypeAnnotation.Type.Resolve(typeArguments)
	if newReturnType == nil {
		return nil
	}

	return &FunctionType{
		Purity:               t.Purity,
		Parameters:           newParameters,
		ReturnTypeAnnotation: NewTypeAnnotation(newReturnType),
		Arity:                t.Arity,
	}

}

func (t *FunctionType) Map(gauge common.MemoryGauge, typeParamMap map[*TypeParameter]*TypeParameter, f func(Type) Type) Type {

	var newTypeParameters []*TypeParameter

	if len(t.TypeParameters) > 0 {
		newTypeParameters = make([]*TypeParameter, 0, len(t.TypeParameters))
		for _, parameter := range t.TypeParameters {

			if param, ok := typeParamMap[parameter]; ok {
				newTypeParameters = append(newTypeParameters, param)
				continue
			}

			newTypeParameterTypeBound := parameter.TypeBound.Map(gauge, typeParamMap, f)
			newParam := &TypeParameter{
				Name:      parameter.Name,
				Optional:  parameter.Optional,
				TypeBound: newTypeParameterTypeBound,
			}
			typeParamMap[parameter] = newParam

			newTypeParameters = append(
				newTypeParameters,
				newParam,
			)
		}
	}

	var newParameters []Parameter

	if len(t.Parameters) > 0 {
		newParameters = make([]Parameter, 0, len(t.Parameters))
		for _, parameter := range t.Parameters {
			newParameterTypeAnnot := parameter.TypeAnnotation.Map(gauge, typeParamMap, f)

			newParameters = append(
				newParameters,
				Parameter{
					Label:          parameter.Label,
					Identifier:     parameter.Identifier,
					TypeAnnotation: newParameterTypeAnnot,
				},
			)
		}
	}

	returnType := t.ReturnTypeAnnotation.Map(gauge, typeParamMap, f)

	functionType := NewSimpleFunctionType(t.Purity, newParameters, returnType)
	functionType.TypeParameters = newTypeParameters
	return f(functionType)
}

func (t *FunctionType) GetMembers() map[string]MemberResolver {
	t.initializeMemberResolvers()
	return t.memberResolvers
}

func (t *FunctionType) initializeMemberResolvers() {
	t.memberResolversOnce.Do(func() {
		var memberResolvers map[string]MemberResolver
		if t.Members != nil {
			memberResolvers = MembersMapAsResolvers(t.Members)
		}
		t.memberResolvers = withBuiltinMembers(t, memberResolvers)
	})
}

func (t *FunctionType) CheckInstantiated(pos ast.HasPosition, memoryGauge common.MemoryGauge, report func(err error)) {
	for _, tyParam := range t.TypeParameters {
		tyParam.TypeBound.CheckInstantiated(pos, memoryGauge, report)
	}

	for _, param := range t.Parameters {
		param.TypeAnnotation.Type.CheckInstantiated(pos, memoryGauge, report)
	}

	t.ReturnTypeAnnotation.Type.CheckInstantiated(pos, memoryGauge, report)
}

type ArgumentExpressionsCheck func(
	checker *Checker,
	argumentExpressions []ast.Expression,
	invocationRange ast.HasPosition,
)

type TypeArgumentsCheck func(
	memoryGauge common.MemoryGauge,
	typeArguments *TypeParameterTypeOrderedMap,
	astTypeArguments []*ast.TypeAnnotation,
	invocationRange ast.HasPosition,
	report func(err error),
)

// BaseTypeActivation is the base activation that contains
// the types available in programs
var BaseTypeActivation = NewVariableActivation(nil)

func init() {

	types := common.Concat(
		AllNumberTypes,
		[]Type{
			MetaType,
			VoidType,
			AnyStructType,
			AnyStructAttachmentType,
			AnyResourceType,
			AnyResourceAttachmentType,
			NeverType,
			BoolType,
			CharacterType,
			StringType,
			TheAddressType,
			AccountType,
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
			StorageCapabilityControllerType,
			AccountCapabilityControllerType,
			DeploymentResultType,
			HashableStructType,
			&InclusiveRangeType{},
			StructStringerType,
		},
	)

	for _, ty := range types {
		addToBaseActivation(ty)
	}

	addToBaseActivation(IdentityType)

	// The AST contains empty type annotations, resolve them to Void

	BaseTypeActivation.Set(
		"",
		BaseTypeActivation.Find("Void"),
	)
}

func addToBaseActivation(ty Type) {
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

const IdentityMappingIdentifier string = "Identity"

// IdentityType represents the `Identity` entitlement mapping type.
// It is an empty map that includes the Identity map,
// and is considered already "resolved" with regards to its (vacuously empty) inclusions.
// defining it this way eliminates the need to do any special casing for its behavior
var IdentityType = func() *EntitlementMapType {
	m := NewEntitlementMapType(nil, nil, IdentityMappingIdentifier)
	m.IncludesIdentity = true
	m.resolveInclusions.Do(func() {})
	return m
}()

func baseTypeVariable(name string, ty Type) *Variable {
	return &Variable{
		Identifier:      name,
		Type:            ty,
		DeclarationKind: common.DeclarationKindType,
		IsConstant:      true,
		Access:          PrimitiveAccess(ast.AccessAll),
	}
}

// BaseValueActivation is the base activation that contains
// the values available in programs
var BaseValueActivation = NewVariableActivation(nil)

var AllSignedFixedPointTypes = []Type{
	Fix64Type,
}

var AllUnsignedFixedPointTypes = []Type{
	UFix64Type,
}

var AllFixedPointTypes = common.Concat(
	AllUnsignedFixedPointTypes,
	AllSignedFixedPointTypes,
	[]Type{
		FixedPointType,
		SignedFixedPointType,
	},
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

var AllFixedSizeUnsignedIntegerTypes = []Type{
	// UInt*
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
	Word128Type,
	Word256Type,
}

var AllUnsignedIntegerTypes = common.Concat(
	AllFixedSizeUnsignedIntegerTypes,
	[]Type{
		UIntType,
	},
)

var AllNonLeafIntegerTypes = []Type{
	IntegerType,
	SignedIntegerType,
	FixedSizeUnsignedIntegerType,
}

var AllIntegerTypes = common.Concat(
	AllUnsignedIntegerTypes,
	AllSignedIntegerTypes,
	AllNonLeafIntegerTypes,
)

var AllNumberTypes = common.Concat(
	AllIntegerTypes,
	AllFixedPointTypes,
	[]Type{
		NumberType,
		SignedNumberType,
	},
)

var BuiltinEntitlements = map[string]*EntitlementType{}

var BuiltinEntitlementMappings = map[string]*EntitlementMapType{
	IdentityType.QualifiedIdentifier(): IdentityType,
}

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
			IntegerType, SignedIntegerType, FixedSizeUnsignedIntegerType,
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
					functionType.Members = &StringMemberOrderedMap{}
				}
				name := member.Identifier.Identifier
				if functionType.Members.Contains(name) {
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

			// add .fromString() method
			fromStringFnType := FromStringFunctionType(numberType)
			fromStringDocstring := FromStringFunctionDocstring(numberType)
			addMember(NewUnmeteredPublicFunctionMember(
				functionType,
				FromStringFunctionName,
				fromStringFnType,
				fromStringDocstring,
			))

			// add .fromBigEndianBytes() method
			fromBigEndianBytesFnType := FromBigEndianBytesFunctionType(numberType)
			fromBigEndianBytesDocstring := FromBigEndianBytesFunctionDocstring(numberType)
			addMember(NewUnmeteredPublicFunctionMember(
				functionType,
				FromBigEndianBytesFunctionName,
				fromBigEndianBytesFnType,
				fromBigEndianBytesDocstring,
			))

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
		Purity: FunctionPurityView,
		Parameters: []Parameter{
			{
				Label:          ArgumentLabelNotRequired,
				Identifier:     "value",
				TypeAnnotation: NumberTypeAnnotation,
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
		Type:            ty,
		Access:          PrimitiveAccess(ast.AccessAll),
		DocString:       docString,
	}
}

var AddressConversionFunctionType = &FunctionType{
	Purity: FunctionPurityView,
	Parameters: []Parameter{
		{
			Label:          ArgumentLabelNotRequired,
			Identifier:     "value",
			TypeAnnotation: IntegerTypeAnnotation,
		},
	},
	ReturnTypeAnnotation: AddressTypeAnnotation,
	ArgumentExpressionsCheck: func(checker *Checker, argumentExpressions []ast.Expression, _ ast.HasPosition) {
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

const AddressTypeFromBytesFunctionName = "fromBytes"
const AddressTypeFromBytesFunctionDocString = `
Returns an Address from the given byte array
`

var AddressTypeFromBytesFunctionType = &FunctionType{
	Purity: FunctionPurityView,
	Parameters: []Parameter{
		{
			Label:          ArgumentLabelNotRequired,
			Identifier:     "bytes",
			TypeAnnotation: NewTypeAnnotation(ByteArrayType),
		},
	},
	ReturnTypeAnnotation: NewTypeAnnotation(TheAddressType),
}

const AddressTypeFromStringFunctionName = "fromString"
const AddressTypeFromStringFunctionDocString = `
Attempts to parse an Address from the input string. Returns nil on invalid input.
`

var AddressTypeFromStringFunctionType = FromStringFunctionType(TheAddressType)

func init() {
	// Declare a conversion function for the address type

	// Check that the function is not accidentally redeclared

	typeName := AddressTypeName

	if BaseValueActivation.Find(typeName) != nil {
		panic(errors.NewUnreachableError())
	}

	functionType := AddressConversionFunctionType

	addMember := func(member *Member) {
		if functionType.Members == nil {
			functionType.Members = &StringMemberOrderedMap{}
		}
		name := member.Identifier.Identifier
		if functionType.Members.Contains(name) {
			panic(errors.NewUnreachableError())
		}
		functionType.Members.Set(name, member)
	}

	addMember(NewUnmeteredPublicFunctionMember(
		functionType,
		AddressTypeFromBytesFunctionName,
		AddressTypeFromBytesFunctionType,
		AddressTypeFromBytesFunctionDocString,
	))
	addMember(NewUnmeteredPublicFunctionMember(
		functionType,
		AddressTypeFromStringFunctionName,
		AddressTypeFromStringFunctionType,
		AddressTypeFromStringFunctionDocString,
	))

	BaseValueActivation.Set(
		typeName,
		baseFunctionVariable(
			typeName,
			functionType,
			numberConversionDocString("an address"),
		),
	)
}

func numberFunctionArgumentExpressionsChecker(targetType Type) ArgumentExpressionsCheck {
	return func(checker *Checker, arguments []ast.Expression, invocationRange ast.HasPosition) {
		if len(arguments) < 1 {
			return
		}

		argument := arguments[0]

		switch argument := argument.(type) {
		case *ast.IntegerExpression:
			if CheckIntegerLiteral(nil, argument, targetType, checker.report) {
				if checker.Config.ExtendedElaborationEnabled {
					checker.Elaboration.SetNumberConversionArgumentTypes(
						argument,
						NumberConversionArgumentTypes{
							Type: targetType,
							Range: ast.NewRangeFromPositioned(
								checker.memoryGauge,
								invocationRange,
							),
						},
					)
				}
			}

		case *ast.FixedPointExpression:
			if CheckFixedPointLiteral(nil, argument, targetType, checker.report) {
				if checker.Config.ExtendedElaborationEnabled {
					checker.Elaboration.SetNumberConversionArgumentTypes(
						argument,
						NumberConversionArgumentTypes{
							Type: targetType,
							Range: ast.NewRangeFromPositioned(
								checker.memoryGauge,
								invocationRange,
							),
						},
					)
				}
			}
		}
	}
}

func pathConversionFunctionType(pathType Type) *FunctionType {
	return NewSimpleFunctionType(
		FunctionPurityView,
		[]Parameter{
			{
				Identifier:     "identifier",
				TypeAnnotation: StringTypeAnnotation,
			},
		},
		NewTypeAnnotation(
			&OptionalType{
				Type: pathType,
			},
		),
	)
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
				Purity:               FunctionPurityView,
				TypeParameters:       []*TypeParameter{{Name: "T"}},
				ReturnTypeAnnotation: MetaTypeAnnotation,
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

type Conformance struct {
	InterfaceType        *InterfaceType
	ConformanceChainRoot *InterfaceType
}

type CompositeType struct {
	Location      common.Location
	EnumRawType   Type
	containerType Type
	NestedTypes   *StringTypeOrderedMap

	// in a language with support for algebraic data types,
	// we would implement this as an argument to the CompositeKind type constructor.
	// Alas, this is Go, so for now these fields are only non-nil when Kind is CompositeKindAttachment
	baseType          Type
	baseTypeDocString string

	DefaultDestroyEvent *CompositeType

	cachedIdentifiers *struct {
		TypeID              TypeID
		QualifiedIdentifier string
	}
	Members               *StringMemberOrderedMap
	memberResolvers       map[string]MemberResolver
	Identifier            string
	Fields                []string
	ConstructorParameters []Parameter
	// an internal set of field `effectiveInterfaceConformances`
	effectiveInterfaceConformanceSet     *InterfaceSet
	effectiveInterfaceConformances       []Conformance
	ExplicitInterfaceConformances        []*InterfaceType
	Kind                                 common.CompositeKind
	cachedIdentifiersLock                sync.RWMutex
	effectiveInterfaceConformanceSetOnce sync.Once
	effectiveInterfaceConformancesOnce   sync.Once
	memberResolversOnce                  sync.Once
	ConstructorPurity                    FunctionPurity
	HasComputedMembers                   bool
	// Only applicable for native composite types
	ImportableBuiltin         bool
	supportedEntitlementsOnce sync.Once
	supportedEntitlements     *EntitlementSet
}

var _ Type = &CompositeType{}
var _ ContainerType = &CompositeType{}
var _ ContainedType = &CompositeType{}
var _ LocatedType = &CompositeType{}
var _ CompositeKindedType = &CompositeType{}
var _ TypeIndexableType = &CompositeType{}

func (t *CompositeType) Tag() TypeTag {
	return CompositeTypeTag
}

func (t *CompositeType) EffectiveInterfaceConformanceSet() *InterfaceSet {
	t.initializeEffectiveInterfaceConformanceSet()
	return t.effectiveInterfaceConformanceSet
}

func (t *CompositeType) initializeEffectiveInterfaceConformanceSet() {
	t.effectiveInterfaceConformanceSetOnce.Do(func() {
		t.effectiveInterfaceConformanceSet = NewInterfaceSet()

		for _, conformance := range t.EffectiveInterfaceConformances() {
			t.effectiveInterfaceConformanceSet.Add(conformance.InterfaceType)
		}
	})
}

func (t *CompositeType) EffectiveInterfaceConformances() []Conformance {
	t.effectiveInterfaceConformancesOnce.Do(func() {
		t.effectiveInterfaceConformances = distinctConformances(
			t.ExplicitInterfaceConformances,
			nil,
			map[*InterfaceType]struct{}{},
		)
	})

	return t.effectiveInterfaceConformances
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

	if t.NestedTypes != nil {
		t.NestedTypes.Foreach(checkIdentifiersCached)
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

func (t *CompositeType) getBaseCompositeKind() common.CompositeKind {
	if t.Kind != common.CompositeKindAttachment {
		return common.CompositeKindUnknown
	}
	switch base := t.baseType.(type) {
	case *CompositeType:
		return base.Kind
	case *InterfaceType:
		return base.CompositeKind
	case *SimpleType:
		return base.CompositeKind()
	}
	return common.CompositeKindUnknown
}

func isAttachmentType(t Type) bool {
	composite, ok := t.(*CompositeType)
	return (ok && composite.Kind == common.CompositeKindAttachment) ||
		t == AnyResourceAttachmentType ||
		t == AnyStructAttachmentType
}

func IsHashableStructType(t Type) bool {
	switch typ := t.(type) {
	case *AddressType:
		return true
	case *CompositeType:
		return typ.Kind == common.CompositeKindEnum
	default:
		switch typ {
		case NeverType, BoolType, CharacterType, StringType, MetaType, HashableStructType:
			return true
		default:
			return IsSubType(typ, NumberType) ||
				IsSubType(typ, PathType)
		}
	}
}

func (t *CompositeType) GetBaseType() Type {
	return t.baseType
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

// clearCachedIdentifiers clears cachedIdentifiers.
// This function currently is only used in tests.
func (t *CompositeType) clearCachedIdentifiers() {
	t.cachedIdentifiersLock.Lock()
	defer t.cachedIdentifiersLock.Unlock()

	t.cachedIdentifiers = nil
}

func (t *CompositeType) initializeIdentifiers() {
	t.cachedIdentifiersLock.Lock()
	defer t.cachedIdentifiersLock.Unlock()

	if t.cachedIdentifiers != nil {
		return
	}

	identifier := qualifiedIdentifier(t.Identifier, t.containerType)

	typeID := common.NewTypeIDFromQualifiedName(nil, t.Location, identifier)

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

func (t *CompositeType) MemberMap() *StringMemberOrderedMap {
	return t.Members
}

func newCompositeOrInterfaceSupportedEntitlementSet(
	members *StringMemberOrderedMap,
	effectiveInterfaceConformanceSet *InterfaceSet,
) *EntitlementSet {
	set := &EntitlementSet{}

	// We need to handle conjunctions and disjunctions separately, in two passes,
	// as adding entitlements after disjunctions does not remove disjunctions from the set,
	// whereas adding disjunctions after entitlements does.

	// First pass: Handle maps and conjunctions
	members.Foreach(func(_ string, member *Member) {
		switch access := member.Access.(type) {
		case *EntitlementMapAccess:
			// Domain is a conjunction, add all entitlements
			domain := access.Domain()
			if domain.SetKind != Conjunction {
				panic(errors.NewUnreachableError())
			}
			domain.Entitlements.
				Foreach(func(entitlementType *EntitlementType, _ struct{}) {
					set.Add(entitlementType)
				})

		case EntitlementSetAccess:
			// Disjunctions are handled in a second pass
			if access.SetKind == Conjunction {
				access.Entitlements.Foreach(func(entitlementType *EntitlementType, _ struct{}) {
					set.Add(entitlementType)
				})
			}
		}
	})

	// Second pass: Handle disjunctions
	for pair := members.Oldest(); pair != nil; pair = pair.Next() {
		member := pair.Value

		if access, ok := member.Access.(EntitlementSetAccess); ok &&
			access.SetKind == Disjunction {

			set.AddDisjunction(access.Entitlements)
		}
	}

	effectiveInterfaceConformanceSet.ForEach(func(it *InterfaceType) {
		set.Merge(it.SupportedEntitlements())
	})

	return set
}

func (t *CompositeType) SupportedEntitlements() *EntitlementSet {
	t.supportedEntitlementsOnce.Do(func() {

		set := newCompositeOrInterfaceSupportedEntitlementSet(
			t.Members,
			t.EffectiveInterfaceConformanceSet(),
		)

		// attachments support at least the entitlements supported by their base,
		// and we must ensure there is no recursive case
		if entitlementSupportingBase, isEntitlementSupportingBase :=
			t.GetBaseType().(EntitlementSupportingType); isEntitlementSupportingBase && entitlementSupportingBase != t {

			set.Merge(entitlementSupportingBase.SupportedEntitlements())
		}

		t.supportedEntitlements = set
	})
	return t.supportedEntitlements
}

func (t *CompositeType) IsResourceType() bool {
	return t.Kind == common.CompositeKindResource ||
		// attachments are always the same kind as their base type
		(t.Kind == common.CompositeKindAttachment &&
			// this check is necessary to prevent `attachment A for A {}`
			// from causing an infinite recursion case here
			t.baseType != t &&
			t.baseType.IsResourceType())
}

func (t *CompositeType) IsPrimitiveType() bool {
	return false
}

func (*CompositeType) IsInvalidType() bool {
	return false
}

func (*CompositeType) IsOrContainsReferenceType() bool {
	return false
}

func (t *CompositeType) IsStorable(results map[*Member]bool) bool {
	if t.HasComputedMembers {
		return false
	}

	// Only structures, resources, attachments, and enums can be stored

	switch t.Kind {
	case common.CompositeKindStructure,
		common.CompositeKindResource,
		common.CompositeKindEnum,
		common.CompositeKindAttachment:
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
		return t.ImportableBuiltin
	}

	// Only structures and enums can be imported

	switch t.Kind {
	case common.CompositeKindStructure,
		common.CompositeKindEnum:
		break
	// attachments can be imported iff they are attached to a structure
	case common.CompositeKindAttachment:
		return t.baseType.IsImportable(results)
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

func (t *CompositeType) IsExportable(results map[*Member]bool) bool {
	// Only structures, resources, attachment, and enums can be stored

	switch t.Kind {
	case common.CompositeKindStructure,
		common.CompositeKindResource,
		common.CompositeKindEnum,
		common.CompositeKindAttachment:
		break
	default:
		return false
	}

	// If this composite type has a member which is not exportable,
	// then the composite type is not exportable.

	for p := t.Members.Oldest(); p != nil; p = p.Next() {
		if !p.Value.IsExportable(results) {
			return false
		}
	}

	return true
}

func (t *CompositeType) IsEquatable() bool {
	// TODO: add support for more composite kinds
	return t.Kind == common.CompositeKindEnum
}

func (*CompositeType) IsComparable() bool {
	return false
}

func (t *CompositeType) ContainFieldsOrElements() bool {
	return t.Kind != common.CompositeKindEnum
}

func (t *CompositeType) TypeAnnotationState() TypeAnnotationState {
	if t.Kind == common.CompositeKindAttachment {
		return TypeAnnotationStateDirectAttachmentTypeAnnotation
	}
	return TypeAnnotationStateValid
}

func (t *CompositeType) RewriteWithIntersectionTypes() (result Type, rewritten bool) {
	return t, false
}

func (*CompositeType) Unify(
	_ Type,
	_ *TypeParameterTypeOrderedMap,
	_ func(err error),
	_ common.MemoryGauge,
	_ ast.HasPosition,
) bool {
	// TODO:
	return false
}

func (t *CompositeType) Resolve(_ *TypeParameterTypeOrderedMap) Type {
	return t
}

func (t *CompositeType) IsContainerType() bool {
	return t.NestedTypes != nil
}

func (t *CompositeType) GetNestedTypes() *StringTypeOrderedMap {
	return t.NestedTypes
}

func (t *CompositeType) isTypeIndexableType() bool {
	// resources and structs only can be indexed for attachments
	return t.Kind.SupportsAttachments()
}

func (t *CompositeType) TypeIndexingElementType(indexingType Type, _ func() ast.Range) (Type, error) {
	var access Access = UnauthorizedAccess
	switch attachment := indexingType.(type) {
	case *CompositeType:
		// when accessed on an owned value, the produced attachment reference is entitled to all the
		// entitlements it supports
		access = attachment.SupportedEntitlements().Access()
	}

	return &OptionalType{
		Type: &ReferenceType{
			Type:          indexingType,
			Authorization: access,
		},
	}, nil
}

func (t *CompositeType) IsValidIndexingType(ty Type) bool {
	attachmentType, isComposite := ty.(*CompositeType)
	return isComposite &&
		IsSubType(t, attachmentType.baseType) &&
		attachmentType.IsResourceType() == t.IsResourceType()
}

const CompositeForEachAttachmentFunctionName = "forEachAttachment"

const compositeForEachAttachmentFunctionDocString = `
Iterates over the attachments present on the receiver, applying the function argument to each.
The order of iteration is undefined.
`

func CompositeForEachAttachmentFunctionType(t common.CompositeKind) *FunctionType {
	attachmentSuperType := AnyStructAttachmentType
	if t == common.CompositeKindResource {
		attachmentSuperType = AnyResourceAttachmentType
	}

	return &FunctionType{
		Parameters: []Parameter{
			{
				Label:      ArgumentLabelNotRequired,
				Identifier: "f",
				TypeAnnotation: NewTypeAnnotation(
					&FunctionType{
						Parameters: []Parameter{
							{
								TypeAnnotation: NewTypeAnnotation(
									&ReferenceType{
										Type:          attachmentSuperType,
										Authorization: UnauthorizedAccess,
									},
								),
							},
						},
						ReturnTypeAnnotation: VoidTypeAnnotation,
					},
				),
			},
		},
		ReturnTypeAnnotation: VoidTypeAnnotation,
	}
}

func (t *CompositeType) Map(_ common.MemoryGauge, _ map[*TypeParameter]*TypeParameter, f func(Type) Type) Type {
	return f(t)
}

func (t *CompositeType) GetMembers() map[string]MemberResolver {
	t.initializeMemberResolvers()
	return t.memberResolvers
}

func (t *CompositeType) initializeMemberResolvers() {
	t.memberResolversOnce.Do(t.initializerMemberResolversFunc())
}

func (t *CompositeType) initializerMemberResolversFunc() func() {
	return func() {
		memberResolvers := MembersMapAsResolvers(t.Members)

		// Check conformances.
		// If this composite type results from a normal composite declaration,
		// it must have members declared for all interfaces it conforms to.
		// However, if this composite type is a type requirement,
		// it acts like an interface and does not have to declare members.

		t.EffectiveInterfaceConformanceSet().
			ForEach(func(conformance *InterfaceType) {
				for name, resolver := range conformance.GetMembers() { //nolint:maprange
					if _, ok := memberResolvers[name]; !ok {
						memberResolvers[name] = resolver
					}
				}
			})

		// resource and struct composites have the ability to iterate over their attachments
		if t.Kind.SupportsAttachments() {
			memberResolvers[CompositeForEachAttachmentFunctionName] = MemberResolver{
				Kind: common.DeclarationKindFunction,
				Resolve: func(
					memoryGauge common.MemoryGauge,
					identifier string,
					_ ast.HasPosition,
					_ func(error),
				) *Member {
					return NewPublicFunctionMember(
						memoryGauge,
						t,
						identifier,
						CompositeForEachAttachmentFunctionType(t.GetCompositeKind()),
						compositeForEachAttachmentFunctionDocString,
					)
				},
			}
		}

		t.memberResolvers = withBuiltinMembers(t, memberResolvers)
	}
}

func (t *CompositeType) ResolveMembers() {
	if t.Members.Len() != len(t.GetMembers()) {
		t.initializerMemberResolversFunc()()
	}
}

func (t *CompositeType) FieldPosition(name string, declaration ast.CompositeLikeDeclaration) ast.Position {
	var pos ast.Position
	if t.Kind == common.CompositeKindEnum &&
		name == EnumRawValueFieldName {

		if len(declaration.ConformanceList()) > 0 {
			pos = declaration.ConformanceList()[0].StartPosition()
		}
	} else {
		pos = declaration.DeclarationMembers().FieldPosition(name, declaration.Kind())
	}
	return pos
}

func (t *CompositeType) SetNestedType(name string, nestedType ContainedType) {
	if t.NestedTypes == nil {
		t.NestedTypes = &StringTypeOrderedMap{}
	}
	t.NestedTypes.Set(name, nestedType)
	nestedType.SetContainerType(t)
}

func (t *CompositeType) ConstructorFunctionType() *FunctionType {
	return &FunctionType{
		IsConstructor:        true,
		Purity:               t.ConstructorPurity,
		Parameters:           t.ConstructorParameters,
		ReturnTypeAnnotation: NewTypeAnnotation(t),
	}
}

func (t *CompositeType) InitializerFunctionType() *FunctionType {
	return &FunctionType{
		IsConstructor:        true,
		Purity:               t.ConstructorPurity,
		Parameters:           t.ConstructorParameters,
		ReturnTypeAnnotation: VoidTypeAnnotation,
	}
}

func (t *CompositeType) InitializerEffectiveArgumentLabels() []string {
	parameters := t.ConstructorParameters
	if len(parameters) == 0 {
		return nil
	}

	argumentLabels := make([]string, 0, len(parameters))
	for _, parameter := range parameters {
		argumentLabels = append(
			argumentLabels,
			parameter.EffectiveArgumentLabel(),
		)
	}
	return argumentLabels
}

func (t *CompositeType) CheckInstantiated(pos ast.HasPosition, memoryGauge common.MemoryGauge, report func(err error)) {
	if t.EnumRawType != nil {
		t.EnumRawType.CheckInstantiated(pos, memoryGauge, report)
	}

	if t.baseType != nil {
		t.baseType.CheckInstantiated(pos, memoryGauge, report)
	}

	for _, typ := range t.ExplicitInterfaceConformances {
		typ.CheckInstantiated(pos, memoryGauge, report)
	}
}

// Member

type Member struct {
	TypeAnnotation TypeAnnotation
	// Parent type where this member can be resolved
	ContainerType  Type
	DocString      string
	ArgumentLabels []string
	Identifier     ast.Identifier
	Access         Access
	// TODO: replace with dedicated MemberKind enum
	DeclarationKind common.DeclarationKind
	VariableKind    ast.VariableKind
	// Predeclared fields can be considered initialized
	Predeclared       bool
	HasImplementation bool
	HasConditions     bool
	// IgnoreInSerialization determines if the field is ignored in serialization
	IgnoreInSerialization bool
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
	return NewFunctionMember(
		memoryGauge,
		containerType,
		UnauthorizedAccess,
		identifier,
		functionType,
		docString,
	)
}

func NewUnmeteredFunctionMember(
	containerType Type,
	access Access,
	identifier string,
	functionType *FunctionType,
	docString string,
) *Member {
	return NewFunctionMember(
		nil,
		containerType,
		access,
		identifier,
		functionType,
		docString,
	)
}

func NewFunctionMember(
	memoryGauge common.MemoryGauge,
	containerType Type,
	access Access,
	identifier string,
	functionType *FunctionType,
	docString string,
) *Member {

	return &Member{
		ContainerType: containerType,
		Access:        access,
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

func NewUnmeteredConstructorMember(
	containerType Type,
	access Access,
	identifier string,
	functionType *FunctionType,
	docString string,
) *Member {
	return NewConstructorMember(
		nil,
		containerType,
		access,
		identifier,
		functionType,
		docString,
	)
}

func NewConstructorMember(
	memoryGauge common.MemoryGauge,
	containerType Type,
	access Access,
	identifier string,
	functionType *FunctionType,
	docString string,
) *Member {

	return &Member{
		ContainerType: containerType,
		Access:        access,
		Identifier: ast.NewIdentifier(
			memoryGauge,
			identifier,
			ast.EmptyPosition,
		),
		DeclarationKind: common.DeclarationKindInitializer,
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
	return NewFieldMember(
		memoryGauge,
		containerType,
		UnauthorizedAccess,
		ast.VariableKindConstant,
		identifier,
		fieldType,
		docString,
	)
}

func NewUnmeteredFieldMember(
	containerType Type,
	access Access,
	variableKind ast.VariableKind,
	identifier string,
	fieldType Type,
	docString string,
) *Member {
	return NewFieldMember(
		nil,
		containerType,
		access,
		variableKind,
		identifier,
		fieldType,
		docString,
	)
}

func NewFieldMember(
	memoryGauge common.MemoryGauge,
	containerType Type,
	access Access,
	variableKind ast.VariableKind,
	identifier string,
	fieldType Type,
	docString string,
) *Member {
	return &Member{
		ContainerType: containerType,
		Access:        access,
		Identifier: ast.NewIdentifier(
			memoryGauge,
			identifier,
			ast.EmptyPosition,
		),
		DeclarationKind: common.DeclarationKindField,
		VariableKind:    variableKind,
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

// IsExportable returns whether a member is exportable
func (m *Member) IsExportable(results map[*Member]bool) (result bool) {
	test := func(t Type) bool {
		return t.IsExportable(results)
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
	Location          common.Location
	containerType     Type
	Members           *StringMemberOrderedMap
	memberResolvers   map[string]MemberResolver
	NestedTypes       *StringTypeOrderedMap
	cachedIdentifiers *struct {
		TypeID              TypeID
		QualifiedIdentifier string
	}

	Identifier                           string
	Fields                               []string
	InitializerParameters                []Parameter
	CompositeKind                        common.CompositeKind
	cachedIdentifiersLock                sync.RWMutex
	memberResolversOnce                  sync.Once
	effectiveInterfaceConformancesOnce   sync.Once
	effectiveInterfaceConformanceSetOnce sync.Once
	InitializerPurity                    FunctionPurity

	ExplicitInterfaceConformances    []*InterfaceType
	effectiveInterfaceConformances   []Conformance
	effectiveInterfaceConformanceSet *InterfaceSet
	supportedEntitlementsOnce        sync.Once
	supportedEntitlements            *EntitlementSet

	DefaultDestroyEvent *CompositeType
}

var _ Type = &InterfaceType{}
var _ ContainerType = &InterfaceType{}
var _ ContainedType = &InterfaceType{}
var _ LocatedType = &InterfaceType{}
var _ CompositeKindedType = &InterfaceType{}

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

	if t.NestedTypes != nil {
		t.NestedTypes.Foreach(checkIdentifiersCached)
	}
}

// clearCachedIdentifiers clears cachedIdentifiers.
// This function currently is only used in tests.
func (t *InterfaceType) clearCachedIdentifiers() {
	t.cachedIdentifiersLock.Lock()
	defer t.cachedIdentifiersLock.Unlock()

	t.cachedIdentifiers = nil
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

	typeID := common.NewTypeIDFromQualifiedName(nil, t.Location, identifier)

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

func (t *InterfaceType) MemberMap() *StringMemberOrderedMap {
	return t.Members
}

func (t *InterfaceType) SupportedEntitlements() *EntitlementSet {
	t.supportedEntitlementsOnce.Do(func() {
		t.supportedEntitlements = newCompositeOrInterfaceSupportedEntitlementSet(
			t.Members,
			t.EffectiveInterfaceConformanceSet(),
		)
	})
	return t.supportedEntitlements
}

func (t *InterfaceType) Map(_ common.MemoryGauge, _ map[*TypeParameter]*TypeParameter, f func(Type) Type) Type {
	return f(t)
}

func (t *InterfaceType) GetMembers() map[string]MemberResolver {
	t.initializeMemberResolvers()
	return t.memberResolvers
}

func (t *InterfaceType) initializeMemberResolvers() {
	t.memberResolversOnce.Do(func() {
		members := MembersMapAsResolvers(t.Members)

		// add any inherited members from up the inheritance chain
		for _, conformance := range t.EffectiveInterfaceConformances() {
			for name, member := range conformance.InterfaceType.GetMembers() { //nolint:maprange
				if _, ok := members[name]; !ok {
					members[name] = member
				}
			}

		}

		t.memberResolvers = withBuiltinMembers(t, members)
	})
}

func (t *InterfaceType) IsResourceType() bool {
	return t.CompositeKind == common.CompositeKindResource
}

func (*InterfaceType) IsPrimitiveType() bool {
	return false
}

func (*InterfaceType) IsInvalidType() bool {
	return false
}

func (*InterfaceType) IsOrContainsReferenceType() bool {
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

func (t *InterfaceType) IsExportable(results map[*Member]bool) bool {

	if t.CompositeKind != common.CompositeKindStructure {
		return false
	}

	// If this interface type has a member which is not exportable,
	// then the interface type is not exportable.

	for pair := t.Members.Oldest(); pair != nil; pair = pair.Next() {
		if !pair.Value.IsExportable(results) {
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

func (*InterfaceType) IsComparable() bool {
	return false
}

func (*InterfaceType) ContainFieldsOrElements() bool {
	return true
}

func (*InterfaceType) TypeAnnotationState() TypeAnnotationState {
	return TypeAnnotationStateValid
}

func (t *InterfaceType) RewriteWithIntersectionTypes() (Type, bool) {
	return &IntersectionType{
		Types: []*InterfaceType{t},
	}, true
}

func (*InterfaceType) Unify(
	_ Type,
	_ *TypeParameterTypeOrderedMap,
	_ func(err error),
	_ common.MemoryGauge,
	_ ast.HasPosition,
) bool {
	// TODO:
	return false
}

func (t *InterfaceType) Resolve(_ *TypeParameterTypeOrderedMap) Type {
	return t
}

func (t *InterfaceType) IsContainerType() bool {
	return t.NestedTypes != nil
}

func (t *InterfaceType) GetNestedTypes() *StringTypeOrderedMap {
	return t.NestedTypes
}

func (t *InterfaceType) FieldPosition(name string, declaration *ast.InterfaceDeclaration) ast.Position {
	return declaration.Members.FieldPosition(name, declaration.CompositeKind)
}

func (t *InterfaceType) EffectiveInterfaceConformances() []Conformance {
	t.effectiveInterfaceConformancesOnce.Do(func() {
		t.effectiveInterfaceConformances = distinctConformances(
			t.ExplicitInterfaceConformances,
			nil,
			map[*InterfaceType]struct{}{},
		)
	})

	return t.effectiveInterfaceConformances
}

func (t *InterfaceType) EffectiveInterfaceConformanceSet() *InterfaceSet {
	t.initializeEffectiveInterfaceConformanceSet()
	return t.effectiveInterfaceConformanceSet
}

func (t *InterfaceType) initializeEffectiveInterfaceConformanceSet() {
	t.effectiveInterfaceConformanceSetOnce.Do(func() {
		t.effectiveInterfaceConformanceSet = NewInterfaceSet()

		for _, conformance := range t.EffectiveInterfaceConformances() {
			t.effectiveInterfaceConformanceSet.Add(conformance.InterfaceType)
		}
	})
}

// distinctConformances recursively visit conformances and their conformances,
// and return all the distinct conformances as an array.
func distinctConformances(
	conformances []*InterfaceType,
	parent *InterfaceType,
	seenConformances map[*InterfaceType]struct{},
) []Conformance {

	if len(conformances) == 0 {
		return nil
	}

	collectedConformances := make([]Conformance, 0)

	var conformanceChainRoot *InterfaceType

	for _, conformance := range conformances {
		if _, ok := seenConformances[conformance]; ok {
			continue
		}
		seenConformances[conformance] = struct{}{}

		if parent == nil {
			conformanceChainRoot = conformance
		} else {
			conformanceChainRoot = parent
		}

		collectedConformances = append(
			collectedConformances,
			Conformance{
				InterfaceType:        conformance,
				ConformanceChainRoot: conformanceChainRoot,
			},
		)

		// Recursively collect conformances
		nestedConformances := distinctConformances(
			conformance.ExplicitInterfaceConformances,
			conformanceChainRoot,
			seenConformances,
		)

		collectedConformances = append(collectedConformances, nestedConformances...)
	}

	return collectedConformances
}

func (t *InterfaceType) CheckInstantiated(pos ast.HasPosition, memoryGauge common.MemoryGauge, report func(err error)) {
	for _, param := range t.InitializerParameters {
		param.TypeAnnotation.Type.CheckInstantiated(pos, memoryGauge, report)
	}
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

var _ Type = &DictionaryType{}
var _ ValueIndexableType = &DictionaryType{}
var _ EntitlementSupportingType = &DictionaryType{}

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

func FormatDictionaryTypeID[T ~string](keyTypeID T, valueTypeID T) T {
	return T(fmt.Sprintf(
		"{%s:%s}",
		keyTypeID,
		valueTypeID,
	))
}

func (t *DictionaryType) ID() TypeID {
	return FormatDictionaryTypeID(
		t.KeyType.ID(),
		t.ValueType.ID(),
	)
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

func (t *DictionaryType) IsPrimitiveType() bool {
	return false
}

func (t *DictionaryType) IsInvalidType() bool {
	return t.KeyType.IsInvalidType() ||
		t.ValueType.IsInvalidType()
}

func (t *DictionaryType) IsOrContainsReferenceType() bool {
	return t.KeyType.IsOrContainsReferenceType() ||
		t.ValueType.IsOrContainsReferenceType()
}

func (t *DictionaryType) IsStorable(results map[*Member]bool) bool {
	return t.KeyType.IsStorable(results) &&
		t.ValueType.IsStorable(results)
}

func (t *DictionaryType) IsExportable(results map[*Member]bool) bool {
	return t.KeyType.IsExportable(results) &&
		t.ValueType.IsExportable(results)
}

func (t *DictionaryType) IsImportable(results map[*Member]bool) bool {
	return t.KeyType.IsImportable(results) &&
		t.ValueType.IsImportable(results)
}

func (t *DictionaryType) IsEquatable() bool {
	return t.KeyType.IsEquatable() &&
		t.ValueType.IsEquatable()
}

func (*DictionaryType) IsComparable() bool {
	return false
}

func (*DictionaryType) ContainFieldsOrElements() bool {
	return true
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

func (t *DictionaryType) RewriteWithIntersectionTypes() (Type, bool) {
	rewrittenKeyType, keyTypeRewritten := t.KeyType.RewriteWithIntersectionTypes()
	rewrittenValueType, valueTypeRewritten := t.ValueType.RewriteWithIntersectionTypes()
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

func (t *DictionaryType) CheckInstantiated(pos ast.HasPosition, memoryGauge common.MemoryGauge, report func(err error)) {
	t.KeyType.CheckInstantiated(pos, memoryGauge, report)
	t.ValueType.CheckInstantiated(pos, memoryGauge, report)
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

const dictionaryTypeForEachKeyFunctionDocString = `
Iterate over each key in this dictionary, exiting early if the passed function returns false.
This method is more performant than calling .keys and then iterating over the resulting array,
since no intermediate storage is allocated.

The order of iteration is undefined
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

func (t *DictionaryType) Map(gauge common.MemoryGauge, typeParamMap map[*TypeParameter]*TypeParameter, f func(Type) Type) Type {
	return f(NewDictionaryType(
		gauge,
		t.KeyType.Map(gauge, typeParamMap, f),
		t.ValueType.Map(gauge, typeParamMap, f),
	))
}

func (t *DictionaryType) GetMembers() map[string]MemberResolver {
	t.initializeMemberResolvers()
	return t.memberResolvers
}

func (t *DictionaryType) initializeMemberResolvers() {
	t.memberResolversOnce.Do(func() {

		t.memberResolvers = withBuiltinMembers(
			t,
			map[string]MemberResolver{
				"containsKey": {
					Kind: common.DeclarationKindFunction,
					Resolve: func(
						memoryGauge common.MemoryGauge,
						identifier string,
						targetRange ast.HasPosition,
						report func(error),
					) *Member {

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
					Resolve: func(
						memoryGauge common.MemoryGauge,
						identifier string,
						_ ast.HasPosition,
						_ func(error),
					) *Member {
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
					Resolve: func(
						memoryGauge common.MemoryGauge,
						identifier string,
						targetRange ast.HasPosition,
						report func(error),
					) *Member {
						// TODO: maybe allow for resource key type

						if t.KeyType.IsResourceType() {
							report(
								&InvalidResourceDictionaryMemberError{
									Name:            identifier,
									DeclarationKind: common.DeclarationKindField,
									Range:           ast.NewRangeFromPositioned(memoryGauge, targetRange),
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
					Resolve: func(
						memoryGauge common.MemoryGauge,
						identifier string,
						targetRange ast.HasPosition,
						report func(error),
					) *Member {
						// TODO: maybe allow for resource value type

						if t.ValueType.IsResourceType() {
							report(
								&InvalidResourceDictionaryMemberError{
									Name:            identifier,
									DeclarationKind: common.DeclarationKindField,
									Range:           ast.NewRangeFromPositioned(memoryGauge, targetRange),
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
					Kind: common.DeclarationKindFunction,
					Resolve: func(
						memoryGauge common.MemoryGauge,
						identifier string,
						_ ast.HasPosition,
						_ func(error),
					) *Member {
						return NewFunctionMember(
							memoryGauge,
							t,
							insertMutateEntitledAccess,
							identifier,
							DictionaryInsertFunctionType(t),
							dictionaryTypeInsertFunctionDocString,
						)
					},
				},
				"remove": {
					Kind: common.DeclarationKindFunction,
					Resolve: func(
						memoryGauge common.MemoryGauge,
						identifier string,
						_ ast.HasPosition,
						_ func(error),
					) *Member {
						return NewFunctionMember(
							memoryGauge,
							t,
							removeMutateEntitledAccess,
							identifier,
							DictionaryRemoveFunctionType(t),
							dictionaryTypeRemoveFunctionDocString,
						)
					},
				},
				"forEachKey": {
					Kind: common.DeclarationKindFunction,
					Resolve: func(
						memoryGauge common.MemoryGauge,
						identifier string,
						targetRange ast.HasPosition,
						report func(error),
					) *Member {
						if t.KeyType.IsResourceType() {
							report(
								&InvalidResourceDictionaryMemberError{
									Name:            identifier,
									DeclarationKind: common.DeclarationKindField,
									Range:           ast.NewRangeFromPositioned(memoryGauge, targetRange),
								},
							)
						}

						return NewPublicFunctionMember(
							memoryGauge,
							t,
							identifier,
							DictionaryForEachKeyFunctionType(t),
							dictionaryTypeForEachKeyFunctionDocString,
						)
					},
				},
			},
		)
	})
}

func DictionaryContainsKeyFunctionType(t *DictionaryType) *FunctionType {
	return NewSimpleFunctionType(
		FunctionPurityView,
		[]Parameter{
			{
				Label:          ArgumentLabelNotRequired,
				Identifier:     "key",
				TypeAnnotation: NewTypeAnnotation(t.KeyType),
			},
		},
		BoolTypeAnnotation,
	)
}

func DictionaryInsertFunctionType(t *DictionaryType) *FunctionType {
	return NewSimpleFunctionType(
		FunctionPurityImpure,
		[]Parameter{
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
		NewTypeAnnotation(
			&OptionalType{
				Type: t.ValueType,
			},
		),
	)
}

func DictionaryRemoveFunctionType(t *DictionaryType) *FunctionType {
	return NewSimpleFunctionType(
		FunctionPurityImpure,
		[]Parameter{
			{
				Identifier:     "key",
				TypeAnnotation: NewTypeAnnotation(t.KeyType),
			},
		},
		NewTypeAnnotation(
			&OptionalType{
				Type: t.ValueType,
			},
		),
	)
}

func DictionaryForEachKeyFunctionType(t *DictionaryType) *FunctionType {
	const functionPurity = FunctionPurityImpure

	// fun(K): Bool
	funcType := NewSimpleFunctionType(
		functionPurity,
		[]Parameter{
			{
				Identifier:     "key",
				TypeAnnotation: NewTypeAnnotation(t.KeyType),
			},
		},
		BoolTypeAnnotation,
	)

	// fun forEachKey(_ function: fun(K): Bool): Void
	return NewSimpleFunctionType(
		functionPurity,
		[]Parameter{
			{
				Label:          ArgumentLabelNotRequired,
				Identifier:     "function",
				TypeAnnotation: NewTypeAnnotation(funcType),
			},
		},
		VoidTypeAnnotation,
	)
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
	memoryGauge common.MemoryGauge,
	outerRange ast.HasPosition,
) bool {

	otherDictionary, ok := other.(*DictionaryType)
	if !ok {
		return false
	}

	keyUnified := t.KeyType.Unify(
		otherDictionary.KeyType,
		typeParameters,
		report,
		memoryGauge,
		outerRange,
	)

	valueUnified := t.ValueType.Unify(
		otherDictionary.ValueType,
		typeParameters,
		report,
		memoryGauge,
		outerRange,
	)

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

func (t *DictionaryType) SupportedEntitlements() *EntitlementSet {
	return arrayDictionaryEntitlements
}

// InclusiveRangeType

type InclusiveRangeType struct {
	MemberType          Type
	memberResolvers     map[string]MemberResolver
	memberResolversOnce sync.Once
}

var _ Type = &InclusiveRangeType{}
var _ ParameterizedType = &InclusiveRangeType{}

func NewInclusiveRangeType(memoryGauge common.MemoryGauge, elementType Type) *InclusiveRangeType {
	common.UseMemory(memoryGauge, common.DictionarySemaTypeMemoryUsage)
	return &InclusiveRangeType{
		MemberType: elementType,
	}
}

func (*InclusiveRangeType) IsType() {}

func (*InclusiveRangeType) Tag() TypeTag {
	return InclusiveRangeTypeTag
}

func (t *InclusiveRangeType) String() string {
	memberString := ""
	if t.MemberType != nil {
		memberString = fmt.Sprintf("<%s>", t.MemberType.String())
	}
	return fmt.Sprintf(
		"InclusiveRange%s",
		memberString,
	)
}

func (t *InclusiveRangeType) QualifiedString() string {
	memberString := ""
	if t.MemberType != nil {
		memberString = fmt.Sprintf("<%s>", t.MemberType.QualifiedString())
	}
	return fmt.Sprintf(
		"InclusiveRange%s",
		memberString,
	)
}

func InclusiveRangeTypeID(memberTypeID string) TypeID {
	if memberTypeID != "" {
		memberTypeID = fmt.Sprintf("<%s>", memberTypeID)
	}
	return TypeID(fmt.Sprintf(
		"InclusiveRange%s",
		memberTypeID,
	))
}

func (t *InclusiveRangeType) ID() TypeID {
	var memberTypeID string
	if t.MemberType != nil {
		memberTypeID = string(t.MemberType.ID())
	}
	return InclusiveRangeTypeID(memberTypeID)
}

func (t *InclusiveRangeType) Equal(other Type) bool {
	otherRange, ok := other.(*InclusiveRangeType)
	if !ok {
		return false
	}
	if otherRange.MemberType == nil {
		return t.MemberType == nil
	}

	return otherRange.MemberType.Equal(t.MemberType)
}

func (*InclusiveRangeType) IsResourceType() bool {
	return false
}
func (t *InclusiveRangeType) IsInvalidType() bool {
	return t.MemberType != nil && t.MemberType.IsInvalidType()
}

func (t *InclusiveRangeType) IsOrContainsReferenceType() bool {
	return t.MemberType != nil && t.MemberType.IsOrContainsReferenceType()
}

func (*InclusiveRangeType) IsStorable(_ map[*Member]bool) bool {
	return false
}

func (t *InclusiveRangeType) IsExportable(results map[*Member]bool) bool {
	return t.MemberType.IsExportable(results)
}

func (t *InclusiveRangeType) IsImportable(results map[*Member]bool) bool {
	return t.MemberType.IsImportable(results)
}

func (t *InclusiveRangeType) IsEquatable() bool {
	return t.MemberType.IsEquatable()
}

func (*InclusiveRangeType) IsComparable() bool {
	return false
}

func (t *InclusiveRangeType) TypeAnnotationState() TypeAnnotationState {
	if t.MemberType == nil {
		return TypeAnnotationStateValid
	}

	return t.MemberType.TypeAnnotationState()
}

func (t *InclusiveRangeType) RewriteWithIntersectionTypes() (Type, bool) {
	if t.MemberType == nil {
		return t, false
	}
	rewrittenMemberType, rewritten := t.MemberType.RewriteWithIntersectionTypes()
	if rewritten {
		return &InclusiveRangeType{
			MemberType: rewrittenMemberType,
		}, true
	}
	return t, false
}

func (t *InclusiveRangeType) BaseType() Type {
	if t.MemberType == nil {
		return nil
	}
	return &InclusiveRangeType{}
}

func (t *InclusiveRangeType) Instantiate(
	memoryGauge common.MemoryGauge,
	typeArguments []Type,
	astTypeArguments []*ast.TypeAnnotation,
	report func(err error),
) Type {

	const typeParameterCount = 1

	getRange := func() ast.Range {
		if astTypeArguments == nil || len(astTypeArguments) != typeParameterCount {
			return ast.EmptyRange
		}
		return ast.NewRangeFromPositioned(memoryGauge, astTypeArguments[0])
	}

	typeArgumentCount := len(typeArguments)

	var memberType Type
	if typeArgumentCount == typeParameterCount {
		memberType = typeArguments[0]
	} else {
		report(&InvalidTypeArgumentCountError{
			TypeParameterCount: typeParameterCount,
			TypeArgumentCount:  typeArgumentCount,
			Range:              getRange(),
		})
	}

	// memberType must only be a leaf integer type.
	for _, ty := range AllNonLeafIntegerTypes {
		if memberType == ty {
			report(&InvalidTypeArgumentError{
				TypeArgumentName: inclusiveRangeTypeParameter.Name,
				Range:            getRange(),
				Details:          fmt.Sprintf("Creation of InclusiveRange<%s> is disallowed", memberType),
			})
			break
		}
	}

	return &InclusiveRangeType{
		MemberType: memberType,
	}
}

func (t *InclusiveRangeType) TypeArguments() []Type {
	memberType := t.MemberType
	return []Type{
		memberType,
	}
}

func (t *InclusiveRangeType) CheckInstantiated(pos ast.HasPosition, memoryGauge common.MemoryGauge, report func(err error)) {
	CheckParameterizedTypeInstantiated(t, pos, memoryGauge, report)
}

var inclusiveRangeTypeParameter = &TypeParameter{
	Name:      "T",
	TypeBound: IntegerType,
}

func (*InclusiveRangeType) TypeParameters() []*TypeParameter {
	return []*TypeParameter{
		inclusiveRangeTypeParameter,
	}
}

const InclusiveRangeTypeStartFieldName = "start"
const inclusiveRangeTypeStartFieldDocString = `
The start of the InclusiveRange sequence
`
const InclusiveRangeTypeEndFieldName = "end"
const inclusiveRangeTypeEndFieldDocString = `
The end of the InclusiveRange sequence
`

const InclusiveRangeTypeStepFieldName = "step"
const inclusiveRangeTypeStepFieldDocString = `
The step size of the InclusiveRange sequence
`

var InclusiveRangeTypeFieldNames = []string{
	InclusiveRangeTypeStartFieldName,
	InclusiveRangeTypeEndFieldName,
	InclusiveRangeTypeStepFieldName,
}

const InclusiveRangeTypeContainsFunctionName = "contains"

const inclusiveRangeTypeContainsFunctionDocString = `
Returns true if the given integer is in the InclusiveRange sequence
`

func (t *InclusiveRangeType) GetMembers() map[string]MemberResolver {
	t.initializeMemberResolvers()
	return t.memberResolvers
}

func InclusiveRangeContainsFunctionType(elementType Type) *FunctionType {
	return NewSimpleFunctionType(
		FunctionPurityView,
		[]Parameter{
			{
				Label:          ArgumentLabelNotRequired,
				Identifier:     "element",
				TypeAnnotation: NewTypeAnnotation(elementType),
			},
		},
		BoolTypeAnnotation,
	)
}

func (t *InclusiveRangeType) initializeMemberResolvers() {
	t.memberResolversOnce.Do(func() {
		t.memberResolvers = withBuiltinMembers(
			t,
			map[string]MemberResolver{
				InclusiveRangeTypeStartFieldName: {
					Kind: common.DeclarationKindField,
					Resolve: func(
						memoryGauge common.MemoryGauge,
						identifier string,
						_ ast.HasPosition,
						_ func(error),
					) *Member {
						return NewPublicConstantFieldMember(
							memoryGauge,
							t,
							identifier,
							t.MemberType,
							inclusiveRangeTypeStartFieldDocString,
						)
					},
				},
				InclusiveRangeTypeEndFieldName: {
					Kind: common.DeclarationKindField,
					Resolve: func(
						memoryGauge common.MemoryGauge,
						identifier string,
						_ ast.HasPosition,
						_ func(error),
					) *Member {
						return NewPublicConstantFieldMember(
							memoryGauge,
							t,
							identifier,
							t.MemberType,
							inclusiveRangeTypeEndFieldDocString,
						)
					},
				},
				InclusiveRangeTypeStepFieldName: {
					Kind: common.DeclarationKindField,
					Resolve: func(
						memoryGauge common.MemoryGauge,
						identifier string,
						_ ast.HasPosition,
						_ func(error),
					) *Member {
						return NewPublicConstantFieldMember(
							memoryGauge,
							t,
							identifier,
							t.MemberType,
							inclusiveRangeTypeStepFieldDocString,
						)
					},
				},
				InclusiveRangeTypeContainsFunctionName: {
					Kind: common.DeclarationKindFunction,
					Resolve: func(
						memoryGauge common.MemoryGauge,
						identifier string,
						targetRange ast.HasPosition,
						report func(error),
					) *Member {
						elementType := t.MemberType

						return NewPublicFunctionMember(
							memoryGauge,
							t,
							identifier,
							InclusiveRangeContainsFunctionType(elementType),
							inclusiveRangeTypeContainsFunctionDocString,
						)
					},
				},
			},
		)
	})
}

func (*InclusiveRangeType) AllowsValueIndexingAssignment() bool {
	return false
}

func (t *InclusiveRangeType) Unify(
	other Type,
	typeParameters *TypeParameterTypeOrderedMap,
	report func(err error),
	memoryGauge common.MemoryGauge,
	outerRange ast.HasPosition,
) bool {
	otherRange, ok := other.(*InclusiveRangeType)
	if !ok {
		return false
	}

	return t.MemberType.Unify(
		otherRange.MemberType,
		typeParameters,
		report,
		memoryGauge,
		outerRange,
	)
}

func (t *InclusiveRangeType) Resolve(typeArguments *TypeParameterTypeOrderedMap) Type {
	memberType := t.MemberType.Resolve(typeArguments)
	if memberType == nil {
		return nil
	}

	return &InclusiveRangeType{
		MemberType: memberType,
	}
}

func (t *InclusiveRangeType) IsPrimitiveType() bool {
	return false
}

func (t *InclusiveRangeType) ContainFieldsOrElements() bool {
	return false
}

func (t *InclusiveRangeType) Map(
	gauge common.MemoryGauge,
	typeParamMap map[*TypeParameter]*TypeParameter,
	f func(Type) Type,
) Type {
	mappedMemberType := t.MemberType.Map(gauge, typeParamMap, f)
	return f(NewInclusiveRangeType(gauge, mappedMemberType))
}

// ReferenceType represents the reference to a value
type ReferenceType struct {
	Type          Type
	Authorization Access
}

var _ Type = &ReferenceType{}

// Not all references are indexable, but some are, depending on the reference's type
var _ ValueIndexableType = &ReferenceType{}
var _ TypeIndexableType = &ReferenceType{}

var UnauthorizedAccess Access = PrimitiveAccess(ast.AccessAll)
var InaccessibleAccess Access = PrimitiveAccess(ast.AccessNone)

func NewReferenceType(
	memoryGauge common.MemoryGauge,
	authorization Access,
	typ Type,
) *ReferenceType {
	common.UseMemory(memoryGauge, common.ReferenceSemaTypeMemoryUsage)
	return &ReferenceType{
		Type:          typ,
		Authorization: authorization,
	}
}

func (*ReferenceType) IsType() {}

func (t *ReferenceType) Tag() TypeTag {
	return ReferenceTypeTag
}

func formatReferenceType[T ~string](
	separator string,
	authorization T,
	typeString T,
) string {
	var builder strings.Builder
	if authorization != "" {
		builder.WriteString("auth(")
		builder.WriteString(string(authorization))
		builder.WriteString(")")
		builder.WriteString(separator)
	}
	builder.WriteByte('&')
	builder.WriteString(string(typeString))
	return builder.String()
}

func FormatReferenceTypeID[T ~string](authorization T, borrowTypeID T) T {
	return T(formatReferenceType("", authorization, borrowTypeID))
}

func (t *ReferenceType) String() string {
	if t.Type == nil {
		return "reference"
	}
	var authorization string
	if t.Authorization != UnauthorizedAccess {
		authorization = t.Authorization.String()
	}
	if _, isMapping := t.Authorization.(*EntitlementMapAccess); isMapping {
		authorization = "mapping " + authorization
	}
	return formatReferenceType(" ", authorization, t.Type.String())
}

func (t *ReferenceType) QualifiedString() string {
	if t.Type == nil {
		return "reference"
	}
	var authorization string
	if t.Authorization != UnauthorizedAccess {
		authorization = t.Authorization.QualifiedString()
	}
	if _, isMapping := t.Authorization.(*EntitlementMapAccess); isMapping {
		authorization = "mapping " + authorization
	}
	return formatReferenceType(" ", authorization, t.Type.QualifiedString())
}

func (t *ReferenceType) ID() TypeID {
	if t.Type == nil {
		return "reference"
	}
	var authorization TypeID
	if t.Authorization != UnauthorizedAccess {
		authorization = t.Authorization.ID()
	}
	return FormatReferenceTypeID(
		authorization,
		t.Type.ID(),
	)
}

func (t *ReferenceType) Equal(other Type) bool {
	otherReference, ok := other.(*ReferenceType)
	if !ok {
		return false
	}

	if !t.Authorization.Equal(otherReference.Authorization) {
		return false
	}

	return t.Type.Equal(otherReference.Type)
}

func (t *ReferenceType) IsResourceType() bool {
	return false
}

func (t *ReferenceType) IsPrimitiveType() bool {
	return false
}

func (t *ReferenceType) IsInvalidType() bool {
	return t.Type.IsInvalidType()
}

func (*ReferenceType) IsOrContainsReferenceType() bool {
	return true
}

func (t *ReferenceType) IsStorable(_ map[*Member]bool) bool {
	return false
}

func (t *ReferenceType) IsExportable(_ map[*Member]bool) bool {
	return true
}

func (t *ReferenceType) IsImportable(_ map[*Member]bool) bool {
	return false
}

func (*ReferenceType) IsEquatable() bool {
	return true
}

func (*ReferenceType) IsComparable() bool {
	return false
}

func (*ReferenceType) ContainFieldsOrElements() bool {
	return false
}

func (t *ReferenceType) TypeAnnotationState() TypeAnnotationState {
	if t.Type.TypeAnnotationState() == TypeAnnotationStateDirectEntitlementTypeAnnotation {
		return TypeAnnotationStateDirectEntitlementTypeAnnotation
	}
	return TypeAnnotationStateValid
}

func (t *ReferenceType) RewriteWithIntersectionTypes() (Type, bool) {
	rewrittenType, rewritten := t.Type.RewriteWithIntersectionTypes()
	if rewritten {
		return &ReferenceType{
			Authorization: t.Authorization,
			Type:          rewrittenType,
		}, true
	} else {
		return t, false
	}
}

func (t *ReferenceType) Map(gauge common.MemoryGauge, typeParamMap map[*TypeParameter]*TypeParameter, f func(Type) Type) Type {
	mappedType := t.Type.Map(gauge, typeParamMap, f)
	return f(NewReferenceType(gauge, t.Authorization, mappedType))
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

func (t *ReferenceType) isTypeIndexableType() bool {
	referencedType, ok := t.Type.(TypeIndexableType)
	return ok && referencedType.isTypeIndexableType()
}

func (t *ReferenceType) TypeIndexingElementType(indexingType Type, _ func() ast.Range) (Type, error) {
	_, ok := t.Type.(TypeIndexableType)
	if !ok {
		return nil, nil
	}

	var access Access = UnauthorizedAccess
	switch indexingType.(type) {
	case *CompositeType:
		// attachment access on a composite reference yields a reference to the attachment entitled to the same
		// entitlements as that reference
		access = t.Authorization
	}

	return &OptionalType{
		Type: &ReferenceType{
			Type:          indexingType,
			Authorization: access,
		},
	}, nil
}

func (t *ReferenceType) IsValidIndexingType(ty Type) bool {
	attachmentType, isComposite := ty.(*CompositeType)
	return isComposite &&
		// we can index into reference types only if their referenced type
		// is a valid base for the attachement;
		// i.e. (&v)[A] is valid only if `v` is a valid base for `A`
		IsSubType(t, &ReferenceType{
			Type:          attachmentType.baseType,
			Authorization: UnauthorizedAccess,
		}) &&
		attachmentType.IsResourceType() == t.Type.IsResourceType()
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

func (t *ReferenceType) Unify(
	other Type,
	typeParameters *TypeParameterTypeOrderedMap,
	report func(err error),
	memoryGauge common.MemoryGauge,
	outerRange ast.HasPosition,
) bool {
	otherReference, ok := other.(*ReferenceType)
	if !ok {
		return false
	}

	return t.Type.Unify(
		otherReference.Type,
		typeParameters,
		report,
		memoryGauge,
		outerRange,
	)
}

func (t *ReferenceType) Resolve(typeArguments *TypeParameterTypeOrderedMap) Type {
	newInnerType := t.Type.Resolve(typeArguments)
	if newInnerType == nil {
		return nil
	}

	return &ReferenceType{
		Authorization: t.Authorization,
		Type:          newInnerType,
	}
}

func (t *ReferenceType) CheckInstantiated(pos ast.HasPosition, memoryGauge common.MemoryGauge, report func(err error)) {
	t.Type.CheckInstantiated(pos, memoryGauge, report)
}

const AddressTypeName = "Address"

// AddressType represents the address type
type AddressType struct {
	memberResolvers                      map[string]MemberResolver
	memberResolversOnce                  sync.Once
	conformances                         []*InterfaceType
	effectiveInterfaceConformanceSet     *InterfaceSet
	effectiveInterfaceConformanceSetOnce sync.Once
}

var TheAddressType = &AddressType{
	conformances: []*InterfaceType{
		StructStringerType,
	},
}
var AddressTypeAnnotation = NewTypeAnnotation(TheAddressType)

var _ Type = &AddressType{}
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

func (*AddressType) IsPrimitiveType() bool {
	return true
}

func (*AddressType) IsInvalidType() bool {
	return false
}

func (*AddressType) IsOrContainsReferenceType() bool {
	return false
}

func (*AddressType) IsStorable(_ map[*Member]bool) bool {
	return true
}

func (*AddressType) IsExportable(_ map[*Member]bool) bool {
	return true
}

func (t *AddressType) IsImportable(_ map[*Member]bool) bool {
	return true
}

func (*AddressType) IsEquatable() bool {
	return true
}

func (*AddressType) IsComparable() bool {
	return false
}

func (*AddressType) ContainFieldsOrElements() bool {
	return false
}

func (*AddressType) TypeAnnotationState() TypeAnnotationState {
	return TypeAnnotationStateValid
}

func (t *AddressType) RewriteWithIntersectionTypes() (Type, bool) {
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

func (*AddressType) Unify(
	_ Type,
	_ *TypeParameterTypeOrderedMap,
	_ func(err error),
	_ common.MemoryGauge,
	_ ast.HasPosition,
) bool {
	return false
}

func (t *AddressType) Resolve(_ *TypeParameterTypeOrderedMap) Type {
	return t
}

func (*AddressType) CheckInstantiated(_ ast.HasPosition, _ common.MemoryGauge, _ func(err error)) {
	// NO-OP
}

const AddressTypeToBytesFunctionName = `toBytes`

var AddressTypeToBytesFunctionType = NewSimpleFunctionType(
	FunctionPurityView,
	nil,
	ByteArrayTypeAnnotation,
)

const addressTypeToBytesFunctionDocString = `
Returns an array containing the byte representation of the address
`

func (t *AddressType) Map(_ common.MemoryGauge, _ map[*TypeParameter]*TypeParameter, f func(Type) Type) Type {
	return f(t)
}

func (t *AddressType) GetMembers() map[string]MemberResolver {
	t.initializeMemberResolvers()
	return t.memberResolvers
}

func (t *AddressType) initializeMemberResolvers() {
	t.memberResolversOnce.Do(func() {
		memberResolvers := MembersAsResolvers([]*Member{
			NewUnmeteredPublicFunctionMember(
				t,
				AddressTypeToBytesFunctionName,
				AddressTypeToBytesFunctionType,
				addressTypeToBytesFunctionDocString,
			),
		})
		t.memberResolvers = withBuiltinMembers(t, memberResolvers)
	})
}

func (t *AddressType) EffectiveInterfaceConformanceSet() *InterfaceSet {
	t.initializeEffectiveInterfaceConformanceSet()
	return t.effectiveInterfaceConformanceSet
}

func (t *AddressType) initializeEffectiveInterfaceConformanceSet() {
	t.effectiveInterfaceConformanceSetOnce.Do(func() {
		t.effectiveInterfaceConformanceSet = NewInterfaceSet()

		for _, conformance := range t.conformances {
			t.effectiveInterfaceConformanceSet.Add(conformance)
		}
	})
}

func IsPrimitiveOrContainerOfPrimitive(referencedType Type) bool {
	switch ty := referencedType.(type) {
	case *VariableSizedType:
		return IsPrimitiveOrContainerOfPrimitive(ty.Type)

	case *ConstantSizedType:
		return IsPrimitiveOrContainerOfPrimitive(ty.Type)

	case *DictionaryType:
		return IsPrimitiveOrContainerOfPrimitive(ty.KeyType) &&
			IsPrimitiveOrContainerOfPrimitive(ty.ValueType)

	default:
		return ty.IsPrimitiveType()
	}
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
//
// The differences between these methods is as follows:
//
//   - IsSubType():
//
//     To check the assignability, e.g: is argument type T is a sub-type
//     of parameter type R? This is the more frequent use-case.
//
//   - IsSameTypeKind():
//
//     To check if a type strictly belongs to a certain category. e.g: Is the
//     expression type T is any of the integer types, but nothing else.
//     Another way to check is, asking the question of "if the subType is Never,
//     should the check still pass?". A common code-smell for potential incorrect
//     usage is, using IsSubType() method with a constant/pre-defined superType.
//     e.g: IsSubType(<<someType>>, FixedPointType)
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

	case AnyResourceAttachmentType:
		return subType.IsResourceType() && isAttachmentType(subType)

	case AnyStructAttachmentType:
		return !subType.IsResourceType() && isAttachmentType(subType)

	case HashableStructType:
		return IsHashableStructType(subType)

	case PathType:
		return IsSubType(subType, StoragePathType) ||
			IsSubType(subType, CapabilityPathType)

	case StorableType:
		storableResults := map[*Member]bool{}
		return subType.IsStorable(storableResults)

	case CapabilityPathType:
		return IsSubType(subType, PrivatePathType) ||
			IsSubType(subType, PublicPathType)

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
		case IntegerType, SignedIntegerType, FixedSizeUnsignedIntegerType,
			UIntType:

			return true

		default:
			return IsSubType(subType, SignedIntegerType) || IsSubType(subType, FixedSizeUnsignedIntegerType)
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

	case FixedSizeUnsignedIntegerType:
		switch subType {
		case UInt8Type, UInt16Type, UInt32Type, UInt64Type, UInt128Type, UInt256Type,
			Word8Type, Word16Type, Word32Type, Word64Type, Word128Type, Word256Type:

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
		typedSubType, ok := subType.(*ReferenceType)
		if !ok {
			return false
		}

		// the authorization of the subtype reference must be usable in all situations where the supertype reference is usable
		if !typedSuperType.Authorization.PermitsAccess(typedSubType.Authorization) {
			return false
		}

		// references are covariant in their referenced type
		return IsSubType(typedSubType.Type, typedSuperType.Type)

	case *FunctionType:
		typedSubType, ok := subType.(*FunctionType)
		if !ok {
			return false
		}

		// view functions are subtypes of impure functions
		if typedSubType.Purity != typedSuperType.Purity && typedSubType.Purity != FunctionPurityView {
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

		if typedSubType.ReturnTypeAnnotation.Type != nil {
			if typedSuperType.ReturnTypeAnnotation.Type == nil {
				return false
			}

			if !IsSubType(
				typedSubType.ReturnTypeAnnotation.Type,
				typedSuperType.ReturnTypeAnnotation.Type,
			) {
				return false
			}
		} else if typedSuperType.ReturnTypeAnnotation.Type != nil {
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

	case *IntersectionType:

		// TODO: replace with
		//
		//switch typedSubType := subType.(type) {
		//case *IntersectionType:
		//
		//	// An intersection type `{Us}` is a subtype of an intersection type `{Vs}` / `{Vs}` / `{Vs}`:
		//	// when `Vs` is a subset of `Us`.
		//
		//	return typedSuperType.EffectiveIntersectionSet().
		//		IsSubsetOf(typedSubType.EffectiveIntersectionSet())
		//
		//case *CompositeType:
		//	// A type `T` is a subtype of an intersection type `{Us}` / `{Us}` / `{Us}`:
		//	// when `T` conforms to `Us`.
		//
		//	return typedSuperType.EffectiveIntersectionSet().
		//		IsSubsetOf(typedSubType.EffectiveInterfaceConformanceSet())
		//}

		intersectionSuperType := typedSuperType.LegacyType //nolint:staticcheck

		switch intersectionSuperType {
		case nil, AnyResourceType, AnyStructType, AnyType:

			switch subType {
			case AnyResourceType:
				// `AnyResource` is a subtype of an intersection type
				// - `AnyResource{Us}`: not statically;
				// - `AnyStruct{Us}`: never.
				// - `Any{Us}`: not statically;

				return false

			case AnyStructType:
				// `AnyStruct` is a subtype of an intersection type
				// - `AnyStruct{Us}`: not statically.
				// - `AnyResource{Us}`: never;
				// - `Any{Us}`: not statically.

				return false

			case AnyType:
				// `Any` is a subtype of an intersection type
				// - `Any{Us}: not statically.`
				// - `AnyStruct{Us}`: never;
				// - `AnyResource{Us}`: never;

				return false
			}

			switch typedSubType := subType.(type) {
			case *IntersectionType:

				// An intersection type `T{Us}`
				// is a subtype of an intersection type `AnyResource{Vs}` / `AnyStruct{Vs}` / `Any{Vs}`:

				intersectionSubtype := typedSubType.LegacyType //nolint:staticcheck
				switch intersectionSubtype {
				case nil:
					// An intersection type `{Us}` is a subtype of an intersection type `{Vs}` / `{Vs}` / `{Vs}`:
					// when `Vs` is a subset of `Us`.

					return typedSuperType.EffectiveIntersectionSet().
						IsSubsetOf(typedSubType.EffectiveIntersectionSet())

				case AnyResourceType, AnyStructType, AnyType:
					// When `T == AnyResource || T == AnyStruct || T == Any`:
					// if the intersection type of the subtype
					// is a subtype of the intersection supertype,
					// and `Vs` is a subset of `Us`.

					if intersectionSuperType != nil &&
						!IsSubType(intersectionSubtype, intersectionSuperType) {

						return false
					}

					return typedSuperType.EffectiveIntersectionSet().
						IsSubsetOf(typedSubType.EffectiveIntersectionSet())
				}

				if intersectionSubtype, ok := intersectionSubtype.(*CompositeType); ok {
					// When `T != AnyResource && T != AnyStruct && T != Any`:
					// if the intersection type of the subtype
					// is a subtype of the intersection supertype,
					// and `T` conforms to `Vs`.
					// `Us` and `Vs` do *not* have to be subsets.

					if intersectionSuperType != nil &&
						!IsSubType(intersectionSubtype, intersectionSuperType) {

						return false
					}

					return typedSuperType.EffectiveIntersectionSet().
						IsSubsetOf(intersectionSubtype.EffectiveInterfaceConformanceSet())
				}

			case ConformingType:
				// A type `T`
				// is a subtype of an intersection type `AnyResource{Us}` / `AnyStruct{Us}` / `Any{Us}`:
				// if `T` is a subtype of the intersection supertype,
				// and `T` conforms to `Us`.

				if intersectionSuperType != nil &&
					!IsSubType(typedSubType, intersectionSuperType) {

					return false
				}

				return typedSuperType.EffectiveIntersectionSet().
					IsSubsetOf(typedSubType.EffectiveInterfaceConformanceSet())
			}

		default:
			// Supertype (intersection) has a non-Any* legacy type

			switch typedSubType := subType.(type) {
			case *IntersectionType:

				// An intersection type `T{Us}`
				// is a subtype of an intersection type `V{Ws}`:

				intersectionSubType := typedSubType.LegacyType //nolint:staticcheck
				switch intersectionSubType {
				case nil, AnyResourceType, AnyStructType, AnyType:
					// When `T == AnyResource || T == AnyStruct || T == Any`:
					// not statically.
					return false
				}

				if intersectionSubType, ok := intersectionSubType.(*CompositeType); ok {
					// When `T != AnyResource && T != AnyStructType && T != Any`: if `T == V`.
					//
					// `Us` and `Ws` do *not* have to be subsets:
					// The owner may freely restrict and unrestrict.

					return intersectionSubType == intersectionSuperType
				}

			case *CompositeType:
				// A type `T`
				// is a subtype of an intersection type `U{Vs}`: if `T <: U`.
				//
				// The owner may freely restrict.

				return IsSubType(typedSubType, intersectionSuperType)
			}

			switch subType {
			case AnyResourceType, AnyStructType, AnyType:
				// A type `T`
				// is a subtype of an intersection type `AnyResource{Vs}` / `AnyStruct{Vs}` / `Any{Vs}`:
				// not statically.

				return false
			}
		}

	case *CompositeType:

		// NOTE: type equality case (composite type `T` is subtype of composite type `U`)
		// is already handled at beginning of function

		switch typedSubType := subType.(type) {
		case *IntersectionType:

			// TODO: bring back once legacy type is removed
			// An intersection type `{Us}` is never a subtype of a type `V`:
			//return false

			// TODO: remove support for legacy type
			// An intersection type `T{Us}`
			// is a subtype of a type `V`:

			legacyType := typedSubType.LegacyType
			switch legacyType {
			case nil, AnyResourceType, AnyStructType, AnyType:
				// When `T == AnyResource || T == AnyStruct || T == Any`: not statically.
				return false
			}

			if intersectionSubType, ok := legacyType.(*CompositeType); ok {
				// When `T != AnyResource && T != AnyStruct`: if `T == V`.
				//
				// The owner may freely unrestrict.

				return intersectionSubType == typedSuperType
			}

		case *CompositeType:
			// Non-equal composite types are never subtypes of each other
			return false
		}

	case *InterfaceType:

		switch typedSubType := subType.(type) {
		case *CompositeType:

			// A composite type `T` is a subtype of an interface type `V`:
			// if `T` conforms to `V`, and `V` and `T` are of the same kind

			if typedSubType.Kind != typedSuperType.CompositeKind {
				return false
			}

			return typedSubType.EffectiveInterfaceConformanceSet().
				Contains(typedSuperType)

		// An interface type is a supertype of an intersection type if at least one value
		// in the intersection set is a subtype of the interface supertype.
		//
		// This particular case comes up when checking attachment access;
		// enabling the following expression to type-checking:
		//
		//   resource interface I { /* ... */ }
		//   attachment A for I { /* ... */ }
		//
		//   let i : {I} = ... // some operation constructing `i`
		//   let a = i[A] // must here check that `i`'s type is a subtype of `A`'s base type, or that {I} <: I
		case *IntersectionType:
			return typedSubType.EffectiveIntersectionSet().Contains(typedSuperType)

		case *InterfaceType:
			return typedSubType.EffectiveInterfaceConformanceSet().
				Contains(typedSuperType)
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
	Fields              []string
	PrepareParameters   []Parameter
	Parameters          []Parameter
	Members             *StringMemberOrderedMap
	memberResolvers     map[string]MemberResolver
	memberResolversOnce sync.Once
}

var _ Type = &TransactionType{}

func (t *TransactionType) EntryPointFunctionType() *FunctionType {
	return NewSimpleFunctionType(
		FunctionPurityImpure,
		append(t.Parameters, t.PrepareParameters...),
		VoidTypeAnnotation,
	)
}

func (t *TransactionType) PrepareFunctionType() *FunctionType {
	return &FunctionType{
		Purity:               FunctionPurityImpure,
		IsConstructor:        true,
		Parameters:           t.PrepareParameters,
		ReturnTypeAnnotation: VoidTypeAnnotation,
	}
}

var transactionTypeExecuteFunctionType = &FunctionType{
	Purity:               FunctionPurityImpure,
	IsConstructor:        true,
	ReturnTypeAnnotation: VoidTypeAnnotation,
}

func (*TransactionType) ExecuteFunctionType() *FunctionType {
	return transactionTypeExecuteFunctionType
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

func (*TransactionType) IsPrimitiveType() bool {
	return false
}

func (*TransactionType) IsInvalidType() bool {
	return false
}

func (*TransactionType) IsOrContainsReferenceType() bool {
	return false
}

func (*TransactionType) IsStorable(_ map[*Member]bool) bool {
	return false
}

func (*TransactionType) IsExportable(_ map[*Member]bool) bool {
	return false
}

func (t *TransactionType) IsImportable(_ map[*Member]bool) bool {
	return false
}

func (*TransactionType) IsEquatable() bool {
	return false
}

func (*TransactionType) IsComparable() bool {
	return false
}

func (*TransactionType) ContainFieldsOrElements() bool {
	return false
}

func (*TransactionType) TypeAnnotationState() TypeAnnotationState {
	return TypeAnnotationStateValid
}

func (t *TransactionType) RewriteWithIntersectionTypes() (Type, bool) {
	return t, false
}

func (t *TransactionType) Map(_ common.MemoryGauge, _ map[*TypeParameter]*TypeParameter, f func(Type) Type) Type {
	return f(t)
}

func (t *TransactionType) GetMembers() map[string]MemberResolver {
	t.initializeMemberResolvers()
	return t.memberResolvers
}

func (t *TransactionType) initializeMemberResolvers() {
	t.memberResolversOnce.Do(func() {
		var memberResolvers map[string]MemberResolver
		if t.Members != nil {
			memberResolvers = MembersMapAsResolvers(t.Members)
		}
		t.memberResolvers = withBuiltinMembers(t, memberResolvers)
	})
}

func (*TransactionType) Unify(
	_ Type,
	_ *TypeParameterTypeOrderedMap,
	_ func(err error),
	_ common.MemoryGauge,
	_ ast.HasPosition,
) bool {
	return false
}

func (t *TransactionType) Resolve(_ *TypeParameterTypeOrderedMap) Type {
	return t
}

func (t *TransactionType) CheckInstantiated(pos ast.HasPosition, memoryGauge common.MemoryGauge, report func(err error)) {
	for _, param := range t.PrepareParameters {
		param.TypeAnnotation.Type.CheckInstantiated(pos, memoryGauge, report)
	}

	for _, param := range t.Parameters {
		param.TypeAnnotation.Type.CheckInstantiated(pos, memoryGauge, report)
	}
}

// IntersectionType

type IntersectionType struct {
	// an internal set of field `Types`
	effectiveIntersectionSet     *InterfaceSet
	Types                        []*InterfaceType
	effectiveIntersectionSetOnce sync.Once
	memberResolvers              map[string]MemberResolver
	memberResolversOnce          sync.Once
	supportedEntitlementsOnce    sync.Once
	supportedEntitlements        *EntitlementSet
	// Deprecated
	LegacyType Type
}

var _ Type = &IntersectionType{}

// TODO: remove `legacyType` once all uses of it are removed
func NewIntersectionType(memoryGauge common.MemoryGauge, legacyType Type, types []*InterfaceType) *IntersectionType {
	if len(types) == 0 && legacyType == nil {
		panic(errors.NewUnreachableError())
	}

	common.UseMemory(memoryGauge, common.IntersectionSemaTypeMemoryUsage)

	// Also meter the cost for the `effectiveIntersectionSet` here, since ordered maps are not separately metered.
	wrapperUsage, entryListUsage, entriesUsage := common.NewOrderedMapMemoryUsages(uint64(len(types)))
	common.UseMemory(memoryGauge, wrapperUsage)
	common.UseMemory(memoryGauge, entryListUsage)
	common.UseMemory(memoryGauge, entriesUsage)

	return &IntersectionType{
		Types:      types,
		LegacyType: legacyType, //nolint:staticcheck
	}
}

func (t *IntersectionType) EffectiveIntersectionSet() *InterfaceSet {
	t.initializeEffectiveIntersectionSet()
	return t.effectiveIntersectionSet
}

func (t *IntersectionType) initializeEffectiveIntersectionSet() {
	t.effectiveIntersectionSetOnce.Do(func() {
		t.effectiveIntersectionSet = NewInterfaceSet()
		for _, typ := range t.Types {
			t.effectiveIntersectionSet.Add(typ)

			// Also add the interfaces to which this restricting interface conforms.
			for _, conformance := range typ.EffectiveInterfaceConformances() {
				t.effectiveIntersectionSet.Add(conformance.InterfaceType)
			}
		}
	})
}

func (*IntersectionType) IsType() {}

func (t *IntersectionType) Tag() TypeTag {
	return IntersectionTypeTag
}

func formatIntersectionType[T ~string](separator string, interfaceStrings []T) string {
	var result strings.Builder
	result.WriteByte('{')
	for i, interfaceString := range interfaceStrings {
		if i > 0 {
			result.WriteByte(',')
			result.WriteString(separator)
		}
		result.WriteString(string(interfaceString))
	}
	result.WriteByte('}')
	return result.String()
}

func FormatIntersectionTypeID[T ~string](interfaceTypeIDs []T) T {
	slices.Sort(interfaceTypeIDs)
	return T(formatIntersectionType("", interfaceTypeIDs))
}

func (t *IntersectionType) string(separator string, typeFormatter func(Type) string) string {
	var intersectionStrings []string
	typeCount := len(t.Types)
	if typeCount > 0 {
		intersectionStrings = make([]string, 0, typeCount)
		for _, typ := range t.Types {
			intersectionStrings = append(intersectionStrings, typeFormatter(typ))
		}
	}
	return formatIntersectionType(separator, intersectionStrings)
}

func (t *IntersectionType) String() string {
	return t.string(" ", func(ty Type) string {
		return ty.String()
	})
}

func (t *IntersectionType) QualifiedString() string {
	return t.string(" ", func(ty Type) string {
		return ty.QualifiedString()
	})
}

func (t *IntersectionType) ID() TypeID {
	var interfaceTypeIDs []TypeID
	typeCount := len(t.Types)
	if typeCount > 0 {
		interfaceTypeIDs = make([]TypeID, 0, typeCount)
		for _, typ := range t.Types {
			interfaceTypeIDs = append(interfaceTypeIDs, typ.ID())
		}
	}
	// FormatIntersectionTypeID sorts
	return FormatIntersectionTypeID(interfaceTypeIDs)
}

func (t *IntersectionType) Equal(other Type) bool {
	otherIntersectionType, ok := other.(*IntersectionType)
	if !ok {
		return false
	}

	// Check that the set of types are equal; order does not matter

	intersectionSet := t.EffectiveIntersectionSet()
	otherIntersectionSet := otherIntersectionType.EffectiveIntersectionSet()

	if intersectionSet.Len() != otherIntersectionSet.Len() {
		return false
	}

	return intersectionSet.IsSubsetOf(otherIntersectionSet)
}

func (t *IntersectionType) IsResourceType() bool {
	// intersections are guaranteed to have all their interfaces be the same kind
	return t.Types[0].IsResourceType()
}

func (*IntersectionType) IsPrimitiveType() bool {
	return false
}

func (t *IntersectionType) IsInvalidType() bool {
	for _, typ := range t.Types {
		if typ.IsInvalidType() {
			return true
		}
	}

	return false
}

func (t *IntersectionType) IsOrContainsReferenceType() bool {
	for _, typ := range t.Types {
		if typ.IsOrContainsReferenceType() {
			return true
		}
	}

	return false
}

func (t *IntersectionType) IsStorable(results map[*Member]bool) bool {
	for _, typ := range t.Types {
		if !typ.IsStorable(results) {
			return false
		}
	}

	return true
}

func (t *IntersectionType) IsExportable(results map[*Member]bool) bool {
	for _, typ := range t.Types {
		if !typ.IsExportable(results) {
			return false
		}
	}

	return true
}

func (t *IntersectionType) IsImportable(results map[*Member]bool) bool {
	for _, typ := range t.Types {
		if !typ.IsImportable(results) {
			return false
		}
	}

	return true
}

func (*IntersectionType) IsEquatable() bool {
	// TODO:
	return false
}

func (t *IntersectionType) IsComparable() bool {
	return false
}

func (*IntersectionType) ContainFieldsOrElements() bool {
	return true
}

func (*IntersectionType) TypeAnnotationState() TypeAnnotationState {
	return TypeAnnotationStateValid
}

func (t *IntersectionType) RewriteWithIntersectionTypes() (Type, bool) {
	// Even though the types should be resource interfaces,
	// they are not on the "first level", i.e. not the intersection type
	return t, false
}

func (t *IntersectionType) Map(gauge common.MemoryGauge, typeParamMap map[*TypeParameter]*TypeParameter, f func(Type) Type) Type {
	var intersectionTypes []*InterfaceType
	if len(t.Types) > 0 {
		intersectionTypes = make([]*InterfaceType, 0, len(t.Types))
		for _, typ := range t.Types {
			mapped := typ.Map(gauge, typeParamMap, f)
			if mappedType, isInterface := mapped.(*InterfaceType); isInterface {
				intersectionTypes = append(intersectionTypes, mappedType)
			} else {
				panic(errors.NewUnexpectedError(fmt.Sprintf("intersection mapped to non-interface type %T", mapped)))
			}
		}
	}

	return f(NewIntersectionType(
		gauge,
		t.LegacyType, //nolint:staticcheck
		intersectionTypes,
	))
}

func (t *IntersectionType) GetMembers() map[string]MemberResolver {
	t.initializeMemberResolvers()
	return t.memberResolvers
}

func (t *IntersectionType) initializeMemberResolvers() {
	t.memberResolversOnce.Do(func() {

		memberResolvers := map[string]MemberResolver{}

		// Return the members of all types.
		// The invariant that types may not have overlapping members is not checked here,
		// but implicitly when the resource declaration's conformances are checked.

		for _, typ := range t.Types {
			for name, resolver := range typ.GetMembers() { //nolint:maprange
				if _, ok := memberResolvers[name]; !ok {
					memberResolvers[name] = resolver
				}
			}
		}

		t.memberResolvers = memberResolvers
	})
}

func (t *IntersectionType) SupportedEntitlements() *EntitlementSet {
	t.supportedEntitlementsOnce.Do(func() {
		// an intersection type supports all the entitlements of its interfaces
		set := &EntitlementSet{}
		t.EffectiveIntersectionSet().
			ForEach(func(interfaceType *InterfaceType) {
				set.Merge(interfaceType.SupportedEntitlements())
			})
		t.supportedEntitlements = set
	})

	return t.supportedEntitlements
}

func (*IntersectionType) Unify(
	_ Type,
	_ *TypeParameterTypeOrderedMap,
	_ func(err error),
	_ common.MemoryGauge,
	_ ast.HasPosition,
) bool {
	// TODO: how do we unify the intersection sets?
	return false
}

func (t *IntersectionType) Resolve(_ *TypeParameterTypeOrderedMap) Type {
	// TODO:
	return t
}

// Intersection types must be type indexable, because this is how we handle access control for attachments.
// Specifically, because in `v[A]`, `v` must be a subtype of `A`'s declared base,
// if `v` is an intersection type `{I}`, only attachments declared for `I` or a supertype can be accessed on `v`.
//
// Attachments declared for concrete types implementing `I` cannot be accessed.
//
// A good elucidating example here is that an attachment declared for `Vault`
// cannot be accessed on a value of type `&{Provider}`
func (t *IntersectionType) isTypeIndexableType() bool {
	// resources and structs only can be indexed for attachments, but all intersection types
	// are necessarily structs and resources, we return true
	return true
}

func (t *IntersectionType) TypeIndexingElementType(indexingType Type, _ func() ast.Range) (Type, error) {
	var access Access = UnauthorizedAccess
	switch attachment := indexingType.(type) {
	case *CompositeType:
		// when accessed on an owned value, the produced attachment reference is entitled to all the
		// entitlements it supports
		access = attachment.SupportedEntitlements().Access()
	}

	return &OptionalType{
		Type: &ReferenceType{
			Type:          indexingType,
			Authorization: access,
		},
	}, nil
}

func (t *IntersectionType) IsValidIndexingType(ty Type) bool {
	attachmentType, isComposite := ty.(*CompositeType)
	return isComposite &&
		IsSubType(t, attachmentType.baseType) &&
		attachmentType.IsResourceType() == t.IsResourceType()
}

func (t *IntersectionType) CheckInstantiated(_ ast.HasPosition, _ common.MemoryGauge, _ func(err error)) {
	// No-OP
}

// CapabilityType

type CapabilityType struct {
	BorrowType          Type
	memberResolvers     map[string]MemberResolver
	memberResolversOnce sync.Once
}

var _ Type = &CapabilityType{}
var _ ParameterizedType = &CapabilityType{}

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

func formatCapabilityType[T ~string](borrowTypeString T) string {
	var builder strings.Builder
	builder.WriteString("Capability")
	if borrowTypeString != "" {
		builder.WriteByte('<')
		builder.WriteString(string(borrowTypeString))
		builder.WriteByte('>')
	}
	return builder.String()
}

func FormatCapabilityTypeID[T ~string](borrowTypeID T) T {
	return T(formatCapabilityType(borrowTypeID))
}

func (t *CapabilityType) String() string {
	var borrowTypeString string
	borrowType := t.BorrowType
	if borrowType != nil {
		borrowTypeString = borrowType.String()
	}
	return formatCapabilityType(borrowTypeString)
}

func (t *CapabilityType) QualifiedString() string {
	var borrowTypeString string
	borrowType := t.BorrowType
	if borrowType != nil {
		borrowTypeString = borrowType.QualifiedString()
	}
	return formatCapabilityType(borrowTypeString)
}

func (t *CapabilityType) ID() TypeID {
	var borrowTypeID TypeID
	borrowType := t.BorrowType
	if borrowType != nil {
		borrowTypeID = borrowType.ID()
	}
	return FormatCapabilityTypeID(borrowTypeID)
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

func (*CapabilityType) IsPrimitiveType() bool {
	return false
}

func (t *CapabilityType) IsInvalidType() bool {
	if t.BorrowType == nil {
		return false
	}
	return t.BorrowType.IsInvalidType()

}

func (t *CapabilityType) IsOrContainsReferenceType() bool {
	if t.BorrowType == nil {
		return false
	}
	return t.BorrowType.IsOrContainsReferenceType()
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

func (*CapabilityType) IsExportable(_ map[*Member]bool) bool {
	return true
}

func (t *CapabilityType) IsImportable(_ map[*Member]bool) bool {
	return true
}

func (*CapabilityType) IsEquatable() bool {
	// TODO:
	return false
}

func (*CapabilityType) IsComparable() bool {
	return false
}

func (*CapabilityType) ContainFieldsOrElements() bool {
	return false
}

func (t *CapabilityType) RewriteWithIntersectionTypes() (Type, bool) {
	if t.BorrowType == nil {
		return t, false
	}
	rewrittenType, rewritten := t.BorrowType.RewriteWithIntersectionTypes()
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
	memoryGauge common.MemoryGauge,
	outerRange ast.HasPosition,
) bool {
	otherCap, ok := other.(*CapabilityType)
	if !ok {
		return false
	}

	if t.BorrowType == nil {
		return false
	}

	return t.BorrowType.Unify(
		otherCap.BorrowType,
		typeParameters,
		report,
		memoryGauge,
		outerRange,
	)
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
		Type:          AnyType,
		Authorization: UnauthorizedAccess,
	},
}

func (t *CapabilityType) TypeParameters() []*TypeParameter {
	return []*TypeParameter{
		capabilityTypeParameter,
	}
}

func (t *CapabilityType) Instantiate(
	_ common.MemoryGauge,
	typeArguments []Type,
	_ []*ast.TypeAnnotation,
	_ func(err error),
) Type {
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
			Type:          AnyType,
			Authorization: UnauthorizedAccess,
		}
	}
	return []Type{
		borrowType,
	}
}

func (t *CapabilityType) CheckInstantiated(pos ast.HasPosition, memoryGauge common.MemoryGauge, report func(err error)) {
	CheckParameterizedTypeInstantiated(t, pos, memoryGauge, report)
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
		Purity:         FunctionPurityView,
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
		Purity:               FunctionPurityView,
		TypeParameters:       typeParameters,
		ReturnTypeAnnotation: BoolTypeAnnotation,
	}
}

const CapabilityTypeBorrowFunctionName = "borrow"

const capabilityTypeBorrowFunctionDocString = `
Returns a reference to the targeted object.

If the capability is revoked, the function returns nil.

If the capability targets an object in account storage,
and and no object is stored at the target storage path,
the function returns nil.

If the targeted object cannot be borrowed using the given type,
the function panics.
`

const CapabilityTypeCheckFunctionName = "check"

const capabilityTypeCheckFunctionDocString = `
Returns true if the capability currently targets an object that satisfies the given type,
i.e. could be borrowed using the given type
`

var CapabilityTypeAddressFieldType = TheAddressType

const CapabilityTypeAddressFieldName = "address"

const capabilityTypeAddressFieldDocString = `
The address of the account which the capability targets.
`

func (t *CapabilityType) Map(gauge common.MemoryGauge, typeParamMap map[*TypeParameter]*TypeParameter, f func(Type) Type) Type {
	var borrowType Type
	if t.BorrowType != nil {
		borrowType = t.BorrowType.Map(gauge, typeParamMap, f)
	}

	return f(NewCapabilityType(gauge, borrowType))
}

var CapabilityTypeIDFieldType = UInt64Type

const CapabilityTypeIDFieldName = "id"

const capabilityTypeIDFieldDocString = `
The ID of the capability
`

func (t *CapabilityType) GetMembers() map[string]MemberResolver {
	t.initializeMemberResolvers()
	return t.memberResolvers
}

func (t *CapabilityType) initializeMemberResolvers() {
	t.memberResolversOnce.Do(func() {
		members := MembersAsResolvers([]*Member{
			NewUnmeteredPublicFunctionMember(
				t,
				CapabilityTypeBorrowFunctionName,
				CapabilityTypeBorrowFunctionType(t.BorrowType),
				capabilityTypeBorrowFunctionDocString,
			),
			NewUnmeteredPublicFunctionMember(
				t,
				CapabilityTypeCheckFunctionName,
				CapabilityTypeCheckFunctionType(t.BorrowType),
				capabilityTypeCheckFunctionDocString,
			),
			NewUnmeteredPublicConstantFieldMember(
				t,
				CapabilityTypeAddressFieldName,
				CapabilityTypeAddressFieldType,
				capabilityTypeAddressFieldDocString,
			),
			NewUnmeteredPublicConstantFieldMember(
				t,
				CapabilityTypeIDFieldName,
				CapabilityTypeIDFieldType,
				capabilityTypeIDFieldDocString,
			),
		})
		t.memberResolvers = withBuiltinMembers(t, members)
	})
}

const AccountKeyTypeName = "AccountKey"
const AccountKeyKeyIndexFieldName = "keyIndex"
const AccountKeyPublicKeyFieldName = "publicKey"
const AccountKeyHashAlgoFieldName = "hashAlgorithm"
const AccountKeyWeightFieldName = "weight"
const AccountKeyIsRevokedFieldName = "isRevoked"

// AccountKeyType represents the key associated with an account.
var AccountKeyType = func() *CompositeType {

	accountKeyType := &CompositeType{
		Identifier:        AccountKeyTypeName,
		Kind:              common.CompositeKindStructure,
		ImportableBuiltin: false,
	}

	const accountKeyKeyIndexFieldDocString = `The index of the account key`
	const accountKeyPublicKeyFieldDocString = `The public key of the account`
	const accountKeyHashAlgorithmFieldDocString = `The hash algorithm used by the public key`
	const accountKeyWeightFieldDocString = `The weight assigned to the public key`
	const accountKeyIsRevokedFieldDocString = `Flag indicating whether the key is revoked`

	var members = []*Member{
		NewUnmeteredPublicConstantFieldMember(
			accountKeyType,
			AccountKeyKeyIndexFieldName,
			IntType,
			accountKeyKeyIndexFieldDocString,
		),
		NewUnmeteredPublicConstantFieldMember(
			accountKeyType,
			AccountKeyPublicKeyFieldName,
			PublicKeyType,
			accountKeyPublicKeyFieldDocString,
		),
		NewUnmeteredPublicConstantFieldMember(
			accountKeyType,
			AccountKeyHashAlgoFieldName,
			HashAlgorithmType,
			accountKeyHashAlgorithmFieldDocString,
		),
		NewUnmeteredPublicConstantFieldMember(
			accountKeyType,
			AccountKeyWeightFieldName,
			UFix64Type,
			accountKeyWeightFieldDocString,
		),
		NewUnmeteredPublicConstantFieldMember(
			accountKeyType,
			AccountKeyIsRevokedFieldName,
			BoolType,
			accountKeyIsRevokedFieldDocString,
		),
	}

	accountKeyType.Members = MembersAsMap(members)
	accountKeyType.Fields = MembersFieldNames(members)
	return accountKeyType
}()

var AccountKeyTypeAnnotation = NewTypeAnnotation(AccountKeyType)

const PublicKeyTypeName = "PublicKey"
const PublicKeyTypePublicKeyFieldName = "publicKey"
const PublicKeyTypeSignAlgoFieldName = "signatureAlgorithm"
const PublicKeyTypeVerifyFunctionName = "verify"
const PublicKeyTypeVerifyPoPFunctionName = "verifyPoP"

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
		HasComputedMembers: true,
		ImportableBuiltin:  true,
	}

	var members = []*Member{
		NewUnmeteredPublicConstantFieldMember(
			publicKeyType,
			PublicKeyTypePublicKeyFieldName,
			ByteArrayType,
			publicKeyKeyFieldDocString,
		),
		NewUnmeteredPublicConstantFieldMember(
			publicKeyType,
			PublicKeyTypeSignAlgoFieldName,
			SignatureAlgorithmType,
			publicKeySignAlgoFieldDocString,
		),
		NewUnmeteredPublicFunctionMember(
			publicKeyType,
			PublicKeyTypeVerifyFunctionName,
			PublicKeyVerifyFunctionType,
			publicKeyVerifyFunctionDocString,
		),
		NewUnmeteredPublicFunctionMember(
			publicKeyType,
			PublicKeyTypeVerifyPoPFunctionName,
			PublicKeyVerifyPoPFunctionType,
			publicKeyVerifyPoPFunctionDocString,
		),
	}

	publicKeyType.Members = MembersAsMap(members)
	publicKeyType.Fields = MembersFieldNames(members)

	return publicKeyType
}()

var PublicKeyTypeAnnotation = NewTypeAnnotation(PublicKeyType)

var PublicKeyArrayType = &VariableSizedType{
	Type: PublicKeyType,
}

var PublicKeyArrayTypeAnnotation = NewTypeAnnotation(PublicKeyArrayType)

var PublicKeyVerifyFunctionType = NewSimpleFunctionType(
	FunctionPurityView,
	[]Parameter{
		{
			Identifier:     "signature",
			TypeAnnotation: ByteArrayTypeAnnotation,
		},
		{
			Identifier:     "signedData",
			TypeAnnotation: ByteArrayTypeAnnotation,
		},
		{
			Identifier:     "domainSeparationTag",
			TypeAnnotation: StringTypeAnnotation,
		},
		{
			Identifier:     "hashAlgorithm",
			TypeAnnotation: HashAlgorithmTypeAnnotation,
		},
	},
	BoolTypeAnnotation,
)

var PublicKeyVerifyPoPFunctionType = NewSimpleFunctionType(
	FunctionPurityView,
	[]Parameter{
		{
			Label:          ArgumentLabelNotRequired,
			Identifier:     "proof",
			TypeAnnotation: ByteArrayTypeAnnotation,
		},
	},
	BoolTypeAnnotation,
)

type CryptoAlgorithm interface {
	RawValue() uint8
	Name() string
	DocString() string
}

func MembersAsMap(members []*Member) *StringMemberOrderedMap {
	membersMap := &StringMemberOrderedMap{}
	for _, member := range members {
		name := member.Identifier.Identifier
		if membersMap.Contains(name) {
			panic(errors.NewUnexpectedError("invalid duplicate member: %s", name))
		}
		membersMap.Set(name, member)
	}

	return membersMap
}

func MembersFieldNames(members []*Member) []string {
	var fields []string
	for _, member := range members {
		if member.DeclarationKind == common.DeclarationKindField {
			fields = append(fields, member.Identifier.Identifier)
		}
	}

	return fields
}

func MembersMapAsResolvers(members *StringMemberOrderedMap) map[string]MemberResolver {
	resolvers := make(map[string]MemberResolver, members.Len())

	members.Foreach(func(name string, member *Member) {
		resolvers[name] = MemberResolver{
			Kind: member.DeclarationKind,
			Resolve: func(_ common.MemoryGauge, _ string, _ ast.HasPosition, _ func(error)) *Member {
				return member
			},
		}
	})
	return resolvers
}

func MembersAsResolvers(members []*Member) map[string]MemberResolver {
	resolvers := make(map[string]MemberResolver, len(members))

	for _, loopMember := range members {
		// NOTE: don't capture loop variable
		member := loopMember
		resolvers[member.Identifier.Identifier] = MemberResolver{
			Kind: member.DeclarationKind,
			Resolve: func(_ common.MemoryGauge, _ string, _ ast.HasPosition, _ func(error)) *Member {
				return member
			},
		}
	}
	return resolvers
}

func isNumericSuperType(typ Type) bool {
	if numberType, ok := typ.(IntegerRangedType); ok {
		return numberType.IsSuperType()
	}

	return false
}

// EntitlementType

type EntitlementType struct {
	Location      common.Location
	containerType Type
	Identifier    string
	isInvalid     bool
}

var _ Type = &EntitlementType{}
var _ ContainedType = &EntitlementType{}
var _ LocatedType = &EntitlementType{}

func NewEntitlementType(memoryGauge common.MemoryGauge, location common.Location, identifier string) *EntitlementType {
	common.UseMemory(memoryGauge, common.EntitlementSemaTypeMemoryUsage)
	return &EntitlementType{
		Location:   location,
		Identifier: identifier,
	}
}

func (*EntitlementType) IsType() {}

func (t *EntitlementType) Tag() TypeTag {
	return InvalidTypeTag // entitlement types may never appear as types, and thus cannot have a computed supertype
}

func (t *EntitlementType) String() string {
	return t.Identifier
}

func (t *EntitlementType) QualifiedString() string {
	return t.QualifiedIdentifier()
}

func (t *EntitlementType) GetContainerType() Type {
	return t.containerType
}

func (t *EntitlementType) SetContainerType(containerType Type) {
	t.containerType = containerType
}

func (t *EntitlementType) GetLocation() common.Location {
	return t.Location
}

func (t *EntitlementType) QualifiedIdentifier() string {
	return qualifiedIdentifier(t.Identifier, t.containerType)
}

func (t *EntitlementType) ID() TypeID {
	return common.NewTypeIDFromQualifiedName(nil, t.Location, t.QualifiedIdentifier())
}

func (t *EntitlementType) Equal(other Type) bool {
	otherEntitlement, ok := other.(*EntitlementType)
	if !ok {
		return false
	}

	return otherEntitlement.ID() == t.ID()
}

func (t *EntitlementType) Map(_ common.MemoryGauge, _ map[*TypeParameter]*TypeParameter, f func(Type) Type) Type {
	return f(t)
}

func (t *EntitlementType) GetMembers() map[string]MemberResolver {
	return withBuiltinMembers(t, nil)
}

func (*EntitlementType) IsPrimitiveType() bool {
	return false
}

func (t *EntitlementType) IsInvalidType() bool {
	return t.isInvalid
}

func (*EntitlementType) IsOrContainsReferenceType() bool {
	return false
}

func (*EntitlementType) IsStorable(_ map[*Member]bool) bool {
	return false
}

func (*EntitlementType) IsExportable(_ map[*Member]bool) bool {
	return false
}

func (*EntitlementType) IsImportable(_ map[*Member]bool) bool {
	return false
}

func (*EntitlementType) IsEquatable() bool {
	return false
}

func (*EntitlementType) IsComparable() bool {
	return false
}

func (*EntitlementType) IsResourceType() bool {
	return false
}

func (*EntitlementType) ContainFieldsOrElements() bool {
	return false
}

func (*EntitlementType) TypeAnnotationState() TypeAnnotationState {
	return TypeAnnotationStateDirectEntitlementTypeAnnotation
}

func (t *EntitlementType) RewriteWithIntersectionTypes() (Type, bool) {
	return t, false
}

func (*EntitlementType) Unify(
	_ Type,
	_ *TypeParameterTypeOrderedMap,
	_ func(err error),
	_ common.MemoryGauge,
	_ ast.HasPosition,
) bool {
	return false
}

func (t *EntitlementType) Resolve(_ *TypeParameterTypeOrderedMap) Type {
	return t
}

func (t *EntitlementType) CheckInstantiated(_ ast.HasPosition, _ common.MemoryGauge, _ func(err error)) {
	// No-OP
}

// EntitlementMapType

type EntitlementRelation struct {
	Input  *EntitlementType
	Output *EntitlementType
}

func NewEntitlementRelation(
	memoryGauge common.MemoryGauge,
	input *EntitlementType,
	output *EntitlementType,
) EntitlementRelation {
	common.UseMemory(memoryGauge, common.EntitlementRelationSemaTypeMemoryUsage)
	return EntitlementRelation{
		Input:  input,
		Output: output,
	}
}

type EntitlementMapType struct {
	Location      common.Location
	containerType Type
	Identifier    string
	Relations     []EntitlementRelation

	// Whether this map type includes the special identity relation,
	// which maps every input to itself. The `Identity` mapping itself
	// is defined as the empty map type that includes the identity relation
	IncludesIdentity  bool
	resolveInclusions sync.Once
}

var _ Type = &EntitlementMapType{}
var _ ContainedType = &EntitlementMapType{}
var _ LocatedType = &EntitlementMapType{}

func NewEntitlementMapType(
	memoryGauge common.MemoryGauge,
	location common.Location,
	identifier string,
) *EntitlementMapType {
	common.UseMemory(memoryGauge, common.EntitlementMapSemaTypeMemoryUsage)
	return &EntitlementMapType{
		Location:   location,
		Identifier: identifier,
	}
}

func (*EntitlementMapType) IsType() {}

func (t *EntitlementMapType) Tag() TypeTag {
	return InvalidTypeTag // entitlement map types may never appear as types, and thus cannot have a computed supertype
}

func (t *EntitlementMapType) String() string {
	return t.Identifier
}

func (t *EntitlementMapType) QualifiedString() string {
	return t.QualifiedIdentifier()
}

func (t *EntitlementMapType) GetContainerType() Type {
	return t.containerType
}

func (t *EntitlementMapType) SetContainerType(containerType Type) {
	t.containerType = containerType
}

func (t *EntitlementMapType) GetLocation() common.Location {
	return t.Location
}

func (t *EntitlementMapType) QualifiedIdentifier() string {
	return qualifiedIdentifier(t.Identifier, t.containerType)
}

func (t *EntitlementMapType) ID() TypeID {
	return common.NewTypeIDFromQualifiedName(nil, t.Location, t.QualifiedIdentifier())
}

func (t *EntitlementMapType) Equal(other Type) bool {
	otherEntitlement, ok := other.(*EntitlementMapType)
	if !ok {
		return false
	}

	return otherEntitlement.ID() == t.ID()
}

func (t *EntitlementMapType) Map(_ common.MemoryGauge, _ map[*TypeParameter]*TypeParameter, f func(Type) Type) Type {
	return f(t)
}

func (t *EntitlementMapType) GetMembers() map[string]MemberResolver {
	return withBuiltinMembers(t, nil)
}

func (*EntitlementMapType) IsPrimitiveType() bool {
	return false
}

func (*EntitlementMapType) IsInvalidType() bool {
	return false
}

func (*EntitlementMapType) IsOrContainsReferenceType() bool {
	return false
}

func (*EntitlementMapType) IsStorable(_ map[*Member]bool) bool {
	return false
}

func (*EntitlementMapType) IsExportable(_ map[*Member]bool) bool {
	return false
}

func (*EntitlementMapType) IsImportable(_ map[*Member]bool) bool {
	return false
}

func (*EntitlementMapType) IsEquatable() bool {
	return false
}

func (*EntitlementMapType) IsComparable() bool {
	return false
}

func (*EntitlementMapType) IsResourceType() bool {
	return false
}

func (*EntitlementMapType) ContainFieldsOrElements() bool {
	return false
}

func (*EntitlementMapType) TypeAnnotationState() TypeAnnotationState {
	return TypeAnnotationStateDirectEntitlementTypeAnnotation
}

func (t *EntitlementMapType) RewriteWithIntersectionTypes() (Type, bool) {
	return t, false
}

func (*EntitlementMapType) Unify(
	_ Type,
	_ *TypeParameterTypeOrderedMap,
	_ func(err error),
	_ common.MemoryGauge,
	_ ast.HasPosition,
) bool {
	return false
}

func (t *EntitlementMapType) Resolve(_ *TypeParameterTypeOrderedMap) Type {
	return t
}

// Recursively resolve the include statements of an entitlement mapping declaration, walking the "hierarchy" defined in this file
// Uses the sync primitive stored in `resolveInclusions` to ensure each map type's includes are computed only once.
// This assumes that any includes coming from imported files are necessarily already completely resolved, since that imported file
// must necessarily already have been fully checked. Additionally, because import cycles are not allowed in Cadence, we only
// need to check for map-include cycles within the currently-checked file
func (t *EntitlementMapType) resolveEntitlementMappingInclusions(
	checker *Checker,
	declaration *ast.EntitlementMappingDeclaration,
	visitedMaps map[*EntitlementMapType]struct{},
) {
	t.resolveInclusions.Do(func() {
		visitedMaps[t] = struct{}{}
		defer delete(visitedMaps, t)

		// track locally included maps to report duplicates, which are unrelated to cycles
		// we do not enforce that no maps are duplicated across the entire chain; only the specific map definition
		// currently being considered. This is to avoid reporting annoying errors when trying to include two
		// maps defined elsewhere that may have small overlap.
		includedMaps := map[*EntitlementMapType]struct{}{}

		for _, inclusion := range declaration.Inclusions() {

			includedType := checker.convertNominalType(inclusion)
			includedMapType, isEntitlementMapping := includedType.(*EntitlementMapType)
			if !isEntitlementMapping {
				checker.report(&InvalidEntitlementMappingInclusionError{
					Map:          t,
					IncludedType: includedType,
					Range:        ast.NewRangeFromPositioned(checker.memoryGauge, inclusion),
				})
				continue
			}
			if _, duplicate := includedMaps[includedMapType]; duplicate {
				checker.report(&DuplicateEntitlementMappingInclusionError{
					Map:          t,
					IncludedType: includedMapType,
					Range:        ast.NewRangeFromPositioned(checker.memoryGauge, inclusion),
				})
				continue
			}
			if _, isCyclical := visitedMaps[includedMapType]; isCyclical {
				checker.report(&CyclicEntitlementMappingError{
					Map:          t,
					IncludedType: includedMapType,
					Range:        ast.NewRangeFromPositioned(checker.memoryGauge, inclusion),
				})
				continue
			}

			// recursively resolve the included map type's includes, skipping any that have already been resolved
			includedMapType.resolveEntitlementMappingInclusions(
				checker,
				checker.Elaboration.EntitlementMapTypeDeclaration(includedMapType),
				visitedMaps,
			)

			for _, relation := range includedMapType.Relations {
				if !slices.Contains(t.Relations, relation) {
					common.UseMemory(checker.memoryGauge, common.EntitlementRelationSemaTypeMemoryUsage)
					t.Relations = append(t.Relations, relation)
				}
			}
			t.IncludesIdentity = t.IncludesIdentity || includedMapType.IncludesIdentity

			includedMaps[includedMapType] = struct{}{}
		}
	})
}

func (t *EntitlementMapType) CheckInstantiated(_ ast.HasPosition, _ common.MemoryGauge, _ func(err error)) {
	// NO-OP
}

func extractNativeTypes(
	types []Type,
) {
	for len(types) > 0 {
		lastIndex := len(types) - 1
		curType := types[lastIndex]
		types[lastIndex] = nil
		types = types[:lastIndex]

		switch actualType := curType.(type) {
		case *CompositeType:
			NativeCompositeTypes[actualType.QualifiedIdentifier()] = actualType

			nestedTypes := actualType.NestedTypes
			if nestedTypes == nil {
				continue
			}

			nestedTypes.Foreach(func(_ string, nestedType Type) {
				nestedCompositeType, ok := nestedType.(*CompositeType)
				if !ok {
					return
				}

				types = append(types, nestedCompositeType)
			})
		case *InterfaceType:
			NativeInterfaceTypes[actualType.QualifiedIdentifier()] = actualType

			nestedTypes := actualType.NestedTypes
			if nestedTypes == nil {
				continue
			}

			nestedTypes.Foreach(func(_ string, nestedType Type) {
				nestedInterfaceType, ok := nestedType.(*InterfaceType)
				if !ok {
					return
				}

				types = append(types, nestedInterfaceType)
			})
		}

	}
}

var NativeCompositeTypes = map[string]*CompositeType{}

func init() {
	compositeTypes := []Type{
		AccountKeyType,
		PublicKeyType,
		HashAlgorithmType,
		SignatureAlgorithmType,
		AccountType,
		DeploymentResultType,
	}

	extractNativeTypes(compositeTypes)
}

var NativeInterfaceTypes = map[string]*InterfaceType{}

func init() {
	interfaceTypes := []Type{
		StructStringerType,
	}

	extractNativeTypes(interfaceTypes)
}
