/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2022 Dapper Labs, Inc.
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

package interpreter_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/opentracing/opentracing-go"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/tests/utils"
)

func TestInterpreterTracing(t *testing.T) {

	t.Parallel()

	t.Run("array tracing", func(t *testing.T) {
		storage := interpreter.NewInMemoryStorage()

		traceOps := make([]string, 0)
		inter, err := interpreter.NewInterpreter(
			&interpreter.Program{},
			utils.TestLocation,
			interpreter.WithOnRecordTraceHandler(
				func(inter *interpreter.Interpreter,
					operationName string,
					duration time.Duration,
					logs []opentracing.LogRecord) {
					traceOps = append(traceOps, operationName)
				},
			),
			interpreter.WithStorage(storage),
			interpreter.WithTracingEnabled(true),
		)
		require.NoError(t, err)

		owner := common.Address{0x1}
		array := interpreter.NewArrayValue(
			inter,
			interpreter.VariableSizedStaticType{
				Type: interpreter.PrimitiveStaticTypeAnyStruct,
			},
			owner,
		)
		require.NotNil(t, array)
		fmt.Println(traceOps)
		require.Equal(t, len(traceOps), 1)
		require.Equal(t, traceOps[0], "array.construct")

		cloned := array.Clone(inter)
		require.NotNil(t, cloned)
		require.Equal(t, len(traceOps), 2)
		require.Equal(t, traceOps[1], "array.clone")

		cloned.DeepRemove(inter)
		require.Equal(t, len(traceOps), 3)
		require.Equal(t, traceOps[2], "array.deepRemove")

		array.Destroy(inter, nil)
		require.Equal(t, len(traceOps), 4)
		require.Equal(t, traceOps[3], "array.destroy")
	})

	t.Run("composite tracing", func(t *testing.T) {
		storage := interpreter.NewInMemoryStorage()

		elaboration := sema.NewElaboration()
		elaboration.CompositeTypes[testCompositeValueType.ID()] = testCompositeValueType

		traceOps := make([]string, 0)
		inter, err := interpreter.NewInterpreter(
			&interpreter.Program{
				Elaboration: elaboration,
			},
			utils.TestLocation,
			interpreter.WithOnRecordTraceHandler(
				func(inter *interpreter.Interpreter,
					operationName string,
					duration time.Duration,
					logs []opentracing.LogRecord) {
					traceOps = append(traceOps, operationName)
				},
			),
			interpreter.WithStorage(storage),
			interpreter.WithTracingEnabled(true),
		)
		require.NoError(t, err)

		owner := common.Address{0x1}

		value := newTestCompositeValue(inter, owner)

		require.Equal(t, len(traceOps), 1)
		require.Equal(t, traceOps[0], "composite.construct")

		cloned := value.Clone(inter)
		require.NotNil(t, cloned)
		require.Equal(t, len(traceOps), 2)
		require.Equal(t, traceOps[1], "composite.clone")

		cloned.DeepRemove(inter)
		require.Equal(t, len(traceOps), 3)
		require.Equal(t, traceOps[2], "composite.deepRemove")

		value.SetMember(inter, nil, "abc", interpreter.NilValue{})
		require.Equal(t, len(traceOps), 4)
		require.Equal(t, traceOps[3], "composite.setMember.abc")

		value.GetMember(inter, nil, "abc")
		require.Equal(t, len(traceOps), 5)
		require.Equal(t, traceOps[4], "composite.getMember.abc")

		value.RemoveMember(inter, nil, "abc")
		require.Equal(t, len(traceOps), 6)
		require.Equal(t, traceOps[5], "composite.removeMember.abc")

		value.Destroy(inter, nil)
		require.Equal(t, len(traceOps), 7)
		require.Equal(t, traceOps[6], "composite.destroy")

		array := interpreter.NewArrayValue(
			inter,
			interpreter.VariableSizedStaticType{
				Type: interpreter.PrimitiveStaticTypeAnyStruct,
			},
			common.Address{},
			cloned,
		)
		require.NotNil(t, array)
		require.Equal(t, len(traceOps), 9)
		require.Equal(t, traceOps[7], "composite.transfer")
		require.Equal(t, traceOps[8], "array.construct")
	})
}
