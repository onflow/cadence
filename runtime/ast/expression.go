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
	"math/big"
	"strings"

	"github.com/turbolent/prettier"

	"github.com/onflow/cadence/runtime/errors"

	"github.com/onflow/cadence/runtime/common"
)

const NilConstant = "nil"

type Expression interface {
	Element
	fmt.Stringer
	IfStatementTest
	isExpression()
	AcceptExp(ExpressionVisitor) Repr
	Doc() prettier.Doc
	precedence() precedence
}

// BoolExpression

type BoolExpression struct {
	Value bool
	Range
}

var _ Element = &BoolExpression{}
var _ Expression = &BoolExpression{}

func NewBoolExpression(gauge common.MemoryGauge, value bool, exprRange Range) *BoolExpression {
	common.UseMemory(gauge, common.BooleanExpressionMemoryUsage)
	return &BoolExpression{
		Value: value,
		Range: exprRange,
	}
}

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
	return Prettier(e)
}

var boolExpressionTrueDoc prettier.Doc = prettier.Text("true")
var boolExpressionFalseDoc prettier.Doc = prettier.Text("false")

func (e *BoolExpression) Doc() prettier.Doc {
	if e.Value {
		return boolExpressionTrueDoc
	} else {
		return boolExpressionFalseDoc
	}
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

func (*BoolExpression) precedence() precedence {
	return precedenceLiteral
}

// NilExpression

type NilExpression struct {
	Pos Position `json:"-"`
}

var _ Element = &NilExpression{}
var _ Expression = &NilExpression{}

func NewNilExpression(gauge common.MemoryGauge, pos Position) *NilExpression {
	common.UseMemory(gauge, common.NilExpressionMemoryUsage)
	return &NilExpression{
		Pos: pos,
	}
}

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
	return Prettier(e)
}

var nilExpressionDoc prettier.Doc = prettier.Text("nil")

func (*NilExpression) Doc() prettier.Doc {
	return nilExpressionDoc
}

func (e *NilExpression) StartPosition() Position {
	return e.Pos
}

func (e *NilExpression) EndPosition(memoryGauge common.MemoryGauge) Position {
	return e.Pos.Shifted(memoryGauge, len(NilConstant)-1)
}

func (e *NilExpression) MarshalJSON() ([]byte, error) {
	type Alias NilExpression
	return json.Marshal(&struct {
		Type string
		Range
		*Alias
	}{
		Type:  "NilExpression",
		Range: NewUnmeteredRangeFromPositioned(e),
		Alias: (*Alias)(e),
	})
}

func (*NilExpression) precedence() precedence {
	return precedenceLiteral
}

// StringExpression

type StringExpression struct {
	Value string
	Range
}

var _ Expression = &StringExpression{}

func NewStringExpression(gauge common.MemoryGauge, value string, exprRange Range) *StringExpression {
	common.UseMemory(gauge, common.StringExpressionMemoryUsage)
	return &StringExpression{
		Value: value,
		Range: exprRange,
	}
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
	return Prettier(e)
}

func (e *StringExpression) Doc() prettier.Doc {
	return prettier.Text(QuoteString(e.Value))
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

func (*StringExpression) precedence() precedence {
	return precedenceLiteral
}

// IntegerExpression

type IntegerExpression struct {
	PositiveLiteral string
	Value           *big.Int `json:"-"`
	Base            int
	Range
}

var _ Element = &IntegerExpression{}
var _ Expression = &IntegerExpression{}

func NewIntegerExpression(
	gauge common.MemoryGauge,
	literal string,
	value *big.Int,
	base int,
	tokenRange Range,
) *IntegerExpression {
	common.UseMemory(gauge, common.IntegerExpressionMemoryUsage)

	return &IntegerExpression{
		PositiveLiteral: literal,
		Value:           value,
		Base:            base,
		Range:           tokenRange,
	}
}

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
	return Prettier(e)
}

