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

package interpreter_test

import (
	"testing"

	"github.com/onflow/cadence/runtime/interpreter"
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
				},
			},
		},
		inter.Globals["c"].GetValue(),
	)

	assert.Equal(t,
		inter.Globals["a"].GetValue(),
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
      let a = ConstantSizedArrayType(Type<String>(), 10)
      let b = ConstantSizedArrayType(Type<Int>(), 5) 

	  resource R {}
	  let c = ConstantSizedArrayType(Type<@R>(), 400)
      let d = ConstantSizedArrayType(a, 6)

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
      let a = DictionaryType(Type<String>(), Type<Int>())!
      let b = DictionaryType(Type<Int>(), Type<String>())!

	  resource R {}
	  let c = DictionaryType(Type<Int>(), Type<@R>())!
      let d = DictionaryType(Type<Bool>(), a)!

	  let e = Type<{String: Int}>()!
	  
	  let f = DictionaryType(Type<[Bool]>(), Type<Int>())
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
    `)

	assert.Equal(t,
		interpreter.TypeValue{
			Type: interpreter.CompositeStaticType{
				QualifiedIdentifier: "R",
				Location:            utils.TestLocation,
			},
		},
		inter.Globals["a"].GetValue(),
	)

	assert.Equal(t,
		interpreter.TypeValue{
			Type: interpreter.CompositeStaticType{
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

	assert.Equal(t,
		inter.Globals["a"].GetValue(),
		inter.Globals["e"].GetValue(),
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
