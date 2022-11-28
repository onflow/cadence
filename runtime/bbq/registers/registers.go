package registers

import "github.com/onflow/cadence/runtime/sema"

type RegisterCounts struct {
	Ints  uint16
	Bools uint16
	Funcs uint16
}

func (c *RegisterCounts) NextIndex(registryType RegistryType) (index uint16) {
	switch registryType {
	case Int:
		index = c.Ints
		c.Ints++
	case Bool:
		index = c.Bools
		c.Bools++
	case Func:
		index = c.Funcs
		c.Funcs++
	default:
		panic("Unknown registry type")
	}

	return
}

type RegistryType int8

const (
	Int = iota
	Bool
	Func
)

func RegistryTypeFromSemaType(semaType sema.Type) RegistryType {
	switch semaType := semaType.(type) {
	case *sema.NumericType:
		switch semaType {
		case sema.IntType:
			return Int
		}
	case *sema.SimpleType:
		switch semaType {
		case sema.BoolType:
			return Bool
		}
	case *sema.FunctionType:
		return Func
	}

	panic("Unknown registry type")
}
