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

package ast

import (
	"encoding/json"
	"fmt"

	"github.com/turbolent/prettier"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
)

const typeSeparatorSpaceDoc = prettier.Text(": ")

// TypeAnnotation

type TypeAnnotation struct {
	Type       Type     `json:"AnnotatedType"`
	StartPos   Position `json:"-"`
	IsResource bool
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
		*Alias
		Range
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
	NestedIdentifiers []Identifier `json:",omitempty"`
	Identifier        Identifier
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
		*Alias
		Type string
		Range
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

func (*NominalType) isEntitlementMapElement() {}

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
		*Alias
		Type string
		Range
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
		*Alias
		Type string
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
		*Alias
		Type string
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
		*Alias
		Type string
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
	PurityAnnotation         FunctionPurity
	ReturnTypeAnnotation     *TypeAnnotation
	ParameterTypeAnnotations []*TypeAnnotation `json:",omitempty"`
	Range
}

var _ Type = &FunctionType{}

func NewFunctionType(
	memoryGauge common.MemoryGauge,
	purity FunctionPurity,
	parameterTypes []*TypeAnnotation,
	returnType *TypeAnnotation,
	astRange Range,
) *FunctionType {
	common.UseMemory(memoryGauge, common.FunctionTypeMemoryUsage)
	return &FunctionType{
		PurityAnnotation:         purity,
		ParameterTypeAnnotations: parameterTypes,
		ReturnTypeAnnotation:     returnType,
		Range:                    astRange,
	}
}

func (*FunctionType) isType() {}

func (t *FunctionType) String() string {
	return Prettier(t)
}

const functionTypeKeywordDoc = prettier.Text("fun")
const openParenthesisDoc = prettier.Text("(")
const closeParenthesisDoc = prettier.Text(")")
const functionTypeParameterSeparatorDoc = prettier.Text(",")

func (t *FunctionType) Doc() prettier.Doc {
	parametersDoc := prettier.Concat{
		prettier.SoftLine{},
	}

	var result prettier.Concat

	if t.PurityAnnotation != FunctionPurityUnspecified {
		result = append(
			result,
			prettier.Text(t.PurityAnnotation.Keyword()),
			prettier.Space,
		)
	}

	result = append(result, functionTypeKeywordDoc, prettier.Space)

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

	result = append(
		result,
		prettier.Group{
			Doc: prettier.Concat{
				openParenthesisDoc,
				prettier.Indent{
					Doc: parametersDoc,
				},
				prettier.SoftLine{},
				closeParenthesisDoc,
			},
		},
		typeSeparatorSpaceDoc,
		t.ReturnTypeAnnotation.Doc(),
	)

	return result
}

func (t *FunctionType) MarshalJSON() ([]byte, error) {
	type Alias FunctionType
	return json.Marshal(&struct {
		*Alias
		Type string
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
	Type             Type     `json:"ReferencedType"`
	StartPos         Position `json:"-"`
	LegacyAuthorized bool
	Authorization    Authorization `json:"Authorization"`
}

var _ Type = &ReferenceType{}

func NewReferenceType(
	memoryGauge common.MemoryGauge,
	authorization Authorization,
	typ Type,
	startPos Position,
) *ReferenceType {
	common.UseMemory(memoryGauge, common.ReferenceTypeMemoryUsage)
	return &ReferenceType{
		Authorization: authorization,
		Type:          typ,
		StartPos:      startPos,
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

const referenceTypeAuthKeywordDoc = prettier.Text("auth")
const referenceTypeMappingKeywordDoc = prettier.Text("mapping ")
const referenceTypeSymbolDoc = prettier.Text("&")

func (t *ReferenceType) Doc() prettier.Doc {
	var doc prettier.Concat
	if t.Authorization != nil {
		doc = append(doc, referenceTypeAuthKeywordDoc)
		doc = append(doc, prettier.Text("("))
		switch authorization := t.Authorization.(type) {
		case EntitlementSet:
			if len(authorization.Entitlements()) > 0 {
				entitlements := authorization.Entitlements()
				// TODO: add indentation, improve separators. follow e.g. ParameterList.Doc()
				for i, entitlement := range entitlements {
					doc = append(doc, entitlement.Doc())
					if i < len(entitlements)-1 {
						doc = append(doc, prettier.Text(authorization.Separator().String()), prettier.Space)
					}
				}
			}
		case *MappedAccess:
			doc = append(doc,
				referenceTypeMappingKeywordDoc,
				authorization.EntitlementMap.Doc(),
			)
		default:
			panic(errors.NewUnreachableError())
		}
		doc = append(doc,
			prettier.Text(")"),
			prettier.Space,
		)
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
		*Alias
		Type string
		Range
	}{
		Type:  "ReferenceType",
		Range: NewUnmeteredRangeFromPositioned(t),
		Alias: (*Alias)(t),
	})
}

func (t *ReferenceType) CheckEqual(other Type, checker TypeEqualityChecker) error {
	return checker.CheckReferenceTypeEquality(t, other)
}

// IntersectionType

type IntersectionType struct {
	LegacyRestrictedType Type // Deprecated
	Types                []*NominalType
	Range
}

var _ Type = &IntersectionType{}

func NewIntersectionType(
	memoryGauge common.MemoryGauge,
	types []*NominalType,
	astRange Range,
) *IntersectionType {
	common.UseMemory(memoryGauge, common.IntersectionTypeMemoryUsage)
	return &IntersectionType{
		Types: types,
		Range: astRange,
	}
}

func (*IntersectionType) isType() {}

func (t *IntersectionType) String() string {
	return Prettier(t)
}

const intersectionTypeStartDoc = prettier.Text("{")
const intersectionTypeEndDoc = prettier.Text("}")
const intersectionTypeSeparatorDoc = prettier.Text(",")

func (t *IntersectionType) Doc() prettier.Doc {
	intersectionDoc := prettier.Concat{
		prettier.SoftLine{},
	}

	for i, typ := range t.Types {
		if i > 0 {
			intersectionDoc = append(
				intersectionDoc,
				intersectionTypeSeparatorDoc,
				prettier.Line{},
			)
		}
		intersectionDoc = append(
			intersectionDoc,
			typ.Doc(),
		)
	}

	var doc prettier.Concat

	return append(doc,
		prettier.Group{
			Doc: prettier.Concat{
				intersectionTypeStartDoc,
				prettier.Indent{
					Doc: intersectionDoc,
				},
				prettier.SoftLine{},
				intersectionTypeEndDoc,
			},
		},
	)

}

func (t *IntersectionType) MarshalJSON() ([]byte, error) {
	type Alias IntersectionType
	return json.Marshal(&struct {
		*Alias
		Type string
	}{
		Type:  "IntersectionType",
		Alias: (*Alias)(t),
	})
}

func (t *IntersectionType) CheckEqual(other Type, checker TypeEqualityChecker) error {
	return checker.CheckIntersectionTypeEquality(t, other)
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
		*Alias
		Type string
		Range
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
	CheckIntersectionTypeEquality(*IntersectionType, Type) error
	CheckInstantiationTypeEquality(*InstantiationType, Type) error
}
