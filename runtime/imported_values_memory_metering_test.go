/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2022 Dapper Labs, Inc.
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

package runtime

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence"
	jsoncdc "github.com/onflow/cadence/encoding/json"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/parser2"
	"github.com/onflow/cadence/runtime/tests/utils"
)

func testUseMemory(meter map[common.MemoryKind]uint64) func(common.MemoryUsage) error {
	return func(usage common.MemoryUsage) error {
		meter[usage.Kind] += usage.Amount
		return nil
	}
}

func TestImportedValueMemoryMetering(t *testing.T) {

	t.Parallel()

	runtime := newTestInterpreterRuntime()

	runtimeInterface := func(meter map[common.MemoryKind]uint64) *testRuntimeInterface {
		intf := &testRuntimeInterface{
			meterMemory: testUseMemory(meter),
		}
		intf.decodeArgument = func(b []byte, t cadence.Type) (cadence.Value, error) {
			return jsoncdc.Decode(intf, b)
		}
		return intf
	}

	executeScript := func(script []byte, meter map[common.MemoryKind]uint64, args ...cadence.Value) {
		_, err := runtime.ExecuteScript(
			Script{
				Source:    script,
				Arguments: encodeArgs(args),
			},
			Context{
				Interface: runtimeInterface(meter),
				Location:  utils.TestLocation,
			},
		)

		require.NoError(t, err)
	}

	t.Run("String", func(t *testing.T) {
		t.Parallel()

		script := []byte(`
            pub fun main(x: String) {}
        `)

		meter := make(map[common.MemoryKind]uint64)

		executeScript(
			script,
			meter,
			cadence.String("hello"),
		)

		assert.Equal(t, uint64(6), meter[common.MemoryKindString])
	})

	t.Run("Optional", func(t *testing.T) {
		t.Parallel()

		script := []byte(`
            pub fun main(x: String?) {}
        `)

		meter := make(map[common.MemoryKind]uint64)

		executeScript(
			script,
			meter,
			cadence.NewUnmeteredOptional(cadence.String("hello")),
		)

		assert.Equal(t, uint64(1), meter[common.MemoryKindOptional])
		assert.Equal(t, uint64(3), meter[common.MemoryKindOptionalStaticType])
	})

	t.Run("UInt", func(t *testing.T) {
		t.Parallel()

		script := []byte(`
            pub fun main(x: UInt) {}
        `)

		meter := make(map[common.MemoryKind]uint64)
		executeScript(script, meter, cadence.NewUnmeteredUInt(2))
		assert.Equal(t, uint64(8), meter[common.MemoryKindBigInt])
	})

	t.Run("UInt8", func(t *testing.T) {
		t.Parallel()

		script := []byte(`
            pub fun main(x: UInt8) {}
        `)

		meter := make(map[common.MemoryKind]uint64)
		executeScript(script, meter, cadence.NewUnmeteredUInt8(2))
		assert.Equal(t, uint64(1), meter[common.MemoryKindNumber])
	})

	t.Run("UInt16", func(t *testing.T) {
		t.Parallel()

		script := []byte(`
            pub fun main(x: UInt16) {}
        `)

		meter := make(map[common.MemoryKind]uint64)
		executeScript(script, meter, cadence.NewUnmeteredUInt16(2))
		assert.Equal(t, uint64(2), meter[common.MemoryKindNumber])
	})

	t.Run("UInt32", func(t *testing.T) {
		t.Parallel()

		script := []byte(`
            pub fun main(x: UInt32) {}
        `)

		meter := make(map[common.MemoryKind]uint64)
		executeScript(script, meter, cadence.NewUnmeteredUInt32(2))
		assert.Equal(t, uint64(4), meter[common.MemoryKindNumber])
	})

	t.Run("UInt64", func(t *testing.T) {
		t.Parallel()

		script := []byte(`
            pub fun main(x: UInt64) {}
        `)

		meter := make(map[common.MemoryKind]uint64)
		executeScript(script, meter, cadence.NewUnmeteredUInt64(2))
		assert.Equal(t, uint64(8), meter[common.MemoryKindNumber])
	})

	t.Run("UInt128", func(t *testing.T) {
		t.Parallel()

		script := []byte(`
            pub fun main(x: UInt128) {}
        `)

		meter := make(map[common.MemoryKind]uint64)
		executeScript(script, meter, cadence.NewUnmeteredUInt128(2))
		assert.Equal(t, uint64(16), meter[common.MemoryKindBigInt])
	})

	t.Run("UInt256", func(t *testing.T) {
		t.Parallel()

		script := []byte(`
            pub fun main(x: UInt256) {}
        `)

		meter := make(map[common.MemoryKind]uint64)
		executeScript(script, meter, cadence.NewUnmeteredUInt256(2))
		assert.Equal(t, uint64(32), meter[common.MemoryKindBigInt])
	})

	t.Run("Int", func(t *testing.T) {
		t.Parallel()

		script := []byte(`
            pub fun main(x: Int) {}
        `)

		meter := make(map[common.MemoryKind]uint64)
		executeScript(script, meter, cadence.NewUnmeteredInt(2))
		assert.Equal(t, uint64(8), meter[common.MemoryKindBigInt])
	})

	t.Run("Int8", func(t *testing.T) {
		t.Parallel()

		script := []byte(`
            pub fun main(x: Int8) {}
        `)

		meter := make(map[common.MemoryKind]uint64)
		executeScript(script, meter, cadence.NewUnmeteredInt8(2))
		assert.Equal(t, uint64(1), meter[common.MemoryKindNumber])
	})

	t.Run("Int16", func(t *testing.T) {
		t.Parallel()

		script := []byte(`
            pub fun main(x: Int16) {}
        `)

		meter := make(map[common.MemoryKind]uint64)
		executeScript(script, meter, cadence.NewUnmeteredInt16(2))
		assert.Equal(t, uint64(2), meter[common.MemoryKindNumber])
	})

	t.Run("Int32", func(t *testing.T) {
		t.Parallel()

		script := []byte(`
            pub fun main(x: Int32) {}
        `)

		meter := make(map[common.MemoryKind]uint64)
		executeScript(script, meter, cadence.NewUnmeteredInt32(2))
		assert.Equal(t, uint64(4), meter[common.MemoryKindNumber])
	})

	t.Run("Int64", func(t *testing.T) {
		t.Parallel()

		script := []byte(`
            pub fun main(x: Int64) {}
        `)

		meter := make(map[common.MemoryKind]uint64)
		executeScript(script, meter, cadence.NewUnmeteredInt64(2))
		assert.Equal(t, uint64(8), meter[common.MemoryKindNumber])
	})

	t.Run("Int128", func(t *testing.T) {
		t.Parallel()

		script := []byte(`
            pub fun main(x: Int128) {}
        `)

		meter := make(map[common.MemoryKind]uint64)
		executeScript(script, meter, cadence.NewUnmeteredInt128(2))
		assert.Equal(t, uint64(16), meter[common.MemoryKindBigInt])
	})

	t.Run("Int256", func(t *testing.T) {
		t.Parallel()

		script := []byte(`
            pub fun main(x: Int256) {}
        `)

		meter := make(map[common.MemoryKind]uint64)
		executeScript(script, meter, cadence.NewUnmeteredInt256(2))
		assert.Equal(t, uint64(32), meter[common.MemoryKindBigInt])
	})

	t.Run("Word8", func(t *testing.T) {
		t.Parallel()

		script := []byte(`
            pub fun main(x: Word8) {}
        `)

		meter := make(map[common.MemoryKind]uint64)
		executeScript(script, meter, cadence.NewUnmeteredWord8(2))
		assert.Equal(t, uint64(1), meter[common.MemoryKindNumber])
	})

	t.Run("Word16", func(t *testing.T) {
		t.Parallel()

		script := []byte(`
            pub fun main(x: Word16) {}
        `)

		meter := make(map[common.MemoryKind]uint64)
		executeScript(script, meter, cadence.NewUnmeteredWord16(2))
		assert.Equal(t, uint64(2), meter[common.MemoryKindNumber])
	})

	t.Run("Word32", func(t *testing.T) {
		t.Parallel()

		script := []byte(`
            pub fun main(x: Word32) {}
        `)

		meter := make(map[common.MemoryKind]uint64)
		executeScript(script, meter, cadence.NewUnmeteredWord32(2))
		assert.Equal(t, uint64(4), meter[common.MemoryKindNumber])
	})

	t.Run("Word64", func(t *testing.T) {
		t.Parallel()

		script := []byte(`
            pub fun main(x: Word64) {}
        `)

		meter := make(map[common.MemoryKind]uint64)
		executeScript(script, meter, cadence.NewUnmeteredWord64(2))
		assert.Equal(t, uint64(8), meter[common.MemoryKindNumber])
	})

	t.Run("Fix64", func(t *testing.T) {
		t.Parallel()

		script := []byte(`
            pub fun main(x: Fix64) {}
        `)

		meter := make(map[common.MemoryKind]uint64)

		fix64Value, err := cadence.NewUnmeteredFix64FromParts(true, 1, 4)
		require.NoError(t, err)

		executeScript(script, meter, fix64Value)
		assert.Equal(t, uint64(8), meter[common.MemoryKindNumber])
	})

	t.Run("UFix64", func(t *testing.T) {
		t.Parallel()

		script := []byte(`
            pub fun main(x: UFix64) {}
        `)

		meter := make(map[common.MemoryKind]uint64)
		ufix64Value, err := cadence.NewUFix64FromParts(1, 4)
		require.NoError(t, err)

		executeScript(script, meter, ufix64Value)
		assert.Equal(t, uint64(8), meter[common.MemoryKindNumber])
	})
}

