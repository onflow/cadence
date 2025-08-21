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

package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPatchCode(t *testing.T) {

	t.Parallel()

	t.Run("nested", func(t *testing.T) {

		t.Parallel()

		change := []byte(`.x.ys[0]: 1 != 2`)
		code := []byte(`Outer{x: Inner{ys: []int{1}}}`)
		changes := [][]byte{
			change,
		}

		assert.Equal(
			t,
			`Outer{x: Inner{ys: []int{2}}}`,
			string(patchCode(
				code,
				changes,
			)),
		)

	})

	t.Run("simple", func(t *testing.T) {

		t.Parallel()

		change := []byte(`[0]: 1 != 2`)
		code := []byte(`[]int{1}`)
		changes := [][]byte{
			change,
		}

		assert.Equal(
			t,
			`[]int{2}`,
			string(patchCode(
				code,
				changes,
			)),
		)

	})
}
