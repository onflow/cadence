package sema_codec

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/sema"
)

//
// Sema
//

// A SemaDecoder decodes custom-encoded representations of Cadence values.
type SemaDecoder struct {
	r           LocatedReader
	typeDefs    map[int]sema.Type
	memoryGauge common.MemoryGauge
}

// Decode returns a Cadence value decoded from its custom-encoded representation.
//
// This function returns an error if the bytes represent a custom encoding that
// is malformed, does not conform to the custom Cadence specification, or contains
// an unknown composite type.
func DecodeSema(gauge common.MemoryGauge, b []byte) (sema.Type, error) {
	r := bytes.NewReader(b)
	dec := NewSemaDecoder(gauge, r)

	v, err := dec.Decode()
	if err != nil {
		return nil, err
	}

	return v, nil
}

// NewSemaDecoder initializes a SemaDecoder that will decode custom-encoded bytes from the
// given io.Reader.
func NewSemaDecoder(memoryGauge common.MemoryGauge, r io.Reader) *SemaDecoder {
	return &SemaDecoder{
		r:           LocatedReader{r: r},
		typeDefs:    map[int]sema.Type{},
		memoryGauge: memoryGauge,
	}
}

// Decode reads custom-encoded bytes from the io.Reader and decodes them to a
// Sema type. There is no assumption about the top-level Sema type so the first
// byte must specify the top-level type. Usually this will be a CompositeType.
//
// This function returns an error if the bytes represent a custom encoding that
// is malformed, does not conform to the custom specification, or contains
// an unknown composite type.
func (d *SemaDecoder) Decode() (t sema.Type, err error) {
	defer func() {
		err = capturePanic("failed to decode sema type: %w")
	}()

	return d.DecodeType()
}

func (d *SemaDecoder) DecodeElaboration() (el *sema.Elaboration, err error) {
	defer func() {
		err = capturePanic("failed to decode elaboration: %w")
	}()

	el = sema.NewElaboration(d.memoryGauge, false)

	err = DecodeMap(d, el.CompositeTypes, d.DecodeCompositeType)
	if err != nil {
		return
	}

	err = DecodeMap(d, el.InterfaceTypes, d.DecodeInterfaceType)
	return
}

func (d *SemaDecoder) DecodeType() (t sema.Type, err error) {
	typeIdentifier, err := d.DecodeTypeIdentifier()
	if err != nil {
		return
	}

	if isSimpleType(typeIdentifier) {
		t, err = EncodingToSimpleType(typeIdentifier)
	} else if isNumericType(typeIdentifier) {
		t, err = EncodingToNumericType(typeIdentifier)
	} else if isFixedPointNumericType(typeIdentifier) {
		t, err = EncodingToFixedPointNumericType(typeIdentifier)
	} else {
		switch typeIdentifier {
		case EncodedSemaOptionalType:
			t, err = d.DecodeOptionalType()
		case EncodedSemaReferenceType:
			t, err = d.DecodeReferenceType()
		case EncodedSemaCapabilityType:
			t, err = d.DecodeCapabilityType()
		case EncodedSemaVariableSizedType:
			t, err = d.DecodeVariableSizedType()
		case EncodedSemaAddressType:
			t = &sema.AddressType{}
		case EncodedSemaNilType:
			t = nil

		case EncodedSemaCompositeType:
			t, err = d.DecodeCompositeType()
		case EncodedSemaInterfaceType:
			t, err = d.DecodeInterfaceType()
		case EncodedSemaGenericType:
			t, err = d.DecodeGenericType()
		case EncodedSemaFunctionType:
			t, err = d.DecodeFunctionType()
		case EncodedSemaDictionaryType:
			t, err = d.DecodeDictionaryType()
		case EncodedSemaTransactionType:
			t, err = d.DecodeTransactionType()
		case EncodedSemaRestrictedType:
			t, err = d.DecodeRestrictedType()
		case EncodedSemaConstantSizedType:
			t, err = d.DecodeConstantSizedType()

		case EncodedSemaPointerType:
			t, err = d.EncodePointer()

		default:
			err = fmt.Errorf("unknown type identifier: %d", typeIdentifier)
		}
	}

	return
}

func (d *SemaDecoder) EncodePointer() (t sema.Type, err error) {
	bufferOffset, err := d.DecodeLength()
	if err != nil {
		return
	}

	if knownType, defined := d.typeDefs[bufferOffset]; defined {
		t = knownType
	} else {
		err = fmt.Errorf(`pointer to unknown type: %d`, bufferOffset)
	}
	return
}

