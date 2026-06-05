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
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/atree"

	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
	. "github.com/onflow/cadence/test_utils/common_utils"
	. "github.com/onflow/cadence/test_utils/sema_utils"
)

// TestInterpretIndexingExpressionTransfer tests if the indexing value
// (not the value that is indexed into) is properly transferred.
// If the indexing value is used for an assignment,
// it will be transferred into the indexed value,
// and as part of it, will get removed.
// Ensure the *copy* is removed, and *not the original*.
func TestInterpretIndexingExpressionTransfer(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndPrepare(t,
		`
          enum E: UInt8 {
              case First
              case Second
              case Third
          }

          resource R {
              let e: E
              init(e: E) {
                  self.e = e
              }
          }

          fun test(): UInt8 {
              let r <- create R(e: E.Third)
              let counts: {E: UInt64} = {}
              counts[r.e] = 42
              let res = r.e.rawValue
              destroy r
              return res
          }
        `,
	)

	result, err := inter.Invoke("test")
	require.NoError(t, err)

	require.Equal(t,
		// E.Third.rawValue
		interpreter.UInt8Value(2),
		result,
	)
}

func TestInterpretIndexingExpressionTransferReadStatement(t *testing.T) {
	t.Parallel()

	inter := parseCheckAndPrepare(t,
		`
          enum E: UInt8 {
              case A
          }

          fun test() {
              let counts: {E: String} = {}
              counts[E.A]
          }
        `,
	)

	_, err := inter.Invoke("test")
	require.NoError(t, err)

	storage := inter.Storage().(interpreter.InMemoryStorage)

	slabID, err := storage.BasicSlabStorage.GenerateSlabID(atree.AddressUndefined)
	require.NoError(t, err)

	var expectedSlabIndex atree.SlabIndex
	binary.BigEndian.PutUint64(expectedSlabIndex[:], 5)

	require.Equal(
		t,
		atree.NewSlabID(
			atree.AddressUndefined,
			expectedSlabIndex,
		),
		slabID,
	)
}

func TestInterpretIndexingExpressionTransferReadExpression(t *testing.T) {
	t.Parallel()

	inter := parseCheckAndPrepare(t,
		`
          enum E: UInt8 {
              case A
          }

          fun test() {
              let counts: {E: String} = {}
              let count = counts[E.A]
          }
        `,
	)

	_, err := inter.Invoke("test")
	require.NoError(t, err)

	storage := inter.Storage().(interpreter.InMemoryStorage)

	slabID, err := storage.BasicSlabStorage.GenerateSlabID(atree.AddressUndefined)
	require.NoError(t, err)

	var expectedSlabIndex atree.SlabIndex
	binary.BigEndian.PutUint64(expectedSlabIndex[:], 5)

	require.Equal(
		t,
		atree.NewSlabID(
			atree.AddressUndefined,
			expectedSlabIndex,
		),
		slabID,
	)
}

func TestInterpretIndexingExpressionTransferWrite(t *testing.T) {
	t.Parallel()

	inter := parseCheckAndPrepare(t,
		`
          enum E: UInt8 {
              case A
          }

          fun test() {
              let counts: {E: String} = {}
              counts[E.A] = "A"
          }
        `,
	)

	_, err := inter.Invoke("test")
	require.NoError(t, err)

	storage := inter.Storage().(interpreter.InMemoryStorage)

	slabID, err := storage.BasicSlabStorage.GenerateSlabID(atree.AddressUndefined)
	require.NoError(t, err)

	var expectedSlabIndex atree.SlabIndex
	binary.BigEndian.PutUint64(expectedSlabIndex[:], 6)

	require.Equal(
		t,
		atree.NewSlabID(
			atree.AddressUndefined,
			expectedSlabIndex,
		),
		slabID,
	)
}

func TestInterpretIndexingExpressionTransferSwapStatement(t *testing.T) {
	t.Parallel()

	inter := parseCheckAndPrepare(t, `
      enum E: UInt8 {
          case A
          case B
      }

      fun test(): String {
          let strings: {E: String} = {E.A: "A", E.B: "B"}
          strings[E.A] <-> strings[E.B]
          return "\(strings[E.A] ?? "")-\(strings[E.B] ?? "")"
      }
    `)

	result, err := inter.Invoke("test")
	require.NoError(t, err)

	storage := inter.Storage().(interpreter.InMemoryStorage)

	require.Equal(
		t,
		interpreter.NewUnmeteredStringValue("B-A"),
		result,
	)

	slabID, err := storage.BasicSlabStorage.GenerateSlabID(atree.AddressUndefined)
	require.NoError(t, err)

	var expectedSlabIndex atree.SlabIndex
	binary.BigEndian.PutUint64(expectedSlabIndex[:], 17)

	require.Equal(
		t,
		atree.NewSlabID(
			atree.AddressUndefined,
			expectedSlabIndex,
		),
		slabID,
	)
}

