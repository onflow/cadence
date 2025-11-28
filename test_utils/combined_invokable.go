package test_utils

import (
	"time"

	"github.com/onflow/atree"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
	. "github.com/onflow/cadence/test_utils/common_utils"
)

type CombinedInvokable struct {
	*interpreter.Interpreter
	*VMInvokable
}

var _ Invokable = &CombinedInvokable{}

func NewCombinedInvokable(
	interpreterInvokable *interpreter.Interpreter,
	vmInvokable *VMInvokable,
) *CombinedInvokable {
	return &CombinedInvokable{
		Interpreter: interpreterInvokable,
		VMInvokable: vmInvokable,
	}
}

func (i *CombinedInvokable) Invoke(functionName string, arguments ...interpreter.Value) (value interpreter.Value, err error) {
	if i.VMInvokable != nil {
		return i.VMInvokable.Invoke(functionName, arguments...)
	}

	return i.Interpreter.Invoke(functionName, arguments...)
}

func (i *CombinedInvokable) InvokeTransaction(arguments []interpreter.Value, signers ...interpreter.Value) (err error) {
	if i.VMInvokable != nil {
		return i.VMInvokable.InvokeTransaction(arguments, signers...)
	}

	return i.Interpreter.InvokeTransaction(arguments, signers...)
}

func (i *CombinedInvokable) GetGlobal(name string) interpreter.Value {
	if i.VMInvokable != nil {
		return i.VMInvokable.GetGlobal(name)
	}

	return i.Interpreter.GetGlobal(name)
}

func (i *CombinedInvokable) GetGlobalType(name string) (*sema.Variable, bool) {
	if i.VMInvokable != nil {
		return i.VMInvokable.GetGlobalType(name)
	}

	return i.Interpreter.GetGlobalType(name)
}

func (i *CombinedInvokable) InitializeContract(contractName string, arguments ...interpreter.Value) (*interpreter.CompositeValue, error) {
	if i.VMInvokable != nil {
		return i.VMInvokable.InitializeContract(contractName, arguments...)
	}

	// Interpreter don't need this functionality explicitly.
	return nil, nil
}

func (i *CombinedInvokable) MeterMemory(usage common.MemoryUsage) error {
	if i.VMInvokable != nil {
		return i.VMInvokable.MeterMemory(usage)
	}

	return i.Interpreter.MeterMemory(usage)
}

func (i *CombinedInvokable) MeterComputation(usage common.ComputationUsage) error {
	if i.VMInvokable != nil {
		return i.VMInvokable.MeterComputation(usage)
	}

	return i.Interpreter.MeterComputation(usage)
}

func (i *CombinedInvokable) ReadStored(storageAddress common.Address, domain common.StorageDomain, identifier interpreter.StorageMapKey) interpreter.Value {
	if i.VMInvokable != nil {
		return i.VMInvokable.ReadStored(storageAddress, domain, identifier)
	}

	return i.Interpreter.ReadStored(storageAddress, domain, identifier)
}

func (i *CombinedInvokable) GetEntitlementType(typeID interpreter.TypeID) (*sema.EntitlementType, error) {
	if i.VMInvokable != nil {
		return i.VMInvokable.GetEntitlementType(typeID)
	}

	return i.Interpreter.GetEntitlementType(typeID)
}

func (i *CombinedInvokable) GetEntitlementMapType(typeID interpreter.TypeID) (*sema.EntitlementMapType, error) {
	if i.VMInvokable != nil {
		return i.VMInvokable.GetEntitlementMapType(typeID)
	}

	return i.Interpreter.GetEntitlementMapType(typeID)
}

func (i *CombinedInvokable) GetInterfaceType(location common.Location, qualifiedIdentifier string, typeID interpreter.TypeID) (*sema.InterfaceType, error) {
	if i.VMInvokable != nil {
		return i.VMInvokable.GetInterfaceType(location, qualifiedIdentifier, typeID)
	}

	return i.Interpreter.GetInterfaceType(location, qualifiedIdentifier, typeID)
}

func (i *CombinedInvokable) GetCompositeType(location common.Location, qualifiedIdentifier string, typeID interpreter.TypeID) (*sema.CompositeType, error) {
	if i.VMInvokable != nil {
		return i.VMInvokable.GetCompositeType(location, qualifiedIdentifier, typeID)
	}

	return i.Interpreter.GetCompositeType(location, qualifiedIdentifier, typeID)
}

