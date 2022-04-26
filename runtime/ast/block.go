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

type Block struct {
	Statements []Statement
	Range
}

var _ Element = &Block{}

func (*Block) ElementType() ElementType {
	return ElementTypeBlock
}

func (b *Block) IsEmpty() bool {
	return len(b.Statements) == 0
}

func (b *Block) Accept(visitor Visitor) Repr {
	return visitor.VisitBlock(b)
}

func (b *Block) Walk(walkChild func(Element)) {
	walkStatements(walkChild, b.Statements)
}

func (b *Block) MarshalJSON() ([]byte, error) {
	type Alias Block
	return json.Marshal(&struct {
		Type string
		*Alias
	}{
		Type:  "Block",
		Alias: (*Alias)(b),
	})
}

// FunctionBlock

type FunctionBlock struct {
	Block          *Block
	PreConditions  *Conditions `json:",omitempty"`
	PostConditions *Conditions `json:",omitempty"`
}

var _ Element = &FunctionBlock{}

func (*FunctionBlock) ElementType() ElementType {
	return ElementTypeFunctionBlock
}

func (b *FunctionBlock) Accept(visitor Visitor) Repr {
	return visitor.VisitFunctionBlock(b)
}

func (b *FunctionBlock) Walk(walkChild func(Element)) {
	// TODO: pre-conditions
	walkChild(b.Block)
	// TODO: post-conditions
}

func (b *FunctionBlock) MarshalJSON() ([]byte, error) {
	type Alias FunctionBlock
	return json.Marshal(&struct {
		Type string
		Range
		*Alias
	}{
		Type:  "FunctionBlock",
		Range: b.Block.Range,
		Alias: (*Alias)(b),
	})
}

func (b *FunctionBlock) StartPosition() Position {
	return b.Block.StartPos
}

func (b *FunctionBlock) EndPosition() Position {
	return b.Block.EndPos
}

// Condition

type Condition struct {
	Kind    ConditionKind
	Test    Expression
	Message Expression
}

// Conditions

type Conditions []*Condition
