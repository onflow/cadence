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
	common.MemoryGauge
	StaticTypeConversionHandler
}

var _ TypeConverter = &Interpreter{}

func MustConvertStaticToSemaType(staticType StaticType, typeConverter TypeConverter) sema.Type {
	semaType, err := ConvertStaticToSemaType(typeConverter, staticType)
	if err != nil {
		panic(err)
	}
	return semaType
}

func MustSemaTypeOfValue(value Value, context ValueStaticTypeContext) sema.Type {
	staticType := value.StaticType(context)
	return MustConvertStaticToSemaType(staticType, context)
}

type StorageReader interface {
	ReadStored(
		storageAddress common.Address,
		domain common.StorageDomain,
		identifier StorageMapKey,
	) Value
}

var _ StorageReader = &Interpreter{}

type StorageWriter interface {
	WriteStored(
		storageAddress common.Address,
		domain common.StorageDomain,
		key StorageMapKey,
		value Value,
	) (existed bool)
}

var _ StorageWriter = &Interpreter{}

type ValueStaticTypeContext interface {
	common.MemoryGauge
	StorageReader
	TypeConverter
	IsRecovered(location common.Location) bool
}

var _ ValueStaticTypeContext = &Interpreter{}

type StorageContext interface {
	ValueStaticTypeContext
	common.MemoryGauge
	StorageMutationTracker
	StorageReader
	StorageWriter

	Storage() Storage
	MaybeValidateAtreeValue(v atree.Value)
	MaybeValidateAtreeStorage()
}

var _ StorageContext = &Interpreter{}

type ReferenceTracker interface {
	ClearReferencedResourceKindedValues(valueID atree.ValueID)
	ReferencedResourceKindedValues(valueID atree.ValueID) map[*EphemeralReferenceValue]struct{}
	CheckInvalidatedResourceOrResourceReference(value Value, locationRange LocationRange)
	MaybeTrackReferencedResourceKindedValue(ref *EphemeralReferenceValue)
}

var _ ReferenceTracker = &Interpreter{}

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

	WithMutationPrevention(valueID atree.ValueID, f func())
	ValidateMutation(valueID atree.ValueID, locationRange LocationRange)
}

var _ ValueTransferContext = &Interpreter{}

type ValueRemoveContext = ValueTransferContext

var _ ValueRemoveContext = &Interpreter{}

type ComputationReporter interface {
	ReportComputation(compKind common.ComputationKind, intensity uint)
}

var _ ComputationReporter = &Interpreter{}

type ContainerMutationContext interface {
	ValueTransferContext
}

var _ ContainerMutationContext = &Interpreter{}

type ValueStringContext interface {
	ValueTransferContext
}

var _ ValueStringContext = &Interpreter{}

type ReferenceCreationContext interface {
	common.MemoryGauge
	ReferenceTracker
}

var _ ReferenceCreationContext = &Interpreter{}

type MemberAccessibleContext interface {
	FunctionCreationContext
	ArrayCreationContext
	ResourceDestructionHandler

	AccountHandler() AccountHandlerFunc
	InjectedCompositeFieldsHandler() InjectedCompositeFieldsHandlerFunc
	GetMemberAccessContextForLocation(location common.Location) MemberAccessibleContext
}

var _ MemberAccessibleContext = &Interpreter{}

type FunctionCreationContext interface {
	StaticTypeAndReferenceContext
	GetCompositeValueFunctions(v *CompositeValue, locationRange LocationRange) *FunctionOrderedMap
}

var _ FunctionCreationContext = &Interpreter{}

type StaticTypeAndReferenceContext interface {
	common.MemoryGauge
	ValueStaticTypeContext
	ReferenceTracker
}

var _ StaticTypeAndReferenceContext = &Interpreter{}

type ArrayCreationContext interface {
	ValueTransferContext
}

var _ ArrayCreationContext = &Interpreter{}

type DictionaryCreationContext interface {
	ContainerMutationContext
}

var _ DictionaryCreationContext = &Interpreter{}

type StorageMutationTracker interface {
	RecordStorageMutation()
}

var _ StorageMutationTracker = &Interpreter{}

type ResourceDestructionHandler interface {
	EnforceNotResourceDestruction(
		valueID atree.ValueID,
		locationRange LocationRange,
	)
}

var _ ResourceDestructionHandler = &Interpreter{}

// NoOpStringContext is the ValueStringContext implementation used in Value.RecursiveString method.
// Since Value.RecursiveString is a non-mutating operation, it should only need the no-op memory metering
// and a WithMutationPrevention implementation.
// All other methods should not be reachable, hence is safe to panic in them.
//
// TODO: Ideally, Value.RecursiveString shouldn't need the full ValueTransferContext.
// But that would require refactoring the iterator methods for arrays and dictionaries.
type NoOpStringContext struct{}

var _ ValueStringContext = NoOpStringContext{}

func (ctx NoOpStringContext) MeterMemory(_ common.MemoryUsage) error {
	return nil
}

func (ctx NoOpStringContext) WithMutationPrevention(_ atree.ValueID, f func()) {
	f()
	return
}

func (ctx NoOpStringContext) ValidateMutation(_ atree.ValueID, _ LocationRange) {
	panic(errors.NewUnreachableError())
}

func (ctx NoOpStringContext) ReadStored(_ common.Address, _ common.StorageDomain, _ StorageMapKey) Value {
	panic(errors.NewUnreachableError())
}

func (ctx NoOpStringContext) ConvertStaticToSemaType(_ StaticType) (sema.Type, error) {
	panic(errors.NewUnreachableError())
}

