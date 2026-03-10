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

package test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/activations"
	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/bbq"
	"github.com/onflow/cadence/bbq/compiler"
	"github.com/onflow/cadence/bbq/opcode"
	. "github.com/onflow/cadence/bbq/test_utils"
	"github.com/onflow/cadence/bbq/vm"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/stdlib"
	. "github.com/onflow/cadence/test_utils/common_utils"
	. "github.com/onflow/cadence/test_utils/interpreter_utils"
	. "github.com/onflow/cadence/test_utils/runtime_utils"
	. "github.com/onflow/cadence/test_utils/sema_utils"
)

const recursiveFib = `
  fun fib(_ n: Int): Int {
      if n < 2 {
         return n
      }
      return fib(n - 1) + fib(n - 2)
  }
`

func scriptLocation() common.Location {
	scriptLocation := NewScriptLocationGenerator()
	return scriptLocation()
}

func TestRecursionFib(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, recursiveFib)
	require.NoError(t, err)

	comp := compiler.NewInstructionCompiler(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
	)
	program := comp.Compile()

	vmConfig := vm.NewConfig(NewUnmeteredInMemoryStorage())
	vmInstance := vm.NewVM(scriptLocation(), program, vmConfig)

	result, err := vmInstance.InvokeExternally(
		"fib",
		interpreter.NewUnmeteredIntValueFromInt64(23),
	)
	require.NoError(t, err)
	require.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(28657), result)
	require.Equal(t, 0, vmInstance.StackSize())
}

const imperativeFib = `
  fun fib(_ n: Int): Int {
      var fib1 = 1
      var fib2 = 1
      var fibonacci = fib1
      var i = 2
      while i < n {
          fibonacci = fib1 + fib2
          fib1 = fib2
          fib2 = fibonacci
          i = i + 1
      }
      return fibonacci
  }
`

func TestImperativeFib(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, imperativeFib)
	require.NoError(t, err)

	comp := compiler.NewInstructionCompiler(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
	)
	program := comp.Compile()

	vmConfig := vm.NewConfig(NewUnmeteredInMemoryStorage())
	vmInstance := vm.NewVM(scriptLocation(), program, vmConfig)

	result, err := vmInstance.InvokeExternally(
		"fib",
		interpreter.NewUnmeteredIntValueFromInt64(7),
	)
	require.NoError(t, err)
	require.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(13), result)
	require.Equal(t, 0, vmInstance.StackSize())
}

func TestVoid(t *testing.T) {

	t.Parallel()

	result, err := CompileAndInvoke(t,
		`
          fun test() {
              return ()
          }
        `,
		"test",
	)
	require.NoError(t, err)
	require.Equal(t, interpreter.Void, result)
}

func TestTrue(t *testing.T) {

	t.Parallel()

	result, err := CompileAndInvoke(t,
		`
          fun test(): Bool {
              return true
          }
        `,
		"test",
	)
	require.NoError(t, err)
	require.Equal(t, interpreter.TrueValue, result)
}

func TestFalse(t *testing.T) {

	t.Parallel()

	result, err := CompileAndInvoke(t,
		`
          fun test(): Bool {
              return false
          }
        `,
		"test",
	)
	require.NoError(t, err)
	require.Equal(t, interpreter.FalseValue, result)
}

func TestNil(t *testing.T) {

	t.Parallel()

	result, err := CompileAndInvoke(t,
		`
          fun test(): Bool? {
              return nil
          }
        `,
		"test",
	)
	require.NoError(t, err)
	require.Equal(t, interpreter.Nil, result)
}

func TestWhileBreak(t *testing.T) {

	t.Parallel()

	result, err := CompileAndInvoke(t,
		`
          fun test(): Int {
              var i = 0
              while true {
                  if i > 3 {
                     break
                  }
                  i = i + 1
              }
              return i
          }
        `,
		"test",
	)
	require.NoError(t, err)
	require.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(4), result)
}

func TestSwitchBreak(t *testing.T) {

	t.Parallel()

	test := func(t *testing.T, value int64) vm.Value {
		result, err := CompileAndInvoke(t,
			`
              fun test(x: Int): Int {
                  switch x {
                      case 1:
                          break
                      default:
                          return 3
                  }
                  return 1
              }
            `,
			"test",
			interpreter.NewUnmeteredIntValueFromInt64(value),
		)
		require.NoError(t, err)
		return result
	}

	t.Run("1", func(t *testing.T) {
		t.Parallel()

		result := test(t, 1)
		require.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(1), result)
	})

	t.Run("2", func(t *testing.T) {
		t.Parallel()

		result := test(t, 2)
		require.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(3), result)
	})

	t.Run("3", func(t *testing.T) {
		t.Parallel()

		result := test(t, 3)
		require.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(3), result)
	})
}

func TestWhileSwitchBreak(t *testing.T) {

	t.Parallel()

	test := func(t *testing.T, value int64) vm.Value {
		result, err := CompileAndInvoke(t,
			`
                fun test(x: Int): Int {
                  while true {
                      switch x {
                          case 1:
                              break
                          default:
                              return 3
                      }
                      return 1
                  }
                  return 2
              }
            `,
			"test",
			interpreter.NewUnmeteredIntValueFromInt64(value),
		)
		require.NoError(t, err)
		return result
	}

	t.Run("1", func(t *testing.T) {
		t.Parallel()

		result := test(t, 1)
		require.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(1), result)
	})

	t.Run("2", func(t *testing.T) {
		t.Parallel()

		result := test(t, 2)
		require.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(3), result)
	})

	t.Run("3", func(t *testing.T) {
		t.Parallel()

		result := test(t, 3)
		require.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(3), result)
	})
}

func TestContinue(t *testing.T) {

	t.Parallel()

	result, err := CompileAndInvoke(t,
		`
          fun test(): Int {
              var i = 0
              while true {
                  i = i + 1
                  if i < 3 {
                     continue
                  }
                  break
              }
              return i
          }
        `,
		"test",
	)
	require.NoError(t, err)

	require.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(3), result)
}

func TestNilCoalesce(t *testing.T) {

	t.Parallel()

	test := func(t *testing.T, argument vm.Value) vm.Value {
		actual, err := CompileAndInvoke(t,
			`
                fun test(i: Int?): Int {
                    var j = i ?? 3
                    return j
                }
            `,
			"test",
			argument,
		)
		require.NoError(t, err)
		return actual
	}

	t.Run("non-nil", func(t *testing.T) {
		t.Parallel()

		actual := test(t, interpreter.NewUnmeteredSomeValueNonCopying(
			interpreter.NewUnmeteredIntValueFromInt64(2),
		))
		require.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(2), actual)
	})

	t.Run("nil", func(t *testing.T) {
		t.Parallel()

		actual := test(t, interpreter.Nil)
		require.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(3), actual)
	})
}

func TestNewStruct(t *testing.T) {

	t.Parallel()

	vmConfig := vm.NewConfig(NewUnmeteredInMemoryStorage())

	vmInstance, err := CompileAndPrepareToInvoke(t,
		`
          struct Foo {
              var id : Int

              init(_ id: Int) {
                  self.id = id
              }
          }

          fun test(count: Int): Foo {
              var i = 0
              var r = Foo(0)
              while i < count {
                  i = i + 1
                  r = Foo(i)
                  r.id = r.id + 2
              }
              return r
          }
        `,
		CompilerAndVMOptions{
			VMConfig: vmConfig,
		},
	)
	require.NoError(t, err)

	result, err := vmInstance.InvokeExternally("test", interpreter.NewUnmeteredIntValueFromInt64(10))
	require.NoError(t, err)
	require.Equal(t, 0, vmInstance.StackSize())

	vmContext := vmInstance.Context()

	require.IsType(t, &interpreter.CompositeValue{}, result)
	structValue := result.(*interpreter.CompositeValue)
	compositeType := structValue.StaticType(vmContext).(*interpreter.CompositeStaticType)

	require.Equal(t, "Foo", compositeType.QualifiedIdentifier)
	require.Equal(
		t,
		interpreter.NewUnmeteredIntValueFromInt64(12),
		structValue.GetMember(vmContext, "id"),
	)
}

func TestStructMethodCall(t *testing.T) {

	t.Parallel()

	t.Run("method", func(t *testing.T) {

		t.Parallel()

		result, err := CompileAndInvoke(t,
			`
              struct Foo {
                  var id : String

                  init(_ id: String) {
                      self.id = id
                  }

                  fun sayHello(_ id: Int): String {
                      return self.id
                  }
              }

              fun test(): String {
                  var r = Foo("Hello from Foo!")
                  return r.sayHello(1)
              }
            `,
			"test",
		)
		require.NoError(t, err)
		require.Equal(t, interpreter.NewUnmeteredStringValue("Hello from Foo!"), result)
	})

	t.Run("function typed field", func(t *testing.T) {

		t.Parallel()

		result, err := CompileAndInvoke(t,
			`
              struct Foo {
                  var sayHello: fun(): String

                  init(_ str: String) {
                      self.sayHello = fun(): String {
                          return str
                      }
                  }
              }

              fun test(): String {
                  var r = Foo("Hello from Foo!")
                  return r.sayHello()
              }
            `,
			"test",
		)
		require.NoError(t, err)
		require.Equal(t, interpreter.NewUnmeteredStringValue("Hello from Foo!"), result)
	})
}

func TestImport(t *testing.T) {

	t.Parallel()

	importedLocation := common.AddressLocation{
		Address: common.MustBytesToAddress([]byte{0x1}),
		Name:    "Foo",
	}

	importedChecker, err := ParseAndCheckWithOptions(t,
		`
          fun helloText(): String {
              return "global function of the imported program"
          }

          struct Foo {
              var id : String

              init(_ id: String) {
                  self.id = id
              }

              fun sayHello(_ id: Int): String {
                  self.id
                  return helloText()
              }
          }
        `,
		ParseAndCheckOptions{
			Location: importedLocation,
		},
	)
	require.NoError(t, err)

	subComp := compiler.NewInstructionCompiler(
		interpreter.ProgramFromChecker(importedChecker),
		importedChecker.Location,
	)
	importedProgram := subComp.Compile()

	checker, err := ParseAndCheckWithOptions(t,
		`
          import Foo from 0x01

          fun test(): String {
              var r = Foo("Hello from Foo!")
              return r.sayHello(1)
          }
        `,
		ParseAndCheckOptions{
			CheckerConfig: &sema.Config{
				LocationHandler: SingleIdentifierLocationResolver(t),
				ImportHandler: func(_ *sema.Checker, location common.Location, _ ast.Range) (sema.Import, error) {
					require.Equal(t, importedChecker.Location, location)
					return sema.ElaborationImport{
						Elaboration: importedChecker.Elaboration,
					}, nil
				},
			},
		},
	)
	require.NoError(t, err)

	importCompiler := compiler.NewInstructionCompiler(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
	)
	importCompiler.Config.ImportHandler = func(location common.Location) *bbq.InstructionProgram {
		require.Equal(t, importedChecker.Location, location)
		return importedProgram
	}

	program := importCompiler.Compile()

	vmConfig := vm.NewConfig(NewUnmeteredInMemoryStorage())
	vmConfig.ImportHandler = func(location common.Location) *bbq.InstructionProgram {
		require.Equal(t, importedChecker.Location, location)
		return importedProgram
	}
	vmConfig.CompositeTypeHandler = func(location common.Location, typeID interpreter.TypeID) *sema.CompositeType {
		require.Equal(t, importedChecker.Location, location)
		elaboration := importedChecker.Elaboration
		return elaboration.CompositeType(typeID)
	}
	vmConfig.InterfaceTypeHandler = func(location common.Location, typeID interpreter.TypeID) *sema.InterfaceType {
		require.Equal(t, importedChecker.Location, location)
		elaboration := importedChecker.Elaboration
		return elaboration.InterfaceType(typeID)
	}

	vmInstance := vm.NewVM(scriptLocation(), program, vmConfig)

	result, err := vmInstance.InvokeExternally("test")
	require.NoError(t, err)
	require.Equal(t, 0, vmInstance.StackSize())

	require.Equal(t, interpreter.NewUnmeteredStringValue("global function of the imported program"), result)
}

func TestImportError(t *testing.T) {

	t.Parallel()

	importLocation := common.AddressLocation{
		Address: common.MustBytesToAddress([]byte{0x1}),
		Name:    "Test",
	}

	importedChecker, err := ParseAndCheckWithOptions(t,
		`
          fun helloText(): String {
              return panic("error in helloText")
          }

          contract Test {
              struct Foo {
                  var id : String

                  init(_ id: String) {
                      self.id = id
                  }

                  fun sayHello(_ id: Int): String {
                      self.id
                      return helloText()
                  }
              }
          }
        `,
		ParseAndCheckOptions{
			Location: importLocation,
			CheckerConfig: &sema.Config{
				BaseValueActivationHandler: TestBaseValueActivation,
			},
		},
	)
	require.NoError(t, err)

	importCompConfig := &compiler.Config{
		BuiltinGlobalsProvider: CompilerDefaultBuiltinGlobalsWithDefaultsAndPanic,
	}

	importCompiler := compiler.NewInstructionCompilerWithConfig(
		interpreter.ProgramFromChecker(importedChecker),
		importedChecker.Location,
		importCompConfig,
	)
	importedProgram := importCompiler.Compile()

	importVMConfig := vm.NewConfig(NewUnmeteredInMemoryStorage())
	importVMConfig.BuiltinGlobalsProvider = VMBuiltinGlobalsProviderWithDefaultsAndPanic

	_, importedContractValue := initializeContract(
		t,
		importLocation,
		importedProgram,
		importVMConfig,
	)

	activation := sema.NewVariableActivation(sema.BaseValueActivation)
	activation.DeclareValue(stdlib.VMPanicFunction)

	checker, err := ParseAndCheckWithOptions(t,
		`
          import Test from 0x1

          fun test(): String {
              var r = Test.Foo("Hello from Foo!")
              return r.sayHello(1)
          }
        `,
		ParseAndCheckOptions{
			CheckerConfig: &sema.Config{
				LocationHandler: SingleIdentifierLocationResolver(t),
				ImportHandler: func(_ *sema.Checker, location common.Location, _ ast.Range) (sema.Import, error) {
					require.Equal(t, importedChecker.Location, location)
					return sema.ElaborationImport{
						Elaboration: importedChecker.Elaboration,
					}, nil
				},
				BaseValueActivationHandler: func(location common.Location) *sema.VariableActivation {
					return activation
				},
			},
		},
	)
	require.NoError(t, err)

	compConfig := &compiler.Config{
		LocationHandler: SingleIdentifierLocationResolver(t),
		ImportHandler: func(location common.Location) *bbq.InstructionProgram {
			require.Equal(t, importLocation, location)
			return importedProgram
		},
		BuiltinGlobalsProvider: CompilerDefaultBuiltinGlobalsWithDefaultsAndPanic,
	}

	comp := compiler.NewInstructionCompilerWithConfig(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
		compConfig,
	)

	program := comp.Compile()

	vmConfig := vm.NewConfig(NewUnmeteredInMemoryStorage())
	vmConfig.ImportHandler = func(location common.Location) *bbq.InstructionProgram {
		require.Equal(t, importLocation, location)
		return importedProgram
	}
	vmConfig.ContractValueHandler = func(_ *vm.Context, location common.Location) *interpreter.CompositeValue {
		require.Equal(t, importLocation, location)
		return importedContractValue
	}
	vmConfig.CompositeTypeHandler = func(location common.Location, typeID interpreter.TypeID) *sema.CompositeType {
		require.Equal(t, importLocation, location)
		elaboration := importedChecker.Elaboration
		return elaboration.CompositeType(typeID)
	}
	vmConfig.InterfaceTypeHandler = func(location common.Location, typeID interpreter.TypeID) *sema.InterfaceType {
		require.Equal(t, importLocation, location)
		elaboration := importedChecker.Elaboration
		return elaboration.InterfaceType(typeID)
	}
	vmConfig.BuiltinGlobalsProvider = VMBuiltinGlobalsProviderWithDefaultsAndPanic

	vmInstance := vm.NewVM(scriptLocation(), program, vmConfig)

	_, err = vmInstance.InvokeExternally("test")
	RequireError(t, err)

	var panicErr *stdlib.PanicError
	require.ErrorAs(t, err, &panicErr)

	assert.Equal(t,
		"error in helloText",
		panicErr.Message,
	)
}

func TestContractImport(t *testing.T) {

	t.Parallel()

	t.Run("nested type def", func(t *testing.T) {

		t.Parallel()

		importLocation := common.AddressLocation{
			Address: common.MustBytesToAddress([]byte{0x1}),
			Name:    "MyContract",
		}

		importedChecker, err := ParseAndCheckWithOptions(t,
			`
              contract MyContract {

                  fun helloText(): String {
                      return "global function of the imported program"
                  }

                  init() {}

                  struct Foo {
                      var id : String

                      init(_ id: String) {
                          self.id = id
                      }

                      fun sayHello(_ id: Int): String {
                          self.id
                          return MyContract.helloText()
                      }
                  }
              }
            `,
			ParseAndCheckOptions{
				Location: importLocation,
			},
		)
		require.NoError(t, err)

		importCompiler := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(importedChecker),
			importedChecker.Location,
		)
		importedProgram := importCompiler.Compile()

		_, importedContractValue := initializeContract(
			t,
			importLocation,
			importedProgram,
			nil,
		)

		checker, err := ParseAndCheckWithOptions(t,
			`
              import MyContract from 0x01

              fun test(): String {
                  var r = MyContract.Foo("Hello from Foo!")
                  return r.sayHello(1)
              }
            `,
			ParseAndCheckOptions{
				CheckerConfig: &sema.Config{
					LocationHandler: SingleIdentifierLocationResolver(t),
					ImportHandler: func(_ *sema.Checker, location common.Location, _ ast.Range) (sema.Import, error) {
						require.Equal(t, importedChecker.Location, location)
						return sema.ElaborationImport{
							Elaboration: importedChecker.Elaboration,
						}, nil
					},
				},
			},
		)
		require.NoError(t, err)

		comp := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(checker),
			checker.Location,
		)
		comp.Config.ImportHandler = func(location common.Location) *bbq.InstructionProgram {
			require.Equal(t, importedChecker.Location, location)
			return importedProgram
		}

		program := comp.Compile()

		vmConfig := vm.NewConfig(NewUnmeteredInMemoryStorage())
		vmConfig.ImportHandler = func(location common.Location) *bbq.InstructionProgram {
			require.Equal(t, importedChecker.Location, location)
			return importedProgram
		}
		vmConfig.ContractValueHandler = func(_ *vm.Context, location common.Location) *interpreter.CompositeValue {
			require.Equal(t, importedChecker.Location, location)
			return importedContractValue
		}
		vmConfig.CompositeTypeHandler = func(location common.Location, typeID interpreter.TypeID) *sema.CompositeType {
			require.Equal(t, importedChecker.Location, location)
			elaboration := importedChecker.Elaboration
			return elaboration.CompositeType(typeID)
		}
		vmConfig.InterfaceTypeHandler = func(location common.Location, typeID interpreter.TypeID) *sema.InterfaceType {
			require.Equal(t, importedChecker.Location, location)
			elaboration := importedChecker.Elaboration
			return elaboration.InterfaceType(typeID)
		}

		vmInstance := vm.NewVM(scriptLocation(), program, vmConfig)

		result, err := vmInstance.InvokeExternally("test")
		require.NoError(t, err)
		require.Equal(t, 0, vmInstance.StackSize())

		require.Equal(t, interpreter.NewUnmeteredStringValue("global function of the imported program"), result)
	})

	t.Run("contract function", func(t *testing.T) {

		t.Parallel()

		importLocation := common.AddressLocation{
			Address: common.MustBytesToAddress([]byte{0x1}),
			Name:    "MyContract",
		}
		importedChecker, err := ParseAndCheckWithOptions(t,
			`
              contract MyContract {

                  var s: String

                  fun helloText(): String {
                      return self.s
                  }

                  init() {
                      self.s = "contract function of the imported program"
                  }
              }
            `,
			ParseAndCheckOptions{
				Location: importLocation,
			},
		)
		require.NoError(t, err)

		importCompiler := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(importedChecker),
			importedChecker.Location,
		)
		importedProgram := importCompiler.Compile()

		var importedContractValue *interpreter.CompositeValue

		vmConfig := vm.NewConfig(NewUnmeteredInMemoryStorage())
		vmConfig.ImportHandler = func(location common.Location) *bbq.InstructionProgram {
			require.Equal(t, importedChecker.Location, location)
			return importedProgram
		}
		vmConfig.ContractValueHandler = func(_ *vm.Context, location common.Location) *interpreter.CompositeValue {
			require.Equal(t, importedChecker.Location, location)
			return importedContractValue
		}
		vmConfig.CompositeTypeHandler = func(location common.Location, typeID interpreter.TypeID) *sema.CompositeType {
			require.Equal(t, importedChecker.Location, location)
			elaboration := importedChecker.Elaboration
			return elaboration.CompositeType(typeID)
		}
		vmConfig.InterfaceTypeHandler = func(location common.Location, typeID interpreter.TypeID) *sema.InterfaceType {
			require.Equal(t, importedChecker.Location, location)
			elaboration := importedChecker.Elaboration
			return elaboration.InterfaceType(typeID)
		}

		_, importedContractValue = initializeContract(
			t,
			importLocation,
			importedProgram,
			vmConfig,
		)

		checker, err := ParseAndCheckWithOptions(t,
			`
              import MyContract from 0x01

              fun test(): String {
                  return MyContract.helloText()
              }
            `,
			ParseAndCheckOptions{
				CheckerConfig: &sema.Config{
					LocationHandler: SingleIdentifierLocationResolver(t),
					ImportHandler: func(_ *sema.Checker, location common.Location, _ ast.Range) (sema.Import, error) {
						require.Equal(t, importedChecker.Location, location)
						return sema.ElaborationImport{
							Elaboration: importedChecker.Elaboration,
						}, nil
					},
				},
			},
		)
		require.NoError(t, err)

		comp := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(checker),
			checker.Location,
		)
		comp.Config.ImportHandler = func(location common.Location) *bbq.InstructionProgram {
			require.Equal(t, importedChecker.Location, location)
			return importedProgram
		}

		program := comp.Compile()

		vmInstance := vm.NewVM(scriptLocation(), program, vmConfig)

		result, err := vmInstance.InvokeExternally("test")
		require.NoError(t, err)
		require.Equal(t, 0, vmInstance.StackSize())

		require.Equal(t, interpreter.NewUnmeteredStringValue("contract function of the imported program"), result)
	})

	t.Run("nested imports", func(t *testing.T) {

		t.Parallel()

		// Initialize Foo

		fooLocation := common.NewAddressLocation(
			nil,
			common.MustBytesToAddress([]byte{0x1}),
			"Foo",
		)

		fooChecker, err := ParseAndCheckWithOptions(t,
			`
              contract Foo {
                  var s: String
                  init() {
                      self.s = "Hello from Foo!"
                  }
                  fun sayHello(): String {
                      return self.s
                  }
              }
            `,
			ParseAndCheckOptions{
				Location: fooLocation,
			},
		)
		require.NoError(t, err)

		fooCompiler := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(fooChecker),
			fooChecker.Location,
		)
		fooProgram := fooCompiler.Compile()

		var fooContractValue *interpreter.CompositeValue

		vmConfig := vm.NewConfig(NewUnmeteredInMemoryStorage())
		vmConfig.ImportHandler = func(location common.Location) *bbq.InstructionProgram {
			require.Equal(t, fooLocation, location)
			return fooProgram
		}
		vmConfig.ContractValueHandler = func(_ *vm.Context, location common.Location) *interpreter.CompositeValue {
			require.Equal(t, fooLocation, location)
			return fooContractValue
		}
		vmConfig.CompositeTypeHandler = func(location common.Location, typeID interpreter.TypeID) *sema.CompositeType {
			require.Equal(t, fooLocation, location)

			elaboration := fooChecker.Elaboration
			return elaboration.CompositeType(typeID)
		}
		vmConfig.InterfaceTypeHandler = func(location common.Location, typeID interpreter.TypeID) *sema.InterfaceType {
			require.Equal(t, fooLocation, location)

			elaboration := fooChecker.Elaboration
			return elaboration.InterfaceType(typeID)
		}

		_, fooContractValue = initializeContract(
			t,
			fooLocation,
			fooProgram,
			vmConfig,
		)

		// Initialize Bar

		barLocation := common.NewAddressLocation(
			nil,
			common.MustBytesToAddress([]byte{0x2}),
			"Bar",
		)

		barChecker, err := ParseAndCheckWithOptions(t,
			`
              import Foo from 0x01

              contract Bar {
                  init() {}
                  fun sayHello(): String {
                      return Foo.sayHello()
                  }
              }
            `,
			ParseAndCheckOptions{
				Location: barLocation,
				CheckerConfig: &sema.Config{
					LocationHandler: SingleIdentifierLocationResolver(t),
					ImportHandler: func(_ *sema.Checker, location common.Location, _ ast.Range) (sema.Import, error) {
						require.Equal(t, fooLocation, location)
						return sema.ElaborationImport{
							Elaboration: fooChecker.Elaboration,
						}, nil
					},
				},
			},
		)
		require.NoError(t, err)

		barCompiler := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(barChecker),
			barChecker.Location,
		)
		barCompiler.Config.LocationHandler = SingleIdentifierLocationResolver(t)
		barCompiler.Config.ImportHandler = func(location common.Location) *bbq.InstructionProgram {
			require.Equal(t, fooLocation, location)
			return fooProgram
		}

		barProgram := barCompiler.Compile()

		_, barContractValue := initializeContract(
			t,
			barLocation,
			barProgram,
			vmConfig,
		)

		// Compile and run main program

		checker, err := ParseAndCheckWithOptions(t,
			`
              import Bar from 0x02

              fun test(): String {
                  return Bar.sayHello()
              }
            `,
			ParseAndCheckOptions{
				CheckerConfig: &sema.Config{
					LocationHandler: SingleIdentifierLocationResolver(t),
					ImportHandler: func(_ *sema.Checker, location common.Location, _ ast.Range) (sema.Import, error) {
						require.IsType(t, common.AddressLocation{}, location)
						addressLocation := location.(common.AddressLocation)
						var elaboration *sema.Elaboration
						switch addressLocation.Address {
						case fooLocation.Address:
							elaboration = fooChecker.Elaboration
						case barLocation.Address:
							elaboration = barChecker.Elaboration
						default:
							assert.FailNow(t, "invalid location")
						}

						return sema.ElaborationImport{
							Elaboration: elaboration,
						}, nil
					},
				},
			},
		)
		require.NoError(t, err)

		comp := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(checker),
			checker.Location,
		)
		comp.Config.LocationHandler = SingleIdentifierLocationResolver(t)
		comp.Config.ImportHandler = func(location common.Location) *bbq.InstructionProgram {
			switch location {
			case fooLocation:
				return fooProgram
			case barLocation:
				return barProgram
			default:
				assert.FailNow(t, "invalid location")
				return nil
			}
		}

		program := comp.Compile()

		vmConfig = vm.NewConfig(NewUnmeteredInMemoryStorage())
		vmConfig.ImportHandler = func(location common.Location) *bbq.InstructionProgram {
			switch location {
			case fooLocation:
				return fooProgram
			case barLocation:
				return barProgram
			default:
				assert.FailNow(t, "invalid location")
				return nil
			}
		}
		vmConfig.ContractValueHandler = func(_ *vm.Context, location common.Location) *interpreter.CompositeValue {
			switch location {
			case fooLocation:
				return fooContractValue
			case barLocation:
				return barContractValue
			default:
				assert.FailNow(t, fmt.Sprintf("invalid location %s", location))
				return nil
			}
		}
		vmConfig.CompositeTypeHandler = func(location common.Location, typeID interpreter.TypeID) *sema.CompositeType {
			var elaboration *sema.Elaboration

			switch location {
			case fooLocation:
				elaboration = fooChecker.Elaboration
			case barLocation:
				elaboration = barChecker.Elaboration
			default:
				assert.FailNow(t, fmt.Sprintf("invalid location %s", location))
			}

			compositeType := elaboration.CompositeType(typeID)
			return compositeType
		}

		vmConfig.InterfaceTypeHandler = func(location common.Location, typeID interpreter.TypeID) *sema.InterfaceType {
			var elaboration *sema.Elaboration

			switch location {
			case fooLocation:
				elaboration = fooChecker.Elaboration
			case barLocation:
				elaboration = barChecker.Elaboration
			default:
				assert.FailNow(t, fmt.Sprintf("invalid location %s", location))
			}

			interfaceType := elaboration.InterfaceType(typeID)
			return interfaceType
		}

		vmInstance := vm.NewVM(scriptLocation(), program, vmConfig)

		result, err := vmInstance.InvokeExternally("test")
		require.NoError(t, err)
		require.Equal(t, 0, vmInstance.StackSize())

		require.Equal(t, interpreter.NewUnmeteredStringValue("Hello from Foo!"), result)
	})

	t.Run("contract interface", func(t *testing.T) {

		t.Parallel()

		// Initialize Foo

		fooLocation := common.NewAddressLocation(
			nil,
			common.MustBytesToAddress([]byte{0x1}),
			"Foo",
		)

		fooChecker, err := ParseAndCheckWithOptions(t,
			`
              contract interface Foo {
                  fun withdraw(_ amount: Int): String {
                      pre {
                          amount < 100: "Withdraw limit exceeds"
                      }
                  }
              }
            `,
			ParseAndCheckOptions{
				Location: fooLocation,
			},
		)
		require.NoError(t, err)

		fooCompiler := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(fooChecker),
			fooChecker.Location,
		)
		fooProgram := fooCompiler.Compile()

		// Initialize Bar

		barLocation := common.NewAddressLocation(
			nil,
			common.MustBytesToAddress([]byte{0x2}),
			"Bar",
		)

		barChecker, err := ParseAndCheckWithOptions(t,
			`
              import Foo from 0x01

              contract Bar: Foo {
                  init() {}
                  fun withdraw(_ amount: Int): String {
                      return "Successfully withdrew"
                  }
              }
            `,
			ParseAndCheckOptions{
				Location: barLocation,
				CheckerConfig: &sema.Config{
					LocationHandler: SingleIdentifierLocationResolver(t),
					ImportHandler: func(_ *sema.Checker, location common.Location, _ ast.Range) (sema.Import, error) {
						require.Equal(t, fooLocation, location)
						return sema.ElaborationImport{
							Elaboration: fooChecker.Elaboration,
						}, nil
					},
				},
			},
		)
		require.NoError(t, err)

		barCompiler := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(barChecker),
			barChecker.Location,
		)
		barCompiler.Config.LocationHandler = SingleIdentifierLocationResolver(t)
		barCompiler.Config.ImportHandler = func(location common.Location) *bbq.InstructionProgram {
			require.Equal(t, fooLocation, location)
			return fooProgram
		}

		barCompiler.Config.ElaborationResolver = func(location common.Location) (*compiler.DesugaredElaboration, error) {
			switch location {
			case fooLocation:
				return compiler.NewDesugaredElaboration(fooChecker.Elaboration), nil
			case barLocation:
				return compiler.NewDesugaredElaboration(barChecker.Elaboration), nil
			default:
				return nil, fmt.Errorf("cannot find elaboration for %s", location)
			}
		}

		barProgram := barCompiler.Compile()

		vmConfig := vm.NewConfig(NewUnmeteredInMemoryStorage())
		vmConfig.ImportHandler = func(location common.Location) *bbq.InstructionProgram {
			require.Equal(t, fooLocation, location)
			return fooProgram
		}
		vmConfig.BuiltinGlobalsProvider = VMBuiltinGlobalsProviderWithDefaultsAndPanic

		_, barContractValue := initializeContract(
			t,
			barLocation,
			barProgram,
			vmConfig,
		)

		// Compile and run main program

		checker, err := ParseAndCheckWithOptions(t,
			`
              import Bar from 0x02

              fun test(): String {
                  return Bar.withdraw(50)
              }
            `,
			ParseAndCheckOptions{
				CheckerConfig: &sema.Config{
					LocationHandler: SingleIdentifierLocationResolver(t),
					ImportHandler: func(_ *sema.Checker, location common.Location, _ ast.Range) (sema.Import, error) {
						require.IsType(t, common.AddressLocation{}, location)
						addressLocation := location.(common.AddressLocation)
						var elaboration *sema.Elaboration
						switch addressLocation.Address {
						case fooLocation.Address:
							elaboration = fooChecker.Elaboration
						case barLocation.Address:
							elaboration = barChecker.Elaboration
						default:
							assert.FailNow(t, "invalid location")
						}

						return sema.ElaborationImport{
							Elaboration: elaboration,
						}, nil
					},
				},
			},
		)
		require.NoError(t, err)

		comp := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(checker),
			checker.Location,
		)
		comp.Config.LocationHandler = SingleIdentifierLocationResolver(t)
		comp.Config.ImportHandler = func(location common.Location) *bbq.InstructionProgram {
			switch location {
			case fooLocation:
				return fooProgram
			case barLocation:
				return barProgram
			default:
				assert.FailNow(t, "invalid location")
				return nil
			}
		}

		program := comp.Compile()

		vmConfig = vm.NewConfig(NewUnmeteredInMemoryStorage())
		vmConfig.ImportHandler = func(location common.Location) *bbq.InstructionProgram {
			switch location {
			case fooLocation:
				return fooProgram
			case barLocation:
				return barProgram
			default:
				assert.FailNow(t, "invalid location")
				return nil
			}
		}
		vmConfig.ContractValueHandler = func(_ *vm.Context, location common.Location) *interpreter.CompositeValue {
			switch location {
			case barLocation:
				return barContractValue
			default:
				assert.FailNow(t, fmt.Sprintf("invalid location %s", location))
				return nil
			}
		}
		vmConfig.CompositeTypeHandler = func(location common.Location, typeID interpreter.TypeID) *sema.CompositeType {
			var elaboration *sema.Elaboration

			switch location {
			case barLocation:
				elaboration = barChecker.Elaboration
			default:
				assert.FailNow(t, fmt.Sprintf("invalid location %s", location))
			}

			compositeType := elaboration.CompositeType(typeID)
			return compositeType
		}
		vmConfig.InterfaceTypeHandler = func(location common.Location, typeID interpreter.TypeID) *sema.InterfaceType {
			var elaboration *sema.Elaboration

			switch location {
			case barLocation:
				elaboration = barChecker.Elaboration
			default:
				assert.FailNow(t, fmt.Sprintf("invalid location %s", location))
			}

			interfaceType := elaboration.InterfaceType(typeID)
			return interfaceType
		}
		vmConfig.BuiltinGlobalsProvider = VMBuiltinGlobalsProviderWithDefaultsAndPanic

		vmInstance := vm.NewVM(scriptLocation(), program, vmConfig)

		result, err := vmInstance.InvokeExternally("test")
		require.NoError(t, err)
		require.Equal(t, 0, vmInstance.StackSize())

		require.Equal(t, interpreter.NewUnmeteredStringValue("Successfully withdrew"), result)
	})

	t.Run("contract field", func(t *testing.T) {

		t.Parallel()

		importLocation := common.AddressLocation{
			Address: common.MustBytesToAddress([]byte{0x1}),
			Name:    "Foo",
		}
		compilerConfig := &compiler.Config{
			BuiltinGlobalsProvider: CompilerDefaultBuiltinGlobalsWithDefaultsAndConditionLog,
		}
		activation := sema.NewVariableActivation(sema.BaseValueActivation)
		activation.DeclareValue(newConditionLogFunction(nil))

		importedChecker, err := ParseAndCheckWithOptions(t,
			`
              contract Foo {
                  var id: UInt8
                  init() {
                      self.id = 2
                  }

                  fun getIDUsingQualifiedName(): UInt8 {
                      return Foo.id
                  }

                  fun getIDUsingSelf(): UInt8 {
                      return Foo.id
                  }
              }
            `,
			ParseAndCheckOptions{
				Location: importLocation,
				CheckerConfig: &sema.Config{
					BaseValueActivationHandler: func(location common.Location) *sema.VariableActivation {
						return activation
					},
				},
			},
		)
		require.NoError(t, err)

		importCompiler := compiler.NewInstructionCompilerWithConfig(
			interpreter.ProgramFromChecker(importedChecker),
			importedChecker.Location,
			compilerConfig,
		)
		importedProgram := importCompiler.Compile()

		var logs []string
		var importedContractValue *interpreter.CompositeValue

		vmConfig := vm.NewConfig(NewUnmeteredInMemoryStorage())
		vmConfig.ImportHandler = func(location common.Location) *bbq.InstructionProgram {
			require.Equal(t, importLocation, location)
			return importedProgram
		}
		vmConfig.ContractValueHandler = func(_ *vm.Context, location common.Location) *interpreter.CompositeValue {
			require.Equal(t, importLocation, location)
			return importedContractValue
		}
		vmConfig.CompositeTypeHandler = func(location common.Location, typeID interpreter.TypeID) *sema.CompositeType {
			require.Equal(t, importLocation, location)
			elaboration := importedChecker.Elaboration
			return elaboration.CompositeType(typeID)
		}
		vmConfig.InterfaceTypeHandler = func(location common.Location, typeID interpreter.TypeID) *sema.InterfaceType {
			require.Equal(t, importLocation, location)
			elaboration := importedChecker.Elaboration
			return elaboration.InterfaceType(typeID)
		}
		vmConfig.BuiltinGlobalsProvider = NewVMBuiltinGlobalsProviderWithDefaultsPanicAndConditionLog(&logs)

		_, importedContractValue = initializeContract(
			t,
			importLocation,
			importedProgram,
			vmConfig,
		)

		checker, err := ParseAndCheckWithOptions(t,
			`
              import Foo from 0x01

              fun main() {
                  conditionLog(Foo.id)
                  conditionLog(Foo.getIDUsingQualifiedName())
                  conditionLog(Foo.getIDUsingSelf())
              }
            `,
			ParseAndCheckOptions{
				CheckerConfig: &sema.Config{
					LocationHandler: SingleIdentifierLocationResolver(t),
					ImportHandler: func(_ *sema.Checker, location common.Location, _ ast.Range) (sema.Import, error) {
						require.Equal(t, importedChecker.Location, location)
						return sema.ElaborationImport{
							Elaboration: importedChecker.Elaboration,
						}, nil
					},
					BaseValueActivationHandler: func(location common.Location) *sema.VariableActivation {
						return activation
					},
				},
			},
		)
		require.NoError(t, err)

		comp := compiler.NewInstructionCompilerWithConfig(
			interpreter.ProgramFromChecker(checker),
			checker.Location,
			compilerConfig,
		)
		comp.Config.ImportHandler = func(location common.Location) *bbq.InstructionProgram {
			require.Equal(t, importedChecker.Location, location)
			return importedProgram
		}
		program := comp.Compile()

		vmInstance := vm.NewVM(scriptLocation(), program, vmConfig)

		_, err = vmInstance.InvokeExternally("main")
		require.NoError(t, err)
		require.Equal(t, 0, vmInstance.StackSize())

		require.Equal(
			t,
			[]string{"2", "2", "2"},
			logs,
		)
	})

}

