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
	"sort"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
)

// LocationCoverage records coverage information for a location.
type LocationCoverage struct {
	// Contains hit count for each line on a given location.
	// A hit count of 0 means the line was not covered.
	LineHits map[int]int
	// Total number of statements on a given location.
	Statements int
}

// AddLineHit increments the hit count for the given line.
func (c *LocationCoverage) AddLineHit(line int) {
	// Lines below 1 are dropped.
	if line < 1 {
		return
	}
	c.LineHits[line]++
}

// Percentage returns a string representation of the covered
// statements percentage. It is defined as the ratio of covered
// lines over the total statements for a given location.
func (c *LocationCoverage) Percentage() string {
	coveredLines := c.CoveredLines()
	// The ground truth of which statements are interpreted/executed
	// is the `interpreterEnvironment.newOnStatementHandler()` function.
	// This means that every call of `CoverageReport.AddLineHit()` from
	// that callback, should be respected. The `CoverageReport.InspectProgram()`
	// may have failed, for whatever reason, to find a specific line.
	// This is a good insight to solidify its implementation and debug
	// the inspection failure. Ideally, this condition will never be true,
	// except for tests. We just leave it here, as a fail-safe mechanism.
	if coveredLines > c.Statements {
		// We saturate the percentage at 100%, when the inspector
		// fails to correctly count all statements for a given
		// location.
		coveredLines = c.Statements
	}
	percentage := 100 * float64(coveredLines) / float64(c.Statements)
	return fmt.Sprintf("%0.1f%%", percentage)
}

// CoveredLines returns the count of covered lines for a given location.
// This is the number of lines with a hit count > 0.
func (c *LocationCoverage) CoveredLines() int {
	coveredLines := 0
	for _, hits := range c.LineHits { // nolint:maprange
		if hits > 0 {
			coveredLines += 1
		}
	}
	return coveredLines
}

// MissedLines returns an array with the missed lines for a given location.
// These are all the lines with a hit count == 0. The resulting array is
// sorted in ascending order.
func (c *LocationCoverage) MissedLines() []int {
	missedLines := make([]int, 0)
	for line, hits := range c.LineHits { // nolint:maprange
		if hits == 0 {
			missedLines = append(missedLines, line)
		}
	}
	sort.Ints(missedLines)
	return missedLines
}

// NewLocationCoverage creates and returns a *LocationCoverage with the
// given lineHits map.
func NewLocationCoverage(lineHits map[int]int) *LocationCoverage {
	return &LocationCoverage{
		LineHits:   lineHits,
		Statements: len(lineHits),
	}
}

// CoverageReport collects coverage information per location.
// It keeps track of inspected programs per location, and can
// also exclude locations from coverage collection.
type CoverageReport struct {
	// Contains a *LocationCoverage per location.
	Coverage map[common.Location]*LocationCoverage `json:"-"`
	// Contains an *ast.Program per location.
	Programs map[common.Location]*ast.Program `json:"-"`
	// Contains locations excluded from coverage collection.
	ExcludedLocations map[common.Location]struct{} `json:"-"`
}

// ExcludeLocation adds the given location to the map of excluded
// locations.
func (r *CoverageReport) ExcludeLocation(location Location) {
	r.ExcludedLocations[location] = struct{}{}
}

// IsLocationExcluded checks whether the given location is excluded
// or not, from coverage collection.
func (r *CoverageReport) IsLocationExcluded(location Location) bool {
	_, ok := r.ExcludedLocations[location]
	return ok
}

// AddLineHit increments the hit count for the given line, on the given
// location. The method call is a NO-OP in two cases:
// - If the location is excluded from coverage collection
// - If the location's *ast.Program, has not been inspected
func (r *CoverageReport) AddLineHit(location Location, line int) {
	if r.IsLocationExcluded(location) {
		return
	}

	if !r.IsProgramInspected(location) {
		return
	}

	locationCoverage := r.Coverage[location]
	locationCoverage.AddLineHit(line)
}

