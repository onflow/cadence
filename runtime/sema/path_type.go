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
	tag:           PathTypeTag,
	IsResource:    false,
	Storable:      true,
	Equatable:     true,
	Comparable:    false,
	Exportable:    true,
	Importable:    true,
	IsSuperTypeOf: func(subType Type) bool {
		return IsSubType(subType, StoragePathType) ||
			IsSubType(subType, CapabilityPathType)
	},
}

var PathTypeAnnotation = NewTypeAnnotation(PathType)

// StoragePathType
var StoragePathType = &SimpleType{
	Name:          "StoragePath",
	QualifiedName: "StoragePath",
	TypeID:        "StoragePath",
	tag:           StoragePathTypeTag,
	IsResource:    false,
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
	tag:           CapabilityPathTypeTag,
	IsResource:    false,
	Storable:      true,
	Equatable:     true,
	Comparable:    false,
	Exportable:    true,
	Importable:    true,
	IsSuperTypeOf: func(subType Type) bool {
		return IsSubType(subType, PrivatePathType) ||
			IsSubType(subType, PublicPathType)
	},
}

var CapabilityPathTypeAnnotation = NewTypeAnnotation(CapabilityPathType)

// PublicPathType
var PublicPathType = &SimpleType{
	Name:          "PublicPath",
	QualifiedName: "PublicPath",
	TypeID:        "PublicPath",
	tag:           PublicPathTypeTag,
	IsResource:    false,
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
	tag:           PrivatePathTypeTag,
	IsResource:    false,
	Storable:      true,
	Equatable:     true,
	Comparable:    false,
	Exportable:    true,
	Importable:    true,
}

var PrivatePathTypeAnnotation = NewTypeAnnotation(PrivatePathType)
