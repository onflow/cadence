package vm

import (
	"github.com/onflow/cadence/runtime/bbq/opcode"
	"github.com/onflow/cadence/runtime/bbq/registers"
	"github.com/onflow/cadence/runtime/errors"
)

type Register struct {
	ints  []IntValue
	bools []BoolValue
	funcs []FunctionValue
}

func NewRegister(counts registers.RegisterCounts) Register {
	var ints []IntValue
	var bools []BoolValue
	var funcs []FunctionValue

	if counts.Ints > 0 {
		ints = make([]IntValue, counts.Ints)
	}

	if counts.Bools > 0 {
		bools = make([]BoolValue, counts.Bools)
	}

	if counts.Funcs > 0 {
		funcs = make([]FunctionValue, counts.Funcs)
	}

	return Register{
		ints:  ints,
		bools: bools,
		funcs: funcs,
	}
}

func (r *Register) initializeWithArguments(arguments []Value) {
	var regCounts registers.RegisterCounts

	for _, argument := range arguments {
		switch argument := argument.(type) {
		case IntValue:
			r.ints[regCounts.Ints] = argument
			regCounts.Ints++
		case BoolValue:
			r.bools[regCounts.Bools] = argument
			regCounts.Bools++
		case FunctionValue:
			r.funcs[regCounts.Funcs] = argument
			regCounts.Funcs++
		default:
			panic(errors.NewUnexpectedError("unknown type"))
		}
	}
}

func (r *Register) copyArguments(otherRegister *Register, arguments []opcode.Argument) {
	var regCounts registers.RegisterCounts

	for _, argument := range arguments {
		switch argument.Type {
		case registers.Int:
			otherRegister.ints[regCounts.Ints] = r.ints[argument.Index]
			regCounts.Ints++
		case registers.Bool:
			otherRegister.bools[regCounts.Bools] = r.bools[argument.Index]
			regCounts.Bools++
		case registers.Func:
			otherRegister.funcs[regCounts.Funcs] = r.funcs[argument.Index]
			regCounts.Funcs++
		}
	}
}
