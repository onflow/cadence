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
	"github.com/onflow/atree"

	"github.com/onflow/cadence/sema"
)

type ReferenceValue interface {
	Value
	AuthorizedValue
	isReference()
	ReferencedValue(context ValueStaticTypeContext, locationRange LocationRange, errorOnFailedDereference bool) *Value
	BorrowType() sema.Type
}

func DereferenceValue(
	inter *Interpreter,
	locationRange LocationRange,
	referenceValue ReferenceValue,
) Value {
	referencedValue := *referenceValue.ReferencedValue(inter, locationRange, true)

	// Defensive check: ensure that the referenced value is not a resource
	if referencedValue.IsResourceKinded(inter) {
		panic(ResourceReferenceDereferenceError{
			LocationRange: locationRange,
		})
	}

	return referencedValue.Transfer(
		inter,
		locationRange,
		atree.Address{},
		false,
		nil,
		nil,
		false,
	)
}