func (ctx NoOpStringContext) IsSubType(_ StaticType, _ StaticType) bool {
	panic(errors.NewUnreachableError())
}

func (ctx NoOpStringContext) IsSubTypeOfSemaType(_ StaticType, _ sema.Type) bool {
	panic(errors.NewUnreachableError())
}

func (ctx NoOpStringContext) WriteStored(_ common.Address, _ common.StorageDomain, _ StorageMapKey, _ Value) (existed bool) {
	panic(errors.NewUnreachableError())
}

func (ctx NoOpStringContext) Storage() Storage {
	panic(errors.NewUnreachableError())
}

func (ctx NoOpStringContext) MaybeValidateAtreeValue(_ atree.Value) {
	panic(errors.NewUnreachableError())
}

func (ctx NoOpStringContext) MaybeValidateAtreeStorage() {
	panic(errors.NewUnreachableError())
}

func (ctx NoOpStringContext) InvalidateReferencedResources(_ Value, _ LocationRange) {
	panic(errors.NewUnreachableError())
}

func (ctx NoOpStringContext) CheckInvalidatedResourceOrResourceReference(_ Value, _ LocationRange) {
	panic(errors.NewUnreachableError())
}

func (ctx NoOpStringContext) MaybeTrackReferencedResourceKindedValue(_ *EphemeralReferenceValue) {
	panic(errors.NewUnreachableError())
}

func (ctx NoOpStringContext) ClearReferencedResourceKindedValues(valueID atree.ValueID) {
	panic(errors.NewUnreachableError())
}

func (ctx NoOpStringContext) ReferencedResourceKindedValues(valueID atree.ValueID) map[*EphemeralReferenceValue]struct{} {
	panic(errors.NewUnreachableError())
}

func (ctx NoOpStringContext) ReportComputation(_ common.ComputationKind, _ uint) {
	panic(errors.NewUnreachableError())
}

func (ctx NoOpStringContext) TracingEnabled() bool {
	panic(errors.NewUnreachableError())
}

func (ctx NoOpStringContext) ReportArrayValueDeepRemoveTrace(_ string, _ int, _ time.Duration) {
	panic(errors.NewUnreachableError())
}

func (ctx NoOpStringContext) ReportArrayValueTransferTrace(_ string, _ int, _ time.Duration) {
	panic(errors.NewUnreachableError())
}

func (ctx NoOpStringContext) ReportArrayValueConstructTrace(_ string, _ int, _ time.Duration) {
	panic(errors.NewUnreachableError())
}

func (ctx NoOpStringContext) ReportDictionaryValueTransferTrace(_ string, _ int, _ time.Duration) {
	panic(errors.NewUnreachableError())
}

func (ctx NoOpStringContext) ReportDictionaryValueDeepRemoveTrace(_ string, _ int, _ time.Duration) {
	panic(errors.NewUnreachableError())
}

func (ctx NoOpStringContext) ReportCompositeValueDeepRemoveTrace(_ string, _ string, _ string, _ time.Duration) {
	panic(errors.NewUnreachableError())
}

func (ctx NoOpStringContext) ReportDictionaryValueGetMemberTrace(_ string, _ int, _ string, _ time.Duration) {
	panic(errors.NewUnreachableError())
}

func (ctx NoOpStringContext) ReportDictionaryValueConstructTrace(_ string, _ int, _ time.Duration) {
	panic(errors.NewUnreachableError())
}

func (ctx NoOpStringContext) ReportCompositeValueTransferTrace(_ string, _ string, _ string, _ time.Duration) {
	panic(errors.NewUnreachableError())
}

func (ctx NoOpStringContext) ReportCompositeValueSetMemberTrace(_ string, _ string, _ string, _ string, _ time.Duration) {
	panic(errors.NewUnreachableError())
}

func (ctx NoOpStringContext) ReportCompositeValueGetMemberTrace(_ string, _ string, _ string, _ string, _ time.Duration) {
	panic(errors.NewUnreachableError())
}

func (ctx NoOpStringContext) ReportCompositeValueConstructTrace(_ string, _ string, _ string, _ time.Duration) {
	panic(errors.NewUnreachableError())
}

func (ctx NoOpStringContext) ReportDomainStorageMapDeepRemoveTrace(_ string, _ int, _ time.Duration) {
	panic(errors.NewUnreachableError())
}

func (ctx NoOpStringContext) OnResourceOwnerChange(_ *CompositeValue, _ common.Address, _ common.Address) {
	panic(errors.NewUnreachableError())
}

func (ctx NoOpStringContext) RecordStorageMutation() {
	panic(errors.NewUnreachableError())
}

func (ctx NoOpStringContext) GetEntitlementType(_ TypeID) (*sema.EntitlementType, error) {
	panic(errors.NewUnreachableError())
}

func (ctx NoOpStringContext) GetEntitlementMapType(_ TypeID) (*sema.EntitlementMapType, error) {
	panic(errors.NewUnreachableError())
}

func (ctx NoOpStringContext) GetInterfaceType(_ common.Location, _ string, _ TypeID) (*sema.InterfaceType, error) {
	panic(errors.NewUnreachableError())
}

func (ctx NoOpStringContext) GetCompositeType(_ common.Location, _ string, _ TypeID) (*sema.CompositeType, error) {
	panic(errors.NewUnreachableError())
}

func (ctx NoOpStringContext) IsRecovered(_ common.Location) bool {
	panic(errors.NewUnreachableError())
}
