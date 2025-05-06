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
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/atree"

	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/bbq"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/test_utils/runtime_utils"
	. "github.com/onflow/cadence/test_utils/sema_utils"

	"github.com/onflow/cadence/bbq/compiler"
	"github.com/onflow/cadence/bbq/vm"
)

func BenchmarkRecursionFib(b *testing.B) {

	scriptLocation := runtime_utils.NewScriptLocationGenerator()

	checker, err := ParseAndCheck(b, recursiveFib)
	require.NoError(b, err)

	comp := compiler.NewInstructionCompiler(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
	)
	program := comp.Compile()

	vmConfig := &vm.Config{}
	vmInstance := vm.NewVM(scriptLocation(), program, vmConfig)

	expected := interpreter.NewUnmeteredIntValueFromInt64(377)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {

		result, err := vmInstance.Invoke(
			"fib",
			interpreter.NewUnmeteredIntValueFromInt64(14),
		)
		require.NoError(b, err)
		require.Equal(b, expected, result)
	}
}

func BenchmarkImperativeFib(b *testing.B) {

	scriptLocation := runtime_utils.NewScriptLocationGenerator()

	checker, err := ParseAndCheck(b, imperativeFib)
	require.NoError(b, err)

	comp := compiler.NewInstructionCompiler(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
	)
	program := comp.Compile()

	vmConfig := &vm.Config{}
	vmInstance := vm.NewVM(scriptLocation(), program, vmConfig)

	b.ReportAllocs()
	b.ResetTimer()

	var value vm.Value = interpreter.NewUnmeteredIntValueFromInt64(14)

	for i := 0; i < b.N; i++ {
		_, err := vmInstance.Invoke("fib", value)
		require.NoError(b, err)
	}
}

func BenchmarkNewStruct(b *testing.B) {

	scriptLocation := runtime_utils.NewScriptLocationGenerator()

	checker, err := ParseAndCheck(b, `
      struct Foo {
          var id : Int

          init(_ id: Int) {
              self.id = id
          }
      }

      fun test(count: Int) {
          var i = 0
          while i < count {
              Foo(i)
              i = i + 1
          }
      }
  `)
	require.NoError(b, err)

	value := interpreter.NewUnmeteredIntValueFromInt64(10)

	comp := compiler.NewInstructionCompiler(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
	)
	program := comp.Compile()

	vmConfig := &vm.Config{}
	vmInstance := vm.NewVM(scriptLocation(), program, vmConfig)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := vmInstance.Invoke("test", value)
		require.NoError(b, err)
	}
}

func BenchmarkNewResource(b *testing.B) {

	checker, err := ParseAndCheck(b, `
      resource Foo {
          var id : Int

          init(_ id: Int) {
              self.id = id
          }
      }

      fun test(count: Int): @Foo {
          var i = 0
          var r <- create Foo(0)
          while i < count {
              i = i + 1
              destroy create Foo(i)
          }
          return <- r
      }
    `)
	require.NoError(b, err)

	b.ReportAllocs()
	b.ResetTimer()

	value := interpreter.NewUnmeteredIntValueFromInt64(9)

	scriptLocation := runtime_utils.NewScriptLocationGenerator()

	for i := 0; i < b.N; i++ {
		comp := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(checker),
			checker.Location,
		)
		program := comp.Compile()

		vmConfig := &vm.Config{}
		vmInstance := vm.NewVM(scriptLocation(), program, vmConfig)
		_, err := vmInstance.Invoke("test", value)
		require.NoError(b, err)
	}
}

func BenchmarkNewStructRaw(b *testing.B) {

	storage := interpreter.NewInMemoryStorage(nil)
	vmConfig := vm.NewContext(
		vm.NewConfig(storage),
	)

	fieldValue := interpreter.NewUnmeteredIntValueFromInt64(7)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < 1; j++ {
			structValue := interpreter.NewCompositeValue(
				vmConfig,
				vm.EmptyLocationRange,
				nil,
				"Foo",
				common.CompositeKindStructure,
				nil,
				common.ZeroAddress,
			)
			structValue.SetMember(vmConfig, vm.EmptyLocationRange, "id", fieldValue)
			structValue.Transfer(
				vmConfig,
				vm.EmptyLocationRange,
				atree.Address{},
				false,
				nil,
				nil,
				true,
			)
		}
	}
}

