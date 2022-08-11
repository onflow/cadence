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
 *
 * Based on https://github.com/wk8/go-ordered-map, Copyright Jean Rougé
 *
 */

package test

import (
	emulator "github.com/onflow/flow-emulator"

	"github.com/onflow/cadence/encoding/json"
	"github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/stdlib"
	"github.com/onflow/cadence/runtime/tests/utils"
)

var _ interpreter.TestFramework = &EmulatorBackend{}

// EmulatorBackend is the emulator-backed implementation of the interpreter.TestFramework.
//
type EmulatorBackend struct {
	blockchain *emulator.Blockchain
}

func NewEmulatorBackend() *EmulatorBackend {
	return &EmulatorBackend{
		blockchain: newBlockchain(),
	}
}

func (e *EmulatorBackend) RunScript(code string, args []interpreter.Value) interpreter.ScriptResult {
	// TODO: maybe re-use interpreter? Only needed for value conversion
	// TODO: Deal with imported/composite types
	inter, err := newInterpreter()
	if err != nil {
		return interpreter.ScriptResult{
			Error: err,
		}
	}

	arguments := make([][]byte, 0, len(args))
	for _, arg := range args {
		exportedValue, err := runtime.ExportValue(arg, inter, interpreter.ReturnEmptyLocationRange)
		if err != nil {
			return interpreter.ScriptResult{
				Error: err,
			}
		}

		encodedArg, err := json.Encode(exportedValue)
		if err != nil {
			return interpreter.ScriptResult{
				Error: err,
			}
		}

		arguments = append(arguments, encodedArg)
	}

	result, err := e.blockchain.ExecuteScript([]byte(code), arguments)
	if err != nil {
		return interpreter.ScriptResult{
			Error: err,
		}
	}

	if result.Error != nil {
		return interpreter.ScriptResult{
			Error: result.Error,
		}
	}

	value, err := runtime.ImportValue(inter, interpreter.ReturnEmptyLocationRange, result.Value, nil)
	if err != nil {
		return interpreter.ScriptResult{
			Error: err,
		}
	}

	return interpreter.ScriptResult{
		Value: value,
	}
}

// newBlockchain returns an emulator blockchain for testing.
func newBlockchain(opts ...emulator.Option) *emulator.Blockchain {
	b, err := emulator.NewBlockchain(
		append(
			[]emulator.Option{
				emulator.WithStorageLimitEnabled(false),
			},
			opts...,
		)...,
	)
	if err != nil {
		panic(err)
	}

	return b
}

// newInterpreter creates an interpreter instance needed for the value conversion.
//
func newInterpreter() (*interpreter.Interpreter, error) {
	predeclaredInterpreterValues := stdlib.BuiltinFunctions.ToInterpreterValueDeclarations()
	predeclaredInterpreterValues = append(predeclaredInterpreterValues, stdlib.BuiltinValues.ToInterpreterValueDeclarations()...)

	return interpreter.NewInterpreter(
		nil,
		utils.TestLocation,
		interpreter.WithStorage(interpreter.NewInMemoryStorage(nil)),
		interpreter.WithPredeclaredValues(predeclaredInterpreterValues),
		interpreter.WithImportLocationHandler(func(inter *interpreter.Interpreter, location common.Location) interpreter.Import {
			switch location {
			case stdlib.CryptoChecker.Location:
				program := interpreter.ProgramFromChecker(stdlib.CryptoChecker)
				subInterpreter, err := inter.NewSubInterpreter(program, location)
				if err != nil {
					panic(err)
				}
				return interpreter.InterpreterImport{
					Interpreter: subInterpreter,
				}

			case stdlib.TestContractLocation:
				program := interpreter.ProgramFromChecker(stdlib.TestContractChecker)
				subInterpreter, err := inter.NewSubInterpreter(program, location)
				if err != nil {
					panic(err)
				}
				return interpreter.InterpreterImport{
					Interpreter: subInterpreter,
				}

			default:
				panic(errors.NewUnexpectedError("importing of programs not implemented"))
			}
		}),
	)
}
