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

package interpreter_test

import (
	"testing"

	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
	. "github.com/onflow/cadence/test_utils/common_utils"

	"github.com/stretchr/testify/assert"
)

func TestInterpretOptionalType(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let a = OptionalType(Type<String>())
      let b = OptionalType(Type<Int>()) 

      resource R {}
      let c = OptionalType(Type<@R>())
      let d = OptionalType(a)

      let e = Type<String?>()
    `)

	assert.Equal(t,
		interpreter.TypeValue{
			Type: &interpreter.OptionalStaticType{
				Type: interpreter.PrimitiveStaticTypeString,
			},
		},
		inter.Globals.Get("a").GetValue(inter),
	)

	assert.Equal(t,
		interpreter.TypeValue{
			Type: &interpreter.OptionalStaticType{
				Type: interpreter.PrimitiveStaticTypeInt,
			},
		},
		inter.Globals.Get("b").GetValue(inter),
	)

	assert.Equal(t,
		interpreter.TypeValue{
			Type: &interpreter.OptionalStaticType{
				Type: interpreter.NewCompositeStaticTypeComputeTypeID(nil, TestLocation, "R"),
			},
		},
		inter.Globals.Get("c").GetValue(inter),
	)

	assert.Equal(t,
		interpreter.TypeValue{
			Type: &interpreter.OptionalStaticType{
				Type: &interpreter.OptionalStaticType{
					Type: interpreter.PrimitiveStaticTypeString,
				},
			},
		},
		inter.Globals.Get("d").GetValue(inter),
	)

	assert.Equal(t,
		inter.Globals.Get("a").GetValue(inter),
		inter.Globals.Get("e").GetValue(inter),
	)
}

func TestInterpretVariableSizedArrayType(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let a = VariableSizedArrayType(Type<String>())
      let b = VariableSizedArrayType(Type<Int>()) 

      resource R {}
      let c = VariableSizedArrayType(Type<@R>())
      let d = VariableSizedArrayType(a)

      let e = Type<[String]>()
    `)

	assert.Equal(t,
		interpreter.TypeValue{
			Type: &interpreter.VariableSizedStaticType{
				Type: interpreter.PrimitiveStaticTypeString,
			},
		},
		inter.Globals.Get("a").GetValue(inter),
	)

	assert.Equal(t,
		interpreter.TypeValue{
			Type: &interpreter.VariableSizedStaticType{
				Type: interpreter.PrimitiveStaticTypeInt,
			},
		},
		inter.Globals.Get("b").GetValue(inter),
	)

	assert.Equal(t,
		interpreter.TypeValue{
			Type: &interpreter.VariableSizedStaticType{
				Type: interpreter.NewCompositeStaticTypeComputeTypeID(nil, TestLocation, "R"),
			},
		},
		inter.Globals.Get("c").GetValue(inter),
	)

	assert.Equal(t,
		interpreter.TypeValue{
			Type: &interpreter.VariableSizedStaticType{
				Type: &interpreter.VariableSizedStaticType{
					Type: interpreter.PrimitiveStaticTypeString,
				},
			},
		},
		inter.Globals.Get("d").GetValue(inter),
	)
	assert.Equal(t,
		inter.Globals.Get("a").GetValue(inter),
		inter.Globals.Get("e").GetValue(inter),
	)
}

