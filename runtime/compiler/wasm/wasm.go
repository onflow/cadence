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

// WebAssembly (https://webassembly.org/) is an open standard for portable executable programs.
// It is designed to be a portable compilation target for programming languages.
//
// The standard defines two formats for encoding WebAssembly programs ("modules"):
//
// - A machine-optimized binary format (WASM, https://webassembly.github.io/spec/core/binary/index.html),
// which is not designed to be used by humans
//
// - A human-readable text format (WAT, https://webassembly.github.io/spec/core/text/index.html)
//
// WebAssembly modules in either format can be converted into the other format.
//
// There exists also another textual format, WAST, which is a superset of WAT,
// but is not part of the official standard.
//
// Package wasm implements a representation of WebAssembly modules (Module) and related types,
// e.g. instructions (Instruction).
//
// Package wasm also implements a reader and writer for the binary format:
//
// - The reader (WASMReader) allows parsing a WebAssembly module in binary form ([]byte)
// into an representation of the module (Module).
//
// - The writer (WASMWriter) allows encoding the representation of the module (Module)
// to a WebAssembly program in binary form ([]byte).
//
// Package wasm does not currently provide a reader and writer for the textual format (WAT).
//
// Package wasm is not a compiler for Cadence programs, but rather a building block that allows
// reading and writing WebAssembly modules.
//
package wasm
