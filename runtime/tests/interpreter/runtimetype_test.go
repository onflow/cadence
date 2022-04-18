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

package interpreter_test

import (
	"testing"

	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/tests/utils"

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
			Type: interpreter.OptionalStaticType{
				Type: interpreter.PrimitiveStaticTypeString,
			},
		},
		inter.Globals["a"].GetValue(),
	)

	assert.Equal(t,
		interpreter.TypeValue{
			Type: interpreter.OptionalStaticType{
				Type: interpreter.PrimitiveStaticTypeInt,
			},
		},
		inter.Globals["b"].GetValue(),
	)

	assert.Equal(t,
		interpreter.TypeValue{
			Type: interpreter.OptionalStaticType{
				Type: interpreter.CompositeStaticType{
					Location:            utils.TestLocation,
					QualifiedIdentifier: "R",
					TypeID:              "S.test.R",
				},
			},
		},
		inter.Globals["c"].GetValue(),
	)

	assert.Equal(t,
		interpreter.TypeValue{
			Type: interpreter.OptionalStaticType{
				Type: interpreter.OptionalStaticType{
					Type: interpreter.PrimitiveStaticTypeString,
				},
			},
		},
		inter.Globals["d"].GetValue(),
	)

	assert.Equal(t,
		inter.Globals["a"].GetValue(),
		inter.Globals["e"].GetValue(),
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
			Type: interpreter.VariableSizedStaticType{
				Type: interpreter.PrimitiveStaticTypeString,
			},
		},
		inter.Globals["a"].GetValue(),
	)

	assert.Equal(t,
		interpreter.TypeValue{
			Type: interpreter.VariableSizedStaticType{
				Type: interpreter.PrimitiveStaticTypeInt,
			},
		},
		inter.Globals["b"].GetValue(),
	)

	assert.Equal(t,
		interpreter.TypeValue{
			Type: interpreter.VariableSizedStaticType{
				Type: interpreter.CompositeStaticType{
					Location:            utils.TestLocation,
					QualifiedIdentifier: "R",
					TypeID:              "S.test.R",
				},
			},
		},
		inter.Globals["c"].GetValue(),
	)

	assert.Equal(t,
		interpreter.TypeValue{
			Type: interpreter.VariableSizedStaticType{
				Type: interpreter.VariableSizedStaticType{
					Type: interpreter.PrimitiveStaticTypeString,
				},
			},
		},
		inter.Globals["d"].GetValue(),
	)
	assert.Equal(t,
		inter.Globals["a"].GetValue(),
		inter.Globals["e"].GetValue(),
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
			Type: interpreter.ConstantSizedStaticType{
				Type: interpreter.PrimitiveStaticTypeString,
				Size: int64(10),
			},
		},
		inter.Globals["a"].GetValue(),
	)

	assert.Equal(t,
		interpreter.TypeValue{
			Type: interpreter.ConstantSizedStaticType{
				Type: interpreter.PrimitiveStaticTypeInt,
				Size: int64(5),
			},
		},
		inter.Globals["b"].GetValue(),
	)

	assert.Equal(t,
		interpreter.TypeValue{
			Type: interpreter.ConstantSizedStaticType{
				Type: interpreter.CompositeStaticType{
					Location:            utils.TestLocation,
					QualifiedIdentifier: "R",
					TypeID:              "S.test.R",
				},
				Size: int64(400),
			},
		},
		inter.Globals["c"].GetValue(),
	)

	assert.Equal(t,
		interpreter.TypeValue{
			Type: interpreter.ConstantSizedStaticType{
				Type: interpreter.ConstantSizedStaticType{
					Type: interpreter.PrimitiveStaticTypeString,
					Size: int64(10),
				},
				Size: int64(6),
			},
		},
		inter.Globals["d"].GetValue(),
	)

	assert.Equal(t,
		inter.Globals["a"].GetValue(),
		inter.Globals["e"].GetValue(),
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
			Type: interpreter.DictionaryStaticType{
				KeyType:   interpreter.PrimitiveStaticTypeString,
				ValueType: interpreter.PrimitiveStaticTypeInt,
			},
		},
		inter.Globals["a"].GetValue(),
	)

	assert.Equal(t,
		interpreter.TypeValue{
			Type: interpreter.DictionaryStaticType{
				KeyType:   interpreter.PrimitiveStaticTypeInt,
				ValueType: interpreter.PrimitiveStaticTypeString,
			},
		},
		inter.Globals["b"].GetValue(),
	)

	assert.Equal(t,
		interpreter.TypeValue{
			Type: interpreter.DictionaryStaticType{
				ValueType: interpreter.CompositeStaticType{
					Location:            utils.TestLocation,
					QualifiedIdentifier: "R",
					TypeID:              "S.test.R",
				},
				KeyType: interpreter.PrimitiveStaticTypeInt,
			},
		},
		inter.Globals["c"].GetValue(),
	)

	assert.Equal(t,
		interpreter.TypeValue{
			Type: interpreter.DictionaryStaticType{
				ValueType: interpreter.DictionaryStaticType{
					KeyType:   interpreter.PrimitiveStaticTypeString,
					ValueType: interpreter.PrimitiveStaticTypeInt,
				},
				KeyType: interpreter.PrimitiveStaticTypeBool,
			},
		},
		inter.Globals["d"].GetValue(),
	)

	assert.Equal(t,
		inter.Globals["a"].GetValue(),
		inter.Globals["e"].GetValue(),
	)

	assert.Equal(t,
		interpreter.NilValue{},
		inter.Globals["f"].GetValue(),
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
			Type: interpreter.CompositeStaticType{
				QualifiedIdentifier: "R",
				Location:            utils.TestLocation,
				TypeID:              "S.test.R",
			},
		},
		inter.Globals["a"].GetValue(),
	)

	assert.Equal(t,
		interpreter.TypeValue{
			Type: interpreter.CompositeStaticType{
				QualifiedIdentifier: "S",
				Location:            utils.TestLocation,
				TypeID:              "S.test.S",
			},
		},
		inter.Globals["b"].GetValue(),
	)

	assert.Equal(t,
		interpreter.NilValue{},
		inter.Globals["c"].GetValue(),
	)

	assert.Equal(t,
		interpreter.NilValue{},
		inter.Globals["d"].GetValue(),
	)

	assert.Equal(t,
		inter.Globals["a"].GetValue(),
		inter.Globals["e"].GetValue(),
	)

	assert.Equal(t,
		interpreter.TypeValue{
			Type: interpreter.CompositeStaticType{
				QualifiedIdentifier: "F",
				Location:            utils.TestLocation,
				TypeID:              "S.test.F",
			},
		},
		inter.Globals["f"].GetValue(),
	)

	assert.Equal(t,
		interpreter.TypeValue{
			Type: interpreter.CompositeStaticType{
				QualifiedIdentifier: "PublicKey",
				Location:            nil,
				TypeID:              "PublicKey",
			},
		},
		inter.Globals["g"].GetValue(),
	)

	assert.Equal(t,
		interpreter.TypeValue{
			Type: interpreter.CompositeStaticType{
				QualifiedIdentifier: "HashAlgorithm",
				Location:            nil,
				TypeID:              "HashAlgorithm",
			},
		},
		inter.Globals["h"].GetValue(),
	)
}

