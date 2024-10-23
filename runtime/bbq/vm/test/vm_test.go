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
 */

package test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/atree"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/bbq/vm"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib"
	. "github.com/onflow/cadence/runtime/tests/checker"
	"github.com/onflow/cadence/runtime/tests/utils"

	"github.com/onflow/cadence/runtime/bbq"
	"github.com/onflow/cadence/runtime/bbq/commons"
	"github.com/onflow/cadence/runtime/bbq/compiler"
)

const recursiveFib = `
  fun fib(_ n: Int): Int {
      if n < 2 {
         return n
      }
      return fib(n - 1) + fib(n - 2)
  }
`

func TestRecursionFib(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, recursiveFib)
	require.NoError(t, err)

	comp := compiler.NewCompiler(checker.Program, checker.Elaboration)
	program := comp.Compile()

	vmConfig := &vm.Config{}
	vmInstance := vm.NewVM(program, vmConfig)

	result, err := vmInstance.Invoke(
		"fib",
		vm.IntValue{SmallInt: 7},
	)
	require.NoError(t, err)
	require.Equal(t, vm.IntValue{SmallInt: 13}, result)
	require.Equal(t, 0, vmInstance.StackSize())
}

func BenchmarkRecursionFib(b *testing.B) {

	checker, err := ParseAndCheck(b, recursiveFib)
	require.NoError(b, err)

	comp := compiler.NewCompiler(checker.Program, checker.Elaboration)
	program := comp.Compile()

	vmConfig := &vm.Config{}
	vmInstance := vm.NewVM(program, vmConfig)

	b.ReportAllocs()
	b.ResetTimer()

	expected := vm.IntValue{SmallInt: 377}

	for i := 0; i < b.N; i++ {

		result, err := vmInstance.Invoke(
			"fib",
			vm.IntValue{SmallInt: 14},
		)
		require.NoError(b, err)
		require.Equal(b, expected, result)
	}
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

	comp := compiler.NewCompiler(checker.Program, checker.Elaboration)
	program := comp.Compile()

	vmConfig := &vm.Config{}
	vmInstance := vm.NewVM(program, vmConfig)

	result, err := vmInstance.Invoke(
		"fib",
		vm.IntValue{SmallInt: 7},
	)
	require.NoError(t, err)
	require.Equal(t, vm.IntValue{SmallInt: 13}, result)
	require.Equal(t, 0, vmInstance.StackSize())
}

func BenchmarkImperativeFib(b *testing.B) {

	checker, err := ParseAndCheck(b, imperativeFib)
	require.NoError(b, err)

	comp := compiler.NewCompiler(checker.Program, checker.Elaboration)
	program := comp.Compile()

	vmConfig := &vm.Config{}
	vmInstance := vm.NewVM(program, vmConfig)

	b.ReportAllocs()
	b.ResetTimer()

	var value vm.Value = vm.IntValue{SmallInt: 14}

	for i := 0; i < b.N; i++ {
		_, err := vmInstance.Invoke("fib", value)
		require.NoError(b, err)
	}
}

func TestBreak(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
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
  `)
	require.NoError(t, err)

	comp := compiler.NewCompiler(checker.Program, checker.Elaboration)
	program := comp.Compile()

	vmConfig := &vm.Config{}
	vmInstance := vm.NewVM(program, vmConfig)

	result, err := vmInstance.Invoke("test")
	require.NoError(t, err)

	require.Equal(t, vm.IntValue{SmallInt: 4}, result)
	require.Equal(t, 0, vmInstance.StackSize())
}

func TestContinue(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
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
  `)
	require.NoError(t, err)

	comp := compiler.NewCompiler(checker.Program, checker.Elaboration)
	program := comp.Compile()

	vmConfig := &vm.Config{}
	vmInstance := vm.NewVM(program, vmConfig)

	result, err := vmInstance.Invoke("test")
	require.NoError(t, err)

	require.Equal(t, vm.IntValue{SmallInt: 3}, result)
	require.Equal(t, 0, vmInstance.StackSize())
}

func TestNilCoalesce(t *testing.T) {

	t.Parallel()

	t.Run("true", func(t *testing.T) {
		t.Parallel()

		checker, err := ParseAndCheck(t, `
            fun test(): Int {
                var i: Int? = 2
                var j = i ?? 3
                return j
            }
        `)
		require.NoError(t, err)

		comp := compiler.NewCompiler(checker.Program, checker.Elaboration)
		program := comp.Compile()

		vmConfig := &vm.Config{}
		vmInstance := vm.NewVM(program, vmConfig)

		result, err := vmInstance.Invoke("test")
		require.NoError(t, err)

		require.Equal(t, vm.IntValue{SmallInt: 2}, result)
		require.Equal(t, 0, vmInstance.StackSize())
	})

	t.Run("false", func(t *testing.T) {
		t.Parallel()

		checker, err := ParseAndCheck(t, `
            fun test(): Int {
                var i: Int? = nil
                var j = i ?? 3
                return j
            }
        `)
		require.NoError(t, err)

		comp := compiler.NewCompiler(checker.Program, checker.Elaboration)
		program := comp.Compile()
		printProgram(program)

		vmConfig := &vm.Config{}
		vmInstance := vm.NewVM(program, vmConfig)

		result, err := vmInstance.Invoke("test")
		require.NoError(t, err)

		require.Equal(t, vm.IntValue{SmallInt: 3}, result)
		require.Equal(t, 0, vmInstance.StackSize())
	})
}

