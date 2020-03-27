package cadence

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dapperlabs/cadence/runtime"
	"github.com/dapperlabs/cadence/runtime/interpreter"
	"github.com/dapperlabs/cadence/runtime/sema"
)

func TestConvertVoidValue(t *testing.T) {
	value := convertValue(interpreter.VoidValue{}, nil)

	assert.Equal(t, NewVoid(), value)
}

func TestConvertSomeValue(t *testing.T) {
	t.Run("Nil", func(t *testing.T) {
		value := convertValue(&interpreter.SomeValue{Value: nil}, nil)

		assert.Equal(t, NewOptional(nil), value)
	})

	t.Run("Non-nil", func(t *testing.T) {
		value := convertValue(
			&interpreter.SomeValue{Value: interpreter.NewIntValue(42)},
			nil,
		)

		assert.Equal(t, NewOptional(NewInt(42)), value)
	})
}

func TestConvertBoolValue(t *testing.T) {
	t.Run("True", func(t *testing.T) {
		value := convertValue(interpreter.BoolValue(true), nil)

		assert.Equal(t, NewBool(true), value)
	})

	t.Run("False", func(t *testing.T) {
		value := convertValue(interpreter.BoolValue(false), nil)

		assert.Equal(t, NewBool(false), value)
	})
}

func TestConvertStringValue(t *testing.T) {
	t.Run("Empty", func(t *testing.T) {
		value := convertValue(&interpreter.StringValue{Str: ""}, nil)

		assert.Equal(t, NewString(""), value)
	})

	t.Run("Non-empty", func(t *testing.T) {
		value := convertValue(&interpreter.StringValue{Str: "foo"}, nil)

		assert.Equal(t, NewString("foo"), value)
	})
}

func TestConvertArrayValue(t *testing.T) {
	t.Run("Empty", func(t *testing.T) {
		value := convertValue(&interpreter.ArrayValue{Values: nil}, nil)

		assert.Equal(t, NewArray([]Value{}), value)
	})

	t.Run("Non-empty", func(t *testing.T) {
		value := convertValue(
			&interpreter.ArrayValue{
				Values: []interpreter.Value{
					interpreter.NewIntValue(42),
					interpreter.NewStringValue("foo"),
				},
			},
			nil,
		)

		expected := NewArray([]Value{
			NewInt(42),
			NewString("foo"),
		})

		assert.Equal(t, expected, value)
	})
}

func TestConvertIntValue(t *testing.T) {
	value := convertValue(interpreter.NewIntValue(42), nil)

	assert.Equal(t, NewInt(42), value)
}

func TestConvertInt8Value(t *testing.T) {
	value := convertValue(interpreter.Int8Value(42), nil)

	assert.Equal(t, NewInt8(42), value)
}

func TestConvertInt16Value(t *testing.T) {
	value := convertValue(interpreter.Int16Value(42), nil)

	assert.Equal(t, NewInt16(42), value)
}

func TestConvertInt32Value(t *testing.T) {
	value := convertValue(interpreter.Int32Value(42), nil)

	assert.Equal(t, NewInt32(42), value)
}

func TestConvertInt64Value(t *testing.T) {
	value := convertValue(interpreter.Int64Value(42), nil)

	assert.Equal(t, NewInt64(42), value)
}

func TestConvertInt128Value(t *testing.T) {
	value := convertValue(interpreter.Int128Value{Int: sema.Int128TypeMaxInt}, nil)

	assert.Equal(t, NewInt128FromBig(sema.Int128TypeMaxInt), value)
}

func TestConvertInt256Value(t *testing.T) {
	value := convertValue(interpreter.Int256Value{Int: sema.Int256TypeMaxInt}, nil)

	assert.Equal(t, NewInt256FromBig(sema.Int256TypeMaxInt), value)
}

func TestConvertUIntValue(t *testing.T) {
	value := convertValue(interpreter.NewUIntValue(42), nil)

	assert.Equal(t, NewUInt(42), value)
}

func TestConvertUInt8Value(t *testing.T) {
	value := convertValue(interpreter.UInt8Value(42), nil)

	assert.Equal(t, NewUInt8(42), value)
}

func TestConvertUInt16Value(t *testing.T) {
	value := convertValue(interpreter.UInt16Value(42), nil)

	assert.Equal(t, NewUInt16(42), value)
}

func TestConvertUInt32Value(t *testing.T) {
	value := convertValue(interpreter.UInt32Value(42), nil)

	assert.Equal(t, NewUInt32(42), value)
}

