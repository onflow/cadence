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

package runtime

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/parser"
	"github.com/onflow/cadence/runtime/stdlib"
)

func TestNewLocationCoverage(t *testing.T) {

	t.Parallel()

	// Represents line numbers with statement execution count.
	// For the time being, if a line has two statements, we cannot
	// distinguish between their hits separately.
	// For example: "if let index = self.index(s, until, startIndex) {"
	lineHits := map[int]int{3: 0, 4: 0, 5: 0, 7: 0, 9: 0, 11: 0}
	locationCoverage := NewLocationCoverage(lineHits)

	assert.Equal(
		t,
		map[int]int{3: 0, 4: 0, 5: 0, 7: 0, 9: 0, 11: 0},
		locationCoverage.LineHits,
	)
	assert.Equal(
		t,
		[]int{3, 4, 5, 7, 9, 11},
		locationCoverage.MissedLines(),
	)
	assert.Equal(t, 6, locationCoverage.Statements)
	assert.Equal(t, "0.0%", locationCoverage.Percentage())
	assert.Equal(t, 0, locationCoverage.CoveredLines())
}

func TestLocationCoverageAddLineHit(t *testing.T) {

	t.Parallel()

	lineHits := map[int]int{3: 0, 4: 0, 5: 0, 7: 0, 9: 0, 11: 0}
	locationCoverage := NewLocationCoverage(lineHits)

	// Lines below 1 are dropped.
	locationCoverage.AddLineHit(0)
	locationCoverage.AddLineHit(3)
	locationCoverage.AddLineHit(3)
	locationCoverage.AddLineHit(7)
	locationCoverage.AddLineHit(9)
	// Line 15 was not included in the lineHits map, however we
	// want it to be tracked. This will help to find out about
	// cases where the inspector does not find all the statements.
	// We should also discuss if the Statements counter should be
	// increased in this case.
	// TBD
	locationCoverage.AddLineHit(15)

	assert.Equal(
		t,
		map[int]int{3: 2, 4: 0, 5: 0, 7: 1, 9: 1, 11: 0, 15: 1},
		locationCoverage.LineHits,
	)
	assert.Equal(t, 6, locationCoverage.Statements)
	assert.Equal(t, "66.7%", locationCoverage.Percentage())
}

func TestLocationCoverageCoveredLines(t *testing.T) {

	t.Parallel()

	lineHits := map[int]int{3: 0, 4: 0, 5: 0, 7: 0, 9: 0, 11: 0}
	locationCoverage := NewLocationCoverage(lineHits)

	locationCoverage.AddLineHit(3)
	locationCoverage.AddLineHit(3)
	locationCoverage.AddLineHit(7)
	locationCoverage.AddLineHit(9)
	locationCoverage.AddLineHit(15)

	assert.Equal(t, 4, locationCoverage.CoveredLines())
}

func TestLocationCoverageMissedLines(t *testing.T) {

	t.Parallel()

	lineHits := map[int]int{3: 0, 4: 0, 5: 0, 7: 0, 9: 0, 11: 0}
	locationCoverage := NewLocationCoverage(lineHits)

	locationCoverage.AddLineHit(3)
	locationCoverage.AddLineHit(3)
	locationCoverage.AddLineHit(7)
	locationCoverage.AddLineHit(9)
	locationCoverage.AddLineHit(15)

	assert.Equal(
		t,
		[]int{4, 5, 11},
		locationCoverage.MissedLines(),
	)
}

func TestLocationCoveragePercentage(t *testing.T) {

	t.Parallel()

	lineHits := map[int]int{3: 0, 4: 0, 5: 0}
	locationCoverage := NewLocationCoverage(lineHits)

	locationCoverage.AddLineHit(3)
	locationCoverage.AddLineHit(4)
	locationCoverage.AddLineHit(5)
	// Note: Line 15 was not included in the lineHits map,
	// but we saturate the percentage at 100%.
	locationCoverage.AddLineHit(15)

	assert.Equal(t, "100.0%", locationCoverage.Percentage())
}

func TestNewCoverageReport(t *testing.T) {

	t.Parallel()

	coverageReport := NewCoverageReport()

	assert.Equal(t, 0, len(coverageReport.Coverage))
	assert.Equal(t, 0, len(coverageReport.Locations))
	assert.Equal(t, 0, len(coverageReport.ExcludedLocations))
}

func TestCoverageReportExcludeLocation(t *testing.T) {

	t.Parallel()

	coverageReport := NewCoverageReport()

	location := common.StringLocation("FooContract")
	coverageReport.ExcludeLocation(location)
	// We do not allow duplicate locations
	coverageReport.ExcludeLocation(location)

	assert.Equal(t, 1, len(coverageReport.ExcludedLocations))
	assert.Equal(t, true, coverageReport.IsLocationExcluded(location))
}

func TestCoverageReportInspectProgram(t *testing.T) {

	t.Parallel()

	script := []byte(`
	  access(all) fun answer(): Int {
	    var i = 0
	    while i < 42 {
	      i = i + 1
	    }
	    return i
	  }
	`)

	program, err := parser.ParseProgram(nil, script, parser.Config{})
	require.NoError(t, err)

	coverageReport := NewCoverageReport()

	location := common.StringLocation("AnswerScript")
	coverageReport.InspectProgram(location, program)

	assert.Equal(t, 1, len(coverageReport.Coverage))
	assert.Equal(t, 1, len(coverageReport.Locations))
	assert.Equal(t, true, coverageReport.IsLocationInspected(location))
}

