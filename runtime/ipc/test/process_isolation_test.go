package test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/opentracing/opentracing-go"

	"github.com/onflow/atree"
	"github.com/onflow/cadence"
	"github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/ipc"
)

// TODO: Move under tests/ and delete redundant *testRuntimeInterface

var proxyRuntime = func() *ipc.ProxyRuntime {
	interfaceImpl := &testRuntimeInterface{}
	return ipc.NewProxyRuntime(interfaceImpl)
}()

func TestExecutingScript(t *testing.T) {
	for i := 0; i < 10; i++ {
		start := time.Now()
		_, err := proxyRuntime.ExecuteScript(
			runtime.Script{
				Source: []byte(`
               pub fun main(): Int {
                 log("hello")
                 return 4 + 8
               }
            `),
			},
			runtime.Context{},
		)

		fmt.Println(time.Since(start))

		assert.NoError(t, err)
	}
}

type testRuntimeInterface struct{}

func (t *testRuntimeInterface) ResolveLocation(identifiers []runtime.Identifier, location runtime.Location) ([]runtime.ResolvedLocation, error) {
	return []runtime.ResolvedLocation{}, nil
}

func (t *testRuntimeInterface) GetCode(location runtime.Location) ([]byte, error) {
	return []byte(`
        pub contract Foo {
            pub fun Add(_ a: Int, _ b: Int): Int {
                return a + b
            }
        }
    `), nil
}

func (t *testRuntimeInterface) GetProgram(location runtime.Location) (*interpreter.Program, error) {
	return nil, nil
}

func (t *testRuntimeInterface) SetProgram(location runtime.Location, program *interpreter.Program) error {
	panic("implement me")
}

func (t *testRuntimeInterface) GetValue(owner, key []byte) (value []byte, err error) {
	panic("implement me")
}

func (t *testRuntimeInterface) SetValue(owner, key, value []byte) (err error) {
	panic("implement me")
}

func (t *testRuntimeInterface) ValueExists(owner, key []byte) (exists bool, err error) {
	panic("implement me")
}

func (t *testRuntimeInterface) AllocateStorageIndex(owner []byte) (atree.StorageIndex, error) {
	panic("implement me")
}

func (t *testRuntimeInterface) CreateAccount(payer runtime.Address) (address runtime.Address, err error) {
	panic("implement me")
}

func (t *testRuntimeInterface) AddEncodedAccountKey(address runtime.Address, publicKey []byte) error {
	panic("implement me")
}

func (t *testRuntimeInterface) RevokeEncodedAccountKey(address runtime.Address, index int) (publicKey []byte, err error) {
	panic("implement me")
}

func (t *testRuntimeInterface) AddAccountKey(address runtime.Address, publicKey *runtime.PublicKey, hashAlgo runtime.HashAlgorithm, weight int) (*runtime.AccountKey, error) {
	panic("implement me")
}

func (t *testRuntimeInterface) GetAccountKey(address runtime.Address, index int) (*runtime.AccountKey, error) {
	panic("implement me")
}

func (t *testRuntimeInterface) RevokeAccountKey(address runtime.Address, index int) (*runtime.AccountKey, error) {
	panic("implement me")
}

func (t *testRuntimeInterface) UpdateAccountContractCode(address runtime.Address, name string, code []byte) (err error) {
	panic("implement me")
}

func (t *testRuntimeInterface) GetAccountContractCode(address runtime.Address, name string) (code []byte, err error) {
	panic("implement me")
}

func (t *testRuntimeInterface) RemoveAccountContractCode(address runtime.Address, name string) (err error) {
	panic("implement me")
}

func (t *testRuntimeInterface) GetSigningAccounts() ([]runtime.Address, error) {
	panic("implement me")
}

func (t *testRuntimeInterface) ProgramLog(s string) error {
	// Test executing a script/transaction within a script/transaction
	_, err := proxyRuntime.ExecuteScript(
		runtime.Script{
			Source: []byte(`
               pub fun main(): Int {
                 return 3 + 2
               }
            `),
		},
		runtime.Context{},
	)

	if err != nil {
		panic(err)
	}

	return nil
}

func (t *testRuntimeInterface) EmitEvent(event cadence.Event) error {
	panic("implement me")
}

func (t *testRuntimeInterface) GenerateUUID() (uint64, error) {
	panic("implement me")
}

func (t *testRuntimeInterface) GetComputationLimit() uint64 {
	panic("implement me")
}

func (t *testRuntimeInterface) SetComputationUsed(used uint64) error {
	panic("implement me")
}

func (t *testRuntimeInterface) DecodeArgument(argument []byte, argumentType cadence.Type) (cadence.Value, error) {
	panic("implement me")
}

func (t *testRuntimeInterface) GetCurrentBlockHeight() (uint64, error) {
	panic("implement me")
}

func (t *testRuntimeInterface) GetBlockAtHeight(height uint64) (block runtime.Block, exists bool, err error) {
	panic("implement me")
}

func (t *testRuntimeInterface) UnsafeRandom() (uint64, error) {
	panic("implement me")
}

func (t *testRuntimeInterface) VerifySignature(signature []byte, tag string, signedData []byte, publicKey []byte, signatureAlgorithm runtime.SignatureAlgorithm, hashAlgorithm runtime.HashAlgorithm) (bool, error) {
	panic("implement me")
}

func (t *testRuntimeInterface) Hash(data []byte, tag string, hashAlgorithm runtime.HashAlgorithm) ([]byte, error) {
	panic("implement me")
}

func (t *testRuntimeInterface) GetAccountBalance(address common.Address) (value uint64, err error) {
	panic("implement me")
}

func (t *testRuntimeInterface) GetAccountAvailableBalance(address common.Address) (value uint64, err error) {
	panic("implement me")
}

func (t *testRuntimeInterface) GetStorageUsed(address runtime.Address) (value uint64, err error) {
	panic("implement me")
}

func (t *testRuntimeInterface) GetStorageCapacity(address runtime.Address) (value uint64, err error) {
	panic("implement me")
}

func (t *testRuntimeInterface) ImplementationDebugLog(message string) error {
	panic("implement me")
}

func (t *testRuntimeInterface) ValidatePublicKey(key *runtime.PublicKey) (bool, error) {
	panic("implement me")
}

func (t *testRuntimeInterface) GetAccountContractNames(address runtime.Address) ([]string, error) {
	panic("implement me")
}

func (t *testRuntimeInterface) RecordTrace(operation string, location common.Location, duration time.Duration, logs []opentracing.LogRecord) {
	panic("implement me")
}

func (t *testRuntimeInterface) BLSVerifyPOP(pk *runtime.PublicKey, s []byte) (bool, error) {
	panic("implement me")
}

func (t *testRuntimeInterface) AggregateBLSSignatures(sigs [][]byte) ([]byte, error) {
	panic("implement me")
}

func (t *testRuntimeInterface) AggregateBLSPublicKeys(keys []*runtime.PublicKey) (*runtime.PublicKey, error) {
	panic("implement me")
}

func (t *testRuntimeInterface) ResourceOwnerChanged(resource *interpreter.CompositeValue, oldOwner common.Address, newOwner common.Address) {
	panic("implement me")
}

var _ runtime.Interface = &testRuntimeInterface{}
