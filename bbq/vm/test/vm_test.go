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

	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/stdlib"
	. "github.com/onflow/cadence/test_utils/common_utils"
	. "github.com/onflow/cadence/test_utils/interpreter_utils"
	. "github.com/onflow/cadence/test_utils/runtime_utils"
	. "github.com/onflow/cadence/test_utils/sema_utils"

	"github.com/onflow/cadence/bbq"
	"github.com/onflow/cadence/bbq/commons"
	"github.com/onflow/cadence/bbq/compiler"
	"github.com/onflow/cadence/bbq/vm"
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

	comp := compiler.NewInstructionCompiler(checker)
	program := comp.Compile()

	vmConfig := &vm.Config{}
	vmInstance := vm.NewVM(scriptLocation(), program, vmConfig)

	result, err := vmInstance.Invoke(
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

	comp := compiler.NewInstructionCompiler(checker)
	program := comp.Compile()

	vmConfig := &vm.Config{}
	vmInstance := vm.NewVM(scriptLocation(), program, vmConfig)

	result, err := vmInstance.Invoke(
		"fib",
		interpreter.NewUnmeteredIntValueFromInt64(7),
	)
	require.NoError(t, err)
	require.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(13), result)
	require.Equal(t, 0, vmInstance.StackSize())
}