func (d *SemaDecoder) DecodeCapabilityType() (ct *sema.CapabilityType, err error) {
	ct = &sema.CapabilityType{}

	d.typeDefs[d.r.location] = ct

	ct.BorrowType, err = d.DecodeType()
	return
}

func (d *SemaDecoder) DecodeRestrictedType() (rt *sema.RestrictedType, err error) {
	rt = &sema.RestrictedType{}

	d.typeDefs[d.r.location] = rt

	rt.Type, err = d.DecodeType()
	if err != nil {
		return
	}

	rt.Restrictions, err = DecodeArray(d, d.DecodeInterfaceType)
	return
}

func (d *SemaDecoder) DecodeTransactionType() (tx *sema.TransactionType, err error) {
	tx = &sema.TransactionType{}

	d.typeDefs[d.r.location] = tx

	tx.Members, err = d.DecodeStringMemberOrderedMap(tx)
	if err != nil {
		return
	}

	tx.Fields, err = DecodeArray(d, d.DecodeString)
	if err != nil {
		return
	}

	tx.PrepareParameters, err = DecodeArray(d, d.DecodeParameter)
	if err != nil {
		return
	}

	tx.Parameters, err = DecodeArray(d, d.DecodeParameter)
	return
}

func (d *SemaDecoder) DecodeReferenceType() (ref *sema.ReferenceType, err error) {
	ref = &sema.ReferenceType{}

	d.typeDefs[d.r.location] = ref

	ref.Authorized, err = d.DecodeBool()
	if err != nil {
		return
	}

	ref.Type, err = d.DecodeType()
	return
}

func (d *SemaDecoder) DecodeDictionaryType() (dict *sema.DictionaryType, err error) {
	dict = &sema.DictionaryType{}

	d.typeDefs[d.r.location] = dict

	dict.KeyType, err = d.DecodeType()
	if err != nil {
		return
	}

	dict.ValueType, err = d.DecodeType()
	return
}

func (d *SemaDecoder) DecodeFunctionType() (ft *sema.FunctionType, err error) {
	ft = &sema.FunctionType{}

	d.typeDefs[d.r.location] = ft

	ft.IsConstructor, err = d.DecodeBool()
	if err != nil {
		return
	}

	ft.TypeParameters, err = DecodeArray(d, d.DecodeTypeParameter)
	if err != nil {
		return
	}

	ft.Parameters, err = DecodeArray(d, d.DecodeParameter)
	if err != nil {
		return
	}

	ft.ReturnTypeAnnotation, err = d.DecodeTypeAnnotation()
	if err != nil {
		return
	}

	ft.RequiredArgumentCount, err = d.DecodeIntPointer()
	if err != nil {
		return
	}

	// TODO is ArgumentExpressionCheck needed?

	ft.Members, err = d.DecodeStringMemberOrderedMap(ft)
	return
}

// DecodeIntPointer always returns a new pointer.
func (d *SemaDecoder) DecodeIntPointer() (ptr *int, err error) {
	isNil, err := d.DecodeBool()
	if isNil || err != nil {
		return
	}

	i64, err := d.DecodeInt64()
	if err != nil {
		return
	}

	i := int(i64)
	ptr = &i

	return
}

func (d *SemaDecoder) DecodeVariableSizedType() (a *sema.VariableSizedType, err error) {
	a = &sema.VariableSizedType{}

	d.typeDefs[d.r.location] = a

	t, err := d.DecodeType()
	if err != nil {
		return
	}

	a.Type = t
	return
}

func (d *SemaDecoder) DecodeConstantSizedType() (a *sema.ConstantSizedType, err error) {
	a = &sema.ConstantSizedType{}

	d.typeDefs[d.r.location] = a

	t, err := d.DecodeType()
	if err != nil {
		return
	}

	size, err := d.DecodeInt64()
	if err != nil {
		return
	}

	a.Type = t
	a.Size = size
	return
}

