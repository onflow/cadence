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
	"github.com/onflow/cadence/bbq/commons"
	"github.com/onflow/cadence/bbq/vm"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
)

var roundingModeStaticType interpreter.StaticType = interpreter.ConvertSemaCompositeTypeToStaticCompositeType(
	nil,
	sema.RoundingModeType,
)

func NewRoundingModeCase(rawValue interpreter.UInt8Value) interpreter.MemberAccessibleValue {

	fields := map[string]interpreter.Value{
		sema.EnumRawValueFieldName: rawValue,
	}

	return interpreter.NewSimpleCompositeValue(
		nil,
		sema.RoundingModeType.ID(),
		roundingModeStaticType,
		[]string{sema.EnumRawValueFieldName},
		fields,
		nil,
		nil,
		nil,
		nil,
	)
}

var roundingModeLookupType = nativeEnumLookupType(
	sema.RoundingModeType,
	sema.RoundingModes,
)

var interpreterRoundingModeConstructorValue, RoundingModeCaseValues = interpreterNativeEnumValueAndCaseValues(
	roundingModeLookupType,
	sema.RoundingModes,
	NewRoundingModeCase,
)

var InterpreterRoundingModeConstructor = StandardLibraryValue{
	Name:  sema.RoundingModeTypeName,
	Type:  roundingModeLookupType,
	Value: interpreterRoundingModeConstructorValue,
	Kind:  common.DeclarationKindEnum,
}

var vmRoundingModeConstructorValue = vm.NewNativeFunctionValue(
	sema.RoundingModeTypeName,
	roundingModeLookupType,
	func(
		context interpreter.NativeFunctionContext,
		_ interpreter.TypeArgumentsIterator,
		_ interpreter.ArgumentTypesIterator,
		_ interpreter.Value,
		args []interpreter.Value,
	) interpreter.Value {
		rawValue := args[0].(interpreter.UInt8Value)

		caseValue, ok := RoundingModeCaseValues[rawValue]
		if !ok {
			return interpreter.Nil
		}

		return interpreter.NewSomeValueNonCopying(context, caseValue)
	},
)

var VMRoundingModeConstructor = StandardLibraryValue{
	Name:  sema.RoundingModeTypeName,
	Type:  roundingModeLookupType,
	Value: vmRoundingModeConstructorValue,
	Kind:  common.DeclarationKindEnum,
}

var VMRoundingModeCaseValues = func() []VMValue {
	values := make([]VMValue, len(sema.RoundingModes))
	for i, roundingMode := range sema.RoundingModes {
		rawValue := interpreter.UInt8Value(roundingMode.RawValue())
		values[i] = VMValue{
			Name: commons.TypeQualifiedName(
				sema.RoundingModeType,
				roundingMode.Name(),
			),
			Value: RoundingModeCaseValues[rawValue],
		}
	}
	return values
}()
