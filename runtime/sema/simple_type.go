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

package sema

import (
	"sync"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
)

type ValueIndexingInfo struct {
	ElementType                   func(_ bool) Type
	IndexingType                  *NumericType
	IsValueIndexableType          bool
	AllowsValueIndexingAssignment bool
}

// SimpleType represents a simple nominal type.
type SimpleType struct {
	ValueIndexingInfo   ValueIndexingInfo
	IsSuperTypeOf       func(subType Type) bool
	NestedTypes         *StringTypeOrderedMap
	memberResolvers     map[string]MemberResolver
	Members             func(*SimpleType) map[string]MemberResolver
	QualifiedName       string
	TypeID              TypeID
	Name                string
	tag                 TypeTag
	memberResolversOnce sync.Once
	Importable          bool
	Exportable          bool
	Equatable           bool
	Comparable          bool
	Storable            bool
	IsResource          bool
}

var _ Type = &SimpleType{}
var _ ValueIndexableType = &SimpleType{}
var _ ContainerType = &SimpleType{}

func (*SimpleType) IsType() {}

func (t *SimpleType) Tag() TypeTag {
	return t.tag
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

func (t *SimpleType) IsInvalidType() bool {
	return t == InvalidType
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

func (*SimpleType) TypeAnnotationState() TypeAnnotationState {
	return TypeAnnotationStateValid
}

func (t *SimpleType) RewriteWithRestrictedTypes() (Type, bool) {
	return t, false
}

func (*SimpleType) Unify(_ Type, _ *TypeParameterTypeOrderedMap, _ func(err error), _ ast.Range) bool {
	return false
}

func (t *SimpleType) Resolve(_ *TypeParameterTypeOrderedMap) Type {
	return t
}

func (t *SimpleType) GetMembers() map[string]MemberResolver {
	t.initializeMembers()
	return t.memberResolvers
}

func (t *SimpleType) initializeMembers() {
	t.memberResolversOnce.Do(func() {
		var members map[string]MemberResolver
		if t.Members != nil {
			members = t.Members(t)
		}
		t.memberResolvers = withBuiltinMembers(t, members)
	})
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
