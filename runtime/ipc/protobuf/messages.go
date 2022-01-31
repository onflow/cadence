package pb

import (
	"fmt"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
)

type Message = proto.Message

type Parameter = anypb.Any

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

func NewString(content string) *String {
	return &String{
		Content: content,
	}
}

func ToRuntimeString(any *anypb.Any) string {
	str := &String{}
	err := any.UnmarshalTo(str)
	if err != nil {
		panic(err)
	}

	return str.Content
}

func NewBytes(content []byte) *Bytes {
	return &Bytes{
		Content: content,
	}
}

func ToRuntimeBytes(any *anypb.Any) []byte {
	bytes := &Bytes{}
	err := any.UnmarshalTo(bytes)
	if err != nil {
		panic(err)
	}

	return bytes.Content
}

func NewScript(source []byte, arguments [][]byte) *Script {
	return &Script{
		Source:    source,
		Arguments: arguments,
	}
}

func ToRuntimeScript(any *anypb.Any) runtime.Script {
	s := &Script{}
	err := any.UnmarshalTo(s)
	if err != nil {
		panic(err)
	}

	script := runtime.Script{
		Source:    s.Source,
		Arguments: s.Arguments,
	}
	return script
}

func NewLocation(runtimeLocation runtime.Location) (proto.Message, error) {
	var location proto.Message

	switch runtimeLocation := runtimeLocation.(type) {
	case common.StringLocation:
		location = &StringLocation{
			Content: string(runtimeLocation),
		}
	case common.IdentifierLocation:
		location = &IdentifierLocation{
			Content: string(runtimeLocation),
		}
	case common.AddressLocation:
		location = &AddressLocation{
			Address: runtimeLocation.Address[:],
			Name:    runtimeLocation.Name,
		}
	case common.TransactionLocation:
		location = &TransactionLocation{
			Content: runtimeLocation,
		}
	case common.ScriptLocation:
		location = &ScriptLocation{
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
	case *StringLocation:
		return common.StringLocation(location.GetContent())
	case *IdentifierLocation:
		return common.IdentifierLocation(location.GetContent())
	case *AddressLocation:
		address, err := common.BytesToAddress(location.GetAddress())
		if err != nil {
			panic(err)
		}

		return common.AddressLocation{
			Address: address,
			Name:    location.GetName(),
		}
	case *ScriptLocation:
		return common.ScriptLocation(location.GetContent())
	case *TransactionLocation:
		return common.TransactionLocation(location.GetContent())
	default:
		panic(errors.UnreachableError{})
	}
}

func NewIdentifiers(identifiers []runtime.Identifier) *Array {
	elements := make([]*anypb.Any, 0, len(identifiers))

	for _, identifier := range identifiers {
		elements = append(
			elements,
			AsAny(NewString(identifier.Identifier)),
		)
	}

	return &Array{
		Elements: elements,
	}
}

func ToRuntimeIdentifiersFromAny(any *anypb.Any) []runtime.Identifier {
	array := &Array{}
	err := any.UnmarshalTo(array)
	if err != nil {
		panic(err)
	}

	return ToRuntimeIdentifiers(array)
}

func ToRuntimeIdentifiers(identifiersArray *Array) []runtime.Identifier {
	identifiers := make([]runtime.Identifier, 0, len(identifiersArray.Elements))

	for _, element := range identifiersArray.Elements {
		identifierStr := ToRuntimeString(element)

		identifiers = append(
			identifiers,
			runtime.Identifier{
				Identifier: identifierStr,
			},
		)
	}

	return identifiers
}

func NewResolvedLocation(resolvedLoc runtime.ResolvedLocation) *ResolvedLocation {
	location, err := NewLocation(resolvedLoc.Location)
	if err != nil {
		panic(err)
	}

	return &ResolvedLocation{
		Location:    AsAny(location),
		Identifiers: NewIdentifiers(resolvedLoc.Identifiers),
	}
}

func ToRuntimeResolvedLocation(resolvedLoc *ResolvedLocation) runtime.ResolvedLocation {
	return runtime.ResolvedLocation{
		Location:    ToRuntimeLocation(resolvedLoc.Location),
		Identifiers: ToRuntimeIdentifiers(resolvedLoc.Identifiers),
	}
}

func NewResolvedLocations(resolvedLocations []runtime.ResolvedLocation) *Array {
	elements := make([]*anypb.Any, 0, len(resolvedLocations))

	for _, location := range resolvedLocations {
		elements = append(elements, AsAny(NewResolvedLocation(location)))
	}

	return &Array{
		Elements: elements,
	}
}

func ToRuntimeResolvedLocationsFromAny(any *anypb.Any) []runtime.ResolvedLocation {
	array := &Array{}
	err := any.UnmarshalTo(array)
	if err != nil {
		panic(err)
	}

	return ToRuntimeResolvedLocations(array)
}

func ToRuntimeResolvedLocations(resolvedLocationsArray *Array) []runtime.ResolvedLocation {
	resolvedLocations := make([]runtime.ResolvedLocation, 0, len(resolvedLocationsArray.Elements))

	for _, location := range resolvedLocationsArray.Elements {
		loc := &ResolvedLocation{}
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