func TestNewStruct(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
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
  `)
	require.NoError(t, err)

	comp := compiler.NewCompiler(checker.Program, checker.Elaboration)
	program := comp.Compile()

	vmConfig := &vm.Config{}
	vmInstance := vm.NewVM(program, vmConfig)

	result, err := vmInstance.Invoke("test", vm.IntValue{SmallInt: 10})
	require.NoError(t, err)
	require.Equal(t, 0, vmInstance.StackSize())

	require.IsType(t, &vm.CompositeValue{}, result)
	structValue := result.(*vm.CompositeValue)

	require.Equal(t, "Foo", structValue.QualifiedIdentifier)
	require.Equal(
		t,
		vm.IntValue{SmallInt: 12},
		structValue.GetMember(vmConfig, "id"),
	)
}

func TestStructMethodCall(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
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
  `)
	require.NoError(t, err)

	comp := compiler.NewCompiler(checker.Program, checker.Elaboration)
	program := comp.Compile()

	vmConfig := &vm.Config{}
	vmInstance := vm.NewVM(program, vmConfig)

	result, err := vmInstance.Invoke("test")
	require.NoError(t, err)
	require.Equal(t, 0, vmInstance.StackSize())

	require.Equal(t, vm.StringValue{Str: []byte("Hello from Foo!")}, result)
}

func BenchmarkNewStruct(b *testing.B) {

	checker, err := ParseAndCheck(b, `
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
          }
          return r
      }
  `)
	require.NoError(b, err)

	value := vm.IntValue{SmallInt: 1}

	b.ReportAllocs()
	b.ResetTimer()

	comp := compiler.NewCompiler(checker.Program, checker.Elaboration)
	program := comp.Compile()

	vmConfig := &vm.Config{}
	vmInstance := vm.NewVM(program, vmConfig)

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

	value := vm.IntValue{SmallInt: 9}

	for i := 0; i < b.N; i++ {
		comp := compiler.NewCompiler(checker.Program, checker.Elaboration)
		program := comp.Compile()

		vmConfig := &vm.Config{}
		vmInstance := vm.NewVM(program, vmConfig)
		_, err := vmInstance.Invoke("test", value)
		require.NoError(b, err)
	}
}

func BenchmarkNewStructRaw(b *testing.B) {

	storage := interpreter.NewInMemoryStorage(nil)
	vmConfig := &vm.Config{
		Storage: storage,
	}

	fieldValue := vm.IntValue{SmallInt: 7}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < 1; j++ {
			structValue := vm.NewCompositeValue(
				nil,
				"Foo",
				common.CompositeKindStructure,
				common.Address{},
				storage.BasicSlabStorage,
			)
			structValue.SetMember(vmConfig, "id", fieldValue)
			structValue.Transfer(vmConfig, atree.Address{}, false, nil)
		}
	}
}

