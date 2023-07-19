/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Dapper Labs, Inc.
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

package checker

import (
	"fmt"
	"math"
	"math/big"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/sema"
	. "github.com/onflow/cadence/runtime/tests/utils"
)

func TestCheckDictionary(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      let z = {"a": 1, "b": 2}
	`)

	assert.NoError(t, err)
}

func TestCheckDictionaryType(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      let z: {String: Int} = {"a": 1, "b": 2}
	`)

	assert.NoError(t, err)
}

func TestCheckInvalidDictionaryTypeKey(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      let z: {Int: Int} = {"a": 1, "b": 2}
	`)

	errs := RequireCheckerErrors(t, err, 2)

	assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
	typeMismatchError := errs[0].(*sema.TypeMismatchError)
	assert.Equal(t, sema.IntType, typeMismatchError.ExpectedType)
	assert.Equal(t, sema.StringType, typeMismatchError.ActualType)

	assert.IsType(t, &sema.TypeMismatchError{}, errs[1])
	typeMismatchError = errs[1].(*sema.TypeMismatchError)
	assert.Equal(t, sema.IntType, typeMismatchError.ExpectedType)
	assert.Equal(t, sema.StringType, typeMismatchError.ActualType)
}

func TestCheckInvalidDictionaryTypeValue(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      let z: {String: String} = {"a": 1, "b": 2}
	`)

	errs := RequireCheckerErrors(t, err, 2)

	assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
	typeMisMatchError := errs[0].(*sema.TypeMismatchError)
	assert.Equal(t, sema.StringType, typeMisMatchError.ExpectedType)
	assert.Equal(t, sema.IntType, typeMisMatchError.ActualType)

	assert.IsType(t, &sema.TypeMismatchError{}, errs[1])
	typeMisMatchError = errs[1].(*sema.TypeMismatchError)
	assert.Equal(t, sema.StringType, typeMisMatchError.ExpectedType)
	assert.Equal(t, sema.IntType, typeMisMatchError.ActualType)
}

func TestCheckInvalidDictionaryTypeSwapped(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      let z: {Int: String} = {"a": 1, "b": 2}
	`)

	errs := RequireCheckerErrors(t, err, 4)

	assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
	typeMisMatchError := errs[0].(*sema.TypeMismatchError)
	assert.Equal(t, sema.IntType, typeMisMatchError.ExpectedType)
	assert.Equal(t, sema.StringType, typeMisMatchError.ActualType)

	assert.IsType(t, &sema.TypeMismatchError{}, errs[1])
	typeMisMatchError = errs[1].(*sema.TypeMismatchError)
	assert.Equal(t, sema.StringType, typeMisMatchError.ExpectedType)
	assert.Equal(t, sema.IntType, typeMisMatchError.ActualType)

	assert.IsType(t, &sema.TypeMismatchError{}, errs[2])
	typeMisMatchError = errs[2].(*sema.TypeMismatchError)
	assert.Equal(t, sema.IntType, typeMisMatchError.ExpectedType)
	assert.Equal(t, sema.StringType, typeMisMatchError.ActualType)

	assert.IsType(t, &sema.TypeMismatchError{}, errs[3])
	typeMisMatchError = errs[3].(*sema.TypeMismatchError)
	assert.Equal(t, sema.StringType, typeMisMatchError.ExpectedType)
	assert.Equal(t, sema.IntType, typeMisMatchError.ActualType)
}

func TestCheckInvalidDictionaryKeys(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      let z = {"a": 1, true: 2}
	`)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.InvalidDictionaryKeyTypeError{}, errs[0])
}

func TestCheckDictionaryIndexingString(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
      let x = {"abc": 1, "def": 2}
      let y = x["abc"]
    `)

	require.NoError(t, err)

	yType := RequireGlobalValue(t, checker.Elaboration, "y")

	assert.Equal(t,
		&sema.OptionalType{
			Type: sema.IntType,
		},
		yType,
	)
}

func TestCheckDictionaryIndexingBool(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      let x = {true: 1, false: 2}
      let y = x[true]
	`)

	assert.NoError(t, err)
}

