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

package common

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenerateSliceWithNoDuplicates(t *testing.T) {
	t.Parallel()

	t.Run("int mod 4", func(t *testing.T) {

		t.Parallel()

		callNumber := 0
		generator := func() *int {
			if callNumber > 20 {
				return nil
			}
			mod := callNumber % 4
			callNumber++
			return &mod
		}

		slice := GenerateSliceWithNoDuplicates(generator)

		require.Equal(t, []int{0, 1, 2, 3}, slice)
	})
}

func TestMappedSliceWithNoDuplicates(t *testing.T) {
	t.Parallel()

	t.Run("identity with dupes", func(t *testing.T) {

		t.Parallel()

		ts := []int{0, 1, 2, 4, 1, 3, 4, 3, 2, 3}

		slice := MappedSliceWithNoDuplicates(ts, func(t int) int { return t })

		require.Equal(t, []int{0, 1, 2, 4, 3}, slice)
	})

	t.Run("first of a pair", func(t *testing.T) {

		t.Parallel()

		ts := []struct {
			A string
			B int
		}{
			{"b", 3},
			{"d", 10},
			{"a", 0},
			{"a", 1},
			{"b", 1},
			{"d", 10},
			{"a", 2},
			{"a", 3},
			{"c", 2},
		}

		slice := MappedSliceWithNoDuplicates(ts, func(t struct {
			A string
			B int
		}) string {
			return t.A
		})

		require.Equal(t, []string{"b", "d", "a", "c"}, slice)
	})
}
