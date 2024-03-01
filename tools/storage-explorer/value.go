/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Dapper Labs, Inc.
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
	"sort"

	"github.com/onflow/cadence/runtime/interpreter"
)

type Value interface {
	isValue()
}

// TypeOnlyValue

type TypeOnlyValue struct {
	Type any `json:"type"`
}

var _ Value = TypeOnlyValue{}

func (TypeOnlyValue) isValue() {}

// BoolValue

type BoolValue struct {
	Type  any  `json:"type"`
	Value bool `json:"value"`
}

var _ Value = BoolValue{}

func (BoolValue) isValue() {}

// NumberValue

type NumberValue struct {
	Type  any    `json:"type"`
	Value string `json:"value"`
}

var _ Value = NumberValue{}

func (NumberValue) isValue() {}

// StringValue

type StringValue struct {
	Type  any    `json:"type"`
	Value string `json:"value"`
}

var _ Value = StringValue{}

func (StringValue) isValue() {}

// DictionaryValue

type DictionaryValue struct {
	Type any     `json:"type"`
	Keys []Value `json:"keys"`
}

var _ Value = DictionaryValue{}

func (DictionaryValue) isValue() {}

// CompositeValue

type CompositeValue struct {
	Type   any      `json:"type"`
	Fields []string `json:"fields"`
}

var _ Value = CompositeValue{}

func (CompositeValue) isValue() {}

// prepareValue

func prepareValue(value interpreter.Value, inter *interpreter.Interpreter) Value {
	ty := prepareType(value, inter)

	switch value := value.(type) {
	case interpreter.BoolValue:
		return BoolValue{
			Type:  ty,
			Value: bool(value),
		}

	case interpreter.NumberValue:
		return NumberValue{
			Type:  ty,
			Value: value.String(),
		}

	case *interpreter.StringValue:
		return StringValue{
			Type:  ty,
			Value: value.Str,
		}

	case *interpreter.CharacterValue:
		return StringValue{
			Type:  ty,
			Value: value.Str,
		}

	case *interpreter.DictionaryValue:
		keys := make([]Value, 0, value.Count())

		value.IterateKeys(inter, func(key interpreter.Value) (resume bool) {
			keys = append(keys, prepareValue(key, inter))

			return true
		})

		return DictionaryValue{
			Type: ty,
			Keys: keys,
		}

	case *interpreter.CompositeValue:
		fields := make([]string, 0, value.FieldCount())

		value.ForEachFieldName(func(field string) (resume bool) {
			fields = append(fields, field)

			return true
		})

		sort.Strings(fields)

		return CompositeValue{
			Type:   ty,
			Fields: fields,
		}

		// TODO:
		//   - AccountCapabilityControllerValue
		//   - AccountLinkValue
		//   - AddressValue
		//   - ArrayValue
		//   - CapabilityControllerValue
		//   - IDCapabilityValue
		//   - PathCapabilityValue
		//   - PathLinkValue
		//   - PathValue
		//   - PublishedValue
		//   - SimpleCompositeValue
		//   - SomeValue
		//   - StorageCapabilityControllerValue
		//   - TypeValue

	default:
		return TypeOnlyValue{
			Type: ty,
		}
	}
}
