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
	"github.com/onflow/cadence/bbq/opcode"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/interpreter"
)

type LinkedGlobals struct {
	// context shared by the globals in the program.
	executable *ExecutableProgram

	// globals defined in the program, indexed by name.
	indexedGlobals map[string]*Variable
}

// LinkGlobals performs the linking of global functions and variables for a given program.
func LinkGlobals(
	location common.Location,
	program *bbq.InstructionProgram,
	context *Context,
	linkedGlobalsCache map[common.Location]LinkedGlobals,
) LinkedGlobals {

	var importedGlobals []*Variable

	for _, programImport := range program.Imports {
		importLocation := programImport.Location
		linkedGlobals, ok := linkedGlobalsCache[importLocation]

		if !ok {
			importedProgram := context.ImportHandler(importLocation)

			// Link and get all globals at the import location.
			linkedGlobals = LinkGlobals(
				importLocation,
				importedProgram,
				context,
				linkedGlobalsCache,
			)

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

	globals := make([]*Variable, 0, globalsLen)
	indexedGlobals := make(map[string]*Variable, indexedGlobalsLen)

	for _, contract := range program.Contracts {
		// Update the globals - both the context and the mapping.
		// Contract value is always at the zero-th index.
		contractVariable := &interpreter.SimpleVariable{}
		contractVariable.InitializeWithGetter(func() interpreter.Value {
			return loadContractValue(contract, context)
		})
		globals = append(globals, contractVariable)
		indexedGlobals[contract.Name] = contractVariable
	}

	for _, variable := range program.Variables {
		simpleVariable := &interpreter.SimpleVariable{}

		// Some globals variables may not have initial values.
		// e.g: Transaction parameters are converted global variables,
		// where the values are being set in the transaction initializer.
		if variable.Getter != nil {
			valueGetter := functionValueFromBBQFunction(executable, variable.Getter)
			simpleVariable.InitializeWithGetter(func() interpreter.Value {
				return context.InvokeFunction(
					valueGetter,
					nil,
					nil,
					EmptyLocationRange,
				)
			})
		}

		globals = append(globals, simpleVariable)
		indexedGlobals[variable.Name] = simpleVariable
	}

	// Iterate through `program.Functions` to be deterministic.
	// Order of globals must be same as index set at `Compiler.addGlobal()`.
	for i := range program.Functions {
		function := &program.Functions[i]

		// Anonymous functions are not needed as global variables.
		// Compiler doesn't reserve global variable for them either.
		if function.IsAnonymous() {
			continue
		}

		var value FunctionValue

		if function.IsNative() {
			// Look-up using the unqualified name, in the common-builtin functions.
			value = IndexedCommonBuiltinTypeBoundFunctions[function.Name]
		} else {
			value = functionValueFromBBQFunction(executable, function)
		}

		variable := &interpreter.SimpleVariable{}
		variable.InitializeWithValue(value)
		globals = append(globals, variable)
		indexedGlobals[function.QualifiedName] = variable
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

func functionValueFromBBQFunction(
	executable *ExecutableProgram,
	function *bbq.Function[opcode.Instruction],
) FunctionValue {
	funcStaticType := getTypeFromExecutable[interpreter.FunctionStaticType](executable, function.TypeIndex)

	return CompiledFunctionValue{
		Function:   function,
		Executable: executable,
		Type:       funcStaticType,
	}
}

func loadContractValue(contract *bbq.Contract, context *Context) Value {

	if context.ContractValueHandler == nil {
		panic(errors.NewUnexpectedError(
			"missing contract value handler",
		))
	}

	location := common.NewAddressLocation(
		context.MemoryGauge,
		common.MustBytesToAddress(contract.Address),
		contract.Name,
	)

	var contractValue interpreter.Value = context.ContractValueHandler(context, location)

	staticType := contractValue.StaticType(context)
	semaType, err := interpreter.ConvertStaticToSemaType(context, staticType)
	if err != nil {
		panic(err)
	}

	return interpreter.NewEphemeralReferenceValue(
		context,
		interpreter.UnauthorizedAccess,
		contractValue,
		semaType,
		EmptyLocationRange,
	)
}
