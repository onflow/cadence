/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2022 Dapper Labs, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package stdlib

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
)

// This file defines functions built in to the Flow runtime.

const authAccountFunctionDocString = `
Creates a new account, paid by the given existing account
`

var authAccountFunctionType = &sema.FunctionType{
	Parameters: []*sema.Parameter{
		{
			Identifier: "payer",
			TypeAnnotation: sema.NewTypeAnnotation(
				sema.AuthAccountType,
			),
		},
	},
	ReturnTypeAnnotation: sema.NewTypeAnnotation(
		sema.AuthAccountType,
	),
}

const getAccountFunctionDocString = `
Returns the public account for the given address
`

var getAccountFunctionType = &sema.FunctionType{
	Parameters: []*sema.Parameter{
		{
			Label:      sema.ArgumentLabelNotRequired,
			Identifier: "address",
			TypeAnnotation: sema.NewTypeAnnotation(
				&sema.AddressType{},
			),
		},
	},
	ReturnTypeAnnotation: sema.NewTypeAnnotation(
		sema.PublicAccountType,
	),
}

var LogFunctionType = &sema.FunctionType{
	Parameters: []*sema.Parameter{
		{
			Label:      sema.ArgumentLabelNotRequired,
			Identifier: "value",
			TypeAnnotation: sema.NewTypeAnnotation(
				sema.AnyStructType,
			),
		},
	},
	ReturnTypeAnnotation: sema.NewTypeAnnotation(
		sema.VoidType,
	),
}

const getCurrentBlockFunctionDocString = `
Returns the current block, i.e. the block which contains the currently executed transaction
`

var getCurrentBlockFunctionType = &sema.FunctionType{
	ReturnTypeAnnotation: sema.NewTypeAnnotation(
		sema.BlockType,
	),
}

const getBlockFunctionDocString = `
Returns the block at the given height. If the given block does not exist the function returns nil
`

var getBlockFunctionType = &sema.FunctionType{
	Parameters: []*sema.Parameter{
		{
			Label:      "at",
			Identifier: "height",
			TypeAnnotation: sema.NewTypeAnnotation(
				sema.UInt64Type,
			),
		},
	},
	ReturnTypeAnnotation: sema.NewTypeAnnotation(
		&sema.OptionalType{
			Type: sema.BlockType,
		},
	),
}

const unsafeRandomFunctionDocString = `
Returns a pseudo-random number.

NOTE: The use of this function is unsafe if not used correctly.

Follow best practices to prevent security issues when using this function
`

var unsafeRandomFunctionType = &sema.FunctionType{
	ReturnTypeAnnotation: sema.NewTypeAnnotation(
		sema.UInt64Type,
	),
}

// FlowBuiltinImpls defines the set of functions needed to implement the Flow
// built-in functions.
type FlowBuiltinImpls struct {
	CreateAccount   interpreter.HostFunction
	GetAccount      interpreter.HostFunction
	Log             interpreter.HostFunction
	GetCurrentBlock interpreter.HostFunction
	GetBlock        interpreter.HostFunction
	UnsafeRandom    interpreter.HostFunction
}

// FlowBuiltInFunctions returns a list of standard library functions, bound to
// the provided implementation.
func FlowBuiltInFunctions(impls FlowBuiltinImpls) StandardLibraryFunctions {
	return StandardLibraryFunctions{
		NewStandardLibraryFunction(
			"AuthAccount",
			authAccountFunctionType,
			authAccountFunctionDocString,
			impls.CreateAccount,
		),
		NewStandardLibraryFunction(
			"getAccount",
			getAccountFunctionType,
			getAccountFunctionDocString,
			impls.GetAccount,
		),
		NewStandardLibraryFunction(
			"log",
			LogFunctionType,
			logFunctionDocString,
			impls.Log,
		),
		NewStandardLibraryFunction(
			"getCurrentBlock",
			getCurrentBlockFunctionType,
			getCurrentBlockFunctionDocString,
			impls.GetCurrentBlock,
		),
		NewStandardLibraryFunction(
			"getBlock",
			getBlockFunctionType,
			getBlockFunctionDocString,
			impls.GetBlock,
		),
		NewStandardLibraryFunction(
			"unsafeRandom",
			unsafeRandomFunctionType,
			unsafeRandomFunctionDocString,
			impls.UnsafeRandom,
		),
	}
}

func DefaultFlowBuiltinImpls() FlowBuiltinImpls {
	return FlowBuiltinImpls{
		CreateAccount: func(invocation interpreter.Invocation) interpreter.Value {
			panic(fmt.Errorf("cannot create accounts"))
		},
		GetAccount: func(invocation interpreter.Invocation) interpreter.Value {
			panic(fmt.Errorf("cannot get accounts"))
		},
		Log: LogFunction.Function.Function,
		GetCurrentBlock: func(invocation interpreter.Invocation) interpreter.Value {
			panic(fmt.Errorf("cannot get blocks"))
		},
		GetBlock: func(invocation interpreter.Invocation) interpreter.Value {
			panic(fmt.Errorf("cannot get blocks"))
		},
		UnsafeRandom: func(invocation interpreter.Invocation) interpreter.Value {
			return interpreter.NewUInt64Value(
				invocation.Interpreter,
				rand.Uint64,
			)
		},
	}
}

// Flow location

type FlowLocation struct{}

const FlowLocationPrefix = "flow"

