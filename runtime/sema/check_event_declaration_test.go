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
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/onflow/cadence/runtime/common"
)

func TestIsValidEventParameterType(t *testing.T) {

	t.Parallel()

	t.Run("no infinite recursion", func(t *testing.T) {

		t.Parallel()

		ty := &CompositeType{
			Kind:       common.CompositeKindStructure,
			Identifier: "Nested",
		}
		ty.Members = func() *StringMemberOrderedMap {
			members := NewStringMemberOrderedMap()
			// field `nested` refers to the container type,
			// leading to a recursive type declaration
			const fieldName = "nested"
			members.Set(fieldName, NewUnmeteredPublicConstantFieldMember(ty, fieldName, ty, ""))
			return members
		}()

		assert.True(t, IsValidEventParameterType(ty, map[*Member]bool{}))
	})
}
