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

// PathType
var PathType = &SimpleType{
	Name:          "Path",
	QualifiedName: "Path",
	TypeID:        "Path",
	TypeTag:       PathTypeTag,
	IsResource:    false,
	IsPrimitive:   true,
	Storable:      true,
	Equatable:     true,
	Comparable:    false,
	Exportable:    true,
	Importable:    true,
}

var PathTypeAnnotation = NewTypeAnnotation(PathType)

// StoragePathType
var StoragePathType = &SimpleType{
	Name:          "StoragePath",
	QualifiedName: "StoragePath",
	TypeID:        "StoragePath",
	TypeTag:       StoragePathTypeTag,
	IsResource:    false,
	IsPrimitive:   true,
	Storable:      true,
	Equatable:     true,
	Comparable:    false,
	Exportable:    true,
	Importable:    true,
}

var StoragePathTypeAnnotation = NewTypeAnnotation(StoragePathType)

// CapabilityPathType
var CapabilityPathType = &SimpleType{
	Name:          "CapabilityPath",
	QualifiedName: "CapabilityPath",
	TypeID:        "CapabilityPath",
	TypeTag:       CapabilityPathTypeTag,
	IsResource:    false,
	IsPrimitive:   true,
	Storable:      true,
	Equatable:     true,
	Comparable:    false,
	Exportable:    true,
	Importable:    true,
}

var CapabilityPathTypeAnnotation = NewTypeAnnotation(CapabilityPathType)

// PublicPathType
var PublicPathType = &SimpleType{
	Name:          "PublicPath",
	QualifiedName: "PublicPath",
	TypeID:        "PublicPath",
	TypeTag:       PublicPathTypeTag,
	IsResource:    false,
	IsPrimitive:   true,
	Storable:      true,
	Equatable:     true,
	Comparable:    false,
	Exportable:    true,
	Importable:    true,
}

var PublicPathTypeAnnotation = NewTypeAnnotation(PublicPathType)

// PrivatePathType
var PrivatePathType = &SimpleType{
	Name:          "PrivatePath",
	QualifiedName: "PrivatePath",
	TypeID:        "PrivatePath",
	TypeTag:       PrivatePathTypeTag,
	IsResource:    false,
	IsPrimitive:   true,
	Storable:      true,
	Equatable:     true,
	Comparable:    false,
	Exportable:    true,
	Importable:    true,
}

var PrivatePathTypeAnnotation = NewTypeAnnotation(PrivatePathType)