func TestCheckInvalidDictionaryIndexing(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      let x = {"abc": 1, "def": 2}
      let y = x[true]
	`)

	errs := RequireCheckerErrors(t, err, 1)

	require.IsType(t, &sema.TypeMismatchError{}, errs[0])
	typeMismatchError := errs[0].(*sema.TypeMismatchError)
	assert.Equal(t, sema.StringType, typeMismatchError.ExpectedType)
	assert.Equal(t, sema.BoolType, typeMismatchError.ActualType)
}

func TestCheckDictionaryIndexingAssignment(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test() {
          let x = {"abc": 1, "def": 2}
          x["abc"] = 3
      }
    `)

	require.NoError(t, err)
}

func TestCheckInvalidDictionaryIndexingAssignment(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test() {
          let x = {"abc": 1, "def": 2}
          x["abc"] = true
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
}

func TestCheckDictionaryRemove(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test() {
          let x = {"abc": 1, "def": 2}
          let old: Int? = x.remove(key: "abc")
      }
    `)

	require.NoError(t, err)
}

func TestCheckInvalidDictionaryRemove(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test() {
          let x = {"abc": 1, "def": 2}
          let old: Int? = x.remove(key: true)
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
}

func TestCheckDictionaryInsert(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test() {
          let x = {"abc": 1, "def": 2}
          let old: Int? = x.insert(key: "abc", 3)
      }
    `)

	require.NoError(t, err)
}

func TestCheckInvalidDictionaryInsert(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test() {
          let x = {"abc": 1, "def": 2}
          let old: Int? = x.insert(key: true, 3)
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
}

func TestCheckDictionaryKeys(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
        let keys = {"abc": 1, "def": 2}.keys
    `)

	require.NoError(t, err)

	keysType := RequireGlobalValue(t, checker.Elaboration, "keys")

	assert.Equal(t,
		&sema.VariableSizedType{Type: sema.StringType},
		keysType,
	)
}

func TestCheckDictionaryValues(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
        let values = {"abc": 1, "def": 2}.values
    `)

	require.NoError(t, err)

	valuesType := RequireGlobalValue(t, checker.Elaboration, "values")

	assert.Equal(t,
		&sema.VariableSizedType{Type: sema.IntType},
		valuesType,
	)
}

