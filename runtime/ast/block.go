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

type Block struct {
	Statements []Statement
	Range
}

var _ Element = &Block{}

func NewBlock(memoryGauge common.MemoryGauge, statements []Statement, astRange Range) *Block {
	common.UseMemory(memoryGauge, common.BlockMemoryUsage)

	return &Block{
		Statements: statements,
		Range:      astRange,
	}
}

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

var blockStartDoc prettier.Doc = prettier.Text("{")
var blockEndDoc prettier.Doc = prettier.Text("}")
var blockEmptyDoc prettier.Doc = prettier.Text("{}")

func (b *Block) Doc() prettier.Doc {
	if b.IsEmpty() {
		return blockEmptyDoc
	}

	return prettier.Concat{
		blockStartDoc,
		prettier.Indent{
			Doc: StatementsDoc(b.Statements),
		},
		prettier.HardLine{},
		blockEndDoc,
	}
}

func StatementsDoc(statements []Statement) prettier.Doc {
	var statementsDoc prettier.Concat

	for _, statement := range statements {
		// TODO: replace once Statement implements Doc
		hasDoc, ok := statement.(interface{ Doc() prettier.Doc })
		if !ok {
			continue
		}

		statementsDoc = append(
			statementsDoc,
			prettier.HardLine{},
			hasDoc.Doc(),
		)
	}

	return statementsDoc
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

func NewFunctionBlock(
	memoryGauge common.MemoryGauge,
	block *Block,
	preConditions *Conditions,
	postConditions *Conditions,
) *FunctionBlock {
	common.UseMemory(memoryGauge, common.FunctionBlockMemoryUsage)
	return &FunctionBlock{
		Block:          block,
		PreConditions:  preConditions,
		PostConditions: postConditions,
	}
}

func (b *FunctionBlock) IsEmpty() bool {
	return b == nil ||
		(b.Block.IsEmpty() &&
			b.PreConditions.IsEmpty() &&
			b.PostConditions.IsEmpty())
}

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

func (b *FunctionBlock) EndPosition(common.MemoryGauge) Position {
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

func (c *Conditions) IsEmpty() bool {
	return c == nil || len(*c) == 0
}
