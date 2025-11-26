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

	"github.com/onflow/cadence/activations"
	"github.com/onflow/cadence/bbq"
	"github.com/onflow/cadence/bbq/opcode"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/interpreter"
)

type LinkedGlobals struct {
	// globals defined in the program, indexed by name.
	indexedGlobals *activations.Activation[Variable]
}

// LinkGlobals performs the linking of global functions and variables for a given program.
func LinkGlobals(
	memoryGauge common.MemoryGauge,
	location common.Location,
	program *bbq.InstructionProgram,
	context *Context,
	linkedGlobalsCache map[common.Location]LinkedGlobals,
) LinkedGlobals {

	if linkedGlobals, ok := linkedGlobalsCache[location]; ok {
		return linkedGlobals
	}

	executable := NewExecutableProgram(location, program, nil)

	// reserved globals for the current program (exact)
	globals := make([]Variable, len(program.Globals))
	indexedGlobals := activations.NewActivation[Variable](memoryGauge, nil)

	// NOTE: ensure both the context and the mapping are updated

	for _, global := range program.Globals {
		index := int(global.GetGlobalInfo().Index)

		switch typedGlobal := global.(type) {
		case *bbq.FunctionGlobal[opcode.Instruction]:
			function := typedGlobal.Function
			var value FunctionValue

			if function.IsNative() {
				// Look-up using the unqualified name, in the common-builtin functions.
				value = IndexedCommonBuiltinTypeBoundFunctions[function.Name]
			} else {
				value = functionValueFromBBQFunction(executable, function)
			}

			variable := &interpreter.SimpleVariable{}
			variable.InitializeWithValue(value)
			// Linker matches the compiled function index with the linked function index
			globals[index] = variable
			indexedGlobals.Set(function.QualifiedName, variable)
		case *bbq.VariableGlobal[opcode.Instruction]:
			variable := typedGlobal.Variable
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
					)
				})
			}
			// Linker matches the compiled variable index with the linked variable index
			globals[index] = simpleVariable
			indexedGlobals.Set(variable.Name, simpleVariable)
		case *bbq.ContractGlobal:
			contract := typedGlobal.Contract
			contractVariable := interpreter.NewContractVariableWithGetter(
				memoryGauge,
				func() interpreter.Value {
					return loadContractValue(contract, context)
				},
			)
			// Linker matches the compiled contract index with the linked contract index
			globals[index] = contractVariable
			indexedGlobals.Set(contract.Name, contractVariable)
		case *bbq.ImportedGlobal:
			importedGlobal := linkImportedGlobal(
				memoryGauge,
				location,
				typedGlobal,
				context,
				linkedGlobalsCache,
			)
			globals[index] = importedGlobal

			// Don't need to add to the `indexedGlobals`, since, like the below comment says,
			// importer/caller doesn't need to know globals of nested imports
		default:
			panic(errors.NewUnexpectedError("unsupported global type: %T", global))
		}
	}

	executable.Globals = globals

	linkedGlobals := LinkedGlobals{
		indexedGlobals: indexedGlobals,
	}

	linkedGlobalsCache[location] = linkedGlobals

	// Ensure all linked globals are initialized, just after linking.
	for _, global := range globals {
		if global.Kind() == interpreter.VariableKindContract {
			continue
		}
		global.GetValue(context)
	}

	// Return only the globals defined in the current program.
	// Because the importer/caller doesn't need to know globals of nested imports.
	return linkedGlobals
}

func linkImportedGlobal(
	memoryGauge common.MemoryGauge,
	location common.Location,
	importedGlobal *bbq.ImportedGlobal,
	context *Context,
	linkedGlobalsCache map[common.Location]LinkedGlobals,
) Variable {
	importLocation := importedGlobal.Location

	var indexedGlobals *activations.Activation[Variable]

	if importLocation == nil {
		if context.BuiltinGlobalsProvider == nil {
			indexedGlobals = DefaultBuiltinGlobals()
		} else {
			indexedGlobals = context.BuiltinGlobalsProvider(location)
		}
	} else {

		linkedGlobals, ok := linkedGlobalsCache[importLocation]
		if !ok {
			importedProgram := context.ImportHandler(importLocation)

			// Link and get all globals at the import location.
			linkedGlobals = LinkGlobals(
				memoryGauge,
				importLocation,
				importedProgram,
				context,
				linkedGlobalsCache,
			)
		}

		indexedGlobals = linkedGlobals.indexedGlobals
	}

	// When linking/finding the global in the imported program,
	// use the unqualified-name.
	// Because
	global := indexedGlobals.Find(importedGlobal.Name)
	if global == nil {
		panic(LinkerError{
			Message: fmt.Sprintf("cannot find import '%s'", importedGlobal.Name),
		})
	}

	linkedImportedGlobal := global

	if global.Kind() == interpreter.VariableKindContract {
		// If the variable is a contract value, then import it as a reference.
		// This must be done at the type of importing, rather than when declaring the contract value.
		linkedImportedGlobal = interpreter.NewContractVariableWithGetter(
			memoryGauge,
			func() interpreter.Value {
				// TODO: Is this the right context?
				contractValue := global.GetValue(context)

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
				)
			},
		)
	}
	return linkedImportedGlobal
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

	return context.ContractValueHandler(context, contract.Location)
}
