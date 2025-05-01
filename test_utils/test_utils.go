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
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/pretty"
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/stdlib"
	. "github.com/onflow/cadence/test_utils/common_utils"
	"github.com/onflow/cadence/test_utils/sema_utils"
)

type ParseCheckAndInterpretOptions struct {
	Config             *interpreter.Config
	CheckerConfig      *sema.Config
	HandleCheckerError func(error)
}

//type VMInvokable struct {
//	vmInstance *vm.VM
//	*vm.Context
//	elaboration *compiler.DesugaredElaboration
//}
//
//var _ Invokable = &VMInvokable{}
//
//func NewVMInvokable(
//	vmInstance *vm.VM,
//	elaboration *compiler.DesugaredElaboration,
//) *VMInvokable {
//	return &VMInvokable{
//		vmInstance:  vmInstance,
//		Context:     vmInstance.Context(),
//		elaboration: elaboration,
//	}
//}
//
//func (v *VMInvokable) Invoke(functionName string, arguments ...interpreter.Value) (value interpreter.Value, err error) {
//	value, err = v.vmInstance.InvokeExternally(functionName, arguments...)
//
//	// Reset the VM after a function invocation,
//	// so the same vm can be re-used for subsequent invocation.
//	v.vmInstance.Reset()
//
//	return
//}
//
//func (v *VMInvokable) GetGlobal(name string) interpreter.Value {
//	return v.vmInstance.Global(name)
//}
//
//func (v *VMInvokable) GetGlobalType(name string) (*sema.Variable, bool) {
//	return v.elaboration.GetGlobalType(name)
//}
//
//func (v *VMInvokable) InitializeContract(arguments ...interpreter.Value) (*interpreter.CompositeValue, error) {
//	return v.vmInstance.InitializeContract(arguments...)
//}
//func (v *VMInvokable) InitializeContract(contractName string, arguments ...interpreter.Value) (*interpreter.CompositeValue, error) {
//	return v.vmInstance.InitializeContract(contractName, arguments...)
//}

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

	//interpreterConfig := options.Config
	//
	//var storage interpreter.Storage
	//if interpreterConfig != nil {
	//	storage = interpreterConfig.Storage
	//}
	//
	//vmConfig := vm.NewConfig(storage).
	//	WithInterpreterConfig(interpreterConfig).
	//	WithDebugEnabled()
	//
	//var compilerConfig *compiler.Config
	//
	//// If there are builtin functions provided externally (e.g: for tests),
	//// then convert them to corresponding functions in compiler and in vm.
	//if interpreterConfig != nil && interpreterConfig.BaseActivationHandler != nil {
	//	baseActivation := interpreterConfig.BaseActivationHandler(nil)
	//	baseActivationVariables := baseActivation.ValuesInFunction()
	//
	//	vmConfig.BuiltinGlobalsProvider = func() map[string]*vm.Variable {
	//		builtinGlobals := vm.NativeFunctions()
	//
	//		// Add the given built-in values.
	//		// Convert the externally provided `interpreter.HostFunctionValue`s into `vm.NativeFunctionValue`s.
	//		for name, variable := range baseActivationVariables { //nolint:maprange
	//
	//			if builtinGlobals[name] != nil {
	//				continue
	//			}
	//
	//			value := variable.GetValue(nil)
	//
	//			if functionValue, ok := value.(*interpreter.HostFunctionValue); ok {
	//				value = vm.NewNativeFunctionValue(
	//					name,
	//					functionValue.Type,
	//					func(context *vm.Context, _ []interpreter.StaticType, arguments ...vm.Value) vm.Value {
	//						invocation := interpreter.NewInvocation(
	//							context,
	//							nil,
	//							nil,
	//							arguments,
	//							nil,
	//							// TODO: provide these if they are needed for tests.
	//							nil,
	//							interpreter.EmptyLocationRange,
	//						)
	//						return functionValue.Function(invocation)
	//					},
	//				)
	//
	//			}
	//
	//			vmVariable := &vm.Variable{}
	//			vmVariable.InitializeWithValue(value)
	//
	//			builtinGlobals[name] = vmVariable
	//		}
	//
	//		return builtinGlobals
	//	}
	//
	//	// Register externally provided globals in compiler.
	//	compilerConfig = &compiler.Config{
	//		BuiltinGlobalsProvider: func() map[string]*compiler.Global {
	//			globals := compiler.NativeFunctions()
	//			for name := range baseActivationVariables { //nolint:maprange
	//				if globals[name] != nil {
	//					continue
	//				}
	//				globals[name] = &compiler.Global{
	//					Name: name,
	//				}
	//			}
	//
	//			return globals
	//		},
	//	}
	//}
	//
	//parseAndCheckOptions := &sema_utils.ParseAndCheckOptions{
	//	Config: options.CheckerConfig,
	//}
	//
	//programs := map[common.Location]*CompiledProgram{}
	//
	//vmInstance := compilerUtils.CompileAndPrepareToInvoke(
	//	tb,
	//	code,
	//	compilerUtils.CompilerAndVMOptions{
	//		VMConfig: vmConfig,
	//		ParseCheckAndCompileOptions: ParseCheckAndCompileOptions{
	//			ParseAndCheckOptions: parseAndCheckOptions,
	//			CompilerConfig:       compilerConfig,
	//			CheckerErrorHandler:  options.HandleCheckerError,
	//		},
	//		Programs: programs,
	//	},
	//)
	//
	//elaboration := programs[parseAndCheckOptions.Location].DesugaredElaboration
	//
	//return NewVMInvokable(vmInstance, elaboration), nil

	// Unsupported for now
	panic(errors.NewUnreachableError())
}

