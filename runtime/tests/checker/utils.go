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

package checker

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/parser2"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/tests/utils"
)

func ParseAndCheck(t *testing.T, code string) (*sema.Checker, error) {
	return ParseAndCheckWithOptions(t, code, ParseAndCheckOptions{})
}

type ParseAndCheckOptions struct {
	Location         ast.Location
	IgnoreParseError bool
	Options          []sema.Option
}

func ParseAndCheckWithOptions(
	t *testing.T,
	code string,
	options ParseAndCheckOptions,
) (*sema.Checker, error) {

	program, err := parser2.ParseProgram(code)
	if !options.IgnoreParseError && !assert.NoError(t, err) {
		assert.FailNow(t, errors.UnrollChildErrors(err))
		return nil, err
	}

	if options.Location == nil {
		options.Location = utils.TestLocation
	}

	checkerOptions := append(
		[]sema.Option{
			sema.WithAccessCheckMode(sema.AccessCheckModeNotSpecifiedUnrestricted),
		},
		options.Options...,
	)

	checker, err := sema.NewChecker(
		program,
		options.Location,
		checkerOptions...,
	)
	if err != nil {
		return checker, err
	}

	err = checker.Check()
	return checker, err
}

func ExpectCheckerErrors(t *testing.T, err error, count int) []error {
	if count <= 0 && err == nil {
		return nil
	}

	require.Error(t, err)

	var checkerErr *sema.CheckerError
	utils.RequireErrorAs(t, err, &checkerErr)

	errs := checkerErr.Errors

	require.Len(t, errs, count)

	// Get the error message, to check that it can be successfully generated

	for _, checkerErr := range errs {
		_ = checkerErr.Error()
	}

	return errs
}