func (e *IntegerExpression) Doc() prettier.Doc {
	literal := e.PositiveLiteral
	if e.Value.Sign() < 0 {
		literal = "-" + literal
	}
	return prettier.Text(literal)
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

func (*IntegerExpression) precedence() precedence {
	return precedenceLiteral
}

// FixedPointExpression

type FixedPointExpression struct {
	PositiveLiteral string
	Negative        bool
	UnsignedInteger *big.Int `json:"-"`
	Fractional      *big.Int `json:"-"`
	Scale           uint
	Range
}

var _ Element = &FixedPointExpression{}
var _ Expression = &FixedPointExpression{}

func NewFixedPointExpression(
	gauge common.MemoryGauge,
	literal string,
	isNegative bool,
	integer *big.Int,
	fractional *big.Int,
	scale uint,
	tokenRange Range,
) *FixedPointExpression {
	common.UseMemory(gauge, common.FixedPointExpressionMemoryUsage)

	return &FixedPointExpression{
		PositiveLiteral: literal,
		Negative:        isNegative,
		UnsignedInteger: integer,
		Fractional:      fractional,
		Scale:           scale,
		Range:           tokenRange,
	}
}

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
	return Prettier(e)
}

func (e *FixedPointExpression) Doc() prettier.Doc {
	literal := e.PositiveLiteral
	if literal != "" {
		if e.Negative {
			literal = "-" + literal
		}
		return prettier.Text(literal)
	}

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
	return prettier.Text(builder.String())
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

func (*FixedPointExpression) precedence() precedence {
	return precedenceLiteral
}

// ArrayExpression

type ArrayExpression struct {
	Values []Expression
	Range
}

var _ Element = &ArrayExpression{}
var _ Expression = &ArrayExpression{}

func NewArrayExpression(
	gauge common.MemoryGauge,
	values []Expression,
	tokenRange Range,
) *ArrayExpression {

	common.UseMemory(gauge, common.NewArrayExpressionMemoryUsage(len(values)))

	return &ArrayExpression{
		Values: values,
		Range:  tokenRange,
	}
}

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
	return Prettier(e)
}

var arrayExpressionSeparatorDoc prettier.Doc = prettier.Concat{
	prettier.Text(","),
	prettier.Line{},
}

func (e *ArrayExpression) Doc() prettier.Doc {
	if len(e.Values) == 0 {
		return prettier.Text("[]")
	}

	elementDocs := make([]prettier.Doc, len(e.Values))
	for i, value := range e.Values {
		elementDocs[i] = value.Doc()
	}
	return prettier.WrapBrackets(
		prettier.Join(arrayExpressionSeparatorDoc, elementDocs...),
		prettier.SoftLine{},
	)
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

func (*ArrayExpression) precedence() precedence {
	return precedenceLiteral
}

// DictionaryExpression

type DictionaryExpression struct {
	Entries []DictionaryEntry
	Range
}

var _ Element = &DictionaryExpression{}
var _ Expression = &DictionaryExpression{}

func NewDictionaryExpression(
	gauge common.MemoryGauge,
	entries []DictionaryEntry,
	tokenRange Range,
) *DictionaryExpression {
	common.UseMemory(gauge, common.NewDictionaryExpressionMemoryUsage(len(entries)))

	return &DictionaryExpression{
		Entries: entries,
		Range:   tokenRange,
	}
}

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
	return Prettier(e)
}

var dictionaryExpressionSeparatorDoc prettier.Doc = prettier.Concat{
	prettier.Text(","),
	prettier.Line{},
}

