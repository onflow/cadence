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

	"github.com/onflow/cadence/bbq/opcode"
	"github.com/onflow/cadence/interpreter"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/stdlib"
	"github.com/onflow/cadence/test_utils/common_utils"
	"github.com/onflow/cadence/test_utils/runtime_utils"
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
	scriptLocation := runtime_utils.NewScriptLocationGenerator()
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
		vm.NewIntValue(23),
	)
	require.NoError(t, err)
	require.Equal(t, vm.NewIntValue(28657), result)
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
		vm.NewIntValue(7),
	)
	require.NoError(t, err)
	require.Equal(t, vm.NewIntValue(13), result)
	require.Equal(t, 0, vmInstance.StackSize())
}

func TestWhileBreak(t *testing.T) {

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

	comp := compiler.NewInstructionCompiler(checker)
	program := comp.Compile()

	vmConfig := &vm.Config{}
	vmInstance := vm.NewVM(scriptLocation(), program, vmConfig)

	result, err := vmInstance.Invoke("test")
	require.NoError(t, err)

	require.Equal(t, vm.NewIntValue(4), result)
	require.Equal(t, 0, vmInstance.StackSize())
}

func TestSwitchBreak(t *testing.T) {

	t.Parallel()

	test := func(t *testing.T, value int64) vm.Value {

		checker, err := ParseAndCheck(t, `
          fun test(x: Int): Int {
              switch x {
                  case 1:
                      break
                  default:
                      return 3
              }
              return 1
          }
        `)
		require.NoError(t, err)

		comp := compiler.NewInstructionCompiler(checker)
		program := comp.Compile()

		vmConfig := &vm.Config{}
		vmInstance := vm.NewVM(scriptLocation(), program, vmConfig)

		result, err := vmInstance.Invoke("test", vm.NewIntValue(value))
		require.NoError(t, err)
		require.Equal(t, 0, vmInstance.StackSize())

		return result
	}

	t.Run("1", func(t *testing.T) {
		t.Parallel()

		result := test(t, 1)
		require.Equal(t, vm.NewIntValue(1), result)
	})

	t.Run("2", func(t *testing.T) {
		t.Parallel()

		result := test(t, 2)
		require.Equal(t, vm.NewIntValue(3), result)
	})

	t.Run("3", func(t *testing.T) {
		t.Parallel()

		result := test(t, 3)
		require.Equal(t, vm.NewIntValue(3), result)
	})
}

func TestWhileSwitchBreak(t *testing.T) {

	t.Parallel()

	test := func(t *testing.T, value int64) vm.Value {

		checker, err := ParseAndCheck(t, `
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
        `)
		require.NoError(t, err)

		comp := compiler.NewInstructionCompiler(checker)
		program := comp.Compile()

		vmConfig := &vm.Config{}
		vmInstance := vm.NewVM(scriptLocation(), program, vmConfig)

		result, err := vmInstance.Invoke("test", vm.NewIntValue(value))
		require.NoError(t, err)
		require.Equal(t, 0, vmInstance.StackSize())

		return result
	}

	t.Run("1", func(t *testing.T) {
		t.Parallel()

		result := test(t, 1)
		require.Equal(t, vm.NewIntValue(1), result)
	})

	t.Run("2", func(t *testing.T) {
		t.Parallel()

		result := test(t, 2)
		require.Equal(t, vm.NewIntValue(3), result)
	})

	t.Run("3", func(t *testing.T) {
		t.Parallel()

		result := test(t, 3)
		require.Equal(t, vm.NewIntValue(3), result)
	})
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

	comp := compiler.NewInstructionCompiler(checker)
	program := comp.Compile()

	vmConfig := &vm.Config{}
	vmInstance := vm.NewVM(scriptLocation(), program, vmConfig)

	result, err := vmInstance.Invoke("test")
	require.NoError(t, err)

	require.Equal(t, vm.NewIntValue(3), result)
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

		comp := compiler.NewInstructionCompiler(checker)
		program := comp.Compile()

		vmConfig := &vm.Config{}
		vmInstance := vm.NewVM(scriptLocation(), program, vmConfig)

		result, err := vmInstance.Invoke("test")
		require.NoError(t, err)

		require.Equal(t, vm.NewIntValue(2), result)
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

		comp := compiler.NewInstructionCompiler(checker)
		program := comp.Compile()

		vmConfig := &vm.Config{}
		vmInstance := vm.NewVM(scriptLocation(), program, vmConfig)

		result, err := vmInstance.Invoke("test")
		require.NoError(t, err)

		require.Equal(t, vm.NewIntValue(3), result)
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

	comp := compiler.NewInstructionCompiler(checker)
	program := comp.Compile()

	vmConfig := &vm.Config{}
	vmInstance := vm.NewVM(scriptLocation(), program, vmConfig)

	result, err := vmInstance.Invoke("test", vm.NewIntValue(10))
	require.NoError(t, err)
	require.Equal(t, 0, vmInstance.StackSize())

	require.IsType(t, &vm.CompositeValue{}, result)
	structValue := result.(*vm.CompositeValue)
	compositeType := structValue.CompositeType

	require.Equal(t, "Foo", compositeType.QualifiedIdentifier)
	require.Equal(
		t,
		vm.NewIntValue(12),
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

	comp := compiler.NewInstructionCompiler(checker)
	program := comp.Compile()

	vmConfig := &vm.Config{}
	vmInstance := vm.NewVM(scriptLocation(), program, vmConfig)

	result, err := vmInstance.Invoke("test")
	require.NoError(t, err)
	require.Equal(t, 0, vmInstance.StackSize())

	require.Equal(t, vm.NewStringValue("Hello from Foo!"), result)
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
			Location: common_utils.ImportedLocation,
		},
	)
	require.NoError(t, err)

	subComp := compiler.NewInstructionCompiler(importedChecker)
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

	importCompiler := compiler.NewInstructionCompiler(checker)
	importCompiler.Config.ImportHandler = func(location common.Location) *bbq.Program[opcode.Instruction] {
		return importedProgram
	}

	program := importCompiler.Compile()

	vmConfig := &vm.Config{
		ImportHandler: func(location common.Location) *bbq.Program[opcode.Instruction] {
			return importedProgram
		},
	}

	vmInstance := vm.NewVM(scriptLocation(), program, vmConfig)

	result, err := vmInstance.Invoke("test")
	require.NoError(t, err)
	require.Equal(t, 0, vmInstance.StackSize())

	require.Equal(t, vm.NewStringValue("global function of the imported program"), result)
}

