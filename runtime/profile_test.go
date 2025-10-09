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

package runtime_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/common"
	. "github.com/onflow/cadence/runtime"
	. "github.com/onflow/cadence/test_utils/runtime_utils"
)

func TestRuntimeProfile(t *testing.T) {

	t.Parallel()

	importedScript := []byte(`
	  access(all) fun factorial(_ n: Int): Int {
	    pre {
	      n >= 0:
	        "factorial is only defined for integers greater than or equal to zero"
	    }
	    post {
	      result >= 1:
	        "the result must be greater than or equal to 1"
	    }

	    if n < 1 {
	      return 1
	    }

	    return n * factorial(n - 1)
	  }
	`)

	script := []byte(`
	  import "imported"

	  access(all) fun main(): Int {
	    factorial(5)
	    return 42
	  }
	`)

	computationProfile := NewComputationProfile()
	computationProfile.WithLocationMappings(
		map[string]string{
			"imported":                   "imported.cdc",
			common.ScriptLocation{}.ID(): "script.cdc",
		},
	)

	runtimeInterface := &TestRuntimeInterface{
		OnGetCode: func(location Location) (bytes []byte, err error) {
			switch location {
			case common.StringLocation("imported"):
				return importedScript, nil
			default:
				return nil, fmt.Errorf("unknown import location: %s", location)
			}
		},
	}

	config := DefaultTestInterpreterConfig
	config.ComputationProfile = computationProfile
	runtime := NewTestRuntimeWithConfig(config)

	value, err := runtime.ExecuteScript(
		Script{
			Source: script,
		},
		Context{
			Interface: runtimeInterface,
			Location:  common.ScriptLocation{},
		},
	)
	require.NoError(t, err)

	assert.Equal(t, cadence.NewInt(42), value)

	profile, err := NewPProfExporter(computationProfile).Export()
	require.NoError(t, err)

	assert.Len(t, profile.Sample, 26)

	require.Len(t, profile.Function, 2)

	factorialFunction := profile.Function[0]
	require.Equal(t, "factorial", factorialFunction.Name)
	require.Equal(t, "imported.cdc", factorialFunction.Filename)
	require.Equal(t, int64(2), factorialFunction.StartLine)

	mainFunction := profile.Function[1]
	require.Equal(t, "main", mainFunction.Name)
	require.Equal(t, "script.cdc", mainFunction.Filename)
	require.Equal(t, int64(4), mainFunction.StartLine)

	assert.Len(t, profile.Location, 7)

	for _, location := range profile.Location {
		require.Len(t, location.Line, 1)
		line := location.Line[0]

		switch line.Function {

		case profile.Function[0]: // factorial
			switch line.Line {
			case 4, 8, 12, 13, 16:
				// valid lines
			default:
				t.Fatalf("unexpected line number for factorial: %d", line.Line)
			}

		case profile.Function[1]: // main
			switch line.Line {
			case 5, 6:
				// valid lines
			default:
				t.Fatalf("unexpected line number for main: %d", line.Line)
			}

		default:
			t.Fatalf("unexpected function for location: %s", line.Function.Name)
		}
	}

}
