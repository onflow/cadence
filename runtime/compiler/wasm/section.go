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

// sectionID is the ID of a section in the WASM binary
//
type sectionID byte

const (
	sectionIDCustom   sectionID = 0
	sectionIDType     sectionID = 1
	sectionIDImport   sectionID = 2
	sectionIDFunction sectionID = 3
	sectionIDTable    sectionID = 4
	sectionIDMemory   sectionID = 5
	sectionIDGlobal   sectionID = 6
	sectionIDExport   sectionID = 7
	sectionIDStart    sectionID = 8
	sectionIDElement  sectionID = 9
	sectionIDCode     sectionID = 10
	sectionIDData     sectionID = 11
)
