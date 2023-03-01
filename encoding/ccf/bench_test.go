/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2022-2023 Dapper Labs, Inc.
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

package ccf_test

import (
	"testing"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/encoding/ccf"
	"github.com/onflow/cadence/encoding/json"
	"github.com/stretchr/testify/require"
)

var encoded []byte
var val cadence.Value

var benchmarks = []struct {
	name  string
	value cadence.Value
}{
	{name: "FlowFees.FeesDeducted", value: createFlowFeesFeesDeductedEvent()},
	{name: "FlowFees.TokensWithdrawn", value: createFlowFeesTokensWithdrawnEvent()},

	{name: "FlowIDTableStaking.DelegatorRewardsPaid", value: createFlowIDTableStakingDelegatorRewardsPaidEvent()},
	{name: "FlowIDTableStaking.EpochTotalRewardsPaid", value: createFlowIDTableStakingEpochTotalRewardsPaidEvent()},
	{name: "FlowIDTableStaking.NewWeeklyPayout", value: createFlowIDTableStakingNewWeeklyPayoutEvent()},
	{name: "FlowIDTableStaking.RewardsPaid", value: createFlowIDTableStakingRewardsPaidEvent()},

	{name: "FlowToken.TokensDeposited with nil receiver", value: createFlowTokenTokensDepositedEventNoReceiver()},
	{name: "FlowToken.TokensDeposited", value: createFlowTokenTokensDepositedEvent()},
	{name: "FlowToken.TokensMinted", value: createFlowTokenTokensMintedEvent()},
	{name: "FlowToken.TokensWithdrawn", value: createFlowTokenTokensWithdrawnEvent()},
}

// Events for transaction 03aa46047cdadfcf7ee23ee86cd53064e05f8b5f8a6f570e9f53b2744eddbee4.
// This transaction was selected from mainnet because of its large number of events (48309).
//   - The number of events for each type are real.
//   - The types of events are real.
//   - To simplify benchmark code, all event values for each event type are the same
//     (i.e. the values are from the first event of that event type).
var batchBenchmarks = []struct {
	count int
	value cadence.Value
}{
	{count: 16102, value: createFlowTokenTokensDepositedEvent()},
	{count: 1, value: createFlowTokenTokensMintedEvent()},
	{count: 16102, value: createFlowTokenTokensWithdrawnEvent()},
	{count: 15783, value: createFlowIDTableStakingDelegatorRewardsPaidEvent()},
	{count: 1, value: createFlowIDTableStakingEpochTotalRewardsPaidEvent()},
	{count: 1, value: createFlowIDTableStakingNewWeeklyPayoutEvent()},
	{count: 317, value: createFlowIDTableStakingRewardsPaidEvent()},
	{count: 1, value: createFlowFeesFeesDeductedEvent()},
	{count: 1, value: createFlowFeesTokensWithdrawnEvent()},
}

func BenchmarkEncodeJSON(b *testing.B) {
	var err error

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				encoded, err = json.Encode(bm.value)
			}
			require.NoError(b, err)
		})
	}
}

func BenchmarkDecodeJSON(b *testing.B) {
	for _, bm := range benchmarks {
		encoded, err := json.Encode(bm.value)
		require.NoError(b, err)

		b.Run(bm.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				val, err = json.Decode(nil, encoded)
			}
			require.NoError(b, err)
		})
	}
}

func BenchmarkEncodeCCF(b *testing.B) {
	var err error

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				encoded, err = ccf.Encode(bm.value)
			}
			require.NoError(b, err)
		})
	}
}

func BenchmarkDecodeCCF(b *testing.B) {
	for _, bm := range benchmarks {
		encoded, err := ccf.Encode(bm.value)
		require.NoError(b, err)

		b.Run(bm.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				val, err = ccf.Decode(nil, encoded)
			}
			require.NoError(b, err)
		})
	}
}

func BenchmarkEncodeBatchEventsJSON(b *testing.B) {
	var err error
	var encodedSize int

	for i := 0; i < b.N; i++ {
		encodedSize = 0
		for _, bm := range batchBenchmarks {
			for i := 0; i < bm.count; i++ {
				encoded, err = json.Encode(bm.value)
				encodedSize += len(encoded)
			}
		}
	}
	require.NoError(b, err)

	// fmt.Printf("Batch events encoded in JSON are %d bytes\n", encodedSize)
}

func BenchmarkDecodeBatchEventsJSON(b *testing.B) {
	type encodedBatchEvent struct {
		count   int
		encoded []byte
	}

	benchmarks := make([]encodedBatchEvent, len(batchBenchmarks))

	for i, bm := range batchBenchmarks {
		benchmarks[i] = encodedBatchEvent{
			count:   bm.count,
			encoded: json.MustEncode(bm.value),
		}
	}

	var err error
	for i := 0; i < b.N; i++ {
		for _, bm := range benchmarks {
			for i := 0; i < bm.count; i++ {
				val, err = json.Decode(nil, bm.encoded)
			}
		}
	}

	require.NoError(b, err)
}

func BenchmarkEncodeBatchEventsCCF(b *testing.B) {
	var err error
	var encodedSize int

	for i := 0; i < b.N; i++ {
		encodedSize = 0
		for _, bm := range batchBenchmarks {
			for i := 0; i < bm.count; i++ {
				encoded, err = ccf.Encode(bm.value)
				encodedSize += len(encoded)
			}
		}
	}
	require.NoError(b, err)

	// fmt.Printf("Batch events encoded in CCF are %d bytes\n", encodedSize)
}

func BenchmarkDecodeBatchEventsCCF(b *testing.B) {
	type encodedBatchEvent struct {
		count   int
		encoded []byte
	}

	benchmarks := make([]encodedBatchEvent, len(batchBenchmarks))

	for i, bm := range batchBenchmarks {
		benchmarks[i] = encodedBatchEvent{
			count:   bm.count,
			encoded: ccf.MustEncode(bm.value),
		}
	}

	var err error
	for i := 0; i < b.N; i++ {
		for _, bm := range benchmarks {
			for i := 0; i < bm.count; i++ {
				val, err = ccf.Decode(nil, bm.encoded)
			}
		}
	}

	require.NoError(b, err)
}
