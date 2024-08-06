package test

import (
	"fmt"
	"github.com/onflow/cadence/runtime/bbq"
	"github.com/stretchr/testify/assert"
	"testing"

	"github.com/onflow/cadence/runtime/bbq/vm"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/stretchr/testify/require"
)

func TestResourceLossViaSelfRugPull(t *testing.T) {

	// ---- Deploy FT Contract -----

	storage := interpreter.NewInMemoryStorage(nil)

	programs := map[common.Location]compiledProgram{}

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

	barProgram := compileCode(t, contractCode, barLocation, programs)

	barVM := vm.NewVM(
		barProgram,
		&vm.Config{
			Storage:        storage,
			AccountHandler: &testAccountHandler{},
		},
	)

	barContractValue, err := barVM.InitializeContract()
	require.NoError(t, err)

	// --------- Execute Transaction ------------

	tx := fmt.Sprintf(`
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
                fun rugPullCallback(): Void{
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

	importHandler := func(location common.Location) *bbq.Program {
		switch location {
		case barLocation:
			return barProgram
		default:
			assert.FailNow(t, "invalid location")
			return nil
		}
	}

	program := compileCode(t, tx, nil, programs)
	printProgram(program)

	vmConfig := &vm.Config{
		Storage:       storage,
		ImportHandler: importHandler,
		ContractValueHandler: func(_ *vm.Config, location common.Location) *vm.CompositeValue {
			switch location {
			case barLocation:
				return barContractValue
			default:
				assert.FailNow(t, "invalid location")
				return nil
			}
		},
	}

	txVM := vm.NewVM(program, vmConfig)

	authorizer := vm.NewAuthAccountReferenceValue(authorizerAddress)
	err = txVM.ExecuteTransaction(nil, authorizer)
	require.NoError(t, err)
	require.Equal(t, 0, txVM.StackSize())
}
