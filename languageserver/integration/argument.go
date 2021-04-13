package integration

import (
	"github.com/onflow/cadence"
	jsoncdc "github.com/onflow/cadence/encoding/json"
)

type Argument struct {
	cadence.Value
}

func (a Argument) MarshalJSON() ([]byte, error) {
	return jsoncdc.Encode(a.Value)
}