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
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/activations"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/stdlib"
	"github.com/onflow/cadence/test_utils"
	. "github.com/onflow/cadence/test_utils/common_utils"
	. "github.com/onflow/cadence/test_utils/interpreter_utils"
	. "github.com/onflow/cadence/test_utils/sema_utils"
)

func TestInterpretCompositeValue(t *testing.T) {

	t.Parallel()

	t.Run("computed fields", func(t *testing.T) {

		t.Parallel()

		inter := testCompositeValue(
			t,
			`
              // Get a static field using member access
              let name: String = fruit.name

              // Get a computed field using member access
              let color: String = fruit.color
            `,
		)

		RequireValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredStringValue("Apple"),
			inter.GetGlobal("name"),
		)

		RequireValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredStringValue("Red"),
			inter.GetGlobal("color"),
		)
	})
}

// Utility methods
func testCompositeValue(t *testing.T, code string) Invokable {

	storage := NewUnmeteredInMemoryStorage()

	// 'fruit' composite type
	fruitType := &sema.CompositeType{
		Location:   TestLocation,
		Identifier: "Fruit",
		Kind:       common.CompositeKindStructure,
	}

	fruitType.Members = &sema.StringMemberOrderedMap{}

	fruitType.Members.Set("name", sema.NewUnmeteredPublicConstantFieldMember(
		fruitType,
		"name",
		sema.StringType,
		"This is the name",
	))

	fruitType.Members.Set("color", sema.NewUnmeteredPublicConstantFieldMember(
		fruitType,
		"color",
		sema.StringType,
		"This is the color",
	))

	fruitStaticType := interpreter.NewCompositeStaticTypeComputeTypeID(
		nil,
		TestLocation,
		fruitType.Identifier,
	)

	fruitValue := interpreter.NewSimpleCompositeValue(
		nil,
		fruitType.ID(),
		fruitStaticType,
		[]string{"name", "color"},
		map[string]interpreter.Value{
			"name": interpreter.NewUnmeteredStringValue("Apple"),
		},
		func(name string, _ interpreter.MemberAccessibleContext) interpreter.Value {
			if name == "color" {
				return interpreter.NewUnmeteredStringValue("Red")
			}

			return nil
		},
		nil,
		nil,
		nil,
	)

	valueDeclaration := stdlib.StandardLibraryValue{
		Name:  "fruit",
		Type:  fruitType,
		Value: fruitValue,
		Kind:  common.DeclarationKindConstant,
	}

	baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
	baseValueActivation.DeclareValue(valueDeclaration)

	baseTypeActivation := sema.NewVariableActivation(sema.BaseTypeActivation)
	baseTypeActivation.DeclareType(stdlib.StandardLibraryType{
		Name: fruitType.Identifier,
		Type: fruitType,
		Kind: common.DeclarationKindStructure,
	})

	baseActivation := activations.NewActivation(nil, interpreter.BaseActivation)
	interpreter.Declare(baseActivation, valueDeclaration)

	inter, err := parseCheckAndPrepareWithOptions(t,
		code,
		ParseCheckAndInterpretOptions{
			ParseAndCheckOptions: &ParseAndCheckOptions{
				CheckerConfig: &sema.Config{
					BaseValueActivationHandler: func(_ common.Location) *sema.VariableActivation {
						return baseValueActivation
					},
					BaseTypeActivationHandler: func(_ common.Location) *sema.VariableActivation {
						return baseTypeActivation
					},
					CheckHandler: func(checker *sema.Checker, check func()) {
						if checker.Location == TestLocation {
							checker.Elaboration.SetCompositeType(
								fruitType.ID(),
								fruitType,
							)
						}
						check()
					},
				},
			},
			InterpreterConfig: &interpreter.Config{
				Storage: storage,
				BaseActivationHandler: func(_ common.Location) *interpreter.VariableActivation {
					return baseActivation
				},
			},
		},
	)
	require.NoError(t, err)

	return inter
}

