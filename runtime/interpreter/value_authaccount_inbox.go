/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Dapper Labs, Inc.
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

// AuthAccount.Inbox

var authAccountInboxTypeID = sema.AuthAccountInboxType.ID()
var authAccountInboxStaticType StaticType = PrimitiveStaticTypeAuthAccountInbox

// NewAuthAccountInboxValue constructs a AuthAccount.Inbox value.
func NewAuthAccountInboxValue(
	gauge common.MemoryGauge,
	addressValue AddressValue,
	publishFunction FunctionValue,
	unpublishFunction FunctionValue,
	claimFunction FunctionValue,
) Value {

	fields := map[string]Value{
		sema.AuthAccountInboxTypePublishFunctionName:   publishFunction,
		sema.AuthAccountInboxTypeUnpublishFunctionName: unpublishFunction,
		sema.AuthAccountInboxTypeClaimFunctionName:     claimFunction,
	}

	var str string
	stringer := func(memoryGauge common.MemoryGauge, seenReferences SeenReferences) string {
		if str == "" {
			common.UseMemory(memoryGauge, common.AuthAccountInboxStringMemoryUsage)
			addressStr := addressValue.MeteredString(memoryGauge, seenReferences)
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
