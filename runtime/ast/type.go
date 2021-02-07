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
	Equal(other Type) bool
}

// NominalType represents a named type

type NominalType struct {
	Identifier        Identifier
	NestedIdentifiers []Identifier `json:",omitempty"`
}

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

func (t *NominalType) Equal(other Type) bool {
	otherNominalType, ok := other.(*NominalType)
	if !ok {
		return false
	}

	if t.Identifier.Identifier != otherNominalType.Identifier.Identifier {
		return false
	}

	if len(t.NestedIdentifiers) != len(otherNominalType.NestedIdentifiers) {
		return false
	}

	for index, identifier := range t.NestedIdentifiers {
		otherIdentifier := otherNominalType.NestedIdentifiers[index]
		if identifier.Identifier != otherIdentifier.Identifier {
			return false
		}
	}
	return true
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

func (t *OptionalType) Equal(other Type) bool {
	otherOptionalType, ok := other.(*OptionalType)
	if !ok {
		return false
	}

	return t.Type.Equal(otherOptionalType.Type)
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

func (t *VariableSizedType) Equal(other Type) bool {
	otherVarSizedType, ok := other.(*VariableSizedType)
	if !ok {
		return false
	}

	return t.Type.Equal(otherVarSizedType.Type)
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

func (t *ConstantSizedType) Equal(other Type) bool {
	otherConstSizedType, ok := other.(*ConstantSizedType)
	if !ok {
		return false
	}

	return t.Size == otherConstSizedType.Size &&
		t.Type.Equal(otherConstSizedType.Type)
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

func (t *DictionaryType) Equal(other Type) bool {
	dictionaryType, ok := other.(*DictionaryType)
	if !ok {
		return false
	}

	return t.KeyType.Equal(dictionaryType.KeyType) &&
		t.KeyType.Equal(dictionaryType.KeyType)
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

func (t *FunctionType) Equal(other Type) bool {
	otherFuncType, ok := other.(*FunctionType)
	if !ok {
		return false
	}

	if len(t.ParameterTypeAnnotations) != len(otherFuncType.ParameterTypeAnnotations) {
		return false
	}

	for index, paramAnnot := range t.ParameterTypeAnnotations {
		otherParamAnnot := otherFuncType.ParameterTypeAnnotations[index]
		if !paramAnnot.Type.Equal(otherParamAnnot.Type) {
			return false
		}
	}

	return t.ReturnTypeAnnotation.Type.Equal(otherFuncType.ReturnTypeAnnotation.Type)
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

func (t *ReferenceType) Equal(other Type) bool {
	otherRefType, ok := other.(*ReferenceType)
	if !ok {
		return false
	}

	return t.Authorized == otherRefType.Authorized &&
		t.Type.Equal(otherRefType.Type)
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

func (t *RestrictedType) Equal(other Type) bool {
	otherRestrictedType, ok := other.(*RestrictedType)
	if !ok {
		return false
	}

	if !t.Type.Equal(otherRestrictedType.Type) {
		return false
	}

	if len(t.Restrictions) != len(otherRestrictedType.Restrictions) {
		return false
	}

	for index, restriction := range t.Restrictions {
		otherRestriction := otherRestrictedType.Restrictions[index]
		if !restriction.Equal(otherRestriction) {
			return false
		}
	}

	return true
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

func (t *InstantiationType) Equal(other Type) bool {
	otherInstType, ok := other.(*InstantiationType)
	if !ok {
		return false
	}

	if !t.Type.Equal(otherInstType.Type) {
		return false
	}

	if len(t.TypeArguments) != len(otherInstType.TypeArguments) {
		return false
	}

	for index, typeArgs := range t.TypeArguments {
		otherTypeArgs := otherInstType.TypeArguments[index]
		if !typeArgs.Type.Equal(otherTypeArgs.Type) {
			return false
		}
	}

	return true
}
