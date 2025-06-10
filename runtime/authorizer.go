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

package runtime

import (
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
)

func newAccountReferenceValueFromAddress(
	context interpreter.AccountCreationContext,
	address common.Address,
	environment Environment,
	authorization sema.Access,
	locationRange interpreter.LocationRange,
) *interpreter.EphemeralReferenceValue {
	addressValue := interpreter.NewAddressValue(context, address)

	accountValue := environment.newAccountValue(context, addressValue)

	staticAuthorization := interpreter.ConvertSemaAccessToStaticAuthorization(
		context,
		authorization,
	)

	accountReferenceValue := interpreter.NewEphemeralReferenceValue(
		context,
		staticAuthorization,
		accountValue,
		sema.AccountType,
		locationRange,
	)
	return accountReferenceValue
}

func authorizerValues(
	environment Environment,
	context interpreter.AccountCreationContext,
	addresses []Address,
	parameters []sema.Parameter,
) []interpreter.Value {

	// gather authorizers

	authorizerValues := make([]interpreter.Value, 0, len(addresses))

	for i, address := range addresses {
		parameter := parameters[i]

		referenceType, ok := parameter.TypeAnnotation.Type.(*sema.ReferenceType)
		if !ok || referenceType.Type != sema.AccountType {
			panic(errors.NewUnreachableError())
		}

		accountReferenceValue := newAccountReferenceValueFromAddress(
			context,
			address,
			environment,
			referenceType.Authorization,
			// okay to pass an empty range here because the account value is never a reference, so this can't fail
			interpreter.EmptyLocationRange,
		)

		authorizerValues = append(authorizerValues, accountReferenceValue)
	}

	return authorizerValues
}
