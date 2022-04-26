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

	"github.com/turbolent/prettier"

	"github.com/onflow/cadence/runtime/common"
)

type Statement interface {
	Element
	isStatement()
}

// ReturnStatement

type ReturnStatement struct {
	Expression Expression
	Range
}

var _ Element = &ReturnStatement{}
var _ Statement = &ReturnStatement{}

func NewReturnStatement(gauge common.MemoryGauge, expression Expression, stmtRange Range) *ReturnStatement {
	common.UseMemory(gauge, common.ReturnStatementMemoryUsage)
	return &ReturnStatement{
		Expression: expression,
		Range:      stmtRange,
	}
}

func (*ReturnStatement) ElementType() ElementType {
	return ElementTypeReturnStatement
}

func (*ReturnStatement) isStatement() {}

func (s *ReturnStatement) Accept(visitor Visitor) Repr {
	return visitor.VisitReturnStatement(s)
}

func (s *ReturnStatement) Walk(walkChild func(Element)) {
	if s.Expression != nil {
		walkChild(s.Expression)
	}
}

const returnStatementKeywordDoc = prettier.Text("return")
const returnStatementKeywordSpaceDoc = prettier.Text("return ")

func (s *ReturnStatement) Doc() prettier.Doc {
	if s.Expression == nil {
		return returnStatementKeywordDoc
	}

	return prettier.Concat{
		returnStatementKeywordSpaceDoc,
		// TODO: potentially parenthesize
		s.Expression.Doc(),
	}
}

func (s *ReturnStatement) MarshalJSON() ([]byte, error) {
	type Alias ReturnStatement
	return json.Marshal(&struct {
		Type string
		*Alias
	}{
		Type:  "ReturnStatement",
		Alias: (*Alias)(s),
	})
}

// BreakStatement

type BreakStatement struct {
	Range
}

var _ Element = &BreakStatement{}
var _ Statement = &BreakStatement{}

func NewBreakStatement(gauge common.MemoryGauge, tokenRange Range) *BreakStatement {
	common.UseMemory(gauge, common.BreakStatementMemoryUsage)
	return &BreakStatement{
		Range: tokenRange,
	}
}

func (*BreakStatement) ElementType() ElementType {
	return ElementTypeBreakStatement
}

func (*BreakStatement) isStatement() {}

func (s *BreakStatement) Accept(visitor Visitor) Repr {
	return visitor.VisitBreakStatement(s)
}

func (*BreakStatement) Walk(_ func(Element)) {
	// NO-OP
}

const breakStatementKeywordDoc = prettier.Text("break")

func (*BreakStatement) Doc() prettier.Doc {
	return breakStatementKeywordDoc
}

func (s *BreakStatement) MarshalJSON() ([]byte, error) {
	type Alias BreakStatement
	return json.Marshal(&struct {
		Type string
		*Alias
	}{
		Type:  "BreakStatement",
		Alias: (*Alias)(s),
	})
}

// ContinueStatement

type ContinueStatement struct {
	Range
}

var _ Element = &ContinueStatement{}
var _ Statement = &ContinueStatement{}

func NewContinueStatement(gauge common.MemoryGauge, tokenRange Range) *ContinueStatement {
	common.UseMemory(gauge, common.ContinueStatementMemoryUsage)
	return &ContinueStatement{
		Range: tokenRange,
	}
}

func (*ContinueStatement) ElementType() ElementType {
	return ElementTypeContinueStatement
}

func (*ContinueStatement) isStatement() {}

func (s *ContinueStatement) Accept(visitor Visitor) Repr {
	return visitor.VisitContinueStatement(s)
}

func (*ContinueStatement) Walk(_ func(Element)) {
	// NO-OP
}

const continueStatementKeywordDoc = prettier.Text("continue")

func (*ContinueStatement) Doc() prettier.Doc {
	return continueStatementKeywordDoc
}

func (s *ContinueStatement) MarshalJSON() ([]byte, error) {
	type Alias ContinueStatement
	return json.Marshal(&struct {
		Type string
		*Alias
	}{
		Type:  "ContinueStatement",
		Alias: (*Alias)(s),
	})
}

// IfStatementTest

type IfStatementTest interface {
	Element
	isIfStatementTest()
}

// IfStatement

type IfStatement struct {
	Test     IfStatementTest
	Then     *Block
	Else     *Block
	StartPos Position `json:"-"`
}

