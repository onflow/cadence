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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/activations"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/pretty"
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/stdlib"
	"github.com/onflow/cadence/test_utils/sema_utils"

	"github.com/onflow/cadence/bbq/vm"
	compilerUtils "github.com/onflow/cadence/bbq/vm/test"
)

type ParseCheckAndInterpretOptions struct {
	Config             *interpreter.Config
	CheckerConfig      *sema.Config
	HandleCheckerError func(error)
}

type Invokable interface {
	interpreter.ValueComparisonContext
	Invoke(functionName string, arguments ...interpreter.Value) (value interpreter.Value, err error)
}

type VMInvokable struct {
	vmInstance *vm.VM
	*vm.Config
}

var _ Invokable = &VMInvokable{}

func (v *VMInvokable) Invoke(functionName string, arguments ...interpreter.Value) (value interpreter.Value, err error) {
	return v.vmInstance.Invoke(functionName, arguments...)
}

func ParseCheckAndPrepare(t testing.TB, code string, compile bool) Invokable {
	t.Helper()

	if !compile {
		return parseCheckAndInterpret(t, code)
	}

	vmConfig := &vm.Config{}
	vmInstance := compilerUtils.CompileAndPrepareToInvoke(
		t,
		code,
		compilerUtils.CompilerAndVMOptions{
			VMConfig: vmConfig,
		},
	)

	return &VMInvokable{
		vmInstance: vmInstance,
		Config:     vmConfig,
	}

}

// Below helper functions were copied as-is from `misc_test.go`.
// Idea is to eventually use the below functions everywhere, and remove them from `misc_test.go`,
// so that the `misc_test.go` would contain only tests.

func parseCheckAndInterpret(t testing.TB, code string) *interpreter.Interpreter {
	inter, err := parseCheckAndInterpretWithOptions(t, code, ParseCheckAndInterpretOptions{})
	require.NoError(t, err)
	return inter
}

func parseCheckAndInterpretWithOptions(
	t testing.TB,
	code string,
	options ParseCheckAndInterpretOptions,
) (
	inter *interpreter.Interpreter,
	err error,
) {
	return parseCheckAndInterpretWithOptionsAndMemoryMetering(t, code, options, nil)
}

func parseCheckAndInterpretWithAtreeValidationsDisabled( // nolint:unused
	t testing.TB,
	code string,
	options ParseCheckAndInterpretOptions,
) (
	inter *interpreter.Interpreter,
	err error,
) {
	return parseCheckAndInterpretWithOptionsAndMemoryMeteringAndAtreeValidations(
		t,
		code,
		options,
		nil,
		false,
	)
}

func parseCheckAndInterpretWithLogs( // nolint:unused
	tb testing.TB,
	code string,
) (
	inter *interpreter.Interpreter,
	getLogs func() []string,
	err error,
) {
	var logs []string

	logFunction := stdlib.NewStandardLibraryStaticFunction(
		"log",
		&sema.FunctionType{
			Parameters: []sema.Parameter{
				{
					Label:          sema.ArgumentLabelNotRequired,
					Identifier:     "value",
					TypeAnnotation: sema.NewTypeAnnotation(sema.AnyStructType),
				},
			},
			ReturnTypeAnnotation: sema.NewTypeAnnotation(
				sema.VoidType,
			),
		},
		``,
		func(invocation interpreter.Invocation) interpreter.Value {
			message := invocation.Arguments[0].MeteredString(
				invocation.InvocationContext,
				interpreter.SeenReferences{},
				invocation.LocationRange,
			)
			logs = append(logs, message)
			return interpreter.Void
		},
	)

	baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
	baseValueActivation.DeclareValue(logFunction)

	baseActivation := activations.NewActivation(nil, interpreter.BaseActivation)
	interpreter.Declare(baseActivation, logFunction)

	result, err := parseCheckAndInterpretWithOptions(
		tb,
		code,
		ParseCheckAndInterpretOptions{
			Config: &interpreter.Config{
				BaseActivationHandler: func(_ common.Location) *interpreter.VariableActivation {
					return baseActivation
				},
			},
			CheckerConfig: &sema.Config{
				BaseValueActivationHandler: func(_ common.Location) *sema.VariableActivation {
					return baseValueActivation
				},
			},
			HandleCheckerError: nil,
		},
	)

	getLogs = func() []string {
		return logs
	}

	return result, getLogs, err
}