// TestInterpretSwapIndexOnDictionaryReference exercises non-resource
// through-reference IndexExpression swap: `dictRef[k1] <-> dictRef[k2]`
// where `dictRef` is `auth(Mutate) &{String: AnyStruct}`.
//
// Pre-B2 the runtime used GetIndex+NewRef for the swap operands and
// SetIndex stored a NonStorable-wrapped reference into the dictionary
// slot. The first set also evicted the slab held by the second
// operand's view, surfacing as InvalidatedContainerViewError at the
// second set (caught by the MutationCount detection added on this
// branch).
//
// Post-B2 the runtime uses extract-then-write (RemoveKey + placeholder)
// for IndexExpression swap operands; sema records target types (not
// cascaded reference types) in SwapStatementTypes so TransferAndConvert
// accepts the extracted underlying values. See sema/check_swap.go.
func TestInterpretSwapIndexOnDictionaryReference(t *testing.T) {
	t.Parallel()

	t.Run("completes without error", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            struct Foo {
                var array: [Int]
                init() {
                    self.array = []
                }
            }

            fun test() {
                let dict: {String: AnyStruct} = {"foo": Foo(), "bar": Foo()}
                let dictRef = &dict as auth(Mutate) &{String: AnyStruct}

                dictRef["foo"] <-> dictRef["bar"]
            }
        `)

		_, err := inter.Invoke("test")
		require.NoError(t, err)
	})

	t.Run("stores values, not references", func(t *testing.T) {
		t.Parallel()

		// Reading via the direct (non-reference) dict variable returns
		// the dict's actual element type (AnyStruct?). The `as! Foo` cast
		// only succeeds when the slot holds a `Foo`, not a `&Foo`.
		//
		// Pre-B2: cast fails with "expected `Foo`, got `&Foo`".
		// Post-B2: cast succeeds; the swap exchanged actual values.
		inter := parseCheckAndPrepare(t, `
            struct Foo {
                var marker: Int
                init(_ m: Int) { self.marker = m }
            }

            fun test(): Int {
                let dict: {String: AnyStruct} = {"foo": Foo(11), "bar": Foo(22)}
                let dictRef = &dict as auth(Mutate) &{String: AnyStruct}

                dictRef["foo"] <-> dictRef["bar"]

                // Read via the underlying dict (not through dictRef), which
                // returns AnyStruct? — castable to Foo iff the slot holds a
                // real Foo value, not a reference.
                let f = (dict["foo"]! as! Foo).marker
                let b = (dict["bar"]! as! Foo).marker

                // Encode the swap result as a single number so the assertion
                // is order-sensitive: 22 at "foo", 11 at "bar".
                return f * 100 + b
            }
        `)

		result, err := inter.Invoke("test")
		require.NoError(t, err)
		assert.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(2211), result)
	})
}

// TestCheckSwapMemberOnCompositeReferenceRejected does two things:
//
//  1. Pins the static rejection of external field-write swaps via a
//     reference. Both sides of `compositeRef.field1 <-> compositeRef.field2`
//     count as assignments, and sema's `isWriteableMember` denies them
//     outside the containing type. This is one of three sema-level
//     closures documented in sema/check_swap.go that make the
//     MemberExpression swap hazard unreachable in practice.
//
//  2. After suppressing the checker errors, invokes the function at
//     runtime to verify the interpreter still produces correct,
//     non-corrupted output. This exercises the defense-in-depth
//     unconditional MemberExpression marking in sema/check_swap.go:
//     if the sema closure above were ever relaxed, the runtime would
//     stay correct.
func TestCheckSwapMemberOnCompositeReferenceRejected(t *testing.T) {
	t.Parallel()

	inter, err := parseCheckAndPrepareWithOptions(t, `
        struct Foo {
            var marker: Int
            init(_ m: Int) { self.marker = m }
        }

        struct Container {
            access(all) var foo: AnyStruct
            access(all) var bar: AnyStruct
            init() {
                self.foo = Foo(11)
                self.bar = Foo(22)
            }
        }

        fun test(): Int {
            let c = Container()
            let cRef = &c as &Container

            cRef.foo <-> cRef.bar

            let f = (c.foo as! Foo).marker
            let b = (c.bar as! Foo).marker

            return f * 100 + b
        }
    `, ParseCheckAndInterpretOptions{
		HandleCheckerError: func(err error) {
			errs := RequireCheckerErrors(t, err, 2)
			require.IsType(t, &sema.InvalidAssignmentAccessError{}, errs[0])
			require.IsType(t, &sema.InvalidAssignmentAccessError{}, errs[1])
		},
	})
	require.NoError(t, err)

	// Even though sema rejected this program (errors swallowed
	// above), invoke at runtime to verify the interpreter would
	// still produce a correct, non-corrupted result if the sema
	// closure were ever relaxed.
	result, err := inter.Invoke("test")
	require.NoError(t, err)
	assert.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(2211), result)
}

// TestInterpretSwapMemberOnCompositeFromMethod exercises the in-method
// MemberExpression swap `self.foo <-> self.bar` where the fields are
// `AnyStruct`-typed. This is the path that survives sema's
// InvalidAssignmentAccessError check (see
// TestCheckSwapMemberOnCompositeReferenceRejected), because writes
// from within the containing type are unconditionally permitted.
//
// Despite the OrderedMap-backed field storage being structurally
// identical to a dictionary, this case carries no through-reference
// hazard: for struct/resource methods `self` is declared as the
// CompositeType itself (see sema.declareSelfValue), not a reference,
// so member reads do not cascade auth and do not return references
// into the composite's inlined slab. Attachments are the only
// composites where `self` is a reference (see
// TestInterpretSwapMemberOnAttachmentFromMethod).
func TestInterpretSwapMemberOnCompositeFromMethod(t *testing.T) {
	t.Parallel()

	t.Run("completes without error", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            struct Foo {
                var array: [Int]
                init() {
                    self.array = []
                }
            }

            struct Container {
                access(all) var foo: AnyStruct
                access(all) var bar: AnyStruct
                init() {
                    self.foo = Foo()
                    self.bar = Foo()
                }
                access(all) fun swap() {
                    self.foo <-> self.bar
                }
            }

            fun test() {
                let c = Container()
                c.swap()
            }
        `)

		_, err := inter.Invoke("test")
		require.NoError(t, err)
	})

	t.Run("stores values, not references", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            struct Foo {
                var marker: Int
                init(_ m: Int) { self.marker = m }
            }

            struct Container {
                access(all) var foo: AnyStruct
                access(all) var bar: AnyStruct
                init() {
                    self.foo = Foo(11)
                    self.bar = Foo(22)
                }
                access(all) fun swap() {
                    self.foo <-> self.bar
                }
            }

            fun test(): Int {
                let c = Container()
                c.swap()

                let f = (c.foo as! Foo).marker
                let b = (c.bar as! Foo).marker

                return f * 100 + b
            }
        `)

		result, err := inter.Invoke("test")
		require.NoError(t, err)
		assert.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(2211), result)
	})
}

// TestInterpretSwapMemberOnAttachmentFromMethod covers in-method
// swap on an attachment, the one composite kind where `self` is a
// reference (`auth(...) &A`, see sema.declareSelfValue's
// CompositeKindAttachment branch). On paper this is structurally
// identical to the IndexExpression-on-dictionary-reference hazard:
// reads through a reference, then two writes.
//
// In practice it is closed at sema: visitMember in
// check_member_expression.go has an explicit special case that
// suppresses the reference cascade for `self.<member>` (the
// `accessedSelfMember == nil` guard), so reads return the underlying
// value and the SetField stores actual values. The test pins that
// post-swap reads yield the swapped underlying values, not stale
// references.
func TestInterpretSwapMemberOnAttachmentFromMethod(t *testing.T) {
	t.Parallel()

	t.Run("completes without error", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            struct Foo {
                var array: [Int]
                init() {
                    self.array = []
                }
            }

            resource R {}

            attachment A for R {
                access(all) var foo: AnyStruct
                access(all) var bar: AnyStruct
                init() {
                    self.foo = Foo()
                    self.bar = Foo()
                }
                access(all) fun doSwap() {
                    self.foo <-> self.bar
                }
            }

            fun test() {
                let r <- create R()
                let r2 <- attach A() to <-r
                r2[A]!.doSwap()
                destroy r2
            }
        `)

		_, err := inter.Invoke("test")
		require.NoError(t, err)
	})

	t.Run("swap exchanges values across fields", func(t *testing.T) {
		t.Parallel()

		// Reading the attachment's AnyStruct fields from outside the
		// attachment is impossible without a reference cascade
		// (attachments are reference-only), so a direct `as! [Int]`
		// cast on the post-swap state can't be observed. Instead the
		// observation is folded into the attachment's own methods,
		// where the array reference is well-defined: read the first
		// element of each field after the swap and encode both into a
		// single Int. Pre-fix: the swap would store references and the
		// reads either fail or return the unswapped 11/22; post-fix:
		// values are exchanged and the reads return 22/11.
		inter := parseCheckAndPrepare(t, `
            resource R {}

            attachment A for R {
                access(all) var foo: AnyStruct
                access(all) var bar: AnyStruct
                init() {
                    self.foo = [11]
                    self.bar = [22]
                }
                access(all) fun doSwap() {
                    self.foo <-> self.bar
                }
                access(all) fun probe(): Int {
                    // If swap stored actual values, the cast to [Int]
                    // succeeds. If swap stored references (pre-fix),
                    // the cast fails or the post-swap state is unreadable.
                    let fooArr = self.foo as! [Int]
                    let barArr = self.bar as! [Int]
                    return fooArr[0] * 100 + barArr[0]
                }
            }

            fun test(): Int {
                let r <- create R()
                let r2 <- attach A() to <-r
                r2[A]!.doSwap()
                let probed = r2[A]!.probe()
                destroy r2
                return probed
            }
        `)

		result, err := inter.Invoke("test")
		require.NoError(t, err)
		assert.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(2211), result)
	})
}