func (i *CombinedInvokable) SemaTypeFromStaticType(staticType interpreter.StaticType) sema.Type {
	if i.VMInvokable != nil {
		return i.VMInvokable.SemaTypeFromStaticType(staticType)
	}

	return i.Interpreter.SemaTypeFromStaticType(staticType)
}

func (i *CombinedInvokable) SemaAccessFromStaticAuthorization(auth interpreter.Authorization) (sema.Access, error) {
	if i.VMInvokable != nil {
		return i.VMInvokable.SemaAccessFromStaticAuthorization(auth)
	}

	return i.Interpreter.SemaAccessFromStaticAuthorization(auth)
}

func (i *CombinedInvokable) IsTypeInfoRecovered(location common.Location) bool {
	if i.VMInvokable != nil {
		return i.VMInvokable.IsTypeInfoRecovered(location)
	}

	return i.Interpreter.IsTypeInfoRecovered(location)
}

func (i *CombinedInvokable) RecordStorageMutation() {
	if i.VMInvokable != nil {
		i.VMInvokable.RecordStorageMutation()
		return
	}

	i.Interpreter.RecordStorageMutation()
}

func (i *CombinedInvokable) StorageMutatedDuringIteration() bool {
	if i.VMInvokable != nil {
		return i.VMInvokable.StorageMutatedDuringIteration()
	}

	return i.Interpreter.StorageMutatedDuringIteration()
}

func (i *CombinedInvokable) InStorageIteration() bool {
	if i.VMInvokable != nil {
		return i.VMInvokable.InStorageIteration()
	}

	return i.Interpreter.InStorageIteration()
}

func (i *CombinedInvokable) SetInStorageIteration(b bool) {
	if i.VMInvokable != nil {
		i.VMInvokable.SetInStorageIteration(b)
		return
	}

	i.Interpreter.SetInStorageIteration(b)
}

func (i *CombinedInvokable) WriteStored(
	storageAddress common.Address,
	domain common.StorageDomain,
	key interpreter.StorageMapKey,
	value interpreter.Value,
) (existed bool) {
	if i.VMInvokable != nil {
		return i.VMInvokable.WriteStored(storageAddress, domain, key, value)
	}

	return i.Interpreter.WriteStored(storageAddress, domain, key, value)
}

func (i *CombinedInvokable) Storage() interpreter.Storage {
	if i.VMInvokable != nil {
		return i.VMInvokable.Storage()
	}

	return i.Interpreter.Storage()
}

func (i *CombinedInvokable) MaybeValidateAtreeValue(v atree.Value) {
	if i.VMInvokable != nil {
		i.VMInvokable.MaybeValidateAtreeValue(v)
		return
	}

	i.Interpreter.MaybeValidateAtreeValue(v)
}

func (i *CombinedInvokable) MaybeValidateAtreeStorage() {
	if i.VMInvokable != nil {
		i.VMInvokable.MaybeValidateAtreeStorage()
		return
	}

	i.Interpreter.MaybeValidateAtreeStorage()
}

func (i *CombinedInvokable) ClearReferencedResourceKindedValues(valueID atree.ValueID) {
	if i.VMInvokable != nil {
		i.VMInvokable.ClearReferencedResourceKindedValues(valueID)
		return
	}

	i.Interpreter.ClearReferencedResourceKindedValues(valueID)
}

func (i *CombinedInvokable) ReferencedResourceKindedValues(valueID atree.ValueID) map[*interpreter.EphemeralReferenceValue]struct{} {
	if i.VMInvokable != nil {
		return i.VMInvokable.ReferencedResourceKindedValues(valueID)
	}

	return i.Interpreter.ReferencedResourceKindedValues(valueID)
}

func (i *CombinedInvokable) MaybeTrackReferencedResourceKindedValue(ref *interpreter.EphemeralReferenceValue) {
	if i.VMInvokable != nil {
		i.VMInvokable.MaybeTrackReferencedResourceKindedValue(ref)
		return
	}

	i.Interpreter.MaybeTrackReferencedResourceKindedValue(ref)
}

func (i *CombinedInvokable) ReportInvokeTrace(functionType string, functionName string, duration time.Duration) {
	if i.VMInvokable != nil {
		i.VMInvokable.ReportInvokeTrace(functionType, functionName, duration)
		return
	}

	i.Interpreter.ReportInvokeTrace(functionType, functionName, duration)
}

func (i *CombinedInvokable) ReportImportTrace(location string, duration time.Duration) {
	if i.VMInvokable != nil {
		i.VMInvokable.ReportImportTrace(location, duration)
		return
	}

	i.Interpreter.ReportImportTrace(location, duration)
}

