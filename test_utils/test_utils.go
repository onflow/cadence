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
	. "github.com/onflow/cadence/test_utils/common_utils"
	"github.com/onflow/cadence/test_utils/sema_utils"

	"github.com/onflow/cadence/bbq/compiler"
	. "github.com/onflow/cadence/bbq/test_utils"
	"github.com/onflow/cadence/bbq/vm"
	compilerUtils "github.com/onflow/cadence/bbq/vm/test"
)

type ParseCheckAndInterpretOptions struct {
	Config             *interpreter.Config
	CheckerConfig      *sema.Config
	HandleCheckerError func(error)
}

type VMInvokable struct {
	vmInstance *vm.VM
	*vm.Context
	elaboration *compiler.DesugaredElaboration
}

var _ Invokable = &VMInvokable{}

func NewVMInvokable(
	vmInstance *vm.VM,
	elaboration *compiler.DesugaredElaboration,
) *VMInvokable {
	return &VMInvokable{
		vmInstance:  vmInstance,
		Context:     vmInstance.Context(),
		elaboration: elaboration,
	}
}

func (v *VMInvokable) Invoke(functionName string, arguments ...interpreter.Value) (value interpreter.Value, err error) {
	value, err = v.vmInstance.InvokeExternally(functionName, arguments...)

	// Reset the VM after a function invocation,
	// so the same vm can be re-used for subsequent invocation.
	v.vmInstance.Reset()

	return
}

func (v *VMInvokable) InvokeTransaction(arguments []interpreter.Value, signers ...interpreter.Value) (err error) {
	err = v.vmInstance.InvokeTransaction(arguments, signers...)

	// Reset the VM after a function invocation,
	// so the same vm can be re-used for subsequent invocation.
	v.vmInstance.Reset()

	return
}

func (v *VMInvokable) GetGlobal(name string) interpreter.Value {
	return v.vmInstance.Global(name)
}

func (v *VMInvokable) GetGlobalType(name string) (*sema.Variable, bool) {
	return v.elaboration.GetGlobalType(name)
}

func (v *VMInvokable) InitializeContract(contractName string, arguments ...interpreter.Value) (*interpreter.CompositeValue, error) {
	return v.vmInstance.InitializeContract(contractName, arguments...)
}

func ParseCheckAndPrepare(tb testing.TB, code string, compile bool) Invokable {
	tb.Helper()

	invokable, err := ParseCheckAndPrepareWithOptions(tb, code, ParseCheckAndInterpretOptions{}, compile)
	require.NoError(tb, err)

	return invokable
}

func ParseCheckAndPrepareWithEvents(tb testing.TB, code string, compile bool) (
	invokable Invokable,
	getEvents func() []TestEvent,
	err error,
) {
	tb.Helper()

	var events []TestEvent
	getEvents = func() []TestEvent {
		return events
	}

	interpreterConfig := &interpreter.Config{
		OnEventEmitted: func(
			_ interpreter.ValueExportContext,
			_ interpreter.LocationRange,
			eventType *sema.CompositeType,
			eventFields []interpreter.Value,
		) error {
			events = append(
				events,
				TestEvent{
					EventType:   eventType,
					EventFields: eventFields,
				},
			)
			return nil
		},
	}

	parseCheckAndInterpretOptions := ParseCheckAndInterpretOptions{
		Config: interpreterConfig,
	}

	if !compile {
		invokable, err = ParseCheckAndInterpretWithOptions(
			tb,
			code,
			parseCheckAndInterpretOptions,
		)
		return invokable, getEvents, err
	}

	invokable, err = ParseCheckAndPrepareWithOptions(tb, code, parseCheckAndInterpretOptions, compile)
	require.NoError(tb, err)

	return invokable, getEvents, err
}

func ParseCheckAndPrepareWithLogs(
	tb testing.TB,
	code string,
	compile bool,
) (
	invokable Invokable,
	getLogs func() []string,
	err error,
) {

	var logs []string

	valueDeclaration := stdlib.NewInterpreterStandardLibraryStaticFunction(
		"log",
		stdlib.LogFunctionType,
		"",
		func(invocation interpreter.Invocation) interpreter.Value {
			value := invocation.Arguments[0]
			logs = append(logs, value.String())
			return interpreter.Void
		},
	)

	baseValueActivation := sema.NewVariableActivation(nil)
	baseValueActivation.DeclareValue(valueDeclaration)

	baseActivation := activations.NewActivation(nil, interpreter.BaseActivation)
	interpreter.Declare(baseActivation, valueDeclaration)

	invokable, err = ParseCheckAndPrepareWithOptions(tb,
		code,
		ParseCheckAndInterpretOptions{
			CheckerConfig: &sema.Config{
				BaseValueActivationHandler: func(_ common.Location) *sema.VariableActivation {
					return baseValueActivation
				},
			},
			Config: &interpreter.Config{
				BaseActivationHandler: func(common.Location) *interpreter.VariableActivation {
					return baseActivation
				},
			},
		},
		compile,
	)

	getLogs = func() []string {
		return logs
	}

	return invokable, getLogs, err
}