func TestInterpretContractTransfer(t *testing.T) {

	t.Parallel()

	test := func(t *testing.T, value string) {

		t.Parallel()

		address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

		code := fmt.Sprintf(
			`
              contract C {}

              fun test() {
                  authAccount.storage.save(%s, to: /storage/c)
              }
		    `,
			value,
		)
		inter, _, _ := testAccountWithErrorHandler(
			t,
			address,
			true,
			nil,
			code,
			sema.Config{},
			func(err error) {
				var invalidMoveError *sema.InvalidMoveError
				require.ErrorAs(t, err, &invalidMoveError)
			},
		)

		_, err := inter.Invoke("test")
		RequireError(t, err)

		var nonTransferableValueError *interpreter.NonTransferableValueError
		require.ErrorAs(t, err, &nonTransferableValueError)
	}

	t.Run("simple", func(t *testing.T) {
		test(t, "C as AnyStruct")
	})

	t.Run("nested", func(t *testing.T) {
		test(t, "[C as AnyStruct]")
	})
}

func TestInterpretFunctionTypedField(t *testing.T) {

	t.Parallel()

	t.Run("user function pointer", func(t *testing.T) {
		t.Parallel()

		inter, getLogs, err := parseCheckAndPrepareWithLogs(t, `
            resource R {
                let f: (fun(AnyStruct): Void)

                init() {
                    self.f = print  // User function (non-bound).
                }
            }

            fun print(_ msg: AnyStruct) {
                log(msg)
            }

            fun test() {
                let r <- create R()

                let f = r.f   // This must return a non-bound function.

                destroy r

                f("hello")  // function pointer should be still valid.
            }
        `)

		require.NoError(t, err)

		_, err = inter.Invoke("test")
		require.NoError(t, err)

		logs := getLogs()
		assert.Equal(t, []string{`"hello"`}, logs)
	})

	t.Run("host function pointer", func(t *testing.T) {
		t.Parallel()

		inter, getLogs, err := parseCheckAndPrepareWithLogs(t, `
            resource R {
                let f: (fun(AnyStruct): Void)

                init() {
                    self.f = log   // host function (non-bound).
                }
            }

            fun test() {
                let r <- create R()

                let f = r.f   // This must return a non-bound function.

                destroy r

                f("hello")  // function pointer should be still valid.
            }
        `)

		require.NoError(t, err)

		_, err = inter.Invoke("test")
		require.NoError(t, err)

		logs := getLogs()
		assert.Equal(t, []string{`"hello"`}, logs)
	})
}

func TestInterpretSimpleCompositeTypeFunctionMember(t *testing.T) {

	t.Parallel()

	t.Run("unwrapped function", func(t *testing.T) {
		// Only test the interpreter for now.
		// TODO: figure out how to register builtin-functions for
		// the vm during test setup.
		if *compile {
			t.SkipNow()
		}

		t.Parallel()

		resourceType := &sema.CompositeType{
			Location:   TestLocation,
			Identifier: "S",
			Kind:       common.CompositeKindStructure,
			Members:    &sema.StringMemberOrderedMap{},
		}

		resourceTypeID := resourceType.ID()

		baseTypeActivation := sema.NewVariableActivation(sema.BaseTypeActivation)
		baseTypeActivation.DeclareType(stdlib.StandardLibraryType{
			Name: resourceType.Identifier,
			Type: resourceType,
			Kind: common.DeclarationKindStructure,
		})

		address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

		inter, _, _ := testAccount(t, address,
			true,
			nil,
			`
            struct interface SI {
                fun foo(): String
            }

            fun test(s: {SI}) {
                account.storage.save(s, to: /storage/s)

                var rRef = account.storage.borrow<&{SI}>(from: /storage/s)!

                let f = rRef.foo   // This must return a bound function.

                // Replace the value
                account.storage.load<AnyStruct>(from: /storage/s)!
                account.storage.save("new value", to: /storage/s)

                f()  // function pointer should NOT be valid.
            }
        `,
			sema.Config{
				BaseTypeActivationHandler: func(_ common.Location) *sema.VariableActivation {
					return baseTypeActivation
				},
				CheckHandler: func(checker *sema.Checker, check func()) {
					if checker.Location == TestLocation {
						checker.Elaboration.SetCompositeType(
							resourceTypeID,
							resourceType,
						)
					}
					check()
				},
			},
		)

		interfaceType, err := inter.GetInterfaceType(TestLocation, "SI", "S.test.SI")
		require.NoError(t, err)

		foo, found := interfaceType.Members.Get("foo")
		require.True(t, found)

		funcType := foo.TypeAnnotation.Type.(*sema.FunctionType)

		resourceType.ExplicitInterfaceConformances = []*sema.InterfaceType{
			interfaceType,
		}

		resourceValue := interpreter.NewSimpleCompositeValue(nil,
			resourceTypeID,
			interpreter.ConvertSemaCompositeTypeToStaticCompositeType(nil, resourceType),
			nil,
			nil,
			nil,
			func(_ string, context interpreter.MemberAccessibleContext, _ interpreter.ReferenceValue) interpreter.FunctionValue {
				// IMPORTANT: Return an unwrapped function.
				return interpreter.NewStaticHostFunctionValue(
					context,
					funcType,
					func(invocation interpreter.Invocation) interpreter.Value {
						return interpreter.NewUnmeteredStringValue("hello from R")
					},
				)
			},
			nil,
			nil,
		)

		_, err = inter.Invoke("test", resourceValue)
		RequireError(t, err)

		var dereferenceError *interpreter.DereferenceError
		require.ErrorAs(t, err, &dereferenceError)
	})
}