func TestWhileBreak(t *testing.T) {

	t.Parallel()

	result, err := compileAndInvoke(t,
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
		result, err := compileAndInvoke(t,
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
		result, err := compileAndInvoke(t,
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

	result, err := compileAndInvoke(t,
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
		actual, err := compileAndInvoke(t,
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

	vmConfig := &vm.Config{}

	vmInstance := CompileAndPrepareToInvoke(t,
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

	result, err := vmInstance.Invoke("test", interpreter.NewUnmeteredIntValueFromInt64(10))
	require.NoError(t, err)
	require.Equal(t, 0, vmInstance.StackSize())

	require.IsType(t, &interpreter.CompositeValue{}, result)
	structValue := result.(*interpreter.CompositeValue)
	compositeType := structValue.StaticType(vmConfig).(*interpreter.CompositeStaticType)

	require.Equal(t, "Foo", compositeType.QualifiedIdentifier)
	require.Equal(
		t,
		interpreter.NewUnmeteredIntValueFromInt64(12),
		structValue.GetMember(vmConfig, vm.EmptyLocationRange, "id"),
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

	comp := compiler.NewInstructionCompiler(checker)
	program := comp.Compile()

	vmConfig := &vm.Config{}
	vmInstance := vm.NewVM(scriptLocation(), program, vmConfig)

	result, err := vmInstance.Invoke("test")
	require.NoError(t, err)
	require.Equal(t, 0, vmInstance.StackSize())

	require.Equal(t, interpreter.NewUnmeteredStringValue("Hello from Foo!"), result)
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
			Location: ImportedLocation,
		},
	)
	require.NoError(t, err)

	subComp := compiler.NewInstructionCompiler(importedChecker)
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

	importCompiler := compiler.NewInstructionCompiler(checker)
	importCompiler.Config.ImportHandler = func(location common.Location) *bbq.InstructionProgram {
		return importedProgram
	}

	program := importCompiler.Compile()

	vmConfig := &vm.Config{
		ImportHandler: func(location common.Location) *bbq.InstructionProgram {
			return importedProgram
		},
	}

	vmInstance := vm.NewVM(scriptLocation(), program, vmConfig)

	result, err := vmInstance.Invoke("test")
	require.NoError(t, err)
	require.Equal(t, 0, vmInstance.StackSize())

	require.Equal(t, interpreter.NewUnmeteredStringValue("global function of the imported program"), result)
}

func TestContractImport(t *testing.T) {

	t.Parallel()

	t.Run("nested type def", func(t *testing.T) {

		t.Parallel()

		importLocation := common.NewAddressLocation(nil, common.Address{0x1}, "MyContract")

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

		importCompiler := compiler.NewInstructionCompiler(importedChecker)
		importedProgram := importCompiler.Compile()

		vmInstance := vm.NewVM(importLocation, importedProgram, nil)
		importedContractValue, err := vmInstance.InitializeContract()
		require.NoError(t, err)

		checker, err := ParseAndCheckWithOptions(t,
			`
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

		comp := compiler.NewInstructionCompiler(checker)
		comp.Config.ImportHandler = func(location common.Location) *bbq.InstructionProgram {
			return importedProgram
		}

		program := comp.Compile()

		vmConfig := &vm.Config{
			ImportHandler: func(location common.Location) *bbq.InstructionProgram {
				return importedProgram
			},
			ContractValueHandler: func(*vm.Config, common.Location) *interpreter.CompositeValue {
				return importedContractValue
			},
		}

		vmInstance = vm.NewVM(scriptLocation(), program, vmConfig)

		result, err := vmInstance.Invoke("test")
		require.NoError(t, err)
		require.Equal(t, 0, vmInstance.StackSize())

		require.Equal(t, interpreter.NewUnmeteredStringValue("global function of the imported program"), result)
	})

	t.Run("contract function", func(t *testing.T) {

		t.Parallel()

		importLocation := common.NewAddressLocation(nil, common.Address{0x1}, "MyContract")

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

		importCompiler := compiler.NewInstructionCompiler(importedChecker)
		importedProgram := importCompiler.Compile()

		vmInstance := vm.NewVM(importLocation, importedProgram, nil)
		importedContractValue, err := vmInstance.InitializeContract()
		require.NoError(t, err)

		checker, err := ParseAndCheckWithOptions(t,
			`
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

		comp := compiler.NewInstructionCompiler(checker)
		comp.Config.ImportHandler = func(location common.Location) *bbq.InstructionProgram {
			return importedProgram
		}

		program := comp.Compile()

		vmConfig := &vm.Config{
			ImportHandler: func(location common.Location) *bbq.InstructionProgram {
				return importedProgram
			},
			ContractValueHandler: func(*vm.Config, common.Location) *interpreter.CompositeValue {
				return importedContractValue
			},
		}

		vmInstance = vm.NewVM(scriptLocation(), program, vmConfig)

		result, err := vmInstance.Invoke("test")
		require.NoError(t, err)
		require.Equal(t, 0, vmInstance.StackSize())

		require.Equal(t, interpreter.NewUnmeteredStringValue("contract function of the imported program"), result)
	})

	t.Run("nested imports", func(t *testing.T) {

		t.Parallel()

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
              }
            `,
			ParseAndCheckOptions{
				Location: fooLocation,
			},
		)
		require.NoError(t, err)

		fooCompiler := compiler.NewInstructionCompiler(fooChecker)
		fooProgram := fooCompiler.Compile()

		vmInstance := vm.NewVM(fooLocation, fooProgram, nil)
		fooContractValue, err := vmInstance.InitializeContract()
		require.NoError(t, err)

		// Initialize Bar

		barLocation := common.NewAddressLocation(
			nil,
			common.Address{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x2},
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

		barCompiler := compiler.NewInstructionCompiler(barChecker)
		barCompiler.Config.LocationHandler = singleIdentifierLocationResolver(t)
		barCompiler.Config.ImportHandler = func(location common.Location) *bbq.InstructionProgram {
			require.Equal(t, fooLocation, location)
			return fooProgram
		}

		barProgram := barCompiler.Compile()

		vmConfig := &vm.Config{
			ImportHandler: func(location common.Location) *bbq.InstructionProgram {
				require.Equal(t, fooLocation, location)
				return fooProgram
			},
			ContractValueHandler: func(_ *vm.Config, location common.Location) *interpreter.CompositeValue {
				require.Equal(t, fooLocation, location)
				return fooContractValue
			},
		}

		vmInstance = vm.NewVM(barLocation, barProgram, vmConfig)
		barContractValue, err := vmInstance.InitializeContract()
		require.NoError(t, err)

		// Compile and run main program

		checker, err := ParseAndCheckWithOptions(t,
			`
              import Bar from 0x02

              fun test(): String {
                  return Bar.sayHello()
              }
            `,
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

		comp := compiler.NewInstructionCompiler(checker)
		comp.Config.LocationHandler = singleIdentifierLocationResolver(t)
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

		vmConfig = &vm.Config{
			ImportHandler: func(location common.Location) *bbq.InstructionProgram {
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
			ContractValueHandler: func(_ *vm.Config, location common.Location) *interpreter.CompositeValue {
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

		vmInstance = vm.NewVM(scriptLocation(), program, vmConfig)

		result, err := vmInstance.Invoke("test")
		require.NoError(t, err)
		require.Equal(t, 0, vmInstance.StackSize())

		require.Equal(t, interpreter.NewUnmeteredStringValue("Hello from Foo!"), result)
	})

	t.Run("contract interface", func(t *testing.T) {

		t.Parallel()

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
              }
            `,
			ParseAndCheckOptions{
				Location: fooLocation,
			},
		)
		require.NoError(t, err)

		fooCompiler := compiler.NewInstructionCompiler(fooChecker)
		fooProgram := fooCompiler.Compile()

		// Initialize Bar

		barLocation := common.NewAddressLocation(
			nil,
			common.Address{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x2},
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

		barCompiler := compiler.NewInstructionCompiler(barChecker)
		barCompiler.Config.LocationHandler = singleIdentifierLocationResolver(t)
		barCompiler.Config.ImportHandler = func(location common.Location) *bbq.InstructionProgram {
			require.Equal(t, fooLocation, location)
			return fooProgram
		}

		barCompiler.Config.ElaborationResolver = func(location common.Location) (*sema.Elaboration, error) {
			switch location {
			case fooLocation:
				return fooChecker.Elaboration, nil
			case barLocation:
				return barChecker.Elaboration, nil
			default:
				return nil, fmt.Errorf("cannot find elaboration for %s", location)
			}
		}

		barProgram := barCompiler.Compile()

		vmConfig := &vm.Config{
			ImportHandler: func(location common.Location) *bbq.InstructionProgram {
				require.Equal(t, fooLocation, location)
				return fooProgram
			},
		}

		vmInstance := vm.NewVM(barLocation, barProgram, vmConfig)
		barContractValue, err := vmInstance.InitializeContract()
		require.NoError(t, err)

		// Compile and run main program

		checker, err := ParseAndCheckWithOptions(t,
			`
              import Bar from 0x02

              fun test(): String {
                  return Bar.withdraw(50)
              }
            `,
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

		comp := compiler.NewInstructionCompiler(checker)
		comp.Config.LocationHandler = singleIdentifierLocationResolver(t)
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

		vmConfig = &vm.Config{
			ImportHandler: func(location common.Location) *bbq.InstructionProgram {
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
			ContractValueHandler: func(_ *vm.Config, location common.Location) *interpreter.CompositeValue {
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

		vmInstance = vm.NewVM(scriptLocation(), program, vmConfig)

		result, err := vmInstance.Invoke("test")
		require.NoError(t, err)
		require.Equal(t, 0, vmInstance.StackSize())

		require.Equal(t, interpreter.NewUnmeteredStringValue("Successfully withdrew"), result)
	})
}

func TestInitializeContract(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheckWithOptions(t,
		`
          contract MyContract {
              var status: String

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

	comp := compiler.NewInstructionCompiler(checker)
	program := comp.Compile()

	vmConfig := &vm.Config{}
	vmInstance := vm.NewVM(scriptLocation(), program, vmConfig)
	contractValue, err := vmInstance.InitializeContract()
	require.NoError(t, err)

	fieldValue := contractValue.GetMember(vmConfig, vm.EmptyLocationRange, "status")
	assert.Equal(t, interpreter.NewUnmeteredStringValue("PENDING"), fieldValue)
}

func TestContractAccessDuringInit(t *testing.T) {

	t.Parallel()

	t.Run("using contract name", func(t *testing.T) {
		t.Parallel()

		checker, err := ParseAndCheckWithOptions(t,
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
			ParseAndCheckOptions{
				Location: common.NewAddressLocation(nil, common.Address{0x1}, "MyContract"),
			},
		)
		require.NoError(t, err)

		comp := compiler.NewInstructionCompiler(checker)
		program := comp.Compile()

		vmConfig := &vm.Config{}
		vmInstance := vm.NewVM(scriptLocation(), program, vmConfig)
		contractValue, err := vmInstance.InitializeContract()
		require.NoError(t, err)

		fieldValue := contractValue.GetMember(vmConfig, vm.EmptyLocationRange, "status")
		assert.Equal(t, interpreter.NewUnmeteredStringValue("PENDING"), fieldValue)
	})

	t.Run("using self", func(t *testing.T) {
		t.Parallel()

		checker, err := ParseAndCheckWithOptions(t,
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
			ParseAndCheckOptions{
				Location: common.NewAddressLocation(nil, common.Address{0x1}, "MyContract"),
			},
		)
		require.NoError(t, err)

		comp := compiler.NewInstructionCompiler(checker)
		program := comp.Compile()

		vmConfig := &vm.Config{}
		vmInstance := vm.NewVM(scriptLocation(), program, vmConfig)
		contractValue, err := vmInstance.InitializeContract()
		require.NoError(t, err)

		fieldValue := contractValue.GetMember(vmConfig, vm.EmptyLocationRange, "status")
		assert.Equal(t, interpreter.NewUnmeteredStringValue("PENDING"), fieldValue)
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
          }
        `)
		require.NoError(t, err)

		comp := compiler.NewInstructionCompiler(checker)
		program := comp.Compile()

		vmConfig := &vm.Config{}
		vmInstance := vm.NewVM(scriptLocation(), program, vmConfig)

		result, err := vmInstance.Invoke("test")
		require.NoError(t, err)
		require.Equal(t, 0, vmInstance.StackSize())

		require.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(5), result)
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
          }
        `

		checker, err := ParseAndCheckWithOptions(
			t,
			code,
			ParseAndCheckOptions{
				Location: common.NewAddressLocation(nil, common.Address{0x1}, "MyContract"),
			},
		)
		require.NoError(t, err)

		comp := compiler.NewInstructionCompiler(checker)
		program := comp.Compile()

		vmConfig := &vm.Config{}
		vmInstance := vm.NewVM(scriptLocation(), program, vmConfig)

		result, err := vmInstance.Invoke("init")
		require.NoError(t, err)
		require.Equal(t, 0, vmInstance.StackSize())

		require.IsType(t, &interpreter.CompositeValue{}, result)
	})
}

func TestContractField(t *testing.T) {

	t.Parallel()

	t.Run("get", func(t *testing.T) {
		t.Parallel()

		importLocation := common.NewAddressLocation(nil, common.Address{0x1}, "MyContract")

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

		importCompiler := compiler.NewInstructionCompiler(importedChecker)
		importedProgram := importCompiler.Compile()

		vmInstance := vm.NewVM(importLocation, importedProgram, nil)
		importedContractValue, err := vmInstance.InitializeContract()
		require.NoError(t, err)

		checker, err := ParseAndCheckWithOptions(t,
			`
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

		comp := compiler.NewInstructionCompilerWithConfig(
			checker,
			&compiler.Config{
				ImportHandler: func(location common.Location) *bbq.InstructionProgram {
					return importedProgram
				},
			},
		)

		program := comp.Compile()

		vmConfig := &vm.Config{
			ImportHandler: func(location common.Location) *bbq.InstructionProgram {
				return importedProgram
			},
			ContractValueHandler: func(vmConfig *vm.Config, location common.Location) *interpreter.CompositeValue {
				return importedContractValue
			},
		}

		vmInstance = vm.NewVM(scriptLocation(), program, vmConfig)
		result, err := vmInstance.Invoke("test")
		require.NoError(t, err)
		require.Equal(t, 0, vmInstance.StackSize())

		require.Equal(t, interpreter.NewUnmeteredStringValue("PENDING"), result)
	})

	t.Run("set", func(t *testing.T) {
		t.Parallel()

		importLocation := common.NewAddressLocation(nil, common.Address{0x1}, "MyContract")

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

		importCompiler := compiler.NewInstructionCompiler(importedChecker)
		importedProgram := importCompiler.Compile()

		vmInstance := vm.NewVM(importLocation, importedProgram, nil)
		importedContractValue, err := vmInstance.InitializeContract()
		require.NoError(t, err)

		checker, err := ParseAndCheckWithOptions(t,
			`
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

		comp := compiler.NewInstructionCompiler(checker)
		comp.Config.ImportHandler = func(location common.Location) *bbq.InstructionProgram {
			return importedProgram
		}

		program := comp.Compile()

		vmConfig := &vm.Config{
			ImportHandler: func(location common.Location) *bbq.InstructionProgram {
				return importedProgram
			},
			ContractValueHandler: func(vmConfig *vm.Config, location common.Location) *interpreter.CompositeValue {
				return importedContractValue
			},
		}

		vmInstance = vm.NewVM(scriptLocation(), program, vmConfig)

		result, err := vmInstance.Invoke("test")
		require.NoError(t, err)
		require.Equal(t, 0, vmInstance.StackSize())

		require.Equal(t, interpreter.NewUnmeteredStringValue("UPDATED"), result)

		fieldValue := importedContractValue.GetMember(vmConfig, vm.EmptyLocationRange, "status")
		assert.Equal(t, interpreter.NewUnmeteredStringValue("UPDATED"), fieldValue)
	})
}

func TestNativeFunctions(t *testing.T) {

	t.Parallel()

	t.Run("static function", func(t *testing.T) {

		t.Parallel()

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

		checker, err := ParseAndCheckWithOptions(t,
			`
              fun test() {
                  log("Hello, World!")
              }
            `,
			ParseAndCheckOptions{
				Config: &sema.Config{
					BaseValueActivationHandler: func(common.Location) *sema.VariableActivation {
						return baseValueActivation
					},
				},
			},
		)
		require.NoError(t, err)

		comp := compiler.NewInstructionCompiler(checker)
		program := comp.Compile()

		vmConfig := &vm.Config{}
		vmInstance := vm.NewVM(scriptLocation(), program, vmConfig)

		_, err = vmInstance.Invoke("test")
		require.NoError(t, err)
		require.Equal(t, 0, vmInstance.StackSize())
	})

	t.Run("bound function", func(t *testing.T) {
		t.Parallel()

		checker, err := ParseAndCheck(t, `
          fun test(): String {
              return "Hello".concat(", World!")
          }
        `)
		require.NoError(t, err)

		comp := compiler.NewInstructionCompiler(checker)
		program := comp.Compile()

		vmConfig := &vm.Config{}
		vmInstance := vm.NewVM(scriptLocation(), program, vmConfig)

		result, err := vmInstance.Invoke("test")
		require.NoError(t, err)
		require.Equal(t, 0, vmInstance.StackSize())

		require.Equal(t, interpreter.NewUnmeteredStringValue("Hello, World!"), result)
	})
}

func TestTransaction(t *testing.T) {

	t.Parallel()

	t.Run("simple", func(t *testing.T) {

		t.Parallel()

		checker, err := ParseAndCheck(t, `
          transaction {
              var a: String

              prepare() {
                  self.a = "Hello!"
              }

              execute {
                  self.a = "Hello again!"
              }
          }
        `)
		require.NoError(t, err)

		comp := compiler.NewInstructionCompiler(checker)
		program := comp.Compile()

		vmConfig := &vm.Config{}
		vmInstance := vm.NewVM(scriptLocation(), program, vmConfig)

		err = vmInstance.ExecuteTransaction(nil)
		require.NoError(t, err)
		require.Equal(t, 0, vmInstance.StackSize())

		// Rerun the same again using internal functions, to get the access to the transaction value.

		transaction, err := vmInstance.Invoke(commons.TransactionWrapperCompositeName)
		require.NoError(t, err)
		require.Equal(t, 0, vmInstance.StackSize())

		require.IsType(t, &interpreter.CompositeValue{}, transaction)
		compositeValue := transaction.(*interpreter.CompositeValue)

		// At the beginning, 'a' is uninitialized
		assert.Nil(t, compositeValue.GetMember(vmConfig, vm.EmptyLocationRange, "a"))

		// Invoke 'prepare'
		_, err = vmInstance.Invoke(commons.TransactionPrepareFunctionName, transaction)
		require.NoError(t, err)
		require.Equal(t, 0, vmInstance.StackSize())

		// Once 'prepare' is called, 'a' is initialized to "Hello!"
		assert.Equal(t, interpreter.NewUnmeteredStringValue("Hello!"), compositeValue.GetMember(vmConfig, vm.EmptyLocationRange, "a"))

		// Invoke 'execute'
		_, err = vmInstance.Invoke(commons.TransactionExecuteFunctionName, transaction)
		require.NoError(t, err)
		require.Equal(t, 0, vmInstance.StackSize())

		// Once 'execute' is called, 'a' is initialized to "Hello, again!"
		assert.Equal(t, interpreter.NewUnmeteredStringValue("Hello again!"), compositeValue.GetMember(vmConfig, vm.EmptyLocationRange, "a"))
	})

	t.Run("with params", func(t *testing.T) {

		t.Parallel()

		checker, err := ParseAndCheck(t, `
            transaction(param1: String, param2: String) {
                var a: String

                prepare() {
                    self.a = param1
                }

                execute {
                    self.a = param2
                }
            }
        `)
		require.NoError(t, err)

		comp := compiler.NewInstructionCompiler(checker)
		program := comp.Compile()

		vmConfig := &vm.Config{}
		vmInstance := vm.NewVM(scriptLocation(), program, vmConfig)

		args := []vm.Value{
			interpreter.NewUnmeteredStringValue("Hello!"),
			interpreter.NewUnmeteredStringValue("Hello again!"),
		}

		err = vmInstance.ExecuteTransaction(args)
		require.NoError(t, err)
		require.Equal(t, 0, vmInstance.StackSize())

		// Rerun the same again using internal functions, to get the access to the transaction value.

		transaction, err := vmInstance.Invoke(commons.TransactionWrapperCompositeName)
		require.NoError(t, err)
		require.Equal(t, 0, vmInstance.StackSize())

		require.IsType(t, &interpreter.CompositeValue{}, transaction)
		compositeValue := transaction.(*interpreter.CompositeValue)

		// At the beginning, 'a' is uninitialized
		assert.Nil(t, compositeValue.GetMember(vmConfig, vm.EmptyLocationRange, "a"))

		// Invoke 'prepare'
		_, err = vmInstance.Invoke(commons.TransactionPrepareFunctionName, transaction)
		require.NoError(t, err)
		require.Equal(t, 0, vmInstance.StackSize())

		// Once 'prepare' is called, 'a' is initialized to "Hello!"
		assert.Equal(t, interpreter.NewUnmeteredStringValue("Hello!"), compositeValue.GetMember(vmConfig, vm.EmptyLocationRange, "a"))

		// Invoke 'execute'
		_, err = vmInstance.Invoke(commons.TransactionExecuteFunctionName, transaction)
		require.NoError(t, err)
		require.Equal(t, 0, vmInstance.StackSize())

		// Once 'execute' is called, 'a' is initialized to "Hello, again!"
		assert.Equal(t, interpreter.NewUnmeteredStringValue("Hello again!"), compositeValue.GetMember(vmConfig, vm.EmptyLocationRange, "a"))
	})

	t.Run("conditions with execute", func(t *testing.T) {

		t.Parallel()

		location := common.TransactionLocation{0x1}

		activation := sema.NewVariableActivation(sema.BaseValueActivation)
		activation.DeclareValue(stdlib.PanicFunction)
		activation.DeclareValue(stdlib.NewStandardLibraryStaticFunction(
			"log",
			sema.NewSimpleFunctionType(
				sema.FunctionPurityView,
				[]sema.Parameter{
					{
						Label:          sema.ArgumentLabelNotRequired,
						Identifier:     "value",
						TypeAnnotation: sema.AnyStructTypeAnnotation,
					},
				},
				sema.VoidTypeAnnotation,
			),
			"",
			nil,
		))

		parseAndCheckOptions := ParseAndCheckOptions{
			Location: location,
			Config: &sema.Config{
				LocationHandler: singleIdentifierLocationResolver(t),
				BaseValueActivationHandler: func(location common.Location) *sema.VariableActivation {
					return activation
				},
			},
		}

		checker, err := ParseAndCheckWithOptions(t,
			`
              transaction {
                  var count: Int

                  prepare() {
                      self.count = 2
                  }

                  pre {
                      print(self.count)
                  }

                  execute {
                      self.count = 10
                  }

                  post {
                      print(self.count)
                  }
              }

              view fun print(_ n: Int): Bool {
                  log(n.toString())
                  return true
              }
            `,
			parseAndCheckOptions,
		)
		require.NoError(t, err)

		comp := compiler.NewInstructionCompiler(checker)
		program := comp.Compile()

		var logs []string
		vmConfig := vm.NewConfig(interpreter.NewInMemoryStorage(nil))

		vmConfig.NativeFunctionsProvider = func() map[string]vm.Value {
			funcs := vm.NativeFunctions()
			funcs[commons.LogFunctionName] = vm.NativeFunctionValue{
				ParameterCount: len(stdlib.LogFunctionType.Parameters),
				Function: func(config *vm.Config, typeArguments []interpreter.StaticType, arguments ...vm.Value) vm.Value {
					logs = append(logs, arguments[0].String())
					return interpreter.Void
				},
			}

			return funcs
		}

		vmInstance := vm.NewVM(scriptLocation(), program, vmConfig)

		err = vmInstance.ExecuteTransaction(nil)
		require.NoError(t, err)
		require.Equal(t, 0, vmInstance.StackSize())

		assert.Equal(t, []string{"\"2\"", "\"10\""}, logs)
	})

	t.Run("conditions without execute", func(t *testing.T) {

		t.Parallel()

		location := common.TransactionLocation{0x1}

		activation := sema.NewVariableActivation(sema.BaseValueActivation)
		activation.DeclareValue(stdlib.PanicFunction)
		activation.DeclareValue(stdlib.NewStandardLibraryStaticFunction(
			commons.LogFunctionName,
			sema.NewSimpleFunctionType(
				sema.FunctionPurityView,
				[]sema.Parameter{
					{
						Label:          sema.ArgumentLabelNotRequired,
						Identifier:     "value",
						TypeAnnotation: sema.AnyStructTypeAnnotation,
					},
				},
				sema.VoidTypeAnnotation,
			),
			"",
			nil,
		))

		parseAndCheckOptions := ParseAndCheckOptions{
			Location: location,
			Config: &sema.Config{
				LocationHandler: singleIdentifierLocationResolver(t),
				BaseValueActivationHandler: func(location common.Location) *sema.VariableActivation {
					return activation
				},
			},
		}

		checker, err := ParseAndCheckWithOptions(t,
			`
              transaction {
                  var count: Int

                  prepare() {
                      self.count = 2
                  }

                  pre {
                      print(self.count)
                  }

                  post {
                      print(self.count)
                  }
              }

              view fun print(_ n: Int): Bool {
                  log(n.toString())
                  return true
              }
            `,
			parseAndCheckOptions,
		)
		require.NoError(t, err)

		comp := compiler.NewInstructionCompiler(checker)
		program := comp.Compile()

		var logs []string
		vmConfig := vm.NewConfig(interpreter.NewInMemoryStorage(nil))

		vmConfig.NativeFunctionsProvider = func() map[string]vm.Value {
			funcs := vm.NativeFunctions()
			funcs[commons.LogFunctionName] = vm.NativeFunctionValue{
				ParameterCount: len(stdlib.LogFunctionType.Parameters),
				Function: func(config *vm.Config, typeArguments []interpreter.StaticType, arguments ...vm.Value) vm.Value {
					logs = append(logs, arguments[0].String())
					return interpreter.Void
				},
			}

			return funcs
		}

		vmInstance := vm.NewVM(scriptLocation(), program, vmConfig)

		err = vmInstance.ExecuteTransaction(nil)
		require.NoError(t, err)
		require.Equal(t, 0, vmInstance.StackSize())

		assert.Equal(t, []string{"\"2\"", "\"2\""}, logs)
	})

	t.Run("pre condition failed", func(t *testing.T) {

		t.Parallel()

		location := common.TransactionLocation{0x1}

		activation := sema.NewVariableActivation(sema.BaseValueActivation)
		activation.DeclareValue(stdlib.PanicFunction)
		activation.DeclareValue(stdlib.NewStandardLibraryStaticFunction(
			commons.LogFunctionName,
			sema.NewSimpleFunctionType(
				sema.FunctionPurityView,
				[]sema.Parameter{
					{
						Label:          sema.ArgumentLabelNotRequired,
						Identifier:     "value",
						TypeAnnotation: sema.AnyStructTypeAnnotation,
					},
				},
				sema.VoidTypeAnnotation,
			),
			"",
			nil,
		))

		parseAndCheckOptions := ParseAndCheckOptions{
			Location: location,
			Config: &sema.Config{
				LocationHandler: singleIdentifierLocationResolver(t),
				BaseValueActivationHandler: func(location common.Location) *sema.VariableActivation {
					return activation
				},
			},
		}

		checker, err := ParseAndCheckWithOptions(t,
			`
              transaction {
                  var count: Int

                  prepare() {
                      self.count = 2
                  }

                  pre {
                      print(self.count)
                      false
                  }

                  execute {
                      self.count = 10
                  }

                  post {
                      print(self.count)
                  }
              }

              view fun print(_ n: Int): Bool {
                  log(n.toString())
                  return true
              }
            `,
			parseAndCheckOptions,
		)
		require.NoError(t, err)

		comp := compiler.NewInstructionCompiler(checker)
		program := comp.Compile()

		var logs []string
		vmConfig := vm.NewConfig(interpreter.NewInMemoryStorage(nil))

		vmConfig.NativeFunctionsProvider = func() map[string]vm.Value {
			funcs := vm.NativeFunctions()
			funcs[commons.LogFunctionName] = vm.NativeFunctionValue{
				ParameterCount: len(stdlib.LogFunctionType.Parameters),
				Function: func(config *vm.Config, typeArguments []interpreter.StaticType, arguments ...vm.Value) vm.Value {
					logs = append(logs, arguments[0].String())
					return interpreter.Void
				},
			}

			return funcs
		}

		vmInstance := vm.NewVM(scriptLocation(), program, vmConfig)

		err = vmInstance.ExecuteTransaction(nil)
		require.Error(t, err)
		assert.ErrorContains(t, err, "pre/post condition failed")

		assert.Equal(t, []string{"\"2\""}, logs)
	})

	t.Run("post condition failed", func(t *testing.T) {

		t.Parallel()

		location := common.TransactionLocation{0x1}

		activation := sema.NewVariableActivation(sema.BaseValueActivation)
		activation.DeclareValue(stdlib.PanicFunction)
		activation.DeclareValue(stdlib.NewStandardLibraryStaticFunction(
			commons.LogFunctionName,
			sema.NewSimpleFunctionType(
				sema.FunctionPurityView,
				[]sema.Parameter{
					{
						Label:          sema.ArgumentLabelNotRequired,
						Identifier:     "value",
						TypeAnnotation: sema.AnyStructTypeAnnotation,
					},
				},
				sema.VoidTypeAnnotation,
			),
			"",
			nil,
		))

		parseAndCheckOptions := ParseAndCheckOptions{
			Location: location,
			Config: &sema.Config{
				LocationHandler: singleIdentifierLocationResolver(t),
				BaseValueActivationHandler: func(location common.Location) *sema.VariableActivation {
					return activation
				},
			},
		}

		checker, err := ParseAndCheckWithOptions(t,
			`
              transaction {
                  var count: Int

                  prepare() {
                      self.count = 2
                  }

                  pre {
                      print(self.count)
                  }

                  execute {
                      self.count = 10
                  }

                  post {
                      print(self.count)
                      false
                  }
              }


              view fun print(_ n: Int): Bool {
                  log(n.toString())
                  return true
              }
            `,
			parseAndCheckOptions,
		)
		require.NoError(t, err)

		comp := compiler.NewInstructionCompiler(checker)
		program := comp.Compile()

		var logs []string
		vmConfig := vm.NewConfig(interpreter.NewInMemoryStorage(nil))

		vmConfig.NativeFunctionsProvider = func() map[string]vm.Value {
			funcs := vm.NativeFunctions()
			funcs[commons.LogFunctionName] = vm.NativeFunctionValue{
				ParameterCount: len(stdlib.LogFunctionType.Parameters),
				Function: func(config *vm.Config, typeArguments []interpreter.StaticType, arguments ...vm.Value) vm.Value {
					logs = append(logs, arguments[0].String())
					return interpreter.Void
				},
			}

			return funcs
		}

		vmInstance := vm.NewVM(scriptLocation(), program, vmConfig)

		err = vmInstance.ExecuteTransaction(nil)
		require.Error(t, err)
		assert.ErrorContains(t, err, "pre/post condition failed")

		assert.Equal(t, []string{"\"2\"", "\"10\""}, logs)
	})
}

func TestInterfaceMethodCall(t *testing.T) {

	t.Parallel()

	t.Run("impl in same program", func(t *testing.T) {

		t.Parallel()

		contractLocation := common.NewAddressLocation(
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
				Location: contractLocation,
			},
		)
		require.NoError(t, err)

		importCompiler := compiler.NewInstructionCompiler(importedChecker)
		importCompiler.Config.ElaborationResolver = func(location common.Location) (*sema.Elaboration, error) {
			if location == contractLocation {
				return importedChecker.Elaboration, nil
			}

			return nil, fmt.Errorf("cannot find elaboration for %s", location)
		}

		importedProgram := importCompiler.Compile()

		vmInstance := vm.NewVM(contractLocation, importedProgram, nil)
		importedContractValue, err := vmInstance.InitializeContract()
		require.NoError(t, err)

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

		comp := compiler.NewInstructionCompiler(checker)
		comp.Config.LocationHandler = singleIdentifierLocationResolver(t)
		comp.Config.ImportHandler = func(location common.Location) *bbq.InstructionProgram {
			return importedProgram
		}

		program := comp.Compile()

		vmConfig := &vm.Config{
			ImportHandler: func(location common.Location) *bbq.InstructionProgram {
				return importedProgram
			},
			ContractValueHandler: func(vmConfig *vm.Config, location common.Location) *interpreter.CompositeValue {
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

		vmInstance = vm.NewVM(scriptLocation(), program, vmConfig)
		result, err := vmInstance.Invoke("test")
		require.NoError(t, err)
		require.Equal(t, 0, vmInstance.StackSize())

		require.Equal(t, interpreter.NewUnmeteredStringValue("Hello from Foo!"), result)
	})

	t.Run("impl in different program", func(t *testing.T) {

		t.Parallel()

		// Define the interface in `Foo`

		fooLocation := common.NewAddressLocation(
			nil,
			common.Address{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1},
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

		interfaceCompiler := compiler.NewInstructionCompiler(fooChecker)
		fooProgram := interfaceCompiler.Compile()

		interfaceVM := vm.NewVM(fooLocation, fooProgram, nil)
		fooContractValue, err := interfaceVM.InitializeContract()
		require.NoError(t, err)

		// Deploy the imported `Bar` program

		barLocation := common.NewAddressLocation(
			nil,
			common.Address{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x2},
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
				Config: &sema.Config{
					LocationHandler: singleIdentifierLocationResolver(t),
				},
				Location: barLocation,
			},
		)
		require.NoError(t, err)

		barCompiler := compiler.NewInstructionCompiler(barChecker)
		barProgram := barCompiler.Compile()

		barVM := vm.NewVM(barLocation, barProgram, nil)
		barContractValue, err := barVM.InitializeContract()
		require.NoError(t, err)

		// Define the implementation

		bazLocation := common.NewAddressLocation(
			nil,
			common.Address{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x3},
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
				Config: &sema.Config{
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
					LocationHandler: singleIdentifierLocationResolver(t),
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

		bazCompiler := compiler.NewInstructionCompiler(bazChecker)
		bazCompiler.Config.LocationHandler = singleIdentifierLocationResolver(t)
		bazCompiler.Config.ImportHandler = bazImportHandler
		bazCompiler.Config.ElaborationResolver = func(location common.Location) (*sema.Elaboration, error) {
			switch location {
			case fooLocation:
				return fooChecker.Elaboration, nil
			case barLocation:
				return barChecker.Elaboration, nil
			default:
				return nil, fmt.Errorf("cannot find elaboration for %s", location)
			}
		}

		bazProgram := bazCompiler.Compile()

		implProgramVMConfig := &vm.Config{
			ImportHandler: bazImportHandler,
			ContractValueHandler: func(vmConfig *vm.Config, location common.Location) *interpreter.CompositeValue {
				switch location {
				case fooLocation:
					return fooContractValue
				case barLocation:
					return barContractValue
				default:
					panic(fmt.Errorf("cannot find contract: %s", location))
				}
			},
		}

		bazVM := vm.NewVM(bazLocation, bazProgram, implProgramVMConfig)
		bazContractValue, err := bazVM.InitializeContract()
		require.NoError(t, err)

		// Get `Bar.GreetingsImpl` value

		checker, err := ParseAndCheckWithOptions(t,
			`
              import Baz from 0x03

              fun test(): Baz.GreetingImpl {
                  return Baz.GreetingImpl()
              }
            `,
			ParseAndCheckOptions{
				Config: &sema.Config{
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
					LocationHandler: singleIdentifierLocationResolver(t),
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

		comp := compiler.NewInstructionCompiler(checker)
		comp.Config.LocationHandler = singleIdentifierLocationResolver(t)
		comp.Config.ImportHandler = scriptImportHandler

		program := comp.Compile()

		vmConfig := &vm.Config{
			ImportHandler: scriptImportHandler,
			ContractValueHandler: func(vmConfig *vm.Config, location common.Location) *interpreter.CompositeValue {
				switch location {
				case barLocation:
					return barContractValue
				case bazLocation:
					return bazContractValue
				default:
					panic(fmt.Errorf("cannot find contract: %s", location))
				}
			},
		}

		scriptVM := vm.NewVM(scriptLocation(), program, vmConfig)
		implValue, err := scriptVM.Invoke("test")
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
				Config: &sema.Config{
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
					LocationHandler: singleIdentifierLocationResolver(t),
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

		comp = compiler.NewInstructionCompiler(checker)
		comp.Config.LocationHandler = singleIdentifierLocationResolver(t)
		comp.Config.ImportHandler = scriptImportHandler

		program = comp.Compile()

		vmConfig = &vm.Config{
			ImportHandler: scriptImportHandler,
			ContractValueHandler: func(vmConfig *vm.Config, location common.Location) *interpreter.CompositeValue {
				switch location {
				case fooLocation:
					return fooContractValue
				case barLocation:
					return barContractValue
				case bazLocation:
					return bazContractValue
				default:
					panic(fmt.Errorf("cannot find contract: %s", location))
				}
			},
		}

		scriptVM = vm.NewVM(scriptLocation(), program, vmConfig)

		result, err := scriptVM.Invoke("test", implValue)
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

		comp := compiler.NewInstructionCompiler(checker)
		program := comp.Compile()

		vmConfig := &vm.Config{}
		vmInstance := vm.NewVM(scriptLocation(), program, vmConfig)

		result, err := vmInstance.Invoke("test")
		require.NoError(t, err)
		require.Equal(t, 0, vmInstance.StackSize())

		require.IsType(t, &interpreter.ArrayValue{}, result)
		array := result.(*interpreter.ArrayValue)
		assert.Equal(t, 2, array.Count())
		assert.Equal(
			t,
			interpreter.NewUnmeteredIntValueFromInt64(2),
			array.Get(vmConfig, vm.EmptyLocationRange, 0),
		)
		assert.Equal(
			t,
			interpreter.NewUnmeteredIntValueFromInt64(5),
			array.Get(vmConfig, vm.EmptyLocationRange, 1),
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

		comp := compiler.NewInstructionCompiler(checker)
		program := comp.Compile()

		vmConfig := &vm.Config{}
		vmInstance := vm.NewVM(scriptLocation(), program, vmConfig)

		result, err := vmInstance.Invoke("test")
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

		comp := compiler.NewInstructionCompiler(checker)
		program := comp.Compile()

		vmConfig := &vm.Config{}
		vmInstance := vm.NewVM(scriptLocation(), program, vmConfig)

		result, err := vmInstance.Invoke("test")
		require.NoError(t, err)
		require.Equal(t, 0, vmInstance.StackSize())

		require.IsType(t, &interpreter.ArrayValue{}, result)
		array := result.(*interpreter.ArrayValue)
		assert.Equal(t, 3, array.Count())
		assert.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(2), array.Get(vmConfig, vm.EmptyLocationRange, 0))
		assert.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(5), array.Get(vmConfig, vm.EmptyLocationRange, 1))
		assert.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(8), array.Get(vmConfig, vm.EmptyLocationRange, 2))
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

		comp := compiler.NewInstructionCompiler(checker)
		program := comp.Compile()

		vmConfig := &vm.Config{}
		vmInstance := vm.NewVM(scriptLocation(), program, vmConfig)

		result, err := vmInstance.Invoke("test")
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
				vmConfig,
				vm.EmptyLocationRange,
				interpreter.NewUnmeteredStringValue("b"),
			),
		)
		assert.Equal(t,
			interpreter.NewUnmeteredSomeValueNonCopying(
				interpreter.NewUnmeteredIntValueFromInt64(5),
			),
			dictionary.GetKey(
				vmConfig,
				vm.EmptyLocationRange,
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

		result, err := compileAndInvoke(t, code, "test")
		require.NoError(t, err)

		require.Equal(t, interpreter.NewUnmeteredStringValue("Hello from Foo!"), result)
	})
}

func TestResource(t *testing.T) {

	t.Parallel()

	t.Run("new", func(t *testing.T) {
		t.Parallel()

		checker, err := ParseAndCheck(t, `
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
        `)
		require.NoError(t, err)

		comp := compiler.NewInstructionCompiler(checker)
		program := comp.Compile()

		var uuid uint64

		vmConfig := (&vm.Config{}).
			WithInterpreterConfig(&interpreter.Config{
				UUIDHandler: func() (uint64, error) {
					uuid++
					return uuid, nil
				},
			})

		vmInstance := vm.NewVM(scriptLocation(), program, vmConfig)

		result, err := vmInstance.Invoke("test")
		require.NoError(t, err)
		require.Equal(t, 0, vmInstance.StackSize())

		require.IsType(t, &interpreter.CompositeValue{}, result)
		structValue := result.(*interpreter.CompositeValue)
		compositeType := structValue.StaticType(vmConfig).(*interpreter.CompositeStaticType)

		require.Equal(t, "Foo", compositeType.QualifiedIdentifier)
		require.Equal(
			t,
			interpreter.NewUnmeteredIntValueFromInt64(5),
			structValue.GetMember(vmConfig, vm.EmptyLocationRange, "id"),
		)
	})

	t.Run("destroy", func(t *testing.T) {
		t.Parallel()

		checker, err := ParseAndCheck(t, `
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
        `)
		require.NoError(t, err)

		comp := compiler.NewInstructionCompiler(checker)
		program := comp.Compile()

		var uuid uint64 = 42

		vmConfig := (&vm.Config{}).
			WithInterpreterConfig(&interpreter.Config{
				UUIDHandler: func() (uint64, error) {
					uuid++
					return uuid, nil
				},
			})
		vmInstance := vm.NewVM(scriptLocation(), program, vmConfig)

		_, err = vmInstance.Invoke("test")
		require.NoError(t, err)
		require.Equal(t, 0, vmInstance.StackSize())
	})
}

func fib(n int) int {
	if n < 2 {
		return n
	}
	return fib(n-1) + fib(n-2)
}

func BenchmarkGoFib(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		fib(46)
	}
}

func TestDefaultFunctions(t *testing.T) {

	t.Parallel()

	t.Run("simple interface", func(t *testing.T) {
		t.Parallel()

		result, err := compileAndInvoke(t,
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

		result, err := compileAndInvoke(t,
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

		result, err := compileAndInvoke(t,
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

		storage := interpreter.NewInMemoryStorage(nil)

		programs := map[common.Location]*compiledProgram{}
		contractValues := map[common.Location]*interpreter.CompositeValue{}

		vmConfig := vm.NewConfig(storage)
		vmConfig.ImportHandler = func(location common.Location) *bbq.InstructionProgram {
			program, ok := programs[location]
			if !ok {
				assert.FailNow(t, "invalid location")
			}
			return program.Program
		}
		vmConfig.ContractValueHandler = func(_ *vm.Config, location common.Location) *interpreter.CompositeValue {
			contractValue, ok := contractValues[location]
			if !ok {
				assert.FailNow(t, "invalid location")
			}
			return contractValue
		}

		var uuid uint64 = 42
		vmConfig.WithInterpreterConfig(&interpreter.Config{
			UUIDHandler: func() (uint64, error) {
				uuid++
				return uuid, nil
			},
		})

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
		parseCheckAndCompile(t, barContract, barLocation, programs)

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

		fooProgram := parseCheckAndCompile(t, fooContract, fooLocation, programs)

		fooVM := vm.NewVM(fooLocation, fooProgram, vmConfig)

		fooContractValue, err := fooVM.InitializeContract()
		require.NoError(t, err)
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

		txProgram := parseCheckAndCompile(t, tx, txLocation(), programs)
		txVM := vm.NewVM(txLocation(), txProgram, vmConfig)

		result, err := txVM.Invoke("main")
		require.NoError(t, err)
		require.Equal(t, 0, txVM.StackSize())
		require.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(7), result)
	})

	t.Run("in different contract with nested call", func(t *testing.T) {

		t.Parallel()

		storage := interpreter.NewInMemoryStorage(nil)

		programs := map[common.Location]*compiledProgram{}
		contractValues := map[common.Location]*interpreter.CompositeValue{}

		vmConfig := vm.NewConfig(storage)
		vmConfig.ImportHandler = func(location common.Location) *bbq.InstructionProgram {
			program, ok := programs[location]
			if !ok {
				assert.FailNow(t, "invalid location")
			}
			return program.Program
		}
		vmConfig.ContractValueHandler = func(_ *vm.Config, location common.Location) *interpreter.CompositeValue {
			contractValue, ok := contractValues[location]
			if !ok {
				assert.FailNow(t, "invalid location")
			}
			return contractValue
		}

		var uuid uint64 = 42
		vmConfig.WithInterpreterConfig(&interpreter.Config{
			UUIDHandler: func() (uint64, error) {
				uuid++
				return uuid, nil
			},
		})

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
		parseCheckAndCompile(t, barContract, barLocation, programs)

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

		fooProgram := parseCheckAndCompile(t, fooContract, fooLocation, programs)

		fooVM := vm.NewVM(fooLocation, fooProgram, vmConfig)

		fooContractValue, err := fooVM.InitializeContract()
		require.NoError(t, err)
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

		txProgram := parseCheckAndCompile(t, tx, txLocation(), programs)
		txVM := vm.NewVM(txLocation(), txProgram, vmConfig)

		result, err := txVM.Invoke("main")
		require.NoError(t, err)
		require.Equal(t, 0, txVM.StackSize())
		require.Equal(t, interpreter.NewUnmeteredStringValue("Hello from HelloInterface"), result)
	})

	t.Run("in different contract nested call overridden", func(t *testing.T) {

		t.Parallel()

		storage := interpreter.NewInMemoryStorage(nil)

		programs := map[common.Location]*compiledProgram{}
		contractValues := map[common.Location]*interpreter.CompositeValue{}

		vmConfig := vm.NewConfig(storage).
			WithAccountHandler(&testAccountHandler{
				emitEvent: func(
					_ interpreter.ValueExportContext,
					_ interpreter.LocationRange,
					_ *sema.CompositeType,
					_ []interpreter.Value,
				) {
					// ignore
				},
			})

		vmConfig.ImportHandler = func(location common.Location) *bbq.InstructionProgram {
			program, ok := programs[location]
			if !ok {
				assert.FailNow(t, "invalid location")
			}
			return program.Program
		}
		vmConfig.ContractValueHandler = func(_ *vm.Config, location common.Location) *interpreter.CompositeValue {
			contractValue, ok := contractValues[location]
			if !ok {
				assert.FailNow(t, "invalid location")
			}
			return contractValue
		}

		var uuid uint64 = 42
		vmConfig.WithInterpreterConfig(&interpreter.Config{
			UUIDHandler: func() (uint64, error) {
				uuid++
				return uuid, nil
			},
		})

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
		parseCheckAndCompile(t, barContract, barLocation, programs)

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

		fooProgram := parseCheckAndCompile(t, fooContract, fooLocation, programs)

		fooVM := vm.NewVM(fooLocation, fooProgram, vmConfig)

		fooContractValue, err := fooVM.InitializeContract()
		require.NoError(t, err)
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

		txProgram := parseCheckAndCompile(t, tx, txLocation(), programs)
		txVM := vm.NewVM(txLocation(), txProgram, vmConfig)

		result, err := txVM.Invoke("main")
		require.NoError(t, err)
		require.Equal(t, 0, txVM.StackSize())
		require.Equal(t, interpreter.NewUnmeteredStringValue("Hello from Hello"), result)
	})
}

func TestFunctionPreConditions(t *testing.T) {

	t.Parallel()

	t.Run("failed", func(t *testing.T) {

		t.Parallel()

		_, err := compileAndInvoke(t,
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

		require.Error(t, err)
		assert.ErrorContains(t, err, "pre/post condition failed")
	})

	t.Run("failed with message", func(t *testing.T) {

		t.Parallel()

		_, err := compileAndInvoke(t,
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

		require.Error(t, err)
		assert.ErrorContains(t, err, "x must be zero")
	})

	t.Run("passed", func(t *testing.T) {

		t.Parallel()

		result, err := compileAndInvoke(t,
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

		_, err := compileAndInvoke(t,
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

		require.Error(t, err)
		assert.ErrorContains(t, err, "a must be larger than 10")
	})

	t.Run("pre conditions order", func(t *testing.T) {

		t.Parallel()

		code := `
          struct A: B {
              fun test() {
                  pre { print("A") }
              }
          }

          struct interface B: C, D {
              fun test() {
                  pre { print("B") }
              }
          }

          struct interface C: E, F {
              fun test() {
                  pre { print("C") }
              }
          }

          struct interface D: F {
              fun test() {
                  pre { print("D") }
              }
          }

          struct interface E {
              fun test() {
                  pre { print("E") }
              }
          }

          struct interface F {
              fun test() {
                  pre { print("F") }
              }
          }

          view fun print(_ msg: String): Bool {
              log(msg)
              return true
          }

          fun main() {
              let a = A()
              a.test()
          }
        `

		location := common.ScriptLocation{0x1}

		activation := sema.NewVariableActivation(sema.BaseValueActivation)
		activation.DeclareValue(stdlib.PanicFunction)
		activation.DeclareValue(stdlib.NewStandardLibraryStaticFunction(
			commons.LogFunctionName,
			sema.NewSimpleFunctionType(
				sema.FunctionPurityView,
				[]sema.Parameter{
					{
						Label:          sema.ArgumentLabelNotRequired,
						Identifier:     "value",
						TypeAnnotation: sema.AnyStructTypeAnnotation,
					},
				},
				sema.VoidTypeAnnotation,
			),
			"",
			nil,
		))

		var logs []string

		config := vm.NewConfig(interpreter.NewInMemoryStorage(nil))
		config.NativeFunctionsProvider = func() map[string]vm.Value {
			return map[string]vm.Value{
				commons.LogFunctionName: vm.NativeFunctionValue{
					ParameterCount: len(stdlib.LogFunctionType.Parameters),
					Function: func(config *vm.Config, typeArguments []interpreter.StaticType, arguments ...vm.Value) vm.Value {
						logs = append(logs, arguments[0].String())
						return interpreter.Void
					},
				},
				commons.PanicFunctionName: vm.NativeFunctionValue{
					ParameterCount: len(stdlib.PanicFunctionType.Parameters),
					Function: func(config *vm.Config, typeArguments []interpreter.StaticType, arguments ...vm.Value) vm.Value {
						messageValue, ok := arguments[0].(*interpreter.StringValue)
						if !ok {
							panic(errors.NewUnreachableError())
						}

						panic(stdlib.PanicError{
							Message: messageValue.Str,
						})
					},
				},
			}
		}

		_, err := compileAndInvokeWithOptions(
			t,
			code,
			"main",
			CompilerAndVMOptions{
				ParseAndCheckOptions: &ParseAndCheckOptions{
					Location: location,
					Config: &sema.Config{
						LocationHandler: singleIdentifierLocationResolver(t),
						BaseValueActivationHandler: func(location common.Location) *sema.VariableActivation {
							return activation
						},
					},
				},
				VMConfig: config,
			},
		)
		require.NoError(t, err)

		// The pre-conditions of the interfaces are executed first, with depth-first pre-order traversal.
		// The pre-condition of the concrete type is executed at the end, after the interfaces.
		assert.Equal(t, []string{"\"B\"", "\"C\"", "\"E\"", "\"F\"", "\"D\"", "\"A\""}, logs)
	})

	t.Run("in different contract with nested call", func(t *testing.T) {

		t.Parallel()

		storage := interpreter.NewInMemoryStorage(nil)

		programs := map[common.Location]*compiledProgram{}
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
		vmConfig.ContractValueHandler = func(_ *vm.Config, location common.Location) *interpreter.CompositeValue {
			contractValue, ok := contractValues[location]
			if !ok {
				assert.FailNow(t, "invalid location")
			}
			return contractValue
		}

		vmConfig.NativeFunctionsProvider = func() map[string]vm.Value {
			funcs := vm.NativeFunctions()
			funcs[commons.LogFunctionName] = vm.NativeFunctionValue{
				ParameterCount: len(stdlib.LogFunctionType.Parameters),
				Function: func(config *vm.Config, typeArguments []interpreter.StaticType, arguments ...vm.Value) vm.Value {
					logs = append(logs, arguments[0].String())
					return interpreter.Void
				},
			}

			return funcs
		}

		activation := sema.NewVariableActivation(sema.BaseValueActivation)
		activation.DeclareValue(stdlib.PanicFunction)
		activation.DeclareValue(stdlib.NewStandardLibraryStaticFunction(
			commons.LogFunctionName,
			sema.NewSimpleFunctionType(
				sema.FunctionPurityView,
				[]sema.Parameter{
					{
						Label:          sema.ArgumentLabelNotRequired,
						Identifier:     "value",
						TypeAnnotation: sema.AnyStructTypeAnnotation,
					},
				},
				sema.VoidTypeAnnotation,
			),
			"",
			nil,
		))

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
                      log("Bar.".concat(msg))
                      return true
                  }
              }

              struct interface F {

                  fun test() {
                      pre { self.printFromF("F") }
                  }

                  view fun printFromF(_ msg: String): Bool {
                      log("Bar.".concat(msg))
                      return true
                  }
              }
          }
        `

		// Only need to compile
		_ = parseCheckAndCompileCodeWithOptions(
			t,
			barContract,
			barLocation,
			CompilerAndVMOptions{
				ParseAndCheckOptions: &ParseAndCheckOptions{
					Location: barLocation,
					Config: &sema.Config{
						LocationHandler: singleIdentifierLocationResolver(t),
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
                      log("Foo.".concat(msg))
                      return true
                  }
              }
            `,
			contractsAddress.HexWithPrefix(),
		)

		fooProgram := parseCheckAndCompileCodeWithOptions(
			t,
			fooContract,
			fooLocation,
			CompilerAndVMOptions{
				ParseAndCheckOptions: &ParseAndCheckOptions{
					Location: fooLocation,
					Config: &sema.Config{
						LocationHandler: singleIdentifierLocationResolver(t),
						BaseValueActivationHandler: func(location common.Location) *sema.VariableActivation {
							return activation
						},
					},
				},
			},
			programs,
		)

		fooVM := vm.NewVM(fooLocation, fooProgram, vmConfig)

		fooContractValue, err := fooVM.InitializeContract()
		require.NoError(t, err)
		contractValues[fooLocation] = fooContractValue

		// Run script

		code := fmt.Sprintf(
			`
              import Foo from %[1]s

              struct A: Foo.B {
                  fun test() {
                      pre { print("A") }
                  }
              }

              view fun print(_ msg: String): Bool {
                  log(msg)
                  return true
              }

              fun main() {
                  let a = A()
                  a.test()
              }
            `,
			contractsAddress.HexWithPrefix(),
		)

		location := common.ScriptLocation{0x1}

		_, err = compileAndInvokeWithOptionsAndPrograms(
			t,
			code,
			"main",
			CompilerAndVMOptions{
				ParseAndCheckOptions: &ParseAndCheckOptions{
					Location: location,
					Config: &sema.Config{
						LocationHandler: singleIdentifierLocationResolver(t),
						BaseValueActivationHandler: func(location common.Location) *sema.VariableActivation {
							return activation
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

		_, err := compileAndInvoke(t,
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

		require.Error(t, err)
		assert.ErrorContains(t, err, "pre/post condition failed")
	})

	t.Run("failed with message", func(t *testing.T) {

		t.Parallel()

		_, err := compileAndInvoke(t,
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

		require.Error(t, err)
		assert.ErrorContains(t, err, "x must be zero")
	})

	t.Run("passed", func(t *testing.T) {

		t.Parallel()

		result, err := compileAndInvoke(t,
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

		result, err := compileAndInvoke(t,
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

		_, err := compileAndInvoke(t,
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

		require.Error(t, err)
		assert.ErrorContains(t, err, "x must be 5")
	})

	t.Run("post conditions order", func(t *testing.T) {

		t.Parallel()

		code := `
          struct A: B {
              fun test() {
                  post { print("A") }
              }
          }

          struct interface B: C, D {
              fun test() {
                  post { print("B") }
              }
          }

          struct interface C: E, F {
              fun test() {
                  post { print("C") }
              }
          }

          struct interface D: F {
              fun test() {
                  post { print("D") }
              }
          }

          struct interface E {
              fun test() {
                  post { print("E") }
              }
          }

          struct interface F {
              fun test() {
                  post { print("F") }
              }
          }

          view fun print(_ msg: String): Bool {
              log(msg)
              return true
          }

          fun main() {
              let a = A()
              a.test()
          }
        `

		location := common.ScriptLocation{0x1}

		activation := sema.NewVariableActivation(sema.BaseValueActivation)
		activation.DeclareValue(stdlib.PanicFunction)
		activation.DeclareValue(stdlib.NewStandardLibraryStaticFunction(
			commons.LogFunctionName,
			sema.NewSimpleFunctionType(
				sema.FunctionPurityView,
				[]sema.Parameter{
					{
						Label:          sema.ArgumentLabelNotRequired,
						Identifier:     "value",
						TypeAnnotation: sema.AnyStructTypeAnnotation,
					},
				},
				sema.VoidTypeAnnotation,
			),
			"",
			nil,
		))

		var logs []string

		config := vm.NewConfig(interpreter.NewInMemoryStorage(nil))
		config.NativeFunctionsProvider = func() map[string]vm.Value {
			funcs := vm.NativeFunctions()
			funcs[commons.LogFunctionName] = vm.NativeFunctionValue{
				ParameterCount: len(stdlib.LogFunctionType.Parameters),
				Function: func(config *vm.Config, typeArguments []interpreter.StaticType, arguments ...vm.Value) vm.Value {
					logs = append(logs, arguments[0].String())
					return interpreter.Void
				},
			}

			return funcs
		}

		_, err := compileAndInvokeWithOptions(
			t,
			code,
			"main",
			CompilerAndVMOptions{
				ParseAndCheckOptions: &ParseAndCheckOptions{
					Location: location,
					Config: &sema.Config{
						LocationHandler: singleIdentifierLocationResolver(t),
						BaseValueActivationHandler: func(location common.Location) *sema.VariableActivation {
							return activation
						},
					},
				},
				VMConfig: config,
			},
		)
		require.NoError(t, err)

		// The post-condition of the concrete type is executed first, before the interfaces.
		// The post-conditions of the interfaces are executed after that, with the reversed depth-first pre-order.
		assert.Equal(t, []string{"\"A\"", "\"D\"", "\"F\"", "\"E\"", "\"C\"", "\"B\""}, logs)
	})

	t.Run("result var failed", func(t *testing.T) {

		t.Parallel()

		_, err := compileAndInvoke(t,
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

		require.Error(t, err)
		assert.ErrorContains(t, err, "x must be zero")
	})

	t.Run("result var passed", func(t *testing.T) {

		t.Parallel()

		result, err := compileAndInvoke(t,
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

		_, err := compileAndInvoke(t,
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

		require.Error(t, err)
		assert.ErrorContains(t, err, "result must be larger than 10")
	})

	t.Run("resource typed result var passed", func(t *testing.T) {

		t.Parallel()

		result, err := compileAndInvoke(t,
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

		_, err := compileAndInvoke(t,
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

		require.Error(t, err)
		assert.ErrorContains(t, err, "pre/post condition failed")
	})
}

func TestIfLet(t *testing.T) {

	t.Parallel()

	test := func(t *testing.T, argument vm.Value) vm.Value {
		result, err := compileAndInvoke(t,
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
		result, err := compileAndInvoke(t,
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

func TestSwitch(t *testing.T) {

	t.Parallel()

	t.Run("1", func(t *testing.T) {
		t.Parallel()

		result, err := compileAndInvoke(t,
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

		result, err := compileAndInvoke(t,
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

		result, err := compileAndInvoke(t,
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

		_, err, logs := compileAndInvokeWithLogs(t,
			`
              struct interface Foo {
                  fun test(_ a: Int) {
                      printMessage("invoked Foo.test()")
                  }
              }

              struct interface Bar: Foo {
                  fun test(_ a: Int) {
                      pre {
                           printMessage("invoked Bar.test() pre-condition")
                      }

                      post {
                           printMessage("invoked Bar.test() post-condition")
                      }
                  }
              }

              struct Test: Bar {}

              view fun printMessage(_ msg: String): Bool {
                  log(msg)
                  return true
              }

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

		_, err, logs := compileAndInvokeWithLogs(t,
			`
              struct interface Foo {
                  fun test(_ a: Int) {
                      pre {
                           printMessage("invoked Foo.test() pre-condition")
                      }
                      post {
                           printMessage("invoked Foo.test() post-condition")
                      }
                      printMessage("invoked Foo.test()")
                  }
              }

              struct interface Bar: Foo {
                  fun test(_ a: Int) {
                      pre {
                           printMessage("invoked Bar.test() pre-condition")
                      }

                      post {
                           printMessage("invoked Bar.test() post-condition")
                      }
                  }
              }

              struct Test: Bar {}

              view fun printMessage(_ msg: String): Bool {
                  log(msg)
                  return true
              }

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

		storage := interpreter.NewInMemoryStorage(nil)

		activation := sema.NewVariableActivation(sema.BaseValueActivation)
		activation.DeclareValue(stdlib.PanicFunction)
		activation.DeclareValue(stdlib.NewStandardLibraryStaticFunction(
			commons.LogFunctionName,
			sema.NewSimpleFunctionType(
				sema.FunctionPurityView,
				[]sema.Parameter{
					{
						Label:          sema.ArgumentLabelNotRequired,
						Identifier:     "value",
						TypeAnnotation: sema.AnyStructTypeAnnotation,
					},
				},
				sema.VoidTypeAnnotation,
			),
			"",
			nil,
		))

		var logs []string
		vmConfig := vm.NewConfig(storage)

		vmConfig.NativeFunctionsProvider = func() map[string]vm.Value {
			funcs := vm.NativeFunctions()
			funcs[commons.LogFunctionName] = vm.NativeFunctionValue{
				ParameterCount: len(stdlib.LogFunctionType.Parameters),
				Function: func(config *vm.Config, typeArguments []interpreter.StaticType, arguments ...vm.Value) vm.Value {
					logs = append(logs, arguments[0].String())
					return interpreter.Void
				},
			}

			return funcs
		}

		_, err := compileAndInvokeWithOptions(t,
			`
              struct Test {
                  var i: Int

                  init() {
                      self.i = 2
                  }

                  fun test() {
                      post {
                          print(before(self.i).toString())
                          print(self.i.toString())
                      }
                      self.i = 5
                  }
              }

              view fun print(_ msg: String): Bool {
                  log(msg)
                  return true
              }

              fun main() {
                 Test().test()
              }
            `,
			"main",
			CompilerAndVMOptions{
				VMConfig: vmConfig,
				ParseAndCheckOptions: &ParseAndCheckOptions{
					Config: &sema.Config{
						LocationHandler: singleIdentifierLocationResolver(t),
						BaseValueActivationHandler: func(location common.Location) *sema.VariableActivation {
							return activation
						},
					},
				},
			},
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

		storage := interpreter.NewInMemoryStorage(nil)

		activation := sema.NewVariableActivation(sema.BaseValueActivation)
		activation.DeclareValue(stdlib.PanicFunction)
		activation.DeclareValue(stdlib.NewStandardLibraryStaticFunction(
			commons.LogFunctionName,
			sema.NewSimpleFunctionType(
				sema.FunctionPurityView,
				[]sema.Parameter{
					{
						Label:          sema.ArgumentLabelNotRequired,
						Identifier:     "value",
						TypeAnnotation: sema.AnyStructTypeAnnotation,
					},
				},
				sema.VoidTypeAnnotation,
			),
			"",
			nil,
		))

		var logs []string
		vmConfig := vm.NewConfig(storage)

		vmConfig.NativeFunctionsProvider = func() map[string]vm.Value {
			funcs := vm.NativeFunctions()
			funcs[commons.LogFunctionName] = vm.NativeFunctionValue{
				ParameterCount: len(stdlib.LogFunctionType.Parameters),
				Function: func(config *vm.Config, typeArguments []interpreter.StaticType, arguments ...vm.Value) vm.Value {
					logs = append(logs, arguments[0].String())
					return interpreter.Void
				},
			}

			return funcs
		}

		_, err := compileAndInvokeWithOptions(t,
			`
                struct interface Foo {
                    var i: Int

                    fun test() {
                        post {
                            print(before(self.i).toString())
                            print(self.i.toString())
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

                view fun print(_ msg: String): Bool {
                    log(msg)
                    return true
                }

                fun main() {
                    Test().test()
                }
            `,
			"main",
			CompilerAndVMOptions{
				VMConfig: vmConfig,
				ParseAndCheckOptions: &ParseAndCheckOptions{
					Config: &sema.Config{
						LocationHandler: singleIdentifierLocationResolver(t),
						BaseValueActivationHandler: func(location common.Location) *sema.VariableActivation {
							return activation
						},
					},
				},
			},
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

		storage := interpreter.NewInMemoryStorage(nil)

		activation := sema.NewVariableActivation(sema.BaseValueActivation)
		activation.DeclareValue(stdlib.PanicFunction)
		activation.DeclareValue(stdlib.NewStandardLibraryStaticFunction(
			commons.LogFunctionName,
			sema.NewSimpleFunctionType(
				sema.FunctionPurityView,
				[]sema.Parameter{
					{
						Label:          sema.ArgumentLabelNotRequired,
						Identifier:     "value",
						TypeAnnotation: sema.AnyStructTypeAnnotation,
					},
				},
				sema.VoidTypeAnnotation,
			),
			"",
			nil,
		))

		var logs []string
		vmConfig := vm.NewConfig(storage)

		vmConfig.NativeFunctionsProvider = func() map[string]vm.Value {
			funcs := vm.NativeFunctions()
			funcs[commons.LogFunctionName] = vm.NativeFunctionValue{
				ParameterCount: len(stdlib.LogFunctionType.Parameters),
				Function: func(config *vm.Config, typeArguments []interpreter.StaticType, arguments ...vm.Value) vm.Value {
					logs = append(logs, arguments[0].String())
					return interpreter.Void
				},
			}

			return funcs
		}

		_, err := compileAndInvokeWithOptions(t,
			`
              struct interface Foo {
                  var i: Int

                  fun test() {
                      post {
                          print(before(self.i).toString())
                          print(before(self.i + 1).toString())
                          print(self.i.toString())
                      }
                      self.i = 8
                  }
              }

              struct interface Bar: Foo {
                  var i: Int

                  fun test() {
                      post {
                          print(before(self.i + 3).toString())
                      }
                  }
              }


              struct Test: Bar {
                  var i: Int

                  init() {
                      self.i = 2
                  }
              }

              view fun print(_ msg: String): Bool {
                  log(msg)
                  return true
              }

              fun main() {
                  Test().test()
              }
            `,
			"main",
			CompilerAndVMOptions{
				VMConfig: vmConfig,
				ParseAndCheckOptions: &ParseAndCheckOptions{
					Config: &sema.Config{
						LocationHandler: singleIdentifierLocationResolver(t),
						BaseValueActivationHandler: func(location common.Location) *sema.VariableActivation {
							return activation
						},
					},
				},
			},
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

		_, err := compileAndInvoke(t,
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

	vmConfig := vm.NewConfig(interpreter.NewInMemoryStorage(nil))
	vmConfig.OnEventEmitted = func(event *interpreter.CompositeValue, eventType *interpreter.CompositeStaticType) error {
		require.False(t, eventEmitted)
		eventEmitted = true

		assert.Equal(t,
			TestLocation.TypeID(nil, "Inc"),
			eventType.ID(),
		)

		return nil
	}

	_, err := compileAndInvokeWithOptions(t,
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

		result, err := compileAndInvoke(t,
			`
              fun test(x: Int): AnyStruct {
                  return x as Int?
              }
            `,
			"test",
			interpreter.NewUnmeteredIntValueFromInt64(2),
		)
		require.NoError(t, err)
		assert.Equal(t, interpreter.NewUnmeteredSomeValueNonCopying(interpreter.NewUnmeteredIntValueFromInt64(2)), result)
	})

	t.Run("force cast success", func(t *testing.T) {
		t.Parallel()

		result, err := compileAndInvoke(t,
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

		_, err := compileAndInvoke(t,
			`
              fun test(x: AnyStruct): Int {
                  return x as! Int
              }
            `,
			"test",
			interpreter.TrueValue,
		)
		require.Error(t, err)
		assert.ErrorIs(
			t,
			err,
			interpreter.ForceCastTypeMismatchError{
				ExpectedType: sema.IntType,
				ActualType:   sema.BoolType,
			},
		)
	})

	t.Run("failable cast success", func(t *testing.T) {
		t.Parallel()

		result, err := compileAndInvoke(t,
			`
              fun test(x: AnyStruct): Int? {
                  return x as? Int
              }
            `,
			"test",
			interpreter.NewUnmeteredIntValueFromInt64(2),
		)
		require.NoError(t, err)
		assert.Equal(t, interpreter.NewUnmeteredSomeValueNonCopying(interpreter.NewUnmeteredIntValueFromInt64(2)), result)
	})

	t.Run("failable cast fail", func(t *testing.T) {
		t.Parallel()

		result, err := compileAndInvoke(t,
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

		result, err := compileAndInvoke(t,
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

		result, err := compileAndInvoke(t,
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

			result, err := compileAndInvoke(t,
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
		// TODO:
		switch integerType {
		case sema.Int128Type, sema.Int256Type,
			sema.UInt128Type, sema.UInt256Type,
			sema.Word128Type, sema.Word256Type:
			continue
		}

		test(integerType)
	}
}

func TestAddress(t *testing.T) {

	t.Parallel()

	result, err := compileAndInvoke(t,
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

			result, err := compileAndInvoke(t,
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

			assert.Equal(t, "10.00000000", result.String())
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

		result, err := compileAndInvoke(t,
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

		result, err := compileAndInvoke(t,
			`
                fun test(): String {
                    var array = [5, 6, 7, 8]
                    var keys = ""
                    var values = ""
                    for i, e in array {
                        keys = keys.concat(i.toString())
                        values = values.concat(e.toString())
                    }

                    return keys.concat("_").concat(values)
                }
            `,
			"test",
		)
		require.NoError(t, err)
		assert.Equal(t, interpreter.NewUnmeteredStringValue("0123_5678"), result)
	})

	t.Run("array loop scoping", func(t *testing.T) {
		t.Parallel()

		result, err := compileAndInvoke(t,
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

func TestCompileIf(t *testing.T) {

	t.Parallel()

	test := func(t *testing.T, argument vm.Value) vm.Value {
		result, err := compileAndInvoke(t,
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

func TestCompileConditional(t *testing.T) {

	t.Parallel()

	test := func(t *testing.T, argument vm.Value) vm.Value {
		result, err := compileAndInvoke(t,
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

func TestCompileOr(t *testing.T) {

	t.Parallel()

	test := func(t *testing.T, x, y vm.Value) vm.Value {
		result, err := compileAndInvoke(t,
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

func TestCompileAnd(t *testing.T) {

	t.Parallel()

	test := func(t *testing.T, x, y vm.Value) vm.Value {
		result, err := compileAndInvoke(t,
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

func TestCompileUnaryNot(t *testing.T) {

	t.Parallel()

	test := func(t *testing.T, argument vm.Value) vm.Value {

		actual, err := compileAndInvoke(t,
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

func TestCompileUnaryNegate(t *testing.T) {

	t.Parallel()

	actual, err := compileAndInvoke(t,
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

func TestCompileUnaryDeref(t *testing.T) {

	t.Parallel()

	actual, err := compileAndInvoke(t,
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

func TestCompileUnaryDerefSome(t *testing.T) {

	t.Parallel()

	actual, err := compileAndInvoke(t,
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
		interpreter.NewUnmeteredSomeValueNonCopying(interpreter.NewUnmeteredIntValueFromInt64(42)),
		actual,
	)
}

func TestCompileUnaryDerefNil(t *testing.T) {

	t.Parallel()

	actual, err := compileAndInvoke(t,
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

			actual, err := compileAndInvoke(t,
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

func TestCompileForce(t *testing.T) {

	t.Parallel()

	t.Run("non-nil", func(t *testing.T) {
		t.Parallel()

		actual, err := compileAndInvoke(t,
			`
                fun test(x: Int?): Int {
                    return x!
                }
            `,
			"test",
			interpreter.NewUnmeteredSomeValueNonCopying(interpreter.NewUnmeteredIntValueFromInt64(42)),
		)

		require.NoError(t, err)
		assert.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(42), actual)
	})

	t.Run("non-nil, AnyStruct", func(t *testing.T) {
		t.Parallel()

		actual, err := compileAndInvoke(t,
			`
                fun test(x: Int?): AnyStruct {
                    let y: AnyStruct = x
                    return y!
                }
            `,
			"test",
			interpreter.NewUnmeteredSomeValueNonCopying(interpreter.NewUnmeteredIntValueFromInt64(42)),
		)

		require.NoError(t, err)
		assert.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(42), actual)
	})

	t.Run("nil", func(t *testing.T) {
		t.Parallel()

		_, err := compileAndInvoke(t,
			`
                fun test(x: Int?): Int {
                    return x!
                }
            `,
			"test",
			interpreter.Nil,
		)
		require.Error(t, err)
		assert.ErrorIs(t, err, vm.ForceNilError{})
	})

	t.Run("nil, AnyStruct", func(t *testing.T) {
		t.Parallel()

		_, err := compileAndInvoke(t,
			`
                fun test(x: Int?): AnyStruct {
                    let y: AnyStruct = x
                    return y!
                }
            `,
			"test",
			interpreter.Nil,
		)
		require.Error(t, err)
		assert.ErrorIs(t, err, vm.ForceNilError{})
	})

	t.Run("non-optional", func(t *testing.T) {
		t.Parallel()

		actual, err := compileAndInvoke(t,
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

		actual, err := compileAndInvoke(t,
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

		actual, err := compileAndInvoke(t,
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

		actual, err := compileAndInvoke(t,
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

		actual, err := compileAndInvoke(t,
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

		actual, err := compileAndInvoke(t,
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

		actual, err := compileAndInvoke(t,
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

		actual, err, logs := compileAndInvokeWithLogs(t,
			`
              fun test(a: Bool): Int {
                  post {
                      printMessage("post condition executed")
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
                      printMessage("second condition reached 1")
                      printMessage("second condition reached 2")
                  }

                  return 2
              }

              view fun printMessage(_ msg: String): Bool {
                  log(msg)
                  return true
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

		actual, err, logs := compileAndInvokeWithLogs(t,
			`
              struct Foo {
                  var i: Int
                  init(_ a: Bool) {
                      post {
                          printMessage("post condition executed")
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

              view fun printMessage(_ msg: String): Bool {
                  log(msg)
                  return true
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

	actual, err := compileAndInvoke(t,
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

	actual, err := compileAndInvoke(t,
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

	importLocation := common.NewAddressLocation(nil, common.Address{0x1}, "C")

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

	importCompiler := compiler.NewInstructionCompiler(importedChecker)
	importedProgram := importCompiler.Compile()

	vmInstance := vm.NewVM(importLocation, importedProgram, nil)
	importedContractValue, err := vmInstance.InitializeContract()
	require.NoError(t, err)

	checker, err := ParseAndCheckWithOptions(t,
		`
          import C from 0x1

          fun test(): Address {
              return C.test()
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

	comp := compiler.NewInstructionCompiler(checker)
	comp.Config.ImportHandler = func(location common.Location) *bbq.InstructionProgram {
		return importedProgram
	}

	program := comp.Compile()

	addressValue := interpreter.NewUnmeteredAddressValueFromBytes([]byte{0x1})

	vmConfig := (&vm.Config{
		ImportHandler: func(location common.Location) *bbq.InstructionProgram {
			return importedProgram
		},
		ContractValueHandler: func(*vm.Config, common.Location) *interpreter.CompositeValue {
			return importedContractValue
		},
	}).WithInterpreterConfig(&interpreter.Config{
		InjectedCompositeFieldsHandler: func(
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
				interpreter.EmptyLocationRange,
			)

			return map[string]interpreter.Value{
				sema.ContractAccountFieldName: accountRef,
			}
		},
	})

	vmInstance = vm.NewVM(scriptLocation(), program, vmConfig)

	result, err := vmInstance.Invoke("test")
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

	importLocation := common.NewAddressLocation(nil, common.Address{0x1}, "C")

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

	importCompiler := compiler.NewInstructionCompiler(importedChecker)
	importedProgram := importCompiler.Compile()

	vmInstance := vm.NewVM(importLocation, importedProgram, nil)
	importedContractValue, err := vmInstance.InitializeContract()
	require.NoError(t, err)

	checker, err := ParseAndCheckWithOptions(t,
		`
          import C from 0x1

          fun test(): Address {
              return C.test()
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

	comp := compiler.NewInstructionCompiler(checker)
	comp.Config.ImportHandler = func(location common.Location) *bbq.InstructionProgram {
		return importedProgram
	}

	program := comp.Compile()

	addressValue := interpreter.NewUnmeteredAddressValueFromBytes([]byte{0x1})

	var uuid uint64 = 42

	vmConfig := (&vm.Config{
		ImportHandler: func(location common.Location) *bbq.InstructionProgram {
			return importedProgram
		},
		ContractValueHandler: func(*vm.Config, common.Location) *interpreter.CompositeValue {
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
	}).WithInterpreterConfig(&interpreter.Config{
		UUIDHandler: func() (uint64, error) {
			uuid++
			return uuid, nil
		},

		InjectedCompositeFieldsHandler: func(
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
				interpreter.EmptyLocationRange,
			)

			return map[string]interpreter.Value{
				sema.ContractAccountFieldName: accountRef,
			}
		},
		AccountHandler: func(context interpreter.AccountCreationContext, address interpreter.AddressValue) interpreter.Value {
			return stdlib.NewAccountValue(context, nil, address)
		},
	})

	vmInstance = vm.NewVM(scriptLocation(), program, vmConfig)

	result, err := vmInstance.Invoke("test")
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

	importLocation := common.NewAddressLocation(nil, common.Address{0x1}, "C")

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

	importCompiler := compiler.NewInstructionCompiler(importedChecker)
	importedProgram := importCompiler.Compile()

	vmInstance := vm.NewVM(importLocation, importedProgram, nil)
	importedContractValue, err := vmInstance.InitializeContract()
	require.NoError(t, err)

	checker, err := ParseAndCheckWithOptions(t,
		`
          import C from 0x1

          fun test(): UInt64 {
              return C.test()
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

	comp := compiler.NewInstructionCompiler(checker)
	comp.Config.ImportHandler = func(location common.Location) *bbq.InstructionProgram {
		return importedProgram
	}

	program := comp.Compile()

	var uuid uint64 = 42

	vmConfig := (&vm.Config{
		ImportHandler: func(location common.Location) *bbq.InstructionProgram {
			return importedProgram
		},
		ContractValueHandler: func(*vm.Config, common.Location) *interpreter.CompositeValue {
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
	}).WithInterpreterConfig(&interpreter.Config{
		UUIDHandler: func() (uint64, error) {
			uuid++
			return uuid, nil
		},
		AccountHandler: func(context interpreter.AccountCreationContext, address interpreter.AddressValue) interpreter.Value {
			return stdlib.NewAccountValue(context, nil, address)
		},
	})

	vmInstance = vm.NewVM(scriptLocation(), program, vmConfig)

	result, err := vmInstance.Invoke("test")
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

	actual, err := compileAndInvoke(t,
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

	actual, err := compileAndInvoke(t,
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

	actual, err := compileAndInvoke(t,
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

	actual, err := compileAndInvoke(t,
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

	actual, err := compileAndInvoke(t,
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

	actual, err := compileAndInvoke(t,
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

	actual, err := compileAndInvoke(t,
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

	actual, err := compileAndInvoke(t,
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

	actual, err := compileAndInvoke(t,
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

	actual, err := compileAndInvoke(t,
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

	actual, err := compileAndInvoke(t,
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

	importLocation := common.NewAddressLocation(nil, common.Address{0x1}, "Counter")

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

	importCompiler := compiler.NewInstructionCompiler(importedChecker)
	importedProgram := importCompiler.Compile()

	vmInstance := vm.NewVM(importLocation, importedProgram, nil)
	importedContractValue, err := vmInstance.InitializeContract()
	require.NoError(t, err)

	activation := sema.NewVariableActivation(sema.BaseValueActivation)
	activation.DeclareValue(stdlib.PanicFunction)

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
			Config: &sema.Config{
				ImportHandler: func(*sema.Checker, common.Location, ast.Range) (sema.Import, error) {
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

	comp := compiler.NewInstructionCompiler(checker)
	comp.Config.ImportHandler = func(location common.Location) *bbq.InstructionProgram {
		return importedProgram
	}

	program := comp.Compile()

	vmConfig := &vm.Config{
		ImportHandler: func(location common.Location) *bbq.InstructionProgram {
			return importedProgram
		},
		ContractValueHandler: func(vmConfig *vm.Config, location common.Location) *interpreter.CompositeValue {
			return importedContractValue
		},
	}

	vmInstance = vm.NewVM(scriptLocation(), program, vmConfig)

	result, err := vmInstance.Invoke("test")
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

			actual, err := compileAndInvoke(t,
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

			actual, err := compileAndInvoke(t,
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
	})

	t.Run("getIsInstance", func(t *testing.T) {

		t.Parallel()

		t.Run("int32, pass", func(t *testing.T) {

			t.Parallel()

			actual, err := compileAndInvoke(t,
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

			actual, err := compileAndInvoke(t,
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

			actual, err := compileAndInvoke(t,
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

			actual, err := compileAndInvoke(t,
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