func ParseCheckAndPrepareWithOptions(
	tb testing.TB,
	code string,
	options ParseCheckAndInterpretOptions,
	compile bool,
) (
	invokable Invokable,
	err error,
) {
	tb.Helper()

	if !compile {
		return ParseCheckAndInterpretWithOptions(tb, code, options)
	}

	interpreterConfig := options.Config

	var storage interpreter.Storage
	if interpreterConfig != nil {
		storage = interpreterConfig.Storage
	}

	programs := CompiledPrograms{}
	var compilerConfig *compiler.Config

	vmConfig := vm.NewConfig(storage).
		WithDebugEnabled()

	typeLoader := compilerUtils.CompiledProgramsTypeLoader(programs)

	if options.CheckerConfig != nil {
		typeActivationHandler := options.CheckerConfig.BaseTypeActivationHandler
		if typeActivationHandler != nil {
			vmConfig.TypeLoader = func(location common.Location, typeID interpreter.TypeID) sema.ContainedType {
				activation := typeActivationHandler(location)
				typeName := location.QualifiedIdentifier(typeID)
				variable := activation.Find(typeName)
				if variable != nil {
					return variable.Type.(sema.ContainedType)
				}

				return typeLoader(location, typeID)
			}
		}
	}

	if interpreterConfig != nil {
		vmConfig.MemoryGauge = interpreterConfig.MemoryGauge
		vmConfig.ComputationGauge = interpreterConfig.ComputationGauge
		vmConfig.CapabilityCheckHandler = interpreterConfig.CapabilityCheckHandler
		vmConfig.CapabilityBorrowHandler = interpreterConfig.CapabilityBorrowHandler
		vmConfig.ValidateAccountCapabilitiesGetHandler = interpreterConfig.ValidateAccountCapabilitiesGetHandler
		vmConfig.ValidateAccountCapabilitiesPublishHandler = interpreterConfig.ValidateAccountCapabilitiesPublishHandler
		vmConfig.OnEventEmitted = interpreterConfig.OnEventEmitted
		vmConfig.AccountHandlerFunc = interpreterConfig.AccountHandler
		vmConfig.InjectedCompositeFieldsHandler = interpreterConfig.InjectedCompositeFieldsHandler
		vmConfig.UUIDHandler = interpreterConfig.UUIDHandler
		vmConfig.AtreeValueValidationEnabled = interpreterConfig.AtreeValueValidationEnabled
		vmConfig.AtreeStorageValidationEnabled = interpreterConfig.AtreeStorageValidationEnabled

		// If there are builtin functions provided externally (e.g: for tests),
		// then convert them to corresponding functions in compiler and in vm.
		if interpreterConfig.BaseActivationHandler != nil {
			interpreterBaseActivation := interpreterConfig.BaseActivationHandler(nil)

			// Only iterate through values defined at the current-level.
			// Do not get the values defined in the parent activation/scope.
			// (i.e: only get the values that were added externally for tests)
			interpreterBaseActivationVariables := interpreterBaseActivation.ValuesInCurrentLevel()

			vmConfig.BuiltinGlobalsProvider = func() *activations.Activation[vm.Variable] {

				activation := activations.NewActivation(nil, vm.DefaultBuiltinGlobals())

				panicVariable := interpreter.NewVariableWithValue(
					nil,
					stdlib.VMPanicFunction.Value,
				)
				activation.Set(stdlib.PanicFunctionName, panicVariable)

				// Add the given built-in values.
				// Convert the externally provided `interpreter.HostFunctionValue`s into `vm.NativeFunctionValue`s.
				for name, variable := range interpreterBaseActivationVariables { //nolint:maprange

					value := variable.GetValue(nil)

					if functionValue, ok := value.(*interpreter.HostFunctionValue); ok {
						value = vm.NewNativeFunctionValue(
							name,
							functionValue.Type,
							func(context *vm.Context, _ []interpreter.StaticType, arguments ...vm.Value) vm.Value {
								invocation := interpreter.NewInvocation(
									context,
									nil,
									nil,
									arguments,
									nil,
									// TODO: provide these if they are needed for tests.
									nil,
									interpreter.EmptyLocationRange,
								)
								return functionValue.Function(invocation)
							},
						)

					}

					vmVariable := interpreter.NewVariableWithValue(
						nil,
						value,
					)

					activation.Set(name, vmVariable)
				}

				return activation
			}

			// Register externally provided globals in compiler.
			compilerConfig = &compiler.Config{
				BuiltinGlobalsProvider: func() *activations.Activation[compiler.GlobalImport] {
					baseActivation := compiler.DefaultBuiltinGlobals()
					activation := activations.NewActivation(nil, baseActivation)
					for name := range interpreterBaseActivationVariables { //nolint:maprange
						existing := activation.Find(name)
						if existing != (compiler.GlobalImport{}) {
							continue
						}
						activation.Set(
							name,
							compiler.GlobalImport{
								Name: name,
							},
						)
					}
					return activation
				},
			}
		}

		if interpreterConfig.ImportLocationHandler != nil {
			vmConfig.TypeLoader = func(location common.Location, typeID interpreter.TypeID) sema.ContainedType {
				impt := interpreterConfig.ImportLocationHandler(nil, location)
				switch impt := impt.(type) {
				case interpreter.VirtualImport:
					return impt.Elaboration.CompositeType(typeID)
				case interpreter.InterpreterImport:
					return impt.Interpreter.Program.Elaboration.CompositeType(typeID)
				}

				return typeLoader(location, typeID)
			}
		}
	}

	if vmConfig.TypeLoader == nil {
		vmConfig.TypeLoader = typeLoader
	}

	if vmConfig.BuiltinGlobalsProvider == nil {
		vmConfig.BuiltinGlobalsProvider = func() *activations.Activation[vm.Variable] {
			activation := activations.NewActivation(nil, vm.DefaultBuiltinGlobals())

			panicVariable := interpreter.NewVariableWithValue(
				nil,
				stdlib.VMPanicFunction.Value,
			)
			activation.Set(stdlib.PanicFunctionName, panicVariable)

			return activation
		}
	}

	parseAndCheckOptions := &sema_utils.ParseAndCheckOptions{
		Config: options.CheckerConfig,
	}

	vmInstance := compilerUtils.CompileAndPrepareToInvoke(
		tb,
		code,
		compilerUtils.CompilerAndVMOptions{
			VMConfig: vmConfig,
			ParseCheckAndCompileOptions: ParseCheckAndCompileOptions{
				ParseAndCheckOptions: parseAndCheckOptions,
				CompilerConfig:       compilerConfig,
				CheckerErrorHandler:  options.HandleCheckerError,
			},
			Programs: programs,
		},
	)

	elaboration := programs[parseAndCheckOptions.Location].DesugaredElaboration

	return NewVMInvokable(vmInstance, elaboration), nil
}

