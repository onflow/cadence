//go:build cadence_tracing

/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Flow Foundation
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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"

	"github.com/onflow/cadence/bbq"
	"github.com/onflow/cadence/bbq/vm"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/test_utils"
	. "github.com/onflow/cadence/test_utils/common_utils"
)

func prepareWithTracingCallBack(
	t *testing.T,
	tracingCallback func(opName string),
) Invokable {
	storage := newUnmeteredInMemoryStorage()

	onRecordTrace := func(
		operationName string,
		_ time.Duration,
		_ []attribute.KeyValue,
	) {
		tracingCallback(operationName)
	}

	if *compile {
		config := vm.NewConfig(storage)
		config.Tracer = interpreter.CallbackTracer(onRecordTrace)
		config.CompositeTypeHandler = func(location common.Location, typeID interpreter.TypeID) *sema.CompositeType {
			if typeID == testCompositeValueType.ID() {
				return testCompositeValueType
			}
			t.Fatalf("unexpected type ID: %s", typeID)
			return nil
		}
		config.ImportHandler = func(_ common.Location) *bbq.InstructionProgram {
			return &bbq.InstructionProgram{}
		}
		vm := vm.NewVM(
			TestLocation,
			&bbq.InstructionProgram{},
			config,
		)
		return test_utils.NewVMInvokable(vm, nil)
	} else {
		inter, err := interpreter.NewInterpreter(
			nil,
			TestLocation,
			&interpreter.Config{
				Storage:       storage,
				OnRecordTrace: onRecordTrace,
			},
		)
		require.NoError(t, err)
		return inter
	}
}

func TestInterpreterTracing(t *testing.T) {

	t.Parallel()

	t.Run("array tracing", func(t *testing.T) {
		t.Parallel()

		var traceOps []string
		inter := prepareWithTracingCallBack(t, func(opName string) {
			traceOps = append(traceOps, opName)
		})
		owner := common.Address{0x1}
		array := interpreter.NewArrayValue(
			inter,
			&interpreter.VariableSizedStaticType{
				Type: interpreter.PrimitiveStaticTypeAnyStruct,
			},
			owner,
		)
		require.NotNil(t, array)
		require.Len(t, traceOps, 2)
		assert.Equal(t, "atreeArray.newFromBatchData", traceOps[0])
		assert.Equal(t, "array.construct", traceOps[1])

		cloned := array.Clone(inter)
		require.NotNil(t, cloned)

		cloned.DeepRemove(inter, true)
		require.Len(t, traceOps, 3)
		assert.Equal(t, "array.deepRemove", traceOps[2])

		array.Destroy(inter, interpreter.EmptyLocationRange)
		require.Len(t, traceOps, 4)
		assert.Equal(t, "array.destroy", traceOps[3])
	})

	t.Run("dictionary tracing", func(t *testing.T) {
		t.Parallel()

		var traceOps []string
		inter := prepareWithTracingCallBack(t, func(opName string) {
			traceOps = append(traceOps, opName)
		})
		dict := interpreter.NewDictionaryValue(
			inter,
			&interpreter.DictionaryStaticType{
				KeyType:   interpreter.PrimitiveStaticTypeString,
				ValueType: interpreter.PrimitiveStaticTypeInt,
			},
			interpreter.NewUnmeteredStringValue("test"), interpreter.NewUnmeteredIntValueFromInt64(42),
		)
		require.NotNil(t, dict)
		require.Len(t, traceOps, 2)
		assert.Equal(t, "atreeMap.new", traceOps[0])
		assert.Equal(t, "dictionary.construct", traceOps[1])

		cloned := dict.Clone(inter)
		require.NotNil(t, cloned)

		cloned.DeepRemove(inter, true)
		require.Len(t, traceOps, 3)
		assert.Equal(t, "dictionary.deepRemove", traceOps[2])

		dict.Destroy(inter, interpreter.EmptyLocationRange)
		require.Len(t, traceOps, 4)
		assert.Equal(t, "dictionary.destroy", traceOps[3])
	})

	t.Run("composite tracing", func(t *testing.T) {
		t.Parallel()

		var traceOps []string
		inter := prepareWithTracingCallBack(t, func(opName string) {
			traceOps = append(traceOps, opName)
		})
		owner := common.Address{0x1}

		value := newTestCompositeValue(inter, owner)

		require.Len(t, traceOps, 2)
		assert.Equal(t, "atreeMap.new", traceOps[0])
		assert.Equal(t, "composite.construct", traceOps[1])

		cloned := value.Clone(inter)
		require.NotNil(t, cloned)

		cloned.DeepRemove(inter, true)
		require.Len(t, traceOps, 3)
		assert.Equal(t, "composite.deepRemove", traceOps[2])

		value.SetMember(inter, interpreter.EmptyLocationRange, "abc", interpreter.Nil)
		require.Len(t, traceOps, 4)
		assert.Equal(t, "composite.setMember", traceOps[3])

		value.GetMember(inter, interpreter.EmptyLocationRange, "abc")
		require.Len(t, traceOps, 5)
		assert.Equal(t, "composite.getMember", traceOps[4])

		value.RemoveMember(inter, interpreter.EmptyLocationRange, "abc")
		require.Len(t, traceOps, 6)
		assert.Equal(t, "composite.removeMember", traceOps[5])

		value.Destroy(inter, interpreter.EmptyLocationRange)
		require.Len(t, traceOps, 7)
		assert.Equal(t, "composite.destroy", traceOps[6])

		array := interpreter.NewArrayValue(
			inter,
			&interpreter.VariableSizedStaticType{
				Type: interpreter.PrimitiveStaticTypeAnyStruct,
			},
			common.ZeroAddress,
			cloned,
		)
		require.NotNil(t, array)
		require.Len(t, traceOps, 11)
		assert.Equal(t, "atreeMap.newFromBatchData", traceOps[7])
		assert.Equal(t, "composite.transfer", traceOps[8])
		assert.Equal(t, "atreeArray.newFromBatchData", traceOps[9])
		assert.Equal(t, "array.construct", traceOps[10])
	})
}
