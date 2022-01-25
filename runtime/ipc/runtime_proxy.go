package ipc

import (
	"github.com/onflow/cadence"
	"github.com/onflow/cadence/encoding/json"
	"github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/ipc/bridge"
	pb "github.com/onflow/cadence/runtime/ipc/protobuf"
	"github.com/onflow/cadence/runtime/sema"
)

// ProxyRuntime calls Cadence functionalities over the sockets.
// Converts the `runtime.Runtime` Go method calls to IPC calls.
// Results are again converted back from IPC serialized format to corresponding Go structs.
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
	scriptParam := bridge.AsAny(
		bridge.NewScript(script.Source, script.Arguments),
	)

	location, err := bridge.NewLocation(context.Location)
	if err != nil {
		return nil, err
	}
	locationParam := bridge.AsAny(location)

	request := bridge.NewRequestMessage(
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

	bytes := &pb.Bytes{}
	err = response.Value.UnmarshalTo(bytes)
	if err != nil {
		return nil, err
	}

	return json.Decode(bytes.Content)
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
