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
	"github.com/onflow/cadence/format"
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/values"
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

func (PathValue) IsValue() {}

func (v PathValue) Accept(context ValueVisitContext, visitor Visitor, _ LocationRange) {
	visitor.VisitPathValue(context, v)
}

func (PathValue) Walk(_ ValueWalkContext, _ func(Value), _ LocationRange) {
	// NO-OP
}

func (v PathValue) StaticType(context ValueStaticTypeContext) StaticType {
	switch v.Domain {
	case common.PathDomainStorage:
		return NewPrimitiveStaticType(context, PrimitiveStaticTypeStoragePath)
	case common.PathDomainPublic:
		return NewPrimitiveStaticType(context, PrimitiveStaticTypePublicPath)
	case common.PathDomainPrivate:
		return NewPrimitiveStaticType(context, PrimitiveStaticTypePrivatePath)
	default:
		panic(errors.NewUnreachableError())
	}
}

func (v PathValue) IsImportable(_ ValueImportableContext, _ LocationRange) bool {
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

func (v PathValue) MeteredString(context ValueStringContext, _ SeenReferences, _ LocationRange) string {
	// len(domain) + len(identifier) + '/' x2
	strLen := len(v.Domain.Identifier()) + len(v.Identifier) + 2
	common.UseMemory(context, common.NewRawStringMemoryUsage(strLen))
	return v.String()
}

func (v PathValue) GetMember(context MemberAccessibleContext, locationRange LocationRange, name string) Value {
	return context.GetMethod(v, name, locationRange)
}

func (v PathValue) GetMethod(
	context MemberAccessibleContext,
	locationRange LocationRange,
	name string,
) FunctionValue {
	switch name {

	case sema.ToStringFunctionName:
		return NewBoundHostFunctionValue(
			context,
			v,
			sema.ToStringFunctionType,
			func(v PathValue, invocation Invocation) Value {
				interpreter := invocation.InvocationContext

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

func (PathValue) RemoveMember(_ ValueTransferContext, _ LocationRange, _ string) Value {
	// Paths have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (PathValue) SetMember(_ ValueTransferContext, _ LocationRange, _ string, _ Value) bool {
	// Paths have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v PathValue) ConformsToStaticType(
	_ ValueStaticTypeConformanceContext,
	_ LocationRange,
	_ TypeConformanceResults,
) bool {
	return true
}

func (v PathValue) Equal(context ValueComparisonContext, _ LocationRange, other Value) bool {
	otherPath, ok := other.(PathValue)
	if !ok {
		return false
	}

	if otherPath.Domain != v.Domain {
		return false
	}

	common.UseComputation(
		context,
		common.ComputationUsage{
			Kind:      common.ComputationKindStringComparison,
			Intensity: uint64(minStringLength(v.Identifier, otherPath.Identifier)),
		},
	)

	return otherPath.Identifier == v.Identifier
}

// HashInput returns a byte slice containing:
// - HashInputTypePath (1 byte)
// - domain (1 byte)
// - identifier (n bytes)
func (v PathValue) HashInput(_ common.Gauge, _ LocationRange, scratch []byte) []byte {
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

func newPathFromStringValue(gauge common.MemoryGauge, domain common.PathDomain, value Value) Value {
	stringValue, ok := value.(*StringValue)
	if !ok {
		return Nil
	}

	// NOTE: any identifier is allowed, it does not have to match the syntax for path literals

	return NewSomeValueNonCopying(
		gauge,
		NewPathValue(
			gauge,
			domain,
			stringValue.Str,
		),
	)
}

func (v PathValue) Storable(
	storage atree.SlabStorage,
	address atree.Address,
	maxInlineSize uint64,
) (atree.Storable, error) {
	return values.MaybeLargeImmutableStorable(
		v,
		storage,
		address,
		maxInlineSize,
	)
}

func (PathValue) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (PathValue) IsResourceKinded(_ ValueStaticTypeContext) bool {
	return false
}

func (v PathValue) Transfer(
	context ValueTransferContext,
	_ LocationRange,
	_ atree.Address,
	remove bool,
	storable atree.Storable,
	_ map[atree.ValueID]struct{},
	_ bool,
) Value {
	if remove {
		RemoveReferencedSlab(context, storable)
	}
	return v
}

func (v PathValue) Clone(_ ValueCloneContext) Value {
	return v
}

func (PathValue) DeepRemove(_ ValueRemoveContext, _ bool) {
	// NO-OP
}

func (v PathValue) ByteSize() uint32 {
	// tag number (2 bytes) + array head (1 byte) + domain (CBOR uint) + identifier (CBOR string)
	return values.CBORTagSize +
		1 +
		values.GetUintCBORSize(uint64(v.Domain)) +
		values.GetBytesCBORSize([]byte(v.Identifier))
}

func (v PathValue) StoredValue(_ atree.SlabStorage) (atree.Value, error) {
	return v, nil
}

func (PathValue) ChildStorables() []atree.Storable {
	return nil
}