func TestInitializeContract(t *testing.T) {

	t.Parallel()

	location := common.AddressLocation{
		Address: common.MustBytesToAddress([]byte{0x1}),
		Name:    "MyContract",
	}
	programs := map[common.Location]*CompiledProgram{}

	program := ParseCheckAndCompileCodeWithOptions(t,
		`
          contract MyContract {
              var status: String

              init() {
                  self.status = "PENDING"
              }
          }
        `,
		location,
		ParseCheckAndCompileOptions{},
		programs,
	)

	vmConfig := PrepareVMConfig(t, nil, programs)
	vmInstance, contractValue := initializeContract(
		t,
		scriptLocation(),
		program,
		vmConfig,
	)

	fieldValue := contractValue.GetMember(vmInstance.Context(), "status")
	assert.Equal(t, interpreter.NewUnmeteredStringValue("PENDING"), fieldValue)
}

func TestContractAccessDuringInit(t *testing.T) {

	t.Parallel()

	t.Run("using contract name", func(t *testing.T) {
		t.Parallel()

		location := common.AddressLocation{
			Address: common.MustBytesToAddress([]byte{0x1}),
			Name:    "MyContract",
		}
		programs := CompiledPrograms{}

		program := ParseCheckAndCompile(t,
			`
              contract MyContract {
                  var status: String

                  fun getInitialStatus(): String {
                      return "PENDING"
                  }

                  init() {
                      self.status = MyContract.getInitialStatus()
                  }
              }
            `,
			location,
			programs,
		)

		vmConfig := PrepareVMConfig(t, nil, programs)

		vmInstance, contractValue := initializeContract(
			t,
			scriptLocation(),
			program,
			vmConfig,
		)

		fieldValue := contractValue.GetMember(vmInstance.Context(), "status")
		assert.Equal(t, interpreter.NewUnmeteredStringValue("PENDING"), fieldValue)
	})

	t.Run("using self", func(t *testing.T) {
		t.Parallel()

		location := common.AddressLocation{
			Address: common.MustBytesToAddress([]byte{0x1}),
			Name:    "MyContract",
		}
		programs := CompiledPrograms{}

		program := ParseCheckAndCompile(t,
			`
              contract MyContract {
                  var status: String

                  fun getInitialStatus(): String {
                      return "PENDING"
                  }

                  init() {
                      self.status = self.getInitialStatus()
                  }
              }
            `,
			location,
			programs,
		)

		vmConfig := PrepareVMConfig(t, nil, programs)

		vmInstance, contractValue := initializeContract(
			t,
			scriptLocation(),
			program,
			vmConfig,
		)

		fieldValue := contractValue.GetMember(vmInstance.Context(), "status")
		assert.Equal(t, interpreter.NewUnmeteredStringValue("PENDING"), fieldValue)
	})
}

func initializeContract(
	tb testing.TB,
	location common.Location,
	program *bbq.Program[opcode.Instruction, bbq.StaticType],
	vmConfig *vm.Config,
) (*vm.VM, *interpreter.CompositeValue) {
	if vmConfig == nil {
		vmConfig = vm.NewConfig(nil)
	}
	if vmConfig.Storage() == nil {
		vmConfig.SetStorage(NewUnmeteredInMemoryStorage())
	}

	vmInstance := vm.NewVM(location, program, vmConfig)

	// Assume only one contract per program
	require.True(tb, len(program.Contracts) > 0)
	contract := program.Contracts[0]

	contractValue, err := vmInstance.InitializeContract(contract.Name)
	require.NoError(tb, err)

	return vmInstance, contractValue
}

func TestFunctionOrder(t *testing.T) {

	t.Parallel()

	t.Run("top level", func(t *testing.T) {
		t.Parallel()

		checker, err := ParseAndCheck(t, `
          fun foo(): Int {
              return 2
          }

          fun test(): Int {
              return foo() + bar()
          }

          fun bar(): Int {
              return 3
          }
        `)
		require.NoError(t, err)

		comp := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(checker),
			checker.Location,
		)
		program := comp.Compile()

		vmConfig := vm.NewConfig(NewUnmeteredInMemoryStorage())
		vmInstance := vm.NewVM(scriptLocation(), program, vmConfig)

		result, err := vmInstance.InvokeExternally("test")
		require.NoError(t, err)
		require.Equal(t, 0, vmInstance.StackSize())

		require.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(5), result)
	})

	t.Run("nested", func(t *testing.T) {
		t.Parallel()

		location := common.AddressLocation{
			Address: common.MustBytesToAddress([]byte{0x1}),
			Name:    "MyContract",
		}
		code := `
          contract MyContract {

              fun helloText(): String {
                  return "global function of the imported program"
              }

              init() {}

              fun initializeFoo() {
                  MyContract.Foo("one")
              }

              struct Foo {
                  var id : String

                  init(_ id: String) {
                      self.id = id
                  }

                  fun sayHello(_ id: Int): String {
                      self.id
                      return MyContract.helloText()
                  }
              }

              fun initializeFooAgain() {
                  MyContract.Foo("two")
              }
          }
        `

		checker, err := ParseAndCheckWithOptions(
			t,
			code,
			ParseAndCheckOptions{
				Location: location,
			},
		)
		require.NoError(t, err)

		comp := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(checker),
			checker.Location,
		)
		program := comp.Compile()

		vmConfig := vm.NewConfig(NewUnmeteredInMemoryStorage())
		vmInstance, contractValue := initializeContract(
			t,
			scriptLocation(),
			program,
			vmConfig,
		)
		require.Equal(t, 0, vmInstance.StackSize())

		require.IsType(t, &interpreter.CompositeValue{}, contractValue)
	})
}

func TestContractField(t *testing.T) {

	t.Parallel()

	t.Run("get", func(t *testing.T) {
		t.Parallel()

		importLocation := common.AddressLocation{
			Address: common.MustBytesToAddress([]byte{0x1}),
			Name:    "MyContract",
		}
		importedChecker, err := ParseAndCheckWithOptions(t,
			`
              contract MyContract {
                  var status : String

                  init() {
                      self.status = "PENDING"
                  }
              }
            `,
			ParseAndCheckOptions{
				Location: importLocation,
			},
		)
		require.NoError(t, err)

		importCompiler := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(importedChecker),
			importedChecker.Location,
		)
		importedProgram := importCompiler.Compile()

		var importedContractValue *interpreter.CompositeValue

		vmConfig := vm.NewConfig(NewUnmeteredInMemoryStorage())
		vmConfig.ImportHandler = func(location common.Location) *bbq.InstructionProgram {
			require.Equal(t, importLocation, location)
			return importedProgram
		}
		vmConfig.ContractValueHandler = func(_ *vm.Context, location common.Location) *interpreter.CompositeValue {
			require.Equal(t, importLocation, location)
			return importedContractValue
		}
		vmConfig.CompositeTypeHandler = func(location common.Location, typeID interpreter.TypeID) *sema.CompositeType {
			require.Equal(t, importLocation, location)
			elaboration := importedChecker.Elaboration
			return elaboration.CompositeType(typeID)
		}
		vmConfig.InterfaceTypeHandler = func(location common.Location, typeID interpreter.TypeID) *sema.InterfaceType {
			require.Equal(t, importLocation, location)
			elaboration := importedChecker.Elaboration
			return elaboration.InterfaceType(typeID)
		}

		_, importedContractValue = initializeContract(
			t,
			importLocation,
			importedProgram,
			vmConfig,
		)

		checker, err := ParseAndCheckWithOptions(t,
			`
              import MyContract from 0x01

              fun test(): String {
                  return MyContract.status
              }
            `,
			ParseAndCheckOptions{
				CheckerConfig: &sema.Config{
					LocationHandler: SingleIdentifierLocationResolver(t),
					ImportHandler: func(_ *sema.Checker, location common.Location, _ ast.Range) (sema.Import, error) {
						require.Equal(t, importedChecker.Location, location)
						return sema.ElaborationImport{
							Elaboration: importedChecker.Elaboration,
						}, nil
					},
				},
			},
		)
		require.NoError(t, err)

		comp := compiler.NewInstructionCompilerWithConfig(
			interpreter.ProgramFromChecker(checker),
			checker.Location,
			&compiler.Config{
				LocationHandler: SingleIdentifierLocationResolver(t),
				ImportHandler: func(location common.Location) *bbq.InstructionProgram {
					require.Equal(t, importedChecker.Location, location)
					return importedProgram
				},
			},
		)

		program := comp.Compile()

		vmInstance := vm.NewVM(scriptLocation(), program, vmConfig)
		result, err := vmInstance.InvokeExternally("test")
		require.NoError(t, err)
		require.Equal(t, 0, vmInstance.StackSize())

		require.Equal(t, interpreter.NewUnmeteredStringValue("PENDING"), result)
	})

	t.Run("set", func(t *testing.T) {
		t.Parallel()

		importLocation := common.AddressLocation{
			Address: common.MustBytesToAddress([]byte{0x1}),
			Name:    "MyContract",
		}
		importedChecker, err := ParseAndCheckWithOptions(t,
			`
              contract MyContract {
                  var status : String

                  init() {
                      self.status = "PENDING"
                  }
              }
            `,
			ParseAndCheckOptions{
				Location: importLocation,
			},
		)
		require.NoError(t, err)

		importCompiler := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(importedChecker),
			importedChecker.Location,
		)
		importedProgram := importCompiler.Compile()

		var importedContractValue *interpreter.CompositeValue

		vmConfig := vm.NewConfig(NewUnmeteredInMemoryStorage())
		vmConfig.ImportHandler = func(location common.Location) *bbq.InstructionProgram {
			require.Equal(t, importLocation, location)
			return importedProgram
		}
		vmConfig.ContractValueHandler = func(_ *vm.Context, location common.Location) *interpreter.CompositeValue {
			require.Equal(t, importLocation, location)
			return importedContractValue
		}
		vmConfig.CompositeTypeHandler = func(location common.Location, typeID interpreter.TypeID) *sema.CompositeType {
			require.Equal(t, importLocation, location)
			elaboration := importedChecker.Elaboration
			return elaboration.CompositeType(typeID)
		}
		vmConfig.InterfaceTypeHandler = func(location common.Location, typeID interpreter.TypeID) *sema.InterfaceType {
			require.Equal(t, importLocation, location)
			elaboration := importedChecker.Elaboration
			return elaboration.InterfaceType(typeID)
		}

		_, importedContractValue = initializeContract(
			t,
			importLocation,
			importedProgram,
			vmConfig,
		)

		checker, err := ParseAndCheckWithOptions(t,
			`
              import MyContract from 0x01

              fun test(): String {
                  MyContract.status = "UPDATED"
                  return MyContract.status
              }
            `,
			ParseAndCheckOptions{
				CheckerConfig: &sema.Config{
					LocationHandler: SingleIdentifierLocationResolver(t),
					ImportHandler: func(_ *sema.Checker, location common.Location, _ ast.Range) (sema.Import, error) {
						require.Equal(t, importedChecker.Location, location)
						return sema.ElaborationImport{
							Elaboration: importedChecker.Elaboration,
						}, nil
					},
				},
			},
		)
		require.NoError(t, err)

		comp := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(checker),
			checker.Location,
		)
		comp.Config.ImportHandler = func(location common.Location) *bbq.InstructionProgram {
			require.Equal(t, importedChecker.Location, location)
			return importedProgram
		}

		program := comp.Compile()

		vmInstance := vm.NewVM(scriptLocation(), program, vmConfig)

		result, err := vmInstance.InvokeExternally("test")
		require.NoError(t, err)
		require.Equal(t, 0, vmInstance.StackSize())

		require.Equal(t, interpreter.NewUnmeteredStringValue("UPDATED"), result)

		fieldValue := importedContractValue.GetMember(vmInstance.Context(), "status")
		assert.Equal(t, interpreter.NewUnmeteredStringValue("UPDATED"), fieldValue)
	})
}

func TestNativeFunctions(t *testing.T) {

	t.Parallel()

	t.Run("static function", func(t *testing.T) {

		t.Parallel()

		var logged []string

		baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
		baseValueActivation.DeclareValue(stdlib.NewVMLogFunction(nil))

		checker, err := ParseAndCheckWithOptions(t,
			`
              fun test() {
                  log("Hello, World!")
              }
            `,
			ParseAndCheckOptions{
				CheckerConfig: &sema.Config{
					BaseValueActivationHandler: func(common.Location) *sema.VariableActivation {
						return baseValueActivation
					},
				},
			},
		)
		require.NoError(t, err)

		compConfig := &compiler.Config{
			BuiltinGlobalsProvider: CompilerDefaultBuiltinGlobalsWithDefaultsAndLog,
		}

		comp := compiler.NewInstructionCompilerWithConfig(
			interpreter.ProgramFromChecker(checker),
			checker.Location,
			compConfig,
		)
		program := comp.Compile()

		vmConfig := vm.NewConfig(NewUnmeteredInMemoryStorage())
		vmConfig.BuiltinGlobalsProvider = NewVMBuiltinGlobalsProviderWithDefaultsPanicAndLog(&logged)

		vmInstance := vm.NewVM(scriptLocation(), program, vmConfig)

		_, err = vmInstance.InvokeExternally("test")
		require.NoError(t, err)
		require.Equal(t, 0, vmInstance.StackSize())

		assert.Equal(t, []string{`"Hello, World!"`}, logged)
	})

	t.Run("bound function", func(t *testing.T) {
		t.Parallel()

		checker, err := ParseAndCheck(t, `
          fun test(): String {
              return "Hello".concat(", World!")
          }
        `)
		require.NoError(t, err)

		comp := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(checker),
			checker.Location,
		)
		program := comp.Compile()

		vmConfig := vm.NewConfig(NewUnmeteredInMemoryStorage())
		vmInstance := vm.NewVM(scriptLocation(), program, vmConfig)

		result, err := vmInstance.InvokeExternally("test")
		require.NoError(t, err)
		require.Equal(t, 0, vmInstance.StackSize())

		require.Equal(t, interpreter.NewUnmeteredStringValue("Hello, World!"), result)
	})

	t.Run("optional args assert", func(t *testing.T) {
		t.Parallel()

		baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
		baseValueActivation.DeclareValue(stdlib.VMAssertFunction)

		checker, err := ParseAndCheckWithOptions(t,
			`
              fun test() {
                  assert(true)
                  assert(false, message: "hello")
              }
            `,
			ParseAndCheckOptions{
				CheckerConfig: &sema.Config{
					BaseValueActivationHandler: func(common.Location) *sema.VariableActivation {
						return baseValueActivation
					},
				},
			},
		)
		require.NoError(t, err)

		compConfig := &compiler.Config{
			BuiltinGlobalsProvider: func(_ common.Location) *activations.Activation[compiler.GlobalImport] {
				activation := activations.NewActivation(nil, compiler.DefaultBuiltinGlobals())
				activation.Set(
					stdlib.AssertFunctionName,
					compiler.NewGlobalImport(stdlib.AssertFunctionName),
				)
				return activation
			},
		}

		comp := compiler.NewInstructionCompilerWithConfig(
			interpreter.ProgramFromChecker(checker),
			checker.Location,
			compConfig,
		)
		program := comp.Compile()

		vmConfig := vm.NewConfig(NewUnmeteredInMemoryStorage())
		vmConfig.BuiltinGlobalsProvider = func(_ common.Location) *activations.Activation[vm.Variable] {
			activation := activations.NewActivation(nil, vm.DefaultBuiltinGlobals())
			variable := &interpreter.SimpleVariable{}
			variable.InitializeWithValue(stdlib.VMAssertFunction.Value)
			activation.Set(stdlib.AssertFunctionName, variable)
			return activation
		}

		vmInstance := vm.NewVM(scriptLocation(), program, vmConfig)

		_, err = vmInstance.InvokeExternally("test")
		require.ErrorContains(t, err, "assertion failed: hello")
		require.Equal(t, 0, vmInstance.StackSize())
	})
}

func TestTransaction(t *testing.T) {

	t.Parallel()

	t.Run("simple", func(t *testing.T) {

		t.Parallel()

		vmConfig := vm.NewConfig(NewUnmeteredInMemoryStorage())

		vmInstance, err := CompileAndPrepareToInvoke(
			t,
			`
              transaction {
                  var a: String

                  prepare() {
                      self.a = "Hello!"
                  }

                  execute {
                      self.a = "Hello again!"
                  }
              }
            `,
			CompilerAndVMOptions{
				VMConfig: vmConfig,
				ParseCheckAndCompileOptions: ParseCheckAndCompileOptions{
					ParseAndCheckOptions: &ParseAndCheckOptions{
						Location: common.TransactionLocation{},
					},
				},
			},
		)
		require.NoError(t, err)

		vmContext := vmInstance.Context()

		err = vmInstance.InvokeTransaction(nil)
		require.NoError(t, err)
		require.Equal(t, 0, vmInstance.StackSize())

		// Rerun the same again using internal functions, to get the access to the transaction value.

		transaction, err := vmInstance.InvokeTransactionWrapper()
		require.NoError(t, err)
		require.Equal(t, 0, vmInstance.StackSize())

		// At the beginning, 'a' is uninitialized
		assert.Nil(t, transaction.GetMember(vmContext, "a"))

		// Invoke 'prepare'
		err = vmInstance.InvokeTransactionPrepare(transaction, nil)
		require.NoError(t, err)
		require.Equal(t, 0, vmInstance.StackSize())

		// Once 'prepare' is called, 'a' is initialized to "Hello!"
		assert.Equal(
			t,
			interpreter.NewUnmeteredStringValue("Hello!"),
			transaction.GetMember(vmContext, "a"),
		)

		// Invoke 'execute'
		err = vmInstance.InvokeTransactionExecute(transaction)
		require.NoError(t, err)
		require.Equal(t, 0, vmInstance.StackSize())

		// Once 'execute' is called, 'a' is initialized to "Hello, again!"
		assert.Equal(
			t,
			interpreter.NewUnmeteredStringValue("Hello again!"),
			transaction.GetMember(vmContext, "a"),
		)
	})

	t.Run("with params", func(t *testing.T) {

		t.Parallel()

		vmConfig := vm.NewConfig(NewUnmeteredInMemoryStorage())

		vmInstance, err := CompileAndPrepareToInvoke(
			t,
			`
                transaction(param1: String, param2: String) {
                    var a: String

                    prepare() {
                        self.a = param1
                    }

                    execute {
                        self.a = param2
                    }
                }
            `,
			CompilerAndVMOptions{
				VMConfig: vmConfig,
				ParseCheckAndCompileOptions: ParseCheckAndCompileOptions{
					ParseAndCheckOptions: &ParseAndCheckOptions{
						Location: common.TransactionLocation{},
					},
				},
			},
		)
		require.NoError(t, err)

		vmContext := vmInstance.Context()

		args := []vm.Value{
			interpreter.NewUnmeteredStringValue("Hello!"),
			interpreter.NewUnmeteredStringValue("Hello again!"),
		}

		err = vmInstance.InvokeTransaction(args)
		require.NoError(t, err)
		require.Equal(t, 0, vmInstance.StackSize())

		// Rerun the same again using internal functions, to get the access to the transaction value.

		transaction, err := vmInstance.InvokeTransactionWrapper()
		require.NoError(t, err)
		require.Equal(t, 0, vmInstance.StackSize())

		// At the beginning, 'a' is uninitialized
		assert.Nil(t, transaction.GetMember(vmContext, "a"))

		// Invoke 'prepare'
		err = vmInstance.InvokeTransactionPrepare(transaction, nil)
		require.NoError(t, err)
		require.Equal(t, 0, vmInstance.StackSize())

		// Once 'prepare' is called, 'a' is initialized to "Hello!"
		assert.Equal(
			t,
			interpreter.NewUnmeteredStringValue("Hello!"),
			transaction.GetMember(vmContext, "a"),
		)

		// Invoke 'execute'
		err = vmInstance.InvokeTransactionExecute(transaction)
		require.NoError(t, err)
		require.Equal(t, 0, vmInstance.StackSize())

		// Once 'execute' is called, 'a' is initialized to "Hello, again!"
		assert.Equal(
			t,
			interpreter.NewUnmeteredStringValue("Hello again!"),
			transaction.GetMember(vmContext, "a"),
		)
	})

	t.Run("conditions with execute", func(t *testing.T) {

		t.Parallel()

		location := common.TransactionLocation{0x1}

		activation := sema.NewVariableActivation(sema.BaseValueActivation)
		activation.DeclareValue(stdlib.VMPanicFunction)
		activation.DeclareValue(newConditionLogFunction(nil))

		parseAndCheckOptions := &ParseAndCheckOptions{
			Location: location,
			CheckerConfig: &sema.Config{
				LocationHandler: SingleIdentifierLocationResolver(t),
				BaseValueActivationHandler: func(location common.Location) *sema.VariableActivation {
					return activation
				},
			},
		}

		compilerConfig := &compiler.Config{
			BuiltinGlobalsProvider: CompilerDefaultBuiltinGlobalsWithDefaultsAndConditionLog,
		}

		vmConfig := vm.NewConfig(NewUnmeteredInMemoryStorage())

		var logs []string

		vmConfig.BuiltinGlobalsProvider = NewVMBuiltinGlobalsProviderWithDefaultsPanicAndConditionLog(&logs)

		vmInstance, err := CompileAndPrepareToInvoke(t,
			`
              transaction {
                  var count: Int

                  prepare() {
                      self.count = 2
                  }

                  pre {
                      conditionLog(self.count)
                  }

                  execute {
                      self.count = 10
                  }

                  post {
                      conditionLog(self.count)
                  }
              }
            `,
			CompilerAndVMOptions{
				VMConfig: vmConfig,
				ParseCheckAndCompileOptions: ParseCheckAndCompileOptions{
					ParseAndCheckOptions: parseAndCheckOptions,
					CompilerConfig:       compilerConfig,
				},
			},
		)
		require.NoError(t, err)

		err = vmInstance.InvokeTransaction(nil)
		require.NoError(t, err)
		require.Equal(t, 0, vmInstance.StackSize())

		assert.Equal(t, []string{"2", "10"}, logs)
	})

	t.Run("conditions without execute", func(t *testing.T) {

		t.Parallel()

		location := common.TransactionLocation{0x1}

		activation := sema.NewVariableActivation(sema.BaseValueActivation)
		activation.DeclareValue(stdlib.VMPanicFunction)
		activation.DeclareValue(newConditionLogFunction(nil))

		parseAndCheckOptions := &ParseAndCheckOptions{
			Location: location,
			CheckerConfig: &sema.Config{
				LocationHandler: SingleIdentifierLocationResolver(t),
				BaseValueActivationHandler: func(location common.Location) *sema.VariableActivation {
					return activation
				},
			},
		}

		compilerConfig := &compiler.Config{
			BuiltinGlobalsProvider: CompilerDefaultBuiltinGlobalsWithDefaultsAndConditionLog,
		}

		vmConfig := vm.NewConfig(NewUnmeteredInMemoryStorage())

		var logs []string

		vmConfig.BuiltinGlobalsProvider = NewVMBuiltinGlobalsProviderWithDefaultsPanicAndConditionLog(&logs)

		vmInstance, err := CompileAndPrepareToInvoke(t,
			`
              transaction {
                  var count: Int

                  prepare() {
                      self.count = 2
                  }

                  pre {
                      conditionLog(self.count)
                  }

                  post {
                      conditionLog(self.count)
                  }
              }
            `,
			CompilerAndVMOptions{
				VMConfig: vmConfig,
				ParseCheckAndCompileOptions: ParseCheckAndCompileOptions{
					ParseAndCheckOptions: parseAndCheckOptions,
					CompilerConfig:       compilerConfig,
				},
			},
		)
		require.NoError(t, err)

		err = vmInstance.InvokeTransaction(nil)
		require.NoError(t, err)
		require.Equal(t, 0, vmInstance.StackSize())

		assert.Equal(t, []string{"2", "2"}, logs)
	})

	t.Run("pre condition failed", func(t *testing.T) {

		t.Parallel()

		location := common.TransactionLocation{0x1}

		activation := sema.NewVariableActivation(sema.BaseValueActivation)
		activation.DeclareValue(stdlib.VMPanicFunction)
		activation.DeclareValue(newConditionLogFunction(nil))

		parseAndCheckOptions := &ParseAndCheckOptions{
			Location: location,
			CheckerConfig: &sema.Config{
				LocationHandler: SingleIdentifierLocationResolver(t),
				BaseValueActivationHandler: func(location common.Location) *sema.VariableActivation {
					return activation
				},
			},
		}

		compilerConfig := &compiler.Config{
			BuiltinGlobalsProvider: CompilerDefaultBuiltinGlobalsWithDefaultsAndConditionLog,
		}

		vmConfig := vm.NewConfig(NewUnmeteredInMemoryStorage())

		var logs []string

		vmConfig.BuiltinGlobalsProvider = NewVMBuiltinGlobalsProviderWithDefaultsPanicAndConditionLog(&logs)

		vmInstance, err := CompileAndPrepareToInvoke(t,
			`
              transaction {
                  var count: Int

                  prepare() {
                      self.count = 2
                  }

                  pre {
                      conditionLog(self.count)
                      false
                  }

                  execute {
                      self.count = 10
                  }

                  post {
                      conditionLog(self.count)
                  }
              }
            `,
			CompilerAndVMOptions{
				VMConfig: vmConfig,
				ParseCheckAndCompileOptions: ParseCheckAndCompileOptions{
					ParseAndCheckOptions: parseAndCheckOptions,
					CompilerConfig:       compilerConfig,
				},
			},
		)
		require.NoError(t, err)

		err = vmInstance.InvokeTransaction(nil)
		RequireError(t, err)

		var conditionError *interpreter.ConditionError
		assert.ErrorAs(t, err, &conditionError)
		assert.Equal(
			t,
			ast.ConditionKindPre,
			conditionError.ConditionKind,
		)
		assert.Equal(
			t,
			"",
			conditionError.Message,
		)

		assert.Equal(t, []string{"2"}, logs)
	})

	t.Run("post condition failed", func(t *testing.T) {

		t.Parallel()

		location := common.TransactionLocation{0x1}

		activation := sema.NewVariableActivation(sema.BaseValueActivation)
		activation.DeclareValue(stdlib.VMPanicFunction)
		activation.DeclareValue(newConditionLogFunction(nil))

		parseAndCheckOptions := &ParseAndCheckOptions{
			Location: location,
			CheckerConfig: &sema.Config{
				LocationHandler: SingleIdentifierLocationResolver(t),
				BaseValueActivationHandler: func(location common.Location) *sema.VariableActivation {
					return activation
				},
			},
		}

		compilerConfig := &compiler.Config{
			BuiltinGlobalsProvider: CompilerDefaultBuiltinGlobalsWithDefaultsAndConditionLog,
		}

		vmConfig := vm.NewConfig(NewUnmeteredInMemoryStorage())

		var logs []string

		vmConfig.BuiltinGlobalsProvider = NewVMBuiltinGlobalsProviderWithDefaultsPanicAndConditionLog(&logs)

		vmInstance, err := CompileAndPrepareToInvoke(t,
			`
              transaction {
                  var count: Int

                  prepare() {
                      self.count = 2
                  }

                  pre {
                      conditionLog(self.count)
                  }

                  execute {
                      self.count = 10
                  }

                  post {
                      conditionLog(self.count)
                      false
                  }
              }
            `,
			CompilerAndVMOptions{
				VMConfig: vmConfig,
				ParseCheckAndCompileOptions: ParseCheckAndCompileOptions{
					ParseAndCheckOptions: parseAndCheckOptions,
					CompilerConfig:       compilerConfig,
				},
			},
		)
		require.NoError(t, err)

		err = vmInstance.InvokeTransaction(nil)
		RequireError(t, err)

		var conditionError *interpreter.ConditionError
		assert.ErrorAs(t, err, &conditionError)
		assert.Equal(
			t,
			ast.ConditionKindPost,
			conditionError.ConditionKind,
		)
		assert.Equal(
			t,
			"",
			conditionError.Message,
		)

		assert.Equal(t, []string{"2", "10"}, logs)
	})
}

