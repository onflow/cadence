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

package interpreter_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/activations"
	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/bbq"
	"github.com/onflow/cadence/bbq/compiler"
	. "github.com/onflow/cadence/bbq/test_utils"
	compilerUtils "github.com/onflow/cadence/bbq/vm/test"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/stdlib"
	. "github.com/onflow/cadence/test_utils/sema_utils"
)

// TestInterpretStatementEndingControlFlowFallthrough tests the defensive check
// for statements which the checker determined to end control flow:
// if execution nevertheless continues past such a statement,
// execution must abort, instead of silently continuing.
// In the interpreter, the check is performed when visiting statements;
// in the compiler/VM, an unreachable instruction is emitted after the statement.
//
// The check can only fire when the checker's control-flow analysis over-claims
// (a checker bug), which cannot be produced from source code here.
// Therefore, simulate such a bug by marking a statement which completes normally.
func TestInterpretStatementEndingControlFlowFallthrough(t *testing.T) {

	t.Parallel()

	inter, err := parseCheckAndPrepareWithOptions(t,
		`
          fun test() {
              let x = 1
          }
        `,
		ParseCheckAndInterpretOptions{
			HandleChecker: func(checker *sema.Checker) {
				// Simulate a checker bug: mark the variable declaration statement
				// as ending control flow, even though execution continues past it
				statement := checker.Program.FunctionDeclarations()[0].FunctionBlock.Block.Statements[0]
				checker.Elaboration.SetStatementEndsControlFlow(statement)
			},
		},
	)
	require.NoError(t, err)

	_, err = inter.Invoke("test")

	var unreachableInstructionErr *interpreter.UnreachableInstructionError
	require.ErrorAs(t, err, &unreachableInstructionErr)
}

// TestInterpretInheritedStatementEndingControlFlowFallthrough tests
// the same defensive check as TestInterpretStatementEndingControlFlowFallthrough,
// but for an *inherited* statement:
// the before-statement of an inherited post-condition,
// which is declared in another program.
// The check must consult the elaboration of the declaring program,
// not the elaboration of the inheriting program:
// the interpreter executes the statement with the declaring program's interpreter,
// and the compiler resolves the declaring program's elaboration
// (see compilePotentiallyInheritedCode).
//
// An inherited statement which ends control flow cannot currently be produced
// from source code, so simulate a checker bug by marking the interface's
// before-statement, which completes normally.
func TestInterpretInheritedStatementEndingControlFlowFallthrough(t *testing.T) {

	t.Parallel()

	importLocation := common.NewAddressLocation(
		nil,
		common.MustBytesToAddress([]byte{0x1}),
		"",
	)

	const importCode = `
       struct interface SI {

           fun test(x: Int): Int {
               post {
                   before(x) < x
               }
           }
       }
    `

	const testCode = `
      import SI from 0x1

      struct S: SI {

          fun test(x: Int): Int {
              return 42
          }
      }

      fun main(): Int {
          return S().test(x: 1)
      }
    `

	// Simulate a checker bug: mark the before-statement
	// of the interface function's post-condition (`var $before_0 = x`)
	// as ending control flow, even though execution continues past it
	markBeforeStatement := func(checker *sema.Checker) {
		interfaceDeclaration := checker.Program.InterfaceDeclarations()[0]
		functionDeclaration := interfaceDeclaration.Members.Functions()[0]
		postConditions := functionDeclaration.FunctionBlock.PostConditions
		rewrite := checker.Elaboration.PostConditionsRewrite(postConditions)
		beforeStatement := rewrite.BeforeStatements[0]
		checker.Elaboration.SetStatementEndsControlFlow(beforeStatement)
	}

	var err error
	if *compile {

		programs := CompiledPrograms{}

		_ = ParseCheckAndCompileCodeWithOptions(t,
			importCode,
			importLocation,
			ParseCheckAndCompileOptions{
				CheckerHandler: markBeforeStatement,
			},
			programs,
		)

		_, err = compilerUtils.CompileAndInvokeWithOptionsAndPrograms(t,
			testCode,
			"main",
			compilerUtils.CompilerAndVMOptions{
				ParseCheckAndCompileOptions: ParseCheckAndCompileOptions{
					ParseAndCheckOptions: &ParseAndCheckOptions{
						CheckerConfig: &sema.Config{
							ImportHandler: func(checker *sema.Checker, importedLocation common.Location, _ ast.Range) (sema.Import, error) {
								importedProgram, ok := programs[importedLocation]
								if !ok {
									return nil, fmt.Errorf("cannot find program for location %s", importedLocation)
								}

								return sema.ElaborationImport{
									Elaboration: importedProgram.DesugaredElaboration.OriginalElaboration(),
								}, nil
							},
						},
					},
					CompilerConfig: &compiler.Config{
						ImportHandler: func(location common.Location) *bbq.InstructionProgram {
							return programs[location].Program
						},
						LocationHandler: func(identifiers []ast.Identifier, location common.Location) ([]sema.ResolvedLocation, error) {
							return []sema.ResolvedLocation{
								{
									Location:    location,
									Identifiers: identifiers,
								},
							}, nil
						},
					},
				},
			},
			programs,
		)

	} else {

		importedChecker, importErr := ParseAndCheckWithOptions(t,
			importCode,
			ParseAndCheckOptions{
				Location: importLocation,
			},
		)
		require.NoError(t, importErr)

		markBeforeStatement(importedChecker)

		inter, prepareErr := parseCheckAndInterpretWithOptions(t,
			testCode,
			ParseCheckAndInterpretOptions{
				ParseAndCheckOptions: &ParseAndCheckOptions{
					CheckerConfig: &sema.Config{
						ImportHandler: func(checker *sema.Checker, importedLocation common.Location, _ ast.Range) (sema.Import, error) {
							return sema.ElaborationImport{
								Elaboration: importedChecker.Elaboration,
							}, nil
						},
					},
				},
				InterpreterConfig: &interpreter.Config{
					ImportLocationHandler: func(inter *interpreter.Interpreter, location common.Location) interpreter.Import {

						program := interpreter.ProgramFromChecker(importedChecker)
						subInterpreter, err := inter.NewSubInterpreter(program, location)
						if err != nil {
							panic(err)
						}

						return interpreter.InterpreterImport{
							Interpreter: subInterpreter,
						}
					},
				},
			},
		)
		require.NoError(t, prepareErr)

		_, err = inter.Invoke("main")
	}

	var unreachableInstructionErr *interpreter.UnreachableInstructionError
	require.ErrorAs(t, err, &unreachableInstructionErr)
}

