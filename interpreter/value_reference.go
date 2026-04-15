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
	ReferencedValue(context ValueStaticTypeContext, errorOnFailedDereference bool) *Value
	BorrowType() sema.Type
	// WithAuthorizationAndBorrowedType returns a new reference of the same kind,
	// but with the given authorization and borrowed type.
	WithAuthorizationAndBorrowedType(
		context ReferenceCreationContext,
		auth Authorization,
		borrowedType sema.Type,
	) ReferenceValue
}

func DereferenceValue(
	context ValueTransferContext,
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

	referencedValue := *referenceValue.ReferencedValue(context, true)

	// Defensive check: ensure that the referenced value is not a resource
	if referencedValue.IsResourceKinded(context) {
		panic(&ResourceReferenceDereferenceError{})
	}

	transferredDereferencedValue := referencedValue.Transfer(
		context,
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

func getReferenceValueMember(
	context MemberAccessibleContext,
	v ReferenceValue,
	referencedValue Value,
	name string,
) FunctionValue {

	switch referencedValue := referencedValue.(type) {
	case *ArrayValue:
		refType := context.SemaTypeFromStaticType(v.StaticType(context))

		arrayType := referencedValue.SemaType(context)

		switch name {
		case sema.ArrayTypeFilterFunctionName:
			return NewBoundHostFunctionValue(
				context,
				v,
				sema.ArrayFilterFunctionType(
					context,
					refType,
					arrayType.ElementType(false),
					func(err error) {
						// TODO:
						panic(err)
					},
				),
				NativeArrayFilterFunction,
			)

		case sema.ArrayTypeMapFunctionName:
			return NewBoundHostFunctionValue(
				context,
				v,
				sema.ArrayMapFunctionType(
					context,
					refType,
					arrayType,
					func(err error) {
						// TODO:
						panic(err)
					},
				),
				NativeArrayMapFunction,
			)
		}
	}

	return nil
}

func getReferenceValueMethod(
	context MemberAccessibleContext,
	v ReferenceValue,
	referencedValue Value,
	name string,
) FunctionValue {
	method := getReferenceValueMember(context, v, referencedValue, name)
	if method != nil {
		return method
	}
	return getBuiltinFunctionMember(context, referencedValue, name)
}
