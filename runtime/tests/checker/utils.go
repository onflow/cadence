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

package checker

import (
	"flag"
	"strings"
	"sync"
	"testing"

	"github.com/go-test/deep"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/parser"
	"github.com/onflow/cadence/runtime/pretty"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/tests/utils"
)

func init() {
	deep.MaxDepth = 20
}

func ParseAndCheck(t testing.TB, code string) (*sema.Checker, error) {
	return ParseAndCheckWithOptions(t, code, ParseAndCheckOptions{})
}

type ParseAndCheckOptions struct {
	Location         common.Location
	Config           *sema.Config
	ParseOptions     parser.Config
	IgnoreParseError bool
}

var checkConcurrently = flag.Int(
	"cadence.checkConcurrently",
	0,
	"check programs N times, concurrently. useful for detecting non-determinism, and data races with the -race flag",
)

func ParseAndCheckWithOptions(
	t testing.TB,
	code string,
	options ParseAndCheckOptions,
) (*sema.Checker, error) {
	return ParseAndCheckWithOptionsAndMemoryMetering(t, code, options, nil)
}

func ParseAndCheckWithOptionsAndMemoryMetering(
	t testing.TB,
	code string,
	options ParseAndCheckOptions,
	memoryGauge common.MemoryGauge,
) (*sema.Checker, error) {

	if options.Location == nil {
		options.Location = utils.TestLocation
	}

	program, err := parser.ParseProgram(memoryGauge, []byte(code), options.ParseOptions)
	if !options.IgnoreParseError && !assert.NoError(t, err) {
		var sb strings.Builder
		location := options.Location
		printErr := pretty.NewErrorPrettyPrinter(&sb, true).
			PrettyPrintError(err, location, map[common.Location][]byte{location: []byte(code)})
		if printErr != nil {
			panic(printErr)
		}
		assert.FailNow(t, sb.String())
		return nil, err
	}

	check := func() (*sema.Checker, error) {

		config := options.Config
		if config == nil {
			config = &sema.Config{}
		}

		if config.AccessCheckMode == sema.AccessCheckModeDefault {
			config.AccessCheckMode = sema.AccessCheckModeNotSpecifiedUnrestricted
		}
		config.ExtendedElaborationEnabled = true

		checker, err := sema.NewChecker(
			program,
			options.Location,
			memoryGauge,
			config,
		)
		if err != nil {
			return checker, err
		}

		err = checker.Check()

		return checker, err
	}

	var checker *sema.Checker

	if *checkConcurrently > 1 {

		// Run 10 additional checks in parallel,
		// and ensure all reported errors are equal.
		//
		// This is useful to detect non-determinism ,
		// and when combined with Go testing's race detector,
		// allows detecting data race conditions.

		concurrency := *checkConcurrently

		type result struct {
			checker *sema.Checker
			err     error
		}

		var wg sync.WaitGroup
		results := make(chan result, concurrency)
		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				checker, err := check()
				results <- result{
					checker: checker,
					err:     err,
				}
			}()
		}
		wg.Wait()
		close(results)

		firstResult := <-results
		checker = firstResult.checker
		err = firstResult.err

		for otherResult := range results {
			diff := deep.Equal(err, otherResult.err)
			if diff != nil {
				t.Error(diff)
			}
		}

	} else {
		checker, err = check()
	}

	return checker, err
}

func RequireCheckerErrors(t *testing.T, err error, count int) []error {
	if count <= 0 {
		require.NoError(t, err)
		return nil
	}

	utils.RequireError(t, err)

	var checkerErr *sema.CheckerError
	require.ErrorAs(t, err, &checkerErr)

	errs := checkerErr.Errors

	require.Len(t, errs, count)

	// Get the error message, to check that it can be successfully generated

	for _, checkerErr := range errs {
		utils.RequireError(t, checkerErr)
	}

	return errs
}

func RequireGlobalType(t *testing.T, elaboration *sema.Elaboration, name string) sema.Type {
	variable, ok := elaboration.GetGlobalType(name)
	require.True(t, ok, "global type '%s' missing", name)
	return variable.Type
}

func RequireGlobalValue(t *testing.T, elaboration *sema.Elaboration, name string) sema.Type {
	variable, ok := elaboration.GetGlobalValue(name)
	require.True(t, ok, "global value '%s' missing", name)
	return variable.Type
}
