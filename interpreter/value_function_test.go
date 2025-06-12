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

	"github.com/stretchr/testify/assert"

	"github.com/onflow/cadence/common"
	. "github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
	. "github.com/onflow/cadence/test_utils/common_utils"
)

func TestFunctionStaticType(t *testing.T) {

	t.Parallel()

	t.Run("HostFunctionValue", func(t *testing.T) {
		t.Parallel()

		inter := newTestInterpreter(t)

		hostFunction := func(_ Invocation) Value {
			return TrueValue
		}

		hostFunctionType := sema.NewSimpleFunctionType(
			sema.FunctionPurityImpure,
			nil,
			sema.BoolTypeAnnotation,
		)

		hostFunctionValue := NewStaticHostFunctionValue(
			inter,
			hostFunctionType,
			hostFunction,
		)

		staticType := hostFunctionValue.StaticType(inter)

		assert.Equal(t, ConvertSemaToStaticType(inter, hostFunctionType), staticType)
	})

	t.Run("BoundFunctionValue", func(t *testing.T) {
		t.Parallel()

		inter := newTestInterpreter(t)

		inter.SharedState.Config.CompositeTypeHandler = func(location common.Location, typeID TypeID) *sema.CompositeType {
			return &sema.CompositeType{
				Location:   TestLocation,
				Identifier: "foo",
				Kind:       common.CompositeKindStructure,
			}
		}

		hostFunction := func(_ Invocation) Value {
			return TrueValue
		}

		hostFunctionType := sema.NewSimpleFunctionType(
			sema.FunctionPurityImpure,
			nil,
			sema.BoolTypeAnnotation,
		)

		hostFunctionValue := NewStaticHostFunctionValue(
			inter,
			hostFunctionType,
			hostFunction,
		)

		var compositeValue Value = NewCompositeValue(
			inter,
			EmptyLocationRange,
			TestLocation,
			"foo",
			common.CompositeKindStructure,
			[]CompositeField{},
			common.MustBytesToAddress([]byte{0}),
		)

		boundFunctionValue := NewBoundFunctionValue(
			inter,
			hostFunctionValue,
			&compositeValue,
			nil,
		)

		staticType := boundFunctionValue.StaticType(inter)

		assert.Equal(t, hostFunctionValue.StaticType(inter), staticType)
	})
}
