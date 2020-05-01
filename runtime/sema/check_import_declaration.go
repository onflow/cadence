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

package sema

import (
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
)

func (checker *Checker) VisitImportDeclaration(_ *ast.ImportDeclaration) ast.Repr {
	// Handled in `declareImportDeclaration`
	panic(&UnreachableStatementError{})
}

func (checker *Checker) declareImportDeclaration(declaration *ast.ImportDeclaration) ast.Repr {

	location := declaration.Location
	locationID := location.ID()

	// TODO: allow repeated imports of different sets

	// Ensure the program is only imported once.
	// If it was imported before, report an error and return

	if checker.seenImports[locationID] {
		checker.report(
			&RepeatedImportError{
				ImportLocation: location,
				Range: ast.Range{
					StartPos: declaration.LocationPos,
					EndPos:   declaration.LocationPos,
				},
			},
		)
		return nil
	}
	checker.seenImports[locationID] = true

	// Find the imported program.
	// If it is not available, ask the import handler, if any.
	// If no handler is available or the location can also not be resolved using the handler,
	// report an error and return.

	imports := checker.Program.ImportedPrograms()

	var imp Import

	importedProgram := imports[locationID]

	if importedProgram != nil {

		importChecker, err := checker.EnsureLoaded(
			location,
			func() *ast.Program {
				return importedProgram
			},
		)
		if err != nil {
			checker.report(
				&ImportedProgramError{
					CheckerError:   err,
					ImportLocation: location,
					Pos:            declaration.LocationPos,
				},
			)
			return nil
		}

		imp = CheckerImport{importChecker}

	} else if checker.importHandler != nil {
		imp = checker.importHandler(location)
	}

	if imp == nil {
		checker.report(
			&UnresolvedImportError{
				ImportLocation: location,
				Range:          ast.NewRangeFromPositioned(declaration),
			},
		)
		return nil
	}

	// Attempt to import the requested value declarations

	foundValues, invalidAccessedValues := checker.importElements(
		checker.valueActivations,
		declaration.Identifiers,
		imp.AllValueElements(),
		imp.IsImportableValue,
	)

	// Attempt to import the requested type declarations

	foundTypes, invalidAccessedTypes := checker.importElements(
		checker.typeActivations,
		declaration.Identifiers,
		imp.AllTypeElements(),
		imp.IsImportableType,
	)

	// For each identifier, report if the import is invalid due to
	// restricted access and report an error (i.e. if there is
	// both a value and type with the same name, only report a single error)

	for _, identifier := range declaration.Identifiers {

		invalidAccessedElement, isInvalidAccess := invalidAccessedValues[identifier]
		if !isInvalidAccess {
			invalidAccessedElement, isInvalidAccess = invalidAccessedTypes[identifier]
		}

		if !isInvalidAccess {
			continue
		}

		checker.report(
			&InvalidAccessError{
				Name:              identifier.Identifier,
				RestrictingAccess: invalidAccessedElement.Access,
				DeclarationKind:   invalidAccessedElement.DeclarationKind,
				Range:             ast.NewRangeFromPositioned(identifier),
			},
		)
	}

	// Determine which requested declarations could neither be found
	// in the value nor in the type declarations of the imported program.
	// For each missing import, report an error and declare both a value
	// with an invalid type and an invalid type to avoid spurious errors
	// due to uses of the inaccessible value or type

	missing := make(map[ast.Identifier]bool, len(declaration.Identifiers))
	for _, identifier := range declaration.Identifiers {
		if foundValues[identifier] || foundTypes[identifier] {
			continue
		}

		missing[identifier] = true
	}

	checker.handleMissingImports(missing, location)

	return nil
}

