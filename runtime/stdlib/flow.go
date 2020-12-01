/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2020 Dapper Labs, Inc.
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

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/trampoline"
)

// This file defines functions built in to the Flow runtime.

// built-in function types

var accountFunctionType = &sema.FunctionType{
	Parameters: []*sema.Parameter{
		{
			Identifier: "payer",
			TypeAnnotation: sema.NewTypeAnnotation(
				&sema.AuthAccountType{},
			),
		},
	},
	ReturnTypeAnnotation: sema.NewTypeAnnotation(
		&sema.AuthAccountType{},
	),
}

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
		&sema.PublicAccountType{},
	),
}

var logFunctionType = &sema.FunctionType{
	Parameters: []*sema.Parameter{
		{
			Label:      sema.ArgumentLabelNotRequired,
			Identifier: "value",
			TypeAnnotation: sema.NewTypeAnnotation(
				&sema.AnyStructType{},
			),
		},
	},
	ReturnTypeAnnotation: sema.NewTypeAnnotation(
		sema.VoidType,
	),
}

var getCurrentBlockFunctionType = &sema.FunctionType{
	ReturnTypeAnnotation: sema.NewTypeAnnotation(&BlockType{}),
}

var getBlockFunctionType = &sema.FunctionType{
	Parameters: []*sema.Parameter{
		{
			Label:      "at",
			Identifier: "height",
			TypeAnnotation: sema.NewTypeAnnotation(
				&sema.UInt64Type{},
			),
		},
	},
	ReturnTypeAnnotation: sema.NewTypeAnnotation(
		&sema.OptionalType{
			Type: &BlockType{},
		},
	),
}

var unsafeRandomFunctionType = &sema.FunctionType{
	ReturnTypeAnnotation: sema.NewTypeAnnotation(
		&sema.UInt64Type{},
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
			accountFunctionType,
			impls.CreateAccount,
		),
		NewStandardLibraryFunction(
			"getAccount",
			getAccountFunctionType,
			impls.GetAccount,
		),
		NewStandardLibraryFunction(
			"log",
			logFunctionType,
			impls.Log,
		),
		NewStandardLibraryFunction(
			"getCurrentBlock",
			getCurrentBlockFunctionType,
			impls.GetCurrentBlock,
		),
		NewStandardLibraryFunction(
			"getBlock",
			getBlockFunctionType,
			impls.GetBlock,
		),
		NewStandardLibraryFunction(
			"unsafeRandom",
			unsafeRandomFunctionType,
			impls.UnsafeRandom,
		),
	}
}

func DefaultFlowBuiltinImpls() FlowBuiltinImpls {
	return FlowBuiltinImpls{
		CreateAccount: func(invocation interpreter.Invocation) trampoline.Trampoline {
			panic(fmt.Errorf("cannot create accounts"))
		},
		GetAccount: func(invocation interpreter.Invocation) trampoline.Trampoline {
			panic(fmt.Errorf("cannot get accounts"))
		},
		Log: LogFunction.Function.Function,
		GetCurrentBlock: func(invocation interpreter.Invocation) trampoline.Trampoline {
			panic(fmt.Errorf("cannot get blocks"))
		},
		GetBlock: func(invocation interpreter.Invocation) trampoline.Trampoline {
			panic(fmt.Errorf("cannot get blocks"))
		},
		UnsafeRandom: func(invocation interpreter.Invocation) trampoline.Trampoline {
			return trampoline.Done{Result: interpreter.UInt64Value(rand.Uint64())}
		},
	}
}

// Flow location

type FlowLocation struct{}

const FlowLocationPrefix = "flow"

func (l FlowLocation) ID() common.LocationID {
	return common.NewLocationID(FlowLocationPrefix)
}