func (e *DictionaryExpression) Doc() prettier.Doc {
	if len(e.Entries) == 0 {
		return prettier.Text("{}")
	}

	entryDocs := make([]prettier.Doc, len(e.Entries))
	for i, entry := range e.Entries {
		entryDocs[i] = entry.Doc()
	}

	return prettier.WrapBraces(
		prettier.Join(dictionaryExpressionSeparatorDoc, entryDocs...),
		prettier.SoftLine{},
	)
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

func (*DictionaryExpression) precedence() precedence {
	return precedenceLiteral
}

type DictionaryEntry struct {
	Key   Expression
	Value Expression
}

func NewDictionaryEntry(
	gauge common.MemoryGauge,
	key Expression,
	value Expression,
) DictionaryEntry {
	common.UseMemory(gauge, common.DictionaryEntryMemoryUsage)

	return DictionaryEntry{
		Key:   key,
		Value: value,
	}
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

var dictionaryKeyValueSeparatorDoc prettier.Doc = prettier.Concat{
	prettier.Text(":"),
	prettier.Line{},
}

func (e DictionaryEntry) Doc() prettier.Doc {
	keyDoc := e.Key.Doc()
	valueDoc := e.Value.Doc()

	return prettier.Group{
		Doc: prettier.Concat{
			keyDoc,
			dictionaryKeyValueSeparatorDoc,
			valueDoc,
		},
	}
}

// IdentifierExpression

type IdentifierExpression struct {
	Identifier Identifier
}

var _ Element = &IdentifierExpression{}
var _ Expression = &IdentifierExpression{}

func NewIdentifierExpression(
	gauge common.MemoryGauge,
	identifier Identifier,
) *IdentifierExpression {
	common.UseMemory(gauge, common.IdentifierExpressionMemoryUsage)

	return &IdentifierExpression{
		Identifier: identifier,
	}
}

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
	return Prettier(e)
}

func (e *IdentifierExpression) Doc() prettier.Doc {
	return prettier.Text(e.Identifier.Identifier)
}

func (e *IdentifierExpression) MarshalJSON() ([]byte, error) {
	type Alias IdentifierExpression
	return json.Marshal(&struct {
		Type string
		*Alias
		Range
	}{
		Type:  "IdentifierExpression",
		Range: NewUnmeteredRangeFromPositioned(e),
		Alias: (*Alias)(e),
	})
}

func (e *IdentifierExpression) StartPosition() Position {
	return e.Identifier.StartPosition()
}

func (e *IdentifierExpression) EndPosition(memoryGauge common.MemoryGauge) Position {
	return e.Identifier.EndPosition(memoryGauge)
}

func (*IdentifierExpression) precedence() precedence {
	return precedenceLiteral
}

// Arguments

type Arguments []*Argument

func (args Arguments) String() string {
	return Prettier(args)
}

var argumentsSeparatorDoc prettier.Doc = prettier.Concat{
	prettier.Text(","),
	prettier.Line{},
}

func (args Arguments) Doc() prettier.Doc {
	if len(args) == 0 {
		return prettier.Text("()")
	}

	argumentDocs := make([]prettier.Doc, len(args))
	for i, argument := range args {
		argumentDocs[i] = argument.Doc()
	}
	return prettier.WrapParentheses(
		prettier.Join(
			argumentsSeparatorDoc,
			argumentDocs...,
		),
		prettier.SoftLine{},
	)
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

func NewInvocationExpression(
	gauge common.MemoryGauge,
	invokedExpression Expression,
	typeArguments []*TypeAnnotation,
	arguments Arguments,
	argsStartPos Position,
	endPos Position,
) *InvocationExpression {
	common.UseMemory(gauge, common.InvocationExpressionMemoryUsage)

	return &InvocationExpression{
		InvokedExpression: invokedExpression,
		TypeArguments:     typeArguments,
		Arguments:         arguments,
		ArgumentsStartPos: argsStartPos,
		EndPos:            endPos,
	}
}

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
	return Prettier(e)
}

func (e *InvocationExpression) Doc() prettier.Doc {

	result := prettier.Concat{
		parenthesizedExpressionDoc(
			e.InvokedExpression,
			e.precedence(),
		),
	}

	if len(e.TypeArguments) > 0 {
		typeArgumentDocs := make([]prettier.Doc, len(e.TypeArguments))
		for i, typeArgument := range e.TypeArguments {
			typeArgumentDocs[i] = typeArgument.Doc()
		}

		result = append(result,
			prettier.Wrap(
				prettier.Text("<"),
				prettier.Join(arrayExpressionSeparatorDoc, typeArgumentDocs...),
				prettier.Text(">"),
				prettier.SoftLine{},
			),
		)
	}

	result = append(result, e.Arguments.Doc())

	return result
}

func (e *InvocationExpression) StartPosition() Position {
	return e.InvokedExpression.StartPosition()
}

func (e *InvocationExpression) EndPosition(_ common.MemoryGauge) Position {
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
		Range: NewUnmeteredRangeFromPositioned(e),
		Alias: (*Alias)(e),
	})
}

func (*InvocationExpression) precedence() precedence {
	return precedenceAccess
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

func NewMemberExpression(
	gauge common.MemoryGauge,
	expression Expression,
	optional bool,
	accessPos Position,
	identifier Identifier,
) *MemberExpression {
	common.UseMemory(gauge, common.MemberExpressionMemoryUsage)

	return &MemberExpression{
		Expression: expression,
		Optional:   optional,
		AccessPos:  accessPos,
		Identifier: identifier,
	}
}

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
	return Prettier(e)
}

var memberExpressionSeparatorDoc prettier.Doc = prettier.Text(".")
var memberExpressionOptionalSeparatorDoc prettier.Doc = prettier.Text("?.")

