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
	"github.com/onflow/cadence/bbq"
	"github.com/onflow/cadence/bbq/commons"
	"github.com/onflow/cadence/bbq/vm"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
)

var hashAlgorithmLookupType = cryptoAlgorithmEnumLookupType(
	sema.HashAlgorithmType,
	sema.HashAlgorithms,
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
		sema.HashAlgorithmTypeHashFunctionName:        newInterpreterHashAlgorithmHashFunction(value, hasher),
		sema.HashAlgorithmTypeHashWithTagFunctionName: newInterpreterHashAlgorithmHashWithTagFunction(value, hasher),
	}
	return value, nil
}

func newInterpreterHashAlgorithmHashFunction(
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

func NewVMHashAlgorithmHashFunction(
	hasher Hasher,
) VMFunction {
	return VMFunction{
		BaseType: sema.HashAlgorithmType,
		FunctionValue: vm.NewNativeFunctionValue(
			sema.HashAlgorithmTypeHashFunctionName,
			sema.HashAlgorithmTypeHashFunctionType,
			func(context *vm.Context, _ []bbq.StaticType, receiver vm.Value, args ...vm.Value) vm.Value {

				hashAlgoValue, ok := receiver.(interpreter.MemberAccessibleValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				dataValue, ok := args[0].(*interpreter.ArrayValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				return hash(
					context,
					vm.EmptyLocationRange,
					hasher,
					dataValue,
					nil,
					hashAlgoValue,
				)
			},
		),
	}
}

func newInterpreterHashAlgorithmHashWithTagFunction(
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

func NewVMHashAlgorithmHashWithTagFunction(
	hasher Hasher,
) VMFunction {
	return VMFunction{
		BaseType: sema.HashAlgorithmType,
		FunctionValue: vm.NewNativeFunctionValue(
			sema.HashAlgorithmTypeHashWithTagFunctionName,
			sema.HashAlgorithmTypeHashWithTagFunctionType,
			func(context *vm.Context, _ []bbq.StaticType, receiver vm.Value, args ...vm.Value) vm.Value {

				hashAlgoValue, ok := receiver.(interpreter.MemberAccessibleValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				dataValue, ok := args[0].(*interpreter.ArrayValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				tagValue, ok := args[1].(*interpreter.StringValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				return hash(
					context,
					vm.EmptyLocationRange,
					hasher,
					dataValue,
					tagValue,
					hashAlgoValue,
				)
			},
		),
	}
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

func NewInterpreterHashAlgorithmConstructor(hasher Hasher) StandardLibraryValue {

	interpreterHashAlgorithmConstructorValue, _ := interpreterCryptoAlgorithmEnumValueAndCaseValues(
		hashAlgorithmLookupType,
		sema.HashAlgorithms,
		func(rawValue interpreter.UInt8Value) interpreter.MemberAccessibleValue {
			// Assume rawValues are all valid, given we iterate over sema.HashAlgorithms
			caseValue, _ := NewHashAlgorithmCase(rawValue, hasher)
			return caseValue
		},
	)

	return StandardLibraryValue{
		Name:  sema.HashAlgorithmTypeName,
		Type:  hashAlgorithmLookupType,
		Value: interpreterHashAlgorithmConstructorValue,
		Kind:  common.DeclarationKindEnum,
	}
}

func NewVMHashAlgorithmConstructor(hasher Hasher) StandardLibraryValue {

	caseCount := len(sema.HashAlgorithms)
	cases := make(map[interpreter.UInt8Value]interpreter.MemberAccessibleValue, caseCount)

	for _, hashAlgorithm := range sema.HashAlgorithms {
		rawValue := interpreter.UInt8Value(hashAlgorithm.RawValue())
		// Assume rawValues are all valid, given we iterate over sema.HashAlgorithms
		caseValue, _ := NewHashAlgorithmCase(rawValue, hasher)
		cases[rawValue] = caseValue
	}

	function := vm.NewNativeFunctionValue(
		sema.HashAlgorithmTypeName,
		hashAlgorithmLookupType,
		func(context *vm.Context, _ []bbq.StaticType, _ vm.Value, args ...vm.Value) vm.Value {
			rawValue := args[0].(interpreter.UInt8Value)

			caseValue, ok := cases[rawValue]
			if !ok {
				return interpreter.Nil
			}

			return interpreter.NewSomeValueNonCopying(context, caseValue)
		},
	)

	return StandardLibraryValue{
		Name:  sema.HashAlgorithmTypeName,
		Type:  hashAlgorithmLookupType,
		Value: function,
		Kind:  common.DeclarationKindEnum,
	}
}

func NewVMHashAlgorithmCaseValues(hasher Hasher) []VMValue {
	values := make([]VMValue, len(sema.HashAlgorithms))
	for i, hashAlgorithm := range sema.HashAlgorithms {
		rawValue := interpreter.UInt8Value(hashAlgorithm.RawValue())
		// Assume rawValues are all valid, given we iterate over sema.HashAlgorithms
		caseValue, _ := NewHashAlgorithmCase(rawValue, hasher)
		values[i] = VMValue{
			Name: commons.TypeQualifiedName(
				sema.HashAlgorithmType,
				hashAlgorithm.Name(),
			),
			Value: caseValue,
		}
	}
	return values
}
