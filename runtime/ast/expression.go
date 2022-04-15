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
	"math/big"
	"strconv"
	"strings"
)

const NilConstant = "nil"

type Expression interface {
	Element
	fmt.Stringer
	IfStatementTest
	isExpression()
	AcceptExp(ExpressionVisitor) Repr
}

// BoolExpression

type BoolExpression struct {
	Value bool
	Range
}

var _ Element = &BoolExpression{}
var _ Expression = &BoolExpression{}

func (*BoolExpression) ElementType() ElementType {
	return ElementTypeBoolExpression
}

func (*BoolExpression) isExpression() {}

func (*BoolExpression) isIfStatementTest() {}

func (e *BoolExpression) Accept(visitor Visitor) Repr {
	return e.AcceptExp(visitor)
}

func (*BoolExpression) Walk(_ func(Element)) {
	// NO-OP
}

func (e *BoolExpression) AcceptExp(visitor ExpressionVisitor) Repr {
	return visitor.VisitBoolExpression(e)
}

func (e *BoolExpression) String() string {
	if e.Value {
		return "true"
	}
	return "false"
}

func (e *BoolExpression) MarshalJSON() ([]byte, error) {
	type Alias BoolExpression
	return json.Marshal(&struct {
		Type string
		*Alias
	}{
		Type:  "BoolExpression",
		Alias: (*Alias)(e),
	})
}

// NilExpression

type NilExpression struct {
	Pos Position `json:"-"`
}

var _ Element = &NilExpression{}
var _ Expression = &NilExpression{}

func (*NilExpression) ElementType() ElementType {
	return ElementTypeNilExpression
}

func (*NilExpression) isExpression() {}

func (*NilExpression) isIfStatementTest() {}

func (e *NilExpression) Accept(visitor Visitor) Repr {
	return e.AcceptExp(visitor)
}

func (*NilExpression) Walk(_ func(Element)) {
	// NO-OP
}

func (e *NilExpression) AcceptExp(visitor ExpressionVisitor) Repr {
	return visitor.VisitNilExpression(e)
}

func (e *NilExpression) String() string {
	return NilConstant
}

func (e *NilExpression) StartPosition() Position {
	return e.Pos
}

func (e *NilExpression) EndPosition() Position {
	return e.Pos.Shifted(len(NilConstant) - 1)
}

func (e *NilExpression) MarshalJSON() ([]byte, error) {
	type Alias NilExpression
	return json.Marshal(&struct {
		Type string
		Range
		*Alias
	}{
		Type:  "NilExpression",
		Range: NewRangeFromPositioned(e),
		Alias: (*Alias)(e),
	})
}

// StringExpression

type StringExpression struct {
	Value string
	Range
}

var _ Element = &StringExpression{}
var _ Expression = &StringExpression{}

func (*StringExpression) ElementType() ElementType {
	return ElementTypeStringExpression
}

func (*StringExpression) isExpression() {}

func (*StringExpression) isIfStatementTest() {}

func (e *StringExpression) Accept(visitor Visitor) Repr {
	return e.AcceptExp(visitor)
}

func (*StringExpression) Walk(_ func(Element)) {
	// NO-OP
}

func (e *StringExpression) AcceptExp(visitor ExpressionVisitor) Repr {
	return visitor.VisitStringExpression(e)
}

func (e *StringExpression) String() string {
	return strconv.Quote(e.Value)
}

func (e *StringExpression) MarshalJSON() ([]byte, error) {
	type Alias StringExpression
	return json.Marshal(&struct {
		Type string
		*Alias
	}{
		Type:  "StringExpression",
		Alias: (*Alias)(e),
	})
}

// IntegerExpression

type IntegerExpression struct {
	Value *big.Int `json:"-"`
	Base  int
	Range
}

var _ Element = &IntegerExpression{}
var _ Expression = &IntegerExpression{}

func (*IntegerExpression) ElementType() ElementType {
	return ElementTypeIntegerExpression
}

func (*IntegerExpression) isExpression() {}

