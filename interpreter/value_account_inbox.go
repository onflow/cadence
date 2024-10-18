/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Flow Foundation
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

	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/sema"
)

// Account.Inbox

var account_InboxTypeID = sema.Account_InboxType.ID()
var account_InboxStaticType StaticType = PrimitiveStaticTypeAccount_Inbox

// NewAccountInboxValue constructs an Account.Inbox value.
func NewAccountInboxValue(
	gauge common.MemoryGauge,
	addressValue AddressValue,
	publishFunction BoundFunctionGenerator,
	unpublishFunction BoundFunctionGenerator,
	claimFunction BoundFunctionGenerator,
) Value {

	var str string
	stringer := func(interpreter *Interpreter, seenReferences SeenReferences, locationRange LocationRange) string {
		if str == "" {
			common.UseMemory(interpreter, common.AccountInboxStringMemoryUsage)
			addressStr := addressValue.MeteredString(interpreter, seenReferences, locationRange)
			str = fmt.Sprintf("Account.Inbox(%s)", addressStr)
		}
		return str
	}

	accountInbox := NewSimpleCompositeValue(
		gauge,
		account_InboxTypeID,
		account_InboxStaticType,
		nil,
		nil,
		nil,
		nil,
		stringer,
	)

	accountInbox.Fields = map[string]Value{
		sema.Account_InboxTypePublishFunctionName:   publishFunction(accountInbox),
		sema.Account_InboxTypeUnpublishFunctionName: unpublishFunction(accountInbox),
		sema.Account_InboxTypeClaimFunctionName:     claimFunction(accountInbox),
	}

	return accountInbox
}