func TestCheckDictionaryEqual(t *testing.T) {
	t.Parallel()

	testValid := func(name, code string) {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := ParseAndCheck(t, code)
			require.NoError(t, err)
		})
	}

	assertInvalid := func(name, code string) {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := ParseAndCheck(t, code)
			errs := RequireCheckerErrors(t, err, 1)
			assert.IsType(t, &sema.InvalidBinaryOperandsError{}, errs[0])
		})
	}

	for _, opStr := range []string{"==", "!="} {
		testValid(
			"self_dict_equality",
			fmt.Sprintf(
				`
					fun test(): Bool {
						let d = {"abc": 1, "def": 2}
						return d %s d
					}
				`,
				opStr,
			),
		)

		testValid(
			"self_dict_equality_nested_1",
			fmt.Sprintf(
				`
					fun test(): Bool {
						let d = {"abc": {1: 100, 2: 200}, "def": {4: 400, 5: 500}}
						return d %s d
					}
				`,
				opStr,
			),
		)

		testValid(
			"self_dict_equality_nested_2",
			fmt.Sprintf(
				`
					fun test(): Bool {
						let d = {"abc": {1: {"a": 1000}, 2: {"b": 2000}}, "def": {4: {"c": 1000}, 5: {"d": 2000}}}
						return d %s d
					}
				`,
				opStr,
			),
		)

		testValid(
			"dict_equality_true",
			fmt.Sprintf(
				`
					fun test(): Bool {
						let d = {"abc": 1, "def": 2}
						let d2 = {"abc": 1, "def": 2}
						return d %s d2
					}
				`,
				opStr,
			),
		)

		testValid(
			"dict_equality_true_nested",
			fmt.Sprintf(
				`
					fun test(): Bool {
						let d = {"abc": {1: {"a": 1000}, 2: {"b": 2000}}, "def": {4: {"c": 1000}, 5: {"d": 2000}}}
						let d2 = {"abc": {1: {"a": 1000}, 2: {"b": 2000}}, "def": {4: {"c": 1000}, 5: {"d": 2000}}}
						return d %s d2
					}
				`,
				opStr,
			),
		)

		testValid(
			"dict_equality_false",
			fmt.Sprintf(
				`
					fun test(): Bool {
						let d = {"abc": 1, "def": 2}
						let d2 = {"abc": 1, "def": 2, "xyz": 4}
						return d %s d2
					}
				`,
				opStr,
			),
		)

		testValid(
			"dict_equality_false_nested",
			fmt.Sprintf(
				`
					fun test(): Bool {
						let d = {"abc": {1: {"a": 1000}, 2: {"b": 2000}}, "def": {4: {"c": 1000}, 5: {"d": 2000}}}
						let d2 = {"abc": {1: {"a": 1000}, 2: {"c": 1000}}, "def": {4: {"c": 1000}, 5: {"d": 2000}}}
						return d %s d2
					}
				`,
				opStr,
			),
		)

		assertInvalid("dict_equality_invalid",
			fmt.Sprintf(
				`
					fun test(): Bool {
						let d = {"abc": 1, "def": 2}
						let d2 = {1: "abc", 2: "def"}
						return d %s d2
					}
				`,
				opStr,
			),
		)

		assertInvalid(
			"dict_equality_invalid_nested",
			fmt.Sprintf(
				`
					fun test(): Bool {
						let d = {"abc": {1: {"a": 1000}, 2: {"b": 2000}}, "def": {4: {"c": 1000}, 5: {"d": 2000}}}
						let d2 = {"abc": {1: {1000: "a"}, 2: {2000: "b"}}, "def": {4: {1000: "c"}, 5: {2000: "d"}}}
						return d %s d2
					}
				`,
				opStr,
			),
		)

		assertInvalid(
			"dict_equality_invalid_inner_type_unequatable",
			fmt.Sprintf(
				`
					fun test(): Bool {
						let d = {"abc": fun (): Void {}}
						let d2 = {"abc": fun (): Void {}}
						return d %s d2
					}
				`,
				opStr,
			),
		)
	}
}

func TestCheckLength(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      let x = "cafe\u{301}".length
      let y = [1, 2, 3].length
    `)

	require.NoError(t, err)
}

func TestCheckArrayAppend(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test(): [Int] {
          let x = [1, 2, 3]
          x.append(4)
          return x
      }
    `)

	require.NoError(t, err)
}

func TestCheckInvalidArrayAppend(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test(): [Int] {
          let x = [1, 2, 3]
          x.append("4")
          return x
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
}

func TestCheckArrayAppendBound(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test(): [Int] {
          let x = [1, 2, 3]
          let y = x.append
          y(4)
          return x
      }
    `)

	require.NoError(t, err)
}

func TestCheckInvalidArrayAppendToConstantSize(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test(): [Int; 3] {
          let x: [Int; 3] = [1, 2, 3]
          x.append(4)
          return x
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.NotDeclaredMemberError{}, errs[0])
}

func TestCheckArrayAppendAll(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
	  fun test(): [Int] {
	 	  let a = [1, 2]
		  let b = [3, 4]
		  a.appendAll(b)
		  return a
      }
    `)

	require.NoError(t, err)
}

func TestCheckInvalidArrayAppendAll(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
	  fun test(): [Int] {
	 	  let a = [1, 2]
		  let b = ["a", "b"]
		  a.appendAll(b)
		  return a
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.TypeMismatchError{}, errs[0])

	_, err = ParseAndCheck(t, `
	  fun test(): [Int] {
	 	  let a = [1, 2]
		  let b = 3
		  a.appendAll(b)
		  return a
      }
    `)

	errs = RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
}

func TestCheckInvalidArrayAppendAllOnConstantSize(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test(): [Int; 3] {
          let x: [Int; 3] = [1, 2, 3]
          x.appendAll([4, 5])
          return x
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.NotDeclaredMemberError{}, errs[0])
}