func TestCoverageReportInspectProgramForExcludedLocation(t *testing.T) {

	t.Parallel()

	script := []byte(`
	  access(all) fun answer(): Int {
	    var i = 0
	    while i < 42 {
	      i = i + 1
	    }
	    return i
	  }
	`)

	program, err := parser.ParseProgram(nil, script, parser.Config{})
	require.NoError(t, err)

	coverageReport := NewCoverageReport()

	location := common.StringLocation("AnswerScript")
	coverageReport.ExcludeLocation(location)
	coverageReport.InspectProgram(location, program)

	assert.Equal(t, 0, len(coverageReport.Coverage))
	assert.Equal(t, 0, len(coverageReport.Locations))
	assert.Equal(t, false, coverageReport.IsLocationInspected(location))
}

func TestCoverageReportInspectProgramWithLocationFilter(t *testing.T) {

	t.Parallel()

	transaction := []byte(`
	  transaction(amount: UFix64) {
	    prepare(account: AuthAccount) {
	      assert(account.balance >= amount)
	    }
	  }
	`)

	program, err := parser.ParseProgram(nil, transaction, parser.Config{})
	require.NoError(t, err)

	coverageReport := NewCoverageReport()
	coverageReport.WithLocationFilter(func(location common.Location) bool {
		_, addressLoc := location.(common.AddressLocation)
		_, stringLoc := location.(common.StringLocation)
		// We only allow inspection of AddressLocation or StringLocation
		return addressLoc || stringLoc
	})

	location := common.TransactionLocation{0x1a, 0x2b}
	coverageReport.InspectProgram(location, program)

	assert.Equal(t, 0, len(coverageReport.Coverage))
	assert.Equal(t, 0, len(coverageReport.Locations))
	assert.Equal(t, false, coverageReport.IsLocationInspected(location))
}

func TestCoverageReportAddLineHit(t *testing.T) {

	t.Parallel()

	script := []byte(`
	  access(all) fun answer(): Int {
	    var i = 0
	    while i < 42 {
	      i = i + 1
	    }
	    return i
	  }
	`)

	program, err := parser.ParseProgram(nil, script, parser.Config{})
	require.NoError(t, err)

	coverageReport := NewCoverageReport()

	location := common.StringLocation("AnswerScript")
	coverageReport.InspectProgram(location, program)

	coverageReport.AddLineHit(location, 3)
	coverageReport.AddLineHit(location, 3)
	coverageReport.AddLineHit(location, 5)

	locationCoverage := coverageReport.Coverage[location]

	assert.Equal(
		t,
		map[int]int{3: 2, 4: 0, 5: 1, 7: 0},
		locationCoverage.LineHits,
	)
	assert.Equal(
		t,
		[]int{4, 7},
		locationCoverage.MissedLines(),
	)
	assert.Equal(t, 4, locationCoverage.Statements)
	assert.Equal(t, "50.0%", locationCoverage.Percentage())
	assert.Equal(t, 2, locationCoverage.CoveredLines())
}

func TestCoverageReportWithFlowLocation(t *testing.T) {

	t.Parallel()

	script := []byte(`
	  access(all) fun answer(): Int {
	    var i = 0
	    while i < 42 {
	      i = i + 1
	    }
	    return i
	  }
	`)

	program, err := parser.ParseProgram(nil, script, parser.Config{})
	require.NoError(t, err)

	coverageReport := NewCoverageReport()

	location := stdlib.FlowLocation{}
	coverageReport.InspectProgram(location, program)

	actual, err := json.Marshal(coverageReport)
	require.NoError(t, err)

	expected := `
	  {
	    "coverage": {
	      "flow": {
	        "line_hits": {
	          "3": 0,
	          "4": 0,
	          "5": 0,
	          "7": 0
	        },
	        "missed_lines": [3, 4, 5, 7],
	        "statements": 4,
	        "percentage": "0.0%"
	      }
	    },
	    "excluded_locations": []
	  }
	`
	require.JSONEq(t, expected, string(actual))
}

func TestCoverageReportWithREPLLocation(t *testing.T) {

	t.Parallel()

	script := []byte(`
	  access(all) fun answer(): Int {
	    var i = 0
	    while i < 42 {
	      i = i + 1
	    }
	    return i
	  }
	`)

	program, err := parser.ParseProgram(nil, script, parser.Config{})
	require.NoError(t, err)

	coverageReport := NewCoverageReport()

	location := common.REPLLocation{}
	coverageReport.InspectProgram(location, program)

	actual, err := json.Marshal(coverageReport)
	require.NoError(t, err)

	expected := `
	  {
	    "coverage": {
	      "REPL": {
	        "line_hits": {
	          "3": 0,
	          "4": 0,
	          "5": 0,
	          "7": 0
	        },
	        "missed_lines": [3, 4, 5, 7],
	        "statements": 4,
	        "percentage": "0.0%"
	      }
	    },
	    "excluded_locations": []
	  }
	`
	require.JSONEq(t, expected, string(actual))
}

func TestCoverageReportWithScriptLocation(t *testing.T) {

	t.Parallel()

	script := []byte(`
	  access(all) fun answer(): Int {
	    var i = 0
	    while i < 42 {
	      i = i + 1
	    }
	    return i
	  }
	`)

	program, err := parser.ParseProgram(nil, script, parser.Config{})
	require.NoError(t, err)

	coverageReport := NewCoverageReport()

	location := common.ScriptLocation{0x1, 0x2}
	coverageReport.InspectProgram(location, program)

	actual, err := json.Marshal(coverageReport)
	require.NoError(t, err)

	expected := `
	  {
	    "coverage": {
	      "s.0102000000000000000000000000000000000000000000000000000000000000": {
	        "line_hits": {
	          "3": 0,
	          "4": 0,
	          "5": 0,
	          "7": 0
	        },
	        "missed_lines": [3, 4, 5, 7],
	        "statements": 4,
	        "percentage": "0.0%"
	      }
	    },
	    "excluded_locations": []
	  }
	`
	require.JSONEq(t, expected, string(actual))
}

