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
	"math/big"
	"strings"
	"testing"

	"github.com/onflow/atree"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/activations"
	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/common/orderedmap"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/parser"
	"github.com/onflow/cadence/pretty"
	"github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/stdlib"
	. "github.com/onflow/cadence/test_utils/common_utils"
	. "github.com/onflow/cadence/test_utils/interpreter_utils"
	. "github.com/onflow/cadence/test_utils/runtime_utils"
	. "github.com/onflow/cadence/test_utils/sema_utils"
)

type ParseCheckAndInterpretOptions struct {
	Config             *interpreter.Config
	CheckerConfig      *sema.Config
	HandleCheckerError func(error)
}

func parseCheckAndInterpret(t testing.TB, code string) *interpreter.Interpreter {
	inter, err := parseCheckAndInterpretWithOptions(t, code, ParseCheckAndInterpretOptions{})
	require.NoError(t, err)
	return inter
}

func parseCheckAndInterpretWithOptions(
	t testing.TB,
	code string,
	options ParseCheckAndInterpretOptions,
) (
	inter *interpreter.Interpreter,
	err error,
) {
	return parseCheckAndInterpretWithOptionsAndMemoryMetering(t, code, options, nil)
}

func parseCheckAndInterpretWithAtreeValidationsDisabled(
	t testing.TB,
	code string,
	options ParseCheckAndInterpretOptions,
) (
	inter *interpreter.Interpreter,
	err error,
) {
	return parseCheckAndInterpretWithOptionsAndMemoryMeteringAndAtreeValidations(
		t,
		code,
		options,
		nil,
		false,
	)
}

func parseCheckAndInterpretWithLogs(
	tb testing.TB,
	code string,
) (
	inter *interpreter.Interpreter,
	getLogs func() []string,
	err error,
) {
	var logs []string

	logFunction := stdlib.NewStandardLibraryStaticFunction(
		"log",
		&sema.FunctionType{
			Parameters: []sema.Parameter{
				{
					Label:          sema.ArgumentLabelNotRequired,
					Identifier:     "value",
					TypeAnnotation: sema.NewTypeAnnotation(sema.AnyStructType),
				},
			},
			ReturnTypeAnnotation: sema.NewTypeAnnotation(
				sema.VoidType,
			),
		},
		``,
		func(invocation interpreter.Invocation) interpreter.Value {
			message := invocation.Arguments[0].MeteredString(
				invocation.Interpreter,
				interpreter.SeenReferences{},
				invocation.LocationRange,
			)
			logs = append(logs, message)
			return interpreter.Void
		},
	)

	baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
	baseValueActivation.DeclareValue(logFunction)

	baseActivation := activations.NewActivation(nil, interpreter.BaseActivation)
	interpreter.Declare(baseActivation, logFunction)

	result, err := parseCheckAndInterpretWithOptions(
		tb,
		code,
		ParseCheckAndInterpretOptions{
			Config: &interpreter.Config{
				BaseActivationHandler: func(_ common.Location) *interpreter.VariableActivation {
					return baseActivation
				},
			},
			CheckerConfig: &sema.Config{
				BaseValueActivationHandler: func(_ common.Location) *sema.VariableActivation {
					return baseValueActivation
				},
			},
			HandleCheckerError: nil,
		},
	)

	getLogs = func() []string {
		return logs
	}

	return result, getLogs, err
}

func parseCheckAndInterpretWithMemoryMetering(
	t testing.TB,
	code string,
	memoryGauge common.MemoryGauge,
) *interpreter.Interpreter {

	baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
	baseValueActivation.DeclareValue(stdlib.PanicFunction)

	inter, err := parseCheckAndInterpretWithOptionsAndMemoryMetering(
		t,
		code,
		ParseCheckAndInterpretOptions{
			CheckerConfig: &sema.Config{
				BaseValueActivationHandler: func(_ common.Location) *sema.VariableActivation {
					return baseValueActivation
				},
			},
		},
		memoryGauge,
	)
	require.NoError(t, err)
	return inter
}

func parseCheckAndInterpretWithOptionsAndMemoryMetering(
	t testing.TB,
	code string,
	options ParseCheckAndInterpretOptions,
	memoryGauge common.MemoryGauge,
) (
	inter *interpreter.Interpreter,
	err error,
) {

	// Atree validation should be disabled for memory metering tests.
	// Otherwise, validation may also affect the memory consumption.
	enableAtreeValidations := memoryGauge == nil

	return parseCheckAndInterpretWithOptionsAndMemoryMeteringAndAtreeValidations(
		t,
		code,
		options,
		memoryGauge,
		enableAtreeValidations,
	)
}

func parseCheckAndInterpretWithOptionsAndMemoryMeteringAndAtreeValidations(
	t testing.TB,
	code string,
	options ParseCheckAndInterpretOptions,
	memoryGauge common.MemoryGauge,
	enableAtreeValidations bool,
) (
	inter *interpreter.Interpreter,
	err error,
) {

	checker, err := ParseAndCheckWithOptionsAndMemoryMetering(t,
		code,
		ParseAndCheckOptions{
			Config: options.CheckerConfig,
		},
		memoryGauge,
	)

	if options.HandleCheckerError != nil {
		options.HandleCheckerError(err)
	} else if !assert.NoError(t, err) {
		var sb strings.Builder
		location := checker.Location
		printErr := pretty.NewErrorPrettyPrinter(&sb, true).
			PrettyPrintError(err, location, map[common.Location][]byte{location: []byte(code)})
		if printErr != nil {
			panic(printErr)
		}
		assert.Fail(t, sb.String())
		return nil, err
	}

	var uuid uint64 = 0

	var config interpreter.Config
	if options.Config != nil {
		config = *options.Config
	}

	if enableAtreeValidations {
		config.AtreeValueValidationEnabled = true
		config.AtreeStorageValidationEnabled = true
	} else {
		config.AtreeValueValidationEnabled = false
		config.AtreeStorageValidationEnabled = false
	}

	if config.UUIDHandler == nil {
		config.UUIDHandler = func() (uint64, error) {
			uuid++
			return uuid, nil
		}
	}
	if config.Storage == nil {
		config.Storage = interpreter.NewInMemoryStorage(memoryGauge)
	}

	if memoryGauge != nil && config.MemoryGauge == nil {
		config.MemoryGauge = memoryGauge
	}

	inter, err = interpreter.NewInterpreter(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
		&config,
	)

	require.NoError(t, err)

	err = inter.Interpret()

	if err == nil {

		// recover internal panics and return them as an error
		defer inter.RecoverErrors(func(internalErr error) {
			err = internalErr
		})

		// Contract declarations are evaluated lazily,
		// so force the contract value handler to be called

		for _, compositeDeclaration := range checker.Program.CompositeDeclarations() {
			if compositeDeclaration.CompositeKind != common.CompositeKindContract {
				continue
			}

			contractVariable := inter.Globals.Get(compositeDeclaration.Identifier.Identifier)

			_ = contractVariable.GetValue(inter)
		}
	}

	return inter, err
}

type testEvent struct {
	event     *interpreter.CompositeValue
	eventType *sema.CompositeType
}

func parseCheckAndInterpretWithEvents(t *testing.T, code string) (
	inter *interpreter.Interpreter,
	getEvents func() []testEvent,
	err error,
) {
	var events []testEvent

	inter, err = parseCheckAndInterpretWithOptions(t,
		code,
		ParseCheckAndInterpretOptions{
			Config: &interpreter.Config{
				OnEventEmitted: func(
					_ *interpreter.Interpreter,
					_ interpreter.LocationRange,
					event *interpreter.CompositeValue,
					eventType *sema.CompositeType,
				) error {
					events = append(events, testEvent{
						event:     event,
						eventType: eventType,
					})
					return nil
				},
			},
		},
	)
	if err != nil {
		return nil, nil, err
	}

	getEvents = func() []testEvent {
		return events
	}
	return inter, getEvents, nil
}

func newUnmeteredInMemoryStorage() interpreter.InMemoryStorage {
	return interpreter.NewInMemoryStorage(nil)
}

func constructorArguments(compositeKind common.CompositeKind, arguments string) string {
	switch compositeKind {
	case common.CompositeKindContract:
		return ""
	case common.CompositeKindEnum:
		return ".a"
	default:
		return fmt.Sprintf("(%s)", arguments)
	}
}

// makeContractValueHandler creates an interpreter option which
// sets the ContractValueHandler.
// The handler immediately invokes the constructor with the given arguments.
func makeContractValueHandler(
	arguments []interpreter.Value,
	argumentTypes []sema.Type,
	parameterTypes []sema.Type,
) interpreter.ContractValueHandlerFunc {
	return func(
		inter *interpreter.Interpreter,
		compositeType *sema.CompositeType,
		constructorGenerator func(common.Address) *interpreter.HostFunctionValue,
		invocationRange ast.Range,
	) interpreter.ContractValue {

		constructor := constructorGenerator(common.ZeroAddress)

		value, err := inter.InvokeFunctionValue(
			constructor,
			arguments,
			argumentTypes,
			parameterTypes,
			compositeType,
			ast.Range{},
		)
		if err != nil {
			panic(err)
		}

		return value.(*interpreter.CompositeValue)
	}
}

func TestInterpretConstantAndVariableDeclarations(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
        let x = 1
        let y = true
        let z = 1 + 2
        var a = 3 == 3
        var b = [1, 2]
        let s = "123"
    `)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(1),
		inter.Globals.Get("x").GetValue(inter),
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.TrueValue,
		inter.Globals.Get("y").GetValue(inter),
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(3),
		inter.Globals.Get("z").GetValue(inter),
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.TrueValue,
		inter.Globals.Get("a").GetValue(inter),
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewArrayValue(
			inter,
			interpreter.EmptyLocationRange,
			&interpreter.VariableSizedStaticType{
				Type: interpreter.PrimitiveStaticTypeInt,
			},
			common.ZeroAddress,
			interpreter.NewUnmeteredIntValueFromInt64(1),
			interpreter.NewUnmeteredIntValueFromInt64(2),
		),
		inter.Globals.Get("b").GetValue(inter),
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredStringValue("123"),
		inter.Globals.Get("s").GetValue(inter),
	)
}

func TestInterpretDeclarations(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
        fun test(): Int {
            return 42
        }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(42),
		value,
	)
}

func TestInterpretInvalidUnknownDeclarationInvocation(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, ``)

	_, err := inter.Invoke("test")
	assert.IsType(t, interpreter.NotDeclaredError{}, err)
}

func TestInterpretInvalidNonFunctionDeclarationInvocation(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
       let test = 1
   `)

	_, err := inter.Invoke("test")
	assert.IsType(t, interpreter.NotInvokableError{}, err)
}

func TestInterpretLexicalScope(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
       let x = 10

       fun f(): Int {
          // check resolution
          return x
       }

       fun g(): Int {
          // check scope is lexical, not dynamic
          let x = 20
          return f()
       }
    `)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(10),
		inter.Globals.Get("x").GetValue(inter),
	)

	value, err := inter.Invoke("f")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(10),
		value,
	)

	value, err = inter.Invoke("g")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(10),
		value,
	)
}

func TestInterpretFunctionSideEffects(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
       var value = 0

       fun test(_ newValue: Int) {
           value = newValue
       }
    `)

	newValue := interpreter.NewUnmeteredIntValueFromInt64(42)

	value, err := inter.Invoke("test", newValue)
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.Void,
		value,
	)

	AssertValuesEqual(
		t,
		inter,
		newValue,
		inter.Globals.Get("value").GetValue(inter),
	)
}

func TestInterpretNoHoisting(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
       let x = 2

       fun test(): Int {
          if x == 0 {
              let x = 3
              return x
          }
          return x
       }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(2),
		value,
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(2),
		inter.Globals.Get("x").GetValue(inter),
	)
}

func TestInterpretFunctionExpressionsAndScope(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
       let x = 10

       // check first-class functions and scope inside them
       let y = (fun (x: Int): Int { return x })(42)
    `)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(10),
		inter.Globals.Get("x").GetValue(inter),
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(42),
		inter.Globals.Get("y").GetValue(inter),
	)
}

func TestInterpretVariableAssignment(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
       fun test(): Int {
           var x = 2
           x = 3
           return x
       }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(3),
		value,
	)
}

func TestInterpretGlobalVariableAssignment(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
       var x = 2

       fun test(): Int {
           x = 3
           return x
       }
    `)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(2),
		inter.Globals.Get("x").GetValue(inter),
	)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(3),
		value,
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(3),
		inter.Globals.Get("x").GetValue(inter),
	)
}

func TestInterpretConstantRedeclaration(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
       let x = 2

       fun test(): Int {
           let x = 3
           return x
       }
    `)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(2),
		inter.Globals.Get("x").GetValue(inter),
	)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(3),
		value,
	)
}

func TestInterpretParameters(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
       fun returnA(a: Int, b: Int): Int {
           return a
       }

       fun returnB(a: Int, b: Int): Int {
           return b
       }
    `)

	a := interpreter.NewUnmeteredIntValueFromInt64(24)
	b := interpreter.NewUnmeteredIntValueFromInt64(42)

	value, err := inter.Invoke("returnA", a, b)
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter, a, value)

	value, err = inter.Invoke("returnB", a, b)
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter, b, value)
}

func TestInterpretArrayEquality(t *testing.T) {
	t.Parallel()

	testBooleanFunction := func(t *testing.T, name string, expected bool, innerCode string) {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			code := fmt.Sprintf("fun test(): Bool { \n %s \n }", innerCode)

			inter := parseCheckAndInterpret(t, code)
			res, err := inter.Invoke("test")

			require.NoError(t, err)

			boolVal, ok := res.(interpreter.BoolValue)
			require.True(t, ok)

			require.Equal(t, bool(boolVal), expected)
		})

	}

	// variable sized arrays
	nestingLimit := 4

	for i := 0; i < nestingLimit; i++ {
		nestingLevel := i
		array := fmt.Sprintf("%s 42 %s", strings.Repeat("[", nestingLevel), strings.Repeat("]", nestingLevel))

		for _, opStr := range []string{"==", "!="} {
			op := opStr
			testname := fmt.Sprintf("test variable size array %s at nesting level %d", op, nestingLevel)
			code := fmt.Sprintf(`
					let xs = %s
					return xs %s xs
				`,
				array,
				op,
			)

			testBooleanFunction(t, testname, op == "==", code)
		}
	}

	// fixed size arrays

	testBooleanFunction(t, "fixed array [Int; 3] should not equal a different array", false, `
		let xs: [Int; 3] = [1, 2, 3]
		let ys: [Int; 3] = [4, 5, 6]
		return xs == ys
	`)

	testBooleanFunction(t, "fixed array [Int; 3] should be unequal to a different array", true, `
		let xs: [Int; 3] = [1, 2, 3]
		let ys: [Int; 3] = [4, 5, 6]
		return xs != ys
	`)

	testBooleanFunction(t, "fixed array [[Int; 2]; 1] should equal itself", true, `
		let xs: [[Int; 2]; 1] = [[42, 1337]]
		return xs == xs
	`)
}

func TestInterpretArrayIndexing(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
       fun test(): Int {
           let z = [0, 3]
           return z[1]
       }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(3),
		value,
	)
}

func TestInterpretInvalidArrayIndexing(t *testing.T) {

	t.Parallel()

	for name, index := range map[string]int{
		"negative":          -1,
		"larger than count": 2,
	} {

		t.Run(name, func(t *testing.T) {

			inter := parseCheckAndInterpret(t, `
               fun test(_ index: Int): Int {
                   let z = [0, 3]
                   return z[index]
               }
            `)

			indexValue := interpreter.NewUnmeteredIntValueFromInt64(int64(index))
			_, err := inter.Invoke("test", indexValue)
			RequireError(t, err)

			var indexErr interpreter.ArrayIndexOutOfBoundsError
			require.ErrorAs(t, err, &indexErr)

			assert.Equal(t, index, indexErr.Index)
			assert.Equal(t, 2, indexErr.Size)
			assert.Equal(t,
				ast.Position{Offset: 106, Line: 4, Column: 26},
				indexErr.HasPosition.StartPosition(),
			)
			assert.Equal(t,
				ast.Position{Offset: 113, Line: 4, Column: 33},
				indexErr.HasPosition.EndPosition(nil),
			)
		})
	}
}

func TestInterpretArrayIndexingAssignment(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
       let z = [0, 3]

       fun test() {
           z[1] = 2
       }
    `)

	_, err := inter.Invoke("test")
	require.NoError(t, err)

	actualArray := inter.Globals.Get("z").GetValue(inter)

	expectedArray := interpreter.NewArrayValue(
		inter,
		interpreter.EmptyLocationRange,
		&interpreter.VariableSizedStaticType{
			Type: interpreter.PrimitiveStaticTypeInt,
		},
		common.ZeroAddress,
		interpreter.NewUnmeteredIntValueFromInt64(0),
		interpreter.NewUnmeteredIntValueFromInt64(2),
	)

	RequireValuesEqual(
		t,
		inter,
		expectedArray,
		actualArray,
	)
}

func TestInterpretInvalidArrayIndexingAssignment(t *testing.T) {

	t.Parallel()

	for name, index := range map[string]int{
		"negative":          -1,
		"larger than count": 2,
	} {

		t.Run(name, func(t *testing.T) {

			inter := parseCheckAndInterpret(t, `
               fun test(_ index: Int) {
                   let z = [0, 3]
                   z[index] = 1
               }
            `)

			indexValue := interpreter.NewUnmeteredIntValueFromInt64(int64(index))
			_, err := inter.Invoke("test", indexValue)
			RequireError(t, err)

			var indexErr interpreter.ArrayIndexOutOfBoundsError
			require.ErrorAs(t, err, &indexErr)

			assert.Equal(t, index, indexErr.Index)
			assert.Equal(t, 2, indexErr.Size)
			assert.Equal(t,
				ast.Position{Offset: 94, Line: 4, Column: 19},
				indexErr.HasPosition.StartPosition(),
			)
			assert.Equal(t,
				ast.Position{Offset: 101, Line: 4, Column: 26},
				indexErr.HasPosition.EndPosition(nil),
			)
		})
	}
}

func TestInterpretStringIndexing(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let a = "abc"
      let x = a[0]
      let y = a[1]
      let z = a[2]
    `)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredCharacterValue("a"),
		inter.Globals.Get("x").GetValue(inter),
	)
	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredCharacterValue("b"),
		inter.Globals.Get("y").GetValue(inter),
	)
	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredCharacterValue("c"),
		inter.Globals.Get("z").GetValue(inter),
	)
}

func TestInterpretInvalidStringIndexing(t *testing.T) {

	t.Parallel()

	for name, index := range map[string]int{
		"negative":          -1,
		"larger than count": 2,
	} {

		t.Run(name, func(t *testing.T) {

			inter := parseCheckAndInterpret(t, `
               fun test(_ index: Int) {
                   let x = "ab"
                   x[index]
               }
            `)

			indexValue := interpreter.NewUnmeteredIntValueFromInt64(int64(index))
			_, err := inter.Invoke("test", indexValue)
			RequireError(t, err)

			var indexErr interpreter.StringIndexOutOfBoundsError
			require.ErrorAs(t, err, &indexErr)

			assert.Equal(t, index, indexErr.Index)
			assert.Equal(t, 2, indexErr.Length)
			assert.Equal(t,
				ast.Position{Offset: 92, Line: 4, Column: 19},
				indexErr.HasPosition.StartPosition(),
			)
			assert.Equal(t,
				ast.Position{Offset: 99, Line: 4, Column: 26},
				indexErr.HasPosition.EndPosition(nil),
			)
		})
	}
}

func TestInterpretStringIndexingUnicode(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      fun testUnicodeA(): Character {
          let a = "caf\u{E9}"
          return a[3]
      }

      fun testUnicodeB(): Character {
        let b = "cafe\u{301}"
        return b[3]
      }
    `)

	value, err := inter.Invoke("testUnicodeA")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredCharacterValue("\u00e9"),
		value,
	)

	value, err = inter.Invoke("testUnicodeB")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredCharacterValue("e\u0301"),
		value,
	)
}

func TestInterpretStringSlicing(t *testing.T) {

	t.Parallel()

	range1 := ast.Range{
		StartPos: ast.Position{Offset: 116, Line: 4, Column: 31},
		EndPos:   ast.Position{Offset: 140, Line: 4, Column: 55},
	}

	range2 := ast.Range{
		StartPos: ast.Position{Offset: 116, Line: 4, Column: 31},
		EndPos:   ast.Position{Offset: 141, Line: 4, Column: 56},
	}

	type test struct {
		str        string
		from       int
		to         int
		result     string
		checkError func(t *testing.T, err error)
	}

	tests := []test{
		{"abcdef", 0, 6, "abcdef", nil},
		{"abcdef", 0, 0, "", nil},
		{"abcdef", 0, 1, "a", nil},
		{"abcdef", 0, 2, "ab", nil},
		{"abcdef", 1, 2, "b", nil},
		{"abcdef", 2, 3, "c", nil},
		{"abcdef", 5, 6, "f", nil},
		{"abcdef", 1, 6, "bcdef", nil},
		// Invalid indices
		{"abcdef", -1, 0, "", func(t *testing.T, err error) {
			var sliceErr interpreter.StringSliceIndicesError
			require.ErrorAs(t, err, &sliceErr)

			assert.Equal(t, -1, sliceErr.FromIndex)
			assert.Equal(t, 0, sliceErr.UpToIndex)
			assert.Equal(t, 6, sliceErr.Length)
			assert.Equal(t,
				range2.StartPos,
				sliceErr.LocationRange.StartPosition(),
			)
			assert.Equal(t,
				range2.EndPos,
				sliceErr.LocationRange.EndPosition(nil),
			)
		}},
		{"abcdef", 0, -1, "", func(t *testing.T, err error) {
			var sliceErr interpreter.StringSliceIndicesError
			require.ErrorAs(t, err, &sliceErr)

			assert.Equal(t, 0, sliceErr.FromIndex)
			assert.Equal(t, -1, sliceErr.UpToIndex)
			assert.Equal(t, 6, sliceErr.Length)
			assert.Equal(t,
				range2.StartPos,
				sliceErr.LocationRange.StartPosition(),
			)
			assert.Equal(t,
				range2.EndPos,
				sliceErr.LocationRange.EndPosition(nil),
			)
		}},
		{"abcdef", 0, 10, "", func(t *testing.T, err error) {
			var sliceErr interpreter.StringSliceIndicesError
			require.ErrorAs(t, err, &sliceErr)

			assert.Equal(t, 0, sliceErr.FromIndex)
			assert.Equal(t, 10, sliceErr.UpToIndex)
			assert.Equal(t, 6, sliceErr.Length)
			assert.Equal(t,
				range2.StartPos,
				sliceErr.LocationRange.StartPosition(),
			)
			assert.Equal(t,
				range2.EndPos,
				sliceErr.LocationRange.EndPosition(nil),
			)
		}},
		{"abcdef", 2, 1, "", func(t *testing.T, err error) {
			var indexErr interpreter.InvalidSliceIndexError
			require.ErrorAs(t, err, &indexErr)

			assert.Equal(t, 2, indexErr.FromIndex)
			assert.Equal(t, 1, indexErr.UpToIndex)
			assert.Equal(t,
				range1.StartPos,
				indexErr.LocationRange.StartPosition(),
			)
			assert.Equal(t,
				range1.EndPos,
				indexErr.LocationRange.EndPosition(nil),
			)
		}},
		// Unicode: indices are based on characters = grapheme clusters
		{"cafe\\u{301}b", 0, 5, "cafe\u0301b", nil},
		{"cafe\\u{301}ba\\u{308}", 0, 6, "cafe\u0301ba\u0308", nil},
		{"cafe\\u{301}ba\\u{308}be", 0, 8, "cafe\u0301ba\u0308be", nil},
		{"cafe\\u{301}b", 3, 5, "e\u0301b", nil},
		{"cafe\\u{301}ba\\u{308}", 3, 6, "e\u0301ba\u0308", nil},
		{"cafe\\u{301}ba\\u{308}be", 3, 8, "e\u0301ba\u0308be", nil},
		{"cafe\\u{301}b", 4, 5, "b", nil},
		{"cafe\\u{301}ba\\u{308}", 4, 6, "ba\u0308", nil},
		{"cafe\\u{301}ba\\u{308}be", 4, 8, "ba\u0308be", nil},
		{"cafe\\u{301}ba\\u{308}be", 3, 4, "e\u0301", nil},
		{"cafe\\u{301}ba\\u{308}be", 5, 6, "a\u0308", nil},
		{"tamil \\u{BA8}\\u{BBF} (ni)", 0, 7, "tamil \u0BA8\u0BBF", nil},
		{"tamil \\u{BA8}\\u{BBF} (ni)", 7, 12, " (ni)", nil},
	}

	runTest := func(test test) {

		name := fmt.Sprintf("%s, %d, %d", test.str, test.from, test.to)

		t.Run(name, func(t *testing.T) {

			t.Parallel()

			inter := parseCheckAndInterpret(t,
				fmt.Sprintf(
					`
                      fun test(): String {
                        let s = "%s"
                        return s.slice(from: %d, upTo: %d)
                      }
                    `,
					test.str,
					test.from,
					test.to,
				),
			)

			value, err := inter.Invoke("test")
			if test.checkError == nil {
				require.NoError(t, err)

				AssertValuesEqual(
					t,
					inter,
					interpreter.NewUnmeteredStringValue(test.result),
					value,
				)
			} else {
				require.IsType(t,
					interpreter.Error{},
					err,
				)

				test.checkError(t, err)
			}
		})
	}

	for _, test := range tests {
		runTest(test)
	}
}

func TestInterpretReturnWithoutExpression(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
       fun returnNothing() {
           return
       }
    `)

	value, err := inter.Invoke("returnNothing")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.Void,
		value,
	)
}

func TestInterpretReturns(t *testing.T) {

	t.Parallel()

	inter, err := parseCheckAndInterpretWithOptions(t,
		`
           access(all) fun returnEarly(): Int {
               return 2
               return 1
           }
        `,
		ParseCheckAndInterpretOptions{
			HandleCheckerError: func(err error) {
				errs := RequireCheckerErrors(t, err, 1)

				assert.IsType(t, &sema.UnreachableStatementError{}, errs[0])
			},
		},
	)
	require.NoError(t, err)

	value, err := inter.Invoke("returnEarly")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(2),
		value,
	)
}

func TestInterpretEqualOperator(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      fun testIntegersUnequal(): Bool {
          return 5 == 3
      }

      fun testIntegersEqual(): Bool {
          return 3 == 3
      }

      fun testTrueAndTrue(): Bool {
          return true == true
      }

      fun testTrueAndFalse(): Bool {
          return true == false
      }

      fun testFalseAndTrue(): Bool {
          return false == true
      }

      fun testFalseAndFalse(): Bool {
          return false == false
      }

      fun testEqualStrings(): Bool {
          return "123" == "123"
      }

      fun testUnequalStrings(): Bool {
          return "123" == "abc"
      }

      fun testUnicodeStrings(): Bool {
          return "caf\u{E9}" == "cafe\u{301}"
      }

      fun testEqualPaths(): Bool {
          // different domains
          return /public/foo == /public/foo &&
                 /private/bar == /private/bar &&
                 /storage/baz == /storage/baz
       }

       fun testUnequalPaths(): Bool {
          return /public/foo == /public/foofoo ||
                 /private/bar == /private/barbar ||
                 /storage/baz == /storage/bazbaz
       }

       fun testCastedPaths(): Bool {
          let foo: StoragePath = /storage/foo
          let bar: PublicPath = /public/foo
          return (foo as Path) == (bar as Path)
       }
    `)

	for name, expected := range map[string]bool{
		"testIntegersUnequal": false,
		"testIntegersEqual":   true,
		"testTrueAndTrue":     true,
		"testTrueAndFalse":    false,
		"testFalseAndTrue":    false,
		"testFalseAndFalse":   true,
		"testEqualStrings":    true,
		"testUnequalStrings":  false,
		"testUnicodeStrings":  true,
		"testEqualPaths":      true,
		"testUnequalPaths":    false,
		"testCastedPaths":     false,
	} {
		t.Run(name, func(t *testing.T) {
			value, err := inter.Invoke(name)
			require.NoError(t, err)

			AssertValuesEqual(
				t,
				inter,
				interpreter.BoolValue(expected),
				value,
			)
		})
	}
}

func TestInterpretUnequalOperator(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      fun testIntegersUnequal(): Bool {
          return 5 != 3
      }

      fun testIntegersEqual(): Bool {
          return 3 != 3
      }

      fun testTrueAndTrue(): Bool {
          return true != true
      }

      fun testTrueAndFalse(): Bool {
          return true != false
      }

      fun testFalseAndTrue(): Bool {
          return false != true
      }

      fun testFalseAndFalse(): Bool {
          return false != false
      }
    `)

	for name, expected := range map[string]bool{
		"testIntegersUnequal": true,
		"testIntegersEqual":   false,
		"testTrueAndTrue":     false,
		"testTrueAndFalse":    true,
		"testFalseAndTrue":    true,
		"testFalseAndFalse":   false,
	} {
		t.Run(name, func(t *testing.T) {
			value, err := inter.Invoke(name)
			require.NoError(t, err)

			AssertValuesEqual(
				t,
				inter,
				interpreter.BoolValue(expected),
				value,
			)
		})
	}
}

func TestInterpretLessOperator(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      fun testIntegersGreater(): Bool {
          return 5 < 3
      }

      fun testIntegersEqual(): Bool {
          return 3 < 3
      }

      fun testIntegersLess(): Bool {
          return 3 < 5
      }
    `)

	for name, expected := range map[string]bool{
		"testIntegersGreater": false,
		"testIntegersEqual":   false,
		"testIntegersLess":    true,
	} {
		t.Run(name, func(t *testing.T) {
			value, err := inter.Invoke(name)
			require.NoError(t, err)

			AssertValuesEqual(
				t,
				inter,
				interpreter.BoolValue(expected),
				value,
			)
		})
	}
}

func TestInterpretLessEqualOperator(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      fun testIntegersGreater(): Bool {
          return 5 <= 3
      }

      fun testIntegersEqual(): Bool {
          return 3 <= 3
      }

      fun testIntegersLess(): Bool {
          return 3 <= 5
      }
    `)

	for name, expected := range map[string]bool{
		"testIntegersGreater": false,
		"testIntegersEqual":   true,
		"testIntegersLess":    true,
	} {
		t.Run(name, func(t *testing.T) {
			value, err := inter.Invoke(name)
			require.NoError(t, err)

			AssertValuesEqual(
				t,
				inter,
				interpreter.BoolValue(expected),
				value,
			)
		})
	}
}

func TestInterpretGreaterOperator(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      fun testIntegersGreater(): Bool {
          return 5 > 3
      }

      fun testIntegersEqual(): Bool {
          return 3 > 3
      }

      fun testIntegersLess(): Bool {
          return 3 > 5
      }
    `)

	for name, expected := range map[string]bool{
		"testIntegersGreater": true,
		"testIntegersEqual":   false,
		"testIntegersLess":    false,
	} {
		t.Run(name, func(t *testing.T) {
			value, err := inter.Invoke(name)
			require.NoError(t, err)

			AssertValuesEqual(
				t,
				inter,
				interpreter.BoolValue(expected),
				value,
			)
		})
	}
}

func TestInterpretGreaterEqualOperator(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      fun testIntegersGreater(): Bool {
          return 5 >= 3
      }

      fun testIntegersEqual(): Bool {
          return 3 >= 3
      }

      fun testIntegersLess(): Bool {
          return 3 >= 5
      }
    `)

	for name, expected := range map[string]bool{
		"testIntegersGreater": true,
		"testIntegersEqual":   true,
		"testIntegersLess":    false,
	} {
		t.Run(name, func(t *testing.T) {
			value, err := inter.Invoke(name)
			require.NoError(t, err)

			AssertValuesEqual(
				t,
				inter,
				interpreter.BoolValue(expected),
				value,
			)
		})
	}
}

func TestInterpretOrOperator(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      fun testTrueTrue(): Bool {
          return true || true
      }

      fun testTrueFalse(): Bool {
          return true || false
      }

      fun testFalseTrue(): Bool {
          return false || true
      }

      fun testFalseFalse(): Bool {
          return false || false
      }
    `)

	for name, expected := range map[string]bool{
		"testTrueTrue":   true,
		"testTrueFalse":  true,
		"testFalseTrue":  true,
		"testFalseFalse": false,
	} {
		t.Run(name, func(t *testing.T) {
			value, err := inter.Invoke(name)
			require.NoError(t, err)

			AssertValuesEqual(
				t,
				inter,
				interpreter.BoolValue(expected),
				value,
			)
		})
	}
}

func TestInterpretOrOperatorShortCircuitLeftSuccess(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      var x = false
      var y = false

      fun changeX(): Bool {
          x = true
          return true
      }

      fun changeY(): Bool {
          y = true
          return true
      }

      let test = changeX() || changeY()
    `)

	AssertValuesEqual(
		t,
		inter,
		interpreter.TrueValue,
		inter.Globals.Get("test").GetValue(inter),
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.TrueValue,
		inter.Globals.Get("x").GetValue(inter),
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.FalseValue,
		inter.Globals.Get("y").GetValue(inter),
	)
}

func TestInterpretOrOperatorShortCircuitLeftFailure(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      var x = false
      var y = false

      fun changeX(): Bool {
          x = true
          return false
      }

      fun changeY(): Bool {
          y = true
          return true
      }

      let test = changeX() || changeY()
    `)

	AssertValuesEqual(
		t,
		inter,
		interpreter.TrueValue,
		inter.Globals.Get("test").GetValue(inter),
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.TrueValue,
		inter.Globals.Get("x").GetValue(inter),
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.TrueValue,
		inter.Globals.Get("y").GetValue(inter),
	)
}

func TestInterpretAndOperator(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      fun testTrueTrue(): Bool {
          return true && true
      }

      fun testTrueFalse(): Bool {
          return true && false
      }

      fun testFalseTrue(): Bool {
          return false && true
      }

      fun testFalseFalse(): Bool {
          return false && false
      }
    `)

	for name, expected := range map[string]bool{
		"testTrueTrue":   true,
		"testTrueFalse":  false,
		"testFalseTrue":  false,
		"testFalseFalse": false,
	} {
		t.Run(name, func(t *testing.T) {
			value, err := inter.Invoke(name)
			require.NoError(t, err)

			AssertValuesEqual(
				t,
				inter,
				interpreter.BoolValue(expected),
				value,
			)
		})
	}
}