func TestCheckArrayConcat(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
	  fun test(): [Int] {
	 	  let a = [1, 2]
		  let b = [3, 4]
          let c = a.concat(b)
          return c
      }
    `)

	require.NoError(t, err)
}

func TestCheckVariableSizedArrayEqual(t *testing.T) {
	t.Parallel()

	for i := 0; i < 4; i++ {
		nestingLevel := i
		array := fmt.Sprintf("%s 42 %s", strings.Repeat("[", nestingLevel), strings.Repeat("]", nestingLevel))

		for _, opStr := range []string{"==", "!="} {
			op := opStr
			testName := fmt.Sprintf("test array %s at nesting level %d", op, nestingLevel)

			t.Run(testName, func(t *testing.T) {
				t.Parallel()
				code := fmt.Sprintf(`
					fun test(): Bool {
						let xs = %s
						return xs %s xs
					}`,
					array,
					op,
				)

				_, err := ParseAndCheck(t, code)
				require.NoError(t, err)
			})
		}
	}
}

func TestCheckFixedSizedArrayEqual(t *testing.T) {
	t.Parallel()

	testValid := func(name, code string) {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := ParseAndCheck(t, code)
			require.NoError(t, err)
		})
	}

	testValid("[Int; 3]", `
		fun test(): Bool {
			let xs: [Int; 3] = [1, 2, 3]
			return xs == xs
		}
	`)

	testValid("[[Int; 3]; 2]", `
		fun test(): Bool {
			let xs: [Int; 3] = [1, 2, 3]
			let ys: [[Int; 3]; 2] = [xs, xs]
			return ys == ys
		}
	`)
}

func TestCheckInvalidArrayEqual(t *testing.T) {
	t.Parallel()

	assertInvalid := func(name, innerCode string) {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			code := fmt.Sprintf("fun test(): Bool { \n %s \n}", innerCode)

			_, err := ParseAndCheck(t, code)
			errs := RequireCheckerErrors(t, err, 1)
			assert.IsType(t, &sema.InvalidBinaryOperandsError{}, errs[0])
		})
	}

	assertInvalid("variable size array", `
		let xs = [fun(){}]
		return xs == xs
	`)

	assertInvalid("fixed size array", `
		let xs: [((): Void); 1] = [fun(){}]
		return xs == xs
	`)

	assertInvalid("fixed size equaling variable-size", `
		let xs: [Int; 3] = [1, 2, 3]
		let ys: [Int] = [1, 2, 3]
		return xs == ys
	`)

	assertInvalid("fixed size arrays of different lengths", `
		let xs: [Int; 2] = [42, 1337]
		let ys: [Int; 3] = [1, 2, 3]
		return xs == ys
	`)

	assertInvalid("fixed size arrays of different types", `
		let xs: [Int; 2] = [42, 1337]
		let ys: [String; 3] = ["O", "w", "O"]
		return xs != ys
	`)
}

func TestCheckInvalidArrayConcat(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test(): [Int] {
		  let a = [1, 2]
		  let b = ["a", "b"]
          let c = a.concat(b)
          return c
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
}

func TestCheckInvalidArrayConcatOfConstantSized(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
	  fun test(): [Int] {
	 	  let a: [Int; 2] = [1, 2]
		  let b: [Int; 2] = [3, 4]
          let c = a.concat(b)
          return c
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.NotDeclaredMemberError{}, errs[0])
}

func TestCheckArrayConcatBound(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test(): [Int] {
		  let a = [1, 2]
		  let b = [3, 4]
		  let c = a.concat
		  return c(b)
      }
    `)

	require.NoError(t, err)
}

func TestCheckArraySlice(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
	  fun test(): [Int] {
	 	  let a = [1, 2, 3, 4]
		  return a.slice(from: 1, upTo: 2)
      }
    `)

	require.NoError(t, err)
}

func TestCheckArraySliceBound(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
	  fun test(): [Int] {
	 	  let a = [1, 2, 3, 4]
          let s = a.slice
		  return s(from: 1, upTo: 2)
      }
    `)

	require.NoError(t, err)
}