var _ Element = &IfStatement{}
var _ Statement = &IfStatement{}

func NewIfStatement(
	gauge common.MemoryGauge,
	test IfStatementTest,
	thenBlock *Block,
	elseBlock *Block,
	startPos Position,
) *IfStatement {
	common.UseMemory(gauge, common.IfStatementMemoryUsage)
	return &IfStatement{
		Test:     test,
		Then:     thenBlock,
		Else:     elseBlock,
		StartPos: startPos,
	}
}

func (*IfStatement) ElementType() ElementType {
	return ElementTypeIfStatement
}

func (*IfStatement) isStatement() {}

func (s *IfStatement) StartPosition() Position {
	return s.StartPos
}

func (s *IfStatement) EndPosition(memoryGauge common.MemoryGauge) Position {
	if s.Else != nil {
		return s.Else.EndPosition(memoryGauge)
	}
	return s.Then.EndPosition(memoryGauge)
}

func (s *IfStatement) Accept(visitor Visitor) Repr {
	return visitor.VisitIfStatement(s)
}

func (s *IfStatement) Walk(walkChild func(Element)) {
	walkChild(s.Test)
	walkChild(s.Then)
	if s.Else != nil {
		walkChild(s.Else)
	}
}

const ifStatementIfKeywordSpaceDoc = prettier.Text("if ")
const ifStatementSpaceElseKeywordSpaceDoc = prettier.Text(" else ")

func (s *IfStatement) Doc() prettier.Doc {
	var testDoc prettier.Doc
	// TODO: replace once IfStatementTest implements Doc
	testWithDoc, ok := s.Test.(interface{ Doc() prettier.Doc })
	if ok {
		testDoc = testWithDoc.Doc()
	}

	doc := prettier.Concat{
		ifStatementIfKeywordSpaceDoc,
		testDoc,
		prettier.Space,
		s.Then.Doc(),
	}

	if s.Else != nil {
		var elseDoc prettier.Doc
		if len(s.Else.Statements) == 1 {
			if elseIfStatement, ok := s.Else.Statements[0].(*IfStatement); ok {
				elseDoc = elseIfStatement.Doc()
			}
		}
		if elseDoc == nil {
			elseDoc = s.Else.Doc()
		}

		doc = append(
			doc,
			ifStatementSpaceElseKeywordSpaceDoc,
			prettier.Group{
				Doc: elseDoc,
			},
		)
	}

	return prettier.Group{
		Doc: doc,
	}
}

func (s *IfStatement) MarshalJSON() ([]byte, error) {
	type Alias IfStatement
	return json.Marshal(&struct {
		Type string
		Range
		*Alias
	}{
		Type:  "IfStatement",
		Range: NewUnmeteredRangeFromPositioned(s),
		Alias: (*Alias)(s),
	})
}

// WhileStatement

type WhileStatement struct {
	Test     Expression
	Block    *Block
	StartPos Position `json:"-"`
}

var _ Element = &WhileStatement{}
var _ Statement = &WhileStatement{}

func NewWhileStatement(
	gauge common.MemoryGauge,
	expression Expression,
	block *Block,
	startPos Position,
) *WhileStatement {
	common.UseMemory(gauge, common.WhileStatementMemoryUsage)
	return &WhileStatement{
		Test:     expression,
		Block:    block,
		StartPos: startPos,
	}
}

func (*WhileStatement) ElementType() ElementType {
	return ElementTypeWhileStatement
}

func (*WhileStatement) isStatement() {}

func (s *WhileStatement) Accept(visitor Visitor) Repr {
	return visitor.VisitWhileStatement(s)
}

func (s *WhileStatement) Walk(walkChild func(Element)) {
	walkChild(s.Test)
	walkChild(s.Block)
}

func (s *WhileStatement) StartPosition() Position {
	return s.StartPos
}

func (s *WhileStatement) EndPosition(memoryGauge common.MemoryGauge) Position {
	return s.Block.EndPosition(memoryGauge)
}

const whileStatementKeywordSpaceDoc = prettier.Text("while ")

func (s *WhileStatement) Doc() prettier.Doc {
	return prettier.Group{
		Doc: prettier.Concat{
			whileStatementKeywordSpaceDoc,
			s.Test.Doc(),
			prettier.Space,
			s.Block.Doc(),
		},
	}
}

