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

package value_codec_test

import (
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/onflow/cadence/encoding/cadence_codec"
	"github.com/onflow/cadence/encoding/custom/value_codec"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/encoding/json"
	"github.com/onflow/cadence/runtime/common"
)

//
// Benchmark
//

func BenchmarkCodec(b *testing.B) {
	tests := []cadence.Value{
		cadence.String("This sentence is composed of exactly fifty-nine characters."),
		cadence.NewResource([]cadence.Value{
			cadence.Bool(true),
		}).WithType(cadence.NewResourceType(
			common.AddressLocation{
				Address: common.Address{1, 2, 3, 4, 5, 6, 7, 8},
				Name:    "NFT",
			},
			"A.12345678.NFT",
			[]cadence.Field{
				{
					Identifier: "Awesome?",
					Type:       cadence.BoolType{},
				},
			},
			[][]cadence.Parameter{},
		)),
	}

	jsonCodec := cadence_codec.CadenceCodec{Encoder: json.JsonCodec{}}
	valueCodec := cadence_codec.CadenceCodec{Encoder: value_codec.ValueCodec{}}

	for _, value := range tests {
		b.Run(fmt.Sprintf("json_%s", value.Type().ID()), func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				blob, err := jsonCodec.Encode(value)
				require.NoError(b, err, "encoding error")

				_, err = jsonCodec.Decode(nil, blob)
				require.NoError(b, err, "decoding error")
			}
		})
		b.Run(fmt.Sprintf("value_%s", value.Type().ID()), func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				blob, err := valueCodec.Encode(value)
				require.NoError(b, err, "encoding error")

				_, err = valueCodec.Decode(nil, blob)
				require.NoError(b, err, "decoding error")
			}

		})
	}
}

//
// Test Driver
//

type MeasureCodec = func(p cadence.Value) (time.Duration, time.Duration, int, error)

func measureValueCodec(value cadence.Value) (
	encodingDuration time.Duration,
	decodingDuration time.Duration,
	size int,
	err error,
) {
	encoder, decoder, buffer := NewTestCodec()

	t0 := time.Now()
	err = encoder.EncodeValue(value)
	t1 := time.Now()
	if err != nil {
		return
	}

	size = buffer.Len()

	t2 := time.Now()
	_, err = decoder.Decode()
	t3 := time.Now()
	if err != nil {
		return
	}

	encodingDuration = t1.Sub(t0)
	decodingDuration = t3.Sub(t2)
	return
}

func measureJsonCodec(value cadence.Value) (
	encodingDuration time.Duration,
	decodingDuration time.Duration,
	size int,
	err error,
) {
	t0 := time.Now()
	blob, err := json.Encode(value)
	t1 := time.Now()
	if err != nil {
		return
	}

	size = len(blob)

	t2 := time.Now()
	_, err = json.Decode(nil, blob)
	t3 := time.Now()
	if err != nil {
		return
	}

	encodingDuration = t1.Sub(t0)
	decodingDuration = t3.Sub(t2)
	return
}

func average(ints []int) int {
	sum := 0

	for _, i := range ints {
		sum += i
	}

	return sum / len(ints)
}

// codec -> value -> iteration -> measurement

type Raw struct {
	EncodingDurations []int
	DecodingDurations []int
}

type Stats struct {
	Average int
	Minimum int
	Maximum int
}

func (s *Stats) Derive(raw []int) {
	sort.Ints(raw)
	s.Average = average(raw)
	s.Minimum = raw[0]
	s.Maximum = raw[len(raw)-1]
}

type Measurements struct {
	Raw      Raw // order not preserved
	Encoding Stats
	Decoding Stats
	Size     int
}

func NewMeasurements(iterations int) Measurements {
	return Measurements{
		Raw: Raw{
			EncodingDurations: make([]int, 0, iterations),
			DecodingDurations: make([]int, 0, iterations),
		},
		Encoding: Stats{
			Average: -1,
			Minimum: -1,
			Maximum: -1,
		},
		Decoding: Stats{
			Average: -1,
			Minimum: -1,
			Maximum: -1,
		},
		Size: -1,
	}
}

