/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2021 Dapper Labs, Inc.
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
	"fmt"

	"github.com/fxamacker/atree"
	"github.com/onflow/cadence/runtime/common"
)

// Cadence needs to encode different kinds of objects in CBOR, for instance,
// dictionaries, structs, resources, etc.
//
// However, CBOR only provides one native map type, and no support
// for directly representing e.g. structs or resources.
//
// To be able to encode/decode such semantically different values,
// we define custom CBOR tags.

// !!! *WARNING* !!!
//
// Only add new fields to encoded structs by
// appending new fields with the next highest key.
//
// DO *NOT* REPLACE EXISTING FIELDS!

const CBORTagBase = 128

// !!! *WARNING* !!!
//
// Only add new types by:
// - replacing existing placeholders (`_`) with new types
// - appending new types
//
// Only remove types by:
// - replace existing types with a placeholder `_`
//
// DO *NOT* REPLACE EXISTING TYPES!
// DO *NOT* ADD NEW TYPES IN BETWEEN!

const (
	CBORTagVoidValue = CBORTagBase + iota
	CBORTagDictionaryValue
	CBORTagSomeValue
	CBORTagAddressValue
	CBORTagCompositeValue
	CBORTagTypeValue
	CBORRTagArrayValue
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_

	// Int*
	CBORTagIntValue
	CBORTagInt8Value
	CBORTagInt16Value
	CBORTagInt32Value
	CBORTagInt64Value
	CBORTagInt128Value
	CBORTagInt256Value
	_

	// UInt*
	CBORTagUIntValue
	CBORTagUInt8Value
	CBORTagUInt16Value
	CBORTagUInt32Value
	CBORTagUInt64Value
	CBORTagUInt128Value
	CBORTagUInt256Value
	_

	// Word*
	_
	CBORTagWord8Value
	CBORTagWord16Value
	CBORTagWord32Value
	CBORTagWord64Value
	_ // future: Word128
	_ // future: Word256
	_

	// Fix*
	_
	_ // future: Fix8
	_ // future: Fix16
	_ // future: Fix32
	CBORTagFix64Value
	_ // future: Fix128
	_ // future: Fix256
	_

	// UFix*
	_
	_ // future: UFix8
	_ // future: UFix16
	_ // future: UFix32
	CBORTagUFix64Value
	_ // future: UFix128
	_ // future: UFix256
	_

	// Locations
	CBORTagAddressLocation
	CBORTagStringLocation
	CBORTagIdentifierLocation
	_
	_
	_
	_
	_

	// Storage

	CBORTagPathValue
	CBORTagCapabilityValue
	_ // DO NOT REPLACE! used to be used for storage references
	CBORTagLinkValue
	_
	_
	_
	_
	_
	_
	_
	_

	// Static Types
	CBORTagPrimitiveStaticType
	CBORTagCompositeStaticType
	CBORTagInterfaceStaticType
	CBORTagVariableSizedStaticType
	CBORTagConstantSizedStaticType
	CBORTagDictionaryStaticType
	CBORTagOptionalStaticType
	CBORTagReferenceStaticType
	CBORTagRestrictedStaticType
	CBORTagCapabilityStaticType
)

// Encode encodes the value as a CBOR nil
//
func (v NilValue) Encode(e *atree.Encoder) error {
	return e.CBOR.EncodeNil()
}

// Encode encodes the value as a CBOR bool
//
func (v BoolValue) Encode(e *atree.Encoder) error {
	return e.CBOR.EncodeBool(bool(v))
}

// Encode encodes the value as a CBOR string
//
func (v *StringValue) Encode(e *atree.Encoder) error {
	return e.CBOR.EncodeString(v.Str)
}

// cborVoidValue represents the CBOR value:
//
// 	cbor.Tag{
// 		Number: CBORTagVoidValue,
// 		Content: nil
// 	}
//
var cborVoidValue = []byte{
	// tag
	0xd8, CBORTagVoidValue,
	// null
	0xf6,
}

// Encode writes a value of type Void to the encoder
//
func (v VoidValue) Encode(e *atree.Encoder) error {

	// TODO: optimize: use 0xf7, but decoded by github.com/fxamacker/cbor/v2 as Go `nil`:
	//   https://github.com/fxamacker/cbor/blob/a6ed6ff68e99cbb076997a08d19f03c453851555/README.md#limitations

	return e.CBOR.EncodeRawBytes(cborVoidValue)
}

// Encode encodes the value as
// cbor.Tag{
//		Number:  CBORTagIntValue,
//		Content: *big.Int(v.BigInt),
// }
func (v IntValue) Encode(e *atree.Encoder) error {
	err := e.CBOR.EncodeRawBytes([]byte{
		// tag number
		0xd8, CBORTagIntValue,
	})
	if err != nil {
		return err
	}
	return e.CBOR.EncodeBigInt(v.BigInt)
}

// Encode encodes Int8Value as
// cbor.Tag{
//		Number:  CBORTagInt8Value,
//		Content: int8(v),
// }
func (v Int8Value) Encode(e *atree.Encoder) error {
	err := e.CBOR.EncodeRawBytes([]byte{
		// tag number
		0xd8, CBORTagInt8Value,
	})
	if err != nil {
		return err
	}
	return e.CBOR.EncodeInt8(int8(v))
}