func (i *CombinedInvokable) ReportEmitEventTrace(eventType string, duration time.Duration) {
	if i.VMInvokable != nil {
		i.VMInvokable.ReportEmitEventTrace(eventType, duration)
		return
	}

	i.Interpreter.ReportEmitEventTrace(eventType, duration)
}

func (i *CombinedInvokable) ReportArrayValueConstructTrace(valueID string, typeID string, duration time.Duration) {
	if i.VMInvokable != nil {
		i.VMInvokable.ReportArrayValueConstructTrace(valueID, typeID, duration)
		return
	}

	i.Interpreter.ReportArrayValueConstructTrace(valueID, typeID, duration)
}

func (i *CombinedInvokable) ReportArrayValueTransferTrace(valueID string, typeID string, duration time.Duration) {
	if i.VMInvokable != nil {
		i.VMInvokable.ReportArrayValueTransferTrace(valueID, typeID, duration)
		return
	}

	i.Interpreter.ReportArrayValueTransferTrace(valueID, typeID, duration)
}

func (i *CombinedInvokable) ReportArrayValueDeepRemoveTrace(valueID string, typeID string, duration time.Duration) {
	if i.VMInvokable != nil {
		i.VMInvokable.ReportArrayValueDeepRemoveTrace(valueID, typeID, duration)
		return
	}

	i.Interpreter.ReportArrayValueDeepRemoveTrace(valueID, typeID, duration)
}

func (i *CombinedInvokable) ReportArrayValueDestroyTrace(valueID string, typeID string, duration time.Duration) {
	if i.VMInvokable != nil {
		i.VMInvokable.ReportArrayValueDestroyTrace(valueID, typeID, duration)
		return
	}

	i.Interpreter.ReportArrayValueDestroyTrace(valueID, typeID, duration)
}

func (i *CombinedInvokable) ReportArrayValueConformsToStaticTypeTrace(valueID string, typeID string, duration time.Duration) {
	if i.VMInvokable != nil {
		i.VMInvokable.ReportArrayValueConformsToStaticTypeTrace(valueID, typeID, duration)
		return
	}

	i.Interpreter.ReportArrayValueConformsToStaticTypeTrace(valueID, typeID, duration)
}

func (i *CombinedInvokable) ReportDictionaryValueConstructTrace(valueID string, typeID string, duration time.Duration) {
	if i.VMInvokable != nil {
		i.VMInvokable.ReportDictionaryValueConstructTrace(valueID, typeID, duration)
		return
	}

	i.Interpreter.ReportDictionaryValueConstructTrace(valueID, typeID, duration)
}

func (i *CombinedInvokable) ReportDictionaryValueTransferTrace(valueID string, typeID string, duration time.Duration) {
	if i.VMInvokable != nil {
		i.VMInvokable.ReportDictionaryValueTransferTrace(valueID, typeID, duration)
		return
	}

	i.Interpreter.ReportDictionaryValueTransferTrace(valueID, typeID, duration)
}

func (i *CombinedInvokable) ReportDictionaryValueDeepRemoveTrace(valueID string, typeID string, duration time.Duration) {
	if i.VMInvokable != nil {
		i.VMInvokable.ReportDictionaryValueDeepRemoveTrace(valueID, typeID, duration)
		return
	}

	i.Interpreter.ReportDictionaryValueDeepRemoveTrace(valueID, typeID, duration)
}

func (i *CombinedInvokable) ReportDictionaryValueDestroyTrace(valueID string, typeID string, duration time.Duration) {
	if i.VMInvokable != nil {
		i.VMInvokable.ReportDictionaryValueDestroyTrace(valueID, typeID, duration)
		return
	}

	i.Interpreter.ReportDictionaryValueDestroyTrace(valueID, typeID, duration)
}

func (i *CombinedInvokable) ReportDictionaryValueConformsToStaticTypeTrace(valueID string, typeID string, duration time.Duration) {
	if i.VMInvokable != nil {
		i.VMInvokable.ReportDictionaryValueConformsToStaticTypeTrace(valueID, typeID, duration)
		return
	}

	i.Interpreter.ReportDictionaryValueConformsToStaticTypeTrace(valueID, typeID, duration)
}