func TestInterpretAndOperatorShortCircuitLeftSuccess(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      var x = false
      var y = false

      fun changeX(): Bool {
          x = true
          return true
      }

      fun changeY(): Bool {
          y = true
          return true
      }

      let test = changeX() && changeY()
    `)

	AssertValuesEqual(
		t,
		inter,
		interpreter.TrueValue,
		inter.Globals.Get("test").GetValue(inter),
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.TrueValue,
		inter.Globals.Get("x").GetValue(inter),
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.TrueValue,
		inter.Globals.Get("y").GetValue(inter),
	)
}

func TestInterpretAndOperatorShortCircuitLeftFailure(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      var x = false
      var y = false

      fun changeX(): Bool {
          x = true
          return false
      }

      fun changeY(): Bool {
          y = true
          return true
      }

      let test = changeX() && changeY()
    `)

	AssertValuesEqual(
		t,
		inter,
		interpreter.FalseValue,
		inter.Globals.Get("test").GetValue(inter),
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.TrueValue,
		inter.Globals.Get("x").GetValue(inter),
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.FalseValue,
		inter.Globals.Get("y").GetValue(inter),
	)
}

func TestInterpretExpressionStatement(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
       var x = 0

       fun incX() {
           x = x + 2
       }

       fun test(): Int {
           incX()
           return x
       }
    `)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(0),
		inter.Globals.Get("x").GetValue(inter),
	)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(2),
		value,
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(2),
		inter.Globals.Get("x").GetValue(inter),
	)
}

func TestInterpretConditionalOperator(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
       fun testTrue(): Int {
           return true ? 2 : 3
       }

       fun testFalse(): Int {
            return false ? 2 : 3
       }
    `)

	value, err := inter.Invoke("testTrue")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(2),
		value,
	)

	value, err = inter.Invoke("testFalse")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(3),
		value,
	)
}

func TestInterpretFunctionBindingInFunction(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      fun foo(): AnyStruct {
          return foo
      }
  `)

	_, err := inter.Invoke("foo")
	require.NoError(t, err)
}

func TestInterpretRecursionFib(t *testing.T) {

	t.Parallel()

	// mainly tests that the function declaration identifier is bound
	// to the function inside the function and that the arguments
	// of the function calls are evaluated in the call-site scope

	inter := parseCheckAndInterpret(t, `
       fun fib(_ n: Int): Int {
           if n < 2 {
              return n
           }
           return fib(n - 1) + fib(n - 2)
       }
   `)

	value, err := inter.Invoke(
		"fib",
		interpreter.NewUnmeteredIntValueFromInt64(14),
	)
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(377),
		value,
	)
}

func TestInterpretRecursionFactorial(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
        fun factorial(_ n: Int): Int {
            if n < 1 {
               return 1
            }

            return n * factorial(n - 1)
        }
   `)

	value, err := inter.Invoke(
		"factorial",
		interpreter.NewUnmeteredIntValueFromInt64(5),
	)
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(120),
		value,
	)
}

func TestInterpretUnaryIntegerNegation(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let x = -2
      let y = -(-2)
    `)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(-2),
		inter.Globals.Get("x").GetValue(inter),
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(2),
		inter.Globals.Get("y").GetValue(inter),
	)
}

func TestInterpretUnaryBooleanNegation(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let a = !true
      let b = !(!true)
      let c = !false
      let d = !(!false)
    `)

	AssertValuesEqual(
		t,
		inter,
		interpreter.FalseValue,
		inter.Globals.Get("a").GetValue(inter),
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.TrueValue,
		inter.Globals.Get("b").GetValue(inter),
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.TrueValue,
		inter.Globals.Get("c").GetValue(inter),
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.FalseValue,
		inter.Globals.Get("d").GetValue(inter),
	)
}

func TestInterpretHostFunction(t *testing.T) {

	t.Parallel()

	const code = `
      access(all) let a = test(1, 2)
    `
	program, err := parser.ParseProgram(nil, []byte(code), parser.Config{})

	require.NoError(t, err)

	testFunction := stdlib.NewStandardLibraryStaticFunction(
		"test",
		&sema.FunctionType{
			Parameters: []sema.Parameter{
				{
					Label:          sema.ArgumentLabelNotRequired,
					Identifier:     "a",
					TypeAnnotation: sema.IntTypeAnnotation,
				},
				{
					Label:          sema.ArgumentLabelNotRequired,
					Identifier:     "b",
					TypeAnnotation: sema.IntTypeAnnotation,
				},
			},
			ReturnTypeAnnotation: sema.IntTypeAnnotation,
		},
		``,
		func(invocation interpreter.Invocation) interpreter.Value {
			a := invocation.Arguments[0].(interpreter.IntValue).ToBigInt(nil)
			b := invocation.Arguments[1].(interpreter.IntValue).ToBigInt(nil)
			value := new(big.Int).Add(a, b)
			return interpreter.NewUnmeteredIntValueFromBigInt(value)
		},
	)

	baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
	baseValueActivation.DeclareValue(testFunction)

	checker, err := sema.NewChecker(
		program,
		TestLocation,
		nil,
		&sema.Config{
			BaseValueActivationHandler: func(_ common.Location) *sema.VariableActivation {
				return baseValueActivation
			},
			AccessCheckMode: sema.AccessCheckModeStrict,
		},
	)
	require.NoError(t, err)

	err = checker.Check()
	require.NoError(t, err)

	storage := newUnmeteredInMemoryStorage()

	baseActivation := activations.NewActivation(nil, interpreter.BaseActivation)
	interpreter.Declare(baseActivation, testFunction)

	inter, err := interpreter.NewInterpreter(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
		&interpreter.Config{
			Storage: storage,
			BaseActivationHandler: func(_ common.Location) *interpreter.VariableActivation {
				return baseActivation
			},
		},
	)
	require.NoError(t, err)

	err = inter.Interpret()
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(3),
		inter.Globals.Get("a").GetValue(inter),
	)
}

func TestInterpretHostFunctionWithVariableArguments(t *testing.T) {

	t.Parallel()

	const code = `
      access(all) let nothing = test(1, true, "test")
    `
	program, err := parser.ParseProgram(nil, []byte(code), parser.Config{})

	require.NoError(t, err)

	called := false

	testFunction := stdlib.NewStandardLibraryStaticFunction(
		"test",
		&sema.FunctionType{
			Parameters: []sema.Parameter{
				{
					Label:          sema.ArgumentLabelNotRequired,
					Identifier:     "value",
					TypeAnnotation: sema.IntTypeAnnotation,
				},
			},
			ReturnTypeAnnotation: sema.VoidTypeAnnotation,
			Arity:                &sema.Arity{Min: 1},
		},
		``,
		func(invocation interpreter.Invocation) interpreter.Value {
			called = true

			require.Len(t, invocation.ArgumentTypes, 3)
			assert.IsType(t, sema.IntType, invocation.ArgumentTypes[0])
			assert.IsType(t, sema.BoolType, invocation.ArgumentTypes[1])
			assert.IsType(t, sema.StringType, invocation.ArgumentTypes[2])

			require.Len(t, invocation.Arguments, 3)

			inter := invocation.Interpreter

			AssertValuesEqual(
				t,
				inter,
				interpreter.NewUnmeteredIntValueFromInt64(1),
				invocation.Arguments[0],
			)

			AssertValuesEqual(
				t,
				inter,
				interpreter.TrueValue,
				invocation.Arguments[1],
			)

			AssertValuesEqual(
				t,
				inter,
				interpreter.NewUnmeteredStringValue("test"),
				invocation.Arguments[2],
			)

			return interpreter.Void
		},
	)

	baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
	baseValueActivation.DeclareValue(testFunction)

	checker, err := sema.NewChecker(
		program,
		TestLocation,
		nil,
		&sema.Config{
			BaseValueActivationHandler: func(_ common.Location) *sema.VariableActivation {
				return baseValueActivation
			},
			AccessCheckMode: sema.AccessCheckModeStrict,
		},
	)
	require.NoError(t, err)

	err = checker.Check()
	require.NoError(t, err)

	storage := newUnmeteredInMemoryStorage()

	baseActivation := activations.NewActivation(nil, interpreter.BaseActivation)
	interpreter.Declare(baseActivation, testFunction)

	inter, err := interpreter.NewInterpreter(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
		&interpreter.Config{
			Storage: storage,
			BaseActivationHandler: func(_ common.Location) *interpreter.VariableActivation {
				return baseActivation
			},
		},
	)
	require.NoError(t, err)

	err = inter.Interpret()
	require.NoError(t, err)

	assert.True(t, called)
}

func TestInterpretHostFunctionWithOptionalArguments(t *testing.T) {

	t.Parallel()

	const code = `
      access(all) let nothing = test(1, true, "test")
    `
	program, err := parser.ParseProgram(nil, []byte(code), parser.Config{})

	require.NoError(t, err)

	called := false

	testFunction := stdlib.NewStandardLibraryStaticFunction(
		"test",
		&sema.FunctionType{
			Parameters: []sema.Parameter{
				{
					Label:          sema.ArgumentLabelNotRequired,
					Identifier:     "value",
					TypeAnnotation: sema.NewTypeAnnotation(sema.IntType),
				},
			},
			ReturnTypeAnnotation: sema.NewTypeAnnotation(
				sema.VoidType,
			),
			Arity: &sema.Arity{Min: 1, Max: 3},
		},
		``,
		func(invocation interpreter.Invocation) interpreter.Value {
			called = true

			require.Len(t, invocation.ArgumentTypes, 3)
			assert.IsType(t, sema.IntType, invocation.ArgumentTypes[0])
			assert.IsType(t, sema.BoolType, invocation.ArgumentTypes[1])
			assert.IsType(t, sema.StringType, invocation.ArgumentTypes[2])

			require.Len(t, invocation.Arguments, 3)

			inter := invocation.Interpreter

			AssertValuesEqual(
				t,
				inter,
				interpreter.NewUnmeteredIntValueFromInt64(1),
				invocation.Arguments[0],
			)

			AssertValuesEqual(
				t,
				inter,
				interpreter.TrueValue,
				invocation.Arguments[1],
			)

			AssertValuesEqual(
				t,
				inter,
				interpreter.NewUnmeteredStringValue("test"),
				invocation.Arguments[2],
			)

			return interpreter.Void
		},
	)

	baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
	baseValueActivation.DeclareValue(testFunction)

	checker, err := sema.NewChecker(
		program,
		TestLocation,
		nil,
		&sema.Config{
			BaseValueActivationHandler: func(_ common.Location) *sema.VariableActivation {
				return baseValueActivation
			},
			AccessCheckMode: sema.AccessCheckModeStrict,
		},
	)
	require.NoError(t, err)

	err = checker.Check()
	require.NoError(t, err)

	storage := newUnmeteredInMemoryStorage()

	baseActivation := activations.NewActivation(nil, interpreter.BaseActivation)
	interpreter.Declare(baseActivation, testFunction)

	inter, err := interpreter.NewInterpreter(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
		&interpreter.Config{
			Storage: storage,
			BaseActivationHandler: func(_ common.Location) *interpreter.VariableActivation {
				return baseActivation
			},
		},
	)
	require.NoError(t, err)

	err = inter.Interpret()
	require.NoError(t, err)

	assert.True(t, called)
}

func TestInterpretCompositeDeclaration(t *testing.T) {

	t.Parallel()

	test := func(compositeKind common.CompositeKind) {

		t.Run(compositeKind.Name(), func(t *testing.T) {

			t.Parallel()

			inter, err := parseCheckAndInterpretWithOptions(t,
				fmt.Sprintf(
					`
                       access(all) %[1]s Test {}

                       access(all) fun test(): %[2]sTest {
                           return %[3]s %[4]s Test%[5]s
                       }
                    `,
					compositeKind.Keyword(),
					compositeKind.Annotation(),
					compositeKind.MoveOperator(),
					compositeKind.ConstructionKeyword(),
					constructorArguments(compositeKind, ""),
				),
				ParseCheckAndInterpretOptions{
					Config: &interpreter.Config{
						ContractValueHandler: makeContractValueHandler(nil, nil, nil),
					},
				},
			)
			require.NoError(t, err)

			value, err := inter.Invoke("test")
			require.NoError(t, err)

			assert.IsType(t,
				&interpreter.CompositeValue{},
				value,
			)
		})
	}

	for _, compositeKind := range common.AllCompositeKinds {

		switch compositeKind {
		case common.CompositeKindContract,
			common.CompositeKindEvent,
			common.CompositeKindEnum,
			common.CompositeKindAttachment:

			continue
		}

		test(compositeKind)
	}
}

func TestInterpretStructureSelfUseInInitializer(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `

      struct Test {

          init() {
              self
          }
      }

      fun test() {
          Test()
      }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.Void,
		value,
	)
}

func TestInterpretStructureConstructorUseInInitializerAndFunction(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `

      struct Test {

          init() {
              Test
          }

          fun test(): Test {
              return Test()
          }
      }

      fun test(): Test {
          return Test()
      }

      fun test2(): Test {
          return Test().test()
      }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	assert.IsType(t,
		&interpreter.CompositeValue{},
		value,
	)

	value, err = inter.Invoke("test2")
	require.NoError(t, err)

	assert.IsType(t,
		&interpreter.CompositeValue{},
		value,
	)
}

func TestInterpretStructureSelfUseInFunction(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `

      struct Test {

          fun test() {
              self
          }
      }

      fun test() {
          Test().test()
      }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.Void,
		value,
	)
}

func TestInterpretStructureConstructorUseInFunction(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      struct Test {

          fun test() {
              Test
          }
      }

      fun test() {
          Test().test()
      }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.Void,
		value,
	)
}

func TestInterpretStructureDeclarationWithField(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `

      struct Test {
          var test: Int

          init(_ test: Int) {
              self.test = test
          }
      }

      fun test(test: Int): Int {
          let test = Test(test)
          return test.test
      }
    `)

	newValue := interpreter.NewUnmeteredIntValueFromInt64(42)

	value, err := inter.Invoke("test", newValue)
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter, newValue, value)
}

func TestInterpretStructureDeclarationWithFunction(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      var value = 0

      struct Test {
          fun test(_ newValue: Int) {
              value = newValue
          }
      }

      fun test(newValue: Int) {
          let test = Test()
          test.test(newValue)
      }
    `)

	newValue := interpreter.NewUnmeteredIntValueFromInt64(42)

	value, err := inter.Invoke("test", newValue)
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.Void,
		value,
	)

	AssertValuesEqual(
		t,
		inter, newValue, inter.Globals.Get("value").GetValue(inter))
}

func TestInterpretStructureFunctionCall(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      struct Test {
          fun foo(): Int {
              return 42
          }

          fun bar(): Int {
              return self.foo()
          }
      }

      let value = Test().bar()
    `)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(42),
		inter.Globals.Get("value").GetValue(inter),
	)
}

func TestInterpretStructureFieldAssignment(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      struct Test {
          var foo: Int

          init() {
              self.foo = 1
              let alsoSelf = self
              alsoSelf.foo = 2
          }

          fun test() {
              self.foo = 3
              let alsoSelf = self
              alsoSelf.foo = 4
          }
      }

      let test = Test()

      fun callTest() {
          test.test()
      }
    `)

	test := inter.Globals.Get("test").GetValue(inter).(*interpreter.CompositeValue)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(1),
		test.GetField(inter, "foo"),
	)

	value, err := inter.Invoke("callTest")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.Void,
		value,
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(3),
		test.GetField(inter, "foo"),
	)
}

func TestInterpretStructureInitializesConstant(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      struct Test {
          let foo: Int

          init() {
              self.foo = 42
          }
      }

      let test = Test()
    `)

	actual := inter.Globals.Get("test").GetValue(inter).(*interpreter.CompositeValue).
		GetMember(inter, interpreter.EmptyLocationRange, "foo")
	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(42),
		actual,
	)
}

func TestInterpretStructureFunctionMutatesSelf(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      struct Test {
          var foo: Int

          init() {
              self.foo = 0
          }

          fun inc() {
              self.foo = self.foo + 1
          }
      }

      fun test(): Int {
          let test = Test()
          test.inc()
          test.inc()
          return test.foo
      }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(2),
		value,
	)
}

func TestInterpretStructCopyOnDeclaration(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      struct Cat {
          var wasFed: Bool

          init() {
              self.wasFed = false
          }
      }

      fun test(): [Bool] {
          let cat = Cat()
          let kitty = cat
          kitty.wasFed = true
          return [cat.wasFed, kitty.wasFed]
      }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewArrayValue(
			inter,
			interpreter.EmptyLocationRange,
			&interpreter.VariableSizedStaticType{
				Type: interpreter.PrimitiveStaticTypeBool,
			},
			common.ZeroAddress,
			interpreter.FalseValue,
			interpreter.TrueValue,
		),
		value,
	)
}

func TestInterpretStructCopyOnDeclarationModifiedWithStructFunction(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      struct Cat {
          var wasFed: Bool

          init() {
              self.wasFed = false
          }

          fun feed() {
              self.wasFed = true
          }
      }

      fun test(): [Bool] {
          let cat = Cat()
          let kitty = cat
          kitty.feed()
          return [cat.wasFed, kitty.wasFed]
      }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewArrayValue(
			inter,
			interpreter.EmptyLocationRange,
			&interpreter.VariableSizedStaticType{
				Type: interpreter.PrimitiveStaticTypeBool,
			},
			common.ZeroAddress,
			interpreter.FalseValue,
			interpreter.TrueValue,
		),
		value,
	)
}

func TestInterpretStructCopyOnIdentifierAssignment(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      struct Cat {
          var wasFed: Bool

          init() {
              self.wasFed = false
          }
      }

      fun test(): [Bool] {
          var cat = Cat()
          let kitty = Cat()
          cat = kitty
          kitty.wasFed = true
          return [cat.wasFed, kitty.wasFed]
      }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewArrayValue(
			inter,
			interpreter.EmptyLocationRange,
			&interpreter.VariableSizedStaticType{
				Type: interpreter.PrimitiveStaticTypeBool,
			},
			common.ZeroAddress,
			interpreter.FalseValue,
			interpreter.TrueValue,
		),
		value,
	)
}

func TestInterpretStructCopyOnIndexingAssignment(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      struct Cat {
          var wasFed: Bool

          init() {
              self.wasFed = false
          }
      }

      fun test(): [Bool] {
          let cats = [Cat()]
          let kitty = Cat()
          cats[0] = kitty
          kitty.wasFed = true
          return [cats[0].wasFed, kitty.wasFed]
      }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewArrayValue(
			inter,
			interpreter.EmptyLocationRange,
			&interpreter.VariableSizedStaticType{
				Type: interpreter.PrimitiveStaticTypeBool,
			},
			common.ZeroAddress,
			interpreter.FalseValue,
			interpreter.TrueValue,
		),
		value,
	)
}

func TestInterpretStructCopyOnMemberAssignment(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      struct Cat {
          var wasFed: Bool

          init() {
              self.wasFed = false
          }
      }

      struct Carrier {
          var cat: Cat
          init(cat: Cat) {
              self.cat = cat
          }
      }

      fun test(): [Bool] {
          let carrier = Carrier(cat: Cat())
          let kitty = Cat()
          carrier.cat = kitty
          kitty.wasFed = true
          return [carrier.cat.wasFed, kitty.wasFed]
      }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewArrayValue(
			inter,
			interpreter.EmptyLocationRange,
			&interpreter.VariableSizedStaticType{
				Type: interpreter.PrimitiveStaticTypeBool,
			},
			common.ZeroAddress,
			interpreter.FalseValue,
			interpreter.TrueValue,
		),
		value,
	)
}

func TestInterpretStructCopyOnPassing(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      struct Cat {
          var wasFed: Bool

          init() {
              self.wasFed = false
          }
      }

      fun feed(cat: Cat) {
          cat.wasFed = true
      }

      fun test(): Bool {
          let kitty = Cat()
          feed(cat: kitty)
          return kitty.wasFed
      }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.FalseValue,
		value,
	)
}

func TestInterpretArrayCopy(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `

      fun change(_ numbers: [Int]): [Int] {
          numbers[0] = 1
          return numbers
      }

      fun test(): [Int] {
          let numbers = [0]
          let numbers2 = change(numbers)
          return [
              numbers[0],
              numbers2[0]
          ]
      }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewArrayValue(
			inter,
			interpreter.EmptyLocationRange,
			&interpreter.VariableSizedStaticType{
				Type: interpreter.PrimitiveStaticTypeInt,
			},
			common.ZeroAddress,
			interpreter.NewUnmeteredIntValueFromInt64(0),
			interpreter.NewUnmeteredIntValueFromInt64(1),
		),
		value,
	)
}

func TestInterpretStructCopyInArray(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      struct Foo {
          var bar: Int
          init(bar: Int) {
              self.bar = bar
          }
      }

      fun test(): [Int] {
        let foo = Foo(bar: 1)
        let foos = [foo, foo]
        foo.bar = 2
        foos[0].bar = 3
        return [foo.bar, foos[0].bar, foos[1].bar]
      }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewArrayValue(
			inter,
			interpreter.EmptyLocationRange,
			&interpreter.VariableSizedStaticType{
				Type: interpreter.PrimitiveStaticTypeInt,
			},
			common.ZeroAddress,
			interpreter.NewUnmeteredIntValueFromInt64(2),
			interpreter.NewUnmeteredIntValueFromInt64(3),
			interpreter.NewUnmeteredIntValueFromInt64(1),
		),
		value,
	)
}

func TestInterpretMutuallyRecursiveFunctions(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      fun isEven(_ n: Int): Bool {
          if n == 0 {
              return true
          }
          return isOdd(n - 1)
      }

      fun isOdd(_ n: Int): Bool {
          if n == 0 {
              return false
          }
          return isEven(n - 1)
      }
    `)

	four := interpreter.NewUnmeteredIntValueFromInt64(4)

	value, err := inter.Invoke("isEven", four)
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.TrueValue,
		value,
	)

	value, err = inter.Invoke("isOdd", four)
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.FalseValue,
		value,
	)
}

func TestInterpretUseBeforeDeclaration(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      var tests = 0

      fun test(): Test {
          return Test()
      }

      struct Test {
         init() {
             tests = tests + 1
         }
      }
    `)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(0),
		inter.Globals.Get("tests").GetValue(inter),
	)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	assert.IsType(t,
		&interpreter.CompositeValue{},
		value,
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(1),
		inter.Globals.Get("tests").GetValue(inter),
	)

	value, err = inter.Invoke("test")
	require.NoError(t, err)

	assert.IsType(t,
		&interpreter.CompositeValue{},
		value,
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(2),
		inter.Globals.Get("tests").GetValue(inter),
	)
}

func TestInterpretOptionalVariableDeclaration(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let x: Int?? = 2
    `)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredSomeValueNonCopying(
			interpreter.NewUnmeteredSomeValueNonCopying(
				interpreter.NewUnmeteredIntValueFromInt64(2),
			),
		),
		inter.Globals.Get("x").GetValue(inter),
	)
}

func TestInterpretOptionalParameterInvokedExternal(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      fun test(x: Int??): Int?? {
          return x
      }
    `)

	value, err := inter.Invoke(
		"test",
		interpreter.NewUnmeteredIntValueFromInt64(2),
	)
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredSomeValueNonCopying(
			interpreter.NewUnmeteredSomeValueNonCopying(
				interpreter.NewUnmeteredIntValueFromInt64(2),
			),
		),
		value,
	)
}

func TestInterpretOptionalParameterInvokedInternal(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      fun testActual(x: Int??): Int?? {
          return x
      }

      fun test(): Int?? {
          return testActual(x: 2)
      }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredSomeValueNonCopying(
			interpreter.NewUnmeteredSomeValueNonCopying(
				interpreter.NewUnmeteredIntValueFromInt64(2),
			),
		),
		value,
	)
}

func TestInterpretOptionalReturn(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      fun test(x: Int): Int?? {
          return x
      }
    `)

	value, err := inter.Invoke("test", interpreter.NewUnmeteredIntValueFromInt64(2))
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredSomeValueNonCopying(
			interpreter.NewUnmeteredSomeValueNonCopying(
				interpreter.NewUnmeteredIntValueFromInt64(2),
			),
		),
		value,
	)
}

func TestInterpretOptionalAssignment(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      var x: Int?? = 1

      fun test() {
          x = 2
      }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.Void,
		value,
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredSomeValueNonCopying(
			interpreter.NewUnmeteredSomeValueNonCopying(
				interpreter.NewUnmeteredIntValueFromInt64(2),
			),
		),
		inter.Globals.Get("x").GetValue(inter),
	)
}

func TestInterpretNil(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
     let x: Int? = nil
   `)

	AssertValuesEqual(
		t,
		inter,
		interpreter.Nil,
		inter.Globals.Get("x").GetValue(inter),
	)
}

func TestInterpretOptionalNestingNil(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
     let x: Int?? = nil
   `)

	AssertValuesEqual(
		t,
		inter,
		interpreter.Nil,
		inter.Globals.Get("x").GetValue(inter),
	)
}

func TestInterpretNilReturnValue(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
     fun test(): Int?? {
         return nil
     }
   `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.Nil,
		value,
	)
}

func TestInterpretSomeReturnValue(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
     fun test(): Int? {
         let x: Int? = 1
         return x
     }
   `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredSomeValueNonCopying(
			interpreter.NewUnmeteredIntValueFromInt64(1),
		),
		value,
	)
}

func TestInterpretSomeReturnValueFromDictionary(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
     fun test(): Int? {
         let foo: {String: Int} = {"a": 1}
         return foo["a"]
     }
   `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredSomeValueNonCopying(
			interpreter.NewUnmeteredIntValueFromInt64(1),
		),
		value,
	)
}

func TestInterpretNilCoalescingNilIntToOptional(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let one = 1
      let none: Int? = nil
      let x: Int? = none ?? one
    `)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredSomeValueNonCopying(
			interpreter.NewUnmeteredIntValueFromInt64(1),
		),
		inter.Globals.Get("x").GetValue(inter),
	)
}

func TestInterpretNilCoalescingNilIntToOptionals(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let one = 1
      let none: Int?? = nil
      let x: Int? = none ?? one
    `)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredSomeValueNonCopying(
			interpreter.NewUnmeteredIntValueFromInt64(1),
		),
		inter.Globals.Get("x").GetValue(inter),
	)
}

func TestInterpretNilCoalescingNilIntToOptionalNilLiteral(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let one = 1
      let x: Int? = nil ?? one
    `)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredSomeValueNonCopying(
			interpreter.NewUnmeteredIntValueFromInt64(1),
		),
		inter.Globals.Get("x").GetValue(inter),
	)
}

func TestInterpretNilCoalescingRightSubtype(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let x: Int? = nil ?? nil
    `)

	AssertValuesEqual(
		t,
		inter,
		interpreter.Nil,
		inter.Globals.Get("x").GetValue(inter),
	)
}

func TestInterpretNilCoalescingNilInt(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let one = 1
      let none: Int? = nil
      let x: Int = none ?? one
    `)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(1),
		inter.Globals.Get("x").GetValue(inter),
	)
}

func TestInterpretNilCoalescingNilLiteralInt(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let one = 1
      let x: Int = nil ?? one
    `)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(1),
		inter.Globals.Get("x").GetValue(inter),
	)
}

func TestInterpretNilCoalescingShortCircuitLeftSuccess(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      var x = false
      var y = false

      fun changeX(): Int? {
          x = true
          return 1
      }

      fun changeY(): Int {
          y = true
          return 2
      }

      let test = changeX() ?? changeY()
    `)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(1),
		inter.Globals.Get("test").GetValue(inter),
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.TrueValue,
		inter.Globals.Get("x").GetValue(inter),
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.FalseValue,
		inter.Globals.Get("y").GetValue(inter),
	)
}

func TestInterpretNilCoalescingShortCircuitLeftFailure(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      var x = false
      var y = false

      fun changeX(): Int? {
          x = true
          return nil
      }

      fun changeY(): Int {
          y = true
          return 2
      }

      let test = changeX() ?? changeY()
    `)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(2),
		inter.Globals.Get("test").GetValue(inter),
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.TrueValue,
		inter.Globals.Get("x").GetValue(inter),
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.TrueValue,
		inter.Globals.Get("y").GetValue(inter),
	)
}

func TestInterpretNilCoalescingOptionalAnyStructNil(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let x: AnyStruct? = nil
      let y = x ?? true
    `)

	AssertValuesEqual(
		t,
		inter,
		interpreter.TrueValue,
		inter.Globals.Get("y").GetValue(inter),
	)
}

func TestInterpretNilCoalescingOptionalAnyStructSome(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let x: AnyStruct? = 2
      let y = x ?? true
    `)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(2),
		inter.Globals.Get("y").GetValue(inter),
	)
}

func TestInterpretNilCoalescingOptionalRightHandSide(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let x: Int? = 1
      let y: Int? = 2
      let z = x ?? y
    `)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredSomeValueNonCopying(
			interpreter.NewUnmeteredIntValueFromInt64(1),
		),
		inter.Globals.Get("z").GetValue(inter),
	)
}

func TestInterpretNilCoalescingBothOptional(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
     let x: Int?? = 1
     let y: Int? = 2
     let z = x ?? y
   `)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredSomeValueNonCopying(
			interpreter.NewUnmeteredIntValueFromInt64(1),
		),
		inter.Globals.Get("z").GetValue(inter),
	)
}

func TestInterpretNilCoalescingBothOptionalLeftNil(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
     let x: Int?? = nil
     let y: Int? = 2
     let z = x ?? y
   `)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredSomeValueNonCopying(
			interpreter.NewUnmeteredIntValueFromInt64(2),
		),
		inter.Globals.Get("z").GetValue(inter),
	)
}

func TestInterpretNilsComparison(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let x = nil == nil
   `)

	AssertValuesEqual(
		t,
		inter,
		interpreter.TrueValue,
		inter.Globals.Get("x").GetValue(inter),
	)
}

func TestInterpretNonOptionalNilComparison(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let x: Int = 1
      let y = x == nil
      let z = nil == x
   `)

	AssertValuesEqual(
		t,
		inter,
		interpreter.FalseValue,
		inter.Globals.Get("y").GetValue(inter),
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.FalseValue,
		inter.Globals.Get("z").GetValue(inter),
	)
}

func TestInterpretOptionalNilComparison(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
     let x: Int? = 1
     let y = x == nil
   `)

	AssertValuesEqual(
		t,
		inter,
		interpreter.FalseValue,
		inter.Globals.Get("y").GetValue(inter),
	)
}

func TestInterpretNestedOptionalNilComparison(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let x: Int?? = 1
      let y = x == nil
    `)

	AssertValuesEqual(
		t,
		inter,
		interpreter.FalseValue,
		inter.Globals.Get("y").GetValue(inter),
	)
}

func TestInterpretOptionalNilComparisonSwapped(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let x: Int? = 1
      let y = nil == x
    `)

	AssertValuesEqual(
		t,
		inter,
		interpreter.FalseValue,
		inter.Globals.Get("y").GetValue(inter),
	)
}

func TestInterpretNestedOptionalNilComparisonSwapped(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let x: Int?? = 1
      let y = nil == x
    `)

	AssertValuesEqual(
		t,
		inter,
		interpreter.FalseValue,
		inter.Globals.Get("y").GetValue(inter),
	)
}

func TestInterpretNestedOptionalComparisonNils(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let x: Int? = nil
      let y: Int?? = nil
      let z = x == y
    `)

	AssertValuesEqual(
		t,
		inter,
		interpreter.TrueValue,
		inter.Globals.Get("z").GetValue(inter),
	)
}

func TestInterpretNestedOptionalComparisonValues(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let x: Int? = 2
      let y: Int?? = 2
      let z = x == y
    `)

	AssertValuesEqual(
		t,
		inter,
		interpreter.TrueValue,
		inter.Globals.Get("z").GetValue(inter),
	)
}

func TestInterpretNestedOptionalComparisonMixed(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let x: Int? = 2
      let y: Int?? = nil
      let z = x == y
    `)

	AssertValuesEqual(
		t,
		inter,
		interpreter.FalseValue,
		inter.Globals.Get("z").GetValue(inter),
	)
}

func TestInterpretOptionalSomeValueComparison(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
     let x: Int? = 1
     let y = x == 1
   `)

	AssertValuesEqual(
		t,
		inter,
		interpreter.TrueValue,
		inter.Globals.Get("y").GetValue(inter),
	)
}

func TestInterpretOptionalNilValueComparison(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
     let x: Int? = nil
     let y = x == 1
   `)

	AssertValuesEqual(
		t,
		inter,
		interpreter.FalseValue,
		inter.Globals.Get("y").GetValue(inter),
	)
}