func TestInterpretIndexingTypeConfusedValue(t *testing.T) {
	t.Parallel()

	t.Run("get", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
          fun test(indexable: [String?]): String? {
              return indexable[0]
          }
        `)

		value := interpreter.NewDictionaryValue(
			inter,
			interpreter.NewDictionaryStaticType(
				nil,
				interpreter.PrimitiveStaticTypeInt,
				interpreter.PrimitiveStaticTypeString,
			),
			interpreter.NewUnmeteredIntValueFromInt64(0),
			interpreter.NewUnmeteredStringValue("foo"),
		)

		// Intentionally passing wrong type of value
		_, err := inter.InvokeUncheckedForTestingOnly("test", value) //nolint:staticcheck
		RequireError(t, err)

		var indexedTypeError *interpreter.IndexedTypeError
		require.ErrorAs(t, err, &indexedTypeError)
	})

	t.Run("set", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
          fun test(indexable: [String?]) {
              indexable[0] = "foo"
          }
        `)

		value := interpreter.NewDictionaryValue(
			inter,
			interpreter.NewDictionaryStaticType(
				nil,
				interpreter.PrimitiveStaticTypeInt,
				interpreter.PrimitiveStaticTypeString,
			),
		)

		// Intentionally passing wrong type of value
		_, err := inter.InvokeUncheckedForTestingOnly("test", value) //nolint:staticcheck
		RequireError(t, err)

		var indexedTypeError *interpreter.IndexedTypeError
		require.ErrorAs(t, err, &indexedTypeError)
	})
}

