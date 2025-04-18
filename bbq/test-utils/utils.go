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

package test_utils

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/stdlib"
	. "github.com/onflow/cadence/test_utils/sema_utils"

	"github.com/onflow/cadence/bbq"
	"github.com/onflow/cadence/bbq/commons"
	"github.com/onflow/cadence/bbq/compiler"
)

func SingleIdentifierLocationResolver(t testing.TB) func(
	identifiers []ast.Identifier,
	location common.Location,
) ([]commons.ResolvedLocation, error) {
	return func(identifiers []ast.Identifier, location common.Location) ([]commons.ResolvedLocation, error) {
		require.Len(t, identifiers, 1)
		require.IsType(t, common.AddressLocation{}, location)

		return []commons.ResolvedLocation{
			{
				Location: common.AddressLocation{
					Address: location.(common.AddressLocation).Address,
					Name:    identifiers[0].Identifier,
				},
				Identifiers: identifiers,
			},
		}, nil
	}
}

func PrintProgram(name string, program *bbq.InstructionProgram) { //nolint:unused
	const resolve = true
	const colorize = true
	printer := bbq.NewInstructionsProgramPrinter(resolve, colorize)
	fmt.Println("===================", name, "===================")
	fmt.Println(printer.PrintProgram(program))
}

func TestBaseValueActivation(common.Location) *sema.VariableActivation {
	// Only need to make the checker happy
	activation := sema.NewVariableActivation(sema.BaseValueActivation)
	activation.DeclareValue(stdlib.PanicFunction)
	activation.DeclareValue(stdlib.AssertFunction)
	activation.DeclareValue(stdlib.NewStandardLibraryStaticFunction(
		"getAccount",
		stdlib.GetAccountFunctionType,
		"",
		nil,
	))
	return activation
}

type CompiledPrograms map[common.Location]*CompiledProgram
type CompiledProgram struct {
	Program *bbq.InstructionProgram
	*sema.Elaboration
}

type ParseCheckAndCompileOptions struct {
	*ParseAndCheckOptions
	CompilerConfig *compiler.Config
}

func ParseCheckAndCompile(
	t testing.TB,
	code string,
	location common.Location,
	programs map[common.Location]*CompiledProgram,
) *bbq.InstructionProgram {
	return ParseCheckAndCompileCodeWithOptions(
		t,
		code,
		location,
		ParseCheckAndCompileOptions{},
		programs,
	)
}

func ParseCheckAndCompileCodeWithOptions(
	t testing.TB,
	code string,
	location common.Location,
	options ParseCheckAndCompileOptions,
	programs CompiledPrograms,
) *bbq.InstructionProgram {
	checker := parseAndCheckWithOptions(
		t,
		code,
		location,
		options.ParseAndCheckOptions,
		programs,
	)
	programs[location] = &CompiledProgram{
		Elaboration: checker.Elaboration,
	}

	program := compile(
		t,
		options.CompilerConfig,
		checker,
		programs,
	)
	programs[location].Program = program

	return program
}

func parseAndCheck( // nolint:unused
	t testing.TB,
	code string,
	location common.Location,
	programs map[common.Location]*CompiledProgram,
) *sema.Checker {
	return parseAndCheckWithOptions(t, code, location, nil, programs)
}

func parseAndCheckWithOptions(
	t testing.TB,
	code string,
	location common.Location,
	options *ParseAndCheckOptions,
	programs map[common.Location]*CompiledProgram,
) *sema.Checker {

	var parseAndCheckOptions ParseAndCheckOptions
	if options != nil {
		parseAndCheckOptions = *options
	} else {
		parseAndCheckOptions = ParseAndCheckOptions{
			Location: location,
			Config: &sema.Config{
				LocationHandler:            SingleIdentifierLocationResolver(t),
				BaseValueActivationHandler: TestBaseValueActivation,
			},
		}
	}

	parseAndCheckOptions.Location = location

	if parseAndCheckOptions.Config.ImportHandler == nil {
		parseAndCheckOptions.Config.ImportHandler = func(_ *sema.Checker, location common.Location, _ ast.Range) (sema.Import, error) {
			imported, ok := programs[location]
			if !ok {
				return nil, fmt.Errorf("cannot find contract in location %s", location)
			}

			return sema.ElaborationImport{
				Elaboration: imported.Elaboration,
			}, nil
		}
	}

	checker, err := ParseAndCheckWithOptions(
		t,
		code,
		parseAndCheckOptions,
	)
	require.NoError(t, err)
	return checker
}

func compile(
	t testing.TB,
	config *compiler.Config,
	checker *sema.Checker,
	programs map[common.Location]*CompiledProgram,
) *bbq.InstructionProgram {

	if config == nil {
		config = &compiler.Config{
			LocationHandler: SingleIdentifierLocationResolver(t),
			ImportHandler: func(location common.Location) *bbq.InstructionProgram {
				imported, ok := programs[location]
				if !ok {
					return nil
				}
				return imported.Program
			},
			ElaborationResolver: func(location common.Location) (*sema.Elaboration, error) {
				imported, ok := programs[location]
				if !ok {
					return nil, fmt.Errorf("cannot find elaboration for %s", location)
				}
				return imported.Elaboration, nil
			},
		}
	}
	comp := compiler.NewInstructionCompilerWithConfig(checker, config)

	program := comp.Compile()
	return program
}