func TestImport(t *testing.T) {

	t.Parallel()

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
			Location: utils.ImportedLocation,
		},
	)
	require.NoError(t, err)

	subComp := compiler.NewCompiler(importedChecker.Program, importedChecker.Elaboration)
	importedProgram := subComp.Compile()

	checker, err := ParseAndCheckWithOptions(t, `
      import Foo from 0x01

      fun test(): String {
          var r = Foo("Hello from Foo!")
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
	require.NoError(t, err)

	importCompiler := compiler.NewCompiler(checker.Program, checker.Elaboration)
	importCompiler.Config.ImportHandler = func(location common.Location) *bbq.Program {
		return importedProgram
	}

	program := importCompiler.Compile()

	vmConfig := &vm.Config{
		ImportHandler: func(location common.Location) *bbq.Program {
			return importedProgram
		},
	}

	vmInstance := vm.NewVM(program, vmConfig)

	result, err := vmInstance.Invoke("test")
	require.NoError(t, err)
	require.Equal(t, 0, vmInstance.StackSize())

	require.Equal(t, vm.StringValue{Str: []byte("global function of the imported program")}, result)
}

func TestContractImport(t *testing.T) {

	t.Parallel()

	t.Run("nested type def", func(t *testing.T) {

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
				Location: common.NewAddressLocation(nil, common.Address{0x1}, "MyContract"),
			},
		)
		require.NoError(t, err)

		importCompiler := compiler.NewCompiler(importedChecker.Program, importedChecker.Elaboration)
		importedProgram := importCompiler.Compile()

		vmInstance := vm.NewVM(importedProgram, nil)
		importedContractValue, err := vmInstance.InitializeContract()
		require.NoError(t, err)

		checker, err := ParseAndCheckWithOptions(t, `
      import MyContract from 0x01

      fun test(): String {
          var r = MyContract.Foo("Hello from Foo!")
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
		require.NoError(t, err)

		comp := compiler.NewCompiler(checker.Program, checker.Elaboration)
		comp.Config.ImportHandler = func(location common.Location) *bbq.Program {
			return importedProgram
		}

		program := comp.Compile()

		vmConfig := &vm.Config{
			ImportHandler: func(location common.Location) *bbq.Program {
				return importedProgram
			},
			ContractValueHandler: func(*vm.Config, common.Location) *vm.CompositeValue {
				return importedContractValue
			},
		}

		vmInstance = vm.NewVM(program, vmConfig)

		result, err := vmInstance.Invoke("test")
		require.NoError(t, err)
		require.Equal(t, 0, vmInstance.StackSize())

		require.Equal(t, vm.StringValue{Str: []byte("global function of the imported program")}, result)
	})

	t.Run("contract function", func(t *testing.T) {

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
				Location: common.NewAddressLocation(nil, common.Address{0x1}, "MyContract"),
			},
		)
		require.NoError(t, err)

		importCompiler := compiler.NewCompiler(importedChecker.Program, importedChecker.Elaboration)
		importedProgram := importCompiler.Compile()

		vmInstance := vm.NewVM(importedProgram, nil)
		importedContractValue, err := vmInstance.InitializeContract()
		require.NoError(t, err)

		checker, err := ParseAndCheckWithOptions(t, `
      import MyContract from 0x01

      fun test(): String {
          return MyContract.helloText()
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
		require.NoError(t, err)

		comp := compiler.NewCompiler(checker.Program, checker.Elaboration)
		comp.Config.ImportHandler = func(location common.Location) *bbq.Program {
			return importedProgram
		}

		program := comp.Compile()

		vmConfig := &vm.Config{
			ImportHandler: func(location common.Location) *bbq.Program {
				return importedProgram
			},
			ContractValueHandler: func(*vm.Config, common.Location) *vm.CompositeValue {
				return importedContractValue
			},
		}

		vmInstance = vm.NewVM(program, vmConfig)

		result, err := vmInstance.Invoke("test")
		require.NoError(t, err)
		require.Equal(t, 0, vmInstance.StackSize())

		require.Equal(t, vm.StringValue{Str: []byte("contract function of the imported program")}, result)
	})

	t.Run("nested imports", func(t *testing.T) {

		// Initialize Foo

		fooLocation := common.NewAddressLocation(
			nil,
			common.Address{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1},
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
            }`,
			ParseAndCheckOptions{
				Location: fooLocation,
			},
		)
		require.NoError(t, err)

		fooCompiler := compiler.NewCompiler(fooChecker.Program, fooChecker.Elaboration)
		fooProgram := fooCompiler.Compile()

		vmInstance := vm.NewVM(fooProgram, nil)
		fooContractValue, err := vmInstance.InitializeContract()
		require.NoError(t, err)

		// Initialize Bar

		barLocation := common.NewAddressLocation(
			nil,
			common.Address{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x2},
			"Bar",
		)

		barChecker, err := ParseAndCheckWithOptions(t, `
            import Foo from 0x01

            contract Bar {
                init() {}
                fun sayHello(): String {
                    return Foo.sayHello()
                }
            }`,
			ParseAndCheckOptions{
				Location: barLocation,
				Config: &sema.Config{
					ImportHandler: func(_ *sema.Checker, location common.Location, _ ast.Range) (sema.Import, error) {
						require.Equal(t, fooLocation, location)
						return sema.ElaborationImport{
							Elaboration: fooChecker.Elaboration,
						}, nil
					},
					LocationHandler: singleIdentifierLocationResolver(t),
				},
			},
		)
		require.NoError(t, err)

		barCompiler := compiler.NewCompiler(barChecker.Program, barChecker.Elaboration)
		barCompiler.Config.LocationHandler = singleIdentifierLocationResolver(t)
		barCompiler.Config.ImportHandler = func(location common.Location) *bbq.Program {
			require.Equal(t, fooLocation, location)
			return fooProgram
		}

		barProgram := barCompiler.Compile()

		vmConfig := &vm.Config{
			ImportHandler: func(location common.Location) *bbq.Program {
				require.Equal(t, fooLocation, location)
				return fooProgram
			},
			ContractValueHandler: func(_ *vm.Config, location common.Location) *vm.CompositeValue {
				require.Equal(t, fooLocation, location)
				return fooContractValue
			},
		}

		vmInstance = vm.NewVM(barProgram, vmConfig)
		barContractValue, err := vmInstance.InitializeContract()
		require.NoError(t, err)

		// Compile and run main program

		checker, err := ParseAndCheckWithOptions(t, `
            import Bar from 0x02

            fun test(): String {
                return Bar.sayHello()
            }`,
			ParseAndCheckOptions{
				Config: &sema.Config{
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
					LocationHandler: singleIdentifierLocationResolver(t),
				},
			},
		)
		require.NoError(t, err)

		comp := compiler.NewCompiler(checker.Program, checker.Elaboration)
		comp.Config.LocationHandler = singleIdentifierLocationResolver(t)
		comp.Config.ImportHandler = func(location common.Location) *bbq.Program {
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

		vmConfig = &vm.Config{
			ImportHandler: func(location common.Location) *bbq.Program {
				switch location {
				case fooLocation:
					return fooProgram
				case barLocation:
					return barProgram
				default:
					assert.FailNow(t, "invalid location")
					return nil
				}
			},
			ContractValueHandler: func(_ *vm.Config, location common.Location) *vm.CompositeValue {
				switch location {
				case fooLocation:
					return fooContractValue
				case barLocation:
					return barContractValue
				default:
					assert.FailNow(t, "invalid location")
					return nil
				}
			},
		}

		vmInstance = vm.NewVM(program, vmConfig)

		result, err := vmInstance.Invoke("test")
		require.NoError(t, err)
		require.Equal(t, 0, vmInstance.StackSize())

		require.Equal(t, vm.StringValue{Str: []byte("Hello from Foo!")}, result)
	})

	t.Run("contract interface", func(t *testing.T) {

		// Initialize Foo

		fooLocation := common.NewAddressLocation(
			nil,
			common.Address{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1},
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
            }`,
			ParseAndCheckOptions{
				Location: fooLocation,
			},
		)
		require.NoError(t, err)

		fooCompiler := compiler.NewCompiler(fooChecker.Program, fooChecker.Elaboration)
		fooProgram := fooCompiler.Compile()

		//vmInstance := NewVM(fooProgram, nil)
		//fooContractValue, err := vmInstance.InitializeContract()
		//require.NoError(t, err)

		// Initialize Bar

		barLocation := common.NewAddressLocation(
			nil,
			common.Address{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x2},
			"Bar",
		)

		barChecker, err := ParseAndCheckWithOptions(t, `
            import Foo from 0x01

            contract Bar: Foo {
                init() {}
                fun withdraw(_ amount: Int): String {
                    return "Successfully withdrew"
                }
            }`,
			ParseAndCheckOptions{
				Location: barLocation,
				Config: &sema.Config{
					ImportHandler: func(_ *sema.Checker, location common.Location, _ ast.Range) (sema.Import, error) {
						require.Equal(t, fooLocation, location)
						return sema.ElaborationImport{
							Elaboration: fooChecker.Elaboration,
						}, nil
					},
					LocationHandler: singleIdentifierLocationResolver(t),
				},
			},
		)
		require.NoError(t, err)

		barCompiler := compiler.NewCompiler(barChecker.Program, barChecker.Elaboration)
		barCompiler.Config.LocationHandler = singleIdentifierLocationResolver(t)
		barCompiler.Config.ImportHandler = func(location common.Location) *bbq.Program {
			require.Equal(t, fooLocation, location)
			return fooProgram
		}

		barProgram := barCompiler.Compile()

		vmConfig := &vm.Config{
			ImportHandler: func(location common.Location) *bbq.Program {
				require.Equal(t, fooLocation, location)
				return fooProgram
			},
			//ContractValueHandler: func(_ *Config, location common.Location) *CompositeValue {
			//	require.Equal(t, fooLocation, location)
			//	return fooContractValue
			//},
		}

		vmInstance := vm.NewVM(barProgram, vmConfig)
		barContractValue, err := vmInstance.InitializeContract()
		require.NoError(t, err)

		// Compile and run main program

		checker, err := ParseAndCheckWithOptions(t, `
            import Bar from 0x02

            fun test(): String {
                return Bar.withdraw(150)
            }`,
			ParseAndCheckOptions{
				Config: &sema.Config{
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
					LocationHandler: singleIdentifierLocationResolver(t),
				},
			},
		)
		require.NoError(t, err)

		comp := compiler.NewCompiler(checker.Program, checker.Elaboration)
		comp.Config.LocationHandler = singleIdentifierLocationResolver(t)
		comp.Config.ImportHandler = func(location common.Location) *bbq.Program {
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

		vmConfig = &vm.Config{
			ImportHandler: func(location common.Location) *bbq.Program {
				switch location {
				case fooLocation:
					return fooProgram
				case barLocation:
					return barProgram
				default:
					assert.FailNow(t, "invalid location")
					return nil
				}
			},
			ContractValueHandler: func(_ *vm.Config, location common.Location) *vm.CompositeValue {
				switch location {
				//case fooLocation:
				//	return fooContractValue
				case barLocation:
					return barContractValue
				default:
					assert.FailNow(t, fmt.Sprintf("invalid location %s", location))
					return nil
				}
			},
		}

		vmInstance = vm.NewVM(program, vmConfig)

		result, err := vmInstance.Invoke("test")
		require.NoError(t, err)
		require.Equal(t, 0, vmInstance.StackSize())

		require.Equal(t, vm.StringValue{Str: []byte("Successfully withdrew")}, result)
	})
}

func BenchmarkContractImport(b *testing.B) {

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
			Location: common.NewAddressLocation(nil, common.Address{0x1}, "MyContract"),
		},
	)
	require.NoError(b, err)

	importCompiler := compiler.NewCompiler(importedChecker.Program, importedChecker.Elaboration)
	importedProgram := importCompiler.Compile()

	vmInstance := vm.NewVM(importedProgram, nil)
	importedContractValue, err := vmInstance.InitializeContract()
	require.NoError(b, err)

	vmConfig := &vm.Config{
		ImportHandler: func(location common.Location) *bbq.Program {
			return importedProgram
		},
		ContractValueHandler: func(vmConfig *vm.Config, location common.Location) *vm.CompositeValue {
			return importedContractValue
		},
	}

	b.ResetTimer()
	b.ReportAllocs()

	value := vm.IntValue{SmallInt: 7}

	for i := 0; i < b.N; i++ {
		checker, err := ParseAndCheckWithOptions(b, `
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

		comp := compiler.NewCompiler(checker.Program, checker.Elaboration)
		comp.Config.ImportHandler = func(location common.Location) *bbq.Program {
			return importedProgram
		}
		program := comp.Compile()

		vmInstance := vm.NewVM(program, vmConfig)
		_, err = vmInstance.Invoke("test", value)
		require.NoError(b, err)
	}
}

