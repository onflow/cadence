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

func TestOptionalSubtyping(t *testing.T) {

	t.Parallel()

	t.Run("Int? <: Int?", func(t *testing.T) {
		assert.True(t,
			IsSubType(
				&OptionalType{Type: IntType},
				&OptionalType{Type: IntType},
			),
		)
	})

	t.Run("Int? <: Bool?", func(t *testing.T) {
		assert.False(t,
			IsSubType(
				&OptionalType{Type: IntType},
				&OptionalType{Type: BoolType},
			),
		)
	})

	t.Run("Int8? <: Integer?", func(t *testing.T) {
		assert.True(t,
			IsSubType(
				&OptionalType{Type: Int8Type},
				&OptionalType{Type: IntegerType},
			),
		)
	})
}

func TestCompositeType_ID(t *testing.T) {

	t.Parallel()

	location := common.StringLocation("x")

	t.Run("composite in composite", func(t *testing.T) {

		compositeInComposite :=
			&CompositeType{
				Location:   location,
				Identifier: "C",
				containerType: &CompositeType{
					Location:   location,
					Identifier: "B",
					containerType: &CompositeType{
						Location:   location,
						Identifier: "A",
					},
				},
			}

		assert.Equal(t,
			TypeID(location.ID()+".A.B.C"),
			compositeInComposite.ID(),
		)
	})

	t.Run("composite in interface", func(t *testing.T) {

		compositeInInterface :=
			&CompositeType{
				Location:   location,
				Identifier: "C",
				containerType: &InterfaceType{
					Location:   location,
					Identifier: "B",
					containerType: &CompositeType{
						Location:   location,
						Identifier: "A",
					},
				},
			}

		assert.Equal(t,
			TypeID(location.ID()+".A.B.C"),
			compositeInInterface.ID(),
		)
	})
}

func TestInterfaceType_ID(t *testing.T) {

	t.Parallel()

	location := common.StringLocation("x")

	t.Run("interface in composite", func(t *testing.T) {

		interfaceInComposite :=
			&InterfaceType{
				Location:   location,
				Identifier: "C",
				containerType: &CompositeType{
					Location:   location,
					Identifier: "B",
					containerType: &CompositeType{
						Location:   location,
						Identifier: "A",
					},
				},
			}

		assert.Equal(t,
			TypeID(location.ID()+".A.B.C"),
			interfaceInComposite.ID(),
		)
	})

	t.Run("interface in interface", func(t *testing.T) {

		interfaceInInterface :=
			&InterfaceType{
				Location:   location,
				Identifier: "C",
				containerType: &InterfaceType{
					Location:   location,
					Identifier: "B",
					containerType: &CompositeType{
						Location:   location,
						Identifier: "A",
					},
				},
			}

		assert.Equal(t,
			TypeID(location.ID()+".A.B.C"),
			interfaceInInterface.ID(),
		)
	})
}

func TestFunctionSubtyping(t *testing.T) {

	t.Parallel()

	t.Run("((Int): Void) <: ((AnyStruct): Void)", func(t *testing.T) {
		assert.False(t,
			IsSubType(
				&FunctionType{
					Parameters: []*Parameter{
						{
							TypeAnnotation: NewTypeAnnotation(IntType),
						},
					},
					ReturnTypeAnnotation: NewTypeAnnotation(
						VoidType,
					),
				},
				&FunctionType{
					Parameters: []*Parameter{
						{
							TypeAnnotation: NewTypeAnnotation(AnyStructType),
						},
					},
					ReturnTypeAnnotation: NewTypeAnnotation(
						VoidType,
					),
				},
			),
		)
	})

	t.Run("((AnyStruct): Void) <: ((Int): Void)", func(t *testing.T) {
		assert.True(t,
			IsSubType(
				&FunctionType{
					Parameters: []*Parameter{
						{
							TypeAnnotation: NewTypeAnnotation(AnyStructType),
						},
					},
					ReturnTypeAnnotation: NewTypeAnnotation(
						VoidType,
					),
				},
				&FunctionType{
					Parameters: []*Parameter{
						{
							TypeAnnotation: NewTypeAnnotation(IntType),
						},
					},
					ReturnTypeAnnotation: NewTypeAnnotation(
						VoidType,
					),
				},
			),
		)
	})

	t.Run("((): Int) <: ((): AnyStruct)", func(t *testing.T) {
		assert.True(t,
			IsSubType(
				&FunctionType{
					ReturnTypeAnnotation: NewTypeAnnotation(IntType),
				},
				&FunctionType{
					ReturnTypeAnnotation: NewTypeAnnotation(AnyStructType),
				},
			),
		)
	})

	t.Run("((): Any) <: ((): Int)", func(t *testing.T) {
		assert.False(t,
			IsSubType(
				&FunctionType{
					ReturnTypeAnnotation: NewTypeAnnotation(AnyStructType),
				},
				&FunctionType{
					ReturnTypeAnnotation: NewTypeAnnotation(IntType),
				},
			),
		)
	})

	t.Run("constructor != non-constructor", func(t *testing.T) {
		assert.False(t,
			IsSubType(
				&FunctionType{
					IsConstructor:        false,
					ReturnTypeAnnotation: NewTypeAnnotation(VoidType),
				},
				&FunctionType{
					IsConstructor:        true,
					ReturnTypeAnnotation: NewTypeAnnotation(VoidType),
				},
			),
		)
	})

	t.Run("different receiver types", func(t *testing.T) {
		// Receiver shouldn't matter
		assert.True(t,
			IsSubType(
				&FunctionType{
					ReturnTypeAnnotation: NewTypeAnnotation(VoidType),
				},
				&FunctionType{
					ReturnTypeAnnotation: NewTypeAnnotation(VoidType),
				},
			),
		)
	})
}
