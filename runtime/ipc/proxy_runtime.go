package ipc

import (
	"fmt"
	"net"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/ipc/bridge"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/tests/utils"
)

// ProxyRuntime calls Cadence functionalities over the sockets.
// Converts the `runtime.Runtime` Go method calls to IPC calls and
// the results are again converted back to Go corresponding structs.
type ProxyRuntime struct {
	conn            net.Conn
	interfaceBridge *bridge.InterfaceBridge
}

var _ runtime.Runtime = &ProxyRuntime{}

func NewProxyRuntime(runtimeInterface runtime.Interface) *ProxyRuntime {
	conn, err := net.Dial(UnixNetwork, SocketAddress)
	HandleError(err)

	return &ProxyRuntime{
		conn:            conn,
		interfaceBridge: bridge.NewInterfaceBridge(runtimeInterface),
	}
}

func (r *ProxyRuntime) ExecuteScript(script runtime.Script, context runtime.Context) (cadence.Value, error) {
	request := &bridge.Request{
		Name: RuntimeMethodExecuteScript,
	}

	WriteMessage(r.conn, request)

	result := r.listen()

	fmt.Println(result)

	return nil, nil
}

func (r *ProxyRuntime) ExecuteTransaction(script runtime.Script, context runtime.Context) error {
	panic("implement me")
}

func (r *ProxyRuntime) InvokeContractFunction(contractLocation common.AddressLocation, functionName string, arguments []interpreter.Value, argumentTypes []sema.Type, context runtime.Context) (cadence.Value, error) {
	panic("implement me")
}

func (r *ProxyRuntime) ParseAndCheckProgram(source []byte, context runtime.Context) (*interpreter.Program, error) {
	panic("implement me")
}

func (r *ProxyRuntime) SetCoverageReport(coverageReport *runtime.CoverageReport) {
	panic("implement me")
}

func (r *ProxyRuntime) SetContractUpdateValidationEnabled(enabled bool) {
	panic("implement me")
}

func (r *ProxyRuntime) SetAtreeValidationEnabled(enabled bool) {
	panic("implement me")
}

func (r *ProxyRuntime) SetTracingEnabled(enabled bool) {
	panic("implement me")
}

func (r *ProxyRuntime) SetResourceOwnerChangeHandlerEnabled(enabled bool) {
	panic("implement me")
}

func (r *ProxyRuntime) ReadStored(address common.Address, path cadence.Path, context runtime.Context) (cadence.Value, error) {
	panic("implement me")
}

func (r *ProxyRuntime) ReadLinked(address common.Address, path cadence.Path, context runtime.Context) (cadence.Value, error) {
	panic("implement me")
}

func (r *ProxyRuntime) listen() bridge.Message {
	// Keep listening until the final response is received.
	//
	// Rationale:
	// Once the initial request is sent to cadence, it may respond back
	// with requests (i.e: 'Interface' method calls). Hence, keep listening
	// to those requests and respond back. Terminate once the final ack
	// is received.

	for {
		message := ReadMessage(r.conn)

		var response bridge.Message

		// TODO: switch on message header/meta_info
		// All 'Interface' methods goes here

		switch message.String() {
		case InterfaceMethodGetCode:
			location := utils.TestLocation
			response = r.interfaceBridge.GetCode(location)

		case InterfaceMethodGetProgram:
			location := utils.TestLocation
			response = r.interfaceBridge.GetProgram(location)

		case InterfaceMethodResolveLocation:

		default:
			return message
		}

		WriteMessage(r.conn, response)
	}
}