func TestInitializeContract(t *testing.T) {

	checker, err := ParseAndCheckWithOptions(t,
		`
      contract MyContract {
          var status : String
          init() {
              self.status = "PENDING"
          }
      }
        `,
		ParseAndCheckOptions{
			Location: common.NewAddressLocation(nil, common.Address{0x1}, "MyContract"),
		},
	)
	require.NoError(t, err)

	comp := compiler.NewCompiler(checker.Program, checker.Elaboration)
	program := comp.Compile()

	vmConfig := &vm.Config{}
	vmInstance := vm.NewVM(program, vmConfig)
	contractValue, err := vmInstance.InitializeContract()
	require.NoError(t, err)

	fieldValue := contractValue.GetMember(vmConfig, "status")
	assert.Equal(t, vm.StringValue{Str: []byte("PENDING")}, fieldValue)
}

func TestContractAccessDuringInit(t *testing.T) {

	t.Parallel()

	t.Run("using contract name", func(t *testing.T) {
		t.Parallel()

		checker, err := ParseAndCheckWithOptions(t, `
            contract MyContract {
                var status : String

                access(all) fun getInitialStatus(): String {
                    return "PENDING"
                }

                init() {
                    self.status = MyContract.getInitialStatus()
                }
            }`,
			ParseAndCheckOptions{
				Location: common.NewAddressLocation(nil, common.Address{0x1}, "MyContract"),
			},
		)
		require.NoError(t, err)

		comp := compiler.NewCompiler(checker.Program, checker.Elaboration)
		program := comp.Compile()

		vmConfig := &vm.Config{}
		vmInstance := vm.NewVM(program, vmConfig)
		contractValue, err := vmInstance.InitializeContract()
		require.NoError(t, err)

		fieldValue := contractValue.GetMember(vmConfig, "status")
		assert.Equal(t, vm.StringValue{Str: []byte("PENDING")}, fieldValue)
	})

	t.Run("using self", func(t *testing.T) {
		t.Parallel()

		checker, err := ParseAndCheckWithOptions(t, `
            contract MyContract {
                var status : String

                access(all) fun getInitialStatus(): String {
                    return "PENDING"
                }

                init() {
                    self.status = self.getInitialStatus()
                }
            }`,
			ParseAndCheckOptions{
				Location: common.NewAddressLocation(nil, common.Address{0x1}, "MyContract"),
			},
		)
		require.NoError(t, err)

		comp := compiler.NewCompiler(checker.Program, checker.Elaboration)
		program := comp.Compile()

		vmConfig := &vm.Config{}
		vmInstance := vm.NewVM(program, vmConfig)
		contractValue, err := vmInstance.InitializeContract()
		require.NoError(t, err)

		fieldValue := contractValue.GetMember(vmConfig, "status")
		assert.Equal(t, vm.StringValue{Str: []byte("PENDING")}, fieldValue)
	})
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
      }`)
		require.NoError(t, err)

		comp := compiler.NewCompiler(checker.Program, checker.Elaboration)
		program := comp.Compile()

		vmConfig := &vm.Config{}
		vmInstance := vm.NewVM(program, vmConfig)

		result, err := vmInstance.Invoke("test")
		require.NoError(t, err)
		require.Equal(t, 0, vmInstance.StackSize())

		require.Equal(t, vm.IntValue{SmallInt: 5}, result)
	})

	t.Run("nested", func(t *testing.T) {
		t.Parallel()

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
      }`

		checker, err := ParseAndCheckWithOptions(
			t,
			code,
			ParseAndCheckOptions{
				Location: common.NewAddressLocation(nil, common.Address{0x1}, "MyContract"),
			},
		)
		require.NoError(t, err)

		comp := compiler.NewCompiler(checker.Program, checker.Elaboration)
		program := comp.Compile()

		vmConfig := &vm.Config{}
		vmInstance := vm.NewVM(program, vmConfig)

		result, err := vmInstance.Invoke("init")
		require.NoError(t, err)
		require.Equal(t, 0, vmInstance.StackSize())

		require.IsType(t, &vm.CompositeValue{}, result)
	})
}

