package checker

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/sema"
	. "github.com/onflow/cadence/runtime/tests/utils"
)

func TestCheckInvalidEventTypeRequirementConformance(t *testing.T) {

	_, err := ParseAndCheck(t, `
      pub contract interface CI {

          pub event E(a: Int)
      }

      pub contract C: CI {

          pub event E(b: String)
      }
    `)

	errs := ExpectCheckerErrors(t, err, 1)

	require.IsType(t, &sema.ConformanceError{}, errs[0])
}
