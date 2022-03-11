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
		fmt.Println("invoked")
		if err != nil {
			panic(err)
		}

		for kind, amount := range memoryGauge.meter {
			fmt.Printf("%s: %d\n", kind.String(), amount)
		}
		runtime.ReadMemStats(&m)
		fmt.Printf("Actual Memory: %d\n", m.TotalAlloc-totalAlloc)
		fmt.Println("--------------------")
		totalAlloc = m.TotalAlloc
	}
}
