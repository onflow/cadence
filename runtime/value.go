package runtime

import (
	"github.com/dapperlabs/cadence/runtime/common"
	"github.com/dapperlabs/cadence/runtime/interpreter"
)

type Value = interpreter.Value

type Event struct {
	Type   Type
	Fields []Value
}

type Address = common.Address
