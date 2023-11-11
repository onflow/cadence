/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Dapper Labs, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package interpreter

import (
	"math"
	"runtime"
	"sync"
	"testing"

	"github.com/onflow/cadence/runtime/sema"
)

const numGoRoutines = 100

func runBench(b *testing.B, getValue func(value int8, staticType StaticType) IntegerValue) {
	semaToStaticType := make(map[sema.Type]StaticType)
	for _, integerType := range sema.AllIntegerTypes {
		switch integerType {
		case sema.IntegerType, sema.SignedIntegerType:
			continue
		}

		integerStaticType := ConvertSemaToStaticType(nil, integerType)
		semaToStaticType[integerType] = integerStaticType
	}

	var wg sync.WaitGroup

	bench := func() {
		defer wg.Done()

		for i := 0; i <= math.MaxInt8; i++ {
			for _, integerType := range sema.AllIntegerTypes {
				switch integerType {
				case sema.IntegerType, sema.SignedIntegerType:
					continue
				}

				value := getValue(int8(i), semaToStaticType[integerType])
				runtime.KeepAlive(value)
			}
		}
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		wg.Add(numGoRoutines)

		for routineIndex := 0; routineIndex < numGoRoutines; routineIndex++ {
			go bench()
		}

		wg.Wait()
	}
}

func BenchmarkSmallIntegerValueCache(b *testing.B) {
	runBench(b, GetSmallIntegerValue)
}

func BenchmarkIntegerCreationWithoutCache(b *testing.B) {
	runBench(b, createNewSmallIntegerValue)
}
