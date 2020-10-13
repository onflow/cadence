package cli

import (
	"fmt"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/parser2"
	"github.com/onflow/cadence/runtime/sema"
)

func ParseTransactionArguments(tx string, args []string) ([]cadence.Value, error) {
	paramTypes, err := transactionParameterTypes(tx, len(args))
	if err != nil {
		return nil, err
	}

	values := make([]cadence.Value, len(args))

	for i, paramType := range paramTypes {
		value, err := ParseArgument(paramType, args[i])
		if err != nil {
			return nil, err
		}

		values[i] = value
	}

	return values, nil
}

func transactionParameterTypes(tx string, expectedCount int) ([]sema.Type, error) {
	program, err := parser2.ParseProgram(tx)
	if err != nil {
		return nil, fmt.Errorf("argparser: failed to parse transaction: %w", err)
	}

	transactions := program.TransactionDeclarations()

	if len(transactions) != 1 {
		return nil, fmt.Errorf("argparser: program must contain a single transaction declaration")
	}

	transaction := transactions[0]

	parameters := transaction.ParameterList.Parameters

	if len(parameters) != expectedCount {
		return nil, fmt.Errorf("argparser: incorrect argument count")
	}

	parameterTypes := make([]sema.Type, len(parameters))

	for i, param := range parameters {
		typeName := param.TypeAnnotation.Type.String()

		semaType, ok := sema.BaseTypes[typeName]
		if !ok {
			return nil, fmt.Errorf("argparser: unsupported argument type")
		}

		parameterTypes[i] = semaType
	}

	return parameterTypes, nil
}

func ParseArgument(t sema.Type, s string) (cadence.Value, error) {
	exp, errs := parser2.ParseExpression(s)
	if len(errs) != 0 {
		return nil, fmt.Errorf("argparser: failed to parse: %s", errs)
	}

	// TODO: implement all base types
	switch t.(type) {
	case *sema.Fix64Type:
		return newFix64(exp)
	case *sema.UFix64Type:
		return newUFix64(exp)
	case *sema.IntType:
		return newInt(exp)
	case *sema.StringType:
		return newString(exp)
	case *sema.AddressType:
		return newAddress(exp)
	default:
		return nil, fmt.Errorf("argparser: unsupported type %T", t)
	}
}

func newFix64(exp ast.Expression) (cadence.Fix64, error) {
	switch typedExp := exp.(type) {
	case *ast.FixedPointExpression:
		value, err := cadence.NewFix64FromParts(
			typedExp.Negative,
			int(typedExp.UnsignedInteger.Int64()),
			uint(typedExp.Fractional.Uint64()),
		)
		if err != nil {
			return 0, fmt.Errorf("argparser: invalid Fix64 value: %w", err)
		}

		return value, nil
	default:
		return 0, fmt.Errorf("argparser: invalid Fix64 value")
	}
}

func newUFix64(exp ast.Expression) (cadence.UFix64, error) {
	switch typedExp := exp.(type) {
	case *ast.FixedPointExpression:
		if typedExp.Negative {
			return 0, fmt.Errorf("argparser: invalid UFix64 value")
		}

		value, err := cadence.NewUFix64FromParts(
			int(typedExp.UnsignedInteger.Int64()),
			uint(typedExp.Fractional.Uint64()),
		)
		if err != nil {
			return 0, fmt.Errorf("argparser: invalid UFix64 value: %w", err)
		}

		return value, nil
	default:
		return 0, fmt.Errorf("argparser: invalid UFix64 value")
	}
}

func newInt(exp ast.Expression) (cadence.Int, error) {
	switch typedExp := exp.(type) {
	case *ast.IntegerExpression:
		return cadence.NewIntFromBig(typedExp.Value), nil
	default:
		return cadence.Int{}, fmt.Errorf("argparser: invalid Int value")
	}
}

func newString(exp ast.Expression) (cadence.String, error) {
	switch typedExp := exp.(type) {
	case *ast.StringExpression:
		return cadence.NewString(typedExp.Value), nil
	default:
		return "", fmt.Errorf("argparser: invalid String value")
	}
}

func newAddress(exp ast.Expression) (cadence.Address, error) {
	switch typedExp := exp.(type) {
	case *ast.IntegerExpression:
		return cadence.BytesToAddress(typedExp.Value.Bytes()), nil
	default:
		return cadence.Address{}, fmt.Errorf("argparser: invalid Address value")
	}
}
