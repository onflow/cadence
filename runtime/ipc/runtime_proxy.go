package ipc

import (
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
}

var _ runtime.Runtime = &ProxyRuntime{}

func NewProxyRuntime() *ProxyRuntime {
	return &ProxyRuntime{}
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

	conn := bridge.NewRuntimeConnection()

	bridge.WriteMessage(conn, request)

	response, err := bridge.ReadResponse(conn)
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

	conn := bridge.NewRuntimeConnection()

	bridge.WriteMessage(conn, request)

	_, err = bridge.ReadResponse(conn)
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
