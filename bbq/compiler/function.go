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

	"github.com/onflow/cadence/errors"
)

type function[E any] struct {
	name           string
	code           []E
	localCount     uint16
	parameterCount uint16
	localsDepth    int
}

func newFunction[E any](
	name string,
	parameterCount uint16,
	localsDepth int,
) *function[E] {
	return &function[E]{
		name:           name,
		parameterCount: parameterCount,
		localsDepth:    localsDepth,
	}
}

func (f *function[E]) generateLocalIndex() uint16 {
	if f.localCount == math.MaxUint16 {
		panic(errors.NewDefaultUserError("invalid local declaration"))
	}
	index := f.localCount
	f.localCount++
	return index
}
