/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2021 Dapper Labs, Inc.
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

	"github.com/fxamacker/atree"
	. "github.com/onflow/cadence/runtime/interpreter"
	"github.com/stretchr/testify/require"
)

func TestValueDeepCopyAndDeepRemove(t *testing.T) {

	t.Parallel()

	address := atree.Address{0x1}

	storage := NewInMemoryStorage()

	dictionaryStaticType := DictionaryStaticType{
		KeyType:   PrimitiveStaticTypeString,
		ValueType: PrimitiveStaticTypeInt256,
	}

	dictValueKey := NewStringValue(strings.Repeat("x", int(atree.MaxInlineElementSize+1)))

	dictValueValue := NewInt256ValueFromInt64(1)
	dictValue := NewDictionaryValue(
		dictionaryStaticType,
		storage,
		dictValueKey, dictValueValue,
	)

	arrayValue := NewArrayValue(
		VariableSizedStaticType{
			Type: dictionaryStaticType,
		},
		storage,
		dictValue,
	)

	optionalValue := NewSomeValueNonCopying(arrayValue)

	compositeValue := newTestCompositeValue(storage, address)
	compositeValue.Fields.Set("value", optionalValue)

	err := compositeValue.DeepRemove(storage)
	require.NoError(t, err)

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