func (s *WhileStatement) MarshalJSON() ([]byte, error) {
	type Alias WhileStatement
	return json.Marshal(&struct {
		Type string
		Range
		*Alias
	}{
		Type:  "WhileStatement",
		Range: NewUnmeteredRangeFromPositioned(s),
		Alias: (*Alias)(s),
	})
}

// ForStatement

type ForStatement struct {
	Identifier Identifier
	Index      *Identifier
	Value      Expression
	Block      *Block
	StartPos   Position `json:"-"`
}

var _ Element = &ForStatement{}
var _ Statement = &ForStatement{}

func NewForStatement(
	gauge common.MemoryGauge,
	identifier Identifier,
	index *Identifier,
	block *Block,
	expression Expression,
	startPos Position,
) *ForStatement {
	common.UseMemory(gauge, common.ForStatementMemoryUsage)

	return &ForStatement{
		Identifier: identifier,
		Index:      index,
		Block:      block,
		Value:      expression,
		StartPos:   startPos,
	}
}

func (*ForStatement) ElementType() ElementType {
	return ElementTypeForStatement
}

func (*ForStatement) isStatement() {}

func (s *ForStatement) Accept(visitor Visitor) Repr {
	return visitor.VisitForStatement(s)
}

func (s *ForStatement) Walk(walkChild func(Element)) {
	walkChild(s.Value)
	walkChild(s.Block)
}

func (s *ForStatement) StartPosition() Position {
	return s.StartPos
}

func (s *ForStatement) EndPosition(memoryGauge common.MemoryGauge) Position {
	return s.Block.EndPosition(memoryGauge)
}

const forStatementForKeywordSpaceDoc = prettier.Text("for ")
const forStatementSpaceInKeywordSpaceDoc = prettier.Text(" in ")

func (s *ForStatement) Doc() prettier.Doc {
	doc := prettier.Concat{
		forStatementForKeywordSpaceDoc,
	}

	if s.Index != nil {
		doc = append(
			doc,
			prettier.Text(s.Index.Identifier),
			prettier.Text(", "),
		)
	}

	doc = append(
		doc,
		prettier.Text(s.Identifier.Identifier),
		forStatementSpaceInKeywordSpaceDoc,
		s.Value.Doc(),
		prettier.Space,
		s.Block.Doc(),
	)

	return prettier.Group{
		Doc: doc,
	}
}

func (s *ForStatement) MarshalJSON() ([]byte, error) {
	type Alias ForStatement
	return json.Marshal(&struct {
		Type string
		Range
		*Alias
	}{
		Type:  "ForStatement",
		Range: NewUnmeteredRangeFromPositioned(s),
		Alias: (*Alias)(s),
	})
}

// EmitStatement

type EmitStatement struct {
	InvocationExpression *InvocationExpression
	StartPos             Position `json:"-"`
}

var _ Element = &EmitStatement{}
var _ Statement = &EmitStatement{}

func NewEmitStatement(
	gauge common.MemoryGauge,
	invocation *InvocationExpression,
	startPos Position,
) *EmitStatement {
	common.UseMemory(gauge, common.EmitStatementMemoryUsage)
	return &EmitStatement{
		InvocationExpression: invocation,
		StartPos:             startPos,
	}
}

func (*EmitStatement) ElementType() ElementType {
	return ElementTypeEmitStatement
}

func (*EmitStatement) isStatement() {}

func (s *EmitStatement) StartPosition() Position {
	return s.StartPos
}

func (s *EmitStatement) EndPosition(memoryGauge common.MemoryGauge) Position {
	return s.InvocationExpression.EndPosition(memoryGauge)
}

func (s *EmitStatement) Accept(visitor Visitor) Repr {
	return visitor.VisitEmitStatement(s)
}

func (s *EmitStatement) Walk(walkChild func(Element)) {
	walkChild(s.InvocationExpression)
}

const emitStatementKeywordSpaceDoc = prettier.Text("emit ")

func (s *EmitStatement) Doc() prettier.Doc {
	return prettier.Concat{
		emitStatementKeywordSpaceDoc,
		// TODO: potentially parenthesize
		s.InvocationExpression.Doc(),
	}
}

func (s *EmitStatement) MarshalJSON() ([]byte, error) {
	type Alias EmitStatement
	return json.Marshal(&struct {
		Type string
		Range
		*Alias
	}{
		Type:  "EmitStatement",
		Range: NewUnmeteredRangeFromPositioned(s),
		Alias: (*Alias)(s),
	})
}

