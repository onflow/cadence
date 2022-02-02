package ipc

import (
	"fmt"
	"net"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/encoding/json"
	"github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/ipc/bridge"
	"github.com/onflow/cadence/runtime/ipc/protobuf"
	"github.com/onflow/cadence/runtime/sema"
)

// ProxyRuntime calls Cadence functionalities over the sockets.
// Converts the `runtime.Runtime` Go method calls to IPC calls.
// Results are again converted back from IPC serialized format to corresponding Go structs.
type ProxyRuntime struct {
	interfaceBridge *bridge.InterfaceBridge
}

var _ runtime.Runtime = &ProxyRuntime{}

func NewProxyRuntime(interfaceBridge *bridge.InterfaceBridge) *ProxyRuntime {
	return &ProxyRuntime{
		interfaceBridge: interfaceBridge,
	}
}

func (r *ProxyRuntime) ExecuteScript(script runtime.Script, context runtime.Context) (cadence.Value, error) {
	scriptParam := pb.AsAny(
		pb.NewScript(script.Source, script.Arguments),
	)

	location, err := pb.NewLocation(context.Location)
	if err != nil {
		return nil, err
	}
	locationParam := pb.AsAny(location)

	request := pb.NewRequestMessage(
		RuntimeMethodExecuteScript,
		scriptParam,
		locationParam,
	)

	conn, err := bridge.NewRuntimeConnection()
	if err != nil {
		return nil, err
	}
	defer bridge.CloseConnection(conn)

	bridge.WriteMessage(conn, request)

	response, err := r.listen(conn)
	if err != nil {
		return nil, err
	}

	bytes := pb.ToRuntimeBytes(response.Value)

	return json.Decode(bytes)
}

func (r *ProxyRuntime) ExecuteTransaction(script runtime.Script, context runtime.Context) error {
	scriptParam := pb.AsAny(
		pb.NewScript(script.Source, script.Arguments),
	)

	location, err := pb.NewLocation(context.Location)
	if err != nil {
		return err
	}
	locationParam := pb.AsAny(location)

	request := pb.NewRequestMessage(
		RuntimeMethodExecuteTransaction,
		scriptParam,
		locationParam,
	)

	// TODO: How to re-use the existing connection for subsequent calls?
	conn, err := bridge.NewRuntimeConnection()
	if err != nil {
		return err
	}
	defer bridge.CloseConnection(conn)

	bridge.WriteMessage(conn, request)

	_, err = r.listen(conn)
	if err != nil {
		return err
	}

	return nil
}

func (r *ProxyRuntime) InvokeContractFunction(contractLocation common.AddressLocation, functionName string, arguments []interpreter.Value, argumentTypes []sema.Type, context runtime.Context) (cadence.Value, error) {
	panic(UnimplementedError())
}

func (r *ProxyRuntime) ParseAndCheckProgram(source []byte, context runtime.Context) (*interpreter.Program, error) {
	panic(UnimplementedError())
}

func (r *ProxyRuntime) SetCoverageReport(coverageReport *runtime.CoverageReport) {
	panic(UnimplementedError())
}

func (r *ProxyRuntime) SetContractUpdateValidationEnabled(enabled bool) {
	panic(UnimplementedError())
}

func (r *ProxyRuntime) SetAtreeValidationEnabled(enabled bool) {
	panic(UnimplementedError())
}

func (r *ProxyRuntime) SetTracingEnabled(enabled bool) {
	panic(UnimplementedError())
}

func (r *ProxyRuntime) SetResourceOwnerChangeHandlerEnabled(enabled bool) {
	panic(UnimplementedError())
}

func (r *ProxyRuntime) ReadStored(address common.Address, path cadence.Path, context runtime.Context) (cadence.Value, error) {
	panic(UnimplementedError())
}

func (r *ProxyRuntime) ReadLinked(address common.Address, path cadence.Path, context runtime.Context) (cadence.Value, error) {
	panic(UnimplementedError())
}

func (r *ProxyRuntime) ServeRequest(request *pb.Request) pb.Message {
	var response pb.Message

	// All 'Interface' methods goes here
	switch request.Name {
	case InterfaceMethodGetCode:
		response = r.interfaceBridge.GetCode(request.Params)

	case InterfaceMethodGetProgram:
		response = r.interfaceBridge.GetProgram(request.Params)

	case InterfaceMethodResolveLocation:
		response = r.interfaceBridge.ResolveLocation(request.Params)

	case InterfaceMethodProgramLog:
		response = r.interfaceBridge.ProgramLog(request.Params)

	case InterfaceMethodGetAccountContractCode:
		response = r.interfaceBridge.GetAccountContractCode(request.Params)

	case InterfaceMethodUpdateAccountContractCode:
		response = r.interfaceBridge.UpdateAccountContractCode(request.Params)

	case InterfaceMethodGetValue:
		response = r.interfaceBridge.GetValue(request.Params)

	case InterfaceMethodSetValue:
		response = r.interfaceBridge.SetValue(request.Params)

	case InterfaceMethodAllocateStorageIndex:
		response = r.interfaceBridge.AllocateStorageIndex(request.Params)

	default:
		fmt.Printf("unsupported request '%s'\n", request.Name)
	}

	return response
}

func (r *ProxyRuntime) listen(conn net.Conn) (*pb.Response, error) {
	// Keep listening until the final response is received.
	//
	// Rationale:
	// Once the initial request is sent to cadence runtime, it may respond back
	// with requests (i.e: 'Interface' method calls). Hence, keep listening
	// to those requests and respond back. Terminate once the final ack
	// is received.

	for {
		msg := bridge.ReadMessage(conn)

		switch msg := msg.(type) {
		case *pb.Request:
			respMsg := r.ServeRequest(msg)
			bridge.WriteMessage(conn, respMsg)
		case *pb.Response:
			return msg, nil
		case *pb.Error:
			return nil, fmt.Errorf(msg.GetErr())
		default:
			return nil, fmt.Errorf("unsupported message")
		}
	}
}
