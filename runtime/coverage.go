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
	"bytes"
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

type LocationFilter func(location Location) bool

// CoverageReport collects coverage information per location.
// It keeps track of inspected locations, and can also exclude
// locations from coverage collection.
type CoverageReport struct {
	// Contains a *LocationCoverage per location.
	Coverage map[common.Location]*LocationCoverage `json:"-"`
	// Contains locations whose programs are already inspected.
	Locations map[common.Location]struct{} `json:"-"`
	// Contains locations excluded from coverage collection.
	ExcludedLocations map[common.Location]struct{} `json:"-"`
	// This filter can be used to inject custom logic on
	// each location/program inspection.
	LocationFilter LocationFilter `json:"-"`
}

// WithLocationFilter sets the LocationFilter for the current
// CoverageReport.
func (r *CoverageReport) WithLocationFilter(
	locationFilter LocationFilter,
) {
	r.LocationFilter = locationFilter
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
// - If the location has not been inspected for its statements
func (r *CoverageReport) AddLineHit(location Location, line int) {
	if r.IsLocationExcluded(location) {
		return
	}

	if !r.IsLocationInspected(location) {
		return
	}

	locationCoverage := r.Coverage[location]
	locationCoverage.AddLineHit(line)
}

// InspectProgram inspects the elements of the given *ast.Program, and counts its
// statements. If inspection is successful, the location is marked as inspected.
// If the given location is excluded from coverage collection, the method call
// results in a NO-OP.
// If the CoverageReport.LocationFilter is present, and calling it with the given
// location results to false, the method call also results in a NO-OP.
func (r *CoverageReport) InspectProgram(location Location, program *ast.Program) {
	if r.LocationFilter != nil && !r.LocationFilter(location) {
		return
	}
	if r.IsLocationExcluded(location) {
		return
	}
	r.Locations[location] = struct{}{}
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

	r.Coverage[location] = NewLocationCoverage(lineHits)
}

// IsLocationInspected checks whether the given location,
// has been inspected or not.
func (r *CoverageReport) IsLocationInspected(location Location) bool {
	_, isInspected := r.Locations[location]
	return isInspected
}

// Percentage returns a string representation of the covered statements
// percentage. It is defined as the ratio of total covered lines over
// total statements, for all locations.
func (r *CoverageReport) Percentage() string {
	totalStatements := r.Statements()
	totalCoveredLines := r.Hits()
	var percentage float64 = 100
	if totalStatements != 0 {
		percentage = 100 * float64(totalCoveredLines) / float64(totalStatements)
	}
	return fmt.Sprintf(
		"%0.1f%%",
		percentage,
	)
}

// String returns a human-friendly message for the covered
// statements percentage.
func (r *CoverageReport) String() string {
	if r.Statements() == 0 {
		return "There are no statements to cover"
	}
	return fmt.Sprintf("Coverage: %v of statements", r.Percentage())
}

// Reset flushes the collected coverage information for all locations
// and inspected locations. Excluded locations remain intact.
func (r *CoverageReport) Reset() {
	for location := range r.Coverage { // nolint:maprange
		delete(r.Coverage, location)
	}
	for location := range r.Locations { // nolint:maprange
		delete(r.Locations, location)
	}
}

// Merge adds all the collected coverage information to the
// calling object. Excluded locations are also taken into
// account.
func (r *CoverageReport) Merge(other CoverageReport) {
	for location, locationCoverage := range other.Coverage { // nolint:maprange
		r.Coverage[location] = locationCoverage
	}
	for location, v := range other.Locations { // nolint:maprange
		r.Locations[location] = v
	}
	for location, v := range other.ExcludedLocations { // nolint:maprange
		r.ExcludedLocations[location] = v
	}
}

// ExcludedLocationIDs returns the ID of each excluded location. This
// is helpful in order to marshal/unmarshal a CoverageReport, without
// losing any valuable information.
func (r *CoverageReport) ExcludedLocationIDs() []string {
	excludedLocationIDs := make([]string, 0, len(r.ExcludedLocations))
	for location := range r.ExcludedLocations { // nolint:maprange
		excludedLocationIDs = append(excludedLocationIDs, location.ID())
	}
	return excludedLocationIDs
}

// TotalLocations returns the count of locations included in
// the CoverageReport. This implies that these locations are:
// - inspected,
// - not marked as exlucded.
func (r *CoverageReport) TotalLocations() int {
	return len(r.Coverage)
}

// Statements returns the total count of statements, for all the
// locations included in the CoverageReport.
func (r *CoverageReport) Statements() int {
	totalStatements := 0
	for _, locationCoverage := range r.Coverage { // nolint:maprange
		totalStatements += locationCoverage.Statements
	}
	return totalStatements
}

// Hits returns the total count of covered lines, for all the
// locations included in the CoverageReport.
func (r *CoverageReport) Hits() int {
	totalCoveredLines := 0
	for _, locationCoverage := range r.Coverage { // nolint:maprange
		totalCoveredLines += locationCoverage.CoveredLines()
	}
	return totalCoveredLines
}

// Misses returns the total count of non-covered lines, for all
// the locations included in the CoverageReport.
func (r *CoverageReport) Misses() int {
	return r.Statements() - r.Hits()
}

// Summary returns a CoverageReportSummary object, containing
// key metrics for a CoverageReport, such as:
// - Total Locations,
// - Total Statements,
// - Total Hits,
// - Total Misses,
// - Overall Coverage Percentage.
func (r *CoverageReport) Summary() CoverageReportSummary {
	return CoverageReportSummary{
		Locations:  r.TotalLocations(),
		Statements: r.Statements(),
		Hits:       r.Hits(),
		Misses:     r.Misses(),
		Coverage:   r.Percentage(),
	}
}

// Diff computes the incremental diff between the calling object and
// a new CoverageReport. The returned result is a CoverageReportSummary
// object.
//
//	CoverageReportSummary{
//		Locations:  0,
//		Statements: 0,
//		Hits:       2,
//		Misses:     -2,
//		Coverage:   "100.0%",
//	}
//
// The above diff is interpreted as follows:
// - No diff in locations,
// - No diff in statements,
// - Hits increased by 2,
// - Misses decreased by 2,
// - Coverage Δ increased by 100.0%.
func (r *CoverageReport) Diff(other CoverageReport) CoverageReportSummary {
	baseCoverage := 100 * float64(r.Hits()) / float64(r.Statements())
	newCoverage := 100 * float64(other.Hits()) / float64(other.Statements())
	coverageDelta := fmt.Sprintf(
		"%0.1f%%",
		100*(newCoverage-baseCoverage)/baseCoverage,
	)
	return CoverageReportSummary{
		Locations:  other.TotalLocations() - r.TotalLocations(),
		Statements: other.Statements() - r.Statements(),
		Hits:       other.Hits() - r.Hits(),
		Misses:     other.Misses() - r.Misses(),
		Coverage:   coverageDelta,
	}
}

// CoverageReportSummary contains key metrics that are derived
// from a CoverageReport object, such as:
// - Total Locations,
// - Total Statements,
// - Total Hits,
// - Total Misses,
// - Overall Coverage Percentage.
// This metrics can be utilized in various ways, such as a CI
// plugin/app.
type CoverageReportSummary struct {
	Locations  int    `json:"locations"`
	Statements int    `json:"statements"`
	Hits       int    `json:"hits"`
	Misses     int    `json:"misses"`
	Coverage   string `json:"coverage"`
}

// NewCoverageReport creates and returns a *CoverageReport.
func NewCoverageReport() *CoverageReport {
	return &CoverageReport{
		Coverage:          map[common.Location]*LocationCoverage{},
		Locations:         map[common.Location]struct{}{},
		ExcludedLocations: map[common.Location]struct{}{},
	}
}

type crAlias CoverageReport

// To avoid the overhead of having the Percentage & MissedLines
// as fields in the LocationCoverage struct, we simply populate
// this lcAlias struct, with the corresponding methods, upon marshalling.
type lcAlias struct {
	LineHits    map[int]int `json:"line_hits"`
	MissedLines []int       `json:"missed_lines"`
	Statements  int         `json:"statements"`
	Percentage  string      `json:"percentage"`
}

// MarshalJSON serializes each common.Location/*LocationCoverage
// key/value pair on the *CoverageReport.Coverage map, as well
// as the IDs on the *CoverageReport.ExcludedLocations map.
func (r *CoverageReport) MarshalJSON() ([]byte, error) {
	coverage := make(map[string]lcAlias, len(r.Coverage))
	for location, locationCoverage := range r.Coverage { // nolint:maprange
		coverage[location.ID()] = lcAlias{
			LineHits:    locationCoverage.LineHits,
			MissedLines: locationCoverage.MissedLines(),
			Statements:  locationCoverage.Statements,
			Percentage:  locationCoverage.Percentage(),
		}
	}
	return json.Marshal(&struct {
		Coverage          map[string]lcAlias `json:"coverage"`
		ExcludedLocations []string           `json:"excluded_locations"`
		*crAlias
	}{
		Coverage:          coverage,
		ExcludedLocations: r.ExcludedLocationIDs(),
		crAlias:           (*crAlias)(r),
	})
}

// UnmarshalJSON deserializes a JSON structure and populates
// the calling object with the respective *CoverageReport.Coverage &
// *CoverageReport.ExcludedLocations maps.
func (r *CoverageReport) UnmarshalJSON(data []byte) error {
	cr := &struct {
		Coverage          map[string]lcAlias `json:"coverage"`
		ExcludedLocations []string           `json:"excluded_locations"`
		*crAlias
	}{
		crAlias: (*crAlias)(r),
	}

	if err := json.Unmarshal(data, cr); err != nil {
		return err
	}

	for locationID, locationCoverage := range cr.Coverage { // nolint:maprange
		location, _, err := common.DecodeTypeID(nil, locationID)
		if err != nil {
			return err
		}
		if location == nil {
			return fmt.Errorf("invalid Location ID: %s", locationID)
		}
		r.Coverage[location] = &LocationCoverage{
			LineHits:   locationCoverage.LineHits,
			Statements: locationCoverage.Statements,
		}
		r.Locations[location] = struct{}{}
	}
	for _, locationID := range cr.ExcludedLocations {
		location, _, err := common.DecodeTypeID(nil, locationID)
		if err != nil {
			return err
		}
		if location == nil {
			return fmt.Errorf("invalid Location ID: %s", locationID)
		}
		r.ExcludedLocations[location] = struct{}{}
	}

	return nil
}

// MarshalLCOV serializes each common.Location/*LocationCoverage
// key/value pair on the *CoverageReport.Coverage map, to the
// LCOV format. Currently supports only line coverage, function
// and branch coverage are not yet available.
func (r *CoverageReport) MarshalLCOV() ([]byte, error) {
	buf := new(bytes.Buffer)
	for location, coverage := range r.Coverage { // nolint:maprange
		_, err := buf.WriteString(
			fmt.Sprintf("TN:\nSF:%s\n", location.ID()),
		)
		if err != nil {
			return nil, err
		}
		lines := make([]int, 0)
		for line := range coverage.LineHits { // nolint:maprange
			lines = append(lines, line)
		}
		sort.Ints(lines)
		for _, line := range lines {
			hits := coverage.LineHits[line]
			_, err = buf.WriteString(
				fmt.Sprintf("DA:%v,%v\n", line, hits),
			)
			if err != nil {
				return nil, err
			}
		}
		_, err = buf.WriteString(
			fmt.Sprintf(
				"LF:%v\nLH:%v\nend_of_record\n",
				coverage.Statements,
				coverage.CoveredLines(),
			),
		)
		if err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}