// AssignmentStatement

type AssignmentStatement struct {
	Target   Expression
	Transfer *Transfer
	Value    Expression
}

var _ Element = &AssignmentStatement{}
var _ Statement = &AssignmentStatement{}

func NewAssignmentStatement(
	gauge common.MemoryGauge,
	expression Expression,
	transfer *Transfer,
	value Expression,
) *AssignmentStatement {
	common.UseMemory(gauge, common.AssignmentStatementMemoryUsage)

	return &AssignmentStatement{
		Target:   expression,
		Transfer: transfer,
		Value:    value,
	}
}

func (*AssignmentStatement) ElementType() ElementType {
	return ElementTypeAssignmentStatement
}

func (*AssignmentStatement) isStatement() {}

func (s *AssignmentStatement) Accept(visitor Visitor) Repr {
	return visitor.VisitAssignmentStatement(s)
}

func (s *AssignmentStatement) StartPosition() Position {
	return s.Target.StartPosition()
}

func (s *AssignmentStatement) EndPosition(memoryGauge common.MemoryGauge) Position {
	return s.Value.EndPosition(memoryGauge)
}

func (s *AssignmentStatement) Walk(walkChild func(Element)) {
	walkChild(s.Target)
	walkChild(s.Value)
}

func (s *AssignmentStatement) Doc() prettier.Doc {
	return prettier.Group{
		Doc: prettier.Concat{
			s.Target.Doc(),
			prettier.Space,
			s.Transfer.Doc(),
			prettier.Space,
			prettier.Group{
				Doc: prettier.Indent{
					Doc: s.Value.Doc(),
				},
			},
		},
	}
}

func (s *AssignmentStatement) MarshalJSON() ([]byte, error) {
	type Alias AssignmentStatement
	return json.Marshal(&struct {
		Type string
		Range
		*Alias
	}{
		Type:  "AssignmentStatement",
		Range: NewUnmeteredRangeFromPositioned(s),
		Alias: (*Alias)(s),
	})
}

// SwapStatement

type SwapStatement struct {
	Left  Expression
	Right Expression
}

var _ Element = &SwapStatement{}
var _ Statement = &SwapStatement{}

func NewSwapStatement(gauge common.MemoryGauge, expression Expression, right Expression) *SwapStatement {
	common.UseMemory(gauge, common.SwapStatementMemoryUsage)
	return &SwapStatement{
		Left:  expression,
		Right: right,
	}
}

func (*SwapStatement) ElementType() ElementType {
	return ElementTypeSwapStatement
}

func (*SwapStatement) isStatement() {}

func (s *SwapStatement) StartPosition() Position {
	return s.Left.StartPosition()
}

func (s *SwapStatement) EndPosition(memoryGauge common.MemoryGauge) Position {
	return s.Right.EndPosition(memoryGauge)
}

func (s *SwapStatement) Accept(visitor Visitor) Repr {
	return visitor.VisitSwapStatement(s)
}

func (s *SwapStatement) Walk(walkChild func(Element)) {
	walkChild(s.Left)
	walkChild(s.Right)
}

const swapStatementSpaceSymbolSpaceDoc = prettier.Text(" <-> ")

func (s *SwapStatement) Doc() prettier.Doc {
	return prettier.Group{
		Doc: prettier.Concat{
			s.Left.Doc(),
			swapStatementSpaceSymbolSpaceDoc,
			s.Right.Doc(),
		},
	}
}

func (s *SwapStatement) MarshalJSON() ([]byte, error) {
	type Alias SwapStatement
	return json.Marshal(&struct {
		Type string
		Range
		*Alias
	}{
		Type:  "SwapStatement",
		Range: NewUnmeteredRangeFromPositioned(s),
		Alias: (*Alias)(s),
	})
}

// ExpressionStatement

type ExpressionStatement struct {
	Expression Expression
}

var _ Element = &ExpressionStatement{}
var _ Statement = &ExpressionStatement{}

func NewExpressionStatement(gauge common.MemoryGauge, expression Expression) *ExpressionStatement {
	common.UseMemory(gauge, common.ExpressionStatementMemoryUsage)
	return &ExpressionStatement{
		Expression: expression,
	}
}

func (*ExpressionStatement) ElementType() ElementType {
	return ElementTypeExpressionStatement
}

func (*ExpressionStatement) isStatement() {}

