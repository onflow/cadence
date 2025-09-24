//go:build cadence_tracing

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

package interpreter_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"

	"github.com/onflow/cadence/activations"
	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/bbq"
	"github.com/onflow/cadence/bbq/compiler"
	. "github.com/onflow/cadence/bbq/test_utils"
	"github.com/onflow/cadence/bbq/vm"
	"github.com/onflow/cadence/bbq/vm/test"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/stdlib"
	"github.com/onflow/cadence/test_utils"
	. "github.com/onflow/cadence/test_utils/common_utils"
	. "github.com/onflow/cadence/test_utils/sema_utils"
)

func prepareWithTracingCallBack(
	t *testing.T,
	tracingCallback func(opName string),
) Invokable {
	storage := newUnmeteredInMemoryStorage()

	onRecordTrace := func(
		operationName string,
		_ time.Duration,
		_ []attribute.KeyValue,
	) {
		tracingCallback(operationName)
	}

	if *compile {
		config := vm.NewConfig(storage)
		config.Tracer = interpreter.CallbackTracer(onRecordTrace)
		config.CompositeTypeHandler = func(location common.Location, typeID interpreter.TypeID) *sema.CompositeType {
			if typeID == testCompositeValueType.ID() {
				return testCompositeValueType
			}
			t.Fatalf("unexpected type ID: %s", typeID)
			return nil
		}
		config.ImportHandler = func(_ common.Location) *bbq.InstructionProgram {
			return &bbq.InstructionProgram{}
		}
		vm := vm.NewVM(
			TestLocation,
			&bbq.InstructionProgram{},
			config,
		)
		return test_utils.NewVMInvokable(vm, nil)
	} else {
		inter, err := interpreter.NewInterpreter(
			nil,
			TestLocation,
			&interpreter.Config{
				Storage:       storage,
				OnRecordTrace: onRecordTrace,
			},
		)
		require.NoError(t, err)
		return inter
	}
}