func TestCoverageReportWithStringLocation(t *testing.T) {

	t.Parallel()

	script := []byte(`
	  access(all) fun answer(): Int {
	    var i = 0
	    while i < 42 {
	      i = i + 1
	    }
	    return i
	  }
	`)

	program, err := parser.ParseProgram(nil, script, parser.Config{})
	require.NoError(t, err)

	coverageReport := NewCoverageReport()

	location := common.StringLocation("AnswerScript")
	coverageReport.InspectProgram(location, program)

	actual, err := json.Marshal(coverageReport)
	require.NoError(t, err)

	expected := `
	  {
	    "coverage": {
	      "S.AnswerScript": {
	        "line_hits": {
	          "3": 0,
	          "4": 0,
	          "5": 0,
	          "7": 0
	        },
	        "missed_lines": [3, 4, 5, 7],
	        "statements": 4,
	        "percentage": "0.0%"
	      }
	    },
	    "excluded_locations": []
	  }
	`
	require.JSONEq(t, expected, string(actual))
}

func TestCoverageReportWithIdentifierLocation(t *testing.T) {

	t.Parallel()

	script := []byte(`
	  access(all) fun answer(): Int {
	    var i = 0
	    while i < 42 {
	      i = i + 1
	    }
	    return i
	  }
	`)

	program, err := parser.ParseProgram(nil, script, parser.Config{})
	require.NoError(t, err)

	coverageReport := NewCoverageReport()

	location := common.IdentifierLocation("Answer")
	coverageReport.InspectProgram(location, program)

	actual, err := json.Marshal(coverageReport)
	require.NoError(t, err)

	expected := `
	  {
	    "coverage": {
	      "I.Answer": {
	        "line_hits": {
	          "3": 0,
	          "4": 0,
	          "5": 0,
	          "7": 0
	        },
	        "missed_lines": [3, 4, 5, 7],
	        "statements": 4,
	        "percentage": "0.0%"
	      }
	    },
	    "excluded_locations": []
	  }
	`
	require.JSONEq(t, expected, string(actual))
}

func TestCoverageReportWithTransactionLocation(t *testing.T) {

	t.Parallel()

	script := []byte(`
	  access(all) fun answer(): Int {
	    var i = 0
	    while i < 42 {
	      i = i + 1
	    }
	    return i
	  }
	`)

	program, err := parser.ParseProgram(nil, script, parser.Config{})
	require.NoError(t, err)

	coverageReport := NewCoverageReport()

	location := common.TransactionLocation{0x1, 0x2}
	coverageReport.InspectProgram(location, program)

	actual, err := json.Marshal(coverageReport)
	require.NoError(t, err)

	expected := `
	  {
	    "coverage": {
	      "t.0102000000000000000000000000000000000000000000000000000000000000": {
	        "line_hits": {
	          "3": 0,
	          "4": 0,
	          "5": 0,
	          "7": 0
	        },
	        "missed_lines": [3, 4, 5, 7],
	        "statements": 4,
	        "percentage": "0.0%"
	      }
	    },
	    "excluded_locations": []
	  }
	`
	require.JSONEq(t, expected, string(actual))
}

func TestCoverageReportWithAddressLocation(t *testing.T) {

	t.Parallel()

	script := []byte(`
	  access(all) fun answer(): Int {
	    var i = 0
	    while i < 42 {
	      i = i + 1
	    }
	    return i
	  }
	`)

	program, err := parser.ParseProgram(nil, script, parser.Config{})
	require.NoError(t, err)

	coverageReport := NewCoverageReport()

	location := common.AddressLocation{
		Address: common.MustBytesToAddress([]byte{1, 2}),
		Name:    "Answer",
	}
	coverageReport.InspectProgram(location, program)

	actual, err := json.Marshal(coverageReport)
	require.NoError(t, err)

	expected := `
	  {
	    "coverage": {
	      "A.0000000000000102.Answer": {
	        "line_hits": {
	          "3": 0,
	          "4": 0,
	          "5": 0,
	          "7": 0
	        },
	        "missed_lines": [3, 4, 5, 7],
	        "statements": 4,
	        "percentage": "0.0%"
	      }
	    },
	    "excluded_locations": []
	  }
	`
	require.JSONEq(t, expected, string(actual))
}

func TestCoverageReportReset(t *testing.T) {

	t.Parallel()

	script := []byte(`
	  access(all) fun answer(): Int {
	    var i = 0
	    while i < 42 {
	      i = i + 1
	    }
	    return i
	  }
	`)

	program, err := parser.ParseProgram(nil, script, parser.Config{})
	require.NoError(t, err)

	coverageReport := NewCoverageReport()

	location := common.StringLocation("AnswerScript")
	coverageReport.InspectProgram(location, program)
	coverageReport.AddLineHit(location, 3)
	coverageReport.AddLineHit(location, 3)
	coverageReport.AddLineHit(location, 5)

	excludedLocation := common.StringLocation("XLocation")
	coverageReport.ExcludeLocation(excludedLocation)

	assert.Equal(t, 1, len(coverageReport.Coverage))
	assert.Equal(t, 1, len(coverageReport.Locations))
	assert.Equal(t, 1, len(coverageReport.ExcludedLocations))
	assert.Equal(t, true, coverageReport.IsLocationInspected(location))
	assert.Equal(t, true, coverageReport.IsLocationExcluded(excludedLocation))

	coverageReport.Reset()

	assert.Equal(t, 0, len(coverageReport.Coverage))
	assert.Equal(t, 0, len(coverageReport.Locations))
	assert.Equal(t, 1, len(coverageReport.ExcludedLocations))
	assert.Equal(t, false, coverageReport.IsLocationInspected(location))
	assert.Equal(t, true, coverageReport.IsLocationExcluded(excludedLocation))
}

