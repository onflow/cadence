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
					id = v.ValueID().String()
				case *interpreter.DictionaryValue:
					id = v.ValueID().String()
				case *interpreter.CompositeValue:
					id = v.ValueID().String()
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

		baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
		baseValueActivation.DeclareValue(liveValueIDOfResourceFunction)
		baseValueActivation.DeclareValue(liveValueIDOfStructFunction)
		baseValueActivation.DeclareValue(stdlib.InterpreterAssertFunction)

		baseActivation := activations.NewActivation(nil, interpreter.BaseActivation)
		interpreter.Declare(baseActivation, liveValueIDOfResourceFunction)
		interpreter.Declare(baseActivation, liveValueIDOfStructFunction)
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

}
