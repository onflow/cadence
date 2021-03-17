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

package sema

// PathType
//
var PathType = &SimpleType{
	Name:          "Path",
	QualifiedName: "Path",
	TypeID:        "Path",
	IsInvalid:     false,
	IsResource:    false,
	Storable:      true,
	// TODO: implement support for equating paths in the future
	Equatable:            false,
	ExternallyReturnable: true,
	IsSuperTypeOf: func(subType Type) bool {
		return IsSubType(subType, StoragePathType) ||
			IsSubType(subType, CapabilityPathType)
	},
}

// StoragePathType
//
var StoragePathType = &SimpleType{
	Name:          "StoragePath",
	QualifiedName: "StoragePath",
	TypeID:        "StoragePath",
	IsResource:    false,
	Storable:      true,
	// TODO: implement support for equating paths in the future
	Equatable:            false,
	ExternallyReturnable: true,
}

// CapabilityPathType
//
var CapabilityPathType = &SimpleType{
	Name:          "CapabilityPath",
	QualifiedName: "CapabilityPath",
	TypeID:        "CapabilityPath",
	IsResource:    false,
	Storable:      true,
	// TODO: implement support for equating paths in the future
	Equatable:            false,
	ExternallyReturnable: true,
	IsSuperTypeOf: func(subType Type) bool {
		return IsSubType(subType, PrivatePathType) ||
			IsSubType(subType, PublicPathType)
	},
}

// PublicPathType
//
var PublicPathType = &SimpleType{
	Name:          "PublicPath",
	QualifiedName: "PublicPath",
	TypeID:        "PublicPath",
	IsResource:    false,
	Storable:      true,
	// TODO: implement support for equating paths in the future
	Equatable:            false,
	ExternallyReturnable: true,
}

// PrivatePathType
//
var PrivatePathType = &SimpleType{
	Name:          "PrivatePath",
	QualifiedName: "PrivatePath",
	TypeID:        "PrivatePath",
	IsResource:    false,
	Storable:      true,
	// TODO: implement support for equating paths in the future
	Equatable:            false,
	ExternallyReturnable: true,
}
