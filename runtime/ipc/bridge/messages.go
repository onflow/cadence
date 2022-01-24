package bridge

import (
	"github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	pb "github.com/onflow/cadence/runtime/ipc/protobuf"
	"google.golang.org/protobuf/types/known/anypb"
)

const (
	REQUEST  = pb.MessageType_REQUEST
	RESPONSE = pb.MessageType_RESPONSE
	ERROR    = pb.MessageType_ERROR
)

type Message = pb.Message

type Request = pb.Request

type Response = pb.Response

type Error = pb.Error

func NewErrorMessage(errMsg string) *Message {
	return &Message{
		Type: ERROR,
		Payloads: &pb.Message_Err{
			Err: &Error{
				Err: errMsg,
			},
		},
	}
}

func NewResponseMessage(value string) *Message {
	return &Message{
		Type: RESPONSE,
		Payloads: &pb.Message_Res{
			Res: &Response{
				Value: value,
			},
		},
	}
}

func NewRequestMessage(name string, params ...*anypb.Any) *Message {
	return &Message{
		Type: REQUEST,
		Payloads: &pb.Message_Req{
			Req: &Request{
				Name:   name,
				Params: params,
			},
		},
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

func NewLocation(location common.Location) (*anypb.Any, error) {
	var pbLocation *anypb.Any
	var err error

	switch location := location.(type) {
	case common.StringLocation:
		pbLocation, err = anypb.New(&pb.StringLocation{
			Content: string(location),
		})
	case common.IdentifierLocation:
		pbLocation, err = anypb.New(&pb.IdentifierLocation{
			Content: string(location),
		})
	case common.AddressLocation:
		pbLocation, err = anypb.New(&pb.AddressLocation{
			Address: location.Address[:],
			Name:    location.Name,
		})
	case common.TransactionLocation:
		pbLocation, err = anypb.New(&pb.TransactionLocation{
			Content: location,
		})
	}

	if err != nil {
		return nil, err
	}

	return pbLocation, nil
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