func TestContractImport(t *testing.T) {

	t.Parallel()

	t.Run("nested type def", func(t *testing.T) {

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

		comp := compiler.NewInstructionCompiler(checker)
		comp.Config.ImportHandler = func(location common.Location) *bbq.Program[opcode.Instruction] {
			return importedProgram
		}

		program := comp.Compile()

		vmConfig := &vm.Config{
			ImportHandler: func(location common.Location) *bbq.Program[opcode.Instruction] {
				return importedProgram
			},
			ContractValueHandler: func(*vm.Config, common.Location) *vm.CompositeValue {
				return importedContractValue
			},
		}

		vmInstance = vm.NewVM(scriptLocation(), program, vmConfig)

		result, err := vmInstance.Invoke("test")
		require.NoError(t, err)
		require.Equal(t, 0, vmInstance.StackSize())

		require.Equal(t, vm.NewStringValue("global function of the imported program"), result)
	})

	t.Run("contract function", func(t *testing.T) {
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

		comp := compiler.NewInstructionCompiler(checker)
		comp.Config.ImportHandler = func(location common.Location) *bbq.Program[opcode.Instruction] {
			return importedProgram
		}

		program := comp.Compile()

		vmConfig := &vm.Config{
			ImportHandler: func(location common.Location) *bbq.Program[opcode.Instruction] {
				return importedProgram
			},
			ContractValueHandler: func(*vm.Config, common.Location) *vm.CompositeValue {
				return importedContractValue
			},
		}

		vmInstance = vm.NewVM(scriptLocation(), program, vmConfig)

		result, err := vmInstance.Invoke("test")
		require.NoError(t, err)
		require.Equal(t, 0, vmInstance.StackSize())

		require.Equal(t, vm.NewStringValue("contract function of the imported program"), result)
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

		barCompiler := compiler.NewInstructionCompiler(barChecker)
		barCompiler.Config.LocationHandler = singleIdentifierLocationResolver(t)
		barCompiler.Config.ImportHandler = func(location common.Location) *bbq.Program[opcode.Instruction] {
			require.Equal(t, fooLocation, location)
			return fooProgram
		}

		barProgram := barCompiler.Compile()

		vmConfig := &vm.Config{
			ImportHandler: func(location common.Location) *bbq.Program[opcode.Instruction] {
				require.Equal(t, fooLocation, location)
				return fooProgram
			},
			ContractValueHandler: func(_ *vm.Config, location common.Location) *vm.CompositeValue {
				require.Equal(t, fooLocation, location)
				return fooContractValue
			},
		}

		vmInstance = vm.NewVM(barLocation, barProgram, vmConfig)
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

		comp := compiler.NewInstructionCompiler(checker)
		comp.Config.LocationHandler = singleIdentifierLocationResolver(t)
		comp.Config.ImportHandler = func(location common.Location) *bbq.Program[opcode.Instruction] {
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
			ImportHandler: func(location common.Location) *bbq.Program[opcode.Instruction] {
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

		vmInstance = vm.NewVM(scriptLocation(), program, vmConfig)

		result, err := vmInstance.Invoke("test")
		require.NoError(t, err)
		require.Equal(t, 0, vmInstance.StackSize())

		require.Equal(t, vm.NewStringValue("Hello from Foo!"), result)
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

		fooCompiler := compiler.NewInstructionCompiler(fooChecker)
		fooProgram := fooCompiler.Compile()

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

		barCompiler := compiler.NewInstructionCompiler(barChecker)
		barCompiler.Config.LocationHandler = singleIdentifierLocationResolver(t)
		barCompiler.Config.ImportHandler = func(location common.Location) *bbq.Program[opcode.Instruction] {
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
			ImportHandler: func(location common.Location) *bbq.Program[opcode.Instruction] {
				require.Equal(t, fooLocation, location)
				return fooProgram
			},
		}

		vmInstance := vm.NewVM(barLocation, barProgram, vmConfig)
		barContractValue, err := vmInstance.InitializeContract()
		require.NoError(t, err)

		// Compile and run main program

		checker, err := ParseAndCheckWithOptions(t, `
            import Bar from 0x02

            fun test(): String {
                return Bar.withdraw(50)
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

		comp := compiler.NewInstructionCompiler(checker)
		comp.Config.LocationHandler = singleIdentifierLocationResolver(t)
		comp.Config.ImportHandler = func(location common.Location) *bbq.Program[opcode.Instruction] {
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
			ImportHandler: func(location common.Location) *bbq.Program[opcode.Instruction] {
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

		vmInstance = vm.NewVM(scriptLocation(), program, vmConfig)

		result, err := vmInstance.Invoke("test")
		require.NoError(t, err)
		require.Equal(t, 0, vmInstance.StackSize())

		require.Equal(t, vm.NewStringValue("Successfully withdrew"), result)
	})
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

	comp := compiler.NewInstructionCompiler(checker)
	program := comp.Compile()

	vmConfig := &vm.Config{}
	vmInstance := vm.NewVM(scriptLocation(), program, vmConfig)
	contractValue, err := vmInstance.InitializeContract()
	require.NoError(t, err)

	fieldValue := contractValue.GetMember(vmConfig, "status")
	assert.Equal(t, vm.NewStringValue("PENDING"), fieldValue)
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

		comp := compiler.NewInstructionCompiler(checker)
		program := comp.Compile()

		vmConfig := &vm.Config{}
		vmInstance := vm.NewVM(scriptLocation(), program, vmConfig)
		contractValue, err := vmInstance.InitializeContract()
		require.NoError(t, err)

		fieldValue := contractValue.GetMember(vmConfig, "status")
		assert.Equal(t, vm.NewStringValue("PENDING"), fieldValue)
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

		comp := compiler.NewInstructionCompiler(checker)
		program := comp.Compile()

		vmConfig := &vm.Config{}
		vmInstance := vm.NewVM(scriptLocation(), program, vmConfig)
		contractValue, err := vmInstance.InitializeContract()
		require.NoError(t, err)

		fieldValue := contractValue.GetMember(vmConfig, "status")
		assert.Equal(t, vm.NewStringValue("PENDING"), fieldValue)
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

		comp := compiler.NewInstructionCompiler(checker)
		program := comp.Compile()

		vmConfig := &vm.Config{}
		vmInstance := vm.NewVM(scriptLocation(), program, vmConfig)

		result, err := vmInstance.Invoke("test")
		require.NoError(t, err)
		require.Equal(t, 0, vmInstance.StackSize())

		require.Equal(t, vm.NewIntValue(5), result)
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

		comp := compiler.NewInstructionCompiler(checker)
		program := comp.Compile()

		vmConfig := &vm.Config{}
		vmInstance := vm.NewVM(scriptLocation(), program, vmConfig)

		result, err := vmInstance.Invoke("init")
		require.NoError(t, err)
		require.Equal(t, 0, vmInstance.StackSize())

		require.IsType(t, &vm.CompositeValue{}, result)
	})
}

func TestContractField(t *testing.T) {

	t.Parallel()

	t.Run("get", func(t *testing.T) {
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
				Location: common.NewAddressLocation(nil, common.Address{0x1}, "MyContract"),
			},
		)
		require.NoError(t, err)

		importCompiler := compiler.NewInstructionCompiler(importedChecker)
		importedProgram := importCompiler.Compile()

		vmInstance := vm.NewVM(importLocation, importedProgram, nil)
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

		comp := compiler.NewInstructionCompiler(checker).
			WithConfig(&compiler.Config{
				ImportHandler: func(location common.Location) *bbq.Program[opcode.Instruction] {
					return importedProgram
				},
			})

		program := comp.Compile()

		vmConfig := &vm.Config{
			ImportHandler: func(location common.Location) *bbq.Program[opcode.Instruction] {
				return importedProgram
			},
			ContractValueHandler: func(vmConfig *vm.Config, location common.Location) *vm.CompositeValue {
				return importedContractValue
			},
		}

		vmInstance = vm.NewVM(scriptLocation(), program, vmConfig)
		result, err := vmInstance.Invoke("test")
		require.NoError(t, err)
		require.Equal(t, 0, vmInstance.StackSize())

		require.Equal(t, vm.NewStringValue("PENDING"), result)
	})

	t.Run("set", func(t *testing.T) {
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
				Location: common.NewAddressLocation(nil, common.Address{0x1}, "MyContract"),
			},
		)
		require.NoError(t, err)

		importCompiler := compiler.NewInstructionCompiler(importedChecker)
		importedProgram := importCompiler.Compile()

		vmInstance := vm.NewVM(importLocation, importedProgram, nil)
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

		comp := compiler.NewInstructionCompiler(checker)
		comp.Config.ImportHandler = func(location common.Location) *bbq.Program[opcode.Instruction] {
			return importedProgram
		}

		program := comp.Compile()

		vmConfig := &vm.Config{
			ImportHandler: func(location common.Location) *bbq.Program[opcode.Instruction] {
				return importedProgram
			},
			ContractValueHandler: func(vmConfig *vm.Config, location common.Location) *vm.CompositeValue {
				return importedContractValue
			},
		}

		vmInstance = vm.NewVM(scriptLocation(), program, vmConfig)

		result, err := vmInstance.Invoke("test")
		require.NoError(t, err)
		require.Equal(t, 0, vmInstance.StackSize())

		require.Equal(t, vm.NewStringValue("UPDATED"), result)

		fieldValue := importedContractValue.GetMember(vmConfig, "status")
		assert.Equal(t, vm.NewStringValue("UPDATED"), fieldValue)
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

		comp := compiler.NewInstructionCompiler(checker)
		program := comp.Compile()

		vmConfig := &vm.Config{}
		vmInstance := vm.NewVM(scriptLocation(), program, vmConfig)

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

		comp := compiler.NewInstructionCompiler(checker)
		program := comp.Compile()

		vmConfig := &vm.Config{}
		vmInstance := vm.NewVM(scriptLocation(), program, vmConfig)

		result, err := vmInstance.Invoke("test")
		require.NoError(t, err)
		require.Equal(t, 0, vmInstance.StackSize())

		require.Equal(t, vm.NewStringValue("Hello, World!"), result)
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

		require.IsType(t, &vm.CompositeValue{}, transaction)
		compositeValue := transaction.(*vm.CompositeValue)

		// At the beginning, 'a' is uninitialized
		assert.Nil(t, compositeValue.GetMember(vmConfig, "a"))

		// Invoke 'prepare'
		_, err = vmInstance.Invoke(commons.TransactionPrepareFunctionName, transaction)
		require.NoError(t, err)
		require.Equal(t, 0, vmInstance.StackSize())

		// Once 'prepare' is called, 'a' is initialized to "Hello!"
		assert.Equal(t, vm.NewStringValue("Hello!"), compositeValue.GetMember(vmConfig, "a"))

		// Invoke 'execute'
		_, err = vmInstance.Invoke(commons.TransactionExecuteFunctionName, transaction)
		require.NoError(t, err)
		require.Equal(t, 0, vmInstance.StackSize())

		// Once 'execute' is called, 'a' is initialized to "Hello, again!"
		assert.Equal(t, vm.NewStringValue("Hello again!"), compositeValue.GetMember(vmConfig, "a"))
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

		comp := compiler.NewInstructionCompiler(checker)
		program := comp.Compile()

		vmConfig := &vm.Config{}
		vmInstance := vm.NewVM(scriptLocation(), program, vmConfig)

		args := []vm.Value{
			vm.NewStringValue("Hello!"),
			vm.NewStringValue("Hello again!"),
		}

		err = vmInstance.ExecuteTransaction(args)
		require.NoError(t, err)
		require.Equal(t, 0, vmInstance.StackSize())

		// Rerun the same again using internal functions, to get the access to the transaction value.

		transaction, err := vmInstance.Invoke(commons.TransactionWrapperCompositeName)
		require.NoError(t, err)
		require.Equal(t, 0, vmInstance.StackSize())

		require.IsType(t, &vm.CompositeValue{}, transaction)
		compositeValue := transaction.(*vm.CompositeValue)

		// At the beginning, 'a' is uninitialized
		assert.Nil(t, compositeValue.GetMember(vmConfig, "a"))

		// Invoke 'prepare'
		_, err = vmInstance.Invoke(commons.TransactionPrepareFunctionName, transaction)
		require.NoError(t, err)
		require.Equal(t, 0, vmInstance.StackSize())

		// Once 'prepare' is called, 'a' is initialized to "Hello!"
		assert.Equal(t, vm.NewStringValue("Hello!"), compositeValue.GetMember(vmConfig, "a"))

		// Invoke 'execute'
		_, err = vmInstance.Invoke(commons.TransactionExecuteFunctionName, transaction)
		require.NoError(t, err)
		require.Equal(t, 0, vmInstance.StackSize())

		// Once 'execute' is called, 'a' is initialized to "Hello, again!"
		assert.Equal(t, vm.NewStringValue("Hello again!"), compositeValue.GetMember(vmConfig, "a"))
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

		comp := compiler.NewInstructionCompiler(checker)
		comp.Config.LocationHandler = singleIdentifierLocationResolver(t)
		comp.Config.ImportHandler = func(location common.Location) *bbq.Program[opcode.Instruction] {
			return importedProgram
		}

		program := comp.Compile()

		vmConfig := &vm.Config{
			ImportHandler: func(location common.Location) *bbq.Program[opcode.Instruction] {
				return importedProgram
			},
			ContractValueHandler: func(vmConfig *vm.Config, location common.Location) *vm.CompositeValue {
				return importedContractValue
			},
			TypeLoader: func(location common.Location, typeID interpreter.TypeID) sema.CompositeKindedType {
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

		require.Equal(t, vm.NewStringValue("Hello from Foo!"), result)
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
        }`,
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
        }`,
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
        }`,
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

		bazImportHandler := func(location common.Location) *bbq.Program[opcode.Instruction] {
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
			ContractValueHandler: func(vmConfig *vm.Config, location common.Location) *vm.CompositeValue {
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

		checker, err := ParseAndCheckWithOptions(t, `
        import Baz from 0x03

        fun test(): Baz.GreetingImpl {
            return Baz.GreetingImpl()
        }`,

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

		scriptImportHandler := func(location common.Location) *bbq.Program[opcode.Instruction] {
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
			ContractValueHandler: func(vmConfig *vm.Config, location common.Location) *vm.CompositeValue {
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

		require.IsType(t, &vm.CompositeValue{}, implValue)
		compositeValue := implValue.(*vm.CompositeValue)
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

		checker, err = ParseAndCheckWithOptions(t, `
        import Foo from 0x01

        fun test(v: {Foo.Greetings}): String {
            return v.sayHello()
        }`,

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

		scriptImportHandler = func(location common.Location) *bbq.Program[opcode.Instruction] {
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
			ContractValueHandler: func(vmConfig *vm.Config, location common.Location) *vm.CompositeValue {
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

		require.Equal(t, vm.NewStringValue("Hello from Bar!"), result)
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

		require.IsType(t, &vm.ArrayValue{}, result)
		array := result.(*vm.ArrayValue)
		assert.Equal(t, 2, array.Count())
		assert.Equal(t, vm.NewIntValue(2), array.Get(vmConfig, 0))
		assert.Equal(t, vm.NewIntValue(5), array.Get(vmConfig, 1))
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
		assert.Equal(t, vm.NewIntValue(5), result)
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

		require.IsType(t, &vm.ArrayValue{}, result)
		array := result.(*vm.ArrayValue)
		assert.Equal(t, 3, array.Count())
		assert.Equal(t, vm.NewIntValue(2), array.Get(vmConfig, 0))
		assert.Equal(t, vm.NewIntValue(5), array.Get(vmConfig, 1))
		assert.Equal(t, vm.NewIntValue(8), array.Get(vmConfig, 2))
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

		require.IsType(t, &vm.DictionaryValue{}, result)
		dictionary := result.(*vm.DictionaryValue)
		assert.Equal(t, 2, dictionary.Count())
		assert.Equal(t,
			vm.NewSomeValueNonCopying(vm.NewIntValue(2)),
			dictionary.GetKey(vmConfig, vm.NewStringValue("b")),
		)
		assert.Equal(t,
			vm.NewSomeValueNonCopying(vm.NewIntValue(5)),
			dictionary.GetKey(vmConfig, vm.NewStringValue("e")),
		)
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

		comp := compiler.NewInstructionCompiler(checker)
		program := comp.Compile()

		vmConfig := vm.NewConfig(nil)

		vmInstance := vm.NewVM(scriptLocation(), program, vmConfig)

		result, err := vmInstance.Invoke("test")
		require.NoError(t, err)
		require.Equal(t, 0, vmInstance.StackSize())

		require.Equal(t, vm.NewStringValue("Hello from Foo!"), result)
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

		vmConfig := &vm.Config{}
		vmInstance := vm.NewVM(scriptLocation(), program, vmConfig)

		result, err := vmInstance.Invoke("test")
		require.NoError(t, err)
		require.Equal(t, 0, vmInstance.StackSize())

		require.IsType(t, &vm.CompositeValue{}, result)
		structValue := result.(*vm.CompositeValue)
		compositeType := structValue.CompositeType

		require.Equal(t, "Foo", compositeType.QualifiedIdentifier)
		require.Equal(
			t,
			vm.NewIntValue(5),
			structValue.GetMember(vmConfig, "id"),
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

		vmConfig := &vm.Config{}
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

		result, err := compileAndInvoke(t, `
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
		require.Equal(t, vm.NewIntValue(42), result)
	})

	t.Run("overridden", func(t *testing.T) {
		t.Parallel()

		result, err := compileAndInvoke(t, `
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
		require.Equal(t, vm.NewIntValue(42), result)
	})

	t.Run("default method via different paths", func(t *testing.T) {

		t.Parallel()

		result, err := compileAndInvoke(t, `
            struct interface A {
                access(all) fun test(): Int {
                    return 3
                }
            }

            struct interface B: A {}

            struct interface C: A {}

            struct D: B, C {}

            access(all) fun main(): Int {
                let d = D()
                return d.test()
            }
        `,
			"main",
		)

		require.NoError(t, err)
		assert.Equal(t, vm.NewIntValue(3), result)
	})

	t.Run("in different contract", func(t *testing.T) {

		t.Parallel()

		storage := interpreter.NewInMemoryStorage(nil)

		programs := map[common.Location]*compiledProgram{}
		contractValues := map[common.Location]*vm.CompositeValue{}

		vmConfig := &vm.Config{
			Storage:        storage,
			AccountHandler: &testAccountHandler{},
			ImportHandler: func(location common.Location) *bbq.Program[opcode.Instruction] {
				program, ok := programs[location]
				if !ok {
					assert.FailNow(t, "invalid location")
				}
				return program.Program
			},
			ContractValueHandler: func(_ *vm.Config, location common.Location) *vm.CompositeValue {
				contractValue, ok := contractValues[location]
				if !ok {
					assert.FailNow(t, "invalid location")
				}
				return contractValue
			},
		}

		contractsAddress := common.MustBytesToAddress([]byte{0x1})

		barLocation := common.NewAddressLocation(nil, contractsAddress, "Bar")
		fooLocation := common.NewAddressLocation(nil, contractsAddress, "Foo")

		// Deploy interface contract

		barContract := `
        access(all) contract interface Bar {

            access(all) resource interface VaultInterface {

                access(all) var balance: Int

                access(all) fun getBalance(): Int {
                    return self.balance
                }
            }
        }
    `

		// Only need to compile
		parseCheckAndCompile(t, barContract, barLocation, programs)

		// Deploy contract with the implementation

		fooContract := fmt.Sprintf(`
        import Bar from %[1]s

        access(all) contract Foo {

            access(all) resource Vault: Bar.VaultInterface {
                access(all) var balance: Int

                init(balance: Int) {
                    self.balance = balance
                }

                access(all) fun withdraw(amount: Int): @Vault {
                    self.balance = self.balance - amount
                    return <-create Vault(balance: amount)
                }
            }

            access(all) fun createVault(balance: Int): @Vault {
                return <- create Vault(balance: balance)
            }
        }`,
			contractsAddress.HexWithPrefix(),
		)

		fooProgram := parseCheckAndCompile(t, fooContract, fooLocation, programs)

		fooVM := vm.NewVM(fooLocation, fooProgram, vmConfig)

		fooContractValue, err := fooVM.InitializeContract()
		require.NoError(t, err)
		contractValues[fooLocation] = fooContractValue

		// Run transaction

		tx := fmt.Sprintf(`
            import Foo from %[1]s

            fun main(): Int {
               var vault <- Foo.createVault(balance: 10)
               destroy vault.withdraw(amount: 3)
               var balance = vault.getBalance()
               destroy vault
               return balance
            }`,
			contractsAddress.HexWithPrefix(),
		)

		txLocation := runtime_utils.NewTransactionLocationGenerator()

		txProgram := parseCheckAndCompile(t, tx, txLocation(), programs)
		txVM := vm.NewVM(txLocation(), txProgram, vmConfig)

		result, err := txVM.Invoke("main")
		require.NoError(t, err)
		require.Equal(t, 0, txVM.StackSize())
		require.Equal(t, vm.NewIntValue(7), result)
	})

	t.Run("in different contract with nested call", func(t *testing.T) {

		t.Parallel()

		storage := interpreter.NewInMemoryStorage(nil)

		programs := map[common.Location]*compiledProgram{}
		contractValues := map[common.Location]*vm.CompositeValue{}

		vmConfig := &vm.Config{
			Storage:        storage,
			AccountHandler: &testAccountHandler{},
			ImportHandler: func(location common.Location) *bbq.Program[opcode.Instruction] {
				program, ok := programs[location]
				if !ok {
					assert.FailNow(t, "invalid location")
				}
				return program.Program
			},
			ContractValueHandler: func(_ *vm.Config, location common.Location) *vm.CompositeValue {
				contractValue, ok := contractValues[location]
				if !ok {
					assert.FailNow(t, "invalid location")
				}
				return contractValue
			},
		}

		contractsAddress := common.MustBytesToAddress([]byte{0x1})

		barLocation := common.NewAddressLocation(nil, contractsAddress, "Bar")
		fooLocation := common.NewAddressLocation(nil, contractsAddress, "Foo")

		// Deploy interface contract

		barContract := `
        access(all) contract interface Bar {

            access(all) resource interface HelloInterface {

                access(all) fun sayHello(): String {
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

		fooContract := fmt.Sprintf(`
        import Bar from %[1]s

        access(all) contract Foo {

            access(all) resource Hello: Bar.HelloInterface { }

            access(all) fun createHello(): @Hello {
                return <- create Hello()
            }
        }`,
			contractsAddress.HexWithPrefix(),
		)

		fooProgram := parseCheckAndCompile(t, fooContract, fooLocation, programs)

		fooVM := vm.NewVM(fooLocation, fooProgram, vmConfig)

		fooContractValue, err := fooVM.InitializeContract()
		require.NoError(t, err)
		contractValues[fooLocation] = fooContractValue

		// Run transaction

		tx := fmt.Sprintf(`
            import Foo from %[1]s

            fun main(): String {
               var hello <- Foo.createHello()
               var msg = hello.sayHello()
               destroy hello
               return msg
            }`,
			contractsAddress.HexWithPrefix(),
		)

		txLocation := runtime_utils.NewTransactionLocationGenerator()

		txProgram := parseCheckAndCompile(t, tx, txLocation(), programs)
		txVM := vm.NewVM(txLocation(), txProgram, vmConfig)

		result, err := txVM.Invoke("main")
		require.NoError(t, err)
		require.Equal(t, 0, txVM.StackSize())
		require.Equal(t, vm.NewStringValue("Hello from HelloInterface"), result)
	})

	t.Run("in different contract nested call overridden", func(t *testing.T) {

		t.Parallel()

		storage := interpreter.NewInMemoryStorage(nil)

		programs := map[common.Location]*compiledProgram{}
		contractValues := map[common.Location]*vm.CompositeValue{}

		vmConfig := &vm.Config{
			Storage:        storage,
			AccountHandler: &testAccountHandler{},
			ImportHandler: func(location common.Location) *bbq.Program[opcode.Instruction] {
				program, ok := programs[location]
				if !ok {
					assert.FailNow(t, "invalid location")
				}
				return program.Program
			},
			ContractValueHandler: func(_ *vm.Config, location common.Location) *vm.CompositeValue {
				contractValue, ok := contractValues[location]
				if !ok {
					assert.FailNow(t, "invalid location")
				}
				return contractValue
			},
		}

		contractsAddress := common.MustBytesToAddress([]byte{0x1})

		barLocation := common.NewAddressLocation(nil, contractsAddress, "Bar")
		fooLocation := common.NewAddressLocation(nil, contractsAddress, "Foo")

		// Deploy interface contract

		barContract := `
        access(all) contract interface Bar {

            access(all) resource interface HelloInterface {

                access(all) fun sayHello(): String {
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

		fooContract := fmt.Sprintf(`
        import Bar from %[1]s

        access(all) contract Foo {

            access(all) resource Hello: Bar.HelloInterface {

                // Override one of the functions (one at the bottom of the call hierarchy)
                access(contract) fun sayHelloImpl(): String {
                    return "Hello from Hello"
                }
            }

            access(all) fun createHello(): @Hello {
                return <- create Hello()
            }
        }`,
			contractsAddress.HexWithPrefix(),
		)

		fooProgram := parseCheckAndCompile(t, fooContract, fooLocation, programs)

		fooVM := vm.NewVM(fooLocation, fooProgram, vmConfig)

		fooContractValue, err := fooVM.InitializeContract()
		require.NoError(t, err)
		contractValues[fooLocation] = fooContractValue

		// Run transaction

		tx := fmt.Sprintf(`
            import Foo from %[1]s

            fun main(): String {
               var hello <- Foo.createHello()
               var msg = hello.sayHello()
               destroy hello
               return msg
            }`,
			contractsAddress.HexWithPrefix(),
		)

		txLocation := runtime_utils.NewTransactionLocationGenerator()

		txProgram := parseCheckAndCompile(t, tx, txLocation(), programs)
		txVM := vm.NewVM(txLocation(), txProgram, vmConfig)

		result, err := txVM.Invoke("main")
		require.NoError(t, err)
		require.Equal(t, 0, txVM.StackSize())
		require.Equal(t, vm.NewStringValue("Hello from Hello"), result)
	})
}

func TestFunctionPreConditions(t *testing.T) {

	t.Parallel()

	t.Run("failed", func(t *testing.T) {

		t.Parallel()

		_, err := compileAndInvoke(t, `
            fun main(x: Int): Int {
                pre {
                    x == 0
                }
                return x
            }`,
			"main",
			vm.NewIntValue(3),
		)

		require.Error(t, err)
		assert.ErrorContains(t, err, "pre/post condition failed")
	})

	t.Run("failed with message", func(t *testing.T) {

		t.Parallel()

		_, err := compileAndInvoke(t, `
            fun main(x: Int): Int {
                pre {
                    x == 0: "x must be zero"
                }
                return x
            }`,
			"main",
			vm.NewIntValue(3),
		)

		require.Error(t, err)
		assert.ErrorContains(t, err, "x must be zero")
	})

	t.Run("passed", func(t *testing.T) {

		t.Parallel()

		result, err := compileAndInvoke(t, `
            fun main(x: Int): Int {
                pre {
                    x != 0
                }
                return x
            }`,
			"main",
			vm.NewIntValue(3),
		)

		require.NoError(t, err)
		assert.Equal(t, vm.NewIntValue(3), result)
	})

	t.Run("inherited", func(t *testing.T) {

		t.Parallel()

		_, err := compileAndInvoke(t, `
            struct interface A {
                access(all) fun test(_ a: Int): Int {
                    pre { a > 10: "a must be larger than 10" }
                }
            }

            struct interface B: A {
                access(all) fun test(_ a: Int): Int
            }

            struct C: B {
                fun test(_ a: Int): Int {
                    return a + 3
                }
            }

            access(all) fun main(_ a: Int): Int {
                let c = C()
                return c.test(a)
            }`,
			"main",
			vm.NewIntValue(4),
		)

		require.Error(t, err)
		assert.ErrorContains(t, err, "a must be larger than 10")
	})

	t.Run("pre conditions order", func(t *testing.T) {

		t.Parallel()

		code := `struct A: B {
                access(all) fun test() {
                    pre { print("A") }
                }
            }

            struct interface B: C, D {
                access(all) fun test() {
                    pre { print("B") }
                }
            }

            struct interface C: E, F {
                access(all) fun test() {
                    pre { print("C") }
                }
            }

            struct interface D: F {
                access(all) fun test() {
                    pre { print("D") }
                }
            }

            struct interface E {
                access(all) fun test() {
                    pre { print("E") }
                }
            }

            struct interface F {
                access(all) fun test() {
                    pre { print("F") }
                }
            }

            access(all) view fun print(_ msg: String): Bool {
                log(msg)
                return true
            }

            access(all) fun main() {
                let a = A()
                a.test()
            }`

		location := common.ScriptLocation{0x1}

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

		var logs []string

		config := vm.NewConfig(interpreter.NewInMemoryStorage(nil))
		config.NativeFunctionsProvider = func() map[string]vm.Value {
			return map[string]vm.Value{
				commons.LogFunctionName: vm.NativeFunctionValue{
					ParameterCount: len(stdlib.LogFunctionType.Parameters),
					Function: func(config *vm.Config, typeArguments []interpreter.StaticType, arguments ...vm.Value) vm.Value {
						logs = append(logs, arguments[0].String())
						return vm.VoidValue{}
					},
				},
				commons.PanicFunctionName: vm.NativeFunctionValue{
					ParameterCount: len(stdlib.PanicFunctionType.Parameters),
					Function: func(config *vm.Config, typeArguments []interpreter.StaticType, arguments ...vm.Value) vm.Value {
						messageValue, ok := arguments[0].(vm.StringValue)
						if !ok {
							panic(errors.NewUnreachableError())
						}

						panic(stdlib.PanicError{
							Message: string(messageValue.Str),
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
		assert.Equal(t, []string{"B", "C", "E", "F", "D", "A"}, logs)
	})

	t.Run("in different contract with nested call", func(t *testing.T) {

		t.Parallel()

		storage := interpreter.NewInMemoryStorage(nil)

		programs := map[common.Location]*compiledProgram{}
		contractValues := map[common.Location]*vm.CompositeValue{}
		var logs []string

		vmConfig := &vm.Config{
			Storage:        storage,
			AccountHandler: &testAccountHandler{},
			ImportHandler: func(location common.Location) *bbq.Program[opcode.Instruction] {
				program, ok := programs[location]
				if !ok {
					assert.FailNow(t, "invalid location")
				}
				return program.Program
			},
			ContractValueHandler: func(_ *vm.Config, location common.Location) *vm.CompositeValue {
				contractValue, ok := contractValues[location]
				if !ok {
					assert.FailNow(t, "invalid location")
				}
				return contractValue
			},

			NativeFunctionsProvider: func() map[string]vm.Value {
				funcs := vm.NativeFunctions()
				funcs[commons.LogFunctionName] = vm.NativeFunctionValue{
					ParameterCount: len(stdlib.LogFunctionType.Parameters),
					Function: func(config *vm.Config, typeArguments []interpreter.StaticType, arguments ...vm.Value) vm.Value {
						logs = append(logs, arguments[0].String())
						return vm.VoidValue{}
					},
				}

				return funcs
			},
		}

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

		contractsAddress := common.MustBytesToAddress([]byte{0x1})

		barLocation := common.NewAddressLocation(nil, contractsAddress, "Bar")
		fooLocation := common.NewAddressLocation(nil, contractsAddress, "Foo")

		// Deploy interface contract

		barContract := `
        access(all) contract interface Bar {

            struct interface E {
                access(all) fun test() {
                    pre { self.printFromE("E") }
                }

                access(all) view fun printFromE(_ msg: String): Bool {
                    log("Bar.".concat(msg))
                    return true
                }
            }

            struct interface F {
                access(all) fun test() {
                    pre { self.printFromF("F") }
                }

                access(all) view fun printFromF(_ msg: String): Bool {
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

		fooContract := fmt.Sprintf(`
        import Bar from %[1]s

        access(all) contract Foo {

            struct interface B: C, D {
                access(all) fun test() {
                    pre { Foo.printFromFoo("B") }
                }
            }

            struct interface C: Bar.E, Bar.F {
                access(all) fun test() {
                    pre { Foo.printFromFoo("C") }
                }
            }

            struct interface D: Bar.F {
                access(all) fun test() {
                    pre { Foo.printFromFoo("D") }
                }
            }

            access(all) view fun printFromFoo(_ msg: String): Bool {
                log("Foo.".concat(msg))
                return true
            }
        }`,
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

		code := fmt.Sprintf(`
            import Foo from %[1]s

            access(all) struct A: Foo.B {
                access(all) fun test() {
                    pre { print("A") }
                }
            }

            access(all) view fun print(_ msg: String): Bool {
                log(msg)
                return true
            }

            access(all) fun main() {
                let a = A()
                a.test()
            }`,
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
		assert.Equal(t, []string{"Foo.B", "Foo.C", "Bar.E", "Bar.F", "Foo.D", "A"}, logs)
	})
}

func TestFunctionPostConditions(t *testing.T) {

	t.Parallel()

	t.Run("failed", func(t *testing.T) {

		t.Parallel()

		_, err := compileAndInvoke(t, `
            fun main(x: Int): Int {
                post {
                    x == 0
                }
                return x
            }`,
			"main",
			vm.NewIntValue(3),
		)

		require.Error(t, err)
		assert.ErrorContains(t, err, "pre/post condition failed")
	})

	t.Run("failed with message", func(t *testing.T) {

		t.Parallel()

		_, err := compileAndInvoke(t, `
            fun main(x: Int): Int {
                post {
                    x == 0: "x must be zero"
                }
                return x
            }`,
			"main",
			vm.NewIntValue(3),
		)

		require.Error(t, err)
		assert.ErrorContains(t, err, "x must be zero")
	})

	t.Run("passed", func(t *testing.T) {

		t.Parallel()

		result, err := compileAndInvoke(t, `
            fun main(x: Int): Int {
                post {
                    x != 0
                }
                return x
            }`,
			"main",
			vm.NewIntValue(3),
		)

		require.NoError(t, err)
		assert.Equal(t, vm.NewIntValue(3), result)
	})

	t.Run("test on local var", func(t *testing.T) {

		t.Parallel()

		result, err := compileAndInvoke(t, `
            fun main(x: Int): Int {
                post {
                    y == 5
                }
                var y = x + 2
                return y
            }`,
			"main",
			vm.NewIntValue(3),
		)

		require.NoError(t, err)
		assert.Equal(t, vm.NewIntValue(5), result)
	})

	t.Run("test on local var failed with message", func(t *testing.T) {

		t.Parallel()

		_, err := compileAndInvoke(t, `
            fun main(x: Int): Int {
                post {
                    y == 5: "x must be 5"
                }
                var y = x + 2
                return y
            }`,
			"main",
			vm.NewIntValue(4),
		)

		require.Error(t, err)
		assert.ErrorContains(t, err, "x must be 5")
	})

	t.Run("post conditions order", func(t *testing.T) {

		t.Parallel()

		code := `
            struct A: B {
                access(all) fun test() {
                    post { print("A") }
                }
            }

            struct interface B: C, D {
                access(all) fun test() {
                    post { print("B") }
                }
            }

            struct interface C: E, F {
                access(all) fun test() {
                    post { print("C") }
                }
            }

            struct interface D: F {
                access(all) fun test() {
                    post { print("D") }
                }
            }

            struct interface E {
                access(all) fun test() {
                    post { print("E") }
                }
            }

            struct interface F {
                access(all) fun test() {
                    post { print("F") }
                }
            }

            access(all) view fun print(_ msg: String): Bool {
                log(msg)
                return true
            }

            access(all) fun main() {
                let a = A()
                a.test()
            }`

		location := common.ScriptLocation{0x1}

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

		var logs []string

		config := vm.NewConfig(interpreter.NewInMemoryStorage(nil))
		config.NativeFunctionsProvider = func() map[string]vm.Value {
			funcs := vm.NativeFunctions()
			funcs[commons.LogFunctionName] = vm.NativeFunctionValue{
				ParameterCount: len(stdlib.LogFunctionType.Parameters),
				Function: func(config *vm.Config, typeArguments []interpreter.StaticType, arguments ...vm.Value) vm.Value {
					logs = append(logs, arguments[0].String())
					return vm.VoidValue{}
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
		assert.Equal(t, []string{"A", "D", "F", "E", "C", "B"}, logs)
	})

	t.Run("result var failed", func(t *testing.T) {

		t.Parallel()

		_, err := compileAndInvoke(t, `
            fun main(x: Int): Int {
                post {
                    result == 0: "x must be zero"
                }
                return x
            }`,
			"main",
			vm.NewIntValue(3),
		)

		require.Error(t, err)
		assert.ErrorContains(t, err, "x must be zero")
	})

	t.Run("result var passed", func(t *testing.T) {

		t.Parallel()

		result, err := compileAndInvoke(t, `
            fun main(x: Int): Int {
                post {
                    result != 0
                }
                return x
            }`,
			"main",
			vm.NewIntValue(3),
		)

		require.NoError(t, err)
		assert.Equal(t, vm.NewIntValue(3), result)
	})

	t.Run("result var in inherited condition", func(t *testing.T) {

		t.Parallel()

		_, err := compileAndInvoke(t, `
            struct interface A {
                access(all) fun test(_ a: Int): Int {
                    post { result > 10: "result must be larger than 10" }
                }
            }

            struct interface B: A {
                access(all) fun test(_ a: Int): Int
            }

            struct C: B {
                fun test(_ a: Int): Int {
                    return a + 3
                }
            }

            access(all) fun main(_ a: Int): Int {
                let c = C()
                return c.test(a)
            }`,
			"main",
			vm.NewIntValue(4),
		)

		require.Error(t, err)
		assert.ErrorContains(t, err, "result must be larger than 10")
	})

	t.Run("resource typed result var passed", func(t *testing.T) {

		t.Parallel()

		result, err := compileAndInvoke(t, `
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
            }`,
			"main",
		)

		require.NoError(t, err)
		assert.IsType(t, &vm.CompositeValue{}, result)
	})

	t.Run("resource typed result var failed", func(t *testing.T) {

		t.Parallel()

		_, err := compileAndInvoke(t, `
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
            }`,
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
			vm.NewSomeValueNonCopying(
				vm.NewIntValue(1),
			),
		)
		assert.Equal(t, vm.NewIntValue(1), actual)
	})

	t.Run("nil", func(t *testing.T) {

		t.Parallel()

		actual := test(t, vm.NilValue{})
		assert.Equal(t, vm.NewIntValue(2), actual)
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
			vm.NewSomeValueNonCopying(
				vm.NewIntValue(10),
			),
		)
		assert.Equal(t, vm.NewIntValue(11), actual)
	})

	t.Run("nil", func(t *testing.T) {

		t.Parallel()

		actual := test(t, vm.NilValue{})
		assert.Equal(t, vm.NewIntValue(2), actual)
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
			vm.NewIntValue(1),
		)
		require.NoError(t, err)
		assert.Equal(t, vm.NewIntValue(1), result)
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
			vm.NewIntValue(2),
		)
		require.NoError(t, err)
		assert.Equal(t, vm.NewIntValue(2), result)
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
			vm.NewIntValue(4),
		)
		require.NoError(t, err)
		assert.Equal(t, vm.NewIntValue(3), result)
	})
}

func TestDefaultFunctionsWithConditions(t *testing.T) {

	t.Parallel()

	t.Run("default in parent, conditions in child", func(t *testing.T) {
		t.Parallel()

		storage := interpreter.NewInMemoryStorage(nil)

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

		var logs []string
		vmConfig := &vm.Config{
			Storage:        storage,
			AccountHandler: &testAccountHandler{},
			NativeFunctionsProvider: func() map[string]vm.Value {
				funcs := vm.NativeFunctions()
				funcs[commons.LogFunctionName] = vm.NativeFunctionValue{
					ParameterCount: len(stdlib.LogFunctionType.Parameters),
					Function: func(config *vm.Config, typeArguments []interpreter.StaticType, arguments ...vm.Value) vm.Value {
						logs = append(logs, arguments[0].String())
						return vm.VoidValue{}
					},
				}

				return funcs
			},
		}

		_, err := compileAndInvokeWithOptions(t, `
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

            access(all) view fun printMessage(_ msg: String): Bool {
                log(msg)
                return true
            }

            fun main() {
               Test().test(5)
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
				"invoked Bar.test() pre-condition",
				"invoked Foo.test()",
				"invoked Bar.test() post-condition",
			}, logs,
		)
	})

	t.Run("default and conditions in parent, more conditions in child", func(t *testing.T) {
		t.Parallel()

		storage := interpreter.NewInMemoryStorage(nil)

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

		var logs []string
		vmConfig := &vm.Config{
			Storage:        storage,
			AccountHandler: &testAccountHandler{},
			NativeFunctionsProvider: func() map[string]vm.Value {
				funcs := vm.NativeFunctions()
				funcs[commons.LogFunctionName] = vm.NativeFunctionValue{
					ParameterCount: len(stdlib.LogFunctionType.Parameters),
					Function: func(config *vm.Config, typeArguments []interpreter.StaticType, arguments ...vm.Value) vm.Value {
						logs = append(logs, arguments[0].String())
						return vm.VoidValue{}
					},
				}

				return funcs
			},
		}

		_, err := compileAndInvokeWithOptions(t, `
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

            access(all) view fun printMessage(_ msg: String): Bool {
                log(msg)
                return true
            }

            fun main() {
               Test().test(5)
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
				"invoked Bar.test() pre-condition",
				"invoked Foo.test() pre-condition",
				"invoked Foo.test()",
				"invoked Foo.test() post-condition",
				"invoked Bar.test() post-condition",
			}, logs,
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

		var logs []string
		vmConfig := &vm.Config{
			Storage:        storage,
			AccountHandler: &testAccountHandler{},
			NativeFunctionsProvider: func() map[string]vm.Value {
				funcs := vm.NativeFunctions()
				funcs[commons.LogFunctionName] = vm.NativeFunctionValue{
					ParameterCount: len(stdlib.LogFunctionType.Parameters),
					Function: func(config *vm.Config, typeArguments []interpreter.StaticType, arguments ...vm.Value) vm.Value {
						logs = append(logs, arguments[0].String())
						return vm.VoidValue{}
					},
				}

				return funcs
			},
		}

		_, err := compileAndInvokeWithOptions(t, `
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

            access(all) view fun print(_ msg: String): Bool {
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
				"2",
				"5",
			}, logs,
		)
	})

	t.Run("inherited condition", func(t *testing.T) {
		t.Parallel()

		storage := interpreter.NewInMemoryStorage(nil)

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

		var logs []string
		vmConfig := &vm.Config{
			Storage:        storage,
			AccountHandler: &testAccountHandler{},
			NativeFunctionsProvider: func() map[string]vm.Value {
				funcs := vm.NativeFunctions()
				funcs[commons.LogFunctionName] = vm.NativeFunctionValue{
					ParameterCount: len(stdlib.LogFunctionType.Parameters),
					Function: func(config *vm.Config, typeArguments []interpreter.StaticType, arguments ...vm.Value) vm.Value {
						logs = append(logs, arguments[0].String())
						return vm.VoidValue{}
					},
				}

				return funcs
			},
		}

		_, err := compileAndInvokeWithOptions(t, `
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

            access(all) view fun print(_ msg: String): Bool {
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
				"2",
				"5",
			}, logs,
		)
	})

	t.Run("multiple inherited conditions", func(t *testing.T) {
		t.Parallel()

		storage := interpreter.NewInMemoryStorage(nil)

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

		var logs []string
		vmConfig := &vm.Config{
			Storage:        storage,
			AccountHandler: &testAccountHandler{},
			NativeFunctionsProvider: func() map[string]vm.Value {
				funcs := vm.NativeFunctions()
				funcs[commons.LogFunctionName] = vm.NativeFunctionValue{
					ParameterCount: len(stdlib.LogFunctionType.Parameters),
					Function: func(config *vm.Config, typeArguments []interpreter.StaticType, arguments ...vm.Value) vm.Value {
						logs = append(logs, arguments[0].String())
						return vm.VoidValue{}
					},
				}

				return funcs
			},
		}

		_, err := compileAndInvokeWithOptions(t, `
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

            access(all) view fun print(_ msg: String): Bool {
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
			[]string{"2", "3", "8", "5"},
			logs,
		)
	})

	t.Run("resource access in inherited before-statement", func(t *testing.T) {

		t.Parallel()

		_, err := compileAndInvoke(t, `
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
            }`,
			"main",
		)

		require.NoError(t, err)
	})
}

func TestEmit(t *testing.T) {

	t.Parallel()

	var eventEmitted bool

	vmConfig := vm.NewConfig(interpreter.NewInMemoryStorage(nil))
	vmConfig.OnEventEmitted = func(event *vm.CompositeValue, eventType *interpreter.CompositeStaticType) error {
		require.False(t, eventEmitted)
		eventEmitted = true

		assert.Equal(t,
			common.ScriptLocation{0x1}.TypeID(nil, "Inc"),
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
		vm.NewIntValue(1),
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
			vm.NewIntValue(2),
		)
		require.NoError(t, err)
		assert.Equal(t, vm.NewSomeValueNonCopying(vm.NewIntValue(2)), result)
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
			vm.NewIntValue(2),
		)
		require.NoError(t, err)
		assert.Equal(t, vm.NewIntValue(2), result)
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
			vm.BoolValue(true),
		)
		require.Error(t, err)
		assert.ErrorIs(
			t,
			err,
			vm.ForceCastTypeMismatchError{
				ExpectedType: interpreter.PrimitiveStaticTypeInt,
				ActualType:   interpreter.PrimitiveStaticTypeBool,
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
			vm.NewIntValue(2),
		)
		require.NoError(t, err)
		assert.Equal(t, vm.NewSomeValueNonCopying(vm.NewIntValue(2)), result)
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
			vm.BoolValue(true),
		)
		require.NoError(t, err)
		assert.Equal(t, vm.Nil, result)
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

		actual := test(t, vm.BoolValue(true))
		require.Equal(t, vm.NewIntValue(1), actual)
	})

	t.Run("false", func(t *testing.T) {
		t.Parallel()

		actual := test(t, vm.BoolValue(false))
		require.Equal(t, vm.NewIntValue(1), actual)
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

		actual := test(t, vm.BoolValue(true))
		require.Equal(t, vm.NewIntValue(3), actual)
	})

	t.Run("false", func(t *testing.T) {
		t.Parallel()

		actual := test(t, vm.BoolValue(false))
		require.Equal(t, vm.NewIntValue(4), actual)
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

			assert.Equal(t, vm.NewIntValue(5), result)
		})
	}

	// TODO:
	//for _, integerType := range common.Concat(
	//	sema.AllUnsignedIntegerTypes,
	//	sema.AllSignedIntegerTypes,
	//) {
	//	test(t, integerType)
	//}

	test(sema.IntType)
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

			assert.Equal(t,
				vm.NewUFix64Value(10*sema.Fix64Factor),
				result,
			)
		})
	}

	for _, fixedPointType := range sema.AllUnsignedFixedPointTypes {
		test(fixedPointType)
	}

	// TODO:
	//for _, fixedPointType := range sema.AllSignedFixedPointTypes {
	//	test(fixedPointType)
	//}
}