func TestInterpretOptionalMap(t *testing.T) {

	t.Parallel()

	t.Run("some", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          let one: Int? = 42
          let result = one.map(fun (v: Int): String {
              return v.toString()
          })
        `)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredSomeValueNonCopying(
				interpreter.NewUnmeteredStringValue("42"),
			),
			inter.Globals.Get("result").GetValue(inter),
		)
	})

	t.Run("nil", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          let none: Int? = nil
          let result = none.map(fun (v: Int): String {
              return v.toString()
          })
        `)

		AssertValuesEqual(
			t,
			inter,
			interpreter.Nil,
			inter.Globals.Get("result").GetValue(inter),
		)
	})

	t.Run("box and convert argument", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          struct S {
              fun map(f: fun(AnyStruct): String): String {
                  return "S.map"
              }
          }

          fun test(): String?? {
              let s: S? = S()
              // NOTE: The outer map has a parameter of type S? instead of just S
              return s.map(fun(s2: S?): String? {
                  // The inner map should call Optional.map, not S.map,
                  // because s2 is S?, not S
                  return s2.map(fun(s3: AnyStruct): String {
                      return "Optional.map"
                  })
              })
          }
        `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(t,
			inter,
			interpreter.NewSomeValueNonCopying(
				nil,
				interpreter.NewSomeValueNonCopying(
					nil,
					interpreter.NewUnmeteredStringValue("Optional.map"),
				),
			),
			value,
		)
	})
}

func TestInterpretCompositeNilEquality(t *testing.T) {

	t.Parallel()

	test := func(compositeKind common.CompositeKind) {

		t.Run(compositeKind.Name(), func(t *testing.T) {

			t.Parallel()

			var setupCode, identifier string
			if compositeKind == common.CompositeKindContract {
				identifier = "X"
			} else {
				setupCode = fmt.Sprintf(
					`access(all) let x: %[1]sX? %[2]s %[3]s X%[4]s`,
					compositeKind.Annotation(),
					compositeKind.TransferOperator(),
					compositeKind.ConstructionKeyword(),
					constructorArguments(compositeKind, ""),
				)
				identifier = "x"
			}

			body := "{}"
			if compositeKind == common.CompositeKindEnum {
				body = "{ case a }"
			}

			conformances := ""
			if compositeKind == common.CompositeKindEnum {
				conformances = ": Int"
			}

			inter, err := parseCheckAndInterpretWithOptions(t,
				fmt.Sprintf(
					`
                      access(all) %[1]s X%[2]s %[3]s

                      %[4]s

                      access(all) let y = %[5]s == nil
                      access(all) let z = nil == %[5]s
                    `,
					compositeKind.Keyword(),
					conformances,
					body,
					setupCode,
					identifier,
				),
				ParseCheckAndInterpretOptions{
					Config: &interpreter.Config{
						ContractValueHandler: makeContractValueHandler(nil, nil, nil),
					},
				},
			)
			require.NoError(t, err)

			AssertValuesEqual(
				t,
				inter,
				interpreter.FalseValue,
				inter.Globals.Get("y").GetValue(inter),
			)

			AssertValuesEqual(
				t,
				inter,
				interpreter.FalseValue,
				inter.Globals.Get("z").GetValue(inter),
			)
		})
	}

	for _, compositeKind := range common.AllCompositeKinds {

		if compositeKind == common.CompositeKindEvent ||
			compositeKind == common.CompositeKindAttachment ||
			compositeKind == common.CompositeKindContract {
			continue
		}

		test(compositeKind)
	}
}

func TestInterpretInterfaceConformanceNoRequirements(t *testing.T) {

	t.Parallel()

	for _, compositeKind := range common.AllCompositeKinds {

		if compositeKind == common.CompositeKindContract {
			continue
		}

		if !compositeKind.SupportsInterfaces() {
			continue
		}

		interfaceType := "{Test}"

		t.Run(compositeKind.Keyword(), func(t *testing.T) {

			inter := parseCheckAndInterpret(t,
				fmt.Sprintf(
					`
                      access(all) %[1]s interface Test {}

                      access(all) %[1]s TestImpl: Test {}

                      access(all) let test: %[2]s%[3]s %[4]s %[5]s TestImpl%[6]s
                    `,
					compositeKind.Keyword(),
					compositeKind.Annotation(),
					interfaceType,
					compositeKind.TransferOperator(),
					compositeKind.ConstructionKeyword(),
					constructorArguments(compositeKind, ""),
				),
			)

			assert.IsType(t,
				&interpreter.CompositeValue{},
				inter.Globals.Get("test").GetValue(inter),
			)
		})
	}
}

func TestInterpretInterfaceFieldUse(t *testing.T) {

	t.Parallel()

	for _, compositeKind := range common.CompositeKindsWithFieldsAndFunctions {

		if !compositeKind.SupportsInterfaces() {
			continue
		}

		var setupCode, identifier string
		if compositeKind == common.CompositeKindContract {
			identifier = "TestImpl"
		} else {
			interfaceType := "{Test}"

			setupCode = fmt.Sprintf(
				`access(all) let test: %[1]s%[2]s %[3]s %[4]s TestImpl%[5]s`,
				compositeKind.Annotation(),
				interfaceType,
				compositeKind.TransferOperator(),
				compositeKind.ConstructionKeyword(),
				constructorArguments(compositeKind, "x: 1"),
			)
			identifier = "test"
		}

		t.Run(compositeKind.Keyword(), func(t *testing.T) {

			inter, err := parseCheckAndInterpretWithOptions(t,
				fmt.Sprintf(
					`
                      access(all) %[1]s interface Test {
                          access(all) x: Int
                      }

                      access(all) %[1]s TestImpl: Test {
                          access(all) var x: Int

                          init(x: Int) {
                              self.x = x
                          }
                      }

                      %[2]s

                      access(all) let x = %[3]s.x
                    `,
					compositeKind.Keyword(),
					setupCode,
					identifier,
				),
				ParseCheckAndInterpretOptions{
					Config: &interpreter.Config{
						ContractValueHandler: makeContractValueHandler(
							[]interpreter.Value{
								interpreter.NewUnmeteredIntValueFromInt64(1),
							},
							[]sema.Type{
								sema.IntType,
							},
							[]sema.Type{
								sema.IntType,
							},
						),
					},
				},
			)
			require.NoError(t, err)

			AssertValuesEqual(
				t,
				inter,
				interpreter.NewUnmeteredIntValueFromInt64(1),
				inter.Globals.Get("x").GetValue(inter),
			)
		})
	}
}

func TestInterpretInterfaceFunctionUse(t *testing.T) {

	t.Parallel()

	for _, compositeKind := range common.CompositeKindsWithFieldsAndFunctions {

		if !compositeKind.SupportsInterfaces() {
			continue
		}

		var setupCode, identifier string
		if compositeKind == common.CompositeKindContract {
			identifier = "TestImpl"
		} else {
			interfaceType := "{Test}"

			setupCode = fmt.Sprintf(
				`access(all) let test: %[1]s %[2]s %[3]s %[4]s TestImpl%[5]s`,
				compositeKind.Annotation(),
				interfaceType,
				compositeKind.TransferOperator(),
				compositeKind.ConstructionKeyword(),
				constructorArguments(compositeKind, ""),
			)
			identifier = "test"
		}

		t.Run(compositeKind.Keyword(), func(t *testing.T) {

			inter, err := parseCheckAndInterpretWithOptions(t,
				fmt.Sprintf(
					`
                      access(all) %[1]s interface Test {
                          access(all) fun test(): Int
                      }

                      access(all) %[1]s TestImpl: Test {
                          access(all) fun test(): Int {
                              return 2
                          }
                      }

                      %[2]s

                      access(all) let val = %[3]s.test()
                    `,
					compositeKind.Keyword(),
					setupCode,
					identifier,
				),
				ParseCheckAndInterpretOptions{
					Config: &interpreter.Config{
						ContractValueHandler: makeContractValueHandler(nil, nil, nil),
					},
				},
			)
			require.NoError(t, err)

			AssertValuesEqual(
				t,
				inter,
				interpreter.NewUnmeteredIntValueFromInt64(2),
				inter.Globals.Get("val").GetValue(inter),
			)
		})
	}
}

func TestInterpretImport(t *testing.T) {

	t.Parallel()

	importedChecker, err := ParseAndCheckWithOptions(t,
		`
          access(all) fun answer(): Int {
              return 42
          }
        `,
		ParseAndCheckOptions{
			Location: ImportedLocation,
		},
	)
	require.NoError(t, err)

	importingChecker, err := ParseAndCheckWithOptions(t,
		`
          import answer from "imported"

          access(all) fun test(): Int {
              return answer()
          }
        `,
		ParseAndCheckOptions{
			Config: &sema.Config{
				ImportHandler: func(_ *sema.Checker, importedLocation common.Location, _ ast.Range) (sema.Import, error) {
					assert.Equal(t,
						ImportedLocation,
						importedLocation,
					)

					return sema.ElaborationImport{
						Elaboration: importedChecker.Elaboration,
					}, nil
				},
			},
		},
	)
	require.NoError(t, err)

	storage := newUnmeteredInMemoryStorage()

	inter, err := interpreter.NewInterpreter(
		interpreter.ProgramFromChecker(importingChecker),
		importingChecker.Location,
		&interpreter.Config{
			Storage: storage,
			ImportLocationHandler: func(inter *interpreter.Interpreter, location common.Location) interpreter.Import {
				assert.Equal(t,
					ImportedLocation,
					location,
				)

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
	)
	require.NoError(t, err)

	err = inter.Interpret()
	require.NoError(t, err)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(42),
		value,
	)
}

func TestInterpretImportError(t *testing.T) {

	t.Parallel()

	const importedLocation1 = common.StringLocation("imported1")
	const importedLocation2 = common.StringLocation("imported2")

	var importedChecker1, importedChecker2 *sema.Checker

	baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
	baseValueActivation.DeclareValue(stdlib.PanicFunction)

	parseAndCheck := func(code string, location common.Location) *sema.Checker {
		checker, err := ParseAndCheckWithOptions(t,
			code,
			ParseAndCheckOptions{
				Location: location,
				Config: &sema.Config{
					BaseValueActivationHandler: func(_ common.Location) *sema.VariableActivation {
						return baseValueActivation
					},
					ImportHandler: func(_ *sema.Checker, importedLocation common.Location, _ ast.Range) (sema.Import, error) {
						switch importedLocation {
						case importedLocation1:
							return sema.ElaborationImport{
								Elaboration: importedChecker1.Elaboration,
							}, nil
						case importedLocation2:
							return sema.ElaborationImport{
								Elaboration: importedChecker2.Elaboration,
							}, nil
						default:
							assert.FailNow(t, "invalid location")
							return nil, nil
						}
					},
				},
			},
		)
		require.NoError(t, err)
		return checker
	}

	const importedCode1 = `
      access(all) fun realAnswer(): Int {
          return panic("?!")
      }
    `

	importedChecker1 = parseAndCheck(importedCode1, importedLocation1)

	const importedCode2 = `
       import realAnswer from "imported1"

      access(all) fun answer(): Int {
          return realAnswer()
      }
    `

	importedChecker2 = parseAndCheck(importedCode2, importedLocation2)

	const code = `
      import answer from "imported2"

      access(all) fun test(): Int {
          return answer()
      }
    `

	mainChecker := parseAndCheck(code, TestLocation)

	baseActivation := activations.NewActivation(nil, interpreter.BaseActivation)
	interpreter.Declare(baseActivation, stdlib.PanicFunction)

	storage := newUnmeteredInMemoryStorage()

	inter, err := interpreter.NewInterpreter(
		interpreter.ProgramFromChecker(mainChecker),
		mainChecker.Location,
		&interpreter.Config{
			Storage: storage,
			BaseActivationHandler: func(_ common.Location) *interpreter.VariableActivation {
				return baseActivation
			},
			ImportLocationHandler: func(inter *interpreter.Interpreter, location common.Location) interpreter.Import {
				var importedChecker *sema.Checker
				switch location {
				case importedLocation1:
					importedChecker = importedChecker1
				case importedLocation2:
					importedChecker = importedChecker2
				default:
					assert.FailNow(t, "invalid location")
				}

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
	)
	require.NoError(t, err)

	err = inter.Interpret()
	require.NoError(t, err)

	_, err = inter.Invoke("test")

	var sb strings.Builder
	printErr := pretty.NewErrorPrettyPrinter(&sb, false).
		PrettyPrintError(
			err,
			mainChecker.Location,
			map[common.Location][]byte{
				TestLocation:      []byte(code),
				importedLocation1: []byte(importedCode1),
				importedLocation2: []byte(importedCode2),
			},
		)
	require.NoError(t, printErr)
	assert.Equal(t,
		" --> test:5:17\n"+
			"  |\n"+
			"5 |           return answer()\n"+
			"  |                  ^^^^^^^^\n"+
			"\n"+
			" --> imported2:5:17\n"+
			"  |\n"+
			"5 |           return realAnswer()\n"+
			"  |                  ^^^^^^^^^^^^\n"+
			"\n"+
			"error: panic: ?!\n"+
			" --> imported1:3:17\n"+
			"  |\n"+
			"3 |           return panic(\"?!\")\n"+
			"  |                  ^^^^^^^^^^^\n",
		sb.String(),
	)
	RequireError(t, err)

	var panicErr stdlib.PanicError
	require.ErrorAs(t, err, &panicErr)

	assert.Equal(t,
		"?!",
		panicErr.Message,
	)
}

func TestInterpretDictionary(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let x = {"a": 1, "b": 2}
    `)

	expectedDict := interpreter.NewDictionaryValue(
		inter,
		interpreter.EmptyLocationRange,
		&interpreter.DictionaryStaticType{
			KeyType:   interpreter.PrimitiveStaticTypeString,
			ValueType: interpreter.PrimitiveStaticTypeInt,
		},
		interpreter.NewUnmeteredStringValue("a"), interpreter.NewUnmeteredIntValueFromInt64(1),
		interpreter.NewUnmeteredStringValue("b"), interpreter.NewUnmeteredIntValueFromInt64(2),
	)

	actualDict := inter.Globals.Get("x").GetValue(inter)

	AssertValuesEqual(
		t,
		inter,
		expectedDict,
		actualDict,
	)
}

func TestInterpretDictionaryInsertionOrder(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let x = {"c": 3, "a": 1, "b": 2}
    `)

	expectedDict := interpreter.NewDictionaryValue(
		inter,
		interpreter.EmptyLocationRange,
		&interpreter.DictionaryStaticType{
			KeyType:   interpreter.PrimitiveStaticTypeString,
			ValueType: interpreter.PrimitiveStaticTypeInt,
		},
		interpreter.NewUnmeteredStringValue("c"), interpreter.NewUnmeteredIntValueFromInt64(3),
		interpreter.NewUnmeteredStringValue("a"), interpreter.NewUnmeteredIntValueFromInt64(1),
		interpreter.NewUnmeteredStringValue("b"), interpreter.NewUnmeteredIntValueFromInt64(2),
	)

	actualDict := inter.Globals.Get("x").GetValue(inter)

	AssertValuesEqual(
		t,
		inter,
		expectedDict,
		actualDict,
	)
}

func TestInterpretDictionaryIndexingString(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let x = {"abc": 1, "def": 2}
      let a = x["abc"]
      let b = x["def"]
      let c = x["ghi"]
    `)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredSomeValueNonCopying(
			interpreter.NewUnmeteredIntValueFromInt64(1),
		),
		inter.Globals.Get("a").GetValue(inter),
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredSomeValueNonCopying(
			interpreter.NewUnmeteredIntValueFromInt64(2),
		),
		inter.Globals.Get("b").GetValue(inter),
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.Nil,
		inter.Globals.Get("c").GetValue(inter),
	)
}

func TestInterpretDictionaryIndexingBool(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let x = {true: 1, false: 2}
      let a = x[true]
      let b = x[false]
    `)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredSomeValueNonCopying(
			interpreter.NewUnmeteredIntValueFromInt64(1),
		),
		inter.Globals.Get("a").GetValue(inter),
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredSomeValueNonCopying(
			interpreter.NewUnmeteredIntValueFromInt64(2),
		),
		inter.Globals.Get("b").GetValue(inter),
	)
}

func TestInterpretDictionaryIndexingInt(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let x = {23: "a", 42: "b"}
      let a = x[23]
      let b = x[42]
      let c = x[100]
    `)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredSomeValueNonCopying(
			interpreter.NewUnmeteredStringValue("a"),
		),
		inter.Globals.Get("a").GetValue(inter),
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredSomeValueNonCopying(
			interpreter.NewUnmeteredStringValue("b"),
		),
		inter.Globals.Get("b").GetValue(inter),
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.Nil,
		inter.Globals.Get("c").GetValue(inter),
	)
}

func TestInterpretDictionaryIndexingType(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      struct TestStruct {}
      resource TestResource {}

      let x: {Type: String} = {
        Type<Int16>(): "a",
        Type<String>(): "b",
        Type<AnyStruct>(): "c",
        Type<@TestResource>(): "f"
      }

      let a = x[Type<Int16>()]
      let b = x[Type<String>()]
      let c = x[Type<AnyStruct>()]
      let d = x[Type<Int>()]
      let e = x[Type<TestStruct>()]
      let f = x[Type<@TestResource>()]
    `)

	assert.Equal(t,
		interpreter.NewUnmeteredSomeValueNonCopying(
			interpreter.NewUnmeteredStringValue("a"),
		),
		inter.Globals.Get("a").GetValue(inter),
	)

	assert.Equal(t,
		interpreter.NewUnmeteredSomeValueNonCopying(
			interpreter.NewUnmeteredStringValue("b"),
		),
		inter.Globals.Get("b").GetValue(inter),
	)

	assert.Equal(t,
		interpreter.NewUnmeteredSomeValueNonCopying(
			interpreter.NewUnmeteredStringValue("c"),
		),
		inter.Globals.Get("c").GetValue(inter),
	)

	assert.Equal(t,
		interpreter.Nil,
		inter.Globals.Get("d").GetValue(inter),
	)

	// types need to match exactly, subtypes won't cut it
	assert.Equal(t,
		interpreter.Nil,
		inter.Globals.Get("e").GetValue(inter),
	)

	assert.Equal(t,
		interpreter.NewUnmeteredSomeValueNonCopying(
			interpreter.NewUnmeteredStringValue("f"),
		),
		inter.Globals.Get("f").GetValue(inter),
	)
}

func TestInterpretDictionaryIndexingAssignmentExisting(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let x = {"abc": 42}
      fun test() {
          x["abc"] = 23
      }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.Void,
		value,
	)

	actualValue := inter.Globals.Get("x").GetValue(inter)
	actualDict := actualValue.(*interpreter.DictionaryValue)

	newValue := actualDict.GetKey(
		inter,
		interpreter.EmptyLocationRange,
		interpreter.NewUnmeteredStringValue("abc"),
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredSomeValueNonCopying(interpreter.NewUnmeteredIntValueFromInt64(23)),
		newValue,
	)

	AssertValueSlicesEqual(
		t,
		inter,
		[]interpreter.Value{
			interpreter.NewUnmeteredStringValue("abc"),
			interpreter.NewUnmeteredIntValueFromInt64(23),
		},
		DictionaryKeyValues(inter, actualDict),
	)
}

func TestInterpretDictionaryIndexingAssignmentNew(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let x = {"def": 42}
      fun test() {
          x["abc"] = 23
      }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.Void,
		value,
	)

	expectedDict := interpreter.NewDictionaryValue(
		inter,
		interpreter.EmptyLocationRange,
		&interpreter.DictionaryStaticType{
			KeyType:   interpreter.PrimitiveStaticTypeString,
			ValueType: interpreter.PrimitiveStaticTypeInt,
		},
		interpreter.NewUnmeteredStringValue("def"), interpreter.NewUnmeteredIntValueFromInt64(42),
		interpreter.NewUnmeteredStringValue("abc"), interpreter.NewUnmeteredIntValueFromInt64(23),
	)

	actualDict := inter.Globals.Get("x").GetValue(inter).(*interpreter.DictionaryValue)

	AssertValuesEqual(
		t,
		inter,
		expectedDict,
		actualDict,
	)

	newValue := actualDict.GetKey(
		inter,
		interpreter.EmptyLocationRange,
		interpreter.NewUnmeteredStringValue("abc"),
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredSomeValueNonCopying(interpreter.NewUnmeteredIntValueFromInt64(23)),
		newValue,
	)

	AssertValueSlicesEqual(
		t,
		inter,
		[]interpreter.Value{
			interpreter.NewUnmeteredStringValue("abc"),
			interpreter.NewUnmeteredIntValueFromInt64(23),
			interpreter.NewUnmeteredStringValue("def"),
			interpreter.NewUnmeteredIntValueFromInt64(42),
		},
		DictionaryKeyValues(inter, actualDict),
	)
}

func TestInterpretDictionaryIndexingAssignmentNil(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let x = {"def": 42, "abc": 23}
      fun test() {
          x["def"] = nil
      }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.Void,
		value,
	)

	expectedDict := interpreter.NewDictionaryValue(
		inter,
		interpreter.EmptyLocationRange,
		&interpreter.DictionaryStaticType{
			KeyType:   interpreter.PrimitiveStaticTypeString,
			ValueType: interpreter.PrimitiveStaticTypeInt,
		},
		interpreter.NewUnmeteredStringValue("abc"), interpreter.NewUnmeteredIntValueFromInt64(23),
	)

	actualDict := inter.Globals.Get("x").GetValue(inter).(*interpreter.DictionaryValue)

	RequireValuesEqual(
		t,
		inter,
		expectedDict,
		actualDict,
	)

	newValue := actualDict.GetKey(
		inter,
		interpreter.EmptyLocationRange,
		interpreter.NewUnmeteredStringValue("def"),
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.Nil,
		newValue,
	)

	AssertValueSlicesEqual(
		t,
		inter,
		[]interpreter.Value{
			interpreter.NewUnmeteredStringValue("abc"),
			interpreter.NewUnmeteredIntValueFromInt64(23),
		},
		DictionaryKeyValues(inter, actualDict),
	)
}

func TestInterpretDictionaryEquality(t *testing.T) {
	t.Parallel()

	testBooleanFunction := func(t *testing.T, name string, expected bool, innerCode string) {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			code := fmt.Sprintf("fun test(): Bool { \n %s \n }", innerCode)

			inter := parseCheckAndInterpret(t, code)
			res, err := inter.Invoke("test")

			require.NoError(t, err)

			boolVal, ok := res.(interpreter.BoolValue)
			require.True(t, ok)

			require.Equal(t, bool(boolVal), expected)
		})

	}

	for _, opStr := range []string{"==", "!="} {
		testBooleanFunction(
			t,
			"dictionary should be equal to itself",
			opStr == "==",
			fmt.Sprintf(
				`
					let d = {"abc": 1, "def": 2}
					return d %s d
				`,
				opStr,
			),
		)

		testBooleanFunction(
			t,
			"nested dictionary should be equal to itself",
			opStr == "==",
			fmt.Sprintf(
				`
					let d = {"abc": {1: {"a": 1000}, 2: {"b": 2000}}, "def": {4: {"c": 1000}, 5: {"d": 2000}}}
					return d %s d
				`,
				opStr,
			),
		)

		testBooleanFunction(
			t,
			"simple dictionary equality",
			opStr == "==",
			fmt.Sprintf(
				`
					let d = {"abc": 1, "def": 2}
					let d2 = {"abc": 1, "def": 2}
					return d %s d2
				`,
				opStr,
			),
		)

		testBooleanFunction(
			t,
			"nested dictionary equality check",
			opStr == "==",
			fmt.Sprintf(
				`
				let d = {"abc": {1: {"a": 1000}, 2: {"b": 2000}}, "def": {4: {"c": 1000}, 5: {"d": 2000}}}
				let d2 = {"abc": {1: {"a": 1000}, 2: {"b": 2000}}, "def": {4: {"c": 1000}, 5: {"d": 2000}}}
				return d %s d2
				`,
				opStr,
			),
		)

		testBooleanFunction(
			t,
			"simple dictionary unequal",
			opStr == "!=",
			fmt.Sprintf(
				`
				let d = {"abc": 1, "def": 2}
				let d2 = {"abc": 1, "def": 2, "xyz": 4}
				return d %s d2
				`,
				opStr,
			),
		)

		testBooleanFunction(
			t,
			"nested dictionary unequal",
			opStr == "!=",
			fmt.Sprintf(
				`
					let d = {"abc": {1: {"a": 1000}, 2: {"b": 2000}}, "def": {4: {"c": 1000}, 5: {"d": 2000}}}
					let d2 = {"abc": {1: {"a": 1000}, 2: {"c": 1000}}, "def": {4: {"c": 1000}, 5: {"d": 2000}}}
					return d %s d2
				`,
				opStr,
			),
		)
	}
}

func TestInterpretComparison(t *testing.T) {
	t.Parallel()

	runBooleanTest := func(t *testing.T, name string, expected bool, innerCode string) {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			code := fmt.Sprintf("fun test(): Bool { \n %s \n }", innerCode)

			inter := parseCheckAndInterpret(t, code)
			res, err := inter.Invoke("test")

			require.NoError(t, err)

			boolVal, ok := res.(interpreter.BoolValue)
			require.True(t, ok)

			require.Equal(t, expected, bool(boolVal))
		})
	}

	tests := []struct {
		name     string
		expected bool
		inner    string
	}{
		{"true < true", false, "return true < true"},
		{"true <= true", true, "return true <= true"},
		{"true > true", false, "return true > true"},
		{"true >= true", true, "return true >= true"},
		{"false < false", false, "return false < false"},
		{"false <= false", true, "return false <= false"},
		{"false > false", false, "return false > false"},
		{"false >= false", true, "return false >= false"},
		{"true < false", false, "return true < false"},
		{"true <= false", false, "return true <= false"},
		{"true > false", true, "return true > false"},
		{"true >= false", true, "return true >= false"},
		{"false < true", true, "return false < true"},
		{"false <= true", true, "return false <= true"},
		{"false > true", false, "return false > true"},
		{"false >= true", false, "return false >= true"},
		{"a < b", true, "let left: Character = \"a\";\nlet right: Character = \"b\"; return left < right"},
		{"b < a", false, "let left: Character = \"b\";\nlet right: Character = \"a\"; return left < right"},
		{"a < A", false, "let left: Character = \"a\";\nlet right: Character = \"A\"; return left < right"},
		{"A < a", true, "let left: Character = \"A\";\nlet right: Character = \"a\"; return left < right"},
		{"A < Z", true, "let left: Character = \"A\";\nlet right: Character = \"Z\"; return left < right"},
		{"a <= b", true, "let left: Character = \"a\";\nlet right: Character = \"b\"; return left <= right"},
		{"a <= a", true, "let left: Character = \"a\";\nlet right: Character = \"a\"; return left <= right"},
		{"A <= a", true, "let left: Character = \"A\";\nlet right: Character = \"a\"; return left <= right"},
		{"a > b", false, "let left: Character = \"a\";\nlet right: Character = \"b\"; return left > right"},
		{"b > a", true, "let left: Character = \"b\";\nlet right: Character = \"a\"; return left > right"},
		{"A > a", false, "let left: Character = \"A\";\nlet right: Character = \"a\"; return left > right"},
		{"a >= b", false, "let left: Character = \"a\";\nlet right: Character = \"b\"; return left >= right"},
		{"a >= a", true, "let left: Character = \"a\";\nlet right: Character = \"a\"; return left >= right"},
		{"A >= a", false, "let left: Character = \"A\";\nlet right: Character = \"a\"; return left >= right"},
		{"\"\" < \"\"", false, "let left: String = \"\";\nlet right: String = \"\"; return left < right"},
		{"\"\" <= \"\"", true, "let left: String = \"\";\nlet right: String = \"\"; return left <= right"},
		{"\"\" > \"\"", false, "let left: String = \"\";\nlet right: String = \"\"; return left > right"},
		{"\"\" >= \"\"", true, "let left: String = \"\";\nlet right: String = \"\"; return left >= right"},
		{"\"\" < \"a\"", true, "let left: String = \"\";\nlet right: String = \"a\"; return left < right"},
		{"\"\" <= \"a\"", true, "let left: String = \"\";\nlet right: String = \"a\"; return left <= right"},
		{"\"\" > \"a\"", false, "let left: String = \"\";\nlet right: String = \"a\"; return left > right"},
		{"\"\" >= \"a\"", false, "let left: String = \"\";\nlet right: String = \"a\"; return left >= right"},
		{"\"az\" < \"b\"", true, "let left: String = \"az\";\nlet right: String = \"b\"; return left < right"},
		{"\"az\" <= \"b\"", true, "let left: String = \"az\";\nlet right: String = \"b\"; return left <= right"},
		{"\"az\" > \"b\"", false, "let left: String = \"az\";\nlet right: String = \"b\"; return left > right"},
		{"\"az\" >= \"b\"", false, "let left: String = \"az\";\nlet right: String = \"b\"; return left >= right"},
		{"\"xAB\" < \"Xab\"", false, "let left: String = \"xAB\";\nlet right: String = \"Xab\"; return left < right"},
		{"\"xAB\" <= \"Xab\"", false, "let left: String = \"xAB\";\nlet right: String = \"Xab\"; return left <= right"},
		{"\"xAB\" > \"Xab\"", true, "let left: String = \"xAB\";\nlet right: String = \"Xab\"; return left > right"},
		{"\"xAB\" >= \"Xab\"", true, "let left: String = \"xAB\";\nlet right: String = \"Xab\"; return left >= right"},
	}

	for _, test := range tests {
		runBooleanTest(t, test.name, test.expected, test.inner)
	}
}

func TestInterpretOptionalAnyStruct(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let x: AnyStruct? = 42
    `)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredSomeValueNonCopying(
			interpreter.NewUnmeteredIntValueFromInt64(42),
		),
		inter.Globals.Get("x").GetValue(inter),
	)
}

func TestInterpretOptionalAnyStructFailableCasting(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let x: AnyStruct? = 42
      let y = (x ?? 23) as? Int
    `)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredSomeValueNonCopying(
			interpreter.NewUnmeteredIntValueFromInt64(42),
		),
		inter.Globals.Get("x").GetValue(inter),
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredSomeValueNonCopying(
			interpreter.NewUnmeteredIntValueFromInt64(42),
		),
		inter.Globals.Get("y").GetValue(inter),
	)
}

func TestInterpretOptionalAnyStructFailableCastingInt(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let x: AnyStruct? = 23
      let y = x ?? 42
      let z = y as? Int
    `)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredSomeValueNonCopying(
			interpreter.NewUnmeteredIntValueFromInt64(23),
		),
		inter.Globals.Get("x").GetValue(inter),
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(23),
		inter.Globals.Get("y").GetValue(inter),
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredSomeValueNonCopying(
			interpreter.NewUnmeteredIntValueFromInt64(23),
		),
		inter.Globals.Get("z").GetValue(inter),
	)
}

func TestInterpretOptionalAnyStructFailableCastingNil(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let x: AnyStruct? = nil
      let y = x ?? 42
      let z = y as? Int
    `)

	AssertValuesEqual(
		t,
		inter,
		interpreter.Nil,
		inter.Globals.Get("x").GetValue(inter),
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(42),
		inter.Globals.Get("y").GetValue(inter),
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredSomeValueNonCopying(
			interpreter.NewUnmeteredIntValueFromInt64(42),
		),
		inter.Globals.Get("z").GetValue(inter),
	)
}

func TestInterpretReferenceFailableDowncasting(t *testing.T) {

	t.Parallel()

	t.Run("ephemeral", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          resource interface RI {}

          resource R: RI {}

		  entitlement E

          fun testValidUnauthorized(): Bool {
              let r  <- create R()
              let ref: AnyStruct = &r as &{RI}
              let ref2 = ref as? &R
              let isNil = ref2 == nil
              destroy r
              return isNil
          }

          fun testValidAuthorized(): Bool {
              let r  <- create R()
              let ref: AnyStruct = &r as auth(E) &{RI}
              let ref2 = ref as? &R
              let isNil = ref2 == nil
              destroy r
              return isNil
          }

          fun testValidIntersection(): Bool {
              let r  <- create R()
              let ref: AnyStruct = &r as &{RI}
              let ref2 = ref as? &{RI}
              let isNil = ref2 == nil
              destroy r
              return isNil
          }
        `)

		result, err := inter.Invoke("testValidUnauthorized")
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.FalseValue,
			result,
		)

		result, err = inter.Invoke("testValidAuthorized")
		require.NoError(t, err)

		assert.IsType(t,
			interpreter.BoolValue(false),
			result,
		)

		result, err = inter.Invoke("testValidIntersection")
		require.NoError(t, err)

		assert.IsType(t,
			interpreter.BoolValue(false),
			result,
		)
	})

	t.Run("storage", func(t *testing.T) {

		t.Parallel()

		var inter *interpreter.Interpreter

		getType := func(name string) sema.Type {
			variable, ok := inter.Program.Elaboration.GetGlobalType(name)
			require.True(t, ok, "missing global type %s", name)
			return variable.Type
		}

		// Inject a function that returns a storage reference value,
		// which is borrowed as:
		// - `&{RI}` (unauthorized, if argument for parameter `authorized` == false)
		// - `auth(E) &{RI}` (authorized, if argument for parameter `authorized` == true)

		storageAddress := common.MustBytesToAddress([]byte{0x42})
		storagePath := interpreter.PathValue{
			Domain:     common.PathDomainStorage,
			Identifier: "test",
		}

		getStorageReferenceFunctionType := &sema.FunctionType{
			Parameters: []sema.Parameter{
				{
					Label:          "authorized",
					Identifier:     "authorized",
					TypeAnnotation: sema.BoolTypeAnnotation,
				},
			},
			ReturnTypeAnnotation: sema.AnyStructTypeAnnotation,
		}

		valueDeclaration := stdlib.NewStandardLibraryStaticFunction(
			"getStorageReference",
			getStorageReferenceFunctionType,
			"",
			func(invocation interpreter.Invocation) interpreter.Value {
				authorized := bool(invocation.Arguments[0].(interpreter.BoolValue))

				var auth = interpreter.UnauthorizedAccess
				if authorized {
					auth = interpreter.ConvertSemaAccessToStaticAuthorization(
						invocation.Interpreter,
						sema.NewEntitlementSetAccess(
							[]*sema.EntitlementType{getType("E").(*sema.EntitlementType)},
							sema.Conjunction,
						),
					)
				}

				riType := getType("RI").(*sema.InterfaceType)

				return &interpreter.StorageReferenceValue{
					Authorization:        auth,
					TargetStorageAddress: storageAddress,
					TargetPath:           storagePath,
					BorrowedType: &sema.IntersectionType{
						Types: []*sema.InterfaceType{
							riType,
						},
					},
				}
			},
		)

		baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
		baseValueActivation.DeclareValue(valueDeclaration)

		baseActivation := activations.NewActivation(nil, interpreter.BaseActivation)
		interpreter.Declare(baseActivation, valueDeclaration)

		storage := newUnmeteredInMemoryStorage()

		var err error
		inter, err = parseCheckAndInterpretWithOptions(t,
			`
	              resource interface RI {}

	              resource R: RI {}

				  entitlement E

	              fun createR(): @R {
	                  return <- create R()
	              }

	              fun testValidUnauthorized(): &R? {
	                  let ref: AnyStruct = getStorageReference(authorized: false)
	                  return ref as? &R
	              }

	              fun testValidAuthorized(): &R? {
	                  let ref: AnyStruct = getStorageReference(authorized: true)
	                  return ref as? &R
	              }

	              fun testValidIntersection(): &{RI}? {
	                  let ref: AnyStruct = getStorageReference(authorized: false)
	                  return ref as? &{RI}
	              }
	            `,
			ParseCheckAndInterpretOptions{
				CheckerConfig: &sema.Config{
					BaseValueActivationHandler: func(_ common.Location) *sema.VariableActivation {
						return baseValueActivation
					},
				},
				Config: &interpreter.Config{
					Storage: storage,
					BaseActivationHandler: func(_ common.Location) *interpreter.VariableActivation {
						return baseActivation
					},
				},
			},
		)
		require.NoError(t, err)

		r, err := inter.Invoke("createR")
		require.NoError(t, err)

		r = r.Transfer(
			inter,
			interpreter.EmptyLocationRange,
			atree.Address(storageAddress),
			true,
			nil,
			nil,
			true, // r is standalone.
		)

		domain := storagePath.Domain.StorageDomain()
		storageMap := storage.GetDomainStorageMap(inter, storageAddress, domain, true)
		storageMapKey := interpreter.StringStorageMapKey(storagePath.Identifier)
		storageMap.WriteValue(inter, storageMapKey, r)

		result, err := inter.Invoke("testValidUnauthorized")
		require.NoError(t, err)

		assert.IsType(t,
			&interpreter.SomeValue{},
			result,
		)

		result, err = inter.Invoke("testValidAuthorized")
		require.NoError(t, err)

		assert.IsType(t,
			&interpreter.SomeValue{},
			result,
		)

		result, err = inter.Invoke("testValidIntersection")
		require.NoError(t, err)

		assert.IsType(t,
			&interpreter.SomeValue{},
			result,
		)
	})
}