func (s *ExpressionStatement) StartPosition() Position {
	return s.Expression.StartPosition()
}

func (s *ExpressionStatement) EndPosition(memoryGauge common.MemoryGauge) Position {
	return s.Expression.EndPosition(memoryGauge)
}

func (s *ExpressionStatement) Accept(visitor Visitor) Repr {
	return visitor.VisitExpressionStatement(s)
}

func (s *ExpressionStatement) Walk(walkChild func(Element)) {
	walkChild(s.Expression)
}

func (s *ExpressionStatement) Doc() prettier.Doc {
	return s.Expression.Doc()
}

func (s *ExpressionStatement) MarshalJSON() ([]byte, error) {
	type Alias ExpressionStatement
	return json.Marshal(&struct {
		Type string
		Range
		*Alias
	}{
		Type:  "ExpressionStatement",
		Range: NewUnmeteredRangeFromPositioned(s),
		Alias: (*Alias)(s),
	})
}

// SwitchStatement

type SwitchStatement struct {
	Expression Expression
	Cases      []*SwitchCase
	Range
}

var _ Element = &SwitchStatement{}
var _ Statement = &SwitchStatement{}

func NewSwitchStatement(
	gauge common.MemoryGauge,
	expression Expression,
	cases []*SwitchCase,
	stmtRange Range,
) *SwitchStatement {
	common.UseMemory(gauge, common.SwitchStatementMemoryUsage)
	return &SwitchStatement{
		Expression: expression,
		Cases:      cases,
		Range:      stmtRange,
	}
}

func (*SwitchStatement) ElementType() ElementType {
	return ElementTypeSwitchStatement
}

func (*SwitchStatement) isStatement() {}

func (s *SwitchStatement) Accept(visitor Visitor) Repr {
	return visitor.VisitSwitchStatement(s)
}

func (s *SwitchStatement) Walk(walkChild func(Element)) {
	walkChild(s.Expression)
	for _, switchCase := range s.Cases {
		// The default case has no expression
		expression := switchCase.Expression
		if expression != nil {
			walkChild(expression)
		}
		walkStatements(walkChild, switchCase.Statements)
	}
}

const switchStatementKeywordSpaceDoc = prettier.Text("switch ")

func (s *SwitchStatement) Doc() prettier.Doc {

	bodyDoc := make(prettier.Concat, 0, len(s.Cases))

	for _, switchCase := range s.Cases {
		bodyDoc = append(
			bodyDoc,
			prettier.HardLine{},
			switchCase.Doc(),
		)
	}

	return prettier.Concat{
		prettier.Group{
			Doc: prettier.Concat{
				switchStatementKeywordSpaceDoc,
				prettier.Indent{
					Doc: prettier.Concat{
						prettier.SoftLine{},
						s.Expression.Doc(),
					},
				},
				prettier.Line{},
			},
		},
		blockStartDoc,
		prettier.Indent{
			Doc: bodyDoc,
		},
		prettier.HardLine{},
		blockEndDoc,
	}
}

func (s *SwitchStatement) MarshalJSON() ([]byte, error) {
	type Alias SwitchStatement
	return json.Marshal(&struct {
		Type string
		*Alias
	}{
		Type:  "SwitchStatement",
		Alias: (*Alias)(s),
	})
}

// SwitchCase

type SwitchCase struct {
	Expression Expression
	Statements []Statement
	Range
}

func (s *SwitchCase) MarshalJSON() ([]byte, error) {
	type Alias SwitchCase
	return json.Marshal(&struct {
		Type string
		*Alias
	}{
		Type:  "SwitchCase",
		Alias: (*Alias)(s),
	})
}

const switchCaseKeywordSpaceDoc = prettier.Text("case ")
const switchCaseColonSymbolDoc = prettier.Text(":")
const switchCaseDefaultKeywordSpaceDoc = prettier.Text("default:")

func (s *SwitchCase) Doc() prettier.Doc {
	statementsDoc := prettier.Indent{
		Doc: StatementsDoc(s.Statements),
	}

	if s.Expression == nil {
		return prettier.Concat{
			switchCaseDefaultKeywordSpaceDoc,
			statementsDoc,
		}
	}

	return prettier.Concat{
		switchCaseKeywordSpaceDoc,
		s.Expression.Doc(),
		switchCaseColonSymbolDoc,
		statementsDoc,
	}
}
