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

package wasm

// MemoryPageSize is the size of a memory page: 64KiB
const MemoryPageSize = 64 * 1024

// Memory represents a memory
type Memory struct {
	// maximum number of pages (each one is 64KiB in size). optional, unlimited if nil
	Max *uint32
	// minimum number of pages (each one is 64KiB in size)
	Min uint32
}

// limitIndicator is the byte used to indicate the kind of limit in the WASM binary
type limitIndicator byte

const (
	// limitIndicatorNoMax is the byte used to indicate a limit with no maximum in the WASM binary
	limitIndicatorNoMax limitIndicator = 0x0
	// limitIndicatorMax is the byte used to indicate a limit with no maximum in the WASM binary
	limitIndicatorMax limitIndicator = 0x1
)
