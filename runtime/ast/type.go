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

package ast

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/turbolent/prettier"
)

// TypeAnnotation

type TypeAnnotation struct {
	IsResource bool
	Type       Type     `json:"AnnotatedType"`
	StartPos   Position `json:"-"`
}

func (t *TypeAnnotation) String() string {
	if t.IsResource {
		return fmt.Sprintf("@%s", t.Type)
	}
	return fmt.Sprint(t.Type)
}

func (t *TypeAnnotation) StartPosition() Position {
	return t.StartPos
}

func (t *TypeAnnotation) EndPosition() Position {
	return t.Type.EndPosition()
}

func (t *TypeAnnotation) MarshalJSON() ([]byte, error) {
	type Alias TypeAnnotation
	return json.Marshal(&struct {
		Range
		*Alias
	}{
		Range: NewRangeFromPositioned(t),
		Alias: (*Alias)(t),
	})
}

// Type

type Type interface {
	HasPosition
	fmt.Stringer
	isType()
	CheckEqual(other Type, checker TypeEqualityChecker) error
}

func IsEmptyType(t Type) bool {
	nominalType, ok := t.(*NominalType)
	return ok && nominalType.Identifier.Identifier == ""
}

// NominalType represents a named type

type NominalType struct {
	Identifier        Identifier
	NestedIdentifiers []Identifier `json:",omitempty"`
}

var _ Type = &NominalType{}

func (*NominalType) isType() {}

func (t *NominalType) String() string {
	var sb strings.Builder
	sb.WriteString(t.Identifier.String())
	for _, identifier := range t.NestedIdentifiers {
		sb.WriteRune('.')
		sb.WriteString(identifier.String())
	}
	return sb.String()
}

func (t *NominalType) StartPosition() Position {
	return t.Identifier.StartPosition()
}

func (t *NominalType) EndPosition() Position {
	nestedCount := len(t.NestedIdentifiers)
	if nestedCount == 0 {
		return t.Identifier.EndPosition()
	}
	lastIdentifier := t.NestedIdentifiers[nestedCount-1]
	return lastIdentifier.EndPosition()
}

func (t *NominalType) Doc() prettier.Doc {
	return prettier.Text(t.String())
}

func (t *NominalType) MarshalJSON() ([]byte, error) {
	type Alias NominalType
	return json.Marshal(&struct {
		Type string
		Range
		*Alias
	}{
		Type:  "NominalType",
		Range: NewRangeFromPositioned(t),
		Alias: (*Alias)(t),
	})
}

func (t *NominalType) IsQualifiedName() bool {
	return len(t.NestedIdentifiers) > 0
}

func (t *NominalType) CheckEqual(other Type, checker TypeEqualityChecker) error {
	return checker.CheckNominalTypeEquality(t, other)
}

// OptionalType represents am optional variant of another type

type OptionalType struct {
	Type   Type     `json:"ElementType"`
	EndPos Position `json:"-"`
}

func (*OptionalType) isType() {}

func (t *OptionalType) String() string {
	return fmt.Sprintf("%s?", t.Type)
}

func (t *OptionalType) StartPosition() Position {
	return t.Type.StartPosition()
}

func (t *OptionalType) EndPosition() Position {
	return t.EndPos
}

func (t *OptionalType) MarshalJSON() ([]byte, error) {
	type Alias OptionalType
	return json.Marshal(&struct {
		Type string
		Range
		*Alias
	}{
		Type:  "OptionalType",
		Range: NewRangeFromPositioned(t),
		Alias: (*Alias)(t),
	})
}

func (t *OptionalType) CheckEqual(other Type, checker TypeEqualityChecker) error {
	return checker.CheckOptionalTypeEquality(t, other)
}

// VariableSizedType is a variable sized array type

type VariableSizedType struct {
	Type Type `json:"ElementType"`
	Range
}

func (*VariableSizedType) isType() {}

func (t *VariableSizedType) String() string {
	return fmt.Sprintf("[%s]", t.Type)
}

func (t *VariableSizedType) MarshalJSON() ([]byte, error) {
	type Alias VariableSizedType
	return json.Marshal(&struct {
		Type string
		*Alias
	}{
		Type:  "VariableSizedType",
		Alias: (*Alias)(t),
	})
}

func (t *VariableSizedType) CheckEqual(other Type, checker TypeEqualityChecker) error {
	return checker.CheckVariableSizedTypeEquality(t, other)
}

// ConstantSizedType is a constant sized array type

type ConstantSizedType struct {
	Type Type `json:"ElementType"`
	Size *IntegerExpression
	Range
}

func (*ConstantSizedType) isType() {}

func (t *ConstantSizedType) String() string {
	return fmt.Sprintf("[%s; %s]", t.Type, t.Size)
}

func (t *ConstantSizedType) MarshalJSON() ([]byte, error) {
	type Alias ConstantSizedType
	return json.Marshal(&struct {
		Type string
		*Alias
	}{
		Type:  "ConstantSizedType",
		Alias: (*Alias)(t),
	})
}

func (t *ConstantSizedType) CheckEqual(other Type, checker TypeEqualityChecker) error {
	return checker.CheckConstantSizedTypeEquality(t, other)
}

// DictionaryType

type DictionaryType struct {
	KeyType   Type
	ValueType Type
	Range
}

func (*DictionaryType) isType() {}

func (t *DictionaryType) String() string {
	return fmt.Sprintf("{%s: %s}", t.KeyType, t.ValueType)
}

