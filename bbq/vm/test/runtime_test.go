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

	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/test_utils/runtime_utils"

	"github.com/onflow/cadence/bbq"
	"github.com/onflow/cadence/bbq/vm"
)

func TestResourceLossViaSelfRugPull(t *testing.T) {

	// TODO:
	t.SkipNow()

	// ---- Deploy FT Contract -----

	storage := interpreter.NewInMemoryStorage(nil)

	programs := map[common.Location]*compiledProgram{}

	contractsAddress := common.MustBytesToAddress([]byte{0x1})
	authorizerAddress := common.MustBytesToAddress([]byte{0x2})

	// ----- Deploy Contract -----

	contractCode := `
        access(all) contract Bar {

            access(all) resource Vault {

                // Balance of a user's Vault
                // we use unsigned fixed point numbers for balances
                // because they can represent decimals and do not allow negative values
                access(all) var balance: Int

                init(balance: Int) {
                    self.balance = balance
                }

                access(all) fun withdraw(amount: Int): @Vault {
                    self.balance = self.balance - amount
                    return <-create Vault(balance: amount)
                }

                access(all) fun deposit(from: @Vault) {
                    self.balance = self.balance + from.balance
                    destroy from
                }
            }

            access(all) fun createEmptyVault(): @Bar.Vault {
                return <- create Bar.Vault(balance: 0)
            }

            access(all) fun createVault(balance: Int): @Bar.Vault {
                return <- create Bar.Vault(balance: balance)
            }
        }
    `
	barLocation := common.NewAddressLocation(nil, contractsAddress, "Bar")

	barProgram := parseCheckAndCompile(t, contractCode, barLocation, programs)

	config := vm.NewConfig(storage)

	barVM := vm.NewVM(
		barLocation,
		barProgram,
		config,
	)

	barContractValue, err := barVM.InitializeContract()
	require.NoError(t, err)

	// --------- Execute Transaction ------------

	fooContract := fmt.Sprintf(`
        import Bar from %[1]s

        access(all) contract Foo {
            access(all) var rCopy1: @R?
            init() {
                self.rCopy1 <- nil
                var r <- Bar.createVault(balance: 1337)
                self.loser(<- r)
            }
            access(all) resource R {
                access(all) var optional: @[Bar.Vault]?

                init() {
                    self.optional <- []
                }

                access(all) fun rugpullAndAssign(_ callback: fun(): Void, _ victim: @Bar.Vault) {
                    callback()
                    // "self" has now been invalidated and accessing "a" for reading would
                    // trigger a "not initialized" error. However, force-assigning to it succeeds
                    // and leaves the victim object hanging from an invalidated resource
                    self.optional <-! [<- victim]
                }
            }

            access(all) fun loser(_ victim: @Bar.Vault): Void{
                var array: @[R] <- [<- create R()]
                let arrRef = &array as auth(Remove) &[R]
                fun rugPullCallback(): Void {
                    // Here we move the R resource from the array to a contract field
                    // invalidating the "self" during the execution of rugpullAndAssign
                    Foo.rCopy1 <-! arrRef.removeLast()
                }
                array[0].rugpullAndAssign(rugPullCallback, <- victim)
                destroy array

                var y: @R? <- nil
                self.rCopy1 <-> y
                destroy y
            }

        }`,
		contractsAddress.HexWithPrefix(),
	)

	importHandler := func(location common.Location) *bbq.InstructionProgram {
		switch location {
		case barLocation:
			return barProgram
		default:
			assert.FailNow(t, "invalid location")
			return nil
		}
	}

	fooLocation := common.NewAddressLocation(nil, contractsAddress, "Foo")
	program := parseCheckAndCompile(t, fooContract, fooLocation, programs)

	vmConfig := vm.NewConfig(storage)
	vmConfig.ImportHandler = importHandler
	vmConfig.ContractValueHandler = func(_ *vm.Config, location common.Location) *vm.CompositeValue {
		switch location {
		case barLocation:
			return barContractValue
		default:
			assert.FailNow(t, "invalid location")
			return nil
		}
	}

	txLocation := runtime_utils.NewTransactionLocationGenerator()

	txVM := vm.NewVM(txLocation(), program, vmConfig)

	authorizer := vm.NewAuthAccountReferenceValue(vmConfig, authorizerAddress)
	err = txVM.ExecuteTransaction(nil, authorizer)
	require.NoError(t, err)
	require.Equal(t, 0, txVM.StackSize())
}

