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
	"encoding/hex"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"

	"github.com/onflow/cadence/activations"
	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/bbq"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/stdlib"
	. "github.com/onflow/cadence/test_utils/common_utils"
	"github.com/onflow/cadence/test_utils/runtime_utils"
	. "github.com/onflow/cadence/test_utils/sema_utils"

	"github.com/onflow/cadence/bbq/compiler"
	"github.com/onflow/cadence/bbq/vm"
)

// compares the traces between vm and interpreter
func TestTrace(t *testing.T) {
	t.Run("simple trace test", func(t *testing.T) {
		t.Parallel()

		code := `
			resource Bar {
                var id : Int

                init(_ id: Int) {
                    self.id = id
                }
            }

            fun test() {
                var i = 1
				var values = [0,1,2,3]
				while i < 3 {
					values[i] = values[i] + values[i-1]
					i = i + 1
				}
				var r <- create Bar(5)
				r.id = r.id + values[3]
				destroy r
            }
        `

		checker, err := ParseAndCheck(t, code)
		require.NoError(t, err)

		comp := compiler.NewCompiler(checker.Program, checker.Elaboration)
		program := comp.Compile()

		var vmLogs []string

		vmConfig := &vm.Config{
			Tracer: interpreter.Tracer{
				TracingEnabled: true,
				OnRecordTrace: func(executer interpreter.Traceable, operationName string, duration time.Duration, attrs []attribute.KeyValue) {
					vmLogs = append(vmLogs, fmt.Sprintf("%s: %v", operationName, attrs))
				},
			},
		}
		vmInstance := vm.NewVM(scriptLocation(), program, vmConfig)

		_, err = vmInstance.Invoke("test")
		require.NoError(t, err)
		require.Equal(t, 0, vmInstance.StackSize())

		var interLogs []string
		storage := interpreter.NewInMemoryStorage(nil)
		var uuid uint64 = 0
		inter, err := interpreter.NewInterpreter(
			interpreter.ProgramFromChecker(checker),
			TestLocation,
			&interpreter.Config{
				Tracer: interpreter.Tracer{
					OnRecordTrace: func(inter interpreter.Traceable,
						operationName string,
						duration time.Duration,
						attrs []attribute.KeyValue) {
						interLogs = append(interLogs, fmt.Sprintf("%s: %v", operationName, attrs))
					},
					TracingEnabled: true,
				},
				Storage: storage,
				UUIDHandler: func() (uint64, error) {
					uuid++
					return uuid, nil
				},
			},
		)
		require.NoError(t, err)

		err = inter.Interpret()
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		require.NoError(t, err)

		// compare traces
		AssertEqualWithDiff(t, vmLogs, interLogs)
	})

	t.Run("ft transfer", func(t *testing.T) {

		// VM VERSION

		var vmLogs []string

		// ---- Deploy FT Contract -----

		storage := interpreter.NewInMemoryStorage(nil)
		programs := map[common.Location]compiledProgram{}

		contractsAddress := common.MustBytesToAddress([]byte{0x1})
		senderAddress := common.MustBytesToAddress([]byte{0x2})
		receiverAddress := common.MustBytesToAddress([]byte{0x3})

		txLocation := runtime_utils.NewTransactionLocationGenerator()
		scriptLocation := runtime_utils.NewScriptLocationGenerator()

		ftLocation := common.NewAddressLocation(nil, contractsAddress, "FungibleToken")

		ftContractProgram := compileCode(t, realFungibleTokenContractInterface, ftLocation, programs)
		printProgram("FungibleToken", ftContractProgram)

		// ----- Deploy FlowToken Contract -----

		flowTokenLocation := common.NewAddressLocation(nil, contractsAddress, "FlowToken")

		flowTokenProgram := compileCode(t, realFlowContract, flowTokenLocation, programs)
		printProgram("FlowToken", flowTokenProgram)

		config := &vm.Config{
			Storage:        storage,
			AccountHandler: &testAccountHandler{},
			Tracer: interpreter.Tracer{
				TracingEnabled: true,
				OnRecordTrace: func(executer interpreter.Traceable, operationName string, duration time.Duration, attrs []attribute.KeyValue) {
					vmLogs = append(vmLogs,
						fmt.Sprintf("%s: %v",
							operationName,
							attrs,
						),
					)
				},
			},
		}

		flowTokenVM := vm.NewVM(
			flowTokenLocation,
			flowTokenProgram,
			config,
		)

		authAccount := vm.NewAuthAccountReferenceValue(config, contractsAddress)
		flowTokenContractValue, err := flowTokenVM.InitializeContract(authAccount)
		require.NoError(t, err)

		// ----- Run setup account transaction -----

		vmConfig := &vm.Config{
			Storage: storage,
			ImportHandler: func(location common.Location) *bbq.Program {
				imported, ok := programs[location]
				if !ok {
					return nil
				}
				return imported.Program
			},
			ContractValueHandler: func(_ *vm.Config, location common.Location) *vm.CompositeValue {
				switch location {
				case ftLocation:
					// interface
					return nil
				case flowTokenLocation:
					return flowTokenContractValue
				default:
					assert.FailNow(t, "invalid location")
					return nil
				}
			},

			AccountHandler: &testAccountHandler{},

			TypeLoader: func(location common.Location, typeID interpreter.TypeID) sema.CompositeKindedType {
				imported, ok := programs[location]
				if !ok {
					panic(fmt.Errorf("cannot find contract in location %s", location))
				}

				compositeType := imported.Elaboration.CompositeType(typeID)
				if compositeType != nil {
					return compositeType
				}

				return imported.Elaboration.InterfaceType(typeID)
			},

			Tracer: interpreter.Tracer{
				TracingEnabled: true,
				OnRecordTrace: func(executer interpreter.Traceable, operationName string, duration time.Duration, attrs []attribute.KeyValue) {
					vmLogs = append(vmLogs,
						fmt.Sprintf("%s: %v",
							operationName,
							attrs,
						),
					)
				},
			},
		}

		for _, address := range []common.Address{
			senderAddress,
			receiverAddress,
		} {
			program := compileCode(t, realSetupFlowTokenAccountTransaction, nil, programs)

			setupTxVM := vm.NewVM(txLocation(), program, vmConfig)

			authorizer := vm.NewAuthAccountReferenceValue(vmConfig, address)
			err = setupTxVM.ExecuteTransaction(nil, authorizer)
			require.NoError(t, err)
			require.Equal(t, 0, setupTxVM.StackSize())
		}

		// Mint FLOW to sender

		program := compileCode(t, realMintFlowTokenTransaction, nil, programs)
		printProgram("Setup FlowToken Tx", program)

		mintTxVM := vm.NewVM(txLocation(), program, vmConfig)

		total := int64(1000000)

		mintTxArgs := []vm.Value{
			vm.AddressValue(senderAddress),
			vm.NewIntValue(total),
		}

		mintTxAuthorizer := vm.NewAuthAccountReferenceValue(vmConfig, contractsAddress)
		err = mintTxVM.ExecuteTransaction(mintTxArgs, mintTxAuthorizer)
		require.NoError(t, err)
		require.Equal(t, 0, mintTxVM.StackSize())

		// ----- Run token transfer transaction -----

		tokenTransferTxProgram := compileCode(t, realFlowTokenTransferTransaction, nil, programs)
		printProgram("FT Transfer Tx", tokenTransferTxProgram)

		tokenTransferTxVM := vm.NewVM(txLocation(), tokenTransferTxProgram, vmConfig)

		transferAmount := int64(1)

		tokenTransferTxArgs := []vm.Value{
			vm.NewIntValue(transferAmount),
			vm.AddressValue(receiverAddress),
		}

		tokenTransferTxAuthorizer := vm.NewAuthAccountReferenceValue(vmConfig, senderAddress)
		err = tokenTransferTxVM.ExecuteTransaction(tokenTransferTxArgs, tokenTransferTxAuthorizer)
		require.NoError(t, err)
		require.Equal(t, 0, tokenTransferTxVM.StackSize())

		// Run validation scripts

		for _, address := range []common.Address{
			senderAddress,
			receiverAddress,
		} {
			program := compileCode(t, realFlowTokenBalanceScript, nil, programs)

			validationScriptVM := vm.NewVM(scriptLocation(), program, vmConfig)

			addressValue := vm.AddressValue(address)
			result, err := validationScriptVM.Invoke("main", addressValue)
			require.NoError(t, err)
			require.Equal(t, 0, validationScriptVM.StackSize())

			if address == senderAddress {
				assert.Equal(t, vm.NewIntValue(total-transferAmount), result)
			} else {
				assert.Equal(t, vm.NewIntValue(transferAmount), result)
			}
		}

		// INTERPRETER VERSION

		var interLogs []string

		// ---- Deploy FT Contract -----

		storage_int := interpreter.NewInMemoryStorage(nil)

		subInterpreters := map[common.Location]*interpreter.Interpreter{}
		codes := map[common.Location][]byte{
			ftLocation: []byte(realFungibleTokenContractInterface),
		}

		var signer interpreter.Value
		var flowTokenContractValue_int *interpreter.CompositeValue

		accountHandler := &testAccountHandler{
			getAccountContractCode: func(location common.AddressLocation) ([]byte, error) {
				code, ok := codes[location]
				if !ok {
					return nil, nil
					//	return nil, fmt.Errorf("cannot find code for %s", location)
				}

				return code, nil
			},
			updateAccountContractCode: func(location common.AddressLocation, code []byte) error {
				codes[location] = code
				return nil
			},
			contractUpdateRecorded: func(location common.AddressLocation) bool {
				return false
			},
			interpretContract: func(
				location common.AddressLocation,
				program *interpreter.Program,
				name string,
				invocation stdlib.DeployedContractConstructorInvocation,
			) (*interpreter.CompositeValue, error) {
				if location == flowTokenLocation {
					return flowTokenContractValue_int, nil
				}
				return nil, fmt.Errorf("cannot interpret contract %s", location)
			},
			temporarilyRecordCode: func(location common.AddressLocation, code []byte) {
				// do nothing
			},
			emitEvent: func(*interpreter.Interpreter, interpreter.LocationRange, *sema.CompositeType, []interpreter.Value) {
				// do nothing
			},
			recordContractUpdate: func(location common.AddressLocation, value *interpreter.CompositeValue) {
				// do nothing
			},
		}

		baseActivation := activations.NewActivation(nil, interpreter.BaseActivation)
		interpreter.Declare(baseActivation, stdlib.PanicFunction)
		interpreter.Declare(baseActivation, stdlib.NewGetAccountFunction(accountHandler))

		checkerConfig := &sema.Config{
			ImportHandler: func(checker *sema.Checker, location common.Location, importRange ast.Range) (sema.Import, error) {
				imported, ok := subInterpreters[location]
				if !ok {
					return nil, fmt.Errorf("cannot find contract in location %s", location)
				}

				return sema.ElaborationImport{
					Elaboration: imported.Program.Elaboration,
				}, nil
			},
			BaseValueActivationHandler: baseValueActivation,
			LocationHandler:            singleIdentifierLocationResolver(t),
		}

		interConfig := &interpreter.Config{
			Tracer: interpreter.Tracer{
				OnRecordTrace: func(inter interpreter.Traceable,
					operationName string,
					duration time.Duration,
					attrs []attribute.KeyValue) {
					interLogs = append(interLogs,
						fmt.Sprintf("%s: %v",
							operationName,
							attrs,
						),
					)
				},
				TracingEnabled: true,
			},
			Storage: storage_int,
			BaseActivationHandler: func(_ common.Location) *interpreter.VariableActivation {
				return baseActivation
			},
			ImportLocationHandler: func(inter *interpreter.Interpreter, location common.Location) interpreter.Import {
				imported, ok := subInterpreters[location]
				if !ok {
					panic(fmt.Errorf("cannot find contract in location %s", location))
				}

				return interpreter.InterpreterImport{
					Interpreter: imported,
				}
			},
			ContractValueHandler: func(
				inter *interpreter.Interpreter,
				compositeType *sema.CompositeType,
				constructorGenerator func(common.Address) *interpreter.HostFunctionValue,
				invocationRange ast.Range,
			) interpreter.ContractValue {

				constructor := constructorGenerator(common.ZeroAddress)

				value, err := inter.InvokeFunctionValue(
					constructor,
					[]interpreter.Value{signer},
					[]sema.Type{
						sema.FullyEntitledAccountReferenceType,
					},
					[]sema.Type{
						sema.FullyEntitledAccountReferenceType,
					},
					compositeType,
					ast.Range{},
				)
				if err != nil {
					panic(err)
				}

				flowTokenContractValue_int = value.(*interpreter.CompositeValue)
				return flowTokenContractValue_int
			},
			CapabilityBorrowHandler: func(
				inter *interpreter.Interpreter,
				locationRange interpreter.LocationRange,
				address interpreter.AddressValue,
				capabilityID interpreter.UInt64Value,
				wantedBorrowType *sema.ReferenceType,
				capabilityBorrowType *sema.ReferenceType,
			) interpreter.ReferenceValue {
				return stdlib.BorrowCapabilityController(
					inter,
					locationRange,
					address,
					capabilityID,
					wantedBorrowType,
					capabilityBorrowType,
					accountHandler,
				)
			},
		}

		accountHandler.parseAndCheckProgram =
			func(code []byte, location common.Location, getAndSetProgram bool) (*interpreter.Program, error) {
				if subInterpreter, ok := subInterpreters[location]; ok {
					return subInterpreter.Program, nil
				}

				inter, err := parseCheckAndInterpretWithOptions(
					t,
					string(code),
					location,
					ParseCheckAndInterpretOptions{
						Config:        interConfig,
						CheckerConfig: checkerConfig,
					},
				)

				if err != nil {
					return nil, err
				}

				subInterpreters[location] = inter

				return inter.Program, err
			}

		// ----- Parse and Check FungibleToken Contract interface -----

		inter, err := parseCheckAndInterpretWithOptions(
			t,
			realFungibleTokenContractInterface,
			ftLocation,
			ParseCheckAndInterpretOptions{
				Config:        interConfig,
				CheckerConfig: checkerConfig,
			},
		)
		require.NoError(t, err)
		subInterpreters[ftLocation] = inter

		// ----- Deploy FlowToken Contract -----

		tx := fmt.Sprintf(`
				transaction {
					prepare(signer: auth(Storage, Capabilities, Contracts) &Account) {
						signer.contracts.add(name: "FlowToken", code: "%s".decodeHex(), signer)
					}
				}`,
			hex.EncodeToString([]byte(realFlowContract)),
		)

		inter, err = parseCheckAndInterpretWithOptions(
			t,
			tx,
			txLocation(),
			ParseCheckAndInterpretOptions{
				Config:        interConfig,
				CheckerConfig: checkerConfig,
			},
		)
		require.NoError(t, err)

		signer = stdlib.NewAccountReferenceValue(
			inter,
			accountHandler,
			interpreter.AddressValue(contractsAddress),
			interpreter.FullyEntitledAccountAccess,
			interpreter.EmptyLocationRange,
		)

		err = inter.InvokeTransaction(0, signer)
		require.NoError(t, err)

		// ----- Run setup account transaction -----

		authorization := sema.NewEntitlementSetAccess(
			[]*sema.EntitlementType{
				sema.BorrowValueType,
				sema.IssueStorageCapabilityControllerType,
				sema.PublishCapabilityType,
				sema.SaveValueType,
			},
			sema.Conjunction,
		)

		for _, address := range []common.Address{
			senderAddress,
			receiverAddress,
		} {
			inter, err := parseCheckAndInterpretWithOptions(
				t,
				realSetupFlowTokenAccountTransaction,
				txLocation(),
				ParseCheckAndInterpretOptions{
					Config:        interConfig,
					CheckerConfig: checkerConfig,
				},
			)
			require.NoError(t, err)

			signer = stdlib.NewAccountReferenceValue(
				inter,
				accountHandler,
				interpreter.AddressValue(address),
				interpreter.ConvertSemaAccessToStaticAuthorization(nil, authorization),
				interpreter.EmptyLocationRange,
			)

			err = inter.InvokeTransaction(0, signer)
			require.NoError(t, err)
		}

		// Mint FLOW to sender

		inter, err = parseCheckAndInterpretWithOptions(
			t,
			realMintFlowTokenTransaction,
			txLocation(),
			ParseCheckAndInterpretOptions{
				Config:        interConfig,
				CheckerConfig: checkerConfig,
			},
		)
		require.NoError(t, err)

		signer = stdlib.NewAccountReferenceValue(
			inter,
			accountHandler,
			interpreter.AddressValue(contractsAddress),
			interpreter.ConvertSemaAccessToStaticAuthorization(nil, authorization),
			interpreter.EmptyLocationRange,
		)

		err = inter.InvokeTransaction(
			0,
			interpreter.AddressValue(senderAddress),
			interpreter.NewUnmeteredIntValueFromInt64(total),
			signer,
		)
		require.NoError(t, err)

		// ----- Run token transfer transaction -----

		inter, err = parseCheckAndInterpretWithOptions(
			t,
			realFlowTokenTransferTransaction,
			txLocation(),
			ParseCheckAndInterpretOptions{
				Config:        interConfig,
				CheckerConfig: checkerConfig,
			},
		)
		require.NoError(t, err)

		signer = stdlib.NewAccountReferenceValue(
			inter,
			accountHandler,
			interpreter.AddressValue(senderAddress),
			interpreter.ConvertSemaAccessToStaticAuthorization(nil, authorization),
			interpreter.EmptyLocationRange,
		)

		err = inter.InvokeTransaction(
			0,
			interpreter.NewUnmeteredIntValueFromInt64(transferAmount),
			interpreter.AddressValue(receiverAddress),
			signer,
		)
		require.NoError(t, err)

		// Run validation scripts

		for _, address := range []common.Address{
			senderAddress,
			receiverAddress,
		} {
			inter, err = parseCheckAndInterpretWithOptions(
				t,
				realFlowTokenBalanceScript,
				scriptLocation(),
				ParseCheckAndInterpretOptions{
					Config:        interConfig,
					CheckerConfig: checkerConfig,
				},
			)
			require.NoError(t, err)

			result, err := inter.Invoke(
				"main",
				interpreter.AddressValue(address),
			)
			require.NoError(t, err)

			if address == senderAddress {
				assert.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(total-transferAmount), result)
			} else {
				assert.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(transferAmount), result)
			}
		}

		// compare traces
		AssertEqualWithDiff(t, vmLogs, interLogs)
	})
}
