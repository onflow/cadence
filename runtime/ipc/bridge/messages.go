package bridge

import (
	"fmt"

	"github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	pb "github.com/onflow/cadence/runtime/ipc/protobuf"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

type Message = proto.Message

type Request = pb.Request

type Response = pb.Response

type Error = pb.Error

func NewErrorMessage(errMsg string) *Error {
	return &Error{
		Err: errMsg,
	}
}

func NewResponseMessage(value string) *Response {
	return &Response{
		Value: value,
	}
}

func NewRequestMessage(name string, params ...*anypb.Any) *Request {
	return &Request{
		Name:   name,
		Params: params,
	}
}

func NewString(content string) *pb.String {
	return &pb.String{
		Content: content,
	}
}

func NewScript(source []byte, arguments [][]byte) *pb.Script {
	return &pb.Script{
		Source:    source,
		Arguments: arguments,
	}
}

func NewLocation(runtimeLocation runtime.Location) (proto.Message, error) {
	var location proto.Message

	switch runtimeLocation := runtimeLocation.(type) {
	case common.StringLocation:
		location = &pb.StringLocation{
			Content: string(runtimeLocation),
		}
	case common.IdentifierLocation:
		location = &pb.IdentifierLocation{
			Content: string(runtimeLocation),
		}
	case common.AddressLocation:
		location = &pb.AddressLocation{
			Address: runtimeLocation.Address[:],
			Name:    runtimeLocation.Name,
		}
	case common.TransactionLocation:
		location = &pb.TransactionLocation{
			Content: runtimeLocation,
		}
	default:
		return nil, fmt.Errorf("unsupported runtime location: %s", runtimeLocation)
	}

	return location, nil
}

func LocationToRuntimeLocation(any *anypb.Any) runtime.Location {
	location, err := any.UnmarshalNew()
	if err != nil {
		panic(err)
	}

	switch location := location.(type) {
	case *pb.StringLocation:
		return common.StringLocation(location.GetContent())
	case *pb.IdentifierLocation:
		return common.IdentifierLocation(location.GetContent())
	case *pb.AddressLocation:
		address, err := common.BytesToAddress(location.GetAddress())
		if err != nil {
			panic(err)
		}

		return common.AddressLocation{
			Address: address,
			Name:    location.GetName(),
		}
	case *pb.TransactionLocation:
		return common.TransactionLocation(location.GetContent())
	default:
		panic(errors.UnreachableError{})
	}
}

func AsParameter(value proto.Message) *anypb.Any {
	param, err := anypb.New(value)

	// These errors are not handle-able. Hence, panic.
	if err != nil {
		panic(err)
	}

	return param
}
