package stdlib

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/sema"
)

func TestCryptoContract(t *testing.T) {
	require.IsType(t, &sema.Checker{}, CryptoChecker)
}
