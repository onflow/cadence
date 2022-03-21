package main

import (
	"fmt"
	"runtime"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/parser2"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib"
	"github.com/onflow/cadence/runtime/tests/utils"

	"gonum.org/v1/gonum/mat"
)

var memory_kinds = []common.MemoryKind{
	common.MemoryKindBool,
	common.MemoryKindAddress,
	common.MemoryKindString,
	common.MemoryKindCharacter,
	// common.MemoryKindMetaType,
	common.MemoryKindNumber,
	common.MemoryKindArray,
	common.MemoryKindDictionary,
	common.MemoryKindComposite,
	common.MemoryKindOptional,
	common.MemoryKindNil,
	common.MemoryKindVoid,
	// common.MemoryKindTypeValue,
	common.MemoryKindPathValue,
	common.MemoryKindCapabilityValue,
	common.MemoryKindLinkValue,
	common.MemoryKindStorageReferenceValue,
	common.MemoryKindEphemeralReferenceValue,
	common.MemoryKindInterpretedFunction,
	common.MemoryKindHostFunction,
	common.MemoryKindBoundFunction,
	common.MemoryKindBigInt,
}

type calibrationMemoryGauge struct {
	meter map[common.MemoryKind]uint64
}

func newTestMemoryGauge() *calibrationMemoryGauge {
	return &calibrationMemoryGauge{
		meter: make(map[common.MemoryKind]uint64),
	}
}

func (g *calibrationMemoryGauge) MeterMemory(usage common.MemoryUsage) error {
	g.meter[usage.Kind] += usage.Amount
	return nil
}

func newTestAuthAccountValue(
	inter *interpreter.Interpreter,
	addressValue interpreter.AddressValue,
) interpreter.Value {
	var returnZeroUInt64 = func(interp *interpreter.Interpreter) interpreter.UInt64Value {
		return interpreter.NewUInt64Value(interp, func() uint64 { return 0 })
	}

	var returnZeroUFix64 = func() interpreter.UFix64Value {
		return interpreter.UFix64Value(0)
	}

	panicFunction := interpreter.NewHostFunctionValue(
		inter,
		func(invocation interpreter.Invocation) interpreter.Value {
			panic(errors.NewUnreachableError())
		},
		stdlib.PanicFunction.Type,
	)

	return interpreter.NewAuthAccountValue(
		inter,
		addressValue,
		returnZeroUFix64,
		returnZeroUFix64,
		returnZeroUInt64,
		returnZeroUInt64,
		panicFunction,
		panicFunction,
		func() interpreter.Value {
			return interpreter.NewAuthAccountContractsValue(
				inter,
				addressValue,
				panicFunction,
				panicFunction,
				panicFunction,
				panicFunction,
				func(inter *interpreter.Interpreter) *interpreter.ArrayValue {
					return interpreter.NewArrayValue(
						inter,
						interpreter.VariableSizedStaticType{
							Type: interpreter.PrimitiveStaticTypeString,
						},
						common.Address{},
					)
				},
			)
		},
		func() interpreter.Value {
			return interpreter.NewAuthAccountKeysValue(
				inter,
				addressValue,
				panicFunction,
				panicFunction,
				panicFunction,
			)
		},
	)
}

