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

package stdlib

import (
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
)

var hashAlgorithmStaticType interpreter.StaticType = interpreter.ConvertSemaCompositeTypeToStaticCompositeType(
	nil,
	sema.HashAlgorithmType,
)

type Hasher interface {
	// Hash returns the digest of hashing the given data with using the given hash algorithm
	Hash(data []byte, tag string, algorithm sema.HashAlgorithm) ([]byte, error)
}

func NewHashAlgorithmCase(
	rawValue interpreter.UInt8Value,
	hasher Hasher,
) (
	interpreter.MemberAccessibleValue,
	error,
) {
	if !sema.HashAlgorithm(rawValue).IsValid() {
		return nil, errors.NewDefaultUserError(
			"unknown HashAlgorithm with rawValue %d",
			rawValue,
		)
	}

	value := interpreter.NewSimpleCompositeValue(
		nil,
		sema.HashAlgorithmType.ID(),
		hashAlgorithmStaticType,
		[]string{sema.EnumRawValueFieldName},
		nil,
		nil,
		nil,
		nil,
		nil,
	)
	value.Fields = map[string]interpreter.Value{
		sema.EnumRawValueFieldName:                    rawValue,
		sema.HashAlgorithmTypeHashFunctionName:        newHashAlgorithmHashFunction(value, hasher),
		sema.HashAlgorithmTypeHashWithTagFunctionName: newHashAlgorithmHashWithTagFunction(value, hasher),
	}
	return value, nil
}

func newHashAlgorithmHashFunction(
	hashAlgoValue interpreter.MemberAccessibleValue,
	hasher Hasher,
) *interpreter.HostFunctionValue {
	// TODO: should ideally create a bound-host function.
	// But the interpreter is not available at this point.
	return interpreter.NewUnmeteredStaticHostFunctionValue(
		sema.HashAlgorithmTypeHashFunctionType,
		func(invocation interpreter.Invocation) interpreter.Value {
			dataValue, ok := invocation.Arguments[0].(*interpreter.ArrayValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			context := invocation.InvocationContext

			locationRange := invocation.LocationRange

			return hash(
				context,
				locationRange,
				hasher,
				dataValue,
				nil,
				hashAlgoValue,
			)
		},
	)
}

func newHashAlgorithmHashWithTagFunction(
	hashAlgorithmValue interpreter.MemberAccessibleValue,
	hasher Hasher,
) *interpreter.HostFunctionValue {
	// TODO: should ideally create a bound-host function.
	// But the interpreter is not available at this point.
	return interpreter.NewUnmeteredStaticHostFunctionValue(
		sema.HashAlgorithmTypeHashWithTagFunctionType,
		func(invocation interpreter.Invocation) interpreter.Value {

			dataValue, ok := invocation.Arguments[0].(*interpreter.ArrayValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			tagValue, ok := invocation.Arguments[1].(*interpreter.StringValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			inter := invocation.InvocationContext

			locationRange := invocation.LocationRange

			return hash(
				inter,
				locationRange,
				hasher,
				dataValue,
				tagValue,
				hashAlgorithmValue,
			)
		},
	)
}

func hash(
	context interpreter.MemberAccessibleContext,
	locationRange interpreter.LocationRange,
	hasher Hasher,
	dataValue *interpreter.ArrayValue,
	tagValue *interpreter.StringValue,
	hashAlgorithmValue interpreter.MemberAccessibleValue,
) interpreter.Value {
	data, err := interpreter.ByteArrayValueToByteSlice(context, dataValue, locationRange)
	if err != nil {
		panic(errors.NewUnexpectedError("failed to get data. %w", err))
	}

	var tag string
	if tagValue != nil {
		tag = tagValue.Str
	}

	hashAlgorithm := NewHashAlgorithmFromValue(context, locationRange, hashAlgorithmValue)

	result, err := hasher.Hash(data, tag, hashAlgorithm)
	if err != nil {
		panic(err)
	}
	return interpreter.ByteSliceToByteArrayValue(context, result)
}

func NewHashAlgorithmConstructor(hasher Hasher) StandardLibraryValue {

	enumLookupType := cryptoAlgorithmEnumLookupType(
		sema.HashAlgorithmType,
		sema.HashAlgorithms,
	)

	hashAlgorithmConstructorValue, _ := cryptoAlgorithmEnumValueAndCaseValues(
		enumLookupType,
		sema.HashAlgorithms,
		func(rawValue interpreter.UInt8Value) interpreter.MemberAccessibleValue {
			// Assume rawValues are all valid, given we iterate over sema.HashAlgorithms
			caseValue, _ := NewHashAlgorithmCase(rawValue, hasher)
			return caseValue
		},
	)

	return StandardLibraryValue{
		Name:  sema.HashAlgorithmTypeName,
		Type:  enumLookupType,
		Value: hashAlgorithmConstructorValue,
		Kind:  common.DeclarationKindEnum,
	}
}