func TestInterpretConstantSizedArrayType(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let a = ConstantSizedArrayType(type: Type<String>(), size: 10)
      let b = ConstantSizedArrayType(type: Type<Int>(), size: 5) 

      resource R {}
      let c = ConstantSizedArrayType(type: Type<@R>(), size: 400)
      let d = ConstantSizedArrayType(type: a, size: 6)

      let e = Type<[String; 10]>()
    `)

	assert.Equal(t,
		interpreter.TypeValue{
			Type: &interpreter.ConstantSizedStaticType{
				Type: interpreter.PrimitiveStaticTypeString,
				Size: int64(10),
			},
		},
		inter.Globals.Get("a").GetValue(inter),
	)

	assert.Equal(t,
		interpreter.TypeValue{
			Type: &interpreter.ConstantSizedStaticType{
				Type: interpreter.PrimitiveStaticTypeInt,
				Size: int64(5),
			},
		},
		inter.Globals.Get("b").GetValue(inter),
	)

	assert.Equal(t,
		interpreter.TypeValue{
			Type: &interpreter.ConstantSizedStaticType{
				Type: interpreter.NewCompositeStaticTypeComputeTypeID(nil, TestLocation, "R"),
				Size: int64(400),
			},
		},
		inter.Globals.Get("c").GetValue(inter),
	)

	assert.Equal(t,
		interpreter.TypeValue{
			Type: &interpreter.ConstantSizedStaticType{
				Type: &interpreter.ConstantSizedStaticType{
					Type: interpreter.PrimitiveStaticTypeString,
					Size: int64(10),
				},
				Size: int64(6),
			},
		},
		inter.Globals.Get("d").GetValue(inter),
	)

	assert.Equal(t,
		inter.Globals.Get("a").GetValue(inter),
		inter.Globals.Get("e").GetValue(inter),
	)
}

func TestInterpretDictionaryType(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let a = DictionaryType(key: Type<String>(), value: Type<Int>())!
      let b = DictionaryType(key: Type<Int>(), value: Type<String>())!

      resource R {}
      let c = DictionaryType(key: Type<Int>(), value: Type<@R>())!
      let d = DictionaryType(key: Type<Bool>(), value: a)!

      let e = Type<{String: Int}>()!
      
      let f = DictionaryType(key: Type<[Bool]>(), value: Type<Int>())
    `)

	assert.Equal(t,
		interpreter.TypeValue{
			Type: &interpreter.DictionaryStaticType{
				KeyType:   interpreter.PrimitiveStaticTypeString,
				ValueType: interpreter.PrimitiveStaticTypeInt,
			},
		},
		inter.Globals.Get("a").GetValue(inter),
	)

	assert.Equal(t,
		interpreter.TypeValue{
			Type: &interpreter.DictionaryStaticType{
				KeyType:   interpreter.PrimitiveStaticTypeInt,
				ValueType: interpreter.PrimitiveStaticTypeString,
			},
		},
		inter.Globals.Get("b").GetValue(inter),
	)

	assert.Equal(t,
		interpreter.TypeValue{
			Type: &interpreter.DictionaryStaticType{
				ValueType: interpreter.NewCompositeStaticTypeComputeTypeID(nil, TestLocation, "R"),
				KeyType:   interpreter.PrimitiveStaticTypeInt,
			},
		},
		inter.Globals.Get("c").GetValue(inter),
	)

	assert.Equal(t,
		interpreter.TypeValue{
			Type: &interpreter.DictionaryStaticType{
				ValueType: &interpreter.DictionaryStaticType{
					KeyType:   interpreter.PrimitiveStaticTypeString,
					ValueType: interpreter.PrimitiveStaticTypeInt,
				},
				KeyType: interpreter.PrimitiveStaticTypeBool,
			},
		},
		inter.Globals.Get("d").GetValue(inter),
	)

	assert.Equal(t,
		inter.Globals.Get("a").GetValue(inter),
		inter.Globals.Get("e").GetValue(inter),
	)

	assert.Equal(t,
		interpreter.Nil,
		inter.Globals.Get("f").GetValue(inter),
	)
}

func TestInterpretCompositeType(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      resource R {}
      struct S {}
      struct interface B {}

      let a = CompositeType("S.test.R")!
      let b = CompositeType("S.test.S")!
      let c = CompositeType("S.test.A")
      let d = CompositeType("S.test.B")

      let e = Type<@R>()

      enum F: UInt8 {}
      let f = CompositeType("S.test.F")!
	  let g = CompositeType("PublicKey")!
	  let h = CompositeType("HashAlgorithm")!
    `)

	assert.Equal(t,
		interpreter.TypeValue{
			Type: interpreter.NewCompositeStaticTypeComputeTypeID(nil, TestLocation, "R"),
		},
		inter.Globals.Get("a").GetValue(inter),
	)

	assert.Equal(t,
		interpreter.TypeValue{
			Type: interpreter.NewCompositeStaticTypeComputeTypeID(nil, TestLocation, "S"),
		},
		inter.Globals.Get("b").GetValue(inter),
	)

	assert.Equal(t,
		interpreter.Nil,
		inter.Globals.Get("c").GetValue(inter),
	)

	assert.Equal(t,
		interpreter.Nil,
		inter.Globals.Get("d").GetValue(inter),
	)

	assert.Equal(t,
		inter.Globals.Get("a").GetValue(inter),
		inter.Globals.Get("e").GetValue(inter),
	)

	assert.Equal(t,
		interpreter.TypeValue{
			Type: interpreter.NewCompositeStaticTypeComputeTypeID(nil, TestLocation, "F"),
		},
		inter.Globals.Get("f").GetValue(inter),
	)

	assert.Equal(t,
		interpreter.TypeValue{
			Type: interpreter.NewCompositeStaticTypeComputeTypeID(nil, nil, "PublicKey"),
		},
		inter.Globals.Get("g").GetValue(inter),
	)

	assert.Equal(t,
		interpreter.TypeValue{
			Type: interpreter.NewCompositeStaticTypeComputeTypeID(nil, nil, "HashAlgorithm"),
		},
		inter.Globals.Get("h").GetValue(inter),
	)
}

func TestInterpretFunctionType(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let a = FunctionType(parameters: [Type<String>()], return: Type<Int>())
      let b = FunctionType(parameters: [Type<String>(), Type<Int>()], return: Type<Bool>())
      let c = FunctionType(parameters: [], return: Type<String>())

      let d = Type<fun(String): Int>();
    `)

	assert.Equal(t,
		interpreter.TypeValue{
			Type: interpreter.FunctionStaticType{
				Type: &sema.FunctionType{
					Parameters: []sema.Parameter{
						{
							TypeAnnotation: sema.StringTypeAnnotation,
						},
					},
					ReturnTypeAnnotation: sema.IntTypeAnnotation,
				},
			},
		},
		inter.Globals.Get("a").GetValue(inter),
	)

	assert.Equal(t,
		interpreter.TypeValue{
			Type: interpreter.FunctionStaticType{
				Type: &sema.FunctionType{
					Parameters: []sema.Parameter{
						{TypeAnnotation: sema.StringTypeAnnotation},
						{TypeAnnotation: sema.IntTypeAnnotation},
					},
					ReturnTypeAnnotation: sema.BoolTypeAnnotation,
				},
			},
		},
		inter.Globals.Get("b").GetValue(inter),
	)

	assert.Equal(t,
		interpreter.TypeValue{
			Type: interpreter.FunctionStaticType{
				Type: &sema.FunctionType{
					ReturnTypeAnnotation: sema.StringTypeAnnotation,
				},
			},
		},
		inter.Globals.Get("c").GetValue(inter),
	)

	assert.Equal(t,
		inter.Globals.Get("a").GetValue(inter),
		inter.Globals.Get("d").GetValue(inter),
	)
}

