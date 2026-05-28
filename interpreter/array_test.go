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
	. "github.com/onflow/cadence/test_utils/sema_utils"
)

func TestInterpretArrayFunctionEntitlements(t *testing.T) {

	t.Parallel()

	t.Run("mutable reference", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            fun test() {
                let array: [String] = ["foo", "bar"]
                var arrayRef = &array as auth(Mutate) &[String]

                // Public functions
                arrayRef.contains("hello")
                arrayRef.firstIndex(of: "hello")
                arrayRef.slice(from: 1, upTo: 1)
                arrayRef.concat(["hello"])

                // Insert functions
                arrayRef.append("baz")
                arrayRef.appendAll(["baz"])
                arrayRef.insert(at:0, "baz")

                // Remove functions
                arrayRef.remove(at: 1)
                arrayRef.removeFirst()
                arrayRef.removeLast()
            }
        `)

		_, err := inter.Invoke("test")
		require.NoError(t, err)
	})

	t.Run("non auth reference", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            fun test() {
                let array: [String] = ["foo", "bar"]
                var arrayRef = &array as &[String]

                // Public functions
                arrayRef.contains("hello")
                arrayRef.firstIndex(of: "hello")
                arrayRef.slice(from: 1, upTo: 1)
                arrayRef.concat(["hello"])
            }
	        `)

		_, err := inter.Invoke("test")
		require.NoError(t, err)
	})

	t.Run("insert reference", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            fun test() {
                let array: [String] = ["foo", "bar"]
                var arrayRef = &array as auth(Insert) &[String]

                // Public functions
                arrayRef.contains("hello")
                arrayRef.firstIndex(of: "hello")
                arrayRef.slice(from: 1, upTo: 1)
                arrayRef.concat(["hello"])

                // Insert functions
                arrayRef.append("baz")
                arrayRef.appendAll(["baz"])
                arrayRef.insert(at:0, "baz")
            }
	        `)

		_, err := inter.Invoke("test")
		require.NoError(t, err)
	})

	t.Run("remove reference", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            fun test() {
                let array: [String] = ["foo", "bar", "baz"]
                var arrayRef = &array as auth(Remove) &[String]

                // Public functions
                arrayRef.contains("hello")
                arrayRef.firstIndex(of: "hello")
                arrayRef.slice(from: 1, upTo: 1)
                arrayRef.concat(["hello"])

                // Remove functions
                arrayRef.remove(at: 1)
                arrayRef.removeFirst()
                arrayRef.removeLast()
            }
	        `)

		_, err := inter.Invoke("test")
		require.NoError(t, err)
	})
}

func TestCheckArrayReferenceTypeInferenceWithDowncasting(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndPrepare(t, `
		entitlement E
		entitlement F 
		entitlement G

		fun test() {
			let ef = &1 as auth(E, F) &Int
			let eg = &1 as auth(E, G) &Int
			let arr = [ef, eg]
			let ref = arr[0]
            let downcastRef = ref as! auth(E, F) &Int
		}
	
	`)

	_, err := inter.Invoke("test")
	require.Error(t, err)
	var forceCastTypeMismatchError *interpreter.ForceCastTypeMismatchError
	require.ErrorAs(t, err, &forceCastTypeMismatchError)
}