func BenchmarkContractImport(b *testing.B) {

	location := common.NewAddressLocation(nil, common.Address{0x1}, "MyContract")

	importedChecker, err := ParseAndCheckWithOptions(b,
		`
      contract MyContract {
          var s: String

          fun helloText(): String {
              return self.s
          }

          init() {
              self.s = "contract function of the imported program"
          }

          struct Foo {
              var id : String

              init(_ id: String) {
                  self.id = id
              }

              fun sayHello(_ id: Int): String {
                  // return self.id
                  return MyContract.helloText()
              }
          }
      }
        `,
		ParseAndCheckOptions{
			Location: location,
		},
	)
	require.NoError(b, err)

	importCompiler := compiler.NewInstructionCompiler(
		interpreter.ProgramFromChecker(importedChecker),
		importedChecker.Location,
	)
	importedProgram := importCompiler.Compile()

	_, importedContractValue := initializeContract(
		b,
		location,
		importedProgram,
		nil,
	)

	vmConfig := &vm.Config{
		ImportHandler: func(location common.Location) *bbq.InstructionProgram {
			return importedProgram
		},
		ContractValueHandler: func(_ *vm.Config, _ common.Location) *interpreter.CompositeValue {
			return importedContractValue
		},
	}

	b.ResetTimer()
	b.ReportAllocs()

	value := interpreter.NewUnmeteredIntValueFromInt64(7)

	for i := 0; i < b.N; i++ {
		checker, err := ParseAndCheckWithOptions(b,
			`
              import MyContract from 0x01

              fun test(count: Int): String {
                  var i = 0
                  var r = MyContract.Foo("Hello from Foo!")
                  while i < count {
                      i = i + 1
                      r = MyContract.Foo("Hello from Foo!")
                      r.sayHello(1)
                  }
                  return r.sayHello(1)
              }
            `,
			ParseAndCheckOptions{
				Config: &sema.Config{
					ImportHandler: func(*sema.Checker, common.Location, ast.Range) (sema.Import, error) {
						return sema.ElaborationImport{
							Elaboration: importedChecker.Elaboration,
						}, nil
					},
				},
			},
		)
		require.NoError(b, err)

		comp := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(checker),
			checker.Location,
		)
		comp.Config.ImportHandler = func(location common.Location) *bbq.InstructionProgram {
			return importedProgram
		}
		program := comp.Compile()

		scriptLocation := runtime_utils.NewScriptLocationGenerator()

		vmInstance := vm.NewVM(scriptLocation(), program, vmConfig)
		_, err = vmInstance.Invoke("test", value)
		require.NoError(b, err)
	}
}

func BenchmarkMethodCall(b *testing.B) {

	b.Run("interface method call", func(b *testing.B) {
		location := common.NewAddressLocation(nil, common.Address{0x1}, "MyContract")

		importedChecker, err := ParseAndCheckWithOptions(b,
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
				Location: location,
			},
		)
		require.NoError(b, err)

		importCompiler := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(importedChecker),
			importedChecker.Location,
		)
		importedProgram := importCompiler.Compile()


		_, importedContractValue := initializeContract(
			b,
			location,
			importedProgram,
			nil,
		)

		checker, err := ParseAndCheckWithOptions(b,
			`
              import MyContract from 0x01

              fun test(count: Int) {
                  var r: {MyContract.Greetings} = MyContract.Foo("Hello from Foo!")
                  var i = 0
                  while i < count {
                      i = i + 1
                      r.sayHello(1)
                  }
              }
            `,
			ParseAndCheckOptions{
				Config: &sema.Config{
					ImportHandler: func(*sema.Checker, common.Location, ast.Range) (sema.Import, error) {
						return sema.ElaborationImport{
							Elaboration: importedChecker.Elaboration,
						}, nil
					},
				},
			},
		)
		require.NoError(b, err)

		comp := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(checker),
			checker.Location,
		)
		comp.Config.ImportHandler = func(location common.Location) *bbq.InstructionProgram {
			return importedProgram
		}

		program := comp.Compile()

		vmConfig := &vm.Config{
			ImportHandler: func(location common.Location) *bbq.InstructionProgram {
				return importedProgram
			},
			ContractValueHandler: func(_ *vm.Config, _ common.Location) *interpreter.CompositeValue {
				return importedContractValue
			},
			TypeLoader: func(location common.Location, typeID interpreter.TypeID) sema.ContainedType {
				elaboration := importedChecker.Elaboration
				compositeType := elaboration.CompositeType(typeID)
				if compositeType != nil {
					return compositeType
				}

				return elaboration.InterfaceType(typeID)
			},
		}

		scriptLocation := runtime_utils.NewScriptLocationGenerator()

		value := interpreter.NewUnmeteredIntValueFromInt64(10)

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			vmInstance := vm.NewVM(scriptLocation(), program, vmConfig)
			_, err := vmInstance.Invoke("test", value)
			require.NoError(b, err)
		}
	})

	b.Run("concrete type method call", func(b *testing.B) {
		location := common.NewAddressLocation(nil, common.Address{0x1}, "MyContract")

		importedChecker, err := ParseAndCheckWithOptions(b,
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
				Location: location,
			},
		)
		require.NoError(b, err)

		importCompiler := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(importedChecker),
			importedChecker.Location,
		)
		importedProgram := importCompiler.Compile()


		_, importedContractValue := initializeContract(
			b,
			location,
			importedProgram,
			nil,
		)

		checker, err := ParseAndCheckWithOptions(b, `
        import MyContract from 0x01

        fun test(count: Int) {
            var r: MyContract.Foo = MyContract.Foo("Hello from Foo!")
            var i = 0
            while i < count {
                i = i + 1
                r.sayHello(1)
            }
        }`,

			ParseAndCheckOptions{
				Config: &sema.Config{
					ImportHandler: func(*sema.Checker, common.Location, ast.Range) (sema.Import, error) {
						return sema.ElaborationImport{
							Elaboration: importedChecker.Elaboration,
						}, nil
					},
				},
			},
		)
		require.NoError(b, err)

		comp := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(checker),
			checker.Location,
		)
		comp.Config.ImportHandler = func(location common.Location) *bbq.InstructionProgram {
			return importedProgram
		}

		program := comp.Compile()

		vmConfig := &vm.Config{
			ImportHandler: func(location common.Location) *bbq.InstructionProgram {
				return importedProgram
			},
			ContractValueHandler: func(_ *vm.Config, _ common.Location) *interpreter.CompositeValue {
				return importedContractValue
			},
		}

		scriptLocation := runtime_utils.NewScriptLocationGenerator()

		vmInstance := vm.NewVM(scriptLocation(), program, vmConfig)

		value := interpreter.NewUnmeteredIntValueFromInt64(10)

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_, err := vmInstance.Invoke("test", value)
			require.NoError(b, err)
		}
	})
}
