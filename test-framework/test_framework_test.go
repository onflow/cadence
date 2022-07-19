package test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
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
	fmt.Println(PrettyPrintResults(results))
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
	fmt.Println(PrettyPrintResult("testFunc1", err))
}

func TestExecuteScript(t *testing.T) {
	code := `
        pub fun test() {
            var bc = Test.Blockchain()
            bc.executeScript("pub fun foo(): String {  return \"hello\" }")
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
