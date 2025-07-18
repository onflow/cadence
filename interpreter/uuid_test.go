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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
	. "github.com/onflow/cadence/test_utils/common_utils"
	. "github.com/onflow/cadence/test_utils/interpreter_utils"
	. "github.com/onflow/cadence/test_utils/sema_utils"
)

func TestInterpretResourceUUID(t *testing.T) {

	t.Parallel()

	importedChecker, err := ParseAndCheckWithOptions(t,
		`
          access(all) resource R {}

          access(all) fun createR(): @R {
              return <- create R()
          }
        `,
		ParseAndCheckOptions{
			Location: ImportedLocation,
		},
	)
	require.NoError(t, err)

	importingChecker, err := ParseAndCheckWithOptions(t,
		`
          import createR from "imported"

          access(all) resource R2 {}

          access(all) fun createRs(): @[AnyResource] {
              return <- [
                  <- (createR() as @AnyResource),
                  <- create R2()
              ]
          }
        `,
		ParseAndCheckOptions{
			CheckerConfig: &sema.Config{
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

	var uuid uint64

	storage := newUnmeteredInMemoryStorage()

	inter, err := interpreter.NewInterpreter(
		interpreter.ProgramFromChecker(importingChecker),
		importingChecker.Location,
		&interpreter.Config{
			Storage: storage,
			UUIDHandler: func() (uint64, error) {
				defer func() { uuid++ }()
				return uuid, nil
			},
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

	value, err := inter.Invoke("createRs")
	require.NoError(t, err)

	require.IsType(t, &interpreter.ArrayValue{}, value)

	array := value.(*interpreter.ArrayValue)

	const length = 2

	require.Equal(t, length, array.Count())

	for i := 0; i < length; i++ {
		element := array.Get(inter, interpreter.EmptyLocationRange, i)

		require.IsType(t, &interpreter.CompositeValue{}, element)
		res := element.(*interpreter.CompositeValue)

		uuidValue := res.GetMember(
			inter,
			interpreter.EmptyLocationRange,
			sema.ResourceUUIDFieldName,
		)

		RequireValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredUInt64Value(uint64(i)),
			uuidValue,
		)
	}
}
