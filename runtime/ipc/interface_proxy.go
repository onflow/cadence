package ipc

import (
	"fmt"
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

	// TODO
	return nil, nil
}

func (p *ProxyInterface) SetProgram(location runtime.Location, program *interpreter.Program) error {
	// TODO: implement
	return nil
}

func (p *ProxyInterface) GetValue(owner, key []byte) (value []byte, err error) {
	conn := bridge.NewInterfaceConnection()

	pbOwner := bridge.NewBytes(owner)
	pbKey := bridge.NewBytes(key)

	ownerParam := bridge.AsAny(pbOwner)
	keyParam := bridge.AsAny(pbKey)

	request := bridge.NewRequestMessage(InterfaceMethodGetValue, ownerParam, keyParam)

	bridge.WriteMessage(conn, request)

	resp, err := bridge.ReadResponse(conn)
	if err != nil {
		return nil, err
	}

	return bridge.ToRuntimeBytes(resp.Value), nil
}

func (p *ProxyInterface) SetValue(owner, key, value []byte) (err error) {
	conn := bridge.NewInterfaceConnection()

	pbOwner := bridge.NewBytes(owner)
	pbKey := bridge.NewBytes(key)
	pbValue := bridge.NewBytes(value)

	ownerParam := bridge.AsAny(pbOwner)
	keyParam := bridge.AsAny(pbKey)
	valueParam := bridge.AsAny(pbValue)

	request := bridge.NewRequestMessage(InterfaceMethodSetValue, ownerParam, keyParam, valueParam)

	bridge.WriteMessage(conn, request)

	_, err = bridge.ReadResponse(conn)
	if err != nil {
		return err
	}

	return nil
}

func (p *ProxyInterface) ValueExists(owner, key []byte) (exists bool, err error) {
	panic(UnimplementedError())
}

func (p *ProxyInterface) AllocateStorageIndex(owner []byte) (atree.StorageIndex, error) {
	conn := bridge.NewInterfaceConnection()

	pbOwner := bridge.NewBytes(owner)
	ownerParam := bridge.AsAny(pbOwner)

	request := bridge.NewRequestMessage(InterfaceMethodAllocateStorageIndex, ownerParam)

	bridge.WriteMessage(conn, request)

	resp, err := bridge.ReadResponse(conn)
	if err != nil {
		return [8]byte{}, err
	}

	indexBytes := bridge.ToRuntimeBytes(resp.Value)

	var storageIndex atree.StorageIndex
	copy(storageIndex[:], indexBytes[:])

	return storageIndex, nil
}

func (p *ProxyInterface) CreateAccount(payer runtime.Address) (address runtime.Address, err error) {
	panic(UnimplementedError())
}

func (p *ProxyInterface) AddEncodedAccountKey(address runtime.Address, publicKey []byte) error {
	panic(UnimplementedError())
}

func (p *ProxyInterface) RevokeEncodedAccountKey(address runtime.Address, index int) (publicKey []byte, err error) {
	panic(UnimplementedError())
}

func (p *ProxyInterface) AddAccountKey(address runtime.Address, publicKey *runtime.PublicKey, hashAlgo runtime.HashAlgorithm, weight int) (*runtime.AccountKey, error) {
	panic(UnimplementedError())
}

func (p *ProxyInterface) GetAccountKey(address runtime.Address, index int) (*runtime.AccountKey, error) {
	panic(UnimplementedError())
}

func (p *ProxyInterface) RevokeAccountKey(address runtime.Address, index int) (*runtime.AccountKey, error) {
	panic(UnimplementedError())
}

func (p *ProxyInterface) UpdateAccountContractCode(address runtime.Address, name string, code []byte) (err error) {
	conn := bridge.NewInterfaceConnection()

	addressBytes := bridge.NewBytes(address[:])
	addressParam := bridge.AsAny(addressBytes)

	nameStr := bridge.NewString(name)
	nameParam := bridge.AsAny(nameStr)

	codeBytes := bridge.NewBytes(code)
	codeParam := bridge.AsAny(codeBytes)

	request := bridge.NewRequestMessage(InterfaceMethodUpdateAccountContractCode, addressParam, nameParam, codeParam)

	bridge.WriteMessage(conn, request)

	_, err = bridge.ReadResponse(conn)
	if err != nil {
		return err
	}

	return nil
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
	panic(UnimplementedError())
}

func (p *ProxyInterface) GetSigningAccounts() ([]runtime.Address, error) {
	// TODO
	return []runtime.Address{
		common.MustBytesToAddress([]byte{0, 0, 0, 0, 0, 0, 1}),
	}, nil
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
	// TODO: implement
	return nil
}

func (p *ProxyInterface) GenerateUUID() (uint64, error) {
	panic(UnimplementedError())
}

func UnimplementedError() error {
	return fmt.Errorf("implement me")
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
	panic(UnimplementedError())
}

func (p *ProxyInterface) GetCurrentBlockHeight() (uint64, error) {
	panic(UnimplementedError())
}

func (p *ProxyInterface) GetBlockAtHeight(height uint64) (block runtime.Block, exists bool, err error) {
	panic(UnimplementedError())
}

func (p *ProxyInterface) UnsafeRandom() (uint64, error) {
	panic(UnimplementedError())
}

func (p *ProxyInterface) VerifySignature(signature []byte, tag string, signedData []byte, publicKey []byte, signatureAlgorithm runtime.SignatureAlgorithm, hashAlgorithm runtime.HashAlgorithm) (bool, error) {
	panic(UnimplementedError())
}

func (p *ProxyInterface) Hash(data []byte, tag string, hashAlgorithm runtime.HashAlgorithm) ([]byte, error) {
	panic(UnimplementedError())
}

func (p *ProxyInterface) GetAccountBalance(address common.Address) (value uint64, err error) {
	panic(UnimplementedError())
}

func (p *ProxyInterface) GetAccountAvailableBalance(address common.Address) (value uint64, err error) {
	panic(UnimplementedError())
}

func (p *ProxyInterface) GetStorageUsed(address runtime.Address) (value uint64, err error) {
	panic(UnimplementedError())
}

func (p *ProxyInterface) GetStorageCapacity(address runtime.Address) (value uint64, err error) {
	panic(UnimplementedError())
}

func (p *ProxyInterface) ImplementationDebugLog(message string) error {
	panic(UnimplementedError())
}

func (p *ProxyInterface) ValidatePublicKey(key *runtime.PublicKey) (bool, error) {
	panic(UnimplementedError())
}

func (p *ProxyInterface) GetAccountContractNames(address runtime.Address) ([]string, error) {
	panic(UnimplementedError())
}

func (p *ProxyInterface) RecordTrace(operation string, location common.Location, duration time.Duration, logs []opentracing.LogRecord) {
	panic(UnimplementedError())
}

func (p *ProxyInterface) BLSVerifyPOP(pk *runtime.PublicKey, s []byte) (bool, error) {
	panic(UnimplementedError())
}

func (p *ProxyInterface) AggregateBLSSignatures(sigs [][]byte) ([]byte, error) {
	panic(UnimplementedError())
}

func (p *ProxyInterface) AggregateBLSPublicKeys(keys []*runtime.PublicKey) (*runtime.PublicKey, error) {
	panic(UnimplementedError())
}

func (p *ProxyInterface) ResourceOwnerChanged(resource *interpreter.CompositeValue, oldOwner common.Address, newOwner common.Address) {
	panic(UnimplementedError())
}