func EncodingToNumericType(b EncodedSema) (t *sema.NumericType, err error) {
	switch b {
	case EncodedSemaNumericTypeNumberType:
		t = sema.NumberType
	case EncodedSemaNumericTypeSignedNumberType:
		t = sema.SignedNumberType
	case EncodedSemaNumericTypeIntegerType:
		t = sema.IntegerType
	case EncodedSemaNumericTypeSignedIntegerType:
		t = sema.SignedIntegerType
	case EncodedSemaNumericTypeIntType:
		t = sema.IntType
	case EncodedSemaNumericTypeInt8Type:
		t = sema.Int8Type
	case EncodedSemaNumericTypeInt16Type:
		t = sema.Int16Type
	case EncodedSemaNumericTypeInt32Type:
		t = sema.Int32Type
	case EncodedSemaNumericTypeInt64Type:
		t = sema.Int64Type
	case EncodedSemaNumericTypeInt128Type:
		t = sema.Int128Type
	case EncodedSemaNumericTypeInt256Type:
		t = sema.Int256Type
	case EncodedSemaNumericTypeUIntType:
		t = sema.UIntType
	case EncodedSemaNumericTypeUInt8Type:
		t = sema.UInt8Type
	case EncodedSemaNumericTypeUInt16Type:
		t = sema.UInt16Type
	case EncodedSemaNumericTypeUInt32Type:
		t = sema.UInt32Type
	case EncodedSemaNumericTypeUInt64Type:
		t = sema.UInt64Type
	case EncodedSemaNumericTypeUInt128Type:
		t = sema.UInt128Type
	case EncodedSemaNumericTypeUInt256Type:
		t = sema.UInt256Type
	case EncodedSemaNumericTypeWord8Type:
		t = sema.Word8Type
	case EncodedSemaNumericTypeWord16Type:
		t = sema.Word16Type
	case EncodedSemaNumericTypeWord32Type:
		t = sema.Word32Type
	case EncodedSemaNumericTypeWord64Type:
		t = sema.Word64Type
	case EncodedSemaNumericTypeFixedPointType:
		t = sema.FixedPointType
	case EncodedSemaNumericTypeSignedFixedPointType:
		t = sema.SignedFixedPointType
	default:
		err = fmt.Errorf("unknown numeric type: %d", b)
	}

	return
}

func EncodingToFixedPointNumericType(b EncodedSema) (t *sema.FixedPointNumericType, err error) {
	switch b {
	case EncodedSemaFix64Type:
		t = sema.Fix64Type
	case EncodedSemaUFix64Type:
		t = sema.UFix64Type
	default:
		err = fmt.Errorf("unknown fixed point numeric type: %d", b)
	}

	return
}

func (d *SemaDecoder) DecodeGenericType() (t *sema.GenericType, err error) {
	t = &sema.GenericType{}

	d.typeDefs[d.r.location] = t

	tp, err := d.DecodeTypeParameter()
	if err != nil {
		return
	}

	t.TypeParameter = tp
	return
}

func (d *SemaDecoder) DecodeOptionalType() (opt *sema.OptionalType, err error) {
	opt = &sema.OptionalType{}

	d.typeDefs[d.r.location] = opt

	t, err := d.DecodeType()
	if err != nil {
		return
	}

	opt.Type = t
	return
}

func (d *SemaDecoder) DecodeTypeIdentifier() (id EncodedSema, err error) {
	b, err := d.read(1)
	if err != nil {
		return
	}

	id = EncodedSema(b[0])
	return
}

func (d *SemaDecoder) DecodeCompositeKind() (kind common.CompositeKind, err error) {
	b, err := d.read(1)
	if err != nil {
		return
	}

	kind = common.CompositeKind(b[0])
	return
}

func EncodingToSimpleType(b EncodedSema) (t *sema.SimpleType, err error) {
	switch b {
	case EncodedSemaSimpleTypeAnyType:
		t = sema.AnyType
	case EncodedSemaSimpleTypeAnyResourceType:
		t = sema.AnyResourceType
	case EncodedSemaSimpleTypeAnyStructType:
		t = sema.AnyStructType
	case EncodedSemaSimpleTypeBlockType:
		t = sema.BlockType
	case EncodedSemaSimpleTypeBoolType:
		t = sema.BoolType
	case EncodedSemaSimpleTypeCharacterType:
		t = sema.CharacterType
	case EncodedSemaSimpleTypeDeployedContractType:
		t = sema.DeployedContractType
	case EncodedSemaSimpleTypeInvalidType:
		t = sema.InvalidType
	case EncodedSemaSimpleTypeMetaType:
		t = sema.MetaType
	case EncodedSemaSimpleTypeNeverType:
		t = sema.NeverType
	case EncodedSemaSimpleTypePathType:
		t = sema.PathType
	case EncodedSemaSimpleTypeStoragePathType:
		t = sema.StoragePathType
	case EncodedSemaSimpleTypeCapabilityPathType:
		t = sema.CapabilityPathType
	case EncodedSemaSimpleTypePublicPathType:
		t = sema.PublicPathType
	case EncodedSemaSimpleTypePrivatePathType:
		t = sema.PrivatePathType
	case EncodedSemaSimpleTypeStorableType:
		t = sema.StorableType
	case EncodedSemaSimpleTypeStringType:
		t = sema.StringType
	case EncodedSemaSimpleTypeVoidType:
		t = sema.VoidType
	default:
		err = fmt.Errorf("unknown simple subtype: %d", b)
	}

	return
}