func TestContractField(t *testing.T) {

	t.Parallel()

	t.Run("get", func(t *testing.T) {

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
				Location: common.NewAddressLocation(nil, common.Address{0x1}, "MyContract"),
			},
		)
		require.NoError(t, err)

		importCompiler := compiler.NewCompiler(importedChecker.Program, importedChecker.Elaboration)
		importedProgram := importCompiler.Compile()

		vmInstance := vm.NewVM(importedProgram, nil)
		importedContractValue, err := vmInstance.InitializeContract()
		require.NoError(t, err)

		checker, err := ParseAndCheckWithOptions(t, `
      import MyContract from 0x01

      fun test(): String {
          return MyContract.status
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
		require.NoError(t, err)

		comp := compiler.NewCompiler(checker.Program, checker.Elaboration)
		comp.Config.ImportHandler = func(location common.Location) *bbq.Program {
			return importedProgram
		}

		program := comp.Compile()

		vmConfig := &vm.Config{
			ImportHandler: func(location common.Location) *bbq.Program {
				return importedProgram
			},
			ContractValueHandler: func(vmConfig *vm.Config, location common.Location) *vm.CompositeValue {
				return importedContractValue
			},
		}

		vmInstance = vm.NewVM(program, vmConfig)
		result, err := vmInstance.Invoke("test")
		require.NoError(t, err)
		require.Equal(t, 0, vmInstance.StackSize())

		require.Equal(t, vm.StringValue{Str: []byte("PENDING")}, result)
	})

	t.Run("set", func(t *testing.T) {

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
				Location: common.NewAddressLocation(nil, common.Address{0x1}, "MyContract"),
			},
		)
		require.NoError(t, err)

		importCompiler := compiler.NewCompiler(importedChecker.Program, importedChecker.Elaboration)
		importedProgram := importCompiler.Compile()

		vmInstance := vm.NewVM(importedProgram, nil)
		importedContractValue, err := vmInstance.InitializeContract()
		require.NoError(t, err)

		checker, err := ParseAndCheckWithOptions(t, `
      import MyContract from 0x01

      fun test(): String {
          MyContract.status = "UPDATED"
          return MyContract.status
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
		require.NoError(t, err)

		comp := compiler.NewCompiler(checker.Program, checker.Elaboration)
		comp.Config.ImportHandler = func(location common.Location) *bbq.Program {
			return importedProgram
		}

		program := comp.Compile()

		vmConfig := &vm.Config{
			ImportHandler: func(location common.Location) *bbq.Program {
				return importedProgram
			},
			ContractValueHandler: func(vmConfig *vm.Config, location common.Location) *vm.CompositeValue {
				return importedContractValue
			},
		}

		vmInstance = vm.NewVM(program, vmConfig)

		result, err := vmInstance.Invoke("test")
		require.NoError(t, err)
		require.Equal(t, 0, vmInstance.StackSize())

		require.Equal(t, vm.StringValue{Str: []byte("UPDATED")}, result)

		fieldValue := importedContractValue.GetMember(vmConfig, "status")
		assert.Equal(t, vm.StringValue{Str: []byte("UPDATED")}, fieldValue)
	})
}