func (t *DictionaryType) MarshalJSON() ([]byte, error) {
	type Alias DictionaryType
	return json.Marshal(&struct {
		Type string
		*Alias
	}{
		Type:  "DictionaryType",
		Alias: (*Alias)(t),
	})
}

func (t *DictionaryType) CheckEqual(other Type, checker TypeEqualityChecker) error {
	return checker.CheckDictionaryTypeEquality(t, other)
}

// FunctionType

type FunctionType struct {
	ParameterTypeAnnotations []*TypeAnnotation `json:",omitempty"`
	ReturnTypeAnnotation     *TypeAnnotation
	Range
}

func (*FunctionType) isType() {}

func (t *FunctionType) String() string {
	var parameters strings.Builder
	for i, parameterTypeAnnotation := range t.ParameterTypeAnnotations {
		if i > 0 {
			parameters.WriteString(", ")
		}
		parameters.WriteString(parameterTypeAnnotation.String())
	}

	return fmt.Sprintf("((%s): %s)", parameters.String(), t.ReturnTypeAnnotation.String())
}

func (t *FunctionType) MarshalJSON() ([]byte, error) {
	type Alias FunctionType
	return json.Marshal(&struct {
		Type string
		*Alias
	}{
		Type:  "FunctionType",
		Alias: (*Alias)(t),
	})
}

func (t *FunctionType) CheckEqual(other Type, checker TypeEqualityChecker) error {
	return checker.CheckFunctionTypeEquality(t, other)
}

// ReferenceType

type ReferenceType struct {
	Authorized bool
	Type       Type     `json:"ReferencedType"`
	StartPos   Position `json:"-"`
}

func (*ReferenceType) isType() {}

func (t *ReferenceType) String() string {
	var builder strings.Builder
	if t.Authorized {
		builder.WriteString("auth ")
	}
	builder.WriteRune('&')
	builder.WriteString(t.Type.String())
	return builder.String()
}

func (t *ReferenceType) StartPosition() Position {
	return t.StartPos
}

func (t *ReferenceType) EndPosition() Position {
	return t.Type.EndPosition()
}

func (t *ReferenceType) MarshalJSON() ([]byte, error) {
	type Alias ReferenceType
	return json.Marshal(&struct {
		Type string
		Range
		*Alias
	}{
		Type:  "ReferenceType",
		Range: NewRangeFromPositioned(t),
		Alias: (*Alias)(t),
	})
}

func (t *ReferenceType) CheckEqual(other Type, checker TypeEqualityChecker) error {
	return checker.CheckReferenceTypeEquality(t, other)
}

// RestrictedType

type RestrictedType struct {
	Type         Type `json:"RestrictedType"`
	Restrictions []*NominalType
	Range
}

func (*RestrictedType) isType() {}

func (t *RestrictedType) String() string {
	var builder strings.Builder
	if t.Type != nil {
		builder.WriteString(t.Type.String())
	}
	builder.WriteRune('{')
	for i, restriction := range t.Restrictions {
		if i > 0 {
			builder.WriteString(", ")
		}
		builder.WriteString(restriction.String())
	}
	builder.WriteRune('}')
	return builder.String()
}

func (t *RestrictedType) MarshalJSON() ([]byte, error) {
	type Alias RestrictedType
	return json.Marshal(&struct {
		Type string
		*Alias
	}{
		Type:  "RestrictedType",
		Alias: (*Alias)(t),
	})
}

func (t *RestrictedType) CheckEqual(other Type, checker TypeEqualityChecker) error {
	return checker.CheckRestrictedTypeEquality(t, other)
}

// InstantiationType represents an instantiation of a generic (nominal) type

type InstantiationType struct {
	Type                  Type `json:"InstantiatedType"`
	TypeArguments         []*TypeAnnotation
	TypeArgumentsStartPos Position
	EndPos                Position `json:"-"`
}

func (*InstantiationType) isType() {}

func (t *InstantiationType) String() string {
	var sb strings.Builder
	sb.WriteString(t.Type.String())
	sb.WriteRune('<')
	for i, typeArgument := range t.TypeArguments {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(typeArgument.String())
	}
	sb.WriteRune('>')
	return sb.String()
}

func (t *InstantiationType) StartPosition() Position {
	return t.Type.StartPosition()
}

func (t *InstantiationType) EndPosition() Position {
	return t.EndPos
}

func (t *InstantiationType) MarshalJSON() ([]byte, error) {
	type Alias InstantiationType
	return json.Marshal(&struct {
		Type string
		Range
		*Alias
	}{
		Type:  "InstantiationType",
		Range: NewRangeFromPositioned(t),
		Alias: (*Alias)(t),
	})
}

func (t *InstantiationType) CheckEqual(other Type, checker TypeEqualityChecker) error {
	return checker.CheckInstantiationTypeEquality(t, other)
}

type TypeEqualityChecker interface {
	CheckNominalTypeEquality(*NominalType, Type) error
	CheckOptionalTypeEquality(*OptionalType, Type) error
	CheckVariableSizedTypeEquality(*VariableSizedType, Type) error
	CheckConstantSizedTypeEquality(*ConstantSizedType, Type) error
	CheckDictionaryTypeEquality(*DictionaryType, Type) error
	CheckFunctionTypeEquality(*FunctionType, Type) error
	CheckReferenceTypeEquality(*ReferenceType, Type) error
	CheckRestrictedTypeEquality(*RestrictedType, Type) error
	CheckInstantiationTypeEquality(*InstantiationType, Type) error
}