// TestInterpretVoidReturnWithMismatchedReturnType tests the defensive handling
// of an invocation whose static return type disagrees with the invoked function,
// and the invoked function returns void:
// the invocation must abort with an internal error,
// instead of continuing with Void masquerading as a value of the return type.
// This is particularly important for return type Never,
// which is assignable to any type.
//
// Such a disagreement cannot be produced from source code,
// so simulate a checker bug by overriding the return type
// of the invocation of a void function in the elaboration.
//
// In the VM, the void return path (InstructionReturn) performs
// a defensive check that the call frame's expected return type allows void
// (see opReturn).
// Unlike the value return path (InstructionReturnValue)
// and native function invocations, which validate the returned value,
// the void return path would otherwise push Void without any validation.
//
// In the interpreter, the return-value validation catches the disagreement,
// when the invocation's result is validated against the invocation's return type
// (see ConvertAndBoxWithValidation, performed in invokeFunctionValueWithEval).
func TestInterpretVoidReturnWithMismatchedReturnType(t *testing.T) {

	t.Parallel()

	inter, err := parseCheckAndPrepareWithOptions(t,
		`
          fun f() {}

          fun test() {
              f()
          }
        `,
		ParseCheckAndInterpretOptions{
			HandleChecker: func(checker *sema.Checker) {
				// Simulate a checker bug: override the return type
				// of the invocation `f()` to Never,
				// even though the invoked function returns void
				functionDeclaration := checker.Program.FunctionDeclarations()[1]
				statement := functionDeclaration.FunctionBlock.Block.Statements[0]
				expressionStatement := statement.(*ast.ExpressionStatement)
				fInvocation := expressionStatement.Expression.(*ast.InvocationExpression)

				invocationTypes := checker.Elaboration.InvocationExpressionTypes(fInvocation)
				invocationTypes.ReturnType = sema.NeverType
				checker.Elaboration.SetInvocationExpressionTypes(fInvocation, invocationTypes)
			},
		},
	)
	require.NoError(t, err)

	_, err = inter.Invoke("test")

	var valueTransferTypeErr *interpreter.ValueTransferTypeError
	require.ErrorAs(t, err, &valueTransferTypeErr)
}

// TestInterpretImplicitVoidReturnWithMismatchedReturnType tests
// the defensive handling of a function which completes
// without an explicit return (an implicit void return),
// even though its return type is not Void:
// the invocation must abort with an internal error,
// instead of continuing with Void masquerading as a value of the return type.
//
// Such a function cannot be produced from source code:
// the checker requires a function with a return type other than Void
// to definitely return or halt.
// Therefore, simulate a checker bug by overriding the function type
// of a void function in the elaboration.
//
// The check is performed in visitFunctionBody, when the body completes
// without an explicit return (see the call in invokeInterpretedFunctionActivated).
// It protects all invocation paths, including internal direct invocations
// which skip the call-site return-value validation
// (see ConvertAndBoxWithValidation, performed in invokeFunctionValueWithEval).
//
// This test is interpreter-only:
// the equivalent check in the VM is the defensive check
// in the void return path (see opReturn),
// which TestInterpretVoidReturnWithMismatchedReturnType covers in compile mode.
func TestInterpretImplicitVoidReturnWithMismatchedReturnType(t *testing.T) {

	t.Parallel()

	inter, err := parseCheckAndInterpretWithOptions(t,
		`
          fun f() {}

          fun test() {
              f()
          }
        `,
		ParseCheckAndInterpretOptions{
			HandleChecker: func(checker *sema.Checker) {
				// Simulate a checker bug: override the function type of `f`
				// to have return type Int,
				// even though the function declaration has no return type (Void),
				// so its body completes without an explicit return
				functionDeclaration := checker.Program.FunctionDeclarations()[0]
				functionType := sema.NewSimpleFunctionType(
					sema.FunctionPurityImpure,
					nil,
					sema.IntTypeAnnotation,
				)
				checker.Elaboration.SetFunctionDeclarationFunctionType(
					functionDeclaration,
					functionType,
				)
			},
		},
	)
	require.NoError(t, err)

	_, err = inter.Invoke("test")

	var valueTransferTypeErr *interpreter.ValueTransferTypeError
	require.ErrorAs(t, err, &valueTransferTypeErr)
}