func TestInterfaceMethodCall(t *testing.T) {

	t.Parallel()

	t.Run("impl in same program", func(t *testing.T) {

		t.Parallel()

		contractLocation := common.NewAddressLocation(
			nil,
			common.MustBytesToAddress([]byte{0x1}),
			"MyContract",
		)

		importedChecker, err := ParseAndCheckWithOptions(t,
			`
              contract MyContract {
                  struct Foo: Greetings {
                      var id : String

                      init(_ id: String) {
                          self.id = id
                      }

                      fun sayHello(_ id: Int): String {
                          return self.id
                      }
                  }

                  struct interface Greetings {
                      fun sayHello(_ id: Int): String
                  }

                  struct interface SomethingElse {
                  }
              }
            `,
			ParseAndCheckOptions{
				Location: contractLocation,
			},
		)
		require.NoError(t, err)

		importCompiler := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(importedChecker),
			importedChecker.Location,
		)
		importCompiler.Config.ElaborationResolver = func(location common.Location) (*compiler.DesugaredElaboration, error) {
			if location == contractLocation {
				return compiler.NewDesugaredElaboration(importedChecker.Elaboration), nil
			}

			return nil, fmt.Errorf("cannot find elaboration for %s", location)
		}

		importedProgram := importCompiler.Compile()

		_, importedContractValue := initializeContract(
			t,
			contractLocation,
			importedProgram,
			nil,
		)

		checker, err := ParseAndCheckWithOptions(t,
			`
              import MyContract from 0x01

              fun test(): String {
                  var r: {MyContract.Greetings} = MyContract.Foo("Hello from Foo!")
                  // first call must link
                  r.sayHello(1)

                  // second call should pick from the cache
                  return r.sayHello(1)
              }
            `,

			ParseAndCheckOptions{
				CheckerConfig: &sema.Config{
					LocationHandler: SingleIdentifierLocationResolver(t),
					ImportHandler: func(_ *sema.Checker, location common.Location, _ ast.Range) (sema.Import, error) {
						require.Equal(t, importedChecker.Location, location)
						return sema.ElaborationImport{
							Elaboration: importedChecker.Elaboration,
						}, nil
					},
				},
			},
		)
		require.NoError(t, err)

		comp := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(checker),
			checker.Location,
		)
		comp.Config.LocationHandler = SingleIdentifierLocationResolver(t)
		comp.Config.ImportHandler = func(location common.Location) *bbq.InstructionProgram {
			require.Equal(t, importedChecker.Location, location)
			return importedProgram
		}

		program := comp.Compile()

		vmConfig := vm.NewConfig(NewUnmeteredInMemoryStorage())
		vmConfig.ImportHandler = func(location common.Location) *bbq.InstructionProgram {
			require.Equal(t, importedChecker.Location, location)
			return importedProgram
		}
		vmConfig.ContractValueHandler = func(_ *vm.Context, location common.Location) *interpreter.CompositeValue {
			require.Equal(t, importedChecker.Location, location)
			return importedContractValue
		}
		vmConfig.CompositeTypeHandler = func(location common.Location, typeID interpreter.TypeID) *sema.CompositeType {
			require.Equal(t, importedChecker.Location, location)
			elaboration := importedChecker.Elaboration
			return elaboration.CompositeType(typeID)
		}
		vmConfig.InterfaceTypeHandler = func(location common.Location, typeID interpreter.TypeID) *sema.InterfaceType {
			require.Equal(t, importedChecker.Location, location)
			elaboration := importedChecker.Elaboration
			return elaboration.InterfaceType(typeID)
		}

		vmInstance := vm.NewVM(scriptLocation(), program, vmConfig)
		result, err := vmInstance.InvokeExternally("test")
		require.NoError(t, err)
		require.Equal(t, 0, vmInstance.StackSize())

		require.Equal(t, interpreter.NewUnmeteredStringValue("Hello from Foo!"), result)
	})

	t.Run("impl in different program", func(t *testing.T) {

		t.Parallel()

		// Define the interface in `Foo`

		fooLocation := common.NewAddressLocation(
			nil,
			common.MustBytesToAddress([]byte{0x1}),
			"Foo",
		)

		fooChecker, err := ParseAndCheckWithOptions(t,
			`
              contract Foo {
                  struct interface Greetings {
                      fun sayHello(): String
                  }
              }
            `,
			ParseAndCheckOptions{
				Location: fooLocation,
			},
		)
		require.NoError(t, err)

		interfaceCompiler := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(fooChecker),
			fooChecker.Location,
		)
		fooProgram := interfaceCompiler.Compile()

		_, fooContractValue := initializeContract(
			t,
			fooLocation,
			fooProgram,
			nil,
		)

		// Deploy the imported `Bar` program

		barLocation := common.NewAddressLocation(
			nil,
			common.MustBytesToAddress([]byte{0x2}),
			"Bar",
		)

		barChecker, err := ParseAndCheckWithOptions(t,
			`
              contract Bar {
                  fun sayHello(): String {
                      return "Hello from Bar!"
                  }
              }
            `,
			ParseAndCheckOptions{
				CheckerConfig: &sema.Config{
					LocationHandler: SingleIdentifierLocationResolver(t),
				},
				Location: barLocation,
			},
		)
		require.NoError(t, err)

		barCompiler := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(barChecker),
			barChecker.Location,
		)
		barProgram := barCompiler.Compile()

		_, barContractValue := initializeContract(
			t,
			barLocation,
			barProgram,
			nil,
		)

		// Define the implementation

		bazLocation := common.NewAddressLocation(
			nil,
			common.MustBytesToAddress([]byte{0x3}),
			"Baz",
		)

		bazChecker, err := ParseAndCheckWithOptions(t,
			`
              import Foo from 0x01
              import Bar from 0x02

              contract Baz {
                  struct GreetingImpl: Foo.Greetings {
                      fun sayHello(): String {
                          return Bar.sayHello()
                      }
                  }
              }
            `,
			ParseAndCheckOptions{
				CheckerConfig: &sema.Config{
					LocationHandler: SingleIdentifierLocationResolver(t),
					ImportHandler: func(_ *sema.Checker, location common.Location, _ ast.Range) (sema.Import, error) {
						var elaboration *sema.Elaboration
						switch location {
						case fooLocation:
							elaboration = fooChecker.Elaboration
						case barLocation:
							elaboration = barChecker.Elaboration
						default:
							return nil, fmt.Errorf("cannot find import for: %s", location)
						}

						return sema.ElaborationImport{
							Elaboration: elaboration,
						}, nil
					},
				},
				Location: bazLocation,
			},
		)
		require.NoError(t, err)

		bazImportHandler := func(location common.Location) *bbq.InstructionProgram {
			switch location {
			case fooLocation:
				return fooProgram
			case barLocation:
				return barProgram
			default:
				panic(fmt.Errorf("cannot find import for: %s", location))
			}
		}

		bazCompiler := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(bazChecker),
			bazChecker.Location,
		)
		bazCompiler.Config.LocationHandler = SingleIdentifierLocationResolver(t)
		bazCompiler.Config.ImportHandler = bazImportHandler
		bazCompiler.Config.ElaborationResolver = func(location common.Location) (*compiler.DesugaredElaboration, error) {
			switch location {
			case fooLocation:
				return compiler.NewDesugaredElaboration(fooChecker.Elaboration), nil
			case barLocation:
				return compiler.NewDesugaredElaboration(barChecker.Elaboration), nil
			default:
				return nil, fmt.Errorf("cannot find elaboration for %s", location)
			}
		}

		bazProgram := bazCompiler.Compile()

		vmConfig := vm.NewConfig(NewUnmeteredInMemoryStorage())
		vmConfig.ImportHandler = bazImportHandler
		vmConfig.ContractValueHandler = func(_ *vm.Context, location common.Location) *interpreter.CompositeValue {
			switch location {
			case fooLocation:
				return fooContractValue
			case barLocation:
				return barContractValue
			default:
				assert.FailNow(t, fmt.Sprintf("invalid location %s", location))
				return nil
			}
		}
		vmConfig.CompositeTypeHandler = func(location common.Location, typeID interpreter.TypeID) *sema.CompositeType {
			var elaboration *sema.Elaboration

			switch location {
			case fooLocation:
				elaboration = fooChecker.Elaboration
			case barLocation:
				elaboration = barChecker.Elaboration
			default:
				assert.FailNow(t, fmt.Sprintf("invalid location %s", location))
			}

			compositeType := elaboration.CompositeType(typeID)
			return compositeType
		}
		vmConfig.InterfaceTypeHandler = func(location common.Location, typeID interpreter.TypeID) *sema.InterfaceType {
			var elaboration *sema.Elaboration

			switch location {
			case fooLocation:
				elaboration = fooChecker.Elaboration
			case barLocation:
				elaboration = barChecker.Elaboration
			default:
				assert.FailNow(t, fmt.Sprintf("invalid location %s", location))
			}

			interfaceType := elaboration.InterfaceType(typeID)
			return interfaceType
		}

		_, bazContractValue := initializeContract(
			t,
			bazLocation,
			bazProgram,
			vmConfig,
		)

		// Get `Bar.GreetingsImpl` value

		checker, err := ParseAndCheckWithOptions(t,
			`
              import Baz from 0x03

              fun test(): Baz.GreetingImpl {
                  return Baz.GreetingImpl()
              }
            `,
			ParseAndCheckOptions{
				CheckerConfig: &sema.Config{
					LocationHandler: SingleIdentifierLocationResolver(t),
					ImportHandler: func(_ *sema.Checker, location common.Location, _ ast.Range) (sema.Import, error) {
						var elaboration *sema.Elaboration
						switch location {
						case bazLocation:
							elaboration = bazChecker.Elaboration
						default:
							return nil, fmt.Errorf("cannot find import for: %s", location)
						}

						return sema.ElaborationImport{
							Elaboration: elaboration,
						}, nil
					},
				},
			},
		)
		require.NoError(t, err)

		scriptImportHandler := func(location common.Location) *bbq.InstructionProgram {
			switch location {
			case barLocation:
				return barProgram
			case bazLocation:
				return bazProgram
			default:
				panic(fmt.Errorf("cannot find import for: %s", location))
			}
		}

		comp := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(checker),
			checker.Location,
		)
		comp.Config.LocationHandler = SingleIdentifierLocationResolver(t)
		comp.Config.ImportHandler = scriptImportHandler

		program := comp.Compile()

		vmConfig = vm.NewConfig(NewUnmeteredInMemoryStorage())
		vmConfig.ImportHandler = scriptImportHandler
		vmConfig.ContractValueHandler = func(_ *vm.Context, location common.Location) *interpreter.CompositeValue {
			switch location {
			case barLocation:
				return barContractValue
			case bazLocation:
				return bazContractValue
			default:
				assert.FailNow(t, fmt.Sprintf("invalid location %s", location))
				return nil
			}
		}
		vmConfig.CompositeTypeHandler = func(location common.Location, typeID interpreter.TypeID) *sema.CompositeType {
			var elaboration *sema.Elaboration

			switch location {
			case barLocation:
				elaboration = barChecker.Elaboration
			case bazLocation:
				elaboration = bazChecker.Elaboration
			default:
				assert.FailNow(t, fmt.Sprintf("invalid location %s", location))
			}

			compositeType := elaboration.CompositeType(typeID)
			return compositeType
		}
		vmConfig.InterfaceTypeHandler = func(location common.Location, typeID interpreter.TypeID) *sema.InterfaceType {
			var elaboration *sema.Elaboration

			switch location {
			case barLocation:
				elaboration = barChecker.Elaboration
			case bazLocation:
				elaboration = bazChecker.Elaboration
			default:
				assert.FailNow(t, fmt.Sprintf("invalid location %s", location))
			}

			interfaceType := elaboration.InterfaceType(typeID)
			return interfaceType
		}

		scriptVM := vm.NewVM(scriptLocation(), program, vmConfig)
		implValue, err := scriptVM.InvokeExternally("test")
		require.NoError(t, err)
		require.Equal(t, 0, scriptVM.StackSize())

		require.IsType(t, &interpreter.CompositeValue{}, implValue)
		compositeValue := implValue.(*interpreter.CompositeValue)
		require.Equal(
			t,
			common.TypeID("A.0000000000000003.Baz.GreetingImpl"),
			compositeValue.TypeID(),
		)

		// Test Script. This program only imports `Foo` statically.
		// But the argument passed into the script is of type `Baz.GreetingImpl`.
		// So the linking of `Baz` happens dynamically at runtime.
		// However, `Baz` also has an import to `Bar`. So when the
		// `Baz` is linked and imported at runtime, its imports also
		// should get linked at runtime (similar to how static linking works).

		checker, err = ParseAndCheckWithOptions(t,
			`
              import Foo from 0x01

              fun test(v: {Foo.Greetings}): String {
                  return v.sayHello()
              }
            `,
			ParseAndCheckOptions{
				CheckerConfig: &sema.Config{
					LocationHandler: SingleIdentifierLocationResolver(t),
					ImportHandler: func(_ *sema.Checker, location common.Location, _ ast.Range) (sema.Import, error) {
						var elaboration *sema.Elaboration
						switch location {
						case fooLocation:
							elaboration = fooChecker.Elaboration
						case bazLocation:
							elaboration = bazChecker.Elaboration
						default:
							return nil, fmt.Errorf("cannot find import for: %s", location)
						}

						return sema.ElaborationImport{
							Elaboration: elaboration,
						}, nil
					},
				},
			},
		)
		require.NoError(t, err)

		scriptImportHandler = func(location common.Location) *bbq.InstructionProgram {
			switch location {
			case fooLocation:
				return fooProgram
			case barLocation:
				return barProgram
			case bazLocation:
				return bazProgram
			default:
				panic(fmt.Errorf("cannot find import for: %s", location))
			}
		}

		comp = compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(checker),
			checker.Location,
		)
		comp.Config.LocationHandler = SingleIdentifierLocationResolver(t)
		comp.Config.ImportHandler = scriptImportHandler

		program = comp.Compile()

		vmConfig = vm.NewConfig(NewUnmeteredInMemoryStorage())
		vmConfig.ImportHandler = scriptImportHandler
		vmConfig.ContractValueHandler = func(_ *vm.Context, location common.Location) *interpreter.CompositeValue {
			switch location {
			case fooLocation:
				return fooContractValue
			case barLocation:
				return barContractValue
			case bazLocation:
				return bazContractValue
			default:
				assert.FailNow(t, fmt.Sprintf("invalid location %s", location))
				return nil
			}
		}
		vmConfig.CompositeTypeHandler = func(location common.Location, typeID interpreter.TypeID) *sema.CompositeType {
			var elaboration *sema.Elaboration

			switch location {
			case fooLocation:
				elaboration = fooChecker.Elaboration
			case barLocation:
				elaboration = barChecker.Elaboration
			case bazLocation:
				elaboration = bazChecker.Elaboration
			default:
				assert.FailNow(t, fmt.Sprintf("invalid location %s", location))
			}

			compositeType := elaboration.CompositeType(typeID)
			return compositeType
		}
		vmConfig.InterfaceTypeHandler = func(location common.Location, typeID interpreter.TypeID) *sema.InterfaceType {
			var elaboration *sema.Elaboration

			switch location {
			case fooLocation:
				elaboration = fooChecker.Elaboration
			case barLocation:
				elaboration = barChecker.Elaboration
			case bazLocation:
				elaboration = bazChecker.Elaboration
			default:
				assert.FailNow(t, fmt.Sprintf("invalid location %s", location))
			}

			interfaceType := elaboration.InterfaceType(typeID)
			return interfaceType
		}

		scriptVM = vm.NewVM(scriptLocation(), program, vmConfig)

		result, err := scriptVM.InvokeExternally("test", implValue)
		require.NoError(t, err)
		require.Equal(t, 0, scriptVM.StackSize())

		require.Equal(t, interpreter.NewUnmeteredStringValue("Hello from Bar!"), result)
	})
}

func TestArrayLiteral(t *testing.T) {

	t.Parallel()

	t.Run("array literal", func(t *testing.T) {
		t.Parallel()

		checker, err := ParseAndCheck(t, `
            fun test(): [Int] {
                return [2, 5]
            }
        `)
		require.NoError(t, err)

		comp := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(checker),
			checker.Location,
		)
		program := comp.Compile()

		vmConfig := vm.NewConfig(NewUnmeteredInMemoryStorage())
		vmInstance := vm.NewVM(scriptLocation(), program, vmConfig)

		vmContext := vmInstance.Context()

		result, err := vmInstance.InvokeExternally("test")
		require.NoError(t, err)
		require.Equal(t, 0, vmInstance.StackSize())

		require.IsType(t, &interpreter.ArrayValue{}, result)
		array := result.(*interpreter.ArrayValue)
		assert.Equal(t, 2, array.Count())
		assert.Equal(
			t,
			interpreter.NewUnmeteredIntValueFromInt64(2),
			array.Get(vmContext, 0),
		)
		assert.Equal(
			t,
			interpreter.NewUnmeteredIntValueFromInt64(5),
			array.Get(vmContext, 1),
		)
	})

	t.Run("array get", func(t *testing.T) {
		t.Parallel()

		checker, err := ParseAndCheck(t, `
            fun test(): Int {
                var a = [2, 5, 7, 3]
                return a[1]
            }
        `)
		require.NoError(t, err)

		comp := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(checker),
			checker.Location,
		)
		program := comp.Compile()

		vmConfig := vm.NewConfig(NewUnmeteredInMemoryStorage())
		vmInstance := vm.NewVM(scriptLocation(), program, vmConfig)

		result, err := vmInstance.InvokeExternally("test")
		require.NoError(t, err)
		require.Equal(t, 0, vmInstance.StackSize())
		assert.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(5), result)
	})

	t.Run("array set", func(t *testing.T) {
		t.Parallel()

		checker, err := ParseAndCheck(t, `
            fun test(): [Int] {
                var a = [2, 5, 4]
                a[2] = 8
                return a
            }
        `)
		require.NoError(t, err)

		comp := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(checker),
			checker.Location,
		)
		program := comp.Compile()

		vmConfig := vm.NewConfig(NewUnmeteredInMemoryStorage())
		vmInstance := vm.NewVM(scriptLocation(), program, vmConfig)
		vmContext := vmInstance.Context()

		result, err := vmInstance.InvokeExternally("test")
		require.NoError(t, err)
		require.Equal(t, 0, vmInstance.StackSize())

		require.IsType(t, &interpreter.ArrayValue{}, result)
		array := result.(*interpreter.ArrayValue)
		assert.Equal(t, 3, array.Count())
		assert.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(2), array.Get(vmContext, 0))
		assert.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(5), array.Get(vmContext, 1))
		assert.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(8), array.Get(vmContext, 2))
	})
}

func TestDictionaryLiteral(t *testing.T) {

	t.Parallel()

	t.Run("dictionary literal", func(t *testing.T) {
		t.Parallel()

		checker, err := ParseAndCheck(t, `
            fun test(): {String: Int} {
                return {"b": 2, "e": 5}
            }
        `)
		require.NoError(t, err)

		comp := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(checker),
			checker.Location,
		)
		program := comp.Compile()

		vmConfig := vm.NewConfig(NewUnmeteredInMemoryStorage())
		vmInstance := vm.NewVM(scriptLocation(), program, vmConfig)
		vmContext := vmInstance.Context()

		result, err := vmInstance.InvokeExternally("test")
		require.NoError(t, err)
		require.Equal(t, 0, vmInstance.StackSize())

		require.IsType(t, &interpreter.DictionaryValue{}, result)
		dictionary := result.(*interpreter.DictionaryValue)
		assert.Equal(t, 2, dictionary.Count())
		assert.Equal(t,
			interpreter.NewUnmeteredSomeValueNonCopying(
				interpreter.NewUnmeteredIntValueFromInt64(2),
			),
			dictionary.GetKey(
				vmContext,
				interpreter.NewUnmeteredStringValue("b"),
			),
		)
		assert.Equal(t,
			interpreter.NewUnmeteredSomeValueNonCopying(
				interpreter.NewUnmeteredIntValueFromInt64(5),
			),
			dictionary.GetKey(
				vmContext,
				interpreter.NewUnmeteredStringValue("e"),
			),
		)
	})
}

func TestReference(t *testing.T) {

	t.Parallel()

	t.Run("method call", func(t *testing.T) {
		t.Parallel()

		code := `
            struct Foo {
                var id : String

                init(_ id: String) {
                    self.id = id
                }

                fun sayHello(_ id: Int): String {
                    return self.id
                }
            }

            fun test(): String {
                var foo = Foo("Hello from Foo!")
                var ref = &foo as &Foo
                return ref.sayHello(1)
            }
        `

		result, err := CompileAndInvoke(t, code, "test")
		require.NoError(t, err)

		require.Equal(t, interpreter.NewUnmeteredStringValue("Hello from Foo!"), result)
	})
}

func TestResource(t *testing.T) {

	t.Parallel()

	t.Run("new", func(t *testing.T) {
		t.Parallel()

		programs := map[common.Location]*CompiledProgram{}

		program := ParseCheckAndCompile(
			t,
			`
                resource Foo {
                    var id : Int

                    init(_ id: Int) {
                        self.id = id
                    }
                }

                fun test(): @Foo {
                    var i = 0
                    var r <- create Foo(5)
                    return <- r
                }
            `,
			TestLocation,
			programs,
		)

		vmConfig := PrepareVMConfig(t, nil, programs)

		vmInstance := vm.NewVM(scriptLocation(), program, vmConfig)

		vmContext := vmInstance.Context()

		result, err := vmInstance.InvokeExternally("test")
		require.NoError(t, err)
		require.Equal(t, 0, vmInstance.StackSize())

		require.IsType(t, &interpreter.CompositeValue{}, result)
		structValue := result.(*interpreter.CompositeValue)
		compositeType := structValue.StaticType(vmContext).(*interpreter.CompositeStaticType)

		require.Equal(t, "Foo", compositeType.QualifiedIdentifier)
		require.Equal(
			t,
			interpreter.NewUnmeteredIntValueFromInt64(5),
			structValue.GetMember(vmContext, "id"),
		)
	})

	t.Run("destroy", func(t *testing.T) {
		t.Parallel()

		_, err := CompileAndInvoke(
			t,
			`
                resource Foo {
                    var id : Int

                    init(_ id: Int) {
                        self.id = id
                    }
                }

                fun test() {
                    var i = 0
                    var r <- create Foo(5)
                    destroy r
                }
            `,
			"test",
		)

		require.NoError(t, err)
	})
}

func TestDefaultFunctions(t *testing.T) {

	t.Parallel()

	t.Run("simple interface", func(t *testing.T) {
		t.Parallel()

		result, err := CompileAndInvoke(t,
			`
              struct interface IA {
                  fun test(): Int {
                      return 42
                  }
              }

              struct Test: IA {}

              fun main(): Int {
                 return Test().test()
              }
            `,
			"main",
		)

		require.NoError(t, err)
		require.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(42), result)
	})

	t.Run("overridden", func(t *testing.T) {
		t.Parallel()

		result, err := CompileAndInvoke(t,
			`
              struct interface IA {
                  fun test(): Int {
                      return 41
                  }
              }

              struct Test: IA {
                  fun test(): Int {
                      return 42
                  }
              }

              fun main(): Int {
                 return Test().test()
              }
            `,
			"main",
		)

		require.NoError(t, err)
		require.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(42), result)
	})

	t.Run("default method via different paths", func(t *testing.T) {

		t.Parallel()

		result, err := CompileAndInvoke(t,
			`
              struct interface A {
                  fun test(): Int {
                      return 3
                  }
              }

              struct interface B: A {}

              struct interface C: A {}

              struct D: B, C {}

              fun main(): Int {
                  let d = D()
                  return d.test()
              }
            `,
			"main",
		)

		require.NoError(t, err)
		assert.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(3), result)
	})

	t.Run("in different contract", func(t *testing.T) {

		t.Parallel()

		storage := NewUnmeteredInMemoryStorage()

		programs := map[common.Location]*CompiledProgram{}
		contractValues := map[common.Location]*interpreter.CompositeValue{}

		vmConfig := vm.NewConfig(storage)
		vmConfig.ImportHandler = func(location common.Location) *bbq.InstructionProgram {
			program, ok := programs[location]
			if !ok {
				assert.FailNow(t, "invalid location")
			}
			return program.Program
		}
		vmConfig.ContractValueHandler = func(_ *vm.Context, location common.Location) *interpreter.CompositeValue {
			contractValue, ok := contractValues[location]
			if !ok {
				assert.FailNow(t, "invalid location")
			}
			return contractValue
		}
		vmConfig.CompositeTypeHandler = CompiledProgramsCompositeTypeLoader(programs)
		vmConfig.InterfaceTypeHandler = CompiledProgramsInterfaceTypeLoader(programs)
		vmConfig.EntitlementTypeHandler = CompiledProgramsEntitlementTypeLoader(programs)
		vmConfig.EntitlementMapTypeHandler = CompiledProgramsEntitlementMapTypeLoader(programs)

		var uuid uint64 = 42
		vmConfig.UUIDHandler = func() (uint64, error) {
			uuid++
			return uuid, nil
		}

		contractsAddress := common.MustBytesToAddress([]byte{0x1})

		barLocation := common.NewAddressLocation(nil, contractsAddress, "Bar")
		fooLocation := common.NewAddressLocation(nil, contractsAddress, "Foo")

		// Deploy interface contract

		barContract := `
          contract interface Bar {

              resource interface VaultInterface {

                  var balance: Int

                  fun getBalance(): Int {
                      return self.balance
                  }
              }
          }
        `

		// Only need to compile
		ParseCheckAndCompile(t, barContract, barLocation, programs)

		// Deploy contract with the implementation

		fooContract := fmt.Sprintf(
			`
              import Bar from %[1]s

              contract Foo {

                  resource Vault: Bar.VaultInterface {
                      var balance: Int

                      init(balance: Int) {
                          self.balance = balance
                      }

                      fun withdraw(amount: Int): @Vault {
                          self.balance = self.balance - amount
                          return <-create Vault(balance: amount)
                      }
                  }

                  fun createVault(balance: Int): @Vault {
                      return <- create Vault(balance: balance)
                  }
              }
            `,
			contractsAddress.HexWithPrefix(),
		)

		fooProgram := ParseCheckAndCompile(t, fooContract, fooLocation, programs)

		_, fooContractValue := initializeContract(
			t,
			fooLocation,
			fooProgram,
			vmConfig,
		)
		contractValues[fooLocation] = fooContractValue

		// Run transaction

		tx := fmt.Sprintf(
			`
              import Foo from %[1]s

              fun main(): Int {
                 var vault <- Foo.createVault(balance: 10)
                 destroy vault.withdraw(amount: 3)
                 var balance = vault.getBalance()
                 destroy vault
                 return balance
              }
            `,
			contractsAddress.HexWithPrefix(),
		)

		txLocation := NewTransactionLocationGenerator()

		txProgram := ParseCheckAndCompile(t, tx, txLocation(), programs)
		txVM := vm.NewVM(txLocation(), txProgram, vmConfig)

		result, err := txVM.InvokeExternally("main")
		require.NoError(t, err)
		require.Equal(t, 0, txVM.StackSize())
		require.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(7), result)
	})

	t.Run("in different contract with nested call", func(t *testing.T) {

		t.Parallel()

		storage := NewUnmeteredInMemoryStorage()

		programs := map[common.Location]*CompiledProgram{}
		contractValues := map[common.Location]*interpreter.CompositeValue{}

		vmConfig := vm.NewConfig(storage)
		vmConfig.ImportHandler = func(location common.Location) *bbq.InstructionProgram {
			program, ok := programs[location]
			if !ok {
				assert.FailNow(t, "invalid location")
			}
			return program.Program
		}
		vmConfig.ContractValueHandler = func(_ *vm.Context, location common.Location) *interpreter.CompositeValue {
			contractValue, ok := contractValues[location]
			if !ok {
				assert.FailNow(t, "invalid location")
			}
			return contractValue
		}
		vmConfig.CompositeTypeHandler = CompiledProgramsCompositeTypeLoader(programs)
		vmConfig.InterfaceTypeHandler = CompiledProgramsInterfaceTypeLoader(programs)
		vmConfig.EntitlementTypeHandler = CompiledProgramsEntitlementTypeLoader(programs)
		vmConfig.EntitlementMapTypeHandler = CompiledProgramsEntitlementMapTypeLoader(programs)

		var uuid uint64 = 42
		vmConfig.UUIDHandler = func() (uint64, error) {
			uuid++
			return uuid, nil
		}

		contractsAddress := common.MustBytesToAddress([]byte{0x1})

		barLocation := common.NewAddressLocation(nil, contractsAddress, "Bar")
		fooLocation := common.NewAddressLocation(nil, contractsAddress, "Foo")

		// Deploy interface contract

		barContract := `
          contract interface Bar {

              resource interface HelloInterface {

                  fun sayHello(): String {
                      // Delegate the call
                      return self.sayHelloImpl()
                  }

                  access(contract) fun sayHelloImpl(): String {
                      return "Hello from HelloInterface"
                  }
              }
          }
        `

		// Only need to compile
		ParseCheckAndCompile(t, barContract, barLocation, programs)

		// Deploy contract with the implementation

		fooContract := fmt.Sprintf(
			`
              import Bar from %[1]s

              contract Foo {

                  resource Hello: Bar.HelloInterface { }

                  fun createHello(): @Hello {
                      return <- create Hello()
                  }
              }
            `,
			contractsAddress.HexWithPrefix(),
		)

		fooProgram := ParseCheckAndCompile(t, fooContract, fooLocation, programs)

		_, fooContractValue := initializeContract(
			t,
			fooLocation,
			fooProgram,
			vmConfig,
		)
		contractValues[fooLocation] = fooContractValue

		// Run transaction

		tx := fmt.Sprintf(
			`
              import Foo from %[1]s

              fun main(): String {
                 var hello <- Foo.createHello()
                 var msg = hello.sayHello()
                 destroy hello
                 return msg
              }
            `,
			contractsAddress.HexWithPrefix(),
		)

		txLocation := NewTransactionLocationGenerator()

		txProgram := ParseCheckAndCompile(t, tx, txLocation(), programs)
		txVM := vm.NewVM(txLocation(), txProgram, vmConfig)

		result, err := txVM.InvokeExternally("main")
		require.NoError(t, err)
		require.Equal(t, 0, txVM.StackSize())
		require.Equal(t, interpreter.NewUnmeteredStringValue("Hello from HelloInterface"), result)
	})

	t.Run("in different contract nested call overridden", func(t *testing.T) {

		t.Parallel()

		storage := NewUnmeteredInMemoryStorage()

		programs := map[common.Location]*CompiledProgram{}
		contractValues := map[common.Location]*interpreter.CompositeValue{}

		vmConfig := vm.NewConfig(storage)

		vmConfig.ImportHandler = func(location common.Location) *bbq.InstructionProgram {
			program, ok := programs[location]
			if !ok {
				assert.FailNow(t, "invalid location")
			}
			return program.Program
		}
		vmConfig.ContractValueHandler = func(_ *vm.Context, location common.Location) *interpreter.CompositeValue {
			contractValue, ok := contractValues[location]
			if !ok {
				assert.FailNow(t, "invalid location")
			}
			return contractValue
		}
		vmConfig.CompositeTypeHandler = CompiledProgramsCompositeTypeLoader(programs)
		vmConfig.InterfaceTypeHandler = CompiledProgramsInterfaceTypeLoader(programs)
		vmConfig.EntitlementTypeHandler = CompiledProgramsEntitlementTypeLoader(programs)
		vmConfig.EntitlementMapTypeHandler = CompiledProgramsEntitlementMapTypeLoader(programs)

		var uuid uint64 = 42
		vmConfig.UUIDHandler = func() (uint64, error) {
			uuid++
			return uuid, nil
		}

		contractsAddress := common.MustBytesToAddress([]byte{0x1})

		barLocation := common.NewAddressLocation(nil, contractsAddress, "Bar")
		fooLocation := common.NewAddressLocation(nil, contractsAddress, "Foo")

		// Deploy interface contract

		barContract := `
          contract interface Bar {

              resource interface HelloInterface {

                  fun sayHello(): String {
                      // Delegate the call
                      return self.sayHelloImpl()
                  }

                  access(contract) fun sayHelloImpl(): String {
                      return "Hello from HelloInterface"
                  }
              }
          }
        `

		// Only need to compile
		ParseCheckAndCompile(t, barContract, barLocation, programs)

		// Deploy contract with the implementation

		fooContract := fmt.Sprintf(
			`
              import Bar from %[1]s

              contract Foo {

                  resource Hello: Bar.HelloInterface {

                      // Override one of the functions (one at the bottom of the call hierarchy)
                      access(contract) fun sayHelloImpl(): String {
                          return "Hello from Hello"
                      }
                  }

                  fun createHello(): @Hello {
                      return <- create Hello()
                  }
              }
            `,
			contractsAddress.HexWithPrefix(),
		)

		fooProgram := ParseCheckAndCompile(t, fooContract, fooLocation, programs)

		_, fooContractValue := initializeContract(
			t,
			fooLocation,
			fooProgram,
			vmConfig,
		)
		contractValues[fooLocation] = fooContractValue

		// Run transaction

		tx := fmt.Sprintf(
			`
              import Foo from %[1]s

              fun main(): String {
                 var hello <- Foo.createHello()
                 var msg = hello.sayHello()
                 destroy hello
                 return msg
              }
            `,
			contractsAddress.HexWithPrefix(),
		)

		txLocation := NewTransactionLocationGenerator()

		txProgram := ParseCheckAndCompile(t, tx, txLocation(), programs)
		txVM := vm.NewVM(txLocation(), txProgram, vmConfig)

		result, err := txVM.InvokeExternally("main")
		require.NoError(t, err)
		require.Equal(t, 0, txVM.StackSize())
		require.Equal(t, interpreter.NewUnmeteredStringValue("Hello from Hello"), result)
	})
}

func TestFunctionPreConditions(t *testing.T) {

	t.Parallel()

	t.Run("failed", func(t *testing.T) {

		t.Parallel()

		_, err := CompileAndInvoke(t,
			`
              fun main(x: Int): Int {
                  pre {
                      x == 0
                  }
                  return x
              }
            `,
			"main",
			interpreter.NewUnmeteredIntValueFromInt64(3),
		)

		RequireError(t, err)

		var conditionError *interpreter.ConditionError
		assert.ErrorAs(t, err, &conditionError)
		assert.Equal(
			t,
			ast.ConditionKindPre,
			conditionError.ConditionKind,
		)
		assert.Equal(
			t,
			"",
			conditionError.Message,
		)
	})

	t.Run("failed with message", func(t *testing.T) {

		t.Parallel()

		_, err := CompileAndInvoke(t,
			`
              fun main(x: Int): Int {
                  pre {
                      x == 0: "x must be zero"
                  }
                  return x
              }
            `,
			"main",
			interpreter.NewUnmeteredIntValueFromInt64(3),
		)

		RequireError(t, err)

		var conditionError *interpreter.ConditionError
		assert.ErrorAs(t, err, &conditionError)
		assert.Equal(
			t,
			ast.ConditionKindPre,
			conditionError.ConditionKind,
		)
		assert.Equal(
			t,
			"x must be zero",
			conditionError.Message,
		)
	})

	t.Run("passed", func(t *testing.T) {

		t.Parallel()

		result, err := CompileAndInvoke(t,
			`
              fun main(x: Int): Int {
                  pre {
                      x != 0
                  }
                  return x
              }
            `,
			"main",
			interpreter.NewUnmeteredIntValueFromInt64(3),
		)

		require.NoError(t, err)
		assert.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(3), result)
	})

	t.Run("inherited", func(t *testing.T) {

		t.Parallel()

		_, err := CompileAndInvoke(t,
			`
              struct interface A {
                  fun test(_ a: Int): Int {
                      pre { a > 10: "a must be larger than 10" }
                  }
              }

              struct interface B: A {
                  fun test(_ a: Int): Int
              }

              struct C: B {
                  fun test(_ a: Int): Int {
                      return a + 3
                  }
              }

              fun main(_ a: Int): Int {
                  let c = C()
                  return c.test(a)
              }
            `,
			"main",
			interpreter.NewUnmeteredIntValueFromInt64(4),
		)

		RequireError(t, err)
		assert.ErrorContains(t, err, "a must be larger than 10")
	})

	t.Run("inherited with different param names", func(t *testing.T) {

		t.Parallel()

		_, err := CompileAndInvoke(t,
			`
              struct interface A {
                  fun test(a x: Int): Int {
                      pre { x > 10: "a must be larger than 10" }
                  }
              }

              struct interface B: A {
                  fun test(a y: Int): Int
              }

              struct C: B {
                  fun test(a z: Int): Int {
                      return z + 3
                  }
              }

              fun main(_ a: Int): Int {
                  let c = C()
                  return c.test(a: a)
              }
            `,
			"main",
			interpreter.NewUnmeteredIntValueFromInt64(4),
		)

		RequireError(t, err)
		assert.ErrorContains(t, err, "a must be larger than 10")
	})

	t.Run("inherited with different param names, conflict with local vars", func(t *testing.T) {

		t.Parallel()

		_, err := CompileAndInvoke(t,
			`
              struct interface A {
                  fun test(a x: Int): Int {
                      pre { x > 10: "a must be larger than 10" }
                  }
              }

              struct interface B: A {
                  fun test(a y: Int): Int
              }

              // Just a global declaration with conflicting names 'x'
              fun x() {}

              struct C: B {
                  fun test(a z: Int): Int {
                      // condition is inherited as:
                      //    'pre { x > 10: "a must be larger than 10" }'
                      // Referring to 'x', instead of 'z'.

                      // Just another local variable with conflicting names 'x'
                      var x = 45

                      return z + 3
                  }
              }

              fun main(_ a: Int): Int {
                  let c = C()
                  return c.test(a: a)
              }
            `,
			"main",
			interpreter.NewUnmeteredIntValueFromInt64(4),
		)

		RequireError(t, err)
		assert.ErrorContains(t, err, "a must be larger than 10")
	})

	t.Run("pre conditions order", func(t *testing.T) {

		t.Parallel()

		code := `
          struct A: B {
              fun test() {
                  pre { conditionLog("A") }
              }
          }

          struct interface B: C, D {
              fun test() {
                  pre { conditionLog("B") }
              }
          }

          struct interface C: E, F {
              fun test() {
                  pre { conditionLog("C") }
              }
          }

          struct interface D: F {
              fun test() {
                  pre { conditionLog("D") }
              }
          }

          struct interface E {
              fun test() {
                  pre { conditionLog("E") }
              }
          }

          struct interface F {
              fun test() {
                  pre { conditionLog("F") }
              }
          }

          fun main() {
              let a = A()
              a.test()
          }
        `

		_, logs, err := CompileAndInvokeWithConditionLogs(
			t,
			code,
			"main",
		)
		require.NoError(t, err)

		// The pre-conditions of the interfaces are executed first, with depth-first pre-order traversal.
		// The pre-condition of the concrete type is executed at the end, after the interfaces.
		assert.Equal(t, []string{"\"B\"", "\"C\"", "\"E\"", "\"F\"", "\"D\"", "\"A\""}, logs)
	})

	t.Run("in different contract with nested call", func(t *testing.T) {

		t.Parallel()

		storage := NewUnmeteredInMemoryStorage()

		programs := map[common.Location]*CompiledProgram{}
		contractValues := map[common.Location]*interpreter.CompositeValue{}
		var logs []string

		vmConfig := vm.NewConfig(storage)
		vmConfig.ImportHandler = func(location common.Location) *bbq.InstructionProgram {
			program, ok := programs[location]
			if !ok {
				assert.FailNow(t, "invalid location")
			}
			return program.Program
		}
		vmConfig.ContractValueHandler = func(_ *vm.Context, location common.Location) *interpreter.CompositeValue {
			contractValue, ok := contractValues[location]
			if !ok {
				assert.FailNow(t, "invalid location")
			}
			return contractValue
		}
		vmConfig.CompositeTypeHandler = CompiledProgramsCompositeTypeLoader(programs)
		vmConfig.InterfaceTypeHandler = CompiledProgramsInterfaceTypeLoader(programs)
		vmConfig.EntitlementTypeHandler = CompiledProgramsEntitlementTypeLoader(programs)
		vmConfig.EntitlementMapTypeHandler = CompiledProgramsEntitlementMapTypeLoader(programs)

		vmConfig.BuiltinGlobalsProvider = NewVMBuiltinGlobalsProviderWithDefaultsPanicAndConditionLog(&logs)

		activation := sema.NewVariableActivation(sema.BaseValueActivation)
		activation.DeclareValue(stdlib.VMPanicFunction)
		activation.DeclareValue(newConditionLogFunction(nil))

		contractsAddress := common.MustBytesToAddress([]byte{0x1})

		barLocation := common.NewAddressLocation(nil, contractsAddress, "Bar")
		fooLocation := common.NewAddressLocation(nil, contractsAddress, "Foo")

		// Deploy interface contract

		barContract := `
          contract interface Bar {

              struct interface E {

                  fun test() {
                      pre { self.printFromE("E") }
                  }

                  view fun printFromE(_ msg: String): Bool {
                      conditionLog("Bar.".concat(msg))
                      return true
                  }
              }

              struct interface F {

                  fun test() {
                      pre { self.printFromF("F") }
                  }

                  view fun printFromF(_ msg: String): Bool {
                      conditionLog("Bar.".concat(msg))
                      return true
                  }
              }
          }
        `

		compilerConfig := &compiler.Config{
			BuiltinGlobalsProvider: CompilerDefaultBuiltinGlobalsWithDefaultsAndConditionLog,
		}

		// Only need to compile
		_ = ParseCheckAndCompileCodeWithOptions(
			t,
			barContract,
			barLocation,
			ParseCheckAndCompileOptions{
				CompilerConfig: compilerConfig,
				ParseAndCheckOptions: &ParseAndCheckOptions{
					Location: barLocation,
					CheckerConfig: &sema.Config{
						LocationHandler: SingleIdentifierLocationResolver(t),
						BaseValueActivationHandler: func(location common.Location) *sema.VariableActivation {
							return activation
						},
					},
				},
			},
			programs,
		)

		// Deploy contract with the implementation

		fooContract := fmt.Sprintf(
			`
              import Bar from %[1]s

              contract Foo {

                  struct interface B: C, D {
                      fun test() {
                          pre { Foo.printFromFoo("B") }
                      }
                  }

                  struct interface C: Bar.E, Bar.F {
                      fun test() {
                          pre { Foo.printFromFoo("C") }
                      }
                  }

                  struct interface D: Bar.F {
                      fun test() {
                          pre { Foo.printFromFoo("D") }
                      }
                  }

                  view fun printFromFoo(_ msg: String): Bool {
                      conditionLog("Foo.".concat(msg))
                      return true
                  }
              }
            `,
			contractsAddress.HexWithPrefix(),
		)

		fooProgram := ParseCheckAndCompileCodeWithOptions(
			t,
			fooContract,
			fooLocation,
			ParseCheckAndCompileOptions{
				CompilerConfig: compilerConfig,
				ParseAndCheckOptions: &ParseAndCheckOptions{
					Location: fooLocation,
					CheckerConfig: &sema.Config{
						LocationHandler: SingleIdentifierLocationResolver(t),
						BaseValueActivationHandler: func(location common.Location) *sema.VariableActivation {
							return activation
						},
					},
				},
			},
			programs,
		)

		_, fooContractValue := initializeContract(
			t,
			fooLocation,
			fooProgram,
			vmConfig,
		)
		contractValues[fooLocation] = fooContractValue

		// Run script

		code := fmt.Sprintf(
			`
              import Foo from %[1]s

              struct A: Foo.B {
                  fun test() {
                      pre { conditionLog("A") }
                  }
              }

              fun main() {
                  let a = A()
                  a.test()
              }
            `,
			contractsAddress.HexWithPrefix(),
		)

		location := common.ScriptLocation{0x1}

		_, err := CompileAndInvokeWithOptionsAndPrograms(
			t,
			code,
			"main",
			CompilerAndVMOptions{
				ParseCheckAndCompileOptions: ParseCheckAndCompileOptions{
					CompilerConfig: compilerConfig,
					ParseAndCheckOptions: &ParseAndCheckOptions{
						Location: location,
						CheckerConfig: &sema.Config{
							LocationHandler: SingleIdentifierLocationResolver(t),
							BaseValueActivationHandler: func(location common.Location) *sema.VariableActivation {
								return activation
							},
						},
					},
				},
				VMConfig: vmConfig,
			},
			programs,
		)
		require.NoError(t, err)
		assert.Equal(t, []string{"\"Foo.B\"", "\"Foo.C\"", "\"Bar.E\"", "\"Bar.F\"", "\"Foo.D\"", "\"A\""}, logs)
	})
}

