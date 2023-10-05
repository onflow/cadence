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

package stdlib

import (
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
)

//go:generate go run ../sema/gen -p stdlib webassembly.cdc webassembly.gen.go

func newWebAssemblyCompileAndInstantiateFunction(
	gauge common.MemoryGauge,
	handler WebAssemblyContractHandler,
) *interpreter.HostFunctionValue {
	return interpreter.NewHostFunctionValue(
		gauge,
		WebAssemblyTypeCompileAndInstantiateFunctionType,
		func(invocation interpreter.Invocation) interpreter.Value {
			inter := invocation.Interpreter
			locationRange := invocation.LocationRange

			bytesValue := invocation.Arguments[0]
			bytes, err := interpreter.ByteArrayValueToByteSlice(inter, bytesValue, locationRange)
			if err != nil {
				panic(err)
			}

			module, err := handler.CompileWebAssembly(bytes)
			if err != nil {
				panic(err)
			}

			instance, err := module.InstantiateWebAssemblyModule(gauge)
			if err != nil {
				panic(err)
			}

			instanceValue := NewWebAssemblyInstanceValue(gauge, instance)
			instanceReferenceValue := interpreter.NewEphemeralReferenceValue(
				gauge,
				interpreter.UnauthorizedAccess,
				instanceValue,
				WebAssembly_InstanceType,
			)

			instantiatedSourceValue := NewWebAssemblyInstantiatedSourceValue(gauge, instanceReferenceValue)
			return interpreter.NewEphemeralReferenceValue(
				gauge,
				interpreter.UnauthorizedAccess,
				instantiatedSourceValue,
				WebAssembly_InstantiatedSourceType,
			)
		},
	)
}

func newWebAssemblyInstanceGetExportFunction(
	gauge common.MemoryGauge,
	instance WebAssemblyInstance,
) *interpreter.HostFunctionValue {
	return interpreter.NewHostFunctionValue(
		gauge,
		WebAssembly_InstanceTypeGetExportFunctionType,
		func(invocation interpreter.Invocation) interpreter.Value {
			inter := invocation.Interpreter
			locationRange := invocation.LocationRange

			// Name
			nameValue, ok := invocation.Arguments[0].(*interpreter.StringValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}
			name := nameValue.Str

			// Type
			typeParameterPair := invocation.TypeParameterTypes.Oldest()
			if typeParameterPair == nil {
				panic(errors.NewUnreachableError())
			}
			ty := typeParameterPair.Value

			// Get and check export
			export, err := instance.GetExport(inter, name)
			if err != nil {
				panic(err)
			}
			if export == nil {
				panic(errors.NewDefaultUserError(
					"WebAssembly module does not have an export with name '%s'",
					name,
				))
			}

			if !sema.IsSubType(export.Type, ty) {
				panic(interpreter.TypeMismatchError{
					ExpectedType:  ty,
					ActualType:    export.Type,
					LocationRange: locationRange,
				})
			}

			return export.Value
		},
	)
}

var webAssembly_InstanceStaticType = interpreter.ConvertSemaCompositeTypeToStaticCompositeType(
	nil,
	WebAssembly_InstanceType,
)

func NewWebAssemblyInstanceValue(
	gauge common.MemoryGauge,
	instance WebAssemblyInstance,
) *interpreter.SimpleCompositeValue {
	return interpreter.NewSimpleCompositeValue(
		gauge,
		WebAssembly_InstanceType.ID(),
		webAssembly_InstanceStaticType,
		WebAssembly_InstanceType.Fields,
		map[string]interpreter.Value{
			WebAssembly_InstanceTypeGetExportFunctionName: newWebAssemblyInstanceGetExportFunction(gauge, instance),
		},
		nil,
		nil,
		nil,
	)
}

var webAssembly_InstantiatedSourceStaticType = interpreter.ConvertSemaCompositeTypeToStaticCompositeType(
	nil,
	WebAssembly_InstantiatedSourceType,
)

func NewWebAssemblyInstantiatedSourceValue(
	gauge common.MemoryGauge,
	instanceValue interpreter.Value,
) *interpreter.SimpleCompositeValue {
	return interpreter.NewSimpleCompositeValue(
		gauge,
		WebAssembly_InstantiatedSourceType.ID(),
		webAssembly_InstantiatedSourceStaticType,
		WebAssembly_InstantiatedSourceType.Fields,
		map[string]interpreter.Value{
			WebAssembly_InstantiatedSourceTypeInstanceFieldName: instanceValue,
		},
		nil,
		nil,
		nil,
	)
}

type WebAssemblyModule interface {
	InstantiateWebAssemblyModule(gauge common.MemoryGauge) (WebAssemblyInstance, error)
}

type WebAssemblyInstance interface {
	GetExport(gauge common.MemoryGauge, name string) (*WebAssemblyExport, error)
}

type WebAssemblyExport struct {
	Type  sema.Type
	Value interpreter.Value
}

type WebAssemblyContractHandler interface {
	CompileWebAssembly(bytes []byte) (WebAssemblyModule, error)
}

var WebAssemblyTypeStaticType = interpreter.ConvertSemaToStaticType(nil, WebAssemblyType)

func NewWebAssemblyContract(
	gauge common.MemoryGauge,
	handler WebAssemblyContractHandler,
) StandardLibraryValue {
	webAssemblyContractFields := map[string]interpreter.Value{
		WebAssemblyTypeCompileAndInstantiateFunctionName: newWebAssemblyCompileAndInstantiateFunction(gauge, handler),
	}

	webAssemblyContractValue := interpreter.NewSimpleCompositeValue(
		gauge,
		WebAssemblyType.ID(),
		WebAssemblyTypeStaticType,
		nil,
		webAssemblyContractFields,
		nil,
		nil,
		nil,
	)

	return StandardLibraryValue{
		Name:  WebAssemblyTypeName,
		Type:  WebAssemblyType,
		Value: webAssemblyContractValue,
		Kind:  common.DeclarationKindContract,
	}
}

var WebAssemblyContractType = StandardLibraryType{
	Name: WebAssemblyTypeName,
	Type: WebAssemblyType,
	Kind: common.DeclarationKindContract,
}