func TestInterpretArrayLength(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let y = [1, 2, 3].length
    `)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(3),
		inter.Globals.Get("y").GetValue(inter),
	)
}

func TestInterpretStringLength(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let x = "cafe\u{301}".length
      let y = x
      let z = "\u{1F3F3}\u{FE0F}\u{200D}\u{1F308}".length
    `)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(4),
		inter.Globals.Get("x").GetValue(inter),
	)
	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(4),
		inter.Globals.Get("y").GetValue(inter),
	)
	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(1),
		inter.Globals.Get("z").GetValue(inter),
	)
}

func TestInterpretStructureFunctionBindingInside(t *testing.T) {

	t.Parallel()

	// TODO: replace AnyStruct return types with (X#(): X),
	//   and test case once bound function types are supported:
	//
	//   fun test(): X {
	//        let x = X()
	//        let bar = x.foo()
	//        return bar()
	//   }

	inter := parseCheckAndInterpret(t, `
        struct X {
            fun foo(): AnyStruct {
                return self.bar
            }

            fun bar(): X {
                return self
            }
        }

        fun test(): AnyStruct {
            let x = X()
            return x.foo()
        }
    `)

	functionValue, err := inter.Invoke("test")
	require.NoError(t, err)

	value, err := inter.InvokeFunctionValue(
		functionValue.(interpreter.FunctionValue),
		nil,
		nil,
		nil,
		nil,
		nil,
	)
	require.NoError(t, err)

	assert.IsType(t,
		&interpreter.CompositeValue{},
		value,
	)
}

func TestInterpretStructureFunctionBindingOutside(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
        struct X {
            fun foo(): X {
                return self
            }
        }

        fun test(): X {
            let x = X()
            let bar = x.foo
            return bar()
        }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	assert.IsType(t,
		&interpreter.CompositeValue{},
		value,
	)
}

func TestInterpretArrayAppend(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let xs = [1, 2, 3]

      fun test() {
          xs.append(4)
      }
    `)

	_, err := inter.Invoke("test")
	require.NoError(t, err)

	actualArray := inter.Globals.Get("xs").GetValue(inter)

	arrayValue := actualArray.(*interpreter.ArrayValue)
	AssertValueSlicesEqual(
		t,
		inter,
		[]interpreter.Value{
			interpreter.NewUnmeteredIntValueFromInt64(1),
			interpreter.NewUnmeteredIntValueFromInt64(2),
			interpreter.NewUnmeteredIntValueFromInt64(3),
			interpreter.NewUnmeteredIntValueFromInt64(4),
		},
		ArrayElements(inter, arrayValue),
	)
}

func TestInterpretArrayAppendBound(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      fun test(): [Int] {
          let x = [1, 2, 3]
          let y = x.append
          y(4)
          return x
      }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	arrayValue := value.(*interpreter.ArrayValue)
	AssertValueSlicesEqual(
		t,
		inter,
		[]interpreter.Value{
			interpreter.NewUnmeteredIntValueFromInt64(1),
			interpreter.NewUnmeteredIntValueFromInt64(2),
			interpreter.NewUnmeteredIntValueFromInt64(3),
			interpreter.NewUnmeteredIntValueFromInt64(4),
		},
		ArrayElements(inter, arrayValue),
	)
}

func TestInterpretArrayAppendAll(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      fun test(): [Int] {
          let a = [1, 2]
          a.appendAll([3, 4])
          return a
      }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	arrayValue := value.(*interpreter.ArrayValue)
	AssertValueSlicesEqual(
		t,
		inter,
		[]interpreter.Value{
			interpreter.NewUnmeteredIntValueFromInt64(1),
			interpreter.NewUnmeteredIntValueFromInt64(2),
			interpreter.NewUnmeteredIntValueFromInt64(3),
			interpreter.NewUnmeteredIntValueFromInt64(4),
		},
		ArrayElements(inter, arrayValue),
	)
}

func TestInterpretArrayAppendAllBound(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      fun test(): [Int] {
          let a = [1, 2]
          let b = a.appendAll
          b([3, 4])
          return a
      }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	arrayValue := value.(*interpreter.ArrayValue)
	AssertValueSlicesEqual(
		t,
		inter,
		[]interpreter.Value{
			interpreter.NewUnmeteredIntValueFromInt64(1),
			interpreter.NewUnmeteredIntValueFromInt64(2),
			interpreter.NewUnmeteredIntValueFromInt64(3),
			interpreter.NewUnmeteredIntValueFromInt64(4),
		},
		ArrayElements(inter, arrayValue),
	)
}

func TestInterpretArrayConcat(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      fun test(): [Int] {
          let a = [1, 2]
          return a.concat([3, 4])
      }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	arrayValue := value.(*interpreter.ArrayValue)
	AssertValueSlicesEqual(
		t,
		inter,
		[]interpreter.Value{
			interpreter.NewUnmeteredIntValueFromInt64(1),
			interpreter.NewUnmeteredIntValueFromInt64(2),
			interpreter.NewUnmeteredIntValueFromInt64(3),
			interpreter.NewUnmeteredIntValueFromInt64(4),
		},
		ArrayElements(inter, arrayValue),
	)
}

func TestInterpretArrayConcatBound(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      fun test(): [Int] {
          let a = [1, 2]
          let b = a.concat
          return b([3, 4])
      }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	arrayValue := value.(*interpreter.ArrayValue)
	AssertValueSlicesEqual(
		t,
		inter,
		[]interpreter.Value{
			interpreter.NewUnmeteredIntValueFromInt64(1),
			interpreter.NewUnmeteredIntValueFromInt64(2),
			interpreter.NewUnmeteredIntValueFromInt64(3),
			interpreter.NewUnmeteredIntValueFromInt64(4),
		},
		ArrayElements(inter, arrayValue),
	)
}

func TestInterpretArrayConcatDoesNotModifyOriginalArray(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      fun test(): [Int] {
          let a = [1, 2]
          a.concat([3, 4])
          return a
      }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	arrayValue := value.(*interpreter.ArrayValue)
	AssertValueSlicesEqual(
		t,
		inter,
		[]interpreter.Value{
			interpreter.NewUnmeteredIntValueFromInt64(1),
			interpreter.NewUnmeteredIntValueFromInt64(2),
		},
		ArrayElements(inter, arrayValue),
	)
}

func TestInterpretArrayInsert(t *testing.T) {

	t.Parallel()

	type testCase struct {
		name           string
		index          int
		expectedValues []interpreter.Value
	}

	for _, testCase := range []testCase{
		{
			name:  "start",
			index: 0,
			expectedValues: []interpreter.Value{
				interpreter.NewUnmeteredIntValueFromInt64(100),
				interpreter.NewUnmeteredIntValueFromInt64(1),
				interpreter.NewUnmeteredIntValueFromInt64(2),
				interpreter.NewUnmeteredIntValueFromInt64(3),
			},
		},
		{
			name:  "middle",
			index: 1,
			expectedValues: []interpreter.Value{
				interpreter.NewUnmeteredIntValueFromInt64(1),
				interpreter.NewUnmeteredIntValueFromInt64(100),
				interpreter.NewUnmeteredIntValueFromInt64(2),
				interpreter.NewUnmeteredIntValueFromInt64(3),
			},
		},
		{
			name:  "end",
			index: 3,
			expectedValues: []interpreter.Value{
				interpreter.NewUnmeteredIntValueFromInt64(1),
				interpreter.NewUnmeteredIntValueFromInt64(2),
				interpreter.NewUnmeteredIntValueFromInt64(3),
				interpreter.NewUnmeteredIntValueFromInt64(100),
			},
		},
	} {

		t.Run(testCase.name, func(t *testing.T) {

			inter := parseCheckAndInterpret(t, `
              let x = [1, 2, 3]

              fun test(_ index: Int) {
                  x.insert(at: index, 100)
              }
            `)

			_, err := inter.Invoke("test", interpreter.NewUnmeteredIntValueFromInt64(int64(testCase.index)))
			require.NoError(t, err)

			actualArray := inter.Globals.Get("x").GetValue(inter)

			require.IsType(t, &interpreter.ArrayValue{}, actualArray)

			AssertValueSlicesEqual(
				t,
				inter,
				testCase.expectedValues,
				ArrayElements(inter, actualArray.(*interpreter.ArrayValue)),
			)
		})
	}
}

func TestInterpretInvalidArrayInsert(t *testing.T) {

	t.Parallel()

	for name, index := range map[string]int{
		"negative":          -1,
		"larger than count": 4,
	} {

		t.Run(name, func(t *testing.T) {

			inter := parseCheckAndInterpret(t, `
               let x = [1, 2, 3]

               fun test(_ index: Int) {
                   x.insert(at: index, 4)
               }
            `)

			indexValue := interpreter.NewUnmeteredIntValueFromInt64(int64(index))
			_, err := inter.Invoke("test", indexValue)
			RequireError(t, err)

			var indexErr interpreter.ArrayIndexOutOfBoundsError
			require.ErrorAs(t, err, &indexErr)

			assert.Equal(t, index, indexErr.Index)
			assert.Equal(t, 3, indexErr.Size)
			assert.Equal(t,
				ast.Position{Offset: 94, Line: 5, Column: 19},
				indexErr.HasPosition.StartPosition(),
			)
			assert.Equal(t,
				ast.Position{Offset: 115, Line: 5, Column: 40},
				indexErr.HasPosition.EndPosition(nil),
			)
		})
	}
}

func TestInterpretArrayRemove(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let x = [1, 2, 3]
      let y = x.remove(at: 1)
    `)

	value := inter.Globals.Get("x").GetValue(inter)

	arrayValue := value.(*interpreter.ArrayValue)
	AssertValueSlicesEqual(
		t,
		inter,
		[]interpreter.Value{
			interpreter.NewUnmeteredIntValueFromInt64(1),
			interpreter.NewUnmeteredIntValueFromInt64(3),
		},
		ArrayElements(inter, arrayValue),
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(2),
		inter.Globals.Get("y").GetValue(inter),
	)
}

func TestInterpretInvalidArrayRemove(t *testing.T) {

	t.Parallel()

	for name, index := range map[string]int{
		"negative":          -1,
		"larger than count": 3,
	} {

		t.Run(name, func(t *testing.T) {

			inter := parseCheckAndInterpret(t, `
               let x = [1, 2, 3]

               fun test(_ index: Int) {
                   x.remove(at: index)
               }
            `)

			indexValue := interpreter.NewUnmeteredIntValueFromInt64(int64(index))
			_, err := inter.Invoke("test", indexValue)
			RequireError(t, err)

			var indexErr interpreter.ArrayIndexOutOfBoundsError
			require.ErrorAs(t, err, &indexErr)

			assert.Equal(t, index, indexErr.Index)
			assert.Equal(t, 3, indexErr.Size)
			assert.Equal(t,
				ast.Position{Offset: 94, Line: 5, Column: 19},
				indexErr.HasPosition.StartPosition(),
			)
			assert.Equal(t,
				ast.Position{Offset: 112, Line: 5, Column: 37},
				indexErr.HasPosition.EndPosition(nil),
			)
		})
	}
}

func TestInterpretArrayRemoveFirst(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let x = [1, 2, 3]
      let y = x.removeFirst()
    `)

	value := inter.Globals.Get("x").GetValue(inter)

	arrayValue := value.(*interpreter.ArrayValue)
	AssertValueSlicesEqual(
		t,
		inter,
		[]interpreter.Value{
			interpreter.NewUnmeteredIntValueFromInt64(2),
			interpreter.NewUnmeteredIntValueFromInt64(3),
		},
		ArrayElements(inter, arrayValue),
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(1),
		inter.Globals.Get("y").GetValue(inter),
	)
}

func TestInterpretInvalidArrayRemoveFirst(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
       let x: [Int] = []

       fun test() {
           x.removeFirst()
       }
    `)

	_, err := inter.Invoke("test")
	RequireError(t, err)

	var indexErr interpreter.ArrayIndexOutOfBoundsError
	require.ErrorAs(t, err, &indexErr)

	assert.Equal(t, 0, indexErr.Index)
	assert.Equal(t, 0, indexErr.Size)
	assert.Equal(t,
		ast.Position{Offset: 58, Line: 5, Column: 11},
		indexErr.HasPosition.StartPosition(),
	)
	assert.Equal(t,
		ast.Position{Offset: 72, Line: 5, Column: 25},
		indexErr.HasPosition.EndPosition(nil),
	)
}

func TestInterpretArrayRemoveLast(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
          let x = [1, 2, 3]
          let y = x.removeLast()
    `)

	value := inter.Globals.Get("x").GetValue(inter)

	arrayValue := value.(*interpreter.ArrayValue)

	AssertValueSlicesEqual(
		t,
		inter,
		[]interpreter.Value{
			interpreter.NewUnmeteredIntValueFromInt64(1),
			interpreter.NewUnmeteredIntValueFromInt64(2),
		},
		ArrayElements(inter, arrayValue),
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(3),
		inter.Globals.Get("y").GetValue(inter),
	)
}

func TestInterpretInvalidArrayRemoveLast(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
       let x: [Int] = []

       fun test() {
           x.removeLast()
       }
    `)

	_, err := inter.Invoke("test")
	RequireError(t, err)

	var indexErr interpreter.ArrayIndexOutOfBoundsError
	require.ErrorAs(t, err, &indexErr)

	assert.Equal(t, -1, indexErr.Index)
	assert.Equal(t, 0, indexErr.Size)
	assert.Equal(t,
		ast.Position{Offset: 58, Line: 5, Column: 11},
		indexErr.HasPosition.StartPosition(),
	)
	assert.Equal(t,
		ast.Position{Offset: 71, Line: 5, Column: 24},
		indexErr.HasPosition.EndPosition(nil),
	)
}

func TestInterpretArraySlicing(t *testing.T) {

	t.Parallel()

	range1 := ast.Range{
		StartPos: ast.Position{Offset: 125, Line: 4, Column: 31},
		EndPos:   ast.Position{Offset: 149, Line: 4, Column: 55},
	}

	range2 := ast.Range{
		StartPos: ast.Position{Offset: 125, Line: 4, Column: 31},
		EndPos:   ast.Position{Offset: 150, Line: 4, Column: 56},
	}

	type test struct {
		literal    string
		from       int
		to         int
		result     string
		checkError func(t *testing.T, err error)
	}

	tests := []test{
		{"[1, 2, 3, 4, 5, 6]", 0, 6, "[1, 2, 3, 4, 5, 6]", nil},
		{"[1, 2, 3, 4, 5, 6]", 0, 0, "[]", nil},
		{"[1, 2, 3, 4, 5, 6]", 0, 1, "[1]", nil},
		{"[1, 2, 3, 4, 5, 6]", 0, 2, "[1, 2]", nil},
		{"[1, 2, 3, 4, 5, 6]", 1, 2, "[2]", nil},
		{"[1, 2, 3, 4, 5, 6]", 2, 3, "[3]", nil},
		{"[1, 2, 3, 4, 5, 6]", 5, 6, "[6]", nil},
		{"[1, 2, 3, 4, 5, 6]", 1, 6, "[2, 3, 4, 5, 6]", nil},
		// Invalid indices
		{"[1, 2, 3, 4, 5, 6]", -1, 0, "", func(t *testing.T, err error) {
			var sliceErr interpreter.ArraySliceIndicesError
			require.ErrorAs(t, err, &sliceErr)

			assert.Equal(t, -1, sliceErr.FromIndex)
			assert.Equal(t, 0, sliceErr.UpToIndex)
			assert.Equal(t, 6, sliceErr.Size)
			assert.Equal(t,
				range2.StartPos,
				sliceErr.LocationRange.StartPosition(),
			)
			assert.Equal(t,
				range2.EndPos,
				sliceErr.LocationRange.EndPosition(nil),
			)
		}},
		{"[1, 2, 3, 4, 5, 6]", 0, -1, "", func(t *testing.T, err error) {
			var sliceErr interpreter.ArraySliceIndicesError
			require.ErrorAs(t, err, &sliceErr)

			assert.Equal(t, 0, sliceErr.FromIndex)
			assert.Equal(t, -1, sliceErr.UpToIndex)
			assert.Equal(t, 6, sliceErr.Size)
			assert.Equal(t,
				range2.StartPos,
				sliceErr.LocationRange.StartPosition(),
			)
			assert.Equal(t,
				range2.EndPos,
				sliceErr.LocationRange.EndPosition(nil),
			)
		}},
		{"[1, 2, 3, 4, 5, 6]", 0, 10, "", func(t *testing.T, err error) {
			var sliceErr interpreter.ArraySliceIndicesError
			require.ErrorAs(t, err, &sliceErr)

			assert.Equal(t, 0, sliceErr.FromIndex)
			assert.Equal(t, 10, sliceErr.UpToIndex)
			assert.Equal(t, 6, sliceErr.Size)
			assert.Equal(t,
				range2.StartPos,
				sliceErr.LocationRange.StartPosition(),
			)
			assert.Equal(t,
				range2.EndPos,
				sliceErr.LocationRange.EndPosition(nil),
			)
		}},
		{"[1, 2, 3, 4, 5, 6]", 2, 1, "", func(t *testing.T, err error) {
			var indexErr interpreter.InvalidSliceIndexError
			require.ErrorAs(t, err, &indexErr)

			assert.Equal(t, 2, indexErr.FromIndex)
			assert.Equal(t, 1, indexErr.UpToIndex)
			assert.Equal(t,
				range1.StartPos,
				indexErr.LocationRange.StartPosition(),
			)
			assert.Equal(t,
				range1.EndPos,
				indexErr.LocationRange.EndPosition(nil),
			)
		}},
	}

	runTest := func(test test) {
		t.Run("", func(t *testing.T) {

			t.Parallel()

			inter := parseCheckAndInterpret(t,
				fmt.Sprintf(
					`
                      fun test(): [Int] {
                        let s = %s
                        return s.slice(from: %d, upTo: %d)
                      }
                    `,
					test.literal,
					test.from,
					test.to,
				),
			)

			value, err := inter.Invoke("test")
			if test.checkError == nil {
				require.NoError(t, err)

				assert.Equal(
					t,
					test.result,
					fmt.Sprint(value),
				)
			} else {
				require.IsType(t,
					interpreter.Error{},
					err,
				)

				test.checkError(t, err)
			}
		})
	}

	for _, test := range tests {
		runTest(test)
	}
}

func TestInterpretArrayContains(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      fun doesContain(): Bool {
          let a = [1, 2]
          return a.contains(1)
      }

      fun doesNotContain(): Bool {
          let a = [1, 2]
          return a.contains(3)
      }
    `)

	value, err := inter.Invoke("doesContain")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.TrueValue,
		value,
	)

	value, err = inter.Invoke("doesNotContain")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.FalseValue,
		value,
	)
}

func TestInterpretDictionaryContainsKey(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      fun doesContainKey(): Bool {
          let x = {
              1: "one",
              2: "two"
          }
          return x.containsKey(1)
      }

      fun doesNotContainKey(): Bool {
          let x = {
              1: "one",
              2: "two"
          }
          return x.containsKey(3)
      }
    `)

	value, err := inter.Invoke("doesContainKey")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.TrueValue,
		value,
	)

	value, err = inter.Invoke("doesNotContainKey")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.FalseValue,
		value,
	)
}

func TestInterpretStringConcat(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      fun test(): String {
          let a = "abc"
          return a.concat("def")
      }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredStringValue("abcdef"),
		value,
	)
}

func TestInterpretStringConcatBound(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      fun test(): String {
          let a = "abc"
          let b = a.concat
          return b("def")
      }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredStringValue("abcdef"),
		value,
	)
}

func TestInterpretDictionaryRemove(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let xs = {"abc": 1, "def": 2}
      let removed = xs.remove(key: "abc")
    `)

	actualValue := inter.Globals.Get("xs").GetValue(inter)

	require.IsType(t, actualValue, &interpreter.DictionaryValue{})
	actualDict := actualValue.(*interpreter.DictionaryValue)

	AssertValueSlicesEqual(
		t,
		inter,
		[]interpreter.Value{
			interpreter.NewUnmeteredStringValue("def"),
			interpreter.NewUnmeteredIntValueFromInt64(2),
		},
		DictionaryKeyValues(inter, actualDict),
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredSomeValueNonCopying(
			interpreter.NewUnmeteredIntValueFromInt64(1),
		),
		inter.Globals.Get("removed").GetValue(inter),
	)
}

func TestInterpretDictionaryInsert(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let xs = {"abc": 1, "def": 2}
      let inserted = xs.insert(key: "abc", 3)
    `)

	actualValue := inter.Globals.Get("xs").GetValue(inter)

	require.IsType(t, actualValue, &interpreter.DictionaryValue{})
	actualDict := actualValue.(*interpreter.DictionaryValue)

	AssertValueSlicesEqual(
		t,
		inter,
		[]interpreter.Value{
			interpreter.NewUnmeteredStringValue("abc"),
			interpreter.NewUnmeteredIntValueFromInt64(3),
			interpreter.NewUnmeteredStringValue("def"),
			interpreter.NewUnmeteredIntValueFromInt64(2),
		},
		DictionaryKeyValues(inter, actualDict),
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredSomeValueNonCopying(
			interpreter.NewUnmeteredIntValueFromInt64(1),
		),
		inter.Globals.Get("inserted").GetValue(inter),
	)
}

func TestInterpretDictionaryKeys(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      fun test(): [String] {
          let dict = {"def": 2, "abc": 1}
          dict.insert(key: "a", 3)
          return dict.keys
      }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	arrayValue := value.(*interpreter.ArrayValue)

	AssertValueSlicesEqual(
		t,
		inter,
		[]interpreter.Value{
			interpreter.NewUnmeteredStringValue("abc"),
			interpreter.NewUnmeteredStringValue("def"),
			interpreter.NewUnmeteredStringValue("a"),
		},
		ArrayElements(inter, arrayValue),
	)
}

func TestInterpretDictionaryForEachKey(t *testing.T) {
	t.Parallel()

	t.Run("iter", func(t *testing.T) {

		type testcase struct {
			n        int64
			endPoint int64
		}
		testcases := []testcase{
			{10, 1},
			{20, 5},
			{100, 10},
			{100, 0},
		}
		inter := parseCheckAndInterpret(t, `
			fun testForEachKey(n: Int, stopIter: Int): {Int: Int} {
				var dict: {Int:Int} = {}
				var counts: {Int:Int} = {}
				var i = 0
				while i < n {
					dict[i] = i
					counts[i] = 0
					i = i + 1
				}
				dict.forEachKey(fun(k: Int): Bool {
					if k == stopIter {
						return false
					}
					let curVal = counts[k]!
					counts[k] = curVal + 1
					return true
				})

				return counts
			}
		`)

		for _, test := range testcases {
			name := fmt.Sprintf("n = %d", test.n)
			t.Run(name, func(t *testing.T) {
				n := test.n
				endPoint := test.endPoint

				nVal := interpreter.NewUnmeteredIntValueFromInt64(n)
				stopIter := interpreter.NewUnmeteredIntValueFromInt64(endPoint)
				res, err := inter.Invoke("testForEachKey", nVal, stopIter)

				require.NoError(t, err)

				dict, ok := res.(*interpreter.DictionaryValue)
				assert.True(t, ok)

				toInt := func(val interpreter.Value) (int, bool) {
					intVal, ok := val.(interpreter.IntValue)
					if !ok {
						return 0, ok
					}
					return intVal.ToInt(interpreter.EmptyLocationRange), true
				}

				entries, ok := DictionaryEntries(inter, dict, toInt, toInt)

				assert.True(t, ok)

				for _, entry := range entries {
					// iteration order is undefined, so the only thing we can deterministically test is
					// whether visited keys exist in the dict
					// and whether iteration is affine

					key := int64(entry.Key)
					require.True(t,
						0 <= key && key < n,
						"Visited key not present in the original dictionary: %d",
						key,
					)
					// assert that we exited early
					if int64(entry.Key) == endPoint {
						AssertEqualWithDiff(t, 0, entry.Value)
					} else {
						// make sure no key was visited twice
						require.LessOrEqual(t,
							entry.Value,
							1,
							"Dictionary entry visited twice during iteration",
						)
					}

				}

			})
		}
	})

	t.Run("box and convert argument", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          fun test(): String? {
              let dict = {"answer": 42}
              var res: String? = nil
              // NOTE: The function has a parameter of type String? instead of just String
              dict.forEachKey(fun(key: String?): Bool {
                  // The map should call Optional.map, not fail,
                  // because key is String?, not String
                  res = key.map(fun(string: AnyStruct): String {
                      return "Optional.map"
                  })
                  return true
              })
              return res
          }
        `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(t,
			inter,
			interpreter.NewSomeValueNonCopying(
				nil,
				interpreter.NewUnmeteredStringValue("Optional.map"),
			),
			value,
		)
	})

}

func TestInterpretDictionaryValues(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      fun test(): [Int] {
          let dict = {"def": 2, "abc": 1}
          dict.insert(key: "a", 3)
          return dict.values
      }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	arrayValue := value.(*interpreter.ArrayValue)

	AssertValueSlicesEqual(
		t,
		inter,
		[]interpreter.Value{
			interpreter.NewUnmeteredIntValueFromInt64(1),
			interpreter.NewUnmeteredIntValueFromInt64(2),
			interpreter.NewUnmeteredIntValueFromInt64(3),
		},
		ArrayElements(inter, arrayValue),
	)
}

func TestInterpretDictionaryKeyTypes(t *testing.T) {

	t.Parallel()

	tests := map[string]string{
		"String":         `"abc"`,
		"Character":      `"X"`,
		"Address":        `0x1`,
		"Bool":           `true`,
		"Path":           `/storage/a`,
		"StoragePath":    `/storage/a`,
		"PublicPath":     `/public/a`,
		"PrivatePath":    `/private/a`,
		"CapabilityPath": `/private/a`,
	}

	for _, integerType := range sema.AllIntegerTypes {
		tests[integerType.String()] = `42`
	}

	for _, fixedPointType := range sema.AllFixedPointTypes {

		var literal string

		if sema.IsSubType(fixedPointType, sema.SignedFixedPointType) {
			literal = "-1.23"
		} else {
			literal = "1.23"
		}

		tests[fixedPointType.String()] = literal
	}

	for ty, code := range tests {
		t.Run(ty, func(t *testing.T) {

			inter := parseCheckAndInterpret(t,
				fmt.Sprintf(
					`
                      let k: %s = %s
                      let xs = {k: "test"}
                      let v = xs[k]
                    `,
					ty,
					code,
				),
			)

			AssertValuesEqual(
				t,
				inter,
				interpreter.NewUnmeteredSomeValueNonCopying(
					interpreter.NewUnmeteredStringValue("test"),
				),
				inter.Globals.Get("v").GetValue(inter),
			)
		})
	}
}

func TestInterpretPathToString(t *testing.T) {

	t.Parallel()

	tests := map[string]string{
		"Path":           `/storage/a`,
		"StoragePath":    `/storage/a`,
		"PublicPath":     `/public/a`,
		"PrivatePath":    `/private/a`,
		"CapabilityPath": `/private/a`,
	}

	for ty, val := range tests {
		t.Run(ty, func(t *testing.T) {
			inter := parseCheckAndInterpret(t,
				fmt.Sprintf(
					`
                           let x: %s = %s
                           let y: String = x.toString()
                         `,
					ty,
					val,
				))

			assert.Equal(t,
				interpreter.NewUnmeteredStringValue(val),
				inter.Globals.Get("y").GetValue(inter),
			)
		})
	}
}

func TestInterpretIndirectDestroy(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      resource X {}

      fun test() {
          let x <- create X()
          destroy x
      }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.Void,
		value,
	)
}

func TestInterpretUnaryMove(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      resource X {}

      fun foo(x: @X): @X {
          return <-x
      }

      fun bar() {
          let x <- foo(x: <-create X())
          destroy x
      }
    `)

	value, err := inter.Invoke("bar")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.Void,
		value,
	)
}

func TestInterpretResourceMoveInArrayAndDestroy(t *testing.T) {

	t.Parallel()

	var events []*interpreter.CompositeValue

	inter, err := parseCheckAndInterpretWithOptions(t, `
      resource Foo {
		  event ResourceDestroyed(bar: Int = self.bar)
          var bar: Int

          init(bar: Int) {
              self.bar = bar
          }
      }

      fun test(): Int {
          let foo1 <- create Foo(bar: 1)
          let foo2 <- create Foo(bar: 2)
          let foos <- [<-foo1, <-foo2]
          let bar = foos[1].bar
          destroy foos
          return bar
      }
    `, ParseCheckAndInterpretOptions{
		Config: &interpreter.Config{
			OnEventEmitted: func(_ *interpreter.Interpreter, _ interpreter.LocationRange, event *interpreter.CompositeValue, eventType *sema.CompositeType) error {
				events = append(events, event)
				return nil
			},
		},
	})
	require.NoError(t, err)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(2),
		value,
	)

	require.Len(t, events, 2)
	require.Equal(t, "Foo.ResourceDestroyed", events[0].QualifiedIdentifier)
	require.Equal(t, interpreter.NewIntValueFromInt64(nil, 1), events[0].GetField(inter, "bar"))
	require.Equal(t, "Foo.ResourceDestroyed", events[1].QualifiedIdentifier)
	require.Equal(t, interpreter.NewIntValueFromInt64(nil, 2), events[1].GetField(inter, "bar"))
}

func TestInterpretResourceMoveInDictionaryAndDestroy(t *testing.T) {

	t.Parallel()

	var events []*interpreter.CompositeValue

	inter, err := parseCheckAndInterpretWithOptions(t, `
      resource Foo {
		  event ResourceDestroyed(bar: Int = self.bar)
          var bar: Int

          init(bar: Int) {
              self.bar = bar
          }
      }

      fun test() {
          let foo1 <- create Foo(bar: 1)
          let foo2 <- create Foo(bar: 2)
          let foos <- {"foo1": <-foo1, "foo2": <-foo2}
          destroy foos
      }
    `, ParseCheckAndInterpretOptions{
		Config: &interpreter.Config{
			OnEventEmitted: func(_ *interpreter.Interpreter, _ interpreter.LocationRange, event *interpreter.CompositeValue, eventType *sema.CompositeType) error {
				events = append(events, event)
				return nil
			},
		},
	})
	require.NoError(t, err)

	_, err = inter.Invoke("test")
	require.NoError(t, err)

	require.Len(t, events, 2)
	require.Equal(t, "Foo.ResourceDestroyed", events[0].QualifiedIdentifier)
	require.Equal(t, interpreter.NewIntValueFromInt64(nil, 1), events[0].GetField(inter, "bar"))
	require.Equal(t, "Foo.ResourceDestroyed", events[1].QualifiedIdentifier)
	require.Equal(t, interpreter.NewIntValueFromInt64(nil, 2), events[1].GetField(inter, "bar"))
}

func TestInterpretClosure(t *testing.T) {

	t.Parallel()

	// Create a closure that increments and returns
	// a variable each time it is invoked.

	inter := parseCheckAndInterpret(t, `
        fun makeCounter(): fun(): Int {
            var count = 0
            return fun (): Int {
                count = count + 1
                return count
            }
        }

        let test = makeCounter()
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(1),
		value,
	)

	value, err = inter.Invoke("test")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(2),
		value,
	)

	value, err = inter.Invoke("test")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(3),
		value,
	)
}

func TestInterpretClosureScopingFunctionExpression(t *testing.T) {
	t.Parallel()

	inter := parseCheckAndInterpret(t, `
        fun test(a: Int): Int {
            let bar = fun(b: Int): Int {
                return a + b
            }
            let a = 2
            return bar(b: 10)
        }
    `)

	actual, err := inter.Invoke("test",
		interpreter.NewUnmeteredIntValueFromInt64(1),
	)
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(11),
		actual,
	)
}

func TestInterpretClosureScopingInnerFunction(t *testing.T) {
	t.Parallel()

	inter := parseCheckAndInterpret(t, `
        fun test(a: Int): Int {
            fun bar(b: Int): Int {
                return a + b
            }
            let a = 2
            return bar(b: 10)
        }
    `)

	value, err := inter.Invoke("test",
		interpreter.NewUnmeteredIntValueFromInt64(1),
	)
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(11),
		value,
	)
}

func TestInterpretClosureScopingFunctionExpressionParameterConfusion(t *testing.T) {
	t.Parallel()

	inter := parseCheckAndInterpret(t, `
        fun foo(a: Int) {
            fun() {}
        }

        fun test(): Int {
            let a = 1
            foo(a: 2)
            return a
        }
    `)

	actual, err := inter.Invoke("test")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(1),
		actual,
	)
}

func TestInterpretClosureScopingInnerFunctionParameterConfusion(t *testing.T) {
	t.Parallel()

	inter := parseCheckAndInterpret(t, `
        fun foo(a: Int) {
            let f = fun() {}
        }

        fun test(): Int {
            let a = 1
            foo(a: 2)
            return a
        }
    `)

	actual, err := inter.Invoke("test")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(1),
		actual,
	)
}

func TestInterpretClosureScopingFunctionExpressionInCall(t *testing.T) {
	t.Parallel()

	inter := parseCheckAndInterpret(t, `
        fun foo() {
            fun() {}
        }

        fun test(): Int {
            let a = 1
            foo()
            return a
        }
    `)

	actual, err := inter.Invoke("test")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(1),
		actual,
	)
}

func TestInterpretClosureScopingInnerFunctionInCall(t *testing.T) {
	t.Parallel()

	inter := parseCheckAndInterpret(t, `
        fun foo() {
            let f = fun() {}
        }

        fun test(): Int {
            let a = 1
            foo()
            return a
        }
    `)

	actual, err := inter.Invoke("test")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(1),
		actual,
	)
}

func TestInterpretAssignmentAfterClosureFunctionExpression(t *testing.T) {
	t.Parallel()

	inter := parseCheckAndInterpret(t, `
        fun test(): Int {
            var a = 1
            let bar = fun(b: Int): Int {
                return a + b
            }
            a = 2
            return bar(b: 10)
        }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(12),
		value,
	)
}