func (i *CombinedInvokable) ReportCompositeValueConstructTrace(valueID string, typeID string, kind string, duration time.Duration) {
	if i.VMInvokable != nil {
		i.VMInvokable.ReportCompositeValueConstructTrace(valueID, typeID, kind, duration)
		return
	}

	i.Interpreter.ReportCompositeValueConstructTrace(valueID, typeID, kind, duration)
}

func (i *CombinedInvokable) ReportCompositeValueTransferTrace(valueID string, typeID string, kind string, duration time.Duration) {
	if i.VMInvokable != nil {
		i.VMInvokable.ReportCompositeValueTransferTrace(valueID, typeID, kind, duration)
		return
	}

	i.Interpreter.ReportCompositeValueTransferTrace(valueID, typeID, kind, duration)
}

func (i *CombinedInvokable) ReportCompositeValueDeepRemoveTrace(valueID string, typeID string, kind string, duration time.Duration) {
	if i.VMInvokable != nil {
		i.VMInvokable.ReportCompositeValueDeepRemoveTrace(valueID, typeID, kind, duration)
		return
	}

	i.Interpreter.ReportCompositeValueDeepRemoveTrace(valueID, typeID, kind, duration)
}

func (i *CombinedInvokable) ReportCompositeValueDestroyTrace(valueID string, typeID string, kind string, duration time.Duration) {
	if i.VMInvokable != nil {
		i.VMInvokable.ReportCompositeValueDestroyTrace(valueID, typeID, kind, duration)
		return
	}

	i.Interpreter.ReportCompositeValueDestroyTrace(valueID, typeID, kind, duration)
}

func (i *CombinedInvokable) ReportCompositeValueConformsToStaticTypeTrace(valueID string, typeID string, kind string, duration time.Duration) {
	if i.VMInvokable != nil {
		i.VMInvokable.ReportCompositeValueConformsToStaticTypeTrace(valueID, typeID, kind, duration)
		return
	}

	i.Interpreter.ReportCompositeValueConformsToStaticTypeTrace(valueID, typeID, kind, duration)
}

func (i *CombinedInvokable) ReportCompositeValueGetMemberTrace(
	valueID string,
	typeID string,
	kind string,
	name string,
	duration time.Duration,
) {
	if i.VMInvokable != nil {
		i.VMInvokable.ReportCompositeValueGetMemberTrace(valueID, typeID, kind, name, duration)
		return
	}

	i.Interpreter.ReportCompositeValueGetMemberTrace(valueID, typeID, kind, name, duration)
}

func (i *CombinedInvokable) ReportCompositeValueSetMemberTrace(
	valueID string,
	typeID string,
	kind string,
	name string,
	duration time.Duration,
) {
	if i.VMInvokable != nil {
		i.VMInvokable.ReportCompositeValueSetMemberTrace(valueID, typeID, kind, name, duration)
		return
	}

	i.Interpreter.ReportCompositeValueSetMemberTrace(valueID, typeID, kind, name, duration)
}

func (i *CombinedInvokable) ReportCompositeValueRemoveMemberTrace(
	valueID string,
	typeID string,
	kind string,
	name string,
	duration time.Duration,
) {
	if i.VMInvokable != nil {
		i.VMInvokable.ReportCompositeValueRemoveMemberTrace(valueID, typeID, kind, name, duration)
		return
	}

	i.Interpreter.ReportCompositeValueRemoveMemberTrace(valueID, typeID, kind, name, duration)
}

func (i *CombinedInvokable) ReportAtreeNewArrayFromBatchDataTrace(valueID string, typeID string, duration time.Duration) {
	if i.VMInvokable != nil {
		i.VMInvokable.ReportAtreeNewArrayFromBatchDataTrace(valueID, typeID, duration)
		return
	}

	i.Interpreter.ReportAtreeNewArrayFromBatchDataTrace(valueID, typeID, duration)
}

func (i *CombinedInvokable) ReportAtreeNewMapTrace(
	valueID string,
	typeID string,
	seed uint64,
	duration time.Duration,
) {
	if i.VMInvokable != nil {
		i.VMInvokable.ReportAtreeNewMapTrace(valueID, typeID, seed, duration)
		return
	}

	i.Interpreter.ReportAtreeNewMapTrace(valueID, typeID, seed, duration)
}

func (i *CombinedInvokable) ReportAtreeNewMapFromBatchDataTrace(
	valueID string,
	typeID string,
	seed uint64,
	duration time.Duration,
) {
	if i.VMInvokable != nil {
		i.VMInvokable.ReportAtreeNewMapFromBatchDataTrace(valueID, typeID, seed, duration)
		return
	}

	i.Interpreter.ReportAtreeNewMapFromBatchDataTrace(valueID, typeID, seed, duration)
}

