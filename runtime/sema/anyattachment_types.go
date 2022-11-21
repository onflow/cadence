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

const AnyResourceAttachmentTypeName = "AnyResourceAttachment"

// AnyResourceAttachmentType represents the top type of all resource attachment types
var AnyResourceAttachmentType = &SimpleType{
	Name:          AnyResourceAttachmentTypeName,
	QualifiedName: AnyResourceAttachmentTypeName,
	TypeID:        AnyResourceAttachmentTypeName,
	tag:           AnyResourceAttachmentTypeTag,
	IsResource:    true,
	// The actual storability of a value is checked at run-time
	Storable:  true,
	Equatable: false,
	// The actual returnability of a value is checked at run-time
	Exportable: true,
	Importable: false,
}

const AnyStructAttachmentTypeName = "AnyStructAttachment"

// AnyStructAttachmentType represents the top type of all struct attachment types
var AnyStructAttachmentType = &SimpleType{
	Name:          AnyStructAttachmentTypeName,
	QualifiedName: AnyStructAttachmentTypeName,
	TypeID:        AnyStructAttachmentTypeName,
	tag:           AnyStructAttachmentTypeTag,
	IsResource:    false,
	// The actual storability of a value is checked at run-time
	Storable:  true,
	Equatable: false,
	// The actual returnability of a value is checked at run-time
	Exportable: true,
	Importable: false,
}
