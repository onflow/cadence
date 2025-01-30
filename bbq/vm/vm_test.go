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

package vm

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/bbq"
	"github.com/onflow/cadence/bbq/opcode"
)

func TestVM_pop(t *testing.T) {
	t.Parallel()

	program := &bbq.Program[opcode.Instruction]{}
	vm := NewVM(nil, program, nil)

	vm.push(NewIntValue(1))
	vm.push(NewIntValue(2))

	require.Equal(t,
		vm.stack,
		[]Value{
			NewIntValue(1),
			NewIntValue(2),
		},
	)

	a := vm.pop()

	assert.Equal(t, NewIntValue(2), a)

	assert.Equal(t,
		vm.stack,
		[]Value{
			NewIntValue(1),
		},
	)
}

func TestVM_peekPop(t *testing.T) {
	t.Parallel()

	program := &bbq.Program[opcode.Instruction]{}
	vm := NewVM(nil, program, nil)

	vm.push(NewIntValue(1))
	vm.push(NewIntValue(2))
	vm.push(NewIntValue(3))

	require.Equal(t,
		vm.stack,
		[]Value{
			NewIntValue(1),
			NewIntValue(2),
			NewIntValue(3),
		},
	)

	a, b := vm.peekPop()

	assert.Equal(t, NewIntValue(2), a)
	assert.Equal(t, NewIntValue(3), b)

	assert.Equal(t,
		vm.stack,
		[]Value{
			NewIntValue(1),
			NewIntValue(2),
		},
	)
}

func TestVM_replaceTop(t *testing.T) {
	t.Parallel()

	program := &bbq.Program[opcode.Instruction]{}
	vm := NewVM(nil, program, nil)

	vm.push(NewIntValue(1))
	vm.push(NewIntValue(2))

	require.Equal(t,
		vm.stack,
		[]Value{
			NewIntValue(1),
			NewIntValue(2),
		},
	)

	vm.replaceTop(NewIntValue(3))

	assert.Equal(t,
		vm.stack,
		[]Value{
			NewIntValue(1),
			NewIntValue(3),
		},
	)
}

func TestVM_pop2(t *testing.T) {
	t.Parallel()

	program := &bbq.Program[opcode.Instruction]{}
	vm := NewVM(nil, program, nil)

	vm.push(NewIntValue(1))
	vm.push(NewIntValue(2))
	vm.push(NewIntValue(3))

	a, b := vm.pop2()

	assert.Equal(t, NewIntValue(2), a)
	assert.Equal(t, NewIntValue(3), b)

	assert.Equal(t,
		vm.stack,
		[]Value{
			NewIntValue(1),
		},
	)
}

func TestVM_pop3(t *testing.T) {
	t.Parallel()

	program := &bbq.Program[opcode.Instruction]{}
	vm := NewVM(nil, program, nil)

	vm.push(NewIntValue(1))
	vm.push(NewIntValue(2))
	vm.push(NewIntValue(3))
	vm.push(NewIntValue(4))

	require.Equal(t,
		vm.stack,
		[]Value{
			NewIntValue(1),
			NewIntValue(2),
			NewIntValue(3),
			NewIntValue(4),
		},
	)

	a, b, c := vm.pop3()

	assert.Equal(t, NewIntValue(2), a)
	assert.Equal(t, NewIntValue(3), b)
	assert.Equal(t, NewIntValue(4), c)

	assert.Equal(t,
		vm.stack,
		[]Value{
			NewIntValue(1),
		},
	)
}

func TestVM_peek(t *testing.T) {
	t.Parallel()

	program := &bbq.Program[opcode.Instruction]{}
	vm := NewVM(nil, program, nil)

	vm.push(NewIntValue(1))
	vm.push(NewIntValue(2))
	vm.push(NewIntValue(3))

	require.Equal(t,
		vm.stack,
		[]Value{
			NewIntValue(1),
			NewIntValue(2),
			NewIntValue(3),
		},
	)

	a := vm.peek()

	assert.Equal(t, NewIntValue(3), a)

	assert.Equal(t,
		vm.stack,
		[]Value{
			NewIntValue(1),
			NewIntValue(2),
			NewIntValue(3),
		},
	)
}

func TestVM_peekN(t *testing.T) {
	t.Parallel()

	program := &bbq.Program[opcode.Instruction]{}
	vm := NewVM(nil, program, nil)

	vm.push(NewIntValue(1))
	vm.push(NewIntValue(2))
	vm.push(NewIntValue(3))

	require.Equal(t,
		vm.stack,
		[]Value{
			NewIntValue(1),
			NewIntValue(2),
			NewIntValue(3),
		},
	)

	values := vm.peekN(2)

	assert.Equal(t, NewIntValue(2), values[0])
	assert.Equal(t, NewIntValue(3), values[1])

	assert.Equal(t,
		vm.stack,
		[]Value{
			NewIntValue(1),
			NewIntValue(2),
			NewIntValue(3),
		},
	)
}

func TestVM_dropN(t *testing.T) {
	t.Parallel()

	program := &bbq.Program[opcode.Instruction]{}
	vm := NewVM(nil, program, nil)

	vm.push(NewIntValue(1))
	vm.push(NewIntValue(2))
	vm.push(NewIntValue(3))

	vm.dropN(2)

	assert.Equal(t,
		vm.stack,
		[]Value{
			NewIntValue(1),
		},
	)
}