func main() {
	unused_mem_kinds := make(map[common.MemoryKind]struct{}, len(memory_kinds))
	for _, kind := range memory_kinds {
		unused_mem_kinds[kind] = struct{}{}
	}
	abstract_measurements := make([]map[common.MemoryKind]uint64, 0, len(test_programs)-1)
	concrete_measurements := make([]float64, 0, len(test_programs)-1)
	empty_abstract_measurement := make(map[common.MemoryKind]uint64)
	var empty_concreate_measurement float64
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	totalAlloc := m.TotalAlloc
	for _, code := range test_programs {
		measurements := make(map[common.MemoryKind]uint64)
		fmt.Println(code.name)
		memoryGauge := newTestMemoryGauge()
		program, err := parser2.ParseProgram(code.code, memoryGauge)
		if err != nil {
			panic(err)
		}
		checker, err := sema.NewChecker(
			program,
			utils.TestLocation,
			sema.WithAccessCheckMode(sema.AccessCheckModeNotSpecifiedUnrestricted),
		)
		if err != nil {
			panic(err)
		}
		err = checker.Check()
		if err != nil {
			panic(err)
		}
		var uuid uint64 = 0

		inter, err := interpreter.NewInterpreter(
			interpreter.ProgramFromChecker(checker),
			checker.Location,
			interpreter.WithUUIDHandler(func() (uint64, error) {
				uuid++
				return uuid, nil
			}),
			interpreter.WithStorage(interpreter.NewInMemoryStorage(nil)),
			interpreter.WithAtreeValueValidationEnabled(true),
			interpreter.WithAtreeStorageValidationEnabled(true),
			interpreter.WithMemoryGauge(memoryGauge),
		)
		if err != nil {
			panic(err)
		}
		err = inter.Interpret()
		if err != nil {
			panic(err)
		}
		account := newTestAuthAccountValue(inter, interpreter.AddressValue{})
		_, err = inter.Invoke("main", account)
		if err != nil {
			panic(err)
		}

		for _, kind := range memory_kinds {
			if amount, ok := memoryGauge.meter[kind]; ok {
				delete(unused_mem_kinds, kind)
				measurements[kind] = amount
				fmt.Printf("%s: %d\n", kind.String(), amount)
			}
		}
		runtime.ReadMemStats(&m)
		allocs := m.TotalAlloc - totalAlloc
		fmt.Printf("Actual Memory: %d\n", allocs)
		fmt.Println("--------------------")
		if code.name != "empty" {
			abstract_measurements = append(abstract_measurements, measurements)
			concrete_measurements = append(concrete_measurements, float64(allocs))
		} else {
			empty_abstract_measurement = measurements
			empty_concreate_measurement = float64(allocs)
		}
		totalAlloc = m.TotalAlloc

	}
	for kind := range unused_mem_kinds { //nolint:maprangecheck
		fmt.Printf("Unusued memory kind: %s\n", kind.String())
	}

	// condition the data by subtracting out the empty program
	for i, measurements := range abstract_measurements {
		for _, kind := range memory_kinds {
			measurements[kind] = measurements[kind] - empty_abstract_measurement[kind]
			abstract_measurements[i] = measurements
		}
	}
	for i, measurement := range concrete_measurements {
		concrete_measurements[i] = measurement - empty_concreate_measurement
	}

	// to decide values for the weights, we have some linear equation A*x=b
	// A here is a matrix holding the abstracted measured values, x is the
	// vector of weights, and b is the vector of measured allocations
	v := make([]float64, 0, len(abstract_measurements)*(len(memory_kinds)+1))
	for _, measurements := range abstract_measurements {
		for _, kind := range memory_kinds {
			measure, ok := measurements[kind]
			if !ok {
				v = append(v, 0)
			} else {
				v = append(v, float64(measure))
			}
		}
		// weight for overhead constant
		v = append(v, 1)
	}

	// we have 1 more column than there are memory kinds, since the final columm will
	// be used to represent the overhead constant, which we will give an abstract allocation
	// amount of 1
	a := mat.NewDense(len(abstract_measurements), len(memory_kinds)+1, v)
	b := mat.NewVecDense(len(concrete_measurements), concrete_measurements)
	x := mat.NewVecDense(len(memory_kinds)+1, nil)
	err := x.SolveVec(a, b)

	if err != nil {
		panic(err)
	}

	weights := x.RawVector().Data
	for i, kind := range memory_kinds {
		fmt.Printf("Weight for %s: %f\n", kind.String(), weights[i])
	}
	fmt.Printf("Weight for constant factor: %f\n", weights[len(memory_kinds)])
}
