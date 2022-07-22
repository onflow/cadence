/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2022 Dapper Labs, Inc.
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

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/tests/utils"
)

func setupInterpreterWithTracingCallBack(
	t *testing.T,
	tracingCallback func(opName string),
) *interpreter.Interpreter {
	storage := newUnmeteredInMemoryStorage()
	inter, err := interpreter.NewInterpreter(
		&interpreter.Program{},
		utils.TestLocation,
		interpreter.WithOnRecordTraceHandler(
			func(inter *interpreter.Interpreter,
				operationName string,
				duration time.Duration,
				attrs []attribute.KeyValue) {
				tracingCallback(operationName)
			},
		),
		interpreter.WithStorage(storage),
		interpreter.WithTracingEnabled(true),
	)
	require.NoError(t, err)
	return inter
}

func TestInterpreterTracing(t *testing.T) {

	t.Parallel()

	t.Run("array tracing", func(t *testing.T) {
		traceOps := make([]string, 0)
		inter := setupInterpreterWithTracingCallBack(t, func(opName string) {
			traceOps = append(traceOps, opName)
		})
		owner := common.Address{0x1}
		array := interpreter.NewArrayValue(
			inter,
			interpreter.ReturnEmptyLocationRange,
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
		cloned.DeepRemove(inter)
		require.Equal(t, len(traceOps), 2)
		require.Equal(t, traceOps[1], "array.deepRemove")

		array.Destroy(inter, nil)
		require.Equal(t, len(traceOps), 3)
		require.Equal(t, traceOps[2], "array.destroy")
	})

	t.Run("dictionary tracing", func(t *testing.T) {
		traceOps := make([]string, 0)
		inter := setupInterpreterWithTracingCallBack(t, func(opName string) {
			traceOps = append(traceOps, opName)
		})
		dict := interpreter.NewDictionaryValue(
			inter,
			interpreter.ReturnEmptyLocationRange,
			interpreter.DictionaryStaticType{
				KeyType:   interpreter.PrimitiveStaticTypeString,
				ValueType: interpreter.PrimitiveStaticTypeInt,
			},
			interpreter.NewUnmeteredStringValue("test"), interpreter.NewUnmeteredIntValueFromInt64(42),
		)
		require.NotNil(t, dict)
		fmt.Println(traceOps)
		require.Equal(t, len(traceOps), 1)
		require.Equal(t, traceOps[0], "dictionary.construct")

		cloned := dict.Clone(inter)
		require.NotNil(t, cloned)
		cloned.DeepRemove(inter)
		require.Equal(t, len(traceOps), 2)
		require.Equal(t, traceOps[1], "dictionary.deepRemove")

		dict.Destroy(inter, nil)
		require.Equal(t, len(traceOps), 3)
		require.Equal(t, traceOps[2], "dictionary.destroy")
	})

	t.Run("composite tracing", func(t *testing.T) {
		traceOps := make([]string, 0)
		inter := setupInterpreterWithTracingCallBack(t, func(opName string) {
			traceOps = append(traceOps, opName)
		})
		owner := common.Address{0x1}

		value := newTestCompositeValue(inter, owner)

		require.Equal(t, len(traceOps), 1)
		require.Equal(t, traceOps[0], "composite.construct")

		cloned := value.Clone(inter)
		require.NotNil(t, cloned)
		cloned.DeepRemove(inter)
		require.Equal(t, len(traceOps), 2)
		require.Equal(t, traceOps[1], "composite.deepRemove")

		value.SetMember(inter, nil, "abc", interpreter.NilValue{})
		require.Equal(t, len(traceOps), 3)
		require.Equal(t, traceOps[2], "composite.setMember.abc")

		value.GetMember(inter, nil, "abc")
		require.Equal(t, len(traceOps), 4)
		require.Equal(t, traceOps[3], "composite.getMember.abc")

		value.RemoveMember(inter, nil, "abc")
		require.Equal(t, len(traceOps), 5)
		require.Equal(t, traceOps[4], "composite.removeMember.abc")

		value.Destroy(inter, nil)
		require.Equal(t, len(traceOps), 6)
		require.Equal(t, traceOps[5], "composite.destroy")

		array := interpreter.NewArrayValue(
			inter,
			interpreter.ReturnEmptyLocationRange,
			interpreter.VariableSizedStaticType{
				Type: interpreter.PrimitiveStaticTypeAnyStruct,
			},
			common.Address{},
			cloned,
		)
		require.NotNil(t, array)
		require.Equal(t, len(traceOps), 8)
		require.Equal(t, traceOps[6], "composite.transfer")
		require.Equal(t, traceOps[7], "array.construct")
	})
}
