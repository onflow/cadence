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
 */

package interpreter_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/activations"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/stdlib"
	"github.com/onflow/cadence/test_utils"
	. "github.com/onflow/cadence/test_utils/sema_utils"
)

// TestInterpretStaleWrapperMutationRejected covers the case where two Go-level
// wrappers (ArrayValue/DictionaryValue/CompositeValue) for the same atree
// container are created (via repeated `&outer[i]` style access), one wrapper
// triggers a slab split, and the sibling wrapper subsequently attempts a
// mutation. Without the staleness check, the mutation writes into the demoted
// (now-leaf) slab and leaves the canonical view of the container out of sync
// with the live data, manifesting as an element that is invisible to
// iteration but visible to consecutive removals — a clear violation of
// resource semantics.
func TestInterpretStaleWrapperMutationRejected(t *testing.T) {
	t.Parallel()

	makeEnv := func(t *testing.T) (
		*sema.VariableActivation,
		*activations.Activation[interpreter.Variable],
	) {

		// liveValueIDOf exposes the underlying atree container's current value ID
		// so the Cadence code can confirm the slab split actually occurred
		// before attempting the stale-wrapper mutation.
		//
		// It takes the *name* of the reference variable rather than the
		// reference itself: passing a stale reference as a function argument
		// would trip the staleness check during expression evaluation (every
		// expression result goes through CheckInvalidatedValueOrValueReference,
		// which recursively descends into reference values), so the check would
		// fire at the call-site rather than the mutation. Resolving the
		// variable internally via GetValueOfVariable bypasses the per-
		// expression check.
		liveValueIDOfFunction := stdlib.NewInterpreterStandardLibraryStaticFunction(
			"liveValueIDOf",
			sema.NewSimpleFunctionType(
				sema.FunctionPurityImpure,
				[]sema.Parameter{
					{
						Label:          sema.ArgumentLabelNotRequired,
						Identifier:     "name",
						TypeAnnotation: sema.StringTypeAnnotation,
					},
				},
				sema.StringTypeAnnotation,
			),
			"",
			func(
				context interpreter.NativeFunctionContext,
				_ interpreter.TypeArgumentsIterator,
				_ interpreter.ArgumentTypesIterator,
				_ interpreter.Value,
				args []interpreter.Value,
			) interpreter.Value {
				name := args[0].(*interpreter.StringValue).Str
				value := context.GetValueOfVariable(name)
				ref := value.(*interpreter.EphemeralReferenceValue)
				var id string
				switch v := ref.Value.(type) {
				case *interpreter.ArrayValue:
					id = v.LiveValueID().String()
				case *interpreter.DictionaryValue:
					id = v.LiveValueID().String()
				case *interpreter.CompositeValue:
					id = v.LiveValueID().String()
				default:
					t.Fatalf("unexpected value type %T", ref.Value)
				}
				return interpreter.NewUnmeteredStringValue(id)
			},
		)

		// liveInlinedOf exposes whether the underlying atree container of a
		// reference variable is currently stored inlined inside its parent's
		// slab. Atree may transition a container between inlined and
		// standalone-slab storage when its parent grows or shrinks. Such
		// transitions are NOT observable through LiveValueID (atree assigns
		// a stable ValueID across inline/uninline transitions), so this
		// helper is needed to assert that uninlining actually occurred in
		// tests that probe the safety of the post-uninlining state.
		liveInlinedOfFunction := stdlib.NewInterpreterStandardLibraryStaticFunction(
			"liveInlinedOf",
			sema.NewSimpleFunctionType(
				sema.FunctionPurityImpure,
				[]sema.Parameter{
					{
						Label:          sema.ArgumentLabelNotRequired,
						Identifier:     "name",
						TypeAnnotation: sema.StringTypeAnnotation,
					},
				},
				sema.BoolTypeAnnotation,
			),
			"",
			func(
				context interpreter.NativeFunctionContext,
				_ interpreter.TypeArgumentsIterator,
				_ interpreter.ArgumentTypesIterator,
				_ interpreter.Value,
				args []interpreter.Value,
			) interpreter.Value {
				name := args[0].(*interpreter.StringValue).Str
				value := context.GetValueOfVariable(name)
				ref := value.(*interpreter.EphemeralReferenceValue)
				var inlined bool
				switch v := ref.Value.(type) {
				case *interpreter.ArrayValue:
					inlined = v.LiveInlined()
				case *interpreter.DictionaryValue:
					inlined = v.LiveInlined()
				case *interpreter.CompositeValue:
					inlined = v.LiveInlined()
				default:
					t.Fatalf("unexpected value type %T", ref.Value)
				}
				return interpreter.BoolValue(inlined)
			},
		)

		baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
		baseValueActivation.DeclareValue(liveValueIDOfFunction)
		baseValueActivation.DeclareValue(liveInlinedOfFunction)
		baseValueActivation.DeclareValue(stdlib.InterpreterAssertFunction)

		baseActivation := activations.NewActivation(nil, interpreter.BaseActivation)
		interpreter.Declare(baseActivation, liveValueIDOfFunction)
		interpreter.Declare(baseActivation, liveInlinedOfFunction)
		interpreter.Declare(baseActivation, stdlib.InterpreterAssertFunction)

		return baseValueActivation, baseActivation
	}

	runInvoke := func(t *testing.T, code string) error {
		baseValueActivation, baseActivation := makeEnv(t)
		inter, err := test_utils.ParseCheckAndInterpretWithAtreeValidationsDisabled(
			t,
			code,
			ParseCheckAndInterpretOptions{
				ParseAndCheckOptions: &ParseAndCheckOptions{
					CheckerConfig: &sema.Config{
						BaseValueActivationHandler: func(_ common.Location) *sema.VariableActivation {
							return baseValueActivation
						},
					},
				},
				InterpreterConfig: &interpreter.Config{
					BaseActivationHandler: func(_ common.Location) *activations.Activation[interpreter.Variable] {
						return baseActivation
					},
				},
			},
		)
		require.NoError(t, err)
		_, err = inter.Invoke("main")
		return err
	}

	t.Run("ArrayValue: append via stale wrapper after split", func(t *testing.T) {
		t.Parallel()

		// Two ArrayValue wrappers point to the same inner inlined-then-grown
		// array. `ref` appends enough elements to trigger an atree slab split.
		// `ref2`'s `*atree.Array` still points to the now-demoted old root data
		// slab. The second mutation through `ref2` must be rejected with
		// InvalidatedContainerViewError; otherwise an element ends up parked on the
		// demoted slab, hidden from iteration but exposed via removeLast.
		err := runInvoke(t, `
            access(all) resource Vault {
                access(all) var balance: UFix64
                init(balance: UFix64) { self.balance = balance }
            }

            access(all) fun main() {
                let outer: @[[Vault]] <- [<-[<-create Vault(balance: 0.0)]]

                let ref  = &outer[0] as auth(Mutate) &[Vault]
                let ref2 = &outer[0] as auth(Mutate) &[Vault]

                assert(
                    liveValueIDOf("ref") == liveValueIDOf("ref2"),
                    message: "before split: both refs should observe the same live atree value ID"
                )

                var i: Int = 0
                while i < 200 {
                    ref.append(<-create Vault(balance: UFix64(i)))
                    i = i + 1
                }

                assert(
                    liveValueIDOf("ref") != liveValueIDOf("ref2"),
                    message: "after split: refs should observe diverged live atree value IDs"
                )

                // This mutation goes through the stale wrapper and must be rejected.
                ref2.append(<- create Vault(balance: 123.456))

                destroy outer
            }
        `)
		var staleViewErr *interpreter.InvalidatedContainerViewError
		assert.ErrorAs(t, err, &staleViewErr)
		assert.Equal(t, 30, staleViewErr.StartPosition().Line)
	})

	t.Run("ArrayValue: insert via stale wrapper after split", func(t *testing.T) {
		t.Parallel()

		err := runInvoke(t, `
            access(all) resource Vault {
                access(all) var balance: UFix64
                init(balance: UFix64) { self.balance = balance }
            }

            access(all) fun main() {
                let outer: @[[Vault]] <- [<-[<-create Vault(balance: 0.0)]]

                let ref  = &outer[0] as auth(Mutate) &[Vault]
                let ref2 = &outer[0] as auth(Mutate) &[Vault]

                assert(
                    liveValueIDOf("ref") == liveValueIDOf("ref2"),
                    message: "before split: both refs should observe the same live atree value ID"
                )

                var i: Int = 0
                while i < 200 {
                    ref.append(<-create Vault(balance: UFix64(i)))
                    i = i + 1
                }

                assert(
                    liveValueIDOf("ref") != liveValueIDOf("ref2"),
                    message: "after split: refs should observe diverged live atree value IDs"
                )

                ref2.insert(at: 0, <- create Vault(balance: 123.456))

                destroy outer
            }
        `)
		var staleViewErr *interpreter.InvalidatedContainerViewError
		assert.ErrorAs(t, err, &staleViewErr)
		assert.Equal(t, 29, staleViewErr.StartPosition().Line)
	})

	t.Run("ArrayValue: remove via stale wrapper after split", func(t *testing.T) {
		t.Parallel()

		err := runInvoke(t, `
            access(all) resource Vault {
                access(all) var balance: UFix64
                init(balance: UFix64) { self.balance = balance }
            }

            access(all) fun main() {
                let outer: @[[Vault]] <- [<-[<-create Vault(balance: 0.0)]]

                let ref  = &outer[0] as auth(Mutate) &[Vault]
                let ref2 = &outer[0] as auth(Mutate) &[Vault]

                assert(
                    liveValueIDOf("ref") == liveValueIDOf("ref2"),
                    message: "before split: both refs should observe the same live atree value ID"
                )

                var i: Int = 0
                while i < 200 {
                    ref.append(<-create Vault(balance: UFix64(i)))
                    i = i + 1
                }

                assert(
                    liveValueIDOf("ref") != liveValueIDOf("ref2"),
                    message: "after split: refs should observe diverged live atree value IDs"
                )

                let extra <- ref2.remove(at: 0)
                destroy extra

                destroy outer
            }
        `)
		var staleViewErr *interpreter.InvalidatedContainerViewError
		assert.ErrorAs(t, err, &staleViewErr)
		assert.Equal(t, 29, staleViewErr.StartPosition().Line)
	})

	t.Run("DictionaryValue: insert via stale wrapper after split", func(t *testing.T) {
		t.Parallel()

		err := runInvoke(t, `
            access(all) resource Vault {
                access(all) var balance: UFix64
                init(balance: UFix64) { self.balance = balance }
            }

            access(all) fun main() {
                let outer: @[{Int: Vault}] <- [<-{0: <-create Vault(balance: 0.0)}]

                let ref  = &outer[0] as auth(Mutate) &{Int: Vault}
                let ref2 = &outer[0] as auth(Mutate) &{Int: Vault}

                assert(
                    liveValueIDOf("ref") == liveValueIDOf("ref2"),
                    message: "before split: both refs should observe the same live atree value ID"
                )

                var i: Int = 1
                while i < 300 {
                    let old <- ref.insert(key: i, <-create Vault(balance: UFix64(i)))
                    assert(old == nil, message: "dict insert should not collide")
                    destroy old
                    i = i + 1
                }

                assert(
                    liveValueIDOf("ref") != liveValueIDOf("ref2"),
                    message: "after split: refs should observe diverged live atree value IDs"
                )

                let old2 <- ref2.insert(key: 9999, <- create Vault(balance: 123.456))
                destroy old2

                destroy outer
            }
        `)
		var staleViewErr *interpreter.InvalidatedContainerViewError
		assert.ErrorAs(t, err, &staleViewErr)
		assert.Equal(t, 31, staleViewErr.StartPosition().Line)
	})

	t.Run("CompositeValue: field assignment via stale wrapper after split", func(t *testing.T) {
		t.Parallel()

		// Two CompositeValue wrappers point to the same Vault resource (via two
		// `&arr[0]` references). `ref` inflates attachments enough to split the
		// resource's underlying atree map; `ref2` is now a stale view. The
		// subsequent field assignment through `ref2` must be rejected.
		err := runInvoke(t, `
            access(all) entitlement Mod
            access(all) resource R {
                access(all) var balance: UFix64
                init(balance: UFix64) { self.balance = balance }
                access(Mod) fun setBalance(_ v: UFix64) { self.balance = v }
            }

            access(all) attachment A1 for R {
                access(all) var a1: String; access(all) var a2: String
                access(all) var a3: String; access(all) var a4: String
                init() { self.a1 = ""; self.a2 = ""; self.a3 = ""; self.a4 = "" }
                access(all) fun inflate() {
                    self.a1 = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
                    self.a2 = self.a1; self.a3 = self.a1; self.a4 = self.a1
                }
            }
            access(all) attachment A2 for R {
                access(all) var b1: String; access(all) var b2: String
                access(all) var b3: String; access(all) var b4: String
                init() { self.b1 = ""; self.b2 = ""; self.b3 = ""; self.b4 = "" }
                access(all) fun inflate() {
                    self.b1 = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
                    self.b2 = self.b1; self.b3 = self.b1; self.b4 = self.b1
                }
            }
            access(all) attachment A3 for R {
                access(all) var d1: String; access(all) var d2: String
                access(all) var d3: String; access(all) var d4: String
                init() { self.d1 = ""; self.d2 = ""; self.d3 = ""; self.d4 = "" }
                access(all) fun inflate() {
                    self.d1 = "dddddddddddddddddddddddddddddddddddddd"
                    self.d2 = self.d1; self.d3 = self.d1; self.d4 = self.d1
                }
            }
            access(all) attachment A4 for R {
                access(all) var e1: String; access(all) var e2: String
                access(all) var e3: String; access(all) var e4: String
                init() { self.e1 = ""; self.e2 = ""; self.e3 = ""; self.e4 = "" }
                access(all) fun inflate() {
                    self.e1 = "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"
                    self.e2 = self.e1; self.e3 = self.e1; self.e4 = self.e1
                }
            }
            access(all) attachment A5 for R {
                access(all) var g1: String; access(all) var g2: String
                access(all) var g3: String; access(all) var g4: String
                init() { self.g1 = ""; self.g2 = ""; self.g3 = ""; self.g4 = "" }
                access(all) fun inflate() {
                    self.g1 = "gggggggggggggggggggggggggggggggggggggg"
                    self.g2 = self.g1; self.g3 = self.g1; self.g4 = self.g1
                }
            }
            access(all) attachment A6 for R {
                access(all) var h1: String; access(all) var h2: String
                access(all) var h3: String; access(all) var h4: String
                init() { self.h1 = ""; self.h2 = ""; self.h3 = ""; self.h4 = "" }
                access(all) fun inflate() {
                    self.h1 = "hhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhh"
                    self.h2 = self.h1; self.h3 = self.h1; self.h4 = self.h1
                }
            }

            access(all) fun main() {
                let r0 <- create R(balance: 0.0)
                let r1 <- attach A1() to <-r0
                let r2 <- attach A2() to <-r1
                let r3 <- attach A3() to <-r2
                let r4 <- attach A4() to <-r3
                let r5 <- attach A5() to <-r4
                let r  <- attach A6() to <-r5

                let arr: @[R] <- [<-r]
                let ref  = &arr[0] as auth(Mod) &R
                let ref2 = &arr[0] as auth(Mod) &R

                assert(
                    liveValueIDOf("ref") == liveValueIDOf("ref2"),
                    message: "before split: both refs should observe the same live atree value ID"
                )

                ref[A1]!.inflate()
                ref[A2]!.inflate()
                ref[A3]!.inflate()
                ref[A4]!.inflate()
                ref[A5]!.inflate()
                ref[A6]!.inflate()

                assert(
                    liveValueIDOf("ref") != liveValueIDOf("ref2"),
                    message: "after split: refs should observe diverged live atree value IDs"
                )

                // Field assignment through stale wrapper must be rejected.
                ref2.setBalance(123.456)

                destroy arr
            }
        `)
		var staleViewErr *interpreter.InvalidatedContainerViewError
		assert.ErrorAs(t, err, &staleViewErr)
		assert.Equal(t, 95, staleViewErr.StartPosition().Line)
	})

	t.Run("DictionaryValue: remove via stale wrapper after split", func(t *testing.T) {
		t.Parallel()

		err := runInvoke(t, `
            access(all) resource Vault {
                access(all) var balance: UFix64
                init(balance: UFix64) { self.balance = balance }
            }

            access(all) fun main() {
                let outer: @[{Int: Vault}] <- [<-{0: <-create Vault(balance: 0.0)}]

                let ref  = &outer[0] as auth(Mutate) &{Int: Vault}
                let ref2 = &outer[0] as auth(Mutate) &{Int: Vault}

                assert(
                    liveValueIDOf("ref") == liveValueIDOf("ref2"),
                    message: "before split: both refs should observe the same live atree value ID"
                )

                var i: Int = 1
                while i < 300 {
                    let old <- ref.insert(key: i, <-create Vault(balance: UFix64(i)))
                    destroy old
                    i = i + 1
                }

                assert(
                    liveValueIDOf("ref") != liveValueIDOf("ref2"),
                    message: "after split: refs should observe diverged live atree value IDs"
                )

                let removed <- ref2.remove(key: 0)
                destroy removed

                destroy outer
            }
        `)
		var staleViewErr *interpreter.InvalidatedContainerViewError
		assert.ErrorAs(t, err, &staleViewErr)
		assert.Equal(t, 30, staleViewErr.StartPosition().Line)
	})

	t.Run("ArrayValue: map procedure mutates sibling wrapper into split mid-iteration", func(t *testing.T) {
		t.Parallel()

		// ArrayValue.Map (and similarly Filter / Reverse / Slice / Concat / ToVariableSized)
		// creates an atree iterator once via v.array.Iterator() and walks it across many
		// user-callback invocations.
		// Between callback invocations there is:
		//   - no WithContainerMutationPrevention(v.ValueID()) (unlike Iterate);
		//   - no CheckInvalidatedValueOrValueReference(v, ...) between iterator.Next() calls.
		//
		// If the user procedure mutates a sibling wrapper of v, the sibling
		// mutation triggers a slab split. The active atree iterator continues
		// walking the now-demoted slab, potentially yielding duplicate elements,
		// skipping elements, or yielding stale state — all silently.
		//
		// Safe contract: as soon as v becomes stale, the next iterator.Next()
		// (or a per-iteration staleness check) must raise InvalidatedContainerViewError.
		//
		// Non-resource Int elements are used because Cadence forbids
		// `map` on resource arrays even when accessed via reference. The
		// iterator-corruption gap is independent of resource-ness.
		err := runInvoke(t, `
            access(all) fun main() {
                // Two ArrayValue wrappers for the same inner Int array.
                let outer: [[Int]] = [[0, 1]]

                let ref1  = &outer[0] as auth(Mutate) &[Int]
                let ref2 = &outer[0] as auth(Mutate) &[Int]

                assert(
                    liveValueIDOf("ref1") == liveValueIDOf("ref2"),
                    message: "before split: both refs should observe the same live atree value ID"
                )

                // ref.map's atree iterator is created when this expression begins evaluating.
                // The procedure runs between iterator.Next() calls.
                // the first invocation mutates ref2 enough to split the slab tree, demoting ref1's view.
                // The map's iterator should then be rejected.
                var calls: Int = 0
                let mapped = ref1.map(fun (v: Int): Int {
                    if calls == 0 {
                        // Push ref2 past the slab-split threshold while ref's
                        // iterator is paused between elements.
                        var j = 0
                        while j < 300 {
                            ref2.append(j + 100)
                            j = j + 1
                        }
                    }
                    calls = calls + 1
                    return v
                })

                // If the check fires correctly, execution never reaches this point.
                // If the gap is unpatched, ref's wrapper is now stale and mapped
                // contains corrupt data (wrong length, duplicates, or stale reads).
                assert(
                    liveValueIDOf("ref1") != liveValueIDOf("ref2"),
                    message: "after callback-induced split: refs should observe diverged live atree value IDs"
                )

                // Without the safety check, this is silent corruption: the canonical
                // view sees 302 elements (2 original + 300 appended); a stale map
                // iterator may yield a different count or duplicate the first element.
                assert(
                    mapped.length == 302,
                    message: "mapped length mismatch — stale iterator yielded wrong element count, got "
                        .concat(mapped.length.toString())
                )
            }
        `)
		var staleViewErr *interpreter.InvalidatedContainerViewError
		assert.ErrorAs(t, err, &staleViewErr)
	})

	// When an attachment method is bound via `composite[B]`, the bound function
	// captures `base = v.GetBaseValue(...)`, a fresh `EphemeralReferenceValue`
	// pointing at the parent composite. `MaybeDereferenceReceiver` at invoke
	// time only validates the *attachment* receiver, not the captured base.
	//
	// The safe contract is that any code path that ultimately walks back through
	// the parent via `base` must re-trigger the staleness check on the parent
	// wrapper — either at the index expression `ref2[B]` (because the stale
	// ref2 is evaluated as the index target) or inside the method body when
	// `base.X` is evaluated.
	//
	// Both directions are exercised by the two sub-tests below. They currently
	// PASS, confirming Gap 3 is fully covered by the centralized check; they
	// serve as positive regression coverage so a future refactor that removes
	// either the index-target check or the in-method `base` re-check would be
	// caught.

	typeDeclarations := `
        access(all) resource R {
            access(all) var balance: UFix64
            init(balance: UFix64) { self.balance = balance }
        }

        access(all) attachment A1 for R {
            access(all) var a1: String; access(all) var a2: String
            access(all) var a3: String; access(all) var a4: String
            init() { self.a1 = ""; self.a2 = ""; self.a3 = ""; self.a4 = "" }
            access(all) fun inflate() {
                self.a1 = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
                self.a2 = self.a1; self.a3 = self.a1; self.a4 = self.a1
            }
        }
        access(all) attachment A2 for R {
            access(all) var b1: String; access(all) var b2: String
            access(all) var b3: String; access(all) var b4: String
            init() { self.b1 = ""; self.b2 = ""; self.b3 = ""; self.b4 = "" }
            access(all) fun inflate() {
                self.b1 = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
                self.b2 = self.b1; self.b3 = self.b1; self.b4 = self.b1
            }
        }
        access(all) attachment A3 for R {
            access(all) var d1: String; access(all) var d2: String
            access(all) var d3: String; access(all) var d4: String
            init() { self.d1 = ""; self.d2 = ""; self.d3 = ""; self.d4 = "" }
            access(all) fun inflate() {
                self.d1 = "dddddddddddddddddddddddddddddddddddddd"
                self.d2 = self.d1; self.d3 = self.d1; self.d4 = self.d1
            }
        }
        access(all) attachment A4 for R {
            access(all) var e1: String; access(all) var e2: String
            access(all) var e3: String; access(all) var e4: String
            init() { self.e1 = ""; self.e2 = ""; self.e3 = ""; self.e4 = "" }
            access(all) fun inflate() {
                self.e1 = "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"
                self.e2 = self.e1; self.e3 = self.e1; self.e4 = self.e1
            }
        }
        access(all) attachment A5 for R {
            access(all) var g1: String; access(all) var g2: String
            access(all) var g3: String; access(all) var g4: String
            init() { self.g1 = ""; self.g2 = ""; self.g3 = ""; self.g4 = "" }
            access(all) fun inflate() {
                self.g1 = "gggggggggggggggggggggggggggggggggggggg"
                self.g2 = self.g1; self.g3 = self.g1; self.g4 = self.g1
            }
        }
        access(all) attachment A6 for R {
            access(all) var h1: String; access(all) var h2: String
            access(all) var h3: String; access(all) var h4: String
            init() { self.h1 = ""; self.h2 = ""; self.h3 = ""; self.h4 = "" }
            access(all) fun inflate() {
                self.h1 = "hhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhh"
                self.h2 = self.h1; self.h3 = self.h1; self.h4 = self.h1
            }
        }

        // Attachment B exposes a method that reads back through base.
        // A correct invariant requires the staleness check to fire any time
        // base ultimately walks a stale parent wrapper.
        access(all) attachment B for R {
            access(all) fun readBaseBalance(): UFix64 {
                return base.balance
            }
        }
    `

	t.Run("CompositeValue: attachment method via stale parent wrapper (direct index)", func(t *testing.T) {
		t.Parallel()

		// Scenario A: invoke an attachment method directly through the stale
		// parent wrapper. The check should fire when ref2 is evaluated as the
		// target of the index expression `ref2[B]`.
		err := runInvoke(t, typeDeclarations+`
            access(all) fun main() {
                let r0 <- create R(balance: 42.0)
                let r1 <- attach A1() to <-r0
                let r2 <- attach A2() to <-r1
                let r3 <- attach A3() to <-r2
                let r4 <- attach A4() to <-r3
                let r5 <- attach A5() to <-r4
                let r6 <- attach A6() to <-r5
                let r  <- attach B()  to <-r6

                let arr: @[R] <- [<-r]
                let ref1  = &arr[0] as &R
                let ref2 = &arr[0] as &R

                assert(
                    liveValueIDOf("ref1") == liveValueIDOf("ref2"),
                    message: "before split: both refs should observe the same live atree value ID"
                )

                ref1[A1]!.inflate()
                ref1[A2]!.inflate()
                ref1[A3]!.inflate()
                ref1[A4]!.inflate()
                ref1[A5]!.inflate()
                ref1[A6]!.inflate()

                assert(
                    liveValueIDOf("ref1") != liveValueIDOf("ref2"),
                    message: "after split: refs should observe diverged live atree value IDs"
                )

                // The stale ref2 is the target of the index expression and is
                // evaluated first by evalExpression, which runs
                // CheckInvalidatedValueOrValueReference and fires
                // InvalidatedContainerViewError before the attachment lookup or
                // method binding can complete.
                let stashed = ref2[B]!.readBaseBalance()
                assert(stashed == 42.0, message: "unreachable if check fires")

                destroy arr
            }
        `)
		var staleViewErr *interpreter.InvalidatedContainerViewError
		assert.ErrorAs(t, err, &staleViewErr)
	})

	t.Run("CompositeValue: attachment method via stale parent wrapper (captured-before-split)", func(t *testing.T) {
		t.Parallel()

		// Scenario B: capture the attachment reference BEFORE the split, then
		// trigger the split through ref, then invoke the attachment method via
		// the captured reference. ref2 no longer appears in the post-split code
		// path, so the check must fire elsewhere — inside the method body, when
		// `base` is evaluated as an identifier and CheckInvalidatedValueOrValueReference
		// recurses into the captured base reference's stale parent composite.
		err := runInvoke(t, typeDeclarations+`
            access(all) fun main() {
                let r0 <- create R(balance: 42.0)
                let r1 <- attach A1() to <-r0
                let r2 <- attach A2() to <-r1
                let r3 <- attach A3() to <-r2
                let r4 <- attach A4() to <-r3
                let r5 <- attach A5() to <-r4
                let r6 <- attach A6() to <-r5
                let r  <- attach B()  to <-r6

                let arr: @[R] <- [<-r]
                let ref1  = &arr[0] as &R
                let ref2 = &arr[0] as &R

                // Capture the attachment reference BEFORE the split. Internally
                // this also wires the attachment's v.base to ref2's CompositeValue.
                let bRef = ref2[B]!

                assert(
                    liveValueIDOf("ref1") == liveValueIDOf("ref2"),
                    message: "before split: both refs should observe the same live atree value ID"
                )

                ref1[A1]!.inflate()
                ref1[A2]!.inflate()
                ref1[A3]!.inflate()
                ref1[A4]!.inflate()
                ref1[A5]!.inflate()
                ref1[A6]!.inflate()

                assert(
                    liveValueIDOf("ref1") != liveValueIDOf("ref2"),
                    message: "after split: refs should observe diverged live atree value IDs"
                )

                // The receiver bRef refers to the attachment (not stale), so
                // MaybeDereferenceReceiver does not fire on the attachment ref1.
                // The check must fire when the method body evaluates base
                // (which resolves to a reference at the now-stale parent).
                let stashed = bRef.readBaseBalance()
                assert(stashed == 42.0, message: "unreachable if check fires")

                destroy arr
            }
        `)

		var staleViewErr *interpreter.InvalidatedContainerViewError
		assert.ErrorAs(t, err, &staleViewErr)
	})

	// Resource-linearity-specific scenarios (Invariant 3) from
	// atree-slab-change-security-analysis.md. These tests re-frame the
	// previously-identified gaps through the lens of resource linearity:
	// a resource must exist in exactly one location at any time, and no
	// sibling wrapper may be used to read, mutate, or extract a resource
	// after its slab tree has been restructured.

	t.Run("ArrayValue.removeFirst via stale sibling", func(t *testing.T) {
		t.Parallel()

		// Scenario from "Destroy + sibling resurrection":
		// the canonical ref1 removes-and-destroys a resource via removeFirst,
		// then the sibling ref1's removeFirst attempt must be rejected.
		// Without the centralized staleness check, the sibling's removeFirst
		// could read from the demoted slab and yield a phantom resource —
		// a resource-linearity violation (the resource ref1 already destroyed
		// would now exist in a second location).
		err := runInvoke(t, `
            access(all) resource Vault {
                access(all) var balance: UFix64
                init(balance: UFix64) { self.balance = balance }
            }

            access(all) fun main() {
                let outer: @[[Vault]] <- [<-[<-create Vault(balance: 1.0)]]

                let ref1  = &outer[0] as auth(Mutate, Remove) &[Vault]
                let ref2 = &outer[0] as auth(Mutate, Remove) &[Vault]

                // Pre-grow ref1's array to trigger split, demoting ref2's view.
                var i: Int = 0
                while i < 200 {
                    ref1.append(<-create Vault(balance: UFix64(i) + 10.0))
                    i = i + 1
                }

                assert(
                    liveValueIDOf("ref1") != liveValueIDOf("ref2"),
                    message: "after split: refs should observe diverged live atree value IDs"
                )

                // ref1 removes and destroys the original first vault.
                let v <- ref1.removeFirst()
                assert(v.balance == 1.0, message: "ref1 removed canonical first vault")
                destroy v

                // Sibling's attempt to remove must be rejected. Without the
                // staleness check, it could read the demoted slab and yield
                // a phantom copy of the already-destroyed resource.
                let phantom <- ref2.removeFirst()
                destroy phantom

                destroy outer
            }
        `)

		var staleViewErr *interpreter.InvalidatedContainerViewError
		assert.ErrorAs(t, err, &staleViewErr)
	})

	t.Run("ArrayValue: inner-growth uninlining surfaces via liveInlinedOf; sibling rejected", func(t *testing.T) {
		t.Parallel()

		// `outer: @[[Vault]]` begins with its single inner array stored
		// inlined inside outer's slab. Atree inlines a child container
		// whenever the child's full content fits within its parent's
		// inline-element budget. When the child grows past that budget,
		// atree uninlines it — physically moving its data to a standalone
		// slab.
		//
		// Important observation: outer-side growth alone (i.e., calling
		// `outer.append(<-[<-Vault])` many times) does NOT uninline a
		// specific child; atree just splits outer into more leaf slabs,
		// each of which can independently hold inlined children. The way
		// to actually trigger uninlining of `outer[0]` is to grow `outer[0]`
		// itself past the inline-element budget.
		//
		// Atree's ValueID is stable across the inline ↔ standalone-slab
		// transition, so the cached-vs-live ValueID comparison used by the
		// staleness check elsewhere in this file cannot detect uninlining.
		// To make the transition observable at the Cadence level, this test
		// uses `liveInlinedOf`, which taps `*atree.Array.Inlined()` (and
		// the analogous method on `*atree.OrderedMap`).
		//
		// Both sibling refs are stale post-uninlining/split, because growth
		// triggered through `ref1` necessarily restructures the slab tree
		// they shared at construction. The centralized check is expected
		// to reject the sibling's subsequent mutation with
		// `InvalidatedContainerViewError`. The Cadence assertions below
		// pin down each observable transition.
		err := runInvoke(t, `
            access(all) resource Vault {
                access(all) var balance: UFix64
                init(balance: UFix64) { self.balance = balance }
            }

            access(all) fun main() {
                let outer: @[[Vault]] <- [<-[<-create Vault(balance: 1.0)]]

                let ref1  = &outer[0] as auth(Mutate) &[Vault]
                let ref2 = &outer[0] as auth(Mutate) &[Vault]

                assert(
                    liveInlinedOf("ref1"),
                    message: "precondition: inner[0] should start inlined inside outer's slab"
                )
                assert(
                    liveValueIDOf("ref1") == liveValueIDOf("ref2"),
                    message: "precondition: siblings should observe the same live atree value ID"
                )

                // Grow the inner array via ref1. As inner[0]'s content
                // exceeds atree's inline-element budget, atree uninlines it
                // into its own standalone slab.
                var i: Int = 0
                while i < 200 {
                    ref1.append(<-create Vault(balance: UFix64(i) + 10.0))
                    i = i + 1
                }

                // Concretely confirm uninlining happened. Without this
                // assertion the test would be vacuously true if atree
                // happened to keep inner[0] inlined.
                assert(
                    !liveInlinedOf("ref1"),
                    message: "expected inner[0] to be uninlined after growth via ref1; if this fires, "
                        .concat("the iteration count is too small for atree's current inline budget")
                )

                // Append through the sibling. The slab tree restructuring
                // (which includes the uninline transition) is also a split
                // demotion at the inner array's tree level, so the cached-vs-
                // live ValueID comparison fires here and the centralized
                // check rejects the mutation.
                ref2.append(<-create Vault(balance: 999.0))

                destroy outer
            }
        `)

		var staleViewErr *interpreter.InvalidatedContainerViewError
		assert.ErrorAs(t, err, &staleViewErr)
	})

	t.Run("forEachAttachment + sibling parent mutation in callback", func(t *testing.T) {
		t.Parallel()

		// `ref1.forEachAttachment(...)` opens an atree iterator on the parent
		// composite's attachment dictionary. The user callback then mutates
		// the parent via a sibling reference `ref2`, growing the dictionary
		// enough to split it. The safe contract is that *some* defense
		// (per-iteration `CheckInvalidatedValueOrValueReference` at the
		// loop head, or `WithContainerMutationPrevention(v.ValueID())`
		// covering the iterator) must fire before any subsequent
		// `iterator.Next()` reads from a demoted slab.
		err := runInvoke(t, typeDeclarations+`
            access(all) fun main() {
                let r0 <- create R(balance: 42.0)
                let r1 <- attach A1() to <-r0
                let r2 <- attach A2() to <-r1
                let r3 <- attach A3() to <-r2
                let r4 <- attach A4() to <-r3
                let r5 <- attach A5() to <-r4
                let r6 <- attach A6() to <-r5
                let r  <- attach B()  to <-r6

                let arr: @[R] <- [<-r]
                let ref1  = &arr[0] as &R
                let ref2 = &arr[0] as &R

                assert(
                    liveValueIDOf("ref1") == liveValueIDOf("ref2"),
                    message: "before split: both refs should observe the same live atree value ID"
                )

                // Inside the forEachAttachment callback, mutate every
                // attachment via the sibling. Each inflate() writes long
                // strings into an attachment's atree map; the cumulative
                // effect restructures the parent's atree dictionary.
                //
                // The expected safe outcome is that either:
                //   (a) the next-iteration head re-check on compositeReference
                //       fires InvalidatedContainerViewError, or
                //   (b) atree's mutation-prevention machinery fires
                //       ContainerMutatedDuringIterationError, because ref1
                //       and ref2 share the same parent ValueID.
                // Either outcome preserves invariants.
                ref1.forEachAttachment(fun (a: &AnyResourceAttachment): Void {
                    ref2[A1]!.inflate()
                    ref2[A2]!.inflate()
                    ref2[A3]!.inflate()
                    ref2[A4]!.inflate()
                    ref2[A5]!.inflate()
                    ref2[A6]!.inflate()
                })

                destroy arr
            }
        `)

		var staleViewErr *interpreter.InvalidatedContainerViewError
		assert.ErrorAs(t, err, &staleViewErr)
	})
}