// TestInterpretArrayValueIDTracking verifies that the Cadence-level
// "immortal reference" exploit attempt is rejected: an attacker takes two
// `&outer[i]` references to the same inner inlined array, drives an atree
// slab split through one ref, and then casts the now-stale sibling ref
// through `&AnyResource` and back hoping to obtain a reference that survives
// invalidation when the inner array is moved.
//
// The runtime defends this in two layers: (1) the staleness check on any
// expression that dereferences the stale ref (the primary defense, which
// fires at the conversion site), and (2) the cached-valueID on ArrayValue
// that keeps a converted reference registered under the same stable ID as
// the original so it gets invalidated alongside it (the secondary defense,
// reachable only if the staleness check is bypassed).
//
// This test exercises the full Cadence-level exploit and asserts that the
// runtime rejects it. With the staleness check in place, the rejection
// surfaces as InvalidatedContainerViewError at the cast site.
func TestInterpretArrayValueIDTracking(t *testing.T) {
	t.Parallel()

	logFunction := stdlib.NewInterpreterLogFunction(stdlib.FunctionLogger(func(message string) error {
		fmt.Fprintln(os.Stderr, message)
		return nil
	}))

	// liveValueIDOf exposes the underlying atree array's current value ID so
	// the Cadence code can confirm the slab split actually occurred.
	//
	// It takes the *name* of the reference variable rather than the reference
	// itself: passing the stale ref directly would trip the staleness check on
	// the call-site expression and shadow the real exploit-site error.
	// Resolving the variable internally via GetValueOfVariable bypasses the
	// per-expression check.
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
			ref := context.GetValueOfVariable(name).(*interpreter.EphemeralReferenceValue)
			arrayValue := ref.Value.(*interpreter.ArrayValue)
			return interpreter.NewUnmeteredStringValue(arrayValue.LiveValueID().String())
		},
	)

	baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
	baseValueActivation.DeclareValue(logFunction)
	baseValueActivation.DeclareValue(liveValueIDOfFunction)
	baseValueActivation.DeclareValue(stdlib.InterpreterAssertFunction)

	baseActivation := activations.NewActivation(nil, interpreter.BaseActivation)
	interpreter.Declare(baseActivation, logFunction)
	interpreter.Declare(baseActivation, liveValueIDOfFunction)
	interpreter.Declare(baseActivation, stdlib.InterpreterAssertFunction)

	inter, err := test_utils.ParseCheckAndInterpretWithAtreeValidationsDisabled(
		t,
		`
    access(all) resource Vault {
        access(all) var balance: UFix64
        init(balance: UFix64) { self.balance = balance }
    }

    access(all) fun main() {
        // Outer array of inner resource arrays; outer[0] is the inner array we
        // will take two references to.
        let outer: @[[Vault]] <- [<-[<-create Vault(balance: 0.0)]]

        // Two EphemeralReferenceValues pointing to two different ArrayValues
        // which wrap the same underlying atree array.
        let ref  = &outer[0] as auth(Mutate) &[Vault]
        let ref2 = &outer[0] as auth(Mutate) &[Vault]

        // Both refs initially see the same root slab in the underlying atree array.
        assert(
            liveValueIDOf("ref") == liveValueIDOf("ref2"),
            message: "before split: both refs should observe the same live atree value ID"
        )

        // Append enough vaults via ref to grow the inner array past one slab
        // and trigger an atree array slab split.
        var i: Int = 0
        while i < 200 {
            ref.append(<-create Vault(balance: UFix64(i)))
            i = i + 1
        }

        // After the split, ref's array.root points to the new root slab while
        // ref2's array.root still points to the old root slab (with a freshly
        // assigned slab ID), so their live value IDs diverge.
        assert(
            liveValueIDOf("ref") != liveValueIDOf("ref2"),
            message: "after split: refs should observe diverged live atree value IDs"
        )

        // Exploit attempt: conversion roundtrip via AnyResource to obtain an
        // EphemeralReferenceValue from the stale ref2. If neither the
        // staleness check nor the cached-valueID mechanism caught this, the
        // resulting reference would be tracked under the (now-different) live
        // ValueID of the stale view and could survive the subsequent move as
        // an "immortal" reference.
        let immortalRef = (ref2 as auth(Mutate) &AnyResource) as! auth(Mutate) &[Vault]

        // Replace the inner array with an empty one. The move would invalidate
        // a properly-tracked immortalRef.
        var empty: @[Vault] <- []
        var extracted <- outer[0] <- empty
        destroy extracted

        // The exploit aims for this access to succeed and return the length
        // of the (now moved-out) old inner array — exposing the resources it
        // held. If the runtime defends correctly, execution never reaches
        // this line.
        log(immortalRef.length.toString())

        destroy outer
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
	RequireError(t, err)
	var staleAtreeViewError *interpreter.InvalidatedContainerViewError
	assert.ErrorAs(t, err, &staleAtreeViewError)
	// The staleness check (the primary defense) fires at the cast operand
	// `ref2` during expression evaluation, before the conversion completes.
	// This pins the rejection to the exploit's cast site; if a future change
	// shifts the rejection earlier (e.g. into a sanity assertion) or later
	// (e.g. all the way to `immortalRef.length`), this assertion will fail
	// and force a re-evaluation of which defense is actually firing.
	assert.Equal(t, 45, staleAtreeViewError.StartPosition().Line)
}
