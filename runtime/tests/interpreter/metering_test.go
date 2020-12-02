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

package interpreter_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/tests/checker"
	"github.com/onflow/cadence/runtime/tests/utils"
)

func TestInterpretStatementHandler(t *testing.T) {

	t.Parallel()

	importedChecker, err := checker.ParseAndCheckWithOptions(t,
		`
          pub fun a() {
              true
              true
          }
        `,
		checker.ParseAndCheckOptions{
			Location: utils.ImportedLocation,
		},
	)
	require.NoError(t, err)

	importingChecker, err := checker.ParseAndCheckWithOptions(t,
		`
          import a from "imported"

          fun b() {
              true
              true
              a()
              true
              true
          }

          fun c() {
              true
              true
              b()
              true
              true
          }
        `,
		checker.ParseAndCheckOptions{
			Options: []sema.Option{
				sema.WithImportHandler(
					func(checker *sema.Checker, location common.Location) (sema.Import, *sema.CheckerError) {
						assert.Equal(t,
							utils.ImportedLocation,
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
	require.NoError(t, err)

	type occurrence struct {
		interpreterID int
		line          int
	}

	var occurrences []occurrence
	var nextInterpreterID int
	interpreterIDs := map[*interpreter.Interpreter]int{}

	inter, err := interpreter.NewInterpreter(
		importingChecker,
		interpreter.WithOnStatementHandler(
			func(statement *interpreter.Statement) {
				inter := statement.Interpreter

				id, ok := interpreterIDs[inter]
				if !ok {
					id = nextInterpreterID
					nextInterpreterID++
					interpreterIDs[inter] = id
				}

				occurrences = append(occurrences, occurrence{
					interpreterID: id,
					line:          statement.Statement.StartPosition().Line,
				})
			},
		),
		interpreter.WithImportLocationHandler(
			func(inter *interpreter.Interpreter, location common.Location) interpreter.Import {
				assert.Equal(t,
					utils.ImportedLocation,
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

	_, err = inter.Invoke("c")
	require.NoError(t, err)

	assert.Equal(t,
		[]occurrence{
			{0, 13},
			{0, 14},
			{0, 15},
			{0, 5},
			{0, 6},
			{0, 7},
			{1, 3},
			{1, 4},
			{0, 8},
			{0, 9},
			{0, 16},
			{0, 17},
		},
		occurrences,
	)
}

func TestInterpretLoopIterationHandler(t *testing.T) {

	t.Parallel()

	importedChecker, err := checker.ParseAndCheckWithOptions(t,
		`
          pub fun a() {
              var i = 1
              while i <= 4 {
                  i = i + 1
              }

              for n in [1, 2, 3, 4, 5] {}
          }
        `,
		checker.ParseAndCheckOptions{},
	)
	require.NoError(t, err)

	importingChecker, err := checker.ParseAndCheckWithOptions(t,
		`
          import a from "imported"

          fun b() {
              var i = 1
              while i <= 2 {
                  i = i + 1
              }

              for n in [1, 2, 3] {}

              a()
          }
        `,
		checker.ParseAndCheckOptions{
			Options: []sema.Option{
				sema.WithImportHandler(
					func(checker *sema.Checker, location common.Location) (sema.Import, *sema.CheckerError) {
						assert.Equal(t,
							utils.ImportedLocation,
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
	require.NoError(t, err)

	type occurrence struct {
		interpreterID int
		line          int
	}

	var occurrences []occurrence
	var nextInterpreterID int
	interpreterIDs := map[*interpreter.Interpreter]int{}

	inter, err := interpreter.NewInterpreter(
		importingChecker,
		interpreter.WithOnLoopIterationHandler(func(inter *interpreter.Interpreter, line int) {

			id, ok := interpreterIDs[inter]
			if !ok {
				id = nextInterpreterID
				nextInterpreterID++
				interpreterIDs[inter] = id
			}

			occurrences = append(occurrences, occurrence{
				interpreterID: id,
				line:          line,
			})
		}),
		interpreter.WithImportLocationHandler(
			func(inter *interpreter.Interpreter, location common.Location) interpreter.Import {
				assert.Equal(t,
					utils.ImportedLocation,
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

	_, err = inter.Invoke("b")
	require.NoError(t, err)

	assert.Equal(t,
		[]occurrence{
			{0, 6},
			{0, 6},
			{0, 10},
			{0, 10},
			{0, 10},
			{1, 4},
			{1, 4},
			{1, 4},
			{1, 4},
			{1, 8},
			{1, 8},
			{1, 8},
			{1, 8},
			{1, 8},
		},
		occurrences,
	)
}

func TestInterpretFunctionInvocationHandler(t *testing.T) {

	t.Parallel()

	importedChecker, err := checker.ParseAndCheckWithOptions(t,
		`
          pub fun a() {}

          pub fun b() {
              true
              true
              a()
              true
              true
          }
        `,
		checker.ParseAndCheckOptions{
			Location: utils.ImportedLocation,
		},
	)
	require.NoError(t, err)

	importingChecker, err := checker.ParseAndCheckWithOptions(t,
		`
          import b from "imported"

          pub fun c() {
              true
              true
              b()
              true
              true
          }

          pub fun d() {
              true
              true
              c()
              true
              true
          }
        `,
		checker.ParseAndCheckOptions{
			Options: []sema.Option{
				sema.WithImportHandler(
					func(checker *sema.Checker, location common.Location) (sema.Import, *sema.CheckerError) {
						assert.Equal(t,
							utils.ImportedLocation,
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
	require.NoError(t, err)

	type occurrence struct {
		interpreterID int
		line          int
	}

	var occurrences []occurrence
	var nextInterpreterID int
	interpreterIDs := map[*interpreter.Interpreter]int{}

	inter, err := interpreter.NewInterpreter(
		importingChecker,
		interpreter.WithOnFunctionInvocationHandler(
			func(inter *interpreter.Interpreter, line int) {

				id, ok := interpreterIDs[inter]
				if !ok {
					id = nextInterpreterID
					nextInterpreterID++
					interpreterIDs[inter] = id
				}

				occurrences = append(occurrences, occurrence{
					interpreterID: id,
					line:          line,
				})
			},
		),
		interpreter.WithImportLocationHandler(
			func(inter *interpreter.Interpreter, location common.Location) interpreter.Import {
				assert.Equal(t,
					utils.ImportedLocation,
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

	_, err = inter.Invoke("d")
	require.NoError(t, err)

	assert.Equal(t,
		[]occurrence{
			{0, 15},
			{0, 7},
			{1, 7},
		},
		occurrences,
	)
}
