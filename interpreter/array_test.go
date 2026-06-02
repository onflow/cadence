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
// TestInterpretCompositeValueIDTracking.
// Two references to the same logical inner array (created by accessing the same
// outer array element twice) must share a canonical wrapper, so when the inner
// array is moved/invalidated, all references see it as invalidated.
// Without canonicalization, an EphemeralReferenceValue created from a separate
// wrapper would survive invalidation through the canonical one and dangle.
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
			return interpreter.NewUnmeteredStringValue(arrayValue.ValueID().String())
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

        // Two EphemeralReferenceValues to the same logical inner array.
        // The shared-state cache (ConvertStoredValue) deduplicates the
        // Cadence wrappers, so both refs hold the same ArrayValue and observe
        // the same underlying *atree.Array.
        let ref  = &outer[0] as auth(Mutate) &[Vault]
        let ref2 = &outer[0] as auth(Mutate) &[Vault]

        // Both refs see the same live atree value ID.
        assert(
            liveValueIDOf("ref") == liveValueIDOf("ref2"),
            message: "before split: both refs should observe the same live atree value ID"
        )

        // Append enough vaults to force the inner array's root to split.
        var i: Int = 0
        while i < 200 {
            ref.append(<-create Vault(balance: UFix64(i)))
            i = i + 1
        }

        // Both refs share the canonical wrapper, so the slab split through ref
        // is visible to ref2; their live value IDs continue to agree.
        assert(
            liveValueIDOf("ref") == liveValueIDOf("ref2"),
            message: "after split: refs must still observe the same live atree value ID"
        )

        // Conversion roundtrip via AnyResource. Because ref and ref2 wrap the
        // same canonical ArrayValue, immortalRef is registered under the same
        // value ID as the others.
        let immortalRef = (ref2 as auth(Mutate) &AnyResource) as! auth(Mutate) &[Vault]

        // Replace the inner array with an empty one. The move invalidates all
        // tracked references to the old inner array, including immortalRef.
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
	var invalidatedResourceReferenceError *interpreter.InvalidatedResourceReferenceError
	require.ErrorAs(t, err, &invalidatedResourceReferenceError)
	assert.Equal(t, 54, invalidatedResourceReferenceError.StartPosition().Line)
}