func (e *MemberExpression) Doc() prettier.Doc {
	var separatorDoc prettier.Doc
	if e.Optional {
		separatorDoc = memberExpressionOptionalSeparatorDoc
	} else {
		separatorDoc = memberExpressionSeparatorDoc
	}

	return prettier.Concat{
		parenthesizedExpressionDoc(
			e.Expression,
			e.precedence(),
		),
		prettier.Group{
			Doc: prettier.Indent{
				Doc: prettier.Concat{
					prettier.SoftLine{},
					separatorDoc,
					prettier.Text(e.Identifier.Identifier),
				},
			},
		},
	}
}

func (e *MemberExpression) StartPosition() Position {
	return e.Expression.StartPosition()
}

func (e *MemberExpression) EndPosition(memoryGauge common.MemoryGauge) Position {
	if e.Identifier.Identifier == "" {
		return e.AccessPos
	} else {
		return e.Identifier.EndPosition(memoryGauge)
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
		Range: NewUnmeteredRangeFromPositioned(e),
		Alias: (*Alias)(e),
	})
}

func (*MemberExpression) precedence() precedence {
	return precedenceAccess
}

// IndexExpression

type IndexExpression struct {
	TargetExpression   Expression
	IndexingExpression Expression
	Range
}

var _ Element = &IndexExpression{}
var _ Expression = &IndexExpression{}

func NewIndexExpression(
	gauge common.MemoryGauge,
	target Expression,
	index Expression,
	tokenRange Range,
) *IndexExpression {
	common.UseMemory(gauge, common.IndexExpressionMemoryUsage)

	return &IndexExpression{
		TargetExpression:   target,
		IndexingExpression: index,
		Range:              tokenRange,
	}
}

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
	return Prettier(e)
}

func (e *IndexExpression) Doc() prettier.Doc {
	return prettier.Concat{
		parenthesizedExpressionDoc(
			e.TargetExpression,
			e.precedence(),
		),
		prettier.WrapBrackets(
			e.IndexingExpression.Doc(),
			prettier.SoftLine{},
		),
	}
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

func (*IndexExpression) precedence() precedence {
	return precedenceAccess
}

// ConditionalExpression

type ConditionalExpression struct {
	Test Expression
	Then Expression
	Else Expression
}

var _ Element = &ConditionalExpression{}
var _ Expression = &ConditionalExpression{}

func NewConditionalExpression(
	gauge common.MemoryGauge,
	testExpr Expression,
	thenExpr Expression,
	elseExpr Expression,
) *ConditionalExpression {
	common.UseMemory(gauge, common.ConditionalExpressionMemoryUsage)

	return &ConditionalExpression{
		Test: testExpr,
		Then: thenExpr,
		Else: elseExpr,
	}
}

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
	return Prettier(e)
}

var conditionalExpressionTestSeparatorDoc prettier.Doc = prettier.Concat{
	prettier.Line{},
	prettier.Text("? "),
}
var conditionalExpressionBranchSeparatorDoc prettier.Doc = prettier.Concat{
	prettier.Line{},
	prettier.Text(": "),
}

func (e *ConditionalExpression) Doc() prettier.Doc {
	ownPrecedence := e.precedence()

	// NOTE: right associative

	testDoc := e.Test.Doc()
	testPrecedence := e.Test.precedence()

	if ownPrecedence >= testPrecedence {
		testDoc = prettier.WrapParentheses(testDoc, prettier.SoftLine{})
	}

	thenDoc := e.Then.Doc()
	thenPrecedence := e.Then.precedence()

	if ownPrecedence >= thenPrecedence {
		thenDoc = prettier.WrapParentheses(thenDoc, prettier.SoftLine{})
	}

	elseDoc := e.Else.Doc()
	elsePrecedence := e.Else.precedence()

	if ownPrecedence > elsePrecedence {
		elseDoc = prettier.WrapParentheses(elseDoc, prettier.SoftLine{})
	}

	return prettier.Group{
		Doc: prettier.Concat{
			testDoc,
			prettier.Indent{
				Doc: prettier.Concat{
					conditionalExpressionTestSeparatorDoc,
					prettier.Indent{
						Doc: thenDoc,
					},
					conditionalExpressionBranchSeparatorDoc,
					prettier.Indent{
						Doc: elseDoc,
					},
				},
			},
		},
	}
}

func (e *ConditionalExpression) StartPosition() Position {
	return e.Test.StartPosition()
}

func (e *ConditionalExpression) EndPosition(memoryGauge common.MemoryGauge) Position {
	return e.Else.EndPosition(memoryGauge)
}

