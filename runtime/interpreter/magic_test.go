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

	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/stretchr/testify/assert"
)

func TestPrependMagic(t *testing.T) {

	t.Run("empty", func(t *testing.T) {
		assert.Equal(t,
			[]byte{0x0, 0xCA, 0xDE, 0x0, 0x1},
			interpreter.PrependMagic([]byte{}, 1),
		)
	})

	t.Run("1, 2, 3", func(t *testing.T) {
		assert.Equal(t,
			[]byte{0x0, 0xCA, 0xDE, 0x0, 0x4, 1, 2, 3},
			interpreter.PrependMagic([]byte{1, 2, 3}, 4),
		)
	})
}