func (*IntegerExpression) isIfStatementTest() {}

func (e *IntegerExpression) Accept(visitor Visitor) Repr {
	return e.AcceptExp(visitor)
}

func (*IntegerExpression) Walk(_ func(Element)) {
	// NO-OP
}

func (e *IntegerExpression) AcceptExp(visitor ExpressionVisitor) Repr {
	return visitor.VisitIntegerExpression(e)
}

func (e *IntegerExpression) String() string {
	return e.Value.String()
}

func (e *IntegerExpression) MarshalJSON() ([]byte, error) {
	type Alias IntegerExpression
	return json.Marshal(&struct {
		Type  string
		Value string
		*Alias
	}{
		Type:  "IntegerExpression",
		Value: e.Value.String(),
		Alias: (*Alias)(e),
	})
}

// FixedPointExpression

type FixedPointExpression struct {
	Negative        bool
	UnsignedInteger *big.Int `json:"-"`
	Fractional      *big.Int `json:"-"`
	Scale           uint
	Range
}

var _ Element = &FixedPointExpression{}
var _ Expression = &FixedPointExpression{}

func (*FixedPointExpression) ElementType() ElementType {
	return ElementTypeFixedPointExpression
}

func (*FixedPointExpression) isExpression() {}

func (*FixedPointExpression) isIfStatementTest() {}

func (e *FixedPointExpression) Accept(visitor Visitor) Repr {
	return e.AcceptExp(visitor)
}

func (*FixedPointExpression) Walk(_ func(Element)) {
	// NO-OP
}

func (e *FixedPointExpression) AcceptExp(visitor ExpressionVisitor) Repr {
	return visitor.VisitFixedPointExpression(e)
}

func (e *FixedPointExpression) String() string {
	var builder strings.Builder
	if e.Negative {
		builder.WriteRune('-')
	}
	builder.WriteString(e.UnsignedInteger.String())
	builder.WriteRune('.')
	fractional := e.Fractional.String()
	for i := uint(0); i < (e.Scale - uint(len(fractional))); i++ {
		builder.WriteRune('0')
	}
	builder.WriteString(fractional)
	return builder.String()
}

func (e *FixedPointExpression) MarshalJSON() ([]byte, error) {
	type Alias FixedPointExpression
	return json.Marshal(&struct {
		Type            string
		UnsignedInteger string
		Fractional      string
		*Alias
	}{
		Type:            "FixedPointExpression",
		UnsignedInteger: e.UnsignedInteger.String(),
		Fractional:      e.Fractional.String(),
		Alias:           (*Alias)(e),
	})
}

// ArrayExpression

type ArrayExpression struct {
	Values []Expression
	Range
}

var _ Element = &ArrayExpression{}
var _ Expression = &ArrayExpression{}

func (*ArrayExpression) ElementType() ElementType {
	return ElementTypeArrayExpression
}

func (*ArrayExpression) isExpression() {}

func (*ArrayExpression) isIfStatementTest() {}

func (e *ArrayExpression) Accept(visitor Visitor) Repr {
	return e.AcceptExp(visitor)
}

func (e *ArrayExpression) Walk(walkChild func(Element)) {
	walkExpressions(walkChild, e.Values)
}

func (e *ArrayExpression) AcceptExp(visitor ExpressionVisitor) Repr {
	return visitor.VisitArrayExpression(e)
}

func (e *ArrayExpression) String() string {
	var builder strings.Builder
	builder.WriteString("[")
	for i, value := range e.Values {
		if i > 0 {
			builder.WriteString(", ")
		}
		builder.WriteString(value.String())
	}
	builder.WriteString("]")
	return builder.String()
}

func (e *ArrayExpression) MarshalJSON() ([]byte, error) {
	type Alias ArrayExpression
	return json.Marshal(&struct {
		Type string
		*Alias
	}{
		Type:  "ArrayExpression",
		Alias: (*Alias)(e),
	})
}

// DictionaryExpression

