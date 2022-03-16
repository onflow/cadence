package main

import (
	"fmt"
	"runtime"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/parser2"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/tests/utils"

	"gonum.org/v1/gonum/mat"
)

var memory_kinds = []common.MemoryKind{
	common.MemoryKindUnknown,
	common.MemoryKindBool,
	common.MemoryKindAddress,
	common.MemoryKindString,
	common.MemoryKindCharacter,
	common.MemoryKindMetaType,
	common.MemoryKindBlock,
	common.MemoryKindNumber,
	common.MemoryKindArray,
	common.MemoryKindDictionary,
	common.MemoryKindComposite,
	common.MemoryKindOptional,
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

func (g *calibrationMemoryGauge) UseMemory(usage common.MemoryUsage) {
	g.meter[usage.Kind] += usage.Amount
}

func main() {
	unused_mem_kinds := make(map[common.MemoryKind]struct{}, len(memory_kinds))
	for _, kind := range memory_kinds {
		unused_mem_kinds[kind] = struct{}{}
	}
	abstract_measurements := make([]map[common.MemoryKind]uint64, 0, len(test_programs))
	concrete_measurements := make([]float64, 0, len(test_programs))
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
		inter, err := interpreter.NewInterpreter(
			interpreter.ProgramFromChecker(checker),
			checker.Location,
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
		_, err = inter.Invoke("main")
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
		abstract_measurements = append(abstract_measurements, measurements)
		concrete_measurements = append(concrete_measurements, float64(allocs))
		totalAlloc = m.TotalAlloc

	}
	for kind := range unused_mem_kinds { //nolint:maprangecheck
		fmt.Printf("Unusued memory kind: %s\n", kind.String())
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
	x.SolveVec(a, b)

	weights := x.RawVector().Data
	for i, kind := range memory_kinds {
		fmt.Printf("Weight for %s: %f\n", kind.String(), weights[i])
	}
	fmt.Printf("Weight for constant factor: %f\n", weights[len(memory_kinds)])
}
