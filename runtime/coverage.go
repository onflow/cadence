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

package runtime

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"sync"

	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/interpreter"
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
	// The ground truth of which statements are interpreted/executed
	// is the `InterpreterEnvironment.newOnStatementHandler()` function.
	// This means that every call of `CoverageReport.AddLineHit()` from
	// that callback, should be respected. The `CoverageReport.InspectProgram()`
	// may have failed, for whatever reason, to find a specific line.
	// This is a good insight to solidify its implementation and debug
	// the inspection failure. Ideally, this condition will never be true,
	// except for tests. We just leave it here, as a fail-safe mechanism.
	// We saturate the percentage at 100%, when the inspector
	// fails to correctly count all statements for a given
	// location.
	coveredLines := min(c.CoveredLines(), c.Statements)

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
	Coverage map[common.Location]*LocationCoverage
	// Contains locations whose programs are already inspected.
	Locations map[common.Location]struct{}
	// Contains locations excluded from coverage collection.
	ExcludedLocations map[common.Location]struct{}
	// This filter can be used to inject custom logic on
	// each location/program inspection.
	locationFilter LocationFilter
	// Contains a mapping with source paths for each
	// location.
	locationMappings map[string]string
	lock             sync.RWMutex
}

// WithLocationFilter sets the LocationFilter for the current
// CoverageReport.
func (r *CoverageReport) WithLocationFilter(
	locationFilter LocationFilter,
) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.locationFilter = locationFilter
}

// WithLocationMappings sets the LocationMappings for the current
// CoverageReport.
func (r *CoverageReport) WithLocationMappings(
	locationMappings map[string]string,
) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.locationMappings = locationMappings
}

// ExcludeLocation adds the given location to the map of excluded
// locations.
func (r *CoverageReport) ExcludeLocation(location Location) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.ExcludedLocations[location] = struct{}{}
}

// IsLocationExcluded checks whether the given location is excluded
// or not, from coverage collection.
func (r *CoverageReport) IsLocationExcluded(location Location) bool {
	r.lock.RLock()
	defer r.lock.RUnlock()
	return r.isLocationExcluded(location)
}

func (r *CoverageReport) isLocationExcluded(location Location) bool {
	_, ok := r.ExcludedLocations[location]
	return ok
}

// AddLineHit increments the hit count for the given line, on the given
// location. The method call is a NO-OP in two cases:
// - If the location is excluded from coverage collection
// - If the location has not been inspected for its statements
func (r *CoverageReport) AddLineHit(location Location, line int) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.addLineHit(location, line)
}

