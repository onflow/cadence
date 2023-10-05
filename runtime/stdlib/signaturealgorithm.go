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
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
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
	)
}

var signatureAlgorithmConstructorValue, SignatureAlgorithmCaseValues = cryptoAlgorithmEnumValueAndCaseValues(
	sema.SignatureAlgorithmType,
	sema.SignatureAlgorithms,
	NewSignatureAlgorithmCase,
)

var SignatureAlgorithmConstructor = StandardLibraryValue{
	Name: sema.SignatureAlgorithmTypeName,
	Type: cryptoAlgorithmEnumConstructorType(
		sema.SignatureAlgorithmType,
		sema.SignatureAlgorithms,
	),
	Value: signatureAlgorithmConstructorValue,
	Kind:  common.DeclarationKindEnum,
}
