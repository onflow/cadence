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
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/sema"
)

func TestCheckNumberConversionReplacementHint(t *testing.T) {

	t.Parallel()

	// to fixed point type

	//// integer literal

	t.Run("positive integer to signed fixed-point type", func(t *testing.T) {

		t.Parallel()

		checker, err := ParseAndCheckWithLinting(t, `
           let x = Fix64(1)
        `)

		require.NoError(t, err)

		hints := checker.Hints()
		require.Len(t, hints, 1)
		require.IsType(t, &sema.ReplacementHint{}, hints[0])

		require.Equal(t,
			"consider replacing with: `1.0 as Fix64`",
			hints[0].(*sema.ReplacementHint).Hint(),
		)
	})

	t.Run("positive integer to unsigned fixed-point type", func(t *testing.T) {

		t.Parallel()

		checker, err := ParseAndCheckWithLinting(t, `
           let x = UFix64(1)
        `)

		require.NoError(t, err)

		hints := checker.Hints()
		require.Len(t, hints, 1)
		require.IsType(t, &sema.ReplacementHint{}, hints[0])

		require.Equal(t,
			"consider replacing with: `1.0`",
			hints[0].(*sema.ReplacementHint).Hint(),
		)
	})

	t.Run("negative integer to signed fixed-point type", func(t *testing.T) {

		t.Parallel()

		checker, err := ParseAndCheckWithLinting(t, `
           let x = Fix64(-1)
        `)

		require.NoError(t, err)

		hints := checker.Hints()
		require.Len(t, hints, 1)
		require.IsType(t, &sema.ReplacementHint{}, hints[0])

		require.Equal(t,
			"consider replacing with: `-1.0`",
			hints[0].(*sema.ReplacementHint).Hint(),
		)
	})

	//// fixed-point literal

	t.Run("positive fixed-point to unsigned fixed-point type", func(t *testing.T) {

		t.Parallel()

		checker, err := ParseAndCheckWithLinting(t, `
           let x = UFix64(1.2)
        `)

		require.NoError(t, err)

		hints := checker.Hints()
		require.Len(t, hints, 1)
		require.IsType(t, &sema.ReplacementHint{}, hints[0])

		require.Equal(t,
			"consider replacing with: `1.2`",
			hints[0].(*sema.ReplacementHint).Hint(),
		)
	})

	t.Run("negative fixed-point to signed fixed-point type", func(t *testing.T) {

		t.Parallel()

		checker, err := ParseAndCheckWithLinting(t, `
           let x = Fix64(-1.2)
        `)

		require.NoError(t, err)

		hints := checker.Hints()
		require.Len(t, hints, 1)
		require.IsType(t, &sema.ReplacementHint{}, hints[0])

		require.Equal(t,
			"consider replacing with: `-1.2`",
			hints[0].(*sema.ReplacementHint).Hint(),
		)
	})

	// to integer type

	//// integer literal

	t.Run("positive integer to unsigned integer type", func(t *testing.T) {

		t.Parallel()

		checker, err := ParseAndCheckWithLinting(t, `
           let x = UInt8(1)
        `)

		require.NoError(t, err)

		hints := checker.Hints()
		require.Len(t, hints, 1)
		require.IsType(t, &sema.ReplacementHint{}, hints[0])

		require.Equal(t,
			"consider replacing with: `1 as UInt8`",
			hints[0].(*sema.ReplacementHint).Hint(),
		)
	})

	t.Run("positive integer to signed integer type", func(t *testing.T) {

		t.Parallel()

		checker, err := ParseAndCheckWithLinting(t, `
           let x = Int8(1)
        `)

		require.NoError(t, err)

		hints := checker.Hints()
		require.Len(t, hints, 1)
		require.IsType(t, &sema.ReplacementHint{}, hints[0])

		require.Equal(t,
			"consider replacing with: `1 as Int8`",
			hints[0].(*sema.ReplacementHint).Hint(),
		)
	})

	t.Run("negative integer to signed integer type", func(t *testing.T) {

		t.Parallel()

		checker, err := ParseAndCheckWithLinting(t, `
           let x = Int8(-1)
        `)

		require.NoError(t, err)

		hints := checker.Hints()
		require.Len(t, hints, 1)
		require.IsType(t, &sema.ReplacementHint{}, hints[0])

		require.Equal(t,
			"consider replacing with: `-1 as Int8`",
			hints[0].(*sema.ReplacementHint).Hint(),
		)
	})

	t.Run("positive integer to Int", func(t *testing.T) {

		t.Parallel()

		checker, err := ParseAndCheckWithLinting(t, `
           let x = Int(1)
        `)

		require.NoError(t, err)

		hints := checker.Hints()
		require.Len(t, hints, 1)
		require.IsType(t, &sema.ReplacementHint{}, hints[0])

		require.Equal(t,
			"consider replacing with: `1`",
			hints[0].(*sema.ReplacementHint).Hint(),
		)
	})

	t.Run("negative integer to Int", func(t *testing.T) {

		t.Parallel()

		checker, err := ParseAndCheckWithLinting(t, `
           let x = Int(-1)
        `)

		require.NoError(t, err)

		hints := checker.Hints()
		require.Len(t, hints, 1)
		require.IsType(t, &sema.ReplacementHint{}, hints[0])

		require.Equal(t,
			"consider replacing with: `-1`",
			hints[0].(*sema.ReplacementHint).Hint(),
		)
	})
}
