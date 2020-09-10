/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2020 Dapper Labs, Inc.
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

// Import represents an import
//
type Import struct {
	Module string
	Name   string
	// TODO: add support for tables, memories, and globals
	TypeID uint32
}

// importTypeIndicator is the byte used to indicate the import type in the WASM binary
type importTypeIndicator byte

const (
	// importTypeIndicatorFunction is the byte used to indicate the import of a function in the WASM binary
	importTypeIndicatorFunction importTypeIndicator = 0x0
)