func TestInterpretAssignmentAfterClosureInnerFunction(t *testing.T) {
	t.Parallel()

	inter := parseCheckAndInterpret(t, `
        fun test(): Int {
            var a = 1
            fun bar(b: Int): Int {
                return a + b
            }
            a = 2
            return bar(b: 10)
        }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(12),
		value,
	)
}

// TestInterpretCompositeFunctionInvocationFromImportingProgram checks
// that member functions of imported composites can be invoked from an importing program.
// See https://github.com/dapperlabs/flow-go/issues/838
func TestInterpretCompositeFunctionInvocationFromImportingProgram(t *testing.T) {

	t.Parallel()

	importedChecker, err := ParseAndCheckWithOptions(t,
		`
          // function must have arguments
          access(all) fun x(x: Int) {}

          // invocation must be in composite
          access(all) struct Y {

              access(all) fun x() {
                  x(x: 1)
              }
          }
        `,
		ParseAndCheckOptions{
			Location: ImportedLocation,
		},
	)
	require.NoError(t, err)

	importingChecker, err := ParseAndCheckWithOptions(t,
		`
          import Y from "imported"

          access(all) fun test() {
              // get member must bind using imported interpreter
              Y().x()
          }
        `,
		ParseAndCheckOptions{
			Config: &sema.Config{
				ImportHandler: func(_ *sema.Checker, importedLocation common.Location, _ ast.Range) (sema.Import, error) {
					assert.Equal(t,
						ImportedLocation,
						importedLocation,
					)

					return sema.ElaborationImport{
						Elaboration: importedChecker.Elaboration,
					}, nil
				},
			},
		},
	)
	require.NoError(t, err)

	storage := newUnmeteredInMemoryStorage()

	inter, err := interpreter.NewInterpreter(
		interpreter.ProgramFromChecker(importingChecker),
		importingChecker.Location,
		&interpreter.Config{
			Storage: storage,
			ImportLocationHandler: func(inter *interpreter.Interpreter, location common.Location) interpreter.Import {
				assert.Equal(t,
					ImportedLocation,
					location,
				)

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
	)
	require.NoError(t, err)

	err = inter.Interpret()
	require.NoError(t, err)

	_, err = inter.Invoke("test")
	require.NoError(t, err)
}

func TestInterpretSwapVariables(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
       fun test(): [Int] {
           var x = 2
           var y = 3
           x <-> y
           return [x, y]
       }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewArrayValue(
			inter,
			interpreter.EmptyLocationRange,
			&interpreter.VariableSizedStaticType{
				Type: interpreter.PrimitiveStaticTypeInt,
			},
			common.ZeroAddress,
			interpreter.NewUnmeteredIntValueFromInt64(3),
			interpreter.NewUnmeteredIntValueFromInt64(2),
		),
		value,
	)
}

func TestInterpretSwapArrayAndField(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
       struct Foo {
           var bar: Int

           init(bar: Int) {
               self.bar = bar
           }
       }

       fun test(): [Int] {
           let foo = Foo(bar: 1)
           let nums = [2]
           foo.bar <-> nums[0]
           return [foo.bar, nums[0]]
       }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewArrayValue(
			inter,
			interpreter.EmptyLocationRange,
			&interpreter.VariableSizedStaticType{
				Type: interpreter.PrimitiveStaticTypeInt,
			},
			common.ZeroAddress,
			interpreter.NewUnmeteredIntValueFromInt64(2),
			interpreter.NewUnmeteredIntValueFromInt64(1),
		),
		value,
	)
}

func TestInterpretResourceDestroyExpressionNoDestructor(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
       resource R {}

       fun test() {
           let r <- create R()
           destroy r
       }
    `)

	_, err := inter.Invoke("test")
	require.NoError(t, err)
}

func TestInterpretResourceDestroyExpressionDestructor(t *testing.T) {

	t.Parallel()

	var events []*interpreter.CompositeValue

	inter, err := parseCheckAndInterpretWithOptions(t, `
        resource R {
			event ResourceDestroyed()
	    }

       fun test() {
           let r <- create R()
           destroy r
       }
    `, ParseCheckAndInterpretOptions{
		Config: &interpreter.Config{
			OnEventEmitted: func(_ *interpreter.Interpreter, _ interpreter.LocationRange, event *interpreter.CompositeValue, eventType *sema.CompositeType) error {
				events = append(events, event)
				return nil
			},
		},
	})

	require.NoError(t, err)

	_, err = inter.Invoke("test")
	require.NoError(t, err)

	require.Len(t, events, 1)
	require.Equal(t, "R.ResourceDestroyed", events[0].QualifiedIdentifier)
}

func TestInterpretResourceDestroyExpressionNestedResources(t *testing.T) {

	t.Parallel()

	var events []*interpreter.CompositeValue

	inter, err := parseCheckAndInterpretWithOptions(t, `
      resource B {
		var foo: Int
		event ResourceDestroyed(foo: Int = self.foo)

		init() {
			self.foo = 5
		}
	  }

      resource A {
		  event ResourceDestroyed(foo: Int = self.b.foo)

          let b: @B

          init(b: @B) {
              self.b <- b
          }
      }

      fun test() {
          let b <- create B()
          let a <- create A(b: <-b)
          destroy a
      }
    `, ParseCheckAndInterpretOptions{
		Config: &interpreter.Config{
			OnEventEmitted: func(_ *interpreter.Interpreter, _ interpreter.LocationRange, event *interpreter.CompositeValue, eventType *sema.CompositeType) error {
				events = append(events, event)
				return nil
			},
		},
	})
	require.NoError(t, err)

	_, err = inter.Invoke("test")
	require.NoError(t, err)

	require.Len(t, events, 2)
	require.Equal(t, "B.ResourceDestroyed", events[0].QualifiedIdentifier)
	require.Equal(t, interpreter.NewIntValueFromInt64(nil, 5), events[0].GetField(inter, "foo"))
	require.Equal(t, "A.ResourceDestroyed", events[1].QualifiedIdentifier)
	require.Equal(t, interpreter.NewIntValueFromInt64(nil, 5), events[1].GetField(inter, "foo"))
}

func TestInterpretResourceDestroyArray(t *testing.T) {

	t.Parallel()

	var events []*interpreter.CompositeValue

	inter, err := parseCheckAndInterpretWithOptions(t, `
      resource R {
		event ResourceDestroyed()
	  }

      fun test() {
          let rs <- [<-create R(), <-create R()]
          destroy rs
      }
    `, ParseCheckAndInterpretOptions{
		Config: &interpreter.Config{
			OnEventEmitted: func(_ *interpreter.Interpreter, _ interpreter.LocationRange, event *interpreter.CompositeValue, eventType *sema.CompositeType) error {
				events = append(events, event)
				return nil
			},
		},
	})
	require.NoError(t, err)

	_, err = inter.Invoke("test")
	require.NoError(t, err)

	require.Len(t, events, 2)
	require.Equal(t, "R.ResourceDestroyed", events[0].QualifiedIdentifier)
	require.Equal(t, "R.ResourceDestroyed", events[1].QualifiedIdentifier)
}

func TestInterpretResourceDestroyDictionary(t *testing.T) {

	t.Parallel()

	var events []*interpreter.CompositeValue

	inter, err := parseCheckAndInterpretWithOptions(t, `
	  resource R {
		event ResourceDestroyed()
	  }

      fun test() {
          let rs <- {"r1": <-create R(), "r2": <-create R()}
          destroy rs
      }
    `, ParseCheckAndInterpretOptions{
		Config: &interpreter.Config{
			OnEventEmitted: func(_ *interpreter.Interpreter, _ interpreter.LocationRange, event *interpreter.CompositeValue, eventType *sema.CompositeType) error {
				events = append(events, event)
				return nil
			},
		},
	})
	require.NoError(t, err)

	_, err = inter.Invoke("test")
	require.NoError(t, err)

	require.Len(t, events, 2)
	require.Equal(t, "R.ResourceDestroyed", events[0].QualifiedIdentifier)
	require.Equal(t, "R.ResourceDestroyed", events[1].QualifiedIdentifier)
}

func TestInterpretResourceDestroyOptionalSome(t *testing.T) {

	t.Parallel()

	var events []*interpreter.CompositeValue

	inter, err := parseCheckAndInterpretWithOptions(t, `
      resource R { 
		event ResourceDestroyed()
	  }

      fun test() {
          let maybeR: @R? <- create R()
          destroy maybeR
      }
    `, ParseCheckAndInterpretOptions{
		Config: &interpreter.Config{
			OnEventEmitted: func(_ *interpreter.Interpreter, _ interpreter.LocationRange, event *interpreter.CompositeValue, eventType *sema.CompositeType) error {
				events = append(events, event)
				return nil
			},
		},
	})
	require.NoError(t, err)

	_, err = inter.Invoke("test")
	require.NoError(t, err)

	require.Len(t, events, 1)
	require.Equal(t, "R.ResourceDestroyed", events[0].QualifiedIdentifier)
}

func TestInterpretResourceDestroyOptionalNil(t *testing.T) {

	t.Parallel()

	var events []*interpreter.CompositeValue

	inter, err := parseCheckAndInterpretWithOptions(t, `
      resource R {
		event ResourceDestroyed()
	  }

      fun test() {
          let maybeR: @R? <- nil
          destroy maybeR
      }
    `, ParseCheckAndInterpretOptions{
		Config: &interpreter.Config{
			OnEventEmitted: func(_ *interpreter.Interpreter, _ interpreter.LocationRange, event *interpreter.CompositeValue, eventType *sema.CompositeType) error {
				events = append(events, event)
				return nil
			},
		},
	})
	require.NoError(t, err)

	_, err = inter.Invoke("test")
	require.NoError(t, err)

	require.Len(t, events, 0)
}

// TestInterpretInterfaceInitializer tests that the interface's initializer
// is called, even if the conforming composite does not have an initializer
func TestInterpretInterfaceInitializer(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      struct interface I {
          init(a a1: Bool) {
              pre { a1 }
          }
      }

      struct S: I {
          init(a a2: Bool) {}
      }

      fun test() {
          S(a: false)
      }
    `)

	_, err := inter.Invoke("test")
	require.IsType(t,
		interpreter.Error{},
		err,
	)
	interpreterErr := err.(interpreter.Error)

	require.IsType(t,
		interpreter.ConditionError{},
		interpreterErr.Err,
	)
}

func TestInterpretEmitEvent(t *testing.T) {

	t.Parallel()

	var actualEvents []interpreter.Value

	inter, err := parseCheckAndInterpretWithOptions(t,
		`
          event Transfer(to: Int, from: Int)
          event TransferAmount(to: Int, from: Int, amount: Int)

          fun test() {
              emit Transfer(to: 1, from: 2)
              emit Transfer(to: 3, from: 4)
              emit TransferAmount(to: 1, from: 2, amount: 100)
          }
        `,
		ParseCheckAndInterpretOptions{
			Config: &interpreter.Config{
				OnEventEmitted: func(
					_ *interpreter.Interpreter,
					_ interpreter.LocationRange,
					event *interpreter.CompositeValue,
					eventType *sema.CompositeType,
				) error {
					actualEvents = append(actualEvents, event)
					return nil
				},
			},
		},
	)
	require.NoError(t, err)

	_, err = inter.Invoke("test")
	require.NoError(t, err)

	transferEventType := RequireGlobalType(t, inter.Program.Elaboration, "Transfer")
	transferAmountEventType := RequireGlobalType(t, inter.Program.Elaboration, "TransferAmount")

	fields1 := []interpreter.CompositeField{
		{
			Name:  "to",
			Value: interpreter.NewUnmeteredIntValueFromInt64(1),
		},
		{
			Name:  "from",
			Value: interpreter.NewUnmeteredIntValueFromInt64(2),
		},
	}

	fields2 := []interpreter.CompositeField{
		{
			Name:  "to",
			Value: interpreter.NewUnmeteredIntValueFromInt64(3),
		},
		{
			Name:  "from",
			Value: interpreter.NewUnmeteredIntValueFromInt64(4),
		},
	}

	fields3 := []interpreter.CompositeField{
		{
			Name:  "to",
			Value: interpreter.NewUnmeteredIntValueFromInt64(1),
		},
		{
			Name:  "from",
			Value: interpreter.NewUnmeteredIntValueFromInt64(2),
		},
		{
			Name:  "amount",
			Value: interpreter.NewUnmeteredIntValueFromInt64(100),
		},
	}

	expectedEvents := []interpreter.Value{
		interpreter.NewCompositeValue(
			inter,
			interpreter.EmptyLocationRange,
			TestLocation,
			TestLocation.QualifiedIdentifier(transferEventType.ID()),
			common.CompositeKindEvent,
			fields1,
			common.ZeroAddress,
		),
		interpreter.NewCompositeValue(
			inter,
			interpreter.EmptyLocationRange,
			TestLocation,
			TestLocation.QualifiedIdentifier(transferEventType.ID()),
			common.CompositeKindEvent,
			fields2,
			common.ZeroAddress,
		),
		interpreter.NewCompositeValue(
			inter,
			interpreter.EmptyLocationRange,
			TestLocation,
			TestLocation.QualifiedIdentifier(transferAmountEventType.ID()),
			common.CompositeKindEvent,
			fields3,
			common.ZeroAddress,
		),
	}

	AssertValueSlicesEqual(
		t,
		inter,
		expectedEvents,
		actualEvents,
	)
}

func TestInterpretReferenceEventParameter(t *testing.T) {

	t.Parallel()

	var actualEvents []interpreter.Value

	inter, err := parseCheckAndInterpretWithOptions(t,
		`
          event TestEvent(ref: &[{Int: String}])

          fun test(ref: &[{Int: String}]) {
              emit TestEvent(ref: ref)
          }
        `,
		ParseCheckAndInterpretOptions{
			Config: &interpreter.Config{
				OnEventEmitted: func(
					_ *interpreter.Interpreter,
					_ interpreter.LocationRange,
					event *interpreter.CompositeValue,
					eventType *sema.CompositeType,
				) error {
					actualEvents = append(actualEvents, event)
					return nil
				},
			},
		},
	)
	require.NoError(t, err)

	dictionaryStaticType := interpreter.NewDictionaryStaticType(
		nil,
		interpreter.PrimitiveStaticTypeInt,
		interpreter.PrimitiveStaticTypeString,
	)

	dictionaryValue := interpreter.NewDictionaryValue(
		inter,
		interpreter.EmptyLocationRange,
		dictionaryStaticType,
		interpreter.NewUnmeteredIntValueFromInt64(42),
		interpreter.NewUnmeteredStringValue("answer"),
	)

	arrayStaticType := interpreter.NewVariableSizedStaticType(nil, dictionaryStaticType)

	arrayValue := interpreter.NewArrayValue(
		inter,
		interpreter.EmptyLocationRange,
		arrayStaticType,
		common.ZeroAddress,
		dictionaryValue,
	)

	ref := interpreter.NewUnmeteredEphemeralReferenceValue(
		inter,
		interpreter.UnauthorizedAccess,
		arrayValue,
		interpreter.MustConvertStaticToSemaType(arrayStaticType, inter),
		interpreter.EmptyLocationRange,
	)

	_, err = inter.Invoke("test", ref)
	require.NoError(t, err)

	eventType := RequireGlobalType(t, inter.Program.Elaboration, "TestEvent")

	expectedEvents := []interpreter.Value{
		interpreter.NewCompositeValue(
			inter,
			interpreter.EmptyLocationRange,
			TestLocation,
			TestLocation.QualifiedIdentifier(eventType.ID()),
			common.CompositeKindEvent,
			[]interpreter.CompositeField{
				{
					Name:  "ref",
					Value: ref,
				},
			},
			common.ZeroAddress,
		),
	}

	AssertValueSlicesEqual(
		t,
		inter,
		expectedEvents,
		actualEvents,
	)
}

type testValue struct {
	value              interpreter.Value
	ty                 sema.Type
	literal            string
	notAsDictionaryKey bool
}

func (v testValue) String() string {
	if v.literal == "" {
		return v.value.String()
	}
	return v.literal
}

func TestInterpretEmitEventParameterTypes(t *testing.T) {

	t.Parallel()

	sType := &sema.CompositeType{
		Location:   TestLocation,
		Identifier: "S",
		Kind:       common.CompositeKindStructure,
		Members:    &sema.StringMemberOrderedMap{},
	}

	storage := newUnmeteredInMemoryStorage()

	inter, err := interpreter.NewInterpreter(
		nil,
		TestLocation,
		&interpreter.Config{Storage: storage},
	)
	require.NoError(t, err)

	sValue := interpreter.NewCompositeValue(
		inter,
		interpreter.EmptyLocationRange,
		TestLocation,
		"S",
		common.CompositeKindStructure,
		nil,
		common.ZeroAddress,
	)
	sValue.Functions = orderedmap.New[interpreter.FunctionOrderedMap](0)

	validTypes := map[string]testValue{
		"String": {
			value: interpreter.NewUnmeteredStringValue("test"),
			ty:    sema.StringType,
		},
		"Character": {
			value: interpreter.NewUnmeteredCharacterValue("X"),
			ty:    sema.CharacterType,
		},
		"Bool": {
			value: interpreter.TrueValue,
			ty:    sema.BoolType,
		},
		"Address": {
			literal: `0x1`,
			value:   interpreter.NewUnmeteredAddressValueFromBytes([]byte{0x1}),
			ty:      sema.TheAddressType,
		},
		// Int*
		"Int": {
			value: interpreter.NewUnmeteredIntValueFromInt64(42),
			ty:    sema.IntType,
		},
		"Int8": {
			value: interpreter.NewUnmeteredInt8Value(42),
			ty:    sema.Int8Type,
		},
		"Int16": {
			value: interpreter.NewUnmeteredInt16Value(42),
			ty:    sema.Int16Type,
		},
		"Int32": {
			value: interpreter.NewUnmeteredInt32Value(42),
			ty:    sema.Int32Type,
		},
		"Int64": {
			value: interpreter.NewUnmeteredInt64Value(42),
			ty:    sema.Int64Type,
		},
		"Int128": {
			value: interpreter.NewUnmeteredInt128ValueFromInt64(42),
			ty:    sema.Int128Type,
		},
		"Int256": {
			value: interpreter.NewUnmeteredInt256ValueFromInt64(42),
			ty:    sema.Int256Type,
		},
		// UInt*
		"UInt": {
			value: interpreter.NewUnmeteredUIntValueFromUint64(42),
			ty:    sema.UIntType,
		},
		"UInt8": {
			value: interpreter.NewUnmeteredUInt8Value(42),
			ty:    sema.UInt8Type,
		},
		"UInt16": {
			value: interpreter.NewUnmeteredUInt16Value(42),
			ty:    sema.UInt16Type,
		},
		"UInt32": {
			value: interpreter.NewUnmeteredUInt32Value(42),
			ty:    sema.UInt32Type,
		},
		"UInt64": {
			value: interpreter.NewUnmeteredUInt64Value(42),
			ty:    sema.UInt64Type,
		},
		"UInt128": {
			value: interpreter.NewUnmeteredUInt128ValueFromUint64(42),
			ty:    sema.UInt128Type,
		},
		"UInt256": {
			value: interpreter.NewUnmeteredUInt256ValueFromUint64(42),
			ty:    sema.UInt256Type,
		},
		// Word*
		"Word8": {
			value: interpreter.NewUnmeteredWord8Value(42),
			ty:    sema.Word8Type,
		},
		"Word16": {
			value: interpreter.NewUnmeteredWord16Value(42),
			ty:    sema.Word16Type,
		},
		"Word32": {
			value: interpreter.NewUnmeteredWord32Value(42),
			ty:    sema.Word32Type,
		},
		"Word64": {
			value: interpreter.NewUnmeteredWord64Value(42),
			ty:    sema.Word64Type,
		},
		"Word128": {
			value: interpreter.NewUnmeteredWord128ValueFromUint64(42),
			ty:    sema.Word128Type,
		},
		"Word256": {
			value: interpreter.NewUnmeteredWord256ValueFromUint64(42),
			ty:    sema.Word256Type,
		},
		// Fix*
		"Fix64": {
			value: interpreter.NewUnmeteredFix64Value(123000000),
			ty:    sema.Fix64Type,
		},
		// UFix*
		"UFix64": {
			value: interpreter.NewUnmeteredUFix64Value(123000000),
			ty:    sema.UFix64Type,
		},
		// TODO:
		//// Struct
		//"S": {
		//     literal:            `s`,
		//     ty:                 sType,
		//     notAsDictionaryKey: true,
		//},
	}

	for _, integerType := range sema.AllIntegerTypes {

		switch integerType {
		case sema.IntegerType, sema.SignedIntegerType, sema.FixedSizeUnsignedIntegerType:
			continue
		}

		if _, ok := validTypes[integerType.String()]; !ok {
			panic(fmt.Sprintf("broken test: missing %s", integerType))
		}
	}

	for _, fixedPointType := range sema.AllFixedPointTypes {

		switch fixedPointType {
		case sema.FixedPointType, sema.SignedFixedPointType:
			continue
		}

		if _, ok := validTypes[fixedPointType.String()]; !ok {
			panic(fmt.Sprintf("broken test: missing %s", fixedPointType))
		}
	}

	tests := map[string]testValue{}

	for validType, testCase := range validTypes {
		tests[validType] = testCase

		tests[fmt.Sprintf("%s?", validType)] =
			testValue{
				value:   interpreter.NewUnmeteredSomeValueNonCopying(testCase.value),
				literal: testCase.literal,
			}

		tests[fmt.Sprintf("[%s]", validType)] =
			testValue{
				value: interpreter.NewArrayValue(
					inter,
					interpreter.EmptyLocationRange,
					&interpreter.VariableSizedStaticType{
						Type: interpreter.ConvertSemaToStaticType(nil, testCase.ty),
					},
					common.ZeroAddress,
					testCase.value,
				),
				literal: fmt.Sprintf("[%s as %s]", testCase, validType),
			}

		tests[fmt.Sprintf("[%s; 1]", validType)] =
			testValue{
				value: interpreter.NewArrayValue(
					inter,
					interpreter.EmptyLocationRange,
					&interpreter.ConstantSizedStaticType{
						Type: interpreter.ConvertSemaToStaticType(nil, testCase.ty),
						Size: 1,
					},
					common.ZeroAddress,
					testCase.value,
				),
				literal: fmt.Sprintf("[%s as %s]", testCase, validType),
			}

		if !testCase.notAsDictionaryKey {

			value := interpreter.NewDictionaryValue(
				inter,
				interpreter.EmptyLocationRange,
				&interpreter.DictionaryStaticType{
					KeyType:   interpreter.ConvertSemaToStaticType(nil, testCase.ty),
					ValueType: interpreter.ConvertSemaToStaticType(nil, testCase.ty),
				},
				testCase.value, testCase.value,
			)

			tests[fmt.Sprintf("{%[1]s: %[1]s}", validType)] =
				testValue{
					value:   value,
					literal: fmt.Sprintf("{%[1]s as %[2]s: %[1]s as %[2]s}", testCase, validType),
				}
		}
	}

	for ty, testCase := range tests {

		t.Run(ty, func(t *testing.T) {

			code := fmt.Sprintf(
				`
                  event Test(_ value: %[1]s)

                  fun test() {
                      emit Test(%[2]s as %[1]s)
                  }
                `,
				ty,
				testCase.String(),
			)

			baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
			baseValueActivation.DeclareValue(stdlib.StandardLibraryValue{
				Name:  "s",
				Type:  sType,
				Value: sValue,
				Kind:  common.DeclarationKindConstant,
			})

			baseTypeActivation := sema.NewVariableActivation(sema.BaseTypeActivation)
			baseTypeActivation.DeclareType(stdlib.StandardLibraryType{
				Name: "S",
				Type: sType,
				Kind: common.DeclarationKindStructure,
			})

			var actualEvents []interpreter.Value

			inter, err := parseCheckAndInterpretWithOptions(
				t, code, ParseCheckAndInterpretOptions{
					CheckerConfig: &sema.Config{
						BaseValueActivationHandler: func(_ common.Location) *sema.VariableActivation {
							return baseValueActivation
						},
						BaseTypeActivationHandler: func(_ common.Location) *sema.VariableActivation {
							return baseTypeActivation
						},
					},
					Config: &interpreter.Config{
						Storage: storage,
						OnEventEmitted: func(
							_ *interpreter.Interpreter,
							_ interpreter.LocationRange,
							event *interpreter.CompositeValue,
							eventType *sema.CompositeType,
						) error {
							actualEvents = append(actualEvents, event)
							return nil
						},
					},
				},
			)
			require.NoError(t, err)

			_, err = inter.Invoke("test")
			require.NoError(t, err)

			testType := RequireGlobalType(t, inter.Program.Elaboration, "Test")

			fields := []interpreter.CompositeField{
				{
					Name:  "value",
					Value: testCase.value,
				},
			}

			expectedEvents := []interpreter.Value{
				interpreter.NewCompositeValue(
					inter,
					interpreter.EmptyLocationRange,
					TestLocation,
					TestLocation.QualifiedIdentifier(testType.ID()),
					common.CompositeKindEvent,
					fields,
					common.ZeroAddress,
				),
			}

			AssertValueSlicesEqual(
				t,
				inter,
				expectedEvents,
				actualEvents,
			)
		})
	}
}

func TestInterpretSwapResourceDictionaryElementReturnSwapped(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      resource X {}

      fun test(): @X? {
          let xs: @{String: X} <- {}
          var x: @X? <- create X()
          xs["foo"] <-> x
          destroy xs
          return <-x
      }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.Nil,
		value,
	)
}

func TestInterpretSwapResourceDictionaryElementReturnDictionary(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      resource X {}

      fun test(): @{String: X} {
          let xs: @{String: X} <- {}
          var x: @X? <- create X()
          xs["foo"] <-> x
          destroy x
          return <-xs
      }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	require.IsType(t,
		&interpreter.DictionaryValue{},
		value,
	)

	foo := value.(*interpreter.DictionaryValue).
		GetKey(inter, interpreter.EmptyLocationRange, interpreter.NewUnmeteredStringValue("foo"))

	require.IsType(t,
		&interpreter.SomeValue{},
		foo,
	)

	assert.IsType(t,
		&interpreter.CompositeValue{},
		foo.(*interpreter.SomeValue).InnerValue(),
	)
}

func TestInterpretSwapResourceDictionaryElementRemoveUsingNil(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      resource X {}

      fun test(): @X? {
          let xs: @{String: X} <- {"foo": <-create X()}
          var x: @X? <- nil
          xs["foo"] <-> x
          destroy xs
          return <-x
      }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	require.IsType(t,
		&interpreter.SomeValue{},
		value,
	)

	assert.IsType(t,
		&interpreter.CompositeValue{},
		value.(*interpreter.SomeValue).InnerValue(),
	)
}

func TestInterpretReferenceExpression(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      resource R {
          access(all) let x: Int

          init(_ x: Int) {
              self.x = x
          }
      }

      fun test(): Int {
          let r <- create R(4)
          let ref = &r as &R
          let x = ref.x
          destroy r
          return x
      }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(4),
		value,
	)
}

func TestInterpretReferenceUse(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      access(all) resource R {
          access(all) var x: Int

          init() {
              self.x = 0
          }

          access(all) fun setX(_ newX: Int) {
              self.x = newX
          }
      }

      access(all) fun test(): [Int] {
          let r <- create R()

          let ref1 = &r as &R
          let ref2 = &r as &R

          ref1.setX(1)
          let x1 = ref1.x
          ref1.setX(2)
          let x2 = ref1.x

          let x3 = ref2.x
          let res = [x1, x2, x3]
          destroy r
          return res
      }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewArrayValue(
			inter,
			interpreter.EmptyLocationRange,
			&interpreter.VariableSizedStaticType{
				Type: interpreter.PrimitiveStaticTypeInt,
			},
			common.ZeroAddress,
			interpreter.NewUnmeteredIntValueFromInt64(1),
			interpreter.NewUnmeteredIntValueFromInt64(2),
			interpreter.NewUnmeteredIntValueFromInt64(2),
		),
		value,
	)
}

func TestInterpretReferenceUseAccess(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      access(all) resource R {
          access(all) var x: Int

          init() {
              self.x = 0
          }

          access(all) fun setX(_ newX: Int) {
              self.x = newX
          }
      }

      access(all) fun test(): [Int] {
          let rs <- [<-create R()]
          let ref = &rs as &[R]
          let x0 = ref[0].x
          ref[0].setX(1)
          let x1 = ref[0].x
          ref[0].setX(2)
          let x2 = ref[0].x
          let res = [x0, x1, x2]
          destroy rs
          return res
      }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewArrayValue(
			inter,
			interpreter.EmptyLocationRange,
			&interpreter.VariableSizedStaticType{
				Type: interpreter.PrimitiveStaticTypeInt,
			},
			common.ZeroAddress,
			interpreter.NewUnmeteredIntValueFromInt64(0),
			interpreter.NewUnmeteredIntValueFromInt64(1),
			interpreter.NewUnmeteredIntValueFromInt64(2),
		),
		value,
	)
}

func TestInterpretVariableDeclarationSecondValue(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      resource R {
          let id: Int
          init(id: Int) {
              self.id = id
          }
      }

      fun test(): @[R?] {
          let x <- create R(id: 1)
          var ys <- {"r": <-create R(id: 2)}
          // NOTE: nested move is valid here
          let z <- ys["r"] <- x

          // NOTE: nested move is invalid here
          let r <- ys.remove(key: "r")

          destroy ys

          return <-[<-z, <-r]
      }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	require.IsType(t,
		&interpreter.ArrayValue{},
		value,
	)

	values := ArrayElements(inter, value.(*interpreter.ArrayValue))

	require.IsType(t,
		&interpreter.SomeValue{},
		values[0],
	)

	firstValue := values[0].(*interpreter.SomeValue).InnerValue()

	require.IsType(t,
		&interpreter.CompositeValue{},
		firstValue,
	)

	firstResource := firstValue.(*interpreter.CompositeValue)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(2),
		firstResource.GetField(inter, "id"),
	)

	require.IsType(t,
		&interpreter.SomeValue{},
		values[1],
	)

	secondValue := values[1].(*interpreter.SomeValue).InnerValue()

	require.IsType(t,
		&interpreter.CompositeValue{},
		secondValue,
	)

	secondResource := secondValue.(*interpreter.CompositeValue)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(1),
		secondResource.GetField(inter, "id"),
	)
}

func TestInterpretCastingIntLiteralToInt8(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let x = 42 as Int8
    `)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredInt8Value(42),
		inter.Globals.Get("x").GetValue(inter),
	)
}

func TestInterpretCastingIntLiteralToAnyStruct(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let x = 42 as AnyStruct
    `)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(42),
		inter.Globals.Get("x").GetValue(inter),
	)
}

func TestInterpretCastingIntLiteralToOptional(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let x = 42 as Int?
    `)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredSomeValueNonCopying(interpreter.NewUnmeteredIntValueFromInt64(42)),
		inter.Globals.Get("x").GetValue(inter),
	)
}

func TestInterpretCastingResourceToAnyResource(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      resource R {}

      fun test(): @AnyResource {
          let r <- create R()
          let x <- r as @AnyResource
          return <-x
      }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	assert.IsType(t,
		&interpreter.CompositeValue{},
		value,
	)
}

func TestInterpretOptionalChainingFieldRead(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t,
		`
          struct Test {
              let x: Int

              init(x: Int) {
                  self.x = x
              }
          }

          let test1: Test? = nil
          let x1 = test1?.x

          let test2: Test? = Test(x: 42)
          let x2 = test2?.x
        `,
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.Nil,
		inter.Globals.Get("x1").GetValue(inter),
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredSomeValueNonCopying(
			interpreter.NewUnmeteredIntValueFromInt64(42),
		),
		inter.Globals.Get("x2").GetValue(inter),
	)
}

func TestInterpretOptionalChainingFunctionRead(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t,
		`
          struct Test {
              fun x(): Int {
                  return 42
              }
          }

          let test1: Test? = nil
          let x1 = test1?.x

          let test2: Test? = Test()
          let x2 = test2?.x
        `,
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.Nil,
		inter.Globals.Get("x1").GetValue(inter),
	)

	require.IsType(t,
		&interpreter.SomeValue{},
		inter.Globals.Get("x2").GetValue(inter),
	)

	assert.IsType(t,
		interpreter.BoundFunctionValue{},
		inter.Globals.Get("x2").GetValue(inter).(*interpreter.SomeValue).InnerValue(),
	)
}

func TestInterpretOptionalChainingFunctionCall(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t,
		`
         struct Test {
             fun x(): Int {
                 return 42
             }
         }

         let test1: Test? = nil
         let x1 = test1?.x()

         let test2: Test? = Test()
         let x2 = test2?.x()
       `,
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.Nil,
		inter.Globals.Get("x1").GetValue(inter),
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredSomeValueNonCopying(
			interpreter.NewUnmeteredIntValueFromInt64(42),
		),
		inter.Globals.Get("x2").GetValue(inter),
	)
}

func TestInterpretOptionalChainingFieldReadAndNilCoalescing(t *testing.T) {

	t.Parallel()

	baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
	baseValueActivation.DeclareValue(stdlib.PanicFunction)

	baseActivation := activations.NewActivation(nil, interpreter.BaseActivation)
	interpreter.Declare(baseActivation, stdlib.PanicFunction)

	inter, err := parseCheckAndInterpretWithOptions(t,
		`
          struct Test {
              let x: Int

              init(x: Int) {
                  self.x = x
              }
          }

          let test: Test? = Test(x: 42)
          let x = test?.x ?? panic("nil")
        `,
		ParseCheckAndInterpretOptions{
			CheckerConfig: &sema.Config{
				BaseValueActivationHandler: func(_ common.Location) *sema.VariableActivation {
					return baseValueActivation
				},
			},
			Config: &interpreter.Config{
				BaseActivationHandler: func(_ common.Location) *interpreter.VariableActivation {
					return baseActivation
				},
			},
		},
	)
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(42),
		inter.Globals.Get("x").GetValue(inter),
	)
}

func TestInterpretOptionalChainingFunctionCallAndNilCoalescing(t *testing.T) {

	t.Parallel()

	baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
	baseValueActivation.DeclareValue(stdlib.PanicFunction)

	baseActivation := activations.NewActivation(nil, interpreter.BaseActivation)
	interpreter.Declare(baseActivation, stdlib.PanicFunction)

	inter, err := parseCheckAndInterpretWithOptions(t,
		`
          struct Test {
              fun x(): Int {
                  return 42
              }
          }

          let test: Test? = Test()
          let x = test?.x() ?? panic("nil")
        `,
		ParseCheckAndInterpretOptions{
			CheckerConfig: &sema.Config{
				BaseValueActivationHandler: func(_ common.Location) *sema.VariableActivation {
					return baseValueActivation
				},
			},
			Config: &interpreter.Config{
				BaseActivationHandler: func(_ common.Location) *interpreter.VariableActivation {
					return baseActivation
				},
			},
		},
	)
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(42),
		inter.Globals.Get("x").GetValue(inter),
	)
}

