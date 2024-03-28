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

package vm

import (
	"github.com/onflow/cadence/runtime/bbq"
)

type callFrame struct {
	parent   *callFrame
	locals   []Value
	function *bbq.Function
	ip       uint16
}

func (f *callFrame) getUint16() uint16 {
	first := f.function.Code[f.ip]
	last := f.function.Code[f.ip+1]
	f.ip += 2
	return uint16(first)<<8 | uint16(last)
}
