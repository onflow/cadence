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

package sema

import "github.com/onflow/cadence/runtime/ast"

// NominalType represents a simple nominal type.
//
type NominalType struct {
	Name                 string
	QualifiedName        string
	TypeID               TypeID
	IsInvalid            bool
	IsResource           bool
	Storable             bool
	Equatable            bool
	ExternallyReturnable bool
	IsSuperTypeOf        func(subType Type) bool
}

func (*NominalType) IsType() {}

func (t *NominalType) String() string {
	return t.Name
}

func (t *NominalType) QualifiedString() string {
	return t.Name
}

func (t *NominalType) ID() TypeID {
	return t.TypeID
}

func (t *NominalType) Equal(other Type) bool {
	otherType, ok := other.(*NominalType)
	return ok && otherType == t
}

func (t *NominalType) IsResourceType() bool {
	return t.IsResource
}

func (t *NominalType) IsInvalidType() bool {
	return t.IsInvalid
}

func (t *NominalType) IsStorable(_ map[*Member]bool) bool {
	return t.Storable
}

func (t *NominalType) IsEquatable() bool {
	return t.Equatable
}

func (t *NominalType) IsExternallyReturnable(_ map[*Member]bool) bool {
	return t.ExternallyReturnable
}

func (*NominalType) TypeAnnotationState() TypeAnnotationState {
	return TypeAnnotationStateValid
}

func (t *NominalType) RewriteWithRestrictedTypes() (Type, bool) {
	return t, false
}

func (*NominalType) Unify(other Type, typeParameters *TypeParameterTypeOrderedMap, report func(err error), outerRange ast.Range) bool {
	return false
}

func (t *NominalType) Resolve(typeArguments *TypeParameterTypeOrderedMap) Type {
	return t
}

func (t *NominalType) GetMembers() map[string]MemberResolver {
	return withBuiltinMembers(t, nil)
}