func TestFunctionPostConditions(t *testing.T) {

	t.Parallel()

	t.Run("failed", func(t *testing.T) {

		t.Parallel()

		_, err := CompileAndInvoke(t,
			`
              fun main(x: Int): Int {
                  post {
                      x == 0
                  }
                  return x
              }
            `,
			"main",
			interpreter.NewUnmeteredIntValueFromInt64(3),
		)

		RequireError(t, err)

		var conditionError *interpreter.ConditionError
		assert.ErrorAs(t, err, &conditionError)
		assert.Equal(
			t,
			ast.ConditionKindPost,
			conditionError.ConditionKind,
		)
		assert.Equal(
			t,
			"",
			conditionError.Message,
		)
	})

	t.Run("failed with message", func(t *testing.T) {

		t.Parallel()

		_, err := CompileAndInvoke(t,
			`
              fun main(x: Int): Int {
                  post {
                      x == 0: "x must be zero"
                  }
                  return x
              }
            `,
			"main",
			interpreter.NewUnmeteredIntValueFromInt64(3),
		)

		RequireError(t, err)
		assert.ErrorContains(t, err, "x must be zero")
	})

	t.Run("passed", func(t *testing.T) {

		t.Parallel()

		result, err := CompileAndInvoke(t,
			`
              fun main(x: Int): Int {
                  post {
                      x != 0
                  }
                  return x
              }
            `,
			"main",
			interpreter.NewUnmeteredIntValueFromInt64(3),
		)

		require.NoError(t, err)
		assert.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(3), result)
	})

	t.Run("test on local var", func(t *testing.T) {

		t.Parallel()

		result, err := CompileAndInvoke(t,
			`
              fun main(x: Int): Int {
                  post {
                      y == 5
                  }
                  var y = x + 2
                  return y
              }
            `,
			"main",
			interpreter.NewUnmeteredIntValueFromInt64(3),
		)

		require.NoError(t, err)
		assert.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(5), result)
	})

	t.Run("test on local var failed with message", func(t *testing.T) {

		t.Parallel()

		_, err := CompileAndInvoke(t,
			`
              fun main(x: Int): Int {
                  post {
                      y == 5: "x must be 5"
                  }
                  var y = x + 2
                  return y
              }
            `,
			"main",
			interpreter.NewUnmeteredIntValueFromInt64(4),
		)

		RequireError(t, err)
		assert.ErrorContains(t, err, "x must be 5")
	})

	t.Run("post conditions order", func(t *testing.T) {

		t.Parallel()

		_, logs, err := CompileAndInvokeWithConditionLogs(
			t,
			`
              struct A: B {
                  fun test() {
                      post { conditionLog("A") }
                  }
              }

              struct interface B: C, D {
                  fun test() {
                      post { conditionLog("B") }
                  }
              }

              struct interface C: E, F {
                  fun test() {
                      post { conditionLog("C") }
                  }
              }

              struct interface D: F {
                  fun test() {
                      post { conditionLog("D") }
                  }
              }

              struct interface E {
                  fun test() {
                      post { conditionLog("E") }
                  }
              }

              struct interface F {
                  fun test() {
                      post { conditionLog("F") }
                  }
              }

              fun main() {
                  let a = A()
                  a.test()
              }
            `,
			"main",
		)
		require.NoError(t, err)

		// The post-condition of the concrete type is executed first, before the interfaces.
		// The post-conditions of the interfaces are executed after that, with the reversed depth-first pre-order.
		assert.Equal(t, []string{"\"A\"", "\"D\"", "\"F\"", "\"E\"", "\"C\"", "\"B\""}, logs)
	})

	t.Run("result var failed", func(t *testing.T) {

		t.Parallel()

		_, err := CompileAndInvoke(t,
			`
              fun main(x: Int): Int {
                  post {
                      result == 0: "x must be zero"
                  }
                  return x
              }
            `,
			"main",
			interpreter.NewUnmeteredIntValueFromInt64(3),
		)

		RequireError(t, err)
		assert.ErrorContains(t, err, "x must be zero")
	})

	t.Run("result var passed", func(t *testing.T) {

		t.Parallel()

		result, err := CompileAndInvoke(t,
			`
              fun main(x: Int): Int {
                  post {
                      result != 0
                  }
                  return x
              }
            `,
			"main",
			interpreter.NewUnmeteredIntValueFromInt64(3),
		)

		require.NoError(t, err)
		assert.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(3), result)
	})

	t.Run("result var in inherited condition", func(t *testing.T) {

		t.Parallel()

		_, err := CompileAndInvoke(t,
			`
              struct interface A {
                  fun test(_ a: Int): Int {
                      post { result > 10: "result must be larger than 10" }
                  }
              }

              struct interface B: A {
                  fun test(_ a: Int): Int
              }

              struct C: B {
                  fun test(_ a: Int): Int {
                      return a + 3
                  }
              }

              fun main(_ a: Int): Int {
                  let c = C()
                  return c.test(a)
              }
            `,
			"main",
			interpreter.NewUnmeteredIntValueFromInt64(4),
		)

		RequireError(t, err)
		assert.ErrorContains(t, err, "result must be larger than 10")
	})

	t.Run("resource typed result var passed", func(t *testing.T) {

		t.Parallel()

		result, err := CompileAndInvoke(t,
			`
              resource R {
                  var i: Int
                  init() {
                      self.i = 4
                  }
              }

              fun main(): @R {
                  post {
                      result.i > 0
                  }

                  return <- create R()
              }
            `,
			"main",
		)

		require.NoError(t, err)
		assert.IsType(t, &interpreter.CompositeValue{}, result)
	})

	t.Run("resource typed result var failed", func(t *testing.T) {

		t.Parallel()

		_, err := CompileAndInvoke(t,
			`
              resource R {
                  var i: Int
                  init() {
                      self.i = 4
                  }
              }

              fun main(): @R {
                  post {
                      result.i > 10
                  }


                  return <- create R()
              }
            `,
			"main",
		)

		RequireError(t, err)

		var conditionError *interpreter.ConditionError
		assert.ErrorAs(t, err, &conditionError)
		assert.Equal(
			t,
			ast.ConditionKindPost,
			conditionError.ConditionKind,
		)
		assert.Equal(
			t,
			"",
			conditionError.Message,
		)
	})

	t.Run("inherited with different param names", func(t *testing.T) {

		t.Parallel()

		_, err := CompileAndInvoke(t,
			`
              struct interface A {
                  fun test(a x: Int): Int {
                      post { x > 10: "a must be larger than 10" }
                  }
              }

              struct interface B: A {
                  fun test(a y: Int): Int
              }

              struct C: B {
                  fun test(a z: Int): Int {
                      return z + 3
                  }
              }

              fun main(_ a: Int): Int {
                  let c = C()
                  return c.test(a: a)
              }
            `,
			"main",
			interpreter.NewUnmeteredIntValueFromInt64(4),
		)

		RequireError(t, err)
		assert.ErrorContains(t, err, "a must be larger than 10")
	})
}

func TestIfLet(t *testing.T) {

	t.Parallel()

	test := func(t *testing.T, argument vm.Value) vm.Value {
		result, err := CompileAndInvoke(t,
			`
              fun main(x: Int?): Int {
                  if let y = x {
                     return y
                  } else {
                     return 2
                  }
              }
            `,
			"main",
			argument,
		)
		require.NoError(t, err)
		return result
	}

	t.Run("some", func(t *testing.T) {

		t.Parallel()

		actual := test(t,
			interpreter.NewUnmeteredSomeValueNonCopying(
				interpreter.NewUnmeteredIntValueFromInt64(1),
			),
		)
		assert.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(1), actual)
	})

	t.Run("nil", func(t *testing.T) {

		t.Parallel()

		actual := test(t, interpreter.NilValue{})
		assert.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(2), actual)
	})
}

func TestIfLetScope(t *testing.T) {

	t.Parallel()

	test := func(t *testing.T, argument vm.Value) vm.Value {
		result, err := CompileAndInvoke(t,
			`
              fun test(y: Int?): Int {
                  let x = 1
                  var z = 0
                  if let x = y {
                      z = x
                  } else {
                      z = x
                  }
                  return x + z
              }
            `,
			"test",
			argument,
		)
		require.NoError(t, err)
		return result
	}

	t.Run("some", func(t *testing.T) {

		t.Parallel()

		actual := test(t,
			interpreter.NewUnmeteredSomeValueNonCopying(
				interpreter.NewUnmeteredIntValueFromInt64(10),
			),
		)
		assert.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(11), actual)
	})

	t.Run("nil", func(t *testing.T) {

		t.Parallel()

		actual := test(t, interpreter.NilValue{})
		assert.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(2), actual)
	})
}

func TestGuard(t *testing.T) {

	t.Parallel()

	test := func(t *testing.T, argument vm.Value) vm.Value {
		result, err := CompileAndInvoke(t,
			`
              fun test(x: Bool): Int {
                  var y = 0
                  guard x else {
                     y = 2
                     return y
                  }
                  y = 1
                  return y
              }
            `,
			"test",
			argument,
		)
		require.NoError(t, err)
		return result
	}

	t.Run("true", func(t *testing.T) {

		t.Parallel()

		actual := test(t, interpreter.TrueValue)
		assert.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(1), actual)
	})

	t.Run("false", func(t *testing.T) {

		t.Parallel()

		actual := test(t, interpreter.FalseValue)
		assert.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(2), actual)
	})
}

func TestGuardLet(t *testing.T) {

	t.Parallel()

	test := func(t *testing.T, argument vm.Value) vm.Value {
		result, err := CompileAndInvoke(t,
			`
              fun main(x: Int?): Int {
                  guard let y = x else {
                     return 2
                  }
                  return y
              }
            `,
			"main",
			argument,
		)
		require.NoError(t, err)
		return result
	}

	t.Run("some", func(t *testing.T) {

		t.Parallel()

		actual := test(t,
			interpreter.NewUnmeteredSomeValueNonCopying(
				interpreter.NewUnmeteredIntValueFromInt64(1),
			),
		)
		assert.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(1), actual)
	})

	t.Run("nil", func(t *testing.T) {

		t.Parallel()

		actual := test(t, interpreter.NilValue{})
		assert.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(2), actual)
	})
}

func TestGuardLetScope(t *testing.T) {

	t.Parallel()

	test := func(t *testing.T, argument vm.Value) vm.Value {
		result, err := CompileAndInvoke(t,
			`
              fun test(y: Int?): Int {
                  var z = 0
                  guard let x = y else {
                      z = 1
                      return z
                  }
                  z = x
                  return z
              }
            `,
			"test",
			argument,
		)
		require.NoError(t, err)
		return result
	}

	t.Run("some", func(t *testing.T) {

		t.Parallel()

		actual := test(t,
			interpreter.NewUnmeteredSomeValueNonCopying(
				interpreter.NewUnmeteredIntValueFromInt64(10),
			),
		)
		assert.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(10), actual)
	})

	t.Run("nil", func(t *testing.T) {

		t.Parallel()

		actual := test(t, interpreter.NilValue{})
		assert.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(1), actual)
	})
}

func TestGuardChained(t *testing.T) {

	t.Parallel()

	test := func(t *testing.T, arg1, arg2, arg3 vm.Value) vm.Value {
		result, err := CompileAndInvoke(t,
			`
              fun test(a: Int?, b: Int?, c: Int?): Int {
                  guard let x = a else {
                      return 1
                  }
                  guard let y = b else {
                      return 2
                  }
                  guard let z = c else {
                      return 3
                  }
                  return x + y + z
              }
            `,
			"test",
			arg1,
			arg2,
			arg3,
		)
		require.NoError(t, err)
		return result
	}

	t.Run("first nil", func(t *testing.T) {

		t.Parallel()

		actual := test(t,
			interpreter.NilValue{},
			interpreter.NewUnmeteredSomeValueNonCopying(
				interpreter.NewUnmeteredIntValueFromInt64(5),
			),
			interpreter.NewUnmeteredSomeValueNonCopying(
				interpreter.NewUnmeteredIntValueFromInt64(6),
			),
		)
		assert.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(1), actual)
	})

	t.Run("second nil", func(t *testing.T) {

		t.Parallel()

		actual := test(t,
			interpreter.NewUnmeteredSomeValueNonCopying(
				interpreter.NewUnmeteredIntValueFromInt64(4),
			),
			interpreter.NilValue{},
			interpreter.NewUnmeteredSomeValueNonCopying(
				interpreter.NewUnmeteredIntValueFromInt64(6),
			),
		)
		assert.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(2), actual)
	})

	t.Run("third nil", func(t *testing.T) {

		t.Parallel()

		actual := test(t,
			interpreter.NewUnmeteredSomeValueNonCopying(
				interpreter.NewUnmeteredIntValueFromInt64(4),
			),
			interpreter.NewUnmeteredSomeValueNonCopying(
				interpreter.NewUnmeteredIntValueFromInt64(5),
			),
			interpreter.NilValue{},
		)
		assert.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(3), actual)
	})

	t.Run("all some", func(t *testing.T) {

		t.Parallel()

		actual := test(t,
			interpreter.NewUnmeteredSomeValueNonCopying(
				interpreter.NewUnmeteredIntValueFromInt64(4),
			),
			interpreter.NewUnmeteredSomeValueNonCopying(
				interpreter.NewUnmeteredIntValueFromInt64(5),
			),
			interpreter.NewUnmeteredSomeValueNonCopying(
				interpreter.NewUnmeteredIntValueFromInt64(6),
			),
		)
		assert.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(15), actual)
	})

}

func TestSwitch(t *testing.T) {

	t.Parallel()

	t.Run("1", func(t *testing.T) {
		t.Parallel()

		result, err := CompileAndInvoke(t,
			`
              fun test(x: Int): Int {
                  var a = 0
                  switch x {
                      case 1:
                          a = a + 1
                      case 2:
                          a = a + 2
                      default:
                          a = a + 3
                  }
                  return a
              }
            `,
			"test",
			interpreter.NewUnmeteredIntValueFromInt64(1),
		)
		require.NoError(t, err)
		assert.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(1), result)
	})

	t.Run("2", func(t *testing.T) {
		t.Parallel()

		result, err := CompileAndInvoke(t,
			`
              fun test(x: Int): Int {
                  var a = 0
                  switch x {
                      case 1:
                          a = a + 1
                      case 2:
                          a = a + 2
                      default:
                          a = a + 3
                  }
                  return a
              }
            `,
			"test",
			interpreter.NewUnmeteredIntValueFromInt64(2),
		)
		require.NoError(t, err)
		assert.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(2), result)
	})

	t.Run("4", func(t *testing.T) {
		t.Parallel()

		result, err := CompileAndInvoke(t,
			`
              fun test(x: Int): Int {
                  var a = 0
                  switch x {
                      case 1:
                          a = a + 1
                      case 2:
                          a = a + 2
                      default:
                          a = a + 3
                  }
                  return a
              }
            `,
			"test",
			interpreter.NewUnmeteredIntValueFromInt64(4),
		)
		require.NoError(t, err)
		assert.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(3), result)
	})
}

func TestDefaultFunctionsWithConditions(t *testing.T) {

	t.Parallel()

	t.Run("default in parent, conditions in child", func(t *testing.T) {
		t.Parallel()

		_, logs, err := CompileAndInvokeWithConditionLogs(t,
			`
              struct interface Foo {
                  fun test(_ a: Int) {
                      conditionLog("invoked Foo.test()")
                  }
              }

              struct interface Bar: Foo {
                  fun test(_ a: Int) {
                      pre {
                           conditionLog("invoked Bar.test() pre-condition")
                      }

                      post {
                           conditionLog("invoked Bar.test() post-condition")
                      }
                  }
              }

              struct Test: Bar {}

              fun main() {
                 Test().test(5)
              }
            `,
			"main",
		)

		require.NoError(t, err)
		require.Equal(
			t,
			[]string{
				"\"invoked Bar.test() pre-condition\"",
				"\"invoked Foo.test()\"",
				"\"invoked Bar.test() post-condition\"",
			},
			logs,
		)
	})

	t.Run("default and conditions in parent, more conditions in child", func(t *testing.T) {
		t.Parallel()

		_, logs, err := CompileAndInvokeWithConditionLogs(t,
			`
              struct interface Foo {
                  fun test(_ a: Int) {
                      pre {
                           conditionLog("invoked Foo.test() pre-condition")
                      }
                      post {
                           conditionLog("invoked Foo.test() post-condition")
                      }
                      conditionLog("invoked Foo.test()")
                  }
              }

              struct interface Bar: Foo {
                  fun test(_ a: Int) {
                      pre {
                           conditionLog("invoked Bar.test() pre-condition")
                      }

                      post {
                           conditionLog("invoked Bar.test() post-condition")
                      }
                  }
              }

              struct Test: Bar {}

              fun main() {
                 Test().test(5)
              }
            `,
			"main",
		)

		require.NoError(t, err)
		require.Equal(
			t,
			[]string{
				"\"invoked Bar.test() pre-condition\"",
				"\"invoked Foo.test() pre-condition\"",
				"\"invoked Foo.test()\"",
				"\"invoked Foo.test() post-condition\"",
				"\"invoked Bar.test() post-condition\"",
			},
			logs,
		)
	})

}

func TestBeforeFunctionInPostConditions(t *testing.T) {

	t.Parallel()

	t.Run("condition in same type", func(t *testing.T) {
		t.Parallel()

		var logs []string

		_, logs, err := CompileAndInvokeWithConditionLogs(t,
			`
              struct Test {
                  var i: Int

                  init() {
                      self.i = 2
                  }

                  fun test() {
                      post {
                          conditionLog(before(self.i).toString())
                          conditionLog(self.i.toString())
                      }
                      self.i = 5
                  }
              }

              fun main() {
                 Test().test()
              }
            `,
			"main",
		)

		require.NoError(t, err)
		require.Equal(
			t,
			[]string{
				"\"2\"",
				"\"5\"",
			},
			logs,
		)
	})

	t.Run("inherited condition", func(t *testing.T) {
		t.Parallel()

		_, logs, err := CompileAndInvokeWithConditionLogs(t,
			`
                struct interface Foo {
                    var i: Int

                    fun test() {
                        post {
                            conditionLog(before(self.i).toString())
                            conditionLog(self.i.toString())
                        }
                        self.i = 5
                    }
                }

                struct Test: Foo {
                    var i: Int

                    init() {
                        self.i = 2
                    }
                }

                fun main() {
                    Test().test()
                }
            `,
			"main",
		)

		require.NoError(t, err)
		require.Equal(
			t,
			[]string{
				"\"2\"",
				"\"5\"",
			},
			logs,
		)
	})

	t.Run("multiple inherited conditions", func(t *testing.T) {
		t.Parallel()

		_, logs, err := CompileAndInvokeWithConditionLogs(t,
			`
              struct interface Foo {
                  var i: Int

                  fun test() {
                      post {
                          conditionLog(before(self.i).toString())
                          conditionLog(before(self.i + 1).toString())
                          conditionLog(self.i.toString())
                      }
                      self.i = 8
                  }
              }

              struct interface Bar: Foo {
                  var i: Int

                  fun test() {
                      post {
                          conditionLog(before(self.i + 3).toString())
                      }
                  }
              }


              struct Test: Bar {
                  var i: Int

                  init() {
                      self.i = 2
                  }
              }

              fun main() {
                  Test().test()
              }
            `,
			"main",
		)

		require.NoError(t, err)
		require.Equal(
			t,
			[]string{"\"2\"", "\"3\"", "\"8\"", "\"5\""},
			logs,
		)
	})

	t.Run("resource access in inherited before-statement", func(t *testing.T) {

		t.Parallel()

		_, err := CompileAndInvoke(t,
			`
              resource interface RI {
                  var i: Int

                  fun test(_ r: @R) {
                      post {
                          before(r.i) == 4
                      }
                  }
              }

              resource R: RI {
                  var i: Int
                  init() {
                      self.i = 4
                  }

                  fun test(_ r: @R) {
                      destroy r
                  }
              }

              fun main() {
                  var r1 <- create R()
                  var r2 <- create R()

                  r1.test(<- r2)

                  destroy r1
              }
            `,
			"main",
		)

		require.NoError(t, err)
	})
}

func TestEmit(t *testing.T) {

	t.Parallel()

	var eventEmitted bool

	vmConfig := vm.NewConfig(NewUnmeteredInMemoryStorage())
	vmConfig.OnEventEmitted = func(
		context interpreter.ValueExportContext,
		eventType *sema.CompositeType,
		eventFields []interpreter.Value,
	) error {
		require.False(t, eventEmitted)
		eventEmitted = true

		assert.Equal(t,
			[]interpreter.Value{
				interpreter.NewUnmeteredIntValueFromInt64(1),
			},
			eventFields,
		)

		assert.Equal(t,
			TestLocation.TypeID(nil, "Inc"),
			eventType.ID(),
		)

		return nil
	}

	_, err := CompileAndInvokeWithOptions(t,
		`
          event Inc(val: Int)

          fun test(x: Int) {
              emit Inc(val: x)
          }
        `,
		"test",
		CompilerAndVMOptions{
			VMConfig: vmConfig,
		},
		interpreter.NewUnmeteredIntValueFromInt64(1),
	)
	require.NoError(t, err)

	require.True(t, eventEmitted)
}

func TestCasting(t *testing.T) {

	t.Parallel()

	t.Run("simple cast success", func(t *testing.T) {
		t.Parallel()

		result, err := CompileAndInvoke(t,
			`
              fun test(x: Int): AnyStruct {
                  return x as Int?
              }
            `,
			"test",
			interpreter.NewUnmeteredIntValueFromInt64(2),
		)
		require.NoError(t, err)
		assert.Equal(t,
			interpreter.NewUnmeteredSomeValueNonCopying(
				interpreter.NewUnmeteredIntValueFromInt64(2),
			),
			result,
		)
	})

	t.Run("force cast success", func(t *testing.T) {
		t.Parallel()

		result, err := CompileAndInvoke(t,
			`
              fun test(x: AnyStruct): Int {
                  return x as! Int
              }
            `,
			"test",
			interpreter.NewUnmeteredIntValueFromInt64(2),
		)
		require.NoError(t, err)
		assert.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(2), result)
	})

	t.Run("force cast fail", func(t *testing.T) {
		t.Parallel()

		_, err := CompileAndInvoke(t,
			`
              fun test(x: AnyStruct): Int {
                  return x as! Int
              }
            `,
			"test",
			interpreter.TrueValue,
		)
		RequireError(t, err)

		castingError := &interpreter.ForceCastTypeMismatchError{}
		require.ErrorAs(t, err, &castingError)

		assert.Equal(
			t,
			&interpreter.ForceCastTypeMismatchError{
				ExpectedType: sema.IntType,
				ActualType:   sema.BoolType,
				LocationRange: interpreter.LocationRange{
					Location: TestLocation,
					HasPosition: bbq.Position{
						StartPos: ast.Position{
							Offset: 70,
							Line:   3,
							Column: 25,
						},
						EndPos: ast.Position{
							Offset: 78,
							Line:   3,
							Column: 33,
						},
					},
				},
			},
			castingError,
		)
	})

	t.Run("failable cast success", func(t *testing.T) {
		t.Parallel()

		result, err := CompileAndInvoke(t,
			`
              fun test(x: AnyStruct): Int? {
                  return x as? Int
              }
            `,
			"test",
			interpreter.NewUnmeteredIntValueFromInt64(2),
		)
		require.NoError(t, err)
		assert.Equal(t,
			interpreter.NewUnmeteredSomeValueNonCopying(
				interpreter.NewUnmeteredIntValueFromInt64(2),
			),
			result,
		)
	})

	t.Run("failable cast fail", func(t *testing.T) {
		t.Parallel()

		result, err := CompileAndInvoke(t,
			`
              fun test(x: AnyStruct): Int? {
                  return x as? Int
              }
            `,
			"test",
			interpreter.TrueValue,
		)
		require.NoError(t, err)
		assert.Equal(t, interpreter.Nil, result)
	})
}

func TestBlockScope(t *testing.T) {

	t.Parallel()

	test := func(t *testing.T, argument vm.Value) vm.Value {

		result, err := CompileAndInvoke(t,
			`
                fun test(y: Bool): Int {
                    let x = 1
                    if y {
                        let x = 2
                    } else {
                        let x = 3
                    }
                    return x
                }
            `,
			"test",
			argument,
		)
		require.NoError(t, err)
		return result
	}

	t.Run("true", func(t *testing.T) {
		t.Parallel()

		actual := test(t, interpreter.TrueValue)
		require.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(1), actual)
	})

	t.Run("false", func(t *testing.T) {
		t.Parallel()

		actual := test(t, interpreter.FalseValue)
		require.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(1), actual)
	})
}

func TestBlockScope2(t *testing.T) {

	t.Parallel()

	test := func(t *testing.T, argument vm.Value) vm.Value {

		result, err := CompileAndInvoke(t,
			`
                fun test(y: Bool): Int {
                    let x = 1
                    var z = 0
                    if y {
                        var x = x
                        x = 2
                        z = x
                    } else {
                        var x = x
                        x = 3
                        z = x
                    }
                    return x + z
                }
            `,
			"test",
			argument,
		)
		require.NoError(t, err)
		return result
	}

	t.Run("true", func(t *testing.T) {
		t.Parallel()

		actual := test(t, interpreter.TrueValue)
		require.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(3), actual)
	})

	t.Run("false", func(t *testing.T) {
		t.Parallel()

		actual := test(t, interpreter.FalseValue)
		require.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(4), actual)
	})
}

func TestIntegers(t *testing.T) {

	t.Parallel()

	test := func(integerType sema.Type) {

		t.Run(integerType.String(), func(t *testing.T) {

			t.Parallel()

			result, err := CompileAndInvoke(t,
				fmt.Sprintf(`
                        fun test(): %s {
                            return 2 + 3
                        }
                    `,
					integerType,
				),
				"test",
			)
			require.NoError(t, err)

			assert.Equal(t, "5", result.String())
			assert.Equal(t,
				integerType,
				interpreter.MustConvertStaticToSemaType(result.StaticType(nil), nil),
			)
		})
	}

	for _, integerType := range common.Concat(
		sema.AllUnsignedIntegerTypes,
		sema.AllSignedIntegerTypes,
	) {
		test(integerType)
	}
}

func TestAddress(t *testing.T) {

	t.Parallel()

	result, err := CompileAndInvoke(t,
		`
            fun test(): Address {
                return 0x2
            }
        `,
		"test",
	)
	require.NoError(t, err)

	assert.Equal(t, "0x0000000000000002", result.String())
	assert.Equal(t,
		interpreter.PrimitiveStaticTypeAddress,
		result.StaticType(nil),
	)
}

func TestFixedPoint(t *testing.T) {

	t.Parallel()

	test := func(fixedPointType sema.Type) {

		t.Run(fixedPointType.String(), func(t *testing.T) {

			t.Parallel()

			var expected string
			switch fixedPointType {
			case sema.Fix128Type, sema.UFix128Type:
				expected = "10.000000000000000000000000"
			default:
				expected = "10.00000000"
			}

			result, err := CompileAndInvoke(t,
				fmt.Sprintf(`
                        fun test(): %s {
                            return 2.1 + 7.9
                        }
                    `,
					fixedPointType,
				),
				"test",
			)
			require.NoError(t, err)

			assert.Equal(t, expected, result.String())
			assert.Equal(t,
				fixedPointType,
				interpreter.MustConvertStaticToSemaType(result.StaticType(nil), nil),
			)
		})
	}

	for _, fixedPointType := range common.Concat(
		sema.AllUnsignedFixedPointTypes,
		sema.AllSignedFixedPointTypes,
	) {
		test(fixedPointType)
	}
}

func TestForLoop(t *testing.T) {

	t.Parallel()

	t.Run("array", func(t *testing.T) {
		t.Parallel()

		result, err := CompileAndInvoke(t,
			`
                fun test(): Int {
                    var array = [5, 6, 7, 8]
                    var sum = 0
                    for e in array {
                        sum = sum + e
                    }

                    return sum
                }
            `,
			"test",
		)
		require.NoError(t, err)
		assert.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(26), result)
	})

	t.Run("array with index", func(t *testing.T) {
		t.Parallel()

		result, err := CompileAndInvoke(t,
			`
                fun test(): String {
                    var array = ["a", "b", "c", "d"]
                    var keys = ""
                    var values = ""
                    for i, e in array {
                        keys = keys.concat(i.toString())
                        values = values.concat(e)
                    }

                    return keys.concat("_").concat(values)
                }
            `,
			"test",
		)
		require.NoError(t, err)
		assert.Equal(t, interpreter.NewUnmeteredStringValue("0123_abcd"), result)
	})

	t.Run("array loop scoping", func(t *testing.T) {
		t.Parallel()

		result, err := CompileAndInvoke(t,
			`
                fun test(): String {
                    var array = [5, 6, 7, 8]

                    var offset = 10
                    var values = ""

                    for e in array {
                        var offset = 1
                        var e = e + offset
                        values = values.concat(e.toString())
                    }

                    return values
                }
            `,
			"test",
		)
		require.NoError(t, err)
		assert.Equal(t, interpreter.NewUnmeteredStringValue("6789"), result)
	})
}

func TestIf(t *testing.T) {

	t.Parallel()

	test := func(t *testing.T, argument vm.Value) vm.Value {
		result, err := CompileAndInvoke(t,
			`
                fun test(x: Bool): Int {
                    var y = 0
                    if x {
                        y = 1
                    } else {
                        y = 2
                    }
                    return y
                }
            `,
			"test",
			argument,
		)
		require.NoError(t, err)
		return result
	}

	t.Run("true", func(t *testing.T) {
		t.Parallel()

		actual := test(t, interpreter.TrueValue)
		require.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(1), actual)
	})

	t.Run("false", func(t *testing.T) {
		t.Parallel()

		actual := test(t, interpreter.FalseValue)
		require.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(2), actual)
	})
}

func TestConditional(t *testing.T) {

	t.Parallel()

	test := func(t *testing.T, argument vm.Value) vm.Value {
		result, err := CompileAndInvoke(t,
			`
                fun test(x: Bool): Int {
                    return x ? 1 : 2
                }
            `,
			"test",
			argument,
		)
		require.NoError(t, err)
		return result
	}

	t.Run("true", func(t *testing.T) {
		t.Parallel()

		actual := test(t, interpreter.TrueValue)
		require.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(1), actual)
	})

	t.Run("false", func(t *testing.T) {
		t.Parallel()

		actual := test(t, interpreter.FalseValue)
		require.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(2), actual)
	})
}

func TestOr(t *testing.T) {

	t.Parallel()

	test := func(t *testing.T, x, y vm.Value) vm.Value {
		result, err := CompileAndInvoke(t,
			`
                struct Tester {
                    let x: Bool
                    let y: Bool
                    var z: Int

                    init(x: Bool, y: Bool) {
                        self.x = x
                        self.y = y
                        self.z = 0
                    }

                    fun a(): Bool {
                        self.z = 1
                        return self.x
                    }

                    fun b(): Bool {
                        self.z = 2
                        return self.y
                    }

                    fun test(): Int {
                        if self.a() || self.b() {
                            return self.z + 10
                        } else {
                            return self.z + 20
                        }
                    }
                }

                fun test(x: Bool, y: Bool): Int {
                    return Tester(x: x, y: y).test()
                }
            `,
			"test",
			x,
			y,
		)
		require.NoError(t, err)
		return result
	}

	t.Run("true, true", func(t *testing.T) {
		t.Parallel()

		actual := test(t, interpreter.TrueValue, interpreter.TrueValue)
		require.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(11), actual)
	})

	t.Run("true, false", func(t *testing.T) {
		t.Parallel()

		actual := test(t, interpreter.TrueValue, interpreter.FalseValue)
		require.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(11), actual)
	})

	t.Run("false, true", func(t *testing.T) {
		t.Parallel()

		actual := test(t, interpreter.FalseValue, interpreter.TrueValue)
		require.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(12), actual)
	})

	t.Run("false, false", func(t *testing.T) {
		t.Parallel()

		actual := test(t, interpreter.FalseValue, interpreter.FalseValue)
		require.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(22), actual)
	})
}

func TestAnd(t *testing.T) {

	t.Parallel()

	test := func(t *testing.T, x, y vm.Value) vm.Value {
		result, err := CompileAndInvoke(t,
			`
                struct Tester {
                    let x: Bool
                    let y: Bool
                    var z: Int

                    init(x: Bool, y: Bool) {
                        self.x = x
                        self.y = y
                        self.z = 0
                    }

                    fun a(): Bool {
                        self.z = 1
                        return self.x
                    }

                    fun b(): Bool {
                        self.z = 2
                        return self.y
                    }

                    fun test(): Int {
                        if self.a() && self.b() {
                            return self.z + 10
                        } else {
                            return self.z + 20
                        }
                    }
                }

                fun test(x: Bool, y: Bool): Int {
                    return Tester(x: x, y: y).test()
                }
            `,
			"test",
			x,
			y,
		)
		require.NoError(t, err)
		return result
	}

	t.Run("true, true", func(t *testing.T) {
		t.Parallel()

		actual := test(t, interpreter.TrueValue, interpreter.TrueValue)
		require.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(12), actual)
	})

	t.Run("true, false", func(t *testing.T) {
		t.Parallel()

		actual := test(t, interpreter.TrueValue, interpreter.FalseValue)
		require.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(22), actual)
	})

	t.Run("false, true", func(t *testing.T) {
		t.Parallel()

		actual := test(t, interpreter.FalseValue, interpreter.TrueValue)
		require.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(21), actual)
	})

	t.Run("false, false", func(t *testing.T) {
		t.Parallel()

		actual := test(t, interpreter.FalseValue, interpreter.FalseValue)
		require.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(21), actual)
	})
}

func TestUnaryNot(t *testing.T) {

	t.Parallel()

	test := func(t *testing.T, argument vm.Value) vm.Value {

		actual, err := CompileAndInvoke(t,
			`
              fun test(x: Bool): Bool {
                  return !x
              }
            `,
			"test",
			argument,
		)
		require.NoError(t, err)

		return actual
	}

	t.Run("true", func(t *testing.T) {
		t.Parallel()

		actual := test(t, interpreter.TrueValue)
		require.Equal(t, interpreter.FalseValue, actual)
	})

	t.Run("false", func(t *testing.T) {
		t.Parallel()

		actual := test(t, interpreter.FalseValue)
		require.Equal(t, interpreter.TrueValue, actual)
	})
}