func TestInterpretOptionalChainingArgumentEvaluation(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t,
		`
          var a = 1
          var b = 1

          fun incA(): Int {
              a = a + 1
              return a
          }

          fun incB(): Int {
              b = b + 1
              return b
          }

          struct Test {
              fun test(_ int: Int) {}
          }

          fun test() {
              let test1: Test? = Test()
              test1?.test(incA())

              let test2: Test? = nil
              test2?.test(incB())
          }
        `,
	)

	_, err := inter.Invoke("test")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewIntValueFromInt64(nil, 2),
		inter.Globals.Get("a").GetValue(inter),
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewIntValueFromInt64(nil, 1),
		inter.Globals.Get("b").GetValue(inter),
	)
}

func TestInterpretCompositeDeclarationNestedTypeScopingOuterInner(t *testing.T) {

	t.Parallel()

	inter, err := parseCheckAndInterpretWithOptions(t,
		`
          access(all) contract Test {

              access(all) struct X {

                  access(all) fun test(): X {
                     return Test.x()
                  }
              }

              access(all) fun x(): X {
                 return X()
              }
          }

          access(all) let x1 = Test.x()
          access(all) let x2 = x1.test()
        `,
		ParseCheckAndInterpretOptions{
			Config: &interpreter.Config{
				ContractValueHandler: makeContractValueHandler(nil, nil, nil),
			},
		},
	)
	require.NoError(t, err)

	x1 := inter.Globals.Get("x1").GetValue(inter)
	x2 := inter.Globals.Get("x2").GetValue(inter)

	require.IsType(t,
		&interpreter.CompositeValue{},
		x1,
	)

	assert.Equal(t,
		sema.TypeID("S.test.Test.X"),
		x1.(*interpreter.CompositeValue).TypeID(),
	)

	require.IsType(t,
		&interpreter.CompositeValue{},
		x2,
	)

	assert.Equal(t,
		sema.TypeID("S.test.Test.X"),
		x2.(*interpreter.CompositeValue).TypeID(),
	)
}

func TestInterpretCompositeDeclarationNestedConstructor(t *testing.T) {

	t.Parallel()

	inter, err := parseCheckAndInterpretWithOptions(t,
		`
          access(all) contract Test {

              access(all) struct X {}
          }

          access(all) let x = Test.X()
        `,
		ParseCheckAndInterpretOptions{
			Config: &interpreter.Config{
				ContractValueHandler: makeContractValueHandler(nil, nil, nil),
			},
		},
	)
	require.NoError(t, err)

	x := inter.Globals.Get("x").GetValue(inter)

	require.IsType(t,
		&interpreter.CompositeValue{},
		x,
	)

	assert.Equal(t,
		sema.TypeID("S.test.Test.X"),
		x.(*interpreter.CompositeValue).TypeID(),
	)
}

func TestInterpretContractAccountFieldUse(t *testing.T) {

	t.Parallel()

	code := `
      access(all) contract Test {
          access(all) let address: Address

          init() {
              // field 'account' can be used, as it is considered initialized
              self.address = self.account.address
          }

          access(all) fun test(): Address {
              return self.account.address
          }
      }

      access(all) let address1 = Test.address
      access(all) let address2 = Test.test()
    `

	t.Run("with custom handler", func(t *testing.T) {
		addressValue := interpreter.AddressValue(common.MustBytesToAddress([]byte{0x1}))

		inter, err := parseCheckAndInterpretWithOptions(t,
			code,
			ParseCheckAndInterpretOptions{
				Config: &interpreter.Config{
					ContractValueHandler: makeContractValueHandler(nil, nil, nil),
					InjectedCompositeFieldsHandler: func(
						inter *interpreter.Interpreter,
						_ common.Location,
						_ string,
						_ common.CompositeKind,
					) map[string]interpreter.Value {

						accountRef := stdlib.NewAccountReferenceValue(
							nil,
							nil,
							addressValue,
							interpreter.FullyEntitledAccountAccess,
							interpreter.EmptyLocationRange,
						)

						return map[string]interpreter.Value{
							sema.ContractAccountFieldName: accountRef,
						}
					},
				},
			},
		)
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			addressValue,
			inter.Globals.Get("address1").GetValue(inter),
		)

		AssertValuesEqual(
			t,
			inter,
			addressValue,
			inter.Globals.Get("address2").GetValue(inter),
		)
	})

	t.Run("with default handler", func(t *testing.T) {
		env := runtime.NewBaseInterpreterEnvironment(runtime.Config{})
		_, err := parseCheckAndInterpretWithOptions(t, code,
			ParseCheckAndInterpretOptions{
				Config: &interpreter.Config{
					ContractValueHandler:           makeContractValueHandler(nil, nil, nil),
					InjectedCompositeFieldsHandler: env.InterpreterConfig.InjectedCompositeFieldsHandler,
				},
			},
		)
		require.Error(t, err)
		assert.ErrorContains(t, err, "error: member `account` is used before it has been initialized")
	})
}

func TestInterpretConformToImportedInterface(t *testing.T) {

	t.Parallel()

	importedChecker, err := ParseAndCheckWithOptions(t,
		`
          struct interface Foo {
              fun check(answer: Int) {
                  pre {
                      answer == 42
                  }
              }
          }
        `,
		ParseAndCheckOptions{
			Location: ImportedLocation,
		},
	)
	require.NoError(t, err)

	importingChecker, err := ParseAndCheckWithOptions(t,
		`
          import Foo from "imported"

          struct Bar: Foo {
              fun check(answer: Int) {}
          }

          fun test() {
              let bar = Bar()
              bar.check(answer: 1)
          }
        `,
		ParseAndCheckOptions{
			Config: &sema.Config{
				ImportHandler: func(_ *sema.Checker, importedLocation common.Location, _ ast.Range) (sema.Import, error) {
					assert.Equal(t,
						ImportedLocation,
						importedLocation,
					)

					return sema.ElaborationImport{
						Elaboration: importedChecker.Elaboration,
					}, nil
				},
			},
		},
	)
	require.NoError(t, err)

	storage := newUnmeteredInMemoryStorage()

	inter, err := interpreter.NewInterpreter(
		interpreter.ProgramFromChecker(importingChecker),
		importingChecker.Location,
		&interpreter.Config{
			Storage: storage,
			ImportLocationHandler: func(inter *interpreter.Interpreter, location common.Location) interpreter.Import {
				assert.Equal(t,
					ImportedLocation,
					location,
				)

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
	)
	require.NoError(t, err)

	err = inter.Interpret()
	require.NoError(t, err)

	_, err = inter.Invoke("test")
	require.IsType(t,
		interpreter.Error{},
		err,
	)
	interpreterErr := err.(interpreter.Error)

	require.IsType(t,
		interpreter.ConditionError{},
		interpreterErr.Err,
	)
}

func TestInterpretContractUseInNestedDeclaration(t *testing.T) {

	t.Parallel()

	inter, err := parseCheckAndInterpretWithOptions(t, `
          access(all) contract C {

              access(all) var i: Int

              access(all) struct S {

                  init() {
                      C.i = C.i + 1
                  }
              }

              init () {
                  self.i = 0
                  S()
                  S()
              }
          }
        `,
		ParseCheckAndInterpretOptions{
			Config: &interpreter.Config{
				ContractValueHandler: makeContractValueHandler(nil, nil, nil),
			},
		},
	)
	require.NoError(t, err)

	i := inter.Globals.Get("C").GetValue(inter).(interpreter.MemberAccessibleValue).
		GetMember(inter, interpreter.EmptyLocationRange, "i")

	require.IsType(t,
		interpreter.NewUnmeteredIntValueFromInt64(2),
		i,
	)
}

func TestInterpretNonStorageReference(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t,
		`
          resource NFT {
              var id: Int

              init(id: Int) {
                  self.id = id
              }
          }

          fun test(): Int {
              let resources <- [
                  <-create NFT(id: 1),
                  <-create NFT(id: 2)
              ]

              let nftRef = &resources[1] as &NFT
              let nftRef2 = nftRef
              nftRef2.id = 3

              let nft <- resources.remove(at: 1)
              destroy resources
              let newID = nft.id
              destroy nft

              return newID
          }
        `,
	)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter, interpreter.NewUnmeteredIntValueFromInt64(3), value)
}

func TestInterpretNonStorageReferenceToOptional(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t,
		`
          resource Foo {
              let name: String

              init(name: String) {
                  self.name = name
              }
          }

          fun testSome(): String {
              let xs: @{String: Foo} <- {"yes": <-create Foo(name: "YES")}
              let ref = (&xs["yes"] as &Foo?)!
              let name = ref.name
              destroy xs
              return name
          }

          fun testNil(): String {
              let xs: @{String: Foo} <- {}
              let ref = (&xs["no"] as &Foo?)!
              let name = ref.name
              destroy xs
              return name
          }
        `,
	)
	t.Run("some", func(t *testing.T) {
		value, err := inter.Invoke("testSome")
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter, interpreter.NewUnmeteredStringValue("YES"), value)
	})

	t.Run("nil", func(t *testing.T) {
		_, err := inter.Invoke("testNil")
		RequireError(t, err)

		require.ErrorAs(t, err, &interpreter.ForceNilError{})
	})
}

func TestInterpretFix64(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t,
		`
          let a = 789.00123010
          let b = 1234.056
          let c = -12345.006789
        `,
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredUFix64Value(78_900_123_010),
		inter.Globals.Get("a").GetValue(inter),
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredUFix64Value(123_405_600_000),
		inter.Globals.Get("b").GetValue(inter),
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredFix64Value(-1_234_500_678_900),
		inter.Globals.Get("c").GetValue(inter),
	)
}

func TestInterpretFix64Mul(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t,
		`
          let a = Fix64(1.1) * -1.1
        `,
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredFix64Value(-121000000),
		inter.Globals.Get("a").GetValue(inter),
	)
}

func TestInterpretHexDecode(t *testing.T) {

	t.Parallel()

	expected := []interpreter.Value{
		interpreter.NewUnmeteredUInt8Value(71),
		interpreter.NewUnmeteredUInt8Value(111),
		interpreter.NewUnmeteredUInt8Value(32),
		interpreter.NewUnmeteredUInt8Value(87),
		interpreter.NewUnmeteredUInt8Value(105),
		interpreter.NewUnmeteredUInt8Value(116),
		interpreter.NewUnmeteredUInt8Value(104),
		interpreter.NewUnmeteredUInt8Value(32),
		interpreter.NewUnmeteredUInt8Value(116),
		interpreter.NewUnmeteredUInt8Value(104),
		interpreter.NewUnmeteredUInt8Value(101),
		interpreter.NewUnmeteredUInt8Value(32),
		interpreter.NewUnmeteredUInt8Value(70),
		interpreter.NewUnmeteredUInt8Value(108),
		interpreter.NewUnmeteredUInt8Value(111),
		interpreter.NewUnmeteredUInt8Value(119),
	}

	t.Run("in Cadence", func(t *testing.T) {

		baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
		baseValueActivation.DeclareValue(stdlib.PanicFunction)

		baseActivation := activations.NewActivation(nil, interpreter.BaseActivation)
		interpreter.Declare(baseActivation, stdlib.PanicFunction)

		inter, err := parseCheckAndInterpretWithOptions(t,
			`
              fun hexDecode(_ s: String): [UInt8] {
                  if s.length % 2 != 0 {
                      panic("Input must have even number of characters")
                  }
                  let table: {String: UInt8} = {
                          "0" : 0,
                          "1" : 1,
                          "2" : 2,
                          "3" : 3,
                          "4" : 4,
                          "5" : 5,
                          "6" : 6,
                          "7" : 7,
                          "8" : 8,
                          "9" : 9,
                          "a" : 10,
                          "A" : 10,
                          "b" : 11,
                          "B" : 11,
                          "c" : 12,
                          "C" : 12,
                          "d" : 13,
                          "D" : 13,
                          "e" : 14,
                          "E" : 14,
                          "f" : 15,
                          "F" : 15
                      }
                  let length = s.length / 2
                  var i = 0
                  var res: [UInt8] = []
                  while i < length {
                      let c = s.slice(from: i * 2, upTo: i * 2 + 1)
                      let in1 = table[c] ?? panic("Invalid character ".concat(c))
                      let c2 = s.slice(from: i * 2 + 1, upTo: i * 2 + 2)
                      let in2 = table[c2] ?? panic("Invalid character ".concat(c2))
                      res.append((16 as UInt8) * in1 + in2)
                      i = i + 1
                  }
                  return res
              }

              fun test(): [UInt8] {
                  return hexDecode("476F20576974682074686520466C6F77")
              }
            `,
			ParseCheckAndInterpretOptions{
				CheckerConfig: &sema.Config{
					BaseValueActivationHandler: func(_ common.Location) *sema.VariableActivation {
						return baseValueActivation
					},
				},
				Config: &interpreter.Config{
					BaseActivationHandler: func(_ common.Location) *interpreter.VariableActivation {
						return baseActivation
					},
				},
			},
		)
		require.NoError(t, err)

		result, err := inter.Invoke("test")
		require.NoError(t, err)

		require.IsType(t, result, &interpreter.ArrayValue{})
		arrayValue := result.(*interpreter.ArrayValue)

		AssertValueSlicesEqual(
			t,
			inter,
			expected,
			ArrayElements(inter, arrayValue),
		)
	})

	t.Run("native", func(t *testing.T) {

		inter := parseCheckAndInterpret(t,
			`
              fun test(): [UInt8] {
                  return "476F20576974682074686520466C6F77".decodeHex()
              }
            `,
		)

		result, err := inter.Invoke("test")
		require.NoError(t, err)

		require.IsType(t, result, &interpreter.ArrayValue{})
		arrayValue := result.(*interpreter.ArrayValue)

		AssertValueSlicesEqual(
			t,
			inter,
			expected,
			ArrayElements(inter, arrayValue),
		)
	})

}

func TestInterpretOptionalChainingOptionalFieldRead(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      struct Test {
          let x: Int?

          init(x: Int?) {
              self.x = x
          }
      }

      let test: Test? = Test(x: 1)
      let x = test?.x
    `)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredSomeValueNonCopying(
			interpreter.NewUnmeteredIntValueFromInt64(1),
		),
		inter.Globals.Get("x").GetValue(inter),
	)
}

func TestInterpretReferenceUseAfterCopy(t *testing.T) {

	t.Parallel()

	t.Run("struct, field write and read", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          struct S {
              var name: String
              init(name: String) {
                  self.name = name
              }
          }

          fun test(): [String] {
              let s = S(name: "1")
              let ref = &s as &S
              let container = [s]
              ref.name = "2"
              container[0].name = "3"
              let s2 = container.remove(at: 0)
              return [s.name, s2.name]
          }
        `)

		result, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewArrayValue(
				inter,
				interpreter.EmptyLocationRange,
				&interpreter.VariableSizedStaticType{
					Type: interpreter.PrimitiveStaticTypeString,
				},
				common.ZeroAddress,
				interpreter.NewUnmeteredStringValue("2"),
				interpreter.NewUnmeteredStringValue("3"),
			),
			result,
		)
	})
}

func TestInterpretResourceOwnerFieldUse(t *testing.T) {

	t.Parallel()

	code := `
      access(all) resource R {}

      access(all) fun test(): [Address?] {
          let addresses: [Address?] = []

          let r <- create R()
          addresses.append(r.owner?.address)

          account.storage.save(<-r, to: /storage/r)

          let ref = account.storage.borrow<&R>(from: /storage/r)
          addresses.append(ref?.owner?.address)

          return addresses
      }
    `
	// `authAccount`

	address := common.MustBytesToAddress([]byte{0x1})

	valueDeclaration := stdlib.StandardLibraryValue{
		Name: "account",
		Type: sema.FullyEntitledAccountReferenceType,
		Value: stdlib.NewAccountReferenceValue(
			nil,
			nil,
			interpreter.AddressValue(address),
			interpreter.FullyEntitledAccountAccess,
			interpreter.EmptyLocationRange,
		),
		Kind: common.DeclarationKindConstant,
	}

	baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
	baseValueActivation.DeclareValue(valueDeclaration)

	baseActivation := activations.NewActivation(nil, interpreter.BaseActivation)
	interpreter.Declare(baseActivation, valueDeclaration)

	inter, err := parseCheckAndInterpretWithOptions(t,
		code,
		ParseCheckAndInterpretOptions{
			CheckerConfig: &sema.Config{
				BaseValueActivationHandler: func(_ common.Location) *sema.VariableActivation {
					return baseValueActivation
				},
			},
			Config: &interpreter.Config{
				BaseActivationHandler: func(_ common.Location) *interpreter.VariableActivation {
					return baseActivation
				},
				AccountHandler: func(inter *interpreter.Interpreter, address interpreter.AddressValue) interpreter.Value {
					return stdlib.NewAccountValue(inter, nil, address)
				},
			},
		},
	)
	require.NoError(t, err)

	result, err := inter.Invoke("test")
	require.NoError(t, err)

	AssertValueSlicesEqual(
		t,
		inter,
		[]interpreter.Value{
			interpreter.Nil,
			interpreter.NewUnmeteredSomeValueNonCopying(interpreter.AddressValue(address)),
		},
		ArrayElements(inter, result.(*interpreter.ArrayValue)),
	)
}

func TestInterpretResourceAssignmentForceTransfer(t *testing.T) {

	t.Parallel()

	t.Run("new to nil", func(t *testing.T) {

		inter := parseCheckAndInterpret(t, `
          resource X {}

          fun test() {
              var x: @X? <- nil
              x <-! create X()
              destroy x
          }
        `)

		_, err := inter.Invoke("test")
		require.NoError(t, err)
	})

	t.Run("new to non-nil", func(t *testing.T) {

		inter := parseCheckAndInterpret(t, `
         resource X {}

         fun test() {
             var x: @X? <- create X()
             x <-! create X()
             destroy x
         }
       `)

		_, err := inter.Invoke("test")
		RequireError(t, err)

		require.ErrorAs(t, err, &interpreter.ResourceLossError{})
	})

	t.Run("existing to nil", func(t *testing.T) {

		inter := parseCheckAndInterpret(t, `
         resource X {}

         fun test() {
             let x <- create X()
             var x2: @X? <- nil
             x2 <-! x
             destroy x2
         }
       `)

		_, err := inter.Invoke("test")
		require.NoError(t, err)
	})

	t.Run("existing to non-nil", func(t *testing.T) {

		inter := parseCheckAndInterpret(t, `
         resource X {}

         fun test() {
             let x <- create X()
             var x2: @X? <- create X()
             x2 <-! x
             destroy x2
         }
       `)

		_, err := inter.Invoke("test")
		RequireError(t, err)

		require.ErrorAs(t, err, &interpreter.ResourceLossError{})
	})

	t.Run("force-assignment initialization", func(t *testing.T) {

		inter := parseCheckAndInterpret(t, `
         resource X {}

         resource Y {

             var x: @X?

             init() {
                 self.x <-! create X()
             }
         }

         fun test() {
             let y <- create Y()
             destroy y
         }
       `)

		_, err := inter.Invoke("test")
		require.NoError(t, err)
	})

}

func TestInterpretForce(t *testing.T) {

	t.Parallel()

	t.Run("non-nil", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          let x: Int? = 1
          let y = x!
        `)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredSomeValueNonCopying(
				interpreter.NewUnmeteredIntValueFromInt64(1),
			),
			inter.Globals.Get("x").GetValue(inter),
		)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredIntValueFromInt64(1),
			inter.Globals.Get("y").GetValue(inter),
		)
	})

	t.Run("nil", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          let x: Int? = nil

          fun test(): Int {
              return x!
          }
        `)

		_, err := inter.Invoke("test")
		RequireError(t, err)

		require.ErrorAs(t, err, &interpreter.ForceNilError{})
	})

	t.Run("non-optional", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          let x: Int = 1
          let y = x!
        `)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredIntValueFromInt64(1),
			inter.Globals.Get("y").GetValue(inter),
		)
	})
}

func TestInterpretEphemeralReferenceToOptional(t *testing.T) {

	t.Parallel()

	_, err := parseCheckAndInterpretWithOptions(t,
		`
          contract C {

              var rs: @{Int: R}

              resource R {
                  access(all) let id: Int

                  init(id: Int) {
                      self.id = id
                  }
              }

              fun borrow(id: Int): &R? {
                  return &C.rs[id] as &R?
              }

              init() {
                  self.rs <- {}
                  self.rs[1] <-! create R(id: 1)
                  let ref = self.borrow(id: 1)!
                  ref.id
              }
          }
        `,
		ParseCheckAndInterpretOptions{
			Config: &interpreter.Config{
				ContractValueHandler: makeContractValueHandler(nil, nil, nil),
			},
		},
	)
	require.NoError(t, err)
}

func TestInterpretNestedDeclarationOrder(t *testing.T) {

	t.Parallel()

	t.Run("A, B", func(t *testing.T) {

		t.Parallel()

		_, err := parseCheckAndInterpretWithOptions(t,
			`
              access(all) contract Test {

                  access(all) resource A {

                      access(all) fun b(): @B {
                          return <-create B()
                      }
                  }

                  access(all) resource B {}

                  init() {
                      let a <- create A()
                      let b <- a.b()
                      destroy a
                      destroy b
                  }
              }
            `,
			ParseCheckAndInterpretOptions{
				Config: &interpreter.Config{
					ContractValueHandler: makeContractValueHandler(nil, nil, nil),
				},
			},
		)
		require.NoError(t, err)
	})

	t.Run("B, A", func(t *testing.T) {

		t.Parallel()

		_, err := parseCheckAndInterpretWithOptions(t,
			`
              access(all) contract Test {

                  access(all) resource B {}

                  access(all) resource A {

                      access(all) fun b(): @B {
                          return <-create B()
                      }
                  }

                  init() {
                      let a <- create A()
                      let b <- a.b()
                      destroy a
                      destroy b
                  }
              }
            `,
			ParseCheckAndInterpretOptions{
				Config: &interpreter.Config{
					ContractValueHandler: makeContractValueHandler(nil, nil, nil),
				},
			},
		)
		require.NoError(t, err)
	})
}

func TestInterpretCountDigits256(t *testing.T) {

	t.Parallel()

	type test struct {
		Type    sema.Type
		Literal string
		Count   int
	}

	for _, test := range []test{
		{
			Type:    sema.Int256Type,
			Literal: "676983016644359394637212096269997871684197836659065544033845082275068334",
			Count:   72,
		},
		{
			Type:    sema.UInt256Type,
			Literal: "676983016644359394637212096269997871684197836659065544033845082275068334",
			Count:   72,
		},
		{
			Type:    sema.Int128Type,
			Literal: "676983016644359394637212096269997871",
			Count:   36,
		},
		{
			Type:    sema.UInt128Type,
			Literal: "676983016644359394637212096269997871",
			Count:   36,
		},
	} {

		t.Run(test.Type.String(), func(t *testing.T) {

			inter := parseCheckAndInterpret(t,
				fmt.Sprintf(
					`
                      fun countDigits(_ x: %[2]s): UInt8 {
                          var count: UInt8 = UInt8(0)
                          var input = x
                          while input != %[2]s(0) {
                              count = count + UInt8(1)
                              input = input / %[2]s(10)
                          }
                          return count
                      }

                      let number: %[2]s = %[1]s
                      let result1 = countDigits(%[1]s)
                      let result2 = countDigits(%[2]s(%[1]s))
                      let result3 = countDigits(number)
                    `,
					test.Literal,
					test.Type,
				),
			)

			bigInt, ok := new(big.Int).SetString(test.Literal, 10)
			require.True(t, ok)

			assert.Equal(t,
				bigInt,
				inter.Globals.Get("number").GetValue(inter).(interpreter.BigNumberValue).ToBigInt(nil),
			)

			expected := interpreter.NewUnmeteredUInt8Value(uint8(test.Count))

			for i := 1; i <= 3; i++ {
				variableName := fmt.Sprintf("result%d", i)
				AssertValuesEqual(
					t,
					inter,
					expected,
					inter.Globals.Get(variableName).GetValue(inter),
				)
			}
		})
	}
}

func TestInterpretFailableCastingCompositeTypeConfusion(t *testing.T) {

	t.Parallel()

	inter, err := parseCheckAndInterpretWithOptions(t,
		`
          contract A {
              struct S {}
          }

          contract B {
              struct S {}
          }

          let s = A.S() as? B.S
        `,
		ParseCheckAndInterpretOptions{
			Config: &interpreter.Config{
				ContractValueHandler: makeContractValueHandler(nil, nil, nil),
			},
		},
	)
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.Nil,
		inter.Globals.Get("s").GetValue(inter),
	)
}

func TestInterpretNestedDestroy(t *testing.T) {

	t.Parallel()

	var events []*interpreter.CompositeValue

	inter, err := parseCheckAndInterpretWithOptions(t, `
      resource B {
            let id: Int
			init(_ id: Int){
				self.id = id
			}

			event ResourceDestroyed(id: Int = self.id)
		}

		resource A {
			let id: Int
			let bs: @[B]

			event ResourceDestroyed(id: Int = self.id, bCount: Int = self.bs.length)

			init(_ id: Int){
				self.id = id
				self.bs <- []
			}

			fun add(_ b: @B){
				self.bs.append(<-b)
			}
		}

      fun test() {
          let a <- create A(1)
          a.add(<- create B(2))
          a.add(<- create B(3))
          a.add(<- create B(4))

              destroy a
          }
        `, ParseCheckAndInterpretOptions{
		Config: &interpreter.Config{
			OnEventEmitted: func(_ *interpreter.Interpreter, _ interpreter.LocationRange, event *interpreter.CompositeValue, eventType *sema.CompositeType) error {
				events = append(events, event)
				return nil
			},
		},
	})
	require.NoError(t, err)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	require.Len(t, events, 4)
	require.Equal(t, "B.ResourceDestroyed", events[0].QualifiedIdentifier)
	require.Equal(t, interpreter.NewIntValueFromInt64(nil, 2), events[0].GetField(inter, "id"))
	require.Equal(t, "B.ResourceDestroyed", events[1].QualifiedIdentifier)
	require.Equal(t, interpreter.NewIntValueFromInt64(nil, 3), events[1].GetField(inter, "id"))
	require.Equal(t, "B.ResourceDestroyed", events[2].QualifiedIdentifier)
	require.Equal(t, interpreter.NewIntValueFromInt64(nil, 4), events[2].GetField(inter, "id"))
	require.Equal(t, "A.ResourceDestroyed", events[3].QualifiedIdentifier)
	require.Equal(t, interpreter.NewIntValueFromInt64(nil, 1), events[3].GetField(inter, "id"))
	require.Equal(t, interpreter.NewIntValueFromInt64(nil, 3), events[3].GetField(inter, "bCount"))

	AssertValuesEqual(
		t,
		inter,
		interpreter.Void,
		value,
	)
}

// TestInterpretInternalAssignment ensures that a modification of an "internal" value
// is not possible, because the value that is assigned into is a copy
func TestInterpretInternalAssignment(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
       struct S {
           access(self) let xs: {String: Int}

           init() {
               self.xs = {"a": 1}
           }

           fun getXS(): {String: Int} {
               return self.xs
           }
       }

       fun test(): [{String: Int}] {
           let s = S()
           let xs = s.getXS()
           xs["b"] = 2
           return [xs, s.getXS()]
       }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	stringIntDictionaryStaticType := &interpreter.DictionaryStaticType{
		KeyType:   interpreter.PrimitiveStaticTypeString,
		ValueType: interpreter.PrimitiveStaticTypeInt,
	}

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewArrayValue(
			inter,
			interpreter.EmptyLocationRange,
			&interpreter.VariableSizedStaticType{
				Type: stringIntDictionaryStaticType,
			},
			common.ZeroAddress,
			interpreter.NewDictionaryValue(
				inter,
				interpreter.EmptyLocationRange,
				stringIntDictionaryStaticType,
				interpreter.NewUnmeteredStringValue("a"),
				interpreter.NewUnmeteredIntValueFromInt64(1),
				interpreter.NewUnmeteredStringValue("b"),
				interpreter.NewUnmeteredIntValueFromInt64(2),
			),
			interpreter.NewDictionaryValue(
				inter,
				interpreter.EmptyLocationRange,
				stringIntDictionaryStaticType,
				interpreter.NewUnmeteredStringValue("a"),
				interpreter.NewUnmeteredIntValueFromInt64(1),
			),
		),
		value,
	)
}

func TestInterpretVoidReturn_(t *testing.T) {
	t.Parallel()

	labelNamed := func(s string) string {
		if s == "" {
			return "unnamed"
		}
		return "named"
	}

	test := func(testName, returnType, returnValue string) {
		var returnSnippet string

		if returnType != "" {
			returnSnippet = ": " + returnType
		}

		var name string
		if testName == "" {
			name = fmt.Sprintf("%s type, %s value", labelNamed(returnType), labelNamed(returnValue))
		} else {
			name = testName
		}

		code := fmt.Sprintf(
			`fun test() %s { return %s }`,
			returnSnippet,
			returnValue,
		)

		t.Run(name, func(t *testing.T) {
			t.Parallel()
			inter := parseCheckAndInterpret(t, code)

			value, err := inter.Invoke("test")
			require.NoError(t, err)

			AssertValuesEqual(t, inter, &interpreter.VoidValue{}, value)
		})
	}

	typeNames := []string{"", "Void"}
	valueNames := []string{"", "()"}

	for _, typ := range typeNames {
		for _, val := range valueNames {
			test("", typ, val)
		}
	}

	test("inline lambda expression", "", "fun(){}()")
	test(
		"inline inline lambda expression",
		"Void",
		`(fun(v: Void): Void {
			let w = fun() { };
			let x: Void = w();
			let y: Void = ();
			let z = v;
			return z;
		 })( () )`,
	)
}

func TestInterpretCopyOnReturn(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t,
		`
          let xs: {String: String} = {}

          fun returnXS(): {String: String} {
              return xs
          }

          fun test(): {String: String} {
              returnXS().insert(key: "foo", "bar")
              return xs
          }
        `,
	)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewDictionaryValue(
			inter,
			interpreter.EmptyLocationRange,
			&interpreter.DictionaryStaticType{
				KeyType:   interpreter.PrimitiveStaticTypeString,
				ValueType: interpreter.PrimitiveStaticTypeString,
			},
		),
		value,
	)
}

func BenchmarkInterpretRecursionFib(b *testing.B) {

	inter := parseCheckAndInterpret(b, `
       fun fib(_ n: Int): Int {
           if n < 2 {
              return n
           }
           return fib(n - 1) + fib(n - 2)
       }
   `)

	expected := interpreter.NewUnmeteredIntValueFromInt64(377)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {

		result, err := inter.Invoke(
			"fib",
			interpreter.NewUnmeteredIntValueFromInt64(14),
		)
		require.NoError(b, err)
		RequireValuesEqual(b, inter, expected, result)
	}
}

func TestInterpretMissingMember(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t,
		`
          struct X {
              let y: Int

              init() {
                  self.y = 1
              }
          }

          let x = X()

          fun test() {
              // access missing field y
              x.y
          }
        `,
	)

	// Remove field `y`
	compositeValue := inter.Globals.Get("x").GetValue(inter).(*interpreter.CompositeValue)
	compositeValue.RemoveField(inter, interpreter.EmptyLocationRange, "y")

	_, err := inter.Invoke("test")
	RequireError(t, err)

	var missingMemberError interpreter.UseBeforeInitializationError
	require.ErrorAs(t, err, &missingMemberError)

	require.Equal(t, "y", missingMemberError.Name)
}

func BenchmarkNewInterpreter(b *testing.B) {

	b.Run("new interpreter", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_, err := interpreter.NewInterpreter(nil, nil, &interpreter.Config{})
			require.NoError(b, err)
		}
	})

	b.Run("new sub-interpreter", func(b *testing.B) {
		b.ReportAllocs()

		inter, err := interpreter.NewInterpreter(nil, nil, &interpreter.Config{})
		require.NoError(b, err)

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, err := inter.NewSubInterpreter(nil, nil)
			require.NoError(b, err)
		}
	})
}

func TestInterpretHostFunctionStaticType(t *testing.T) {

	t.Parallel()

	t.Run("toString function", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            let x = 5
            let y = x.toString
        `)

		value := inter.Globals.Get("y").GetValue(inter)
		assert.Equal(
			t,
			interpreter.ConvertSemaToStaticType(nil, sema.ToStringFunctionType),
			value.StaticType(inter),
		)
	})

	t.Run("Type function", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            let x = Type
            let y = x<Int8>()
        `)

		value := inter.Globals.Get("x").GetValue(inter)
		assert.Equal(
			t,
			interpreter.ConvertSemaToStaticType(
				nil,
				&sema.FunctionType{
					Purity:               sema.FunctionPurityView,
					ReturnTypeAnnotation: sema.MetaTypeAnnotation,
				},
			),
			value.StaticType(inter),
		)

		value = inter.Globals.Get("y").GetValue(inter)
		assert.Equal(
			t,
			interpreter.PrimitiveStaticTypeMetaType,
			value.StaticType(inter),
		)

		require.IsType(t, interpreter.TypeValue{}, value)
		typeValue := value.(interpreter.TypeValue)
		assert.Equal(t, interpreter.PrimitiveStaticTypeInt8, typeValue.Type)
	})

	t.Run("toString function", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            let a: Int8 = 5
            let b: Fix64 = 4.0

            let x = a.toString
            let y = b.toString
        `)

		// Both `x` and `y` are two functions that returns a string.
		// Hence, their types are equal. i.e: Receivers shouldn't matter.

		xValue := inter.Globals.Get("x").GetValue(inter)
		assert.Equal(
			t,
			interpreter.ConvertSemaToStaticType(nil, sema.ToStringFunctionType),
			xValue.StaticType(inter),
		)

		yValue := inter.Globals.Get("y").GetValue(inter)
		assert.Equal(
			t,
			interpreter.ConvertSemaToStaticType(nil, sema.ToStringFunctionType),
			yValue.StaticType(inter),
		)

		assert.Equal(t, xValue.StaticType(inter), yValue.StaticType(inter))
	})
}

