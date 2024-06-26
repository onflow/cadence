/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Flow Foundation
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

// AnyStructType represents the top type of all non-resource types
var AnyStructType = &SimpleType{
	Name:          "AnyStruct",
	QualifiedName: "AnyStruct",
	TypeID:        "AnyStruct",
	TypeTag:       AnyStructTypeTag,
	IsResource:    false,
	// The actual storability of a value is checked at run-time
	Storable:   true,
	Primitive:  false,
	Equatable:  false,
	Comparable: false,
	Exportable: true,
	// The actual importability is checked at runtime
	Importable:    true,
	ContainFields: true,
}

var AnyStructTypeAnnotation = NewTypeAnnotation(AnyStructType)
