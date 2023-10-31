package interpreter

import (
	"github.com/onflow/atree"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/format"
)

// AccountLinkValue

type AccountLinkValue struct{}

func NewUnmeteredAccountLinkValue() AccountLinkValue {
	return EmptyAccountLinkValue
}

func NewAccountLinkValue(memoryGauge common.MemoryGauge) AccountLinkValue {
	common.UseMemory(memoryGauge, common.AccountLinkValueMemoryUsage)
	return NewUnmeteredAccountLinkValue()
}

var EmptyAccountLinkValue = AccountLinkValue{}

var _ Value = AccountLinkValue{}
var _ atree.Value = AccountLinkValue{}
var _ EquatableValue = AccountLinkValue{}
var _ LinkValue = AccountLinkValue{}

func (AccountLinkValue) isValue() {}

func (AccountLinkValue) isLinkValue() {}

func (v AccountLinkValue) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitAccountLinkValue(interpreter, v)
}

func (AccountLinkValue) Walk(_ *Interpreter, _ func(Value)) {
	// NO-OP
}

func (v AccountLinkValue) StaticType(interpreter *Interpreter) StaticType {
	// When iterating over public/private paths,
	// the values at these paths are AccountLinkValues,
	// placed there by the `linkAccount` function.
	//
	// These are loaded as links, however,
	// for the purposes of checking their type,
	// we treat them as capabilities
	return NewCapabilityStaticType(
		interpreter,
		ReferenceStaticType{
			BorrowedType:   authAccountStaticType,
			ReferencedType: authAccountStaticType,
		},
	)
}

func (AccountLinkValue) IsImportable(_ *Interpreter) bool {
	return false
}

func (v AccountLinkValue) String() string {
	return v.RecursiveString(SeenReferences{})
}

func (v AccountLinkValue) RecursiveString(_ SeenReferences) string {
	return format.AccountLink
}

func (v AccountLinkValue) MeteredString(_ common.MemoryGauge, _ SeenReferences) string {
	return format.AccountLink
}

func (v AccountLinkValue) ConformsToStaticType(
	_ *Interpreter,
	_ LocationRange,
	_ TypeConformanceResults,
) bool {
	return true
}

func (v AccountLinkValue) Equal(_ *Interpreter, _ LocationRange, other Value) bool {
	_, ok := other.(AccountLinkValue)
	return ok
}

func (AccountLinkValue) IsStorable() bool {
	return true
}

func (v AccountLinkValue) Storable(storage atree.SlabStorage, address atree.Address, maxInlineSize uint64) (atree.Storable, error) {
	return maybeLargeImmutableStorable(v, storage, address, maxInlineSize)
}

func (AccountLinkValue) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (AccountLinkValue) IsResourceKinded(_ *Interpreter) bool {
	return false
}

func (v AccountLinkValue) Transfer(
	interpreter *Interpreter,
	_ LocationRange,
	_ atree.Address,
	remove bool,
	storable atree.Storable,
	_ map[atree.StorageID]struct{},
) Value {
	if remove {
		interpreter.RemoveReferencedSlab(storable)
	}
	return v
}

func (AccountLinkValue) Clone(_ *Interpreter) Value {
	return AccountLinkValue{}
}

func (AccountLinkValue) DeepRemove(_ *Interpreter) {
	// NO-OP
}

func (v AccountLinkValue) ByteSize() uint32 {
	return mustStorableSize(v)
}

func (v AccountLinkValue) StoredValue(_ atree.SlabStorage) (atree.Value, error) {
	return v, nil
}

func (v AccountLinkValue) ChildStorables() []atree.Storable {
	return nil
}