func TestInterpreterTracing(t *testing.T) {

	t.Parallel()

	t.Run("array tracing", func(t *testing.T) {
		var traceOps []string
		inter := prepareWithTracingCallBack(t, func(opName string) {
			traceOps = append(traceOps, opName)
		})
		owner := common.Address{0x1}
		array := interpreter.NewArrayValue(
			inter,
			interpreter.EmptyLocationRange,
			&interpreter.VariableSizedStaticType{
				Type: interpreter.PrimitiveStaticTypeAnyStruct,
			},
			owner,
		)
		require.NotNil(t, array)
		require.Equal(t, len(traceOps), 1)
		require.Equal(t, traceOps[0], "array.construct")

		cloned := array.Clone(inter)
		require.NotNil(t, cloned)
		cloned.DeepRemove(inter, true)
		require.Equal(t, len(traceOps), 2)
		require.Equal(t, traceOps[1], "array.deepRemove")

		array.Destroy(inter, interpreter.EmptyLocationRange)
		require.Equal(t, len(traceOps), 3)
		require.Equal(t, traceOps[2], "array.destroy")
	})

	t.Run("dictionary tracing", func(t *testing.T) {
		var traceOps []string
		inter := prepareWithTracingCallBack(t, func(opName string) {
			traceOps = append(traceOps, opName)
		})
		dict := interpreter.NewDictionaryValue(
			inter,
			interpreter.EmptyLocationRange,
			&interpreter.DictionaryStaticType{
				KeyType:   interpreter.PrimitiveStaticTypeString,
				ValueType: interpreter.PrimitiveStaticTypeInt,
			},
			interpreter.NewUnmeteredStringValue("test"), interpreter.NewUnmeteredIntValueFromInt64(42),
		)
		require.NotNil(t, dict)
		require.Equal(t, len(traceOps), 1)
		require.Equal(t, traceOps[0], "dictionary.construct")

		cloned := dict.Clone(inter)
		require.NotNil(t, cloned)
		cloned.DeepRemove(inter, true)
		require.Equal(t, len(traceOps), 2)
		require.Equal(t, traceOps[1], "dictionary.deepRemove")

		dict.Destroy(inter, interpreter.EmptyLocationRange)
		require.Equal(t, len(traceOps), 3)
		require.Equal(t, traceOps[2], "dictionary.destroy")
	})

	t.Run("composite tracing", func(t *testing.T) {
		var traceOps []string
		inter := prepareWithTracingCallBack(t, func(opName string) {
			traceOps = append(traceOps, opName)
		})
		owner := common.Address{0x1}

		value := newTestCompositeValue(inter, owner)

		require.Equal(t, len(traceOps), 1)
		require.Equal(t, traceOps[0], "composite.construct")

		cloned := value.Clone(inter)
		require.NotNil(t, cloned)
		cloned.DeepRemove(inter, true)
		require.Equal(t, len(traceOps), 2)
		require.Equal(t, traceOps[1], "composite.deepRemove")

		value.SetMember(inter, interpreter.EmptyLocationRange, "abc", interpreter.Nil)
		require.Equal(t, len(traceOps), 3)
		require.Equal(t, traceOps[2], "composite.setMember.abc")

		value.GetMember(inter, interpreter.EmptyLocationRange, "abc")
		require.Equal(t, len(traceOps), 4)
		require.Equal(t, traceOps[3], "composite.getMember.abc")

		value.RemoveMember(inter, interpreter.EmptyLocationRange, "abc")
		require.Equal(t, len(traceOps), 5)
		require.Equal(t, traceOps[4], "composite.removeMember.abc")

		value.Destroy(inter, interpreter.EmptyLocationRange)
		require.Equal(t, len(traceOps), 6)
		require.Equal(t, traceOps[5], "composite.destroy")

		array := interpreter.NewArrayValue(
			inter,
			interpreter.EmptyLocationRange,
			&interpreter.VariableSizedStaticType{
				Type: interpreter.PrimitiveStaticTypeAnyStruct,
			},
			common.ZeroAddress,
			cloned,
		)
		require.NotNil(t, array)
		require.Equal(t, len(traceOps), 8)
		require.Equal(t, traceOps[6], "composite.transfer")
		require.Equal(t, traceOps[7], "array.construct")
	})
}

