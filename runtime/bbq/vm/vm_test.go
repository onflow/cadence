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

package vm

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/atree"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
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

	vm := NewVM(program, nil)

	result, err := vm.Invoke(
		"fib",
		IntValue{SmallInt: 7},
	)
	require.NoError(t, err)
	require.Equal(t, IntValue{SmallInt: 13}, result)
}

func BenchmarkRecursionFib(b *testing.B) {

	checker, err := ParseAndCheck(b, recursiveFib)
	require.NoError(b, err)

	comp := compiler.NewCompiler(checker.Program, checker.Elaboration)
	program := comp.Compile()

	vm := NewVM(program, nil)

	b.ReportAllocs()
	b.ResetTimer()

	expected := IntValue{SmallInt: 377}

	for i := 0; i < b.N; i++ {

		result, err := vm.Invoke(
			"fib",
			IntValue{SmallInt: 14},
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

	vm := NewVM(program, nil)

	result, err := vm.Invoke(
		"fib",
		IntValue{SmallInt: 7},
	)
	require.NoError(t, err)
	require.Equal(t, IntValue{SmallInt: 13}, result)
}

func BenchmarkImperativeFib(b *testing.B) {

	checker, err := ParseAndCheck(b, imperativeFib)
	require.NoError(b, err)

	comp := compiler.NewCompiler(checker.Program, checker.Elaboration)
	program := comp.Compile()

	vm := NewVM(program, nil)

	b.ReportAllocs()
	b.ResetTimer()

	var value Value = IntValue{SmallInt: 14}

	for i := 0; i < b.N; i++ {
		_, err := vm.Invoke("fib", value)
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

	vm := NewVM(program, nil)

	result, err := vm.Invoke("test")
	require.NoError(t, err)

	require.Equal(t, IntValue{SmallInt: 4}, result)
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

	vm := NewVM(program, nil)

	result, err := vm.Invoke("test")
	require.NoError(t, err)

	require.Equal(t, IntValue{SmallInt: 3}, result)
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

	vm := NewVM(program, nil)

	result, err := vm.Invoke("test", IntValue{SmallInt: 10})
	require.NoError(t, err)

	require.IsType(t, &CompositeValue{}, result)
	structValue := result.(*CompositeValue)

	require.Equal(t, "Foo", structValue.QualifiedIdentifier)
	require.Equal(
		t,
		IntValue{SmallInt: 12},
		structValue.GetMember(vm.config, "id"),
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

	vm := NewVM(program, nil)

	result, err := vm.Invoke("test")
	require.NoError(t, err)

	require.Equal(t, StringValue{String: []byte("Hello from Foo!")}, result)
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

	value := IntValue{SmallInt: 1}

	b.ReportAllocs()
	b.ResetTimer()

	comp := compiler.NewCompiler(checker.Program, checker.Elaboration)
	program := comp.Compile()

	vm := NewVM(program, nil)

	for i := 0; i < b.N; i++ {
		_, err := vm.Invoke("test", value)
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

	value := IntValue{SmallInt: 9}

	for i := 0; i < b.N; i++ {
		comp := compiler.NewCompiler(checker.Program, checker.Elaboration)
		program := comp.Compile()

		vm := NewVM(program, nil)
		_, err := vm.Invoke("test", value)
		require.NoError(b, err)
	}
}

func BenchmarkNewStructRaw(b *testing.B) {

	storage := interpreter.NewInMemoryStorage(nil)
	conf := &Config{
		Storage: storage,
	}

	fieldValue := IntValue{SmallInt: 7}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < 1; j++ {
			structValue := NewCompositeValue(
				nil,
				"Foo",
				common.CompositeKindStructure,
				common.Address{},
				storage.BasicSlabStorage,
			)
			structValue.SetMember(conf, "id", fieldValue)
			structValue.Transfer(conf, atree.Address{}, false, nil)
		}
	}
}

func printProgram(program *bbq.Program) {
	byteCodePrinter := &bbq.BytecodePrinter{}
	fmt.Println(byteCodePrinter.PrintProgram(program))
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

	vmConfig := &Config{
		ImportHandler: func(location common.Location) *bbq.Program {
			return importedProgram
		},
	}

	vm := NewVM(program, vmConfig)

	result, err := vm.Invoke("test")
	require.NoError(t, err)

	require.Equal(t, StringValue{String: []byte("global function of the imported program")}, result)
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

		vm := NewVM(importedProgram, nil)
		importedContractValue, err := vm.InitializeContract()
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

		vmConfig := &Config{
			ImportHandler: func(location common.Location) *bbq.Program {
				return importedProgram
			},
			ContractValueHandler: func(*Config, common.Location) *CompositeValue {
				return importedContractValue
			},
		}

		vm = NewVM(program, vmConfig)

		result, err := vm.Invoke("test")
		require.NoError(t, err)
		require.Equal(t, StringValue{String: []byte("global function of the imported program")}, result)
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

		vm := NewVM(importedProgram, nil)
		importedContractValue, err := vm.InitializeContract()
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

		vmConfig := &Config{
			ImportHandler: func(location common.Location) *bbq.Program {
				return importedProgram
			},
			ContractValueHandler: func(*Config, common.Location) *CompositeValue {
				return importedContractValue
			},
		}

		vm = NewVM(program, vmConfig)

		result, err := vm.Invoke("test")
		require.NoError(t, err)
		require.Equal(t, StringValue{String: []byte("contract function of the imported program")}, result)
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

		vm := NewVM(fooProgram, nil)
		fooContractValue, err := vm.InitializeContract()
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

		vmConfig := &Config{
			ImportHandler: func(location common.Location) *bbq.Program {
				require.Equal(t, fooLocation, location)
				return fooProgram
			},
			ContractValueHandler: func(_ *Config, location common.Location) *CompositeValue {
				require.Equal(t, fooLocation, location)
				return fooContractValue
			},
		}

		vm = NewVM(barProgram, vmConfig)
		barContractValue, err := vm.InitializeContract()
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

		vmConfig = &Config{
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
			ContractValueHandler: func(_ *Config, location common.Location) *CompositeValue {
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

		vm = NewVM(program, vmConfig)

		result, err := vm.Invoke("test")
		require.NoError(t, err)
		require.Equal(t, StringValue{String: []byte("Hello from Foo!")}, result)
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

	vm := NewVM(importedProgram, nil)
	importedContractValue, err := vm.InitializeContract()
	require.NoError(b, err)

	vmConfig := &Config{
		ImportHandler: func(location common.Location) *bbq.Program {
			return importedProgram
		},
		ContractValueHandler: func(conf *Config, location common.Location) *CompositeValue {
			return importedContractValue
		},
	}

	b.ResetTimer()
	b.ReportAllocs()

	value := IntValue{SmallInt: 7}

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

		vm := NewVM(program, vmConfig)
		_, err = vm.Invoke("test", value)
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

	vm := NewVM(program, nil)
	contractValue, err := vm.InitializeContract()
	require.NoError(t, err)

	fieldValue := contractValue.GetMember(vm.config, "status")
	assert.Equal(t, StringValue{String: []byte("PENDING")}, fieldValue)
}

func TestContractAccessDuringInit(t *testing.T) {

	t.Parallel()

	t.Run("using contract name", func(t *testing.T) {
		t.Parallel()

		checker, err := ParseAndCheckWithOptions(t, `
            contract MyContract {
                var status : String

                pub fun getInitialStatus(): String {
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

		vm := NewVM(program, nil)
		contractValue, err := vm.InitializeContract()
		require.NoError(t, err)

		fieldValue := contractValue.GetMember(vm.config, "status")
		assert.Equal(t, StringValue{String: []byte("PENDING")}, fieldValue)
	})

	t.Run("using self", func(t *testing.T) {
		t.Parallel()

		checker, err := ParseAndCheckWithOptions(t, `
            contract MyContract {
                var status : String

                pub fun getInitialStatus(): String {
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

		vm := NewVM(program, nil)
		contractValue, err := vm.InitializeContract()
		require.NoError(t, err)

		fieldValue := contractValue.GetMember(vm.config, "status")
		assert.Equal(t, StringValue{String: []byte("PENDING")}, fieldValue)
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

		vm := NewVM(program, nil)

		result, err := vm.Invoke("test")
		require.NoError(t, err)

		require.Equal(t, IntValue{SmallInt: 5}, result)
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

		vm := NewVM(program, nil)

		result, err := vm.Invoke("init")
		require.NoError(t, err)

		require.IsType(t, &CompositeValue{}, result)
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

		vm := NewVM(importedProgram, nil)
		importedContractValue, err := vm.InitializeContract()
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

		vmConfig := &Config{
			ImportHandler: func(location common.Location) *bbq.Program {
				return importedProgram
			},
			ContractValueHandler: func(conf *Config, location common.Location) *CompositeValue {
				return importedContractValue
			},
		}

		vm = NewVM(program, vmConfig)
		result, err := vm.Invoke("test")
		require.NoError(t, err)
		require.Equal(t, StringValue{String: []byte("PENDING")}, result)
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

		vm := NewVM(importedProgram, nil)
		importedContractValue, err := vm.InitializeContract()
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

		vmConfig := &Config{
			ImportHandler: func(location common.Location) *bbq.Program {
				return importedProgram
			},
			ContractValueHandler: func(conf *Config, location common.Location) *CompositeValue {
				return importedContractValue
			},
		}

		vm = NewVM(program, vmConfig)

		result, err := vm.Invoke("test")
		require.NoError(t, err)
		require.Equal(t, StringValue{String: []byte("UPDATED")}, result)

		fieldValue := importedContractValue.GetMember(vm.config, "status")
		assert.Equal(t, StringValue{String: []byte("UPDATED")}, fieldValue)
	})
}

func TestEvaluationOrder(t *testing.T) {
	f := Foo{"pending"}
	f.GetFoo().printArgument(getArg())
}

type Foo struct {
	id string
}

func (f Foo) GetFoo() Foo {
	fmt.Println("evaluating receiver")
	return f
}

func (f Foo) printArgument(s string) {
	fmt.Println(s)
}

func getArg() string {
	fmt.Println("evaluating argument")
	return "argument"
}

func singleIdentifierLocationResolver(t testing.TB) func(
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
