/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2022 Dapper Labs, Inc.
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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/tests/utils"
)

func TestCheckRange(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheckWithOptions(t,
		`
          fun _TEST_foo(_TEST_a: Int) {
              let _TEST_b = 2
              if true {
                  var _TEST_c = 3
              } else {
                  let _TEST_d = 4
              }
              while true {
                  let _TEST_e = "5"
              }
          }

          struct _TEST_Bar {
              let _TEST_x: Int

              init() {
                  self._TEST_x = 0
              }

              fun _TEST_bar() {}

              fun _TEST_baz() {}
          }

          resource _TEST_Baz {}
        `,
		ParseAndCheckOptions{
			Config: &sema.Config{
				PositionInfoEnabled: true,
			},
		},
	)
	assert.NoError(t, err)

	var ranges map[sema.Range]int

	// we don't care about the ordering of these ranges, but that finding all the ranges at a position returns the correct values.
	getCounts := func(ranges []sema.Range) map[sema.Range]int {
		bag := make(map[sema.Range]int, len(ranges))
		for _, rnge := range ranges {
			if !strings.HasPrefix(rnge.Identifier, "_TEST_") {
				continue
			}
			count, _ := bag[rnge] // default to 0
			bag[rnge] = count + 1
		}
		return bag
	}

	getUnorderedRanges := func(pos *sema.Position) map[sema.Range]int {
		if pos == nil {
			return getCounts(checker.PositionInfo.Ranges.All())
		}
		return getCounts(checker.PositionInfo.Ranges.FindAll(*pos))
	}

	// assert that the unordered repr of expected matches that of ranges
	assertSetsEqual := func(t *testing.T, expected []sema.Range, ranges map[sema.Range]int) {
		bag := getCounts(expected)
		utils.AssertEqualWithDiff(t, bag, ranges)
	}

	barTypeVariable, ok := checker.Elaboration.GlobalTypes.Get("_TEST_Bar")
	require.True(t, ok, "missing global type _TEST_Bar")

	barValueVariable, ok := checker.Elaboration.GlobalValues.Get("_TEST_Bar")
	require.True(t, ok, "missing global value _TEST_Bar")

	bazTypeVariable, ok := checker.Elaboration.GlobalTypes.Get("_TEST_Baz")
	require.True(t, ok, "missing global type _TEST_Baz")

	bazValueVariable, ok := checker.Elaboration.GlobalValues.Get("_TEST_Baz")
	require.True(t, ok, "missing global value _TEST_Baz")

	fooValueVariable, ok := checker.Elaboration.GlobalValues.Get("_TEST_foo")
	require.True(t, ok, "missing global value _TEST_foo")

	ranges = getUnorderedRanges(nil)

	assertSetsEqual(t,
		[]sema.Range{
			{
				Identifier:      "_TEST_Bar",
				Type:            barValueVariable.Type,
				DeclarationKind: common.DeclarationKindStructure,
			},
			{
				Identifier:      "_TEST_Bar",
				Type:            barTypeVariable.Type,
				DeclarationKind: common.DeclarationKindStructure,
			},
			{
				Identifier:      "_TEST_foo",
				Type:            fooValueVariable.Type,
				DeclarationKind: common.DeclarationKindFunction,
			},
			{
				Identifier:      "_TEST_Baz",
				Type:            bazValueVariable.Type,
				DeclarationKind: common.DeclarationKindResource,
			},
			{
				Identifier:      "_TEST_Baz",
				Type:            bazTypeVariable.Type,
				DeclarationKind: common.DeclarationKindResource,
			},
			{
				Identifier:      "_TEST_a",
				Type:            sema.IntType,
				DeclarationKind: common.DeclarationKindParameter,
			},
			{
				Identifier:      "_TEST_b",
				Type:            sema.IntType,
				DeclarationKind: common.DeclarationKindConstant,
			},
			{
				Identifier:      "_TEST_c",
				Type:            sema.IntType,
				DeclarationKind: common.DeclarationKindVariable,
			},
			{
				Identifier:      "_TEST_d",
				Type:            sema.IntType,
				DeclarationKind: common.DeclarationKindConstant,
			},
			{
				Identifier:      "_TEST_e",
				Type:            sema.StringType,
				DeclarationKind: common.DeclarationKindConstant,
			},
		},
		ranges,
	)

	ranges = getUnorderedRanges(&sema.Position{Line: 8, Column: 0})
	assertSetsEqual(t,
		[]sema.Range{
			{
				Identifier:      "_TEST_Bar",
				Type:            barValueVariable.Type,
				DeclarationKind: common.DeclarationKindStructure,
			},
			{
				Identifier:      "_TEST_Bar",
				Type:            barTypeVariable.Type,
				DeclarationKind: common.DeclarationKindStructure,
			},
			{
				Identifier:      "_TEST_foo",
				Type:            fooValueVariable.Type,
				DeclarationKind: common.DeclarationKindFunction,
			},
			{
				Identifier:      "_TEST_Baz",
				Type:            bazValueVariable.Type,
				DeclarationKind: common.DeclarationKindResource,
			},
			{
				Identifier:      "_TEST_Baz",
				Type:            bazTypeVariable.Type,
				DeclarationKind: common.DeclarationKindResource,
			},
			{
				Identifier:      "_TEST_a",
				Type:            sema.IntType,
				DeclarationKind: common.DeclarationKindParameter,
			},
			{
				Identifier:      "_TEST_b",
				Type:            sema.IntType,
				DeclarationKind: common.DeclarationKindConstant,
			},
			{
				Identifier:      "_TEST_d",
				Type:            sema.IntType,
				DeclarationKind: common.DeclarationKindConstant,
			},
		},
		ranges,
	)

	ranges = getUnorderedRanges(&sema.Position{Line: 8, Column: 100})
	assertSetsEqual(t,
		[]sema.Range{
			{
				Identifier:      "_TEST_Bar",
				Type:            barValueVariable.Type,
				DeclarationKind: common.DeclarationKindStructure,
			},
			{
				Identifier:      "_TEST_Bar",
				Type:            barTypeVariable.Type,
				DeclarationKind: common.DeclarationKindStructure,
			},
			{
				Identifier:      "_TEST_Baz",
				Type:            bazValueVariable.Type,
				DeclarationKind: common.DeclarationKindResource,
			},
			{
				Identifier:      "_TEST_Baz",
				Type:            bazTypeVariable.Type,
				DeclarationKind: common.DeclarationKindResource,
			},
			{
				Identifier:      "_TEST_a",
				Type:            sema.IntType,
				DeclarationKind: common.DeclarationKindParameter,
			},
			{
				Identifier:      "_TEST_b",
				Type:            sema.IntType,
				DeclarationKind: common.DeclarationKindConstant,
			},
			{
				Identifier:      "_TEST_foo",
				Type:            fooValueVariable.Type,
				DeclarationKind: common.DeclarationKindFunction,
			},
		},
		ranges,
	)
}