// EnsureLoaded finds or create a checker for the imported program and checks it.
//
func (checker *Checker) EnsureLoaded(location ast.Location, loadProgram func() *ast.Program) (*Checker, *CheckerError) {
	locationID := location.ID()

	subChecker, ok := checker.allCheckers[locationID]
	if ok {
		return subChecker, nil
	}

	if !ok || subChecker == nil {
		var err error
		subChecker, err = NewChecker(
			loadProgram(),
			location,
			WithPredeclaredValues(checker.PredeclaredValues),
			WithPredeclaredTypes(checker.PredeclaredTypes),
			WithAccessCheckMode(checker.accessCheckMode),
			WithValidTopLevelDeclarationsHandler(checker.validTopLevelDeclarationsHandler),
			WithAllCheckers(checker.allCheckers),
			WithCheckHandler(checker.checkHandler),
			WithImportHandler(checker.importHandler),
		)
		if err == nil {
			checker.allCheckers[locationID] = subChecker
		}
	}

	// Check the imported program, if any.

	var checkerErr *CheckerError
	if subChecker.Program != nil {
		// NOTE: ignore generic `error`-typed result, get internal `*CheckerError`

		_ = subChecker.Check()
		checkerErr = subChecker.CheckerError()
	}

	return subChecker, checkerErr
}

func (checker *Checker) handleMissingImports(missing map[ast.Identifier]bool, importLocation ast.Location) {
	for identifier := range missing {
		checker.report(
			&NotExportedError{
				Name:           identifier.Identifier,
				ImportLocation: importLocation,
				Pos:            identifier.Pos,
			},
		)

		// NOTE: declare constant variable with invalid type to silence rest of program
		const access = ast.AccessPrivate

		_, err := checker.valueActivations.Declare(variableDeclaration{
			identifier:               identifier.Identifier,
			ty:                       &InvalidType{},
			access:                   access,
			kind:                     common.DeclarationKindValue,
			pos:                      identifier.Pos,
			isConstant:               true,
			allowOuterScopeShadowing: false,
		})
		checker.report(err)

		// NOTE: declare type with invalid type to silence rest of program
		_, err = checker.typeActivations.DeclareType(typeDeclaration{
			identifier:               identifier,
			ty:                       &InvalidType{},
			declarationKind:          common.DeclarationKindType,
			access:                   access,
			allowOuterScopeShadowing: false,
		})
		checker.report(err)
	}
}

func (checker *Checker) importElements(
	valueActivations *VariableActivations,
	requestedIdentifiers []ast.Identifier,
	availableElements map[string]ImportElement,
	filter func(name string) bool,
) (
	found map[ast.Identifier]bool,
	invalidAccessed map[ast.Identifier]ImportElement,
) {
	found = map[ast.Identifier]bool{}
	invalidAccessed = map[ast.Identifier]ImportElement{}

	// Determine which identifiers are imported /
	// which variables need to be declared

	explicitlyImported := map[string]ast.Identifier{}

	var elements map[string]ImportElement
	identifiersCount := len(requestedIdentifiers)
	if identifiersCount > 0 && availableElements != nil {
		elements = make(map[string]ImportElement, identifiersCount)
		for _, identifier := range requestedIdentifiers {
			name := identifier.Identifier
			element, ok := availableElements[name]
			if !ok {
				continue
			}
			elements[name] = element
			found[identifier] = true
			explicitlyImported[name] = identifier
		}
	} else {
		elements = availableElements
	}

	for name, element := range elements {

		if !filter(name) {
			continue
		}

		// If the variable can't be imported due to restricted access,
		// report an error, but still import the variable

		access := element.Access

		if !checker.isReadableAccess(access) {

			// If the variable was imported explicitly, report an error

			if identifier, ok := explicitlyImported[name]; ok {
				invalidAccessed[identifier] = element
			} else {
				// Don't import not explicitly imported inaccessible variable
				continue
			}
		}

		_, err := valueActivations.Declare(variableDeclaration{
			identifier: name,
			ty:         element.Type,
			// TODO: implies that type is "re-exported"
			access: access,
			kind:   element.DeclarationKind,
			// TODO:
			pos:                      ast.Position{},
			isConstant:               true,
			argumentLabels:           element.ArgumentLabels,
			allowOuterScopeShadowing: false,
		})
		checker.report(err)
	}

	return
}