func TestConvertUInt64Value(t *testing.T) {
	value := convertValue(interpreter.UInt64Value(42), nil)

	assert.Equal(t, NewUInt64(42), value)
}

func TestConvertUInt128Value(t *testing.T) {
	value := convertValue(interpreter.UInt128Value{Int: sema.UInt128TypeMaxInt}, nil)

	assert.Equal(t, NewUInt128FromBig(sema.UInt128TypeMaxInt), value)
}

func TestConvertUInt256Value(t *testing.T) {
	value := convertValue(interpreter.UInt256Value{Int: sema.UInt256TypeMaxInt}, nil)

	assert.Equal(t, NewUInt256FromBig(sema.UInt256TypeMaxInt), value)
}

func TestConvertWord8Value(t *testing.T) {
	value := convertValue(interpreter.UInt8Value(42), nil)

	assert.Equal(t, NewUInt8(42), value)
}

func TestConvertWord16Value(t *testing.T) {
	value := convertValue(interpreter.UInt16Value(42), nil)

	assert.Equal(t, NewUInt16(42), value)
}

func TestConvertWord32Value(t *testing.T) {
	value := convertValue(interpreter.UInt32Value(42), nil)

	assert.Equal(t, NewUInt32(42), value)
}

func TestConvertWord64Value(t *testing.T) {
	value := convertValue(interpreter.UInt64Value(42), nil)

	assert.Equal(t, NewUInt64(42), value)
}

func TestConvertFix64Value(t *testing.T) {
	value := convertValue(interpreter.Fix64Value(-123000000), nil)

	assert.Equal(t, NewFix64(-123000000), value)
}

func TestConvertUFix64Value(t *testing.T) {
	value := convertValue(interpreter.UFix64Value(123000000), nil)

	assert.Equal(t, NewUFix64(123000000), value)
}

func TestConvertIntegerValuesFromScript(t *testing.T) {
	for _, integerType := range sema.AllIntegerTypes {

		script := fmt.Sprintf(
			`
              pub fun main(): %s {
                  return 42
              }
            `,
			integerType,
		)

		assert.NotPanics(t, func() {
			convertValueFromScript(t, script)
		})
	}
}

func TestConvertFixedPointValuesFromScript(t *testing.T) {
	for _, fixedPointType := range sema.AllFixedPointTypes {
		script := fmt.Sprintf(
			`
              pub fun main(): %s {
                  return 1.23
              }
            `,
			fixedPointType,
		)

		assert.NotPanics(t, func() {
			convertValueFromScript(t, script)
		})
	}
}

func TestConvertDictionaryValue(t *testing.T) {
	t.Run("Empty", func(t *testing.T) {
		script := `
            access(all) fun main(): {String: Int} {
                return {}
            }
        `

		actual := convertValueFromScript(t, script)
		expected := NewDictionary([]KeyValuePair{})

		assert.Equal(t, expected, actual)
	})

	t.Run("Non-empty", func(t *testing.T) {
		script := `
            access(all) fun main(): {String: Int} {
                return {
                    "a": 1,
                    "b": 2
                }
            }
        `

		actual := convertValueFromScript(t, script)
		expected := NewDictionary([]KeyValuePair{
			{
				Key:   NewString("a"),
				Value: NewInt(1),
			},
			{
				Key:   NewString("b"),
				Value: NewInt(2),
			},
		})

		assert.Equal(t, expected, actual)
	})
}

func TestConvertAddressValue(t *testing.T) {
	script := `
        access(all) fun main(): Address {
            return 0x42
        }
    `

	actual := convertValueFromScript(t, script)
	expected := NewAddressFromBytes(
		[]byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x42},
	)

	assert.Equal(t, expected, actual)
}

func TestConvertStructValue(t *testing.T) {
	script := `
        access(all) struct Foo {
            access(all) let bar: Int
        
            init(bar: Int) {
                self.bar = bar
            }
        }
    
        access(all) fun main(): Foo {
            return Foo(bar: 42)
        }
    `

	actual := convertValueFromScript(t, script)
	expected := NewStruct([]Value{NewInt(42)}).WithType(fooStructType)

	assert.Equal(t, expected, actual)
}

func TestConvertResourceValue(t *testing.T) {
	script := `
        access(all) resource Foo {
            access(all) let bar: Int
        
            init(bar: Int) {
                self.bar = bar
            }
        }
    
        access(all) fun main(): @Foo {
            return <- create Foo(bar: 42)
        }
    `

	actual := convertValueFromScript(t, script)
	expected := NewResource([]Value{NewInt(42)}).WithType(fooResourceType)

	assert.Equal(t, expected, actual)
}

