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

	pprof "github.com/google/pprof/profile"
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
	    factorial(2)
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
	computationProfile.WithComputationWeights(
		map[common.ComputationKind]uint64{
			common.ComputationKindStatement:          2,
			common.ComputationKindLoop:               2,
			common.ComputationKindFunctionInvocation: 2,
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

	// Locations. NOT in execution  or appearance order,
	// but lexically ordered aggregate key of stack trace

	assert.Equal(
		t,
		[]pprof.Line{{
			Function: profile.Function[1], // main
			Line:     5,
		}},
		profile.Location[0].Line,
	)
	assert.Equal(
		t,
		[]pprof.Line{{
			Function: profile.Function[0], // factorial
			Line:     12,
		}},
		profile.Location[1].Line,
	)
	assert.Equal(
		t,
		[]pprof.Line{{
			Function: profile.Function[0], // factorial
			Line:     16,
		}},
		profile.Location[2].Line,
	)
	assert.Equal(
		t,
		[]pprof.Line{{
			Function: profile.Function[0], // factorial
			Line:     13,
		}},
		profile.Location[3].Line,
	)
	assert.Equal(
		t,
		[]pprof.Line{{
			Function: profile.Function[0], // factorial
			Line:     4,
		}},
		profile.Location[4].Line,
	)
	assert.Equal(
		t,
		[]pprof.Line{{
			Function: profile.Function[0], // factorial
			Line:     8,
		}},
		profile.Location[5].Line,
	)
	assert.Equal(
		t,
		[]pprof.Line{{
			Function: profile.Function[1], // main
			Line:     6,
		}},
		profile.Location[6].Line,
	)

	// Samples. NOT in execution / appearance order,
	// but lexically ordered aggregate key of stack trace

	require.Len(t, profile.Sample, 14)

	assert.Equal(t,
		[]int64{4},
		profile.Sample[0].Value,
	)
	assert.Equal(t,
		[]*pprof.Location{
			profile.Location[0],
		},
		profile.Sample[0].Location,
	)

	assert.Equal(t,
		[]int64{2},
		profile.Sample[1].Value,
	)
	assert.Equal(t,
		[]*pprof.Location{
			profile.Location[1],
			profile.Location[0],
		},
		profile.Sample[1].Location,
	)

	assert.Equal(t,
		[]int64{4},
		profile.Sample[2].Value,
	)
	assert.Equal(t,
		[]*pprof.Location{
			profile.Location[2],
			profile.Location[0],
		},
		profile.Sample[2].Location,
	)

	assert.Equal(t,
		[]int64{2},
		profile.Sample[3].Value,
	)
	assert.Equal(t,
		[]*pprof.Location{
			profile.Location[1],
			profile.Location[2],
			profile.Location[0],
		},
		profile.Sample[3].Location,
	)

	assert.Equal(t,
		[]int64{4},
		profile.Sample[4].Value,
	)
	assert.Equal(t,
		[]*pprof.Location{
			profile.Location[2],
			profile.Location[2],
			profile.Location[0],
		},
		profile.Sample[4].Location,
	)

	assert.Equal(t,
		[]int64{2},
		profile.Sample[5].Value,
	)
	assert.Equal(t,
		[]*pprof.Location{
			profile.Location[1],
			profile.Location[2],
			profile.Location[2],
			profile.Location[0],
		},
		profile.Sample[5].Location,
	)

	assert.Equal(t,
		[]int64{2},
		profile.Sample[6].Value,
	)
	assert.Equal(t,
		[]*pprof.Location{
			profile.Location[3],
			profile.Location[2],
			profile.Location[2],
			profile.Location[0],
		},
		profile.Sample[6].Location,
	)

	assert.Equal(t,
		[]int64{2},
		profile.Sample[7].Value,
	)
	assert.Equal(t,
		[]*pprof.Location{
			profile.Location[4],
			profile.Location[2],
			profile.Location[2],
			profile.Location[0],
		},
		profile.Sample[7].Location,
	)

	assert.Equal(t,
		[]int64{2},
		profile.Sample[8].Value,
	)
	assert.Equal(t,
		[]*pprof.Location{
			profile.Location[5],
			profile.Location[2],
			profile.Location[2],
			profile.Location[0],
		},
		profile.Sample[8].Location,
	)

	assert.Equal(t,
		[]int64{2},
		profile.Sample[9].Value,
	)
	assert.Equal(t,
		[]*pprof.Location{
			profile.Location[4],
			profile.Location[2],
			profile.Location[0],
		},
		profile.Sample[9].Location,
	)

	assert.Equal(t,
		[]int64{2},
		profile.Sample[10].Value,
	)
	assert.Equal(t,
		[]*pprof.Location{
			profile.Location[5],
			profile.Location[2],
			profile.Location[0],
		},
		profile.Sample[10].Location,
	)

	assert.Equal(t,
		[]int64{2},
		profile.Sample[11].Value,
	)
	assert.Equal(t,
		[]*pprof.Location{
			profile.Location[4],
			profile.Location[0],
		},
		profile.Sample[11].Location,
	)

	assert.Equal(t,
		[]int64{2},
		profile.Sample[12].Value,
	)
	assert.Equal(t,
		[]*pprof.Location{
			profile.Location[5],
			profile.Location[0],
		},
		profile.Sample[12].Location,
	)

	assert.Equal(t,
		[]int64{2},
		profile.Sample[13].Value,
	)
	assert.Equal(t,
		[]*pprof.Location{
			profile.Location[6],
		},
		profile.Sample[13].Location,
	)
}