func (r *CoverageReport) addLineHit(location Location, line int) {
	if r.isLocationExcluded(location) {
		return
	}

	if !r.isLocationInspected(location) {
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
	r.lock.Lock()
	defer r.lock.Unlock()
	r.inspectProgram(location, program)
}

func (r *CoverageReport) inspectProgram(location Location, program *ast.Program) {
	if r.locationFilter != nil && !r.locationFilter(location) {
		return
	}
	if r.isLocationExcluded(location) {
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
						for _, condition := range functionBlock.PreConditions.Conditions {
							recordLine(condition.CodeElement())
						}
					}
					if functionBlock.PostConditions != nil {
						for _, condition := range functionBlock.PostConditions.Conditions {
							recordLine(condition.CodeElement())
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
	r.lock.RLock()
	defer r.lock.RUnlock()
	return r.isLocationInspected(location)
}

func (r *CoverageReport) isLocationInspected(location Location) bool {
	_, isInspected := r.Locations[location]
	return isInspected
}

// Percentage returns a string representation of the covered statements
// percentage. It is defined as the ratio of total covered lines over
// total statements, for all locations.
func (r *CoverageReport) Percentage() string {
	r.lock.RLock()
	defer r.lock.RUnlock()
	return r.percentage()
}

func (r *CoverageReport) percentage() string {
	totalStatements := r.statements()
	totalCoveredLines := r.hits()
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
	r.lock.RLock()
	defer r.lock.RUnlock()
	if r.statements() == 0 {
		return "There are no statements to cover"
	}
	return fmt.Sprintf("Coverage: %v of statements", r.percentage())
}

// Reset clears the collected coverage information for all locations and inspected locations.
// Excluded locations remain intact.
func (r *CoverageReport) Reset() {
	r.lock.Lock()
	defer r.lock.Unlock()
	clear(r.Coverage)
	clear(r.Locations)
}

// Merge adds all the collected coverage information to the
// calling object. Excluded locations are also taken into
// account.
func (r *CoverageReport) Merge(other *CoverageReport) {
	other.lock.RLock()
	defer other.lock.RUnlock()

	r.lock.Lock()
	defer r.lock.Unlock()

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
	r.lock.RLock()
	defer r.lock.RUnlock()
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
	r.lock.RLock()
	defer r.lock.RUnlock()
	return r.totalLocations()
}

func (r *CoverageReport) totalLocations() int {
	return len(r.Coverage)
}

// Statements returns the total count of statements, for all the
// locations included in the CoverageReport.
func (r *CoverageReport) Statements() int {
	r.lock.RLock()
	defer r.lock.RUnlock()
	return r.statements()
}

func (r *CoverageReport) statements() int {
	totalStatements := 0
	for _, locationCoverage := range r.Coverage { // nolint:maprange
		totalStatements += locationCoverage.Statements
	}
	return totalStatements
}

// Hits returns the total count of covered lines, for all the
// locations included in the CoverageReport.
func (r *CoverageReport) Hits() int {
	r.lock.RLock()
	defer r.lock.RUnlock()
	return r.hits()
}

func (r *CoverageReport) hits() int {
	totalCoveredLines := 0
	for _, locationCoverage := range r.Coverage { // nolint:maprange
		totalCoveredLines += locationCoverage.CoveredLines()
	}
	return totalCoveredLines
}

// Misses returns the total count of non-covered lines, for all
// the locations included in the CoverageReport.
func (r *CoverageReport) Misses() int {
	r.lock.RLock()
	defer r.lock.RUnlock()
	return r.misses()
}

func (r *CoverageReport) misses() int {
	return r.statements() - r.hits()
}

// Summary returns a CoverageReportSummary object, containing
// key metrics for a CoverageReport, such as:
// - Total Locations,
// - Total Statements,
// - Total Hits,
// - Total Misses,
// - Overall Coverage Percentage.
func (r *CoverageReport) Summary() CoverageReportSummary {
	r.lock.RLock()
	defer r.lock.RUnlock()
	return CoverageReportSummary{
		Locations:  r.totalLocations(),
		Statements: r.statements(),
		Hits:       r.hits(),
		Misses:     r.misses(),
		Coverage:   r.percentage(),
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
func (r *CoverageReport) Diff(other *CoverageReport) CoverageReportSummary {
	other.lock.RLock()
	defer other.lock.RUnlock()

	r.lock.RLock()
	defer r.lock.RUnlock()

	baseCoverage := 100 * float64(r.hits()) / float64(r.statements())
	newCoverage := 100 * float64(other.hits()) / float64(other.statements())
	coverageDelta := fmt.Sprintf(
		"%0.1f%%",
		100*(newCoverage-baseCoverage)/baseCoverage,
	)
	return CoverageReportSummary{
		Locations:  other.totalLocations() - r.totalLocations(),
		Statements: other.statements() - r.statements(),
		Hits:       other.hits() - r.hits(),
		Misses:     other.misses() - r.misses(),
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
	r.lock.RLock()
	defer r.lock.RUnlock()

	coverage := make(map[string]lcAlias, len(r.Coverage))
	for location, locationCoverage := range r.Coverage { // nolint:maprange
		locationSource := r.sourcePathForLocation(location)
		coverage[locationSource] = lcAlias{
			LineHits:    locationCoverage.LineHits,
			MissedLines: locationCoverage.MissedLines(),
			Statements:  locationCoverage.Statements,
			Percentage:  locationCoverage.Percentage(),
		}
	}

	excludedLocationIDs := make([]string, 0, len(r.ExcludedLocations))
	for location := range r.ExcludedLocations { // nolint:maprange
		excludedLocationIDs = append(excludedLocationIDs, location.ID())
	}

	return json.Marshal(&struct {
		Coverage          map[string]lcAlias `json:"coverage"`
		ExcludedLocations []string           `json:"excluded_locations"`
	}{
		Coverage:          coverage,
		ExcludedLocations: excludedLocationIDs,
	})
}

// UnmarshalJSON deserializes a JSON structure and populates
// the calling object with the respective *CoverageReport.Coverage &
// *CoverageReport.ExcludedLocations maps.
func (r *CoverageReport) UnmarshalJSON(data []byte) error {
	cr := &struct {
		Coverage          map[string]lcAlias `json:"coverage"`
		ExcludedLocations []string           `json:"excluded_locations"`
	}{}

	if err := json.Unmarshal(data, cr); err != nil {
		return err
	}

	r.lock.Lock()
	defer r.lock.Unlock()

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
// Description for the LCOV file format, can be found here
// https://github.com/linux-test-project/lcov/blob/master/man/geninfo.1#L948.
func (r *CoverageReport) MarshalLCOV() ([]byte, error) {
	r.lock.RLock()
	defer r.lock.RUnlock()

	i := 0
	locations := make([]common.Location, len(r.Coverage))
	for location := range r.Coverage { // nolint:maprange
		locations[i] = location
		i++
	}
	sort.Slice(locations, func(i, j int) bool {
		return locations[i].ID() < locations[j].ID()
	})

	buf := new(bytes.Buffer)
	for _, location := range locations {
		coverage := r.Coverage[location]
		locationSource := r.sourcePathForLocation(location)
		_, err := fmt.Fprintf(buf, "TN:\nSF:%s\n", locationSource)
		if err != nil {
			return nil, err
		}

		i := 0
		lines := make([]int, len(coverage.LineHits))
		for line := range coverage.LineHits { // nolint:maprange
			lines[i] = line
			i++
		}
		sort.Ints(lines)

		for _, line := range lines {
			hits := coverage.LineHits[line]
			_, err = fmt.Fprintf(buf, "DA:%v,%v\n", line, hits)
			if err != nil {
				return nil, err
			}
		}

		_, err = fmt.Fprintf(
			buf,
			"LF:%v\nLH:%v\nend_of_record\n",
			coverage.Statements,
			coverage.CoveredLines(),
		)
		if err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

// Given a common.Location, returns its mapped source, if any.
// Defaults to the location's ID().
func (r *CoverageReport) sourcePathForLocation(location common.Location) string {
	var locationIdentifier string

	switch loc := location.(type) {
	case common.AddressLocation:
		locationIdentifier = loc.Name
	case common.StringLocation:
		locationIdentifier = loc.String()
	case common.IdentifierLocation:
		locationIdentifier = loc.String()
	default:
		locationIdentifier = loc.ID()
	}

	locationSource, ok := r.locationMappings[locationIdentifier]
	if !ok {
		locationSource = location.ID()
	}

	return locationSource
}

func (r *CoverageReport) newOnStatementHandler() interpreter.OnStatementFunc {
	return func(inter *interpreter.Interpreter, statement ast.Statement) {
		location := inter.Location
		line := statement.StartPosition().Line

		r.lock.Lock()
		defer r.lock.Unlock()

		if !r.isLocationInspected(location) {
			r.inspectProgram(location, inter.Program.Program)
		}
		r.addLineHit(location, line)
	}
}
