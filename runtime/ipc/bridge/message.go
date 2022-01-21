package bridge

import (
	pb "github.com/onflow/cadence/runtime/ipc/protobuf"
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

func NewRequestMessage(name string, params ...string) *Message {
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