func TestInterpretReferenceType(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      resource R {}
      struct S {}
	  entitlement X

      let a = ReferenceType(entitlements: ["S.test.X"], type: Type<@R>())!
      let b = ReferenceType(entitlements: [], type: Type<String>())!
      let c = ReferenceType(entitlements: ["S.test.X"], type: Type<S>())!
      let d = Type<auth(X) &R>()
	  let e = ReferenceType(entitlements: ["S.test.Y"], type: Type<S>())
    `)

	assert.Equal(t,
		interpreter.TypeValue{
			Type: &interpreter.ReferenceStaticType{
				ReferencedType: interpreter.NewCompositeStaticTypeComputeTypeID(nil, TestLocation, "R"),
				Authorization: interpreter.NewEntitlementSetAuthorization(
					nil,
					func() []common.TypeID { return []common.TypeID{"S.test.X"} },
					1,
					sema.Conjunction,
				),
			},
		},
		inter.Globals.Get("a").GetValue(inter),
	)

	assert.Equal(t,
		interpreter.TypeValue{
			Type: &interpreter.ReferenceStaticType{
				ReferencedType: interpreter.PrimitiveStaticTypeString,
				Authorization:  interpreter.UnauthorizedAccess,
			},
		},
		inter.Globals.Get("b").GetValue(inter),
	)

	assert.Equal(t,
		interpreter.TypeValue{
			Type: &interpreter.ReferenceStaticType{
				ReferencedType: interpreter.NewCompositeStaticTypeComputeTypeID(nil, TestLocation, "S"),
				Authorization: interpreter.NewEntitlementSetAuthorization(
					nil,
					func() []common.TypeID { return []common.TypeID{"S.test.X"} },
					1,
					sema.Conjunction,
				),
			},
		},
		inter.Globals.Get("c").GetValue(inter),
	)

	assert.Equal(t,
		inter.Globals.Get("a").GetValue(inter),
		inter.Globals.Get("d").GetValue(inter),
	)

	assert.Equal(t,
		interpreter.Nil,
		inter.Globals.Get("e").GetValue(inter),
	)
}

func TestInterpretIntersectionType(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      resource interface R {}
      struct interface S {}
      resource A : R {}
      struct B : S {}

      struct interface S2 {
        access(all) let foo : Int
      }

      let a = IntersectionType(types: ["S.test.R"])!
      let b = IntersectionType(types: ["S.test.S"])!

	  let c = IntersectionType(types: [])

      let f = IntersectionType(types: ["X"])

      let h = Type<@{R}>()
      let i = Type<{S}>()

      let j = IntersectionType(types: ["S.test.R", "S.test.S" ])
      let k = IntersectionType(types: ["S.test.S", "S.test.S2"])!
    `)

	assert.Equal(t,
		interpreter.TypeValue{
			Type: &interpreter.IntersectionStaticType{
				Types: []*interpreter.InterfaceStaticType{
					interpreter.NewInterfaceStaticTypeComputeTypeID(nil, TestLocation, "R"),
				},
			},
		},
		inter.Globals.Get("a").GetValue(inter),
	)

	assert.Equal(t,
		interpreter.Nil,
		inter.Globals.Get("c").GetValue(inter),
	)

	assert.Equal(t,
		interpreter.TypeValue{
			Type: &interpreter.IntersectionStaticType{
				Types: []*interpreter.InterfaceStaticType{
					interpreter.NewInterfaceStaticTypeComputeTypeID(nil, TestLocation, "S"),
				},
			},
		},
		inter.Globals.Get("b").GetValue(inter),
	)

	assert.Equal(t,
		interpreter.Nil,
		inter.Globals.Get("j").GetValue(inter),
	)

	assert.Equal(t,
		interpreter.TypeValue{
			Type: &interpreter.IntersectionStaticType{
				Types: []*interpreter.InterfaceStaticType{
					interpreter.NewInterfaceStaticTypeComputeTypeID(nil, TestLocation, "S"),
					interpreter.NewInterfaceStaticTypeComputeTypeID(nil, TestLocation, "S2"),
				},
			},
		},
		inter.Globals.Get("k").GetValue(inter),
	)

	assert.Equal(t,
		interpreter.Nil,
		inter.Globals.Get("f").GetValue(inter),
	)

	assert.Equal(t,
		inter.Globals.Get("a").GetValue(inter),
		inter.Globals.Get("h").GetValue(inter),
	)

	assert.Equal(t,
		inter.Globals.Get("b").GetValue(inter),
		inter.Globals.Get("i").GetValue(inter),
	)
}

