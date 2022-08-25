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
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
)

var blsContractType = func() *sema.CompositeType {
	ty := &sema.CompositeType{
		Identifier: "BLS",
		Kind:       common.CompositeKindContract,
	}

	ty.Members = sema.GetMembersAsMap([]*sema.Member{
		sema.NewUnmeteredPublicFunctionMember(
			ty,
			blsAggregatePublicKeysFunctionName,
			blsAggregatePublicKeysFunctionType,
			blsAggregatePublicKeysFunctionDocString,
		),
		sema.NewUnmeteredPublicFunctionMember(
			ty,
			blsAggregateSignaturesFunctionName,
			blsAggregateSignaturesFunctionType,
			blsAggregateSignaturesFunctionDocString,
		),
	})
	return ty
}()

var blsContractTypeID = blsContractType.ID()
var blsContractStaticType interpreter.StaticType = interpreter.CompositeStaticType{
	QualifiedIdentifier: blsContractType.Identifier,
	TypeID:              blsContractTypeID,
}

const blsAggregateSignaturesFunctionDocString = `
Aggregates multiple BLS signatures into one,
considering the proof of possession as a defense against rogue attacks.

Signatures could be generated from the same or distinct messages,
they could also be the aggregation of other signatures.
The order of the signatures in the slice does not matter since the aggregation is commutative.
No subgroup membership check is performed on the input signatures.
The function returns nil if the array is empty or if decoding one of the signature fails.
`

const blsAggregateSignaturesFunctionName = "aggregateSignatures"

var blsAggregateSignaturesFunctionType = &sema.FunctionType{
	Purity: sema.ViewFunction,
	Parameters: []*sema.Parameter{
		{
			Label:      sema.ArgumentLabelNotRequired,
			Identifier: "signatures",
			TypeAnnotation: sema.NewTypeAnnotation(
				sema.ByteArrayArrayType,
			),
		},
	},
	ReturnTypeAnnotation: sema.NewTypeAnnotation(
		&sema.OptionalType{
			Type: sema.ByteArrayType,
		},
	),
}

const blsAggregatePublicKeysFunctionDocString = `
Aggregates multiple BLS public keys into one.

The order of the public keys in the slice does not matter since the aggregation is commutative.
No subgroup membership check is performed on the input keys.
The function returns nil if the array is empty or any of the input keys is not a BLS key.
`

const blsAggregatePublicKeysFunctionName = "aggregatePublicKeys"

var blsAggregatePublicKeysFunctionType = &sema.FunctionType{
	Purity: sema.ViewFunction,
	Parameters: []*sema.Parameter{
		{
			Label:      sema.ArgumentLabelNotRequired,
			Identifier: "keys",
			TypeAnnotation: sema.NewTypeAnnotation(
				sema.PublicKeyArrayType,
			),
		},
	},
	ReturnTypeAnnotation: sema.NewTypeAnnotation(
		&sema.OptionalType{
			Type: sema.PublicKeyType,
		},
	),
}

var blsAggregatePublicKeysFunction = interpreter.NewUnmeteredHostFunctionValue(
	func(invocation interpreter.Invocation) interpreter.Value {
		publicKeys, ok := invocation.Arguments[0].(*interpreter.ArrayValue)
		if !ok {
			panic(errors.NewUnreachableError())
		}

		inter := invocation.Interpreter
		getLocationRange := invocation.GetLocationRange

		inter.ExpectType(
			publicKeys,
			sema.PublicKeyArrayType,
			getLocationRange,
		)

		return invocation.Interpreter.BLSAggregatePublicKeysHandler(
			inter,
			getLocationRange,
			publicKeys,
		)
	},
	blsAggregatePublicKeysFunctionType,
)

var blsAggregateSignaturesFunction = interpreter.NewUnmeteredHostFunctionValue(
	func(invocation interpreter.Invocation) interpreter.Value {
		signatures, ok := invocation.Arguments[0].(*interpreter.ArrayValue)
		if !ok {
			panic(errors.NewUnreachableError())
		}

		inter := invocation.Interpreter
		getLocationRange := invocation.GetLocationRange

		inter.ExpectType(
			signatures,
			sema.ByteArrayArrayType,
			getLocationRange,
		)

		return inter.BLSAggregateSignaturesHandler(
			inter,
			getLocationRange,
			signatures,
		)
	},
	blsAggregateSignaturesFunctionType,
)

var blsContractFields = map[string]interpreter.Value{
	blsAggregatePublicKeysFunctionName: blsAggregatePublicKeysFunction,
	blsAggregateSignaturesFunctionName: blsAggregateSignaturesFunction,
}

var blsContract = StandardLibraryValue{
	Name: "BLS",
	Type: blsContractType,
	ValueFactory: func(inter *interpreter.Interpreter) interpreter.Value {
		return interpreter.NewSimpleCompositeValue(
			inter,
			blsContractType.ID(),
			blsContractStaticType,
			nil,
			blsContractFields,
			nil,
			nil,
			nil,
		)
	},
	Kind: common.DeclarationKindContract,
}
