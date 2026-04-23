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

var roundingRuleStaticType interpreter.StaticType = interpreter.ConvertSemaCompositeTypeToStaticCompositeType(
	nil,
	sema.RoundingRuleType,
)

func NewRoundingRuleCase(rawValue interpreter.UInt8Value) interpreter.MemberAccessibleValue {

	fields := map[string]interpreter.Value{
		sema.EnumRawValueFieldName: rawValue,
	}

	return interpreter.NewSimpleCompositeValue(
		nil,
		sema.RoundingRuleType.ID(),
		roundingRuleStaticType,
		[]string{sema.EnumRawValueFieldName},
		fields,
		nil,
		nil,
		nil,
		nil,
	)
}

var roundingRuleLookupType = nativeEnumLookupType(
	sema.RoundingRuleType,
	sema.RoundingRules,
)

var interpreterRoundingRuleConstructorValue, RoundingRuleCaseValues = interpreterNativeEnumValueAndCaseValues(
	roundingRuleLookupType,
	sema.RoundingRules,
	NewRoundingRuleCase,
)

var InterpreterRoundingRuleConstructor = StandardLibraryValue{
	Name:  sema.RoundingRuleTypeName,
	Type:  roundingRuleLookupType,
	Value: interpreterRoundingRuleConstructorValue,
	Kind:  common.DeclarationKindEnum,
}

var vmRoundingRuleConstructorValue = vm.NewNativeFunctionValue(
	sema.RoundingRuleTypeName,
	roundingRuleLookupType,
	func(
		context interpreter.NativeFunctionContext,
		_ interpreter.TypeArgumentsIterator,
		_ interpreter.ArgumentTypesIterator,
		_ interpreter.Value,
		args []interpreter.Value,
	) interpreter.Value {
		rawValue := args[0].(interpreter.UInt8Value)

		caseValue, ok := RoundingRuleCaseValues[rawValue]
		if !ok {
			return interpreter.Nil
		}

		return interpreter.NewSomeValueNonCopying(context, caseValue)
	},
)

var VMRoundingRuleConstructor = StandardLibraryValue{
	Name:  sema.RoundingRuleTypeName,
	Type:  roundingRuleLookupType,
	Value: vmRoundingRuleConstructorValue,
	Kind:  common.DeclarationKindEnum,
}

var VMRoundingRuleCaseValues = func() []VMValue {
	values := make([]VMValue, len(sema.RoundingRules))
	for i, roundingRule := range sema.RoundingRules {
		rawValue := interpreter.UInt8Value(roundingRule.RawValue())
		values[i] = VMValue{
			Name: commons.TypeQualifiedName(
				sema.RoundingRuleType,
				roundingRule.Name(),
			),
			Value: RoundingRuleCaseValues[rawValue],
		}
	}
	return values
}()
