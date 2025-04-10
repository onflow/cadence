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

package vm

import (
	"fmt"

	"github.com/onflow/cadence/bbq"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/interpreter"
)

type LinkedGlobals struct {
	// context shared by the globals in the program.
	executable *ExecutableProgram

	// globals defined in the program, indexed by name.
	indexedGlobals map[string]Value
}

// LinkGlobals performs the linking of global functions and variables for a given program.
func LinkGlobals(
	location common.Location,
	program *bbq.InstructionProgram,
	conf *Config,
	linkedGlobalsCache map[common.Location]LinkedGlobals,
) LinkedGlobals {

	var importedGlobals []Value

	for _, programImport := range program.Imports {
		importLocation := programImport.Location
		linkedGlobals, ok := linkedGlobalsCache[importLocation]

		if !ok {
			importedProgram := conf.ImportHandler(importLocation)

			// Link and get all globals at the import location.
			linkedGlobals = LinkGlobals(
				importLocation,
				importedProgram,
				conf,
				linkedGlobalsCache,
			)

			// If the imported program is a contract,
			// load the contract value and populate the global variable.
			if importedProgram.Contract != nil {
				contract := importedProgram.Contract
				location := common.NewAddressLocation(
					conf.MemoryGauge,
					common.MustBytesToAddress(contract.Address),
					contract.Name,
				)

				// TODO: remove this check. This shouldn't be nil ideally.
				if !contract.IsInterface && conf.ContractValueHandler != nil {
					contractValue := conf.ContractValueHandler(conf, location)
					// Update the globals - both the context and the mapping.
					// Contract value is always at the zero-th index.
					linkedGlobals.executable.Globals[0] = contractValue
					linkedGlobals.indexedGlobals[contract.Name] = contractValue
				}
			}

			linkedGlobalsCache[importLocation] = linkedGlobals
		}

		importedGlobal, ok := linkedGlobals.indexedGlobals[programImport.Name]
		if !ok {
			panic(LinkerError{
				Message: fmt.Sprintf("cannot find import '%s'", programImport.Name),
			})
		}
		importedGlobals = append(importedGlobals, importedGlobal)
	}

	executable := NewExecutableProgram(location, program, nil)

	globalsLen := len(program.Variables) + len(program.Functions) + len(importedGlobals) + 1
	indexedGlobalsLen := len(program.Functions)

	globals := make([]Value, 0, globalsLen)
	indexedGlobals := make(map[string]Value, indexedGlobalsLen)

	// If the current program is a contract, reserve a global variable for the contract value.
	// The reserved position is always the zero-th index.
	// This value will be populated either by the `init` method invocation of the contract,
	// Or when this program is imported by another (loads the value from storage).
	if program.Contract != nil && !program.Contract.IsInterface {
		globals = append(globals, nil)
	}

	for range program.Variables {
		globals = append(globals, nil)
	}

	// Iterate through `program.Functions` to be deterministic.
	// Order of globals must be same as index set at `Compiler.addGlobal()`.
	// TODO: include non-function globals
	for i := range program.Functions {
		function := &program.Functions[i]

		staticType := executable.StaticTypes[function.TypeIndex]
		funcStaticType, ok := staticType.(interpreter.FunctionStaticType)
		if !ok {
			panic(errors.NewUnreachableError())
		}

		value := FunctionValue{
			Function:   function,
			Executable: executable,
			Type:       funcStaticType,
		}

		globals = append(globals, value)
		indexedGlobals[function.Name] = value
	}

	// Globals of the current program are added first.
	// This is the same order as they are added in the compiler.
	// e.g: [global1, global2, ... [importedGlobal1, importedGlobal2, ...]]
	executable.Globals = globals
	executable.Globals = append(executable.Globals, importedGlobals...)

	// Return only the globals defined in the current program.
	// Because the importer/caller doesn't need to know globals of nested imports.
	return LinkedGlobals{
		executable:     executable,
		indexedGlobals: indexedGlobals,
	}
}