func TestCheckInvalidResourceArraySlice(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource X {}

      fun test(): @[X] {
          let xs <- [<-create X()]
          return <-xs.slice(from: 0, upTo: 1)
      }
    `)

	errs := RequireCheckerErrors(t, err, 2)

	assert.IsType(t, &sema.InvalidResourceArrayMemberError{}, errs[0])
	assert.IsType(t, &sema.ResourceLossError{}, errs[1])
}

func TestCheckArrayInsert(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test(): [Int] {
          let x = [1, 2, 3]
          x.insert(at: 1, 4)
          return x
      }
    `)

	require.NoError(t, err)
}

func TestCheckInvalidArrayInsert(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test(): [Int] {
          let x = [1, 2, 3]
          x.insert(at: 1, "4")
          return x
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
}

func TestCheckInvalidArrayInsertIntoConstantSized(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test(): [Int; 3] {
          let x: [Int; 3] = [1, 2, 3]
          x.insert(at: 1, 4)
          return x
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.NotDeclaredMemberError{}, errs[0])
}

func TestCheckArrayRemove(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test(): [Int] {
          let x = [1, 2, 3]
          let old: Int? = x.remove(at: 1)
          return x
      }
    `)

	require.NoError(t, err)
}

func TestCheckInvalidArrayRemove(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test(): [Int] {
          let x = [1, 2, 3]
          let old: Int? = x.remove(at: "1")
          return x
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
}

func TestCheckInvalidArrayRemoveFromConstantSized(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test(): [Int; 3] {
          let x: [Int; 3] = [1, 2, 3]
          let old: Int? = x.remove(at: 1)
          return x
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.NotDeclaredMemberError{}, errs[0])
}

func TestCheckArrayRemoveFirst(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test(): [Int] {
          let x = [1, 2, 3]
          let old: Int? = x.removeFirst()
          return x
      }
    `)

	require.NoError(t, err)
}

func TestCheckInvalidArrayRemoveFirst(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test(): [Int] {
          let x = [1, 2, 3]
          let old: Int? = x.removeFirst(1)
          return x
      }
	`)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.ExcessiveArgumentsError{}, errs[0])
}

func TestCheckInvalidArrayRemoveFirstFromConstantSized(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test(): [Int; 3] {
          let x: [Int; 3] = [1, 2, 3]
          let old: Int? = x.removeFirst()
          return x
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.NotDeclaredMemberError{}, errs[0])
}

func TestCheckArrayRemoveLast(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test(): [Int] {
          let x = [1, 2, 3]
          let old: Int? = x.removeLast()
          return x
      }
    `)

	require.NoError(t, err)
}

func TestCheckInvalidArrayRemoveLastFromConstantSized(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test(): [Int; 3] {
          let x: [Int; 3] = [1, 2, 3]
          let old: Int? = x.removeLast()
          return x
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.NotDeclaredMemberError{}, errs[0])
}

func TestCheckArrayIndexOf(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test(): Int? {
          let x = [1, 2, 3]
          return x.firstIndex(of: 2)
      }
    `)

	require.NoError(t, err)
}

func TestCheckArrayIndexOfNonEquatableValueArray(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test(): Int? {
          let x = [[fun(){}, fun(){}], [fun(){}]]
          return x.firstIndex(of: [fun(){}])
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)
	assert.IsType(t, &sema.NotEquatableTypeError{}, errs[0])
}

func TestCheckArrayFirstIndexWrongType(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test(): Int? {
          let x = [1, 2, 3]
          return x.firstIndex(of: "foo")
      }
    `)
	errs := RequireCheckerErrors(t, err, 1)
	assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
}

func TestCheckInvalidResourceFirstIndex(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource X {}

      fun test(): Int? {
          let xs <- [<-create X()]
          return xs.firstIndex(of: <-create X())
      }
    `)

	errs := RequireCheckerErrors(t, err, 3)

	assert.IsType(t, &sema.InvalidResourceArrayMemberError{}, errs[0])
	assert.IsType(t, &sema.NotEquatableTypeError{}, errs[1])
	assert.IsType(t, &sema.ResourceLossError{}, errs[2])
}

func TestCheckArrayReverse(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test() {
          let x = [1, 2, 3]
          let y = x.reverse()
      }
    `)

	require.NoError(t, err)
}