type DictionaryExpression struct {
	Entries []DictionaryEntry
	Range
}

var _ Element = &DictionaryExpression{}
var _ Expression = &DictionaryExpression{}

func (*DictionaryExpression) ElementType() ElementType {
	return ElementTypeDictionaryExpression
}

func (*DictionaryExpression) isExpression() {}

func (*DictionaryExpression) isIfStatementTest() {}

func (e *DictionaryExpression) Accept(visitor Visitor) Repr {
	return e.AcceptExp(visitor)
}

func (e *DictionaryExpression) Walk(walkChild func(Element)) {
	for _, entry := range e.Entries {
		walkChild(entry.Key)
		walkChild(entry.Value)
	}
}

func (e *DictionaryExpression) AcceptExp(visitor ExpressionVisitor) Repr {
	return visitor.VisitDictionaryExpression(e)
}

func (e *DictionaryExpression) String() string {
	var builder strings.Builder
	builder.WriteString("{")
	for i, entry := range e.Entries {
		if i > 0 {
			builder.WriteString(", ")
		}
		builder.WriteString(entry.Key.String())
		builder.WriteString(": ")
		builder.WriteString(entry.Value.String())
	}
	builder.WriteString("}")
	return builder.String()
}

func (e *DictionaryExpression) MarshalJSON() ([]byte, error) {
	type Alias DictionaryExpression
	return json.Marshal(&struct {
		Type string
		*Alias
	}{
		Type:  "DictionaryExpression",
		Alias: (*Alias)(e),
	})
}

type DictionaryEntry struct {
	Key   Expression
	Value Expression
}

func (e DictionaryEntry) MarshalJSON() ([]byte, error) {
	type Alias DictionaryEntry
	return json.Marshal(&struct {
		Type string
		*Alias
	}{
		Type:  "DictionaryEntry",
		Alias: (*Alias)(&e),
	})
}

// IdentifierExpression

type IdentifierExpression struct {
	Identifier Identifier
}

var _ Element = &IdentifierExpression{}
var _ Expression = &IdentifierExpression{}

func (*IdentifierExpression) ElementType() ElementType {
	return ElementTypeIdentifierExpression
}

func (*IdentifierExpression) isExpression() {}

func (*IdentifierExpression) isIfStatementTest() {}

func (e *IdentifierExpression) Accept(visitor Visitor) Repr {
	return e.AcceptExp(visitor)
}

func (*IdentifierExpression) Walk(_ func(Element)) {
	// NO-OP
}

func (e *IdentifierExpression) AcceptExp(visitor ExpressionVisitor) Repr {
	return visitor.VisitIdentifierExpression(e)
}

func (e *IdentifierExpression) String() string {
	return e.Identifier.Identifier
}

func (e *IdentifierExpression) MarshalJSON() ([]byte, error) {
	type Alias IdentifierExpression
	return json.Marshal(&struct {
		Type string
		*Alias
		Range
	}{
		Type:  "IdentifierExpression",
		Range: NewRangeFromPositioned(e),
		Alias: (*Alias)(e),
	})
}

func (e *IdentifierExpression) StartPosition() Position {
	return e.Identifier.StartPosition()
}

func (e *IdentifierExpression) EndPosition() Position {
	return e.Identifier.EndPosition()
}

// Arguments

type Arguments []*Argument

func (args Arguments) String() string {
	var builder strings.Builder
	builder.WriteRune('(')
	for i, argument := range args {
		if i > 0 {
			builder.WriteString(", ")
		}
		builder.WriteString(argument.String())
	}
	builder.WriteRune(')')
	return builder.String()
}

// InvocationExpression

type InvocationExpression struct {
	InvokedExpression Expression
	TypeArguments     []*TypeAnnotation
	Arguments         Arguments
	ArgumentsStartPos Position
	EndPos            Position `json:"-"`
}

var _ Element = &InvocationExpression{}
var _ Expression = &InvocationExpression{}

func (*InvocationExpression) ElementType() ElementType {
	return ElementTypeInvocationExpression
}