func TestInterpretImportEnums(t *testing.T) {

	t.Parallel()

	const logFunctionName = "log"

	const codeA = `
        enum A: Int8 {
            case A1
            case A2
        }
    `

	const codeB = `
        enum B: Int8 {
            case B1
            case B2
        }
    `

	const codeC = `
        import A from 0x1
        import B from 0x2

        enum C: Int8 {
            case C1
            case C2
        }

        fun test() {}
    `

	var traces []string

	onRecordTrace := func(
		operationName string,
		_ time.Duration,
		_ []attribute.KeyValue,
	) {
		traces = append(traces, operationName)
	}

	if *compile {
		valueDeclaration := stdlib.NewVMStandardLibraryStaticFunction(
			logFunctionName,
			stdlib.LogFunctionType,
			"",
			func(_ *vm.Context, _ []bbq.StaticType, _ vm.Value, arguments ...vm.Value) vm.Value {
				value := arguments[0]
				traces = append(traces, value.String())
				return interpreter.Void
			},
		)

		baseValueActivation := sema.NewVariableActivation(nil)
		baseValueActivation.DeclareValue(valueDeclaration)

		programs := CompiledPrograms{}

		builtinGlobalsProvider := func(_ common.Location) *activations.Activation[compiler.GlobalImport] {
			activation := activations.NewActivation(nil, compiler.DefaultBuiltinGlobals())

			activation.Set(
				logFunctionName,
				compiler.NewGlobalImport(logFunctionName),
			)

			return activation
		}

		addressA := common.MustBytesToAddress([]byte{0x1})
		addressB := common.MustBytesToAddress([]byte{0x2})

		_ = ParseCheckAndCompileCodeWithOptions(t,
			codeA,
			common.NewAddressLocation(nil, addressA, ""),
			ParseCheckAndCompileOptions{
				ParseAndCheckOptions: &ParseAndCheckOptions{
					CheckerConfig: &sema.Config{
						BaseValueActivationHandler: func(_ common.Location) *sema.VariableActivation {
							return baseValueActivation
						},
					},
				},
				CompilerConfig: &compiler.Config{
					BuiltinGlobalsProvider: builtinGlobalsProvider,
				},
			},
			programs,
		)

		_ = ParseCheckAndCompileCodeWithOptions(t,
			codeB,
			common.NewAddressLocation(nil, addressB, ""),
			ParseCheckAndCompileOptions{
				ParseAndCheckOptions: &ParseAndCheckOptions{
					CheckerConfig: &sema.Config{
						BaseValueActivationHandler: func(_ common.Location) *sema.VariableActivation {
							return baseValueActivation
						},
					},
				},
				CompilerConfig: &compiler.Config{
					BuiltinGlobalsProvider: builtinGlobalsProvider,
					ImportHandler: func(location common.Location) *bbq.InstructionProgram {
						return programs[location].Program
					},
					LocationHandler: func(identifiers []ast.Identifier, location common.Location) ([]sema.ResolvedLocation, error) {
						return []sema.ResolvedLocation{
							{
								Location:    location,
								Identifiers: identifiers,
							},
						}, nil
					},
				},
			},
			programs,
		)

		vmConfig := test.PrepareVMConfig(t, nil, programs)

		vmConfig.BuiltinGlobalsProvider = func(location common.Location) *activations.Activation[vm.Variable] {
			activation := activations.NewActivation(nil, vm.DefaultBuiltinGlobals())

			logVariable := &interpreter.SimpleVariable{}
			logVariable.InitializeWithValue(valueDeclaration.Value)
			activation.Set(
				logFunctionName,
				logVariable,
			)

			return activation
		}

		vmConfig.Tracer = interpreter.CallbackTracer(onRecordTrace)

		_, err := test.CompileAndInvokeWithOptionsAndPrograms(t,
			codeC,
			"test",
			test.CompilerAndVMOptions{
				VMConfig: vmConfig,
				ParseCheckAndCompileOptions: ParseCheckAndCompileOptions{
					ParseAndCheckOptions: &ParseAndCheckOptions{
						CheckerConfig: &sema.Config{
							BaseValueActivationHandler: func(_ common.Location) *sema.VariableActivation {
								return baseValueActivation
							},
							ImportHandler: func(checker *sema.Checker, importedLocation common.Location, _ ast.Range) (sema.Import, error) {
								importedProgram, ok := programs[importedLocation]
								if !ok {
									return nil, fmt.Errorf("cannot find program for location %s", importedLocation)
								}

								return sema.ElaborationImport{
									Elaboration: importedProgram.DesugaredElaboration.OriginalElaboration(),
								}, nil
							},
						},
					},
					CompilerConfig: &compiler.Config{
						BuiltinGlobalsProvider: builtinGlobalsProvider,
						ImportHandler: func(location common.Location) *bbq.InstructionProgram {
							return programs[location].Program
						},
						LocationHandler: func(identifiers []ast.Identifier, location common.Location) ([]sema.ResolvedLocation, error) {
							return []sema.ResolvedLocation{
								{
									Location:    location,
									Identifiers: identifiers,
								},
							}, nil
						},
					},
				},
			},
			programs,
		)
		require.NoError(t, err)

	} else {

		valueDeclaration := stdlib.NewInterpreterStandardLibraryStaticFunction(
			logFunctionName,
			stdlib.LogFunctionType,
			"",
			func(invocation interpreter.Invocation) interpreter.Value {
				value := invocation.Arguments[0]
				traces = append(traces, value.String())
				return interpreter.Void
			},
		)

		baseValueActivation := sema.NewVariableActivation(nil)
		baseValueActivation.DeclareValue(valueDeclaration)

		baseActivation := activations.NewActivation(nil, interpreter.BaseActivation)
		interpreter.Declare(baseActivation, valueDeclaration)

		// Program A
		addressA := common.MustBytesToAddress([]byte{0x1})

		importedCheckerA, err := ParseAndCheckWithOptions(t,
			codeA,
			ParseAndCheckOptions{
				Location: common.AddressLocation{
					Address: addressA,
					Name:    "",
				},
				CheckerConfig: &sema.Config{
					BaseValueActivationHandler: func(_ common.Location) *sema.VariableActivation {
						return baseValueActivation
					},
				},
			},
		)
		require.NoError(t, err)

		// Program B

		addressB := common.MustBytesToAddress([]byte{0x2})

		importedCheckerB, err := ParseAndCheckWithOptions(t,
			codeB,
			ParseAndCheckOptions{
				Location: common.AddressLocation{
					Address: addressB,
					Name:    "",
				},
				CheckerConfig: &sema.Config{
					BaseValueActivationHandler: func(_ common.Location) *sema.VariableActivation {
						return baseValueActivation
					},
					ImportHandler: func(checker *sema.Checker, importedLocation common.Location, _ ast.Range) (sema.Import, error) {
						addressLocation := importedLocation.(common.AddressLocation)

						var importedElaboration *sema.Elaboration

						switch addressLocation.Address {
						case addressA:
							importedElaboration = importedCheckerA.Elaboration
						default:
							panic(fmt.Errorf("unknown address: %s", addressLocation.Address))
						}

						return sema.ElaborationImport{
							Elaboration: importedElaboration,
						}, nil
					},
				},
			},
		)
		require.NoError(t, err)

		// Program C

		inter, err := parseCheckAndInterpretWithOptions(t,
			codeC,
			ParseCheckAndInterpretOptions{
				ParseAndCheckOptions: &ParseAndCheckOptions{
					CheckerConfig: &sema.Config{
						BaseValueActivationHandler: func(_ common.Location) *sema.VariableActivation {
							return baseValueActivation
						},
						ImportHandler: func(checker *sema.Checker, importedLocation common.Location, _ ast.Range) (sema.Import, error) {
							addressLocation := importedLocation.(common.AddressLocation)

							var importedElaboration *sema.Elaboration

							switch addressLocation.Address {
							case addressA:
								importedElaboration = importedCheckerA.Elaboration
							case addressB:
								importedElaboration = importedCheckerB.Elaboration
							default:
								panic(fmt.Errorf("unknown address: %s", addressLocation.Address))
							}

							return sema.ElaborationImport{
								Elaboration: importedElaboration,
							}, nil
						},
					},
				},
				InterpreterConfig: &interpreter.Config{
					BaseActivationHandler: func(common.Location) *interpreter.VariableActivation {
						return baseActivation
					},

					ImportLocationHandler: func(inter *interpreter.Interpreter, location common.Location) interpreter.Import {
						addressLocation := location.(common.AddressLocation)

						var importedChecker *sema.Checker

						switch addressLocation.Address {
						case addressA:
							importedChecker = importedCheckerA
						case addressB:
							importedChecker = importedCheckerB
						default:
							panic(fmt.Errorf("unknown address: %s", addressLocation.Address))
						}

						program := interpreter.ProgramFromChecker(importedChecker)
						subInterpreter, err := inter.NewSubInterpreter(program, location)
						if err != nil {
							panic(err)
						}

						return interpreter.InterpreterImport{
							Interpreter: subInterpreter,
						}
					},

					OnRecordTrace: onRecordTrace,
				},
			},
		)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		require.NoError(t, err)
	}

	require.Equal(
		t,
		[]string{
			"atreeMap.new",
			"composite.setMember",
			"composite.construct",
			"atreeMap.new",
			"composite.setMember",
			"composite.construct",
			"import",
			"atreeMap.new",
			"composite.setMember",
			"composite.construct",
			"atreeMap.new",
			"composite.setMember",
			"composite.construct",
			"import",
			"atreeMap.new",
			"composite.setMember",
			"composite.construct",
			"atreeMap.new",
			"composite.setMember",
			"composite.construct",
		},
		traces,
	)
}
