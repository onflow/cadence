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

//go:generate go run github.com/cheekybits/genny -pkg=sema -in=../common/orderedmap/orderedmap.go -out=ordered_map_string_type.go gen "KeyType=string ValueType=Type"
//go:generate go run github.com/cheekybits/genny -pkg=sema -in=../common/orderedmap/orderedmap.go -out=ordered_map_string_member.go gen "KeyType=string ValueType=*Member"
//go:generate go run github.com/cheekybits/genny -pkg=sema -in=../common/orderedmap/orderedmap.go -out=ordered_map_interfacetype_struct.go gen "KeyType=*InterfaceType ValueType=struct{}"
//go:generate go run github.com/cheekybits/genny -pkg=sema -in=../common/orderedmap/orderedmap.go -out=ordered_map_member_struct.go gen "KeyType=*Member ValueType=struct{}"
//go:generate go run github.com/cheekybits/genny -pkg=sema -in=../common/orderedmap/orderedmap.go -out=ordered_map_string_variable.go gen "KeyType=string ValueType=*Variable"
//go:generate go run github.com/cheekybits/genny -pkg=sema -in=../common/orderedmap/orderedmap.go -out=ordered_map_string_valuedeclaration.go gen "KeyType=string ValueType=ValueDeclaration"
//go:generate go run github.com/cheekybits/genny -pkg=sema -in=../common/orderedmap/orderedmap.go -out=ordered_map_resourceinvalidation_struct.go gen "KeyType=ResourceInvalidation ValueType=struct{}"
//go:generate go run github.com/cheekybits/genny -pkg=sema -in=../common/orderedmap/orderedmap.go -out=ordered_map_position_resourceuse.go gen "KeyType=ast.Position ValueType=ResourceUse"
//go:generate go run github.com/cheekybits/genny -pkg=sema -in=../common/orderedmap/orderedmap.go -out=ordered_map_interface_resourceinfo.go gen "KeyType=any ValueType=ResourceInfo"
//go:generate go run github.com/cheekybits/genny -pkg=sema -in=../common/orderedmap/orderedmap.go -out=ordered_map_string_importelement.go gen "KeyType=string ValueType=ImportElement"
//go:generate go run github.com/cheekybits/genny -pkg=sema -in=../common/orderedmap/orderedmap.go -out=ordered_map_typeparameter_type.go gen "KeyType=*TypeParameter ValueType=Type"
//go:generate go run github.com/cheekybits/genny -pkg=sema -in=../common/orderedmap/orderedmap.go -out=ordered_map_member_fielddeclaration.go gen "KeyType=*Member ValueType=*ast.FieldDeclaration"
