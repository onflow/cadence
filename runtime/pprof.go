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

	pprof "github.com/google/pprof/profile"

	"github.com/onflow/cadence/common"
)

type PProfExporter struct {
	ComputationProfile *ComputationProfile
	profile            *pprof.Profile
	lineLocations      map[pprof.Line]*pprof.Location
	functions          map[profiledFunction]*pprof.Function
}

func NewPProfExporter(computationProfile *ComputationProfile) *PProfExporter {
	return &PProfExporter{
		ComputationProfile: computationProfile,
		lineLocations:      make(map[pprof.Line]*pprof.Location),
		functions:          make(map[profiledFunction]*pprof.Function),
	}
}

func (e *PProfExporter) Export() (*pprof.Profile, error) {
	e.profile = &pprof.Profile{}

	e.profile.SampleType = []*pprof.ValueType{
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
		functions := e.ComputationProfile.locationFunctions[location]

		for _, function := range functions.Values() {
			pprofFunction := &pprof.Function{
				ID:        e.nextFunctionID(),
				Name:      function.name,
				Filename:  e.ComputationProfile.sourcePathForLocation(location),
				StartLine: int64(function.startLine),
			}
			e.profile.Function = append(
				e.profile.Function,
				pprofFunction,
			)
			e.functions[function] = pprofFunction
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

		var sampleLocations []*pprof.Location

		for i := len(usage.stackTrace) - 1; i >= 0; i-- {
			locationLine := usage.stackTrace[i]

			location := locationLine.Location
			if location == nil {
				continue
			}

			line := locationLine.Line

			function, ok := e.ComputationProfile.functionAtLine(locationLine)
			if !ok {
				panic(fmt.Errorf(
					"missing profile function at %s:%d",
					location,
					line,
				))
			}

			pprofFunction, ok := e.functions[function]
			if !ok {
				panic(fmt.Errorf(
					"missing exported function at %s:%d",
					location,
					line,
				))
			}

			sampleLocation := e.getOrAddLocation(pprofFunction, line)
			sampleLocations = append(sampleLocations, sampleLocation)
		}

		pprofSample := &pprof.Sample{
			Location: sampleLocations,
			Value: []int64{
				int64(usage.computation),
			},
		}

		e.profile.Sample = append(
			e.profile.Sample,
			pprofSample,
		)
	}
}

func (e *PProfExporter) getOrAddLocation(pprofFunction *pprof.Function, line int) *pprof.Location {
	pprofLine := pprof.Line{
		Function: pprofFunction,
		Line:     int64(line),
	}

	pprofLocation, ok := e.lineLocations[pprofLine]
	if ok {
		return pprofLocation
	}

	pprofLocation = &pprof.Location{
		ID:   e.nextLocationID(),
		Line: []pprof.Line{pprofLine},
	}
	e.lineLocations[pprofLine] = pprofLocation
	e.profile.Location = append(
		e.profile.Location,
		pprofLocation,
	)

	return pprofLocation
}

func (e *PProfExporter) nextLocationID() uint64 {
	// ID must be non-zero
	return uint64(len(e.profile.Location) + 1)
}