// Encode encodes Int16Value as
// cbor.Tag{
//		Number:  CBORTagInt16Value,
//		Content: int16(v),
// }
func (v Int16Value) Encode(e *atree.Encoder) error {
	err := e.CBOR.EncodeRawBytes([]byte{
		// tag number
		0xd8, CBORTagInt16Value,
	})
	if err != nil {
		return err
	}
	return e.CBOR.EncodeInt16(int16(v))
}

// Encode encodes Int32Value as
// cbor.Tag{
//		Number:  CBORTagInt32Value,
//		Content: int32(v),
// }
func (v Int32Value) Encode(e *atree.Encoder) error {
	err := e.CBOR.EncodeRawBytes([]byte{
		// tag number
		0xd8, CBORTagInt32Value,
	})
	if err != nil {
		return err
	}
	return e.CBOR.EncodeInt32(int32(v))
}

// Encode encodes Int64Value as
// cbor.Tag{
//		Number:  CBORTagInt64Value,
//		Content: int64(v),
// }
func (v Int64Value) Encode(e *atree.Encoder) error {
	err := e.CBOR.EncodeRawBytes([]byte{
		// tag number
		0xd8, CBORTagInt64Value,
	})
	if err != nil {
		return err
	}
	return e.CBOR.EncodeInt64(int64(v))
}

// Encode encodes Int128Value as
// cbor.Tag{
//		Number:  CBORTagInt128Value,
//		Content: *big.Int(v.BigInt),
// }
func (v Int128Value) Encode(e *atree.Encoder) error {
	err := e.CBOR.EncodeRawBytes([]byte{
		// tag number
		0xd8, CBORTagInt128Value,
	})
	if err != nil {
		return err
	}
	return e.CBOR.EncodeBigInt(v.BigInt)
}

// Encode encodes Int256Value as
// cbor.Tag{
//		Number:  CBORTagInt256Value,
//		Content: *big.Int(v.BigInt),
// }
func (v Int256Value) Encode(e *atree.Encoder) error {
	err := e.CBOR.EncodeRawBytes([]byte{
		// tag number
		0xd8, CBORTagInt256Value,
	})
	if err != nil {
		return err
	}
	return e.CBOR.EncodeBigInt(v.BigInt)
}

// Encode encodes UIntValue as
// cbor.Tag{
//		Number:  CBORTagUIntValue,
//		Content: *big.Int(v.BigInt),
// }
func (v UIntValue) Encode(e *atree.Encoder) error {
	err := e.CBOR.EncodeRawBytes([]byte{
		// tag number
		0xd8, CBORTagUIntValue,
	})
	if err != nil {
		return err
	}
	return e.CBOR.EncodeBigInt(v.BigInt)
}

// Encode encodes UInt8Value as
// cbor.Tag{
//		Number:  CBORTagUInt8Value,
//		Content: uint8(v),
// }
func (v UInt8Value) Encode(e *atree.Encoder) error {
	err := e.CBOR.EncodeRawBytes([]byte{
		// tag number
		0xd8, CBORTagUInt8Value,
	})
	if err != nil {
		return err
	}
	return e.CBOR.EncodeUint8(uint8(v))
}

// Encode encodes UInt16Value as
// cbor.Tag{
//		Number:  CBORTagUInt16Value,
//		Content: uint16(v),
// }
func (v UInt16Value) Encode(e *atree.Encoder) error {
	err := e.CBOR.EncodeRawBytes([]byte{
		// tag number
		0xd8, CBORTagUInt16Value,
	})
	if err != nil {
		return err
	}
	return e.CBOR.EncodeUint16(uint16(v))
}

// Encode encodes UInt32Value as
// cbor.Tag{
//		Number:  CBORTagUInt32Value,
//		Content: uint32(v),
// }
func (v UInt32Value) Encode(e *atree.Encoder) error {
	err := e.CBOR.EncodeRawBytes([]byte{
		// tag number
		0xd8, CBORTagUInt32Value,
	})
	if err != nil {
		return err
	}
	return e.CBOR.EncodeUint32(uint32(v))
}

// Encode encodes UInt64Value as
// cbor.Tag{
//		Number:  CBORTagUInt64Value,
//		Content: uint64(v),
// }
func (v UInt64Value) Encode(e *atree.Encoder) error {
	err := e.CBOR.EncodeRawBytes([]byte{
		// tag number
		0xd8, CBORTagUInt64Value,
	})
	if err != nil {
		return err
	}
	return e.CBOR.EncodeUint64(uint64(v))
}

// Encode encodes UInt128Value as
// cbor.Tag{
//		Number:  CBORTagUInt128Value,
//		Content: *big.Int(v.BigInt),
// }
func (v UInt128Value) Encode(e *atree.Encoder) error {
	err := e.CBOR.EncodeRawBytes([]byte{
		// tag number
		0xd8, CBORTagUInt128Value,
	})
	if err != nil {
		return err
	}
	return e.CBOR.EncodeBigInt(v.BigInt)
}

