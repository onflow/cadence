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

import (
	"github.com/onflow/cadence/runtime/common/persistent"
)

type InitializationInfo struct {
	ContainerType           Type
	FieldMembers            *MemberFieldDeclarationOrderedMap
	InitializedFieldMembers *persistent.OrderedSet[*Member]
}

func NewInitializationInfo(
	containerType Type,
	fieldMembers *MemberFieldDeclarationOrderedMap,
) *InitializationInfo {
	return &InitializationInfo{
		ContainerType:           containerType,
		FieldMembers:            fieldMembers,
		InitializedFieldMembers: persistent.NewOrderedSet[*Member](nil),
	}
}

// InitializationComplete returns true if all fields of the container
// were initialized, false if some fields are uninitialized
func (info *InitializationInfo) InitializationComplete() bool {
	for pair := info.FieldMembers.Oldest(); pair != nil; pair = pair.Next() {
		member := pair.Key

		if !info.InitializedFieldMembers.Contains(member) {
			return false
		}
	}

	return true
}
