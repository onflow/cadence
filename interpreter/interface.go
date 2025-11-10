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
	"github.com/onflow/atree"

	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/sema"
)

type TypeConverter interface {
	common.MemoryGauge
	StaticTypeConversionHandler
	SemaTypeFromStaticType(staticType StaticType) sema.Type
	SemaAccessFromStaticAuthorization(auth Authorization) sema.Access
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
	IsTypeInfoRecovered(location common.Location) bool
}

var _ ValueStaticTypeContext = &Interpreter{}

type ValueStaticTypeConformanceContext interface {
	ValueStaticTypeContext
	ContainerMutationContext
}

var _ ValueStaticTypeConformanceContext = &Interpreter{}

type StorageContext interface {
	ValueStaticTypeContext
	common.MemoryGauge
	StorageMutationTracker
	StorageIterationTracker
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
	MaybeTrackReferencedResourceKindedValue(ref *EphemeralReferenceValue)
}

var _ ReferenceTracker = &Interpreter{}

type ValueTransferContext interface {
	StorageContext
	ReferenceTracker
	common.ComputationGauge
	Tracer

	OnResourceOwnerChange(
		resource *CompositeValue,
		oldOwner common.Address,
		newOwner common.Address,
	)

	WithContainerMutationPrevention(valueID atree.ValueID, f func())
	ValidateContainerMutation(valueID atree.ValueID)

	EnforceNotResourceDestruction(valueID atree.ValueID)
}

var _ ValueTransferContext = &Interpreter{}

type ValueConversionContext interface {
	ValueTransferContext
}

var _ ValueTransferContext = &Interpreter{}

type ValueCreationContext interface {
	ArrayCreationContext
	DictionaryCreationContext
}

var _ ValueCreationContext = &Interpreter{}

type ValueRemoveContext = ValueTransferContext

var _ ValueRemoveContext = &Interpreter{}

type ContainerMutationContext interface {
	ValueTransferContext
}

var _ ContainerMutationContext = &Interpreter{}

type ValueStringContext interface {
	ValueTransferContext
}

var _ ValueStringContext = &Interpreter{}

type ValueCloneContext interface {
	StorageContext
	ReferenceTracker
}

var _ ValueCloneContext = &Interpreter{}

type ValueImportableContext interface {
	ContainerMutationContext
}

var _ ValueImportableContext = &Interpreter{}

type ValueVisitContext interface {
	ValueWalkContext
}

var _ ValueVisitContext = &Interpreter{}

type ReferenceCreationContext interface {
	common.MemoryGauge
	ReferenceTracker
	ValueStaticTypeContext
}

var _ ReferenceCreationContext = &Interpreter{}

type GetReferenceContext interface {
	ReferenceCreationContext
}

var _ GetReferenceContext = &Interpreter{}

type IterableValueForeachContext interface {
	ValueTransferContext
}

var _ IterableValueForeachContext = &Interpreter{}

type AccountHandlerContext interface {
	GetAccountHandlerFunc() AccountHandlerFunc
}

var _ AccountHandlerContext = &Interpreter{}

type MemberAccessibleContext interface {
	FunctionCreationContext
	ArrayCreationContext
	ResourceDestructionHandler
	AccountHandlerContext
	CapabilityControllerIterationContext
	AccountContractBorrowContext
	AttachmentContext

	GetInjectedCompositeFieldsHandler() InjectedCompositeFieldsHandlerFunc
	GetMemberAccessContextForLocation(location common.Location) MemberAccessibleContext

	GetMethod(value MemberAccessibleValue, name string) FunctionValue
	MaybeUpdateStorageReferenceMemberReceiver(
		storageReference *StorageReferenceValue,
		referencedValue Value,
		member Value,
	) Value
}

var _ MemberAccessibleContext = &Interpreter{}

type FunctionCreationContext interface {
	StaticTypeAndReferenceContext
	CompositeFunctionContext
}

var _ FunctionCreationContext = &Interpreter{}

type CompositeFunctionContext interface {
	GetCompositeValueFunctions(v *CompositeValue) *FunctionOrderedMap
}

var _ CompositeFunctionContext = &Interpreter{}

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
	StorageMutatedDuringIteration() bool
}

var _ StorageMutationTracker = &Interpreter{}

type StorageIterationTracker interface {
	InStorageIteration() bool
	SetInStorageIteration(bool)
}

var _ StorageIterationTracker = &Interpreter{}

type ResourceDestructionHandler interface {
	WithResourceDestruction(valueID atree.ValueID, f func())
}

var _ ResourceDestructionHandler = &Interpreter{}