// Encode encodes UInt256Value as
// cbor.Tag{
//		Number:  CBORTagUInt256Value,
//		Content: *big.Int(v.BigInt),
// }
func (v UInt256Value) Encode(e *atree.Encoder) error {
	err := e.CBOR.EncodeRawBytes([]byte{
		// tag number
		0xd8, CBORTagUInt256Value,
	})
	if err != nil {
		return err
	}
	return e.CBOR.EncodeBigInt(v.BigInt)
}

// Encode encodes Word8Value as
// cbor.Tag{
//		Number:  CBORTagWord8Value,
//		Content: uint8(v),
// }
func (v Word8Value) Encode(e *atree.Encoder) error {
	err := e.CBOR.EncodeRawBytes([]byte{
		// tag number
		0xd8, CBORTagWord8Value,
	})
	if err != nil {
		return err
	}
	return e.CBOR.EncodeUint8(uint8(v))
}

// Encode encodes Word16Value as
// cbor.Tag{
//		Number:  CBORTagWord16Value,
//		Content: uint16(v),
// }
func (v Word16Value) Encode(e *atree.Encoder) error {
	err := e.CBOR.EncodeRawBytes([]byte{
		// tag number
		0xd8, CBORTagWord16Value,
	})
	if err != nil {
		return err
	}
	return e.CBOR.EncodeUint16(uint16(v))
}

// Encode encodes Word32Value as
// cbor.Tag{
//		Number:  CBORTagWord32Value,
//		Content: uint32(v),
// }
func (v Word32Value) Encode(e *atree.Encoder) error {
	err := e.CBOR.EncodeRawBytes([]byte{
		// tag number
		0xd8, CBORTagWord32Value,
	})
	if err != nil {
		return err
	}
	return e.CBOR.EncodeUint32(uint32(v))
}

// Encode encodes Word64Value as
// cbor.Tag{
//		Number:  CBORTagWord64Value,
//		Content: uint64(v),
// }
func (v Word64Value) Encode(e *atree.Encoder) error {
	err := e.CBOR.EncodeRawBytes([]byte{
		// tag number
		0xd8, CBORTagWord64Value,
	})
	if err != nil {
		return err
	}
	return e.CBOR.EncodeUint64(uint64(v))
}

// Encode encodes Fix64Value as
// cbor.Tag{
//		Number:  CBORTagFix64Value,
//		Content: int64(v),
// }
func (v Fix64Value) Encode(e *atree.Encoder) error {
	err := e.CBOR.EncodeRawBytes([]byte{
		// tag number
		0xd8, CBORTagFix64Value,
	})
	if err != nil {
		return err
	}
	return e.CBOR.EncodeInt64(int64(v))
}

// Encode encodes UFix64Value as
// cbor.Tag{
//		Number:  CBORTagUFix64Value,
//		Content: uint64(v),
// }
func (v UFix64Value) Encode(e *atree.Encoder) error {
	err := e.CBOR.EncodeRawBytes([]byte{
		// tag number
		0xd8, CBORTagUFix64Value,
	})
	if err != nil {
		return err
	}
	return e.CBOR.EncodeUint64(uint64(v))
}

// NOTE: NEVER change, only add/increment; ensure uint64
const (
	// encodedDictionaryValueTypeFieldKeyV6    uint64 = 0
	// encodedDictionaryValueKeysFieldKeyV6    uint64 = 1
	// encodedDictionaryValueEntriesFieldKeyV6 uint64 = 2

	// !!! *WARNING* !!!
	//
	// encodedDictionaryValueLength MUST be updated when new element is added.
	// It is used to verify encoded dictionaries length during decoding.
	encodedDictionaryValueLength = 3
)

