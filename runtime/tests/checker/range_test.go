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
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/sema"
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
			Options: []sema.Option{
				sema.WithPositionInfoEnabled(true),
			},
		},
	)
	assert.NoError(t, err)

	var ranges []sema.Range

	isLess := func(a, b sema.Range) bool {
		res := strings.Compare(a.Identifier, b.Identifier)
		switch res {
		case -1:
			return true
		case 1:
			return false
		default:
			if a.DeclarationKind < b.DeclarationKind {
				return true
			} else if a.DeclarationKind > b.DeclarationKind {
				return false
			}
			return strings.Compare(string(a.Type.ID()), string(b.Type.ID())) < 0
		}
	}

	sortAndFilterRanges := func() {
		filteredRanges := make([]sema.Range, 0, len(ranges))
		for _, r := range ranges {
			if !strings.HasPrefix(r.Identifier, "_TEST_") {
				continue
			}
			filteredRanges = append(filteredRanges, r)
		}

		ranges = filteredRanges

		sort.SliceStable(ranges, func(i, j int) bool {
			a := ranges[i]
			b := ranges[j]
			return isLess(a, b)
		})
	}

	ranges = checker.Ranges.All()
	sortAndFilterRanges()

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

	assert.Equal(t,
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
			{
				Identifier:      "_TEST_foo",
				Type:            fooValueVariable.Type,
				DeclarationKind: common.DeclarationKindFunction,
			},
		},
		ranges,
	)

	ranges = checker.Ranges.FindAll(sema.Position{Line: 8, Column: 0})
	sortAndFilterRanges()
	assert.Equal(t,
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
				Identifier:      "_TEST_d",
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

	ranges = checker.Ranges.FindAll(sema.Position{Line: 8, Column: 100})
	sortAndFilterRanges()
	assert.Equal(t,
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