func TestCoverageReportAddLineHitForExcludedLocation(t *testing.T) {

	t.Parallel()

	coverageReport := NewCoverageReport()

	location := common.StringLocation("AnswerScript")
	coverageReport.ExcludeLocation(location)

	coverageReport.AddLineHit(location, 3)
	coverageReport.AddLineHit(location, 5)

	assert.Equal(t, 0, len(coverageReport.Coverage))
	assert.Equal(t, 0, len(coverageReport.Locations))
	assert.Equal(t, false, coverageReport.IsLocationInspected(location))
}

func TestCoverageReportAddLineHitWithLocationFilter(t *testing.T) {

	t.Parallel()

	coverageReport := NewCoverageReport()
	coverageReport.WithLocationFilter(func(location common.Location) bool {
		_, addressLoc := location.(common.AddressLocation)
		_, stringLoc := location.(common.StringLocation)
		// We only allow inspection of AddressLocation or StringLocation
		return addressLoc || stringLoc
	})

	location := common.TransactionLocation{0x1a, 0x2b}
	coverageReport.AddLineHit(location, 3)
	coverageReport.AddLineHit(location, 5)

	assert.Equal(t, 0, len(coverageReport.Coverage))
	assert.Equal(t, 0, len(coverageReport.Locations))
	assert.Equal(t, false, coverageReport.IsLocationInspected(location))
}

func TestCoverageReportAddLineHitForNonInspectedProgram(t *testing.T) {

	t.Parallel()

	coverageReport := NewCoverageReport()

	location := common.StringLocation("AnswerScript")

	coverageReport.AddLineHit(location, 3)
	coverageReport.AddLineHit(location, 5)

	assert.Equal(t, 0, len(coverageReport.Coverage))
	assert.Equal(t, 0, len(coverageReport.Locations))
	assert.Equal(t, false, coverageReport.IsLocationInspected(location))
}

func TestCoverageReportPercentage(t *testing.T) {

	t.Parallel()

	script := []byte(`
	  access(all) fun answer(): Int {
	    var i = 0
	    while i < 42 {
	      i = i + 1
	    }
	    return i
	  }
	`)

	program, err := parser.ParseProgram(nil, script, parser.Config{})
	require.NoError(t, err)

	coverageReport := NewCoverageReport()

	location := common.StringLocation("AnswerScript")
	coverageReport.InspectProgram(location, program)
	coverageReport.AddLineHit(location, 3)
	coverageReport.AddLineHit(location, 4)

	actual, err := json.Marshal(coverageReport)
	require.NoError(t, err)

	expected := `
	  {
	    "coverage": {
	      "S.AnswerScript": {
	        "line_hits": {
	          "3": 1,
	          "4": 1,
	          "5": 0,
	          "7": 0
	        },
	        "missed_lines": [5, 7],
	        "statements": 4,
	        "percentage": "50.0%"
	      }
	    },
	    "excluded_locations": []
	  }
	`
	require.JSONEq(t, expected, string(actual))

	assert.Equal(t, "50.0%", coverageReport.Percentage())
}

func TestCoverageReportString(t *testing.T) {

	t.Parallel()

	script := []byte(`
	  access(all) fun answer(): Int {
	    var i = 0
	    while i < 42 {
	      i = i + 1
	    }
	    return i
	  }
	`)

	program, err := parser.ParseProgram(nil, script, parser.Config{})
	require.NoError(t, err)

	coverageReport := NewCoverageReport()

	location := common.StringLocation("AnswerScript")
	coverageReport.InspectProgram(location, program)
	coverageReport.AddLineHit(location, 3)
	coverageReport.AddLineHit(location, 4)
	coverageReport.AddLineHit(location, 5)

	actual, err := json.Marshal(coverageReport)
	require.NoError(t, err)

	expected := `
	  {
	    "coverage": {
	      "S.AnswerScript": {
	        "line_hits": {
	          "3": 1,
	          "4": 1,
	          "5": 1,
	          "7": 0
	        },
	        "missed_lines": [7],
	        "statements": 4,
	        "percentage": "75.0%"
	      }
	    },
	    "excluded_locations": []
	  }
	`
	require.JSONEq(t, expected, string(actual))

	assert.Equal(
		t,
		"Coverage: 75.0% of statements",
		coverageReport.String(),
	)
}

func TestCoverageReportDiff(t *testing.T) {

	t.Parallel()

	script := []byte(`
	  access(all) fun answer(): Int {
	    var i = 0
	    while i < 42 {
	      i = i + 1
	    }
	    return i
	  }
	`)

	program, err := parser.ParseProgram(nil, script, parser.Config{})
	require.NoError(t, err)

	coverageReport := NewCoverageReport()

	location := common.StringLocation("AnswerScript")
	coverageReport.InspectProgram(location, program)
	coverageReport.AddLineHit(location, 3)
	coverageReport.AddLineHit(location, 4)

	summary := coverageReport.Summary()

	actual, err := json.Marshal(summary)
	require.NoError(t, err)

	expected := `
	  {
	    "coverage": "50.0%",
	    "hits": 2,
	    "locations": 1,
	    "misses": 2,
	    "statements": 4
	  }
	`
	require.JSONEq(t, expected, string(actual))

	otherCoverageReport := NewCoverageReport()
	otherCoverageReport.InspectProgram(location, program)
	otherCoverageReport.AddLineHit(location, 3)
	otherCoverageReport.AddLineHit(location, 4)
	otherCoverageReport.AddLineHit(location, 5)
	otherCoverageReport.AddLineHit(location, 5)
	otherCoverageReport.AddLineHit(location, 7)

	diff := coverageReport.Diff(*otherCoverageReport)

	actual, err = json.Marshal(diff)
	require.NoError(t, err)

	expected = `
	  {
	    "coverage": "100.0%",
	    "hits": 2,
	    "locations": 0,
	    "misses": -2,
	    "statements": 0
	  }
	`
	require.JSONEq(t, expected, string(actual))
}