func TestUnaryNegate(t *testing.T) {

	t.Parallel()

	actual, err := CompileAndInvoke(t,
		`
            fun test(x: Int): Int {
                return -x
            }
        `,
		"test",
		interpreter.NewUnmeteredIntValueFromInt64(42),
	)
	require.NoError(t, err)

	assert.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(-42), actual)
}

func TestUnaryDeref(t *testing.T) {

	t.Parallel()

	actual, err := CompileAndInvoke(t,
		`
            fun test(): Int {
                let x = 42
                let ref: &Int = &x
                return *ref
            }
        `,
		"test",
	)
	require.NoError(t, err)

	assert.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(42), actual)
}

func TestUnaryDerefSome(t *testing.T) {

	t.Parallel()

	actual, err := CompileAndInvoke(t,
		`
            fun test(): Int? {
                let x = 42
                let ref: &Int = &x
                let optRef = ref as? &Int
                return *optRef
            }
        `,
		"test",
	)
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.NewUnmeteredSomeValueNonCopying(
			interpreter.NewUnmeteredIntValueFromInt64(42),
		),
		actual,
	)
}

func TestUnaryDerefNil(t *testing.T) {

	t.Parallel()

	actual, err := CompileAndInvoke(t,
		`
            fun test(): Int? {
                let optRef: &Int? = nil
                return *optRef
            }
        `,
		"test",
	)
	require.NoError(t, err)

	assert.Equal(t, interpreter.Nil, actual)
}

func TestBinary(t *testing.T) {

	t.Parallel()

	test := func(op string, expected vm.Value) {

		t.Run(op, func(t *testing.T) {

			t.Parallel()

			actual, err := CompileAndInvoke(t,
				fmt.Sprintf(`
                        fun test(): AnyStruct {
                            return 6 %s 4
                        }
                    `,
					op,
				),
				"test",
			)
			require.NoError(t, err)

			ValuesAreEqual(nil, expected, actual)
		})
	}

	tests := map[string]vm.Value{
		"+": interpreter.NewUnmeteredIntValueFromInt64(10),
		"-": interpreter.NewUnmeteredIntValueFromInt64(2),
		"*": interpreter.NewUnmeteredIntValueFromInt64(24),
		"/": interpreter.NewUnmeteredIntValueFromInt64(1),
		"%": interpreter.NewUnmeteredIntValueFromInt64(2),

		"<":  interpreter.FalseValue,
		"<=": interpreter.FalseValue,
		">":  interpreter.TrueValue,
		">=": interpreter.TrueValue,

		"==": interpreter.FalseValue,
		"!=": interpreter.TrueValue,

		"&":  interpreter.NewUnmeteredIntValueFromInt64(4),
		"|":  interpreter.NewUnmeteredIntValueFromInt64(6),
		"^":  interpreter.NewUnmeteredIntValueFromInt64(2),
		"<<": interpreter.NewUnmeteredIntValueFromInt64(96),
		">>": interpreter.NewUnmeteredIntValueFromInt64(0),
	}

	for op, value := range tests {
		test(op, value)
	}
}

func TestForce(t *testing.T) {

	t.Parallel()

	t.Run("non-nil", func(t *testing.T) {
		t.Parallel()

		actual, err := CompileAndInvoke(t,
			`
                fun test(x: Int?): Int {
                    return x!
                }
            `,
			"test",
			interpreter.NewUnmeteredSomeValueNonCopying(
				interpreter.NewUnmeteredIntValueFromInt64(42),
			),
		)

		require.NoError(t, err)
		assert.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(42), actual)
	})

	t.Run("non-nil, AnyStruct", func(t *testing.T) {
		t.Parallel()

		actual, err := CompileAndInvoke(t,
			`
                fun test(x: Int?): AnyStruct {
                    let y: AnyStruct = x
                    return y!
                }
            `,
			"test",
			interpreter.NewUnmeteredSomeValueNonCopying(
				interpreter.NewUnmeteredIntValueFromInt64(42),
			),
		)

		require.NoError(t, err)
		assert.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(42), actual)
	})

	t.Run("nil", func(t *testing.T) {
		t.Parallel()

		_, err := CompileAndInvoke(t,
			`
                fun test(x: Int?): Int {
                    return x!
                }
            `,
			"test",
			interpreter.Nil,
		)
		RequireError(t, err)

		var forceNilError *interpreter.ForceNilError
		require.ErrorAs(t, err, &forceNilError)
	})

	t.Run("nil, AnyStruct", func(t *testing.T) {
		t.Parallel()

		_, err := CompileAndInvoke(t,
			`
                fun test(x: Int?): AnyStruct {
                    let y: AnyStruct = x
                    return y!
                }
            `,
			"test",
			interpreter.Nil,
		)
		RequireError(t, err)
		var forceNilError *interpreter.ForceNilError
		require.ErrorAs(t, err, &forceNilError)
	})

	t.Run("non-optional", func(t *testing.T) {
		t.Parallel()

		actual, err := CompileAndInvoke(t,
			`
                fun test(x: Int): Int {
                    return x!
                }
            `,
			"test",
			interpreter.NewUnmeteredIntValueFromInt64(42),
		)
		require.NoError(t, err)
		assert.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(42), actual)
	})

	t.Run("non-optional, AnyStruct", func(t *testing.T) {
		t.Parallel()

		actual, err := CompileAndInvoke(t,
			`
                fun test(x: Int): AnyStruct {
                    let y: AnyStruct = x
                    return y!
                }
            `,
			"test",
			interpreter.NewUnmeteredIntValueFromInt64(42),
		)
		require.NoError(t, err)
		assert.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(42), actual)
	})

}

func TestTypeConstructor(t *testing.T) {
	t.Parallel()

	t.Run("simple type", func(t *testing.T) {
		t.Parallel()

		actual, err := CompileAndInvoke(t,
			`
                fun test(): Type {
                    return Type<Int>()
                }
            `,
			"test",
		)
		require.NoError(t, err)
		assert.Equal(
			t,
			interpreter.NewTypeValue(nil, interpreter.PrimitiveStaticTypeInt),
			actual,
		)
	})

	t.Run("user defined type", func(t *testing.T) {
		t.Parallel()

		actual, err := CompileAndInvoke(t,
			`
                struct Foo{}
                fun test(): Type {
                    return Type<Foo>()
                }
            `,
			"test",
		)
		require.NoError(t, err)
		assert.Equal(
			t,
			interpreter.NewTypeValue(
				nil,
				interpreter.NewCompositeStaticTypeComputeTypeID(
					nil,
					TestLocation,
					"Foo",
				),
			),
			actual,
		)
	})
}

func TestTypeConversions(t *testing.T) {
	t.Parallel()

	t.Run("address", func(t *testing.T) {
		t.Parallel()

		actual, err := CompileAndInvoke(t,
			`
                fun test(): Address {
                    return Address(0x2)
                }
            `,
			"test",
		)
		require.NoError(t, err)
		assert.Equal(
			t,
			interpreter.AddressValue{
				0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x2,
			},
			actual,
		)
	})

	t.Run("Int", func(t *testing.T) {
		t.Parallel()

		actual, err := CompileAndInvoke(t,
			`
                fun test(): Int {
                    var v: Int64 = 5
                    return Int(v)
                }
            `,
			"test",
		)
		require.NoError(t, err)
		assert.Equal(
			t,
			interpreter.NewUnmeteredIntValueFromInt64(5),
			actual,
		)
	})
}

func TestReturnStatements(t *testing.T) {

	t.Parallel()

	t.Run("conditional return", func(t *testing.T) {
		t.Parallel()

		actual, err := CompileAndInvoke(t,
			`
              fun test(a: Bool): Int {
                  if a {
                      return 1
                  }
                  return 2
              }
            `,
			"test",
			interpreter.TrueValue,
		)

		require.NoError(t, err)
		assert.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(1), actual)
	})

	t.Run("conditional return with post condition", func(t *testing.T) {
		t.Parallel()

		actual, logs, err := CompileAndInvokeWithConditionLogs(t,
			`
              fun test(a: Bool): Int {
                  post {
                      conditionLog("post condition executed")
                  }

                  if a {
                      return 1
                  }

                  if a {
                      // some statements, just to increase the number
                      // of statements inside the nested block
                      var b = 1
                      var c = 2
                      var d = 3
                      conditionLog("second condition reached 1")
                      conditionLog("second condition reached 2")
                  }

                  return 2
              }
            `,
			"test",
			interpreter.TrueValue,
		)

		require.NoError(t, err)
		assert.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(1), actual)

		require.Equal(
			t,
			[]string{
				"\"post condition executed\"",
			},
			logs,
		)
	})

	t.Run("conditional return with post condition in initializer", func(t *testing.T) {
		t.Parallel()

		actual, logs, err := CompileAndInvokeWithConditionLogs(t,
			`
              struct Foo {
                  var i: Int
                  init(_ a: Bool) {
                      post {
                          conditionLog("post condition executed")
                      }
                      if a {
                          self.i = 5
                          return
                      } else {
                          self.i = 8
                      }
                  }
              }

              fun test(a: Bool): Int {
                  var foo = Foo(a)
                  return foo.i
              }
            `,
			"test",
			interpreter.TrueValue,
		)

		require.NoError(t, err)
		assert.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(5), actual)

		require.Equal(
			t,
			[]string{
				"\"post condition executed\"",
			},
			logs,
		)
	})
}

func TestFunctionExpression(t *testing.T) {

	t.Parallel()

	actual, err := CompileAndInvoke(t,
		`
          fun test(): Int {
              let addOne = fun(_ x: Int): Int {
                  return x + 1
              }
              let x = 2
              return x + addOne(3)
          }
        `,
		"test",
	)
	require.NoError(t, err)

	require.NoError(t, err)
	assert.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(6), actual)
}

func TestInnerFunction(t *testing.T) {

	t.Parallel()

	actual, err := CompileAndInvoke(t,
		`
          fun test(): Int {
              fun addOne(_ x: Int): Int {
                  return x + 1
              }
              let x = 2
              return x + addOne(3)
          }
        `,
		"test",
	)
	require.NoError(t, err)
	assert.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(6), actual)
}

func TestContractAccount(t *testing.T) {
	t.Parallel()

	importLocation := common.AddressLocation{
		Address: common.MustBytesToAddress([]byte{0x1}),
		Name:    "C",
	}
	importedChecker, err := ParseAndCheckWithOptions(t,
		`
          contract C {
              fun test(): Address {
                  return self.account.address
              }
          }
        `,
		ParseAndCheckOptions{
			Location: importLocation,
		},
	)
	require.NoError(t, err)

	importCompiler := compiler.NewInstructionCompiler(
		interpreter.ProgramFromChecker(importedChecker),
		importedChecker.Location,
	)
	importedProgram := importCompiler.Compile()

	_, importedContractValue := initializeContract(
		t,
		importLocation,
		importedProgram,
		nil,
	)

	checker, err := ParseAndCheckWithOptions(t,
		`
          import C from 0x1

          fun test(): Address {
              return C.test()
          }
        `,
		ParseAndCheckOptions{
			CheckerConfig: &sema.Config{
				LocationHandler: SingleIdentifierLocationResolver(t),
				ImportHandler: func(_ *sema.Checker, location common.Location, _ ast.Range) (sema.Import, error) {
					require.Equal(t, importLocation, location)
					return sema.ElaborationImport{
						Elaboration: importedChecker.Elaboration,
					}, nil
				},
			},
		},
	)
	require.NoError(t, err)

	comp := compiler.NewInstructionCompiler(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
	)
	comp.Config.ImportHandler = func(location common.Location) *bbq.InstructionProgram {
		require.Equal(t, importLocation, location)
		return importedProgram
	}

	program := comp.Compile()

	addressValue := interpreter.NewUnmeteredAddressValueFromBytes([]byte{0x1})

	vmConfig := vm.NewConfig(NewUnmeteredInMemoryStorage())
	vmConfig.ImportHandler = func(location common.Location) *bbq.InstructionProgram {
		require.Equal(t, importLocation, location)
		return importedProgram
	}
	vmConfig.ContractValueHandler = func(_ *vm.Context, location common.Location) *interpreter.CompositeValue {
		require.Equal(t, importLocation, location)
		return importedContractValue
	}
	vmConfig.CompositeTypeHandler = func(location common.Location, typeID interpreter.TypeID) *sema.CompositeType {
		require.Equal(t, importLocation, location)
		elaboration := importedChecker.Elaboration
		return elaboration.CompositeType(typeID)
	}
	vmConfig.InterfaceTypeHandler = func(location common.Location, typeID interpreter.TypeID) *sema.InterfaceType {
		require.Equal(t, importLocation, location)
		elaboration := importedChecker.Elaboration
		return elaboration.InterfaceType(typeID)
	}

	vmConfig.InjectedCompositeFieldsHandler = func(
		context interpreter.AccountCreationContext,
		_ common.Location,
		_ string,
		_ common.CompositeKind,
	) map[string]interpreter.Value {

		accountRef := stdlib.NewAccountReferenceValue(
			context,
			nil,
			addressValue,
			interpreter.FullyEntitledAccountAccess,
		)

		return map[string]interpreter.Value{
			sema.ContractAccountFieldName: accountRef,
		}
	}

	vmInstance := vm.NewVM(scriptLocation(), program, vmConfig)

	result, err := vmInstance.InvokeExternally("test")
	require.NoError(t, err)
	require.Equal(t, 0, vmInstance.StackSize())

	require.Equal(
		t,
		interpreter.NewUnmeteredAddressValueFromBytes([]byte{0x1}),
		result,
	)
}

func TestResourceOwner(t *testing.T) {
	t.Parallel()

	importLocation := common.AddressLocation{
		Address: common.MustBytesToAddress([]byte{0x1}),
		Name:    "C",
	}
	importedChecker, err := ParseAndCheckWithOptions(t,
		`
          contract C {

              resource R {}

              fun test(): Address {
                  let r <- create R()
                  let path = /storage/r
                  self.account.storage.save(<- r, to: path)
                  let rRef = self.account.storage.borrow<&R>(from: path)!
                  return rRef.owner!.address
              }
          }
        `,
		ParseAndCheckOptions{
			Location: importLocation,
		},
	)
	require.NoError(t, err)

	importCompiler := compiler.NewInstructionCompiler(
		interpreter.ProgramFromChecker(importedChecker),
		importedChecker.Location,
	)
	importedProgram := importCompiler.Compile()

	vmConfig := vm.NewConfig(NewUnmeteredInMemoryStorage())

	_, importedContractValue := initializeContract(
		t,
		importLocation,
		importedProgram,
		vmConfig,
	)

	checker, err := ParseAndCheckWithOptions(t,
		`
          import C from 0x1

          fun test(): Address {
              return C.test()
          }
        `,
		ParseAndCheckOptions{
			CheckerConfig: &sema.Config{
				LocationHandler: SingleIdentifierLocationResolver(t),
				ImportHandler: func(_ *sema.Checker, location common.Location, _ ast.Range) (sema.Import, error) {
					require.Equal(t, importLocation, location)
					return sema.ElaborationImport{
						Elaboration: importedChecker.Elaboration,
					}, nil
				},
			},
		},
	)
	require.NoError(t, err)

	comp := compiler.NewInstructionCompiler(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
	)
	comp.Config.ImportHandler = func(location common.Location) *bbq.InstructionProgram {
		require.Equal(t, importLocation, location)
		return importedProgram
	}

	program := comp.Compile()

	addressValue := interpreter.NewUnmeteredAddressValueFromBytes([]byte{0x1})

	var uuid uint64 = 42

	vmConfig.ImportHandler = func(location common.Location) *bbq.InstructionProgram {
		require.Equal(t, importLocation, location)
		return importedProgram
	}
	vmConfig.ContractValueHandler = func(_ *vm.Context, location common.Location) *interpreter.CompositeValue {
		require.Equal(t, importLocation, location)
		return importedContractValue
	}
	vmConfig.CompositeTypeHandler = func(location common.Location, typeID interpreter.TypeID) *sema.CompositeType {
		require.Equal(t, importLocation, location)
		elaboration := importedChecker.Elaboration
		return elaboration.CompositeType(typeID)
	}
	vmConfig.InterfaceTypeHandler = func(location common.Location, typeID interpreter.TypeID) *sema.InterfaceType {
		require.Equal(t, importLocation, location)
		elaboration := importedChecker.Elaboration
		return elaboration.InterfaceType(typeID)
	}

	vmConfig.UUIDHandler = func() (uint64, error) {
		uuid++
		return uuid, nil
	}

	vmConfig.InjectedCompositeFieldsHandler = func(
		context interpreter.AccountCreationContext,
		_ common.Location,
		_ string,
		_ common.CompositeKind,
	) map[string]interpreter.Value {

		accountRef := stdlib.NewAccountReferenceValue(
			context,
			nil,
			addressValue,
			interpreter.FullyEntitledAccountAccess,
		)

		return map[string]interpreter.Value{
			sema.ContractAccountFieldName: accountRef,
		}
	}

	vmConfig.AccountHandlerFunc = func(
		context interpreter.AccountCreationContext,
		address interpreter.AddressValue,
	) interpreter.Value {
		return stdlib.NewAccountValue(context, nil, address)
	}

	vmInstance := vm.NewVM(scriptLocation(), program, vmConfig)

	result, err := vmInstance.InvokeExternally("test")
	require.NoError(t, err)
	require.Equal(t, 0, vmInstance.StackSize())

	require.Equal(
		t,
		interpreter.NewUnmeteredAddressValueFromBytes([]byte{0x1}),
		result,
	)
}

func TestResourceUUID(t *testing.T) {
	t.Parallel()

	importLocation := common.AddressLocation{
		Address: common.MustBytesToAddress([]byte{0x1}),
		Name:    "C",
	}
	importedChecker, err := ParseAndCheckWithOptions(t,
		`
          contract C {

              resource R {}

              fun test(): UInt64 {
                  let r <- create R()
                  let uuid = r.uuid
                  destroy r
                  return uuid
              }
          }
        `,
		ParseAndCheckOptions{
			Location: importLocation,
		},
	)
	require.NoError(t, err)

	importCompiler := compiler.NewInstructionCompiler(
		interpreter.ProgramFromChecker(importedChecker),
		importedChecker.Location,
	)
	importedProgram := importCompiler.Compile()

	_, importedContractValue := initializeContract(
		t,
		importLocation,
		importedProgram,
		nil,
	)

	checker, err := ParseAndCheckWithOptions(t,
		`
          import C from 0x1

          fun test(): UInt64 {
              return C.test()
          }
        `,
		ParseAndCheckOptions{
			CheckerConfig: &sema.Config{
				LocationHandler: SingleIdentifierLocationResolver(t),
				ImportHandler: func(_ *sema.Checker, location common.Location, _ ast.Range) (sema.Import, error) {
					require.Equal(t, importLocation, location)
					return sema.ElaborationImport{
						Elaboration: importedChecker.Elaboration,
					}, nil
				},
			},
		},
	)
	require.NoError(t, err)

	comp := compiler.NewInstructionCompiler(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
	)
	comp.Config.ImportHandler = func(location common.Location) *bbq.InstructionProgram {
		require.Equal(t, importLocation, location)
		return importedProgram
	}

	program := comp.Compile()

	var uuid uint64 = 42

	vmConfig := vm.NewConfig(NewUnmeteredInMemoryStorage())
	vmConfig.ImportHandler = func(location common.Location) *bbq.InstructionProgram {
		require.Equal(t, importLocation, location)
		return importedProgram
	}
	vmConfig.ContractValueHandler = func(_ *vm.Context, location common.Location) *interpreter.CompositeValue {
		require.Equal(t, importLocation, location)
		return importedContractValue
	}
	vmConfig.CompositeTypeHandler = func(location common.Location, typeID interpreter.TypeID) *sema.CompositeType {
		require.Equal(t, importLocation, location)
		elaboration := importedChecker.Elaboration
		return elaboration.CompositeType(typeID)
	}
	vmConfig.InterfaceTypeHandler = func(location common.Location, typeID interpreter.TypeID) *sema.InterfaceType {
		require.Equal(t, importLocation, location)
		elaboration := importedChecker.Elaboration
		return elaboration.InterfaceType(typeID)
	}

	vmConfig.UUIDHandler = func() (uint64, error) {
		uuid++
		return uuid, nil
	}

	vmConfig.AccountHandlerFunc = func(
		context interpreter.AccountCreationContext,
		address interpreter.AddressValue,
	) interpreter.Value {
		return stdlib.NewAccountValue(context, nil, address)
	}

	vmInstance := vm.NewVM(scriptLocation(), program, vmConfig)

	result, err := vmInstance.InvokeExternally("test")
	require.NoError(t, err)
	require.Equal(t, 0, vmInstance.StackSize())

	require.Equal(
		t,
		interpreter.NewUnmeteredUInt64Value(43),
		result,
	)
}

func TestUnclosedUpvalue(t *testing.T) {

	t.Parallel()

	actual, err := CompileAndInvoke(t,
		`
          fun test(): Int {
              let x = 1
              fun addToX(_ y: Int): Int {
                  return x + y
              }
              return addToX(2)
          }
        `,
		"test",
	)
	require.NoError(t, err)
	assert.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(3), actual)
}

func TestUnclosedUpvalueNested(t *testing.T) {

	t.Parallel()

	actual, err := CompileAndInvoke(t,
		`
          fun test(): Int {
              let x = 1
              fun middle(): Int {
                  fun inner(): Int {
                      let y = 2
                      return x + y
                  }
                  return inner()
              }
              return middle()
          }
        `,
		"test",
	)
	require.NoError(t, err)
	assert.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(3), actual)
}

func TestUnclosedUpvalueDeeplyNested(t *testing.T) {

	t.Parallel()

	actual, err := CompileAndInvoke(t,
		`
          fun test(): Int {
              let a = 1
              let b = 2
              fun middle(): Int {
                  let c = 3
                  let d = 4
                  fun inner(): Int {
                      let e = 5
                      let f = 6
                      return f + e + d + b + c + a
                  }
                  return inner()
              }
              return middle()
          }
        `,
		"test",
	)
	require.NoError(t, err)
	assert.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(21), actual)
}

func TestUnclosedUpvalueAssignment(t *testing.T) {

	t.Parallel()

	actual, err := CompileAndInvoke(t,
		`
          fun test(): Int {
              var x = 1
              fun addToX(_ y: Int) {
                  x = x + y
              }
              addToX(2)
              return x
          }
        `,
		"test",
	)
	require.NoError(t, err)
	assert.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(3), actual)
}

func TestUnclosedUpvalueAssignment2(t *testing.T) {

	t.Parallel()

	actual, err := CompileAndInvoke(t,
		`
          fun test(): Int {
              var x = 1
              fun addToX(_ y: Int) {
                  x = x + y
              }
              addToX(2)
              addToX(2)
              return x
          }
        `,
		"test",
	)
	require.NoError(t, err)
	assert.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(5), actual)
}

func TestClosedUpvalue(t *testing.T) {

	t.Parallel()

	actual, err := CompileAndInvoke(t,
		`
          fun new(): fun(Int): Int {
              let x = 1
              fun addToX(_ y: Int): Int {
                  return x + y
              }
              return addToX
          }

          fun test(): Int {
              let f = new()
              return f(1) + f(2)
          }
        `,
		"test",
	)
	require.NoError(t, err)
	assert.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(5), actual)
}

func TestClosedUpvalueVariableAssignmentBeforeReturn(t *testing.T) {

	t.Parallel()

	actual, err := CompileAndInvoke(t,
		`
          fun new(): fun(Int): Int {
              var x = 1
              fun addToX(_ y: Int): Int {
                  return x + y
              }
              x = 10
              return addToX
          }

          fun test(): Int {
              let f = new()
              return f(1) + f(2)
          }
        `,
		"test",
	)
	require.NoError(t, err)
	assert.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(23), actual)
}

func TestClosedUpvalueAssignment(t *testing.T) {

	t.Parallel()

	actual, err := CompileAndInvoke(t,
		`
          fun new(): fun(Int): Int {
              var x = 1
              fun addToX(_ y: Int): Int {
                  x = x + y
                  return x
              }
              return addToX
          }

          fun test(): Int {
              let f = new()
              return f(1) + f(2)
          }
        `,
		"test",
	)
	require.NoError(t, err)
	assert.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(6), actual)
}

func TestCounter(t *testing.T) {

	t.Parallel()

	actual, err := CompileAndInvoke(t,
		`
          fun newCounter(): fun(): Int {
              var count = 0
              return fun(): Int {
                  count = count + 1
                  return count
              }
          }

          fun test(): Int {
              let counter = newCounter()
              return counter() + counter() + counter()
          }
        `,
		"test",
	)
	require.NoError(t, err)
	assert.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(6), actual)
}

func TestCounters(t *testing.T) {

	t.Parallel()

	actual, err := CompileAndInvoke(t,
		`
          fun newCounter(): fun(): Int {
              var count = 0
              return fun(): Int {
                  count = count + 1
                  return count
              }
          }

          fun test(): Int {
              let counter1 = newCounter()
              let counter2 = newCounter()
              return counter1() + counter2()
          }
        `,
		"test",
	)
	require.NoError(t, err)
	assert.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(2), actual)
}

func TestCounterWithInitialization(t *testing.T) {

	t.Parallel()

	actual, err := CompileAndInvoke(t,
		`
          fun newCounter(): fun(): Int {
              var count = 0
              let res = fun(): Int {
                  count = count + 1
                  return count
              }
              res()
              return res
          }

          fun test(): Int {
              let counter1 = newCounter()
              let counter2 = newCounter()
              return counter1() + counter2()
          }
        `,
		"test",
	)
	require.NoError(t, err)
	assert.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(4), actual)
}

func TestContractClosure(t *testing.T) {

	t.Parallel()

	importLocation := common.AddressLocation{
		Address: common.MustBytesToAddress([]byte{0x1}),
		Name:    "Counter",
	}
	importedChecker, err := ParseAndCheckWithOptions(t,
		`
          contract Counter {
              fun newCounter(): fun(): Int {
                  var count = 0
                  return fun(): Int {
                      count = count + 1
                      return count
                  }
              }
          }
        `,
		ParseAndCheckOptions{
			Location: importLocation,
		},
	)
	require.NoError(t, err)

	importCompiler := compiler.NewInstructionCompiler(
		interpreter.ProgramFromChecker(importedChecker),
		importedChecker.Location,
	)
	importedProgram := importCompiler.Compile()

	_, importedContractValue := initializeContract(
		t,
		importLocation,
		importedProgram,
		nil,
	)

	activation := sema.NewVariableActivation(sema.BaseValueActivation)
	activation.DeclareValue(stdlib.VMPanicFunction)

	checker, err := ParseAndCheckWithOptions(t,
		`
          import Counter from 0x1

          fun test(): Int {
              let counter1 = Counter.newCounter()
              let counter2 = Counter.newCounter()

              if counter1() != 1 { panic("first count wrong") }
              if counter1() != 2 { panic("second count wrong") }
              if counter2() != 1 { panic("third count wrong") }
              if counter2() != 2 { panic("fourth count wrong") }
              if counter1() != 3 { panic("fifth count wrong") }
              if counter2() != 3 { panic("sixth count wrong") }
              if counter2() != 4 { panic("seventh count wrong") }

              return counter1() + counter2()
          }
        `,
		ParseAndCheckOptions{
			CheckerConfig: &sema.Config{
				LocationHandler: SingleIdentifierLocationResolver(t),
				ImportHandler: func(_ *sema.Checker, location common.Location, _ ast.Range) (sema.Import, error) {
					require.Equal(t, importLocation, location)
					return sema.ElaborationImport{
						Elaboration: importedChecker.Elaboration,
					}, nil
				},
				BaseValueActivationHandler: func(location common.Location) *sema.VariableActivation {
					return activation
				},
			},
		},
	)
	require.NoError(t, err)

	compConfig := &compiler.Config{
		LocationHandler: SingleIdentifierLocationResolver(t),
		ImportHandler: func(location common.Location) *bbq.InstructionProgram {
			require.Equal(t, importLocation, location)
			return importedProgram
		},
		BuiltinGlobalsProvider: CompilerDefaultBuiltinGlobalsWithDefaultsAndPanic,
	}

	comp := compiler.NewInstructionCompilerWithConfig(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
		compConfig,
	)

	program := comp.Compile()

	vmConfig := vm.NewConfig(NewUnmeteredInMemoryStorage())
	vmConfig.ImportHandler = func(location common.Location) *bbq.InstructionProgram {
		require.Equal(t, importLocation, location)
		return importedProgram
	}
	vmConfig.ContractValueHandler = func(_ *vm.Context, location common.Location) *interpreter.CompositeValue {
		require.Equal(t, importLocation, location)
		return importedContractValue
	}
	vmConfig.CompositeTypeHandler = func(location common.Location, typeID interpreter.TypeID) *sema.CompositeType {
		require.Equal(t, importLocation, location)
		elaboration := importedChecker.Elaboration
		return elaboration.CompositeType(typeID)
	}
	vmConfig.InterfaceTypeHandler = func(location common.Location, typeID interpreter.TypeID) *sema.InterfaceType {
		require.Equal(t, importLocation, location)
		elaboration := importedChecker.Elaboration
		return elaboration.InterfaceType(typeID)
	}
	vmConfig.BuiltinGlobalsProvider = VMBuiltinGlobalsProviderWithDefaultsAndPanic

	vmInstance := vm.NewVM(scriptLocation(), program, vmConfig)

	result, err := vmInstance.InvokeExternally("test")
	require.NoError(t, err)
	require.Equal(t, 0, vmInstance.StackSize())
	assert.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(9), result)
}

func TestCommonBuiltinTypeBoundFunctions(t *testing.T) {

	t.Parallel()

	t.Run("getType", func(t *testing.T) {

		t.Parallel()

		t.Run("int32", func(t *testing.T) {

			t.Parallel()

			actual, err := CompileAndInvoke(t,
				`
                    struct S {}

                    fun test(): Type {
                        let i: Int32 = 6
                        return i.getType()
                    }
                `,
				"test",
			)
			require.NoError(t, err)
			assert.Equal(
				t,
				interpreter.NewUnmeteredTypeValue(
					interpreter.PrimitiveStaticTypeInt32,
				),
				actual,
			)
		})

		t.Run("struct", func(t *testing.T) {

			t.Parallel()

			actual, err := CompileAndInvoke(t,
				`
                    struct S {}

                    fun test(): Type {
                        let s = S()
                        return s.getType()
                    }
                `,
				"test",
			)
			require.NoError(t, err)
			assert.Equal(
				t,
				interpreter.NewUnmeteredTypeValue(
					interpreter.NewCompositeStaticTypeComputeTypeID(
						nil,
						TestLocation,
						"S",
					),
				),
				actual,
			)
		})

		t.Run("struct interface", func(t *testing.T) {

			t.Parallel()

			actual, err := CompileAndInvoke(t,
				`
                    struct interface I {}
                    struct S: I {}

                    fun test(): Type {
                        let i: {I} = S()
                        return i.getType()
                    }
                `,
				"test",
			)
			require.NoError(t, err)
			assert.Equal(
				t,
				interpreter.NewUnmeteredTypeValue(
					interpreter.NewCompositeStaticTypeComputeTypeID(
						nil,
						TestLocation,
						"S",
					),
				),
				actual,
			)
		})
	})

	t.Run("getIsInstance", func(t *testing.T) {

		t.Parallel()

		t.Run("int32, pass", func(t *testing.T) {

			t.Parallel()

			actual, err := CompileAndInvoke(t,
				`
                    struct S {}

                    fun test(): Bool {
                        let i: Int32 = 6
                        return i.isInstance(Type<Int32>())
                    }
                `,
				"test",
			)
			require.NoError(t, err)
			assert.Equal(
				t,
				interpreter.BoolValue(true),
				actual,
			)
		})

		t.Run("int32, fail", func(t *testing.T) {

			t.Parallel()

			actual, err := CompileAndInvoke(t,
				`
                    struct S {}

                    fun test(): Bool {
                        let i: Int32 = 6
                        return i.isInstance(Type<Int64>())
                    }
                `,
				"test",
			)
			require.NoError(t, err)
			assert.Equal(
				t,
				interpreter.BoolValue(false),
				actual,
			)
		})

		t.Run("struct, pass", func(t *testing.T) {

			t.Parallel()

			actual, err := CompileAndInvoke(t,
				`
                    struct S {}

                    fun test(): Bool {
                        let s = S()
                        return s.isInstance(Type<S>())
                    }
                `,
				"test",
			)
			require.NoError(t, err)
			assert.Equal(
				t,
				interpreter.BoolValue(true),
				actual,
			)
		})

		t.Run("struct, fail", func(t *testing.T) {

			t.Parallel()

			actual, err := CompileAndInvoke(t,
				`
                    struct S1 {}
                    struct S2 {}

                    fun test(): Bool {
                        let s1 = S1()
                        return s1.isInstance(Type<S2>())
                    }
                `,
				"test",
			)
			require.NoError(t, err)
			assert.Equal(
				t,
				interpreter.BoolValue(false),
				actual,
			)
		})
	})
}

func TestEmitInContract(t *testing.T) {

	t.Parallel()

	t.Run("emit", func(t *testing.T) {

		t.Parallel()

		storage := NewUnmeteredInMemoryStorage()

		programs := map[common.Location]*CompiledProgram{}
		contractValues := map[common.Location]*interpreter.CompositeValue{}

		vmConfig := vm.NewConfig(storage)
		vmConfig.ImportHandler = func(location common.Location) *bbq.InstructionProgram {
			program, ok := programs[location]
			if !ok {
				assert.FailNow(t, "invalid location")
			}
			return program.Program
		}
		vmConfig.ContractValueHandler = func(_ *vm.Context, location common.Location) *interpreter.CompositeValue {
			contractValue, ok := contractValues[location]
			if !ok {
				assert.FailNow(t, "invalid location")
			}
			return contractValue
		}
		vmConfig.CompositeTypeHandler = CompiledProgramsCompositeTypeLoader(programs)
		vmConfig.InterfaceTypeHandler = CompiledProgramsInterfaceTypeLoader(programs)
		vmConfig.EntitlementTypeHandler = CompiledProgramsEntitlementTypeLoader(programs)
		vmConfig.EntitlementMapTypeHandler = CompiledProgramsEntitlementMapTypeLoader(programs)

		var uuid uint64 = 42
		vmConfig.UUIDHandler = func() (uint64, error) {
			uuid++
			return uuid, nil
		}

		contractsAddress := common.MustBytesToAddress([]byte{0x1})

		cLocation := common.NewAddressLocation(nil, contractsAddress, "C")

		// Deploy contract with the implementation

		cContract := `
            contract C {

                event TestEvent()

                resource Vault {
                    var balance: Int

                    init(balance: Int) {
                        self.balance = balance
                    }

                    fun getBalance(): Int {
                        pre { emit TestEvent() }
                        return self.balance
                    }
                }

                fun createVault(balance: Int): @Vault {
                    return <- create Vault(balance: balance)
                }
            }`

		cProgram := ParseCheckAndCompile(t, cContract, cLocation, programs)

		_, cContractValue := initializeContract(
			t,
			cLocation,
			cProgram,
			vmConfig,
		)
		contractValues[cLocation] = cContractValue

		// Run script

		code := fmt.Sprintf(
			`
              import C from %[1]s

              fun main(): Int {
                 var vault <- C.createVault(balance: 10)
                 var balance = vault.getBalance()
                 destroy vault
                 return balance
              }
            `,
			contractsAddress.HexWithPrefix(),
		)

		eventEmitted := false
		vmConfig.OnEventEmitted = func(
			context interpreter.ValueExportContext,
			eventType *sema.CompositeType,
			eventFields []interpreter.Value,
		) error {
			require.False(t, eventEmitted)
			eventEmitted = true

			assert.Equal(t,
				cLocation.TypeID(nil, "C.TestEvent"),
				eventType.ID(),
			)

			assert.Equal(t,
				[]interpreter.Value{},
				eventFields,
			)

			return nil
		}

		require.False(t, eventEmitted)

		result, err := CompileAndInvokeWithOptions(
			t,
			code,
			"main",
			CompilerAndVMOptions{
				VMConfig: vmConfig,
				Programs: programs,
			},
		)

		require.NoError(t, err)

		require.True(t, eventEmitted)
		require.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(10), result)
	})
}

