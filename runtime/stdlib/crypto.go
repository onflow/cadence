package stdlib

import (
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
)

const CryptoContractLocation = common.IdentifierLocation("Crypto")

func cryptoAlgorithmEnumConstructorType[T sema.CryptoAlgorithm](
	enumType *sema.CompositeType,
	enumCases []T,
) *sema.FunctionType {

	members := make([]*sema.Member, len(enumCases))
	for i, algo := range enumCases {
		members[i] = sema.NewUnmeteredPublicConstantFieldMember(
			enumType,
			algo.Name(),
			enumType,
			algo.DocString(),
		)
	}

	return &sema.FunctionType{
		Purity:        sema.FunctionPurityView,
		IsConstructor: true,
		Parameters: []sema.Parameter{
			{
				Identifier:     sema.EnumRawValueFieldName,
				TypeAnnotation: sema.NewTypeAnnotation(enumType.EnumRawType),
			},
		},
		ReturnTypeAnnotation: sema.NewTypeAnnotation(
			&sema.OptionalType{
				Type: enumType,
			},
		),
		Members: sema.MembersAsMap(members),
	}
}

type enumCaseConstructor func(rawValue interpreter.UInt8Value) interpreter.MemberAccessibleValue

func cryptoAlgorithmEnumValueAndCaseValues[T sema.CryptoAlgorithm](
	enumType *sema.CompositeType,
	enumCases []T,
	caseConstructor enumCaseConstructor,
) (
	value interpreter.Value,
	cases map[interpreter.UInt8Value]interpreter.MemberAccessibleValue,
) {

	caseCount := len(enumCases)
	caseValues := make([]interpreter.EnumCase, caseCount)
	constructorNestedVariables := make(map[string]interpreter.Variable, caseCount)
	cases = make(map[interpreter.UInt8Value]interpreter.MemberAccessibleValue, caseCount)

	for i, enumCase := range enumCases {
		rawValue := interpreter.UInt8Value(enumCase.RawValue())
		caseValue := caseConstructor(rawValue)
		cases[rawValue] = caseValue
		caseValues[i] = interpreter.EnumCase{
			Value:    caseValue,
			RawValue: rawValue,
		}
		constructorNestedVariables[enumCase.Name()] =
			interpreter.NewVariableWithValue(nil, caseValue)
	}

	value = interpreter.EnumConstructorFunction(
		nil,
		interpreter.EmptyLocationRange,
		enumType,
		caseValues,
		constructorNestedVariables,
	)

	return
}