func (i *CombinedInvokable) OnResourceOwnerChange(
	resource *interpreter.CompositeValue,
	oldOwner common.Address,
	newOwner common.Address,
) {
	if i.VMInvokable != nil {
		i.VMInvokable.OnResourceOwnerChange(resource, oldOwner, newOwner)
		return
	}

	i.Interpreter.OnResourceOwnerChange(resource, oldOwner, newOwner)
}

func (i *CombinedInvokable) WithContainerMutationPrevention(valueID atree.ValueID, f func()) {
	if i.VMInvokable != nil {
		i.VMInvokable.WithContainerMutationPrevention(valueID, f)
		return
	}

	i.Interpreter.WithContainerMutationPrevention(valueID, f)
}

func (i *CombinedInvokable) ValidateContainerMutation(valueID atree.ValueID) {
	if i.VMInvokable != nil {
		i.VMInvokable.ValidateContainerMutation(valueID)
		return
	}

	i.Interpreter.ValidateContainerMutation(valueID)
}

func (i *CombinedInvokable) EnforceNotResourceDestruction(valueID atree.ValueID) {
	if i.VMInvokable != nil {
		i.VMInvokable.EnforceNotResourceDestruction(valueID)
		return
	}

	i.Interpreter.EnforceNotResourceDestruction(valueID)
}

func (i *CombinedInvokable) GetCompositeValueFunctions(v *interpreter.CompositeValue) *interpreter.FunctionOrderedMap {
	if i.VMInvokable != nil {
		return i.VMInvokable.GetCompositeValueFunctions(v)
	}

	return i.Interpreter.GetCompositeValueFunctions(v)
}

func (i *CombinedInvokable) WithResourceDestruction(valueID atree.ValueID, f func()) {
	if i.VMInvokable != nil {
		i.VMInvokable.WithResourceDestruction(valueID, f)
		return
	}

	i.Interpreter.WithResourceDestruction(valueID, f)
}

func (i *CombinedInvokable) GetAccountHandlerFunc() interpreter.AccountHandlerFunc {
	if i.VMInvokable != nil {
		return i.VMInvokable.GetAccountHandlerFunc()
	}

	return i.Interpreter.GetAccountHandlerFunc()
}

func (i *CombinedInvokable) GetCapabilityControllerIterations() map[interpreter.AddressPath]int {
	if i.VMInvokable != nil {
		return i.VMInvokable.GetCapabilityControllerIterations()
	}

	return i.Interpreter.GetCapabilityControllerIterations()
}

func (i *CombinedInvokable) SetMutationDuringCapabilityControllerIteration() {
	if i.VMInvokable != nil {
		i.VMInvokable.SetMutationDuringCapabilityControllerIteration()
		return
	}

	i.Interpreter.SetMutationDuringCapabilityControllerIteration()
}

func (i *CombinedInvokable) MutationDuringCapabilityControllerIteration() bool {
	if i.VMInvokable != nil {
		return i.VMInvokable.MutationDuringCapabilityControllerIteration()
	}

	return i.Interpreter.MutationDuringCapabilityControllerIteration()
}

func (i *CombinedInvokable) GetContractValue(contractLocation common.AddressLocation) *interpreter.CompositeValue {
	if i.VMInvokable != nil {
		return i.VMInvokable.GetContractValue(contractLocation)
	}

	return i.Interpreter.GetContractValue(contractLocation)
}

func (i *CombinedInvokable) SetAttachmentIteration(composite *interpreter.CompositeValue, state bool) bool {
	if i.VMInvokable != nil {
		return i.VMInvokable.SetAttachmentIteration(composite, state)
	}

	return i.Interpreter.SetAttachmentIteration(composite, state)
}

func (i *CombinedInvokable) GetInjectedCompositeFieldsHandler() interpreter.InjectedCompositeFieldsHandlerFunc {
	if i.VMInvokable != nil {
		return i.VMInvokable.GetInjectedCompositeFieldsHandler()
	}

	return i.Interpreter.GetInjectedCompositeFieldsHandler()
}

func (i *CombinedInvokable) GetMemberAccessContextForLocation(location common.Location) interpreter.MemberAccessibleContext {
	if i.VMInvokable != nil {
		return i.VMInvokable.GetMemberAccessContextForLocation(location)
	}

	return i.Interpreter.GetMemberAccessContextForLocation(location)
}

