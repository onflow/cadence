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

// sectionID is the ID of a section in the WASM binary.
//
// See https://webassembly.github.io/spec/core/binary/modules.html#sections:
//
// The following section ids are used:
//
// 0 = custom section
// 1 = type section
// 2 = import section
// 3 = function section
// 4 = table section
// 5 = memory section
// 6 = global section
// 7 = export section
// 8 = start section
// 9 = element section
// 10 = code section
// 11 = data section
//
type sectionID byte

const (
	sectionIDType     sectionID = 1
	sectionIDImport   sectionID = 2
	sectionIDFunction sectionID = 3
	sectionIDCode     sectionID = 10
)
