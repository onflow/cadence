package bridge

import (
	"fmt"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	pb "github.com/onflow/cadence/runtime/ipc/protobuf"
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

func NewResponseMessage(value *anypb.Any) *Response {
	return &Response{
		Value: value,
	}
}

var EmptyResponse = &Response{}

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

func NewBytes(content []byte) *pb.Bytes {
	return &pb.Bytes{
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
	case common.ScriptLocation:
		location = &pb.ScriptLocation{
			Content: runtimeLocation,
		}
	default:
		return nil, fmt.Errorf("unsupported runtime location: %s", runtimeLocation)
	}

	return location, nil
}

func ToRuntimeLocation(any *anypb.Any) runtime.Location {
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
	case *pb.ScriptLocation:
		return common.ScriptLocation(location.GetContent())
	case *pb.TransactionLocation:
		return common.TransactionLocation(location.GetContent())
	default:
		panic(errors.UnreachableError{})
	}
}

func NewIdentifiers(identifiers []runtime.Identifier) *pb.Array {
	elements := make([]*anypb.Any, 0, len(identifiers))

	for _, identifier := range identifiers {
		elements = append(
			elements,
			AsAny(NewString(identifier.Identifier)),
		)
	}

	return &pb.Array{
		Elements: elements,
	}
}

func ToRuntimeIdentifiersFromAny(any *anypb.Any) []runtime.Identifier {
	array := &pb.Array{}
	err := any.UnmarshalTo(array)
	if err != nil {
		panic(err)
	}

	return ToRuntimeIdentifiers(array)
}

func ToRuntimeIdentifiers(identifiersArray *pb.Array) []runtime.Identifier {
	identifiers := make([]runtime.Identifier, 0, len(identifiersArray.Elements))

	for _, element := range identifiersArray.Elements {
		str := &pb.String{}
		err := element.UnmarshalTo(str)
		if err != nil {
			panic(err)
		}

		identifiers = append(
			identifiers,
			runtime.Identifier{
				Identifier: str.Content,
			},
		)
	}

	return identifiers
}

func NewResolvedLocation(resolvedLoc runtime.ResolvedLocation) *pb.ResolvedLocation {
	location, err := NewLocation(resolvedLoc.Location)
	if err != nil {
		panic(err)
	}

	return &pb.ResolvedLocation{
		Location:    AsAny(location),
		Identifiers: NewIdentifiers(resolvedLoc.Identifiers),
	}
}

func ToRuntimeResolvedLocation(resolvedLoc *pb.ResolvedLocation) runtime.ResolvedLocation {
	return runtime.ResolvedLocation{
		Location:    ToRuntimeLocation(resolvedLoc.Location),
		Identifiers: ToRuntimeIdentifiers(resolvedLoc.Identifiers),
	}
}

func NewResolvedLocations(resolvedLocations []runtime.ResolvedLocation) *pb.Array {
	elements := make([]*anypb.Any, 0, len(resolvedLocations))

	for _, location := range resolvedLocations {
		elements = append(elements, AsAny(NewResolvedLocation(location)))
	}

	return &pb.Array{
		Elements: elements,
	}
}

func ToRuntimeResolvedLocationsFromAny(any *anypb.Any) []runtime.ResolvedLocation {
	array := &pb.Array{}
	err := any.UnmarshalTo(array)
	if err != nil {
		panic(err)
	}

	return ToRuntimeResolvedLocations(array)
}

func ToRuntimeResolvedLocations(resolvedLocationsArray *pb.Array) []runtime.ResolvedLocation {
	resolvedLocations := make([]runtime.ResolvedLocation, 0, len(resolvedLocationsArray.Elements))

	for _, location := range resolvedLocationsArray.Elements {
		loc := &pb.ResolvedLocation{}
		err := location.UnmarshalTo(loc)
		if err != nil {
			panic(err)
		}

		resolvedLocations = append(resolvedLocations, ToRuntimeResolvedLocation(loc))
	}

	return resolvedLocations
}

func AsAny(value proto.Message) *anypb.Any {
	param, err := anypb.New(value)

	// These errors are not handle-able. Hence, panic.
	if err != nil {
		panic(err)
	}

	return param
}
