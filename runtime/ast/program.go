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

type Program struct {
	// all declarations, in the order they are defined
	declarations []Declaration
	indices      programIndices
}

var _ Element = &Program{}

func NewProgram(memoryGauge common.MemoryGauge, declarations []Declaration) *Program {
	common.UseMemory(memoryGauge, common.ProgramMemoryUsage)
	return &Program{
		declarations: declarations,
	}
}

func (*Program) ElementType() ElementType {
	return ElementTypeProgram
}

func (p *Program) Declarations() []Declaration {
	return p.declarations
}

func (p *Program) StartPosition() Position {
	if len(p.declarations) == 0 {
		return EmptyPosition
	}
	firstDeclaration := p.declarations[0]
	return firstDeclaration.StartPosition()
}

func (p *Program) EndPosition(memoryGauge common.MemoryGauge) Position {
	count := len(p.declarations)
	if count == 0 {
		return EmptyPosition
	}
	lastDeclaration := p.declarations[count-1]
	return lastDeclaration.EndPosition(memoryGauge)
}

func (p *Program) Walk(walkChild func(Element)) {
	walkDeclarations(walkChild, p.declarations)
}

func (p *Program) PragmaDeclarations() []*PragmaDeclaration {
	return p.indices.pragmaDeclarations(p.declarations)
}

func (p *Program) ImportDeclarations() []*ImportDeclaration {
	return p.indices.importDeclarations(p.declarations)
}

func (p *Program) InterfaceDeclarations() []*InterfaceDeclaration {
	return p.indices.interfaceDeclarations(p.declarations)
}

func (p *Program) CompositeDeclarations() []*CompositeDeclaration {
	return p.indices.compositeDeclarations(p.declarations)
}

func (p *Program) AttachmentDeclarations() []*AttachmentDeclaration {
	return p.indices.attachmentDeclarations(p.declarations)
}

func (p *Program) FunctionDeclarations() []*FunctionDeclaration {
	return p.indices.functionDeclarations(p.declarations)
}

func (p *Program) TransactionDeclarations() []*TransactionDeclaration {
	return p.indices.transactionDeclarations(p.declarations)
}

func (p *Program) VariableDeclarations() []*VariableDeclaration {
	return p.indices.variableDeclarations(p.declarations)
}

// SoleContractDeclaration returns the sole contract declaration, if any,
// and if there are no other actionable declarations.
func (p *Program) SoleContractDeclaration() *CompositeDeclaration {

	compositeDeclarations := p.CompositeDeclarations()

	if len(compositeDeclarations) != 1 ||
		len(p.TransactionDeclarations()) > 0 ||
		len(p.InterfaceDeclarations()) > 0 ||
		len(p.FunctionDeclarations()) > 0 {

		return nil
	}

	compositeDeclaration := compositeDeclarations[0]

	if compositeDeclaration.CompositeKind != common.CompositeKindContract {
		return nil
	}

	return compositeDeclaration
}

// SoleContractInterfaceDeclaration returns the sole contract interface declaration, if any,
// and if there are no other actionable declarations.
func (p *Program) SoleContractInterfaceDeclaration() *InterfaceDeclaration {

	interfaceDeclarations := p.InterfaceDeclarations()

	if len(interfaceDeclarations) != 1 ||
		len(p.TransactionDeclarations()) > 0 ||
		len(p.FunctionDeclarations()) > 0 ||
		len(p.CompositeDeclarations()) > 0 ||
		len(p.AttachmentDeclarations()) > 0 {

		return nil
	}

	interfaceDeclaration := interfaceDeclarations[0]

	if interfaceDeclaration.CompositeKind != common.CompositeKindContract {
		return nil
	}

	return interfaceDeclaration
}

// SoleTransactionDeclaration returns the sole transaction declaration, if any,
// and if there are no other actionable declarations.
func (p *Program) SoleTransactionDeclaration() *TransactionDeclaration {

	transactionDeclarations := p.TransactionDeclarations()

	if len(transactionDeclarations) != 1 ||
		len(p.CompositeDeclarations()) > 0 ||
		len(p.InterfaceDeclarations()) > 0 ||
		len(p.FunctionDeclarations()) > 0 ||
		len(p.AttachmentDeclarations()) > 0 {

		return nil
	}

	return transactionDeclarations[0]
}

func (p *Program) MarshalJSON() ([]byte, error) {
	type Alias Program
	return json.Marshal(&struct {
		*Alias
		Type         string
		Declarations []Declaration
	}{
		Type:         "Program",
		Declarations: p.declarations,
		Alias:        (*Alias)(p),
	})
}

var programSeparatorDoc = prettier.Concat{
	prettier.HardLine{},
	prettier.HardLine{},
}

func (p *Program) Doc() prettier.Doc {
	declarations := p.Declarations()

	docs := make([]prettier.Doc, 0, len(declarations))

	for _, declaration := range declarations {
		docs = append(docs, declaration.Doc())
	}

	return prettier.Join(programSeparatorDoc, docs...)
}
