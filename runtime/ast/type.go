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

package ast

import (
	"encoding/json"
	"fmt"

	"github.com/turbolent/prettier"

	"github.com/onflow/cadence/runtime/common"
)

const typeSeparatorSpaceDoc = prettier.Text(": ")

// TypeAnnotation

type TypeAnnotation struct {
	IsResource bool
	Type       Type     `json:"AnnotatedType"`
	StartPos   Position `json:"-"`
}

func NewTypeAnnotation(
	memoryGauge common.MemoryGauge,
	isResource bool,
	typ Type,
	startPos Position,
) *TypeAnnotation {
	common.UseMemory(memoryGauge, common.TypeAnnotationMemoryUsage)

	return &TypeAnnotation{
		IsResource: isResource,
		Type:       typ,
		StartPos:   startPos,
	}
}

func (t *TypeAnnotation) String() string {
	return Prettier(t)
}

func (t *TypeAnnotation) StartPosition() Position {
	return t.StartPos
}

func (t *TypeAnnotation) EndPosition(memoryGauge common.MemoryGauge) Position {
	return t.Type.EndPosition(memoryGauge)
}

const typeAnnotationResourceSymbolDoc = prettier.Text("@")

func (t *TypeAnnotation) Doc() prettier.Doc {
	if !t.IsResource {
		return t.Type.Doc()
	}

	return prettier.Concat{
		typeAnnotationResourceSymbolDoc,
		t.Type.Doc(),
	}
}

func (t *TypeAnnotation) MarshalJSON() ([]byte, error) {
	type Alias TypeAnnotation
	return json.Marshal(&struct {
		Range
		*Alias
	}{
		Range: NewUnmeteredRangeFromPositioned(t),
		Alias: (*Alias)(t),
	})
}

// Type

