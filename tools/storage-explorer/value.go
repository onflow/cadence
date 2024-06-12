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

package main

import (
	"encoding/json"
	"sort"

	jsoncdc "github.com/onflow/cadence/encoding/json"
	"github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
)

type Value interface {
	isValue()
}

// TypeOnlyValue

type FallbackValue struct {
	Type        any    `json:"type"`
	TypeString  string `json:"typeString"`
	Description string `json:"description"`
}

var _ Value = FallbackValue{}

func (FallbackValue) isValue() {}

func (v FallbackValue) MarshalJSON() ([]byte, error) {
	type Alias FallbackValue
	return json.Marshal(&struct {
		Kind string `json:"kind"`
		Alias
	}{
		Kind:  "fallback",
		Alias: (Alias)(v),
	})
}

// PrimitiveValue

type PrimitiveValue struct {
	Type        any             `json:"type"`
	TypeString  string          `json:"typeString"`
	Value       json.RawMessage `json:"value"`
	Description string          `json:"description"`
}

var _ Value = PrimitiveValue{}

func (PrimitiveValue) isValue() {}

func (v PrimitiveValue) MarshalJSON() ([]byte, error) {
	type Alias PrimitiveValue
	return json.Marshal(&struct {
		Kind string `json:"kind"`
		Alias
	}{
		Kind:  "primitive",
		Alias: (Alias)(v),
	})
}

// DictionaryValue

type DictionaryValue struct {
	Type       any             `json:"type"`
	TypeString string          `json:"typeString"`
	Keys       []DictionaryKey `json:"keys"`
}

var _ Value = DictionaryValue{}

func (DictionaryValue) isValue() {}

func (v DictionaryValue) MarshalJSON() ([]byte, error) {
	type Alias DictionaryValue
	return json.Marshal(&struct {
		Kind string `json:"kind"`
		Alias
	}{
		Kind:  "dictionary",
		Alias: (Alias)(v),
	})
}

type DictionaryKey struct {
	Description string `json:"description"`
	Value       Value  `json:"value"`
}

// ArrayValue

type ArrayValue struct {
	Type       any    `json:"type"`
	TypeString string `json:"typeString"`
	Count      int    `json:"count"`
}

var _ Value = ArrayValue{}

func (ArrayValue) isValue() {}

func (v ArrayValue) MarshalJSON() ([]byte, error) {
	type Alias ArrayValue
	return json.Marshal(&struct {
		Kind string `json:"kind"`
		Alias
	}{
		Kind:  "array",
		Alias: (Alias)(v),
	})
}

// CompositeValue

type CompositeValue struct {
	Type       any      `json:"type"`
	TypeString string   `json:"typeString"`
	Fields     []string `json:"fields"`
}

var _ Value = CompositeValue{}

func (CompositeValue) isValue() {}

func (v CompositeValue) MarshalJSON() ([]byte, error) {
	type Alias CompositeValue
	return json.Marshal(&struct {
		Kind string `json:"kind"`
		Alias
	}{
		Kind:  "composite",
		Alias: (Alias)(v),
	})
}

// SomeValue

type SomeValue struct {
	Type       any    `json:"type"`
	TypeString string `json:"typeString"`
	Value      Value  `json:"value"`
}

var _ Value = SomeValue{}

func (SomeValue) isValue() {}

func (v SomeValue) MarshalJSON() ([]byte, error) {
	type Alias SomeValue
	return json.Marshal(&struct {
		Kind string `json:"kind"`
		Alias
	}{
		Kind:  "some",
		Alias: (Alias)(v),
	})
}

// prepareValue

var pathLinkValueFieldNames = []string{
	"targetPath",
	"type",
}

var publishedValueFieldNames = []string{
	"recipient",
	"type",
}

var pathCapabilityValueFieldNames = []string{
	"address",
	"path",
}

var idCapabilityValueFieldNames = []string{
	sema.CapabilityTypeAddressFieldName,
	sema.CapabilityTypeIDFieldName,
}

var accountCapabilityControllerValueFieldNames = []string{
	sema.AccountCapabilityControllerTypeCapabilityIDFieldName,
	sema.AccountCapabilityControllerTypeBorrowTypeFieldName,
}

