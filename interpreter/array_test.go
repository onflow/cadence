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

// TestInterpretArrayValueIDTracking is the ArrayValue counterpart to
// TestInterpretCompositeValueIDTracking. It exercises the same atree slab-split
// stale-view scenario, where two ArrayValue instances wrap the same underlying
// atree array (created by accessing the same outer array element twice) and a
// split through one instance leaves the other with a different live value ID.
// Without the cached valueID on ArrayValue, an EphemeralReferenceValue created
// from the stale view would register under that different ID, bypass
// invalidation when the inner array is moved, and survive as a dangling ref.
func TestInterpretArrayValueIDTracking(t *testing.T) {
	t.Parallel()

	logFunction := stdlib.NewInterpreterLogFunction(stdlib.FunctionLogger(func(message string) error {
		fmt.Fprintln(os.Stderr, message)
		return nil
	}))

	// liveValueID exposes the underlying atree array's current value ID so the
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
			arrayValue := ref.Value.(*interpreter.ArrayValue)
			return interpreter.NewUnmeteredStringValue(arrayValue.LiveValueID().String())
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
        // Outer array of inner resource arrays; outer[0] is the inner array we
        // will take two references to.
        let outer: @[[Vault]] <- [<-[<-create Vault(balance: 0.0)]]

        // Two EphemeralReferenceValues pointing to two different ArrayValues
        // which wrap the same underlying atree array.
        let ref  = &outer[0] as auth(Mutate) &[Vault]
        let ref2 = &outer[0] as auth(Mutate) &[Vault]

        // Both refs initially see the same root slab in the underlying atree array.
        assert(
            liveValueID(ref) == liveValueID(ref2),
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
            liveValueID(ref) != liveValueID(ref2),
            message: "after split: refs should observe diverged live atree value IDs"
        )

        // Conversion roundtrip via AnyResource to create an EphemeralReferenceValue
        // from ref2. Without the stable cached valueID on ArrayValue, this reference
        // would be tracked under the (now-different) live ValueID of the stale view.
        let immortalRef = (ref2 as auth(Mutate) &AnyResource) as! auth(Mutate) &[Vault]

        // Replace the inner array with an empty one. The move invalidates all
        // tracked references to the old inner array.
        var empty: @[Vault] <- []
        var extracted <- outer[0] <- empty
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
	var staleAtreeViewError *interpreter.InvalidatedContainerViewError
	assert.ErrorAs(t, err, &staleAtreeViewError)
}