func (e *ConditionalExpression) MarshalJSON() ([]byte, error) {
	type Alias ConditionalExpression
	return json.Marshal(&struct {
		Type string
		Range
		*Alias
	}{
		Type:  "ConditionalExpression",
		Range: NewUnmeteredRangeFromPositioned(e),
		Alias: (*Alias)(e),
	})
}

func (*ConditionalExpression) precedence() precedence {
	return precedenceTernary
}

// UnaryExpression

type UnaryExpression struct {
	Operation  Operation
	Expression Expression
	StartPos   Position `json:"-"`
}

var _ Element = &UnaryExpression{}
var _ Expression = &UnaryExpression{}

func NewUnaryExpression(
	gauge common.MemoryGauge,
	operation Operation,
	expression Expression,
	startPos Position,
) *UnaryExpression {
	common.UseMemory(gauge, common.UnaryExpressionMemoryUsage)

	return &UnaryExpression{
		Operation:  operation,
		Expression: expression,
		StartPos:   startPos,
	}
}

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
	return Prettier(e)
}

func parenthesizedExpressionDoc(e Expression, parentPrecedence precedence) prettier.Doc {
	doc := e.Doc()
	subPrecedence := e.precedence()
	if parentPrecedence <= subPrecedence {
		return doc
	}
	return prettier.WrapParentheses(
		doc,
		prettier.SoftLine{},
	)
}

func (e *UnaryExpression) Doc() prettier.Doc {
	return prettier.Concat{
		prettier.Text(e.Operation.Symbol()),
		parenthesizedExpressionDoc(
			e.Expression,
			e.precedence(),
		),
	}
}

func (e *UnaryExpression) StartPosition() Position {
	return e.StartPos
}

func (e *UnaryExpression) EndPosition(memoryGauge common.MemoryGauge) Position {
	return e.Expression.EndPosition(memoryGauge)
}

func (e *UnaryExpression) MarshalJSON() ([]byte, error) {
	type Alias UnaryExpression
	return json.Marshal(&struct {
		Type string
		Range
		*Alias
	}{
		Type:  "UnaryExpression",
		Range: NewUnmeteredRangeFromPositioned(e),
		Alias: (*Alias)(e),
	})
}

func (*UnaryExpression) precedence() precedence {
	return precedenceUnaryPrefix
}

// BinaryExpression

type BinaryExpression struct {
	Operation Operation
	Left      Expression
	Right     Expression
}

var _ Element = &BinaryExpression{}
var _ Expression = &BinaryExpression{}

func NewBinaryExpression(
	gauge common.MemoryGauge,
	operation Operation,
	left Expression,
	right Expression,
) *BinaryExpression {
	common.UseMemory(gauge, common.BinaryExpressionMemoryUsage)

	return &BinaryExpression{
		Operation: operation,
		Left:      left,
		Right:     right,
	}
}

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
	return Prettier(e)
}

func (e *BinaryExpression) Doc() prettier.Doc {

	ownPrecedence := e.precedence()
	isLeftAssociative := e.IsLeftAssociative()
	isRightAssociative := !isLeftAssociative

	leftDoc := e.Left.Doc()
	leftPrecedence := e.Left.precedence()

	if (isLeftAssociative && ownPrecedence > leftPrecedence) ||
		(isRightAssociative && ownPrecedence >= leftPrecedence) {

		leftDoc = prettier.WrapParentheses(leftDoc, prettier.SoftLine{})
	}

	rightDoc := e.Right.Doc()
	rightPrecedence := e.Right.precedence()

	if (isLeftAssociative && ownPrecedence >= rightPrecedence) ||
		(isRightAssociative && ownPrecedence > rightPrecedence) {

		rightDoc = prettier.WrapParentheses(rightDoc, prettier.SoftLine{})
	}

	return prettier.Group{
		Doc: prettier.Concat{
			prettier.Group{
				Doc: leftDoc,
			},
			prettier.Line{},
			prettier.Text(e.Operation.Symbol()),
			prettier.Space,
			prettier.Group{
				Doc: rightDoc,
			},
		},
	}
}

func (e *BinaryExpression) StartPosition() Position {
	return e.Left.StartPosition()
}

func (e *BinaryExpression) EndPosition(memoryGauge common.MemoryGauge) Position {
	return e.Right.EndPosition(memoryGauge)
}

func (e *BinaryExpression) MarshalJSON() ([]byte, error) {
	type Alias BinaryExpression
	return json.Marshal(&struct {
		Type string
		Range
		*Alias
	}{
		Type:  "BinaryExpression",
		Range: NewUnmeteredRangeFromPositioned(e),
		Alias: (*Alias)(e),
	})
}