func (m *Measurements) Derive() {
	m.Encoding.Derive(m.Raw.EncodingDurations)
	m.Decoding.Derive(m.Raw.DecodingDurations)
}

func measure(value cadence.Value, iterations int, measureCodec MeasureCodec) (measurements Measurements, err error) {
	measurements = NewMeasurements(iterations)

	size := -1

	for i := 0; i < iterations; i++ {
		var encodingDuration, decodingDuration time.Duration
		encodingDuration, decodingDuration, size, err = measureCodec(value)
		if err != nil {
			return
		}
		measurements.Raw.EncodingDurations = append(measurements.Raw.EncodingDurations, int(encodingDuration))
		measurements.Raw.DecodingDurations = append(measurements.Raw.DecodingDurations, int(decodingDuration))
	}

	measurements.Derive()
	measurements.Size = size

	return
}

func TestCodecPerformance(t *testing.T) {
	// not parallel

	const iterations = 100
	values := []cadence.Value{
		cadence.String("This sentence is composed of exactly fifty-nine characters."),
		cadence.NewResource([]cadence.Value{
			cadence.Bool(true),
		}).WithType(cadence.NewResourceType(
			common.AddressLocation{
				Address: common.Address{1, 2, 3, 4, 5, 6, 7, 8},
				Name:    "NFT",
			},
			"A.12345678.NFT",
			[]cadence.Field{
				{
					Identifier: "Awesome?",
					Type:       cadence.BoolType{},
				},
			},
			[][]cadence.Parameter{},
		)),
	}

	fmt.Println("Results:")
	fmt.Println("(higher speedup/reduction is better)")
	fmt.Println()

	for _, value := range values {
		jsonMeasurements, err := measure(value, iterations, measureJsonCodec)
		require.NoError(t, err, "json codec error")

		valueMeasurements, err := measure(value, iterations, measureValueCodec)
		require.NoError(t, err, "value codec error")

		encodingSpeedup := float64(jsonMeasurements.Encoding.Average) / float64(valueMeasurements.Encoding.Average)
		decodingSpeedup := float64(jsonMeasurements.Decoding.Average) / float64(valueMeasurements.Decoding.Average)
		sizeReduction := float64(jsonMeasurements.Size) / float64(valueMeasurements.Size)

		fmt.Println("Cadence Value: ", value.String())
		fmt.Printf("(Type: %s)\n", value.Type().ID())
		fmt.Println("Encoding Time (ns):")
		fmt.Println("\tJSON Average: ", jsonMeasurements.Encoding.Average)
		fmt.Println("\tJSON Minimum: ", jsonMeasurements.Encoding.Minimum)
		fmt.Println("\tJSON Maximum: ", jsonMeasurements.Encoding.Maximum)
		fmt.Println("\tValue Average: ", valueMeasurements.Encoding.Average)
		fmt.Println("\tValue Minimum: ", valueMeasurements.Encoding.Minimum)
		fmt.Println("\tValue Maximum: ", valueMeasurements.Encoding.Maximum)
		fmt.Println("\tSpeedup:", encodingSpeedup)
		fmt.Println("Decoding Time (ns):")
		fmt.Println("\tJSON Average: ", jsonMeasurements.Decoding.Average)
		fmt.Println("\tJSON Minimum: ", jsonMeasurements.Decoding.Minimum)
		fmt.Println("\tJSON Maximum: ", jsonMeasurements.Decoding.Maximum)
		fmt.Println("\tValue Average: ", valueMeasurements.Decoding.Average)
		fmt.Println("\tValue Minimum: ", valueMeasurements.Decoding.Minimum)
		fmt.Println("\tValue Maximum: ", valueMeasurements.Decoding.Maximum)
		fmt.Println("\tSpeedup:", decodingSpeedup)
		fmt.Println("Size (bytes):")
		fmt.Println("\tJSON: ", jsonMeasurements.Size)
		fmt.Println("\tValue: ", valueMeasurements.Size)
		fmt.Println("\tReduction:", sizeReduction)
		fmt.Println()
	}

	t.Fail()
}
