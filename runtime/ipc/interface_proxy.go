package ipc

import (
	"time"

	"github.com/opentracing/opentracing-go"

	"github.com/onflow/atree"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/ipc/bridge"
)

var _ runtime.Interface = &ProxyInterface{}

// ProxyInterface converts the `runtime.Interface` Go method calls to IPC calls over the socket.
// Results are again converted back from IPC serialized format to corresponding Go structs.
type ProxyInterface struct {
}

func NewProxyInterface() *ProxyInterface {
	return &ProxyInterface{}
}

func (p *ProxyInterface) ResolveLocation(identifiers []runtime.Identifier, location runtime.Location) ([]runtime.ResolvedLocation, error) {
	conn := bridge.NewInterfaceConnection()

	loc, err := bridge.NewLocation(location)
	if err != nil {
		return nil, err
	}

	idents := bridge.NewIdentifiers(identifiers)

	locationParam := bridge.AsAny(loc)
	identifiersParam := bridge.AsAny(idents)

	request := bridge.NewRequestMessage(InterfaceMethodResolveLocation, identifiersParam, locationParam)
	bridge.WriteMessage(conn, request)

	resp, err := bridge.ReadResponse(conn)
	if err != nil {
		return nil, err
	}

	return bridge.ToRuntimeResolvedLocationsFromAny(resp.Value), nil
}

func (p *ProxyInterface) GetCode(location runtime.Location) ([]byte, error) {
	conn := bridge.NewInterfaceConnection()

	loc, err := bridge.NewLocation(location)
	if err != nil {
		return nil, err
	}

	locationParam := bridge.AsAny(loc)

	request := bridge.NewRequestMessage(InterfaceMethodGetCode, locationParam)

	bridge.WriteMessage(conn, request)

	response, err := bridge.ReadResponse(conn)
	if err != nil {
		return nil, err
	}

	code := bridge.ToRuntimeBytes(response.Value)
	return code, nil
}

