/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Dapper Labs, Inc.
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

		inter := parseCheckAndInterpret(t, `
            let dictionary: {String: String} = {"one" : "foo", "two" : "bar"}

            fun test() {
                var dictionaryRef = &dictionary as auth(Mutable) &{String: String}

                // Public functions
                dictionaryRef.containsKey("foo")
                dictionaryRef.forEachKey(fun(key: String): Bool {return true} )

                // Insertable functions
                dictionaryRef.insert(key: "three", "baz")

                // Removable functions
                dictionaryRef.remove(key: "foo")
            }
        `)

		_, err := inter.Invoke("test")
		require.NoError(t, err)
	})

	t.Run("non auth reference", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            let dictionary: {String: String} = {"one" : "foo", "two" : "bar"}

            fun test() {
                var dictionaryRef = &dictionary as &{String: String}

                // Public functions
                dictionaryRef.containsKey("foo")
                dictionaryRef.forEachKey(fun(key: String): Bool {return true} )
            }
	    `)

		_, err := inter.Invoke("test")
		require.NoError(t, err)
	})

	t.Run("insertable reference", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            let dictionary: {String: String} = {"one" : "foo", "two" : "bar"}

            fun test() {
                var dictionaryRef = &dictionary as auth(Mutable) &{String: String}

                // Public functions
                dictionaryRef.containsKey("foo")
                dictionaryRef.forEachKey(fun(key: String): Bool {return true} )

                // Insertable functions
                dictionaryRef.insert(key: "three", "baz")
            }
	    `)

		_, err := inter.Invoke("test")
		require.NoError(t, err)
	})

	t.Run("removable reference", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            let dictionary: {String: String} = {"one" : "foo", "two" : "bar"}

            fun test() {
                var dictionaryRef = &dictionary as auth(Mutable) &{String: String}

                // Public functions
                dictionaryRef.containsKey("foo")
                dictionaryRef.forEachKey(fun(key: String): Bool {return true} )

                // Removable functions
                dictionaryRef.remove(key: "foo")
            }
	    `)

		_, err := inter.Invoke("test")
		require.NoError(t, err)
	})
}
