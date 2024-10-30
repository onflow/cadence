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
	"strings"
	"testing"

	"github.com/onflow/atree"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/interpreter"
	. "github.com/onflow/cadence/test_utils/common_utils"
)

func TestValueDeepCopyAndDeepRemove(t *testing.T) {

	t.Parallel()

	address := common.Address{0x1}

	storage := newUnmeteredInMemoryStorage()

	inter, err := interpreter.NewInterpreter(
		nil,
		TestLocation,
		&interpreter.Config{
			Storage: storage,
		},
	)
	require.NoError(t, err)

	dictionaryStaticType := &interpreter.DictionaryStaticType{
		KeyType:   interpreter.PrimitiveStaticTypeString,
		ValueType: interpreter.PrimitiveStaticTypeInt256,
	}

	dictValueKey := interpreter.NewUnmeteredStringValue(
		strings.Repeat("x", int(atree.MaxInlineMapKeySize()+1)),
	)

	dictValueValue := interpreter.NewUnmeteredInt256ValueFromInt64(1)
	dictValue := interpreter.NewDictionaryValue(
		inter,
		interpreter.EmptyLocationRange,
		dictionaryStaticType,
		dictValueKey, dictValueValue,
	)

	arrayValue := interpreter.NewArrayValue(
		inter,
		interpreter.EmptyLocationRange,
		&interpreter.VariableSizedStaticType{
			Type: dictionaryStaticType,
		},
		common.ZeroAddress,
		dictValue,
	)

	optionalValue := interpreter.NewUnmeteredSomeValueNonCopying(arrayValue)

	compositeValue := newTestCompositeValue(inter, address)

	compositeValue.SetMember(
		inter,
		interpreter.EmptyLocationRange,
		"value",
		optionalValue,
	)

	compositeValue.DeepRemove(inter, true)

	// Only count non-temporary slabs,
	// i.e. ones which have a non-empty address

	count := 0
	for id := range storage.Slabs {
		if !id.HasTempAddress() {
			count++
		}
	}

	require.Equal(t, 1, count)
}
