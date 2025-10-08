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

var signatureAlgorithmStaticType interpreter.StaticType = interpreter.ConvertSemaCompositeTypeToStaticCompositeType(
	nil,
	sema.SignatureAlgorithmType,
)

func NewSignatureAlgorithmCase(rawValue interpreter.UInt8Value) interpreter.MemberAccessibleValue {

	fields := map[string]interpreter.Value{
		sema.EnumRawValueFieldName: rawValue,
	}

	return interpreter.NewSimpleCompositeValue(
		nil,
		sema.SignatureAlgorithmType.ID(),
		signatureAlgorithmStaticType,
		[]string{sema.EnumRawValueFieldName},
		fields,
		nil,
		nil,
		nil,
		nil,
	)
}

var signatureAlgorithmLookupType = cryptoAlgorithmEnumLookupType(
	sema.SignatureAlgorithmType,
	sema.SignatureAlgorithms,
)

var interpreterSignatureAlgorithmConstructorValue, SignatureAlgorithmCaseValues = interpreterCryptoAlgorithmEnumValueAndCaseValues(
	signatureAlgorithmLookupType,
	sema.SignatureAlgorithms,
	NewSignatureAlgorithmCase,
)

// these functions are left as is, since there are differences in the implementations between interpreter and vm
var InterpreterSignatureAlgorithmConstructor = StandardLibraryValue{
	Name:  sema.SignatureAlgorithmTypeName,
	Type:  signatureAlgorithmLookupType,
	Value: interpreterSignatureAlgorithmConstructorValue,
	Kind:  common.DeclarationKindEnum,
}

var vmSignatureAlgorithmConstructorValue = vm.NewNativeFunctionValue(
	sema.SignatureAlgorithmTypeName,
	signatureAlgorithmLookupType,
	func(
		context interpreter.NativeFunctionContext,
		_ interpreter.TypeParameterGetter,
		_ interpreter.Value,
		args ...interpreter.Value,
	) interpreter.Value {
		rawValue := args[0].(interpreter.UInt8Value)

		caseValue, ok := SignatureAlgorithmCaseValues[rawValue]
		if !ok {
			return interpreter.Nil
		}

		return interpreter.NewSomeValueNonCopying(context, caseValue)
	},
)

var VMSignatureAlgorithmConstructor = StandardLibraryValue{
	Name:  sema.SignatureAlgorithmTypeName,
	Type:  signatureAlgorithmLookupType,
	Value: vmSignatureAlgorithmConstructorValue,
	Kind:  common.DeclarationKindEnum,
}

var VMSignatureAlgorithmCaseValues = func() []VMValue {
	values := make([]VMValue, len(sema.SignatureAlgorithms))
	for i, signatureAlgorithm := range sema.SignatureAlgorithms {
		rawValue := interpreter.UInt8Value(signatureAlgorithm.RawValue())
		values[i] = VMValue{
			Name: commons.TypeQualifiedName(
				sema.SignatureAlgorithmType,
				signatureAlgorithm.Name(),
			),
			Value: SignatureAlgorithmCaseValues[rawValue],
		}
	}
	return values
}()