func (d *SemaDecoder) DecodeCompositeType() (t *sema.CompositeType, err error) {
	t = &sema.CompositeType{}

	d.typeDefs[d.r.location] = t

	t.Location, err = d.DecodeLocation()
	if err != nil {
		return
	}

	t.Identifier, err = d.DecodeString()
	if err != nil {
		return
	}

	t.Kind, err = d.DecodeCompositeKind()
	if err != nil {
		return
	}

	t.ExplicitInterfaceConformances, err = DecodeArray(d, d.DecodeInterfaceType)
	if err != nil {
		return
	}

	t.ImplicitTypeRequirementConformances, err = DecodeArray(d, d.DecodeCompositeType)
	if err != nil {
		return
	}

	t.Members, err = d.DecodeStringMemberOrderedMap(t)
	if err != nil {
		return
	}

	t.Fields, err = DecodeArray(d, d.DecodeString)
	if err != nil {
		return
	}

	t.ConstructorParameters, err = DecodeArray(d, d.DecodeParameter)
	if err != nil {
		return
	}

	nestedTypes, err := d.DecodeStringTypeOrderedMap()
	if err != nil {
		return
	}
	t.SetNestedTypes(nestedTypes)

	containerType, err := d.DecodeType()
	if err != nil {
		return
	}
	t.SetContainerType(containerType)

	t.EnumRawType, err = d.DecodeType()
	if err != nil {
		return
	}

	hasComputedMembers, err := d.DecodeBool()
	if err != nil {
		return
	}
	t.SetHasComputedMembers(hasComputedMembers)

	t.ImportableWithoutLocation, err = d.DecodeBool()
	if err != nil {
		return
	}

	return
}

func (d *SemaDecoder) DecodeInterfaceType() (t *sema.InterfaceType, err error) {
	t = &sema.InterfaceType{}

	d.typeDefs[d.r.location] = t

	t.Location, err = d.DecodeLocation()
	if err != nil {
		return
	}

	t.Identifier, err = d.DecodeString()
	if err != nil {
		return
	}

	t.CompositeKind, err = d.DecodeCompositeKind()
	if err != nil {
		return
	}

	t.Members, err = d.DecodeStringMemberOrderedMap(t)
	if err != nil {
		return
	}

	t.Fields, err = DecodeArray(d, d.DecodeString)
	if err != nil {
		return
	}

	t.InitializerParameters, err = DecodeArray(d, d.DecodeParameter)
	if err != nil {
		return
	}

	containerType, err := d.DecodeType()
	if err != nil {
		return
	}
	t.SetContainerType(containerType)

	return
}

func (d *SemaDecoder) DecodeTypeParameter() (p *sema.TypeParameter, err error) {
	name, err := d.DecodeString()
	if err != nil {
		return
	}

	bound, err := d.DecodeType()
	if err != nil {
		return
	}

	optional, err := d.DecodeBool()
	if err != nil {
		return
	}

	p = &sema.TypeParameter{
		Name:      name,
		TypeBound: bound,
		Optional:  optional,
	}
	return
}

func (d *SemaDecoder) DecodeParameter() (parameter *sema.Parameter, err error) {
	label, err := d.DecodeString()
	if err != nil {
		return
	}

	id, err := d.DecodeString()
	if err != nil {
		return
	}

	anno, err := d.DecodeTypeAnnotation()
	if err != nil {
		return
	}

	parameter = &sema.Parameter{
		Label:          label,
		Identifier:     id,
		TypeAnnotation: anno,
	}

	return
}

