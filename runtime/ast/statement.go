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

func (*IfStatement) ElementType() ElementType {
	return ElementTypeIfStatement
}

func (*IfStatement) isStatement() {}

func (s *IfStatement) StartPosition() Position {
	return s.StartPos
}

func (s *IfStatement) EndPosition() Position {
	if s.Else != nil {
		return s.Else.EndPosition()
	}
	return s.Then.EndPosition()
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

func (s *IfStatement) MarshalJSON() ([]byte, error) {
	type Alias IfStatement
	return json.Marshal(&struct {
		Type string
		Range
		*Alias
	}{
		Type:  "IfStatement",
		Range: NewRangeFromPositioned(s),
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

func (s *WhileStatement) EndPosition() Position {
	return s.Block.EndPosition()
}

func (s *WhileStatement) MarshalJSON() ([]byte, error) {
	type Alias WhileStatement
	return json.Marshal(&struct {
		Type string
		Range
		*Alias
	}{
		Type:  "WhileStatement",
		Range: NewRangeFromPositioned(s),
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

func (s *ForStatement) EndPosition() Position {
	return s.Block.EndPosition()
}

func (s *ForStatement) MarshalJSON() ([]byte, error) {
	type Alias ForStatement
	return json.Marshal(&struct {
		Type string
		Range
		*Alias
	}{
		Type:  "ForStatement",
		Range: NewRangeFromPositioned(s),
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

func (*EmitStatement) ElementType() ElementType {
	return ElementTypeEmitStatement
}

func (*EmitStatement) isStatement() {}

func (s *EmitStatement) StartPosition() Position {
	return s.StartPos
}

func (s *EmitStatement) EndPosition() Position {
	return s.InvocationExpression.EndPosition()
}

func (s *EmitStatement) Accept(visitor Visitor) Repr {
	return visitor.VisitEmitStatement(s)
}

func (s *EmitStatement) Walk(walkChild func(Element)) {
	walkChild(s.InvocationExpression)
}

func (s *EmitStatement) MarshalJSON() ([]byte, error) {
	type Alias EmitStatement
	return json.Marshal(&struct {
		Type string
		Range
		*Alias
	}{
		Type:  "EmitStatement",
		Range: NewRangeFromPositioned(s),
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

func (s *AssignmentStatement) EndPosition() Position {
	return s.Value.EndPosition()
}

func (s *AssignmentStatement) Walk(walkChild func(Element)) {
	walkChild(s.Target)
	walkChild(s.Value)
}

func (s *AssignmentStatement) MarshalJSON() ([]byte, error) {
	type Alias AssignmentStatement
	return json.Marshal(&struct {
		Type string
		Range
		*Alias
	}{
		Type:  "AssignmentStatement",
		Range: NewRangeFromPositioned(s),
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

func (*SwapStatement) ElementType() ElementType {
	return ElementTypeSwapStatement
}

func (*SwapStatement) isStatement() {}

func (s *SwapStatement) StartPosition() Position {
	return s.Left.StartPosition()
}

func (s *SwapStatement) EndPosition() Position {
	return s.Right.EndPosition()
}

func (s *SwapStatement) Accept(visitor Visitor) Repr {
	return visitor.VisitSwapStatement(s)
}

func (s *SwapStatement) Walk(walkChild func(Element)) {
	walkChild(s.Left)
	walkChild(s.Right)
}

func (s *SwapStatement) MarshalJSON() ([]byte, error) {
	type Alias SwapStatement
	return json.Marshal(&struct {
		Type string
		Range
		*Alias
	}{
		Type:  "SwapStatement",
		Range: NewRangeFromPositioned(s),
		Alias: (*Alias)(s),
	})
}

// ExpressionStatement

type ExpressionStatement struct {
	Expression Expression
}

var _ Element = &ExpressionStatement{}
var _ Statement = &ExpressionStatement{}

func (*ExpressionStatement) ElementType() ElementType {
	return ElementTypeExpressionStatement
}

func (*ExpressionStatement) isStatement() {}

func (s *ExpressionStatement) StartPosition() Position {
	return s.Expression.StartPosition()
}

func (s *ExpressionStatement) EndPosition() Position {
	return s.Expression.EndPosition()
}

func (s *ExpressionStatement) Accept(visitor Visitor) Repr {
	return visitor.VisitExpressionStatement(s)
}

func (s *ExpressionStatement) Walk(walkChild func(Element)) {
	walkChild(s.Expression)
}

func (s *ExpressionStatement) MarshalJSON() ([]byte, error) {
	type Alias ExpressionStatement
	return json.Marshal(&struct {
		Type string
		Range
		*Alias
	}{
		Type:  "ExpressionStatement",
		Range: NewRangeFromPositioned(s),
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
		walkChild(switchCase.Expression)
		walkStatements(walkChild, switchCase.Statements)
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
