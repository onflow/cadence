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

import "fmt"

type Program struct {
	// all declarations, in the order they are defined
	Declarations []Declaration
	// Use `PragmaDeclarations()` instead
	_pragmaDeclarations []*PragmaDeclaration
	// Use `ImportDeclarations()` instead
	_importDeclarations []*ImportDeclaration
	// Use `InterfaceDeclarations()` instead
	_interfaceDeclarations []*InterfaceDeclaration
	// Use `CompositeDeclarations()` instead
	_compositeDeclarations []*CompositeDeclaration
	// Use `FunctionDeclarations()` instead
	_functionDeclarations []*FunctionDeclaration
	// Use `TransactionDeclarations()` instead
	_transactionDeclarations []*TransactionDeclaration
	// Use `ImportedPrograms()` instead
	_importedPrograms map[LocationID]*Program
	// Use `ImportedLocations()` instead
	_importLocations []Location
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

// ImportedPrograms returns the sub-programs imported by this program, indexed by location ID.
func (p *Program) ImportedPrograms() map[LocationID]*Program {
	if p._importedPrograms == nil {
		p._importedPrograms = make(map[LocationID]*Program)
	}

	return p._importedPrograms
}

// ImportLocations returns the import locations declared by this program.
func (p *Program) ImportLocations() []Location {
	if p._importLocations == nil {
		p.updateDeclarations()
	}

	return p._importLocations
}

type ImportResolver func(location Location) (*Program, error)

func (p *Program) ResolveImports(resolver ImportResolver) error {
	return p.resolveImports(
		resolver,
		map[LocationID]bool{},
		map[LocationID]*Program{},
	)
}

type CyclicImportsError struct {
	Location Location
}

func (e CyclicImportsError) Error() string {
	return fmt.Sprintf("cyclic import of `%s`", e.Location)
}

func (p *Program) resolveImports(
	resolver ImportResolver,
	resolving map[LocationID]bool,
	resolved map[LocationID]*Program,
) error {
	locations := p.ImportLocations()

	for _, location := range locations {

		imported, ok := resolved[location.ID()]
		if !ok {
			var err error
			imported, err = resolver(location)
			if err != nil {
				return err
			}
			if imported != nil {
				resolved[location.ID()] = imported
			}
		}

		if imported == nil {
			continue
		}

		p.setImportedProgram(location.ID(), imported)

		if resolving[location.ID()] {
			return CyclicImportsError{Location: location}
		}

		resolving[location.ID()] = true

		err := imported.resolveImports(resolver, resolving, resolved)
		if err != nil {
			return err
		}

		delete(resolving, location.ID())
	}

	return nil
}

// setImportedProgram adds an imported program to the set of imports, indexed by location ID.
func (p *Program) setImportedProgram(locationID LocationID, program *Program) {
	if p._importedPrograms == nil {
		p._importedPrograms = make(map[LocationID]*Program)
	}

	p._importedPrograms[locationID] = program
}

func (p *Program) updateDeclarations() {
	// Important: allocate instead of nil

	p._pragmaDeclarations = make([]*PragmaDeclaration, 0)
	p._importDeclarations = make([]*ImportDeclaration, 0)
	p._compositeDeclarations = make([]*CompositeDeclaration, 0)
	p._interfaceDeclarations = make([]*InterfaceDeclaration, 0)
	p._functionDeclarations = make([]*FunctionDeclaration, 0)
	p._transactionDeclarations = make([]*TransactionDeclaration, 0)
	p._importLocations = make([]Location, 0)

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

	for _, importDeclaration := range p.ImportDeclarations() {
		p._importLocations = append(p._importLocations, importDeclaration.Location)
	}
}