// Encode encodes DictionaryValue as
// cbor.Tag{
//			Number: CBORTagDictionaryValue,
//			Content: cborArray{
// 				encodedDictionaryValueTypeFieldKeyV6:    []interface{}(type),
//				encodedDictionaryValueKeysFieldKeyV6:    []interface{}(keys),
//				encodedDictionaryValueEntriesFieldKeyV6: []interface{}(entries),
//			},
// }
func (v *DictionaryValue) Encode(e *atree.Encoder) error {
	// TODO:
	//	// Encode CBOR tag number
	//	err := e.CBOR.EncodeRawBytes([]byte{
	//		// tag number
	//		0xd8, CBORTagDictionaryValue,
	//	})
	//	if err != nil {
	//		return err
	//	}
	//
	//	if v.content != nil {
	//		err := e.CBOR.EncodeRawBytes(v.content)
	//		if err != nil {
	//			return err
	//		}
	//
	//		return nil
	//	}
	//
	//	// Encode array head
	//	err = e.CBOR.EncodeRawBytes([]byte{
	//		// array, 3 items follow
	//		0x83,
	//	})
	//	if err != nil {
	//		return err
	//	}
	//
	//	//nolint:gocritic
	//	keysPath := append(path, dictionaryKeyPathPrefix)
	//
	//	// (1) Encode dictionary static type at array index encodedDictionaryValueTypeFieldKeyV6
	//	err = e.encodeStaticType(v.StaticType())
	//	if err != nil {
	//		return err
	//	}
	//
	//	// (2) Encode keys (as array) at array index encodedDictionaryValueKeysFieldKeyV6
	//	err = e.encodeArray(v.Keys(), keysPath, deferrals)
	//	if err != nil {
	//		return err
	//	}
	//
	//	// Deferring the encoding of values is only supported if all
	//	// values are resources: resource typed dictionaries are moved
	//
	//	deferred := e.deferred
	//	if deferred {
	//
	//		// Iterating over the map in a non-deterministic way is OK,
	//		// we only determine check if all values are resources.
	//
	//		for pair := v.Entries().Oldest(); pair != nil; pair = pair.Next() {
	//			compositeValue, ok := pair.Value.(*CompositeValue)
	//			if !ok || compositeValue.Kind() != common.CompositeKindResource {
	//				deferred = false
	//				break
	//			}
	//		}
	//	}
	//
	//	// entries is empty if encoding of values is deferred,
	//	// otherwise entries size is the same as keys size.
	//	keys := v.Keys().Elements()
	//	entriesLength := len(keys)
	//	if deferred {
	//		entriesLength = 0
	//	}
	//
	//	// (3) Encode values (as array) at array index encodedDictionaryValueEntriesFieldKeyV6
	//	err = e.CBOR.EncodeArrayHead(uint64(entriesLength))
	//	if err != nil {
	//		return err
	//	}
	//
	//	// Pre-allocate and reuse valuePath.
	//	//nolint:gocritic
	//	valuePath := append(path, dictionaryValuePathPrefix, "")
	//
	//	lastValuePathIndex := len(path) + 1
	//
	//	for _, keyValue := range keys {
	//		key := dictionaryKey(keyValue)
	//		entryValue, _ := v.Entries().Get(key)
	//		valuePath[lastValuePathIndex] = key
	//
	//		if deferred {
	//
	//			var isDeferred bool
	//			if v.deferredKeys != nil {
	//				_, isDeferred = v.deferredKeys.Get(key)
	//			}
	//
	//			// If the value is not deferred, i.e. it is in memory,
	//			// then it must be stored under a separate storage key
	//			// in the owner's storage.
	//
	//			if !isDeferred {
	//				deferrals.Values = append(deferrals.Values,
	//					EncodingDeferralValue{
	//						Key:   joinPath(valuePath),
	//						Value: entryValue,
	//					},
	//				)
	//			} else {
	//
	//				// If the value is deferred, and the deferred value
	//				// is stored in another account's storage,
	//				// it must be moved.
	//
	//				deferredOwner := *v.deferredOwner
	//				owner := *v.Owner
	//
	//				if deferredOwner != owner {
	//
	//					deferredStorageKey := joinPathElements(v.deferredStorageKeyBase, key)
	//
	//					deferrals.Moves = append(deferrals.Moves,
	//						EncodingDeferralMove{
	//							DeferredOwner:      deferredOwner,
	//							DeferredStorageKey: deferredStorageKey,
	//							NewOwner:           owner,
	//							NewStorageKey:      joinPath(valuePath),
	//						},
	//					)
	//				}
	//			}
	//		} else {
	//			// Encode value as element in values array
	//			err = e.Encode(entryValue, valuePath, deferrals)
	//			if err != nil {
	//				return err
	//			}
	//		}
	//	}
	//
	return nil
}

// NOTE: NEVER change, only add/increment; ensure uint64
const (
	// encodedCompositeValueLocationFieldKeyV6            uint64 = 0
	// encodedCompositeValueKindFieldKeyV6                uint64 = 1
	// encodedCompositeValueFieldsFieldKeyV6              uint64 = 2
	// encodedCompositeValueQualifiedIdentifierFieldKeyV6 uint64 = 3

	// !!! *WARNING* !!!
	//
	// encodedCompositeValueLength MUST be updated when new element is added.
	// It is used to verify encoded composites length during decoding.
	encodedCompositeValueLength = 4
)

// Encode encodes CompositeValue as
// cbor.Tag{
//		Number: CBORTagCompositeValue,
//		Content: cborArray{
//			encodedCompositeValueLocationFieldKeyV6:            common.Location(location),
//			encodedCompositeValueKindFieldKeyV6:                uint(v.Kind),
//			encodedCompositeValueFieldsFieldKeyV6:              []interface{}(fields),
//			encodedCompositeValueQualifiedIdentifierFieldKeyV6: string(v.QualifiedIdentifier),
//		},
// }
func (v *CompositeValue) Encode(e *atree.Encoder) error {

	// Encode CBOR tag number
	err := e.CBOR.EncodeRawBytes([]byte{
		// tag number
		0xd8, CBORTagCompositeValue,
	})

	if err != nil {
		return err
	}

	// Encode array head
	err = e.CBOR.EncodeRawBytes([]byte{
		// array, 4 items follow
		0x84,
	})
	if err != nil {
		return err
	}

	// Encode location at array index encodedCompositeValueLocationFieldKeyV6
	err = EncodeLocation(e, v.Location)
	if err != nil {
		return err
	}

	// Encode kind at array index encodedCompositeValueKindFieldKeyV6
	err = e.CBOR.EncodeUint(uint(v.Kind))
	if err != nil {
		return err
	}

	// Encode fields (as array) at array index encodedCompositeValueFieldsFieldKeyV6

	fields := v.Fields
	err = e.CBOR.EncodeArrayHead(uint64(fields.Len() * 2))
	if err != nil {
		return err
	}

	for pair := fields.Oldest(); pair != nil; pair = pair.Next() {
		fieldName := pair.Key

		// Encode field name as fields array element
		err := e.CBOR.EncodeString(fieldName)
		if err != nil {
			return err
		}

		value := pair.Value

		// Encode value as fields array element
		err = value.Storable(e.Storage).Encode(e)
		if err != nil {
			return err
		}
	}

	// Encode qualified identifier at array index encodedCompositeValueQualifiedIdentifierFieldKeyV6
	err = e.CBOR.EncodeString(v.QualifiedIdentifier)
	if err != nil {
		return err
	}

	return nil
}

