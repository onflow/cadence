/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2022 Dapper Labs, Inc.
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

package test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunningMultipleTests(t *testing.T) {
	code := `
        pub fun testFunc1() {
            assert(false)
        }

        pub fun testFunc2() {
            assert(true)
        }
    `

	results := RunTests(code)
	require.Len(t, results, 2)
	assert.Error(t, results["testFunc1"])
	assert.NoError(t, results["testFunc2"])
}

func TestRunningSingleTest(t *testing.T) {
	code := `
        pub fun testFunc1() {
            assert(false)
        }

        pub fun testFunc2() {
            assert(true)
        }
    `

	err := RunTest(code, "testFunc1")
	assert.Error(t, err)

	err = RunTest(code, "testFunc2")
	assert.NoError(t, err)
}

func TestExecuteScript(t *testing.T) {
	code := `
        import Test

        pub fun test() {
            var blockchain = Test.newEmulatorBlockchain()
            var result = blockchain.executeScript("pub fun main(): Int {  return 2 + 3 }")

            assert(result.status == Test.ResultStatus.succeeded)
            assert((result.returnValue! as! Int) == 5)

            log(result.returnValue)
        }
    `

	err := RunTest(code, "test")
	assert.NoError(t, err)
}

func TestLoadContract(t *testing.T) {
	code := `
        import FooContract from "./FooContract"

        pub fun test() {
            var foo = FooContract()
            foo.hello()
        }

        pub struct Bar {
        }
    `

	err := RunTest(code, "test")
	assert.NoError(t, err)
}
