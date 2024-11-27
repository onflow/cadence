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
	"github.com/onflow/cadence/bbq"
)

type callFrame struct {
	executable   *ExecutableProgram
	localsOffset uint16
	localsCount  uint16
	function     *bbq.Function
}

//
//func (f *callFrame) getUint16() uint16 {
//	first := f.function.Code[f.ip]
//	last := f.function.Code[f.ip+1]
//	f.ip += 2
//	return uint16(first)<<8 | uint16(last)
//}
//
//func (f *callFrame) getByte() byte {
//	byt := f.function.Code[f.ip]
//	f.ip++
//	return byt
//}
//
//func (f *callFrame) getBool() bool {
//	byt := f.function.Code[f.ip]
//	f.ip++
//	return byt == 1
//}
//
//func (f *callFrame) getString() string {
//	strLen := f.getUint16()
//	str := string(f.function.Code[f.ip : f.ip+strLen])
//	f.ip += strLen
//	return str
//}
