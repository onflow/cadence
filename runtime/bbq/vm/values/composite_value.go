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

package values

import (
	"github.com/onflow/atree"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/interpreter"

	"github.com/onflow/cadence/runtime/bbq/vm/context"
	"github.com/onflow/cadence/runtime/bbq/vm/types"
)

type StructValue struct {
	dictionary          *atree.OrderedMap
	Location            common.Location
	QualifiedIdentifier string
	typeID              common.TypeID
	staticType          types.StaticType
}

var _ Value = &StructValue{}

func NewStructValue(
	location common.Location,
	qualifiedIdentifier string,
	address common.Address,
	storage atree.SlabStorage,
) *StructValue {

	const kind = common.CompositeKindStructure

	dictionary, err := atree.NewMap(
		storage,
		atree.Address(address),
		atree.NewDefaultDigesterBuilder(),
		interpreter.NewCompositeTypeInfo(
			nil,
			location,
			qualifiedIdentifier,
			kind,
		),
	)

	if err != nil {
		panic(errors.NewExternalError(err))
	}

	return &StructValue{
		QualifiedIdentifier: qualifiedIdentifier,
		Location:            location,
		dictionary:          dictionary,
	}
}

func (*StructValue) isValue() {}

func (v *StructValue) StaticType(memoryGauge common.MemoryGauge) types.StaticType {
	if v.staticType == nil {
		// NOTE: Instead of using NewCompositeStaticType, which always generates the type ID,
		// use the TypeID accessor, which may return an already computed type ID
		v.staticType = interpreter.NewCompositeStaticType(
			memoryGauge,
			v.Location,
			v.QualifiedIdentifier,
			v.TypeID(),
		)
	}
	return v.staticType
}

func (v *StructValue) GetMember(ctx context.Context, name string) Value {
	storable, err := v.dictionary.Get(
		interpreter.StringAtreeComparator,
		interpreter.StringAtreeHashInput,
		interpreter.StringAtreeValue(name),
	)
	if err != nil {
		if _, ok := err.(*atree.KeyNotFoundError); !ok {
			panic(errors.NewExternalError(err))
		}
	}

	if storable != nil {
		interpreterValue := interpreter.StoredValue(ctx.MemoryGauge, storable, ctx.Storage)
		// TODO: Temp conversion
		return InterpreterValueToVMValue(interpreterValue)
	}

	return nil
}

func (v *StructValue) SetMember(ctx context.Context, name string, value Value) {

	// TODO:
	//address := v.StorageID().Address
	//value = value.Transfer(
	//	interpreter,
	//	locationRange,
	//	address,
	//	true,
	//	nil,
	//)

	interpreterValue := VMValueToInterpreterValue(value)

	existingStorable, err := v.dictionary.Set(
		interpreter.StringAtreeComparator,
		interpreter.StringAtreeHashInput,
		interpreter.NewStringAtreeValue(ctx.MemoryGauge, name),
		interpreterValue,
	)

	if err != nil {
		panic(errors.NewExternalError(err))
	}

	if existingStorable != nil {
		// TODO:
		//existingValue := interpreter.StoredValue(nil, existingStorable, context.Storage)
		//existingValue.DeepRemove(interpreter)

		context.RemoveReferencedSlab(ctx.Storage, existingStorable)
	}
}

func (v *StructValue) StorageID() atree.StorageID {
	return v.dictionary.StorageID()
}

func (v *StructValue) TypeID() common.TypeID {
	if v.typeID == "" {
		location := v.Location
		qualifiedIdentifier := v.QualifiedIdentifier
		if location == nil {
			return common.TypeID(qualifiedIdentifier)
		}

		// TODO: TypeID metering
		v.typeID = location.TypeID(nil, qualifiedIdentifier)
	}
	return v.typeID
}