type AccountCapabilityCreationContext interface {
	StorageCapabilityCreationContext
}

var _ AccountCapabilityCreationContext = &Interpreter{}

type ValueCapabilityControllerReferenceValueContext interface {
	FunctionCreationContext
	ValueStaticTypeContext
	AccountHandlerContext
	AccountCreationContext
}

var _ ValueCapabilityControllerReferenceValueContext = &Interpreter{}

type StorageCapabilityCreationContext interface {
	FunctionCreationContext
	CapabilityControllerContext
}

var _ StorageCapabilityCreationContext = &Interpreter{}

type CapabilityControllerReferenceContext interface {
	StorageReader
	ReferenceCreationContext
}

var _ CapabilityControllerReferenceContext = &Interpreter{}

type CapabilityControllerContext interface {
	StorageContext
	DictionaryCreationContext
	ValueExportContext
	CapabilityControllerIterationContext
}

var _ CapabilityControllerContext = &Interpreter{}

type CapabilityControllerIterationContext interface {
	GetCapabilityControllerIterations() map[AddressPath]int
	SetMutationDuringCapabilityControllerIteration()
	MutationDuringCapabilityControllerIteration() bool
}

var _ CapabilityControllerContext = &Interpreter{}

type GetCapabilityControllerContext interface {
	TypeConverter
	StorageReader
}

var _ GetCapabilityControllerContext = &Interpreter{}

type GetCapabilityControllerReferenceContext interface {
	GetCapabilityControllerContext
	ValueCapabilityControllerReferenceValueContext
}

var _ GetCapabilityControllerReferenceContext = &Interpreter{}

type CheckCapabilityControllerContext interface {
	GetCapabilityControllerReferenceContext
}

var _ CheckCapabilityControllerContext = &Interpreter{}

type BorrowCapabilityControllerContext interface {
	GetCapabilityControllerReferenceContext
}

var _ BorrowCapabilityControllerContext = &Interpreter{}

type CapabilityHandlers interface {
	GetValidateAccountCapabilitiesGetHandler() ValidateAccountCapabilitiesGetHandlerFunc
	GetValidateAccountCapabilitiesPublishHandler() ValidateAccountCapabilitiesPublishHandlerFunc
	GetCapabilityBorrowHandler() CapabilityBorrowHandlerFunc
}

var _ CapabilityHandlers = &Interpreter{}

type StringValueFunctionContext interface {
	common.Gauge
}

var _ StringValueFunctionContext = &Interpreter{}

// TODO: This is used by the FVM.
//
//	Check and the functionalities needed.
type AccountCapabilityGetValidationContext interface {
}

var _ AccountCapabilityGetValidationContext = &Interpreter{}

// TODO: This is used by the FVM.
//
//	Check and the functionalities needed.
type AccountCapabilityPublishValidationContext interface {
}

var _ AccountCapabilityPublishValidationContext = &Interpreter{}

type LocationRangeProvider interface {
	LocationRange() LocationRange
}

type ResourceDestructionContext interface {
	ValueWalkContext
	ResourceDestructionHandler
	CompositeFunctionContext
	EventContext
	InvocationContext
	LocationRangeProvider

	GetResourceDestructionContextForLocation(location common.Location) ResourceDestructionContext
	DefaultDestroyEvents(resourceValue *CompositeValue) []*CompositeValue
}

var _ ResourceDestructionContext = &Interpreter{}

type ValueWalkContext interface {
	ContainerMutationContext
}

var _ ValueWalkContext = &Interpreter{}

type EventContext interface {
	EmitEvent(
		context ValueExportContext,
		eventType *sema.CompositeType,
		eventFields []Value,
	)
}

var _ EventContext = &Interpreter{}

type AttachmentContext interface {
	ValueStaticTypeContext
	ReferenceCreationContext
	SetAttachmentIteration(composite *CompositeValue, state bool) bool
}

var _ AttachmentContext = &Interpreter{}

type StoredValueCheckContext interface {
	TypeConverter
	CheckCapabilityControllerContext
	GetCapabilityCheckHandler() CapabilityCheckHandlerFunc
}

var _ StoredValueCheckContext = &Interpreter{}

// InvocationContext is a composite of all contexts, since function invocations
// can perform various operations, and hence need to provide all possible contexts to it.
type InvocationContext interface {
	StorageContext
	ValueStringContext
	MemberAccessibleContext
	AttachmentContext
	ErrorHandler
	ArrayCreationContext
	AccountCreationContext
	BorrowCapabilityControllerContext
	AccountCapabilityGetValidationContext
	CapabilityHandlers
	StoredValueCheckContext
	VariableResolver
	LocationRangeProvider

	GetLocation() common.Location

	InvokeFunction(
		fn FunctionValue,
		arguments []Value,
	) Value
}