func (p *ProxyInterface) GetProgram(location runtime.Location) (*interpreter.Program, error) {
	conn := bridge.NewInterfaceConnection()

	loc, err := bridge.NewLocation(location)
	if err != nil {
		return nil, err
	}

	locationParam := bridge.AsAny(loc)

	request := bridge.NewRequestMessage(InterfaceMethodGetProgram, locationParam)

	bridge.WriteMessage(conn, request)

	_, err = bridge.ReadResponse(conn)
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func (p *ProxyInterface) SetProgram(location runtime.Location, program *interpreter.Program) error {
	// TODO: implement
	return nil
}

func (p *ProxyInterface) GetValue(owner, key []byte) (value []byte, err error) {
	panic("implement me")
}

func (p *ProxyInterface) SetValue(owner, key, value []byte) (err error) {
	panic("implement me")
}

func (p *ProxyInterface) ValueExists(owner, key []byte) (exists bool, err error) {
	panic("implement me")
}

func (p *ProxyInterface) AllocateStorageIndex(owner []byte) (atree.StorageIndex, error) {
	panic("implement me")
}

func (p *ProxyInterface) CreateAccount(payer runtime.Address) (address runtime.Address, err error) {
	panic("implement me")
}

func (p *ProxyInterface) AddEncodedAccountKey(address runtime.Address, publicKey []byte) error {
	panic("implement me")
}

func (p *ProxyInterface) RevokeEncodedAccountKey(address runtime.Address, index int) (publicKey []byte, err error) {
	panic("implement me")
}

func (p *ProxyInterface) AddAccountKey(address runtime.Address, publicKey *runtime.PublicKey, hashAlgo runtime.HashAlgorithm, weight int) (*runtime.AccountKey, error) {
	panic("implement me")
}

func (p *ProxyInterface) GetAccountKey(address runtime.Address, index int) (*runtime.AccountKey, error) {
	panic("implement me")
}

func (p *ProxyInterface) RevokeAccountKey(address runtime.Address, index int) (*runtime.AccountKey, error) {
	panic("implement me")
}

func (p *ProxyInterface) UpdateAccountContractCode(address runtime.Address, name string, code []byte) (err error) {
	panic("implement me")
}

func (p *ProxyInterface) GetAccountContractCode(address runtime.Address, name string) ([]byte, error) {
	conn := bridge.NewInterfaceConnection()

	addressBytes := bridge.NewBytes(address[:])
	addressParam := bridge.AsAny(addressBytes)

	nameStr := bridge.NewString(name)
	nameParam := bridge.AsAny(nameStr)

	request := bridge.NewRequestMessage(InterfaceMethodGetAccountContractCode, addressParam, nameParam)

	bridge.WriteMessage(conn, request)

	response, err := bridge.ReadResponse(conn)
	if err != nil {
		return nil, err
	}

	code := bridge.ToRuntimeBytes(response.Value)

	return code, nil
}

func (p *ProxyInterface) RemoveAccountContractCode(address runtime.Address, name string) (err error) {
	panic("implement me")
}

func (p *ProxyInterface) GetSigningAccounts() ([]runtime.Address, error) {
	panic("implement me")
}

func (p *ProxyInterface) ProgramLog(s string) error {
	conn := bridge.NewInterfaceConnection()

	str := bridge.NewString(s)
	stringParam := bridge.AsAny(str)

	request := bridge.NewRequestMessage(InterfaceMethodProgramLog, stringParam)

	bridge.WriteMessage(conn, request)

	_, err := bridge.ReadResponse(conn)
	return err
}

func (p *ProxyInterface) EmitEvent(event cadence.Event) error {
	panic("implement me")
}

func (p *ProxyInterface) GenerateUUID() (uint64, error) {
	panic("implement me")
}

func (p *ProxyInterface) GetComputationLimit() uint64 {
	// TODO: implement
	return 1000
}

func (p *ProxyInterface) SetComputationUsed(used uint64) error {
	// TODO: implement
	return nil
}

func (p *ProxyInterface) DecodeArgument(argument []byte, argumentType cadence.Type) (cadence.Value, error) {
	panic("implement me")
}

func (p *ProxyInterface) GetCurrentBlockHeight() (uint64, error) {
	panic("implement me")
}

func (p *ProxyInterface) GetBlockAtHeight(height uint64) (block runtime.Block, exists bool, err error) {
	panic("implement me")
}

func (p *ProxyInterface) UnsafeRandom() (uint64, error) {
	panic("implement me")
}

func (p *ProxyInterface) VerifySignature(signature []byte, tag string, signedData []byte, publicKey []byte, signatureAlgorithm runtime.SignatureAlgorithm, hashAlgorithm runtime.HashAlgorithm) (bool, error) {
	panic("implement me")
}

func (p *ProxyInterface) Hash(data []byte, tag string, hashAlgorithm runtime.HashAlgorithm) ([]byte, error) {
	panic("implement me")
}

func (p *ProxyInterface) GetAccountBalance(address common.Address) (value uint64, err error) {
	panic("implement me")
}

func (p *ProxyInterface) GetAccountAvailableBalance(address common.Address) (value uint64, err error) {
	panic("implement me")
}

func (p *ProxyInterface) GetStorageUsed(address runtime.Address) (value uint64, err error) {
	panic("implement me")
}

func (p *ProxyInterface) GetStorageCapacity(address runtime.Address) (value uint64, err error) {
	panic("implement me")
}

func (p *ProxyInterface) ImplementationDebugLog(message string) error {
	panic("implement me")
}

func (p *ProxyInterface) ValidatePublicKey(key *runtime.PublicKey) (bool, error) {
	panic("implement me")
}

func (p *ProxyInterface) GetAccountContractNames(address runtime.Address) ([]string, error) {
	panic("implement me")
}

func (p *ProxyInterface) RecordTrace(operation string, location common.Location, duration time.Duration, logs []opentracing.LogRecord) {
	panic("implement me")
}

func (p *ProxyInterface) BLSVerifyPOP(pk *runtime.PublicKey, s []byte) (bool, error) {
	panic("implement me")
}

func (p *ProxyInterface) AggregateBLSSignatures(sigs [][]byte) ([]byte, error) {
	panic("implement me")
}

func (p *ProxyInterface) AggregateBLSPublicKeys(keys []*runtime.PublicKey) (*runtime.PublicKey, error) {
	panic("implement me")
}

func (p *ProxyInterface) ResourceOwnerChanged(resource *interpreter.CompositeValue, oldOwner common.Address, newOwner common.Address) {
	panic("implement me")
}