type testMemoryError struct{}

func (testMemoryError) Error() string {
	return "memory limit exceeded"
}

func TestMemoryMeteringErrors(t *testing.T) {

	t.Parallel()

	runtime := newTestInterpreterRuntime()

	type memoryMeter map[common.MemoryKind]uint64

	runtimeInterface := func(meter memoryMeter) *testRuntimeInterface {
		intf := &testRuntimeInterface{
			meterMemory: func(usage common.MemoryUsage) error {
				if usage.Kind == common.MemoryKindString ||
					usage.Kind == common.MemoryKindArrayBase {
					return testMemoryError{}
				}
				return nil
			},
		}
		intf.decodeArgument = func(b []byte, t cadence.Type) (cadence.Value, error) {
			return jsoncdc.Decode(intf, b)
		}
		return intf
	}

	executeScript := func(script []byte, meter memoryMeter, args ...cadence.Value) error {
		_, err := runtime.ExecuteScript(
			Script{
				Source:    script,
				Arguments: encodeArgs(args),
			},
			Context{
				Interface: runtimeInterface(meter),
				Location:  utils.TestLocation,
			},
		)

		return err
	}

	t.Run("no errors", func(t *testing.T) {
		t.Parallel()

		script := []byte(`
            pub fun main() {}
        `)

		err := executeScript(script, memoryMeter{})
		assert.NoError(t, err)
	})

	t.Run("importing", func(t *testing.T) {
		t.Parallel()

		script := []byte(`
            pub fun main(x: String) {}
        `)

		err := executeScript(
			script,
			memoryMeter{},
			cadence.String("hello"),
		)

		assert.ErrorIs(t, err, testMemoryError{})
	})

	t.Run("at lexer", func(t *testing.T) {
		t.Parallel()

		script := []byte(`
            pub fun main() {
                let x = "hello"
            }
        `)

		err := executeScript(script, memoryMeter{})

		require.IsType(t, Error{}, err)
		runtimeError := err.(Error)

		require.IsType(t, &ParsingCheckingError{}, runtimeError.Err)
		parsingCheckingError := runtimeError.Err.(*ParsingCheckingError)

		require.IsType(t, parser2.Error{}, parsingCheckingError.Err)
		parserError := parsingCheckingError.Err.(parser2.Error)

		assert.Contains(t, parserError.Error(), "memory limit exceeded")
	})

	t.Run("at interpreter", func(t *testing.T) {
		t.Parallel()

		script := []byte(`
            pub fun main() {
                let x: [AnyStruct] = []
            }
        `)

		err := executeScript(script, memoryMeter{})
		assert.ErrorIs(t, err, testMemoryError{})
	})
}