func (d *SemaDecoder) DecodeStringMemberOrderedMap(containerType sema.Type) (om *sema.StringMemberOrderedMap, err error) {
	isNil, err := d.DecodeBool()
	if isNil || err != nil {
		return
	}

	length, err := d.DecodeLength()
	if err != nil {
		return
	}

	om = &sema.StringMemberOrderedMap{}

	for i := 0; i < length; i++ {
		var key string
		key, err = d.DecodeString()
		if err != nil {
			return
		}

		var member *sema.Member
		member, err = d.DecodeMember(containerType)
		if err != nil {
			return
		}

		om.Set(key, member)
	}

	return
}

func (d *SemaDecoder) DecodeStringTypeOrderedMap() (om *sema.StringTypeOrderedMap, err error) {
	isNil, err := d.DecodeBool()
	if isNil || err != nil {
		return
	}

	length, err := d.DecodeLength()
	if err != nil {
		return
	}

	om = &sema.StringTypeOrderedMap{}

	for i := 0; i < length; i++ {
		var key string
		key, err = d.DecodeString()
		if err != nil {
			return
		}

		var typ sema.Type
		typ, err = d.DecodeType()
		if err != nil {
			return
		}

		om.Set(key, typ)
	}

	return
}

func (d *SemaDecoder) DecodeMember(containerType sema.Type) (member *sema.Member, err error) {
	access, err := d.DecodeUInt64()
	if err != nil {
		return
	}

	identifier, err := d.DecodeAstIdentifier()
	if err != nil {
		return
	}

	typeAnnotation, err := d.DecodeTypeAnnotation()
	if err != nil {
		return
	}

	declarationKind, err := d.DecodeUInt64()
	if err != nil {
		return
	}

	variableKind, err := d.DecodeUInt64()
	if err != nil {
		return
	}

	argumentLabels, err := DecodeArray(d, d.DecodeString)
	if err != nil {
		return
	}

	predeclared, err := d.DecodeBool()
	if err != nil {
		return
	}

	docString, err := d.DecodeString()
	if err != nil {
		return
	}

	member = &sema.Member{
		ContainerType:         containerType,
		Access:                ast.Access(access),
		Identifier:            identifier,
		TypeAnnotation:        typeAnnotation,
		DeclarationKind:       common.DeclarationKind(declarationKind),
		VariableKind:          ast.VariableKind(variableKind),
		ArgumentLabels:        argumentLabels,
		Predeclared:           predeclared,
		IgnoreInSerialization: false, // wouldn't be encoded in the first place if true
		DocString:             docString,
	}
	return
}

func (d *SemaDecoder) DecodeAstIdentifier() (id ast.Identifier, err error) {
	identifier, err := d.DecodeString()
	if err != nil {
		return
	}

	position, err := d.DecodeAstPosition()
	if err != nil {
		return
	}

	id = ast.Identifier{
		Identifier: identifier,
		Pos:        position,
	}
	return
}

func (d *SemaDecoder) DecodeAstPosition() (pos ast.Position, err error) {
	offset, err := d.DecodeInt64()
	if err != nil {
		return
	}

	line, err := d.DecodeInt64()
	if err != nil {
		return
	}

	column, err := d.DecodeInt64()
	if err != nil {
		return
	}

	pos = ast.Position{
		Offset: int(offset),
		Line:   int(line),
		Column: int(column),
	}
	return
}

func (d *SemaDecoder) DecodeTypeAnnotation() (anno *sema.TypeAnnotation, err error) {
	isNil, err := d.DecodeBool()
	if isNil || err != nil {
		return
	}

	isResource, err := d.DecodeBool()
	if err != nil {
		return
	}

	t, err := d.DecodeType()
	if err != nil {
		return
	}

	anno = &sema.TypeAnnotation{
		IsResource: isResource,
		Type:       t,
	}
	return
}

func (d *SemaDecoder) DecodeLocation() (location common.Location, err error) {
	prefix, err := d.DecodeLocationPrefix()

	switch prefix {
	case common.AddressLocationPrefix:
		return d.DecodeAddressLocation()
	case common.IdentifierLocationPrefix:
		return d.DecodeIdentifierLocation()
	case common.ScriptLocationPrefix:
		return d.DecodeScriptLocation()
	case common.StringLocationPrefix:
		return d.DecodeStringLocation()
	case common.TransactionLocationPrefix:
		return d.DecodeTransactionLocation()
	case string(common.REPLLocationPrefix[0]):
		location = common.REPLLocation{}
	case NilLocationPrefix:
		return

	// TODO more locations
	default:
		err = fmt.Errorf("unknown location prefix: %s", prefix)
	}
	return
}

