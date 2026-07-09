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
	"github.com/onflow/cadence"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
)

type ValidatedArgumentImportContext interface {
	common.MemoryGauge
	interpreter.ValueStaticTypeContext
	interpreter.ValueImportableContext
	interpreter.ValueWalkContext
	ValueImportContext
}

func importValidatedArguments(
	context ValidatedArgumentImportContext,
	decoder ArgumentDecoder,
	arguments [][]byte,
	parameters []sema.Parameter,
) (
	[]interpreter.Value,
	error,
) {
	argumentCount := len(arguments)
	parameterCount := len(parameters)

	if argumentCount != parameterCount {
		return nil, InvalidEntryPointParameterCountError{
			Expected: parameterCount,
			Actual:   argumentCount,
		}
	}

	argumentValues := make([]interpreter.Value, len(arguments))

	// Decode arguments against parameter types
	for parameterIndex, parameter := range parameters {
		parameterType := parameter.TypeAnnotation.Type
		argument := arguments[parameterIndex]

		exportedParameterType := ExportMeteredType(context, parameterType, map[sema.TypeID]cadence.Type{})

		value, err := decoder.DecodeArgument(
			argument,
			exportedParameterType,
		)
		if err != nil {
			return nil, &InvalidEntryPointArgumentError{
				Index: parameterIndex,
				Err:   err,
			}
		}

		var arg interpreter.Value
		panicError := UserPanicToError(func() {
			// if importing an invalid public key, this call panics
			arg, err = ImportValue(
				context,
				decoder,
				decoder.ResolveLocation,
				value,
				parameterType,
			)
		})

		if panicError != nil {
			return nil, &InvalidEntryPointArgumentError{
				Index: parameterIndex,
				Err:   panicError,
			}
		}

		if err != nil {
			return nil, &InvalidEntryPointArgumentError{
				Index: parameterIndex,
				Err:   err,
			}
		}

		// Ensure the argument is of an importable type
		if !arg.IsImportable(context) {
			return nil, &ArgumentNotImportableError{
				Type: arg.StaticType(context),
			}
		}

		// Validate the imported argument against the parameter type
		err = validateImportedArgument(context, arg, parameterType)
		if err != nil {
			return nil, &InvalidEntryPointArgumentError{
				Index: parameterIndex,
				Err:   err,
			}
		}

		argumentValues[parameterIndex] = arg
	}

	return argumentValues, nil
}

// validateImportedArgument checks that an imported argument value:
//   - is a subtype of the expected parameter type,
//   - conforms to its associated static type, and
//   - has valid static type information for all its nested values.
func validateImportedArgument(
	context ValidatedArgumentImportContext,
	value interpreter.Value,
	parameterType sema.Type,
) error {
	valueType := value.StaticType(context)

	// Check that the value is a subtype of the static parameter type
	if !interpreter.IsSubTypeOfSemaType(context, valueType, parameterType) {
		return &InvalidValueTypeError{
			ExpectedType: parameterType,
		}
	}

	// Check whether the value conforms to the type associated with the value
	if !value.ConformsToStaticType(context, interpreter.TypeConformanceResults{}) {
		return &MalformedValueError{
			ExpectedType: parameterType,
		}
	}

	// Ensure static type info is available for all values
	interpreter.InspectValue(
		context,
		value,
		func(value interpreter.Value) bool {
			if value == nil {
				return true
			}

			if !hasValidStaticType(context, value) {
				panic(errors.NewUnexpectedError("invalid static type for argument"))
			}

			return true
		},
	)

	return nil
}

func hasValidStaticType(context interpreter.ValueStaticTypeContext, value interpreter.Value) bool {
	switch value := value.(type) {
	case *interpreter.ArrayValue:
		return value.Type != nil
	case *interpreter.DictionaryValue:
		return value.Type.KeyType != nil &&
			value.Type.ValueType != nil
	default:
		// For other values, static type is NOT inferred.
		// Hence no need to validate it here.
		return value.StaticType(context) != nil
	}
}