func (i *CombinedInvokable) GetMethod(value interpreter.MemberAccessibleValue, name string) interpreter.FunctionValue {
	if i.VMInvokable != nil {
		return i.VMInvokable.GetMethod(value, name)
	}

	return i.Interpreter.GetMethod(value, name)
}

func (i *CombinedInvokable) MaybeUpdateStorageReferenceMemberReceiver(
	storageReference *interpreter.StorageReferenceValue,
	referencedValue interpreter.Value,
	member interpreter.Value,
) interpreter.Value {
	if i.VMInvokable != nil {
		return i.VMInvokable.MaybeUpdateStorageReferenceMemberReceiver(storageReference, referencedValue, member)
	}

	return i.Interpreter.MaybeUpdateStorageReferenceMemberReceiver(storageReference, referencedValue, member)
}

func (i *CombinedInvokable) RecoverErrors(onError func(error)) {
	if i.VMInvokable != nil {
		i.VMInvokable.RecoverErrors(onError)
		return
	}

	i.Interpreter.RecoverErrors(onError)
}

func (i *CombinedInvokable) GetValidateAccountCapabilitiesGetHandler() interpreter.ValidateAccountCapabilitiesGetHandlerFunc {
	if i.VMInvokable != nil {
		return i.VMInvokable.GetValidateAccountCapabilitiesGetHandler()
	}

	return i.Interpreter.GetValidateAccountCapabilitiesGetHandler()
}

func (i *CombinedInvokable) GetValidateAccountCapabilitiesPublishHandler() interpreter.ValidateAccountCapabilitiesPublishHandlerFunc {
	if i.VMInvokable != nil {
		return i.VMInvokable.GetValidateAccountCapabilitiesPublishHandler()
	}

	return i.Interpreter.GetValidateAccountCapabilitiesPublishHandler()
}

func (i *CombinedInvokable) GetCapabilityBorrowHandler() interpreter.CapabilityBorrowHandlerFunc {
	if i.VMInvokable != nil {
		return i.VMInvokable.GetCapabilityBorrowHandler()
	}

	return i.Interpreter.GetCapabilityBorrowHandler()
}

func (i *CombinedInvokable) GetCapabilityCheckHandler() interpreter.CapabilityCheckHandlerFunc {
	if i.VMInvokable != nil {
		return i.VMInvokable.GetCapabilityCheckHandler()
	}

	return i.Interpreter.GetCapabilityCheckHandler()
}

func (i *CombinedInvokable) GetValueOfVariable(name string) interpreter.Value {
	if i.VMInvokable != nil {
		return i.VMInvokable.GetValueOfVariable(name)
	}

	return i.Interpreter.GetValueOfVariable(name)
}

func (i *CombinedInvokable) LocationRange() interpreter.LocationRange {
	if i.VMInvokable != nil {
		return i.VMInvokable.LocationRange()
	}

	return i.Interpreter.LocationRange()
}

func (i *CombinedInvokable) GetLocation() common.Location {
	if i.VMInvokable != nil {
		return i.VMInvokable.GetLocation()
	}

	return i.Interpreter.GetLocation()
}

func (i *CombinedInvokable) InvokeFunction(fn interpreter.FunctionValue, arguments []interpreter.Value) interpreter.Value {
	if i.VMInvokable != nil {
		return i.VMInvokable.InvokeFunction(fn, arguments)
	}

	return i.Interpreter.InvokeFunction(fn, arguments)
}

func (i *CombinedInvokable) EmitEvent(
	context interpreter.ValueExportContext,
	eventType *sema.CompositeType,
	eventFields []interpreter.Value,
) {
	if i.VMInvokable != nil {
		i.VMInvokable.EmitEvent(context, eventType, eventFields)
		return
	}

	i.Interpreter.EmitEvent(context, eventType, eventFields)
}

func (i *CombinedInvokable) GetResourceDestructionContextForLocation(location common.Location) interpreter.ResourceDestructionContext {
	if i.VMInvokable != nil {
		return i.VMInvokable.GetResourceDestructionContextForLocation(location)
	}

	return i.Interpreter.GetResourceDestructionContextForLocation(location)
}

func (i *CombinedInvokable) DefaultDestroyEvents(resourceValue *interpreter.CompositeValue) []*interpreter.CompositeValue {
	if i.VMInvokable != nil {
		return i.VMInvokable.DefaultDestroyEvents(resourceValue)
	}

	return i.Interpreter.DefaultDestroyEvents(resourceValue)
}