func (d *SemaDecoder) DecodeLocationPrefix() (prefix string, err error) {
	b, err := d.read(1)
	prefix = string(b)
	return
}

func (d *SemaDecoder) DecodeAddressLocation() (location common.AddressLocation, err error) {
	address, err := d.DecodeAddress()
	if err != nil {
		return
	}

	name, err := d.DecodeString()
	if err != nil {
		return
	}

	location = common.NewAddressLocation(d.memoryGauge, address, name)

	return
}

func (d *SemaDecoder) DecodeIdentifierLocation() (location common.IdentifierLocation, err error) {
	s, err := d.DecodeString()
	location = common.IdentifierLocation(s)
	return
}

func (d *SemaDecoder) DecodeScriptLocation() (location common.ScriptLocation, err error) {
	byteArray, err := d.read(len(location))
	if err != nil {
		return
	}

	for i, b := range byteArray {
		location[i] = b
	}

	return
}

func (d *SemaDecoder) DecodeStringLocation() (location common.StringLocation, err error) {
	s, err := d.DecodeString()
	location = common.StringLocation(s)
	return
}

func (d *SemaDecoder) DecodeTransactionLocation() (location common.TransactionLocation, err error) {
	byteArray, err := d.read(len(location))
	if err != nil {
		return
	}

	for i, b := range byteArray {
		location[i] = b
	}

	return
}

func (d *SemaDecoder) DecodeAddress() (address common.Address, err error) {
	byteArray, err := d.read(common.AddressLength)
	if err != nil {
		return
	}

	for i, b := range byteArray {
		address[i] = b
	}

	return
}

func (d *SemaDecoder) DecodeString() (s string, err error) {
	b, err := d.DecodeBytes()
	if err != nil {
		return
	}

	s = string(b)
	return
}

func (d *SemaDecoder) DecodeBytes() (bytes []byte, err error) {
	length, err := d.DecodeLength()
	if err != nil {
		return
	}

	return d.read(length)
}

func (d *SemaDecoder) DecodeLength() (length int, err error) {
	b, err := d.read(4)
	if err != nil {
		return
	}

	asUint32 := binary.BigEndian.Uint32(b)
	length = int(asUint32)
	return
}

func (d *SemaDecoder) DecodeBool() (boolean bool, err error) {
	b, err := d.read(1)
	if err != nil {
		return
	}

	switch EncodedBool(b[0]) {
	case EncodedBoolFalse:
		boolean = false
	case EncodedBoolTrue:
		boolean = true
	default:
		err = fmt.Errorf("invalid boolean value: %d", b[0])
	}

	return
}

func (d *SemaDecoder) DecodeUInt64() (u uint64, err error) {
	err = binary.Read(&d.r, binary.BigEndian, &u)
	return
}

func (d *SemaDecoder) DecodeInt64() (i int64, err error) {
	err = binary.Read(&d.r, binary.BigEndian, &i)
	return
}

func (d *SemaDecoder) read(howManyBytes int) (b []byte, err error) {
	b = make([]byte, howManyBytes)
	_, err = d.r.Read(b)
	return
}

func DecodeArray[T any](d *SemaDecoder, decodeFn func() (T, error)) (arr []T, err error) {
	isNil, err := d.DecodeBool()
	if isNil || err != nil {
		return
	}

	length, err := d.DecodeLength()
	if err != nil {
		return
	}

	arr = make([]T, length)
	for i := 0; i < length; i++ {
		var element T
		element, err = decodeFn()
		if err != nil {
			return
		}

		arr[i] = element
	}

	return
}

func DecodeMap[V sema.Type](d *SemaDecoder, mapToPopulate map[common.TypeID]V, decodeFn func() (V, error)) (err error) {
	length, err := d.DecodeLength()
	if err != nil {
		return
	}

	for i := 0; i < length; i++ {
		var (
			k string
			v V
		)

		k, err = d.DecodeString()
		if err != nil {
			return
		}

		v, err = decodeFn()
		if err != nil {
			return
		}

		mapToPopulate[common.TypeID(k)] = v
	}

	return
}

type LocatedReader struct {
	r        io.Reader
	location int
}

func (l *LocatedReader) Read(p []byte) (n int, err error) {
	n, err = l.r.Read(p)
	l.location += n
	return
}
