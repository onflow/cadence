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
 *
 * Based on https://github.com/wk8/go-ordered-map, Copyright Jean Rougé
 *
 */

package orderedmap

//go:generate go run github.com/cheekybits/genny -pkg=orderedmap -in=orderedmap.go -out=orderedmap_string_string.go gen "KeyType=string ValueType=string"
//go:generate go run github.com/cheekybits/genny -pkg=orderedmap -in=orderedmap.go -out=orderedmap_string_interface.go gen "KeyType=string ValueType=interface{}"
//go:generate go run github.com/cheekybits/genny -pkg=orderedmap -in=orderedmap.go -out=orderedmap_string_struct.go gen "KeyType=string ValueType=struct{}"