func (*InvocationExpression) isExpression() {}

func (*InvocationExpression) isIfStatementTest() {}

func (e *InvocationExpression) Accept(visitor Visitor) Repr {
	return e.AcceptExp(visitor)
}

func (e *InvocationExpression) Walk(walkChild func(Element)) {
	walkChild(e.InvokedExpression)
	for _, argument := range e.Arguments {
		walkChild(argument.Expression)
	}
}

func (e *InvocationExpression) AcceptExp(visitor ExpressionVisitor) Repr {
	return visitor.VisitInvocationExpression(e)
}

func (e *InvocationExpression) String() string {
	var builder strings.Builder
	builder.WriteString(e.InvokedExpression.String())
	if len(e.TypeArguments) > 0 {
		builder.WriteRune('<')
		for i, ty := range e.TypeArguments {
			if i > 0 {
				builder.WriteString(", ")
			}
			builder.WriteString(ty.String())
		}
		builder.WriteRune('>')
	}
	builder.WriteString(e.Arguments.String())
	return builder.String()
}

func (e *InvocationExpression) StartPosition() Position {
	return e.InvokedExpression.StartPosition()
}

func (e *InvocationExpression) EndPosition() Position {
	return e.EndPos
}

func (e *InvocationExpression) MarshalJSON() ([]byte, error) {
	type Alias InvocationExpression
	return json.Marshal(&struct {
		Type string
		*Alias
		Range
	}{
		Type:  "InvocationExpression",
		Range: NewRangeFromPositioned(e),
		Alias: (*Alias)(e),
	})
}

// AccessExpression

type AccessExpression interface {
	Expression
	isAccessExpression()
	AccessedExpression() Expression
}

// MemberExpression

type MemberExpression struct {
	Expression Expression
	Optional   bool
	// The position of the token (`.`, `?.`) that separates the accessed expression
	// and the identifier of the member
	AccessPos  Position
	Identifier Identifier
}

var _ Element = &MemberExpression{}
var _ Expression = &MemberExpression{}

func (*MemberExpression) ElementType() ElementType {
	return ElementTypeMemberExpression
}

func (*MemberExpression) isExpression() {}

func (*MemberExpression) isIfStatementTest() {}

func (*MemberExpression) isAccessExpression() {}

func (e *MemberExpression) AccessedExpression() Expression {
	return e.Expression
}

func (e *MemberExpression) Accept(visitor Visitor) Repr {
	return e.AcceptExp(visitor)
}

func (e *MemberExpression) Walk(walkChild func(Element)) {
	walkChild(e.Expression)
}

func (e *MemberExpression) AcceptExp(visitor ExpressionVisitor) Repr {
	return visitor.VisitMemberExpression(e)
}

func (e *MemberExpression) String() string {
	optional := ""
	if e.Optional {
		optional = "?"
	}
	return fmt.Sprintf(
		"%s%s.%s",
		e.Expression, optional, e.Identifier,
	)
}

func (e *MemberExpression) StartPosition() Position {
	return e.Expression.StartPosition()
}

func (e *MemberExpression) EndPosition() Position {
	if e.Identifier.Identifier == "" {
		return e.AccessPos
	} else {
		return e.Identifier.EndPosition()
	}
}

func (e *MemberExpression) MarshalJSON() ([]byte, error) {
	type Alias MemberExpression
	return json.Marshal(&struct {
		Type string
		Range
		*Alias
	}{
		Type:  "MemberExpression",
		Range: NewRangeFromPositioned(e),
		Alias: (*Alias)(e),
	})
}

// IndexExpression

type IndexExpression struct {
	TargetExpression   Expression
	IndexingExpression Expression
	Range
}

var _ Element = &IndexExpression{}
var _ Expression = &IndexExpression{}

func (*IndexExpression) ElementType() ElementType {
	return ElementTypeIndexExpression
}

func (*IndexExpression) isExpression() {}

func (*IndexExpression) isIfStatementTest() {}

