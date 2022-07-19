package integration

import (
	"testing"

	jsoncdc "github.com/onflow/cadence/encoding/json"

	"github.com/onflow/cadence"
	"github.com/stretchr/testify/assert"
)

func Test_Argument(t *testing.T) {

	cadenceVal, _ := cadence.NewString("test")
	val := Argument{cadenceVal}

	out, err := val.MarshalJSON()
	compare, _ := jsoncdc.Encode(cadenceVal)

	assert.NoError(t, err)
	assert.Equal(t, compare, out)
}