func TestInterpretArrayTypeInference(t *testing.T) {

	t.Parallel()

	t.Run("anystruct with empty array", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            fun test(): Type {
                let x: AnyStruct = []
                return x.getType()
            }
        `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.TypeValue{
				Type: &interpreter.VariableSizedStaticType{
					Type: interpreter.PrimitiveStaticTypeAnyStruct,
				},
			},
			value,
		)
	})

	t.Run("anystruct with numeric array", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            fun test(): Type {
                let x: AnyStruct = [1, 2, 3]
                return x.getType()
            }
        `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.TypeValue{
				Type: &interpreter.VariableSizedStaticType{
					Type: interpreter.PrimitiveStaticTypeInt,
				},
			},
			value,
		)
	})
}

func TestInterpretArrayFirstIndex(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let xs = [1, 2, 3]

      fun test(): Int? {
          return xs.firstIndex(of: 2)
      }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredSomeValueNonCopying(
			interpreter.NewUnmeteredIntValueFromInt64(1),
		),
		value,
	)
}

func TestInterpretArrayFirstIndexDoesNotExist(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let xs = [1, 2, 3]

      fun test(): Int? {
      return xs.firstIndex(of: 5)
      }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.Nil,
		value,
	)
}

func TestInterpretArrayReverse(t *testing.T) {
	t.Parallel()

	inter := parseCheckAndInterpret(t, `
		let xs = [1, 2, 3, 100, 200]
		let ys = [100, 467, 297, 23]
		let xs_fixed: [Int; 5] = [1, 2, 3, 100, 200]
		let ys_fixed: [Int; 4] = [100, 467, 297, 23]
		let emptyVals: [Int] = []
		let emptyVals_fixed: [Int; 0] = []

		fun reversexs(): [Int] {
			return xs.reverse()
		}
		fun originalxs(): [Int] {
			return xs
		}

		fun reverseys(): [Int] {
			return ys.reverse()
		}
		fun originalys(): [Int] {
			return ys
		}

		fun reversexs_fixed(): [Int; 5] {
			return xs_fixed.reverse()
		}
		fun originalxs_fixed(): [Int; 5] {
			return xs_fixed
		}

		fun reverseys_fixed(): [Int; 4] {
			return ys_fixed.reverse()
		}
		fun originalys_fixed(): [Int; 4] {
			return ys_fixed
		}

		fun reverseempty(): [Int] {
			return emptyVals.reverse()
		}
		fun originalempty(): [Int] {
			return emptyVals
		}

		fun reverseempty_fixed(): [Int; 0] {
			return emptyVals_fixed.reverse()
		}
		fun originalempty_fixed(): [Int; 0] {
			return emptyVals_fixed
		}

		access(all)  struct TestStruct {
			access(all)  var test: Int

			init(_ t: Int) {
				self.test = t
			}
		}

		let sa = [TestStruct(1), TestStruct(2), TestStruct(3)]
		let sa_fixed: [TestStruct; 3] = [TestStruct(1), TestStruct(2), TestStruct(3)]

		fun reversesa(): [Int] {
			let sa_rev = sa.reverse()

			let res: [Int] = [];
			for s in sa_rev {
				res.append(s.test)
			}

			return res
		}

		fun originalsa(): [Int] {
			let res: [Int] = [];
			for s in sa {
				res.append(s.test)
			}

			return res
		}

		fun reversesa_fixed(): [Int] {
			let sa_rev = sa_fixed.reverse()

			let res: [Int] = [];
			for s in sa_rev {
				res.append(s.test)
			}

			return res
		}

		fun originalsa_fixed(): [Int] {
			let res: [Int] = [];
			for s in sa_fixed {
				res.append(s.test)
			}

			return res
		}
	`)

	runValidCase := func(t *testing.T, reverseFuncName, originalFuncName string, reversedArray, originalArray *interpreter.ArrayValue) {
		val, err := inter.Invoke(reverseFuncName)
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			reversedArray,
			val,
		)

		origVal, err := inter.Invoke(originalFuncName)
		require.NoError(t, err)

		// Original array remains unchanged
		AssertValuesEqual(
			t,
			inter,
			originalArray,
			origVal,
		)
	}

	for _, suffix := range []string{"_fixed", ""} {
		fixed := suffix == "_fixed"

		var arrayType interpreter.ArrayStaticType
		if fixed {
			arrayType = &interpreter.ConstantSizedStaticType{
				Type: interpreter.PrimitiveStaticTypeInt,
			}
		} else {
			arrayType = &interpreter.VariableSizedStaticType{
				Type: interpreter.PrimitiveStaticTypeInt,
			}
		}

		setFixedSize := func(size int64) {
			if fixed {
				constSized, ok := arrayType.(*interpreter.ConstantSizedStaticType)
				assert.True(t, ok)

				constSized.Size = size
			}
		}

		setFixedSize(0)
		runValidCase(t, "reverseempty"+suffix, "originalempty"+suffix,
			interpreter.NewArrayValue(
				inter,
				interpreter.EmptyLocationRange,
				arrayType,
				common.ZeroAddress,
			), interpreter.NewArrayValue(
				inter,
				interpreter.EmptyLocationRange,
				arrayType,
				common.ZeroAddress,
			))

		setFixedSize(5)
		runValidCase(t, "reversexs"+suffix, "originalxs"+suffix,
			interpreter.NewArrayValue(
				inter,
				interpreter.EmptyLocationRange,
				arrayType,
				common.ZeroAddress,
				interpreter.NewUnmeteredIntValueFromInt64(200),
				interpreter.NewUnmeteredIntValueFromInt64(100),
				interpreter.NewUnmeteredIntValueFromInt64(3),
				interpreter.NewUnmeteredIntValueFromInt64(2),
				interpreter.NewUnmeteredIntValueFromInt64(1),
			), interpreter.NewArrayValue(
				inter,
				interpreter.EmptyLocationRange,
				arrayType,
				common.ZeroAddress,
				interpreter.NewUnmeteredIntValueFromInt64(1),
				interpreter.NewUnmeteredIntValueFromInt64(2),
				interpreter.NewUnmeteredIntValueFromInt64(3),
				interpreter.NewUnmeteredIntValueFromInt64(100),
				interpreter.NewUnmeteredIntValueFromInt64(200),
			))

		setFixedSize(4)
		runValidCase(t, "reverseys"+suffix, "originalys"+suffix,
			interpreter.NewArrayValue(
				inter,
				interpreter.EmptyLocationRange,
				arrayType,
				common.ZeroAddress,
				interpreter.NewUnmeteredIntValueFromInt64(23),
				interpreter.NewUnmeteredIntValueFromInt64(297),
				interpreter.NewUnmeteredIntValueFromInt64(467),
				interpreter.NewUnmeteredIntValueFromInt64(100),
			), interpreter.NewArrayValue(
				inter,
				interpreter.EmptyLocationRange,
				arrayType,
				common.ZeroAddress,
				interpreter.NewUnmeteredIntValueFromInt64(100),
				interpreter.NewUnmeteredIntValueFromInt64(467),
				interpreter.NewUnmeteredIntValueFromInt64(297),
				interpreter.NewUnmeteredIntValueFromInt64(23),
			))

		runValidCase(t, "reversesa"+suffix, "originalsa"+suffix,
			interpreter.NewArrayValue(
				inter,
				interpreter.EmptyLocationRange,
				&interpreter.VariableSizedStaticType{
					Type: interpreter.PrimitiveStaticTypeInt,
				},
				common.ZeroAddress,
				interpreter.NewUnmeteredIntValueFromInt64(3),
				interpreter.NewUnmeteredIntValueFromInt64(2),
				interpreter.NewUnmeteredIntValueFromInt64(1),
			), interpreter.NewArrayValue(
				inter,
				interpreter.EmptyLocationRange,
				&interpreter.VariableSizedStaticType{
					Type: interpreter.PrimitiveStaticTypeInt,
				},
				common.ZeroAddress,
				interpreter.NewUnmeteredIntValueFromInt64(1),
				interpreter.NewUnmeteredIntValueFromInt64(2),
				interpreter.NewUnmeteredIntValueFromInt64(3),
			))
	}
}

func TestInterpretArrayFilter(t *testing.T) {

	runValidCase := func(
		t *testing.T,
		inter *interpreter.Interpreter,
		filterFuncName,
		originalFuncName string,
		filteredArray, originalArray *interpreter.ArrayValue,
	) {
		val, err := inter.Invoke(filterFuncName)
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			filteredArray,
			val,
		)

		origVal, err := inter.Invoke(originalFuncName)
		require.NoError(t, err)

		// Original array remains unchanged
		AssertValuesEqual(
			t,
			inter,
			originalArray,
			origVal,
		)
	}

	t.Run("with variable sized empty array", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
			let emptyVals: [Int] = []

			let onlyEven =
				view fun (_ x: Int): Bool {
					return x % 2 == 0
				}

			fun filterempty(): [Int] {
				return emptyVals.filter(onlyEven)
			}
			fun originalempty(): [Int] {
				return emptyVals
			}
		`)

		emptyVarSizedArray := interpreter.NewArrayValue(
			inter,
			interpreter.EmptyLocationRange,
			&interpreter.VariableSizedStaticType{
				Type: interpreter.PrimitiveStaticTypeInt,
			},
			common.ZeroAddress,
		)

		runValidCase(
			t,
			inter,
			"filterempty",
			"originalempty",
			emptyVarSizedArray,
			emptyVarSizedArray,
		)
	})

	t.Run("with variable sized array of integer", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
			let xs = [1, 2, 3, 100, 201]

			let onlyEven =
				view fun (_ x: Int): Bool {
					return x % 2 == 0
				}

			fun filterxs(): [Int] {
				return xs.filter(onlyEven)
			}
			fun originalxs(): [Int] {
				return xs
			}
		`)

		varSizedArrayType := &interpreter.VariableSizedStaticType{
			Type: interpreter.PrimitiveStaticTypeInt,
		}

		runValidCase(
			t,
			inter,
			"filterxs",
			"originalxs",
			interpreter.NewArrayValue(
				inter,
				interpreter.EmptyLocationRange,
				varSizedArrayType,
				common.ZeroAddress,
				interpreter.NewUnmeteredIntValueFromInt64(2),
				interpreter.NewUnmeteredIntValueFromInt64(100),
			),
			interpreter.NewArrayValue(
				inter,
				interpreter.EmptyLocationRange,
				varSizedArrayType,
				common.ZeroAddress,
				interpreter.NewUnmeteredIntValueFromInt64(1),
				interpreter.NewUnmeteredIntValueFromInt64(2),
				interpreter.NewUnmeteredIntValueFromInt64(3),
				interpreter.NewUnmeteredIntValueFromInt64(100),
				interpreter.NewUnmeteredIntValueFromInt64(201),
			),
		)
	})

	t.Run("with variable sized array of struct", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            struct TestStruct {

                var test: Int

				init(_ t: Int) {
					self.test = t
				}
			}

			let onlyOddStruct =
				view fun (_ x: TestStruct): Bool {
					return x.test % 2 == 1
				}

			let sa = [TestStruct(1), TestStruct(2), TestStruct(3)]

			fun filtersa(): [Int] {
				let sa_filtered = sa.filter(onlyOddStruct)
				let res: [Int] = [];
				for s in sa_filtered {
					res.append(s.test)
				}
				return res
			}

			fun originalsa(): [Int] {
				let res: [Int] = [];
				for s in sa {
					res.append(s.test)
				}
				return res
			}
		`)

		varSizedArrayType := &interpreter.VariableSizedStaticType{
			Type: interpreter.PrimitiveStaticTypeInt,
		}

		runValidCase(
			t,
			inter,
			"filtersa",
			"originalsa",
			interpreter.NewArrayValue(
				inter,
				interpreter.EmptyLocationRange,
				varSizedArrayType,
				common.ZeroAddress,
				interpreter.NewUnmeteredIntValueFromInt64(1),
				interpreter.NewUnmeteredIntValueFromInt64(3),
			),
			interpreter.NewArrayValue(
				inter,
				interpreter.EmptyLocationRange,
				varSizedArrayType,
				common.ZeroAddress,
				interpreter.NewUnmeteredIntValueFromInt64(1),
				interpreter.NewUnmeteredIntValueFromInt64(2),
				interpreter.NewUnmeteredIntValueFromInt64(3),
			),
		)
	})

	t.Run("with fixed sized empty array", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
			let emptyVals_fixed: [Int; 0] = []

			let onlyEven =
				view fun (_ x: Int): Bool {
					return x % 2 == 0
				}

			fun filterempty_fixed(): [Int] {
				return emptyVals_fixed.filter(onlyEven)
			}
			fun originalempty_fixed(): [Int; 0] {
				return emptyVals_fixed
			}
		`)

		runValidCase(
			t,
			inter,
			"filterempty_fixed",
			"originalempty_fixed",
			interpreter.NewArrayValue(
				inter,
				interpreter.EmptyLocationRange,
				&interpreter.VariableSizedStaticType{
					Type: interpreter.PrimitiveStaticTypeInt,
				},
				common.ZeroAddress,
			),
			interpreter.NewArrayValue(
				inter,
				interpreter.EmptyLocationRange,
				&interpreter.ConstantSizedStaticType{
					Type: interpreter.PrimitiveStaticTypeInt,
					Size: 0,
				},
				common.ZeroAddress,
			),
		)
	})

	t.Run("with fixed sized array of integer", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
			let xs_fixed: [Int; 5] = [1, 2, 3, 100, 201]

			let onlyEven =
				view fun (_ x: Int): Bool {
					return x % 2 == 0
				}

			fun filterxs_fixed(): [Int] {
				return xs_fixed.filter(onlyEven)
			}
			fun originalxs_fixed(): [Int; 5] {
				return xs_fixed
			}
		`)

		runValidCase(
			t,
			inter,
			"filterxs_fixed",
			"originalxs_fixed",
			interpreter.NewArrayValue(
				inter,
				interpreter.EmptyLocationRange,
				&interpreter.VariableSizedStaticType{
					Type: interpreter.PrimitiveStaticTypeInt,
				},
				common.ZeroAddress,
				interpreter.NewUnmeteredIntValueFromInt64(2),
				interpreter.NewUnmeteredIntValueFromInt64(100),
			),
			interpreter.NewArrayValue(
				inter,
				interpreter.EmptyLocationRange,
				&interpreter.ConstantSizedStaticType{
					Type: interpreter.PrimitiveStaticTypeInt,
					Size: 5,
				},
				common.ZeroAddress,
				interpreter.NewUnmeteredIntValueFromInt64(1),
				interpreter.NewUnmeteredIntValueFromInt64(2),
				interpreter.NewUnmeteredIntValueFromInt64(3),
				interpreter.NewUnmeteredIntValueFromInt64(100),
				interpreter.NewUnmeteredIntValueFromInt64(201),
			),
		)
	})

	t.Run("with fixed sized array of struct", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
			struct TestStruct {

				var test: Int

				init(_ t: Int) {
					self.test = t
				}
			}

			let onlyOddStruct =
				view fun (_ x: TestStruct): Bool {
					return x.test % 2 == 1
				}

			let sa_fixed: [TestStruct; 3] = [TestStruct(1), TestStruct(2), TestStruct(3)]

			fun filtersa_fixed(): [Int] {
				let sa_rev = sa_fixed.filter(onlyOddStruct)
				let res: [Int] = [];
				for s in sa_rev {
					res.append(s.test)
				}
				return res
			}
			fun originalsa_fixed(): [Int] {
				let res: [Int] = [];
				for s in sa_fixed {
					res.append(s.test)
				}
				return res
			}
		`)

		runValidCase(
			t,
			inter,
			"filtersa_fixed",
			"originalsa_fixed",
			interpreter.NewArrayValue(
				inter,
				interpreter.EmptyLocationRange,
				&interpreter.VariableSizedStaticType{
					Type: interpreter.PrimitiveStaticTypeInt,
				},
				common.ZeroAddress,
				interpreter.NewUnmeteredIntValueFromInt64(1),
				interpreter.NewUnmeteredIntValueFromInt64(3),
			),
			interpreter.NewArrayValue(
				inter,
				interpreter.EmptyLocationRange,
				&interpreter.VariableSizedStaticType{
					Type: interpreter.PrimitiveStaticTypeInt,
				},
				common.ZeroAddress,
				interpreter.NewUnmeteredIntValueFromInt64(1),
				interpreter.NewUnmeteredIntValueFromInt64(2),
				interpreter.NewUnmeteredIntValueFromInt64(3),
			),
		)
	})

	t.Run("box and convert argument", func(t *testing.T) {
		t.Parallel()

		inter, err := parseCheckAndInterpretWithOptions(t, `
              struct S {
                  fun map(f: fun(AnyStruct): String): Bool {
                      return true
                  }
              }

              fun test(): [S] {
                  let ss = [S()]
                  // NOTE: The filter has a parameter of type S? instead of just S
                  return ss.filter(view fun(s2: S?): Bool {
                      // The map should call Optional.map, not S.map,
                      // because s2 is S?, not S
                      return s2.map(fun(s3: AnyStruct): Bool {
                          return false
                      })!
                  })
              }
            `,
			ParseCheckAndInterpretOptions{
				HandleCheckerError: func(err error) {
					errs := RequireCheckerErrors(t, err, 1)
					require.IsType(t, &sema.PurityError{}, errs[0])
				},
			},
		)
		require.NoError(t, err)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		require.IsType(t, &interpreter.ArrayValue{}, value)
		array := value.(*interpreter.ArrayValue)
		require.Equal(t, 0, array.Count())
	})
}

func TestInterpretArrayMap(t *testing.T) {
	t.Parallel()

	runValidCase := func(
		t *testing.T,
		inter *interpreter.Interpreter,
		mapFuncName,
		originalFuncName string,
		mappedArray, originalArray *interpreter.ArrayValue,
	) {
		val, err := inter.Invoke(mapFuncName)
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			mappedArray,
			val,
		)

		origVal, err := inter.Invoke(originalFuncName)
		require.NoError(t, err)

		// Original array remains unchanged
		AssertValuesEqual(
			t,
			inter,
			originalArray,
			origVal,
		)
	}

	t.Run("with variable sized empty array", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
			let emptyVals: [Int] = []

			let plusTen =
				fun (_ x: Int): Int {
					return x + 10
				}

			fun mapempty(): [Int] {
				return emptyVals.map(plusTen)
			}
			fun originalempty(): [Int] {
				return emptyVals
			}
		`)

		emptyVarSizedArray := interpreter.NewArrayValue(
			inter,
			interpreter.EmptyLocationRange,
			&interpreter.VariableSizedStaticType{
				Type: interpreter.PrimitiveStaticTypeInt,
			},
			common.ZeroAddress,
		)

		runValidCase(
			t,
			inter,
			"mapempty",
			"originalempty",
			emptyVarSizedArray,
			emptyVarSizedArray,
		)
	})

	t.Run("with variable sized array of integer to Int16", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
			let xs = [1, 2, 3, 100, 201]

			let plusTen =
				fun (_ x: Int): Int16 {
					return Int16(x) + 10
				}

			fun mapxs(): [Int16] {
				return xs.map(plusTen)
			}
			fun originalxs(): [Int] {
				return xs
			}
		`)

		runValidCase(
			t,
			inter,
			"mapxs",
			"originalxs",
			interpreter.NewArrayValue(
				inter,
				interpreter.EmptyLocationRange,
				&interpreter.VariableSizedStaticType{
					Type: interpreter.PrimitiveStaticTypeInt16,
				},
				common.ZeroAddress,
				interpreter.NewUnmeteredInt16Value(11),
				interpreter.NewUnmeteredInt16Value(12),
				interpreter.NewUnmeteredInt16Value(13),
				interpreter.NewUnmeteredInt16Value(110),
				interpreter.NewUnmeteredInt16Value(211),
			),
			interpreter.NewArrayValue(
				inter,
				interpreter.EmptyLocationRange,
				&interpreter.VariableSizedStaticType{
					Type: interpreter.PrimitiveStaticTypeInt,
				},
				common.ZeroAddress,
				interpreter.NewUnmeteredIntValueFromInt64(1),
				interpreter.NewUnmeteredIntValueFromInt64(2),
				interpreter.NewUnmeteredIntValueFromInt64(3),
				interpreter.NewUnmeteredIntValueFromInt64(100),
				interpreter.NewUnmeteredIntValueFromInt64(201),
			),
		)
	})

	t.Run("with variable sized array of struct to Int", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
			struct TestStruct {
				var test: Int

				init(_ t: Int) {
					self.test = t
				}
			}

			let innerValueMinusOne =
				fun (_ x: TestStruct): Int {
					return x.test - 1
				}

			let sa = [TestStruct(1), TestStruct(2), TestStruct(3)]

			fun mapsa(): [Int] {
				return sa.map(innerValueMinusOne)
			}

			fun originalsa(): [Int] {
				let res: [Int] = [];
				for s in sa {
					res.append(s.test)
				}
				return res
			}
		`)

		varSizedArrayType := &interpreter.VariableSizedStaticType{
			Type: interpreter.PrimitiveStaticTypeInt,
		}

		runValidCase(
			t,
			inter,
			"mapsa",
			"originalsa",
			interpreter.NewArrayValue(
				inter,
				interpreter.EmptyLocationRange,
				varSizedArrayType,
				common.ZeroAddress,
				interpreter.NewUnmeteredIntValueFromInt64(0),
				interpreter.NewUnmeteredIntValueFromInt64(1),
				interpreter.NewUnmeteredIntValueFromInt64(2),
			),
			interpreter.NewArrayValue(
				inter,
				interpreter.EmptyLocationRange,
				varSizedArrayType,
				common.ZeroAddress,
				interpreter.NewUnmeteredIntValueFromInt64(1),
				interpreter.NewUnmeteredIntValueFromInt64(2),
				interpreter.NewUnmeteredIntValueFromInt64(3),
			),
		)
	})

	t.Run("with variable sized array of int to struct", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
			struct TestStruct {
				var test: Int

				init(_ t: Int) {
					self.test = t
				}
			}

			let intPlusTenToStruct =
				fun (_ x: Int): TestStruct {
					return TestStruct(x + 10)
				}

			let orig = [1, 2, 3]

			fun mapToStruct(): [Int] {
				let mapped = orig.map(intPlusTenToStruct)
				let res: [Int] = [];
				for s in mapped {
					res.append(s.test)
				}
				return res
			}
			fun original(): [Int] {
				return orig
			}
		`)

		varSizedArrayType := &interpreter.VariableSizedStaticType{
			Type: interpreter.PrimitiveStaticTypeInt,
		}

		runValidCase(
			t,
			inter,
			"mapToStruct",
			"original",
			interpreter.NewArrayValue(
				inter,
				interpreter.EmptyLocationRange,
				varSizedArrayType,
				common.ZeroAddress,
				interpreter.NewUnmeteredIntValueFromInt64(11),
				interpreter.NewUnmeteredIntValueFromInt64(12),
				interpreter.NewUnmeteredIntValueFromInt64(13),
			),
			interpreter.NewArrayValue(
				inter,
				interpreter.EmptyLocationRange,
				varSizedArrayType,
				common.ZeroAddress,
				interpreter.NewUnmeteredIntValueFromInt64(1),
				interpreter.NewUnmeteredIntValueFromInt64(2),
				interpreter.NewUnmeteredIntValueFromInt64(3),
			),
		)
	})

	t.Run("with fixed sized empty array", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
			let emptyVals_fixed: [Int; 0] = []

			let trueForEven =
				fun (_ x: Int): Bool {
					return x % 2 == 0
				}

			fun mapempty_fixed(): [Bool; 0] {
				return emptyVals_fixed.map(trueForEven)
			}
			fun originalempty_fixed(): [Int; 0] {
				return emptyVals_fixed
			}
		`)

		runValidCase(
			t,
			inter,
			"mapempty_fixed",
			"originalempty_fixed",
			interpreter.NewArrayValue(
				inter,
				interpreter.EmptyLocationRange,
				&interpreter.ConstantSizedStaticType{
					Type: interpreter.PrimitiveStaticTypeBool,
					Size: 0,
				},
				common.ZeroAddress,
			),
			interpreter.NewArrayValue(
				inter,
				interpreter.EmptyLocationRange,
				&interpreter.ConstantSizedStaticType{
					Type: interpreter.PrimitiveStaticTypeInt,
					Size: 0,
				},
				common.ZeroAddress,
			),
		)
	})

	t.Run("with fixed sized array of integer to Int16", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
			let xs_fixed: [Int; 5] = [1, 2, 3, 100, 201]

			let plusTen =
				fun (_ x: Int): Int16 {
					return Int16(x) + 10
				}

			fun mapxs_fixed(): [Int16; 5] {
				return xs_fixed.map(plusTen)
			}
			fun originalxs_fixed(): [Int; 5] {
				return xs_fixed
			}
		`)

		runValidCase(
			t,
			inter,
			"mapxs_fixed",
			"originalxs_fixed",
			interpreter.NewArrayValue(
				inter,
				interpreter.EmptyLocationRange,
				&interpreter.ConstantSizedStaticType{
					Type: interpreter.PrimitiveStaticTypeInt16,
					Size: 5,
				},
				common.ZeroAddress,
				interpreter.NewUnmeteredInt16Value(11),
				interpreter.NewUnmeteredInt16Value(12),
				interpreter.NewUnmeteredInt16Value(13),
				interpreter.NewUnmeteredInt16Value(110),
				interpreter.NewUnmeteredInt16Value(211),
			),
			interpreter.NewArrayValue(
				inter,
				interpreter.EmptyLocationRange,
				&interpreter.ConstantSizedStaticType{
					Type: interpreter.PrimitiveStaticTypeInt,
					Size: 5,
				},
				common.ZeroAddress,
				interpreter.NewUnmeteredIntValueFromInt64(1),
				interpreter.NewUnmeteredIntValueFromInt64(2),
				interpreter.NewUnmeteredIntValueFromInt64(3),
				interpreter.NewUnmeteredIntValueFromInt64(100),
				interpreter.NewUnmeteredIntValueFromInt64(201),
			),
		)
	})

	t.Run("with fixed sized array of struct to Int", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
			struct TestStruct {
				var test: Int

				init(_ t: Int) {
					self.test = t
				}
			}

			let innerValueMinusOne =
				fun (_ x: TestStruct): Int {
					return x.test - 1
				}

			let sa_fixed: [TestStruct; 3] = [TestStruct(1), TestStruct(2), TestStruct(3)]

			fun mapsa_fixed(): [Int; 3] {
				return sa_fixed.map(innerValueMinusOne)
			}

			fun originalsa_fixed(): [Int] {
				let res: [Int] = [];
				for s in sa_fixed {
					res.append(s.test)
				}
				return res
			}
		`)

		runValidCase(
			t,
			inter,
			"mapsa_fixed",
			"originalsa_fixed",
			interpreter.NewArrayValue(
				inter,
				interpreter.EmptyLocationRange,
				&interpreter.ConstantSizedStaticType{
					Type: interpreter.PrimitiveStaticTypeInt,
					Size: 3,
				},
				common.ZeroAddress,
				interpreter.NewUnmeteredIntValueFromInt64(0),
				interpreter.NewUnmeteredIntValueFromInt64(1),
				interpreter.NewUnmeteredIntValueFromInt64(2),
			),
			interpreter.NewArrayValue(
				inter,
				interpreter.EmptyLocationRange,
				&interpreter.VariableSizedStaticType{
					Type: interpreter.PrimitiveStaticTypeInt,
				},
				common.ZeroAddress,
				interpreter.NewUnmeteredIntValueFromInt64(1),
				interpreter.NewUnmeteredIntValueFromInt64(2),
				interpreter.NewUnmeteredIntValueFromInt64(3),
			),
		)
	})

	t.Run("with fixed sized array of Int to Struct", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
			struct TestStruct {
				var test: Int

				init(_ t: Int) {
					self.test = t
				}
			}

			let intPlusTenToStruct =
				fun (_ x: Int): TestStruct {
					return TestStruct(x + 10)
				}

			let array_fixed: [Int; 3] = [1, 2, 3]

			fun map_fixed(): [Int] {
				let sa = array_fixed.map(intPlusTenToStruct)
				let res: [Int] = [];
				for s in sa {
					res.append(s.test)
				}
				return res
			}
			fun original_fixed(): [Int; 3] {
				return array_fixed
			}
		`)

		runValidCase(
			t,
			inter,
			"map_fixed",
			"original_fixed",
			interpreter.NewArrayValue(
				inter,
				interpreter.EmptyLocationRange,
				&interpreter.VariableSizedStaticType{
					Type: interpreter.PrimitiveStaticTypeInt,
				},
				common.ZeroAddress,
				interpreter.NewUnmeteredIntValueFromInt64(11),
				interpreter.NewUnmeteredIntValueFromInt64(12),
				interpreter.NewUnmeteredIntValueFromInt64(13),
			),
			interpreter.NewArrayValue(
				inter,
				interpreter.EmptyLocationRange,
				&interpreter.ConstantSizedStaticType{
					Type: interpreter.PrimitiveStaticTypeInt,
					Size: 3,
				},
				common.ZeroAddress,
				interpreter.NewUnmeteredIntValueFromInt64(1),
				interpreter.NewUnmeteredIntValueFromInt64(2),
				interpreter.NewUnmeteredIntValueFromInt64(3),
			),
		)
	})

	t.Run("box and convert argument", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          struct S {
              fun map(f: fun(AnyStruct): String): String {
                  return "S.map"
              }
          }

          fun test(): [String?] {
              let ss = [S()]
              // NOTE: The outer map has a parameter of type S? instead of just S
              return ss.map(fun(s2: S?): String? {
                  // The inner map should call Optional.map, not S.map,
                  // because s2 is S?, not S
                  return s2.map(fun(s3: AnyStruct): String {
                      return "Optional.map"
                  })
              })
          }
        `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(t,
			inter,
			interpreter.NewArrayValue(
				inter,
				interpreter.EmptyLocationRange,
				interpreter.NewVariableSizedStaticType(
					nil,
					interpreter.NewOptionalStaticType(
						nil,
						interpreter.PrimitiveStaticTypeString,
					),
				),
				common.ZeroAddress,
				interpreter.NewSomeValueNonCopying(
					nil,
					interpreter.NewUnmeteredStringValue("Optional.map"),
				),
			),
			value,
		)
	})
}

func TestInterpretArrayToVariableSized(t *testing.T) {
	t.Parallel()

	runValidCase := func(
		t *testing.T,
		inter *interpreter.Interpreter,
		expectedArray *interpreter.ArrayValue,
	) {
		val, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			expectedArray,
			val,
		)
	}

	t.Run("with empty array", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
			let emptyVals_fixed: [Int; 0] = []

			fun test(): [Int] {
				return emptyVals_fixed.toVariableSized()
			}
		`)

		runValidCase(
			t,
			inter,
			interpreter.NewArrayValue(
				inter,
				interpreter.EmptyLocationRange,
				&interpreter.VariableSizedStaticType{
					Type: interpreter.PrimitiveStaticTypeInt,
				},
				common.ZeroAddress,
			),
		)
	})

	t.Run("with integer array", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
			let xs_fixed: [Int; 5] = [1, 2, 3, 100, 201]

			fun test(): [Int] {
				return xs_fixed.toVariableSized()
			}
		`)

		runValidCase(
			t,
			inter,
			interpreter.NewArrayValue(
				inter,
				interpreter.EmptyLocationRange,
				&interpreter.VariableSizedStaticType{
					Type: interpreter.PrimitiveStaticTypeInt,
				},
				common.ZeroAddress,
				interpreter.NewUnmeteredIntValueFromInt64(1),
				interpreter.NewUnmeteredIntValueFromInt64(2),
				interpreter.NewUnmeteredIntValueFromInt64(3),
				interpreter.NewUnmeteredIntValueFromInt64(100),
				interpreter.NewUnmeteredIntValueFromInt64(201),
			),
		)
	})

	t.Run("with string array", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
			let xs_fixed: [String; 2] = ["abc", "def"]

			fun test(): [String] {
				return xs_fixed.toVariableSized()
			}
		`)

		runValidCase(
			t,
			inter,
			interpreter.NewArrayValue(
				inter,
				interpreter.EmptyLocationRange,
				&interpreter.VariableSizedStaticType{
					Type: interpreter.PrimitiveStaticTypeString,
				},
				common.ZeroAddress,
				interpreter.NewUnmeteredStringValue("abc"),
				interpreter.NewUnmeteredStringValue("def"),
			),
		)
	})

	t.Run("with array of struct", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
			struct TestStruct {
				var test: Int

				init(_ t: Int) {
					self.test = t
				}
			}

			let sa_fixed: [TestStruct; 3] = [TestStruct(1), TestStruct(2), TestStruct(3)]

			fun test(): [TestStruct] {
				return sa_fixed.toVariableSized()
			}
		`)

		location := common.Location(common.StringLocation("test"))
		value1 := interpreter.NewCompositeValue(
			inter,
			interpreter.EmptyLocationRange,
			location,
			"TestStruct",
			common.CompositeKindStructure,
			[]interpreter.CompositeField{
				{
					Name:  "test",
					Value: interpreter.NewUnmeteredIntValueFromInt64(1),
				},
			},
			common.ZeroAddress,
		)
		value2 := interpreter.NewCompositeValue(
			inter,
			interpreter.EmptyLocationRange,
			location,
			"TestStruct",
			common.CompositeKindStructure,
			[]interpreter.CompositeField{
				{
					Name:  "test",
					Value: interpreter.NewUnmeteredIntValueFromInt64(2),
				},
			},
			common.ZeroAddress,
		)
		value3 := interpreter.NewCompositeValue(
			inter,
			interpreter.EmptyLocationRange,
			location,
			"TestStruct",
			common.CompositeKindStructure,
			[]interpreter.CompositeField{
				{
					Name:  "test",
					Value: interpreter.NewUnmeteredIntValueFromInt64(3),
				},
			},
			common.ZeroAddress,
		)

		runValidCase(
			t,
			inter,
			interpreter.NewArrayValue(
				inter,
				interpreter.EmptyLocationRange,
				&interpreter.VariableSizedStaticType{
					Type: interpreter.NewCompositeStaticType(
						nil,
						common.Location(common.StringLocation("test")),
						"TestStruct",
						"S.test.TestStruct",
					),
				},
				common.ZeroAddress,
				value1,
				value2,
				value3,
			),
		)
	})
}

func TestInterpretArrayToConstantSized(t *testing.T) {
	t.Parallel()

	runValidCase := func(
		t *testing.T,
		inter *interpreter.Interpreter,
		expectedArray interpreter.Value,
	) {
		val, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			expectedArray,
			val,
		)
	}

	t.Run("with empty array", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
			let emptyVals: [Int] = []

			fun test(): [Int;0] {
				let constArray = emptyVals.toConstantSized<[Int; 0]>()
				return constArray!
			}
		`)

		runValidCase(
			t,
			inter,
			interpreter.NewArrayValue(
				inter,
				interpreter.EmptyLocationRange,
				&interpreter.ConstantSizedStaticType{
					Type: interpreter.PrimitiveStaticTypeInt,
					Size: 0,
				},
				common.ZeroAddress,
			),
		)
	})

	t.Run("with integer array", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
			let xs: [Int] = [1, 2, 3, 100, 201]

			fun test(): [Int; 5]? {
				return xs.toConstantSized<[Int; 5]>()
			}
		`)

		runValidCase(
			t,
			inter,
			interpreter.NewSomeValueNonCopying(
				inter,
				interpreter.NewArrayValue(
					inter,
					interpreter.EmptyLocationRange,
					&interpreter.ConstantSizedStaticType{
						Type: interpreter.PrimitiveStaticTypeInt,
						Size: 5,
					},
					common.ZeroAddress,
					interpreter.NewUnmeteredIntValueFromInt64(1),
					interpreter.NewUnmeteredIntValueFromInt64(2),
					interpreter.NewUnmeteredIntValueFromInt64(3),
					interpreter.NewUnmeteredIntValueFromInt64(100),
					interpreter.NewUnmeteredIntValueFromInt64(201),
				),
			),
		)
	})

	t.Run("with string array", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
			let xs: [String] = ["abc", "def"]

			fun test(): [String; 2]? {
				return xs.toConstantSized<[String; 2]>()
			}
		`)

		runValidCase(
			t,
			inter,
			interpreter.NewSomeValueNonCopying(
				inter,
				interpreter.NewArrayValue(
					inter,
					interpreter.EmptyLocationRange,
					&interpreter.ConstantSizedStaticType{
						Type: interpreter.PrimitiveStaticTypeString,
						Size: 2,
					},
					common.ZeroAddress,
					interpreter.NewUnmeteredStringValue("abc"),
					interpreter.NewUnmeteredStringValue("def"),
				),
			),
		)
	})

	t.Run("with wrong size", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
			let xs: [Int] = [1, 2, 3, 100, 201]

			fun test(): [Int; 4]? {
				return xs.toConstantSized<[Int; 4]>()
			}
		`)

		runValidCase(
			t,
			inter,
			interpreter.NilOptionalValue,
		)
	})

	t.Run("with array of struct", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
			struct TestStruct {
				var test: Int

				init(_ t: Int) {
					self.test = t
				}
			}

			let sa: [TestStruct] = [TestStruct(1), TestStruct(2), TestStruct(3)]

			fun test(): [TestStruct;3]? {
				return sa.toConstantSized<[TestStruct;3]>()
			}
		`)

		location := common.Location(common.StringLocation("test"))
		value1 := interpreter.NewCompositeValue(
			inter,
			interpreter.EmptyLocationRange,
			location,
			"TestStruct",
			common.CompositeKindStructure,
			[]interpreter.CompositeField{
				{
					Name:  "test",
					Value: interpreter.NewUnmeteredIntValueFromInt64(1),
				},
			},
			common.ZeroAddress,
		)
		value2 := interpreter.NewCompositeValue(
			inter,
			interpreter.EmptyLocationRange,
			location,
			"TestStruct",
			common.CompositeKindStructure,
			[]interpreter.CompositeField{
				{
					Name:  "test",
					Value: interpreter.NewUnmeteredIntValueFromInt64(2),
				},
			},
			common.ZeroAddress,
		)
		value3 := interpreter.NewCompositeValue(
			inter,
			interpreter.EmptyLocationRange,
			location,
			"TestStruct",
			common.CompositeKindStructure,
			[]interpreter.CompositeField{
				{
					Name:  "test",
					Value: interpreter.NewUnmeteredIntValueFromInt64(3),
				},
			},
			common.ZeroAddress,
		)

		runValidCase(
			t,
			inter,
			interpreter.NewSomeValueNonCopying(
				inter,
				interpreter.NewArrayValue(
					inter,
					interpreter.EmptyLocationRange,
					&interpreter.ConstantSizedStaticType{
						Type: interpreter.NewCompositeStaticType(
							nil,
							common.Location(common.StringLocation("test")),
							"TestStruct",
							"S.test.TestStruct",
						),
						Size: 3,
					},
					common.ZeroAddress,
					value1,
					value2,
					value3,
				),
			),
		)
	})

	t.Run("ensure result is optional", func(t *testing.T) {
		t.Parallel()

		baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
		baseValueActivation.DeclareValue(stdlib.PanicFunction)

		baseActivation := activations.NewActivation(nil, interpreter.BaseActivation)
		interpreter.Declare(baseActivation, stdlib.PanicFunction)

		inter, err := parseCheckAndInterpretWithOptions(t,
			`
               fun test(): [UInt8; 20] {
                    return "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
                        .decodeHex()
                        .toConstantSized<[UInt8; 20]>()
                        ?? panic("toConstantSized failed")
               }
            `,
			ParseCheckAndInterpretOptions{
				CheckerConfig: &sema.Config{
					BaseValueActivationHandler: func(_ common.Location) *sema.VariableActivation {
						return baseValueActivation
					},
				},
				Config: &interpreter.Config{
					BaseActivationHandler: func(_ common.Location) *interpreter.VariableActivation {
						return baseActivation
					},
				},
			},
		)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		require.NoError(t, err)
	})
}