func (e *BinaryExpression) precedence() precedence {
	switch e.Operation {
	case OperationOr:
		return precedenceLogicalOr
	case OperationAnd:
		return precedenceLogicalAnd
	case OperationEqual,
		OperationNotEqual,
		OperationLess,
		OperationLessEqual,
		OperationGreater,
		OperationGreaterEqual:
		return precedenceComparison
	case OperationNilCoalesce:
		return precedenceNilCoalescing
	case OperationBitwiseOr:
		return precedenceBitwiseOr
	case OperationBitwiseXor:
		return precedenceBitwiseXor
	case OperationBitwiseAnd:
		return precedenceBitwiseAnd
	case OperationBitwiseLeftShift, OperationBitwiseRightShift:
		return precedenceBitwiseShift
	case OperationPlus, OperationMinus:
		return precedenceAddition
	case OperationMul, OperationDiv, OperationMod:
		return precedenceMultiplication
	default:
		panic(errors.NewUnreachableError())
	}
}

func (e *BinaryExpression) IsLeftAssociative() bool {
	return e.Operation != OperationNilCoalesce
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

func NewFunctionExpression(
	gauge common.MemoryGauge,
	parameters *ParameterList,
	returnType *TypeAnnotation,
	functionBlock *FunctionBlock,
	startPos Position,
) *FunctionExpression {
	common.UseMemory(gauge, common.FunctionExpressionMemoryUsage)

	return &FunctionExpression{
		ParameterList:        parameters,
		ReturnTypeAnnotation: returnType,
		FunctionBlock:        functionBlock,
		StartPos:             startPos,
	}
}

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
	return Prettier(e)
}

var functionFunKeywordSpaceDoc prettier.Doc = prettier.Text("fun ")

var functionExpressionEmptyBlockDoc prettier.Doc = prettier.Text(" {}")

func FunctionDocument(
	access Access,
	includeKeyword bool,
	identifier string,
	parameterList *ParameterList,
	returnTypeAnnotation *TypeAnnotation,
	block *FunctionBlock,
) prettier.Doc {

	var signatureDoc prettier.Concat
	if parameterList != nil {
		signatureDoc = append(
			signatureDoc,
			parameterList.Doc(),
		)

		if returnTypeAnnotation != nil &&
			!IsEmptyType(returnTypeAnnotation.Type) {

			signatureDoc = append(
				signatureDoc,
				typeSeparatorSpaceDoc,
				returnTypeAnnotation.Doc(),
			)
		}
	}

	var doc prettier.Concat

	if access != AccessNotSpecified {
		doc = append(
			doc,
			prettier.Text(access.Keyword()),
			prettier.Space,
		)
	}

	if includeKeyword {
		doc = append(
			doc,
			functionFunKeywordSpaceDoc,
		)
	}

	if identifier != "" {
		doc = append(
			doc,
			prettier.Text(identifier),
		)
	}

	if signatureDoc != nil {
		doc = append(
			doc,
			prettier.Group{
				Doc: signatureDoc,
			},
		)
	}

	if block.IsEmpty() {
		return append(doc, functionExpressionEmptyBlockDoc)
	} else {
		blockDoc := block.Doc()

		return append(
			doc,
			prettier.Space,
			blockDoc,
		)
	}
}

func (e *FunctionExpression) Doc() prettier.Doc {
	return FunctionDocument(
		AccessNotSpecified,
		true,
		"",
		e.ParameterList,
		e.ReturnTypeAnnotation,
		e.FunctionBlock,
	)
}

func (e *FunctionExpression) StartPosition() Position {
	return e.StartPos
}

func (e *FunctionExpression) EndPosition(memoryGauge common.MemoryGauge) Position {
	return e.FunctionBlock.EndPosition(memoryGauge)
}

func (e *FunctionExpression) MarshalJSON() ([]byte, error) {
	type Alias FunctionExpression
	return json.Marshal(&struct {
		Type string
		Range
		*Alias
	}{
		Type:  "FunctionExpression",
		Range: NewUnmeteredRangeFromPositioned(e),
		Alias: (*Alias)(e),
	})
}

