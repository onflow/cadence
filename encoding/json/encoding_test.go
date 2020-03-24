package json_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dapperlabs/cadence"
	"github.com/dapperlabs/cadence/encoding/json"
	"github.com/dapperlabs/cadence/runtime"
)

func TestEncodeNestedResource(t *testing.T) {
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

	actualJSON, err := json.Encode(actual)
	assert.NoError(t, err)

	t.Log(string(actualJSON))
}

func convertValueFromScript(t *testing.T, script string) cadence.Value {
	rt := runtime.NewInterpreterRuntime()

	value, err := rt.ExecuteScript(
		[]byte(script),
		nil,
		runtime.StringLocation("test"),
	)

	require.NoError(t, err)

	return cadence.ConvertValue(value)
}