func TestImportedValueMemoryMeteringForSimpleTypes(t *testing.T) {

	t.Parallel()

	runtime := newTestInterpreterRuntime()

	type importTest struct {
		TypeName     string
		MemoryKind   common.MemoryKind
		Weight       uint64
		TypeInstance cadence.Value
	}

	tests := []importTest{
		{
			TypeName:     "String",
			MemoryKind:   common.MemoryKindString,
			Weight:       7 + 1,
			TypeInstance: cadence.String("forever"),
		},
		{
			TypeName:     "Character",
			MemoryKind:   common.MemoryKindCharacter,
			Weight:       1,
			TypeInstance: cadence.Character("a"),
		},
		//{
		//	TypeName:   "Type",
		//	MemoryKind: common.MemoryKindTypeValue,
		//	Weight:     3,
		//	TypeInstance: cadence.TypeValue{
		//		StaticType: cadence.AnyType{},
		//	},
		//},
		{
			TypeName:     "Bool",
			MemoryKind:   common.MemoryKindBool,
			Weight:       1,
			TypeInstance: cadence.Bool(true),
		},
		{
			TypeName:     "Address",
			MemoryKind:   common.MemoryKindAddress,
			Weight:       1,
			TypeInstance: cadence.Address{},
		},

		// Verify Path and its composing type, String
		{
			TypeName:   "Path",
			MemoryKind: common.MemoryKindPathValue,
			Weight:     1,
			TypeInstance: cadence.Path{
				Domain:     "storage",
				Identifier: "id3",
			},
		},
		{
			TypeName:   "Path",
			MemoryKind: common.MemoryKindRawString,
			Weight:     3 + 1 + 68, // 68 is for tokens
			TypeInstance: cadence.Path{
				Domain:     "storage",
				Identifier: "id3",
			},
		},

		// Verify Capability and its composing values, Path and Type.
		{
			TypeName:   "Capability",
			MemoryKind: common.MemoryKindCapabilityValue,
			Weight:     1,
			TypeInstance: cadence.Capability{
				Path: cadence.Path{
					Domain:     "public",
					Identifier: "foobarrington",
				},
				Address: cadence.Address{},
				BorrowType: cadence.ReferenceType{
					Authorized: true,
					Type:       cadence.AnyType{},
				},
			},
		},
		{
			TypeName:   "Capability",
			MemoryKind: common.MemoryKindPathValue,
			Weight:     1,
			TypeInstance: cadence.Capability{
				Path: cadence.Path{
					Domain:     "public",
					Identifier: "foobarrington",
				},
				Address: cadence.Address{},
				BorrowType: cadence.ReferenceType{
					Authorized: true,
					Type:       cadence.AnyType{},
				},
			},
		},
		{
			TypeName:   "Capability",
			MemoryKind: common.MemoryKindRawString,
			Weight:     13 + 1 + 74, // 74 is for tokens
			TypeInstance: cadence.Capability{
				Path: cadence.Path{
					Domain:     "public",
					Identifier: "foobarrington",
				},
				Address: cadence.Address{},
				BorrowType: cadence.ReferenceType{
					Authorized: true,
					Type:       cadence.AnyType{},
				},
			},
		},
		//{
		//	TypeName:   "Capability",
		//	MemoryKind: common.MemoryKindTypeValue,
		//	Weight:     3,
		//	TypeInstance: cadence.Capability{
		//		Path: cadence.Path{
		//			Domain:     "public",
		//			Identifier: "foobarrington",
		//		},
		//		Address: cadence.Address{},
		//		BorrowType: cadence.ReferenceType{
		//			Authorized: true,
		//			Type:       cadence.AnyType{},
		//		},
		//	},
		//},

		// Verify Optional and its composing type
		{
			TypeName:     "String?",
			MemoryKind:   common.MemoryKindOptional,
			Weight:       1,
			TypeInstance: cadence.NewUnmeteredOptional(cadence.String("hello")),
		},
		{
			TypeName:     "String?",
			MemoryKind:   common.MemoryKindOptionalStaticType,
			Weight:       3,
			TypeInstance: cadence.NewUnmeteredOptional(cadence.String("hello")),
		},
		{
			TypeName:     "String?",
			MemoryKind:   common.MemoryKindString,
			Weight:       5 + 1,
			TypeInstance: cadence.NewUnmeteredOptional(cadence.String("hello")),
		},

		// Not importable: Void
		// Not a user-visible type (not in BaseTypeActivation): Link
		// TODO: Bytes, U?Int\d*, Word\d+, U?Fix64, Array, Dictionary, Struct, Resource, Event, Contract, Enum
	}
	for _, test := range tests {
		testName := fmt.Sprintf("%s_%s", test.TypeName, test.MemoryKind.String())
		t.Run(testName, func(test importTest) func(t *testing.T) {
			return func(t *testing.T) {
				t.Parallel()

				meter := make(map[common.MemoryKind]uint64)
				runtimeInterface := &testRuntimeInterface{
					meterMemory: testUseMemory(meter),
				}
				runtimeInterface.decodeArgument = func(b []byte, t cadence.Type) (cadence.Value, error) {
					return jsoncdc.Decode(runtimeInterface, b)
				}

				script := []byte(fmt.Sprintf(`
                    pub fun main(x: %s) {}
                `, test.TypeName))

				_, err := runtime.ExecuteScript(
					Script{
						Source: script,
						Arguments: encodeArgs([]cadence.Value{
							test.TypeInstance,
						}),
					},
					Context{
						Interface: runtimeInterface,
						Location:  utils.TestLocation,
					},
				)

				require.NoError(t, err)

				assert.Equal(t, test.Weight, meter[test.MemoryKind])
			}
		}(test))
	}
}