func TestCheckArrayReverseInvalidArgs(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test() {
          let x = [1, 2, 3]
          let y = x.reverse(100)
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.ExcessiveArgumentsError{}, errs[0])
}

func TestCheckResourceArrayReverseInvalid(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
		resource X {}

		fun test(): @[X] {
			let xs <- [<-create X()]
			let revxs <-xs.reverse()
			destroy xs
			return <- revxs
		}
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.InvalidResourceArrayMemberError{}, errs[0])
}

func TestCheckArrayContains(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test(): Bool {
          let x = [1, 2, 3]
          return x.contains(2)
      }
    `)

	require.NoError(t, err)
}

func TestCheckInvalidArrayContains(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test(): Bool {
          let x = [1, 2, 3]
          return x.contains("abc")
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
}

func TestCheckInvalidArrayContainsNotEquatable(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test(): Bool {
          let z = [[fun(){}], [fun(){}], [fun(){}]]
          return z.contains([fun(){}])
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.NotEquatableTypeError{}, errs[0])
}

func TestCheckEmptyArray(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      let xs: [Int] = []
	`)

	require.NoError(t, err)
}

func TestCheckEmptyArrayCall(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun foo(xs: [Int]) {
          foo(xs: [])
      }
	`)

	require.NoError(t, err)
}

func TestCheckDictionaryContainsKey(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test(): Bool {
          let x = {1: "One", 2: "Two", 3: "Three"}
          return x.containsKey(2)
      }
    `)

	require.NoError(t, err)
}

func TestCheckInvalidDictionaryContainsKey(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test(): Bool {
          let x = {1: "One", 2: "Two", 3: "Three"}
          return x.containsKey("abc")
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
}

func TestCheckEmptyDictionary(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      let xs: {String: Int} = {}
	`)

	require.NoError(t, err)
}

func TestCheckEmptyDictionaryCall(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun foo(xs: {String: Int}) {
          foo(xs: {})
      }
	`)

	require.NoError(t, err)
}

func TestCheckArraySubtyping(t *testing.T) {

	t.Parallel()

	for _, kind := range common.AllCompositeKinds {

		if !kind.SupportsInterfaces() {
			continue
		}

		t.Run(kind.Keyword(), func(t *testing.T) {

			body := "{}"
			if kind == common.CompositeKindEvent {
				body = "()"
			}

			interfaceType := AsInterfaceType("I", kind)

			_, err := ParseAndCheck(t,
				fmt.Sprintf(
					`
                      %[1]s interface I %[2]s
                      %[1]s S: I %[2]s

                      let xs: %[3]s[S] %[4]s []
                      let ys: %[3]s[%[5]s] %[4]s xs
	                `,
					kind.Keyword(),
					body,
					kind.Annotation(),
					kind.TransferOperator(),
					interfaceType,
				),
			)
			require.NoError(t, err)
		})
	}
}

func TestCheckInvalidArraySubtyping(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      let xs: [Bool] = []
      let ys: [Int] = xs
	`)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
}

func TestCheckDictionarySubtyping(t *testing.T) {

	t.Parallel()

	for _, kind := range common.AllCompositeKinds {

		if !kind.SupportsInterfaces() {
			continue
		}

		t.Run(kind.Keyword(), func(t *testing.T) {

			body := "{}"
			if kind == common.CompositeKindEvent {
				body = "()"
			}

			interfaceType := AsInterfaceType("I", kind)

			_, err := ParseAndCheck(t,
				fmt.Sprintf(
					`
                      %[1]s interface I %[2]s
                      %[1]s S: I %[2]s

                      let xs: %[3]s{String: S} %[4]s {}
                      let ys: %[3]s{String: %[5]s} %[4]s xs
	                `,
					kind.Keyword(),
					body,
					kind.Annotation(),
					kind.TransferOperator(),
					interfaceType,
				),
			)

			require.NoError(t, err)
		})
	}
}

func TestCheckInvalidDictionarySubtyping(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      let xs: {String: Bool} = {}
      let ys: {String: Int} = xs
	`)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
}

