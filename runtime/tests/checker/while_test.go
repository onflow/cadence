package checker

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dapperlabs/cadence/runtime/sema"
	. "github.com/dapperlabs/cadence/runtime/tests/utils"
)

func TestCheckInvalidWhileTest(t *testing.T) {

	_, err := ParseAndCheck(t, `
      fun test() {
          while 1 {}
      }
    `)

	errs := ExpectCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
}

func TestCheckWhileTest(t *testing.T) {

	_, err := ParseAndCheck(t, `
      fun test() {
          while true {}
      }
    `)

	assert.NoError(t, err)
}

func TestCheckInvalidWhileBlock(t *testing.T) {

	_, err := ParseAndCheck(t, `
      fun test() {
          while true { x }
      }
    `)

	errs := ExpectCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
}

func TestCheckWhileBreakStatement(t *testing.T) {

	_, err := ParseAndCheck(t, `
       fun test() {
           while true {
               break
           }
       }
    `)

	assert.NoError(t, err)
}

func TestCheckInvalidWhileBreakStatement(t *testing.T) {

	_, err := ParseAndCheck(t, `
       fun test() {
           while true {
               fun () {
                   break
               }
           }
       }
    `)

	errs := ExpectCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.ControlStatementError{}, errs[0])
}

func TestCheckWhileContinueStatement(t *testing.T) {

	_, err := ParseAndCheck(t, `
       fun test() {
           while true {
               continue
           }
       }
    `)

	assert.NoError(t, err)
}

func TestCheckInvalidWhileContinueStatement(t *testing.T) {

	_, err := ParseAndCheck(t, `
       fun test() {
           while true {
               fun () {
                   continue
               }
           }
       }
    `)

	errs := ExpectCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.ControlStatementError{}, errs[0])
}
