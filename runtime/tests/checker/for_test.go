package checker

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dapperlabs/cadence/runtime/sema"
	. "github.com/dapperlabs/cadence/runtime/tests/utils"
)

func TestCheckForVariableSized(t *testing.T) {

	_, err := ParseAndCheck(t, `
      fun test() {
          let xs: [Int] = [1, 2, 3]
          for x in xs {
              x
          }
      }
    `)

	assert.NoError(t, err)
}

func TestCheckForConstantSized(t *testing.T) {

	_, err := ParseAndCheck(t, `
      fun test() {
          let xs: [Int; 3] = [1, 2, 3]
          for x in xs {
              x
          }
      }
    `)

	assert.NoError(t, err)
}

func TestCheckForEmpty(t *testing.T) {

	_, err := ParseAndCheck(t, `
      fun test() {
          for x in [] {}
      }
    `)

	assert.NoError(t, err)
}

func TestCheckInvalidForValueNonArray(t *testing.T) {

	_, err := ParseAndCheck(t, `
      fun test() {
          for x in 1 { }
      }
    `)

	errs := ExpectCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.TypeMismatchWithDescriptionError{}, errs[0])
}

func TestCheckInvalidForValueResource(t *testing.T) {

	_, err := ParseAndCheck(t, `
      resource R {}

      fun test() {
          let xs <- [<-create R()]
          for x in xs { }
          destroy xs
      }
    `)

	errs := ExpectCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.UnsupportedResourceForLoopError{}, errs[0])
}

func TestCheckInvalidForBlock(t *testing.T) {

	_, err := ParseAndCheck(t, `
      fun test() {
          for x in [1, 2, 3] { y }
      }
    `)

	errs := ExpectCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
}

func TestCheckForBreakStatement(t *testing.T) {

	_, err := ParseAndCheck(t, `
       fun test() {
           for x in [1, 2, 3] {
               break
           }
       }
    `)

	assert.NoError(t, err)
}

func TestCheckInvalidForBreakStatement(t *testing.T) {

	_, err := ParseAndCheck(t, `
       fun test() {
           for x in [1, 2, 3] {
               fun () {
                   break
               }
           }
       }
    `)

	errs := ExpectCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.ControlStatementError{}, errs[0])
}

func TestCheckForContinueStatement(t *testing.T) {

	_, err := ParseAndCheck(t, `
       fun test() {
           for x in [1, 2, 3] {
               continue
           }
       }
    `)

	assert.NoError(t, err)
}

func TestCheckInvalidForContinueStatement(t *testing.T) {

	_, err := ParseAndCheck(t, `
       fun test() {
           for x in [1, 2, 3] {
               fun () {
                   continue
               }
           }
       }
    `)

	errs := ExpectCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.ControlStatementError{}, errs[0])
}
