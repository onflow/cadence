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

package sema_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/sema"
	. "github.com/onflow/cadence/test_utils/sema_utils"
)

func TestCheckCharacter(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
        let x: Character = "x"
	`)

	require.NoError(t, err)

	assert.Equal(t,
		sema.CharacterType,
		RequireGlobalValue(t, checker.Elaboration, "x"),
	)
}

func TestCheckCharacterUnicodeScalar(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
        let x: Character = "\u{1F1FA}\u{1F1F8}"
	`)

	require.NoError(t, err)

	assert.Equal(t,
		sema.CharacterType,
		RequireGlobalValue(t, checker.Elaboration, "x"),
	)
}

func TestCheckString(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
        let x = "x"
	`)

	require.NoError(t, err)

	assert.Equal(t,
		sema.StringType,
		RequireGlobalValue(t, checker.Elaboration, "x"),
	)
}

func TestCheckStringConcat(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
	  fun test(): String {
	 	  let a = "abc"
		  let b = "def"
		  let c = a.concat(b)
		  return c
      }
    `)

	require.NoError(t, err)
}

func TestCheckInvalidStringConcat(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test(): String {
		  let a = "abc"
		  let b = [1, 2]
		  let c = a.concat(b)
		  return c
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
}

func TestCheckStringConcatBound(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test(): String {
		  let a = "abc"
		  let b = "def"
		  let c = a.concat
		  return c(b)
      }
    `)

	require.NoError(t, err)
}

func TestCheckStringSlice(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
	  fun test(): String {
	 	  let a = "abcdef"
		  return a.slice(from: 0, upTo: 1)
      }
    `)

	require.NoError(t, err)
}

func TestCheckInvalidStringSlice(t *testing.T) {

	t.Parallel()

	t.Run("MissingBothArgumentLabels", func(t *testing.T) {

		_, err := ParseAndCheck(t, `
		  let a = "abcdef"
		  let x = a.slice(0, 1)
		`)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.MissingArgumentLabelError{}, errs[0])
		assert.IsType(t, &sema.MissingArgumentLabelError{}, errs[1])
	})

	t.Run("MissingOneArgumentLabel", func(t *testing.T) {

		_, err := ParseAndCheck(t, `
		  let a = "abcdef"
		  let x = a.slice(from: 0, 1)
		`)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.MissingArgumentLabelError{}, errs[0])
	})

	t.Run("InvalidArgumentType", func(t *testing.T) {

		_, err := ParseAndCheck(t, `
		  let a = "abcdef"
		  let x = a.slice(from: "a", upTo: "b")
		`)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		assert.IsType(t, &sema.TypeMismatchError{}, errs[1])
	})
}

func TestCheckStringSliceBound(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test(): String {
		  let a = "abcdef"
		  let c = a.slice
		  return c(from: 0, upTo: 1)
      }
    `)

	require.NoError(t, err)
}

func TestCheckStringIndexing(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test() {
          let z = "abc"
          let y: Character = z[0]
      }
	`)

	require.NoError(t, err)
}

func TestCheckInvalidStringIndexingAssignment(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test() {
		  let z = "abc"
		  let y: Character = "d"
          z[0] = y
      }
	`)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.NotIndexingAssignableTypeError{}, errs[0])
}

func TestCheckInvalidStringIndexingAssignmentWithCharacterLiteral(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test() {
          let z = "abc"
          z[0] = "d"
      }
	`)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.NotIndexingAssignableTypeError{}, errs[0])
}

func TestCheckStringFunction(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
        let x = String()
	`)

	require.NoError(t, err)

	assert.Equal(t,
		sema.StringType,
		RequireGlobalValue(t, checker.Elaboration, "x"),
	)
}

func TestCheckStringDecodeHex(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
        let x = "01CADE".decodeHex()
	`)

	require.NoError(t, err)

	assert.Equal(t,
		sema.ByteArrayType,
		RequireGlobalValue(t, checker.Elaboration, "x"),
	)
}

func TestCheckStringEncodeHex(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
        let x = String.encodeHex([1, 2, 3, 0xCA, 0xDE])
	`)

	require.NoError(t, err)

	assert.Equal(t,
		sema.StringType,
		RequireGlobalValue(t, checker.Elaboration, "x"),
	)
}

func TestCheckStringFromUTF8(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
        let x = String.fromUTF8([0xEA, 0x99, 0xAE])
	`)
	require.NoError(t, err)

	assert.Equal(t,
		&sema.OptionalType{Type: sema.StringType},
		RequireGlobalValue(t, checker.Elaboration, "x"),
	)
}

func TestCheckStringFromCharacters(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
        let x = String.fromCharacters(["👪", "❤️"])
	`)
	require.NoError(t, err)

	assert.Equal(t,
		sema.StringType,
		RequireGlobalValue(t, checker.Elaboration, "x"),
	)
}

func TestCheckStringUtf8Field(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `

      let x = "abc".utf8
	`)

	require.NoError(t, err)

	assert.Equal(t,
		sema.ByteArrayType,
		RequireGlobalValue(t, checker.Elaboration, "x"),
	)
}

func TestCheckStringToLower(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
        let x = "Abc".toLower()
	`)

	require.NoError(t, err)

	assert.Equal(t,
		sema.StringType,
		RequireGlobalValue(t, checker.Elaboration, "x"),
	)
}

func TestCheckStringJoin(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
		let s = String.join(["👪", "❤️", "Abc"], separator: "/")
	`)
	require.NoError(t, err)

	assert.Equal(t,
		sema.StringType,
		RequireGlobalValue(t, checker.Elaboration, "s"),
	)
}

func TestCheckStringJoinTypeMismatchStrs(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
		let s = String.join([1], separator: "/")
	`)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
}