func (*IndexExpression) isAccessExpression() {}

func (e *IndexExpression) AccessedExpression() Expression {
	return e.TargetExpression
}

func (e *IndexExpression) Accept(visitor Visitor) Repr {
	return e.AcceptExp(visitor)
}

func (e *IndexExpression) Walk(walkChild func(Element)) {
	walkChild(e.TargetExpression)
	walkChild(e.IndexingExpression)
}

func (e *IndexExpression) AcceptExp(visitor ExpressionVisitor) Repr {
	return visitor.VisitIndexExpression(e)
}
func (e *IndexExpression) String() string {
	return fmt.Sprintf(
		"%s[%s]",
		e.TargetExpression, e.IndexingExpression,
	)
}

func (e *IndexExpression) MarshalJSON() ([]byte, error) {
	type Alias IndexExpression
	return json.Marshal(&struct {
		Type string
		*Alias
	}{
		Type:  "IndexExpression",
		Alias: (*Alias)(e),
	})
}

// ConditionalExpression

type ConditionalExpression struct {
	Test Expression
	Then Expression
	Else Expression
}

var _ Element = &ConditionalExpression{}
var _ Expression = &ConditionalExpression{}

func (*ConditionalExpression) ElementType() ElementType {
	return ElementTypeConditionalExpression
}

func (*ConditionalExpression) isExpression() {}

func (*ConditionalExpression) isIfStatementTest() {}

func (e *ConditionalExpression) Accept(visitor Visitor) Repr {
	return e.AcceptExp(visitor)
}

func (e *ConditionalExpression) Walk(walkChild func(Element)) {
	walkChild(e.Test)
	walkChild(e.Then)
	if e.Else != nil {
		walkChild(e.Else)
	}
}

func (e *ConditionalExpression) AcceptExp(visitor ExpressionVisitor) Repr {
	return visitor.VisitConditionalExpression(e)
}
func (e *ConditionalExpression) String() string {
	return fmt.Sprintf(
		"(%s ? %s : %s)",
		e.Test, e.Then, e.Else,
	)
}

func (e *ConditionalExpression) StartPosition() Position {
	return e.Test.StartPosition()
}

func (e *ConditionalExpression) EndPosition() Position {
	return e.Else.EndPosition()
}

func (e *ConditionalExpression) MarshalJSON() ([]byte, error) {
	type Alias ConditionalExpression
	return json.Marshal(&struct {
		Type string
		Range
		*Alias
	}{
		Type:  "ConditionalExpression",
		Range: NewRangeFromPositioned(e),
		Alias: (*Alias)(e),
	})
}

// UnaryExpression

type UnaryExpression struct {
	Operation  Operation
	Expression Expression
	StartPos   Position `json:"-"`
}

var _ Element = &UnaryExpression{}
var _ Expression = &UnaryExpression{}

func (*UnaryExpression) ElementType() ElementType {
	return ElementTypeUnaryExpression
}

func (*UnaryExpression) isExpression() {}

func (*UnaryExpression) isIfStatementTest() {}

func (e *UnaryExpression) Accept(visitor Visitor) Repr {
	return e.AcceptExp(visitor)
}

func (e *UnaryExpression) Walk(walkChild func(Element)) {
	walkChild(e.Expression)
}

func (e *UnaryExpression) AcceptExp(visitor ExpressionVisitor) Repr {
	return visitor.VisitUnaryExpression(e)
}

func (e *UnaryExpression) String() string {
	return fmt.Sprintf(
		"%s%s",
		e.Operation.Symbol(), e.Expression,
	)
}

func (e *UnaryExpression) StartPosition() Position {
	return e.StartPos
}

func (e *UnaryExpression) EndPosition() Position {
	return e.Expression.EndPosition()
}

func (e *UnaryExpression) MarshalJSON() ([]byte, error) {
	type Alias UnaryExpression
	return json.Marshal(&struct {
		Type string
		Range
		*Alias
	}{
		Type:  "UnaryExpression",
		Range: NewRangeFromPositioned(e),
		Alias: (*Alias)(e),
	})
}