func TestNativeFunctions(t *testing.T) {

	t.Parallel()

	t.Run("static function", func(t *testing.T) {

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
			nil,
		)

		baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
		baseValueActivation.DeclareValue(logFunction)

		checker, err := ParseAndCheckWithOptions(t, `
            fun test() {
                log("Hello, World!")
            }`,
			ParseAndCheckOptions{
				Config: &sema.Config{
					BaseValueActivationHandler: func(common.Location) *sema.VariableActivation {
						return baseValueActivation
					},
				},
			},
		)
		require.NoError(t, err)

		comp := compiler.NewCompiler(checker.Program, checker.Elaboration)
		program := comp.Compile()
		printProgram(program)

		vmConfig := &vm.Config{}
		vmInstance := vm.NewVM(program, vmConfig)

		_, err = vmInstance.Invoke("test")
		require.NoError(t, err)
		require.Equal(t, 0, vmInstance.StackSize())
	})

	t.Run("bound function", func(t *testing.T) {
		checker, err := ParseAndCheck(t, `
            fun test(): String {
                return "Hello".concat(", World!")
            }`,
		)
		require.NoError(t, err)

		comp := compiler.NewCompiler(checker.Program, checker.Elaboration)
		program := comp.Compile()
		printProgram(program)

		vmConfig := &vm.Config{}
		vmInstance := vm.NewVM(program, vmConfig)

		result, err := vmInstance.Invoke("test")
		require.NoError(t, err)
		require.Equal(t, 0, vmInstance.StackSize())

		require.Equal(t, vm.StringValue{Str: []byte("Hello, World!")}, result)
	})
}

