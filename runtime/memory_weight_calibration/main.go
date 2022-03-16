package main

import (
	"fmt"
	"runtime"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/parser2"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/tests/utils"
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
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	totalAlloc := m.TotalAlloc
	for _, code := range test_programs {
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
				fmt.Printf("%s: %d\n", kind.String(), amount)
			}
		}
		runtime.ReadMemStats(&m)
		fmt.Printf("Actual Memory: %d\n", m.TotalAlloc-totalAlloc)
		fmt.Println("--------------------")
		totalAlloc = m.TotalAlloc
	}
	for kind := range unused_mem_kinds { //nolint:maprangecheck
		fmt.Printf("Unusued memory kind: %s\n", kind.String())
	}
}