func TestCoverageReportMerge(t *testing.T) {

	t.Parallel()

	integerTraitsScript := []byte(`
	  access(all) let specialNumbers: {Int: String} = {
	    1729: "Harshad",
	    8128: "Harmonic",
	    41041: "Carmichael"
	  }

	  access(all) fun addSpecialNumber(_ n: Int, _ trait: String) {
	    specialNumbers[n] = trait
	  }

	  access(all) fun getIntegerTrait(_ n: Int): String {
	    if n < 0 {
	      return "Negative"
	    } else if n == 0 {
	      return "Zero"
	    } else if n < 10 {
	      return "Small"
	    } else if n < 100 {
	      return "Big"
	    } else if n < 1000 {
	      return "Huge"
	    }

	    if specialNumbers.containsKey(n) {
	      return specialNumbers[n]!
	    }

	    return "Enormous"
	  }
	`)

	program, err := parser.ParseProgram(nil, integerTraitsScript, parser.Config{})
	require.NoError(t, err)

	coverageReport := NewCoverageReport()

	location := common.StringLocation("IntegerTraits")
	coverageReport.InspectProgram(location, program)

	factorialScript := []byte(`
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

	otherProgram, err := parser.ParseProgram(nil, factorialScript, parser.Config{})
	require.NoError(t, err)

	otherCoverageReport := NewCoverageReport()

	otherLocation := common.StringLocation("Factorial")
	otherCoverageReport.InspectProgram(otherLocation, otherProgram)
	// We add `IntegerTraits` to both coverage reports, to test that their
	// line hits are properly merged.
	coverageReport.InspectProgram(location, program)
	coverageReport.AddLineHit(location, 9)

	excludedLocation := common.StringLocation("FooContract")
	otherCoverageReport.ExcludeLocation(excludedLocation)

	coverageReport.Merge(*otherCoverageReport)

	actual, err := json.Marshal(coverageReport)
	require.NoError(t, err)

	expected := `
	  {
	    "coverage": {
	      "S.Factorial": {
	        "line_hits": {
	          "12": 0,
	          "13": 0,
	          "16": 0,
	          "4": 0,
	          "8": 0
	        },
	        "missed_lines": [4, 8, 12, 13, 16],
	        "statements": 5,
	        "percentage": "0.0%"
	      },
	      "S.IntegerTraits": {
	        "line_hits": {
	          "13": 0,
	          "14": 0,
	          "15": 0,
	          "16": 0,
	          "17": 0,
	          "18": 0,
	          "19": 0,
	          "20": 0,
	          "21": 0,
	          "22": 0,
	          "25": 0,
	          "26": 0,
	          "29": 0,
	          "9": 1
	        },
	        "missed_lines": [13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 25, 26, 29],
	        "statements": 14,
	        "percentage": "7.1%"
	      }
	    },
	    "excluded_locations": ["S.FooContract"]
	  }
	`
	require.JSONEq(t, expected, string(actual))
}

func TestCoverageReportUnmarshalJSON(t *testing.T) {

	t.Parallel()

	data := `
	  {
	    "coverage": {
	      "S.Factorial": {
	        "line_hits": {
	          "12": 0,
	          "13": 0,
	          "16": 0,
	          "4": 0,
	          "8": 0
	        },
	        "missed_lines": [4, 8, 12, 13, 16],
	        "statements": 5,
	        "percentage": "0.0%"
	      },
	      "S.IntegerTraits": {
	        "line_hits": {
	          "13": 0,
	          "14": 0,
	          "15": 0,
	          "16": 0,
	          "17": 0,
	          "18": 0,
	          "19": 0,
	          "20": 0,
	          "21": 0,
	          "22": 0,
	          "25": 0,
	          "26": 0,
	          "29": 0,
	          "9": 0
	        },
	        "missed_lines": [9, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 25, 26, 29],
	        "statements": 14,
	        "percentage": "0.0%"
	      }
	    },
	    "excluded_locations": ["I.Test"]
	  }
	`

	coverageReport := NewCoverageReport()
	err := json.Unmarshal([]byte(data), coverageReport)
	require.NoError(t, err)

	assert.Equal(t, 2, coverageReport.TotalLocations())

	factorialLocation := common.StringLocation("Factorial")

	assert.Equal(
		t,
		5,
		coverageReport.Coverage[factorialLocation].Statements,
	)
	assert.Equal(
		t,
		"0.0%",
		coverageReport.Coverage[factorialLocation].Percentage(),
	)
	assert.EqualValues(
		t,
		[]int{4, 8, 12, 13, 16},
		coverageReport.Coverage[factorialLocation].MissedLines(),
	)
	assert.Equal(
		t,
		map[int]int{4: 0, 8: 0, 12: 0, 13: 0, 16: 0},
		coverageReport.Coverage[factorialLocation].LineHits,
	)

	actual, err := json.Marshal(coverageReport)
	require.NoError(t, err)

	require.JSONEq(t, data, string(actual))

	integerTraitsLocation := common.StringLocation("IntegerTraits")

	assert.Equal(
		t,
		coverageReport.Coverage[integerTraitsLocation].Statements,
		14,
	)
	assert.Equal(
		t,
		"0.0%",
		coverageReport.Coverage[integerTraitsLocation].Percentage(),
	)
	assert.EqualValues(
		t,
		[]int{9, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 25, 26, 29},
		coverageReport.Coverage[integerTraitsLocation].MissedLines(),
	)
	assert.EqualValues(
		t,
		map[int]int{9: 0, 13: 0, 14: 0, 15: 0, 16: 0, 17: 0, 18: 0, 19: 0, 20: 0, 21: 0, 22: 0, 25: 0, 26: 0, 29: 0},
		coverageReport.Coverage[integerTraitsLocation].LineHits,
	)

	assert.Equal(
		t,
		2,
		len(coverageReport.Locations),
	)

	assert.Equal(
		t,
		[]string{"I.Test"},
		coverageReport.ExcludedLocationIDs(),
	)
}

func TestCoverageReportUnmarshalJSONWithFormatError(t *testing.T) {

	t.Parallel()

	data := "My previous coverage report.txt"

	coverageReport := NewCoverageReport()
	err := coverageReport.UnmarshalJSON([]byte(data))
	require.Error(t, err)
}

func TestCoverageReportUnmarshalJSONWithDecodeLocationError(t *testing.T) {

	t.Parallel()

	data := `
	  {
	    "coverage": {
	      "X.Factorial": {
	        "line_hits": {
	          "12": 0,
	          "13": 0,
	          "16": 0,
	          "4": 0,
	          "8": 0
	        },
	        "missed_lines": [4, 8, 12, 13, 16],
	        "statements": 5,
	        "percentage": "0.0%"
	      }
	    },
	    "excluded_locations": ["I.Test"]
	  }
	`

	coverageReport := NewCoverageReport()
	err := json.Unmarshal([]byte(data), coverageReport)
	require.ErrorContains(t, err, "invalid Location ID: X.Factorial")
}

func TestCoverageReportUnmarshalJSONWithDecodeExcludedLocationError(t *testing.T) {

	t.Parallel()

	data := `
	  {
	    "coverage": {
	      "S.Factorial": {
	        "line_hits": {
	          "12": 0,
	          "13": 0,
	          "16": 0,
	          "4": 0,
	          "8": 0
	        },
	        "missed_lines": [4, 8, 12, 13, 16],
	        "statements": 5,
	        "percentage": "0.0%"
	      }
	    },
	    "excluded_locations": ["XI.Test"]
	  }
	`

	coverageReport := NewCoverageReport()
	err := json.Unmarshal([]byte(data), coverageReport)
	require.ErrorContains(t, err, "invalid Location ID: XI.Test")
}

func TestRuntimeCoverage(t *testing.T) {

	t.Parallel()

	importedScript := []byte(`
	  access(all) let specialNumbers: {Int: String} = {
	    1729: "Harshad",
	    8128: "Harmonic",
	    41041: "Carmichael"
	  }

	  access(all) fun addSpecialNumber(_ n: Int, _ trait: String) {
	    specialNumbers[n] = trait
	  }

	  access(all) fun getIntegerTrait(_ n: Int): String {
	    if n < 0 {
	      return "Negative"
	    } else if n == 0 {
	      return "Zero"
	    } else if n < 10 {
	      return "Small"
	    } else if n < 100 {
	      return "Big"
	    } else if n < 1000 {
	      return "Huge"
	    }

	    if specialNumbers.containsKey(n) {
	      return specialNumbers[n]!
	    }

	    return "Enormous"
	  }

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
	    let testInputs: {Int: String} = {
	      -1: "Negative",
	      0: "Zero",
	      9: "Small",
	      99: "Big",
	      999: "Huge",
	      1001: "Enormous",
	      1729: "Harshad",
	      8128: "Harmonic",
	      41041: "Carmichael"
	    }

	    for input in testInputs.keys {
	      let result = getIntegerTrait(input)
	      assert(result == testInputs[input])
	    }

	    addSpecialNumber(78557, "Sierpinski")
	    assert("Sierpinski" == getIntegerTrait(78557))

	    factorial(5)
	    factorial(0)

	    return 42
	  }
	`)

	coverageReport := NewCoverageReport()
	runtimeInterface := &testRuntimeInterface{
		getCode: func(location Location) (bytes []byte, err error) {
			switch location {
			case common.StringLocation("imported"):
				return importedScript, nil
			default:
				return nil, fmt.Errorf("unknown import location: %s", location)
			}
		},
	}

	runtime := newTestInterpreterRuntime()
	runtime.defaultConfig.CoverageReport = coverageReport

	value, err := runtime.ExecuteScript(
		Script{
			Source: script,
		},
		Context{
			Interface:      runtimeInterface,
			Location:       common.ScriptLocation{},
			CoverageReport: coverageReport,
		},
	)
	require.NoError(t, err)

	assert.Equal(t, cadence.NewInt(42), value)

	actual, err := json.Marshal(coverageReport)
	require.NoError(t, err)

	expected := `
	  {
	    "coverage": {
	      "S.imported": {
	        "line_hits": {
	          "13": 10,
	          "14": 1,
	          "15": 9,
	          "16": 1,
	          "17": 8,
	          "18": 1,
	          "19": 7,
	          "20": 1,
	          "21": 6,
	          "22": 1,
	          "25": 5,
	          "26": 4,
	          "29": 1,
	          "34": 7,
	          "38": 7,
	          "42": 7,
	          "43": 2,
	          "46": 5,
	          "9": 1
	        },
	        "missed_lines": [],
	        "statements": 19,
	        "percentage": "100.0%"
	      },
	      "s.0000000000000000000000000000000000000000000000000000000000000000": {
	        "line_hits": {
	          "17": 1,
	          "18": 9,
	          "19": 9,
	          "22": 1,
	          "23": 1,
	          "25": 1,
	          "26": 1,
	          "28": 1,
	          "5": 1
	        },
	        "missed_lines": [],
	        "statements": 9,
	        "percentage": "100.0%"
	      }
	    },
	    "excluded_locations": []
	  }
	`
	require.JSONEq(t, expected, string(actual))

	assert.Equal(
		t,
		"Coverage: 100.0% of statements",
		coverageReport.String(),
	)
}

func TestRuntimeCoverageWithExcludedLocation(t *testing.T) {

	t.Parallel()

	importedScript := []byte(`
	  access(all) let specialNumbers: {Int: String} = {
	    1729: "Harshad",
	    8128: "Harmonic",
	    41041: "Carmichael"
	  }

	  access(all) fun addSpecialNumber(_ n: Int, _ trait: String) {
	    specialNumbers[n] = trait
	  }

	  access(all) fun getIntegerTrait(_ n: Int): String {
	    if n < 0 {
	      return "Negative"
	    } else if n == 0 {
	      return "Zero"
	    } else if n < 10 {
	      return "Small"
	    } else if n < 100 {
	      return "Big"
	    } else if n < 1000 {
	      return "Huge"
	    }

	    if specialNumbers.containsKey(n) {
	      return specialNumbers[n]!
	    }

	    return "Enormous"
	  }
	`)

	script := []byte(`
	  import "imported"

	  access(all) fun main(): Int {
	    let testInputs: {Int: String} = {
	      -1: "Negative",
	      0: "Zero",
	      9: "Small",
	      99: "Big",
	      999: "Huge",
	      1001: "Enormous",
	      1729: "Harshad",
	      8128: "Harmonic",
	      41041: "Carmichael"
	    }

	    for input in testInputs.keys {
	      let result = getIntegerTrait(input)
	      assert(result == testInputs[input])
	    }

	    addSpecialNumber(78557, "Sierpinski")
	    assert("Sierpinski" == getIntegerTrait(78557))

	    return 42
	  }
	`)

	coverageReport := NewCoverageReport()
	scriptlocation := common.ScriptLocation{}
	coverageReport.ExcludeLocation(scriptlocation)

	runtimeInterface := &testRuntimeInterface{
		getCode: func(location Location) (bytes []byte, err error) {
			switch location {
			case common.StringLocation("imported"):
				return importedScript, nil
			default:
				return nil, fmt.Errorf("unknown import location: %s", location)
			}
		},
	}

	runtime := newTestInterpreterRuntime()
	runtime.defaultConfig.CoverageReport = coverageReport

	value, err := runtime.ExecuteScript(
		Script{
			Source: script,
		},
		Context{
			Interface:      runtimeInterface,
			Location:       scriptlocation,
			CoverageReport: coverageReport,
		},
	)
	require.NoError(t, err)

	assert.Equal(t, cadence.NewInt(42), value)

	actual, err := json.Marshal(coverageReport)
	require.NoError(t, err)

	expected := `
	  {
	    "coverage": {
	      "S.imported": {
	        "line_hits": {
	          "13": 10,
	          "14": 1,
	          "15": 9,
	          "16": 1,
	          "17": 8,
	          "18": 1,
	          "19": 7,
	          "20": 1,
	          "21": 6,
	          "22": 1,
	          "25": 5,
	          "26": 4,
	          "29": 1,
	          "9": 1
	        },
	        "missed_lines": [],
	        "statements": 14,
	        "percentage": "100.0%"
	      }
	    },
	    "excluded_locations": ["s.0000000000000000000000000000000000000000000000000000000000000000"]
	  }
	`
	require.JSONEq(t, expected, string(actual))

	assert.Equal(
		t,
		"Coverage: 100.0% of statements",
		coverageReport.String(),
	)
}

func TestRuntimeCoverageWithLocationFilter(t *testing.T) {

	t.Parallel()

	importedScript := []byte(`
	  access(all) let specialNumbers: {Int: String} = {
	    1729: "Harshad",
	    8128: "Harmonic",
	    41041: "Carmichael"
	  }

	  access(all) fun addSpecialNumber(_ n: Int, _ trait: String) {
	    specialNumbers[n] = trait
	  }

	  access(all) fun getIntegerTrait(_ n: Int): String {
	    if n < 0 {
	      return "Negative"
	    } else if n == 0 {
	      return "Zero"
	    } else if n < 10 {
	      return "Small"
	    } else if n < 100 {
	      return "Big"
	    } else if n < 1000 {
	      return "Huge"
	    }

	    if specialNumbers.containsKey(n) {
	      return specialNumbers[n]!
	    }

	    return "Enormous"
	  }
	`)

	script := []byte(`
	  import "imported"

	  access(all) fun main(): Int {
	    let testInputs: {Int: String} = {
	      -1: "Negative",
	      0: "Zero",
	      9: "Small",
	      99: "Big",
	      999: "Huge",
	      1001: "Enormous",
	      1729: "Harshad",
	      8128: "Harmonic",
	      41041: "Carmichael"
	    }

	    for input in testInputs.keys {
	      let result = getIntegerTrait(input)
	      assert(result == testInputs[input])
	    }

	    addSpecialNumber(78557, "Sierpinski")
	    assert("Sierpinski" == getIntegerTrait(78557))

	    return 42
	  }
	`)

	coverageReport := NewCoverageReport()
	coverageReport.WithLocationFilter(func(location common.Location) bool {
		_, addressLoc := location.(common.AddressLocation)
		_, stringLoc := location.(common.StringLocation)
		// We only allow inspection of AddressLocation or StringLocation
		return addressLoc || stringLoc
	})
	scriptlocation := common.ScriptLocation{0x1b, 0x2c}

	runtimeInterface := &testRuntimeInterface{
		getCode: func(location Location) (bytes []byte, err error) {
			switch location {
			case common.StringLocation("imported"):
				return importedScript, nil
			default:
				return nil, fmt.Errorf("unknown import location: %s", location)
			}
		},
	}
	runtime := NewInterpreterRuntime(Config{
		CoverageReport: coverageReport,
	})

	value, err := runtime.ExecuteScript(
		Script{
			Source: script,
		},
		Context{
			Interface:      runtimeInterface,
			Location:       scriptlocation,
			CoverageReport: coverageReport,
		},
	)
	require.NoError(t, err)

	assert.Equal(t, cadence.NewInt(42), value)

	actual, err := json.Marshal(coverageReport)
	require.NoError(t, err)

	expected := `
	  {
	    "coverage": {
	      "S.imported": {
	        "line_hits": {
	          "13": 10,
	          "14": 1,
	          "15": 9,
	          "16": 1,
	          "17": 8,
	          "18": 1,
	          "19": 7,
	          "20": 1,
	          "21": 6,
	          "22": 1,
	          "25": 5,
	          "26": 4,
	          "29": 1,
	          "9": 1
	        },
	        "missed_lines": [],
	        "statements": 14,
	        "percentage": "100.0%"
	      }
	    },
	    "excluded_locations": []
	  }
	`
	require.JSONEq(t, expected, string(actual))

	assert.Equal(
		t,
		"Coverage: 100.0% of statements",
		coverageReport.String(),
	)
}

func TestRuntimeCoverageWithNoStatements(t *testing.T) {

	t.Parallel()

	importedScript := []byte(`
	  access(all) contract FooContract {
	    access(all) resource interface Receiver {
	    }
	  }
	`)

	script := []byte(`
	  import "FooContract"
	  access(all) fun main(): Int {
		Type<@{FooContract.Receiver}>().identifier
		return 42
	  }
	`)

	coverageReport := NewCoverageReport()

	scriptlocation := common.ScriptLocation{0x1b, 0x2c}

	runtimeInterface := &testRuntimeInterface{
		getCode: func(location Location) (bytes []byte, err error) {
			switch location {
			case common.StringLocation("FooContract"):
				return importedScript, nil
			default:
				return nil, fmt.Errorf("unknown import location: %s", location)
			}
		},
	}
	runtime := NewInterpreterRuntime(Config{
		CoverageReport: coverageReport,
	})
	coverageReport.ExcludeLocation(scriptlocation)
	value, err := runtime.ExecuteScript(
		Script{
			Source: script,
		},
		Context{
			Interface:      runtimeInterface,
			Location:       scriptlocation,
			CoverageReport: coverageReport,
		},
	)
	require.NoError(t, err)

	assert.Equal(t, cadence.NewInt(42), value)

	_, err = json.Marshal(coverageReport)
	require.NoError(t, err)

	assert.Equal(
		t,
		"There are no statements to cover",
		coverageReport.String(),
	)

	summary := coverageReport.Summary()

	actual, err := json.Marshal(summary)
	require.NoError(t, err)

	expected := `
	  {
	    "coverage": "100.0%",
	    "hits": 0,
	    "locations": 0,
	    "misses": 0,
	    "statements": 0
	  }
	`
	require.JSONEq(t, expected, string(actual))
}

func TestCoverageReportLCOVFormat(t *testing.T) {

	t.Parallel()

	integerTraits := []byte(`
	  access(all) let specialNumbers: {Int: String} = {
	    1729: "Harshad",
	    8128: "Harmonic",
	    41041: "Carmichael"
	  }

	  access(all) fun addSpecialNumber(_ n: Int, _ trait: String) {
	    specialNumbers[n] = trait
	  }

	  access(all) fun getIntegerTrait(_ n: Int): String {
	    if n < 0 {
	      return "Negative"
	    } else if n == 0 {
	      return "Zero"
	    } else if n < 10 {
	      return "Small"
	    } else if n < 100 {
	      return "Big"
	    } else if n < 1000 {
	      return "Huge"
	    }

	    if specialNumbers.containsKey(n) {
	      return specialNumbers[n]!
	    }

	    return "Enormous"
	  }
	`)

	script := []byte(`
	  import "IntegerTraits"

	  access(all) fun main(): Int {
	    let testInputs: {Int: String} = {
	      -1: "Negative",
	      0: "Zero",
	      9: "Small",
	      99: "Big",
	      999: "Huge",
	      1001: "Enormous",
	      1729: "Harshad",
	      8128: "Harmonic",
	      41041: "Carmichael"
	    }

	    for input in testInputs.keys {
	      let result = getIntegerTrait(input)
	      assert(result == testInputs[input])
	    }

	    addSpecialNumber(78557, "Sierpinski")
	    assert("Sierpinski" == getIntegerTrait(78557))

	    return 42
	  }
	`)

	coverageReport := NewCoverageReport()
	scriptlocation := common.ScriptLocation{}
	coverageReport.ExcludeLocation(scriptlocation)

	runtimeInterface := &testRuntimeInterface{
		getCode: func(location Location) (bytes []byte, err error) {
			switch location {
			case common.StringLocation("IntegerTraits"):
				return integerTraits, nil
			default:
				return nil, fmt.Errorf("unknown import location: %s", location)
			}
		},
	}

	runtime := newTestInterpreterRuntime()
	runtime.defaultConfig.CoverageReport = coverageReport

	value, err := runtime.ExecuteScript(
		Script{
			Source: script,
		},
		Context{
			Interface:      runtimeInterface,
			Location:       scriptlocation,
			CoverageReport: coverageReport,
		},
	)
	require.NoError(t, err)

	assert.Equal(t, cadence.NewInt(42), value)

	actual, err := coverageReport.MarshalLCOV()
	require.NoError(t, err)

	expected := `TN:
SF:S.IntegerTraits
DA:9,1
DA:13,10
DA:14,1
DA:15,9
DA:16,1
DA:17,8
DA:18,1
DA:19,7
DA:20,1
DA:21,6
DA:22,1
DA:25,5
DA:26,4
DA:29,1
LF:14
LH:14
end_of_record
`
	require.Equal(t, expected, string(actual))

	assert.Equal(
		t,
		"Coverage: 100.0% of statements",
		coverageReport.String(),
	)
}
