package cadence

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dapperlabs/cadence/runtime"
	"github.com/dapperlabs/cadence/runtime/interpreter"
)

func TestConvertVoidValue(t *testing.T) {
	value := convertValue(interpreter.VoidValue{}, nil)

	assert.Equal(t, NewVoid(), value)
}

func TestConvertNilValue(t *testing.T) {
	value := convertValue(interpreter.NilValue{}, nil)

	assert.Equal(t, NewNil(), value)
}

func TestConvertSomeValue(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		value := convertValue(&interpreter.SomeValue{Value: nil}, nil)

		assert.Equal(t, NewOptional(nil), value)
	})

	t.Run("value", func(t *testing.T) {
		value := convertValue(
			&interpreter.SomeValue{Value: interpreter.NewIntValue(42)},
			nil,
		)

		assert.Equal(t, NewOptional(NewInt(42)), value)
	})
}

func TestConvertBoolValue(t *testing.T) {
	t.Run("true", func(t *testing.T) {
		value := convertValue(interpreter.BoolValue(true), nil)

		assert.Equal(t, NewBool(true), value)
	})

	t.Run("false", func(t *testing.T) {
		value := convertValue(interpreter.BoolValue(false), nil)

		assert.Equal(t, NewBool(false), value)
	})
}

func TestConvertStringValue(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		value := convertValue(&interpreter.StringValue{Str: ""}, nil)

		assert.Equal(t, NewString(""), value)
	})

	t.Run("non-empty", func(t *testing.T) {
		value := convertValue(&interpreter.StringValue{Str: "foo"}, nil)

		assert.Equal(t, NewString("foo"), value)
	})
}

func TestConvertArrayValue(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		value := convertValue(&interpreter.ArrayValue{Values: nil}, nil)

		assert.Equal(t, NewArray([]Value{}), value)
	})

	t.Run("non-empty", func(t *testing.T) {
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

var fooResourceType = ResourceType{
	CompositeType{
		typeID:     "test.Foo",
		Identifier: "Foo",
		Fields: []Field{
			{
				Identifier: "bar",
				Type:       IntType{},
			},
		},
	},
}

func TestConvertCompositeValue(t *testing.T) {
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
	expected :=
		NewComposite([]Value{NewInt(42)}).WithType(fooResourceType)

	assert.Equal(t, expected, actual)
}

func TestConvertDictionaryValue(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		script := `
            access(all) fun main(): {String: Int} {
                return {}
            }
        `

		actual := convertValueFromScript(t, script)
		expected := NewDictionary([]KeyValuePair{})

		assert.Equal(t, expected, actual)
	})

	t.Run("non-empty", func(t *testing.T) {
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

func TestConvertResourceArray(t *testing.T) {
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
		NewComposite([]Value{NewInt(1)}).WithType(fooResourceType),
		NewComposite([]Value{NewInt(2)}).WithType(fooResourceType),
	})

	assert.Equal(t, expected, actual)
}

func TestConvertResourceDictionary(t *testing.T) {
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
			Value: NewComposite([]Value{NewInt(1)}).WithType(fooResourceType),
		},
		{
			Key:   NewString("b"),
			Value: NewComposite([]Value{NewInt(2)}).WithType(fooResourceType),
		},
	})

	assert.Equal(t, expected, actual)
}

func TestConvertNestedResource(t *testing.T) {
	barResourceType := ResourceType{
		CompositeType{
			typeID:     "test.Bar",
			Identifier: "Bar",
			Fields: []Field{
				{
					Identifier: "x",
					Type:       IntType{},
				},
			},
		},
	}

	fooResourceType := ResourceType{
		CompositeType{
			typeID:     "test.Foo",
			Identifier: "Foo",
			Fields: []Field{
				{
					Identifier: "bar",
					Type:       barResourceType,
				},
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
	expected := NewComposite([]Value{
		NewComposite([]Value{NewInt(42)}).WithType(barResourceType),
	}).WithType(fooResourceType)

	assert.Equal(t, expected, actual)
}

func convertValueFromScript(t *testing.T, script string) Value {
	rt := runtime.NewInterpreterRuntime()

	value, err := rt.ExecuteScript(
		[]byte(script),
		nil,
		runtime.StringLocation("test"),
	)

	require.NoError(t, err)

	return ConvertValue(value)
}