func (*FunctionExpression) precedence() precedence {
	return precedenceLiteral
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

func NewCastingExpression(
	gauge common.MemoryGauge,
	expression Expression,
	operation Operation,
	typeAnnotation *TypeAnnotation,
	parentVariableDecl *VariableDeclaration,
) *CastingExpression {
	common.UseMemory(gauge, common.CastingExpressionMemoryUsage)

	return &CastingExpression{
		Expression:                expression,
		Operation:                 operation,
		TypeAnnotation:            typeAnnotation,
		ParentVariableDeclaration: parentVariableDecl,
	}
}

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
	return Prettier(e)
}

func (e *CastingExpression) Doc() prettier.Doc {
	doc := parenthesizedExpressionDoc(
		e.Expression,
		e.precedence(),
	)

	return prettier.Group{
		Doc: prettier.Concat{
			prettier.Group{
				Doc: doc,
			},
			prettier.Line{},
			prettier.Text(e.Operation.Symbol()),
			prettier.Line{},
			e.TypeAnnotation.Doc(),
		},
	}
}

func (e *CastingExpression) StartPosition() Position {
	return e.Expression.StartPosition()
}

func (e *CastingExpression) EndPosition(memoryGauge common.MemoryGauge) Position {
	return e.TypeAnnotation.EndPosition(memoryGauge)
}

func (e *CastingExpression) MarshalJSON() ([]byte, error) {
	type Alias CastingExpression
	return json.Marshal(&struct {
		Type string
		Range
		*Alias
	}{
		Type:  "CastingExpression",
		Range: NewUnmeteredRangeFromPositioned(e),
		Alias: (*Alias)(e),
	})
}

func (*CastingExpression) precedence() precedence {
	return precedenceCasting
}

// CreateExpression

type CreateExpression struct {
	InvocationExpression *InvocationExpression
	StartPos             Position `json:"-"`
}

var _ Element = &CreateExpression{}
var _ Expression = &CreateExpression{}

func NewCreateExpression(
	gauge common.MemoryGauge,
	invocationExpression *InvocationExpression,
	startPos Position,
) *CreateExpression {
	common.UseMemory(gauge, common.CreateExpressionMemoryUsage)

	return &CreateExpression{
		InvocationExpression: invocationExpression,
		StartPos:             startPos,
	}
}

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
	return Prettier(e)
}

var createKeywordSpaceDoc = prettier.Text("create ")

func (e *CreateExpression) Doc() prettier.Doc {
	return prettier.Concat{
		createKeywordSpaceDoc,
		e.InvocationExpression.Doc(),
	}
}

func (e *CreateExpression) StartPosition() Position {
	return e.StartPos
}

func (e *CreateExpression) EndPosition(common.MemoryGauge) Position {
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
		Range: NewUnmeteredRangeFromPositioned(e),
		Alias: (*Alias)(e),
	})
}

func (*CreateExpression) precedence() precedence {
	return precedenceUnaryPrefix
}

// DestroyExpression

type DestroyExpression struct {
	Expression Expression
	StartPos   Position `json:"-"`
}

var _ Element = &DestroyExpression{}
var _ Expression = &DestroyExpression{}

func NewDestroyExpression(
	gauge common.MemoryGauge,
	expression Expression,
	startPos Position,
) *DestroyExpression {
	common.UseMemory(gauge, common.DestroyExpressionMemoryUsage)

	return &DestroyExpression{
		Expression: expression,
		StartPos:   startPos,
	}
}

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
	return Prettier(e)
}

const destroyExpressionKeywordDoc = prettier.Text("destroy ")

func (e *DestroyExpression) Doc() prettier.Doc {
	return prettier.Concat{
		destroyExpressionKeywordDoc,
		parenthesizedExpressionDoc(
			e.Expression,
			e.precedence(),
		),
	}
}

func (e *DestroyExpression) StartPosition() Position {
	return e.StartPos
}

func (e *DestroyExpression) EndPosition(memoryGauge common.MemoryGauge) Position {
	return e.Expression.EndPosition(memoryGauge)
}

func (e *DestroyExpression) MarshalJSON() ([]byte, error) {
	type Alias DestroyExpression
	return json.Marshal(&struct {
		Type string
		Range
		*Alias
	}{
		Type:  "DestroyExpression",
		Range: NewUnmeteredRangeFromPositioned(e),
		Alias: (*Alias)(e),
	})
}

func (*DestroyExpression) precedence() precedence {
	return precedenceUnaryPrefix
}

// ReferenceExpression

type ReferenceExpression struct {
	Expression Expression
	Type       Type     `json:"TargetType"`
	StartPos   Position `json:"-"`
}

var _ Element = &ReferenceExpression{}
var _ Expression = &ReferenceExpression{}

