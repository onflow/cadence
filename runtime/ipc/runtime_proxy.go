package ipc

import (
	"github.com/onflow/cadence"
	"github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/ipc/bridge"
	"github.com/onflow/cadence/runtime/sema"
)

// ProxyRuntime calls Cadence functionalities over the sockets.
// Converts the `runtime.Runtime` Go method calls to IPC calls and
// the results are again converted back to Go corresponding structs.
type ProxyRuntime struct {
}

var _ runtime.Runtime = &ProxyRuntime{}

func NewProxyRuntime(runtimeInterface runtime.Interface) *ProxyRuntime {
	// TODO: Move to an appropriate place
	// TODO: handle termination
	go StartInterfaceService(runtimeInterface)

	return &ProxyRuntime{}
}

func (r *ProxyRuntime) ExecuteScript(script runtime.Script, context runtime.Context) (cadence.Value, error) {
	request := bridge.NewRequestMessage(
		RuntimeMethodExecuteScript,
		string(script.Source),
	)

	conn := NewRuntimeConnection()

	WriteMessage(conn, request)

	_ = ReadMessage(conn)

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