func TestCheckConstantSizedArrayDeclaration(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      let x: [Int; 3] = [1, 2, 3]
    `)

	require.NoError(t, err)
}

func TestCheckInvalidConstantSizedArrayDeclarationCountMismatchTooMany(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      let x: [Int; 2] = [1, 2, 3]
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.ConstantSizedArrayLiteralSizeError{}, errs[0])
}

func TestCheckInvalidConstantSizedArrayDeclarationOutOfRangeSize(t *testing.T) {

	t.Parallel()

	t.Run("too large", func(t *testing.T) {

		tooLarge := new(big.Int).SetUint64(math.MaxUint64)
		tooLarge.Add(tooLarge, big.NewInt(1))

		_, err := ParseAndCheck(t,
			fmt.Sprintf(
				`
                  let x: [Int; %s] = []
			    `,
				tooLarge,
			),
		)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.InvalidConstantSizedTypeSizeError{}, errs[0])
		assert.IsType(t, &sema.ConstantSizedArrayLiteralSizeError{}, errs[1])
	})
}

func TestCheckInvalidConstantSizedArrayDeclarationBase(t *testing.T) {

	t.Parallel()

	for _, size := range []string{"0x42", "0b1010", "0o10"} {

		t.Run(size, func(t *testing.T) {

			_, err := ParseAndCheck(t,
				fmt.Sprintf(
					`
                      let x: [Int; %s] = []
                    `,
					size,
				),
			)

			errs := RequireCheckerErrors(t, err, 2)

			assert.IsType(t, &sema.InvalidConstantSizedTypeBaseError{}, errs[0])
			assert.IsType(t, &sema.ConstantSizedArrayLiteralSizeError{}, errs[1])
		})
	}
}

func TestCheckDictionaryKeyTypesExpressions(t *testing.T) {

	t.Parallel()

	tests := map[string]string{
		"String":         `"abc"`,
		"Character":      `"X"`,
		"Address":        `0x1`,
		"Bool":           `true`,
		"Path":           `/storage/a`,
		"StoragePath":    `/storage/a`,
		"PublicPath":     `/public/a`,
		"PrivatePath":    `/private/a`,
		"CapabilityPath": `/private/a`,
	}

	for _, integerType := range sema.AllIntegerTypes {
		tests[integerType.String()] = `42`
	}

	for _, fixedPointType := range sema.AllFixedPointTypes {
		tests[fixedPointType.String()] = `1.23`
	}

	for ty, code := range tests {
		t.Run(fmt.Sprintf("valid: %s", ty), func(t *testing.T) {

			_, err := ParseAndCheck(t,
				fmt.Sprintf(
					`
                      let k: %s = %s
                      let xs = {k: "x"}
                    `,
					ty,
					code,
				),
			)

			require.NoError(t, err)
		})
	}

	for name, code := range map[string]string{
		"struct": `
           struct X {}
           let k = X()
        `,
		"array":      `let k = [1]`,
		"dictionary": `let k = {"a": 1}`,
	} {
		t.Run(fmt.Sprintf("invalid: %s", name), func(t *testing.T) {

			_, err := ParseAndCheck(t,
				fmt.Sprintf(
					`
                      %s
                      let xs = {k: "x"}
                    `,
					code,
				),
			)

			errs := RequireCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.InvalidDictionaryKeyTypeError{}, errs[0])
		})
	}
}

func TestNilAssignmentToDictionary(t *testing.T) {

	t.Parallel()

	t.Run("non-nillable value space", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            let x: {String: Int} = {"def": 42, "abc": 23}
            fun test() {
                x["def"] = nil
            }
	    `)

		require.NoError(t, err)
	})

	t.Run("nillable value space", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            let x: {String: Int?} = {"def": 42, "abc": 23}
            fun test() {
                x["def"] = nil
            }
	    `)

		require.NoError(t, err)
	})
}
