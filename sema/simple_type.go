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
	"sync/atomic"

	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/common"
)

type ValueIndexingInfo struct {
	ElementType                   func(_ bool) Type
	IndexingType                  *NumericType
	IsValueIndexableType          bool
	AllowsValueIndexingAssignment bool
}

// SimpleType represents a simple nominal type.
type SimpleType struct {
	ValueIndexingInfo ValueIndexingInfo
	NestedTypes       *StringTypeOrderedMap
	memberResolvers   atomic.Pointer[map[string]MemberResolver]
	Members           func(*SimpleType) map[string]MemberResolver
	QualifiedName     string
	TypeID            TypeID
	Name              string
	TypeTag           TypeTag
	Importable        bool
	Exportable        bool
	Equatable         bool
	Comparable        bool
	Storable          bool
	Primitive         bool
	IsResource        bool
	ContainFields     bool

	// allow simple types to define a set of interfaces it conforms to
	// e.g. StructStringer
	conformances                     []*InterfaceType
	effectiveInterfaceConformanceSet atomic.Pointer[InterfaceSet]
	effectiveInterfaceConformances   atomic.Pointer[[]Conformance]
}

var _ Type = &SimpleType{}
var _ ValueIndexableType = &SimpleType{}
var _ ContainerType = &SimpleType{}
var _ ConformingType = &SimpleType{}

func (*SimpleType) IsType() {}

func (t *SimpleType) Tag() TypeTag {
	return t.TypeTag
}

func (*SimpleType) Precedence() ast.TypePrecedence {
	return ast.TypePrecedencePrimary
}

func (t *SimpleType) String() string {
	return t.Name
}

func (t *SimpleType) QualifiedString() string {
	return t.Name
}

func (t *SimpleType) ID() TypeID {
	return t.TypeID
}

func (t *SimpleType) Equal(other Type) bool {
	return other == t
}

func (t *SimpleType) IsResourceType() bool {
	return t.IsResource
}

func (t *SimpleType) IsPrimitiveType() bool {
	return t.Primitive
}

func (t *SimpleType) IsInvalidType() bool {
	return t == InvalidType
}

func (*SimpleType) IsOrContainsReferenceType() bool {
	return false
}

func (t *SimpleType) IsStorable(_ map[*Member]bool) bool {
	return t.Storable
}

func (t *SimpleType) IsEquatable() bool {
	return t.Equatable
}

func (t *SimpleType) IsComparable() bool {
	return t.Comparable
}

func (t *SimpleType) IsExportable(_ map[*Member]bool) bool {
	return t.Exportable
}

func (t *SimpleType) IsImportable(_ map[*Member]bool) bool {
	return t.Importable
}

func (t *SimpleType) ContainFieldsOrElements() bool {
	return t.ContainFields
}

func (*SimpleType) TypeAnnotationState() TypeAnnotationState {
	return TypeAnnotationStateValid
}

func (t *SimpleType) Rewrite(rewrite TypeRewriter) (Type, bool) {
	return applyTypeRewriter(rewrite, t, false)
}

func (*SimpleType) Unify(
	_ Type,
	_ *TypeParameterTypeOrderedMap,
	_ func(err error),
	_ common.MemoryGauge,
	_ ast.HasPosition,
) bool {
	return false
}

func (t *SimpleType) Resolve(_ *TypeParameterTypeOrderedMap) Type {
	return t
}

func (t *SimpleType) GetMembers() map[string]MemberResolver {
	// Return cached members if already computed
	if cachedMembers := t.memberResolvers.Load(); cachedMembers != nil {
		return *cachedMembers
	}

	// Compute members and cache them
	var computedMembers map[string]MemberResolver
	if t.Members != nil {
		computedMembers = t.Members(t)
	}
	computedMembers = withBuiltinMembers(t, computedMembers)
	t.memberResolvers.Store(&computedMembers)
	return computedMembers
}

func (t *SimpleType) IsContainerType() bool {
	return t.NestedTypes != nil
}

func (t *SimpleType) GetNestedTypes() *StringTypeOrderedMap {
	return t.NestedTypes
}

func (t *SimpleType) isValueIndexableType() bool {
	return t.ValueIndexingInfo.IsValueIndexableType
}

func (t *SimpleType) AllowsValueIndexingAssignment() bool {
	return t.ValueIndexingInfo.AllowsValueIndexingAssignment
}

func (t *SimpleType) ElementType(isAssignment bool) Type {
	return t.ValueIndexingInfo.ElementType(isAssignment)
}

func (t *SimpleType) IndexingType() Type {
	return t.ValueIndexingInfo.IndexingType
}

func (t *SimpleType) CompositeKind() common.CompositeKind {
	if t.IsResource {
		return common.CompositeKindResource
	} else {
		return common.CompositeKindStructure
	}
}

func (t *SimpleType) CheckInstantiated(
	_ ast.HasPosition,
	_ common.MemoryGauge,
	_ func(err error),
	_ SeenTypes,
) {
	// NO-OP
}

func (t *SimpleType) EffectiveInterfaceConformanceSet() *InterfaceSet {
	// Return cached set if already computed
	if cachedSet := t.effectiveInterfaceConformanceSet.Load(); cachedSet != nil {
		return cachedSet
	}

	// Compute set and cache it
	computedSet := NewInterfaceSet()
	for _, conformance := range t.conformances {
		computedSet.Add(conformance)
	}
	t.effectiveInterfaceConformanceSet.Store(computedSet)
	return computedSet
}

func (t *SimpleType) EffectiveInterfaceConformances() []Conformance {
	// Return cached conformances if already computed
	if cachedConformances := t.effectiveInterfaceConformances.Load(); cachedConformances != nil {
		return *cachedConformances
	}

	// Compute conformances and cache them
	computedConformances := distinctConformances(
		t.conformances,
		nil,
		map[*InterfaceType]struct{}{},
	)
	t.effectiveInterfaceConformances.Store(&computedConformances)
	return computedConformances
}