func TestInheritedConditions(t *testing.T) {

	t.Parallel()

	t.Run("emit in parent", func(t *testing.T) {

		t.Parallel()

		storage := NewUnmeteredInMemoryStorage()

		programs := map[common.Location]*CompiledProgram{}
		contractValues := map[common.Location]*interpreter.CompositeValue{}

		vmConfig := vm.NewConfig(storage)
		vmConfig.ImportHandler = func(location common.Location) *bbq.InstructionProgram {
			program, ok := programs[location]
			if !ok {
				assert.FailNow(t, "invalid location")
			}
			return program.Program
		}
		vmConfig.ContractValueHandler = func(_ *vm.Context, location common.Location) *interpreter.CompositeValue {
			contractValue, ok := contractValues[location]
			if !ok {
				assert.FailNow(t, "invalid location")
			}
			return contractValue
		}
		vmConfig.CompositeTypeHandler = CompiledProgramsCompositeTypeLoader(programs)
		vmConfig.InterfaceTypeHandler = CompiledProgramsInterfaceTypeLoader(programs)
		vmConfig.EntitlementTypeHandler = CompiledProgramsEntitlementTypeLoader(programs)
		vmConfig.EntitlementMapTypeHandler = CompiledProgramsEntitlementMapTypeLoader(programs)

		var uuid uint64 = 42
		vmConfig.UUIDHandler = func() (uint64, error) {
			uuid++
			return uuid, nil
		}

		contractsAddress := common.MustBytesToAddress([]byte{0x1})

		bLocation := common.NewAddressLocation(nil, contractsAddress, "B")
		cLocation := common.NewAddressLocation(nil, contractsAddress, "C")

		// Deploy interface contract

		bContract := `
          contract interface B {

              event TestEvent()

              resource interface VaultInterface {
                  var balance: Int

                  fun getBalance(): Int {
                      pre { emit TestEvent() }
                  }
              }
          }
        `

		// Only need to compile
		ParseCheckAndCompile(t, bContract, bLocation, programs)

		// Deploy contract with the implementation

		cContract := fmt.Sprintf(
			`
              import B from %[1]s

              contract C {

                  resource Vault: B.VaultInterface {
                      var balance: Int

                      init(balance: Int) {
                          self.balance = balance
                      }

                      fun getBalance(): Int {
                          return self.balance
                      }
                  }

                  fun createVault(balance: Int): @Vault {
                      return <- create Vault(balance: balance)
                  }
              }
            `,
			contractsAddress.HexWithPrefix(),
		)

		cProgram := ParseCheckAndCompile(t, cContract, cLocation, programs)

		_, cContractValue := initializeContract(
			t,
			cLocation,
			cProgram,
			vmConfig,
		)
		contractValues[cLocation] = cContractValue

		// Run transaction

		tx := fmt.Sprintf(
			`
              import C from %[1]s

              fun main(): Int {
                 var vault <- C.createVault(balance: 10)
                 var balance = vault.getBalance()
                 destroy vault
                 return balance
              }
            `,
			contractsAddress.HexWithPrefix(),
		)

		eventEmitted := false
		vmConfig.OnEventEmitted = func(
			context interpreter.ValueExportContext,
			eventType *sema.CompositeType,
			eventFields []interpreter.Value,
		) error {
			require.False(t, eventEmitted)
			eventEmitted = true

			assert.Equal(t,
				cLocation.TypeID(nil, "B.TestEvent"),
				eventType.ID(),
			)

			assert.Equal(t,
				[]interpreter.Value{},
				eventFields,
			)

			return nil
		}

		require.False(t, eventEmitted)

		result, err := CompileAndInvokeWithOptions(
			t,
			tx,
			"main",
			CompilerAndVMOptions{
				VMConfig: vmConfig,
				Programs: programs,
			},
		)
		require.NoError(t, err)

		require.True(t, eventEmitted)
		require.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(10), result)
	})

	t.Run("emit in grand parent", func(t *testing.T) {

		t.Parallel()

		storage := NewUnmeteredInMemoryStorage()

		programs := map[common.Location]*CompiledProgram{}
		contractValues := map[common.Location]*interpreter.CompositeValue{}

		vmConfig := vm.NewConfig(storage)
		vmConfig.ImportHandler = func(location common.Location) *bbq.InstructionProgram {
			program, ok := programs[location]
			if !ok {
				assert.FailNow(t, "invalid location")
			}
			return program.Program
		}
		vmConfig.ContractValueHandler = func(_ *vm.Context, location common.Location) *interpreter.CompositeValue {
			contractValue, ok := contractValues[location]
			if !ok {
				assert.FailNow(t, "invalid location")
			}
			return contractValue
		}
		vmConfig.CompositeTypeHandler = CompiledProgramsCompositeTypeLoader(programs)
		vmConfig.InterfaceTypeHandler = CompiledProgramsInterfaceTypeLoader(programs)
		vmConfig.EntitlementTypeHandler = CompiledProgramsEntitlementTypeLoader(programs)
		vmConfig.EntitlementMapTypeHandler = CompiledProgramsEntitlementMapTypeLoader(programs)

		var uuid uint64 = 42
		vmConfig.UUIDHandler = func() (uint64, error) {
			uuid++
			return uuid, nil
		}

		contractsAddress := common.MustBytesToAddress([]byte{0x1})

		aLocation := common.NewAddressLocation(nil, contractsAddress, "A")
		bLocation := common.NewAddressLocation(nil, contractsAddress, "B")
		cLocation := common.NewAddressLocation(nil, contractsAddress, "C")

		// Deploy interface contract

		aContract := `
          contract interface A {

              event TestEvent()

              resource interface VaultSuperInterface {
                  var balance: Int

                  fun getBalance(): Int {
                      pre { emit TestEvent() }
                  }
              }
          }
        `

		// Only need to compile
		ParseCheckAndCompile(t, aContract, aLocation, programs)

		// Deploy interface contract

		bContract := fmt.Sprintf(
			`
              import A from %[1]s

              contract interface B: A {
                  resource interface VaultInterface: A.VaultSuperInterface {}
              }
            `,
			contractsAddress.HexWithPrefix(),
		)

		// Only need to compile
		ParseCheckAndCompile(t, bContract, bLocation, programs)

		// Deploy contract with the implementation

		cContract := fmt.Sprintf(
			`
              import B from %[1]s

              contract C {

                  resource Vault: B.VaultInterface {
                      var balance: Int

                      init(balance: Int) {
                          self.balance = balance
                      }

                      fun getBalance(): Int {
                          return self.balance
                      }
                  }

                  fun createVault(balance: Int): @Vault {
                      return <- create Vault(balance: balance)
                  }
              }
            `,
			contractsAddress.HexWithPrefix(),
		)

		cProgram := ParseCheckAndCompile(t, cContract, cLocation, programs)

		_, cContractValue := initializeContract(
			t,
			cLocation,
			cProgram,
			vmConfig,
		)
		contractValues[cLocation] = cContractValue

		// Run transaction

		tx := fmt.Sprintf(
			`
              import C from %[1]s

              fun main(): Int {
                 var vault <- C.createVault(balance: 10)
                 var balance = vault.getBalance()
                 destroy vault
                 return balance
              }
            `,
			contractsAddress.HexWithPrefix(),
		)

		eventEmitted := false
		vmConfig.OnEventEmitted = func(
			context interpreter.ValueExportContext,
			eventType *sema.CompositeType,
			eventFields []interpreter.Value,
		) error {
			require.False(t, eventEmitted)
			eventEmitted = true

			assert.Equal(t,
				cLocation.TypeID(nil, "A.TestEvent"),
				eventType.ID(),
			)

			assert.Equal(t,
				[]interpreter.Value{},
				eventFields,
			)

			return nil
		}

		require.False(t, eventEmitted)

		result, err := CompileAndInvokeWithOptionsAndPrograms(
			t,
			tx,
			"main",
			CompilerAndVMOptions{
				VMConfig: vmConfig,
			},
			programs,
		)
		require.NoError(t, err)

		require.True(t, eventEmitted)
		require.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(10), result)
	})

	t.Run("transitive dependency function call in parent", func(t *testing.T) {

		t.Parallel()

		storage := NewUnmeteredInMemoryStorage()

		programs := map[common.Location]*CompiledProgram{}
		contractValues := map[common.Location]*interpreter.CompositeValue{}

		vmConfig := vm.NewConfig(storage)

		vmConfig.ContractValueHandler = func(_ *vm.Context, location common.Location) *interpreter.CompositeValue {
			contractValue, ok := contractValues[location]
			if !ok {
				assert.FailNow(t, "invalid location")
			}
			return contractValue
		}

		var logs []string
		vmConfig.BuiltinGlobalsProvider = NewVMBuiltinGlobalsProviderWithDefaultsPanicAndConditionLog(&logs)

		PrepareVMConfig(t, vmConfig, programs)

		contractsAddress := common.MustBytesToAddress([]byte{0x1})

		aLocation := common.NewAddressLocation(nil, contractsAddress, "A")
		bLocation := common.NewAddressLocation(nil, contractsAddress, "B")
		cLocation := common.NewAddressLocation(nil, contractsAddress, "C")
		dLocation := common.NewAddressLocation(nil, contractsAddress, "D")

		// Deploy contract with a type

		aContract := `
            contract A {
                struct TestStruct {
                    view fun test(): Bool {
                        conditionLog("invoked TestStruct.test()")
                        return true
                    }
                }
            }
        `

		baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
		baseValueActivation.DeclareValue(stdlib.VMPanicFunction)
		baseValueActivation.DeclareValue(newConditionLogFunction(nil))

		compilerConfig := &compiler.Config{
			BuiltinGlobalsProvider: CompilerDefaultBuiltinGlobalsWithDefaultsAndConditionLog,
		}

		aProgram := ParseCheckAndCompileCodeWithOptions(
			t,
			aContract,
			aLocation,
			ParseCheckAndCompileOptions{
				CompilerConfig: compilerConfig,
				ParseAndCheckOptions: &ParseAndCheckOptions{
					CheckerConfig: &sema.Config{
						BaseValueActivationHandler: func(location common.Location) *sema.VariableActivation {
							return baseValueActivation
						},
					},
				},
			},
			programs,
		)

		_, aContractValue := initializeContract(
			t,
			aLocation,
			aProgram,
			vmConfig,
		)
		contractValues[aLocation] = aContractValue

		// Deploy contract interface

		bContract := fmt.Sprintf(
			`
              import A from %[1]s

              contract interface B {

                  resource interface VaultInterface {
                      var balance: Int

                      fun getBalance(): Int {
                          // Call 'A.TestStruct()' which is only available to this contract interface.
                          pre { self.test(A.TestStruct()) }
                      }

                      view fun test(_ a: A.TestStruct): Bool {
                         return a.test()
                      }
                  }
              }
            `,
			contractsAddress.HexWithPrefix(),
		)

		// Only need to compile
		ParseCheckAndCompile(t, bContract, bLocation, programs)

		// Deploy another intermediate contract interface

		cContract := fmt.Sprintf(
			`
              import B from %[1]s

              contract interface C: B {
                  resource interface VaultIntermediateInterface: B.VaultInterface {}
              }
            `,
			contractsAddress.HexWithPrefix(),
		)

		// Only need to compile
		ParseCheckAndCompile(t, cContract, cLocation, programs)

		// Deploy contract with the implementation

		dContract := fmt.Sprintf(
			`
              import C from %[1]s

              contract D: C {

                  resource Vault: C.VaultIntermediateInterface {
                      var balance: Int

                      init(balance: Int) {
                          self.balance = balance
                      }

                      fun getBalance(): Int {
                          // Inherits a function call 'A.TestStruct()' from the grand-parent 'B',
                          // But 'A' is NOT available to this contract (as an import).
                          return self.balance
                      }
                  }

                  fun createVault(balance: Int): @Vault {
                      return <- create Vault(balance: balance)
                  }
              }
            `,
			contractsAddress.HexWithPrefix(),
		)

		dProgram := ParseCheckAndCompile(t, dContract, dLocation, programs)

		_, dContractValue := initializeContract(
			t,
			dLocation,
			dProgram,
			vmConfig,
		)
		contractValues[dLocation] = dContractValue

		// Run transaction

		tx := fmt.Sprintf(
			`
              import D from %[1]s

              fun main(): Int {
                 var vault <- D.createVault(balance: 10)
                 var balance = vault.getBalance()
                 destroy vault
                 return balance
              }
            `,
			contractsAddress.HexWithPrefix(),
		)

		txLocation := NewTransactionLocationGenerator()
		txProgram := ParseCheckAndCompile(t, tx, txLocation(), programs)

		txVM := vm.NewVM(txLocation(), txProgram, vmConfig)
		result, err := txVM.InvokeExternally("main")
		require.NoError(t, err)

		require.Equal(t, 0, txVM.StackSize())
		assert.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(10), result)
		assert.Equal(
			t,
			[]string{"\"invoked TestStruct.test()\""},
			logs,
		)
	})
}

func TestMethodsAsFunctionPointers(t *testing.T) {

	t.Parallel()

	t.Run("composite value, user function", func(t *testing.T) {
		t.Parallel()

		result, err := CompileAndInvoke(
			t,
			`
                struct Test {
                    access(all) fun add(_ a: Int, _ b: Int): Int {
                        return a + b
                    }
                }

                fun test(): Int {
                    let test = Test()
                    var add: (fun(Int, Int): Int) = test.add
                    return add( 1, 3)
                }
            `,
			"test",
		)

		require.NoError(t, err)
		assert.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(4), result)
	})

	t.Run("composite value, builtin function", func(t *testing.T) {
		t.Parallel()

		result, err := CompileAndInvoke(
			t,
			`
                struct Test {}

                fun test(): Bool {
                    let test = Test()
                    var isInstance = test.isInstance
                    return isInstance(Type<Test>())
                }
            `,
			"test",
		)

		require.NoError(t, err)
		assert.Equal(t, interpreter.BoolValue(true), result)
	})

	t.Run("interface function", func(t *testing.T) {
		t.Parallel()

		result, err := CompileAndInvoke(
			t,
			`
                struct interface Interface {
                    access(all) fun add(_ a: Int, _ b: Int): Int {
                        return a + b
                    }
                }

                struct Test: Interface {
                    access(all) fun add(_ a: Int, _ b: Int): Int {
                        return a + b
                    }
                }

                fun test(): Int {
                    let test: {Interface} = Test()
                    var add = test.add
                    return add( 1, 3)
                }
            `,
			"test",
		)

		require.NoError(t, err)
		assert.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(4), result)
	})

	t.Run("primitive value, builtin function", func(t *testing.T) {
		t.Parallel()

		result, err := CompileAndInvoke(
			t,
			`
                fun test(): Bool {
                    let a: Int64 = 5
                    var isInstance = a.isInstance
                    return isInstance(Type<Int32>())
                }
            `,
			"test",
		)

		require.NoError(t, err)
		assert.Equal(t, interpreter.BoolValue(false), result)
	})
}

func TestArrayFunctions(t *testing.T) {

	t.Parallel()

	t.Run("append", func(t *testing.T) {
		t.Parallel()

		result, err := CompileAndInvoke(t,
			`
                fun test(): Type {
                    var array: [UInt8] = [2, 5]
                    var append = array.append
                    return append.getType()
                }
            `,
			"test",
		)

		require.NoError(t, err)
		AssertValuesEqual(
			t,
			nil,
			interpreter.TypeValue{
				Type: interpreter.FunctionStaticType{
					FunctionType: sema.ArrayAppendFunctionType(sema.UInt8Type),
				},
			},
			result,
		)
	})

	t.Run("variable sized reverse", func(t *testing.T) {
		t.Parallel()

		result, err := CompileAndInvoke(t,
			`
                fun test(): Type {
                    var array: [UInt8] = [2, 5]
                    var reverse = array.reverse
                    return reverse.getType()
                }
            `,
			"test",
		)

		require.NoError(t, err)
		AssertValuesEqual(
			t,
			nil,
			interpreter.TypeValue{
				Type: interpreter.FunctionStaticType{
					FunctionType: sema.ArrayReverseFunctionType(
						sema.NewVariableSizedType(
							nil,
							sema.UInt8Type,
						),
					),
				},
			},
			result,
		)
	})

	t.Run("constant sized reverse", func(t *testing.T) {
		t.Parallel()

		result, err := CompileAndInvoke(t,
			`
                fun test(): Type {
                    var array: [UInt8; 2] = [2, 5]
                    var reverse = array.reverse
                    return reverse.getType()
                }
            `,
			"test",
		)

		require.NoError(t, err)
		AssertValuesEqual(
			t,
			nil,
			interpreter.TypeValue{
				Type: interpreter.FunctionStaticType{
					FunctionType: sema.ArrayReverseFunctionType(
						sema.NewConstantSizedType(
							nil,
							sema.UInt8Type,
							2,
						),
					),
				},
			},
			result,
		)
	})
}

func TestInvocationExpressionEvaluationOrder(t *testing.T) {

	t.Parallel()

	t.Run("static function", func(t *testing.T) {
		t.Parallel()

		_, logs, err := CompileAndInvokeWithLogs(t,
			`
              fun test() {
                  getFunction()(getArgument())
              }

              fun getFunction(): (fun(Int)) {
                  log("invoked expression")
                  return fun(_ n: Int) {
                      log("function implementation")
                  }
              }

              fun getArgument(): Int {
                  log("arguments")
                  return 3
              }
            `,
			"test",
		)
		require.NoError(t, err)

		assert.Equal(
			t,
			[]string{
				`"invoked expression"`,
				`"arguments"`,
				`"function implementation"`,
			},
			logs,
		)
	})

	t.Run("object method", func(t *testing.T) {
		t.Parallel()

		_, logs, err := CompileAndInvokeWithLogs(t,
			`
              fun test() {
                  getReceiver().method(getArgument())
              }

              struct Foo {
                  fun method(_ n: Int) {
                      log("method implementation")
                  }
              }

              fun getReceiver(): Foo {
                  log("receiver")
                  return Foo()
              }

              fun getArgument(): Int {
                  log("arguments")
                  return 3
              }
            `,
			"test",
		)
		require.NoError(t, err)

		assert.Equal(
			t,
			[]string{
				`"receiver"`,
				`"arguments"`,
				`"method implementation"`,
			},
			logs,
		)
	})
}

func TestSaturatingArithmetic(t *testing.T) {

	t.Parallel()

	result, err := CompileAndInvoke(t,
		`
          fun test(): UFix64 {
              return (1.0).saturatingSubtract(2.0)
          }
        `,
		"test",
	)
	require.NoError(t, err)

	assert.Equal(
		t,
		interpreter.NewUnmeteredUFix64Value(0),
		result,
	)
}

func TestGlobalVariables(t *testing.T) {

	t.Parallel()

	t.Run("simple", func(t *testing.T) {
		t.Parallel()

		result, err := CompileAndInvoke(t,
			`
              var a = 5

              fun test(): Int {
                  return a
              }
            `,
			"test",
		)
		require.NoError(t, err)

		assert.Equal(
			t,
			interpreter.NewUnmeteredIntValueFromInt64(5),
			result,
		)
	})

	t.Run("referenced another var", func(t *testing.T) {
		t.Parallel()

		result, err := CompileAndInvoke(t,
			`
              var a = 5
              var b = a

              fun test(): Int {
                  return b
              }
            `,
			"test",
		)
		require.NoError(t, err)

		assert.Equal(
			t,
			interpreter.NewUnmeteredIntValueFromInt64(5),
			result,
		)
	})

	t.Run("update referenced var before first use", func(t *testing.T) {
		t.Parallel()

		result, err := CompileAndInvoke(t,
			`
              var a = 5
              var b = a

              fun test(): Int {
                  // Update 'a' before getting 'b'.
                  a = 8

                  // Get 'b' for the fist time.
                  return b
              }
            `,
			"test",
		)
		require.NoError(t, err)

		assert.Equal(
			t,
			interpreter.NewUnmeteredIntValueFromInt64(5),
			result,
		)
	})

	t.Run("update referenced var after first use", func(t *testing.T) {
		t.Parallel()

		result, err := CompileAndInvoke(t,
			`
              var a = 5
              var b = a

              fun test(): Int {
                  // Use 'b' to get the value once.
                  var c = b

                  // Update 'a' before getting 'b' for the first time.
                  a = 8

                  // Get 'b' for a second time.
                  // Should return the initial value.
                  return b
              }
            `,
			"test",
		)
		require.NoError(t, err)

		assert.Equal(
			t,
			interpreter.NewUnmeteredIntValueFromInt64(5),
			result,
		)
	})

	t.Run("overridden local var", func(t *testing.T) {
		t.Parallel()

		result, err := CompileAndInvoke(t,
			`
              var a = 5

              fun test(): Int {
                  var a = 8
                  return getGlobalA()
              }

              fun getGlobalA(): Int {
                  return a
              }
            `,
			"test",
		)
		require.NoError(t, err)

		assert.Equal(
			t,
			interpreter.NewUnmeteredIntValueFromInt64(5),
			result,
		)
	})

	t.Run("forward references", func(t *testing.T) {
		t.Parallel()

		result, logs, err := CompileAndInvokeWithLogs(t,
			`
              var a = initializeA()
              var b = initializeB()

              fun test(): Int {
                  log("invoked test")
                  return a
              }

              fun initializeA(): Int {
                  log("invoked initializeA")

                  // Indirect forward reference to b
                  var c = b

                  log("exiting initializeA")
                  return 5
              }

              fun initializeB(): Int {
                  log("invoked initializeB")
                  return 5
              }
            `,
			"test",
		)
		require.NoError(t, err)

		assert.Equal(
			t,
			interpreter.NewUnmeteredIntValueFromInt64(5),
			result,
		)

		assert.Equal(
			t,
			[]string{
				// Variables must be initialized before calling the function test.
				`"invoked initializeA"`,
				`"invoked initializeB"`,
				`"exiting initializeA"`,

				`"invoked test"`,
			},
			logs,
		)
	})

	t.Run("initialization on program start", func(t *testing.T) {
		t.Parallel()

		var logs []string

		activation := sema.NewVariableActivation(sema.BaseValueActivation)
		activation.DeclareValue(stdlib.VMPanicFunction)
		activation.DeclareValue(stdlib.NewVMLogFunction(nil))

		compilerConfig := &compiler.Config{
			BuiltinGlobalsProvider: CompilerDefaultBuiltinGlobalsWithDefaultsAndLog,
		}
		storage := NewUnmeteredInMemoryStorage()
		vmConfig := vm.NewConfig(storage)

		vmConfig.BuiltinGlobalsProvider = NewVMBuiltinGlobalsProviderWithDefaultsPanicAndLog(&logs)

		// Only prepare, do not invoke anything.

		_, err := CompileAndPrepareToInvoke(t,
			`
              var a = initializeA()
              var b = initializeB()

              fun initializeA(): Int {
                  log("invoked initializeA")

                  // Indirect forward reference to b
                  var c = b

                  log("exiting initializeA")
                  return 5
              }

              fun initializeB(): Int {
                  log("invoked initializeB")
                  return 5
              }
            `,
			CompilerAndVMOptions{
				ParseCheckAndCompileOptions: ParseCheckAndCompileOptions{
					ParseAndCheckOptions: &ParseAndCheckOptions{
						CheckerConfig: &sema.Config{
							LocationHandler: SingleIdentifierLocationResolver(t),
							BaseValueActivationHandler: func(location common.Location) *sema.VariableActivation {
								return activation
							},
						},
					},
					CompilerConfig: compilerConfig,
				},
				VMConfig: vmConfig,
			},
		)
		require.NoError(t, err)

		assert.Equal(
			t,
			[]string{
				// Variables must be initialized before calling the function test.
				`"invoked initializeA"`,
				`"invoked initializeB"`,
				`"exiting initializeA"`,
			},
			logs,
		)
	})
}

func TestUserInvokesNativeFunction(t *testing.T) {

	t.Parallel()

	result, err := CompileAndInvoke(t,
		`
          fun test(): Int? {
              let opt: UInt8? = 1
              return opt.map(Int)
          }
        `,
		"test",
	)
	require.NoError(t, err)
	require.Equal(t,
		interpreter.NewUnmeteredSomeValueNonCopying(
			interpreter.NewUnmeteredIntValueFromInt64(1),
		),
		result,
	)
}

func TestOptionalChaining(t *testing.T) {
	t.Parallel()

	t.Run("nil, field", func(t *testing.T) {
		t.Parallel()

		result, err := CompileAndInvoke(t,
			`
            struct Foo {
                var bar: Int
                init(value: Int) {
                    self.bar = value
                }
            }

            fun test(): Int? {
                let foo: Foo? = nil
                return foo?.bar
            }
            `,
			"test",
		)
		require.NoError(t, err)

		assert.Equal(
			t,
			interpreter.Nil,
			result,
		)
	})

	t.Run("non-nil, field", func(t *testing.T) {
		t.Parallel()

		result, err := CompileAndInvoke(t,
			`
            struct Foo {
                var bar: Int
                init(_ value: Int) {
                    self.bar = value
                }
            }

            fun test(): Int? {
                let foo: Foo? = Foo(5)
                return foo?.bar
            }
            `,
			"test",
		)
		require.NoError(t, err)

		assert.Equal(
			t,
			interpreter.NewUnmeteredSomeValueNonCopying(
				interpreter.NewUnmeteredIntValueFromInt64(5),
			),
			result,
		)
	})

	t.Run("non-nil, nested field", func(t *testing.T) {
		t.Parallel()

		result, err := CompileAndInvoke(t,
			`
            struct Foo {
                var bar: Bar
                init() {
                    self.bar = Bar(5)
                }
            }

            struct Bar {
                var id: Int
                init(_ id: Int) {
                    self.id = id
                }
            }

            fun test(): Int? {
                let foo: Foo? = Foo()
                return foo?.bar?.id
            }
            `,
			"test",
		)
		require.NoError(t, err)

		assert.Equal(
			t,
			interpreter.NewUnmeteredSomeValueNonCopying(
				interpreter.NewUnmeteredIntValueFromInt64(5),
			),
			result,
		)
	})

	t.Run("nil, method", func(t *testing.T) {
		t.Parallel()

		result, err := CompileAndInvoke(t,
			`
            struct Foo {
                fun bar(): Int {
                    return 1
                }
            }

            fun test(): Int? {
                let foo: Foo? = nil
                return foo?.bar()
            }
            `,
			"test",
		)
		require.NoError(t, err)

		assert.Equal(
			t,
			interpreter.Nil,
			result,
		)
	})

	t.Run("non-nil, method", func(t *testing.T) {
		t.Parallel()

		result, err := CompileAndInvoke(t,
			`
            struct Foo {
                fun bar(): Int {
                    return 4
                }
            }

            fun test(): Int? {
                let foo: Foo? = Foo()
                return foo?.bar()
            }
            `,
			"test",
		)
		require.NoError(t, err)

		assert.Equal(
			t,
			interpreter.NewUnmeteredSomeValueNonCopying(
				interpreter.NewUnmeteredIntValueFromInt64(4),
			),
			result,
		)
	})
}

func TestBoundStaticFunction(t *testing.T) {

	t.Parallel()

	result, err := CompileAndInvoke(t,
		`
          fun test(): UInt8? {
              let f = UInt8.fromString
              return f("1")
          }
        `,
		"test",
	)
	require.NoError(t, err)
	require.Equal(t,
		interpreter.NewUnmeteredSomeValueNonCopying(
			interpreter.NewUnmeteredUInt8Value(1),
		),
		result,
	)
}

func TestEnumAccess(t *testing.T) {

	t.Parallel()

	result, err := CompileAndInvoke(t,
		`
            enum Test: UInt8 {
                case a
                case b
                case c
            }

            fun test(): UInt8 {
                return Test.b.rawValue
            }
        `,
		"test",
	)
	require.NoError(t, err)
	assert.Equal(t, interpreter.NewUnmeteredUInt8Value(1), result)
}

func TestEnumLookupSuccess(t *testing.T) {

	t.Parallel()

	result, err := CompileAndInvoke(t,
		`
            enum Test: UInt8 {
                case a
                case b
                case c
            }

            fun test(): AnyStruct {
                return Test(rawValue: 1)
            }
        `,
		"test",
	)
	require.NoError(t, err)
	assert.IsType(t, &interpreter.SomeValue{}, result)
}

func TestEnumLookupFailure(t *testing.T) {

	t.Parallel()

	result, err := CompileAndInvoke(t,
		`
            enum Test: UInt8 {
                case a
                case b
                case c
            }

            fun test(): AnyStruct {
                return Test(rawValue: 5)
            }
        `,
		"test",
	)
	require.NoError(t, err)
	assert.Equal(t, interpreter.Nil, result)
}

func TestNestedEnumLookupFailure(t *testing.T) {

	t.Parallel()

	result, err := CompileAndInvoke(t,
		`
            contract C {
                enum Test: UInt8 {
                    case a
                    case b
                    case c
                }
            }

            fun test(): AnyStruct {
                return C.Test(rawValue: 5)
            }
        `,
		"test",
	)
	require.NoError(t, err)
	assert.Equal(t, interpreter.Nil, result)
}

func TestFunctionInvocationWithOptionalArgs(t *testing.T) {

	t.Parallel()

	const functionName = "foo"

	functionType := &sema.FunctionType{
		Purity:               sema.FunctionPurityView,
		ReturnTypeAnnotation: sema.IntTypeAnnotation,
		Arity:                &sema.Arity{Min: 1, Max: -1},
	}

	activation := sema.NewVariableActivation(sema.BaseValueActivation)
	activation.DeclareValue(stdlib.StandardLibraryValue{
		Name: functionName,
		Type: functionType,
		Kind: common.DeclarationKindFunction,
	})

	compilerConfig := &compiler.Config{
		BuiltinGlobalsProvider: func(_ common.Location) *activations.Activation[compiler.GlobalImport] {
			activation := activations.NewActivation(nil, compiler.DefaultBuiltinGlobals())
			activation.Set(
				functionName,
				compiler.NewGlobalImport(functionName),
			)
			return activation
		},
	}

	functionValue := vm.NewNativeFunctionValue(
		functionName,
		functionType,
		func(
			context interpreter.NativeFunctionContext,
			_ interpreter.TypeArgumentsIterator,
			_ interpreter.ArgumentTypesIterator,
			_ interpreter.Value,
			arguments []vm.Value,
		) vm.Value {
			require.GreaterOrEqual(t, len(arguments), 1)

			require.IsType(t, interpreter.IntValue{}, arguments[0])
			first := arguments[0].(interpreter.IntValue)

			if len(arguments) < 2 {
				return first
			}

			require.IsType(t, interpreter.IntValue{}, arguments[1])
			second := arguments[1].(interpreter.IntValue)

			return first.Plus(context, second)
		},
	)

	vmConfig := vm.NewConfig(NewUnmeteredInMemoryStorage())
	vmConfig.BuiltinGlobalsProvider = func(_ common.Location) *activations.Activation[vm.Variable] {
		activation := activations.NewActivation(nil, vm.DefaultBuiltinGlobals())
		variable := &interpreter.SimpleVariable{}
		variable.InitializeWithValue(functionValue)
		activation.Set(functionName, variable)
		return activation
	}

	result, err := CompileAndInvokeWithOptions(
		t,
		`
          fun test(): Int {
              return foo(2) + foo(5, 11)
          }
        `,
		"test",
		CompilerAndVMOptions{
			ParseCheckAndCompileOptions: ParseCheckAndCompileOptions{
				CompilerConfig: compilerConfig,
				ParseAndCheckOptions: &ParseAndCheckOptions{
					CheckerConfig: &sema.Config{
						BaseValueActivationHandler: func(location common.Location) *sema.VariableActivation {
							return activation
						},
					},
				},
			},
			VMConfig: vmConfig,
		},
	)
	require.NoError(t, err)
	require.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(18), result)
}

type testMemoryGauge struct {
	meter map[common.MemoryKind]uint64
}

func newTestMemoryGauge() *testMemoryGauge {
	return &testMemoryGauge{
		meter: make(map[common.MemoryKind]uint64),
	}
}

func (g *testMemoryGauge) MeterMemory(usage common.MemoryUsage) error {
	g.meter[usage.Kind] += usage.Amount
	return nil
}

func (g *testMemoryGauge) getMemory(kind common.MemoryKind) uint64 {
	return g.meter[kind]
}

type testComputationGauge struct {
	meter map[common.ComputationKind]uint64
}

func newTestComputationGauge() *testComputationGauge {
	return &testComputationGauge{
		meter: make(map[common.ComputationKind]uint64),
	}
}

func (g *testComputationGauge) MeterComputation(usage common.ComputationUsage) error {
	g.meter[usage.Kind] += usage.Intensity
	return nil
}

func (g *testComputationGauge) getComputation(kind common.ComputationKind) uint64 {
	return g.meter[kind]
}

func TestMetering(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, imperativeFib)
	require.NoError(t, err)

	comp := compiler.NewInstructionCompiler(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
	)
	program := comp.Compile()

	memoryGauge := newTestMemoryGauge()
	computationGauge := newTestComputationGauge()

	vmConfig := vm.NewConfig(NewUnmeteredInMemoryStorage())
	vmConfig.MemoryGauge = memoryGauge
	vmConfig.ComputationGauge = computationGauge

	vmInstance := vm.NewVM(scriptLocation(), program, vmConfig)

	result, err := vmInstance.InvokeExternally(
		"fib",
		interpreter.NewUnmeteredIntValueFromInt64(23),
	)
	require.NoError(t, err)
	assert.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(28657), result)
	assert.Equal(t, 0, vmInstance.StackSize())

	assert.Equal(t, uint64(2016), memoryGauge.getMemory(common.MemoryKindBigInt))
	assert.Equal(t, uint64(21), computationGauge.getComputation(common.ComputationKindLoop))
}

func TestSwapIdentifiers(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
        fun test(): [Int] {
            var x = 1
            var y = 2
            x <-> y
            return [x, y]
        }
    `)
	require.NoError(t, err)

	comp := compiler.NewInstructionCompiler(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
	)
	program := comp.Compile()

	vmConfig := vm.NewConfig(NewUnmeteredInMemoryStorage())
	vmInstance := vm.NewVM(TestLocation, program, vmConfig)

	result, err := vmInstance.InvokeExternally("test")
	require.NoError(t, err)

	context := vmInstance.Context()

	AssertValuesEqual(
		t,
		context,
		interpreter.NewArrayValue(
			context,
			interpreter.NewVariableSizedStaticType(nil, interpreter.PrimitiveStaticTypeInt),
			common.ZeroAddress,
			interpreter.NewUnmeteredIntValueFromInt64(2),
			interpreter.NewUnmeteredIntValueFromInt64(1),
		),
		result,
	)
}

func TestSwapMembers(t *testing.T) {

	t.Parallel()

	vmInstance, err := CompileAndPrepareToInvoke(t,
		`
            struct S {
                var x: Int
                var y: Int

                init() {
                    self.x = 1
                    self.y = 2
                    }
            }

            fun test(): [Int] {
                let s = S()
                s.x <-> s.y
                return [s.x, s.y]
            }
        `,
		CompilerAndVMOptions{},
	)
	require.NoError(t, err)

	result, err := vmInstance.InvokeExternally("test")
	require.NoError(t, err)

	context := vmInstance.Context()

	AssertValuesEqual(
		t,
		context,
		interpreter.NewArrayValue(
			context,
			interpreter.NewVariableSizedStaticType(nil, interpreter.PrimitiveStaticTypeInt),
			common.ZeroAddress,
			interpreter.NewUnmeteredIntValueFromInt64(2),
			interpreter.NewUnmeteredIntValueFromInt64(1),
		),
		result,
	)
}

func TestSwapIndex(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
        fun test(): [String] {
            let chars = ["a", "b"]
            chars[0] <-> chars[1]
            return chars
        }
    `)
	require.NoError(t, err)

	comp := compiler.NewInstructionCompiler(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
	)
	program := comp.Compile()

	vmConfig := vm.NewConfig(NewUnmeteredInMemoryStorage())
	vmInstance := vm.NewVM(TestLocation, program, vmConfig)

	result, err := vmInstance.InvokeExternally("test")
	require.NoError(t, err)

	context := vmInstance.Context()

	AssertValuesEqual(
		t,
		context,
		interpreter.NewArrayValue(
			context,
			interpreter.NewVariableSizedStaticType(nil, interpreter.PrimitiveStaticTypeString),
			common.ZeroAddress,
			interpreter.NewUnmeteredStringValue("b"),
			interpreter.NewUnmeteredStringValue("a"),
		),
		result,
	)
}

