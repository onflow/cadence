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

	"github.com/onflow/cadence/common"
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
	walkElements(walkChild, p.declarations)
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

func (p *Program) EntitlementDeclarations() []*EntitlementDeclaration {
	return p.indices.entitlementDeclarations(p.declarations)
}

func (p *Program) EntitlementMappingDeclarations() []*EntitlementMappingDeclaration {
	return p.indices.entitlementMappingDeclarations(p.declarations)
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

// importGroupOrder returns the sort group for an import:
// 0 = identifier (standard), 1 = address, 2 = string, 3 = other.
// Imports in the same group render tight (no blank line between); imports
// in different groups render with a blank line between.
func importGroupOrder(imp *ImportDeclaration) int {
	switch imp.Location.(type) {
	case common.IdentifierLocation:
		return 0
	case common.AddressLocation:
		return 1
	case common.StringLocation:
		return 2
	default:
		return 3
	}
}

// declSeparatorHardLineCount returns the number of HardLines to insert
// between two consecutive top-level declarations.
// Same-group imports get one; otherwise at least two (a blank line).
// The blank line is also requested when the source had one between the pair.
func declSeparatorHardLineCount(ctx PrettyContext, prev, next Declaration) int {
	prevImp, prevIsImport := prev.(*ImportDeclaration)
	nextImp, nextIsImport := next.(*ImportDeclaration)
	if prevIsImport && nextIsImport {
		if importGroupOrder(prevImp) == importGroupOrder(nextImp) {
			return 1
		}
		return 2
	}
	if ctx.BlankLineBetween(prev, next) {
		return 2
	}
	return 2
}

func (p *Program) Doc(ctx PrettyContext) prettier.Doc {
	declarations := p.Declarations()

	var parts prettier.Concat

	if header := ctx.Header(); header != nil {
		parts = append(parts, header)
	}

	for i, declaration := range declarations {
		if i > 0 {
			previousDeclaration := declarations[i-1]
			sep := declSeparatorHardLineCount(ctx, previousDeclaration, declaration)
			for j := 0; j < sep; j++ {
				parts = append(parts, prettier.HardLine{})
			}
		}
		parts = append(parts, docOrEmpty(declaration, ctx))
	}

	if footer := ctx.Footer(); footer != nil {
		parts = append(parts, footer)
	}

	if len(declarations) == 0 && len(parts) == 0 {
		return ctx.Wrap(p, prettier.Text(""))
	}

	return ctx.Wrap(p, parts)
}

func (p *Program) String() string {
	return Prettier(p)
}
