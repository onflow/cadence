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

package interpreter

import (
	"time"

	"github.com/onflow/atree"

	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/sema"
)

type TypeConverter interface {
	ConvertStaticToSemaType(staticType StaticType) (sema.Type, error)
}

var _ TypeConverter = &Interpreter{}

func MustConvertStaticToSemaType(staticType StaticType, typeConverter TypeConverter) sema.Type {
	semaType, err := typeConverter.ConvertStaticToSemaType(staticType)
	if err != nil {
		panic(err)
	}
	return semaType
}

func MustSemaTypeOfValue(value Value, context ValueStaticTypeContext) sema.Type {
	staticType := value.StaticType(context)
	return MustConvertStaticToSemaType(staticType, context)
}

type SubTypeChecker interface {
	IsSubType(subType StaticType, superType StaticType) bool
	IsSubTypeOfSemaType(staticSubType StaticType, superType sema.Type) bool
}

var _ SubTypeChecker = &Interpreter{}

type StorageReader interface {
	ReadStored(
		storageAddress common.Address,
		domain common.StorageDomain,
		identifier StorageMapKey,
	) Value
}

type StorageWriter interface {
	WriteStored(
		storageAddress common.Address,
		domain common.StorageDomain,
		key StorageMapKey,
		value Value,
	) (existed bool)
}

var _ StorageReader = &Interpreter{}

type ValueStaticTypeContext interface {
	common.MemoryGauge
	StorageReader
	TypeConverter
	SubTypeChecker
}

var _ ValueStaticTypeContext = &Interpreter{}

type StorageContext interface {
	ValueStaticTypeContext
	common.MemoryGauge

	StorageReader
	StorageWriter

	Storage() Storage
	RemoveReferencedSlab(storable atree.Storable)
	MaybeValidateAtreeValue(v atree.Value)
	MaybeValidateAtreeStorage()
}

type ReferenceTracker interface {
	InvalidateReferencedResources(v Value, locationRange LocationRange)
	CheckInvalidatedResourceOrResourceReference(value Value, locationRange LocationRange)
	MaybeTrackReferencedResourceKindedValue(ref *EphemeralReferenceValue)
}

type ValueTransferContext interface {
	StorageContext
	ReferenceTracker
	ComputationReporter
	Tracer

	OnResourceOwnerChange(
		resource *CompositeValue,
		oldOwner common.Address,
		newOwner common.Address,
	)
}

var _ ValueTransferContext = &Interpreter{}

type ValueRemoveContext = ValueTransferContext

type ComputationReporter interface {
	ReportComputation(compKind common.ComputationKind, intensity uint)
}

type ValueIterationContext interface {
	ValueTransferContext
	WithMutationPrevention(valueID atree.ValueID, f func())
}

type ValueStringContext interface {
	ValueIterationContext
}

// NoOpStringContext is the ValueStringContext implementation used in Value.RecursiveString method.
// Since Value.RecursiveString is a non-mutating operation, it should only need the no-op memory metering
// and a WithMutationPrevention implementation.
// All other methods should not be reachable, hence is safe to panic in them.
//
// TODO: Ideally, Value.RecursiveString shouldn't need the full ValueTransferContext.
// But that would require refactoring the iterator methods for arrays and dictionaries.
type NoOpStringContext struct{}

var _ ValueStringContext = NoOpStringContext{}

func (n NoOpStringContext) MeterMemory(_ common.MemoryUsage) error {
	return nil
}

func (n NoOpStringContext) WithMutationPrevention(_ atree.ValueID, f func()) {
	f()
	return
}

func (n NoOpStringContext) ReadStored(_ common.Address, _ common.StorageDomain, _ StorageMapKey) Value {
	panic(errors.NewUnreachableError())
}

func (n NoOpStringContext) ConvertStaticToSemaType(_ StaticType) (sema.Type, error) {
	panic(errors.NewUnreachableError())
}

func (n NoOpStringContext) IsSubType(_ StaticType, _ StaticType) bool {
	panic(errors.NewUnreachableError())
}

func (n NoOpStringContext) IsSubTypeOfSemaType(_ StaticType, _ sema.Type) bool {
	panic(errors.NewUnreachableError())
}

func (n NoOpStringContext) WriteStored(_ common.Address, _ common.StorageDomain, _ StorageMapKey, _ Value) (existed bool) {
	panic(errors.NewUnreachableError())
}

func (n NoOpStringContext) Storage() Storage {
	panic(errors.NewUnreachableError())
}

func (n NoOpStringContext) RemoveReferencedSlab(_ atree.Storable) {
	panic(errors.NewUnreachableError())
}

func (n NoOpStringContext) MaybeValidateAtreeValue(_ atree.Value) {
	panic(errors.NewUnreachableError())
}

func (n NoOpStringContext) MaybeValidateAtreeStorage() {
	panic(errors.NewUnreachableError())
}

func (n NoOpStringContext) InvalidateReferencedResources(_ Value, _ LocationRange) {
	panic(errors.NewUnreachableError())
}

func (n NoOpStringContext) CheckInvalidatedResourceOrResourceReference(_ Value, _ LocationRange) {
	panic(errors.NewUnreachableError())
}

func (n NoOpStringContext) MaybeTrackReferencedResourceKindedValue(_ *EphemeralReferenceValue) {
	panic(errors.NewUnreachableError())
}

func (n NoOpStringContext) ReportComputation(_ common.ComputationKind, _ uint) {
	panic(errors.NewUnreachableError())
}

func (n NoOpStringContext) TracingEnabled() bool {
	panic(errors.NewUnreachableError())
}

func (n NoOpStringContext) reportArrayValueDeepRemoveTrace(_ string, _ int, _ time.Duration) {
	panic(errors.NewUnreachableError())
}

func (n NoOpStringContext) reportArrayValueTransferTrace(_ string, _ int, _ time.Duration) {
	panic(errors.NewUnreachableError())
}

func (n NoOpStringContext) reportDictionaryValueTransferTrace(_ string, _ int, _ time.Duration) {
	panic(errors.NewUnreachableError())
}

func (n NoOpStringContext) reportDictionaryValueDeepRemoveTrace(_ string, _ int, _ time.Duration) {
	panic(errors.NewUnreachableError())
}

func (n NoOpStringContext) reportCompositeValueDeepRemoveTrace(_ string, _ string, _ string, _ time.Duration) {
	panic(errors.NewUnreachableError())
}

func (n NoOpStringContext) reportCompositeValueTransferTrace(_ string, _ string, _ string, _ time.Duration) {
	panic(errors.NewUnreachableError())
}

func (n NoOpStringContext) reportDomainStorageMapDeepRemoveTrace(_ string, _ int, _ time.Duration) {
	panic(errors.NewUnreachableError())
}

func (n NoOpStringContext) OnResourceOwnerChange(_ *CompositeValue, _ common.Address, _ common.Address) {
	panic(errors.NewUnreachableError())
}
