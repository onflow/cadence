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

		// liveValueID exposes the underlying atree container's current value ID
		// so the Cadence code can confirm the slab split actually occurred
		// before attempting the stale-wrapper mutation.
		liveValueIDFunction := stdlib.NewInterpreterStandardLibraryStaticFunction(
			"liveValueID",
			sema.NewSimpleFunctionType(
				sema.FunctionPurityImpure,
				[]sema.Parameter{
					{
						Label:      sema.ArgumentLabelNotRequired,
						Identifier: "ref",
						TypeAnnotation: sema.NewTypeAnnotation(
							&sema.ReferenceType{
								Type:          sema.AnyResourceType,
								Authorization: sema.UnauthorizedAccess,
							},
						),
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

		baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
		baseValueActivation.DeclareValue(liveValueIDFunction)
		baseValueActivation.DeclareValue(stdlib.InterpreterAssertFunction)

		baseActivation := activations.NewActivation(nil, interpreter.BaseActivation)
		interpreter.Declare(baseActivation, liveValueIDFunction)
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
                    liveValueID(ref) == liveValueID(ref2),
                    message: "before split: both refs should observe the same live atree value ID"
                )

                var i: Int = 0
                while i < 200 {
                    ref.append(<-create Vault(balance: UFix64(i)))
                    i = i + 1
                }

                assert(
                    liveValueID(ref) != liveValueID(ref2),
                    message: "after split: refs should observe diverged live atree value IDs"
                )

                // This mutation goes through the stale wrapper and must be rejected.
                ref2.append(<- create Vault(balance: 123.456))

                destroy outer
            }
        `)
		var staleViewErr *interpreter.InvalidatedContainerViewError
		assert.ErrorAs(t, err, &staleViewErr)
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
                    liveValueID(ref) == liveValueID(ref2),
                    message: "before split: both refs should observe the same live atree value ID"
                )

                var i: Int = 0
                while i < 200 {
                    ref.append(<-create Vault(balance: UFix64(i)))
                    i = i + 1
                }

                assert(
                    liveValueID(ref) != liveValueID(ref2),
                    message: "after split: refs should observe diverged live atree value IDs"
                )

                ref2.insert(at: 0, <- create Vault(balance: 123.456))

                destroy outer
            }
        `)
		var staleViewErr *interpreter.InvalidatedContainerViewError
		assert.ErrorAs(t, err, &staleViewErr)
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
                    liveValueID(ref) == liveValueID(ref2),
                    message: "before split: both refs should observe the same live atree value ID"
                )

                var i: Int = 0
                while i < 200 {
                    ref.append(<-create Vault(balance: UFix64(i)))
                    i = i + 1
                }

                assert(
                    liveValueID(ref) != liveValueID(ref2),
                    message: "after split: refs should observe diverged live atree value IDs"
                )

                let extra <- ref2.remove(at: 0)
                destroy extra

                destroy outer
            }
        `)
		var staleViewErr *interpreter.InvalidatedContainerViewError
		assert.ErrorAs(t, err, &staleViewErr)
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
                    liveValueID(ref) == liveValueID(ref2),
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
                    liveValueID(ref) != liveValueID(ref2),
                    message: "after split: refs should observe diverged live atree value IDs"
                )

                let old2 <- ref2.insert(key: 9999, <- create Vault(balance: 123.456))
                destroy old2

                destroy outer
            }
        `)
		var staleViewErr *interpreter.InvalidatedContainerViewError
		assert.ErrorAs(t, err, &staleViewErr)
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
                    liveValueID(ref) == liveValueID(ref2),
                    message: "before split: both refs should observe the same live atree value ID"
                )

                ref[A1]!.inflate()
                ref[A2]!.inflate()
                ref[A3]!.inflate()
                ref[A4]!.inflate()
                ref[A5]!.inflate()
                ref[A6]!.inflate()

                assert(
                    liveValueID(ref) != liveValueID(ref2),
                    message: "after split: refs should observe diverged live atree value IDs"
                )

                // Field assignment through stale wrapper must be rejected.
                ref2.setBalance(123.456)

                destroy arr
            }
        `)
		var staleViewErr *interpreter.InvalidatedContainerViewError
		assert.ErrorAs(t, err, &staleViewErr)
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
                    liveValueID(ref) == liveValueID(ref2),
                    message: "before split: both refs should observe the same live atree value ID"
                )

                var i: Int = 1
                while i < 300 {
                    let old <- ref.insert(key: i, <-create Vault(balance: UFix64(i)))
                    destroy old
                    i = i + 1
                }

                assert(
                    liveValueID(ref) != liveValueID(ref2),
                    message: "after split: refs should observe diverged live atree value IDs"
                )

                let removed <- ref2.remove(key: 0)
                destroy removed

                destroy outer
            }
        `)
		var staleViewErr *interpreter.InvalidatedContainerViewError
		assert.ErrorAs(t, err, &staleViewErr)
	})
}
