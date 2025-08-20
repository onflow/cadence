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

package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/common"
)

func TestMemoryMetering(t *testing.T) {
	t.Parallel()

	t.Run("arguments", func(t *testing.T) {
		t.Parallel()

		script := `
          fun main() {
              foo(a: "hello", b: 23)
              bar("hello", 23)
          }

          fun foo(a: String, b: Int) {
          }

          fun bar(_ a: String, _ b: Int) {
          }
        `
		meter := newTestMemoryGauge()
		_, err := ParseProgram(meter, []byte(script), Config{})
		require.NoError(t, err)

		assert.Equal(t, uint64(4), meter.getMemory(common.MemoryKindArgument))
	})

	t.Run("blocks", func(t *testing.T) {
		t.Parallel()

		script := `
	      fun main() {
	          var i = 0
	          if i != 0 {
	              i = 0
	          }

	          while i < 2 {
	              i = i + 1
	          }

	          var a = "foo"
	          switch i {
	              case 1:
	                  a = "foo_1"
	              case 2:
	                  a = "foo_2"
	              case 3:
	                  a = "foo_3"
	          }
	      }
	    `
		meter := newTestMemoryGauge()
		_, err := ParseProgram(meter, []byte(script), Config{})
		require.NoError(t, err)

		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindBlock))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindFunctionBlock))
	})

	t.Run("declarations", func(t *testing.T) {
		t.Parallel()

		script := `
	      import Foo from 0x42

	      let x = 1
	      var y = 2

	      fun main() {
	          var z = 3
	      }

	      fun foo(_ x: String, _ y: Int) {}

	      struct A {
	          var a: String

	          init() {
	              self.a = "hello"
	          }
	      }

	      struct interface B {}

	      resource C {
	          let a: Int

	          init() {
	              self.a = 6
	          }
	      }

	      resource interface D {}

	      enum E: Int8 {
	          case a
	          case b
	          case c
	      }

	      transaction {}

	      #pragma
	    `

		meter := newTestMemoryGauge()
		_, err := ParseProgram(meter, []byte(script), Config{})
		require.NoError(t, err)

		assert.Equal(t, uint64(4), meter.getMemory(common.MemoryKindFunctionDeclaration))
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindCompositeDeclaration))
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindInterfaceDeclaration))
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindEnumCaseDeclaration))
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindFieldDeclaration))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindTransactionDeclaration))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindImportDeclaration))
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindVariableDeclaration))
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindSpecialFunctionDeclaration))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindPragmaDeclaration))

		assert.Equal(t, uint64(4), meter.getMemory(common.MemoryKindFunctionBlock))
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindParameter))
		assert.Equal(t, uint64(4), meter.getMemory(common.MemoryKindParameterList))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindProgram))
		assert.Equal(t, uint64(12), meter.getMemory(common.MemoryKindMembers))
	})

	t.Run("statements", func(t *testing.T) {
		t.Parallel()

		script := `
	      fun main() {
	          var a = 5

	          while a < 10 {               // while
	              if a == 5 {              // if
	                  a = a + 1            // assignment
	                  continue             // continue
	              }
	              break                    // break
	          }

	          foo()                        // expression statement

	          for value in [1, 2, 3] {}    // for

	          var r1 <- create bar()
	          var r2 <- create bar()
	          r1 <-> r2                    // swap

	          destroy r1                   // expression statement
	          destroy r2                   // expression statement

	          switch a {                   // switch
	              case 1:
	                  a = 2                // assignment
	          }
	      }

	      fun foo(): Int {
	           return 5                    // return
	      }

	      resource bar {}

	      contract Events {
	          event FooEvent(x: Int, y: Int)

	          fun events() {
	              emit FooEvent(x: 1, y: 2)    // emit
	          }
	      }
	    `

		meter := newTestMemoryGauge()
		_, err := ParseProgram(meter, []byte(script), Config{})
		require.NoError(t, err)

		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindAssignmentStatement))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindBreakStatement))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindContinueStatement))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindIfStatement))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindForStatement))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindWhileStatement))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindReturnStatement))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindSwapStatement))
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindExpressionStatement))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindSwitchStatement))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindEmitStatement))

		assert.Equal(t, uint64(5), meter.getMemory(common.MemoryKindTransfer))
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindMembers))
	})

	t.Run("expressions", func(t *testing.T) {
		t.Parallel()

		script := `
	      fun main() {
	          var a = 5                                // integer expr
	          var b = 1.2 + 2.3                        // binary, fixed-point expr
	          var c = !true                            // unary, boolean expr
	          var d: String? = "hello"                 // string expr
	          var e = nil                              // nil expr
	          var f: [AnyStruct] = [[], [], []]        // array expr
	          var g: {Int: {Int: AnyStruct}} = {1:{}}  // nil expr
	          var h <- create bar()                    // create, identifier, invocation
	          var i = h.baz                            // member access, identifier x2
	          destroy h                                // destroy
	          var j = f[0]                             // index access, identifier, integer
	          var k = fun() {}                         // function expr
	          k()                                      // identifier, invocation
	          var l = c ? 1 : 2                        // conditional, identifier, integer x2
	          var m = d as AnyStruct                   // casting, identifier
	          var n = &d as &AnyStruct?                // reference, casting, identifier
	          var o = d!                               // force, identifier
	          var p = /public/somepath                 // path
	      }

	      resource bar {
	          let baz: Int
	          init() {
	              self.baz = 0x4
	          }
	        }
	    `

		meter := newTestMemoryGauge()
		_, err := ParseProgram(meter, []byte(script), Config{})
		require.NoError(t, err)

		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindBooleanExpression))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindNilExpression))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindStringExpression))
		assert.Equal(t, uint64(6), meter.getMemory(common.MemoryKindIntegerExpression))
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindFixedPointExpression))
		assert.Equal(t, uint64(7), meter.getMemory(common.MemoryKindArrayExpression))
		assert.Equal(t, uint64(3), meter.getMemory(common.MemoryKindDictionaryExpression))
		assert.Equal(t, uint64(10), meter.getMemory(common.MemoryKindIdentifierExpression))
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindInvocationExpression))
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindMemberExpression))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindIndexExpression))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindConditionalExpression))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindUnaryExpression))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindBinaryExpression))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindFunctionExpression))
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindCastingExpression))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindCreateExpression))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindDestroyExpression))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindReferenceExpression))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindForceExpression))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindPathExpression))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindDictionaryEntry))
	})

	t.Run("types", func(t *testing.T) {
		t.Parallel()

		script := `
	      fun main() {
	          var a: Int = 5                                     // nominal type
	          var b: String? = "hello"                           // optional type
	          var c: [Int; 2] = [1, 2]                           // constant sized type
	          var d: [String] = []                               // variable sized type
	          var e: {Int: String} = {}                          // dictionary type

	          var f: fun(String):Int = fun(_a: String): Int {     // function type
	              return 1
	          }

	          var g = &a as &Int                                 // reference type
	          var h: {foo} = bar()                      // intersection type
	          var i: Capability<&bar>? = nil                     // instantiation type
	      }

	      struct interface foo {}

	      struct bar: foo {}
	    `

		meter := newTestMemoryGauge()
		_, err := ParseProgram(meter, []byte(script), Config{})
		require.NoError(t, err)

		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindConstantSizedType))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindDictionaryType))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindFunctionType))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindInstantiationType))
		assert.Equal(t, uint64(15), meter.getMemory(common.MemoryKindNominalType))
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindOptionalType))
		assert.Equal(t, uint64(2), meter.getMemory(common.MemoryKindReferenceType))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindIntersectionType))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindVariableSizedType))

		assert.Equal(t, uint64(14), meter.getMemory(common.MemoryKindTypeAnnotation))
	})

	t.Run("position info", func(t *testing.T) {
		t.Parallel()

		script := `
	      let x = 1
	      var y = 2

	      fun main() {
	          var z = 3
	      }

	      fun foo(_ x: String, _ y: Int) {}

	      struct A {
	          var a: String

	          init() {
	              self.a = "hello"
	          }
	      }

	      struct interface B {}
	    `

		meter := newTestMemoryGauge()
		_, err := ParseProgram(meter, []byte(script), Config{})
		require.NoError(t, err)

		assert.Equal(t, uint64(199), meter.getMemory(common.MemoryKindPosition))
		assert.Equal(t, uint64(109), meter.getMemory(common.MemoryKindRange))
	})

	t.Run("locations", func(t *testing.T) {
		t.Parallel()

		script := `
	      import A from 0x42
	      import B from "string-location"
	    `

		meter := newTestMemoryGauge()
		_, err := ParseProgram(meter, []byte(script), Config{})
		require.NoError(t, err)

		assert.Equal(t, uint64(16), meter.getMemory(common.MemoryKindRawString))
		assert.Equal(t, uint64(1), meter.getMemory(common.MemoryKindAddressLocation))
	})

}