// BinaryExpression

type BinaryExpression struct {
	Operation Operation
	Left      Expression
	Right     Expression
}

var _ Element = &BinaryExpression{}
var _ Expression = &BinaryExpression{}

func (*BinaryExpression) ElementType() ElementType {
	return ElementTypeBinaryExpression
}

func (*BinaryExpression) isExpression() {}

func (*BinaryExpression) isIfStatementTest() {}

func (e *BinaryExpression) Accept(visitor Visitor) Repr {
	return e.AcceptExp(visitor)
}

func (e *BinaryExpression) Walk(walkChild func(Element)) {
	walkChild(e.Left)
	walkChild(e.Right)
}

func (e *BinaryExpression) AcceptExp(visitor ExpressionVisitor) Repr {
	return visitor.VisitBinaryExpression(e)
}

func (e *BinaryExpression) String() string {
	return fmt.Sprintf(
		"(%s %s %s)",
		e.Left, e.Operation.Symbol(), e.Right,
	)
}

func (e *BinaryExpression) StartPosition() Position {
	return e.Left.StartPosition()
}

func (e *BinaryExpression) EndPosition() Position {
	return e.Right.EndPosition()
}

func (e *BinaryExpression) MarshalJSON() ([]byte, error) {
	type Alias BinaryExpression
	return json.Marshal(&struct {
		Type string
		Range
		*Alias
	}{
		Type:  "BinaryExpression",
		Range: NewRangeFromPositioned(e),
		Alias: (*Alias)(e),
	})
}

// FunctionExpression

type FunctionExpression struct {
	ParameterList        *ParameterList
	ReturnTypeAnnotation *TypeAnnotation
	FunctionBlock        *FunctionBlock
	StartPos             Position `json:"-"`
}

var _ Element = &FunctionExpression{}
var _ Expression = &FunctionExpression{}

func (*FunctionExpression) ElementType() ElementType {
	return ElementTypeFunctionExpression
}

func (*FunctionExpression) isExpression() {}

func (*FunctionExpression) isIfStatementTest() {}

func (e *FunctionExpression) Accept(visitor Visitor) Repr {
	return e.AcceptExp(visitor)
}

func (e *FunctionExpression) Walk(walkChild func(Element)) {
	// TODO: walk parameters
	// TODO: walk return type
	walkChild(e.FunctionBlock)
}

func (e *FunctionExpression) AcceptExp(visitor ExpressionVisitor) Repr {
	return visitor.VisitFunctionExpression(e)
}

func (e *FunctionExpression) String() string {
	// TODO:
	return "func ..."
}

func (e *FunctionExpression) StartPosition() Position {
	return e.StartPos
}

func (e *FunctionExpression) EndPosition() Position {
	return e.FunctionBlock.EndPosition()
}

func (e *FunctionExpression) MarshalJSON() ([]byte, error) {
	type Alias FunctionExpression
	return json.Marshal(&struct {
		Type string
		Range
		*Alias
	}{
		Type:  "FunctionExpression",
		Range: NewRangeFromPositioned(e),
		Alias: (*Alias)(e),
	})
}

// CastingExpression

type CastingExpression struct {
	Expression                Expression
	Operation                 Operation
	TypeAnnotation            *TypeAnnotation
	ParentVariableDeclaration *VariableDeclaration `json:"-"`
}

var _ Element = &CastingExpression{}
var _ Expression = &CastingExpression{}

func (*CastingExpression) ElementType() ElementType {
	return ElementTypeCastingExpression
}

func (*CastingExpression) isExpression() {}

func (*CastingExpression) isIfStatementTest() {}

func (e *CastingExpression) Accept(visitor Visitor) Repr {
	return e.AcceptExp(visitor)
}
func (e *CastingExpression) Walk(walkChild func(Element)) {
	walkChild(e.Expression)
	// TODO: also walk type
}