func TestTransaction(t *testing.T) {

	t.Parallel()

	t.Run("simple", func(t *testing.T) {

		checker, err := ParseAndCheck(t, `
            transaction {
				var a: String
                prepare() {
                    self.a = "Hello!"
                }
                execute {
                    self.a = "Hello again!"
                }
            }`,
		)
		require.NoError(t, err)

		comp := compiler.NewCompiler(checker.Program, checker.Elaboration)
		program := comp.Compile()
		printProgram(program)

		vmConfig := &vm.Config{}
		vmInstance := vm.NewVM(program, vmConfig)

		err = vmInstance.ExecuteTransaction(nil)
		require.NoError(t, err)
		require.Equal(t, 0, vmInstance.StackSize())

		// Rerun the same again using internal functions, to get the access to the transaction value.

		transaction, err := vmInstance.Invoke(commons.TransactionWrapperCompositeName)
		require.NoError(t, err)

		require.IsType(t, &vm.CompositeValue{}, transaction)
		compositeValue := transaction.(*vm.CompositeValue)

		// At the beginning, 'a' is uninitialized
		assert.Nil(t, compositeValue.GetMember(vmConfig, "a"))

		// Invoke 'prepare'
		_, err = vmInstance.Invoke(commons.TransactionPrepareFunctionName, transaction)
		require.NoError(t, err)

		// Once 'prepare' is called, 'a' is initialized to "Hello!"
		assert.Equal(t, vm.StringValue{Str: []byte("Hello!")}, compositeValue.GetMember(vmConfig, "a"))

		// Invoke 'execute'
		_, err = vmInstance.Invoke(commons.TransactionExecuteFunctionName, transaction)
		require.NoError(t, err)

		// Once 'execute' is called, 'a' is initialized to "Hello, again!"
		assert.Equal(t, vm.StringValue{Str: []byte("Hello again!")}, compositeValue.GetMember(vmConfig, "a"))
	})

	t.Run("with params", func(t *testing.T) {

		checker, err := ParseAndCheck(t, `
            transaction(param1: String, param2: String) {
				var a: String
                prepare() {
                    self.a = param1
                }
                execute {
                    self.a = param2
                }
            }`,
		)
		require.NoError(t, err)

		comp := compiler.NewCompiler(checker.Program, checker.Elaboration)
		program := comp.Compile()
		printProgram(program)

		vmConfig := &vm.Config{}
		vmInstance := vm.NewVM(program, vmConfig)

		args := []vm.Value{
			vm.StringValue{[]byte("Hello!")},
			vm.StringValue{[]byte("Hello again!")},
		}

		err = vmInstance.ExecuteTransaction(args)
		require.NoError(t, err)
		require.Equal(t, 0, vmInstance.StackSize())

		// Rerun the same again using internal functions, to get the access to the transaction value.

		transaction, err := vmInstance.Invoke(commons.TransactionWrapperCompositeName)
		require.NoError(t, err)

		require.IsType(t, &vm.CompositeValue{}, transaction)
		compositeValue := transaction.(*vm.CompositeValue)

		// At the beginning, 'a' is uninitialized
		assert.Nil(t, compositeValue.GetMember(vmConfig, "a"))

		// Invoke 'prepare'
		_, err = vmInstance.Invoke(commons.TransactionPrepareFunctionName, transaction)
		require.NoError(t, err)

		// Once 'prepare' is called, 'a' is initialized to "Hello!"
		assert.Equal(t, vm.StringValue{Str: []byte("Hello!")}, compositeValue.GetMember(vmConfig, "a"))

		// Invoke 'execute'
		_, err = vmInstance.Invoke(commons.TransactionExecuteFunctionName, transaction)
		require.NoError(t, err)

		// Once 'execute' is called, 'a' is initialized to "Hello, again!"
		assert.Equal(t, vm.StringValue{Str: []byte("Hello again!")}, compositeValue.GetMember(vmConfig, "a"))
	})
}

func TestInterfaceMethodCall(t *testing.T) {

	t.Parallel()

	location := common.NewAddressLocation(
		nil,
		common.Address{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1},
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
			Location: location,
		},
	)
	require.NoError(t, err)

	importCompiler := compiler.NewCompiler(importedChecker.Program, importedChecker.Elaboration)
	importedProgram := importCompiler.Compile()

	vmInstance := vm.NewVM(importedProgram, nil)
	importedContractValue, err := vmInstance.InitializeContract()
	require.NoError(t, err)

	checker, err := ParseAndCheckWithOptions(t, `
        import MyContract from 0x01

        fun test(): String {
            var r: {MyContract.Greetings} = MyContract.Foo("Hello from Foo!")
            // first call must link
            r.sayHello(1)

            // second call should pick from the cache
            return r.sayHello(1)
        }`,

		ParseAndCheckOptions{
			Config: &sema.Config{
				ImportHandler: func(*sema.Checker, common.Location, ast.Range) (sema.Import, error) {
					return sema.ElaborationImport{
						Elaboration: importedChecker.Elaboration,
					}, nil
				},
				LocationHandler: singleIdentifierLocationResolver(t),
			},
		},
	)
	require.NoError(t, err)

	comp := compiler.NewCompiler(checker.Program, checker.Elaboration)
	comp.Config.LocationHandler = singleIdentifierLocationResolver(t)
	comp.Config.ImportHandler = func(location common.Location) *bbq.Program {
		return importedProgram
	}

	program := comp.Compile()

	vmConfig := &vm.Config{
		ImportHandler: func(location common.Location) *bbq.Program {
			return importedProgram
		},
		ContractValueHandler: func(vmConfig *vm.Config, location common.Location) *vm.CompositeValue {
			return importedContractValue
		},
	}

	vmInstance = vm.NewVM(program, vmConfig)
	result, err := vmInstance.Invoke("test")
	require.NoError(t, err)
	require.Equal(t, 0, vmInstance.StackSize())

	require.Equal(t, vm.StringValue{Str: []byte("Hello from Foo!")}, result)
}

