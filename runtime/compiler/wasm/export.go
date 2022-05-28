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

package wasm

// Exports represents an export
//
type Export struct {
	Name       string
	Descriptor ExportDescriptor
}

// exportIndicator is the byte used to indicate the kind of export in the WASM binary
type exportIndicator byte

const (
	// exportIndicatorFunction is the byte used to indicate the export of a function in the WASM binary
	exportIndicatorFunction exportIndicator = 0x0
	// exportIndicatorMemory is the byte used to indicate the export of a memory in the WASM binary
	exportIndicatorMemory exportIndicator = 0x2
)

// ExportDescriptor represents an export (e.g. a function, memory, etc.)
//
type ExportDescriptor interface {
	isExportDescriptor()
}

// FunctionExport represents the export of a function
//
type FunctionExport struct {
	FunctionIndex uint32
}

func (FunctionExport) isExportDescriptor() {}

// MemoryExport represents the export of a memory
//
type MemoryExport struct {
	MemoryIndex uint32
}

func (MemoryExport) isExportDescriptor() {}
