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

type Program struct {
	// all declarations, in the order they are defined
	Declarations []Declaration
	// Use `PragmaDeclarations()` instead
	_pragmaDeclarations []*PragmaDeclaration
	// Use `ImportDeclarations()` instead
	_importDeclarations []*ImportDeclaration
	// Use `Interfaces()` instead
	_interfaceDeclarations []*InterfaceDeclaration
	// Use `Composites()` instead
	_compositeDeclarations []*CompositeDeclaration
	// Use `FunctionDeclarations()` instead
	_functionDeclarations []*FunctionDeclaration
	// Use `TransactionDeclarations()` instead
	_transactionDeclarations []*TransactionDeclaration
}

func (p *Program) StartPosition() Position {
	if len(p.Declarations) == 0 {
		return Position{}
	}
	firstDeclaration := p.Declarations[0]
	return firstDeclaration.StartPosition()
}

func (p *Program) EndPosition() Position {
	count := len(p.Declarations)
	if count == 0 {
		return Position{}
	}
	lastDeclaration := p.Declarations[count-1]
	return lastDeclaration.EndPosition()
}

func (p *Program) Accept(visitor Visitor) Repr {
	return visitor.VisitProgram(p)
}

func (p *Program) PragmaDeclarations() []*PragmaDeclaration {
	if p._pragmaDeclarations == nil {
		p.updateDeclarations()
	}
	return p._pragmaDeclarations
}

func (p *Program) ImportDeclarations() []*ImportDeclaration {
	if p._importDeclarations == nil {
		p.updateDeclarations()
	}
	return p._importDeclarations
}

func (p *Program) InterfaceDeclarations() []*InterfaceDeclaration {
	if p._interfaceDeclarations == nil {
		p.updateDeclarations()
	}
	return p._interfaceDeclarations
}

func (p *Program) CompositeDeclarations() []*CompositeDeclaration {
	if p._compositeDeclarations == nil {
		p.updateDeclarations()
	}
	return p._compositeDeclarations
}

func (p *Program) FunctionDeclarations() []*FunctionDeclaration {
	if p._functionDeclarations == nil {
		p.updateDeclarations()
	}
	return p._functionDeclarations
}

func (p *Program) TransactionDeclarations() []*TransactionDeclaration {
	if p._transactionDeclarations == nil {
		p.updateDeclarations()
	}
	return p._transactionDeclarations
}

func (p *Program) updateDeclarations() {
	// Important: allocate instead of nil

	p._pragmaDeclarations = make([]*PragmaDeclaration, 0)
	p._importDeclarations = make([]*ImportDeclaration, 0)
	p._compositeDeclarations = make([]*CompositeDeclaration, 0)
	p._interfaceDeclarations = make([]*InterfaceDeclaration, 0)
	p._functionDeclarations = make([]*FunctionDeclaration, 0)
	p._transactionDeclarations = make([]*TransactionDeclaration, 0)

	for _, declaration := range p.Declarations {

		switch declaration := declaration.(type) {
		case *PragmaDeclaration:
			p._pragmaDeclarations = append(p._pragmaDeclarations, declaration)

		case *ImportDeclaration:
			p._importDeclarations = append(p._importDeclarations, declaration)

		case *CompositeDeclaration:
			p._compositeDeclarations = append(p._compositeDeclarations, declaration)

		case *InterfaceDeclaration:
			p._interfaceDeclarations = append(p._interfaceDeclarations, declaration)

		case *FunctionDeclaration:
			p._functionDeclarations = append(p._functionDeclarations, declaration)

		case *TransactionDeclaration:
			p._transactionDeclarations = append(p._transactionDeclarations, declaration)
		}
	}
}

func (p *Program) MarshalJSON() ([]byte, error) {
	type Alias Program
	return json.Marshal(&struct {
		Type string
		*Alias
	}{
		Type:  "Program",
		Alias: (*Alias)(p),
	})
}