type Type interface {
	HasPosition
	fmt.Stringer
	isType()
	Doc() prettier.Doc
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

func NewNominalType(
	memoryGauge common.MemoryGauge,
	identifier Identifier,
	nestedIdentifiers []Identifier,
) *NominalType {
	common.UseMemory(memoryGauge, common.NominalTypeMemoryUsage)
	return &NominalType{
		Identifier:        identifier,
		NestedIdentifiers: nestedIdentifiers,
	}
}

func (*NominalType) isType() {}

func (t *NominalType) String() string {
	return Prettier(t)
}

func (t *NominalType) StartPosition() Position {
	return t.Identifier.StartPosition()
}

func (t *NominalType) EndPosition(memoryGauge common.MemoryGauge) Position {
	nestedCount := len(t.NestedIdentifiers)
	if nestedCount == 0 {
		return t.Identifier.EndPosition(memoryGauge)
	}
	lastIdentifier := t.NestedIdentifiers[nestedCount-1]
	return lastIdentifier.EndPosition(memoryGauge)
}

var nominalTypeSeparatorDoc = prettier.Text(".")

func (t *NominalType) Doc() prettier.Doc {
	var doc prettier.Doc = prettier.Text(t.Identifier.String())
	if len(t.NestedIdentifiers) > 0 {
		concat := prettier.Concat{doc}
		for _, identifier := range t.NestedIdentifiers {
			concat = append(
				concat,
				nominalTypeSeparatorDoc,
				prettier.Text(identifier.String()),
			)
		}
		doc = concat
	}
	return doc
}

func (t *NominalType) MarshalJSON() ([]byte, error) {
	type Alias NominalType
	return json.Marshal(&struct {
		Type string
		Range
		*Alias
	}{
		Type:  "NominalType",
		Range: NewUnmeteredRangeFromPositioned(t),
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

var _ Type = &OptionalType{}

func NewOptionalType(
	memoryGauge common.MemoryGauge,
	typ Type,
	endPos Position,
) *OptionalType {
	common.UseMemory(memoryGauge, common.OptionalTypeMemoryUsage)
	return &OptionalType{
		Type:   typ,
		EndPos: endPos,
	}
}

func (*OptionalType) isType() {}

func (t *OptionalType) String() string {
	return Prettier(t)
}

func (t *OptionalType) StartPosition() Position {
	return t.Type.StartPosition()
}

func (t *OptionalType) EndPosition(memoryGauge common.MemoryGauge) Position {
	return t.EndPos
}

const optionalTypeSymbolDoc = prettier.Text("?")

func (t *OptionalType) Doc() prettier.Doc {
	return prettier.Concat{
		t.Type.Doc(),
		optionalTypeSymbolDoc,
	}
}

func (t *OptionalType) MarshalJSON() ([]byte, error) {
	type Alias OptionalType
	return json.Marshal(&struct {
		Type string
		Range
		*Alias
	}{
		Type:  "OptionalType",
		Range: NewUnmeteredRangeFromPositioned(t),
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

var _ Type = &VariableSizedType{}

func NewVariableSizedType(
	memoryGauge common.MemoryGauge,
	typ Type,
	astRange Range,
) *VariableSizedType {
	common.UseMemory(memoryGauge, common.VariableSizedTypeMemoryUsage)
	return &VariableSizedType{
		Type:  typ,
		Range: astRange,
	}
}

func (*VariableSizedType) isType() {}

func (t *VariableSizedType) String() string {
	return Prettier(t)
}

const arrayTypeStartDoc = prettier.Text("[")
const arrayTypeEndDoc = prettier.Text("]")

func (t *VariableSizedType) Doc() prettier.Doc {
	return prettier.Concat{
		arrayTypeStartDoc,
		prettier.Indent{
			Doc: prettier.Concat{
				prettier.SoftLine{},
				t.Type.Doc(),
			},
		},
		prettier.SoftLine{},
		arrayTypeEndDoc,
	}
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

// ConstantSizedType is a constant-sized array type

type ConstantSizedType struct {
	Type Type `json:"ElementType"`
	Size *IntegerExpression
	Range
}

var _ Type = &ConstantSizedType{}

func NewConstantSizedType(
	memoryGauge common.MemoryGauge,
	typ Type,
	size *IntegerExpression,
	astRange Range,
) *ConstantSizedType {
	common.UseMemory(memoryGauge, common.ConstantSizedTypeMemoryUsage)
	return &ConstantSizedType{
		Type:  typ,
		Size:  size,
		Range: astRange,
	}
}

func (*ConstantSizedType) isType() {}

func (t *ConstantSizedType) String() string {
	return Prettier(t)
}

const constantSizedTypeSeparatorSpaceDoc = prettier.Text("; ")

func (t *ConstantSizedType) Doc() prettier.Doc {
	return prettier.Concat{
		arrayTypeStartDoc,
		prettier.Indent{
			Doc: prettier.Concat{
				prettier.SoftLine{},
				t.Type.Doc(),
				constantSizedTypeSeparatorSpaceDoc,
				t.Size.Doc(),
			},
		},
		prettier.SoftLine{},
		arrayTypeEndDoc,
	}
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

var _ Type = &DictionaryType{}

func NewDictionaryType(
	memoryGauge common.MemoryGauge,
	keyType Type,
	valueType Type,
	astRange Range,
) *DictionaryType {
	common.UseMemory(memoryGauge, common.DictionaryTypeMemoryUsage)
	return &DictionaryType{
		KeyType:   keyType,
		ValueType: valueType,
		Range:     astRange,
	}
}

func (*DictionaryType) isType() {}

func (t *DictionaryType) String() string {
	return Prettier(t)
}

const dictionaryTypeStartDoc = prettier.Text("{")
const dictionaryTypeEndDoc = prettier.Text("}")

func (t *DictionaryType) Doc() prettier.Doc {
	return prettier.Concat{
		dictionaryTypeStartDoc,
		prettier.Indent{
			Doc: prettier.Concat{
				prettier.SoftLine{},
				t.KeyType.Doc(),
				typeSeparatorSpaceDoc,
				t.ValueType.Doc(),
			},
		},
		prettier.SoftLine{},
		dictionaryTypeEndDoc,
	}
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

var _ Type = &FunctionType{}

func NewFunctionType(
	memoryGauge common.MemoryGauge,
	parameterTypes []*TypeAnnotation,
	returnType *TypeAnnotation,
	astRange Range,
) *FunctionType {
	common.UseMemory(memoryGauge, common.FunctionTypeMemoryUsage)
	return &FunctionType{
		ParameterTypeAnnotations: parameterTypes,
		ReturnTypeAnnotation:     returnType,
		Range:                    astRange,
	}
}

func (*FunctionType) isType() {}

func (t *FunctionType) String() string {
	return Prettier(t)
}

const functionTypeStartDoc = prettier.Text("(")
const functionTypeEndDoc = prettier.Text(")")
const functionTypeParameterSeparatorDoc = prettier.Text(",")

func (t *FunctionType) Doc() prettier.Doc {
	parametersDoc := prettier.Concat{
		prettier.SoftLine{},
	}

	for i, parameterTypeAnnotation := range t.ParameterTypeAnnotations {
		if i > 0 {
			parametersDoc = append(
				parametersDoc,
				functionTypeParameterSeparatorDoc,
				prettier.Line{},
			)
		}
		parametersDoc = append(
			parametersDoc,
			parameterTypeAnnotation.Doc(),
		)
	}

	return prettier.Concat{
		functionTypeStartDoc,
		prettier.Group{
			Doc: prettier.Concat{
				functionTypeStartDoc,
				prettier.Indent{
					Doc: parametersDoc,
				},
				prettier.SoftLine{},
				functionTypeEndDoc,
			},
		},
		typeSeparatorSpaceDoc,
		t.ReturnTypeAnnotation.Doc(),
		functionTypeEndDoc,
	}
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

var _ Type = &ReferenceType{}

func NewReferenceType(
	memoryGauge common.MemoryGauge,
	authorized bool,
	typ Type,
	startPos Position,
) *ReferenceType {
	common.UseMemory(memoryGauge, common.ReferenceTypeMemoryUsage)
	return &ReferenceType{
		Authorized: authorized,
		Type:       typ,
		StartPos:   startPos,
	}
}

func (*ReferenceType) isType() {}

func (t *ReferenceType) String() string {
	return Prettier(t)
}

func (t *ReferenceType) StartPosition() Position {
	return t.StartPos
}

func (t *ReferenceType) EndPosition(memoryGauge common.MemoryGauge) Position {
	return t.Type.EndPosition(memoryGauge)
}

const referenceTypeAuthKeywordSpaceDoc = prettier.Text("auth ")
const referenceTypeSymbolDoc = prettier.Text("&")

func (t *ReferenceType) Doc() prettier.Doc {
	var doc prettier.Concat
	if t.Authorized {
		doc = append(doc, referenceTypeAuthKeywordSpaceDoc)
	}

	return append(
		doc,
		referenceTypeSymbolDoc,
		t.Type.Doc(),
	)
}

func (t *ReferenceType) MarshalJSON() ([]byte, error) {
	type Alias ReferenceType
	return json.Marshal(&struct {
		Type string
		Range
		*Alias
	}{
		Type:  "ReferenceType",
		Range: NewUnmeteredRangeFromPositioned(t),
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

var _ Type = &RestrictedType{}

func NewRestrictedType(
	memoryGauge common.MemoryGauge,
	typ Type,
	restrictions []*NominalType,
	astRange Range,
) *RestrictedType {
	common.UseMemory(memoryGauge, common.RestrictedTypeMemoryUsage)
	return &RestrictedType{
		Type:         typ,
		Restrictions: restrictions,
		Range:        astRange,
	}
}

func (*RestrictedType) isType() {}

func (t *RestrictedType) String() string {
	return Prettier(t)
}

const restrictedTypeStartDoc = prettier.Text("{")
const restrictedTypeEndDoc = prettier.Text("}")
const restrictedTypeSeparatorDoc = prettier.Text(",")

func (t *RestrictedType) Doc() prettier.Doc {
	restrictionsDoc := prettier.Concat{
		prettier.SoftLine{},
	}

	for i, restriction := range t.Restrictions {
		if i > 0 {
			restrictionsDoc = append(
				restrictionsDoc,
				restrictedTypeSeparatorDoc,
				prettier.Line{},
			)
		}
		restrictionsDoc = append(
			restrictionsDoc,
			restriction.Doc(),
		)
	}

	var doc prettier.Concat
	if t.Type != nil {
		doc = append(doc, t.Type.Doc())
	}

	return append(doc,
		prettier.Group{
			Doc: prettier.Concat{
				restrictedTypeStartDoc,
				prettier.Indent{
					Doc: restrictionsDoc,
				},
				prettier.SoftLine{},
				restrictedTypeEndDoc,
			},
		},
	)

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

var _ Type = &InstantiationType{}

func NewInstantiationType(
	memoryGauge common.MemoryGauge,
	typ Type,
	typeArguments []*TypeAnnotation,
	typeArgumentsStartPos Position,
	endPos Position,
) *InstantiationType {
	common.UseMemory(memoryGauge, common.InstantiationTypeMemoryUsage)
	return &InstantiationType{
		Type:                  typ,
		TypeArguments:         typeArguments,
		TypeArgumentsStartPos: typeArgumentsStartPos,
		EndPos:                endPos,
	}
}

func (*InstantiationType) isType() {}

func (t *InstantiationType) String() string {
	return Prettier(t)
}

func (t *InstantiationType) StartPosition() Position {
	return t.Type.StartPosition()
}

func (t *InstantiationType) EndPosition(common.MemoryGauge) Position {
	return t.EndPos
}

const instantiationTypeStartDoc = prettier.Text("<")
const instantiationTypeEndDoc = prettier.Text(">")
const instantiationTypeSeparatorDoc = prettier.Text(",")

func (t *InstantiationType) Doc() prettier.Doc {
	typeArgumentsDoc := prettier.Concat{
		prettier.SoftLine{},
	}

	for i, typeArgument := range t.TypeArguments {
		if i > 0 {
			typeArgumentsDoc = append(
				typeArgumentsDoc,
				instantiationTypeSeparatorDoc,
				prettier.Line{},
			)
		}
		typeArgumentsDoc = append(
			typeArgumentsDoc,
			typeArgument.Doc(),
		)
	}

	return prettier.Concat{
		t.Type.Doc(),
		prettier.Group{
			Doc: prettier.Concat{
				instantiationTypeStartDoc,
				prettier.Indent{
					Doc: typeArgumentsDoc,
				},
				prettier.SoftLine{},
				instantiationTypeEndDoc,
			},
		},
	}
}

func (t *InstantiationType) MarshalJSON() ([]byte, error) {
	type Alias InstantiationType
	return json.Marshal(&struct {
		Type string
		Range
		*Alias
	}{
		Type:  "InstantiationType",
		Range: NewUnmeteredRangeFromPositioned(t),
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
