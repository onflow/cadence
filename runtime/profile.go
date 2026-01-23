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
	"fmt"
	"strings"

	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/common/intervalst"
	"github.com/onflow/cadence/interpreter"
)

type lineNumber int

var _ intervalst.Position = lineNumber(0)

func (l lineNumber) Compare(other intervalst.Position) int {
	if _, ok := other.(intervalst.MinPosition); ok {
		return 1
	}

	otherLine := other.(lineNumber)
	if l < otherLine {
		return -1
	} else if l > otherLine {
		return 1
	} else {
		return 0
	}
}

type profiledFunction struct {
	location  common.Location
	name      string
	startLine int
}

// ComputationProfile collects computation profiling information per location.
type ComputationProfile struct {
	locationFunctions  map[common.Location]*intervalst.IntervalST[profiledFunction]
	currentStackTrace  profileStackTrace
	stackTraceUsages   map[string]stackTraceUsage
	locationMappings   map[string]string
	computationWeights map[common.ComputationKind]uint64
}

// NewComputationProfile creates and returns a *ComputationProfile.
// The Compuation Profile and compuation profiling is not thread safe!
// This means its ok to use in a testing/debuging scenarios,
// but it is not suitable for production use.
func NewComputationProfile() *ComputationProfile {
	return &ComputationProfile{
		locationFunctions: make(map[common.Location]*intervalst.IntervalST[profiledFunction]),
		stackTraceUsages:  make(map[string]stackTraceUsage),
	}
}

// computationProfileInstance is a ComputationProfile that delegates metering to a ComputationGauge.
// the ComputationGauge is not transferable between individual environments, so we create a computationProfileInstance
// that has the same lifetime as an environment.
type computationProfileInstance struct {
	*ComputationProfile
	// delegatedComputationGauge is the computation gauge to which
	// delegated computation metering is reported.
	delegatedComputationGauge common.ComputationGauge
}

var _ common.ComputationGauge = computationProfileInstance{}

// newComputationProfileInstance
func newComputationProfileInstance(
	profile *ComputationProfile,
	delegatedComputationGauge common.ComputationGauge,
) computationProfileInstance {
	return computationProfileInstance{
		ComputationProfile:        profile,
		delegatedComputationGauge: delegatedComputationGauge,
	}
}

// WithLocationMappings sets the location mappings for this profile.
func (p *ComputationProfile) WithLocationMappings(
	locationMappings map[string]string,
) {
	p.locationMappings = locationMappings
}

// WithComputationWeights sets the computation weights for this profile.
func (p *ComputationProfile) WithComputationWeights(
	weights map[common.ComputationKind]uint64,
) {
	p.computationWeights = weights
}

type LocationLine struct {
	Location common.Location
	Line     int
}

type profileStackTrace []LocationLine

type stackTraceUsage struct {
	computation uint64
	stackTrace  profileStackTrace
}

func (s profileStackTrace) aggregateKey() string {
	var sb strings.Builder
	for i, locationLine := range s {
		if i > 0 {
			sb.WriteByte(',')
		}
		location := locationLine.Location
		var locationID string
		if location != nil {
			locationID = location.ID()
		}

		_, _ = fmt.Fprintf(
			&sb,
			"%s:%d",
			locationID,
			locationLine.Line,
		)
	}
	return sb.String()
}

func (p *ComputationProfile) newOnStatementHandler() interpreter.OnStatementFunc {
	return func(inter *interpreter.Interpreter, statement ast.Statement) {
		location := inter.Location

		// Ensure the program is inspected
		p.InspectProgram(location, inter.Program.Program)

		var stackTrace profileStackTrace

		for _, invocation := range inter.CallStack() {
			locationRange := invocation.LocationRange
			stackTrace = append(
				stackTrace,
				LocationLine{
					Location: locationRange.Location,
					Line:     locationRange.StartPosition().Line,
				},
			)
		}

		stackTrace = append(
			stackTrace,
			LocationLine{
				Location: location,
				Line:     statement.StartPosition().Line,
			},
		)

		p.currentStackTrace = stackTrace
	}
}

func (p computationProfileInstance) MeterComputation(computationUsage common.ComputationUsage) error {

	gauge := p.delegatedComputationGauge
	if gauge != nil {
		err := gauge.MeterComputation(computationUsage)
		if err != nil {
			return err
		}
	}

	weight := p.computationWeights[computationUsage.Kind]
	if weight == 0 {
		// No need to record zero-weight computation
		return nil
	}

	aggregateKey := p.currentStackTrace.aggregateKey()
	traceUsage := p.stackTraceUsages[aggregateKey]
	traceUsage.stackTrace = p.currentStackTrace
	traceUsage.computation += computationUsage.Intensity * weight
	p.stackTraceUsages[aggregateKey] = traceUsage

	return nil
}

// InspectProgram inspects the elements of the given *ast.Program,
// and determines the ranges of functions.
func (p *ComputationProfile) InspectProgram(location Location, program *ast.Program) {

	functions, ok := p.locationFunctions[location]
	if ok {
		return
	}

	functions = &intervalst.IntervalST[profiledFunction]{}
	p.locationFunctions[location] = functions

	var stack []*ast.CompositeDeclaration

	addFunction := func(identifier string, pos ast.HasPosition) {
		startLine := pos.StartPosition().Line
		endLine := pos.EndPosition(nil).Line

		interval := intervalst.NewInterval(
			lineNumber(startLine),
			lineNumber(endLine),
		)

		var nameBuilder strings.Builder
		for _, composite := range stack {
			nameBuilder.WriteString(composite.Identifier.Identifier)
			nameBuilder.WriteString(".")
		}
		nameBuilder.WriteString(identifier)
		name := nameBuilder.String()

		function := profiledFunction{
			location:  location,
			name:      name,
			startLine: startLine,
		}

		functions.Put(interval, function)
	}

	inspector := ast.NewInspector(program)
	inspector.Elements(
		[]ast.Element{
			(*ast.CompositeDeclaration)(nil),
			(*ast.FunctionDeclaration)(nil),
			(*ast.SpecialFunctionDeclaration)(nil),
		},
		func(element ast.Element, push bool) bool {
			if push {
				switch decl := element.(type) {
				case *ast.CompositeDeclaration:
					stack = append(stack, decl)

				case *ast.FunctionDeclaration:
					addFunction(
						decl.Identifier.Identifier,
						decl,
					)

				case *ast.SpecialFunctionDeclaration:
					addFunction(
						decl.FunctionDeclaration.Identifier.Identifier,
						decl.FunctionDeclaration,
					)
				}
			} else {
				if _, ok := element.(*ast.CompositeDeclaration); ok {
					stack = stack[:len(stack)-1]
				}
			}

			return true
		},
	)
}

// functionAtLine returns the function at the given location and line, if any.
func (p *ComputationProfile) functionAtLine(locationLine LocationLine) (profiledFunction, bool) {
	functions, ok := p.locationFunctions[locationLine.Location]
	if !ok {
		return profiledFunction{}, false
	}

	_, function, ok := functions.Search(lineNumber(locationLine.Line))
	return function, ok
}

// sourcePathForLocation returns the mapped source for the given Location, if any.
// Defaults to the location's ID().
func (p *ComputationProfile) sourcePathForLocation(location common.Location) string {
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

	locationSource, ok := p.locationMappings[locationIdentifier]
	if !ok {
		locationSource = location.ID()
	}

	return locationSource
}

// Reset clears the collected profiling information for all locations and inspected locations.
func (p *ComputationProfile) Reset() {
	p.currentStackTrace = nil
	clear(p.stackTraceUsages)
	clear(p.locationFunctions)
}
