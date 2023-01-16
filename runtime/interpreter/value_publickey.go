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
	"github.com/onflow/atree"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/sema"
)

// PublicKeyValidationHandlerFunc is a function that validates a given public key.
// Parameter types:
// - publicKey: PublicKey
type PublicKeyValidationHandlerFunc func(
	interpreter *Interpreter,
	locationRange LocationRange,
	publicKey *CompositeValue,
) error

// NewPublicKeyValue constructs a PublicKey value.
func NewPublicKeyValue(
	interpreter *Interpreter,
	locationRange LocationRange,
	publicKey *ArrayValue,
	signAlgo Value,
	validatePublicKey PublicKeyValidationHandlerFunc,
	publicKeyVerifySignatureFunction FunctionValue,
	publicKeyVerifyPoPFunction FunctionValue,
) *CompositeValue {

	fields := []CompositeField{
		{
			Name:  sema.PublicKeyTypeSignAlgoFieldName,
			Value: signAlgo,
		},
	}

	// TODO: refactor to SimpleCompositeValue
	publicKeyValue := NewCompositeValue(
		interpreter,
		locationRange,
		sema.PublicKeyType.Location,
		sema.PublicKeyType.QualifiedIdentifier(),
		sema.PublicKeyType.Kind,
		fields,
		common.NilAddress,
	)

	publicKeyValue.ComputedFields = map[string]ComputedField{
		sema.PublicKeyTypePublicKeyFieldName: func(interpreter *Interpreter, locationRange LocationRange) Value {
			return publicKey.Transfer(interpreter, locationRange, atree.Address{}, false, nil)
		},
	}
	publicKeyValue.Functions = map[string]FunctionValue{
		sema.PublicKeyTypeVerifyFunctionName:    publicKeyVerifySignatureFunction,
		sema.PublicKeyTypeVerifyPoPFunctionName: publicKeyVerifyPoPFunction,
	}

	err := validatePublicKey(interpreter, locationRange, publicKeyValue)
	if err != nil {
		panic(InvalidPublicKeyError{
			PublicKey:     publicKey,
			Err:           err,
			LocationRange: locationRange,
		})
	}

	// Public key value to string should include the key even though it is a computed field
	publicKeyValue.Stringer = func(
		memoryGauge common.MemoryGauge,
		publicKeyValue *CompositeValue,
		seenReferences SeenReferences,
	) string {

		stringerFields := []CompositeField{
			{
				Name:  sema.PublicKeyTypePublicKeyFieldName,
				Value: publicKey,
			},
			{
				Name: sema.PublicKeyTypeSignAlgoFieldName,
				// TODO: provide proper location range
				Value: publicKeyValue.GetField(interpreter, EmptyLocationRange, sema.PublicKeyTypeSignAlgoFieldName),
			},
		}

		return formatComposite(
			memoryGauge,
			string(publicKeyValue.TypeID()),
			stringerFields,
			seenReferences,
		)
	}

	return publicKeyValue
}
