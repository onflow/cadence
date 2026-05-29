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

		// liveMutationCountOf exposes the underlying atree container's current
		// root-slab mutation counter — the second signal isStaleAtreeView
		// compares against the cached snapshot. Returning UInt64 lets Cadence
		// test code compare counts directly with == / >.
		liveMutationCountOfFunction := stdlib.NewInterpreterStandardLibraryStaticFunction(
			"liveMutationCountOf",
			sema.NewSimpleFunctionType(
				sema.FunctionPurityImpure,
				[]sema.Parameter{
					{
						Label:          sema.ArgumentLabelNotRequired,
						Identifier:     "name",
						TypeAnnotation: sema.StringTypeAnnotation,
					},
				},
				sema.UInt64TypeAnnotation,
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
				var count uint64
				switch v := ref.Value.(type) {
				case *interpreter.ArrayValue:
					count = v.LiveMutationCount()
				case *interpreter.DictionaryValue:
					count = v.LiveMutationCount()
				case *interpreter.CompositeValue:
					count = v.LiveMutationCount()
				default:
					t.Fatalf("unexpected value type %T", ref.Value)
				}
				return interpreter.NewUnmeteredUInt64Value(count)
			},
		)

		baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
		baseValueActivation.DeclareValue(liveValueIDOfFunction)
		baseValueActivation.DeclareValue(liveMutationCountOfFunction)
		baseValueActivation.DeclareValue(stdlib.InterpreterAssertFunction)

		baseActivation := activations.NewActivation(nil, interpreter.BaseActivation)
		interpreter.Declare(baseActivation, liveValueIDOfFunction)
		interpreter.Declare(baseActivation, liveMutationCountOfFunction)
		interpreter.Declare(baseActivation, stdlib.InterpreterAssertFunction)

		return baseValueActivation, baseActivation
	}

	// Some tests need to bypass a specific checker diagnostic (e.g.
	// Filter's procedure must be `view` per sema, but to exercise the
	// runtime mutation-prevention barrier we need an impure procedure).
	runInvokeWithHandleCheckerError := func(
		t *testing.T,
		code string,
		handleCheckerError func(error),
	) error {
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
				HandleCheckerError: handleCheckerError,
			},
		)
		require.NoError(t, err)
		_, err = inter.Invoke("main")
		return err
	}
	runInvoke := func(t *testing.T, code string) error {
		return runInvokeWithHandleCheckerError(t, code, nil)
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

	t.Run("ArrayValue: silent-corruption reproducer is now blocked", func(t *testing.T) {
		t.Parallel()

		// Pre-patch silent-corruption demonstration: the stale-wrapper
		// append parks the new element on the demoted leaf slab. The
		// canonical view's iteration would not see the parked element,
		// but removeLast in reverse order would surface it — meaning the
		// inner array has divergent length depending on which path you
		// read it through.
		//
		// With the staleness check in place, ref2.append now panics with
		// InvalidatedContainerViewError; execution never reaches the
		// post-mutation forensics. The forensics are retained here as
		// documentation of what the attack looked like — they would
		// surface iteratedCount != removalCount or
		// elementFoundIterating != elementFoundRemoving on a vulnerable
		// build.
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

                // Mutation via the now-stale ref2 must be rejected here.
                ref2.append(<- create Vault(balance: 123.456))

                // Unreachable in a fixed build. On a vulnerable build the
                // following would observe the divergence between the
                // canonical view's iterator and a reverse-removeLast walk.
                var empty: @[Vault] <- []
                var extracted <- outer[0] <- empty
                var iteratedCount: Int = 0
                var removalCount: Int = 0
                var elementFoundIterating = false
                var elementFoundRemoving = false

                for element in &extracted as &[Vault] {
                    iteratedCount = iteratedCount + 1
                    if element.balance == 123.456 {
                        elementFoundIterating = true
                    }
                }
                while extracted.length > 0 {
                    let element <- extracted.removeLast()
                    if element.balance == 123.456 {
                        elementFoundRemoving = true
                    }
                    destroy element
                    removalCount = removalCount + 1
                }

                assert(iteratedCount == removalCount)
                assert(elementFoundIterating == elementFoundRemoving)
                destroy extracted
                destroy outer
            }
        `)
		var staleViewErr *interpreter.InvalidatedContainerViewError
		assert.ErrorAs(t, err, &staleViewErr)
		// The ref2.append line — the staleness check must fire here, not
		// at any later point.
		assert.Equal(t, 25, staleViewErr.StartPosition().Line)
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

	// The remaining subtests cover the gap that motivated the
	// MutationCount mechanism: atree's promoteChildAsNewRoot leaves the
	// orphaned old root's SlabID unchanged, so the ValueID-only check
	// (which works for splitRoot via SetSlabID side effects) cannot see it.
	//
	// Setup pattern:
	//   1. Grow the inner container past one split via `ref` alone — the
	//      root is now a metaslab.
	//   2. Create `ref2` AFTER the split — ref2 caches the post-split
	//      ValueID AND mutationCount, both reflecting the metaslab root.
	//   3. Through `ref`, remove enough elements to collapse the tree back
	//      to a single data slab. This triggers promoteChildAsNewRoot,
	//      which preserves ValueID but bumps the orphaned metaslab's
	//      mutation counter.
	//   4. Through `ref2`, attempt a mutation. The valueID check passes,
	//      but the mutationCount check trips — InvalidatedContainerViewError.
	//
	// Note on the intermediate assertions: liveMutationCountOf reads the
	// live root's counter for the wrapper passed in. After promote, ref's
	// live root is the freshly-promoted child (counter 0), while ref2's
	// .root pointer still references the orphaned metaslab (counter > 0).
	// So the divergence check is between the two wrappers, not pre/post on
	// a single wrapper. This guards against the false-pass mode
	// "promote did not fire" cleanly rather than producing a confusing
	// "did not expect a stale-view error".

	t.Run("ArrayValue: append via stale wrapper after promote", func(t *testing.T) {
		t.Parallel()

		err := runInvoke(t, `
            access(all) resource Vault {
                access(all) var balance: UFix64
                init(balance: UFix64) { self.balance = balance }
            }

            access(all) fun main() {
                let outer: @[[Vault]] <- [<-[<-create Vault(balance: 0.0)]]

                let ref = &outer[0] as auth(Mutate) &[Vault]
                var i: Int = 0
                while i < 200 {
                    ref.append(<-create Vault(balance: UFix64(i)))
                    i = i + 1
                }

                // Construct ref2 AFTER the split — it caches the
                // multi-level state's valueID and mutationCount.
                let ref2 = &outer[0] as auth(Mutate) &[Vault]

                assert(
                    liveValueIDOf("ref") == liveValueIDOf("ref2"),
                    message: "after split, before promote: refs share the same live atree value ID"
                )
                assert(
                    liveMutationCountOf("ref") == liveMutationCountOf("ref2"),
                    message: "after split, before promote: refs share the same live mutation counter"
                )

                // Collapse: must shrink all the way to one element so the
                // tree fully collapses and promoteChildAsNewRoot fires.
                while i > 0 {
                    let v <- ref.removeLast()
                    destroy v
                    i = i - 1
                }

                // Guard against "promote did not fire". After promote, ref
                // reads the new (freshly-promoted) root's counter, ref2's
                // .root pointer still references the orphaned metaslab —
                // whose counter was bumped pre-swap. So the two diverge.
                assert(
                    liveMutationCountOf("ref") != liveMutationCountOf("ref2"),
                    message: "after promote: ref's live root (new, fresh counter) must diverge from ref2's live root (orphaned, bumped)"
                )
                // ValueID alone would NOT diverge (the gap).
                assert(
                    liveValueIDOf("ref") == liveValueIDOf("ref2"),
                    message: "ValueID is preserved across promote — this is the gap the counter closes"
                )

                // Mutation through the stale ref2 must be rejected.
                ref2.append(<- create Vault(balance: 123.456))

                destroy outer
            }
        `)
		var staleViewErr *interpreter.InvalidatedContainerViewError
		assert.ErrorAs(t, err, &staleViewErr)
		assert.Equal(t, 53, staleViewErr.StartPosition().Line)
	})

	t.Run("DictionaryValue: insert via stale wrapper after promote", func(t *testing.T) {
		t.Parallel()

		err := runInvoke(t, `
            access(all) resource Vault {
                access(all) var balance: UFix64
                init(balance: UFix64) { self.balance = balance }
            }

            access(all) fun main() {
                let outer: @[{Int: Vault}] <- [<-{0: <-create Vault(balance: 0.0)}]

                let ref = &outer[0] as auth(Mutate) &{Int: Vault}
                var i: Int = 1
                while i < 300 {
                    let old <- ref.insert(key: i, <-create Vault(balance: UFix64(i)))
                    assert(old == nil, message: "dict insert should not collide")
                    destroy old
                    i = i + 1
                }

                let ref2 = &outer[0] as auth(Mutate) &{Int: Vault}

                assert(
                    liveValueIDOf("ref") == liveValueIDOf("ref2"),
                    message: "after split, before promote: refs share the same live atree value ID"
                )
                assert(
                    liveMutationCountOf("ref") == liveMutationCountOf("ref2"),
                    message: "after split, before promote: refs share the same live mutation counter"
                )

                // Collapse via removes (remove ALL keys including the
                // original 0-key) so promote fires.
                let original <- ref.remove(key: 0)
                destroy original
                while i > 1 {
                    i = i - 1
                    let removed <- ref.remove(key: i)
                    destroy removed
                }

                assert(
                    liveMutationCountOf("ref") != liveMutationCountOf("ref2"),
                    message: "after promote: ref's live root (new, fresh counter) must diverge from ref2's live root (orphaned, bumped)"
                )
                assert(
                    liveValueIDOf("ref") == liveValueIDOf("ref2"),
                    message: "ValueID is preserved across promote — this is the gap the counter closes"
                )

                let old <- ref2.insert(key: 9999, <- create Vault(balance: 123.456))
                destroy old

                destroy outer
            }
        `)
		var staleViewErr *interpreter.InvalidatedContainerViewError
		assert.ErrorAs(t, err, &staleViewErr)
		assert.Equal(t, 49, staleViewErr.StartPosition().Line)
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
		var containerMutationErr *interpreter.ContainerMutatedDuringIterationError
		assert.ErrorAs(t, err, &containerMutationErr)
	})

	t.Run("ArrayValue: filter procedure mutates sibling wrapper into split mid-iteration", func(t *testing.T) {
		t.Parallel()

		// Parallel to the Map test, with one wrinkle: sema requires the
		// Filter procedure to be `view` (pure), so a Cadence program that
		// mutates a sibling wrapper inside the procedure fails type-checking.
		// To exercise the runtime mutation-prevention barrier on Filter, we
		// pass a HandleCheckerError that swallows the impurity diagnostic.
		//
		// This test guards against the "view enforcement has a hole" scenario:
		// if any future checker change accidentally lets an impure call slip
		// into a Filter procedure, the runtime barrier must still reject the
		// resulting sibling mutation.
		err := runInvokeWithHandleCheckerError(
			t,
			`
                access(all) fun main() {
                    let outer: [[Int]] = [[0, 1]]

                    let ref1 = &outer[0] as auth(Mutate) &[Int]
                    let ref2 = &outer[0] as auth(Mutate) &[Int]

                    var calls: Int = 0
                    let filtered = ref1.filter(view fun (v: Int): Bool {
                        if calls == 0 {
                            var j = 0
                            while j < 300 {
                                ref2.append(j + 100)
                                j = j + 1
                            }
                        }
                        calls = calls + 1
                        return true
                    })

                    // Unreachable: the ref2.append inside the procedure must
                    // raise ContainerMutatedDuringIterationError on the very
                    // first callback invocation.
                    assert(false, message: "unreachable")
                }
            `,
			func(checkerErr error) {
				// Swallow the impurity diagnostics that come from running
				// state-mutating code inside the view-typed filter procedure.
				// We deliberately exercise an impure procedure to verify the
				// runtime barrier; the impurity errors are not what this
				// test is about.
				errs := RequireCheckerErrors(t, checkerErr, 2)
				for _, e := range errs {
					assert.IsType(t, &sema.PurityError{}, e)
				}
			},
		)
		var containerMutationErr *interpreter.ContainerMutatedDuringIterationError
		assert.ErrorAs(t, err, &containerMutationErr)
	})
}
