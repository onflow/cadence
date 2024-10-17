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

package interpreter

import (
	"sort"
	"time"

	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/sema"
)

func (interpreter *Interpreter) VisitImportDeclaration(declaration *ast.ImportDeclaration) StatementResult {

	resolvedLocations := interpreter.Program.Elaboration.ImportDeclarationsResolvedLocations(declaration)

	for _, resolvedLocation := range resolvedLocations {
		interpreter.importResolvedLocation(resolvedLocation)
	}

	return nil
}

func (interpreter *Interpreter) importResolvedLocation(resolvedLocation sema.ResolvedLocation) {
	config := interpreter.SharedState.Config

	// tracing
	if config.TracingEnabled {
		startTime := time.Now()
		defer func() {
			interpreter.reportImportTrace(
				resolvedLocation.Location.String(),
				time.Since(startTime),
			)
		}()
	}

	subInterpreter := interpreter.EnsureLoaded(resolvedLocation.Location)

	// determine which identifiers are imported /
	// which variables need to be declared

	var variables map[string]Variable
	identifierLength := len(resolvedLocation.Identifiers)
	if identifierLength > 0 {
		variables = make(map[string]Variable, identifierLength)
		for _, identifier := range resolvedLocation.Identifiers {
			variables[identifier.Identifier] =
				subInterpreter.Globals.Get(identifier.Identifier)
		}
	} else {
		// Only take the global values defined in the program.
		variables = subInterpreter.Globals.variables
	}

	// Gather all variable names and sort them lexicographically

	var names []string

	for name := range variables { //nolint:maprange
		names = append(names, name)
	}

	// Set variables for all imported values in lexicographic order

	sort.Strings(names)

	for _, name := range names {
		variable := variables[name]
		if variable == nil {
			continue
		}

		// Lazily load the value
		getter := func() Value {
			value := variable.GetValue(interpreter)

			// If the variable is a contract value, then import it as a reference.
			// This must be done at the type of importing, rather than when declaring the contract value.
			compositeValue, ok := value.(*CompositeValue)
			if !ok || compositeValue.Kind != common.CompositeKindContract {
				return value
			}

			staticType := compositeValue.StaticType(interpreter)
			semaType, err := interpreter.ConvertStaticToSemaType(staticType)
			if err != nil {
				panic(err)
			}

			return NewEphemeralReferenceValue(
				interpreter,
				UnauthorizedAccess,
				compositeValue,
				semaType,
				LocationRange{
					Location: interpreter.Location,
				},
			)
		}

		importedVariable := NewVariableWithGetter(interpreter, getter)
		interpreter.setVariable(name, importedVariable)
		interpreter.Globals.Set(name, importedVariable)
	}
}
