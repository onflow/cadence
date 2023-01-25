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

package sema

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestActivations(t *testing.T) {

	t.Parallel()

	activations := NewVariableActivations(nil)

	one := &Variable{}
	two := &Variable{}
	three := &Variable{}
	four := &Variable{}
	five := &Variable{}

	activations.Set("a", one)

	assert.Same(t, activations.Find("a"), one)
	assert.Nil(t, activations.Find("b"))

	activations.Enter()

	activations.Set("a", two)
	activations.Set("b", three)

	assert.Same(t, activations.Find("a"), two)
	assert.Same(t, activations.Find("b"), three)
	assert.Nil(t, activations.Find("c"))

	activations.Enter()

	activations.Set("a", five)
	activations.Set("c", four)

	assert.Same(t, activations.Find("a"), five)
	assert.Same(t, activations.Find("b"), three)
	assert.Same(t, activations.Find("c"), four)

	activations.Leave(nil)

	assert.Same(t, activations.Find("a"), two)
	assert.Same(t, activations.Find("b"), three)
	assert.Nil(t, activations.Find("c"))

	activations.Leave(nil)

	assert.Same(t, activations.Find("a"), one)
	assert.Nil(t, activations.Find("b"))
	assert.Nil(t, activations.Find("c"))

	activations.Leave(nil)

	assert.Nil(t, activations.Find("a"))
	assert.Nil(t, activations.Find("b"))
	assert.Nil(t, activations.Find("c"))
}
