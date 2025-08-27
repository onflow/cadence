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

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"

	"github.com/onflow/cadence/bbq"
	"github.com/onflow/cadence/bbq/vm"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/test_utils"
	. "github.com/onflow/cadence/test_utils/common_utils"
	. "github.com/onflow/cadence/test_utils/interpreter_utils"
)

func prepareWithTracingCallBack(
	t *testing.T,
	tracingCallback func(opName string),
) Invokable {
	storage := NewUnmeteredInMemoryStorage()

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
		config.TypeLoader = func(location common.Location, typeID interpreter.TypeID) (sema.Type, error) {
			if typeID == testCompositeValueType.ID() {
				return testCompositeValueType, nil
			}
			t.Fatalf("unexpected type ID: %s", typeID)
			return nil, nil
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
				Storage:        storage,
				TracingEnabled: true,
				OnRecordTrace:  onRecordTrace,
			},
		)
		require.NoError(t, err)
		return inter
	}
}

func TestInterpreterTracing(t *testing.T) {

	t.Parallel()

	t.Run("array tracing", func(t *testing.T) {
		var traceOps []string
		inter := prepareWithTracingCallBack(t, func(opName string) {
			traceOps = append(traceOps, opName)
		})
		owner := common.Address{0x1}
		array := interpreter.NewArrayValue(
			inter,
			interpreter.EmptyLocationRange,
			&interpreter.VariableSizedStaticType{
				Type: interpreter.PrimitiveStaticTypeAnyStruct,
			},
			owner,
		)
		require.NotNil(t, array)
		require.Equal(t, len(traceOps), 1)
		require.Equal(t, traceOps[0], "array.construct")

		cloned := array.Clone(inter)
		require.NotNil(t, cloned)
		cloned.DeepRemove(inter, true)
		require.Equal(t, len(traceOps), 2)
		require.Equal(t, traceOps[1], "array.deepRemove")

		array.Destroy(inter, interpreter.EmptyLocationRange)
		require.Equal(t, len(traceOps), 3)
		require.Equal(t, traceOps[2], "array.destroy")
	})

	t.Run("dictionary tracing", func(t *testing.T) {
		var traceOps []string
		inter := prepareWithTracingCallBack(t, func(opName string) {
			traceOps = append(traceOps, opName)
		})
		dict := interpreter.NewDictionaryValue(
			inter,
			interpreter.EmptyLocationRange,
			&interpreter.DictionaryStaticType{
				KeyType:   interpreter.PrimitiveStaticTypeString,
				ValueType: interpreter.PrimitiveStaticTypeInt,
			},
			interpreter.NewUnmeteredStringValue("test"), interpreter.NewUnmeteredIntValueFromInt64(42),
		)
		require.NotNil(t, dict)
		require.Equal(t, len(traceOps), 1)
		require.Equal(t, traceOps[0], "dictionary.construct")

		cloned := dict.Clone(inter)
		require.NotNil(t, cloned)
		cloned.DeepRemove(inter, true)
		require.Equal(t, len(traceOps), 2)
		require.Equal(t, traceOps[1], "dictionary.deepRemove")

		dict.Destroy(inter, interpreter.EmptyLocationRange)
		require.Equal(t, len(traceOps), 3)
		require.Equal(t, traceOps[2], "dictionary.destroy")
	})

	t.Run("composite tracing", func(t *testing.T) {
		var traceOps []string
		inter := prepareWithTracingCallBack(t, func(opName string) {
			traceOps = append(traceOps, opName)
		})
		owner := common.Address{0x1}

		value := newTestCompositeValue(inter, owner)

		require.Equal(t, len(traceOps), 1)
		require.Equal(t, traceOps[0], "composite.construct")

		cloned := value.Clone(inter)
		require.NotNil(t, cloned)
		cloned.DeepRemove(inter, true)
		require.Equal(t, len(traceOps), 2)
		require.Equal(t, traceOps[1], "composite.deepRemove")

		value.SetMember(inter, interpreter.EmptyLocationRange, "abc", interpreter.Nil)
		require.Equal(t, len(traceOps), 3)
		require.Equal(t, traceOps[2], "composite.setMember.abc")

		value.GetMember(inter, interpreter.EmptyLocationRange, "abc")
		require.Equal(t, len(traceOps), 4)
		require.Equal(t, traceOps[3], "composite.getMember.abc")

		value.RemoveMember(inter, interpreter.EmptyLocationRange, "abc")
		require.Equal(t, len(traceOps), 5)
		require.Equal(t, traceOps[4], "composite.removeMember.abc")

		value.Destroy(inter, interpreter.EmptyLocationRange)
		require.Equal(t, len(traceOps), 6)
		require.Equal(t, traceOps[5], "composite.destroy")

		array := interpreter.NewArrayValue(
			inter,
			interpreter.EmptyLocationRange,
			&interpreter.VariableSizedStaticType{
				Type: interpreter.PrimitiveStaticTypeAnyStruct,
			},
			common.ZeroAddress,
			cloned,
		)
		require.NotNil(t, array)
		require.Equal(t, len(traceOps), 8)
		require.Equal(t, traceOps[6], "composite.transfer")
		require.Equal(t, traceOps[7], "array.construct")
	})
}