func TestInterpretCompositeValueIDTracking(t *testing.T) {
	t.Parallel()

	logFunction := stdlib.NewInterpreterLogFunction(stdlib.FunctionLogger(func(message string) error {
		fmt.Fprintln(os.Stderr, message)
		return nil
	}))

	// liveValueID exposes the underlying atree map's current value ID for a
	// composite resource, used by the Cadence code to assert that the slab
	// split actually occurred.
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
			context interpreter.NativeFunctionContext,
			_ interpreter.TypeArgumentsIterator,
			_ interpreter.ArgumentTypesIterator,
			_ interpreter.Value,
			args []interpreter.Value,
		) interpreter.Value {
			ref := args[0].(*interpreter.EphemeralReferenceValue)
			composite := ref.Value.(*interpreter.CompositeValue)
			return interpreter.NewUnmeteredStringValue(composite.ValueID().String())
		},
	)

	baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
	baseValueActivation.DeclareValue(logFunction)
	baseValueActivation.DeclareValue(liveValueIDFunction)
	baseValueActivation.DeclareValue(stdlib.InterpreterAssertFunction)

	baseActivation := activations.NewActivation(nil, interpreter.BaseActivation)
	interpreter.Declare(baseActivation, logFunction)
	interpreter.Declare(baseActivation, liveValueIDFunction)
	interpreter.Declare(baseActivation, stdlib.InterpreterAssertFunction)

	inter, err := test_utils.ParseCheckAndInterpretWithAtreeValidationsDisabled(
		t,
		`
    access(all) entitlement Withdraw
    access(all) resource Vault {
        access(all) var balance: UFix64

        init(balance: UFix64) {
            self.balance = balance
        }

        access(Withdraw) fun withdraw(amount: UFix64): @Vault {
            self.balance = self.balance - amount
            return <- create Vault(balance: amount)
        }

        access(all) fun deposit(from: @Vault) {
            self.balance = self.balance + from.balance
            destroy from
        }
    }

    access(all) attachment A1 for Vault {
        access(all) var a1: String; access(all) var a2: String
        access(all) var a3: String; access(all) var a4: String
        init() { self.a1 = ""; self.a2 = ""; self.a3 = ""; self.a4 = "" }
        access(all) fun inflate() {
            self.a1 = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
            self.a2 = self.a1; self.a3 = self.a1; self.a4 = self.a1
        }
    }
    access(all) attachment A2 for Vault {
        access(all) var b1: String; access(all) var b2: String
        access(all) var b3: String; access(all) var b4: String
        init() { self.b1 = ""; self.b2 = ""; self.b3 = ""; self.b4 = "" }
        access(all) fun inflate() {
            self.b1 = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
            self.b2 = self.b1; self.b3 = self.b1; self.b4 = self.b1
        }
    }
    access(all) attachment A3 for Vault {
        access(all) var d1: String; access(all) var d2: String
        access(all) var d3: String; access(all) var d4: String
        init() { self.d1 = ""; self.d2 = ""; self.d3 = ""; self.d4 = "" }
        access(all) fun inflate() {
            self.d1 = "dddddddddddddddddddddddddddddddddddddd"
            self.d2 = self.d1; self.d3 = self.d1; self.d4 = self.d1
        }
    }
    access(all) attachment A4 for Vault {
        access(all) var e1: String; access(all) var e2: String
        access(all) var e3: String; access(all) var e4: String
        init() { self.e1 = ""; self.e2 = ""; self.e3 = ""; self.e4 = "" }
        access(all) fun inflate() {
            self.e1 = "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"
            self.e2 = self.e1; self.e3 = self.e1; self.e4 = self.e1
        }
    }
    access(all) attachment A5 for Vault {
        access(all) var g1: String; access(all) var g2: String
        access(all) var g3: String; access(all) var g4: String
        init() { self.g1 = ""; self.g2 = ""; self.g3 = ""; self.g4 = "" }
        access(all) fun inflate() {
            self.g1 = "gggggggggggggggggggggggggggggggggggggg"
            self.g2 = self.g1; self.g3 = self.g1; self.g4 = self.g1
        }
    }
    access(all) attachment A6 for Vault {
        access(all) var h1: String; access(all) var h2: String
        access(all) var h3: String; access(all) var h4: String
        init() { self.h1 = ""; self.h2 = ""; self.h3 = ""; self.h4 = "" }
        access(all) fun inflate() {
            self.h1 = "hhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhh"
            self.h2 = self.h1; self.h3 = self.h1; self.h4 = self.h1
        }
    }

    access(all) fun double(_ original: @Vault): @Vault {
        let empty <- original.withdraw(amount: 0.0)
        let stash <- original.withdraw(amount: 0.0)

        // Preparatory step: Attach a bunch of small and empty attachments
        let r1 <- attach A1() to <-original
        let r2 <- attach A2() to <-r1
        let r3 <- attach A3() to <-r2
        let r4 <- attach A4() to <-r3
        let r5 <- attach A5() to <-r4
        let r  <- attach A6() to <-r5

        // Create two EphemeralReferenceValues pointing to two different CompositeValues
        // which point to the same underlying dictionary
        var arr: @[Vault] <- [<-r]
        let ref  = &arr[0] as auth(Withdraw) &Vault
        let ref2 = &arr[0] as auth(Withdraw) &Vault

        // The shared-state cache (ConvertStoredValue) deduplicates the Cadence
        // wrappers, so both refs hold the same CompositeValue and observe the
        // same underlying atree map.
        assert(
            liveValueID(ref) == liveValueID(ref2),
            message: "before split: both refs should observe the same live atree value ID"
        )

        // Trigger an atree slab split on the underlying dictionary of Vault
        // by "inflating" those attachments
        ref[A1]!.inflate()
        ref[A2]!.inflate()
        ref[A3]!.inflate()
        ref[A4]!.inflate()
        ref[A5]!.inflate()
        ref[A6]!.inflate()

        // Both refs share the canonical wrapper, so the slab split through ref
        // is visible to ref2; their live value IDs continue to agree.
        assert(
            liveValueID(ref) == liveValueID(ref2),
            message: "after split: refs must still observe the same live atree value ID"
        )

        // Conversion roundtrip via AnyResource. Because ref and ref2 wrap the
        // same canonical CompositeValue, immortalRef is registered under the
        // same value ID as the others.
        let immortalRef = (ref2 as auth(Withdraw) &AnyResource) as! auth(Withdraw) &Vault
        // Move the vault. Reference invalidation must void immortalRef alongside
        // ref and ref2: all three are tracked under the same value ID.
        var extracted <- arr[0] <- empty

        stash.deposit(from: <- extracted)
        // This second withdraw must panic with InvalidatedResourceReferenceError,
        // because immortalRef was invalidated when the vault was moved above.
        stash.deposit(from: <- immortalRef.withdraw(amount: immortalRef.balance))

        destroy arr
        return <- stash
    }

    access(all) fun main() {
        let original <- create Vault(balance: 100.0)
        let second <- double(<- double(<- original))
        log("Successfully withdrawn: \(second.balance)")
        destroy second
    }
        `,
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

	// Multiple CompositeValue references to the same resource (e.g. via
	// repeated `&` accesses) must share a canonical wrapper, so when the
	// resource is moved/destroyed, all references see it as invalidated.
	// Without canonicalization, an EphemeralReferenceValue created from a
	// separate wrapper would survive invalidation through the canonical one,
	// allowing the balance to be withdrawn twice.
	// atree's shared per-container state keeps siblings structurally consistent;
	// the canonical wrapper cache ensures they also share Cadence-level state
	// like `isDestroyed`.
	_, err = inter.Invoke("main")
	RequireError(t, err)
	var invalidatedResourceReferenceError *interpreter.InvalidatedResourceReferenceError
	assert.ErrorAs(t, err, &invalidatedResourceReferenceError)
}

