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

package sema

// HashableStructType represents the type that can be used as a Dictionary key type.
var HashableStructType = &SimpleType{
	Name:          "HashableStruct",
	QualifiedName: "HashableStruct",
	TypeID:        "HashableStruct",
	tag:           HashableStructTypeTag,
	IsResource:    false,
	// The actual storability of a value is checked at run-time
	Storable:   true,
	Equatable:  false,
	Comparable: false,
	Exportable: true,
	// The actual importability is checked at runtime
	Importable: true,
}