// Encode encodes SomeValue as
// cbor.Tag{
//		Number: CBORTagSomeValue,
//		Content: Value(v.Value),
// }
func (v *SomeValue) Encode(e *atree.Encoder) error {
	err := e.CBOR.EncodeRawBytes([]byte{
		// tag number
		0xd8, CBORTagSomeValue,
	})
	if err != nil {
		return err
	}
	return v.Value.Storable(e.Storage).Encode(e)
}

// Encode encodes AddressValue as
// cbor.Tag{
//		Number:  CBORTagAddressValue,
//		Content: []byte(v.ToAddress().Bytes()),
// }
func (v AddressValue) Encode(e *atree.Encoder) error {
	err := e.CBOR.EncodeRawBytes([]byte{
		// tag number
		0xd8, CBORTagAddressValue,
	})
	if err != nil {
		return err
	}
	return e.CBOR.EncodeBytes(v.ToAddress().Bytes())
}

// NOTE: NEVER change, only add/increment; ensure uint64
const (
	// encodedPathValueDomainFieldKeyV6     uint64 = 0
	// encodedPathValueIdentifierFieldKeyV6 uint64 = 1

	// !!! *WARNING* !!!
	//
	// encodedPathValueLength MUST be updated when new element is added.
	// It is used to verify encoded path length during decoding.
	encodedPathValueLength = 2
)

// Encode encodes PathValue as
// cbor.Tag{
//			Number: CBORTagPathValue,
//			Content: []interface{}{
//				encodedPathValueDomainFieldKeyV6:     uint(v.Domain),
//				encodedPathValueIdentifierFieldKeyV6: string(v.Identifier),
//			},
// }
func (v PathValue) Encode(e *atree.Encoder) error {
	// Encode tag number and array head
	err := e.CBOR.EncodeRawBytes([]byte{
		// tag number
		0xd8, CBORTagPathValue,
		// array, 2 items follow
		0x82,
	})
	if err != nil {
		return err
	}

	// Encode domain at array index encodedPathValueDomainFieldKeyV6
	err = e.CBOR.EncodeUint(uint(v.Domain))
	if err != nil {
		return err
	}

	// Encode identifier at array index encodedPathValueIdentifierFieldKeyV6
	return e.CBOR.EncodeString(v.Identifier)
}

// NOTE: NEVER change, only add/increment; ensure uint64
const (
	// encodedCapabilityValueAddressFieldKeyV6    uint64 = 0
	// encodedCapabilityValuePathFieldKeyV6       uint64 = 1
	// encodedCapabilityValueBorrowTypeFieldKeyV6 uint64 = 2

	// !!! *WARNING* !!!
	//
	// encodedCapabilityValueLength MUST be updated when new element is added.
	// It is used to verify encoded capability length during decoding.
	encodedCapabilityValueLength = 3
)

// Encode encodes CapabilityValue as
// cbor.Tag{
//			Number: CBORTagCapabilityValue,
//			Content: []interface{}{
//					encodedCapabilityValueAddressFieldKeyV6:    AddressValue(v.Address),
// 					encodedCapabilityValuePathFieldKeyV6:       PathValue(v.Path),
// 					encodedCapabilityValueBorrowTypeFieldKeyV6: StaticType(v.BorrowType),
// 				},
// }
func (v CapabilityValue) Encode(e *atree.Encoder) error {
	// Encode tag number and array head
	err := e.CBOR.EncodeRawBytes([]byte{
		// tag number
		0xd8, CBORTagCapabilityValue,
		// array, 3 items follow
		0x83,
	})
	if err != nil {
		return err
	}

	// Encode address at array index encodedCapabilityValueAddressFieldKeyV6
	err = v.Address.Encode(e)
	if err != nil {
		return err
	}

	// Encode path at array index encodedCapabilityValuePathFieldKeyV6
	err = v.Path.Encode(e)
	if err != nil {
		return err
	}

	// Encode borrow type at array index encodedCapabilityValueBorrowTypeFieldKeyV6
	return EncodeStaticType(e, v.BorrowType)
}

// NOTE: NEVER change, only add/increment; ensure uint64
const (
	// encodedAddressLocationAddressFieldKeyV6 uint64 = 0
	// encodedAddressLocationNameFieldKeyV6    uint64 = 1

	// !!! *WARNING* !!!
	//
	// encodedAddressLocationLength MUST be updated when new element is added.
	// It is used to verify encoded address location length during decoding.
	encodedAddressLocationLength = 2
)