func TestInterpretCastingBoxing(t *testing.T) {

	t.Parallel()

	t.Run("failable cast", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          let a = (1 as? Int?!)?.getType()
        `)

		variable := inter.Globals.Get("a")
		require.NotNil(t, variable)

		require.Equal(
			t,
			interpreter.NewUnmeteredSomeValueNonCopying(
				interpreter.TypeValue{
					Type: interpreter.PrimitiveStaticTypeInt,
				},
			),
			variable.GetValue(inter),
		)
	})

	t.Run("force cast", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          let a = (1 as! Int?)?.getType()
        `)

		variable := inter.Globals.Get("a")
		require.NotNil(t, variable)

		require.Equal(
			t,
			interpreter.NewUnmeteredSomeValueNonCopying(
				interpreter.TypeValue{
					Type: interpreter.PrimitiveStaticTypeInt,
				},
			),
			variable.GetValue(inter),
		)
	})

	t.Run("cast", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          let a = (1 as Int?)?.getType()
        `)

		variable := inter.Globals.Get("a")
		require.NotNil(t, variable)

		require.Equal(
			t,
			interpreter.NewUnmeteredSomeValueNonCopying(
				interpreter.TypeValue{
					Type: interpreter.PrimitiveStaticTypeInt,
				},
			),
			variable.GetValue(inter),
		)
	})
}

func TestInterpretNilCoalesceReference(t *testing.T) {

	t.Parallel()

	baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
	baseValueActivation.DeclareValue(stdlib.PanicFunction)

	baseActivation := activations.NewActivation(nil, interpreter.BaseActivation)
	interpreter.Declare(baseActivation, stdlib.PanicFunction)

	inter, err := parseCheckAndInterpretWithOptions(t,
		`
          let xs = {"a": 2}
          let ref = &xs["a"] as &Int? ?? panic("no a")
        `,
		ParseCheckAndInterpretOptions{
			CheckerConfig: &sema.Config{
				BaseValueActivationHandler: func(_ common.Location) *sema.VariableActivation {
					return baseValueActivation
				},
			},
			Config: &interpreter.Config{
				BaseActivationHandler: func(_ common.Location) *interpreter.VariableActivation {
					return baseActivation
				},
			},
		},
	)
	require.NoError(t, err)

	variable := inter.Globals.Get("ref")
	require.NotNil(t, variable)

	require.Equal(
		t,
		&interpreter.EphemeralReferenceValue{
			Value:         interpreter.NewUnmeteredIntValueFromInt64(2),
			BorrowedType:  sema.IntType,
			Authorization: interpreter.UnauthorizedAccess,
		},
		variable.GetValue(inter),
	)
}

func TestInterpretNilCoalesceAnyResourceAndPanic(t *testing.T) {

	t.Parallel()

	baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
	baseValueActivation.DeclareValue(stdlib.PanicFunction)

	baseActivation := activations.NewActivation(nil, interpreter.BaseActivation)
	interpreter.Declare(baseActivation, stdlib.PanicFunction)

	_, err := parseCheckAndInterpretWithOptions(t,
		`
          resource R {}

          fun f(): @AnyResource? {
              return <-create R()
          }

          let y <- f() ?? panic("no R")
        `,
		ParseCheckAndInterpretOptions{
			CheckerConfig: &sema.Config{
				BaseValueActivationHandler: func(_ common.Location) *sema.VariableActivation {
					return baseValueActivation
				},
			},
			Config: &interpreter.Config{
				BaseActivationHandler: func(_ common.Location) *interpreter.VariableActivation {
					return baseActivation
				},
			},
		},
	)
	require.NoError(t, err)
}

func TestInterpretDictionaryDuplicateKey(t *testing.T) {

	t.Parallel()

	t.Run("struct", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `

          struct S {}

          fun test() {
              let s1 = S()
              let s2 = S()
              {"a": s1, "a": s2}
          }
        `)

		_, err := inter.Invoke("test")
		require.NoError(t, err)
	})

	t.Run("resource in literal", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `

          resource R {}

          fun test() {
              let r1 <- create R()
              let r2 <- create R()
              let rs <- {"a": <-r1, "a": <-r2}
              destroy rs
          }
        `)

		_, err := inter.Invoke("test")
		RequireError(t, err)

		require.ErrorAs(t, err, &interpreter.DuplicateKeyInResourceDictionaryError{})
	})

	t.Run("resource", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `

          resource R {}

          fun test() {
              let r1 <- create R()
              let r2 <- create R()
              let rs: @{String: R?} <- {}
              rs["a"] <-! r1
              rs["a"] <-! r2

              destroy rs
          }
        `)

		_, err := inter.Invoke("test")
		RequireError(t, err)
		require.ErrorAs(t, err, &interpreter.ResourceLossError{})
	})
}

func TestInterpretReferenceUpAndDowncast(t *testing.T) {

	t.Parallel()

	type testCase struct {
		name     string
		typeName string
		code     string
	}

	testFunctionReturn := func(tc testCase) {

		t.Run(fmt.Sprintf("function return: %s", tc.name), func(t *testing.T) {

			t.Parallel()

			inter, _ := testAccount(t, interpreter.NewUnmeteredAddressValueFromBytes([]byte{0x1}), true, nil, fmt.Sprintf(
				`
                      #allowAccountLinking

                      struct S {}

					  entitlement E

                      fun getRef(): &AnyStruct  {
                         %[2]s
                         return ref
                      }

                      fun test(): &%[1]s {
                          let ref2 = getRef()
                          return (ref2 as AnyStruct) as! &%[1]s
                      }
                    `,
				tc.typeName,
				tc.code,
			), sema.Config{})

			_, err := inter.Invoke("test")
			require.NoError(t, err)
		})
	}

	testVariableDeclaration := func(tc testCase) {

		t.Run(fmt.Sprintf("variable declaration: %s", tc.name), func(t *testing.T) {

			t.Parallel()

			inter, _ := testAccount(t, interpreter.NewUnmeteredAddressValueFromBytes([]byte{0x1}), true, nil, fmt.Sprintf(
				`
                      #allowAccountLinking

                      struct S {}

					  entitlement E

                      fun test(): &%[1]s {
                          %[2]s
                          let ref2: &AnyStruct = ref
                          return (ref2 as AnyStruct) as! &%[1]s
                      }
                    `,
				tc.typeName,
				tc.code,
			), sema.Config{})

			_, err := inter.Invoke("test")
			require.NoError(t, err)
		})
	}

	testCases := []testCase{
		{
			name:     "account reference",
			typeName: "Account",
			code: `
		      let ref = account
		    `,
		},
	}

	for _, authorized := range []bool{true, false} {

		var authKeyword, testNameSuffix string
		if authorized {
			authKeyword = "auth(E)"
			testNameSuffix = ", auth"
		}

		testCases = append(testCases,
			testCase{
				name:     fmt.Sprintf("ephemeral reference%s", testNameSuffix),
				typeName: "S",
				code: fmt.Sprintf(`
                      var s = S()
                      let ref = &s as %s &S
                    `,
					authKeyword,
				),
			},
			testCase{
				name:     fmt.Sprintf("storage reference%s", testNameSuffix),
				typeName: "S",
				code: fmt.Sprintf(`
                      account.storage.save(S(), to: /storage/s)
                      let ref = account.storage.borrow<%s &S>(from: /storage/s)!
                    `,
					authKeyword,
				),
			},
		)
	}

	for _, tc := range testCases {
		testFunctionReturn(tc)
		testVariableDeclaration(tc)
	}
}

func TestInterpretCompositeTypeHandler(t *testing.T) {

	t.Parallel()

	testType := interpreter.NewCompositeStaticTypeComputeTypeID(nil, stdlib.FlowLocation{}, "AccountContractAdded")

	inter, err := parseCheckAndInterpretWithOptions(t,
		`
          fun test(): Type? {
              return CompositeType("flow.AccountContractAdded")
          }
        `,
		ParseCheckAndInterpretOptions{
			Config: &interpreter.Config{
				CompositeTypeHandler: func(location common.Location, typeID common.TypeID) *sema.CompositeType {
					if _, ok := location.(stdlib.FlowLocation); ok {
						return stdlib.FlowEventTypes[typeID]
					}

					return nil
				},
			},
		},
	)
	require.NoError(t, err)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	require.Equal(t,
		interpreter.NewUnmeteredSomeValueNonCopying(interpreter.NewUnmeteredTypeValue(testType)),
		value,
	)
}

func TestInterpretConditionsWrapperFunctionType(t *testing.T) {

	t.Parallel()

	t.Run("interface", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          struct interface SI {
              fun test(x: Int) {
                  pre { true }
              }
          }

          struct S: SI {
              fun test(x: Int) {}
          }

          fun test(): fun (Int): Void {
              let s = S()
              return s.test
          }
        `)

		_, err := inter.Invoke("test")
		require.NoError(t, err)
	})
}

func TestInterpretSwapInSameArray(t *testing.T) {

	t.Parallel()

	t.Run("resources, different indices", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          resource R {
              let value: Int

              init(value: Int) {
                  self.value = value
              }
          }

          fun test(): [Int] {
             let rs <- [
                 <- create R(value: 0),
                 <- create R(value: 1),
                 <- create R(value: 2)
             ]

             // We swap only '0' and '1'
             rs[0] <-> rs[1]

             let values = [
                 rs[0].value,
                 rs[1].value,
                 rs[2].value
             ]

             destroy rs

             return values
          }
        `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewArrayValue(
				inter,
				interpreter.EmptyLocationRange,
				&interpreter.VariableSizedStaticType{
					Type: interpreter.PrimitiveStaticTypeInt,
				},
				common.ZeroAddress,
				interpreter.NewUnmeteredIntValueFromInt64(1),
				interpreter.NewUnmeteredIntValueFromInt64(0),
				interpreter.NewUnmeteredIntValueFromInt64(2),
			),
			value,
		)
	})

	t.Run("resources, same indices", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          resource R {
              let value: Int

              init(value: Int) {
                  self.value = value
              }
          }

          fun test(): [Int] {
             let rs <- [
                 <- create R(value: 0),
                 <- create R(value: 1),
                 <- create R(value: 2)
             ]

             // We swap only '1'
             rs[1] <-> rs[1]

             let values = [
                 rs[0].value,
                 rs[1].value,
                 rs[2].value
             ]

             destroy rs

             return values
          }
        `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewArrayValue(
				inter,
				interpreter.EmptyLocationRange,
				&interpreter.VariableSizedStaticType{
					Type: interpreter.PrimitiveStaticTypeInt,
				},
				common.ZeroAddress,
				interpreter.NewUnmeteredIntValueFromInt64(0),
				interpreter.NewUnmeteredIntValueFromInt64(1),
				interpreter.NewUnmeteredIntValueFromInt64(2),
			),
			value,
		)
	})

	t.Run("structs", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          struct S {
              let value: Int

              init(value: Int) {
                  self.value = value
              }
          }

          fun test(): [Int] {
             let structs = [
                 S(value: 0),
                 S(value: 1),
                 S(value: 2)
             ]

             // We swap only '0' and '1'
             structs[0] <-> structs[1]

             return [
                 structs[0].value,
                 structs[1].value,
                 structs[2].value
             ]
          }
        `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewArrayValue(
				inter,
				interpreter.EmptyLocationRange,
				&interpreter.VariableSizedStaticType{
					Type: interpreter.PrimitiveStaticTypeInt,
				},
				common.ZeroAddress,
				interpreter.NewUnmeteredIntValueFromInt64(1),
				interpreter.NewUnmeteredIntValueFromInt64(0),
				interpreter.NewUnmeteredIntValueFromInt64(2),
			),
			value,
		)
	})
}

func TestInterpretSwapDictionaryKeysWithSideEffects(t *testing.T) {

	t.Parallel()

	t.Run("simple", func(t *testing.T) {
		t.Parallel()

		inter, getLogs, err := parseCheckAndInterpretWithLogs(t, `
          let xs: [{Int: String}] = [{2: "x"}, {3: "y"}]

          fun a(): Int {
              log("a")
              return 0
          }

          fun b(): Int {
              log("b")
              return 2
          }

          fun c(): Int {
              log("c")
              return 1
          }

          fun d(): Int {
              log("d")
              return 3
          }

          fun test() {
              log(xs)
              xs[a()][b()] <-> xs[c()][d()]
              log(xs)
          }
        `)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		require.NoError(t, err)

		assert.Equal(t,
			[]string{
				`[{2: "x"}, {3: "y"}]`,
				`"a"`,
				`"b"`,
				`"c"`,
				`"d"`,
				`[{2: "y"}, {3: "x"}]`,
			},
			getLogs(),
		)

	})

	t.Run("resources", func(t *testing.T) {
		t.Parallel()

		inter, getEvents, err := parseCheckAndInterpretWithEvents(t, `
          resource Resource {
			  event ResourceDestroyed(value: Int = self.value)
              var value: Int

              init(_ value: Int) {
                  self.value = value
              }
          }

          resource ResourceLoser {
              var dict: @{Int: Resource}
              var toBeLost: @Resource

              init(_ victim: @Resource) {
                  self.dict <- {1: <- create Resource(2)}

                  self.toBeLost <- victim

                  // Magic happens during the swap below.
                  self.dict[1] <-> self.dict[self.shenanigans()]
              }

              fun shenanigans(): Int {
                  var d <- create Resource(3)

                  self.toBeLost <-> d

                  // This takes advantage of the fact that self.dict[1] has been
                  // temporarily removed at the point of the swap when this gets called
                  // We take advantage of this window of opportunity to
                  // insert the "to-be-lost" resource in its place. The swap implementation
                  // will blindly overwrite it.
                  var old <- self.dict.insert(key: 1, <- d)

                  // "old" will be nil here thanks to the removal done by the swap
                  // implementation. We have to destroy it to please sema checker.
                  destroy old

                  return 1
              }
          }

          fun test() {
              destroy <- create ResourceLoser(<- create Resource(1))
          }
        `)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		RequireError(t, err)

		assert.ErrorAs(t, err, &interpreter.UseBeforeInitializationError{})

		require.Empty(t, getEvents())
	})
}

func TestInterpretOptionalAddressInConditional(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      fun test(ok: Bool): Address? {
         return ok ? 0x1 : nil
      }
    `)

	value, err := inter.Invoke("test", interpreter.TrueValue)
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewSomeValueNonCopying(nil,
			interpreter.NewUnmeteredAddressValueFromBytes([]byte{0x1}),
		),
		value,
	)
}

func TestInterpretStringTemplates(t *testing.T) {

	t.Parallel()

	t.Run("int", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
			let x = 123
			let y = "x = \(x)"
		`)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredIntValueFromInt64(123),
			inter.Globals.Get("x").GetValue(inter),
		)
		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredStringValue("x = 123"),
			inter.Globals.Get("y").GetValue(inter),
		)
	})

	t.Run("multiple", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
			let x = 123.321
			let y = "abc"
			let z = "\(y) and \(x)"
		`)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredStringValue("abc and 123.32100000"),
			inter.Globals.Get("z").GetValue(inter),
		)
	})

	t.Run("nested template", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
			let x = "{}"
			let y = "[\(x)]"
			let z = "(\(y))"
		`)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredStringValue("([{}])"),
			inter.Globals.Get("z").GetValue(inter),
		)
	})

	t.Run("boolean", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
			let x = false
			let y = "\(x)"
		`)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredStringValue("false"),
			inter.Globals.Get("y").GetValue(inter),
		)
	})

	t.Run("func extracted", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
			let add = fun(): Int {
				return 2+2
			}
			let y = add()
			let x: String = "\(y)"
		`)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredStringValue("4"),
			inter.Globals.Get("x").GetValue(inter),
		)
	})

	t.Run("path expr", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
			let a = /public/foo
			let x = "file at \(a)"
		`)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredStringValue("file at /public/foo"),
			inter.Globals.Get("x").GetValue(inter),
		)
	})

	t.Run("consecutive", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
			let c = "C"
			let a: Character = "A"
			let n = "N"
			let x = "\(c)\(a)\(n)"
		`)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredStringValue("CAN"),
			inter.Globals.Get("x").GetValue(inter),
		)
	})

	t.Run("func", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
			let add = fun(): Int {
				return 2+2
			}
			let x: String = "\(add())"
		`)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredStringValue("4"),
			inter.Globals.Get("x").GetValue(inter),
		)
	})

	t.Run("ternary", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
			let z = false
			let x: String = "\(z ? "foo" : "bar" )"
		`)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredStringValue("bar"),
			inter.Globals.Get("x").GetValue(inter),
		)
	})

	t.Run("nested", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
			let x: String = "\(2*(4-2) + 1 == 5)"
		`)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredStringValue("true"),
			inter.Globals.Get("x").GetValue(inter),
		)
	})
}

func TestInterpretSomeValueChildContainerMutation(t *testing.T) {

	t.Parallel()

	test := func(t *testing.T, code string) {

		t.Parallel()

		ledger := NewTestLedger(nil, nil)

		newInter := func() *interpreter.Interpreter {

			inter, err := parseCheckAndInterpretWithOptions(t,
				code,
				ParseCheckAndInterpretOptions{
					Config: &interpreter.Config{
						Storage: runtime.NewStorage(ledger, nil, runtime.StorageConfig{}),
					},
				},
			)
			require.NoError(t, err)

			return inter
		}

		// Setup

		inter := newInter()

		foo, err := inter.Invoke("setup")
		require.NoError(t, err)

		address := common.MustBytesToAddress([]byte{0x1})
		path := interpreter.NewUnmeteredPathValue(common.PathDomainStorage, "foo")

		storage := inter.Storage().(*runtime.Storage)
		storageMap := storage.GetDomainStorageMap(
			inter,
			address,
			common.StorageDomain(path.Domain),
			true,
		)

		foo = foo.Transfer(
			inter,
			interpreter.EmptyLocationRange,
			atree.Address(address),
			false,
			nil,
			nil,
			true,
		)

		// Write the value to the storage map.
		// However, the value is not referenced by the root of the storage yet
		// (a storage map), so atree storage validation must be temporarily disabled
		// to not report any "unreferenced slab" errors.
		withoutAtreeStorageValidationEnabled(
			inter,
			func() struct{} {
				storageMap.WriteValue(
					inter,
					interpreter.StringStorageMapKey(path.Identifier),
					foo,
				)
				return struct{}{}
			},
		)

		err = storage.Commit(inter, false)
		require.NoError(t, err)

		// Update

		inter = newInter()

		storage = inter.Storage().(*runtime.Storage)
		storageMap = storage.GetDomainStorageMap(
			inter,
			address,
			common.StorageDomain(path.Domain),
			false,
		)
		require.NotNil(t, storageMap)

		ref := interpreter.NewStorageReferenceValue(
			nil,
			interpreter.UnauthorizedAccess,
			address,
			path,
			nil,
		)

		result, err := inter.Invoke("update", ref)
		require.NoError(t, err)
		assert.Equal(t, interpreter.TrueValue, result)

		err = storage.Commit(inter, false)
		require.NoError(t, err)

		// Update again

		inter = newInter()

		storage = inter.Storage().(*runtime.Storage)
		storageMap = storage.GetDomainStorageMap(
			inter,
			address,
			common.StorageDomain(path.Domain),
			false,
		)
		require.NotNil(t, storageMap)

		ref = interpreter.NewStorageReferenceValue(
			nil,
			interpreter.UnauthorizedAccess,
			address,
			path,
			nil,
		)

		result, err = inter.Invoke("updateAgain", ref)
		require.NoError(t, err)
		assert.Equal(t, interpreter.TrueValue, result)
	}

	t.Run("dictionary, one level", func(t *testing.T) {

		test(t, `
            struct Foo {
                let values: {String: Int}?

                init() {
                    self.values = {}
                }

                fun set(key: String, value: Int) {
                    if let ref: auth(Mutate) &{String: Int} = &self.values {
                        ref[key] = value
                    }
                }

                fun get(key: String): Int? {
                    if let ref: &{String: Int} = &self.values {
                        return ref[key]
                    }
                    return nil
                }
            }

            fun setup(): Foo {
                let foo = Foo()
                foo.set(key: "a", value: 1)
                return foo
            }

            fun update(foo: &Foo): Bool {
                if foo.get(key: "a") != 1 {
                     return false
                }
                foo.set(key: "a", value: 2)
                return true
            }

            fun updateAgain(foo: &Foo): Bool {
                if foo.get(key: "a") != 2 {
                     return false
                }
                foo.set(key: "a", value: 3)
                return true
            }
        `)
	})

	t.Run("dictionary, two levels", func(t *testing.T) {
		test(t, `
            struct Foo {
                let values: {String: Int}??

                init() {
                    self.values = {}
                }

                fun set(key: String, value: Int) {
                    if let optRef: auth(Mutate) &{String: Int}? = &self.values {
                        if let ref: auth(Mutate) &{String: Int} = optRef {
                            ref[key] = value
                        }
                    }
                }

                fun get(key: String): Int? {
                    if let optRef: &{String: Int}? = &self.values {
                        if let ref: &{String: Int} = optRef {
                            return ref[key]
                        }
                    }
                    return nil
                }
            }

            fun setup(): Foo {
                let foo = Foo()
                foo.set(key: "a", value: 1)
                return foo
            }

            fun update(foo: &Foo): Bool {
                if foo.get(key: "a") != 1 {
                     return false
                }
                foo.set(key: "a", value: 2)
                return true
            }

            fun updateAgain(foo: &Foo): Bool {
                if foo.get(key: "a") != 2 {
                     return false
                }
                foo.set(key: "a", value: 3)
                return true
            }
       `)
	})

	t.Run("dictionary, nested", func(t *testing.T) {

		test(t, `
            struct Bar {
                let values: {String: Int}?

                init() {
                    self.values = {}
                }

                fun set(key: String, value: Int) {
                    if let ref: auth(Mutate) &{String: Int} = &self.values {
                        ref[key] = value
                    }
                }

                fun get(key: String): Int? {
                    if let ref: &{String: Int} = &self.values {
                        return ref[key]
                    }
                    return nil
                }
            }

            struct Foo {
                let values: {String: Bar}?

                init() {
                    self.values = {}
                }

                fun set(key: String, value: Int) {
                    if let ref: auth(Mutate) &{String: Bar} = &self.values {
                        if ref[key] == nil {
                            ref[key] = Bar()
                        }
                        ref[key]?.set(key: key, value: value)
                    }
                }

                fun get(key: String): Int? {
                    if let ref: &{String: Bar} = &self.values {
                        return ref[key]?.get(key: key) ?? nil
                    }
                    return nil
                }
            }

            fun setup(): Foo {
                let foo = Foo()
                foo.set(key: "a", value: 1)
                return foo
            }

            fun update(foo: &Foo): Bool {
                if foo.get(key: "a") != 1 {
                     return false
                }
                foo.set(key: "a", value: 2)
                return true
            }

            fun updateAgain(foo: &Foo): Bool {
                if foo.get(key: "a") != 2 {
                     return false
                }
                foo.set(key: "a", value: 3)
                return true
            }
        `)
	})

	t.Run("resource, one level", func(t *testing.T) {

		test(t, `

              resource Bar {
                  var value: Int

                  init() {
                      self.value = 0
                  }
              }

              resource Foo {
                  let bar: @Bar?

                  init() {
                      self.bar <- create Bar()
                  }

                  fun set(value: Int) {
                      if let ref: &Bar = &self.bar {
                          ref.value = value
                      }
                  }

                  fun getValue(): Int? {
                      return self.bar?.value
                  }
              }

              fun setup(): @Foo {
                  let foo <- create Foo()
                  foo.set(value: 1)
                  return <-foo
              }

              fun update(foo: &Foo): Bool {
                  if foo.getValue() != 1 {
                       return false
                  }
                  foo.set(value: 2)
                  return true
              }

              fun updateAgain(foo: &Foo): Bool {
                  if foo.getValue() != 2 {
                       return false
                  }
                  foo.set(value: 3)
                  return true
              }
        `)

	})

	t.Run("resource, two levels", func(t *testing.T) {

		test(t, `

              resource Bar {
                  var value: Int

                  init() {
                      self.value = 0
                  }
              }

              resource Foo {
                  let bar: @Bar??

                  init() {
                      self.bar <- create Bar()
                  }

                  fun set(value: Int) {
                      if let optRef: &Bar? = &self.bar {
                          if let ref = optRef {
                              ref.value = value
                          }
                      }
                  }

                  fun getValue(): Int? {
                      if let optRef: &Bar? = &self.bar {
                          return optRef?.value
                      }
                      return nil
                  }
              }

              fun setup(): @Foo {
                  let foo <- create Foo()
                  foo.set(value: 1)
                  return <-foo
              }

              fun update(foo: &Foo): Bool {
                  if foo.getValue() != 1 {
                       return false
                  }
                  foo.set(value: 2)
                  return true
              }

              fun updateAgain(foo: &Foo): Bool {
                  if foo.getValue() != 2 {
                       return false
                  }
                  foo.set(value: 3)
                  return true
              }
        `)
	})

	t.Run("resource, nested", func(t *testing.T) {

		test(t, `
              resource Baz {
                  var value: Int

                  init() {
                      self.value = 0
                  }
              }

              resource Bar {
                  let baz: @Baz?

                  init() {
                      self.baz <- create Baz()
                  }

                  fun set(value: Int) {
                      if let ref: &Baz = &self.baz {
                          ref.value = value
                      }
                  }

                  fun getValue(): Int? {
                      return self.baz?.value
                  }
              }

              resource Foo {
                  let bar: @Bar?

                  init() {
                      self.bar <- create Bar()
                  }

                  fun set(value: Int) {
                      if let ref: &Bar = &self.bar {
                          ref.set(value: value)
                      }
                  }

                  fun getValue(): Int? {
                      return self.bar?.getValue() ?? nil
                  }
              }

              fun setup(): @Foo {
                  let foo <- create Foo()
                  foo.set(value: 1)
                  return <-foo
              }

              fun update(foo: &Foo): Bool {
                  if foo.getValue() != 1 {
                       return false
                  }
                  foo.set(value: 2)
                  return true
              }

              fun updateAgain(foo: &Foo): Bool {
                  if foo.getValue() != 2 {
                       return false
                  }
                  foo.set(value: 3)
                  return true
              }
        `)
	})

	t.Run("array, one level", func(t *testing.T) {

		test(t, `

          struct Foo {
              let values: [Int]?

              init() {
                  self.values = []
              }

              fun set(value: Int) {
                  if let ref: auth(Mutate) &[Int] = &self.values {
                      if ref.length == 0 {
                         ref.append(value)
                      } else {
                         ref[0] = value
                      }
                  }
              }

              fun getValue(): Int? {
                  if let ref: &[Int] = &self.values {
                      return ref[0]
                  }
                  return nil
              }
          }

          fun setup(): Foo {
              let foo = Foo()
              foo.set(value: 1)
              return foo
          }

          fun update(foo: &Foo): Bool {
              if foo.getValue() != 1 {
                   return false
              }
              foo.set(value: 2)
              return true
          }

          fun updateAgain(foo: &Foo): Bool {
              if foo.getValue() != 2 {
                   return false
              }
              foo.set(value: 3)
              return true
          }
        `)

	})

	t.Run("array, two levels", func(t *testing.T) {

		test(t, `

          struct Foo {
              let values: [Int]??

              init() {
                  self.values = []
              }

              fun set(value: Int) {
                  if let optRef: auth(Mutate) &[Int]? = &self.values {
                      if let ref = optRef {
                          if ref.length == 0 {
                             ref.append(value)
                          } else {
                             ref[0] = value
                          }
                      }
                  }
              }

              fun getValue(): Int? {
                  if let optRef: &[Int]? = &self.values {
                      if let ref = optRef {
                          return ref[0]
                      }
                  }
                  return nil
              }
          }

          fun setup(): Foo {
              let foo = Foo()
              foo.set(value: 1)
              return foo
          }

          fun update(foo: &Foo): Bool {
              if foo.getValue() != 1 {
                   return false
              }
              foo.set(value: 2)
              return true
          }

          fun updateAgain(foo: &Foo): Bool {
              if foo.getValue() != 2 {
                   return false
              }
              foo.set(value: 3)
              return true
          }
        `)
	})

	t.Run("array, nested", func(t *testing.T) {

		test(t, `

           struct Bar {
              let values: [Int]?

              init() {
                  self.values = []
              }

              fun set(value: Int) {
                  if let ref: auth(Mutate) &[Int] = &self.values {
                      if ref.length == 0 {
                         ref.append(value)
                      } else {
                         ref[0] = value
                      }
                  }
              }

              fun getValue(): Int? {
                  if let ref: &[Int] = &self.values {
                      return ref[0]
                  }
                  return nil
              }
          }

          struct Foo {
              let values: [Bar]?

              init() {
                  self.values = []
              }

              fun set(value: Int) {
                  if let ref: auth(Mutate) &[Bar] = &self.values {
                      if ref.length == 0 {
                         ref.append(Bar())
                      }
                      ref[0].set(value: value)
                  }
              }

              fun getValue(): Int? {
                  if let ref: &[Bar] = &self.values {
                      return ref[0].getValue()
                  }
                  return nil
              }
          }

          fun setup(): Foo {
              let foo = Foo()
              foo.set(value: 1)
              return foo
          }

          fun update(foo: &Foo): Bool {
              if foo.getValue() != 1 {
                   return false
              }
              foo.set(value: 2)
              return true
          }

          fun updateAgain(foo: &Foo): Bool {
              if foo.getValue() != 2 {
                   return false
              }
              foo.set(value: 3)
              return true
          }
        `)

	})
}
