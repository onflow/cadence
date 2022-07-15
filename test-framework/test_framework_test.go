package test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib"
	"github.com/onflow/cadence/runtime/tests/checker"
)

func executeFunction(t *testing.T, code string, funcName string) (interpreter.Value, error) {
	predeclaredSemaValues := stdlib.BuiltinFunctions.ToSemaValueDeclarations()
	predeclaredSemaValues = append(predeclaredSemaValues, stdlib.BuiltinValues.ToSemaValueDeclarations()...)

	checker, err := checker.ParseAndCheckWithOptions(t, code, checker.ParseAndCheckOptions{
		Location:         nil,
		IgnoreParseError: false,
		Options: []sema.Option{
			sema.WithPredeclaredValues(predeclaredSemaValues),
			sema.WithPredeclaredTypes(stdlib.FlowDefaultPredeclaredTypes),
			sema.WithImportHandler(
				func(checker *sema.Checker, importedLocation common.Location, importRange ast.Range) (sema.Import, error) {
					var elaboration *sema.Elaboration
					switch importedLocation {
					case stdlib.CryptoChecker.Location:
						elaboration = stdlib.CryptoChecker.Elaboration

					case stdlib.TestContractLocation:
						elaboration = stdlib.TestContractChecker.Elaboration

					default:
						return nil, errors.NewUnexpectedError("importing programs not implemented")
					}

					return sema.ElaborationImport{
						Elaboration: elaboration,
					}, nil
				},
			),
		},
	})

	predeclaredInterpreterValues := stdlib.BuiltinFunctions.ToInterpreterValueDeclarations()
	predeclaredInterpreterValues = append(predeclaredInterpreterValues, stdlib.BuiltinValues.ToInterpreterValueDeclarations()...)

	inter, err := interpreter.NewInterpreter(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
		interpreter.WithStorage(interpreter.NewInMemoryStorage(nil)),
		interpreter.WithTestFramework(NewEmulatorBackend()),
		interpreter.WithPredeclaredValues(predeclaredInterpreterValues),
		interpreter.WithImportLocationHandler(func(inter *interpreter.Interpreter, location common.Location) interpreter.Import {
			switch location {
			case stdlib.CryptoChecker.Location:
				program := interpreter.ProgramFromChecker(stdlib.CryptoChecker)
				subInterpreter, err := inter.NewSubInterpreter(program, location)
				if err != nil {
					panic(err)
				}
				return interpreter.InterpreterImport{
					Interpreter: subInterpreter,
				}

			case stdlib.TestContractLocation:
				program := interpreter.ProgramFromChecker(stdlib.TestContractChecker)
				subInterpreter, err := inter.NewSubInterpreter(program, location)
				if err != nil {
					panic(err)
				}
				return interpreter.InterpreterImport{
					Interpreter: subInterpreter,
				}

			default:
				panic(errors.NewUnexpectedError("importing programs not implemented"))
			}
		}),
	)

	require.NoError(t, err)

	err = inter.Interpret()
	require.NoError(t, err)

	return inter.Invoke(funcName)
}

func TestExecuteScript(t *testing.T) {
	code := `
        pub fun test() {
            var bc = Test.Blockchain()
            bc.executeScript("pub fun foo(): String {  return \"hello\" }")
        }
    `

	_, err := executeFunction(t, code, "test")
	assert.NoError(t, err)
}
