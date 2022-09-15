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

package interpreter

import (
	"fmt"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/sema"
)

// AuthAccountInbox

var authAccountInboxTypeID = sema.AuthAccountInboxType.ID()
var authAccountInboxStaticType StaticType = PrimitiveStaticTypeAuthAccountKeys

// NewAuthAccountInboxValue constructs a AuthAccount.Inbox value.
func NewAuthAccountInboxValue(
	gauge common.MemoryGauge,
	address AddressValue,
	allowlist Value,
	permitFunction FunctionValue,
	unpermitFunction FunctionValue,
	publishFunction FunctionValue,
	unpublishFunction FunctionValue,
	claimFunction FunctionValue,
) Value {

	fields := map[string]Value{
		sema.AuthAccountInboxAllowlistField: allowlist,
		sema.AuthAccountInboxPermitField:    permitFunction,
		sema.AuthAccountInboxUnpermitField:  unpermitFunction,
		sema.AuthAccountInboxPublishField:   publishFunction,
		sema.AuthAccountInboxUnpublishField: unpublishFunction,
		sema.AuthAccountInboxClaimField:     claimFunction,
	}

	var str string
	stringer := func(memoryGauge common.MemoryGauge, _ SeenReferences) string {
		if str == "" {
			common.UseMemory(memoryGauge, common.AuthAccountInboxStringMemoryUsage)
			addressStr := address.MeteredString(memoryGauge, SeenReferences{})
			str = fmt.Sprintf("AuthAccount.Inbox(%s)", addressStr)
		}
		return str
	}

	return NewSimpleCompositeValue(
		gauge,
		authAccountInboxTypeID,
		authAccountInboxStaticType,
		nil,
		fields,
		nil,
		nil,
		stringer,
	)
}

// PublicAccountInbox

var publicAccountInboxTypeID = sema.PublicAccountInboxType.ID()
var publicAccountInboxStaticType StaticType = PrimitiveStaticTypePublicAccountKeys

// NewPublicAccountInboxValue constructs a PublicAccount.Inbox value.
func NewPublicAccountInboxValue(
	gauge common.MemoryGauge,
	address AddressValue,
	allowlist Value,
) Value {

	fields := map[string]Value{
		sema.PublicAccountInboxAllowlistField: allowlist,
	}

	var str string
	stringer := func(memoryGauge common.MemoryGauge, _ SeenReferences) string {
		if str == "" {
			common.UseMemory(memoryGauge, common.PublicAccountInboxStringMemoryUsage)
			addressStr := address.MeteredString(memoryGauge, SeenReferences{})
			str = fmt.Sprintf("PublicAccount.Inbox(%s)", addressStr)
		}
		return str
	}

	return NewSimpleCompositeValue(
		gauge,
		publicAccountInboxTypeID,
		publicAccountInboxStaticType,
		nil,
		fields,
		nil,
		nil,
		stringer,
	)
}