func ParseCheckAndPrepareWithEvents(tb testing.TB, code string, compile bool) (
	invokable Invokable,
	getEvents func() []TestEvent,
	err error,
) {
	tb.Helper()

	if !compile {
		return ParseCheckAndInterpretWithEvents(tb, code)
	}

	var events []TestEvent
	getEvents = func() []TestEvent {
		return events
	}

	vmConfig := &vm.Config{}

	vmInstance := compilerUtils.CompileAndPrepareToInvoke(
		tb,
		code,
		compilerUtils.CompilerAndVMOptions{
			VMConfig: vmConfig,
		},
	)

	vmConfig.OnEventEmitted = func(eventValues []interpreter.Value, eventType *interpreter.CompositeStaticType) error {
		eventSemaType, eventValue := eventTypeAndValue(
			vmInstance.Context(),
			eventType,
			eventValues,
		)

		events = append(events, TestEvent{
			Event:     eventValue,
			EventType: eventSemaType,
		})
		return nil
	}

	return NewVMInvokable(vmInstance), getEvents, nil
}

func eventTypeAndValue(
	context *vm.Context,
	eventType *interpreter.CompositeStaticType,
	eventValues []interpreter.Value,
) (*sema.CompositeType, *interpreter.CompositeValue) {
	eventSemaType := interpreter.MustConvertStaticToSemaType(eventType, context).(*sema.CompositeType)

	var fields []interpreter.CompositeField
	for index, value := range eventValues {
		fields = append(fields, interpreter.CompositeField{
			Value: value,
			Name:  eventSemaType.Fields[index],
		})
	}

	event := interpreter.NewCompositeValue(
		context,
		interpreter.EmptyLocationRange,
		eventType.Location,
		eventType.QualifiedIdentifier,
		eventSemaType.Kind,
		fields,
		common.ZeroAddress,
	)
	return eventSemaType, event
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

	vmConfig := &vm.Config{}

	vmConfig.WithDebugEnabled()

	var parseAndCheckOptions *sema_utils.ParseAndCheckOptions
	if options.CheckerConfig != nil {
		parseAndCheckOptions = &sema_utils.ParseAndCheckOptions{
			Config: options.CheckerConfig,
		}
	}

	vmInstance := compilerUtils.CompileAndPrepareToInvoke(
		tb,
		code,
		compilerUtils.CompilerAndVMOptions{
			VMConfig: vmConfig,
			ParseCheckAndCompileOptions: ParseCheckAndCompileOptions{
				ParseAndCheckOptions: parseAndCheckOptions,
			},
		},
	)

	vmConfig.OnEventEmitted = func(eventValues []interpreter.Value, eventType *interpreter.CompositeStaticType) error {
		eventSemaType, eventValue := eventTypeAndValue(
			vmInstance.Context(),
			eventType,
			eventValues,
		)

		return options.Config.OnEventEmitted(
			nil,
			interpreter.EmptyLocationRange,
			eventValue,
			eventSemaType,
		)
	}

	return NewVMInvokable(vmInstance), nil
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

func ParseCheckAndInterpretWithLogs(
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

	result, err := ParseCheckAndInterpretWithOptions(
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

func ParseCheckAndInterpretWithMemoryMetering(
	t testing.TB,
	code string,
	memoryGauge common.MemoryGauge,
) *interpreter.Interpreter {

	baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
	baseValueActivation.DeclareValue(stdlib.PanicFunction)

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

func ParseCheckAndInterpretWithEvents(tb testing.TB, code string) (
	inter *interpreter.Interpreter,
	getEvents func() []TestEvent,
	err error,
) {
	var events []TestEvent

	inter, err = ParseCheckAndInterpretWithOptions(tb,
		code,
		ParseCheckAndInterpretOptions{
			Config: &interpreter.Config{
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
			},
		},
	)
	if err != nil {
		return nil, nil, err
	}

	getEvents = func() []TestEvent {
		return events
	}
	return inter, getEvents, nil
}