var storageCapabilityControllerValueFieldNames = []string{
	sema.StorageCapabilityControllerTypeCapabilityIDFieldName,
	sema.StorageCapabilityControllerTypeBorrowTypeFieldName,
}

func prepareValue(value interpreter.Value, inter *interpreter.Interpreter) (Value, error) {
	ty, typeString := prepareType(value, inter)

	switch value := value.(type) {
	case interpreter.BoolValue,
		interpreter.NumberValue,
		*interpreter.StringValue,
		interpreter.CharacterValue,
		interpreter.AddressValue,
		interpreter.PathValue,
		interpreter.TypeValue:

		exported, err := runtime.ExportValue(value, inter, interpreter.EmptyLocationRange)
		if err != nil {
			return nil, err
		}

		exportedJSON, err := jsoncdc.Encode(exported)
		if err != nil {
			return nil, err
		}

		return PrimitiveValue{
			Type:        ty,
			TypeString:  typeString,
			Value:       exportedJSON,
			Description: value.String(),
		}, nil

	case *interpreter.DictionaryValue:
		keys := make([]DictionaryKey, 0, value.Count())

		var err error

		value.IterateKeys(inter, func(key interpreter.Value) (resume bool) {
			var preparedKey Value
			preparedKey, err = prepareValue(key, inter)
			if err != nil {
				return false
			}

			keys = append(keys, DictionaryKey{
				Description: key.String(),
				Value:       preparedKey,
			})

			return true
		})

		if err != nil {
			return nil, err
		}

		return DictionaryValue{
			Type:       ty,
			TypeString: typeString,
			Keys:       keys,
		}, nil

	case *interpreter.CompositeValue:
		fields := make([]string, 0, value.FieldCount())

		value.ForEachFieldName(func(field string) (resume bool) {
			fields = append(fields, field)

			return true
		})

		sort.Strings(fields)

		return CompositeValue{
			Type:       ty,
			TypeString: typeString,
			Fields:     fields,
		}, nil

	case *interpreter.SimpleCompositeValue:
		fieldNames := value.FieldNames

		fields := make([]string, 0, len(fieldNames))
		copy(fields, fieldNames)

		sort.Strings(fields)

		return CompositeValue{
			Type:       ty,
			TypeString: typeString,
			Fields:     fields,
		}, nil

	case interpreter.PathLinkValue: //nolint:staticcheck
		return CompositeValue{
			Type:       ty,
			TypeString: typeString,
			Fields:     pathLinkValueFieldNames,
		}, nil

	case interpreter.AccountLinkValue: //nolint:staticcheck
		return CompositeValue{
			Type:       ty,
			TypeString: typeString,
		}, nil

	case *interpreter.PathCapabilityValue: //nolint:staticcheck
		return CompositeValue{
			Type:       ty,
			TypeString: typeString,
			Fields:     pathCapabilityValueFieldNames,
		}, nil

	case *interpreter.IDCapabilityValue:
		return CompositeValue{
			Type:       ty,
			TypeString: typeString,
			Fields:     idCapabilityValueFieldNames,
		}, nil

	case *interpreter.AccountCapabilityControllerValue:
		return CompositeValue{
			Type:       ty,
			TypeString: typeString,
			Fields:     accountCapabilityControllerValueFieldNames,
		}, nil

	case *interpreter.StorageCapabilityControllerValue:
		return CompositeValue{
			Type:       ty,
			TypeString: typeString,
			Fields:     storageCapabilityControllerValueFieldNames,
		}, nil

	case *interpreter.PublishedValue:
		return CompositeValue{
			Type:       ty,
			TypeString: typeString,
			Fields:     publishedValueFieldNames,
		}, nil

	case *interpreter.ArrayValue:
		return ArrayValue{
			Type:       ty,
			TypeString: typeString,
			Count:      value.Count(),
		}, nil

	case *interpreter.SomeValue:
		innerValue := value.InnerValue(inter, interpreter.EmptyLocationRange)

		preparedInnerValue, err := prepareValue(innerValue, inter)
		if err != nil {
			return nil, err
		}

		return SomeValue{
			Type:       ty,
			TypeString: typeString,
			Value:      preparedInnerValue,
		}, nil

	default:
		return FallbackValue{
			Type:        ty,
			TypeString:  typeString,
			Description: value.String(),
		}, nil
	}
}
