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
	"testing"

	"github.com/stretchr/testify/require"
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
