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
	b.PreConditions.Walk(walkChild)
	walkChild(b.Block)
	b.PostConditions.Walk(walkChild)
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

func (b *FunctionBlock) HasConditions() bool {
	return b != nil &&
		(!b.PreConditions.IsEmpty() || !b.PostConditions.IsEmpty())
}

// Condition

type Condition interface {
	Element
	isCondition()
	CodeElement() Element
	Doc() prettier.Doc
	HasPosition
}

// TestCondition

type TestCondition struct {
	Test    Expression
	Message Expression
}

func (c TestCondition) ElementType() ElementType {
	return ElementTypeUnknown
}

func (c TestCondition) Walk(walkChild func(Element)) {
	walkChild(c.Test)
	if c.Message != nil {
		walkChild(c.Message)
	}
}

var _ Condition = TestCondition{}

func (c TestCondition) isCondition() {}

func (c TestCondition) CodeElement() Element {
	return c.Test
}

func (c TestCondition) StartPosition() Position {
	return c.Test.StartPosition()
}

func (c TestCondition) EndPosition(memoryGauge common.MemoryGauge) Position {
	if c.Message == nil {
		return c.Test.EndPosition(memoryGauge)
	}
	return c.Message.EndPosition(memoryGauge)
}

func (c TestCondition) MarshalJSON() ([]byte, error) {
	type Alias TestCondition
	return json.Marshal(&struct {
		Alias
		Type string
		Range
	}{
		Type:  "TestCondition",
		Range: NewUnmeteredRangeFromPositioned(c),
		Alias: (Alias)(c),
	})
}

func (c TestCondition) Doc() prettier.Doc {
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

// EmitCondition

type EmitCondition EmitStatement

var _ Condition = &EmitCondition{}

func (c *EmitCondition) isCondition() {}

func (c *EmitCondition) CodeElement() Element {
	return (*EmitStatement)(c)
}

func (c *EmitCondition) StartPosition() Position {
	return (*EmitStatement)(c).StartPosition()
}

func (c *EmitCondition) EndPosition(memoryGauge common.MemoryGauge) Position {
	return (*EmitStatement)(c).EndPosition(memoryGauge)
}

func (c *EmitCondition) Doc() prettier.Doc {
	return (*EmitStatement)(c).Doc()
}

func (c *EmitCondition) MarshalJSON() ([]byte, error) {
	type Alias EmitCondition
	return json.Marshal(&struct {
		*Alias
		Type string
		Range
	}{
		Type:  "EmitCondition",
		Range: NewUnmeteredRangeFromPositioned(c),
		Alias: (*Alias)(c),
	})
}

func (c *EmitCondition) ElementType() ElementType {
	return (*EmitStatement)(c).ElementType()
}

func (c *EmitCondition) Walk(walkChild func(Element)) {
	(*EmitStatement)(c).Walk(walkChild)
}

// Conditions

type Conditions []Condition

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

func (c *Conditions) Walk(walkChild func(Element)) {
	if c.IsEmpty() {
		return
	}

	for _, condition := range *c {
		walkChild(condition)
	}
}