func ParseCheckAndPrepareWithAtreeValidationsDisabled(
	tb testing.TB,
	code string,
	options ParseCheckAndInterpretOptions,
	compile bool,
) (Invokable, error) {
	tb.Helper()

	if !compile {
		return ParseCheckAndInterpretWithAtreeValidationsDisabled(tb, code, options)
	}

	interpreterConfig := options.Config
	if interpreterConfig == nil {
		interpreterConfig = &interpreter.Config{}
		options.Config = interpreterConfig
	}

	interpreterConfig.AtreeStorageValidationEnabled = false
	interpreterConfig.AtreeValueValidationEnabled = false

	invokable, err := ParseCheckAndPrepareWithOptions(tb, code, options, compile)
	return invokable, err
}

// Below helper functions were copied as-is from `misc_test.go`.
// Idea is to eventually use the below functions everywhere, and remove them from `misc_test.go`,
// so that the `misc_test.go` would contain only tests.

func ParseCheckAndInterpret(t testing.TB, code string) *interpreter.Interpreter {
	inter, err := ParseCheckAndInterpretWithOptions(t, code, ParseCheckAndInterpretOptions{})
	require.NoError(t, err)
	return inter
}

func ParseCheckAndInterpretWithOptions(
	t testing.TB,
	code string,
	options ParseCheckAndInterpretOptions,
) (
	inter *interpreter.Interpreter,
	err error,
) {
	return ParseCheckAndInterpretWithOptionsAndMemoryMetering(t, code, options, nil)
}

func ParseCheckAndInterpretWithAtreeValidationsDisabled(
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

func ParseCheckAndInterpretWithMemoryMetering(
	t testing.TB,
	code string,
	memoryGauge common.MemoryGauge,
) *interpreter.Interpreter {

	baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
	baseValueActivation.DeclareValue(stdlib.InterpreterPanicFunction)

	inter, err := ParseCheckAndInterpretWithOptionsAndMemoryMetering(
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

func ParseCheckAndInterpretWithOptionsAndMemoryMetering(
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

type TestEvent struct {
	EventType   *sema.CompositeType
	EventFields []interpreter.Value
}
