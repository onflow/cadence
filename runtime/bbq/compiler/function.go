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

package compiler

import (
	"math"

	"github.com/onflow/cadence/runtime/activations"
	"github.com/onflow/cadence/runtime/bbq/opcode"
	"github.com/onflow/cadence/runtime/errors"
)

type function struct {
	name       string
	localCount uint16
	// TODO: use byte.Buffer?
	code           []byte
	locals         *activations.Activations[*local]
	parameterCount uint16
}

func newFunction(name string, parameterCount uint16) *function {
	return &function{
		name:           name,
		parameterCount: parameterCount,
		locals:         activations.NewActivations[*local](nil),
	}
}

func (f *function) emit(opcode opcode.Opcode, args ...byte) int {
	offset := len(f.code)
	f.code = append(f.code, byte(opcode))
	f.code = append(f.code, args...)
	return offset
}

func (f *function) declareLocal(name string) *local {
	if f.localCount >= math.MaxUint16 {
		panic(errors.NewDefaultUserError("invalid local declaration"))
	}
	index := f.localCount
	f.localCount++
	local := &local{index: index}
	f.locals.Set(name, local)
	return local
}

func (f *function) findLocal(name string) *local {
	return f.locals.Find(name)
}