func TestInterpretInterfaceType(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      resource interface R {}
      struct interface S {}
      struct B {}

      let a = InterfaceType("S.test.R")!
      let b = InterfaceType("S.test.S")!
      let c = InterfaceType("S.test.A")
      let d = InterfaceType("S.test.B")
    `)

	assert.Equal(t,
		interpreter.TypeValue{
			Type: interpreter.InterfaceStaticType{
				QualifiedIdentifier: "R",
				Location:            utils.TestLocation,
			},
		},
		inter.Globals["a"].GetValue(),
	)

	assert.Equal(t,
		interpreter.TypeValue{
			Type: interpreter.InterfaceStaticType{
				QualifiedIdentifier: "S",
				Location:            utils.TestLocation,
			},
		},
		inter.Globals["b"].GetValue(),
	)

	assert.Equal(t,
		interpreter.NilValue{},
		inter.Globals["c"].GetValue(),
	)

	assert.Equal(t,
		interpreter.NilValue{},
		inter.Globals["d"].GetValue(),
	)
}

func TestInterpretFunctionType(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let a = FunctionType(parameters: [Type<String>()], return: Type<Int>())
      let b = FunctionType(parameters: [Type<String>(), Type<Int>()], return: Type<Bool>())
      let c = FunctionType(parameters: [], return: Type<String>())

      let d = Type<((String): Int)>();
    `)

	assert.Equal(t,
		interpreter.TypeValue{
			Type: interpreter.FunctionStaticType{
				Type: &sema.FunctionType{
					Parameters:           []*sema.Parameter{{TypeAnnotation: &sema.TypeAnnotation{Type: sema.StringType}}},
					ReturnTypeAnnotation: &sema.TypeAnnotation{Type: sema.IntType},
				},
			},
		},
		inter.Globals["a"].GetValue(),
	)

	assert.Equal(t,
		interpreter.TypeValue{
			Type: interpreter.FunctionStaticType{
				Type: &sema.FunctionType{
					Parameters: []*sema.Parameter{
						{TypeAnnotation: &sema.TypeAnnotation{Type: sema.StringType}},
						{TypeAnnotation: &sema.TypeAnnotation{Type: sema.IntType}},
					},
					ReturnTypeAnnotation: &sema.TypeAnnotation{Type: sema.BoolType},
				},
			},
		},
		inter.Globals["b"].GetValue(),
	)

	assert.Equal(t,
		interpreter.TypeValue{
			Type: interpreter.FunctionStaticType{
				Type: &sema.FunctionType{
					Parameters:           []*sema.Parameter{},
					ReturnTypeAnnotation: &sema.TypeAnnotation{Type: sema.StringType},
				},
			},
		},
		inter.Globals["c"].GetValue(),
	)

	assert.Equal(t,
		inter.Globals["a"].GetValue(),
		inter.Globals["d"].GetValue(),
	)
}