func TestCheckStringJoinTypeMismatchSeparator(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
		let s = String.join(["Abc", "1"], separator: 1234)
	`)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
}

func TestCheckStringJoinTypeMissingArgumentLabelSeparator(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
	let s = String.join(["👪", "❤️", "Abc"], "/")
	`)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.MissingArgumentLabelError{}, errs[0])
}

func TestCheckStringSplit(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
		let s = "👪.❤️.Abc".split(separator: ".")
	`)
	require.NoError(t, err)

	assert.Equal(t,
		&sema.VariableSizedType{
			Type: sema.StringType,
		},
		RequireGlobalValue(t, checker.Elaboration, "s"),
	)
}

func TestCheckStringSplitTypeMismatchSeparator(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
		let s = "Abc:1".split(separator: 1234)
	`)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
}

func TestCheckStringSplitTypeMissingArgumentLabelSeparator(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
	let s = "👪Abc".split("/")
	`)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.MissingArgumentLabelError{}, errs[0])
}

func TestCheckStringReplaceAll(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
		let s = "👪.❤️.Abc".replaceAll(of: "❤️", with: "|")
	`)
	require.NoError(t, err)

	assert.Equal(t,
		sema.StringType,
		RequireGlobalValue(t, checker.Elaboration, "s"),
	)
}

func TestCheckStringReplaceAllTypeMismatchOf(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
		let s = "Abc:1".replaceAll(of: 1234, with: "/")
	`)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
}

func TestCheckStringReplaceAllTypeMismatchWith(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
		let s = "Abc:1".replaceAll(of: "1", with: true)
	`)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
}

func TestCheckStringReplaceAllTypeMismatchCharacters(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
		let a: Character = "x"
		let b: Character = "y"
		let s = "Abc:1".replaceAll(of: a, with: b)
	`)

	errs := RequireCheckerErrors(t, err, 2)

	assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
	assert.IsType(t, &sema.TypeMismatchError{}, errs[1])
}

func TestCheckStringReplaceAllTypeMissingArgumentLabelOf(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
	let s = "👪Abc".replaceAll("/", with: "abc")
	`)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.MissingArgumentLabelError{}, errs[0])
}

func TestCheckStringReplaceAllTypeMissingArgumentLabelWith(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
	let s = "👪Abc".replaceAll(of: "/", "abc")
	`)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.MissingArgumentLabelError{}, errs[0])
}

func TestCheckStringContains(t *testing.T) {

	t.Parallel()

	t.Run("missing argument", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
		  let a = "abcdef"
		  let x: Bool = a.contains()
		`)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InsufficientArgumentsError{}, errs[0])
	})

	t.Run("wrong argument type", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
		  let a = "abcdef"
		  let x: Bool = a.contains(1)
		`)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
	})

	t.Run("valid", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
		  let a = "abcdef"
		  let x: Bool = a.contains("abc")
		`)

		require.NoError(t, err)
	})
}

func TestCheckStringIndex(t *testing.T) {

	t.Parallel()

	t.Run("missing argument", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
		  let a = "abcdef"
		  let x: Int = a.index()
		`)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InsufficientArgumentsError{}, errs[0])
	})

	t.Run("wrong argument type", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
		  let a = "abcdef"
		  let x: Int = a.index(of: 1)
		`)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
	})

	t.Run("wrong argument label", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
		  let a = "abcdef"
		  let x: Int = a.index(foo: "bc")
		`)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.IncorrectArgumentLabelError{}, errs[0])
	})

	t.Run("missing argument label", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
		  let a = "abcdef"
		  let x: Int = a.index("bc")
		`)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.MissingArgumentLabelError{}, errs[0])
	})

	t.Run("valid", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
		  let a = "abcdef"
		  let x: Int = a.index(of: "bc")
		`)

		require.NoError(t, err)
	})
}

func TestCheckStringCount(t *testing.T) {

	t.Parallel()

	t.Run("missing argument", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
		  let a = "abcdef"
		  let x: Int = a.count()
		`)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InsufficientArgumentsError{}, errs[0])
	})

	t.Run("wrong argument type", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
		  let a = "abcdef"
		  let x: Int = a.count(1)
		`)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
	})

	t.Run("valid", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
		  let a = "abcdef"
		  let x: Int = a.count("b")
		`)

		require.NoError(t, err)
	})
}

func TestCheckStringTemplate(t *testing.T) {

	t.Parallel()

	t.Run("valid, int", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
			let a = 1
			let x: String = "The value of a is: \(a)" 
		`)

		require.NoError(t, err)
	})

	t.Run("valid, string", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
			let a = "abc def"
			let x: String = "\(a) ghi" 
		`)

		require.NoError(t, err)
	})

	t.Run("invalid, struct", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
			access(all)
			struct SomeStruct {}
			let a = SomeStruct()
			let x: String = "\(a)" 
		`)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.TypeMismatchWithDescriptionError{}, errs[0])
	})

	t.Run("invalid, array", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
			let x: [AnyStruct] = ["tmp", 1]
			let y = "\(x)"
		`)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.TypeMismatchWithDescriptionError{}, errs[0])
	})

	t.Run("invalid, missing variable", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
			let x: String = "\(a)" 
		`)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
		assert.IsType(t, &sema.TypeMismatchWithDescriptionError{}, errs[1])
	})

	t.Run("invalid, resource", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
			access(all) resource TestResource {}
			fun test(): String {
				var x <- create TestResource()
				var y = "\(x)"
				destroy x
				return y
			} 
		`)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.TypeMismatchWithDescriptionError{}, errs[0])
	})
}