func (l FlowLocation) ID() common.LocationID {
	return common.NewLocationID(FlowLocationPrefix)
}

func (l FlowLocation) MeteredID(memoryGauge common.MemoryGauge) common.LocationID {
	return common.NewMeteredLocationID(
		memoryGauge,
		FlowLocationPrefix,
	)
}

func (l FlowLocation) TypeID(memoryGauge common.MemoryGauge, qualifiedIdentifier string) common.TypeID {
	return common.NewMeteredTypeID(
		memoryGauge,
		FlowLocationPrefix,
		qualifiedIdentifier,
	)
}

func (l FlowLocation) QualifiedIdentifier(typeID common.TypeID) string {
	pieces := strings.SplitN(string(typeID), ".", 2)

	if len(pieces) < 2 {
		return ""
	}

	return pieces[1]
}

func (l FlowLocation) String() string {
	return "flow"
}

func (l FlowLocation) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Type string
	}{
		Type: "FlowLocation",
	})
}

func init() {
	common.RegisterTypeIDDecoder(
		FlowLocationPrefix,
		func(_ common.MemoryGauge, typeID string) (location common.Location, qualifiedIdentifier string, err error) {
			return decodeFlowLocationTypeID(typeID)
		},
	)
}

func decodeFlowLocationTypeID(typeID string) (FlowLocation, string, error) {

	const errorMessagePrefix = "invalid Flow location type ID"

	newError := func(message string) (FlowLocation, string, error) {
		return FlowLocation{}, "", fmt.Errorf("%s: %s", errorMessagePrefix, message)
	}

	if typeID == "" {
		return newError("missing prefix")
	}

	parts := strings.SplitN(typeID, ".", 2)

	pieceCount := len(parts)
	if pieceCount == 1 {
		return newError("missing qualified identifier")
	}

	prefix := parts[0]

	if prefix != FlowLocationPrefix {
		return FlowLocation{}, "", fmt.Errorf(
			"%s: invalid prefix: expected %q, got %q",
			errorMessagePrefix,
			FlowLocationPrefix,
			prefix,
		)
	}

	qualifiedIdentifier := parts[1]

	return FlowLocation{}, qualifiedIdentifier, nil
}

// built-in event types

func newFlowEventType(identifier string, parameters ...*sema.Parameter) *sema.CompositeType {

	eventType := &sema.CompositeType{
		Kind:       common.CompositeKindEvent,
		Location:   FlowLocation{},
		Identifier: identifier,
		Fields:     []string{},
		Members:    sema.NewStringMemberOrderedMap(),
	}

	for _, parameter := range parameters {

		eventType.Fields = append(eventType.Fields,
			parameter.Identifier,
		)

		eventType.Members.Set(
			parameter.Identifier,
			sema.NewUnmeteredPublicConstantFieldMember(
				eventType,
				parameter.Identifier,
				parameter.TypeAnnotation.Type,
				// TODO: add docstring support for parameters
				"",
			))

		eventType.ConstructorParameters = append(
			eventType.ConstructorParameters,
			parameter,
		)
	}

	return eventType
}

const HashSize = 32

var HashType = &sema.ConstantSizedType{
	Size: HashSize,
	Type: sema.UInt8Type,
}

var TypeIDsType = &sema.VariableSizedType{
	Type: sema.StringType,
}

var AccountEventAddressParameter = &sema.Parameter{
	Identifier:     "address",
	TypeAnnotation: sema.NewTypeAnnotation(&sema.AddressType{}),
}

var AccountEventCodeHashParameter = &sema.Parameter{
	Identifier:     "codeHash",
	TypeAnnotation: sema.NewTypeAnnotation(HashType),
}

var AccountEventPublicKeyParameter = &sema.Parameter{
	Identifier: "publicKey",
	TypeAnnotation: sema.NewTypeAnnotation(
		sema.ByteArrayType,
	),
}

var AccountEventContractsParameter = &sema.Parameter{
	Identifier:     "contracts",
	TypeAnnotation: sema.NewTypeAnnotation(TypeIDsType),
}

var AccountEventContractParameter = &sema.Parameter{
	Identifier:     "contract",
	TypeAnnotation: sema.NewTypeAnnotation(sema.StringType),
}

var AccountCreatedEventType = newFlowEventType(
	"AccountCreated",
	AccountEventAddressParameter,
)

var AccountKeyAddedEventType = newFlowEventType(
	"AccountKeyAdded",
	AccountEventAddressParameter,
	AccountEventPublicKeyParameter,
)

var AccountKeyRemovedEventType = newFlowEventType(
	"AccountKeyRemoved",
	AccountEventAddressParameter,
	AccountEventPublicKeyParameter,
)

var AccountContractAddedEventType = newFlowEventType(
	"AccountContractAdded",
	AccountEventAddressParameter,
	AccountEventCodeHashParameter,
	AccountEventContractParameter,
)

var AccountContractUpdatedEventType = newFlowEventType(
	"AccountContractUpdated",
	AccountEventAddressParameter,
	AccountEventCodeHashParameter,
	AccountEventContractParameter,
)

var AccountContractRemovedEventType = newFlowEventType(
	"AccountContractRemoved",
	AccountEventAddressParameter,
	AccountEventCodeHashParameter,
	AccountEventContractParameter,
)

var FlowBuiltInTypes StandardLibraryTypes
