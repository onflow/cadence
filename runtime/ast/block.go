/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Dapper Labs, Inc.
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
	var doc prettier.Concat

	for _, statement := range statements {
		doc = append(
			doc,
			prettier.HardLine{},
			statement.Doc(),
		)
	}

	return doc
}

func (b *Block) String() string {
	return Prettier(b)
}

func (b *Block) MarshalJSON() ([]byte, error) {
	type Alias Block
	return json.Marshal(&struct {
		*Alias
		Type string
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

func (b *FunctionBlock) Walk(walkChild func(Element)) {
	// TODO: pre-conditions
	walkChild(b.Block)
	// TODO: post-conditions
}

func (b *FunctionBlock) MarshalJSON() ([]byte, error) {
	type Alias FunctionBlock
	return json.Marshal(&struct {
		*Alias
		Type string
		Range
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

var preConditionsKeywordDoc = prettier.Text("pre")
var postConditionsKeywordDoc = prettier.Text("post")

func (b *FunctionBlock) Doc() prettier.Doc {
	if b.IsEmpty() {
		return blockEmptyDoc
	}

	var conditionDocs []prettier.Doc

	if conditionsDoc := b.PreConditions.Doc(preConditionsKeywordDoc); conditionsDoc != nil {
		conditionDocs = append(
			conditionDocs,
			prettier.HardLine{},
			conditionsDoc,
		)
	}

	if conditionsDoc := b.PostConditions.Doc(postConditionsKeywordDoc); conditionsDoc != nil {
		conditionDocs = append(
			conditionDocs,
			prettier.HardLine{},
			conditionsDoc,
		)
	}

	var bodyDoc prettier.Doc

	statementsDoc := StatementsDoc(b.Block.Statements)

	if len(conditionDocs) > 0 {
		bodyConcatDoc := prettier.Concat(conditionDocs)
		bodyConcatDoc = append(
			bodyConcatDoc,
			statementsDoc,
		)
		bodyDoc = bodyConcatDoc
	} else {
		bodyDoc = statementsDoc
	}

	return prettier.Concat{
		blockStartDoc,
		prettier.Indent{
			Doc: bodyDoc,
		},
		prettier.HardLine{},
		blockEndDoc,
	}
}

func (b *FunctionBlock) String() string {
	return Prettier(b)
}

func (b *FunctionBlock) HasStatements() bool {
	return b != nil && len(b.Block.Statements) > 0
}

// Condition

type Condition struct {
	Test    Expression
	Message Expression
	Kind    ConditionKind
}

func (c Condition) Doc() prettier.Doc {
	doc := c.Test.Doc()
	if c.Message != nil {
		doc = prettier.Concat{
			doc,
			prettier.Text(":"),
			prettier.Indent{
				Doc: prettier.Concat{
					prettier.HardLine{},
					c.Message.Doc(),
				},
			},
		}
	}

	return prettier.Group{
		Doc: doc,
	}
}

// Conditions

type Conditions []*Condition

func (c *Conditions) IsEmpty() bool {
	return c == nil || len(*c) == 0
}

func (c *Conditions) Doc(keywordDoc prettier.Doc) prettier.Doc {
	if c.IsEmpty() {
		return nil
	}

	var doc prettier.Concat

	for _, condition := range *c {
		doc = append(
			doc,
			prettier.HardLine{},
			condition.Doc(),
		)
	}

	return prettier.Group{
		Doc: prettier.Concat{
			keywordDoc,
			prettier.Space,
			blockStartDoc,
			prettier.Indent{
				Doc: doc,
			},
			prettier.HardLine{},
			blockEndDoc,
		},
	}
}
