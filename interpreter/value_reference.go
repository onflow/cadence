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

	"github.com/onflow/cadence/errors"
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
	context ValueTransferContext,
	locationRange LocationRange,
	value Value,
) Value {
	if _, ok := value.(NilValue); ok {
		return Nil
	}

	var isOptional bool

	if someValue, ok := value.(*SomeValue); ok {
		isOptional = true
		value = someValue.InnerValue()
	}

	referenceValue, ok := value.(ReferenceValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	referencedValue := *referenceValue.ReferencedValue(context, locationRange, true)

	// Defensive check: ensure that the referenced value is not a resource
	if referencedValue.IsResourceKinded(context) {
		panic(&ResourceReferenceDereferenceError{
			LocationRange: locationRange,
		})
	}

	transferredDereferencedValue := referencedValue.Transfer(
		context,
		locationRange,
		atree.Address{},
		false,
		nil,
		nil,
		false,
	)

	if isOptional {
		return NewSomeValueNonCopying(context, transferredDereferencedValue)
	} else {
		return transferredDereferencedValue
	}
}