// InspectProgram inspects the elements of the given *ast.Program, and counts its
// statements. If inspection is successful, the *ast.Program is marked as inspected.
// If the given location is excluded from coverage collection, the method call
// results in a NO-OP.
func (r *CoverageReport) InspectProgram(location Location, program *ast.Program) {
	if r.IsLocationExcluded(location) {
		return
	}
	r.Programs[location] = program
	lineHits := make(map[int]int, 0)
	recordLine := func(hasPosition ast.HasPosition) {
		line := hasPosition.StartPosition().Line
		lineHits[line] = 0
	}
	var depth int

	inspector := ast.NewInspector(program)
	inspector.Elements(
		nil, func(element ast.Element, push bool) bool {
			if push {
				depth++

				_, isStatement := element.(ast.Statement)
				_, isDeclaration := element.(ast.Declaration)
				_, isVariableDeclaration := element.(*ast.VariableDeclaration)

				// Track only the statements that are not declarations, such as:
				// - *ast.CompositeDeclaration
				// - *ast.SpecialFunctionDeclaration
				// - *ast.FunctionDeclaration
				// However, also track local (i.e. non-top level) variable declarations.
				if (isStatement && !isDeclaration) ||
					(isVariableDeclaration && depth > 2) {
					recordLine(element)
				}

				functionBlock, isFunctionBlock := element.(*ast.FunctionBlock)
				// Track also pre/post conditions defined inside functions.
				if isFunctionBlock {
					if functionBlock.PreConditions != nil {
						for _, condition := range *functionBlock.PreConditions {
							recordLine(condition.Test)
						}
					}
					if functionBlock.PostConditions != nil {
						for _, condition := range *functionBlock.PostConditions {
							recordLine(condition.Test)
						}
					}
				}
			} else {
				depth--
			}

			return true
		})

	locationCoverage := NewLocationCoverage(lineHits)
	r.Coverage[location] = locationCoverage
}

// IsProgramInspected checks whether the *ast.Program on the given
// location, has been inspected or not.
func (r *CoverageReport) IsProgramInspected(location Location) bool {
	_, isInspected := r.Programs[location]
	return isInspected
}

// Percentage returns a string representation of the covered statements
// percentage. It is defined as the ratio of total covered lines over
// total statements, for all locations.
func (r *CoverageReport) Percentage() string {
	totalStatements := 0
	totalCoveredLines := 0
	for _, locationCoverage := range r.Coverage { // nolint:maprange
		totalStatements += locationCoverage.Statements
		totalCoveredLines += locationCoverage.CoveredLines()
	}
	return fmt.Sprintf(
		"%0.1f%%",
		100*float64(totalCoveredLines)/float64(totalStatements),
	)
}

// String returns a human-friendly message for the covered
// statements percentage.
func (r *CoverageReport) String() string {
	return fmt.Sprintf("Coverage: %v of statements", r.Percentage())
}

// Reset flushes the collected coverage information for all locations
// and inspected programs. Excluded locations remain intact.
func (r *CoverageReport) Reset() {
	for location := range r.Coverage { // nolint:maprange
		delete(r.Coverage, location)
	}
	for location := range r.Programs { // nolint:maprange
		delete(r.Programs, location)
	}
}

// Merge adds all the collected coverage information to the
// calling object. Excluded locations are also taken into
// account.
func (r *CoverageReport) Merge(other CoverageReport) {
	for location, locationCoverage := range other.Coverage { // nolint:maprange
		r.Coverage[location] = locationCoverage
	}
	for location, program := range other.Programs { // nolint:maprange
		r.Programs[location] = program
	}
	for location := range other.ExcludedLocations {
		r.ExcludedLocations[location] = struct{}{}
	}
}

// NewCoverageReport creates and returns a *CoverageReport.
func NewCoverageReport() *CoverageReport {
	return &CoverageReport{
		Coverage:          map[common.Location]*LocationCoverage{},
		Programs:          map[common.Location]*ast.Program{},
		ExcludedLocations: map[common.Location]struct{}{},
	}
}

// MarshalJSON serializes each common.Location/*LocationCoverage
// key/value pair on the *CoverageReport.Coverage map.
func (r *CoverageReport) MarshalJSON() ([]byte, error) {
	type Alias CoverageReport

	// To avoid the overhead of having the Percentage & MissedLines
	// as fields in the LocationCoverage struct, we simply populate
	// this LC struct, with the corresponding methods, upon marshalling.
	type LC struct {
		LineHits    map[int]int `json:"line_hits"`
		MissedLines []int       `json:"missed_lines"`
		Statements  int         `json:"statements"`
		Percentage  string      `json:"percentage"`
	}

	coverage := make(map[string]LC, len(r.Coverage))
	for location, locationCoverage := range r.Coverage { // nolint:maprange
		coverage[location.ID()] = LC{
			LineHits:    locationCoverage.LineHits,
			MissedLines: locationCoverage.MissedLines(),
			Statements:  locationCoverage.Statements,
			Percentage:  locationCoverage.Percentage(),
		}
	}
	return json.Marshal(&struct {
		Coverage map[string]LC `json:"coverage"`
		*Alias
	}{
		Coverage: coverage,
		Alias:    (*Alias)(r),
	})
}
