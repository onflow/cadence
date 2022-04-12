/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2022 Dapper Labs, Inc.
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

import "github.com/onflow/cadence/runtime/common"

// LocationCoverage records coverage information for a location
//
type LocationCoverage struct {
	LineHits map[int]int `json:"line_hits"`
}

func (c *LocationCoverage) AddLineHit(line int) {
	c.LineHits[line]++
}

func NewLocationCoverage() *LocationCoverage {
	return &LocationCoverage{
		LineHits: map[int]int{},
	}
}

// CoverageReport is a collection of coverage per location
//
type CoverageReport struct {
	Coverage map[common.LocationID]*LocationCoverage `json:"coverage"`
}

func (r *CoverageReport) AddLineHit(location common.Location, line int) {
	locationID := location.ID()
	locationCoverage := r.Coverage[locationID]
	if locationCoverage == nil {
		locationCoverage = NewLocationCoverage()
		r.Coverage[locationID] = locationCoverage
	}
	locationCoverage.AddLineHit(line)
}

func NewCoverageReport() *CoverageReport {
	return &CoverageReport{
		Coverage: map[common.LocationID]*LocationCoverage{},
	}
}
