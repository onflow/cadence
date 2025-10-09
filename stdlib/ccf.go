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

package stdlib

//go:generate go run ../sema/gen -p stdlib ccf.cdc ccf.gen.go

import (
	"github.com/onflow/cadence"
	"github.com/onflow/cadence/bbq/vm"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/encoding/ccf"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/interpreter"
)

type Exporter interface {
	ExportValue(
		value interpreter.Value,
		context interpreter.ValueExportContext,
	) (
		cadence.Value,
		error,
	)
}

type CCFContractHandler interface {
	Exporter
}

func NativeCCFEncodeFunction(handler CCFContractHandler) interpreter.NativeFunction {
	return func(
		context interpreter.NativeFunctionContext,
		_ interpreter.TypeArgumentsIterator,
		_ interpreter.Value,
		args []interpreter.Value,
	) interpreter.Value {

		referenceValue, ok := args[0].(interpreter.ReferenceValue)
		if !ok {
			panic(errors.NewUnreachableError())
		}

		referencedValue := referenceValue.ReferencedValue(context, true)
		if referencedValue == nil {
			return interpreter.Nil
		}

		exportedValue, err := handler.ExportValue(*referencedValue, context)
		if err != nil {
			return interpreter.Nil
		}

		encoded, err := ccf.Encode(exportedValue)
		if err != nil {
			return interpreter.Nil
		}

		res := interpreter.ByteSliceToByteArrayValue(context, encoded)

		return interpreter.NewSomeValueNonCopying(context, res)
	}
}

func newInterpreterCCFEncodeFunction(
	gauge common.MemoryGauge,
	handler CCFContractHandler,
) *interpreter.HostFunctionValue {
	// TODO: Should create a bound-host function here, but interpreter is not available at this point.
	// However, this is not a problem for now, since underlying contract doesn't get moved.
	return interpreter.NewStaticHostFunctionValueFromNativeFunction(
		gauge,
		BLSTypeAggregatePublicKeysFunctionType,
		NativeCCFEncodeFunction(handler),
	)
}

func NewVMCCFEncodeFunction(handler CCFContractHandler) VMFunction {
	return VMFunction{
		BaseType: CCFType,
		FunctionValue: vm.NewNativeFunctionValue(
			CCFTypeEncodeFunctionName,
			CCFTypeEncodeFunctionType,
			NativeCCFEncodeFunction(handler),
		),
	}
}

var CCFTypeStaticType = interpreter.ConvertSemaToStaticType(nil, CCFType)

func NewCCFContract(
	gauge common.MemoryGauge,
	handler CCFContractHandler,
) StandardLibraryValue {

	ccfContractFields := map[string]interpreter.Value{
		CCFTypeEncodeFunctionName: newInterpreterCCFEncodeFunction(gauge, handler),
	}

	var ccfContractValue = interpreter.NewSimpleCompositeValue(
		gauge,
		CCFType.ID(),
		CCFTypeStaticType,
		nil,
		ccfContractFields,
		nil,
		nil,
		nil,
		nil,
	)

	return StandardLibraryValue{
		Name:  CCFTypeName,
		Type:  CCFType,
		Value: ccfContractValue,
		Kind:  common.DeclarationKindContract,
	}
}