var _ InvocationContext = &Interpreter{}

type ValueExportContext interface {
	ContainerMutationContext // needed for container iteration
	CompositeValueExportContext
}

var _ ValueExportContext = &Interpreter{}

type CompositeValueExportContext interface {
	MemberAccessibleContext
	AttachmentContext
}

var _ CompositeValueExportContext = &Interpreter{}

type PublicKeyCreationContext interface {
	MemberAccessibleContext
}

var _ PublicKeyCreationContext = &Interpreter{}

type PublicKeyValidationContext interface {
	PublicKeyCreationContext
}

var _ PublicKeyValidationContext = &Interpreter{}

type AccountKeyCreationContext interface {
	PublicKeyCreationContext
	AccountCapabilityCreationContext
}

var _ AccountKeyCreationContext = &Interpreter{}

type AccountCreationContext interface {
	AccountKeyCreationContext
	AccountContractCreationContext
}

var _ AccountCreationContext = &Interpreter{}

type AccountContractCreationContext interface {
	AccountContractBorrowContext
}

var _ AccountContractCreationContext = &Interpreter{}

type AccountContractBorrowContext interface {
	FunctionCreationContext
	GetContractValue(contractLocation common.AddressLocation) *CompositeValue
}

var _ AccountContractBorrowContext = &Interpreter{}

type ErrorHandler interface {
	RecoverErrors(onError func(error))
}

var _ ErrorHandler = &Interpreter{}

type VariableResolver interface {
	GetValueOfVariable(name string) Value
}

var _ VariableResolver = &Interpreter{}

// NoOpStringContext is the ValueStringContext implementation used in Value.RecursiveString method.
// Since Value.RecursiveString is a non-mutating operation, it should only need the no-op memory metering
// and a WithMutationPrevention implementation.
// All other methods should not be reachable, hence is safe to panic in them.
//
// TODO: Ideally, Value.RecursiveString shouldn't need the full ValueTransferContext.
// But that would require refactoring the iterator methods for arrays and dictionaries.
type NoOpStringContext struct {
	NoOpTracer
}

var _ ValueStringContext = NoOpStringContext{}

func (ctx NoOpStringContext) MeterMemory(_ common.MemoryUsage) error {
	return nil
}

func (ctx NoOpStringContext) MeterComputation(_ common.ComputationUsage) error {
	panic(errors.NewUnreachableError())
}

func (ctx NoOpStringContext) WithContainerMutationPrevention(_ atree.ValueID, f func()) {
	f()
}

func (ctx NoOpStringContext) ValidateContainerMutation(_ atree.ValueID) {
	panic(errors.NewUnreachableError())
}

func (ctx NoOpStringContext) EnforceNotResourceDestruction(_ atree.ValueID) {
	panic(errors.NewUnreachableError())
}

func (ctx NoOpStringContext) ReadStored(_ common.Address, _ common.StorageDomain, _ StorageMapKey) Value {
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

func (ctx NoOpStringContext) MaybeTrackReferencedResourceKindedValue(_ *EphemeralReferenceValue) {
	panic(errors.NewUnreachableError())
}

func (ctx NoOpStringContext) ClearReferencedResourceKindedValues(_ atree.ValueID) {
	panic(errors.NewUnreachableError())
}

func (ctx NoOpStringContext) ReferencedResourceKindedValues(_ atree.ValueID) map[*EphemeralReferenceValue]struct{} {
	panic(errors.NewUnreachableError())
}

func (ctx NoOpStringContext) OnResourceOwnerChange(_ *CompositeValue, _ common.Address, _ common.Address) {
	panic(errors.NewUnreachableError())
}

func (ctx NoOpStringContext) RecordStorageMutation() {
	panic(errors.NewUnreachableError())
}

func (ctx NoOpStringContext) StorageMutatedDuringIteration() bool {
	panic(errors.NewUnreachableError())
}

func (ctx NoOpStringContext) InStorageIteration() bool {
	panic(errors.NewUnreachableError())
}

func (ctx NoOpStringContext) SetInStorageIteration(_ bool) {
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

func (ctx NoOpStringContext) IsTypeInfoRecovered(_ common.Location) bool {
	panic(errors.NewUnreachableError())
}

func (ctx NoOpStringContext) SemaTypeFromStaticType(_ StaticType) sema.Type {
	panic(errors.NewUnreachableError())
}

func (ctx NoOpStringContext) SemaAccessFromStaticAuthorization(auth Authorization) sema.Access {
	panic(errors.NewUnreachableError())
}
