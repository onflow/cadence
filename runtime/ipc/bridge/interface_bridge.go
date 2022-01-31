package bridge

import (
	"fmt"

	"github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/ipc/protobuf"
)

// InterfaceBridge converts the IPC call to the `runtime.Interface` method invocation
// and convert the results back to IPC serializable format.
type InterfaceBridge struct {
	Interface runtime.Interface
}

func NewInterfaceBridge(runtimeInterface runtime.Interface) *InterfaceBridge {
	return &InterfaceBridge{
		Interface: runtimeInterface,
	}
}

func (b *InterfaceBridge) GetCode(params []*pb.Parameter) pb.Message {
	if len(params) != 1 {
		panic(errors.UnreachableError{})
	}

	location := pb.ToRuntimeLocation(params[0])

	code, err := b.Interface.GetCode(location)
	if err != nil {
		return pb.NewErrorMessage(
			fmt.Sprintf("error occured while retrieving code: '%s'", err.Error()),
		)
	}

	return pb.NewResponseMessage(
		pb.AsAny(pb.NewBytes(code)),
	)
}

func (b *InterfaceBridge) GetProgram(params []*pb.Parameter) pb.Message {
	if len(params) != 1 {
		panic(errors.UnreachableError{})
	}

	location := pb.ToRuntimeLocation(params[0])

	_, err := b.Interface.GetProgram(location)
	if err != nil {
		return pb.NewErrorMessage(
			fmt.Sprintf("error occured while retrieving program: '%s'", err.Error()),
		)
	}

	return pb.NewResponseMessage(
		pb.AsAny(pb.NewString("some program")),
	)
}

func (b *InterfaceBridge) ResolveLocation(params []*pb.Parameter) pb.Message {
	if len(params) != 2 {
		panic(errors.UnreachableError{})
	}

	identifiers := pb.ToRuntimeIdentifiersFromAny(params[0])
	location := pb.ToRuntimeLocation(params[1])

	resolvedLocation, err := b.Interface.ResolveLocation(identifiers, location)
	if err != nil {
		return pb.NewErrorMessage(
			fmt.Sprintf("error occured while retrieving location: '%s'", err.Error()),
		)
	}

	return pb.NewResponseMessage(
		pb.AsAny(pb.NewResolvedLocations(resolvedLocation)),
	)
}

func (b *InterfaceBridge) ProgramLog(params []*pb.Parameter) pb.Message {
	if len(params) != 1 {
		panic(errors.UnreachableError{})
	}

	msg := pb.ToRuntimeString(params[0])

	err := b.Interface.ProgramLog(msg)
	if err != nil {
		return pb.NewErrorMessage(
			fmt.Sprintf("error occured while logging: '%s'", err.Error()),
		)
	}

	return pb.EmptyResponse
}

func (b *InterfaceBridge) GetAccountContractCode(params []*pb.Parameter) pb.Message {
	if len(params) != 2 {
		panic(errors.UnreachableError{})
	}

	addressBytes := pb.ToRuntimeBytes(params[0])
	address := common.MustBytesToAddress(addressBytes)

	name := pb.ToRuntimeString(params[1])

	code, err := b.Interface.GetAccountContractCode(address, name)
	if err != nil {
		return pb.NewErrorMessage(
			fmt.Sprintf("error occured while retrieving contract code: '%s'", err.Error()),
		)
	}

	return pb.NewResponseMessage(
		pb.AsAny(pb.NewBytes(code)),
	)
}

func (b *InterfaceBridge) UpdateAccountContractCode(params []*pb.Parameter) pb.Message {
	if len(params) != 3 {
		panic(errors.UnreachableError{})
	}

	addressBytes := pb.ToRuntimeBytes(params[0])
	address := common.MustBytesToAddress(addressBytes)

	name := pb.ToRuntimeString(params[1])

	code := pb.ToRuntimeBytes(params[2])

	err := b.Interface.UpdateAccountContractCode(address, name, code)
	if err != nil {
		return pb.NewErrorMessage(
			fmt.Sprintf("error occured while updating contract code: '%s'", err.Error()),
		)
	}

	return pb.EmptyResponse
}

func (b *InterfaceBridge) GetValue(params []*pb.Parameter) pb.Message {
	if len(params) != 2 {
		panic(errors.UnreachableError{})
	}

	owner := pb.ToRuntimeBytes(params[0])
	key := pb.ToRuntimeBytes(params[1])

	value, err := b.Interface.GetValue(owner, key)
	if err != nil {
		return pb.NewErrorMessage(
			fmt.Sprintf("error occured while retrieving value: '%s'", err.Error()),
		)
	}

	return pb.NewResponseMessage(
		pb.AsAny(pb.NewBytes(value)),
	)
}

func (b *InterfaceBridge) SetValue(params []*pb.Parameter) pb.Message {
	if len(params) != 3 {
		panic(errors.UnreachableError{})
	}

	owner := pb.ToRuntimeBytes(params[0])
	key := pb.ToRuntimeBytes(params[1])
	value := pb.ToRuntimeBytes(params[2])

	err := b.Interface.SetValue(owner, key, value)
	if err != nil {
		return pb.NewErrorMessage(
			fmt.Sprintf("error occured while setting value: '%s'", err.Error()),
		)
	}

	return pb.EmptyResponse
}

func (b *InterfaceBridge) AllocateStorageIndex(params []*pb.Parameter) pb.Message {
	if len(params) != 1 {
		panic(errors.UnreachableError{})
	}

	owner := pb.ToRuntimeBytes(params[0])

	storageIndex, err := b.Interface.AllocateStorageIndex(owner)
	if err != nil {
		return pb.NewErrorMessage(
			fmt.Sprintf("error occured while allocating storage index: '%s'", err.Error()),
		)
	}

	return pb.NewResponseMessage(
		pb.AsAny(pb.NewBytes(storageIndex[:])),
	)
}
