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

package sema

// AnyType represents the top type of all types.
// NOTE: This type is only used internally and is not available in programs.
var AnyType = &SimpleType{
	Name:          "Any",
	QualifiedName: "Any",
	TypeID:        "Any",
	tag:           AnyTypeTag,
	IsResource:    false,
	// `Any` is never a valid type in user programs
	Storable:  true,
	Equatable: false,
	// `Any` is never a valid type in user programs
	ExternallyReturnable: false,
	Importable:           false,
}