func parseCheckAndInterpretWithMemoryMetering( // nolint:unused
	t testing.TB,
	code string,
	memoryGauge common.MemoryGauge,
) *interpreter.Interpreter {

	baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
	baseValueActivation.DeclareValue(stdlib.PanicFunction)

	inter, err := parseCheckAndInterpretWithOptionsAndMemoryMetering(
		t,
		code,
		ParseCheckAndInterpretOptions{
			CheckerConfig: &sema.Config{
				BaseValueActivationHandler: func(_ common.Location) *sema.VariableActivation {
					return baseValueActivation
				},
			},
		},
		memoryGauge,
	)
	require.NoError(t, err)
	return inter
}

func parseCheckAndInterpretWithOptionsAndMemoryMetering(
	t testing.TB,
	code string,
	options ParseCheckAndInterpretOptions,
	memoryGauge common.MemoryGauge,
) (
	inter *interpreter.Interpreter,
	err error,
) {

	// Atree validation should be disabled for memory metering tests.
	// Otherwise, validation may also affect the memory consumption.
	enableAtreeValidations := memoryGauge == nil

	return parseCheckAndInterpretWithOptionsAndMemoryMeteringAndAtreeValidations(
		t,
		code,
		options,
		memoryGauge,
		enableAtreeValidations,
	)
}

func parseCheckAndInterpretWithOptionsAndMemoryMeteringAndAtreeValidations(
	t testing.TB,
	code string,
	options ParseCheckAndInterpretOptions,
	memoryGauge common.MemoryGauge,
	enableAtreeValidations bool,
) (
	inter *interpreter.Interpreter,
	err error,
) {

	checker, err := sema_utils.ParseAndCheckWithOptionsAndMemoryMetering(t,
		code,
		sema_utils.ParseAndCheckOptions{
			Config: options.CheckerConfig,
		},
		memoryGauge,
	)

	if options.HandleCheckerError != nil {
		options.HandleCheckerError(err)
	} else if !assert.NoError(t, err) {
		var sb strings.Builder
		location := checker.Location
		printErr := pretty.NewErrorPrettyPrinter(&sb, true).
			PrettyPrintError(err, location, map[common.Location][]byte{location: []byte(code)})
		if printErr != nil {
			panic(printErr)
		}
		assert.Fail(t, sb.String())
		return nil, err
	}

	var uuid uint64 = 0

	var config interpreter.Config
	if options.Config != nil {
		config = *options.Config
	}

	if enableAtreeValidations {
		config.AtreeValueValidationEnabled = true
		config.AtreeStorageValidationEnabled = true
	} else {
		config.AtreeValueValidationEnabled = false
		config.AtreeStorageValidationEnabled = false
	}

	if config.UUIDHandler == nil {
		config.UUIDHandler = func() (uint64, error) {
			uuid++
			return uuid, nil
		}
	}
	if config.Storage == nil {
		config.Storage = interpreter.NewInMemoryStorage(memoryGauge)
	}

	if memoryGauge != nil && config.MemoryGauge == nil {
		config.MemoryGauge = memoryGauge
	}

	inter, err = interpreter.NewInterpreter(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
		&config,
	)

	require.NoError(t, err)

	err = inter.Interpret()

	if err == nil {

		// recover internal panics and return them as an error
		defer inter.RecoverErrors(func(internalErr error) {
			err = internalErr
		})

		// Contract declarations are evaluated lazily,
		// so force the contract value handler to be called

		for _, compositeDeclaration := range checker.Program.CompositeDeclarations() {
			if compositeDeclaration.CompositeKind != common.CompositeKindContract {
				continue
			}

			contractVariable := inter.Globals.Get(compositeDeclaration.Identifier.Identifier)

			_ = contractVariable.GetValue(inter)
		}
	}

	return inter, err
}

type testEvent struct { // nolint:unused
	event     *interpreter.CompositeValue
	eventType *sema.CompositeType
}

func parseCheckAndInterpretWithEvents(t *testing.T, code string) ( // nolint:unused
	inter *interpreter.Interpreter,
	getEvents func() []testEvent,
	err error,
) {
	var events []testEvent

	inter, err = parseCheckAndInterpretWithOptions(t,
		code,
		ParseCheckAndInterpretOptions{
			Config: &interpreter.Config{
				OnEventEmitted: func(
					_ *interpreter.Interpreter,
					_ interpreter.LocationRange,
					event *interpreter.CompositeValue,
					eventType *sema.CompositeType,
				) error {
					events = append(events, testEvent{
						event:     event,
						eventType: eventType,
					})
					return nil
				},
			},
		},
	)
	if err != nil {
		return nil, nil, err
	}

	getEvents = func() []testEvent {
		return events
	}
	return inter, getEvents, nil
}