func TestInterpretResourceReferenceInvalidationOnMove(t *testing.T) {

	t.Parallel()

	//t.Run("stack to account", func(t *testing.T) {
	//
	//	t.Parallel()
	//
	//	code := `
	//        resource R {
	//            access(all) var id: Int
	//
	//            access(all) fun setID(_ id: Int) {
	//                self.id = id
	//            }
	//
	//            init() {
	//                self.id = 1
	//            }
	//        }
	//
	//        fun getRef(_ ref: &R): &R {
	//            return ref
	//        }
	//
	//        fun test() {
	//            let r <-create R()
	//            let ref = getRef(&r as &R)
	//
	//            // Move the resource into the account
	//            account.storage.save(<-r, to: /storage/r)
	//
	//            // Update the reference
	//            ref.setID(2)
	//        }`
	//
	//	_, err := compileAndInvoke(t, code, "test")
	//	require.Error(t, err)
	//	require.ErrorAs(t, err, &interpreter.InvalidatedResourceReferenceError{})
	//})
	//
	//t.Run("stack to account readonly", func(t *testing.T) {
	//
	//	t.Parallel()
	//
	//	address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})
	//
	//	inter, _ := testAccountWithErrorHandler(t, address, true, nil, `
	//        resource R {
	//            access(all) var id: Int
	//
	//            init() {
	//                self.id = 1
	//            }
	//        }
	//
	//        fun test() {
	//            let r <-create R()
	//            let ref = &r as &R
	//
	//            // Move the resource into the account
	//            account.storage.save(<-r, to: /storage/r)
	//
	//            // 'Read' a field from the reference
	//            let id = ref.id
	//        }`, sema.Config{}, errorHandler(t))
	//
	//	_, err := inter.Invoke("test")
	//	RequireError(t, err)
	//	require.ErrorAs(t, err, &interpreter.InvalidatedResourceReferenceError{})
	//})
	//
	//t.Run("account to stack", func(t *testing.T) {
	//
	//	t.Parallel()
	//
	//	inter := parseCheckAndInterpret(t, `
	//        resource R {
	//            access(all) var id: Int
	//
	//            access(all) fun setID(_ id: Int) {
	//                self.id = id
	//            }
	//
	//            init() {
	//                self.id = 1
	//            }
	//        }
	//
	//        fun test(target: auth(Mutate) &[R]) {
	//            target.append(<- create R())
	//
	//            // Take reference while in the account
	//            let ref = target[0]
	//
	//            // Move the resource out of the account onto the stack
	//            let movedR <- target.remove(at: 0)
	//
	//            // Update the reference
	//            ref.setID(2)
	//
	//            destroy movedR
	//        }
	//    `)
	//
	//	address := common.Address{0x1}
	//
	//	rType := checker.RequireGlobalType(t, inter.Program.Elaboration, "R").(*sema.CompositeType)
	//
	//	array := interpreter.NewArrayValue(
	//		inter,
	//		interpreter.EmptyLocationRange,
	//		&interpreter.VariableSizedStaticType{
	//			Type: interpreter.ConvertSemaToStaticType(nil, rType),
	//		},
	//		address,
	//	)
	//
	//	arrayRef := interpreter.NewUnmeteredEphemeralReferenceValue(
	//		inter,
	//		interpreter.NewEntitlementSetAuthorization(
	//			nil,
	//			func() []common.TypeID { return []common.TypeID{"Mutate"} },
	//			1,
	//			sema.Conjunction,
	//		),
	//		array,
	//		&sema.VariableSizedType{
	//			Type: rType,
	//		},
	//		interpreter.EmptyLocationRange,
	//	)
	//
	//	_, err := inter.Invoke("test", arrayRef)
	//	RequireError(t, err)
	//	require.ErrorAs(t, err, &interpreter.InvalidatedResourceReferenceError{})
	//})

	t.Run("stack to stack", func(t *testing.T) {

		t.Parallel()

		code := `
	       resource R {
	           access(all) var id: Int

	           access(all) fun setID(_ id: Int) {
	               self.id = id
	           }

	           init() {
	               self.id = 1
	           }
	       }

	       fun test() {
	           let r1 <- create R()
	           let ref = reference(&r1 as &R)

	           // Move the resource onto the same stack
	           let r2 <- r1

	           // Update the reference
	           ref.setID(2)

	           destroy r2
	       }

	       fun reference(_ ref: &R): &R {
	           return ref
	       }`

		_, err := compileAndInvoke(t, code, "test")
		require.Error(t, err)
		require.ErrorAs(t, err, &interpreter.InvalidatedResourceReferenceError{})
	})

	//t.Run("one account to another account", func(t *testing.T) {
	//
	//	t.Parallel()
	//
	//	inter := parseCheckAndInterpret(t, `
	//        resource R {
	//            access(all) var id: Int
	//
	//            access(all) fun setID(_ id: Int) {
	//                self.id = id
	//            }
	//
	//            init() {
	//                self.id = 1
	//            }
	//        }
	//
	//        fun test(target1: auth(Mutate) &[R], target2: auth(Mutate) &[R]) {
	//            target1.append(<- create R())
	//
	//            // Take reference while in the account_1
	//            let ref = target1[0]
	//
	//            // Move the resource out of the account_1 into the account_2
	//            target2.append(<- target1.remove(at: 0))
	//
	//            // Update the reference
	//            ref.setID(2)
	//        }
	//    `)
	//
	//	rType := checker.RequireGlobalType(t, inter.Program.Elaboration, "R").(*sema.CompositeType)
	//
	//	// Resource array in account 0x01
	//
	//	array1 := interpreter.NewArrayValue(
	//		inter,
	//		interpreter.EmptyLocationRange,
	//		&interpreter.VariableSizedStaticType{
	//			Type: interpreter.ConvertSemaToStaticType(nil, rType),
	//		},
	//		common.Address{0x1},
	//	)
	//
	//	arrayRef1 := interpreter.NewUnmeteredEphemeralReferenceValue(
	//		inter,
	//		interpreter.NewEntitlementSetAuthorization(
	//			nil,
	//			func() []common.TypeID { return []common.TypeID{"Mutate"} },
	//			1,
	//			sema.Conjunction,
	//		),
	//		array1,
	//		&sema.VariableSizedType{
	//			Type: rType,
	//		},
	//		interpreter.EmptyLocationRange,
	//	)
	//
	//	// Resource array in account 0x02
	//
	//	array2 := interpreter.NewArrayValue(
	//		inter,
	//		interpreter.EmptyLocationRange,
	//		&interpreter.VariableSizedStaticType{
	//			Type: interpreter.ConvertSemaToStaticType(nil, rType),
	//		},
	//		common.Address{0x2},
	//	)
	//
	//	arrayRef2 := interpreter.NewUnmeteredEphemeralReferenceValue(
	//		inter,
	//		interpreter.NewEntitlementSetAuthorization(
	//			nil,
	//			func() []common.TypeID { return []common.TypeID{"Mutate"} },
	//			1,
	//			sema.Conjunction,
	//		),
	//		array2,
	//		&sema.VariableSizedType{
	//			Type: rType,
	//		},
	//		interpreter.EmptyLocationRange,
	//	)
	//
	//	_, err := inter.Invoke("test", arrayRef1, arrayRef2)
	//	RequireError(t, err)
	//	require.ErrorAs(t, err, &interpreter.InvalidatedResourceReferenceError{})
	//})
}
