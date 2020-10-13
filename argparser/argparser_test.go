package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/runtime/sema"
)

// TODO: improved and extend tests

func TestParseFix64(t *testing.T) {
	value, err := ParseArgument(&sema.Fix64Type{}, `-10.0`)
	require.NoError(t, err)

	expected, _ := cadence.NewFix64("-10.0")
	assert.Equal(t, expected, value)
}

func TestParseUFix64(t *testing.T) {
	value, err := ParseArgument(&sema.UFix64Type{}, `10.0`)
	require.NoError(t, err)

	expected, _ := cadence.NewUFix64("10.0")
	assert.Equal(t, expected, value)
}

func TestParseInt(t *testing.T) {
	value, err := ParseArgument(&sema.IntType{}, `10`)
	require.NoError(t, err)

	expected := cadence.NewInt(10)
	assert.Equal(t, expected, value)
}

func TestParseAddress(t *testing.T) {
	value, err := ParseArgument(&sema.AddressType{}, `0xe1f2a091f7bb5245`)
	require.NoError(t, err)

	expected, _ := cadence.NewAddressFromHex("e1f2a091f7bb5245")
	assert.Equal(t, expected, value)
}

func TestParseTransactionArguments(t *testing.T) {
	tx := `transaction(to: Address, amount: UFix64) { execute { log(foo) } }`

	values, err := ParseTransactionArguments(tx, []string{`0xe1f2a091f7bb5245`, `10.0`})
	require.NoError(t, err)

	require.Len(t, values, 2)

	expectedAddress, _ := cadence.NewAddressFromHex("e1f2a091f7bb5245")
	expectedAmount, _ := cadence.NewUFix64("10.0")

	assert.Equal(t, expectedAddress, values[0])
	assert.Equal(t, expectedAmount, values[1])
}