// TestInterpretCompositeAliasedMutationConsistency is the CompositeValue
// counterpart to TestInterpretArrayAliasedMutationConsistency / Dictionary
// version. It inflates attachments through `ref` to force an atree slab
// split of the resource's underlying dictionary, then withdraws through
// `ref2` (which would previously have observed a stale root). The
// canonical state must reflect both withdrawals; pre-fix, a withdrawal
// through the stale ref2 either silently wrote into a demoted child slab
// or read a stale balance, allowing double-spend.
func TestInterpretCompositeAliasedMutationConsistency(t *testing.T) {
	t.Parallel()

	logFunction := stdlib.NewInterpreterLogFunction(stdlib.FunctionLogger(func(message string) error {
		fmt.Fprintln(os.Stderr, message)
		return nil
	}))

	baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
	baseValueActivation.DeclareValue(logFunction)
	baseValueActivation.DeclareValue(stdlib.InterpreterAssertFunction)

	baseActivation := activations.NewActivation(nil, interpreter.BaseActivation)
	interpreter.Declare(baseActivation, logFunction)
	interpreter.Declare(baseActivation, stdlib.InterpreterAssertFunction)

	inter, err := test_utils.ParseCheckAndInterpretWithAtreeValidationsDisabled(
		t,
		`
    access(all) entitlement Withdraw
    access(all) resource Vault {
        access(all) var balance: UFix64

        init(balance: UFix64) {
            self.balance = balance
        }

        access(Withdraw) fun withdraw(amount: UFix64): @Vault {
            self.balance = self.balance - amount
            return <- create Vault(balance: amount)
        }

        access(all) fun deposit(from: @Vault) {
            self.balance = self.balance + from.balance
            destroy from
        }
    }

    access(all) attachment A1 for Vault {
        access(all) var a1: String; access(all) var a2: String
        access(all) var a3: String; access(all) var a4: String
        init() { self.a1 = ""; self.a2 = ""; self.a3 = ""; self.a4 = "" }
        access(all) fun inflate() {
            self.a1 = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
            self.a2 = self.a1; self.a3 = self.a1; self.a4 = self.a1
        }
    }
    access(all) attachment A2 for Vault {
        access(all) var b1: String; access(all) var b2: String
        access(all) var b3: String; access(all) var b4: String
        init() { self.b1 = ""; self.b2 = ""; self.b3 = ""; self.b4 = "" }
        access(all) fun inflate() {
            self.b1 = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
            self.b2 = self.b1; self.b3 = self.b1; self.b4 = self.b1
        }
    }
    access(all) attachment A3 for Vault {
        access(all) var d1: String; access(all) var d2: String
        access(all) var d3: String; access(all) var d4: String
        init() { self.d1 = ""; self.d2 = ""; self.d3 = ""; self.d4 = "" }
        access(all) fun inflate() {
            self.d1 = "dddddddddddddddddddddddddddddddddddddd"
            self.d2 = self.d1; self.d3 = self.d1; self.d4 = self.d1
        }
    }
    access(all) attachment A4 for Vault {
        access(all) var e1: String; access(all) var e2: String
        access(all) var e3: String; access(all) var e4: String
        init() { self.e1 = ""; self.e2 = ""; self.e3 = ""; self.e4 = "" }
        access(all) fun inflate() {
            self.e1 = "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"
            self.e2 = self.e1; self.e3 = self.e1; self.e4 = self.e1
        }
    }
    access(all) attachment A5 for Vault {
        access(all) var g1: String; access(all) var g2: String
        access(all) var g3: String; access(all) var g4: String
        init() { self.g1 = ""; self.g2 = ""; self.g3 = ""; self.g4 = "" }
        access(all) fun inflate() {
            self.g1 = "gggggggggggggggggggggggggggggggggggggg"
            self.g2 = self.g1; self.g3 = self.g1; self.g4 = self.g1
        }
    }
    access(all) attachment A6 for Vault {
        access(all) var h1: String; access(all) var h2: String
        access(all) var h3: String; access(all) var h4: String
        init() { self.h1 = ""; self.h2 = ""; self.h3 = ""; self.h4 = "" }
        access(all) fun inflate() {
            self.h1 = "hhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhh"
            self.h2 = self.h1; self.h3 = self.h1; self.h4 = self.h1
        }
    }

    access(all) fun main() {
        // 1000 = 100 (withdraw via ref) + 200 (withdraw via ref2) + 700 (remaining)
        let v <- create Vault(balance: 1000.0)
        let r1 <- attach A1() to <-v
        let r2 <- attach A2() to <-r1
        let r3 <- attach A3() to <-r2
        let r4 <- attach A4() to <-r3
        let r5 <- attach A5() to <-r4
        let r  <- attach A6() to <-r5

        var arr: @[Vault] <- [<-r]
        let ref  = &arr[0] as auth(Withdraw) &Vault
        let ref2 = &arr[0] as auth(Withdraw) &Vault

        // Inflate attachments to force a slab split of the resource's
        // underlying dictionary. After the split ref2 would, pre-fix, hold
        // a stale root pointer.
        ref[A1]!.inflate()
        ref[A2]!.inflate()
        ref[A3]!.inflate()
        ref[A4]!.inflate()
        ref[A5]!.inflate()
        ref[A6]!.inflate()

        // Both withdraw paths must observe the same canonical balance, so
        // their cumulative effect is exactly the sum.
        let w1 <- ref.withdraw(amount: 100.0)
        let w2 <- ref2.withdraw(amount: 200.0)

        assert(w1.balance == 100.0, message: "first withdrawal must produce 100")
        assert(w2.balance == 200.0, message: "second withdrawal must produce 200")
        assert(ref.balance == 700.0,
               message: "canonical balance after two withdrawals must be 1000 - 100 - 200 = 700")
        assert(ref.balance == ref2.balance,
               message: "both refs must observe the same canonical balance")

        destroy w1
        destroy w2
        destroy arr
    }
        `,
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
	require.NoError(t, err)
}

// TestInterpretCompositeFieldAliasedMutationConsistency exercises
// CompositeValue.GetField's canonicalization: two `&owner.bucket`
// references must alias the same inner-container wrapper, so a split
// triggered through one ref is observable through the other. Without
// canonicalization at the GetField path, the second `&owner.bucket`
// would build a fresh `*atree.Array` over the same slab and the first
// ref's appends - which trigger the split - would leave the second
// ref's root pointer stale.
func TestInterpretCompositeFieldAliasedMutationConsistency(t *testing.T) {
	t.Parallel()

	logFunction := stdlib.NewInterpreterLogFunction(stdlib.FunctionLogger(func(message string) error {
		fmt.Fprintln(os.Stderr, message)
		return nil
	}))

	baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
	baseValueActivation.DeclareValue(logFunction)
	baseValueActivation.DeclareValue(stdlib.InterpreterAssertFunction)

	baseActivation := activations.NewActivation(nil, interpreter.BaseActivation)
	interpreter.Declare(baseActivation, logFunction)
	interpreter.Declare(baseActivation, stdlib.InterpreterAssertFunction)

	inter, err := test_utils.ParseCheckAndInterpretWithAtreeValidationsDisabled(
		t,
		`
    access(all) resource Vault {
        access(all) var balance: UFix64
        init(balance: UFix64) { self.balance = balance }
    }

    access(all) resource Wallet {
        access(all) var bucket: @[Vault]
        init() {
            self.bucket <- [<-create Vault(balance: 0.0)]
        }
    }

    access(all) fun main() {
        let w <- create Wallet()

        // Two refs to the same composite field. Both must observe the
        // same canonical inner container, including after the underlying
        // atree slab splits.
        let ref  = &w.bucket as auth(Mutate) &[Vault]
        let ref2 = &w.bucket as auth(Mutate) &[Vault]

        var i: Int = 0
        while i < 200 {
            ref.append(<-create Vault(balance: UFix64(i)))
            i = i + 1
        }

        // Append through ref2. Pre-fix this would have written into a
        // demoted child slab, leaving the canonical root's count stale.
        ref2.append(<-create Vault(balance: 123.456))

        assert(ref.length == 202,
               message: "first ref must see the second ref's append")
        assert(ref2.length == 202,
               message: "second ref must see all appends through the canonical wrapper")

        // A fresh ref taken via GetField after the splits must still be
        // aliased with the originals.
        let postRef = &w.bucket as auth(Mutate) &[Vault]
        postRef.append(<-create Vault(balance: 999.0))
        assert(ref.length == 203,
               message: "post-split GetField must hand back the canonical wrapper")

        destroy w
    }
        `,
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
	require.NoError(t, err)
}

// TestInterpretCompositeForEachAttachmentAliasingConsistency verifies
// the ForEachAttachment iteration path canonicalizes the attachment
// wrappers it yields. A mutation through the iteration's callback must
// be visible to an externally-held reference to the same attachment
// (and vice versa), since both must wrap the same canonical
// CompositeValue.
func TestInterpretCompositeForEachAttachmentAliasingConsistency(t *testing.T) {
	t.Parallel()

	logFunction := stdlib.NewInterpreterLogFunction(stdlib.FunctionLogger(func(message string) error {
		fmt.Fprintln(os.Stderr, message)
		return nil
	}))

	baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
	baseValueActivation.DeclareValue(logFunction)
	baseValueActivation.DeclareValue(stdlib.InterpreterAssertFunction)

	baseActivation := activations.NewActivation(nil, interpreter.BaseActivation)
	interpreter.Declare(baseActivation, logFunction)
	interpreter.Declare(baseActivation, stdlib.InterpreterAssertFunction)

	inter, err := test_utils.ParseCheckAndInterpretWithAtreeValidationsDisabled(
		t,
		`
    access(all) resource Vault {
        access(all) var balance: UFix64
        init(balance: UFix64) { self.balance = balance }
    }

    access(all) attachment A1 for Vault {
        access(all) var s: String
        init() { self.s = "" }
        access(all) fun inflate() {
            self.s = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
        }
    }
    access(all) attachment A2 for Vault {
        access(all) var s: String
        init() { self.s = "" }
        access(all) fun inflate() {
            self.s = "BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB"
        }
    }

    access(all) fun main() {
        let v <- create Vault(balance: 1000.0)
        let v1 <- attach A1() to <-v
        let v2 <- attach A2() to <-v1

        var arr: @[Vault] <- [<-v2]

        // External reference to attachment A1, taken before iteration.
        // The forEachAttachment callback must yield the same canonical
        // A1 wrapper, so a mutation in the callback is observable here.
        let extA1Ref = arr[0][A1]!

        // Iterate attachments and inflate A1 via the callback's reference.
        // If the callback's wrapper is not canonical, the mutation lands
        // in a fresh wrapper and extA1Ref observes the pre-inflation
        // state.
        arr[0].forEachAttachment(fun (attRef: &AnyResourceAttachment) {
            if let a1Ref = attRef as? &A1 {
                a1Ref.inflate()
            }
        })

        assert(extA1Ref.s == "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
               message: "mutation in forEachAttachment must be visible through external ref")

        destroy arr
    }
        `,
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
	require.NoError(t, err)
}