func EncodeLocation(e *atree.Encoder, l common.Location) error {
	switch l := l.(type) {

	case common.StringLocation:
		// common.StringLocation is encoded as
		// cbor.Tag{
		//		Number:  CBORTagStringLocation,
		//		Content: string(l),
		// }
		err := e.CBOR.EncodeRawBytes([]byte{
			// tag number
			0xd8, CBORTagStringLocation,
		})
		if err != nil {
			return err
		}
		return e.CBOR.EncodeString(string(l))

	case common.IdentifierLocation:
		// common.IdentifierLocation is encoded as
		// cbor.Tag{
		//		Number:  CBORTagIdentifierLocation,
		//		Content: string(l),
		// }
		err := e.CBOR.EncodeRawBytes([]byte{
			// tag number
			0xd8, CBORTagIdentifierLocation,
		})
		if err != nil {
			return err
		}
		return e.CBOR.EncodeString(string(l))

	case common.AddressLocation:
		// common.AddressLocation is encoded as
		// cbor.Tag{
		//		Number: CBORTagAddressLocation,
		//		Content: []interface{}{
		//			encodedAddressLocationAddressFieldKeyV6: []byte{l.Address.Bytes()},
		//			encodedAddressLocationNameFieldKeyV6:    string(l.Name),
		//		},
		// }
		// Encode tag number and array head
		err := e.CBOR.EncodeRawBytes([]byte{
			// tag number
			0xd8, CBORTagAddressLocation,
			// array, 2 items follow
			0x82,
		})
		if err != nil {
			return err
		}
		// Encode address at array index encodedAddressLocationAddressFieldKeyV6
		err = e.CBOR.EncodeBytes(l.Address.Bytes())
		if err != nil {
			return err
		}
		// Encode name at array index encodedAddressLocationNameFieldKeyV6
		return e.CBOR.EncodeString(l.Name)

	default:
		return fmt.Errorf("unsupported location: %T", l)
	}
}

// NOTE: NEVER change, only add/increment; ensure uint64
const (
	// encodedLinkValueTargetPathFieldKeyV6 uint64 = 0
	// encodedLinkValueTypeFieldKeyV6       uint64 = 1

	// !!! *WARNING* !!!
	//
	// encodedLinkValueLength MUST be updated when new element is added.
	// It is used to verify encoded link length during decoding.
	encodedLinkValueLength = 2
)

// Encode encodes LinkValue as
// cbor.Tag{
//			Number: CBORTagLinkValue,
//			Content: []interface{}{
//				encodedLinkValueTargetPathFieldKeyV6: PathValue(v.TargetPath),
//				encodedLinkValueTypeFieldKeyV6:       StaticType(v.Type),
//			},
// }
func (v LinkValue) Encode(e *atree.Encoder) error {
	// Encode tag number and array head
	err := e.CBOR.EncodeRawBytes([]byte{
		// tag number
		0xd8, CBORTagLinkValue,
		// array, 2 items follow
		0x82,
	})
	if err != nil {
		return err
	}
	// Encode path at array index encodedLinkValueTargetPathFieldKeyV6
	err = v.TargetPath.Encode(e)
	if err != nil {
		return err
	}
	// Encode type at array index encodedLinkValueTypeFieldKeyV6
	return EncodeStaticType(e, v.Type)
}

func EncodeStaticType(e *atree.Encoder, t StaticType) error {
	if t == nil {
		return e.CBOR.EncodeNil()
	}

	return t.Encode(e)
}

// Encode encodes PrimitiveStaticType as
// cbor.Tag{
//		Number:  CBORTagPrimitiveStaticType,
//		Content: uint(v),
// }
func (t PrimitiveStaticType) Encode(e *atree.Encoder) error {
	err := e.CBOR.EncodeRawBytes([]byte{
		// tag number
		0xd8, CBORTagPrimitiveStaticType,
	})
	if err != nil {
		return err
	}
	return e.CBOR.EncodeUint(uint(t))
}

// Encode encodes OptionalStaticType as
// cbor.Tag{
//		Number:  CBORTagOptionalStaticType,
//		Content: StaticType(v.Type),
// }
func (t OptionalStaticType) Encode(e *atree.Encoder) error {
	err := e.CBOR.EncodeRawBytes([]byte{
		// tag number
		0xd8, CBORTagOptionalStaticType,
	})
	if err != nil {
		return err
	}
	return EncodeStaticType(e, t.Type)
}

// NOTE: NEVER change, only add/increment; ensure uint64
const (
	// encodedCompositeStaticTypeLocationFieldKeyV6            uint64 = 0
	// encodedCompositeStaticTypeQualifiedIdentifierFieldKeyV6 uint64 = 1

	// !!! *WARNING* !!!
	//
	// encodedCompositeStaticTypeLength MUST be updated when new element is added.
	// It is used to verify encoded composite static type length during decoding.
	encodedCompositeStaticTypeLength = 2
)

// Encode encodes CompositeStaticType as
// cbor.Tag{
//			Number: CBORTagCompositeStaticType,
// 			Content: cborArray{
//				encodedCompositeStaticTypeLocationFieldKeyV6:            Location(v.Location),
//				encodedCompositeStaticTypeQualifiedIdentifierFieldKeyV6: string(v.QualifiedIdentifier),
//		},
// }
func (t CompositeStaticType) Encode(e *atree.Encoder) error {
	// Encode tag number and array head
	err := e.CBOR.EncodeRawBytes([]byte{
		// tag number
		0xd8, CBORTagCompositeStaticType,
		// array, 2 items follow
		0x82,
	})
	if err != nil {
		return err
	}

	// Encode location at array index encodedCompositeStaticTypeLocationFieldKeyV6
	err = EncodeLocation(e, t.Location)
	if err != nil {
		return err
	}

	// Encode qualified identifier at array index encodedCompositeStaticTypeQualifiedIdentifierFieldKeyV6
	return e.CBOR.EncodeString(t.QualifiedIdentifier)
}