func TestConvertResourceArrayValue(t *testing.T) {
	script := `
        access(all) resource Foo {
            access(all) let bar: Int
        
            init(bar: Int) {
                self.bar = bar
            }
        }
    
        access(all) fun main(): @[Foo] {
            return <- [<- create Foo(bar: 1), <- create Foo(bar: 2)] 
        }
    `

	actual := convertValueFromScript(t, script)
	expected := NewArray([]Value{
		NewResource([]Value{NewInt(1)}).WithType(fooResourceType),
		NewResource([]Value{NewInt(2)}).WithType(fooResourceType),
	})

	assert.Equal(t, expected, actual)
}

func TestConvertResourceDictionaryValue(t *testing.T) {
	script := `
        access(all) resource Foo {
            access(all) let bar: Int
        
            init(bar: Int) {
                self.bar = bar
            }
        }
    
        access(all) fun main(): @{String: Foo} {
            return <- {
                "a": <- create Foo(bar: 1), 
                "b": <- create Foo(bar: 2)
            }
        }
    `

	actual := convertValueFromScript(t, script)
	expected := NewDictionary([]KeyValuePair{
		{
			Key:   NewString("a"),
			Value: NewResource([]Value{NewInt(1)}).WithType(fooResourceType),
		},
		{
			Key:   NewString("b"),
			Value: NewResource([]Value{NewInt(2)}).WithType(fooResourceType),
		},
	})

	assert.Equal(t, expected, actual)
}

func TestConvertNestedResourceValue(t *testing.T) {
	barResourceType := ResourceType{
		TypeID:     "test.Bar",
		Identifier: "Bar",
		Fields: []Field{
			{
				Identifier: "x",
				Type:       IntType{},
			},
		},
	}

	fooResourceType := ResourceType{
		TypeID:     "test.Foo",
		Identifier: "Foo",
		Fields: []Field{
			{
				Identifier: "bar",
				Type:       barResourceType,
			},
		},
	}

	script := `
        access(all) resource Bar {
            access(all) let x: Int

            init(x: Int) {
                self.x = x
            }
        }

        access(all) resource Foo {
            access(all) let bar: @Bar
        
            init(bar: @Bar) {
                self.bar <- bar
            }

            destroy() {
                destroy self.bar
            }
        }
    
        access(all) fun main(): @Foo {
            return <- create Foo(bar: <- create Bar(x: 42))
        }
    `

	actual := convertValueFromScript(t, script)
	expected := NewResource([]Value{
		NewResource([]Value{NewInt(42)}).WithType(barResourceType),
	}).WithType(fooResourceType)

	assert.Equal(t, expected, actual)
}

func TestConvertEventValue(t *testing.T) {
	script := `
        access(all) event Foo(bar: Int)
    
        access(all) fun main() {
            emit Foo(bar: 42)
        }
    `

	actual := convertEventFromScript(t, script)
	expected := NewEvent([]Value{NewInt(42)}).WithType(fooEventType)

	assert.Equal(t, expected, actual)
}

// mock runtime.Interface to capture events
type eventCapturingInterface struct {
	runtime.EmptyRuntimeInterface
	events []runtime.Event
}

func (t *eventCapturingInterface) EmitEvent(event runtime.Event) {
	t.events = append(t.events, event)
}

func convertEventFromScript(t *testing.T, script string) Event {
	rt := runtime.NewInterpreterRuntime()

	inter := &eventCapturingInterface{}

	_, err := rt.ExecuteScript(
		[]byte(script),
		inter,
		testLocation,
	)

	require.NoError(t, err)
	require.Len(t, inter.events, 1)

	event := inter.events[0]

	return ConvertEvent(event)
}

func convertValueFromScript(t *testing.T, script string) Value {
	rt := runtime.NewInterpreterRuntime()

	value, err := rt.ExecuteScript(
		[]byte(script),
		nil,
		testLocation,
	)

	require.NoError(t, err)

	return ConvertValue(value)
}

const testLocation = runtime.StringLocation("test")

const fooID = "Foo"

var fooTypeID = fmt.Sprintf("%s.%s", testLocation, fooID)
var fooFields = []Field{
	{
		Identifier: "bar",
		Type:       IntType{},
	},
}

var fooStructType = StructType{
	TypeID:     fooTypeID,
	Identifier: fooID,
	Fields:     fooFields,
}

var fooResourceType = ResourceType{
	TypeID:     fooTypeID,
	Identifier: fooID,
	Fields:     fooFields,
}

var fooEventType = EventType{
	TypeID:     fooTypeID,
	Identifier: fooID,
	Fields:     fooFields,
}
