package interpreter

import (
	"github.com/onflow/atree"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/format"
	"github.com/onflow/cadence/runtime/sema"
)

// PathValue

type PathValue struct {
	Identifier string
	Domain     common.PathDomain
}

func NewUnmeteredPathValue(domain common.PathDomain, identifier string) PathValue {
	return PathValue{Domain: domain, Identifier: identifier}
}

func NewPathValue(
	memoryGauge common.MemoryGauge,
	domain common.PathDomain,
	identifier string,
) PathValue {
	common.UseMemory(memoryGauge, common.PathValueMemoryUsage)
	return NewUnmeteredPathValue(domain, identifier)
}

var EmptyPathValue = PathValue{}

var _ Value = PathValue{}
var _ atree.Storable = PathValue{}
var _ EquatableValue = PathValue{}
var _ HashableValue = PathValue{}
var _ MemberAccessibleValue = PathValue{}

func (PathValue) isValue() {}

func (v PathValue) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitPathValue(interpreter, v)
}

func (PathValue) Walk(_ *Interpreter, _ func(Value)) {
	// NO-OP
}

func (v PathValue) StaticType(interpreter *Interpreter) StaticType {
	switch v.Domain {
	case common.PathDomainStorage:
		return NewPrimitiveStaticType(interpreter, PrimitiveStaticTypeStoragePath)
	case common.PathDomainPublic:
		return NewPrimitiveStaticType(interpreter, PrimitiveStaticTypePublicPath)
	case common.PathDomainPrivate:
		return NewPrimitiveStaticType(interpreter, PrimitiveStaticTypePrivatePath)
	default:
		panic(errors.NewUnreachableError())
	}
}

func (v PathValue) IsImportable(_ *Interpreter) bool {
	switch v.Domain {
	case common.PathDomainStorage:
		return sema.StoragePathType.Importable
	case common.PathDomainPublic:
		return sema.PublicPathType.Importable
	case common.PathDomainPrivate:
		return sema.PrivatePathType.Importable
	default:
		panic(errors.NewUnreachableError())
	}
}

func (v PathValue) String() string {
	return format.Path(
		v.Domain.Identifier(),
		v.Identifier,
	)
}

func (v PathValue) RecursiveString(_ SeenReferences) string {
	return v.String()
}

func (v PathValue) MeteredString(memoryGauge common.MemoryGauge, _ SeenReferences) string {
	// len(domain) + len(identifier) + '/' x2
	strLen := len(v.Domain.Identifier()) + len(v.Identifier) + 2
	common.UseMemory(memoryGauge, common.NewRawStringMemoryUsage(strLen))
	return v.String()
}

func (v PathValue) GetMember(inter *Interpreter, locationRange LocationRange, name string) Value {
	switch name {

	case sema.ToStringFunctionName:
		return NewHostFunctionValue(
			inter,
			sema.ToStringFunctionType,
			func(invocation Invocation) Value {
				interpreter := invocation.Interpreter

				domainLength := len(v.Domain.Identifier())
				identifierLength := len(v.Identifier)

				memoryUsage := common.NewStringMemoryUsage(
					safeAdd(domainLength, identifierLength, locationRange),
				)

				return NewStringValue(
					interpreter,
					memoryUsage,
					v.String,
				)
			},
		)
	}

	return nil
}

func (PathValue) RemoveMember(_ *Interpreter, _ LocationRange, _ string) Value {
	// Paths have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (PathValue) SetMember(_ *Interpreter, _ LocationRange, _ string, _ Value) bool {
	// Paths have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v PathValue) ConformsToStaticType(
	_ *Interpreter,
	_ LocationRange,
	_ TypeConformanceResults,
) bool {
	return true
}

func (v PathValue) Equal(_ *Interpreter, _ LocationRange, other Value) bool {
	otherPath, ok := other.(PathValue)
	if !ok {
		return false
	}

	return otherPath.Identifier == v.Identifier &&
		otherPath.Domain == v.Domain
}

// HashInput returns a byte slice containing:
// - HashInputTypePath (1 byte)
// - domain (1 byte)
// - identifier (n bytes)
func (v PathValue) HashInput(_ *Interpreter, _ LocationRange, scratch []byte) []byte {
	length := 1 + 1 + len(v.Identifier)
	var buffer []byte
	if length <= len(scratch) {
		buffer = scratch[:length]
	} else {
		buffer = make([]byte, length)
	}

	buffer[0] = byte(HashInputTypePath)
	buffer[1] = byte(v.Domain)
	copy(buffer[2:], v.Identifier)
	return buffer
}

func (PathValue) IsStorable() bool {
	return true
}

func convertPath(interpreter *Interpreter, domain common.PathDomain, value Value) Value {
	stringValue, ok := value.(*StringValue)
	if !ok {
		return Nil
	}

	_, err := sema.CheckPathLiteral(
		domain.Identifier(),
		stringValue.Str,
		ReturnEmptyRange,
		ReturnEmptyRange,
	)
	if err != nil {
		return Nil
	}

	return NewSomeValueNonCopying(
		interpreter,
		NewPathValue(
			interpreter,
			domain,
			stringValue.Str,
		),
	)
}

func ConvertPublicPath(interpreter *Interpreter, value Value) Value {
	return convertPath(interpreter, common.PathDomainPublic, value)
}

func ConvertPrivatePath(interpreter *Interpreter, value Value) Value {
	return convertPath(interpreter, common.PathDomainPrivate, value)
}

func ConvertStoragePath(interpreter *Interpreter, value Value) Value {
	return convertPath(interpreter, common.PathDomainStorage, value)
}

func (v PathValue) Storable(
	storage atree.SlabStorage,
	address atree.Address,
	maxInlineSize uint64,
) (atree.Storable, error) {
	return maybeLargeImmutableStorable(
		v,
		storage,
		address,
		maxInlineSize,
	)
}

func (PathValue) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (PathValue) IsResourceKinded(_ *Interpreter) bool {
	return false
}

func (v PathValue) Transfer(
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

func (v PathValue) Clone(_ *Interpreter) Value {
	return v
}

func (PathValue) DeepRemove(_ *Interpreter) {
	// NO-OP
}

func (v PathValue) ByteSize() uint32 {
	// tag number (2 bytes) + array head (1 byte) + domain (CBOR uint) + identifier (CBOR string)
	return cborTagSize + 1 + getUintCBORSize(uint64(v.Domain)) + getBytesCBORSize([]byte(v.Identifier))
}

func (v PathValue) StoredValue(_ atree.SlabStorage) (atree.Value, error) {
	return v, nil
}

func (PathValue) ChildStorables() []atree.Storable {
	return nil
}
