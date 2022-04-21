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

package ast

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParameterList_ParametersByIdentifier(t *testing.T) {

	t.Run("result", func(t *testing.T) {
		l := &ParameterList{
			Parameters: []*Parameter{
				{
					Label:      "c",
					Identifier: Identifier{Identifier: "d"},
				},
				{
					Label:      "a",
					Identifier: Identifier{Identifier: "b"},
				},
			},
		}

		require.Equal(t,
			map[string]*Parameter{
				"d": l.Parameters[0],
				"b": l.Parameters[1],
			},
			l.ParametersByIdentifier(),
		)
	})

	t.Run("thread-safety", func(t *testing.T) {

		// Ensure the ParametersByIdentifier index
		// is constructed in a thread-safe way

		l := &ParameterList{
			Parameters: []*Parameter{
				{
					Label:      "c",
					Identifier: Identifier{Identifier: "d"},
				},
				{
					Label:      "a",
					Identifier: Identifier{Identifier: "b"},
				},
			},
		}

		var wg sync.WaitGroup
		const parallelExecutionCount = 10

		for i := 0; i < parallelExecutionCount; i++ {
			wg.Add(1)

			go func() {
				defer wg.Done()
				l.ParametersByIdentifier()
			}()
		}

		wg.Wait()
	})
}
