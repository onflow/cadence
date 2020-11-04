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

package ast

import (
	"encoding/json"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProgram_MarshalJSON(t *testing.T) {

	t.Parallel()

	program := &Program{
		Declarations: []Declaration{},
	}

	actual, err := json.Marshal(program)
	require.NoError(t, err)

	assert.JSONEq(t,
		`
        {
            "Type": "Program",
            "Declarations": []
        }
        `,
		string(actual),
	)
}

func TestProgramIndices(t *testing.T) {

	functionA := &FunctionDeclaration{}
	functionB := &FunctionDeclaration{}
	functionC := &FunctionDeclaration{}

	compositeA := &CompositeDeclaration{}
	compositeB := &CompositeDeclaration{}
	compositeC := &CompositeDeclaration{}

	program := &Program{
		Declarations: []Declaration{
			functionC,
			compositeB,
			functionA,
			compositeC,
			functionB,
			compositeA,
		},
	}

	var wg sync.WaitGroup
	const parallelExecutionCount = 10

	for i := 0; i < parallelExecutionCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			require.Equal(t,
				[]*CompositeDeclaration{
					compositeB,
					compositeC,
					compositeA,
				},
				program.CompositeDeclarations(),
			)

			require.Equal(t,
				[]*FunctionDeclaration{
					functionB,
					functionC,
					functionA,
				},
				program.FunctionDeclarations(),
			)
		}()
	}

	wg.Wait()

}