// NOTE: NEVER change, only add/increment; ensure uint64
const (
	// encodedInterfaceStaticTypeLocationFieldKeyV6            uint64 = 0
	// encodedInterfaceStaticTypeQualifiedIdentifierFieldKeyV6 uint64 = 1

	// !!! *WARNING* !!!
	//
	// encodedInterfaceStaticTypeLength MUST be updated when new element is added.
	// It is used to verify encoded interface static type length during decoding.
	encodedInterfaceStaticTypeLength = 2
)

// Encode encodes InterfaceStaticType as
// cbor.Tag{
//		Number: CBORTagInterfaceStaticType,
//		Content: cborArray{
//				encodedInterfaceStaticTypeLocationFieldKeyV6:            Location(v.Location),
//				encodedInterfaceStaticTypeQualifiedIdentifierFieldKeyV6: string(v.QualifiedIdentifier),
//		},
// }
func (t InterfaceStaticType) Encode(e *atree.Encoder) error {
	// Encode tag number and array head
	err := e.CBOR.EncodeRawBytes([]byte{
		// tag number
		0xd8, CBORTagInterfaceStaticType,
		// array, 2 items follow
		0x82,
	})
	if err != nil {
		return err
	}

	// Encode location at array index encodedInterfaceStaticTypeLocationFieldKeyV6
	err = EncodeLocation(e, t.Location)
	if err != nil {
		return err
	}

	// Encode qualified identifier at array index encodedInterfaceStaticTypeQualifiedIdentifierFieldKeyV6
	return e.CBOR.EncodeString(t.QualifiedIdentifier)
}

// Encode encodes VariableSizedStaticType as
// cbor.Tag{
//		Number:  CBORTagVariableSizedStaticType,
//		Content: StaticType(v.Type),
// }
func (t VariableSizedStaticType) Encode(e *atree.Encoder) error {
	err := e.CBOR.EncodeRawBytes([]byte{
		// tag number
		0xd8, CBORTagVariableSizedStaticType,
	})
	if err != nil {
		return err
	}
	return EncodeStaticType(e, t.Type)
}

// NOTE: NEVER change, only add/increment; ensure uint64
const (
	// encodedConstantSizedStaticTypeSizeFieldKeyV6 uint64 = 0
	// encodedConstantSizedStaticTypeTypeFieldKeyV6 uint64 = 1

	// !!! *WARNING* !!!
	//
	// encodedConstantSizedStaticTypeLength MUST be updated when new element is added.
	// It is used to verify encoded constant sized static type length during decoding.
	encodedConstantSizedStaticTypeLength = 2
)

// Encode encodes ConstantSizedStaticType as
// cbor.Tag{
//		Number: CBORTagConstantSizedStaticType,
//		Content: cborArray{
//				encodedConstantSizedStaticTypeSizeFieldKeyV6: int64(v.Size),
//				encodedConstantSizedStaticTypeTypeFieldKeyV6: StaticType(v.Type),
//		},
// }
func (t ConstantSizedStaticType) Encode(e *atree.Encoder) error {
	// Encode tag number and array head
	err := e.CBOR.EncodeRawBytes([]byte{
		// tag number
		0xd8, CBORTagConstantSizedStaticType,
		// array, 2 items follow
		0x82,
	})
	if err != nil {
		return err
	}
	// Encode size at array index encodedConstantSizedStaticTypeSizeFieldKeyV6
	err = e.CBOR.EncodeInt64(t.Size)
	if err != nil {
		return err
	}
	// Encode type at array index encodedConstantSizedStaticTypeTypeFieldKeyV6
	return EncodeStaticType(e, t.Type)
}

// NOTE: NEVER change, only add/increment; ensure uint64
const (
	// encodedReferenceStaticTypeAuthorizedFieldKeyV6 uint64 = 0
	// encodedReferenceStaticTypeTypeFieldKeyV6       uint64 = 1

	// !!! *WARNING* !!!
	//
	// encodedReferenceStaticTypeLength MUST be updated when new element is added.
	// It is used to verify encoded reference static type length during decoding.
	encodedReferenceStaticTypeLength = 2
)

// Encode encodes ReferenceStaticType as
// cbor.Tag{
//		Number: CBORTagReferenceStaticType,
//		Content: cborArray{
//				encodedReferenceStaticTypeAuthorizedFieldKeyV6: bool(v.Authorized),
//				encodedReferenceStaticTypeTypeFieldKeyV6:       StaticType(v.Type),
//		},
//	}
func (t ReferenceStaticType) Encode(e *atree.Encoder) error {
	// Encode tag number and array head
	err := e.CBOR.EncodeRawBytes([]byte{
		// tag number
		0xd8, CBORTagReferenceStaticType,
		// array, 2 items follow
		0x82,
	})
	if err != nil {
		return err
	}
	// Encode authorized at array index encodedReferenceStaticTypeAuthorizedFieldKeyV6
	err = e.CBOR.EncodeBool(t.Authorized)
	if err != nil {
		return err
	}
	// Encode type at array index encodedReferenceStaticTypeTypeFieldKeyV6
	return EncodeStaticType(e, t.Type)
}