func BenchmarkMethodCall(b *testing.B) {

	b.Run("interface method call", func(b *testing.B) {

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
				Location: common.NewAddressLocation(nil, common.Address{0x1}, "MyContract"),
			},
		)
		require.NoError(b, err)

		importCompiler := compiler.NewCompiler(importedChecker.Program, importedChecker.Elaboration)
		importedProgram := importCompiler.Compile()

		vmInstance := vm.NewVM(importedProgram, nil)
		importedContractValue, err := vmInstance.InitializeContract()
		require.NoError(b, err)

		checker, err := ParseAndCheckWithOptions(b, `
        import MyContract from 0x01

        fun test(count: Int) {
            var r: {MyContract.Greetings} = MyContract.Foo("Hello from Foo!")
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

		comp := compiler.NewCompiler(checker.Program, checker.Elaboration)
		comp.Config.ImportHandler = func(location common.Location) *bbq.Program {
			return importedProgram
		}

		program := comp.Compile()

		vmConfig := &vm.Config{
			ImportHandler: func(location common.Location) *bbq.Program {
				return importedProgram
			},
			ContractValueHandler: func(vmConfig *vm.Config, location common.Location) *vm.CompositeValue {
				return importedContractValue
			},
		}

		vmInstance = vm.NewVM(program, vmConfig)

		value := vm.IntValue{SmallInt: 10}

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_, err := vmInstance.Invoke("test", value)
			require.NoError(b, err)
		}
	})

	b.Run("concrete type method call", func(b *testing.B) {

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
				Location: common.NewAddressLocation(nil, common.Address{0x1}, "MyContract"),
			},
		)
		require.NoError(b, err)

		importCompiler := compiler.NewCompiler(importedChecker.Program, importedChecker.Elaboration)
		importedProgram := importCompiler.Compile()

		vmInstance := vm.NewVM(importedProgram, nil)
		importedContractValue, err := vmInstance.InitializeContract()
		require.NoError(b, err)

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

		comp := compiler.NewCompiler(checker.Program, checker.Elaboration)
		comp.Config.ImportHandler = func(location common.Location) *bbq.Program {
			return importedProgram
		}

		program := comp.Compile()

		vmConfig := &vm.Config{
			ImportHandler: func(location common.Location) *bbq.Program {
				return importedProgram
			},
			ContractValueHandler: func(vmConfig *vm.Config, location common.Location) *vm.CompositeValue {
				return importedContractValue
			},
		}

		vmInstance = vm.NewVM(program, vmConfig)

		value := vm.IntValue{SmallInt: 10}

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_, err := vmInstance.Invoke("test", value)
			require.NoError(b, err)
		}
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

		comp := compiler.NewCompiler(checker.Program, checker.Elaboration)
		program := comp.Compile()

		vmConfig := &vm.Config{}
		vmInstance := vm.NewVM(program, vmConfig)

		result, err := vmInstance.Invoke("test")
		require.NoError(t, err)
		require.Equal(t, 0, vmInstance.StackSize())

		require.IsType(t, &vm.ArrayValue{}, result)
		array := result.(*vm.ArrayValue)
		assert.Equal(t, 2, array.Count())
		assert.Equal(t, vm.IntValue{SmallInt: 2}, array.Get(vmConfig, 0))
		assert.Equal(t, vm.IntValue{SmallInt: 5}, array.Get(vmConfig, 1))
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

		comp := compiler.NewCompiler(checker.Program, checker.Elaboration)
		program := comp.Compile()

		vmConfig := &vm.Config{}
		vmInstance := vm.NewVM(program, vmConfig)

		result, err := vmInstance.Invoke("test")
		require.NoError(t, err)
		require.Equal(t, 0, vmInstance.StackSize())
		assert.Equal(t, vm.IntValue{SmallInt: 5}, result)
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

		comp := compiler.NewCompiler(checker.Program, checker.Elaboration)
		program := comp.Compile()

		vmConfig := &vm.Config{}
		vmInstance := vm.NewVM(program, vmConfig)

		result, err := vmInstance.Invoke("test")
		require.NoError(t, err)
		require.Equal(t, 0, vmInstance.StackSize())

		require.IsType(t, &vm.ArrayValue{}, result)
		array := result.(*vm.ArrayValue)
		assert.Equal(t, 3, array.Count())
		assert.Equal(t, vm.IntValue{SmallInt: 2}, array.Get(vmConfig, 0))
		assert.Equal(t, vm.IntValue{SmallInt: 5}, array.Get(vmConfig, 1))
		assert.Equal(t, vm.IntValue{SmallInt: 8}, array.Get(vmConfig, 2))
	})
}

func TestReference(t *testing.T) {

	t.Parallel()

	t.Run("method call", func(t *testing.T) {
		t.Parallel()

		checker, err := ParseAndCheck(t, `
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
        `)
		require.NoError(t, err)

		comp := compiler.NewCompiler(checker.Program, checker.Elaboration)
		program := comp.Compile()

		vmConfig := vm.NewConfig(nil)
		vmInstance := vm.NewVM(program, vmConfig)

		result, err := vmInstance.Invoke("test")
		require.NoError(t, err)
		require.Equal(t, 0, vmInstance.StackSize())

		require.Equal(t, vm.StringValue{Str: []byte("Hello from Foo!")}, result)
	})
}
