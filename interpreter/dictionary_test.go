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

func TestInterpretDictionaryFunctionEntitlements(t *testing.T) {

	t.Parallel()

	t.Run("mutable reference", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            fun test() {
                let dictionary: {String: String} = {"one" : "foo", "two" : "bar"}
                var dictionaryRef = &dictionary as auth(Mutate) &{String: String}

                // Public functions
                dictionaryRef.containsKey("foo")
                dictionaryRef.forEachKey(fun(key: String): Bool {return true} )

                // Insert functions
                dictionaryRef.insert(key: "three", "baz")

                // Remove functions
                dictionaryRef.remove(key: "foo")
            }
        `)

		_, err := inter.Invoke("test")
		require.NoError(t, err)
	})

	t.Run("non auth reference", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            fun test() {
                let dictionary: {String: String} = {"one" : "foo", "two" : "bar"}
                var dictionaryRef = &dictionary as &{String: String}

                // Public functions
                dictionaryRef.containsKey("foo")
                dictionaryRef.forEachKey(fun(key: String): Bool {return true} )
            }
	    `)

		_, err := inter.Invoke("test")
		require.NoError(t, err)
	})

	t.Run("insert reference", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            fun test() {
                let dictionary: {String: String} = {"one" : "foo", "two" : "bar"}
                var dictionaryRef = &dictionary as auth(Mutate) &{String: String}

                // Public functions
                dictionaryRef.containsKey("foo")
                dictionaryRef.forEachKey(fun(key: String): Bool {return true} )

                // Insert functions
                dictionaryRef.insert(key: "three", "baz")
            }
	    `)

		_, err := inter.Invoke("test")
		require.NoError(t, err)
	})

	t.Run("remove reference", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            fun test() {
                let dictionary: {String: String} = {"one" : "foo", "two" : "bar"}
                var dictionaryRef = &dictionary as auth(Mutate) &{String: String}

                // Public functions
                dictionaryRef.containsKey("foo")
                dictionaryRef.forEachKey(fun(key: String): Bool {return true} )

                // Remove functions
                dictionaryRef.remove(key: "foo")
            }
	    `)

		_, err := inter.Invoke("test")
		require.NoError(t, err)
	})
}

// TestInterpretDictionaryValueIDTracking is the DictionaryValue counterpart
// to TestInterpretArrayValueIDTracking: it exercises the same Cadence-level
// "immortal reference" exploit attempt against an inner inlined dictionary
// and asserts the runtime rejects it. See TestInterpretArrayValueIDTracking
// for the full rationale around the two defense layers (staleness check
// then cached-valueID) and why the rejection currently surfaces at the
// cast site as InvalidatedContainerViewError.
func TestInterpretDictionaryValueIDTracking(t *testing.T) {
	t.Parallel()

	logFunction := stdlib.NewInterpreterLogFunction(stdlib.FunctionLogger(func(message string) error {
		fmt.Fprintln(os.Stderr, message)
		return nil
	}))

	// liveValueIDOf exposes the underlying atree map's current value ID so the
	// Cadence code can confirm the slab split actually occurred. It takes the
	// *name* of the reference variable so that resolving the (potentially
	// stale) reference happens inside Go via GetValueOfVariable, bypassing the
	// per-expression staleness check that would otherwise fire at this call.
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
			dictValue := ref.Value.(*interpreter.DictionaryValue)
			return interpreter.NewUnmeteredStringValue(dictValue.LiveValueID().String())
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
        // Outer dictionary mapping to inner resource dictionaries; outer["a"]
        // is the inner dictionary we will take two references to.
        let outer: @{String: {String: Vault}} <- {"a": <-{"k0": <-create Vault(balance: 0.0)}}

        // Two EphemeralReferenceValues pointing to two different DictionaryValues
        // which wrap the same underlying atree map.
        let ref  = (&outer["a"] as auth(Mutate) &{String: Vault}?)!
        let ref2 = (&outer["a"] as auth(Mutate) &{String: Vault}?)!

        // Both refs initially see the same root slab in the underlying atree map.
        assert(
            liveValueIDOf("ref") == liveValueIDOf("ref2"),
            message: "before split: both refs should observe the same live atree value ID"
        )

        // Insert enough entries via ref to grow the inner map past one slab
        // and trigger an atree map slab split.
        var i: Int = 0
        while i < 200 {
            let old <- ref.insert(key: "k".concat(i.toString()), <-create Vault(balance: UFix64(i)))
            destroy old
            i = i + 1
        }

        // After the split, ref's dictionary.root points to the new root slab while
        // ref2's dictionary.root still points to the old root slab (with a freshly
        // assigned slab ID), so their live value IDs diverge.
        assert(
            liveValueIDOf("ref") != liveValueIDOf("ref2"),
            message: "after split: refs should observe diverged live atree value IDs"
        )

        // Exploit attempt: conversion roundtrip via AnyResource to obtain an
        // EphemeralReferenceValue from the stale ref2. If neither the
        // staleness check nor the cached-valueID mechanism caught this, the
        // resulting reference would be tracked under the (now-different) live
        // ValueID of the stale view and survive the subsequent move.
        let immortalRef = (ref2 as auth(Mutate) &AnyResource) as! auth(Mutate) &{String: Vault}

        // Replace the inner dictionary with an empty one. The move would
        // invalidate a properly-tracked immortalRef.
        var empty: @{String: Vault} <- {}
        var extracted <- outer["a"] <- empty
        destroy extracted

        // The exploit aims for this access to succeed. If the runtime defends
        // correctly, execution never reaches this line.
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
	// Pinning the rejection here ensures the test fails loudly if a future
	// change shifts the rejection somewhere else (e.g. into a sanity
	// assertion above, or all the way to `immortalRef.length`).
	assert.Equal(t, 45, staleAtreeViewError.StartPosition().Line)
}