// TestInterpretArrayAliasedMutationConsistency verifies that mutations
// through one of multiple references to the same inner array are observed
// consistently by the others. Specifically: 200 appends through `ref`
// trigger an atree slab split, then 1 append through `ref2` must land in
// the canonical structure (not in a stale child slab). Iteration count
// and length-driven removeLast count must agree after extraction.
//
// Before the canonical wrapper cache, each `&outer[0]` produced its own
// wrapper around a separate *atree.Array. After the split through `ref`,
// `ref2`'s root pointer went stale; the stale append silently inserted
// a phantom Vault into a demoted child slab without updating the
// canonical root's count, producing iteration/length disagreement and a
// resource-duplication vector.
func TestInterpretArrayAliasedMutationConsistency(t *testing.T) {
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
        let outer: @[[Vault]] <- [<-[<-create Vault(balance: 0.0)]]

        let ref  = &outer[0] as auth(Mutate) &[Vault]
        let ref2 = &outer[0] as auth(Mutate) &[Vault]

        var i: Int = 0
        while i < 200 {
            ref.append(<-create Vault(balance: UFix64(i)))
            i = i + 1
        }

        ref2.append(<- create Vault(balance: 123.456))

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

        assert(iteratedCount == extracted.length + removalCount,
               message: "for-iteration count must match length seen at start")
        assert(iteratedCount == removalCount,
               message: "iteration count must match removeLast count")
        assert(elementFoundIterating == elementFoundRemoving,
               message: "stale-append must be either seen by both traversals or by neither")

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

// TestInterpretArrayForLoopAliasingConsistency verifies the for-loop
// iteration path (ArrayIterator.Next) hands back canonical wrappers, so a
// loop variable taken via `for x in &arr as &[T]` is aliased with an
// `&arr[i]` reference. Without canonicalization at the iterator, the loop
// variable would hold a fresh `*atree.Array` over the same slab; a split
// triggered through the external reference would leave the loop's wrapper
// stale (and vice versa).
func TestInterpretArrayForLoopAliasingConsistency(t *testing.T) {
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
        let outer: @[[Vault]] <- [<-[<-create Vault(balance: 0.0)]]

        let externalRef = &outer[0] as auth(Mutate) &[Vault]

        // Cause a split through the external reference so any non-
        // canonical wrapper the for-loop subsequently constructs would
        // observe a different live root.
        var i: Int = 0
        while i < 200 {
            externalRef.append(<-create Vault(balance: UFix64(i)))
            i = i + 1
        }

        // Iterate via reference. The loop variable's element references
        // are unauthorized (Cadence doesn't propagate the outer auth into
        // element refs), but they must still observe the canonical
        // post-split state - which means the iterator's element load
        // (ArrayIterator.Next) must hand back the canonical wrapper.
        var loopMaxLength: Int = 0
        var loopCount: Int = 0
        for inner in &outer as &[[Vault]] {
            loopCount = loopCount + 1
            if inner.length > loopMaxLength {
                loopMaxLength = inner.length
            }
        }

        assert(loopCount == 1,
               message: "outer has exactly one inner array")
        assert(loopMaxLength == 201,
               message: "for-loop ref must see all 201 elements through canonical wrapper")

        // After the loop, a freshly-taken authorized ref must still be
        // aliased with externalRef.
        let postRef = &outer[0] as auth(Mutate) &[Vault]
        postRef.append(<-create Vault(balance: 999.0))
        assert(externalRef.length == 202,
               message: "append through fresh ref must be visible to original external ref")

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

// TestInterpretArraySliceDoesNotInvalidateAliases verifies that calling
// container methods that build a new array (slice, reverse, filter, map)
// on a source whose elements are externally aliased does not invalidate
// the source's aliased references. The slice/reverse/etc. element load
// is deliberately *not* canonicalized (the element is fed through a
// closure and/or transferred into the result via a read-only iterator),
// so it must not poison the cache for the source. The complementary
// asReference aliasing bug (where the reference stored in the result
// holds a transient wrapper that goes stale after an external split)
// is tracked separately - fixing it requires switching slice's
// iterator to non-read-only so canonicalization can adopt a fresh
// mutable atree instance.
func TestInterpretArraySliceDoesNotInvalidateAliases(t *testing.T) {
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
    access(all) fun main() {
        let outer: [[Int]] = [[1, 2, 3], [4, 5, 6], [7, 8, 9]]

        let innerRef = &outer[0] as auth(Mutate) &[Int]

        // Slice the outer array. This iterates outer's elements and
        // transfers them into a new array. It must not invalidate
        // innerRef, which is a separate aliased reference to outer[0].
        let outerRef = &outer as &[[Int]]
        let sliced = outerRef.slice(from: 0, upTo: 2)
        assert(sliced.length == 2)

        // innerRef must still observe outer[0] correctly.
        assert(innerRef.length == 3,
               message: "outer slice must not invalidate alias to outer[0]")
        innerRef.append(99)
        assert(innerRef.length == 4,
               message: "alias must remain mutable after outer.slice")
        assert(outer[0].length == 4,
               message: "mutation through alias must be visible on outer[0]")
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

// TestInterpretArrayAsReferenceContainerMethodAliasingConsistency covers
// the asReference branch of Array.slice / .concat / .filter / .map /
// .toVariableSized / .toConstantSized: each builds a result whose
// elements are references back to source elements (stored in the result
// via NonStorable, surviving the call). Without canonicalization +
// mutable iterator, the reference wraps a transient *atree.Array that
// goes stale after a split triggered through a separate canonical
// reference - result[0].length would report a child-slab count rather
// than the full count after 200 large-string appends.
//
// Uses large strings so 200 appends genuinely exceed a single slab and
// trigger splitRoot. With smaller payloads (e.g. small ints), the array
// fits in one data slab and both wrappers share that slab struct,
// hiding the bug.
func TestInterpretArrayAsReferenceContainerMethodAliasingConsistency(t *testing.T) {
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

	cases := []struct {
		name   string
		setup  string
		expr   string
		alias  string
		mutVia string
	}{
		{
			name:   "slice",
			setup:  `let outer: [[String]] = [["x"]]; let outerRef = &outer as &[[String]]`,
			expr:   "outerRef.slice(from: 0, upTo: 1)",
			alias:  "&outer[0] as auth(Mutate) &[String]",
			mutVia: "aliasRef",
		},
		{
			name:   "reverse",
			setup:  `let outer: [[String]] = [["x"]]; let outerRef = &outer as &[[String]]`,
			expr:   "outerRef.reverse()",
			alias:  "&outer[0] as auth(Mutate) &[String]",
			mutVia: "aliasRef",
		},
		{
			name:   "concat",
			setup:  `let outer: [[String]] = [["x"]]; let outerRef = &outer as &[[String]]`,
			expr:   "outerRef.concat([])",
			alias:  "&outer[0] as auth(Mutate) &[String]",
			mutVia: "aliasRef",
		},
		{
			name:   "filter",
			setup:  `let outer: [[String]] = [["x"]]; let outerRef = &outer as &[[String]]`,
			expr:   "outerRef.filter(view fun (e: &[String]): Bool { return true })",
			alias:  "&outer[0] as auth(Mutate) &[String]",
			mutVia: "aliasRef",
		},
		{
			name:   "map",
			setup:  `let outer: [[String]] = [["x"]]; let outerRef = &outer as &[[String]]`,
			expr:   "outerRef.map(view fun (e: &[String]): &[String] { return e })",
			alias:  "&outer[0] as auth(Mutate) &[String]",
			mutVia: "aliasRef",
		},
		{
			name:   "toVariableSized",
			setup:  `let outer: [[String]; 1] = [["x"]]; let outerRef = &outer as &[[String]; 1]`,
			expr:   "outerRef.toVariableSized()",
			alias:  "&outer[0] as auth(Mutate) &[String]",
			mutVia: "aliasRef",
		},
		{
			name:   "toConstantSized",
			setup:  `let outer: [[String]] = [["x"]]; let outerRef = &outer as &[[String]]`,
			expr:   "outerRef.toConstantSized<[&[String]; 1]>()!",
			alias:  "&outer[0] as auth(Mutate) &[String]",
			mutVia: "aliasRef",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			code := fmt.Sprintf(`
    access(all) fun main(): Int {
        %s

        let result = %s

        let aliasRef = %s

        let big = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
        var i: Int = 0
        while i < 200 {
            %s.append(big)
            i = i + 1
        }

        return result[0].length
    }
`, tc.setup, tc.expr, tc.alias, tc.mutVia)

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

			result, err := inter.Invoke("main")
			require.NoError(t, err)
			require.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(201), result,
				"result[0].length must reflect post-split state via canonicalized wrapper")
		})
	}
}

// TestInterpretArrayAsReferenceContainerMethodMutationPropagation verifies
// that mutations performed through a reference returned by the asReference branch
// of Array.slice / .concat / .filter / .map / .toVariableSized / .toConstantSized
// propagate to the original container.
//
// `access(all)` mutating functions on a struct (like `S.inc()`) are reachable
// even on unauthorized references — Cadence's checker does not require an
// entitlement to call them. The implementations must therefore use the
// mutable atree iterator on the asReference branch, so that mutations through
// the result-stored reference fire the real parent-notification callback
// rather than tripping the trap callback installed by a read-only iterator.
func TestInterpretArrayAsReferenceContainerMethodMutationPropagation(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		body string
	}{
		{
			name: "slice",
			body: `let part = ref.slice(from: 0, upTo: 2); part[0].inc()`,
		},
		{
			name: "concat",
			body: `let part = ref.concat([]); part[0].inc()`,
		},
		{
			name: "filter",
			body: `let part = ref.filter(view fun (_: &S): Bool { return true }); part[0].inc()`,
		},
		{
			name: "map",
			body: `let _ = ref.map(fun (s: &S) { s.inc() })`,
		},
		{
			name: "toConstantSized",
			body: `let part = ref.toConstantSized<[&S; 3]>()!; part[0].inc()`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			code := fmt.Sprintf(`
                access(all) struct S {
                    access(all) var x: Int
                    init() { self.x = 1 }
                    access(all) fun inc() { self.x = self.x + 1 }
                }

                access(all) fun main(): Int {
                    let xs = [S(), S(), S()]
                    let ref = &xs as &[S]
                    %s
                    return xs[0].x
                }
            `, tc.body)

			inter := parseCheckAndPrepare(t, code)
			result, err := inter.Invoke("main")
			require.NoError(t, err)
			require.Equal(t,
				interpreter.NewUnmeteredIntValueFromInt64(2),
				result,
				"mutation through result reference must propagate to original")
		})
	}

	// toVariableSized takes a fixed-sized array, so test it separately.
	t.Run("toVariableSized", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            access(all) struct S {
                access(all) var x: Int
                init() { self.x = 1 }
                access(all) fun inc() { self.x = self.x + 1 }
            }

            access(all) fun main(): Int {
                let xs: [S; 3] = [S(), S(), S()]
                let ref = &xs as &[S; 3]
                let part = ref.toVariableSized()
                part[0].inc()
                return xs[0].x
            }
        `)
		result, err := inter.Invoke("main")
		require.NoError(t, err)
		require.Equal(t,
			interpreter.NewUnmeteredIntValueFromInt64(2),
			result,
			"mutation through toVariableSized result reference must propagate to original")
	})
}

// TestInterpretOptionalContainerAliasingConsistency verifies the
// SomeStorable.StoredValue path: when a stored optional wraps a
// container (e.g. `{String: [Vault]?}`), accessing the optional should
// produce a SomeValue whose inner container wrapper is canonical, so
// aliased references through different access paths share state.
//
// SomeStorable.StoredValue itself does not have cache access (gauge is
// the decode-time gauge, not the current context), so canonicalization
// happens in canonicalizeContainerElement by recursing into SomeValue.
func TestInterpretOptionalContainerAliasingConsistency(t *testing.T) {
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
        access(all) var vaults: @[Vault]?
        init() {
            self.vaults <- [<-create Vault(balance: 0.0)]
        }
    }

    access(all) fun main() {
        // Wallet has an optional resource-array field. In storage the
        // field is Some<[Vault]>; both refs taken through &w.vaults
        // must alias the same canonical inner ArrayValue. Without
        // SomeValue handling in canonicalizeContainerElement, each
        // GetField would yield a fresh SomeValue whose inner ArrayValue
        // is non-canonical, and split-through-one would leave the other
        // stale.
        let w <- create Wallet()

        let ref  = (&w.vaults as auth(Mutate) &[Vault]?)!
        let ref2 = (&w.vaults as auth(Mutate) &[Vault]?)!

        var i: Int = 0
        while i < 200 {
            ref.append(<-create Vault(balance: UFix64(i)))
            i = i + 1
        }

        ref2.append(<-create Vault(balance: 999.0))

        assert(ref.length == 202,
               message: "ref must see ref2's append")
        assert(ref2.length == 202,
               message: "ref2 must see all appends through canonical wrapper")

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

// TestInterpretOptionalContainerAliasingViaArrayIndex exercises the
// SomeStorable path through array indexing: when an array's elements
// are optional containers, ArrayValue.Get retrieves a SomeValue whose
// inner must be canonical so aliased references through different
// outer[i] lookups share state.
func TestInterpretOptionalContainerAliasingViaArrayIndex(t *testing.T) {
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
        // Outer array of optional resource arrays - the storable for
        // each element is SomeStorable wrapping an ArrayStorable.
        let outer: @[[Vault]?] <- [<-[<-create Vault(balance: 0.0)]]

        let ref  = (&outer[0] as auth(Mutate) &[Vault]?)!
        let ref2 = (&outer[0] as auth(Mutate) &[Vault]?)!

        var i: Int = 0
        while i < 200 {
            ref.append(<-create Vault(balance: UFix64(i)))
            i = i + 1
        }

        ref2.append(<-create Vault(balance: 999.0))

        assert(ref.length == 202,
               message: "ref must see ref2's append")
        assert(ref2.length == 202,
               message: "ref2 must see all appends")

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

// TestInterpretOptionalContainerAliasingViaDictionaryLookup exercises
// the SomeStorable path through dictionary lookup: when a dict's value
// type is an optional container, DictionaryValue.Get retrieves a
// SomeValue whose inner must be canonical.
func TestInterpretOptionalContainerAliasingViaDictionaryLookup(t *testing.T) {
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
        // Dictionary value type is itself an optional resource array.
        // Storage element is SomeStorable wrapping ArrayStorable; the
        // dictionary lookup is itself optional, so the final type after
        // unwrapping is auth(Mutate) &[Vault]?.
        let outer: @{String: [Vault]?} <- {"a": <-[<-create Vault(balance: 0.0)]}

        let ref  = (&outer["a"] as auth(Mutate) &[Vault]??)!!
        let ref2 = (&outer["a"] as auth(Mutate) &[Vault]??)!!

        var i: Int = 0
        while i < 200 {
            ref.append(<-create Vault(balance: UFix64(i)))
            i = i + 1
        }

        ref2.append(<-create Vault(balance: 999.0))

        assert(ref.length == 202,
               message: "ref must see ref2's append")
        assert(ref2.length == 202,
               message: "ref2 must see all appends")

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

// TestInterpretArrayPromoteRootAliasingConsistency exercises the dual
// of the split-induced aliasing scenario: enough removals through one
// aliased reference to drive atree's `promoteChildAsNewRoot` (which
// fires when a meta-slab root shrinks to a single child). The
// previously-identified gap was that a per-Cadence-wrapper slab-ID
// check (cached `valueID` vs live `array.ValueID()`) does NOT detect
// promote — promoteChildAsNewRoot keeps the root slab ID stable. With
// canonicalization, both refs share one Cadence wrapper whose `.array`
// is kept in sync, so the gap does not manifest at the user-visible
// level. This test pins that behavior.
func TestInterpretArrayPromoteRootAliasingConsistency(t *testing.T) {
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
            // Build an inner resource array large enough that the atree
            // tree has multiple data slabs under a meta-slab root.
            let outer: @[[Vault]] <- [<-[]]

            let ref  = &outer[0] as auth(Mutate) &[Vault]
            let ref2 = &outer[0] as auth(Mutate) &[Vault]

            // Fill: triggers splitRoot at some threshold.
            var i: Int = 0
            while i < 400 {
                ref.append(<-create Vault(balance: UFix64(i)))
                i = i + 1
            }
            assert(ref.length == 400, message: "after fill: length must be 400")
            assert(ref2.length == 400, message: "after fill: ref2 must observe same length")

            // Drain via ref2 until only one element remains: this drives
            // the meta-slab root to one-child state, triggering
            // promoteChildAsNewRoot.
            while ref2.length > 1 {
                let v <- ref2.removeLast()
                destroy v
            }
            assert(ref.length == 1, message: "after drain: ref must see promoted length")
            assert(ref2.length == 1, message: "after drain: ref2 must see promoted length")

            // Cross-mutate again: append via ref, observe via ref2.
            ref.append(<-create Vault(balance: 999.0))
            assert(ref2.length == 2, message: "post-promote append must be visible to alias")

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

// TestInterpretArraySplitAndPromoteAliasingConsistency drives both
// structural transitions through different aliased references in one
// run: grow via ref1 until splitRoot, shrink via ref2 until
// promoteChildAsNewRoot, then grow again. At every step both
// references must observe identical state through the canonical
// wrapper.
func TestInterpretArraySplitAndPromoteAliasingConsistency(t *testing.T) {
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
            let outer: @[[Vault]] <- [<-[]]

            let ref  = &outer[0] as auth(Mutate) &[Vault]
            let ref2 = &outer[0] as auth(Mutate) &[Vault]

            // Phase 1: split via ref1.
            var i: Int = 0
            while i < 300 {
                ref.append(<-create Vault(balance: UFix64(i)))
                i = i + 1
            }
            assert(ref2.length == 300, message: "phase 1: ref2 must see split-grown length")

            // Phase 2: promote via ref2.
            while ref2.length > 1 {
                let v <- ref2.removeLast()
                destroy v
            }
            assert(ref.length == 1, message: "phase 2: ref must see promoted length")

            // Phase 3: grow again via ref1, observe via ref2.
            i = 0
            while i < 50 {
                ref.append(<-create Vault(balance: UFix64(i + 1000)))
                i = i + 1
            }
            assert(ref2.length == 51, message: "phase 3: ref2 must see post-promote growth")

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
