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

package migrations

import (
	"github.com/onflow/cadence/runtime/interpreter"
)

// LegacyStringValue simulates the old string-value
// which uses the un-normalized string for hashing.
type LegacyStringValue struct {
	*interpreter.StringValue
}

var _ interpreter.Value = &LegacyStringValue{}

func (v *LegacyStringValue) HashInput(_ *interpreter.Interpreter, _ interpreter.LocationRange, scratch []byte) []byte {
	// Use the un-normalized `v.UnnormalizedStr` for generating the hash.
	length := 1 + len(v.UnnormalizedStr)
	var buffer []byte
	if length <= len(scratch) {
		buffer = scratch[:length]
	} else {
		buffer = make([]byte, length)
	}

	buffer[0] = byte(interpreter.HashInputTypeString)
	copy(buffer[1:], v.UnnormalizedStr)
	return buffer
}
