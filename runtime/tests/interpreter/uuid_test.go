/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2020 Dapper Labs, Inc.
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

package interpreter

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/cmd"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/tests/checker"
	. "github.com/onflow/cadence/runtime/tests/utils"
)

func TestInterpretResourceUUID(t *testing.T) {

	t.Parallel()

	importedChecker, err := checker.ParseAndCheckWithOptions(t,
		`
          pub resource R {}

          pub fun createR(): @R {
              return <- create R()
          }
        `,
		checker.ParseAndCheckOptions{
			Location: ImportedLocation,
		},
	)
	require.NoError(t, err)

	importingChecker, err := checker.ParseAndCheckWithOptions(t,
		`
          import createR from "imported"

          pub resource R2 {}

          pub fun createRs(): @[AnyResource] {
              return <- [
                  <- (createR() as @AnyResource),
                  <- create R2()
              ]
          }
        `,
		checker.ParseAndCheckOptions{
			Options: []sema.Option{
				sema.WithImportHandler(
					func(checker *sema.Checker, location common.Location) (sema.Import, *sema.CheckerError) {
						assert.Equal(t,
							ImportedLocation,
							location,
						)

						return sema.CheckerImport{
							Checker: importedChecker,
						}, nil
					},
				),
			},
		},
	)

	if err != nil {
		cmd.PrettyPrintError(os.Stdout, err, "", map[string]string{"": ""})
	}

	require.NoError(t, err)

	var uuid uint64

	inter, err := interpreter.NewInterpreter(
		importingChecker,
		interpreter.WithUUIDHandler(
			func() (uint64, error) {
				defer func() { uuid++ }()
				return uuid, nil
			},
		),
		interpreter.WithImportLocationHandler(
			func(inter *interpreter.Interpreter, location common.Location) interpreter.Import {
				assert.Equal(t,
					ImportedLocation,
					location,
				)
				return interpreter.ProgramImport{
					Program: importedChecker.Program,
				}
			},
		),
	)
	require.NoError(t, err)

	err = inter.Interpret()
	require.NoError(t, err)

	value, err := inter.Invoke("createRs")
	require.NoError(t, err)

	require.IsType(t, &interpreter.ArrayValue{}, value)

	array := value.(*interpreter.ArrayValue)

	const length = 2
	require.Len(t, array.Values, length)

	for i := 0; i < length; i++ {
		element := array.Values[i]

		require.IsType(t, &interpreter.CompositeValue{}, element)
		res := element.(*interpreter.CompositeValue)

		require.Equal(t,
			interpreter.UInt64Value(i),
			res.Fields[sema.ResourceUUIDFieldName],
		)
	}
}
