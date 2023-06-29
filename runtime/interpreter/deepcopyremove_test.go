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

package interpreter_test

import (
	"strings"
	"testing"

	"github.com/onflow/atree"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/common"
	. "github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/tests/utils"
)

func TestValueDeepCopyAndDeepRemove(t *testing.T) {

	t.Parallel()

	address := common.Address{0x1}

	storage := newUnmeteredInMemoryStorage()

	inter, err := NewInterpreter(
		nil,
		utils.TestLocation,
		&Config{
			Storage: storage,
		},
	)
	require.NoError(t, err)

	dictionaryStaticType := DictionaryStaticType{
		KeyType:   PrimitiveStaticTypeString,
		ValueType: PrimitiveStaticTypeInt256,
	}

	dictValueKey := NewUnmeteredStringValue(
		strings.Repeat("x", int(atree.MaxInlineMapKeySize()+1)),
	)

	dictValueValue := NewUnmeteredInt256ValueFromInt64(1)
	dictValue := NewDictionaryValue(
		inter,
		EmptyLocationRange,
		dictionaryStaticType,
		dictValueKey, dictValueValue,
	)

	arrayValue := NewArrayValue(
		inter,
		EmptyLocationRange,
		VariableSizedStaticType{
			Type: dictionaryStaticType,
		},
		common.ZeroAddress,
		dictValue,
	)

	optionalValue := NewUnmeteredSomeValueNonCopying(arrayValue)

	compositeValue := newTestCompositeValue(inter, address)

	compositeValue.SetMember(
		inter,
		EmptyLocationRange,
		"value",
		optionalValue,
	)

	compositeValue.DeepRemove(inter)

	// Only count non-temporary slabs,
	// i.e. ones which have a non-empty address

	count := 0
	for id := range storage.Slabs {
		if id.Address != (atree.Address{}) {
			count++
		}
	}

	require.Equal(t, 1, count)
}

func newUnmeteredInMemoryStorage() InMemoryStorage {
	return NewInMemoryStorage(nil)
}