func TestInterpretReferenceType(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      resource R {}
      struct S {}

      let a = ReferenceType(authorized: true, type: Type<@R>())
      let b = ReferenceType(authorized: false, type: Type<String>())
      let c = ReferenceType(authorized: true, type: Type<S>()) 
      let d = Type<auth &R>()
    `)

	assert.Equal(t,
		interpreter.TypeValue{
			Type: interpreter.ReferenceStaticType{
				BorrowedType: interpreter.CompositeStaticType{
					QualifiedIdentifier: "R",
					Location:            utils.TestLocation,
					TypeID:              "S.test.R",
				},
				Authorized: true,
			},
		},
		inter.Globals["a"].GetValue(),
	)

	assert.Equal(t,
		interpreter.TypeValue{
			Type: interpreter.ReferenceStaticType{
				BorrowedType: interpreter.PrimitiveStaticTypeString,
				Authorized:   false,
			},
		},
		inter.Globals["b"].GetValue(),
	)

	assert.Equal(t,
		interpreter.TypeValue{
			Type: interpreter.ReferenceStaticType{
				BorrowedType: interpreter.CompositeStaticType{
					QualifiedIdentifier: "S",
					Location:            utils.TestLocation,
					TypeID:              "S.test.S",
				},
				Authorized: true,
			},
		},
		inter.Globals["c"].GetValue(),
	)

	assert.Equal(t,
		inter.Globals["a"].GetValue(),
		inter.Globals["d"].GetValue(),
	)
}

func TestInterpretRestrictedType(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      resource interface R {}
      struct interface S {}
      resource A : R {}
      struct B : S {}

      struct interface S2 {
        pub let foo : Int
      }

      let a = RestrictedType(identifier: "S.test.A", restrictions: ["S.test.R"])!
      let b = RestrictedType(identifier: "S.test.B", restrictions: ["S.test.S"])!

      let c = RestrictedType(identifier: "S.test.B", restrictions: ["S.test.R"])
      let d = RestrictedType(identifier: "S.test.A", restrictions: ["S.test.S"])
      let e = RestrictedType(identifier: "S.test.B", restrictions: ["S.test.S2"])

      let f = RestrictedType(identifier: "S.test.B", restrictions: ["X"])
      let g = RestrictedType(identifier: "S.test.N", restrictions: ["S.test.S2"])

      let h = Type<@A{R}>()
      let i = Type<B{S}>()

      let j = RestrictedType(identifier: nil, restrictions: ["S.test.R"])!
      let k = RestrictedType(identifier: nil, restrictions: ["S.test.S"])!
    `)

	assert.Equal(t,
		interpreter.TypeValue{
			Type: &interpreter.RestrictedStaticType{
				Type: interpreter.CompositeStaticType{
					QualifiedIdentifier: "A",
					Location:            utils.TestLocation,
					TypeID:              "S.test.A",
				},
				Restrictions: []interpreter.InterfaceStaticType{
					{
						QualifiedIdentifier: "R",
						Location:            utils.TestLocation,
					},
				},
			},
		},
		inter.Globals["a"].GetValue(),
	)

	assert.Equal(t,
		interpreter.TypeValue{
			Type: &interpreter.RestrictedStaticType{
				Type: interpreter.CompositeStaticType{
					QualifiedIdentifier: "B",
					Location:            utils.TestLocation,
					TypeID:              "S.test.B",
				},
				Restrictions: []interpreter.InterfaceStaticType{
					{
						QualifiedIdentifier: "S",
						Location:            utils.TestLocation,
					},
				},
			},
		},
		inter.Globals["b"].GetValue(),
	)

	assert.Equal(t,
		interpreter.TypeValue{
			Type: &interpreter.RestrictedStaticType{
				Type: interpreter.PrimitiveStaticTypeAnyResource,
				Restrictions: []interpreter.InterfaceStaticType{
					{
						QualifiedIdentifier: "R",
						Location:            utils.TestLocation,
					},
				},
			},
		},
		inter.Globals["j"].GetValue(),
	)

	assert.Equal(t,
		interpreter.TypeValue{
			Type: &interpreter.RestrictedStaticType{
				Type: interpreter.PrimitiveStaticTypeAnyStruct,
				Restrictions: []interpreter.InterfaceStaticType{
					{
						QualifiedIdentifier: "S",
						Location:            utils.TestLocation,
					},
				},
			},
		},
		inter.Globals["k"].GetValue(),
	)

	assert.Equal(t,
		interpreter.NilValue{},
		inter.Globals["c"].GetValue(),
	)

	assert.Equal(t,
		interpreter.NilValue{},
		inter.Globals["d"].GetValue(),
	)

	assert.Equal(t,
		interpreter.NilValue{},
		inter.Globals["e"].GetValue(),
	)

	assert.Equal(t,
		interpreter.NilValue{},
		inter.Globals["f"].GetValue(),
	)

	assert.Equal(t,
		interpreter.NilValue{},
		inter.Globals["g"].GetValue(),
	)

	assert.Equal(t,
		inter.Globals["a"].GetValue(),
		inter.Globals["h"].GetValue(),
	)

	assert.Equal(t,
		inter.Globals["b"].GetValue(),
		inter.Globals["i"].GetValue(),
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
			Type: interpreter.CapabilityStaticType{
				BorrowType: interpreter.ReferenceStaticType{
					BorrowedType: interpreter.PrimitiveStaticTypeString,
					Authorized:   false,
				},
			},
		},
		inter.Globals["a"].GetValue(),
	)

	assert.Equal(t,
		interpreter.TypeValue{
			Type: interpreter.CapabilityStaticType{
				BorrowType: interpreter.ReferenceStaticType{
					BorrowedType: interpreter.PrimitiveStaticTypeInt,
					Authorized:   false,
				},
			},
		},
		inter.Globals["b"].GetValue(),
	)

	assert.Equal(t,
		interpreter.TypeValue{
			Type: interpreter.CapabilityStaticType{
				BorrowType: interpreter.ReferenceStaticType{
					BorrowedType: interpreter.CompositeStaticType{
						QualifiedIdentifier: "R",
						Location:            utils.TestLocation,
						TypeID:              "S.test.R",
					},
					Authorized: false,
				},
			},
		},
		inter.Globals["c"].GetValue(),
	)

	assert.Equal(t,
		interpreter.NilValue{},
		inter.Globals["d"].GetValue(),
	)

	assert.Equal(t,
		inter.Globals["a"].GetValue(),
		inter.Globals["e"].GetValue(),
	)
}
