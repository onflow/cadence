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

import "github.com/onflow/cadence/interpreter"

const goIntSize = 32 << (^uint(0) >> 63) // 32 or 64
const goMaxInt = 1<<(goIntSize-1) - 1
const goMinInt = -1 << (goIntSize - 1)

func safeAdd(a, b int) int {
	// INT32-C
	if (b > 0) && (a > (goMaxInt - b)) {
		panic(interpreter.OverflowError{})
	} else if (b < 0) && (a < (goMinInt - b)) {
		panic(interpreter.UnderflowError{})
	}
	return a + b
}

func safeAddUint64(a, b uint64) uint64 {
	sum := a + b
	// INT30-C
	if sum < a {
		panic(interpreter.OverflowError{})
	}
	return sum
}
