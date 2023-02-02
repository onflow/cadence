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

// wasmMagic is the magic byte sequence that appears at the start of the WASM binary.
//
// See https://webassembly.github.io/spec/core/binary/modules.html#binary-module:
//
// The encoding of a module starts with a preamble containing a 4-byte magic number (the string '\0asm')
//
// magic ::= 0x00 0x61 0x73 0x6d
var wasmMagic = []byte{0x00, 0x61, 0x73, 0x6d}

// wasmVersion is the byte sequence that appears after wasmMagic
// and indicated the version of the WASM binary.
//
// See https://webassembly.github.io/spec/core/binary/modules.html#binary-module:
//
// The encoding of a module starts with [...] a version field.
// The current version of the WebAssembly binary format is 1.
//
// version ::= 0x01 0x00 0x00 0x00
var wasmVersion = []byte{0x01, 0x00, 0x00, 0x00}