func (e *CastingExpression) AcceptExp(visitor ExpressionVisitor) Repr {
	return visitor.VisitCastingExpression(e)
}

func (e *CastingExpression) String() string {
	return fmt.Sprintf(
		"(%s %s %s)",
		e.Expression, e.Operation.Symbol(), e.TypeAnnotation,
	)
}

func (e *CastingExpression) StartPosition() Position {
	return e.Expression.StartPosition()
}

func (e *CastingExpression) EndPosition() Position {
	return e.TypeAnnotation.EndPosition()
}

func (e *CastingExpression) MarshalJSON() ([]byte, error) {
	type Alias CastingExpression
	return json.Marshal(&struct {
		Type string
		Range
		*Alias
	}{
		Type:  "CastingExpression",
		Range: NewRangeFromPositioned(e),
		Alias: (*Alias)(e),
	})
}

// CreateExpression

type CreateExpression struct {
	InvocationExpression *InvocationExpression
	StartPos             Position `json:"-"`
}

var _ Element = &CreateExpression{}
var _ Expression = &CreateExpression{}

func (*CreateExpression) ElementType() ElementType {
	return ElementTypeCreateExpression
}

func (*CreateExpression) isExpression() {}

func (*CreateExpression) isIfStatementTest() {}

func (e *CreateExpression) Accept(visitor Visitor) Repr {
	return e.AcceptExp(visitor)
}

func (e *CreateExpression) Walk(walkChild func(Element)) {
	walkChild(e.InvocationExpression)
}

func (e *CreateExpression) AcceptExp(visitor ExpressionVisitor) Repr {
	return visitor.VisitCreateExpression(e)
}

func (e *CreateExpression) String() string {
	return fmt.Sprintf(
		"(create %s)",
		e.InvocationExpression,
	)
}

func (e *CreateExpression) StartPosition() Position {
	return e.StartPos
}

func (e *CreateExpression) EndPosition() Position {
	return e.InvocationExpression.EndPos
}

func (e *CreateExpression) MarshalJSON() ([]byte, error) {
	type Alias CreateExpression
	return json.Marshal(&struct {
		Type string
		Range
		*Alias
	}{
		Type:  "CreateExpression",
		Range: NewRangeFromPositioned(e),
		Alias: (*Alias)(e),
	})
}

// DestroyExpression

type DestroyExpression struct {
	Expression Expression
	StartPos   Position `json:"-"`
}

var _ Element = &DestroyExpression{}
var _ Expression = &DestroyExpression{}

func (*DestroyExpression) ElementType() ElementType {
	return ElementTypeDestroyExpression
}

func (*DestroyExpression) isExpression() {}

func (*DestroyExpression) isIfStatementTest() {}

func (e *DestroyExpression) Accept(visitor Visitor) Repr {
	return e.AcceptExp(visitor)
}

func (e *DestroyExpression) Walk(walkChild func(Element)) {
	walkChild(e.Expression)
}

func (e *DestroyExpression) AcceptExp(visitor ExpressionVisitor) Repr {
	return visitor.VisitDestroyExpression(e)
}

func (e *DestroyExpression) String() string {
	return fmt.Sprintf(
		"(destroy %s)",
		e.Expression,
	)
}

func (e *DestroyExpression) StartPosition() Position {
	return e.StartPos
}

func (e *DestroyExpression) EndPosition() Position {
	return e.Expression.EndPosition()
}

func (e *DestroyExpression) MarshalJSON() ([]byte, error) {
	type Alias DestroyExpression
	return json.Marshal(&struct {
		Type string
		Range
		*Alias
	}{
		Type:  "DestroyExpression",
		Range: NewRangeFromPositioned(e),
		Alias: (*Alias)(e),
	})
}

// ReferenceExpression

type ReferenceExpression struct {
	Expression Expression
	Type       Type     `json:"TargetType"`
	StartPos   Position `json:"-"`
}

var _ Element = &ReferenceExpression{}
var _ Expression = &ReferenceExpression{}

