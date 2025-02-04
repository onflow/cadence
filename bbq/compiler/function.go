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
	name                string
	code                []E
	localCount          uint16
	locals              *activations.Activations[*local]
	parameterCount      uint16
	isCompositeFunction bool
}

func newFunction[E any](name string, parameterCount uint16, isCompositeFunction bool) *function[E] {
	return &function[E]{
		name:                name,
		parameterCount:      parameterCount,
		locals:              activations.NewActivations[*local](nil),
		isCompositeFunction: isCompositeFunction,
	}
}

func (f *function[E]) generateLocalIndex() uint16 {
	index := f.localCount
	f.localCount++
	return index
}

func (f *function[E]) declareLocal(name string) *local {
	if f.localCount == math.MaxUint16 {
		panic(errors.NewDefaultUserError("invalid local declaration"))
	}
	index := f.generateLocalIndex()
	local := &local{index: index}
	f.locals.Set(name, local)
	return local
}

func (f *function[E]) findLocal(name string) *local {
	return f.locals.Find(name)
}