func (l FlowLocation) TypeID(qualifiedIdentifier string) common.TypeID {
	return common.NewTypeID(
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
		func(typeID string) (location common.Location, qualifiedIdentifier string, err error) {
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
		Members:    map[string]*sema.Member{},
	}

	for _, parameter := range parameters {

		eventType.Fields = append(eventType.Fields,
			parameter.Identifier,
		)

		eventType.Members[parameter.Identifier] =
			sema.NewPublicConstantFieldMember(
				eventType,
				parameter.Identifier,
				parameter.TypeAnnotation.Type,
				// TODO: add docstring support for parameters
				"",
			)

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
	Type: &sema.UInt8Type{},
}

var TypeIDsType = &sema.VariableSizedType{
	Type: &sema.StringType{},
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
		&sema.VariableSizedType{
			Type: &sema.UInt8Type{},
		},
	),
}

var AccountEventContractsParameter = &sema.Parameter{
	Identifier:     "contracts",
	TypeAnnotation: sema.NewTypeAnnotation(TypeIDsType),
}

var AccountEventContractParameter = &sema.Parameter{
	Identifier:     "contract",
	TypeAnnotation: sema.NewTypeAnnotation(&sema.StringType{}),
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

// BlockType

type BlockType struct{}

func (*BlockType) IsType() {}

func (*BlockType) String() string {
	return "Block"
}

func (*BlockType) QualifiedString() string {
	return "Block"
}

func (*BlockType) ID() sema.TypeID {
	return "Block"
}

func (*BlockType) Equal(other sema.Type) bool {
	_, ok := other.(*BlockType)
	return ok
}

func (*BlockType) IsResourceType() bool {
	return false
}

func (*BlockType) TypeAnnotationState() sema.TypeAnnotationState {
	return sema.TypeAnnotationStateValid
}

func (*BlockType) IsInvalidType() bool {
	return false
}

func (*BlockType) IsStorable(_ map[*sema.Member]bool) bool {
	return false
}

func (*BlockType) IsExternallyReturnable(_ map[*sema.Member]bool) bool {
	return false
}

func (*BlockType) IsEquatable() bool {
	// TODO:
	return false
}

func (t *BlockType) RewriteWithRestrictedTypes() (sema.Type, bool) {
	return t, false
}

const BlockIDSize = 32

var blockIDFieldType = &sema.ConstantSizedType{
	Type: &sema.UInt8Type{},
	Size: BlockIDSize,
}

const blockTypeHeightFieldDocString = `
The height of the block.

If the blockchain is viewed as a tree with the genesis block at the root, the height of a node is the number of edges between the node and the genesis block
`

const blockTypeViewFieldDocString = `
The view of the block.

It is a detail of the consensus algorithm. It is a monotonically increasing integer and counts rounds in the consensus algorithm. Since not all rounds result in a finalized block, the view number is strictly greater than or equal to the block height
`

const blockTypeTimestampFieldDocString = `
The ID of the block.

It is essentially the hash of the block
`

const blockTypeIdFieldDocString = `
The timestamp of the block.

It is the local clock time of the block proposer when it generates the block
`

func (t *BlockType) GetMembers() map[string]sema.MemberResolver {
	return map[string]sema.MemberResolver{
		"height": {
			Kind: common.DeclarationKindField,
			Resolve: func(identifier string, _ ast.Range, _ func(error)) *sema.Member {
				return sema.NewPublicConstantFieldMember(
					t,
					identifier,
					&sema.UInt64Type{},
					blockTypeHeightFieldDocString,
				)
			},
		},
		"view": {
			Kind: common.DeclarationKindField,
			Resolve: func(identifier string, _ ast.Range, _ func(error)) *sema.Member {
				return sema.NewPublicConstantFieldMember(
					t,
					identifier,
					&sema.UInt64Type{},
					blockTypeViewFieldDocString,
				)
			},
		},
		"timestamp": {
			Kind: common.DeclarationKindField,
			Resolve: func(identifier string, _ ast.Range, _ func(error)) *sema.Member {
				return sema.NewPublicConstantFieldMember(
					t,
					identifier,
					&sema.UFix64Type{},
					blockTypeTimestampFieldDocString,
				)
			},
		},
		"id": {
			Kind: common.DeclarationKindField,
			Resolve: func(identifier string, _ ast.Range, _ func(error)) *sema.Member {
				return sema.NewPublicConstantFieldMember(
					t,
					identifier,
					blockIDFieldType,
					blockTypeIdFieldDocString,
				)
			},
		},
	}
}

func (t *BlockType) Unify(_ sema.Type, _ map[*sema.TypeParameter]sema.Type, _ func(err error), _ ast.Range) bool {
	return false
}

func (t *BlockType) Resolve(_ map[*sema.TypeParameter]sema.Type) sema.Type {
	return t
}

var FlowBuiltInTypes = StandardLibraryTypes{
	StandardLibraryType{
		Name: "Block",
		Type: &BlockType{},
		Kind: common.DeclarationKindType,
	},
}