func TestScriptDecodedLocationMetering(t *testing.T) {

	t.Parallel()

	runtime := newTestInterpreterRuntime()

	type importTest struct {
		Location   common.Location
		MemoryKind common.MemoryKind
		Weight     uint64
		Name       string
	}

	tests := []importTest{
		{
			MemoryKind: common.MemoryKindBytes,
			Weight:     3 + 1,
			Name:       "script",
			Location:   common.ScriptLocation([]byte{1, 2, 3}),
		},
		{
			MemoryKind: common.MemoryKindRawString,
			Weight:     3 + 1 + 106, // 106 is for tokens
			Name:       "string",
			Location:   common.StringLocation("abc"),
		},
	}
	for _, test := range tests {
		t.Run(test.Name, func(test importTest) func(t *testing.T) {
			return func(t *testing.T) {
				t.Parallel()

				meter := make(map[common.MemoryKind]uint64)
				runtimeInterface := &testRuntimeInterface{
					meterMemory: testUseMemory(meter),
				}
				runtimeInterface.decodeArgument = func(b []byte, t cadence.Type) (cadence.Value, error) {
					return jsoncdc.Decode(runtimeInterface, b)
				}

				value := cadence.NewUnmeteredStruct([]cadence.Value{}).WithType(
					&cadence.StructType{
						Location:            test.Location,
						QualifiedIdentifier: "S",
					})

				script := []byte(`
                    pub struct S {}
                    pub fun main(x: S) {}
                `)

				_, err := runtime.ExecuteScript(
					Script{
						Source:    script,
						Arguments: encodeArgs([]cadence.Value{value}),
					},
					Context{
						Interface: runtimeInterface,
						Location:  test.Location,
					},
				)

				require.NoError(t, err)

				assert.Equal(t, test.Weight, meter[test.MemoryKind])
			}
		}(test))
	}
}
