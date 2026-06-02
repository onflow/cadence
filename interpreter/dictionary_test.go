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

// TestInterpretDictionaryValueIDTracking is the DictionaryValue counterpart to
// TestInterpretCompositeValueIDTracking. It exercises the same atree slab-split
// stale-view scenario, where two DictionaryValue instances wrap the same
// underlying atree map (created by accessing the same outer dictionary key
// twice) and a split through one instance leaves the other with a different
// live value ID. Without the cached valueID on DictionaryValue, an
// EphemeralReferenceValue created from the stale view would register under
// that different ID, bypass invalidation when the inner dictionary is moved,
// and survive as a dangling ref.
func TestInterpretDictionaryValueIDTracking(t *testing.T) {
	t.Parallel()

	logFunction := stdlib.NewInterpreterLogFunction(stdlib.FunctionLogger(func(message string) error {
		fmt.Fprintln(os.Stderr, message)
		return nil
	}))

	// liveValueID exposes the underlying atree map's current value ID so the
	// Cadence code can confirm the slab split actually occurred.
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
			dictValue := ref.Value.(*interpreter.DictionaryValue)
			return interpreter.NewUnmeteredStringValue(dictValue.ValueID().String())
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
    access(all) resource Vault {
        access(all) var balance: UFix64
        init(balance: UFix64) { self.balance = balance }
    }

    access(all) fun main() {
        // Outer dictionary mapping to inner resource dictionaries; outer["a"]
        // is the inner dictionary we will take two references to.
        let outer: @{String: {String: Vault}} <- {"a": <-{"k0": <-create Vault(balance: 0.0)}}

        // Two EphemeralReferenceValues to the same logical inner dictionary.
        // The shared-state cache (ConvertStoredValue) deduplicates the
        // Cadence wrappers, so both refs hold the same DictionaryValue and
        // observe the same underlying atree map.
        let ref  = (&outer["a"] as auth(Mutate) &{String: Vault}?)!
        let ref2 = (&outer["a"] as auth(Mutate) &{String: Vault}?)!

        // Both refs see the same live atree value ID.
        assert(
            liveValueID(ref) == liveValueID(ref2),
            message: "before split: both refs should observe the same live atree value ID"
        )

        // Insert enough entries to force the inner map's root to split.
        var i: Int = 0
        while i < 200 {
            let old <- ref.insert(key: "k".concat(i.toString()), <-create Vault(balance: UFix64(i)))
            destroy old
            i = i + 1
        }

        // Both refs share the canonical wrapper, so the slab split through ref
        // is visible to ref2; their live value IDs continue to agree.
        assert(
            liveValueID(ref) == liveValueID(ref2),
            message: "after split: refs must still observe the same live atree value ID"
        )

        // Conversion roundtrip via AnyResource. Because ref and ref2 wrap the
        // same canonical DictionaryValue, immortalRef is registered under the
        // same value ID as the others.
        let immortalRef = (ref2 as auth(Mutate) &AnyResource) as! auth(Mutate) &{String: Vault}

        // Replace the inner dictionary with an empty one. The move invalidates
        // all tracked references to the old inner dictionary.
        var empty: @{String: Vault} <- {}
        var extracted <- outer["a"] <- empty
        destroy extracted

        // immortalRef must be invalidated; touching it must panic with
        // InvalidatedResourceReferenceError.
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
	var invalidatedResourceReferenceError *interpreter.InvalidatedResourceReferenceError
	assert.ErrorAs(t, err, &invalidatedResourceReferenceError)
}

// TestInterpretDictionaryAliasedMutationConsistency is the DictionaryValue
// counterpart to TestInterpretArrayAliasedMutationConsistency. 200 inserts
// through ref trigger an atree slab split; an insert through ref2 (which
// previously would have observed a stale root) must land in the canonical
// structure. Iteration count and length-driven removal count must agree
// after extraction, and the ref2-inserted key must be observable from
// each traversal in the same way.
func TestInterpretDictionaryAliasedMutationConsistency(t *testing.T) {
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

    access(all) fun main() {
        let outer: @{String: {String: Vault}} <- {"a": <-{"k0": <-create Vault(balance: 0.0)}}

        let ref  = (&outer["a"] as auth(Mutate) &{String: Vault}?)!
        let ref2 = (&outer["a"] as auth(Mutate) &{String: Vault}?)!

        var i: Int = 0
        while i < 200 {
            let old <- ref.insert(key: "k".concat(i.toString()), <-create Vault(balance: UFix64(i)))
            destroy old
            i = i + 1
        }

        // Insert through ref2. Pre-fix this would have written into a
        // demoted child slab, leaving the canonical root's count stale.
        let staleOld <- ref2.insert(key: "stale", <-create Vault(balance: 123.456))
        destroy staleOld

        var empty: @{String: Vault}? <- {}
        var extractedOpt <- outer["a"] <- empty
        var extracted <- extractedOpt!

        let expectedLength = extracted.length

        var iteratedCount: Int = 0
        var keyFoundIterating = false
        for key in extracted.keys {
            iteratedCount = iteratedCount + 1
            if key == "stale" {
                keyFoundIterating = true
            }
        }

        var removalCount: Int = 0
        var keyFoundRemoving = false
        let keys = extracted.keys
        for key in keys {
            let removed <- extracted.remove(key: key)!
            if key == "stale" {
                keyFoundRemoving = true
            }
            destroy removed
            removalCount = removalCount + 1
        }

        assert(expectedLength == iteratedCount,
               message: "length must match for-iteration count")
        assert(iteratedCount == removalCount,
               message: "iteration count must match key-driven removal count")
        assert(keyFoundIterating == keyFoundRemoving,
               message: "stale-insert must be either seen by both traversals or by neither")

        destroy extracted
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
	require.NoError(t, err)
}