func TestInterpretCapabilityType(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let a = CapabilityType(Type<&String>())!
      let b = CapabilityType(Type<&Int>())!

      resource R {}
      let c = CapabilityType(Type<&R>())!
      let d = CapabilityType(Type<String>())

      let e = Type<Capability<&String>>()
    `)

	assert.Equal(t,
		interpreter.TypeValue{
			Type: &interpreter.CapabilityStaticType{
				BorrowType: &interpreter.ReferenceStaticType{
					ReferencedType: interpreter.PrimitiveStaticTypeString,
					Authorization:  interpreter.UnauthorizedAccess,
				},
			},
		},
		inter.Globals.Get("a").GetValue(inter),
	)

	assert.Equal(t,
		interpreter.TypeValue{
			Type: &interpreter.CapabilityStaticType{
				BorrowType: &interpreter.ReferenceStaticType{
					ReferencedType: interpreter.PrimitiveStaticTypeInt,
					Authorization:  interpreter.UnauthorizedAccess,
				},
			},
		},
		inter.Globals.Get("b").GetValue(inter),
	)

	assert.Equal(t,
		interpreter.TypeValue{
			Type: &interpreter.CapabilityStaticType{
				BorrowType: &interpreter.ReferenceStaticType{
					ReferencedType: interpreter.NewCompositeStaticTypeComputeTypeID(nil, TestLocation, "R"),
					Authorization:  interpreter.UnauthorizedAccess,
				},
			},
		},
		inter.Globals.Get("c").GetValue(inter),
	)

	assert.Equal(t,
		interpreter.Nil,
		inter.Globals.Get("d").GetValue(inter),
	)

	assert.Equal(t,
		inter.Globals.Get("a").GetValue(inter),
		inter.Globals.Get("e").GetValue(inter),
	)
}

func TestInterpretInclusiveRangeType(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
		let a = InclusiveRangeType(Type<Int>())!
		let b = InclusiveRangeType(Type<&Int>())

		resource R {}
		let c = InclusiveRangeType(Type<@R>())
		let d = InclusiveRangeType(Type<String>())

		let e = InclusiveRangeType(Type<Int>())!
	`)

	assert.Equal(t,
		interpreter.TypeValue{
			Type: interpreter.InclusiveRangeStaticType{
				ElementType: interpreter.PrimitiveStaticTypeInt,
			},
		},
		inter.Globals.Get("a").GetValue(inter),
	)

	assert.Equal(t,
		interpreter.Nil,
		inter.Globals.Get("b").GetValue(inter),
	)

	assert.Equal(t,
		interpreter.Nil,
		inter.Globals.Get("c").GetValue(inter),
	)

	assert.Equal(t,
		interpreter.Nil,
		inter.Globals.Get("d").GetValue(inter),
	)

	assert.Equal(t,
		inter.Globals.Get("a").GetValue(inter),
		inter.Globals.Get("e").GetValue(inter),
	)
}
