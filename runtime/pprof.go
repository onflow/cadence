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
	"slices"
	"strings"

	"github.com/google/pprof/profile"

	"github.com/onflow/cadence/common"
)

type PProfExporter struct {
	ComputationProfile *ComputationProfile
	profile            *profile.Profile
	locations          map[profile.Line]*profile.Location
	functions          map[profiledFunction]*profile.Function
}

func NewPProfExporter(computationProfile *ComputationProfile) *PProfExporter {
	return &PProfExporter{
		ComputationProfile: computationProfile,
		locations:          make(map[profile.Line]*profile.Location),
		functions:          make(map[profiledFunction]*profile.Function),
	}
}

func (e *PProfExporter) Export() (*profile.Profile, error) {
	e.profile = &profile.Profile{}

	e.profile.SampleType = []*profile.ValueType{
		{
			Type: "computation",
			Unit: "count",
		},
	}

	e.exportFunctions()

	e.exportSamples()

	return e.profile, nil
}

func (e *PProfExporter) exportFunctions() {
	// Get locations in a stable order
	var locations []common.Location
	for location := range e.ComputationProfile.locationFunctions { //nolint:maprange
		locations = append(locations, location)
	}
	slices.SortFunc(locations, func(a, b common.Location) int {
		return strings.Compare(a.ID(), b.ID())
	})

	for _, location := range locations {
		profiledFunction := e.ComputationProfile.locationFunctions[location]

		for _, profiledFunction := range profiledFunction.Values() {
			function := &profile.Function{
				ID:        e.nextFunctionID(),
				Name:      profiledFunction.name,
				Filename:  e.ComputationProfile.sourcePathForLocation(location),
				StartLine: int64(profiledFunction.startLine),
			}
			e.profile.Function = append(
				e.profile.Function,
				function,
			)
			e.functions[profiledFunction] = function
		}
	}
}

func (e *PProfExporter) nextFunctionID() uint64 {
	// ID must be non-zero
	return uint64(len(e.profile.Function) + 1)
}

func (e *PProfExporter) exportSamples() {
	// Get aggregate keys in a stable order
	var aggregateKeys []string
	for aggregateKey := range e.ComputationProfile.stackTraceUsages { //nolint:maprange
		aggregateKeys = append(aggregateKeys, aggregateKey)
	}
	slices.Sort(aggregateKeys)

	for _, aggregateKey := range aggregateKeys {
		usage := e.ComputationProfile.stackTraceUsages[aggregateKey]

		var sampleLocations []*profile.Location

		for i := len(usage.stackTrace) - 1; i >= 0; i-- {
			locationLine := usage.stackTrace[i]

			location := locationLine.Location
			if location == nil {
				continue
			}

			line := locationLine.Line

			profiledFunction, ok := e.ComputationProfile.functionAtLine(locationLine)
			if !ok {
				panic(fmt.Errorf(
					"missing profile function at %s:%d",
					location,
					line,
				))
			}

			function, ok := e.functions[profiledFunction]
			if !ok {
				panic(fmt.Errorf(
					"missing exported function at %s:%d",
					location,
					line,
				))
			}

			sampleLocation := e.getOrAddLocation(function, line)
			sampleLocations = append(sampleLocations, sampleLocation)
		}

		sample := &profile.Sample{
			Location: sampleLocations,
			Value: []int64{
				int64(usage.computation),
			},
		}

		e.profile.Sample = append(
			e.profile.Sample,
			sample,
		)
	}
}

func (e *PProfExporter) getOrAddLocation(function *profile.Function, line int) *profile.Location {
	pLine := profile.Line{
		Function: function,
		Line:     int64(line),
	}

	location, ok := e.locations[pLine]
	if ok {
		return location
	}

	location = &profile.Location{
		ID:   e.nextLocationID(),
		Line: []profile.Line{pLine},
	}
	e.locations[pLine] = location
	e.profile.Location = append(
		e.profile.Location,
		location,
	)

	return location
}

func (e *PProfExporter) nextLocationID() uint64 {
	// ID must be non-zero
	return uint64(len(e.profile.Location) + 1)
}
