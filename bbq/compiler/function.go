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

package compiler

import (
	"math"

	"github.com/onflow/cadence/activations"
	"github.com/onflow/cadence/errors"
)

type function[E any] struct {
	enclosing      *function[E]
	name           string
	code           []E
	locals         *activations.Activations[*local]
	localCount     uint16
	parameterCount uint16
	upvalues       map[upvalue]uint16
}

func newFunction[E any](
	enclosing *function[E],
	name string,
	parameterCount uint16,
) *function[E] {
	return &function[E]{
		enclosing:      enclosing,
		name:           name,
		locals:         activations.NewActivations[*local](nil),
		parameterCount: parameterCount,
	}
}

func (f *function[E]) generateLocalIndex() uint16 {
	if f.localCount == math.MaxUint16 {
		panic(errors.NewDefaultUserError("invalid local declaration"))
	}
	localIndex := f.localCount
	f.localCount++
	return localIndex
}

func (f *function[E]) declareLocal(name string) *local {
	local := &local{
		index: f.generateLocalIndex(),
	}
	f.locals.Set(name, local)
	return local
}

func (f *function[E]) findLocal(name string) *local {
	return f.locals.Find(name)
}

func (f *function[E]) addUpvalue(targetIndex uint16, targetIsLocal bool) uint16 {
	upval := upvalue{
		targetIndex:   targetIndex,
		targetIsLocal: targetIsLocal,
	}

	if upvalueIndex, ok := f.upvalues[upval]; ok {
		return upvalueIndex
	}

	count := len(f.upvalues)
	if count == math.MaxUint16 {
		panic(errors.NewDefaultUserError("invalid upvalue declaration"))
	}

	upvalueIndex := uint16(count)
	if f.upvalues == nil {
		f.upvalues = make(map[upvalue]uint16)
	}
	f.upvalues[upval] = upvalueIndex
	return upvalueIndex
}

func (f *function[E]) findOrAddUpvalue(name string) (upvalueIndex uint16, ok bool) {
	if f.enclosing == nil {
		return 0, false
	}

	enclosingLocal := f.enclosing.findLocal(name)
	if enclosingLocal != nil {
		targetIndex := enclosingLocal.index
		const targetIsLocal = true
		return f.addUpvalue(targetIndex, targetIsLocal), true
	}

	enclosingUpvalueIndex, ok := f.enclosing.findOrAddUpvalue(name)
	if ok {
		targetIndex := enclosingUpvalueIndex
		// target is upvalue
		const targetIsLocal = false
		return f.addUpvalue(targetIndex, targetIsLocal), true
	}

	return 0, false
}
