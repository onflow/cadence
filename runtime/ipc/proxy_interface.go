package ipc

import (
	"fmt"
	pb "github.com/onflow/cadence/runtime/ipc/protobuf"
	"net"
	"time"

	"github.com/onflow/atree"
	"github.com/onflow/cadence"
	"github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/ipc/bridge"
	"github.com/onflow/cadence/runtime/tests/utils"
	"github.com/opentracing/opentracing-go"
)

var _ runtime.Interface = &ProxyInterface{}

// ProxyInterface converts the `runtime.Interface` Go method calls to IPC calls over the socket.
// Results are again converted back from IPC serialized format to corresponding Go structs.
type ProxyInterface struct {
	conn          net.Conn
	runtimeBridge *bridge.RuntimeBridge
}

func NewProxyInterface(conn net.Conn, runtimeBridge *bridge.RuntimeBridge) *ProxyInterface {
	return &ProxyInterface{
		conn:          conn,
		runtimeBridge: runtimeBridge,
	}
}

func (p *ProxyInterface) listen() *bridge.Message {
	// Keep listening until the final response is received.
	//
	// Rationale:
	// Once the initial request is sent to cadence, it may respond back
	// with requests (i.e: 'Interface' method calls). Hence, keep listening
	// to those requests and respond back. Terminate once the final ack
	// is received.

	for {
		message := ReadMessage(p.conn)

		var response *bridge.Message

		switch message.Type {
		case pb.MessageType_REQUEST:
			response = ServeRequest(p, message.GetReq())

		case pb.MessageType_RESPONSE:
			return message

		case pb.MessageType_ERROR:
			return message

		default:
			panic("unsupported")
		}

		WriteMessage(p.conn, response)
	}
}

func ServeRequest(
	proxyInterface *ProxyInterface,
	request *bridge.Request,
) *bridge.Message {

	context := runtime.Context{
		Interface: proxyInterface,
		Location:  utils.TestLocation,
	}
	context.InitializeCodesAndPrograms()

	runtimeBridge := proxyInterface.runtimeBridge

	var response *bridge.Message

	// TODO: change to switch on message type + meta-info
	switch request.Name {
	case RuntimeMethodExecuteScript:
		response = runtimeBridge.ExecuteScript(request.Params, context)

	case RuntimeMethodExecuteTransaction:
		response = runtimeBridge.ExecuteTransaction(request.Params, context)

	case RuntimeMethodInvokeContractFunction:
		response = runtimeBridge.InvokeContractFunction()

	default:
		response = bridge.NewErrorMessage(
			fmt.Sprintf("unsupported request '%s'", request.Name),
		)
	}

	return response
}

func (p *ProxyInterface) ResolveLocation(identifiers []runtime.Identifier, location runtime.Location) ([]runtime.ResolvedLocation, error) {
	request := bridge.NewRequestMessage(InterfaceMethodResolveLocation)
	WriteMessage(p.conn, request)

	// NOTE: this assumes when cadence call any 'Interface' method,
	// there will only be one response from the host-env.
	// i.e: Assumes there will not be any more requests from the host-env,
	// until the current request is served.

	// TODO: allow back and forth messaging rather than just one response??
	_ = p.listen()

	// TODO: implement
	return []runtime.ResolvedLocation{
		{
			Location:    utils.TestLocation,
			Identifiers: []ast.Identifier{},
		},
	}, nil
}

func (p *ProxyInterface) GetCode(location runtime.Location) ([]byte, error) {
	request := bridge.NewRequestMessage(InterfaceMethodGetCode)

	WriteMessage(p.conn, request)
	msg := p.listen()

	return []byte(msg.GetRes().Value), nil
}

func (p *ProxyInterface) GetProgram(location runtime.Location) (*interpreter.Program, error) {
	request := bridge.NewRequestMessage(InterfaceMethodGetProgram)

	WriteMessage(p.conn, request)
	_ = p.listen()

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

func (p *ProxyInterface) GetAccountContractCode(address runtime.Address, name string) (code []byte, err error) {
	panic("implement me")
}

func (p *ProxyInterface) RemoveAccountContractCode(address runtime.Address, name string) (err error) {
	panic("implement me")
}

func (p *ProxyInterface) GetSigningAccounts() ([]runtime.Address, error) {
	panic("implement me")
}

func (p *ProxyInterface) ProgramLog(s string) error {
	request := bridge.NewRequestMessage(InterfaceMethodProgramLog, s)

	WriteMessage(p.conn, request)

	_ = p.listen()

	return nil
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
