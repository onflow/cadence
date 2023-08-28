// Code generated from bls.cdc. DO NOT EDIT.
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

package stdlib

import (
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/sema"
)

const BLSTypeAggregateSignaturesFunctionName = "aggregateSignatures"

var BLSTypeAggregateSignaturesFunctionType = &sema.FunctionType{
	Parameters: []sema.Parameter{
		{
			Label:      sema.ArgumentLabelNotRequired,
			Identifier: "signatures",
			TypeAnnotation: sema.NewTypeAnnotation(&sema.VariableSizedType{
				Type: &sema.VariableSizedType{
					Type: UInt8Type,
				},
			}),
		},
	},
	ReturnTypeAnnotation: sema.NewTypeAnnotation(
		&sema.OptionalType{
			Type: &sema.VariableSizedType{
				Type: UInt8Type,
			},
		},
	),
}

const BLSTypeAggregateSignaturesFunctionDocString = `
Aggregates multiple BLS signatures into one,
considering the proof of possession as a defense against rogue attacks.

Signatures could be generated from the same or distinct messages,
they could also be the aggregation of other signatures.
The order of the signatures in the slice does not matter since the aggregation is commutative.
No subgroup membership check is performed on the input signatures.
The function returns nil if the array is empty or if decoding one of the signature fails.
`

const BLSTypeAggregatePublicKeysFunctionName = "aggregatePublicKeys"

var BLSTypeAggregatePublicKeysFunctionType = &sema.FunctionType{
	Parameters: []sema.Parameter{
		{
			Label:      sema.ArgumentLabelNotRequired,
			Identifier: "keys",
			TypeAnnotation: sema.NewTypeAnnotation(&sema.VariableSizedType{
				Type: PublicKeyType,
			}),
		},
	},
	ReturnTypeAnnotation: sema.NewTypeAnnotation(
		&sema.OptionalType{
			Type: PublicKeyType,
		},
	),
}

const BLSTypeAggregatePublicKeysFunctionDocString = `
Aggregates multiple BLS public keys into one.

The order of the public keys in the slice does not matter since the aggregation is commutative.
No subgroup membership check is performed on the input keys.
The function returns nil if the array is empty or any of the input keys is not a BLS key.
`

const BLSTypeName = "BLS"

var BLSType = func() *sema.CompositeType {
	var t = &sema.CompositeType{
		Identifier:         BLSTypeName,
		Kind:               common.CompositeKindContract,
		ImportableBuiltin:  false,
		HasComputedMembers: true,
	}

	return t
}()

func init() {
	var members = []*sema.Member{
		sema.NewUnmeteredFunctionMember(
			BLSType,
			ast.AccessPublic,
			BLSTypeAggregateSignaturesFunctionName,
			BLSTypeAggregateSignaturesFunctionType,
			BLSTypeAggregateSignaturesFunctionDocString,
		),
		sema.NewUnmeteredFunctionMember(
			BLSType,
			ast.AccessPublic,
			BLSTypeAggregatePublicKeysFunctionName,
			BLSTypeAggregatePublicKeysFunctionType,
			BLSTypeAggregatePublicKeysFunctionDocString,
		),
	}

	BLSType.Members = sema.MembersAsMap(members)
	BLSType.Fields = sema.MembersFieldNames(members)
}