func TestInterpretTypeIndexingTypeConfusedValue(t *testing.T) {
	t.Parallel()

	t.Run("get", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
          struct S {}
          struct T {}

          attachment A for S {}

          fun test(indexable: S): AnyStruct {
              return indexable[A]
          }
        `)

		value := interpreter.NewCompositeValue(
			inter,
			TestLocation,
			"T",
			common.CompositeKindStructure,
			nil,
			common.ZeroAddress,
		)

		// Intentionally passing wrong type of value
		_, err := inter.InvokeUncheckedForTestingOnly("test", value) //nolint:staticcheck
		RequireError(t, err)

		var indexedTypeError *interpreter.IndexedTypeError
		require.ErrorAs(t, err, &indexedTypeError)
	})

	t.Run("attach", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
          struct S {}
          struct T {}

          attachment A for S {}

          fun test(indexable: S) {
              attach A() to indexable
          }
        `)

		value := interpreter.NewCompositeValue(
			inter,
			TestLocation,
			"T",
			common.CompositeKindStructure,
			nil,
			common.ZeroAddress,
		)

		// Intentionally passing wrong type of value
		_, err := inter.InvokeUncheckedForTestingOnly("test", value) //nolint:staticcheck
		RequireError(t, err)

		// The base is set in the attachment first (before the attachment is set to the base).
		// So the invalid base type error occurs first.
		var invalidBaseTypeError *interpreter.InvalidBaseTypeError
		require.ErrorAs(t, err, &invalidBaseTypeError)
	})

	t.Run("remove", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
          struct S {}
          struct T {}

          attachment A for S {}

          fun test(indexable: S) {
              remove A from indexable
          }
        `)

		value := interpreter.NewCompositeValue(
			inter,
			TestLocation,
			"T",
			common.CompositeKindStructure,
			nil,
			common.ZeroAddress,
		)

		// Intentionally passing wrong type of value
		_, err := inter.InvokeUncheckedForTestingOnly("test", value) //nolint:staticcheck
		RequireError(t, err)

		var indexedTypeError *interpreter.IndexedTypeError
		require.ErrorAs(t, err, &indexedTypeError)
	})
}
