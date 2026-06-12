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
	. "github.com/onflow/cadence/test_utils/sema_utils"
)

// TestInterpretAliasedWrapperMutationPropagation covers the case where two
// references (`&outer[i]` taken twice) project two `auth(Mutate) &T` handles
// onto the same atree container.
// Because of the canonical-wrapper cache (`SharedState.canonicalAtreeContainers`),
// both handles resolve to the same Go-level wrapper —
// even after a structural change like a slab split triggered through one of them.
// A subsequent mutation through the second handle must therefore land on the
// live tree and be observable through the first.
//
// The mid-iteration subtests (Map/Filter) cover a separate concern:
// the canonical wrapper is shared, but mutating it while another method has
// an active atree iterator must still raise `ContainerMutatedDuringIterationError`.
func TestInterpretAliasedWrapperMutationPropagation(t *testing.T) {
	t.Parallel()

	// liveValueIDOf{Resource,Struct} expose the underlying atree container's
	// value ID for any wrapped ArrayValue / DictionaryValue / CompositeValue.
	// Two functions are needed because Cadence reference subtyping puts resource
	// and struct references in disjoint hierarchies:
	// `&AnyResource` won't accept struct refs and vice versa.
	makeLiveValueIDOfFunction := func(
		t *testing.T,
		name string,
		refType sema.Type,
	) stdlib.StandardLibraryValue {
		return stdlib.NewInterpreterStandardLibraryStaticFunction(
			name,
			sema.NewSimpleFunctionType(
				sema.FunctionPurityImpure,
				[]sema.Parameter{
					{
						Label:          sema.ArgumentLabelNotRequired,
						Identifier:     "ref",
						TypeAnnotation: sema.NewTypeAnnotation(refType),
					},
				},
				sema.StringTypeAnnotation,
			),
			"",
			func(
				_ interpreter.NativeFunctionContext,
				_ interpreter.TypeArgumentsIterator,
				_ interpreter.ArgumentTypesIterator,
				_ interpreter.Value,
				args []interpreter.Value,
			) interpreter.Value {
				ref := args[0].(*interpreter.EphemeralReferenceValue)
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
	}

	makeEnv := func(t *testing.T) (
		*sema.VariableActivation,
		*activations.Activation[interpreter.Variable],
	) {

		liveValueIDOfResourceFunction := makeLiveValueIDOfFunction(
			t,
			"liveValueIDOfResource",
			&sema.ReferenceType{
				Type:          sema.AnyResourceType,
				Authorization: sema.UnauthorizedAccess,
			},
		)
		liveValueIDOfStructFunction := makeLiveValueIDOfFunction(
			t,
			"liveValueIDOfStruct",
			&sema.ReferenceType{
				Type:          sema.AnyStructType,
				Authorization: sema.UnauthorizedAccess,
			},
		)

		// liveInlinedOf{Resource,Struct} expose whether the underlying atree
		// container of a reference is currently stored inlined inside its
		// parent's slab. Atree may transition a container between inlined and
		// standalone-slab storage when its parent grows or shrinks. Such
		// transitions are NOT observable through LiveValueID (atree assigns
		// a stable ValueID across inline/uninline transitions), so this
		// helper is needed to assert that uninlining actually occurred in
		// tests that probe the safety of the post-uninlining state.
		// Two functions are needed for the same reason as the value-ID helpers:
		// resource and struct references live in disjoint type hierarchies.
		makeLiveInlinedOfFunction := func(
			name string,
			refType sema.Type,
		) stdlib.StandardLibraryValue {
			return stdlib.NewInterpreterStandardLibraryStaticFunction(
				name,
				sema.NewSimpleFunctionType(
					sema.FunctionPurityImpure,
					[]sema.Parameter{
						{
							Label:          sema.ArgumentLabelNotRequired,
							Identifier:     "ref",
							TypeAnnotation: sema.NewTypeAnnotation(refType),
						},
					},
					sema.BoolTypeAnnotation,
				),
				"",
				func(
					_ interpreter.NativeFunctionContext,
					_ interpreter.TypeArgumentsIterator,
					_ interpreter.ArgumentTypesIterator,
					_ interpreter.Value,
					args []interpreter.Value,
				) interpreter.Value {
					ref := args[0].(*interpreter.EphemeralReferenceValue)
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
		}

		liveInlinedOfResourceFunction := makeLiveInlinedOfFunction(
			"liveInlinedOfResource",
			&sema.ReferenceType{
				Type:          sema.AnyResourceType,
				Authorization: sema.UnauthorizedAccess,
			},
		)
		liveInlinedOfStructFunction := makeLiveInlinedOfFunction(
			"liveInlinedOfStruct",
			&sema.ReferenceType{
				Type:          sema.AnyStructType,
				Authorization: sema.UnauthorizedAccess,
			},
		)

		baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
		baseValueActivation.DeclareValue(liveValueIDOfResourceFunction)
		baseValueActivation.DeclareValue(liveValueIDOfStructFunction)
		baseValueActivation.DeclareValue(liveInlinedOfResourceFunction)
		baseValueActivation.DeclareValue(liveInlinedOfStructFunction)
		baseValueActivation.DeclareValue(stdlib.InterpreterAssertFunction)

		baseActivation := activations.NewActivation(nil, interpreter.BaseActivation)
		interpreter.Declare(baseActivation, liveValueIDOfResourceFunction)
		interpreter.Declare(baseActivation, liveValueIDOfStructFunction)
		interpreter.Declare(baseActivation, liveInlinedOfResourceFunction)
		interpreter.Declare(baseActivation, liveInlinedOfStructFunction)
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
		invokable, err := parseCheckAndPrepareWithAtreeValidationsDisabled(
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
		_, err = invokable.Invoke("main")
		return err
	}
	runInvoke := func(t *testing.T, code string) error {
		return runInvokeWithHandleCheckerError(t, code, nil)
	}

	t.Run("ArrayValue: append via sibling wrapper after split propagates", func(t *testing.T) {
		t.Parallel()

		// Two `auth(Mutate) &[Vault]` references project onto the same inner array.
		// `ref` appends enough elements to trigger an atree slab split.
		// Without the canonical wrapper cache, `ref2`'s wrapper would point at the
		// now-demoted leaf slab and mutate the wrong slab.
		// With the cache, both refs resolve to the same wrapper,
		// so `ref2.append(...)` lands on the live tree and is observable through `ref`.
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
                    liveValueIDOfResource(ref) == liveValueIDOfResource(ref2),
                    message: "before split: both refs should observe the same live atree value ID"
                )

                var i: Int = 0
                while i < 200 {
                    ref.append(<-create Vault(balance: UFix64(i)))
                    i = i + 1
                }

                assert(
                    liveValueIDOfResource(ref) == liveValueIDOfResource(ref2),
                    message: "after split: refs must still observe the same live atree value ID"
                )

                // Append through ref2: both refs share the canonical wrapper, so
                // the append lands on the live tree.
                ref2.append(<- create Vault(balance: 123.456))

                // Both refs observe the same length (1 initial + 200 + 1 = 202).
                assert(ref.length == 202, message: "ref must observe ref2's append")
                assert(ref2.length == 202, message: "ref2 must observe its own append")

                // The last element appended via ref2 is visible through ref.
                assert(
                    ref[201].balance == 123.456,
                    message: "ref must read back the value appended via ref2"
                )

                destroy outer
            }
        `)
		require.NoError(t, err)
	})

	t.Run("ArrayValue: silent-corruption reproducer is now blocked", func(t *testing.T) {
		t.Parallel()

		// Pre-canonicalization silent-corruption reproducer:
		// the sibling `ref2.append` would park the new element on the
		// demoted leaf slab of its stale Go-level *atree.Array.
		// The canonical view's iteration would not see the parked element,
		// but `removeLast` in reverse order would surface it — meaning
		// the inner array had divergent length depending on which path
		// you read it through.
		// With canonicalization, `ref` and `ref2` resolve to the same Go
		// wrapper, so `ref2.append` lands on the live tree and both
		// traversals see the appended element. The post-append forensics
		// (iteratedCount/removalCount agreement, element visibility
		// agreement) now pass.
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
                    liveValueIDOfResource(ref) == liveValueIDOfResource(ref2),
                    message: "before split: both refs should observe the same live atree value ID"
                )

                var i: Int = 0
                while i < 200 {
                    ref.append(<-create Vault(balance: UFix64(i)))
                    i = i + 1
                }

                // Mutation via the sibling ref2 lands on the canonical wrapper.
                ref2.append(<- create Vault(balance: 123.456))

                // Forensics: pre-canonicalization, a stale ref2.append would
                // park the element on a demoted leaf, causing the iterator's
                // count to diverge from the reverse-removeLast count, and the
                // appended element to be visible to only one of the two walks.
                // With canonicalization both walks agree.
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

                assert(iteratedCount == removalCount,
                       message: "iteration must agree with reverse-removeLast count")
                assert(elementFoundIterating == elementFoundRemoving,
                       message: "the appended element must be visible to both walks")
                assert(iteratedCount == 202,
                       message: "1 initial + 200 + 1 appended via ref2")
                assert(elementFoundIterating,
                       message: "ref2's append must be observable through the canonical wrapper")
                destroy extracted
                destroy outer
            }
        `)
		require.NoError(t, err)
	})

	t.Run("ArrayValue: insert via sibling wrapper after split propagates", func(t *testing.T) {
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
                    liveValueIDOfResource(ref) == liveValueIDOfResource(ref2),
                    message: "before split: both refs should observe the same live atree value ID"
                )

                var i: Int = 0
                while i < 200 {
                    ref.append(<-create Vault(balance: UFix64(i)))
                    i = i + 1
                }

                assert(
                    liveValueIDOfResource(ref) == liveValueIDOfResource(ref2),
                    message: "after split: refs must still observe the same live atree value ID"
                )

                ref2.insert(at: 0, <- create Vault(balance: 123.456))

                assert(ref.length == 202, message: "ref must observe ref2's insert")
                assert(
                    ref[0].balance == 123.456,
                    message: "ref must read back the value inserted via ref2"
                )

                destroy outer
            }
        `)
		require.NoError(t, err)
	})

	t.Run("ArrayValue: remove via sibling wrapper after split propagates", func(t *testing.T) {
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
                    liveValueIDOfResource(ref) == liveValueIDOfResource(ref2),
                    message: "before split: both refs should observe the same live atree value ID"
                )

                var i: Int = 0
                while i < 200 {
                    ref.append(<-create Vault(balance: UFix64(i)))
                    i = i + 1
                }

                assert(
                    liveValueIDOfResource(ref) == liveValueIDOfResource(ref2),
                    message: "after split: refs must still observe the same live atree value ID"
                )

                // The element at index 0 has balance 0.0 (the original initial element).
                // After removal, ref must observe length 200 and the new index 0 must
                // be the first appended element (balance 0.0 from UFix64(0)).
                let extra <- ref2.remove(at: 0)
                assert(extra.balance == 0.0, message: "expected to remove the original initial element")
                destroy extra

                assert(ref.length == 200, message: "ref must observe ref2's remove")

                destroy outer
            }
        `)
		require.NoError(t, err)
	})

	t.Run("DictionaryValue: insert via sibling wrapper after split propagates", func(t *testing.T) {
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
                    liveValueIDOfResource(ref) == liveValueIDOfResource(ref2),
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
                    liveValueIDOfResource(ref) == liveValueIDOfResource(ref2),
                    message: "after split: refs must still observe the same live atree value ID"
                )

                let old2 <- ref2.insert(key: 9999, <- create Vault(balance: 123.456))
                assert(old2 == nil, message: "key 9999 should not have collided")
                destroy old2

                // ref observes ref2's insert: 1 (key 0) + 299 (keys 1..299) + 1 (key 9999) = 301.
                assert(ref.length == 301, message: "ref must observe ref2's insert")
                assert(
                    ref.containsKey(9999),
                    message: "ref must see the new key inserted via ref2"
                )

                destroy outer
            }
        `)
		require.NoError(t, err)
	})

	t.Run("CompositeValue: field assignment via sibling wrapper after split propagates", func(t *testing.T) {
		t.Parallel()

		// Two CompositeValue wrappers project onto the same R resource (via two
		// `&arr[0]` references). `ref` inflates attachments enough to split the
		// resource's underlying atree map. With the canonical wrapper cache,
		// `ref2` resolves to the same wrapper as `ref`, so a field assignment
		// through `ref2` must succeed and be observable through `ref`.
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
                    liveValueIDOfResource(ref) == liveValueIDOfResource(ref2),
                    message: "before split: both refs should observe the same live atree value ID"
                )

                ref[A1]!.inflate()
                ref[A2]!.inflate()
                ref[A3]!.inflate()
                ref[A4]!.inflate()
                ref[A5]!.inflate()
                ref[A6]!.inflate()

                assert(
                    liveValueIDOfResource(ref) == liveValueIDOfResource(ref2),
                    message: "after split: refs must still observe the same live atree value ID"
                )

                // Field assignment through ref2 must land on the live wrapper
                // and be observable through ref.
                ref2.setBalance(123.456)
                assert(
                    ref.balance == 123.456,
                    message: "ref must observe the balance set via ref2"
                )

                destroy arr
            }
        `)
		require.NoError(t, err)
	})

	t.Run("DictionaryValue: remove via sibling wrapper after split propagates", func(t *testing.T) {
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
                    liveValueIDOfResource(ref) == liveValueIDOfResource(ref2),
                    message: "before split: both refs should observe the same live atree value ID"
                )

                var i: Int = 1
                while i < 300 {
                    let old <- ref.insert(key: i, <-create Vault(balance: UFix64(i)))
                    destroy old
                    i = i + 1
                }

                assert(
                    liveValueIDOfResource(ref) == liveValueIDOfResource(ref2),
                    message: "after split: refs must still observe the same live atree value ID"
                )

                let removed <- ref2.remove(key: 0)
                assert(removed != nil, message: "key 0 should have been present")
                destroy removed

                // ref observes ref2's remove: 1 (key 0) + 299 (keys 1..299) - 1 = 299.
                assert(ref.length == 299, message: "ref must observe ref2's remove")
                assert(
                    !ref.containsKey(0),
                    message: "ref must no longer see the removed key"
                )

                destroy outer
            }
        `)
		require.NoError(t, err)
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
                    liveValueIDOfResource(ref) == liveValueIDOfResource(ref2),
                    message: "after split, before promote: refs share the same live atree value ID"
                )

                // Collapse: must shrink all the way to one element so the
                // tree fully collapses and promoteChildAsNewRoot fires.
                while i > 0 {
                    let v <- ref.removeLast()
                    destroy v
                    i = i - 1
                }

                // After promote, ValueID is preserved (atree assigns the new
                // root the prior root's SlabID), so the canonical wrapper
                // returns the same ID for both refs.
                assert(
                    liveValueIDOfResource(ref) == liveValueIDOfResource(ref2),
                    message: "ValueID is preserved across promote"
                )

                // Mutation through ref2 lands on the canonical wrapper.
                ref2.append(<- create Vault(balance: 123.456))

                // After all 200 removeLast + 1 ref2.append: 1 (original) + 1 = 2.
                assert(ref.length == 2,
                       message: "ref must observe ref2's post-promote append")
                assert(ref[1].balance == 123.456,
                       message: "ref must read back the value appended via ref2")

                destroy outer
            }
        `)
		require.NoError(t, err)
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
                    liveValueIDOfResource(ref) == liveValueIDOfResource(ref2),
                    message: "after split, before promote: refs share the same live atree value ID"
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
                    liveValueIDOfResource(ref) == liveValueIDOfResource(ref2),
                    message: "ValueID is preserved across promote"
                )

                let old <- ref2.insert(key: 9999, <- create Vault(balance: 123.456))
                assert(old == nil, message: "key 9999 should not have collided")
                destroy old

                // After 1 (original) + 299 inserts + 300 removals + 1 ref2.insert: 1.
                assert(ref.length == 1,
                       message: "ref must observe ref2's post-promote insert")
                assert(ref.containsKey(9999),
                       message: "ref must see the key inserted via ref2")

                destroy outer
            }
        `)
		require.NoError(t, err)
	})

	t.Run("ArrayValue: map procedure mutates sibling wrapper into split mid-iteration", func(t *testing.T) {
		t.Parallel()

		// ArrayValue.Map creates an atree iterator once via v.array.Iterator()
		// and walks it across many user-callback invocations. If the user
		// procedure mutates the same container through a sibling reference
		// (which, with the canonical wrapper cache, is the same wrapper as v),
		// the iterator may yield duplicate or stale elements.
		//
		// Map wraps the iteration in WithContainerMutationPrevention(v.ValueID()),
		// so the mutation through the sibling reference must raise
		// ContainerMutatedDuringIterationError on the very first callback.
		//
		// Non-resource Int elements are used because Cadence forbids
		// `map` on resource arrays even when accessed via reference.
		err := runInvoke(t, `
            access(all) fun main() {
                // Two ArrayValue wrappers for the same inner Int array.
                let outer: [[Int]] = [[0, 1]]

                let ref1  = &outer[0] as auth(Mutate) &[Int]
                let ref2 = &outer[0] as auth(Mutate) &[Int]

                assert(
                    liveValueIDOfStruct(ref1) == liveValueIDOfStruct(ref2),
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
                    liveValueIDOfStruct(ref1) == liveValueIDOfStruct(ref2),
                    message: "after callback-induced split: refs must still observe the same live atree value ID"
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
	// Both directions are exercised by the two sub-tests below.
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
                    liveValueIDOfResource(ref1) == liveValueIDOfResource(ref2),
                    message: "before split: both refs should observe the same live atree value ID"
                )

                ref1[A1]!.inflate()
                ref1[A2]!.inflate()
                ref1[A3]!.inflate()
                ref1[A4]!.inflate()
                ref1[A5]!.inflate()
                ref1[A6]!.inflate()

                assert(
                    liveValueIDOfResource(ref1) == liveValueIDOfResource(ref2),
                    message: "after split: ValueID is preserved by atree, and both refs share the canonical wrapper"
                )

                // ref2 resolves to the same canonical wrapper as ref1.
                // The attachment lookup and method call succeed.
                let stashed = ref2[B]!.readBaseBalance()
                assert(stashed == 42.0,
                       message: "method call through sibling wrapper observes live state")

                destroy arr
            }
        `)
		require.NoError(t, err)
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
                    liveValueIDOfResource(ref1) == liveValueIDOfResource(ref2),
                    message: "before split: both refs should observe the same live atree value ID"
                )

                ref1[A1]!.inflate()
                ref1[A2]!.inflate()
                ref1[A3]!.inflate()
                ref1[A4]!.inflate()
                ref1[A5]!.inflate()
                ref1[A6]!.inflate()

                assert(
                    liveValueIDOfResource(ref1) == liveValueIDOfResource(ref2),
                    message: "after split: ValueID is preserved by atree, and both refs share the canonical wrapper"
                )

                // bRef's base resolves to the canonical R wrapper; the method
                // body reads its live balance.
                let stashed = bRef.readBaseBalance()
                assert(stashed == 42.0,
                       message: "captured attachment ref observes the canonical parent's live state")

                destroy arr
            }
        `)
		require.NoError(t, err)
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
                    liveValueIDOfResource(ref1) == liveValueIDOfResource(ref2),
                    message: "after split: ValueID is preserved by atree, and both refs share the canonical wrapper"
                )

                // ref1 removes and destroys the original first vault.
                let v <- ref1.removeFirst()
                assert(v.balance == 1.0, message: "ref1 removed canonical first vault")
                destroy v

                // Sibling removeFirst lands on the canonical wrapper, so the
                // next element (the first one ref1 appended, balance 10.0)
                // is removed cleanly — no phantom of the already-destroyed vault.
                let next <- ref2.removeFirst()
                assert(next.balance == 10.0,
                       message: "ref2 must see ref1's removeFirst, no phantom resurrection")
                destroy next

                destroy outer
            }
        `)
		require.NoError(t, err)
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
                    liveInlinedOfResource(ref1),
                    message: "precondition: inner[0] should start inlined inside outer's slab"
                )
                assert(
                    liveValueIDOfResource(ref1) == liveValueIDOfResource(ref2),
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

                // Confirm uninlining happened.
                assert(
                    !liveInlinedOfResource(ref1),
                    message: "expected inner[0] to be uninlined after growth via ref1"
                )

                // Append through the sibling. With canonicalization, ref1
                // and ref2 share the canonical wrapper; the append lands on
                // the live tree post-uninline.
                ref2.append(<-create Vault(balance: 999.0))

                // 1 (original) + 200 + 1 = 202.
                assert(ref1.length == 202,
                       message: "ref1 must observe ref2's append after uninline")
                assert(ref1[201].balance == 999.0,
                       message: "ref1 must read back the value appended via ref2")

                destroy outer
            }
        `)
		require.NoError(t, err)
	})

	t.Run("forEachAttachment + sibling parent mutation in callback", func(t *testing.T) {
		t.Parallel()

		// `ref1.forEachAttachment(...)` opens an atree iterator on the parent
		// composite's attachment dictionary. The callback mutates the parent
		// indirectly through `ref2` by inflating each attachment, which can
		// uninline an attachment slab and so restructure the parent dictionary.
		//
		// With canonicalization, `ref1` and `ref2` resolve to the same
		// canonical wrapper. forEachAttachment re-checks the composite
		// reference at the head of every iteration via
		// CheckInvalidatedValueOrValueReference; with that check intact and
		// both refs sharing one wrapper, the mutations land on the live tree
		// and iteration completes safely. The post-iteration probe then
		// asserts the inflates were observable through the canonical wrapper.
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
                    liveValueIDOfResource(ref1) == liveValueIDOfResource(ref2),
                    message: "before split: both refs should observe the same live atree value ID"
                )

                ref1.forEachAttachment(fun (a: &AnyResourceAttachment): Void {
                    ref2[A1]!.inflate()
                    ref2[A2]!.inflate()
                    ref2[A3]!.inflate()
                    ref2[A4]!.inflate()
                    ref2[A5]!.inflate()
                    ref2[A6]!.inflate()
                })

                // The inflates landed on the canonical wrapper; ref1 reads
                // back the post-inflation state through the same wrapper.
                assert(
                    ref1[A1]!.a1 == "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
                    message: "ref1 must observe ref2's A1 inflate"
                )
                assert(
                    ref1[A6]!.h1 == "hhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhh",
                    message: "ref1 must observe ref2's A6 inflate"
                )

                destroy arr
            }
        `)
		require.NoError(t, err)
	})

	t.Run("ArrayValue: inline to standalone round-trip; sibling wrapper probed", func(t *testing.T) {
		t.Parallel()

		// Probes the scenario flagged in atree-slab-change-security-analysis.md:
		// "a single small standalone slab gets re-inlined when its parent shrinks,
		// or vice-versa — would not be caught by the current check".
		//
		// Setup: two sibling refs to outer[0]. Grow inner via `ref` enough to
		// uninline it. Then shrink via `ref` back down so atree re-inlines.
		// The shape returns toward the original (inner empty/tiny, inlined),
		// but `ref2`'s wrapper has been carried across both transitions and
		// many intermediate slab tree restructurings.
		//
		err := runInvoke(t, `
            access(all) resource Vault {
                access(all) var balance: UFix64
                init(balance: UFix64) { self.balance = balance }
            }

            access(all) fun main() {
                let outer: @[[Vault]] <- [<-[<-create Vault(balance: 1.0)]]

                let ref  = &outer[0] as auth(Mutate, Remove) &[Vault]
                let ref2 = &outer[0] as auth(Mutate, Remove) &[Vault]

                assert(
                    liveInlinedOfResource(ref),
                    message: "precondition: inner[0] should start inlined"
                )
                assert(
                    liveValueIDOfResource(ref) == liveValueIDOfResource(ref2),
                    message: "precondition: siblings should share live ValueID"
                )

                // Phase 1: grow inner via ref until atree uninlines outer[0].
                var i: Int = 0
                while i < 200 {
                    ref.append(<-create Vault(balance: UFix64(i) + 10.0))
                    i = i + 1
                }
                assert(
                    !liveInlinedOfResource(ref),
                    message: "phase 1: expected inner[0] to be uninlined after growth"
                )

                // Phase 2: shrink inner via ref so atree re-inlines outer[0].
                var j: Int = 0
                while j < 200 {
                    let v <- ref.removeLast()
                    destroy v
                    j = j + 1
                }
                assert(
                    liveInlinedOfResource(ref),
                    message: "phase 2: expected inner[0] to be re-inlined after shrink"
                )

                // Probe ref2 after the inline <-> standalone round-trip.
                // The sibling has not participated in either transition, and
                // its cached pointers may reference now-freed standalone slabs
                // or otherwise stale tree state.
                ref2.append(<-create Vault(balance: 999.0))

                // If the previous line succeeded (no panic from the staleness check),
                // verify canonical state is consistent. A length other
                // than 2 or wrong balances would indicate silent corruption.
                let canonical = &outer[0] as &[Vault]
                assert(
                    canonical.length == 2,
                    message: "canonical inner[0] length mismatch after round-trip: "
                        .concat(canonical.length.toString())
                )
                assert(canonical[0].balance == 1.0, message: "canonical inner[0][0] mismatch")
                assert(canonical[1].balance == 999.0, message: "canonical inner[0][1] mismatch")

                destroy outer
            }
        `)

		require.NoError(t, err)
	})

	t.Run("ArrayValue: minimal inline to standalone transition; sibling probed", func(t *testing.T) {
		t.Parallel()

		// Probes the narrowest inline -> standalone transition (no round-trip back).
		// The earlier "inner-growth uninlining" test drives 200
		// successive appends to push inner past atree's inline-element budget,
		// that produces many intermediate slab restructurings beyond the
		// uninlining itself, each of which independently changes the slab
		// tree shape and so guarantees a cached-vs-live ValueID mismatch on
		// the sibling.
		//
		// This test instead appends one element at a time and stops the
		// moment `liveInlinedOf` flips from `true` to `false`. That isolates
		// the uninline transition as cleanly as the language layer allows:
		// no extra post-transition restructuring, no round-trip back, and
		// the loop count `i` records exactly how many small elements were
		// needed.
		//
		// Why this matters: atree's stated contract is that a value's
		// `ValueID` is stable across the inline <-> standalone transition.
		// If the minimal uninlining produces no other observable tree change,
		// the centralized staleness check (which compares cached vs. live
		// `ValueID`) is structurally blind to it.
		// The sibling's subsequent mutation succeeds, and the canonical-state
		// assertions hold — meaning atree's storage indirection transparently
		// rebinds the sibling's `*atree.Array` across the transition;

		err := runInvoke(t, `
            access(all) resource Vault {
                access(all) var balance: UFix64
                init(balance: UFix64) { self.balance = balance }
            }

            access(all) fun main() {
                let outer: @[[Vault]] <- [<-[]]
                let ref1 = &outer[0] as auth(Mutate) &[Vault]
                let ref2 = &outer[0] as auth(Mutate) &[Vault]

                assert(
                    liveInlinedOfResource(ref1),
                    message: "precondition: inner[0] should start inlined"
                )
                assert(
                    liveValueIDOfResource(ref1) == liveValueIDOfResource(ref2),
                    message: "precondition: siblings should share live ValueID"
                )

                // Append one element at a time via ref1, stopping the moment
                // atree uninlines inner[0]. This is the minimal mutation
                // sequence that drives the inline → standalone transition.
                var i: Int = 0
                while liveInlinedOfResource(ref1) && i < 1000 {
                    ref1.append(<-create Vault(balance: UFix64(i)))
                    i = i + 1
                }
                assert(
                    !liveInlinedOfResource(ref1),
                    message: "expected inner[0] to be uninlined within 1000 small appends; "
                        .concat("loop count: ").concat(i.toString())
                )

                // Sibling probe immediately at the boundary crossing.
                ref2.append(<-create Vault(balance: 9999.0))

                // Canonical-state assertions catch silent corruption.
                let canonical = &outer[0] as &[Vault]
                let expectedLen = i + 1
                assert(
                    canonical.length == expectedLen,
                    message: "canonical inner[0] length mismatch after minimal uninlining: got "
                        .concat(canonical.length.toString())
                        .concat(", want ")
                        .concat(expectedLen.toString())
                )
                assert(
                    canonical[0].balance == 0.0,
                    message: "canonical inner[0][0] balance mismatch after minimal uninlining"
                )
                assert(
                    canonical[expectedLen - 1].balance == 9999.0,
                    message: "canonical inner[0][last] balance mismatch after minimal uninlining"
                )

                destroy outer
            }
        `)

		require.NoError(t, err)
	})
}