func TestImplicitBoxing(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
        fun f(_ y: Int?): AnyStruct {
            return y
        }

        fun g(): Int? {
            return 3
        }

        fun test(): [AnyStruct] {
            let x: Int? = 1
            return [x, f(2), g()]
        }
    `)
	require.NoError(t, err)

	comp := compiler.NewInstructionCompiler(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
	)
	program := comp.Compile()

	vmConfig := vm.NewConfig(NewUnmeteredInMemoryStorage())
	vmInstance := vm.NewVM(TestLocation, program, vmConfig)

	result, err := vmInstance.InvokeExternally("test")
	require.NoError(t, err)

	context := vmInstance.Context()

	AssertValuesEqual(
		t,
		context,
		interpreter.NewArrayValue(
			context,
			interpreter.NewVariableSizedStaticType(nil, interpreter.PrimitiveStaticTypeAnyStruct),
			common.ZeroAddress,
			interpreter.NewUnmeteredSomeValueNonCopying(
				interpreter.NewUnmeteredIntValueFromInt64(1),
			),
			interpreter.NewUnmeteredSomeValueNonCopying(
				interpreter.NewUnmeteredIntValueFromInt64(2),
			),
			interpreter.NewUnmeteredSomeValueNonCopying(
				interpreter.NewUnmeteredIntValueFromInt64(3),
			),
		),
		result,
	)
}

func TestGetAuthAccount(t *testing.T) {

	t.Parallel()

	t.Run("available in script", func(t *testing.T) {

		t.Parallel()

		code := `
          fun main(): &Account {
              return getAuthAccount<auth(Storage) &Account>(0x1)
          }
        `

		location := common.ScriptLocation{0x1}

		activation := sema.NewVariableActivation(sema.BaseValueActivation)
		activation.DeclareValue(stdlib.VMPanicFunction)
		activation.DeclareValue(stdlib.NewVMGetAuthAccountFunction(nil))

		compilerConfig := &compiler.Config{
			BuiltinGlobalsProvider: func(_ common.Location) *activations.Activation[compiler.GlobalImport] {
				activation := activations.NewActivation(nil, compiler.DefaultBuiltinGlobals())
				activation.Set(
					stdlib.GetAuthAccountFunctionName,
					compiler.NewGlobalImport(stdlib.GetAuthAccountFunctionName),
				)
				return activation
			},
		}

		vmConfig := vm.NewConfig(NewUnmeteredInMemoryStorage())
		vmConfig.BuiltinGlobalsProvider = func(_ common.Location) *activations.Activation[vm.Variable] {
			activation := activations.NewActivation(nil, vm.DefaultBuiltinGlobals())

			variable := &interpreter.SimpleVariable{}
			variable.InitializeWithValue(stdlib.NewVMGetAuthAccountFunction(&testAccountHandler{}).Value)
			activation.Set(stdlib.GetAuthAccountFunctionName, variable)

			return activation
		}

		result, err := CompileAndInvokeWithOptions(
			t,
			code,
			"main",
			CompilerAndVMOptions{
				ParseCheckAndCompileOptions: ParseCheckAndCompileOptions{
					CompilerConfig: compilerConfig,
					ParseAndCheckOptions: &ParseAndCheckOptions{
						Location: location,
						CheckerConfig: &sema.Config{
							LocationHandler: SingleIdentifierLocationResolver(t),
							BaseValueActivationHandler: func(location common.Location) *sema.VariableActivation {
								return activation
							},
						},
					},
				},
				VMConfig: vmConfig,
			},
		)
		require.NoError(t, err)
		require.IsType(t, &interpreter.EphemeralReferenceValue{}, result)
	})

	t.Run("not available by default", func(t *testing.T) {

		t.Parallel()

		code := `
          fun main(): &Account {
              return getAuthAccount<auth(Storage) &Account>(0x1)
          }
        `

		location := common.ScriptLocation{0x1}

		activation := sema.NewVariableActivation(sema.BaseValueActivation)
		activation.DeclareValue(stdlib.NewVMGetAuthAccountFunction(nil))

		compilerConfig := &compiler.Config{}
		// NOTE: default globals do not include `getAuthAccount`

		vmConfig := vm.NewConfig(NewUnmeteredInMemoryStorage())
		// NOTE: default globals do not include `getAuthAccount`

		var recovered any
		defer func() {
			recovered = recover()
			require.IsType(t, errors.UnexpectedError{}, recovered)
			unexpectedError := recovered.(errors.UnexpectedError)
			require.ErrorContains(t, unexpectedError, "cannot find global declaration 'getAuthAccount'")
		}()

		_, err := CompileAndInvokeWithOptions(
			t,
			code,
			"main",
			CompilerAndVMOptions{
				ParseCheckAndCompileOptions: ParseCheckAndCompileOptions{
					CompilerConfig: compilerConfig,
					ParseAndCheckOptions: &ParseAndCheckOptions{
						Location: location,
						CheckerConfig: &sema.Config{
							LocationHandler: SingleIdentifierLocationResolver(t),
							BaseValueActivationHandler: func(location common.Location) *sema.VariableActivation {
								return activation
							},
						},
					},
				},
				VMConfig: vmConfig,
			},
		)
		RequireError(t, err)
	})
}

func TestStringTemplate(t *testing.T) {

	t.Parallel()

	t.Run("simple", func(t *testing.T) {
		t.Parallel()

		result, err := CompileAndInvoke(t,
			`
                fun test(): String {
                    var s = "2+2=\(2+2)"
                    return s
                }
            `,
			"test",
		)
		require.NoError(t, err)
		require.Equal(t, interpreter.NewUnmeteredStringValue("2+2=4"), result)
	})

	t.Run("multiple exprs", func(t *testing.T) {
		t.Parallel()

		result, err := CompileAndInvoke(t,
			`
                fun test(): String {
                    let a = "A"
                    let b = "B"
                    let c = 4
                    let str = "\(a) + \(b) = \(c)"
                    return str
                }
            `,
			"test",
		)
		require.NoError(t, err)
		require.Equal(t, interpreter.NewUnmeteredStringValue("A + B = 4"), result)
	})
}

type assumeValidPublicKeyValidator struct{}

var _ stdlib.PublicKeyValidator = assumeValidPublicKeyValidator{}

func (assumeValidPublicKeyValidator) ValidatePublicKey(_ *stdlib.PublicKey) error {
	return nil
}

func TestAttachments(t *testing.T) {

	t.Parallel()

	t.Run("basic", func(t *testing.T) {
		t.Parallel()

		value, err := CompileAndInvoke(t, `
		resource R {}
		attachment A for R {
			fun foo(): Int { return 3 }
		}
		fun test(): Int {
			let r <- create R()
			let r2 <- attach A() to <-r
			let i = r2[A]?.foo()!
			destroy r2
			return i
		}
		`, "test")
		require.NoError(t, err)

		require.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(3), value)
	})

	t.Run("built-in type", func(t *testing.T) {
		t.Parallel()

		code := `
			attachment A for AnyStruct {
				fun foo(): Int {
					return 42
				}
			}

			fun main(): Int {
				var key = PublicKey(
					publicKey: "0102".decodeHex(),
					signatureAlgorithm: SignatureAlgorithm.ECDSA_P256
				)
				key = attach A() to key
				return key[A]!.foo()
			}
        `

		validator := stdlib.NewVMPublicKeyConstructor(
			assumeValidPublicKeyValidator{},
		)

		baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
		for _, valueDeclaration := range []stdlib.StandardLibraryValue{
			validator,
			stdlib.InterpreterSignatureAlgorithmConstructor,
		} {
			baseValueActivation.DeclareValue(valueDeclaration)
		}

		compilerConfig := &compiler.Config{
			BuiltinGlobalsProvider: func(_ common.Location) *activations.Activation[compiler.GlobalImport] {
				activation := activations.NewActivation(nil, compiler.DefaultBuiltinGlobals())
				activation.Set(
					stdlib.VMSignatureAlgorithmConstructor.Name,
					compiler.GlobalImport{
						Name:          stdlib.VMSignatureAlgorithmConstructor.Name,
						QualifiedName: stdlib.VMSignatureAlgorithmConstructor.Name,
					},
				)
				activation.Set(
					validator.Name,
					compiler.GlobalImport{
						Name:          validator.Name,
						QualifiedName: validator.Name,
					},
				)
				for _, v := range stdlib.VMSignatureAlgorithmCaseValues {
					activation.Set(
						v.Name,
						compiler.GlobalImport{
							Name:          v.Name,
							QualifiedName: v.Name,
						},
					)
				}
				return activation
			},
		}

		storage := interpreter.NewInMemoryStorage(nil, nil)

		vmConfig := vm.NewConfig(storage)

		vmConfig.BuiltinGlobalsProvider = func(_ common.Location) *activations.Activation[vm.Variable] {
			activation := activations.NewActivation(nil, vm.DefaultBuiltinGlobals())
			signatureVar := &interpreter.SimpleVariable{}
			signatureVar.InitializeWithValue(stdlib.VMSignatureAlgorithmConstructor.Value)
			activation.Set(
				stdlib.VMSignatureAlgorithmConstructor.Name,
				signatureVar,
			)

			publicKeyVar := &interpreter.SimpleVariable{}
			publicKeyVar.InitializeWithValue(validator.Value)
			activation.Set(
				validator.Name,
				publicKeyVar,
			)

			for _, v := range stdlib.VMSignatureAlgorithmCaseValues {
				variable := interpreter.NewVariableWithValue(nil, v.Value)
				activation.Set(v.Name, variable)
			}
			return activation
		}

		value, err := CompileAndInvokeWithOptions(
			t,
			code,
			"main",
			CompilerAndVMOptions{
				ParseCheckAndCompileOptions: ParseCheckAndCompileOptions{
					CompilerConfig: compilerConfig,
					ParseAndCheckOptions: &ParseAndCheckOptions{
						CheckerConfig: &sema.Config{
							LocationHandler: SingleIdentifierLocationResolver(t),
							BaseValueActivationHandler: func(location common.Location) *sema.VariableActivation {
								return baseValueActivation
							},
						},
					},
				},
				VMConfig: vmConfig,
			},
		)
		require.NoError(t, err)

		require.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(42), value)
	})

	t.Run("call function in initializer", func(t *testing.T) {
		t.Parallel()

		value, err := CompileAndInvoke(t, `
		struct S {
			var value: Int
			init(value: Int) {
				self.value = value
			}
			fun getValue(): Int {
				return self.value
			}
		}
		attachment A for S {
			var value: Int
			init() {
				self.value = self.foo()
			}
			fun foo(): Int { 
				return base.getValue() 
			}
			fun getValue(): Int {
				return self.value
			}
		}
		fun test(): Int {
			let s = S(value: 3)
			let s2 = attach A() to s
			let i = s2[A]?.getValue()!
			return i
		}
		`, "test")
		require.NoError(t, err)

		require.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(3), value)
	})
}

func TestDynamicMethodInvocationViaOptionalChaining(t *testing.T) {

	t.Parallel()

	actual, err := CompileAndInvoke(t,
		`
          struct interface SI {
              fun answer(): Int
          }

          struct S: SI {
              fun answer(): Int {
                  return 42
              }
          }

          fun answer(_ si: {SI}?): Int? {
              return si?.answer()
          }

          fun test(): Int? {
              return answer(S())
          }
        `,
		"test",
	)
	require.NoError(t, err)
	assert.Equal(
		t,
		interpreter.NewUnmeteredSomeValueNonCopying(
			interpreter.NewUnmeteredIntValueFromInt64(42),
		),
		actual,
	)
}

func TestInjectedContract(t *testing.T) {

	t.Parallel()

	cType := &sema.FunctionType{
		Parameters: []sema.Parameter{
			{
				Label:          sema.ArgumentLabelNotRequired,
				Identifier:     "n",
				TypeAnnotation: sema.IntTypeAnnotation,
			},
		},
		ReturnTypeAnnotation: sema.NewTypeAnnotation(sema.IntType),
	}

	bType := &sema.CompositeType{
		Location:   TestLocation,
		Identifier: "B",
		Kind:       common.CompositeKindContract,
	}

	bType.Members = sema.MembersAsMap([]*sema.Member{
		sema.NewUnmeteredPublicFunctionMember(
			bType,
			"c",
			cType,
			"",
		),
		sema.NewUnmeteredPublicConstantFieldMember(
			bType,
			"d",
			sema.IntType,
			"",
		),
	})

	bStaticType := interpreter.ConvertSemaCompositeTypeToStaticCompositeType(nil, bType)

	bValue := interpreter.NewSimpleCompositeValue(
		nil,
		bType.ID(),
		bStaticType,
		[]string{"d"},
		map[string]interpreter.Value{
			"d": interpreter.NewUnmeteredIntValueFromInt64(1),
		},
		nil,
		nil,
		nil,
		nil,
	)

	baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
	baseValueActivation.DeclareValue(stdlib.StandardLibraryValue{
		Name:  bType.Identifier,
		Type:  bType,
		Value: bValue,
		Kind:  common.DeclarationKindContract,
	})

	compilerConfig := &compiler.Config{
		BuiltinGlobalsProvider: func(location common.Location) *activations.Activation[compiler.GlobalImport] {
			assert.Equal(t, TestLocation, location)
			activation := activations.NewActivation(nil, compiler.DefaultBuiltinGlobals())
			activation.Set(
				"B",
				compiler.NewGlobalImport("B"),
			)
			activation.Set(
				"B.c",
				compiler.NewGlobalImport("B.c"),
			)
			return activation
		},
	}

	cValue := vm.NewNativeFunctionValue(
		"B.c",
		cType,
		func(
			context interpreter.NativeFunctionContext,
			_ interpreter.TypeArgumentsIterator,
			_ interpreter.ArgumentTypesIterator,
			receiver interpreter.Value,
			args []interpreter.Value,
		) interpreter.Value {
			assert.Same(t, bValue, receiver)

			require.Len(t, args, 1)
			require.IsType(t, interpreter.IntValue{}, args[0])
			arg := args[0].(interpreter.IntValue)

			return arg.Plus(context, arg)
		},
	)

	vmConfig := vm.NewConfig(NewUnmeteredInMemoryStorage())
	vmConfig.BuiltinGlobalsProvider = func(location common.Location) *activations.Activation[vm.Variable] {
		assert.Equal(t, TestLocation, location)
		activation := activations.NewActivation(nil, vm.DefaultBuiltinGlobals())

		bVariable := &interpreter.SimpleVariable{}
		bVariable.InitializeWithValue(bValue)
		activation.Set("B", bVariable)

		cVariable := &interpreter.SimpleVariable{}
		cVariable.InitializeWithValue(cValue)
		activation.Set("B.c", cVariable)

		return activation
	}

	programs := map[common.Location]*CompiledProgram{}

	compositeTypeLoader := CompiledProgramsCompositeTypeLoader(programs)
	vmConfig.InterfaceTypeHandler = CompiledProgramsInterfaceTypeLoader(programs)
	vmConfig.EntitlementTypeHandler = CompiledProgramsEntitlementTypeLoader(programs)
	vmConfig.EntitlementMapTypeHandler = CompiledProgramsEntitlementMapTypeLoader(programs)

	vmConfig.CompositeTypeHandler = func(location common.Location, typeID interpreter.TypeID) *sema.CompositeType {
		if location == TestLocation && typeID == "S.test.B" {
			return bType
		}

		return compositeTypeLoader(location, typeID)
	}

	result, err := CompileAndInvokeWithOptions(
		t,
		`
          contract A {
              fun test(): Int {
                  return B.c(B.d)
              }
          }

          fun main(): Int {
              return A.test()
          }
        `,
		"main",
		CompilerAndVMOptions{
			ParseCheckAndCompileOptions: ParseCheckAndCompileOptions{
				CompilerConfig: compilerConfig,
				ParseAndCheckOptions: &ParseAndCheckOptions{
					Location: TestLocation,
					CheckerConfig: &sema.Config{
						BaseValueActivationHandler: func(location common.Location) *sema.VariableActivation {
							assert.Equal(t, TestLocation, location)
							return baseValueActivation
						},
					},
				},
			},
			VMConfig: vmConfig,
			Programs: programs,
		},
	)
	require.NoError(t, err)
	assert.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(2), result)
}

func TestStackDepthLimit(t *testing.T) {

	t.Parallel()

	vmConfig := vm.NewConfig(NewUnmeteredInMemoryStorage())
	vmConfig.StackDepthLimit = 10

	_, err := CompileAndInvokeWithOptions(t,
		`

            fun test() {
                test()
            }
        `,
		"test",
		CompilerAndVMOptions{
			VMConfig: vmConfig,
		},
	)
	RequireError(t, err)

	var callStackLimitExceededErr *interpreter.CallStackLimitExceededError
	require.ErrorAs(t, err, &callStackLimitExceededErr)
}

func TestNestedLoops(t *testing.T) {

	t.Parallel()

	_, err := CompileAndInvoke(t,
		`
             fun test() {
                 for x in [1, 2] {
                     for y in [1] {}
                 }
             }
        `,
		"test",
	)
	require.NoError(t, err)

}

func TestInheritedDefaultDestroyEvent(t *testing.T) {
	t.Parallel()

	storage := NewUnmeteredInMemoryStorage()

	programs := map[common.Location]*CompiledProgram{}
	contractValues := map[common.Location]*interpreter.CompositeValue{}
	var logs []string

	vmConfig := vm.NewConfig(storage)
	vmConfig.ImportHandler = func(location common.Location) *bbq.InstructionProgram {
		program, ok := programs[location]
		if !ok {
			assert.FailNow(t, "invalid location")
		}
		return program.Program
	}
	vmConfig.ContractValueHandler = func(_ *vm.Context, location common.Location) *interpreter.CompositeValue {
		contractValue, ok := contractValues[location]
		if !ok {
			assert.FailNow(t, "invalid location")
		}
		return contractValue
	}
	vmConfig.CompositeTypeHandler = CompiledProgramsCompositeTypeLoader(programs)
	vmConfig.InterfaceTypeHandler = CompiledProgramsInterfaceTypeLoader(programs)
	vmConfig.EntitlementTypeHandler = CompiledProgramsEntitlementTypeLoader(programs)
	vmConfig.EntitlementMapTypeHandler = CompiledProgramsEntitlementMapTypeLoader(programs)

	vmConfig.BuiltinGlobalsProvider = NewVMBuiltinGlobalsProviderWithDefaultsPanicAndConditionLog(&logs)

	activation := sema.NewVariableActivation(sema.BaseValueActivation)
	activation.DeclareValue(stdlib.VMPanicFunction)
	activation.DeclareValue(newConditionLogFunction(nil))

	contractsAddress := common.MustBytesToAddress([]byte{0x1})

	barLocation := common.NewAddressLocation(nil, contractsAddress, "Bar")
	fooLocation := common.NewAddressLocation(nil, contractsAddress, "Foo")

	// Deploy interface contract

	barContract := `
        contract interface Bar {
            resource interface XYZ {
                var x: Int
                event ResourceDestroyed(x: Int = self.x)
            }
        }
        `

	compilerConfig := &compiler.Config{
		BuiltinGlobalsProvider: CompilerDefaultBuiltinGlobalsWithDefaultsAndConditionLog,
	}

	// Only need to compile
	_ = ParseCheckAndCompileCodeWithOptions(
		t,
		barContract,
		barLocation,
		ParseCheckAndCompileOptions{
			CompilerConfig: compilerConfig,
			ParseAndCheckOptions: &ParseAndCheckOptions{
				Location: barLocation,
				CheckerConfig: &sema.Config{
					LocationHandler: SingleIdentifierLocationResolver(t),
					BaseValueActivationHandler: func(location common.Location) *sema.VariableActivation {
						return activation
					},
				},
			},
		},
		programs,
	)

	// Deploy contract with the implementation

	fooContract := fmt.Sprintf(
		`
        import Bar from %[1]s

        contract Foo {

            resource ABC: Bar.XYZ {
                var x: Int

                init() {
                    self.x = 6
                }
            }

            fun createABC(): @ABC {
                return <- create ABC()
            }
        }`,
		contractsAddress.HexWithPrefix(),
	)

	fooProgram := ParseCheckAndCompileCodeWithOptions(
		t,
		fooContract,
		fooLocation,
		ParseCheckAndCompileOptions{
			CompilerConfig: compilerConfig,
			ParseAndCheckOptions: &ParseAndCheckOptions{
				Location: fooLocation,
				CheckerConfig: &sema.Config{
					LocationHandler: SingleIdentifierLocationResolver(t),
					BaseValueActivationHandler: func(location common.Location) *sema.VariableActivation {
						return activation
					},
				},
			},
		},
		programs,
	)

	_, fooContractValue := initializeContract(
		t,
		fooLocation,
		fooProgram,
		vmConfig,
	)
	contractValues[fooLocation] = fooContractValue

	// Run script

	code := fmt.Sprintf(
		`
              import Foo from %[1]s

              fun main() {
                  let abc <- Foo.createABC()
                  destroy abc
              }
            `,
		contractsAddress.HexWithPrefix(),
	)

	location := common.ScriptLocation{0x1}

	var eventEmitted bool

	vmConfig.OnEventEmitted = func(
		context interpreter.ValueExportContext,
		eventType *sema.CompositeType,
		eventFields []interpreter.Value,
	) error {
		require.False(t, eventEmitted)
		eventEmitted = true

		assert.Equal(t,
			[]interpreter.Value{
				interpreter.NewUnmeteredIntValueFromInt64(6),
			},
			eventFields,
		)

		assert.Equal(t,
			barLocation.TypeID(nil, "Bar.XYZ.ResourceDestroyed"),
			eventType.ID(),
		)

		return nil
	}

	_, err := CompileAndInvokeWithOptionsAndPrograms(
		t,
		code,
		"main",
		CompilerAndVMOptions{
			ParseCheckAndCompileOptions: ParseCheckAndCompileOptions{
				CompilerConfig: compilerConfig,
				ParseAndCheckOptions: &ParseAndCheckOptions{
					Location: location,
					CheckerConfig: &sema.Config{
						LocationHandler: SingleIdentifierLocationResolver(t),
						BaseValueActivationHandler: func(location common.Location) *sema.VariableActivation {
							return activation
						},
					},
				},
			},
			VMConfig: vmConfig,
		},
		programs,
	)
	require.NoError(t, err)
	require.True(t, eventEmitted)
}

func TestFunctionInclusiveRangeConstruction(t *testing.T) {

	t.Parallel()

	activation := sema.NewVariableActivation(sema.BaseValueActivation)
	activation.DeclareValue(stdlib.VMInclusiveRangeConstructor)

	compilerConfig := &compiler.Config{
		BuiltinGlobalsProvider: func(_ common.Location) *activations.Activation[compiler.GlobalImport] {
			activation := activations.NewActivation(nil, compiler.DefaultBuiltinGlobals())
			activation.Set(
				stdlib.VMInclusiveRangeConstructor.Name,
				compiler.NewGlobalImport(stdlib.VMInclusiveRangeConstructor.Name),
			)
			return activation
		},
	}
	vmConfig := vm.NewConfig(NewUnmeteredInMemoryStorage())
	vmConfig.BuiltinGlobalsProvider = func(_ common.Location) *activations.Activation[vm.Variable] {
		activation := activations.NewActivation(nil, vm.DefaultBuiltinGlobals())
		variable := &interpreter.SimpleVariable{}
		variable.InitializeWithValue(stdlib.VMInclusiveRangeConstructor.Value)
		activation.Set(stdlib.VMInclusiveRangeConstructor.Name, variable)
		return activation
	}

	result, err := CompileAndInvokeWithOptions(
		t,
		`
          fun test(): InclusiveRange<Int> {
              return InclusiveRange(5, 10)
          }
        `,
		"test",
		CompilerAndVMOptions{
			ParseCheckAndCompileOptions: ParseCheckAndCompileOptions{
				CompilerConfig: compilerConfig,
				ParseAndCheckOptions: &ParseAndCheckOptions{
					CheckerConfig: &sema.Config{
						BaseValueActivationHandler: func(location common.Location) *sema.VariableActivation {
							return activation
						},
					},
				},
			},
			VMConfig: vmConfig,
		},
	)
	require.NoError(t, err)
	require.IsType(t, &interpreter.CompositeValue{}, result)
}

func TestVMImportAliasing(t *testing.T) {

	t.Parallel()

	t.Run("basic", func(t *testing.T) {

		t.Parallel()

		importLocation := common.AddressLocation{
			Address: common.MustBytesToAddress([]byte{0x1}),
			Name:    "MyContract",
		}

		importedChecker, err := ParseAndCheckWithOptions(t,
			`
                contract MyContract {

                    fun helloText(): String {
                        return "global function of the imported program"
                    }

                    init() {}

                    struct Foo {
                        var id : String

                        init(_ id: String) {
                            self.id = id
                        }

                        fun sayHello(_ id: Int): String {
                            self.id
                            return MyContract.helloText()
                        }
                    }
                }
            `,
			ParseAndCheckOptions{
				Location: importLocation,
			},
		)
		require.NoError(t, err)

		importCompiler := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(importedChecker),
			importedChecker.Location,
		)
		importedProgram := importCompiler.Compile()

		_, importedContractValue := initializeContract(
			t,
			importLocation,
			importedProgram,
			nil,
		)

		checker, err := ParseAndCheckWithOptions(t,
			`
                import MyContract as TheirContract from 0x01

                fun test(): String {
                    var r = TheirContract.Foo("Hello from Foo!")
                    return r.sayHello(1)
                }
            `,
			ParseAndCheckOptions{
				CheckerConfig: &sema.Config{
					LocationHandler: SingleIdentifierLocationResolver(t),
					ImportHandler: func(_ *sema.Checker, location common.Location, _ ast.Range) (sema.Import, error) {
						require.Equal(t, importLocation, location)
						return sema.ElaborationImport{
							Elaboration: importedChecker.Elaboration,
						}, nil
					},
				},
			},
		)
		require.NoError(t, err)

		comp := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(checker),
			checker.Location,
		)
		comp.Config.ImportHandler = func(location common.Location) *bbq.InstructionProgram {
			require.Equal(t, importLocation, location)
			return importedProgram
		}

		program := comp.Compile()

		vmConfig := vm.NewConfig(NewUnmeteredInMemoryStorage())
		vmConfig.ImportHandler = func(location common.Location) *bbq.InstructionProgram {
			require.Equal(t, importLocation, location)
			return importedProgram
		}
		vmConfig.ContractValueHandler = func(_ *vm.Context, location common.Location) *interpreter.CompositeValue {
			require.Equal(t, importLocation, location)
			return importedContractValue
		}
		vmConfig.CompositeTypeHandler = func(location common.Location, typeID interpreter.TypeID) *sema.CompositeType {
			require.Equal(t, importLocation, location)
			elaboration := importedChecker.Elaboration
			return elaboration.CompositeType(typeID)
		}
		vmConfig.InterfaceTypeHandler = func(location common.Location, typeID interpreter.TypeID) *sema.InterfaceType {
			require.Equal(t, importLocation, location)
			elaboration := importedChecker.Elaboration
			return elaboration.InterfaceType(typeID)
		}

		vmInstance := vm.NewVM(scriptLocation(), program, vmConfig)

		result, err := vmInstance.InvokeExternally("test")
		require.NoError(t, err)
		require.Equal(t, 0, vmInstance.StackSize())

		require.Equal(t, interpreter.NewUnmeteredStringValue("global function of the imported program"), result)
	})

	t.Run("same contract name", func(t *testing.T) {

		t.Parallel()

		// Initialize Foo

		fooLocation := common.NewAddressLocation(
			nil,
			common.MustBytesToAddress([]byte{0x1}),
			"Foo",
		)

		fooChecker, err := ParseAndCheckWithOptions(t,
			`
            contract Foo {
                fun foo(): Int {
                    return 1
                }
            }
            `,
			ParseAndCheckOptions{
				Location: fooLocation,
			},
		)
		require.NoError(t, err)

		fooCompiler := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(fooChecker),
			fooChecker.Location,
		)
		fooProgram := fooCompiler.Compile()

		var fooContractValue *interpreter.CompositeValue

		vmConfig := vm.NewConfig(NewUnmeteredInMemoryStorage())
		vmConfig.ImportHandler = func(location common.Location) *bbq.InstructionProgram {
			require.Equal(t, fooLocation, location)
			return fooProgram
		}
		vmConfig.ContractValueHandler = func(_ *vm.Context, location common.Location) *interpreter.CompositeValue {
			require.Equal(t, fooLocation, location)
			return fooContractValue
		}
		vmConfig.CompositeTypeHandler = func(location common.Location, typeID interpreter.TypeID) *sema.CompositeType {
			require.Equal(t, fooLocation, location)

			elaboration := fooChecker.Elaboration
			return elaboration.CompositeType(typeID)
		}
		vmConfig.InterfaceTypeHandler = func(location common.Location, typeID interpreter.TypeID) *sema.InterfaceType {
			require.Equal(t, fooLocation, location)

			elaboration := fooChecker.Elaboration
			return elaboration.InterfaceType(typeID)
		}

		_, fooContractValue = initializeContract(
			t,
			fooLocation,
			fooProgram,
			vmConfig,
		)

		// Initialize duplicate Foo (Bar)

		barLocation := common.NewAddressLocation(
			nil,
			common.MustBytesToAddress([]byte{0x2}),
			"Foo",
		)

		barChecker, err := ParseAndCheckWithOptions(t,
			`
            contract Foo {
                fun bar(): Int {
                    return 2
                }
            }
            `,
			ParseAndCheckOptions{
				Location: barLocation,
				CheckerConfig: &sema.Config{
					LocationHandler: SingleIdentifierLocationResolver(t),
					ImportHandler: func(_ *sema.Checker, location common.Location, _ ast.Range) (sema.Import, error) {
						require.Equal(t, fooLocation, location)
						return sema.ElaborationImport{
							Elaboration: fooChecker.Elaboration,
						}, nil
					},
				},
			},
		)
		require.NoError(t, err)

		barCompiler := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(barChecker),
			barChecker.Location,
		)
		barCompiler.Config.LocationHandler = SingleIdentifierLocationResolver(t)
		barCompiler.Config.ImportHandler = func(location common.Location) *bbq.InstructionProgram {
			require.Equal(t, fooLocation, location)
			return fooProgram
		}

		barProgram := barCompiler.Compile()

		_, barContractValue := initializeContract(
			t,
			barLocation,
			barProgram,
			vmConfig,
		)

		// Compile and run main program

		checker, err := ParseAndCheckWithOptions(t,
			`
            import Foo as Foo1 from 0x01
            import Foo as Foo2 from 0x02

            fun test(): Int {
                return Foo1.foo() + Foo2.bar()
            }
            `,
			ParseAndCheckOptions{
				CheckerConfig: &sema.Config{
					LocationHandler: SingleIdentifierLocationResolver(t),
					ImportHandler: func(_ *sema.Checker, location common.Location, _ ast.Range) (sema.Import, error) {
						require.IsType(t, common.AddressLocation{}, location)
						addressLocation := location.(common.AddressLocation)
						var elaboration *sema.Elaboration
						switch addressLocation.Address {
						case fooLocation.Address:
							elaboration = fooChecker.Elaboration
						case barLocation.Address:
							elaboration = barChecker.Elaboration
						default:
							assert.FailNow(t, "invalid location")
						}

						return sema.ElaborationImport{
							Elaboration: elaboration,
						}, nil
					},
				},
			},
		)
		require.NoError(t, err)

		comp := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(checker),
			checker.Location,
		)
		comp.Config.LocationHandler = SingleIdentifierLocationResolver(t)
		comp.Config.ImportHandler = func(location common.Location) *bbq.InstructionProgram {
			switch location {
			case fooLocation:
				return fooProgram
			case barLocation:
				return barProgram
			default:
				assert.FailNow(t, "invalid location")
				return nil
			}
		}

		program := comp.Compile()

		vmConfig = vm.NewConfig(NewUnmeteredInMemoryStorage())
		vmConfig.ImportHandler = func(location common.Location) *bbq.InstructionProgram {
			switch location {
			case fooLocation:
				return fooProgram
			case barLocation:
				return barProgram
			default:
				assert.FailNow(t, "invalid location")
				return nil
			}
		}
		vmConfig.ContractValueHandler = func(_ *vm.Context, location common.Location) *interpreter.CompositeValue {
			switch location {
			case fooLocation:
				return fooContractValue
			case barLocation:
				return barContractValue
			default:
				assert.FailNow(t, fmt.Sprintf("invalid location %s", location))
				return nil
			}
		}
		vmConfig.CompositeTypeHandler = func(location common.Location, typeID interpreter.TypeID) *sema.CompositeType {

			var elaboration *sema.Elaboration

			switch location {
			case fooLocation:
				elaboration = fooChecker.Elaboration
			case barLocation:
				elaboration = barChecker.Elaboration
			default:
				assert.FailNow(t, fmt.Sprintf("invalid location %s", location))
				return nil
			}

			return elaboration.CompositeType(typeID)
		}
		vmConfig.InterfaceTypeHandler = func(location common.Location, typeID interpreter.TypeID) *sema.InterfaceType {

			var elaboration *sema.Elaboration

			switch location {
			case fooLocation:
				elaboration = fooChecker.Elaboration
			case barLocation:
				elaboration = barChecker.Elaboration
			default:
				assert.FailNow(t, fmt.Sprintf("invalid location %s", location))
				return nil
			}

			return elaboration.InterfaceType(typeID)
		}

		vmInstance := vm.NewVM(scriptLocation(), program, vmConfig)

		result, err := vmInstance.InvokeExternally("test")
		require.NoError(t, err)
		require.Equal(t, 0, vmInstance.StackSize())

		require.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(3), result)
	})

	t.Run("same contract name, same function name", func(t *testing.T) {

		t.Parallel()

		// Initialize Foo

		fooLocation := common.NewAddressLocation(
			nil,
			common.MustBytesToAddress([]byte{0x1}),
			"Foo",
		)

		fooChecker, err := ParseAndCheckWithOptions(t,
			`
            contract Foo {
                fun value(): Int {
                    return 1
                }
            }
            `,
			ParseAndCheckOptions{
				Location: fooLocation,
			},
		)
		require.NoError(t, err)

		fooCompiler := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(fooChecker),
			fooChecker.Location,
		)
		fooProgram := fooCompiler.Compile()

		var fooContractValue *interpreter.CompositeValue

		vmConfig := vm.NewConfig(NewUnmeteredInMemoryStorage())
		vmConfig.ImportHandler = func(location common.Location) *bbq.InstructionProgram {
			require.Equal(t, fooLocation, location)
			return fooProgram
		}
		vmConfig.ContractValueHandler = func(_ *vm.Context, location common.Location) *interpreter.CompositeValue {
			require.Equal(t, fooLocation, location)
			return fooContractValue
		}
		vmConfig.CompositeTypeHandler = func(location common.Location, typeID interpreter.TypeID) *sema.CompositeType {
			require.Equal(t, fooLocation, location)

			elaboration := fooChecker.Elaboration
			return elaboration.CompositeType(typeID)
		}
		vmConfig.InterfaceTypeHandler = func(location common.Location, typeID interpreter.TypeID) *sema.InterfaceType {
			require.Equal(t, fooLocation, location)

			elaboration := fooChecker.Elaboration
			return elaboration.InterfaceType(typeID)
		}

		_, fooContractValue = initializeContract(
			t,
			fooLocation,
			fooProgram,
			vmConfig,
		)

		// Initialize duplicate Foo (Bar)

		barLocation := common.NewAddressLocation(
			nil,
			common.MustBytesToAddress([]byte{0x2}),
			"Foo",
		)

		barChecker, err := ParseAndCheckWithOptions(t,
			`
            contract Foo {
                fun value(): Int {
                    return 2
                }
            }
            `,
			ParseAndCheckOptions{
				Location: barLocation,
				CheckerConfig: &sema.Config{
					LocationHandler: SingleIdentifierLocationResolver(t),
					ImportHandler: func(_ *sema.Checker, location common.Location, _ ast.Range) (sema.Import, error) {
						require.Equal(t, fooLocation, location)
						return sema.ElaborationImport{
							Elaboration: fooChecker.Elaboration,
						}, nil
					},
				},
			},
		)
		require.NoError(t, err)

		barCompiler := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(barChecker),
			barChecker.Location,
		)
		barCompiler.Config.LocationHandler = SingleIdentifierLocationResolver(t)
		barCompiler.Config.ImportHandler = func(location common.Location) *bbq.InstructionProgram {
			require.Equal(t, fooLocation, location)
			return fooProgram
		}

		barProgram := barCompiler.Compile()

		_, barContractValue := initializeContract(
			t,
			barLocation,
			barProgram,
			vmConfig,
		)

		// Compile and run main program

		checker, err := ParseAndCheckWithOptions(t,
			`
            import Foo as Foo1 from 0x01
            import Foo as Foo2 from 0x02

            fun test(): Int {
                return Foo1.value() + Foo2.value()
            }
            `,
			ParseAndCheckOptions{
				CheckerConfig: &sema.Config{
					LocationHandler: SingleIdentifierLocationResolver(t),
					ImportHandler: func(_ *sema.Checker, location common.Location, _ ast.Range) (sema.Import, error) {
						require.IsType(t, common.AddressLocation{}, location)
						addressLocation := location.(common.AddressLocation)
						var elaboration *sema.Elaboration
						switch addressLocation.Address {
						case fooLocation.Address:
							elaboration = fooChecker.Elaboration
						case barLocation.Address:
							elaboration = barChecker.Elaboration
						default:
							assert.FailNow(t, "invalid location")
						}

						return sema.ElaborationImport{
							Elaboration: elaboration,
						}, nil
					},
				},
			},
		)
		require.NoError(t, err)

		comp := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(checker),
			checker.Location,
		)
		comp.Config.LocationHandler = SingleIdentifierLocationResolver(t)
		comp.Config.ImportHandler = func(location common.Location) *bbq.InstructionProgram {
			switch location {
			case fooLocation:
				return fooProgram
			case barLocation:
				return barProgram
			default:
				assert.FailNow(t, "invalid location")
				return nil
			}
		}

		program := comp.Compile()

		vmConfig = vm.NewConfig(NewUnmeteredInMemoryStorage())
		vmConfig.ImportHandler = func(location common.Location) *bbq.InstructionProgram {
			switch location {
			case fooLocation:
				return fooProgram
			case barLocation:
				return barProgram
			default:
				assert.FailNow(t, "invalid location")
				return nil
			}
		}
		vmConfig.ContractValueHandler = func(_ *vm.Context, location common.Location) *interpreter.CompositeValue {
			switch location {
			case fooLocation:
				return fooContractValue
			case barLocation:
				return barContractValue
			default:
				assert.FailNow(t, fmt.Sprintf("invalid location %s", location))
				return nil
			}
		}
		vmConfig.CompositeTypeHandler = func(location common.Location, typeID interpreter.TypeID) *sema.CompositeType {

			var elaboration *sema.Elaboration

			switch location {
			case fooLocation:
				elaboration = fooChecker.Elaboration
			case barLocation:
				elaboration = barChecker.Elaboration
			default:
				assert.FailNow(t, fmt.Sprintf("invalid location %s", location))
				return nil
			}

			return elaboration.CompositeType(typeID)
		}
		vmConfig.InterfaceTypeHandler = func(location common.Location, typeID interpreter.TypeID) *sema.InterfaceType {

			var elaboration *sema.Elaboration

			switch location {
			case fooLocation:
				elaboration = fooChecker.Elaboration
			case barLocation:
				elaboration = barChecker.Elaboration
			default:
				assert.FailNow(t, fmt.Sprintf("invalid location %s", location))
				return nil
			}

			return elaboration.InterfaceType(typeID)
		}

		vmInstance := vm.NewVM(scriptLocation(), program, vmConfig)

		result, err := vmInstance.InvokeExternally("test")
		require.NoError(t, err)
		require.Equal(t, 0, vmInstance.StackSize())

		require.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(3), result)
	})

	t.Run("alias same contract twice", func(t *testing.T) {

		t.Parallel()

		// Initialize Foo

		fooLocation := common.NewAddressLocation(
			nil,
			common.MustBytesToAddress([]byte{0x1}),
			"Foo",
		)

		fooChecker, err := ParseAndCheckWithOptions(t,
			`
            contract Foo {
                fun value(): Int {
                    return 1
                }
            }
            `,
			ParseAndCheckOptions{
				Location: fooLocation,
			},
		)
		require.NoError(t, err)

		fooCompiler := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(fooChecker),
			fooChecker.Location,
		)
		fooProgram := fooCompiler.Compile()

		var fooContractValue *interpreter.CompositeValue

		vmConfig := vm.NewConfig(NewUnmeteredInMemoryStorage())
		vmConfig.ImportHandler = func(location common.Location) *bbq.InstructionProgram {
			require.Equal(t, fooLocation, location)
			return fooProgram
		}
		vmConfig.ContractValueHandler = func(_ *vm.Context, location common.Location) *interpreter.CompositeValue {
			require.Equal(t, fooLocation, location)
			return fooContractValue
		}
		vmConfig.CompositeTypeHandler = func(location common.Location, typeID interpreter.TypeID) *sema.CompositeType {
			require.Equal(t, fooLocation, location)

			elaboration := fooChecker.Elaboration
			return elaboration.CompositeType(typeID)
		}
		vmConfig.InterfaceTypeHandler = func(location common.Location, typeID interpreter.TypeID) *sema.InterfaceType {
			require.Equal(t, fooLocation, location)

			elaboration := fooChecker.Elaboration
			return elaboration.InterfaceType(typeID)
		}

		_, fooContractValue = initializeContract(
			t,
			fooLocation,
			fooProgram,
			vmConfig,
		)

		// Initialize Bar

		barLocation := common.NewAddressLocation(
			nil,
			common.MustBytesToAddress([]byte{0x2}),
			"Bar",
		)

		barChecker, err := ParseAndCheckWithOptions(t,
			`
            import Foo as Cab from 0x01
            contract Bar {
                fun value(): Int {
                    return Cab.value()
                }
            }
            `,
			ParseAndCheckOptions{
				Location: barLocation,
				CheckerConfig: &sema.Config{
					LocationHandler: SingleIdentifierLocationResolver(t),
					ImportHandler: func(_ *sema.Checker, location common.Location, _ ast.Range) (sema.Import, error) {
						require.Equal(t, fooLocation, location)
						return sema.ElaborationImport{
							Elaboration: fooChecker.Elaboration,
						}, nil
					},
				},
			},
		)
		require.NoError(t, err)

		barCompiler := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(barChecker),
			barChecker.Location,
		)
		barCompiler.Config.LocationHandler = SingleIdentifierLocationResolver(t)
		barCompiler.Config.ImportHandler = func(location common.Location) *bbq.InstructionProgram {
			require.Equal(t, fooLocation, location)
			return fooProgram
		}

		barProgram := barCompiler.Compile()

		_, barContractValue := initializeContract(
			t,
			barLocation,
			barProgram,
			vmConfig,
		)

		// Compile and run main program

		checker, err := ParseAndCheckWithOptions(t,
			`
            import Bar as Foo from 0x02

            fun test(): Int {
                return Foo.value()
            }
            `,
			ParseAndCheckOptions{
				CheckerConfig: &sema.Config{
					LocationHandler: SingleIdentifierLocationResolver(t),
					ImportHandler: func(_ *sema.Checker, location common.Location, _ ast.Range) (sema.Import, error) {
						require.IsType(t, common.AddressLocation{}, location)
						addressLocation := location.(common.AddressLocation)
						var elaboration *sema.Elaboration
						switch addressLocation.Address {
						case fooLocation.Address:
							elaboration = fooChecker.Elaboration
						case barLocation.Address:
							elaboration = barChecker.Elaboration
						default:
							assert.FailNow(t, "invalid location")
						}

						return sema.ElaborationImport{
							Elaboration: elaboration,
						}, nil
					},
				},
			},
		)
		require.NoError(t, err)

		comp := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(checker),
			checker.Location,
		)
		comp.Config.LocationHandler = SingleIdentifierLocationResolver(t)
		comp.Config.ImportHandler = func(location common.Location) *bbq.InstructionProgram {
			switch location {
			case fooLocation:
				return fooProgram
			case barLocation:
				return barProgram
			default:
				assert.FailNow(t, "invalid location")
				return nil
			}
		}

		program := comp.Compile()

		vmConfig = vm.NewConfig(NewUnmeteredInMemoryStorage())
		vmConfig.ImportHandler = func(location common.Location) *bbq.InstructionProgram {
			switch location {
			case fooLocation:
				return fooProgram
			case barLocation:
				return barProgram
			default:
				assert.FailNow(t, "invalid location")
				return nil
			}
		}
		vmConfig.ContractValueHandler = func(_ *vm.Context, location common.Location) *interpreter.CompositeValue {
			switch location {
			case fooLocation:
				return fooContractValue
			case barLocation:
				return barContractValue
			default:
				assert.FailNow(t, fmt.Sprintf("invalid location %s", location))
				return nil
			}
		}
		vmConfig.CompositeTypeHandler = func(location common.Location, typeID interpreter.TypeID) *sema.CompositeType {

			var elaboration *sema.Elaboration

			switch location {
			case fooLocation:
				elaboration = fooChecker.Elaboration
			case barLocation:
				elaboration = barChecker.Elaboration
			default:
				assert.FailNow(t, fmt.Sprintf("invalid location %s", location))
				return nil
			}

			return elaboration.CompositeType(typeID)
		}
		vmConfig.InterfaceTypeHandler = func(location common.Location, typeID interpreter.TypeID) *sema.InterfaceType {

			var elaboration *sema.Elaboration

			switch location {
			case fooLocation:
				elaboration = fooChecker.Elaboration
			case barLocation:
				elaboration = barChecker.Elaboration
			default:
				assert.FailNow(t, fmt.Sprintf("invalid location %s", location))
				return nil
			}

			return elaboration.InterfaceType(typeID)
		}

		vmInstance := vm.NewVM(scriptLocation(), program, vmConfig)

		result, err := vmInstance.InvokeExternally("test")
		require.NoError(t, err)
		require.Equal(t, 0, vmInstance.StackSize())

		require.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(1), result)
	})

	t.Run("get nested types of alias", func(t *testing.T) {

		t.Parallel()

		importLocation := common.AddressLocation{
			Address: common.MustBytesToAddress([]byte{0x1}),
			Name:    "MyContract",
		}

		importedChecker, err := ParseAndCheckWithOptions(t,
			`
                contract MyContract {
                    struct MyStruct {
                        var value : Int
                        init (_ v: Int) {self.value = v}
                        fun getValue(): Int {
                            return self.value
                        }
                    }
                }
            `,
			ParseAndCheckOptions{
				Location: importLocation,
			},
		)
		require.NoError(t, err)

		importCompiler := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(importedChecker),
			importedChecker.Location,
		)
		importedProgram := importCompiler.Compile()

		_, importedContractValue := initializeContract(
			t,
			importLocation,
			importedProgram,
			nil,
		)

		checker, err := ParseAndCheckWithOptions(t,
			`
                import MyContract as TheirContract from 0x01

                struct Foo {
                    var tmp: TheirContract.MyStruct
                    init() {
                        self.tmp = TheirContract.MyStruct(5)
                    }
                }

                fun test(): Int {
                    return Foo().tmp.getValue()
                }
            `,
			ParseAndCheckOptions{
				CheckerConfig: &sema.Config{
					LocationHandler: SingleIdentifierLocationResolver(t),
					ImportHandler: func(_ *sema.Checker, location common.Location, _ ast.Range) (sema.Import, error) {
						require.Equal(t, importLocation, location)
						return sema.ElaborationImport{
							Elaboration: importedChecker.Elaboration,
						}, nil
					},
				},
			},
		)
		require.NoError(t, err)

		comp := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(checker),
			checker.Location,
		)
		comp.Config.ImportHandler = func(location common.Location) *bbq.InstructionProgram {
			require.Equal(t, importLocation, location)
			return importedProgram
		}

		program := comp.Compile()

		vmConfig := vm.NewConfig(NewUnmeteredInMemoryStorage())
		vmConfig.ImportHandler = func(location common.Location) *bbq.InstructionProgram {
			switch location {
			case importLocation:
				return importedProgram
			case checker.Location:
				return program
			default:
				assert.FailNow(t, "invalid location")
				return nil
			}
		}
		vmConfig.ContractValueHandler = func(_ *vm.Context, location common.Location) *interpreter.CompositeValue {
			require.Equal(t, importLocation, location)
			return importedContractValue
		}
		vmConfig.CompositeTypeHandler = func(location common.Location, typeID interpreter.TypeID) *sema.CompositeType {
			elaboration := importedChecker.Elaboration
			compositeType := elaboration.CompositeType(typeID)
			if compositeType != nil {
				return compositeType
			}

			elaboration = checker.Elaboration
			compositeType = elaboration.CompositeType(typeID)
			if compositeType != nil {
				return compositeType
			}

			return nil
		}
		vmConfig.InterfaceTypeHandler = func(location common.Location, typeID interpreter.TypeID) *sema.InterfaceType {
			elaboration := importedChecker.Elaboration
			interfaceType := elaboration.InterfaceType(typeID)
			if interfaceType != nil {
				return interfaceType
			}

			elaboration = checker.Elaboration
			interfaceType = elaboration.InterfaceType(typeID)
			if interfaceType != nil {
				return interfaceType
			}

			return nil
		}

		vmInstance := vm.NewVM(scriptLocation(), program, vmConfig)

		result, err := vmInstance.InvokeExternally("test")
		require.NoError(t, err)
		require.Equal(t, 0, vmInstance.StackSize())

		require.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(5), result)
	})

	t.Run("aliased interface", func(t *testing.T) {

		t.Parallel()

		// Initialize Foo

		fooLocation := common.NewAddressLocation(
			nil,
			common.MustBytesToAddress([]byte{0x1}),
			"Foo",
		)

		fooChecker, err := ParseAndCheckWithOptions(t,
			`
            contract interface Foo {
                fun withdraw(_ amount: Int): String {
                    pre {
                        amount < 100: "Withdraw limit exceeds"
                    }
                }
            }
            `,
			ParseAndCheckOptions{
				Location: fooLocation,
			},
		)
		require.NoError(t, err)

		fooCompiler := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(fooChecker),
			fooChecker.Location,
		)
		fooProgram := fooCompiler.Compile()

		// Initialize Bar

		barLocation := common.NewAddressLocation(
			nil,
			common.MustBytesToAddress([]byte{0x2}),
			"Bar",
		)

		barChecker, err := ParseAndCheckWithOptions(t,
			`
            import Foo as F from 0x01

            contract Bar: F {
                init() {}
                fun withdraw(_ amount: Int): String {
                    return "Successfully withdrew"
                }
            }
            `,
			ParseAndCheckOptions{
				Location: barLocation,
				CheckerConfig: &sema.Config{
					LocationHandler: SingleIdentifierLocationResolver(t),
					ImportHandler: func(_ *sema.Checker, location common.Location, _ ast.Range) (sema.Import, error) {
						require.Equal(t, fooLocation, location)
						return sema.ElaborationImport{
							Elaboration: fooChecker.Elaboration,
						}, nil
					},
				},
			},
		)
		require.NoError(t, err)

		barCompiler := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(barChecker),
			barChecker.Location,
		)
		barCompiler.Config.LocationHandler = SingleIdentifierLocationResolver(t)
		barCompiler.Config.ImportHandler = func(location common.Location) *bbq.InstructionProgram {
			require.Equal(t, fooLocation, location)
			return fooProgram
		}

		barCompiler.Config.ElaborationResolver = func(location common.Location) (*compiler.DesugaredElaboration, error) {
			switch location {
			case fooLocation:
				return compiler.NewDesugaredElaboration(fooChecker.Elaboration), nil
			case barLocation:
				return compiler.NewDesugaredElaboration(barChecker.Elaboration), nil
			default:
				return nil, fmt.Errorf("cannot find elaboration for %s", location)
			}
		}

		barProgram := barCompiler.Compile()

		vmConfig := vm.NewConfig(NewUnmeteredInMemoryStorage())
		vmConfig.ImportHandler = func(location common.Location) *bbq.InstructionProgram {
			require.Equal(t, fooLocation, location)
			return fooProgram
		}
		vmConfig.BuiltinGlobalsProvider = VMBuiltinGlobalsProviderWithDefaultsAndPanic

		_, barContractValue := initializeContract(
			t,
			barLocation,
			barProgram,
			vmConfig,
		)

		// Compile and run main program

		checker, err := ParseAndCheckWithOptions(t,
			`
            import Bar from 0x02

            fun test(): String {
                return Bar.withdraw(50)
            }
            `,
			ParseAndCheckOptions{
				CheckerConfig: &sema.Config{
					LocationHandler: SingleIdentifierLocationResolver(t),
					ImportHandler: func(_ *sema.Checker, location common.Location, _ ast.Range) (sema.Import, error) {
						require.IsType(t, common.AddressLocation{}, location)
						addressLocation := location.(common.AddressLocation)
						var elaboration *sema.Elaboration
						switch addressLocation.Address {
						case fooLocation.Address:
							elaboration = fooChecker.Elaboration
						case barLocation.Address:
							elaboration = barChecker.Elaboration
						default:
							assert.FailNow(t, "invalid location")
						}

						return sema.ElaborationImport{
							Elaboration: elaboration,
						}, nil
					},
				},
			},
		)
		require.NoError(t, err)

		comp := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(checker),
			checker.Location,
		)
		comp.Config.LocationHandler = SingleIdentifierLocationResolver(t)
		comp.Config.ImportHandler = func(location common.Location) *bbq.InstructionProgram {
			switch location {
			case fooLocation:
				return fooProgram
			case barLocation:
				return barProgram
			default:
				assert.FailNow(t, "invalid location")
				return nil
			}
		}

		program := comp.Compile()

		vmConfig = vm.NewConfig(NewUnmeteredInMemoryStorage())
		vmConfig.ImportHandler = func(location common.Location) *bbq.InstructionProgram {
			switch location {
			case fooLocation:
				return fooProgram
			case barLocation:
				return barProgram
			default:
				assert.FailNow(t, "invalid location")
				return nil
			}
		}
		vmConfig.ContractValueHandler = func(_ *vm.Context, location common.Location) *interpreter.CompositeValue {
			switch location {
			case barLocation:
				return barContractValue
			default:
				assert.FailNow(t, fmt.Sprintf("invalid location %s", location))
				return nil
			}
		}
		vmConfig.CompositeTypeHandler = func(location common.Location, typeID interpreter.TypeID) *sema.CompositeType {
			require.Equal(t, barLocation, location)

			elaboration := barChecker.Elaboration
			return elaboration.CompositeType(typeID)
		}
		vmConfig.InterfaceTypeHandler = func(location common.Location, typeID interpreter.TypeID) *sema.InterfaceType {
			require.Equal(t, barLocation, location)

			elaboration := barChecker.Elaboration
			return elaboration.InterfaceType(typeID)
		}
		vmConfig.BuiltinGlobalsProvider = VMBuiltinGlobalsProviderWithDefaultsAndPanic

		vmInstance := vm.NewVM(scriptLocation(), program, vmConfig)

		result, err := vmInstance.InvokeExternally("test")
		require.NoError(t, err)
		require.Equal(t, 0, vmInstance.StackSize())

		require.Equal(t, interpreter.NewUnmeteredStringValue("Successfully withdrew"), result)
	})

	t.Run("multiple aliases, all declaration types", func(t *testing.T) {

		t.Parallel()

		address := common.MustBytesToAddress([]byte{0x1})

		fooLocation := common.AddressLocation{
			Address: address,
			Name:    "Foo",
		}
		fooChecker, err := ParseAndCheckWithOptions(t,
			`
              contract C {
                  fun v(): Int {
                      return 1
                  }
              }

              struct S {
                  fun v(): Int {
                      return 2
                  }
              }

              fun v(): Int {
                  return 3
              }
            `,
			ParseAndCheckOptions{
				Location: fooLocation,
			},
		)
		require.NoError(t, err)

		fooCompiler := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(fooChecker),
			fooChecker.Location,
		)
		fooProgram := fooCompiler.Compile()

		// Compile and run main program

		locationHandler := func(identifiers []ast.Identifier, location common.Location) (result []sema.ResolvedLocation, err error) {

			require.Equal(t,
				common.AddressLocation{
					Address: address,
					Name:    "",
				},
				location,
			)

			for _, identifier := range identifiers {
				result = append(result, sema.ResolvedLocation{
					Location: common.AddressLocation{
						Address: location.(common.AddressLocation).Address,
						Name:    identifier.Identifier,
					},
					Identifiers: []ast.Identifier{
						identifier,
					},
				})
			}
			return
		}

		checker, err := ParseAndCheckWithOptions(t,
			`
              import C as SomeContract, S as SomeStruct, v as SomeFunc from 0x01

              fun test(): Int {
                  return SomeContract.v() + SomeStruct().v() + SomeFunc()
              }
            `,
			ParseAndCheckOptions{
				CheckerConfig: &sema.Config{
					LocationHandler: locationHandler,
					ImportHandler: func(_ *sema.Checker, location common.Location, _ ast.Range) (sema.Import, error) {
						require.IsType(t, common.AddressLocation{}, location)
						addressLocation := location.(common.AddressLocation)
						var elaboration *sema.Elaboration
						switch addressLocation.Address {
						case address:
							elaboration = fooChecker.Elaboration
						default:
							assert.FailNow(t, "invalid location")
						}

						return sema.ElaborationImport{
							Elaboration: elaboration,
						}, nil
					},
				},
			},
		)
		require.NoError(t, err)

		comp := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(checker),
			checker.Location,
		)
		comp.Config.LocationHandler = locationHandler
		comp.Config.ImportHandler = func(location common.Location) *bbq.InstructionProgram {
			return fooProgram
		}

		program := comp.Compile()
		vmConfig := vm.NewConfig(NewUnmeteredInMemoryStorage())
		vmConfig.ImportHandler = func(location common.Location) *bbq.InstructionProgram {
			return fooProgram
		}
		vmConfig.BuiltinGlobalsProvider = VMBuiltinGlobalsProviderWithDefaultsAndPanic
		_, fooContractValue := initializeContract(
			t,
			fooLocation,
			fooProgram,
			vmConfig,
		)
		vmConfig.ContractValueHandler = func(_ *vm.Context, location common.Location) *interpreter.CompositeValue {
			require.Equal(t, fooLocation, location)
			return fooContractValue
		}
		vmConfig.CompositeTypeHandler = func(location common.Location, typeID interpreter.TypeID) *sema.CompositeType {
			require.Equal(t, fooChecker.Location, location)

			elaboration := fooChecker.Elaboration
			return elaboration.CompositeType(typeID)
		}
		vmConfig.InterfaceTypeHandler = func(location common.Location, typeID interpreter.TypeID) *sema.InterfaceType {
			require.Equal(t, fooChecker.Location, location)

			elaboration := fooChecker.Elaboration
			return elaboration.InterfaceType(typeID)
		}
		vmInstance := vm.NewVM(scriptLocation(), program, vmConfig)

		result, err := vmInstance.InvokeExternally("test")
		require.NoError(t, err)
		require.Equal(t, 0, vmInstance.StackSize())

		require.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(6), result)
	})

	t.Run("get type", func(t *testing.T) {

		t.Parallel()

		address := common.MustBytesToAddress([]byte{0x1})

		fooLocation := common.AddressLocation{
			Address: address,
			Name:    "S",
		}

		fooChecker, err := ParseAndCheckWithOptions(t,
			`
            struct S {}
            `,
			ParseAndCheckOptions{
				Location: fooLocation,
			},
		)
		require.NoError(t, err)

		fooCompiler := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(fooChecker),
			fooChecker.Location,
		)
		fooProgram := fooCompiler.Compile()

		// Compile and run main program

		locationHandler := SingleIdentifierLocationResolver(t)

		checker, err := ParseAndCheckWithOptions(t,
			`
            import S as SomeStruct from 0x01

            fun test(): String {
                var s = SomeStruct()
                return s.getType().identifier
            }
            `,
			ParseAndCheckOptions{
				CheckerConfig: &sema.Config{
					LocationHandler: locationHandler,
					ImportHandler: func(_ *sema.Checker, location common.Location, _ ast.Range) (sema.Import, error) {
						require.IsType(t, common.AddressLocation{}, location)
						addressLocation := location.(common.AddressLocation)
						var elaboration *sema.Elaboration
						switch addressLocation.Address {
						case address:
							elaboration = fooChecker.Elaboration
						default:
							assert.FailNow(t, "invalid location")
						}

						return sema.ElaborationImport{
							Elaboration: elaboration,
						}, nil
					},
				},
			},
		)
		require.NoError(t, err)

		comp := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(checker),
			checker.Location,
		)
		comp.Config.LocationHandler = locationHandler
		comp.Config.ImportHandler = func(location common.Location) *bbq.InstructionProgram {
			require.Equal(t, fooLocation, location)
			return fooProgram
		}

		program := comp.Compile()
		vmConfig := vm.NewConfig(NewUnmeteredInMemoryStorage())
		vmConfig.ImportHandler = func(location common.Location) *bbq.InstructionProgram {
			require.Equal(t, fooLocation, location)
			return fooProgram
		}
		vmConfig.BuiltinGlobalsProvider = VMBuiltinGlobalsProviderWithDefaultsAndPanic
		vmConfig.CompositeTypeHandler = func(location common.Location, typeID interpreter.TypeID) *sema.CompositeType {
			require.Equal(t, fooChecker.Location, location)

			elaboration := fooChecker.Elaboration
			return elaboration.CompositeType(typeID)
		}
		vmConfig.InterfaceTypeHandler = func(location common.Location, typeID interpreter.TypeID) *sema.InterfaceType {
			require.Equal(t, fooChecker.Location, location)

			elaboration := fooChecker.Elaboration
			return elaboration.InterfaceType(typeID)
		}
		vmInstance := vm.NewVM(scriptLocation(), program, vmConfig)

		result, err := vmInstance.InvokeExternally("test")
		require.NoError(t, err)
		require.Equal(t, 0, vmInstance.StackSize())

		require.Equal(t, interpreter.NewUnmeteredStringValue("A.0000000000000001.S"), result)
	})

	t.Run("nested type with same name as alias", func(t *testing.T) {

		t.Parallel()

		address := common.MustBytesToAddress([]byte{0x1})

		fooLocation := common.AddressLocation{
			Address: address,
			Name:    "Foo",
		}
		fooChecker, err := ParseAndCheckWithOptions(t,
			`
            access(all) contract Foo {
                access(all) struct Foo1 {
                    fun Test(): Int {
                        return 1
                    }
                }
            }
            `,
			ParseAndCheckOptions{
				Location: fooLocation,
			},
		)
		require.NoError(t, err)

		fooCompiler := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(fooChecker),
			fooChecker.Location,
		)
		fooProgram := fooCompiler.Compile()

		// Compile and run main program

		locationHandler := SingleIdentifierLocationResolver(t)

		checker, err := ParseAndCheckWithOptions(t,
			`
            import Foo as Foo1 from 0x01

            access(all) fun test(): Int {
                var foo = Foo1.Foo1()
                return foo.Test()
            }
            `,
			ParseAndCheckOptions{
				CheckerConfig: &sema.Config{
					LocationHandler: locationHandler,
					ImportHandler: func(_ *sema.Checker, location common.Location, _ ast.Range) (sema.Import, error) {
						require.IsType(t, common.AddressLocation{}, location)
						addressLocation := location.(common.AddressLocation)
						var elaboration *sema.Elaboration
						switch addressLocation.Address {
						case address:
							elaboration = fooChecker.Elaboration
						default:
							assert.FailNow(t, "invalid location")
						}

						return sema.ElaborationImport{
							Elaboration: elaboration,
						}, nil
					},
				},
			},
		)
		require.NoError(t, err)

		comp := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(checker),
			checker.Location,
		)
		comp.Config.LocationHandler = locationHandler
		comp.Config.ImportHandler = func(location common.Location) *bbq.InstructionProgram {
			require.Equal(t, fooLocation, location)
			return fooProgram
		}

		program := comp.Compile()
		vmConfig := vm.NewConfig(NewUnmeteredInMemoryStorage())
		vmConfig.BuiltinGlobalsProvider = VMBuiltinGlobalsProviderWithDefaultsAndPanic
		_, fooContractValue := initializeContract(
			t,
			fooLocation,
			fooProgram,
			vmConfig,
		)
		vmConfig.ImportHandler = func(location common.Location) *bbq.InstructionProgram {
			require.Equal(t, fooLocation, location)
			return fooProgram
		}
		vmConfig.ContractValueHandler = func(_ *vm.Context, location common.Location) *interpreter.CompositeValue {
			require.Equal(t, fooLocation, location)
			return fooContractValue
		}
		vmConfig.CompositeTypeHandler = func(location common.Location, typeID interpreter.TypeID) *sema.CompositeType {
			require.Equal(t, fooChecker.Location, location)

			elaboration := fooChecker.Elaboration
			return elaboration.CompositeType(typeID)
		}
		vmConfig.InterfaceTypeHandler = func(location common.Location, typeID interpreter.TypeID) *sema.InterfaceType {
			require.Equal(t, fooChecker.Location, location)

			elaboration := fooChecker.Elaboration
			return elaboration.InterfaceType(typeID)
		}

		vmInstance := vm.NewVM(scriptLocation(), program, vmConfig)

		result, err := vmInstance.InvokeExternally("test")
		require.NoError(t, err)
		require.Equal(t, 0, vmInstance.StackSize())

		require.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(1), result)
	})
}

func TestImportSameProgramFromMultiplePaths(t *testing.T) {

	t.Parallel()

	programs := map[common.Location]*CompiledProgram{}
	contractValues := map[common.Location]*interpreter.CompositeValue{}

	vmConfig := PrepareVMConfig(t, nil, programs)
	vmConfig.ContractValueHandler = func(context *vm.Context, location common.Location) *interpreter.CompositeValue {
		contractValue, ok := contractValues[location]
		if !ok {
			assert.FailNow(t, "invalid location")
		}
		return contractValue
	}

	// Program A

	locationA := common.NewAddressLocation(
		nil,
		common.MustBytesToAddress([]byte{0x1}),
		"A",
	)

	programA := ParseCheckAndCompileCodeWithOptions(t,
		`
            contract A {
                enum E: UInt8 {
                    case a
                    case b
                }
            }
        `,
		locationA,
		ParseCheckAndCompileOptions{},
		programs,
	)

	_, contractValueA := initializeContract(
		t,
		locationA,
		programA,
		vmConfig,
	)
	contractValues[locationA] = contractValueA

	// Program B
	locationB := common.NewAddressLocation(
		nil,
		common.MustBytesToAddress([]byte{0x2}),
		"B",
	)

	programB := ParseCheckAndCompileCodeWithOptions(t,
		`
            import A from 0x1

            contract B {
                fun foo() {
                    A.E.a
                }
            }
        `,
		locationB,
		ParseCheckAndCompileOptions{},
		programs,
	)

	_, contractValueB := initializeContract(
		t,
		locationB,
		programB,
		vmConfig,
	)
	contractValues[locationB] = contractValueB

	// Program C

	locationC := common.NewAddressLocation(
		nil,
		common.MustBytesToAddress([]byte{0x3}),
		"C",
	)

	programC := ParseCheckAndCompileCodeWithOptions(t,
		`
            import A from 0x1

            contract C {
                fun bar() {
                    A.E.b
                }
            }
        `,
		locationC,
		ParseCheckAndCompileOptions{},
		programs,
	)

	_, contractValueC := initializeContract(
		t,
		locationC,
		programC,
		vmConfig,
	)
	contractValues[locationC] = contractValueC

	// Test script
	_, err := CompileAndInvokeWithOptions(
		t,
		`
            import B from 0x2
            import C from 0x3

            fun test() {
                B.foo()
                C.bar()
            }
        `,
		"test",
		CompilerAndVMOptions{
			VMConfig: vmConfig,
			Programs: programs,
		},
	)

	require.NoError(t, err)
}

func TestBorrowContractLinksGlobals(t *testing.T) {

	t.Parallel()

	var logs []string
	conditionLogFunction := newConditionLogFunction(&logs)

	const functionName = "borrowContract"

	functionType := &sema.FunctionType{
		Purity:               sema.FunctionPurityView,
		ReturnTypeAnnotation: sema.VoidTypeAnnotation,
	}

	activation := sema.NewVariableActivation(sema.BaseValueActivation)
	activation.DeclareValue(stdlib.StandardLibraryValue{
		Name: functionName,
		Type: functionType,
		Kind: common.DeclarationKindFunction,
	})
	activation.DeclareValue(conditionLogFunction)

	const contractCode = `
      let ok = conditionLog("x")
    `

	contractAddress := common.MustBytesToAddress([]byte{0x1})
	const contractName = "Test"

	contractLocation := common.AddressLocation{
		Name:    contractName,
		Address: contractAddress,
	}

	importedChecker, err := ParseAndCheckWithOptions(t,
		contractCode,
		ParseAndCheckOptions{
			Location: contractLocation,
			CheckerConfig: &sema.Config{
				BaseValueActivationHandler: func(_ common.Location) *sema.VariableActivation {
					return activation
				},
			},
		},
	)
	require.NoError(t, err)

	compilerConfig := &compiler.Config{
		BuiltinGlobalsProvider: func(_ common.Location) *activations.Activation[compiler.GlobalImport] {
			activation := activations.NewActivation(nil, compiler.DefaultBuiltinGlobals())
			activation.Set(
				functionName,
				compiler.NewGlobalImport(functionName),
			)
			activation.Set(
				conditionLogFunctionName,
				compiler.NewGlobalImport(conditionLogFunctionName),
			)
			return activation
		},
	}

	subComp := compiler.NewInstructionCompilerWithConfig(
		interpreter.ProgramFromChecker(importedChecker),
		importedChecker.Location,
		compilerConfig,
	)
	subProgram := subComp.Compile()

	accountHandler := &testAccountHandler{
		getAccountContractCode: func(location common.AddressLocation) ([]byte, error) {
			assert.Equal(t,
				contractLocation,
				location,
			)
			return []byte(contractCode), nil
		},
	}

	functionValue := vm.NewNativeFunctionValue(
		functionName,
		functionType,
		func(
			context interpreter.NativeFunctionContext,
			_ interpreter.TypeArgumentsIterator,
			_ interpreter.ArgumentTypesIterator,
			_ interpreter.Value,
			args []interpreter.Value,
		) vm.Value {
			stdlib.AccountContractsBorrow(
				context,
				contractAddress,
				interpreter.NewUnmeteredStringValue(contractName),
				sema.NewReferenceType(nil, sema.UnauthorizedAccess, sema.AnyStructType),
				accountHandler,
			)

			return interpreter.Void
		},
	)

	vmConfig := vm.NewConfig(interpreter.NewInMemoryStorage(nil, nil))
	vmConfig.BuiltinGlobalsProvider = func(_ common.Location) *activations.Activation[vm.Variable] {
		activation := activations.NewActivation(nil, vm.DefaultBuiltinGlobals())

		variable := &interpreter.SimpleVariable{}
		variable.InitializeWithValue(functionValue)
		activation.Set(functionName, variable)

		logFunctionVariable := &interpreter.SimpleVariable{}
		logFunctionVariable.InitializeWithValue(conditionLogFunction.Value)
		activation.Set(conditionLogFunctionName, logFunctionVariable)

		return activation
	}
	vmConfig.ContractValueHandler = func(_ *vm.Context, location common.Location) *interpreter.CompositeValue {
		return nil
	}
	vmConfig.ImportHandler = func(location common.Location) *bbq.InstructionProgram {
		require.Equal(t,
			contractLocation,
			location,
		)
		return subProgram
	}

	_, err = CompileAndInvokeWithOptions(
		t,
		`
          fun test() {
              borrowContract()
          }
        `,
		"test",
		CompilerAndVMOptions{
			ParseCheckAndCompileOptions: ParseCheckAndCompileOptions{
				CompilerConfig: compilerConfig,
				ParseAndCheckOptions: &ParseAndCheckOptions{
					CheckerConfig: &sema.Config{
						BaseValueActivationHandler: func(location common.Location) *sema.VariableActivation {
							return activation
						},
					},
				},
			},
			VMConfig: vmConfig,
		},
	)
	require.NoError(t, err)

	assert.Equal(t, []string{`"x"`}, logs)
}

func TestInitializerFunctionType(t *testing.T) {

	t.Parallel()

	programVM, err := CompileAndPrepareToInvoke(t,
		`
          struct S {}
          struct T {}

          fun getTypeOfConstructorS(): Type {
              return S.getType()
          }

          fun getTypeOfConstructorT(): Type {
              return T.getType()
          }
        `,
		CompilerAndVMOptions{},
	)
	require.NoError(t, err)

	result, err := programVM.InvokeExternally("getTypeOfConstructorS")
	if err == nil {
		require.Equal(t, 0, programVM.StackSize())
	}
	metaType := result.(interpreter.TypeValue)
	assert.Equal(t, common.TypeID("view fun():S.test.S"), metaType.Type.ID())

	result, err = programVM.InvokeExternally("getTypeOfConstructorT")
	if err == nil {
		require.Equal(t, 0, programVM.StackSize())
	}
	metaType = result.(interpreter.TypeValue)
	assert.Equal(t, common.TypeID("view fun():S.test.T"), metaType.Type.ID())
}