// NOTE: NEVER change, only add/increment; ensure uint64
const (
	// encodedDictionaryStaticTypeKeyTypeFieldKeyV6   uint64 = 0
	// encodedDictionaryStaticTypeValueTypeFieldKeyV6 uint64 = 1

	// !!! *WARNING* !!!
	//
	// encodedDictionaryStaticTypeLength MUST be updated when new element is added.
	// It is used to verify encoded dictionary static type length during decoding.
	encodedDictionaryStaticTypeLength = 2
)

// Encode encodes DictionaryStaticType as
// cbor.Tag{
//		Number: CBORTagDictionaryStaticType,
// 		Content: []interface{}{
//				encodedDictionaryStaticTypeKeyTypeFieldKeyV6:   StaticType(v.KeyType),
//				encodedDictionaryStaticTypeValueTypeFieldKeyV6: StaticType(v.ValueType),
//		},
// }
func (t DictionaryStaticType) Encode(e *atree.Encoder) error {
	// Encode tag number and array head
	err := e.CBOR.EncodeRawBytes([]byte{
		// tag number
		0xd8, CBORTagDictionaryStaticType,
		// array, 2 items follow
		0x82,
	})
	if err != nil {
		return err
	}
	// Encode key type at array index encodedDictionaryStaticTypeKeyTypeFieldKeyV6
	err = EncodeStaticType(e, t.KeyType)
	if err != nil {
		return err
	}
	// Encode value type at array index encodedDictionaryStaticTypeValueTypeFieldKeyV6
	return EncodeStaticType(e, t.ValueType)
}

// NOTE: NEVER change, only add/increment; ensure uint64
const (
	// encodedRestrictedStaticTypeTypeFieldKeyV6         uint64 = 0
	// encodedRestrictedStaticTypeRestrictionsFieldKeyV6 uint64 = 1

	// !!! *WARNING* !!!
	//
	// encodedRestrictedStaticTypeLength MUST be updated when new element is added.
	// It is used to verify encoded restricted static type length during decoding.
	encodedRestrictedStaticTypeLength = 2
)

// Encode encodes RestrictedStaticType as
// cbor.Tag{
//		Number: CBORTagRestrictedStaticType,
//		Content: cborArray{
//				encodedRestrictedStaticTypeTypeFieldKeyV6:         StaticType(v.Type),
//				encodedRestrictedStaticTypeRestrictionsFieldKeyV6: []interface{}(v.Restrictions),
//		},
// }
func (t *RestrictedStaticType) Encode(e *atree.Encoder) error {
	// Encode tag number and array head
	err := e.CBOR.EncodeRawBytes([]byte{
		// tag number
		0xd8, CBORTagRestrictedStaticType,
		// array, 2 items follow
		0x82,
	})
	if err != nil {
		return err
	}
	// Encode type at array index encodedRestrictedStaticTypeTypeFieldKeyV6
	err = EncodeStaticType(e, t.Type)
	if err != nil {
		return err
	}
	// Encode restrictions (as array) at array index encodedRestrictedStaticTypeRestrictionsFieldKeyV6
	err = e.CBOR.EncodeArrayHead(uint64(len(t.Restrictions)))
	if err != nil {
		return err
	}
	for _, restriction := range t.Restrictions {
		// Encode restriction as array restrictions element
		err = restriction.Encode(e)
		if err != nil {
			return err
		}
	}
	return nil
}

// NOTE: NEVER change, only add/increment; ensure uint64
const (
	// encodedTypeValueTypeFieldKeyV6 uint64 = 0

	// !!! *WARNING* !!!
	//
	// encodedTypeValueTypeLength MUST be updated when new element is added.
	// It is used to verify encoded type length during decoding.
	encodedTypeValueTypeLength = 1
)

// Encode encodes TypeValue as
// cbor.Tag{
//			Number: CBORTagTypeValue,
//			Content: cborArray{
//				encodedTypeValueTypeFieldKeyV6: StaticType(v.Type),
//			},
//	}
func (v TypeValue) Encode(e *atree.Encoder) error {
	// Encode tag number and array head
	err := e.CBOR.EncodeRawBytes([]byte{
		// tag number
		0xd8, CBORTagTypeValue,
		// array, 1 item follow
		0x81,
	})
	if err != nil {
		return err
	}

	// Encode type at array index encodedTypeValueTypeFieldKeyV6
	return EncodeStaticType(e, v.Type)
}

// Encode encodes CapabilityStaticType as
// cbor.Tag{
//		Number:  CBORTagCapabilityStaticType,
//		Content: StaticType(v.BorrowType),
// }
func (v CapabilityStaticType) Encode(e *atree.Encoder) error {
	err := e.CBOR.EncodeRawBytes([]byte{
		// tag number
		0xd8, CBORTagCapabilityStaticType,
	})
	if err != nil {
		return err
	}
	return EncodeStaticType(e, v.BorrowType)
}
