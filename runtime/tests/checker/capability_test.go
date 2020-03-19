package checker

import (
	"testing"

	"github.com/dapperlabs/cadence/runtime/sema"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckCapability(t *testing.T) {

	checker, err := ParseAndCheckWithPanic(t, `
      let x: Capability = panic("")
    `)

	require.NoError(t, err)

	assert.IsType(t,
		&sema.CapabilityType{},
		checker.GlobalValues["x"].Type,
	)
}
