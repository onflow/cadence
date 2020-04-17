package runtime

import (
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
)

// A Value is a Cadence value emitted by the runtime.
//
// Runtime values can be converted to a simplified representation
// and then further encoded for transport or use in other languages
// and environments.
type Value struct {
	interpreter.Value
	inter *interpreter.Interpreter
}

func newRuntimeValue(value interpreter.Value, inter *interpreter.Interpreter) Value {
	return Value{
		Value: value,
		inter: inter,
	}
}

func (v Value) Interpreter() *interpreter.Interpreter {
	return v.inter
}

type Event struct {
	Type   Type
	Fields []Value
}

type Address = common.Address