func NewReferenceExpression(
	gauge common.MemoryGauge,
	expression Expression,
	targetType Type,
	startPos Position,
) *ReferenceExpression {
	common.UseMemory(gauge, common.ReferenceExpressionMemoryUsage)

	return &ReferenceExpression{
		Expression: expression,
		Type:       targetType,
		StartPos:   startPos,
	}
}

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
	return Prettier(e)
}

var referenceExpressionRefOperatorDoc prettier.Doc = prettier.Text("&")
var referenceExpressionAsOperatorDoc prettier.Doc = prettier.Text("as")

func (e *ReferenceExpression) Doc() prettier.Doc {
	doc := parenthesizedExpressionDoc(
		e.Expression,
		e.precedence(),
	)

	return prettier.Group{
		Doc: prettier.Concat{
			referenceExpressionRefOperatorDoc,
			prettier.Group{
				Doc: doc,
			},
			prettier.Line{},
			referenceExpressionAsOperatorDoc,
			prettier.Line{},
			e.Type.Doc(),
		},
	}
}

func (e *ReferenceExpression) StartPosition() Position {
	return e.StartPos
}

func (e *ReferenceExpression) EndPosition(memoryGauge common.MemoryGauge) Position {
	return e.Type.EndPosition(memoryGauge)
}

func (e *ReferenceExpression) MarshalJSON() ([]byte, error) {
	type Alias ReferenceExpression
	return json.Marshal(&struct {
		Type string
		Range
		*Alias
	}{
		Type:  "ReferenceExpression",
		Range: NewUnmeteredRangeFromPositioned(e),
		Alias: (*Alias)(e),
	})
}

func (*ReferenceExpression) precedence() precedence {
	return precedenceUnaryPrefix
}

// ForceExpression

type ForceExpression struct {
	Expression Expression
	EndPos     Position `json:"-"`
}

var _ Element = &ForceExpression{}
var _ Expression = &ForceExpression{}

func NewForceExpression(
	gauge common.MemoryGauge,
	expression Expression,
	endPos Position,
) *ForceExpression {
	common.UseMemory(gauge, common.ForceExpressionMemoryUsage)

	return &ForceExpression{
		Expression: expression,
		EndPos:     endPos,
	}
}

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
	return Prettier(e)
}

const forceExpressionOperatorDoc = prettier.Text("!")

func (e *ForceExpression) Doc() prettier.Doc {
	return prettier.Concat{
		parenthesizedExpressionDoc(
			e.Expression,
			e.precedence(),
		),
		forceExpressionOperatorDoc,
	}
}

func (e *ForceExpression) StartPosition() Position {
	return e.Expression.StartPosition()
}

func (e *ForceExpression) EndPosition(common.MemoryGauge) Position {
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
		Range: NewUnmeteredRangeFromPositioned(e),
		Alias: (*Alias)(e),
	})
}

func (*ForceExpression) precedence() precedence {
	return precedenceUnaryPostfix
}

// PathExpression

type PathExpression struct {
	StartPos   Position `json:"-"`
	Domain     Identifier
	Identifier Identifier
}

var _ Element = &PathExpression{}
var _ Expression = &PathExpression{}

func NewPathExpression(
	gauge common.MemoryGauge,
	domain Identifier,
	identifier Identifier,
	startPos Position,
) *PathExpression {
	common.UseMemory(gauge, common.PathExpressionMemoryUsage)

	return &PathExpression{
		Domain:     domain,
		Identifier: identifier,
		StartPos:   startPos,
	}
}

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
	return Prettier(e)
}

var pathSeparatorDoc = prettier.Text("/")

func (e *PathExpression) Doc() prettier.Doc {
	return prettier.Concat{
		pathSeparatorDoc,
		prettier.Text(e.Domain.String()),
		pathSeparatorDoc,
		prettier.Text(e.Identifier.String()),
	}
}

func (e *PathExpression) StartPosition() Position {
	return e.StartPos
}

func (e *PathExpression) EndPosition(memoryGauge common.MemoryGauge) Position {
	return e.Identifier.EndPosition(memoryGauge)
}

func (e *PathExpression) MarshalJSON() ([]byte, error) {
	type Alias PathExpression
	return json.Marshal(&struct {
		Type string
		Range
		*Alias
	}{
		Type:  "PathExpression",
		Range: NewUnmeteredRangeFromPositioned(e),
		Alias: (*Alias)(e),
	})
}

func (*PathExpression) precedence() precedence {
	return precedenceLiteral
}