func (*ReferenceExpression) ElementType() ElementType {
	return ElementTypeReferenceExpression
}

func (*ReferenceExpression) isExpression() {}

func (*ReferenceExpression) isIfStatementTest() {}

func (e *ReferenceExpression) Accept(visitor Visitor) Repr {
	return e.AcceptExp(visitor)
}

func (e *ReferenceExpression) Walk(walkChild func(Element)) {
	walkChild(e.Expression)
	// TODO: walk type
}

func (e *ReferenceExpression) AcceptExp(visitor ExpressionVisitor) Repr {
	return visitor.VisitReferenceExpression(e)
}

func (e *ReferenceExpression) String() string {
	return fmt.Sprintf(
		"(&%s as %s)",
		e.Expression,
		e.Type,
	)
}

func (e *ReferenceExpression) StartPosition() Position {
	return e.StartPos
}

func (e *ReferenceExpression) EndPosition() Position {
	return e.Type.EndPosition()
}

func (e *ReferenceExpression) MarshalJSON() ([]byte, error) {
	type Alias ReferenceExpression
	return json.Marshal(&struct {
		Type string
		Range
		*Alias
	}{
		Type:  "ReferenceExpression",
		Range: NewRangeFromPositioned(e),
		Alias: (*Alias)(e),
	})
}

// ForceExpression

type ForceExpression struct {
	Expression Expression
	EndPos     Position `json:"-"`
}

var _ Element = &ForceExpression{}
var _ Expression = &ForceExpression{}

func (*ForceExpression) ElementType() ElementType {
	return ElementTypeForceExpression
}

func (*ForceExpression) isExpression() {}

func (*ForceExpression) isIfStatementTest() {}

func (e *ForceExpression) Accept(visitor Visitor) Repr {
	return e.AcceptExp(visitor)
}

func (e *ForceExpression) Walk(walkChild func(Element)) {
	walkChild(e.Expression)
}

func (e *ForceExpression) AcceptExp(visitor ExpressionVisitor) Repr {
	return visitor.VisitForceExpression(e)
}

func (e *ForceExpression) String() string {
	return fmt.Sprintf("%s!", e.Expression)
}

func (e *ForceExpression) StartPosition() Position {
	return e.Expression.StartPosition()
}

func (e *ForceExpression) EndPosition() Position {
	return e.EndPos
}

func (e *ForceExpression) MarshalJSON() ([]byte, error) {
	type Alias ForceExpression
	return json.Marshal(&struct {
		Type string
		Range
		*Alias
	}{
		Type:  "ForceExpression",
		Range: NewRangeFromPositioned(e),
		Alias: (*Alias)(e),
	})
}

// PathExpression

type PathExpression struct {
	StartPos   Position `json:"-"`
	Domain     Identifier
	Identifier Identifier
}

var _ Element = &PathExpression{}
var _ Expression = &PathExpression{}

func (*PathExpression) ElementType() ElementType {
	return ElementTypePathExpression
}

func (*PathExpression) isExpression() {}

func (*PathExpression) isIfStatementTest() {}

func (e *PathExpression) Accept(visitor Visitor) Repr {
	return e.AcceptExp(visitor)
}

func (*PathExpression) Walk(_ func(Element)) {
	// NO-OP
}

func (e *PathExpression) AcceptExp(visitor ExpressionVisitor) Repr {
	return visitor.VisitPathExpression(e)
}

func (e *PathExpression) String() string {
	return fmt.Sprintf("/%s/%s", e.Domain, e.Identifier)
}

func (e *PathExpression) StartPosition() Position {
	return e.StartPos
}

func (e *PathExpression) EndPosition() Position {
	return e.Identifier.EndPosition()
}

func (e *PathExpression) MarshalJSON() ([]byte, error) {
	type Alias PathExpression
	return json.Marshal(&struct {
		Type string
		Range
		*Alias
	}{
		Type:  "PathExpression",
		Range: NewRangeFromPositioned(e),
		Alias: (*Alias)(e),
	})
}
