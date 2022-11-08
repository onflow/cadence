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
	"strings"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/sema"
)

// Flow location

type FlowLocation struct{}

var _ common.Location = FlowLocation{}

const FlowLocationPrefix = "flow"

func (l FlowLocation) TypeID(memoryGauge common.MemoryGauge, qualifiedIdentifier string) common.TypeID {
	var i int

	// FlowLocationPrefix '.' qualifiedIdentifier
	length := len(FlowLocationPrefix) + 1 + len(qualifiedIdentifier)

	common.UseMemory(memoryGauge, common.NewRawStringMemoryUsage(length))

	b := make([]byte, length)

	copy(b, FlowLocationPrefix)
	i += len(FlowLocationPrefix)

	b[i] = '.'
	i += 1

	copy(b[i:], qualifiedIdentifier)

	return common.TypeID(b)
}

func (l FlowLocation) QualifiedIdentifier(typeID common.TypeID) string {
	pieces := strings.SplitN(string(typeID), ".", 2)

	if len(pieces) < 2 {
		return ""
	}

	return pieces[1]
}

func (l FlowLocation) String() string {
	return FlowLocationPrefix
}

func (l FlowLocation) Description() string {
	return FlowLocationPrefix
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
		return FlowLocation{}, "", errors.NewDefaultUserError("%s: %s", errorMessagePrefix, message)
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
		return FlowLocation{}, "", errors.NewDefaultUserError(
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

func newFlowEventType(identifier string, parameters ...sema.Parameter) *sema.CompositeType {

	eventType := &sema.CompositeType{
		Kind:       common.CompositeKindEvent,
		Location:   FlowLocation{},
		Identifier: identifier,
		Fields:     []string{},
		Members:    &sema.StringMemberOrderedMap{},
	}

	for _, parameter := range parameters {

		eventType.Fields = append(
			eventType.Fields,
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

var AccountEventAddressParameter = sema.Parameter{
	Identifier:     "address",
	TypeAnnotation: sema.NewTypeAnnotation(&sema.AddressType{}),
}

var AccountEventCodeHashParameter = sema.Parameter{
	Identifier:     "codeHash",
	TypeAnnotation: sema.NewTypeAnnotation(HashType),
}

var AccountEventPublicKeyParameter = sema.Parameter{
	Identifier: "publicKey",
	TypeAnnotation: sema.NewTypeAnnotation(
		sema.ByteArrayType,
	),
}

var AccountEventContractParameter = sema.Parameter{
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

var AccountEventProviderParameter = sema.Parameter{
	Identifier:     "provider",
	TypeAnnotation: sema.NewTypeAnnotation(&sema.AddressType{}),
}

var AccountEventRecipientParameter = sema.Parameter{
	Identifier:     "recipient",
	TypeAnnotation: sema.NewTypeAnnotation(&sema.AddressType{}),
}

var AccountEventNameParameter = sema.Parameter{
	Identifier:     "name",
	TypeAnnotation: sema.NewTypeAnnotation(sema.StringType),
}

var AccountEventTypeParameter = sema.Parameter{
	Identifier:     "type",
	TypeAnnotation: sema.NewTypeAnnotation(sema.MetaType),
}

var AccountInboxPublishedEventType = newFlowEventType(
	"InboxValuePublished",
	AccountEventProviderParameter,
	AccountEventRecipientParameter,
	AccountEventNameParameter,
	AccountEventTypeParameter,
)

var AccountInboxUnpublishedEventType = newFlowEventType(
	"InboxValueUnpublished",
	AccountEventProviderParameter,
	AccountEventNameParameter,
)

var AccountInboxClaimedEventType = newFlowEventType(
	"InboxValueClaimed",
	AccountEventProviderParameter,
	AccountEventRecipientParameter,
	AccountEventNameParameter,
)
