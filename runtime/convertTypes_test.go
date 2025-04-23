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

package runtime_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/common"
	. "github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/sema"
	. "github.com/onflow/cadence/test_utils/common_utils"
)

func TestRuntimeExportRecursiveType(t *testing.T) {

	t.Parallel()

	ty := &sema.CompositeType{
		Location:   TestLocation,
		Identifier: "Foo",
		Kind:       common.CompositeKindResource,
		Members:    &sema.StringMemberOrderedMap{},
		Fields:     []string{"foo"},
	}

	ty.Members.Set("foo", &sema.Member{
		ContainerType: ty,
		Access:        sema.PrimitiveAccess(ast.AccessNotSpecified),
		Identifier:    ast.Identifier{Identifier: "foo"},
		// NOTE: recursive type
		TypeAnnotation:  sema.NewTypeAnnotation(ty),
		DeclarationKind: common.DeclarationKindField,
		VariableKind:    ast.VariableKindVariable,
	})

	fields := []cadence.Field{
		{
			Identifier: "foo",
		},
	}

	expected := cadence.NewResourceType(
		TestLocation,
		"Foo",
		fields,
		nil,
	)

	// NOTE: recursion should be kept
	fields[0].Type = expected

	assert.Equal(t,
		expected,
		ExportType(ty, map[sema.TypeID]cadence.Type{}),
	)
}

func BenchmarkExportType(b *testing.B) {

	b.Run("simple type", func(b *testing.B) {
		ty := sema.StringType

		exportedType := ExportType(ty, map[sema.TypeID]cadence.Type{})
		assert.Equal(b, cadence.StringType, exportedType)

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			ExportType(ty, map[sema.TypeID]cadence.Type{})
		}
	})

	b.Run("composite type", func(b *testing.B) {

		ty := &sema.CompositeType{
			Location:   TestLocation,
			Identifier: "Foo",
			Kind:       common.CompositeKindResource,
			Members:    &sema.StringMemberOrderedMap{},
			Fields:     []string{"foo"},
		}

		ty.Members.Set("foo", &sema.Member{
			ContainerType: ty,
			Access:        sema.PrimitiveAccess(ast.AccessNotSpecified),
			Identifier:    ast.Identifier{Identifier: "foo"},
			// NOTE: recursive type
			TypeAnnotation:  sema.NewTypeAnnotation(ty),
			DeclarationKind: common.DeclarationKindField,
			VariableKind:    ast.VariableKindVariable,
		})

		fields := []cadence.Field{
			{
				Identifier: "foo",
			},
		}

		expected := cadence.NewResourceType(
			TestLocation,
			"Foo",
			fields,
			nil,
		)

		// NOTE: recursion should be kept
		fields[0].Type = expected

		exportedType := ExportType(ty, map[sema.TypeID]cadence.Type{})
		assert.Equal(b, expected, exportedType)

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			ExportType(ty, map[sema.TypeID]cadence.Type{})
		}
	})
}
