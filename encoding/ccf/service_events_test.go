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

package ccf_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/encoding/ccf"
)

func TestEpochSetupEvent(t *testing.T) {
	event := createEpochSetupEvent()

	b, err := ccf.Encode(event)
	require.NoError(t, err)

	// Test that encoded value isn't sorted.
	_, err = deterministicDecMode.Decode(nil, b)
	require.Error(t, err)

	decodedValue, err := ccf.Decode(nil, b)
	require.NoError(t, err)

	// Test decoded event has unsorted fields.
	// If field is struct (such as NodeInfo), struct fields should be unsorted as well.

	evt, ok := decodedValue.(cadence.Event)
	require.True(t, ok)

	fields := cadence.FieldsMappedByName(evt)
	require.Len(t, fields, 9)

	evtType, ok := decodedValue.Type().(*cadence.EventType)
	require.True(t, ok)

	typeFields := evtType.FieldsMappedByName()

	require.Len(t, typeFields, 9)

	// field 0: counter
	require.Equal(t,
		cadence.UInt64Type,
		typeFields["counter"],
	)
	require.Equal(t,
		cadence.UInt64(1),
		fields["counter"],
	)

	// field 1: nodeInfo
	require.IsType(t,
		cadence.NewVariableSizedArrayType(newFlowIDTableStakingNodeInfoStructType()),
		typeFields["nodeInfo"],
	)
	nodeInfos, ok := fields["nodeInfo"].(cadence.Array)
	require.True(t, ok)
	testNodeInfos(t, nodeInfos)

	// field 2: firstView
	require.Equal(t,
		cadence.UInt64Type,
		typeFields["firstView"],
	)
	require.Equal(t,
		cadence.UInt64(100),
		fields["firstView"],
	)

	// field 3: finalView
	require.Equal(t,
		cadence.UInt64Type,
		typeFields["finalView"],
	)
	require.Equal(t,
		cadence.UInt64(200),
		fields["finalView"],
	)

	// field 4: collectorClusters
	require.Equal(t,
		cadence.NewVariableSizedArrayType(newFlowClusterQCClusterStructType()),
		typeFields["collectorClusters"],
	)
	epochCollectors, ok := fields["collectorClusters"].(cadence.Array)
	require.True(t, ok)
	testEpochCollectors(t, epochCollectors)

	// field 5: randomSource
	require.Equal(t,
		cadence.StringType,
		typeFields["randomSource"],
	)
	require.Equal(t,
		cadence.String("01020304"),
		fields["randomSource"],
	)

	// field 6: DKGPhase1FinalView
	require.Equal(t,
		cadence.UInt64Type,
		typeFields["DKGPhase1FinalView"],
	)
	require.Equal(t,
		cadence.UInt64(150),
		fields["DKGPhase1FinalView"],
	)

	// field 7: DKGPhase2FinalView
	require.Equal(t,
		cadence.UInt64Type,
		typeFields["DKGPhase2FinalView"],
	)
	require.Equal(t,
		cadence.UInt64(160),
		fields["DKGPhase2FinalView"],
	)

	// field 8: DKGPhase3FinalView
	require.Equal(t,
		cadence.UInt64Type,
		typeFields["DKGPhase3FinalView"],
	)
	require.Equal(t,
		cadence.UInt64(170),
		fields["DKGPhase3FinalView"],
	)
}

func testNodeInfos(t *testing.T, nodeInfos cadence.Array) {
	require.Len(t, nodeInfos.Values, 7)

	// Test nodeInfo 0

	node0, ok := nodeInfos.Values[0].(cadence.Struct)
	require.True(t, ok)

	node0Fields := cadence.FieldsMappedByName(node0)
	require.Len(t, node0Fields, 14)

	nodeInfoType, ok := node0.Type().(*cadence.StructType)
	require.True(t, ok)

	node0FieldTypes := nodeInfoType.FieldsMappedByName()
	require.Len(t, node0FieldTypes, 14)

	// field 0: id
	require.Equal(t,
		cadence.StringType,
		node0FieldTypes["id"],
	)
	require.Equal(t,
		cadence.String("0000000000000000000000000000000000000000000000000000000000000001"),
		node0Fields["id"],
	)

	// field 1: role
	require.Equal(t,
		cadence.UInt8Type,
		node0FieldTypes["role"],
	)
	require.Equal(t,
		cadence.UInt8(1),
		node0Fields["role"],
	)

	// field 2: networkingAddress
	require.Equal(t,
		cadence.StringType,
		node0FieldTypes["networkingAddress"],
	)
	require.Equal(t,
		cadence.String("1.flow.com"),
		node0Fields["networkingAddress"],
	)

	// field 3: networkingKey
	require.Equal(t,
		cadence.StringType,
		node0FieldTypes["networkingKey"],
	)
	require.Equal(t,
		cadence.String("378dbf45d85c614feb10d8bd4f78f4b6ef8eec7d987b937e123255444657fb3da031f232a507e323df3a6f6b8f50339c51d188e80c0e7a92420945cc6ca893fc"),
		node0Fields["networkingKey"],
	)

	// field 4: stakingKey
	require.Equal(t,
		cadence.StringType,
		node0FieldTypes["stakingKey"],
	)
	require.Equal(t,
		cadence.String("af4aade26d76bb2ab15dcc89adcef82a51f6f04b3cb5f4555214b40ec89813c7a5f95776ea4fe449de48166d0bbc59b919b7eabebaac9614cf6f9461fac257765415f4d8ef1376a2365ec9960121888ea5383d88a140c24c29962b0a14e4e4e7"),
		node0Fields["stakingKey"],
	)

	// field 5: tokensStaked
	require.Equal(t,
		cadence.UFix64Type,
		node0FieldTypes["tokensStaked"],
	)
	require.Equal(t,
		ufix64FromString("0.00000000"),
		node0Fields["tokensStaked"],
	)

	// field 6: tokensCommitted
	require.Equal(t,
		cadence.UFix64Type,
		node0FieldTypes["tokensCommitted"],
	)
	require.Equal(t,
		ufix64FromString("1350000.00000000"),
		node0Fields["tokensCommitted"],
	)

	// field 7: tokensUnstaking
	require.Equal(t,
		cadence.UFix64Type,
		node0FieldTypes["tokensUnstaking"],
	)
	require.Equal(t,
		ufix64FromString("0.00000000"),
		node0Fields["tokensUnstaking"],
	)

	// field 8: tokensUnstaked
	require.Equal(t,
		cadence.UFix64Type,
		node0FieldTypes["tokensUnstaked"],
	)
	require.Equal(t,
		ufix64FromString("0.00000000"),
		node0Fields["tokensUnstaked"],
	)

	// field 9: tokensRewarded
	require.Equal(t,
		cadence.UFix64Type,
		node0FieldTypes["tokensRewarded"],
	)
	require.Equal(t,
		ufix64FromString("0.00000000"),
		node0Fields["tokensRewarded"],
	)

	// field 10: delegators
	require.Equal(t,
		cadence.NewVariableSizedArrayType(cadence.UInt32Type),
		node0FieldTypes["delegators"],
	)
	delegators, ok := node0Fields["delegators"].(cadence.Array)

	require.True(t, ok)
	require.Len(t, delegators.Values, 0)

	// field 11: delegatorIDCounter
	require.Equal(
		t,
		cadence.UInt32Type,
		node0FieldTypes["delegatorIDCounter"],
	)
	require.Equal(t,
		cadence.UInt32(0),
		node0Fields["delegatorIDCounter"],
	)

	// field 12: tokensRequestedToUnstake
	require.Equal(t,
		cadence.UFix64Type,
		node0FieldTypes["tokensRequestedToUnstake"],
	)
	require.Equal(t,
		ufix64FromString("0.00000000"),
		node0Fields["tokensRequestedToUnstake"],
	)

	// field 13: initialWeight
	require.Equal(t,
		cadence.UInt64Type,
		node0FieldTypes["initialWeight"],
	)
	require.Equal(t,
		cadence.UInt64(100),
		node0Fields["initialWeight"],
	)

	// Test nodeInfo 6 (last nodeInfo struct)

	node6, ok := nodeInfos.Values[6].(cadence.Struct)
	require.True(t, ok)
	node6Fields := cadence.FieldsMappedByName(node6)
	require.Len(t, node6Fields, 14)

	nodeInfoType, ok = node6.Type().(*cadence.StructType)
	require.True(t, ok)

	node6FieldTypes := nodeInfoType.FieldsMappedByName()
	require.Len(t, node6FieldTypes, 14)

	// field 0: id
	require.Equal(t,
		cadence.StringType,
		node6FieldTypes["id"],
	)
	require.Equal(t,
		cadence.String("0000000000000000000000000000000000000000000000000000000000000031"),
		node6Fields["id"],
	)

	// field 1: role
	require.Equal(t,
		cadence.UInt8Type,
		node6FieldTypes["role"],
	)
	require.Equal(t,
		cadence.UInt8(4),
		node6Fields["role"],
	)

	// field 2: networkingAddress
	require.Equal(t,
		cadence.StringType,
		node6FieldTypes["networkingAddress"],
	)
	require.Equal(t,
		cadence.String("31.flow.com"),
		node6Fields["networkingAddress"],
	)

	// field 3: networkingKey
	require.Equal(t,
		cadence.StringType,
		node6FieldTypes["networkingKey"],
	)
	require.Equal(t,
		cadence.String("697241208dcc9142b6f53064adc8ff1c95760c68beb2ba083c1d005d40181fd7a1b113274e0163c053a3addd47cd528ec6a1f190cf465aac87c415feaae011ae"),
		node6Fields["networkingKey"],
	)

	// field 4: stakingKey
	require.Equal(t,
		cadence.StringType,
		node6FieldTypes["stakingKey"],
	)
	require.Equal(t,
		cadence.String("b1f97d0a06020eca97352e1adde72270ee713c7daf58da7e74bf72235321048b4841bdfc28227964bf18e371e266e32107d238358848bcc5d0977a0db4bda0b4c33d3874ff991e595e0f537c7b87b4ddce92038ebc7b295c9ea20a1492302aa7"),
		node6Fields["stakingKey"],
	)

	// field 5: tokensStaked
	require.Equal(t,
		cadence.UFix64Type,
		node6FieldTypes["tokensStaked"],
	)
	require.Equal(t,
		ufix64FromString("0.00000000"),
		node6Fields["tokensStaked"],
	)

	// field 6: tokensCommitted
	require.Equal(t,
		cadence.UFix64Type,
		node6FieldTypes["tokensCommitted"],
	)
	require.Equal(t,
		ufix64FromString("1350000.00000000"),
		node6Fields["tokensCommitted"],
	)

	// field 7: tokensUnstaking
	require.Equal(t,
		cadence.UFix64Type,
		node6FieldTypes["tokensUnstaking"],
	)
	require.Equal(t,
		ufix64FromString("0.00000000"),
		node6Fields["tokensUnstaking"],
	)

	// field 8: tokensUnstaked
	require.Equal(t,
		cadence.UFix64Type,
		node6FieldTypes["tokensUnstaked"],
	)
	require.Equal(t,
		ufix64FromString("0.00000000"),
		node6Fields["tokensUnstaked"],
	)

	// field 9: tokensRewarded
	require.Equal(t,
		cadence.UFix64Type,
		node6FieldTypes["tokensRewarded"],
	)
	require.Equal(t,
		ufix64FromString("0.00000000"),
		node6Fields["tokensRewarded"],
	)

	// field 10: delegators
	require.Equal(t,
		cadence.NewVariableSizedArrayType(cadence.UInt32Type),
		node6FieldTypes["delegators"],
	)
	delegators, ok = node6Fields["delegators"].(cadence.Array)
	require.True(t, ok)
	require.Len(t, delegators.Values, 0)

	// field 11: delegatorIDCounter
	require.Equal(t,
		cadence.UInt32Type,
		node6FieldTypes["delegatorIDCounter"],
	)
	require.Equal(t,
		cadence.UInt32(0),
		node6Fields["delegatorIDCounter"],
	)

	// field 12: tokensRequestedToUnstake
	require.Equal(t,
		cadence.UFix64Type,
		node6FieldTypes["tokensRequestedToUnstake"],
	)
	require.Equal(t,
		ufix64FromString("0.00000000"),
		node6Fields["tokensRequestedToUnstake"],
	)

	// field 13: initialWeight
	require.Equal(t,
		cadence.UInt64Type,
		node6FieldTypes["initialWeight"],
	)
	require.Equal(t,
		cadence.UInt64(100),
		node6Fields["initialWeight"],
	)
}

func testEpochCollectors(t *testing.T, collectors cadence.Array) {
	require.Len(t, collectors.Values, 2)

	// collector 0
	collector0, ok := collectors.Values[0].(cadence.Struct)
	require.True(t, ok)

	collector0Type, ok := collector0.Type().(*cadence.StructType)
	require.True(t, ok)

	collector0Fields := cadence.FieldsMappedByName(collector0)

	collector0FieldTypes := collector0Type.FieldsMappedByName()

	// field 0: index
	require.Equal(t,
		cadence.UInt16Type,
		collector0FieldTypes["index"],
	)
	require.Equal(t,
		cadence.UInt16(0),
		collector0Fields["index"],
	)

	// field 1: nodeWeights
	require.Equal(t,
		cadence.NewDictionaryType(cadence.StringType, cadence.UInt64Type),
		collector0FieldTypes["nodeWeights"],
	)
	weights, ok := collector0Fields["nodeWeights"].(cadence.Dictionary)
	require.True(t, ok)
	require.Len(t, weights.Pairs, 2)

	require.Equal(t,
		cadence.KeyValuePair{
			Key:   cadence.String("0000000000000000000000000000000000000000000000000000000000000001"),
			Value: cadence.UInt64(100),
		},
		weights.Pairs[0],
	)
	require.Equal(t,
		cadence.KeyValuePair{
			Key:   cadence.String("0000000000000000000000000000000000000000000000000000000000000002"),
			Value: cadence.UInt64(100),
		},
		weights.Pairs[1],
	)

	// field 2: totalWeight
	require.Equal(t,
		cadence.UInt64Type,
		collector0FieldTypes["totalWeight"],
	)
	require.Equal(t,
		cadence.NewUInt64(100),
		collector0Fields["totalWeight"],
	)

	// field 3: generatedVotes
	require.Equal(t,
		cadence.NewDictionaryType(cadence.StringType, newFlowClusterQCVoteStructType()),
		collector0FieldTypes["generatedVotes"],
	)
	generatedVotes, ok := collector0Fields["generatedVotes"].(cadence.Dictionary)
	require.True(t, ok)
	require.Len(t, generatedVotes.Pairs, 0)

	// field 4: uniqueVoteMessageTotalWeights
	require.Equal(t,
		cadence.NewDictionaryType(cadence.StringType, cadence.UInt64Type),
		collector0FieldTypes["uniqueVoteMessageTotalWeights"],
	)
	uniqueVoteMessageTotalWeights, ok := collector0Fields["uniqueVoteMessageTotalWeights"].(cadence.Dictionary)
	require.True(t, ok)
	require.Len(t, uniqueVoteMessageTotalWeights.Pairs, 0)

	// collector 1
	collector1, ok := collectors.Values[1].(cadence.Struct)
	require.True(t, ok)

	collector1Type, ok := collector1.Type().(*cadence.StructType)
	require.True(t, ok)

	collector1Fields := collector1.FieldsMappedByName()
	collector1FieldTypes := collector1Type.FieldsMappedByName()

	// field 0: index
	require.Equal(t,
		cadence.UInt16Type,
		collector1FieldTypes["index"],
	)
	require.Equal(t,
		cadence.UInt16(1),
		collector1Fields["index"],
	)

	// field 1: nodeWeights
	require.Equal(t,
		cadence.NewDictionaryType(cadence.StringType, cadence.UInt64Type),
		collector1FieldTypes["nodeWeights"],
	)
	weights, ok = collector1Fields["nodeWeights"].(cadence.Dictionary)
	require.True(t, ok)
	require.Len(t, weights.Pairs, 2)
	require.Equal(t,
		cadence.KeyValuePair{
			Key:   cadence.String("0000000000000000000000000000000000000000000000000000000000000003"),
			Value: cadence.UInt64(100),
		},
		weights.Pairs[0],
	)
	require.Equal(t,
		cadence.KeyValuePair{
			Key:   cadence.String("0000000000000000000000000000000000000000000000000000000000000004"),
			Value: cadence.UInt64(100),
		},
		weights.Pairs[1],
	)

	// field 2: totalWeight
	require.Equal(t,
		cadence.UInt64Type,
		collector1FieldTypes["totalWeight"],
	)
	require.Equal(t,
		cadence.NewUInt64(0),
		collector1Fields["totalWeight"],
	)

	// field 3: generatedVotes
	require.Equal(t,
		cadence.NewDictionaryType(cadence.StringType, newFlowClusterQCVoteStructType()),
		collector1FieldTypes["generatedVotes"],
	)
	generatedVotes, ok = collector1Fields["generatedVotes"].(cadence.Dictionary)
	require.True(t, ok)
	require.Len(t, generatedVotes.Pairs, 0)

	// field 4: uniqueVoteMessageTotalWeights
	require.Equal(t,
		cadence.NewDictionaryType(cadence.StringType, cadence.UInt64Type),
		collector1FieldTypes["uniqueVoteMessageTotalWeights"],
	)
	uniqueVoteMessageTotalWeights, ok = collector1Fields["uniqueVoteMessageTotalWeights"].(cadence.Dictionary)
	require.True(t, ok)
	require.Len(t, uniqueVoteMessageTotalWeights.Pairs, 0)
}

func TestEpochCommitEvent(t *testing.T) {
	event := createEpochCommittedEvent()

	b, err := ccf.Encode(event)
	require.NoError(t, err)

	// Test that encoded value isn't sorted.
	_, err = deterministicDecMode.Decode(nil, b)
	require.Error(t, err)

	decodedValue, err := ccf.Decode(nil, b)
	require.NoError(t, err)

	// Test decoded event has unsorted fields.
	// If field is struct (such as ClusterQC), struct fields should be unsorted as well.

	evt, ok := decodedValue.(cadence.Event)
	require.True(t, ok)

	fields := cadence.FieldsMappedByName(evt)
	require.Len(t, fields, 3)

	evtType, ok := decodedValue.Type().(*cadence.EventType)
	require.True(t, ok)

	fieldTypes := evtType.FieldsMappedByName()
	require.Len(t, fieldTypes, 3)

	// field 0: counter
	require.Equal(t,
		cadence.UInt64Type,
		fieldTypes["counter"],
	)
	require.Equal(t,
		cadence.UInt64(1),
		fields["counter"],
	)

	// field 1: clusterQCs
	require.Equal(t,
		cadence.NewVariableSizedArrayType(newFlowClusterQCClusterQCStructType()),
		fieldTypes["clusterQCs"],
	)
	clusterQCs, ok := fields["clusterQCs"].(cadence.Array)
	require.True(t, ok)
	testClusterQCs(t, clusterQCs)

	// field 2: dkgPubKeys
	require.Equal(t,
		cadence.NewVariableSizedArrayType(cadence.StringType),
		fieldTypes["dkgPubKeys"],
	)
	dkgPubKeys, ok := fields["dkgPubKeys"].(cadence.Array)
	require.True(t, ok)

	require.Equal(t,
		[]cadence.Value{
			cadence.String("8c588266db5f5cda629e83f8aa04ae9413593fac19e4865d06d291c9d14fbdd9bdb86a7a12f9ef8590c79cb635e3163315d193087e9336092987150d0cd2b14ac6365f7dc93eec573752108b8c12368abb65f0652d9f644e5aed611c37926950"),
			cadence.String("87a339e4e5c74f089da20a33f515d8c8f4464ab53ede5a74aa2432cd1ae66d522da0c122249ee176cd747ddc83ca81090498389384201614caf51eac392c1c0a916dfdcfbbdf7363f9552b6468434add3d3f6dc91a92bbe3ee368b59b7828488"),
		},
		dkgPubKeys.Values,
	)
}

func testClusterQCs(t *testing.T, clusterQCs cadence.Array) {
	require.Len(t, clusterQCs.Values, 2)

	// Test clusterQC0

	clusterQC0, ok := clusterQCs.Values[0].(cadence.Struct)
	require.True(t, ok)

	clusterQCType, ok := clusterQC0.Type().(*cadence.StructType)
	require.True(t, ok)

	clusterQC0Fields := clusterQC0.FieldsMappedByName()
	clusterQC0FieldTypes := clusterQCType.FieldsMappedByName()

	// field 0: index
	require.Equal(t,
		cadence.UInt16Type,
		clusterQC0FieldTypes["index"],
	)
	require.Equal(t,
		cadence.UInt16(0),
		clusterQC0Fields["index"],
	)

	// field 1: voteSignatures
	require.Equal(t,
		cadence.NewVariableSizedArrayType(cadence.StringType),
		clusterQC0FieldTypes["voteSignatures"],
	)
	sigs, ok := clusterQC0Fields["voteSignatures"].(cadence.Array)
	require.True(t, ok)
	require.Equal(t,
		[]cadence.Value{
			cadence.String("a39cd1e1bf7e2fb0609b7388ce5215a6a4c01eef2aee86e1a007faa28a6b2a3dc876e11bb97cdb26c3846231d2d01e4d"),
			cadence.String("91673ad9c717d396c9a0953617733c128049ac1a639653d4002ab245b121df1939430e313bcbfd06948f6a281f6bf853"),
		},
		sigs.Values,
	)

	// field 2: voteMessage
	require.Equal(t,
		cadence.StringType,
		clusterQC0FieldTypes["voteMessage"],
	)
	require.Equal(t,
		cadence.String("irrelevant_for_these_purposes"),
		clusterQC0Fields["voteMessage"],
	)

	// field 3: voterIDs
	require.Equal(t,
		cadence.NewVariableSizedArrayType(cadence.StringType),
		clusterQC0FieldTypes["voterIDs"],
	)
	ids, ok := clusterQC0Fields["voterIDs"].(cadence.Array)
	require.True(t, ok)
	require.Equal(t,
		[]cadence.Value{
			cadence.String("0000000000000000000000000000000000000000000000000000000000000001"),
			cadence.String("0000000000000000000000000000000000000000000000000000000000000002"),
		},
		ids.Values,
	)

	// Test clusterQC1

	clusterQC1, ok := clusterQCs.Values[1].(cadence.Struct)
	require.True(t, ok)

	clusterQC1Type, ok := clusterQC1.Type().(*cadence.StructType)
	require.True(t, ok)

	clusterQC1Fields := clusterQC1.FieldsMappedByName()
	clusterQC1FieldTypes := clusterQC1Type.FieldsMappedByName()

	// field 0: index
	require.Equal(t,
		cadence.UInt16Type,
		clusterQC1FieldTypes["index"],
	)
	require.Equal(t,
		cadence.UInt16(1),
		clusterQC1Fields["index"],
	)

	// field 1: voteSignatures
	require.Equal(t,
		cadence.NewVariableSizedArrayType(cadence.StringType),
		clusterQC1FieldTypes["voteSignatures"],
	)
	sigs, ok = clusterQC1Fields["voteSignatures"].(cadence.Array)
	require.True(t, ok)
	require.Equal(t,
		[]cadence.Value{
			cadence.String("b2bff159971852ed63e72c37991e62c94822e52d4fdcd7bf29aaf9fb178b1c5b4ce20dd9594e029f3574cb29533b857a"),
			cadence.String("9931562f0248c9195758da3de4fb92f24fa734cbc20c0cb80280163560e0e0348f843ac89ecbd3732e335940c1e8dccb"),
		},
		sigs.Values,
	)

	// field 2: voteMessage
	require.Equal(t,
		cadence.StringType,
		clusterQC1FieldTypes["voteMessage"],
	)
	require.Equal(t,
		cadence.String("irrelevant_for_these_purposes"),
		clusterQC1Fields["voteMessage"],
	)

	// field 3: voterIDs
	require.Equal(t,
		cadence.NewVariableSizedArrayType(cadence.StringType),
		clusterQC1FieldTypes["voterIDs"],
	)
	ids, ok = clusterQC1Fields["voterIDs"].(cadence.Array)
	require.True(t, ok)
	require.Equal(t,
		[]cadence.Value{
			cadence.String("0000000000000000000000000000000000000000000000000000000000000003"),
			cadence.String("0000000000000000000000000000000000000000000000000000000000000004"),
		},
		ids.Values,
	)
}

func TestVersionBeaconEvent(t *testing.T) {
	event := createVersionBeaconEvent()

	b, err := ccf.Encode(event)
	require.NoError(t, err)

	// Test that encoded value isn't sorted.
	_, err = deterministicDecMode.Decode(nil, b)
	require.Error(t, err)

	decodedValue, err := ccf.Decode(nil, b)
	require.NoError(t, err)

	// Test decoded event has unsorted fields.
	// If field is struct (such as semver), struct fields should be unsorted as well.

	evt, ok := decodedValue.(cadence.Event)
	require.True(t, ok)

	fields := cadence.FieldsMappedByName(evt)
	require.Len(t, fields, 2)

	evtType, ok := decodedValue.Type().(*cadence.EventType)
	require.True(t, ok)

	fieldTypes := evtType.FieldsMappedByName()
	require.Len(t, fieldTypes, 2)

	// field 0: versionBoundaries
	require.Equal(t,
		cadence.NewVariableSizedArrayType(newNodeVersionBeaconVersionBoundaryStructType()),
		fieldTypes["versionBoundaries"],
	)
	versionBoundaries, ok := fields["versionBoundaries"].(cadence.Array)
	require.True(t, ok)
	testVersionBoundaries(t, versionBoundaries)

	// field 1: sequence
	require.Equal(t,
		cadence.UInt64Type,
		fieldTypes["sequence"],
	)
	require.Equal(t,
		cadence.UInt64(5),
		fields["sequence"],
	)
}

func testVersionBoundaries(t *testing.T, versionBoundaries cadence.Array) {
	require.Len(t, versionBoundaries.Values, 1)

	boundary, ok := versionBoundaries.Values[0].(cadence.Struct)
	require.True(t, ok)

	fields := cadence.FieldsMappedByName(boundary)
	require.Len(t, fields, 2)

	boundaryType, ok := boundary.Type().(*cadence.StructType)
	require.True(t, ok)

	fieldTypes := boundaryType.FieldsMappedByName()
	require.Len(t, fieldTypes, 2)

	// field 0: blockHeight
	require.Equal(t,
		cadence.UInt64Type,
		fieldTypes["blockHeight"],
	)
	require.Equal(t,
		cadence.UInt64(44),
		fields["blockHeight"],
	)

	// field 1: version
	require.Equal(t,
		newNodeVersionBeaconSemverStructType(),
		fieldTypes["version"],
	)
	version, ok := fields["version"].(cadence.Struct)
	require.True(t, ok)
	testSemver(t, version)
}

func testSemver(t *testing.T, version cadence.Struct) {
	versionFields := cadence.FieldsMappedByName(version)

	require.Len(t, versionFields, 4)

	semverType, ok := version.Type().(*cadence.StructType)
	require.True(t, ok)

	fieldTypes := semverType.FieldsMappedByName()
	require.Len(t, fieldTypes, 4)

	// field 0: preRelease
	require.Equal(t,
		cadence.NewOptionalType(cadence.StringType),
		fieldTypes["preRelease"],
	)
	require.Equal(t,
		cadence.NewOptional(cadence.String("")),
		versionFields["preRelease"],
	)

	// field 1: major
	require.Equal(t,
		cadence.UInt8Type,
		fieldTypes["major"],
	)
	require.Equal(t,
		cadence.UInt8(2),
		versionFields["major"],
	)

	// field 2: minor
	require.Equal(t,
		cadence.UInt8Type,
		fieldTypes["minor"],
	)
	require.Equal(t,
		cadence.UInt8(13),
		versionFields["minor"],
	)

	// field 3: patch
	require.Equal(t,
		cadence.UInt8Type,
		fieldTypes["patch"],
	)
	require.Equal(t,
		cadence.UInt8(7),
		versionFields["patch"],
	)
}

func createEpochSetupEvent() cadence.Event {
	return cadence.NewEvent([]cadence.Value{
		// counter
		cadence.NewUInt64(1),

		// nodeInfo
		createEpochNodes(),

		// firstView
		cadence.NewUInt64(100),

		// finalView
		cadence.NewUInt64(200),

		// collectorClusters
		createEpochCollectors(),

		// randomSource
		cadence.String("01020304"),

		// DKGPhase1FinalView
		cadence.UInt64(150),

		// DKGPhase2FinalView
		cadence.UInt64(160),

		// DKGPhase3FinalView
		cadence.UInt64(170),
	}).WithType(newFlowEpochEpochSetupEventType())
}

func createEpochNodes() cadence.Array {

	nodeInfoType := newFlowIDTableStakingNodeInfoStructType()

	nodeInfo1 := cadence.NewStruct([]cadence.Value{
		// id
		cadence.String("0000000000000000000000000000000000000000000000000000000000000001"),

		// role
		cadence.UInt8(1),

		// networkingAddress
		cadence.String("1.flow.com"),

		// networkingKey
		cadence.String("378dbf45d85c614feb10d8bd4f78f4b6ef8eec7d987b937e123255444657fb3da031f232a507e323df3a6f6b8f50339c51d188e80c0e7a92420945cc6ca893fc"),

		// stakingKey
		cadence.String("af4aade26d76bb2ab15dcc89adcef82a51f6f04b3cb5f4555214b40ec89813c7a5f95776ea4fe449de48166d0bbc59b919b7eabebaac9614cf6f9461fac257765415f4d8ef1376a2365ec9960121888ea5383d88a140c24c29962b0a14e4e4e7"),

		// tokensStaked
		ufix64FromString("0.00000000"),

		// tokensCommitted
		ufix64FromString("1350000.00000000"),

		// tokensUnstaking
		ufix64FromString("0.00000000"),

		// tokensUnstaked
		ufix64FromString("0.00000000"),

		// tokensRewarded
		ufix64FromString("0.00000000"),

		// delegators
		cadence.NewArray([]cadence.Value{}).WithType(cadence.NewVariableSizedArrayType(cadence.UInt32Type)),

		// delegatorIDCounter
		cadence.UInt32(0),

		// tokensRequestedToUnstake
		ufix64FromString("0.00000000"),

		// initialWeight
		cadence.UInt64(100),
	}).WithType(nodeInfoType)

	nodeInfo2 := cadence.NewStruct([]cadence.Value{
		// id
		cadence.String("0000000000000000000000000000000000000000000000000000000000000002"),

		// role
		cadence.UInt8(1),

		// networkingAddress
		cadence.String("2.flow.com"),

		// networkingKey
		cadence.String("378dbf45d85c614feb10d8bd4f78f4b6ef8eec7d987b937e123255444657fb3da031f232a507e323df3a6f6b8f50339c51d188e80c0e7a92420945cc6ca893fc"),

		// stakingKey
		cadence.String("af4aade26d76bb2ab15dcc89adcef82a51f6f04b3cb5f4555214b40ec89813c7a5f95776ea4fe449de48166d0bbc59b919b7eabebaac9614cf6f9461fac257765415f4d8ef1376a2365ec9960121888ea5383d88a140c24c29962b0a14e4e4e7"),

		// tokensStaked
		ufix64FromString("0.00000000"),

		// tokensCommitted
		ufix64FromString("1350000.00000000"),

		// tokensUnstaking
		ufix64FromString("0.00000000"),

		// tokensUnstaked
		ufix64FromString("0.00000000"),

		// tokensRewarded
		ufix64FromString("0.00000000"),

		// delegators
		cadence.NewArray([]cadence.Value{}).WithType(cadence.NewVariableSizedArrayType(cadence.UInt32Type)),

		// delegatorIDCounter
		cadence.UInt32(0),

		// tokensRequestedToUnstake
		ufix64FromString("0.00000000"),

		// initialWeight
		cadence.UInt64(100),
	}).WithType(nodeInfoType)

	nodeInfo3 := cadence.NewStruct([]cadence.Value{
		// id
		cadence.String("0000000000000000000000000000000000000000000000000000000000000003"),

		// role
		cadence.UInt8(1),

		// networkingAddress
		cadence.String("3.flow.com"),

		// networkingKey
		cadence.String("378dbf45d85c614feb10d8bd4f78f4b6ef8eec7d987b937e123255444657fb3da031f232a507e323df3a6f6b8f50339c51d188e80c0e7a92420945cc6ca893fc"),

		// stakingKey
		cadence.String("af4aade26d76bb2ab15dcc89adcef82a51f6f04b3cb5f4555214b40ec89813c7a5f95776ea4fe449de48166d0bbc59b919b7eabebaac9614cf6f9461fac257765415f4d8ef1376a2365ec9960121888ea5383d88a140c24c29962b0a14e4e4e7"),

		// tokensStaked
		ufix64FromString("0.00000000"),

		// tokensCommitted
		ufix64FromString("1350000.00000000"),

		// tokensUnstaking
		ufix64FromString("0.00000000"),

		// tokensUnstaked
		ufix64FromString("0.00000000"),

		// tokensRewarded
		ufix64FromString("0.00000000"),

		// delegators
		cadence.NewArray([]cadence.Value{}).WithType(cadence.NewVariableSizedArrayType(cadence.UInt32Type)),

		// delegatorIDCounter
		cadence.UInt32(0),

		// tokensRequestedToUnstake
		ufix64FromString("0.00000000"),

		// initialWeight
		cadence.UInt64(100),
	}).WithType(nodeInfoType)

	nodeInfo4 := cadence.NewStruct([]cadence.Value{
		// id
		cadence.String("0000000000000000000000000000000000000000000000000000000000000004"),

		// role
		cadence.UInt8(1),

		// networkingAddress
		cadence.String("4.flow.com"),

		// networkingKey
		cadence.String("378dbf45d85c614feb10d8bd4f78f4b6ef8eec7d987b937e123255444657fb3da031f232a507e323df3a6f6b8f50339c51d188e80c0e7a92420945cc6ca893fc"),

		// stakingKey
		cadence.String("af4aade26d76bb2ab15dcc89adcef82a51f6f04b3cb5f4555214b40ec89813c7a5f95776ea4fe449de48166d0bbc59b919b7eabebaac9614cf6f9461fac257765415f4d8ef1376a2365ec9960121888ea5383d88a140c24c29962b0a14e4e4e7"),

		// tokensStaked
		ufix64FromString("0.00000000"),

		// tokensCommitted
		ufix64FromString("1350000.00000000"),

		// tokensUnstaking
		ufix64FromString("0.00000000"),

		// tokensUnstaked
		ufix64FromString("0.00000000"),

		// tokensRewarded
		ufix64FromString("0.00000000"),

		// delegators
		cadence.NewArray([]cadence.Value{}).WithType(cadence.NewVariableSizedArrayType(cadence.UInt32Type)),

		// delegatorIDCounter
		cadence.UInt32(0),

		// tokensRequestedToUnstake
		ufix64FromString("0.00000000"),

		// initialWeight
		cadence.UInt64(100),
	}).WithType(nodeInfoType)

	nodeInfo5 := cadence.NewStruct([]cadence.Value{
		// id
		cadence.String("0000000000000000000000000000000000000000000000000000000000000011"),

		// role
		cadence.UInt8(2),

		// networkingAddress
		cadence.String("11.flow.com"),

		// networkingKey
		cadence.String("cfdfe8e4362c8f79d11772cb7277ab16e5033a63e8dd5d34caf1b041b77e5b2d63c2072260949ccf8907486e4cfc733c8c42ca0e4e208f30470b0d950856cd47"),

		// stakingKey
		cadence.String("8207559cd7136af378bba53a8f0196dee3849a3ab02897c1995c3e3f6ca0c4a776c3ae869d1ddbb473090054be2400ad06d7910aa2c5d1780220fdf3765a3c1764bce10c6fe66a5a2be51a422e878518bd750424bb56b8a0ecf0f8ad2057e83f"),

		// tokensStaked
		ufix64FromString("0.00000000"),

		// tokensCommitted
		ufix64FromString("1350000.00000000"),

		// tokensUnstaking
		ufix64FromString("0.00000000"),

		// tokensUnstaked
		ufix64FromString("0.00000000"),

		// tokensRewarded
		ufix64FromString("0.00000000"),

		// delegators
		cadence.NewArray([]cadence.Value{}).WithType(cadence.NewVariableSizedArrayType(cadence.UInt32Type)),

		// delegatorIDCounter
		cadence.UInt32(0),

		// tokensRequestedToUnstake
		ufix64FromString("0.00000000"),

		// initialWeight
		cadence.UInt64(100),
	}).WithType(nodeInfoType)

	nodeInfo6 := cadence.NewStruct([]cadence.Value{
		// id
		cadence.String("0000000000000000000000000000000000000000000000000000000000000021"),

		// role
		cadence.UInt8(3),

		// networkingAddress
		cadence.String("21.flow.com"),

		// networkingKey
		cadence.String("d64318ba0dbf68f3788fc81c41d507c5822bf53154530673127c66f50fe4469ccf1a054a868a9f88506a8999f2386d86fcd2b901779718cba4fb53c2da258f9e"),

		// stakingKey
		cadence.String("880b162b7ec138b36af401d07868cb08d25746d905395edbb4625bdf105d4bb2b2f4b0f4ae273a296a6efefa7ce9ccb914e39947ce0e83745125cab05d62516076ff0173ed472d3791ccef937597c9ea12381d76f547a092a4981d77ff3fba83"),

		// tokensStaked
		ufix64FromString("0.00000000"),

		// tokensCommitted
		ufix64FromString("1350000.00000000"),

		// tokensUnstaking
		ufix64FromString("0.00000000"),

		// tokensUnstaked
		ufix64FromString("0.00000000"),

		// tokensRewarded
		ufix64FromString("0.00000000"),

		// delegators
		cadence.NewArray([]cadence.Value{}).WithType(cadence.NewVariableSizedArrayType(cadence.UInt32Type)),

		// delegatorIDCounter
		cadence.UInt32(0),

		// tokensRequestedToUnstake
		ufix64FromString("0.00000000"),

		// initialWeight
		cadence.UInt64(100),
	}).WithType(nodeInfoType)

	nodeInfo7 := cadence.NewStruct([]cadence.Value{
		// id
		cadence.String("0000000000000000000000000000000000000000000000000000000000000031"),

		// role
		cadence.UInt8(4),

		// networkingAddress
		cadence.String("31.flow.com"),

		// networkingKey
		cadence.String("697241208dcc9142b6f53064adc8ff1c95760c68beb2ba083c1d005d40181fd7a1b113274e0163c053a3addd47cd528ec6a1f190cf465aac87c415feaae011ae"),

		// stakingKey
		cadence.String("b1f97d0a06020eca97352e1adde72270ee713c7daf58da7e74bf72235321048b4841bdfc28227964bf18e371e266e32107d238358848bcc5d0977a0db4bda0b4c33d3874ff991e595e0f537c7b87b4ddce92038ebc7b295c9ea20a1492302aa7"),

		// tokensStaked
		ufix64FromString("0.00000000"),

		// tokensCommitted
		ufix64FromString("1350000.00000000"),

		// tokensUnstaking
		ufix64FromString("0.00000000"),

		// tokensUnstaked
		ufix64FromString("0.00000000"),

		// tokensRewarded
		ufix64FromString("0.00000000"),

		// delegators
		cadence.NewArray([]cadence.Value{}).WithType(cadence.NewVariableSizedArrayType(cadence.UInt32Type)),

		// delegatorIDCounter
		cadence.UInt32(0),

		// tokensRequestedToUnstake
		ufix64FromString("0.00000000"),

		// initialWeight
		cadence.UInt64(100),
	}).WithType(nodeInfoType)

	return cadence.NewArray([]cadence.Value{
		nodeInfo1,
		nodeInfo2,
		nodeInfo3,
		nodeInfo4,
		nodeInfo5,
		nodeInfo6,
		nodeInfo7,
	}).WithType(cadence.NewVariableSizedArrayType(nodeInfoType))
}

func createEpochCollectors() cadence.Array {

	clusterType := newFlowClusterQCClusterStructType()

	voteType := newFlowClusterQCVoteStructType()

	cluster1 := cadence.NewStruct([]cadence.Value{
		// index
		cadence.NewUInt16(0),

		// nodeWeights
		cadence.NewDictionary([]cadence.KeyValuePair{
			{
				Key:   cadence.String("0000000000000000000000000000000000000000000000000000000000000001"),
				Value: cadence.UInt64(100),
			},
			{
				Key:   cadence.String("0000000000000000000000000000000000000000000000000000000000000002"),
				Value: cadence.UInt64(100),
			},
		}).WithType(cadence.NewMeteredDictionaryType(nil, cadence.StringType, cadence.UInt64Type)),

		// totalWeight
		cadence.NewUInt64(100),

		// generatedVotes
		cadence.NewDictionary(nil).WithType(cadence.NewDictionaryType(cadence.StringType, voteType)),

		// uniqueVoteMessageTotalWeights
		cadence.NewDictionary(nil).WithType(cadence.NewDictionaryType(cadence.StringType, cadence.UInt64Type)),
	}).WithType(clusterType)

	cluster2 := cadence.NewStruct([]cadence.Value{
		// index
		cadence.NewUInt16(1),

		// nodeWeights
		cadence.NewDictionary([]cadence.KeyValuePair{
			{
				Key:   cadence.String("0000000000000000000000000000000000000000000000000000000000000003"),
				Value: cadence.UInt64(100),
			},
			{
				Key:   cadence.String("0000000000000000000000000000000000000000000000000000000000000004"),
				Value: cadence.UInt64(100),
			},
		}).WithType(cadence.NewMeteredDictionaryType(nil, cadence.StringType, cadence.UInt64Type)),

		// totalWeight
		cadence.NewUInt64(0),

		// generatedVotes
		cadence.NewDictionary(nil).WithType(cadence.NewDictionaryType(cadence.StringType, voteType)),

		// uniqueVoteMessageTotalWeights
		cadence.NewDictionary(nil).WithType(cadence.NewDictionaryType(cadence.StringType, cadence.UInt64Type)),
	}).WithType(clusterType)

	return cadence.NewArray([]cadence.Value{
		cluster1,
		cluster2,
	}).WithType(cadence.NewVariableSizedArrayType(clusterType))
}

func createEpochCommittedEvent() cadence.Event {

	clusterQCType := newFlowClusterQCClusterQCStructType()

	cluster1 := cadence.NewStruct([]cadence.Value{
		// index
		cadence.UInt16(0),

		// voteSignatures
		cadence.NewArray([]cadence.Value{
			cadence.String("a39cd1e1bf7e2fb0609b7388ce5215a6a4c01eef2aee86e1a007faa28a6b2a3dc876e11bb97cdb26c3846231d2d01e4d"),
			cadence.String("91673ad9c717d396c9a0953617733c128049ac1a639653d4002ab245b121df1939430e313bcbfd06948f6a281f6bf853"),
		}).WithType(cadence.NewVariableSizedArrayType(cadence.StringType)),

		// voteMessage
		cadence.String("irrelevant_for_these_purposes"),

		// voterIDs
		cadence.NewArray([]cadence.Value{
			cadence.String("0000000000000000000000000000000000000000000000000000000000000001"),
			cadence.String("0000000000000000000000000000000000000000000000000000000000000002"),
		}).WithType(cadence.NewVariableSizedArrayType(cadence.StringType)),
	}).WithType(clusterQCType)

	cluster2 := cadence.NewStruct([]cadence.Value{
		// index
		cadence.UInt16(1),

		// voteSignatures
		cadence.NewArray([]cadence.Value{
			cadence.String("b2bff159971852ed63e72c37991e62c94822e52d4fdcd7bf29aaf9fb178b1c5b4ce20dd9594e029f3574cb29533b857a"),
			cadence.String("9931562f0248c9195758da3de4fb92f24fa734cbc20c0cb80280163560e0e0348f843ac89ecbd3732e335940c1e8dccb"),
		}).WithType(cadence.NewVariableSizedArrayType(cadence.StringType)),

		// voteMessage
		cadence.String("irrelevant_for_these_purposes"),

		// voterIDs
		cadence.NewArray([]cadence.Value{
			cadence.String("0000000000000000000000000000000000000000000000000000000000000003"),
			cadence.String("0000000000000000000000000000000000000000000000000000000000000004"),
		}).WithType(cadence.NewVariableSizedArrayType(cadence.StringType)),
	}).WithType(clusterQCType)

	return cadence.NewEvent([]cadence.Value{
		// counter
		cadence.NewUInt64(1),

		// clusterQCs
		cadence.NewArray([]cadence.Value{
			cluster1,
			cluster2,
		}).WithType(cadence.NewVariableSizedArrayType(clusterQCType)),

		// dkgPubKeys
		cadence.NewArray([]cadence.Value{
			cadence.String("8c588266db5f5cda629e83f8aa04ae9413593fac19e4865d06d291c9d14fbdd9bdb86a7a12f9ef8590c79cb635e3163315d193087e9336092987150d0cd2b14ac6365f7dc93eec573752108b8c12368abb65f0652d9f644e5aed611c37926950"),
			cadence.String("87a339e4e5c74f089da20a33f515d8c8f4464ab53ede5a74aa2432cd1ae66d522da0c122249ee176cd747ddc83ca81090498389384201614caf51eac392c1c0a916dfdcfbbdf7363f9552b6468434add3d3f6dc91a92bbe3ee368b59b7828488"),
		}).WithType(cadence.NewVariableSizedArrayType(cadence.StringType)),
	}).WithType(newFlowEpochEpochCommittedEventType())
}

func createVersionBeaconEvent() cadence.Event {
	versionBoundaryType := newNodeVersionBeaconVersionBoundaryStructType()

	semverType := newNodeVersionBeaconSemverStructType()

	semver := cadence.NewStruct([]cadence.Value{
		// preRelease
		cadence.NewOptional(cadence.String("")),

		// major
		cadence.UInt8(2),

		// minor
		cadence.UInt8(13),

		// patch
		cadence.UInt8(7),
	}).WithType(semverType)

	versionBoundary := cadence.NewStruct([]cadence.Value{
		// blockHeight
		cadence.UInt64(44),

		// version
		semver,
	}).WithType(versionBoundaryType)

	return cadence.NewEvent([]cadence.Value{
		// versionBoundaries
		cadence.NewArray([]cadence.Value{
			versionBoundary,
		}).WithType(cadence.NewVariableSizedArrayType(versionBoundaryType)),

		// sequence
		cadence.UInt64(5),
	}).WithType(newNodeVersionBeaconVersionBeaconEventType())
}

func newFlowClusterQCVoteStructType() cadence.Type {

	// A.01cf0e2f2f715450.FlowClusterQC.Vote

	address, _ := common.HexToAddress("01cf0e2f2f715450")
	location := common.NewAddressLocation(nil, address, "FlowClusterQC")

	return cadence.NewStructType(
		location,
		"FlowClusterQC.Vote",
		[]cadence.Field{
			{
				Identifier: "nodeID",
				Type:       cadence.StringType,
			},
			{
				Identifier: "signature",
				Type:       cadence.NewOptionalType(cadence.StringType),
			},
			{
				Identifier: "message",
				Type:       cadence.NewOptionalType(cadence.StringType),
			},
			{
				Identifier: "clusterIndex",
				Type:       cadence.UInt16Type,
			},
			{
				Identifier: "weight",
				Type:       cadence.UInt64Type,
			},
		},
		nil,
	)
}

func newFlowClusterQCClusterStructType() *cadence.StructType {

	// A.01cf0e2f2f715450.FlowClusterQC.Cluster

	address, _ := common.HexToAddress("01cf0e2f2f715450")
	location := common.NewAddressLocation(nil, address, "FlowClusterQC")

	return cadence.NewStructType(
		location,
		"FlowClusterQC.Cluster",
		[]cadence.Field{
			{
				Identifier: "index",
				Type:       cadence.UInt16Type,
			},
			{
				Identifier: "nodeWeights",
				Type:       cadence.NewDictionaryType(cadence.StringType, cadence.UInt64Type),
			},
			{
				Identifier: "totalWeight",
				Type:       cadence.UInt64Type,
			},
			{
				Identifier: "generatedVotes",
				Type:       cadence.NewDictionaryType(cadence.StringType, newFlowClusterQCVoteStructType()),
			},
			{
				Identifier: "uniqueVoteMessageTotalWeights",
				Type:       cadence.NewDictionaryType(cadence.StringType, cadence.UInt64Type),
			},
		},
		nil,
	)
}

func newFlowIDTableStakingNodeInfoStructType() *cadence.StructType {

	// A.01cf0e2f2f715450.FlowIDTableStaking.NodeInfo

	address, _ := common.HexToAddress("01cf0e2f2f715450")
	location := common.NewAddressLocation(nil, address, "FlowIDTableStaking")

	return cadence.NewStructType(
		location,
		"FlowIDTableStaking.NodeInfo",
		[]cadence.Field{
			{
				Identifier: "id",
				Type:       cadence.StringType,
			},
			{
				Identifier: "role",
				Type:       cadence.UInt8Type,
			},
			{
				Identifier: "networkingAddress",
				Type:       cadence.StringType,
			},
			{
				Identifier: "networkingKey",
				Type:       cadence.StringType,
			},
			{
				Identifier: "stakingKey",
				Type:       cadence.StringType,
			},
			{
				Identifier: "tokensStaked",
				Type:       cadence.UFix64Type,
			},
			{
				Identifier: "tokensCommitted",
				Type:       cadence.UFix64Type,
			},
			{
				Identifier: "tokensUnstaking",
				Type:       cadence.UFix64Type,
			},
			{
				Identifier: "tokensUnstaked",
				Type:       cadence.UFix64Type,
			},
			{
				Identifier: "tokensRewarded",
				Type:       cadence.UFix64Type,
			},
			{
				Identifier: "delegators",
				Type:       cadence.NewVariableSizedArrayType(cadence.UInt32Type),
			},
			{
				Identifier: "delegatorIDCounter",
				Type:       cadence.UInt32Type,
			},
			{
				Identifier: "tokensRequestedToUnstake",
				Type:       cadence.UFix64Type,
			},
			{
				Identifier: "initialWeight",
				Type:       cadence.UInt64Type,
			},
		},
		nil,
	)
}

func newFlowEpochEpochSetupEventType() *cadence.EventType {

	// A.01cf0e2f2f715450.FlowEpoch.EpochSetup

	address, _ := common.HexToAddress("01cf0e2f2f715450")
	location := common.NewAddressLocation(nil, address, "FlowEpoch")

	return cadence.NewEventType(
		location,
		"FlowEpoch.EpochSetup",
		[]cadence.Field{
			{
				Identifier: "counter",
				Type:       cadence.UInt64Type,
			},
			{
				Identifier: "nodeInfo",
				Type:       cadence.NewVariableSizedArrayType(newFlowIDTableStakingNodeInfoStructType()),
			},
			{
				Identifier: "firstView",
				Type:       cadence.UInt64Type,
			},
			{
				Identifier: "finalView",
				Type:       cadence.UInt64Type,
			},
			{
				Identifier: "collectorClusters",
				Type:       cadence.NewVariableSizedArrayType(newFlowClusterQCClusterStructType()),
			},
			{
				Identifier: "randomSource",
				Type:       cadence.StringType,
			},
			{
				Identifier: "DKGPhase1FinalView",
				Type:       cadence.UInt64Type,
			},
			{
				Identifier: "DKGPhase2FinalView",
				Type:       cadence.UInt64Type,
			},
			{
				Identifier: "DKGPhase3FinalView",
				Type:       cadence.UInt64Type,
			},
		},
		nil,
	)
}

func newFlowEpochEpochCommittedEventType() *cadence.EventType {

	// A.01cf0e2f2f715450.FlowEpoch.EpochCommitted

	address, _ := common.HexToAddress("01cf0e2f2f715450")
	location := common.NewAddressLocation(nil, address, "FlowEpoch")

	return cadence.NewEventType(
		location,
		"FlowEpoch.EpochCommitted",
		[]cadence.Field{
			{
				Identifier: "counter",
				Type:       cadence.UInt64Type,
			},
			{
				Identifier: "clusterQCs",
				Type:       cadence.NewVariableSizedArrayType(newFlowClusterQCClusterQCStructType()),
			},
			{
				Identifier: "dkgPubKeys",
				Type:       cadence.NewVariableSizedArrayType(cadence.StringType),
			},
		},
		nil,
	)
}

func newFlowClusterQCClusterQCStructType() *cadence.StructType {

	// A.01cf0e2f2f715450.FlowClusterQC.ClusterQC"

	address, _ := common.HexToAddress("01cf0e2f2f715450")
	location := common.NewAddressLocation(nil, address, "FlowClusterQC")

	return cadence.NewStructType(
		location,
		"FlowClusterQC.ClusterQC",
		[]cadence.Field{
			{
				Identifier: "index",
				Type:       cadence.UInt16Type,
			},
			{
				Identifier: "voteSignatures",
				Type:       cadence.NewVariableSizedArrayType(cadence.StringType),
			},
			{
				Identifier: "voteMessage",
				Type:       cadence.StringType,
			},
			{
				Identifier: "voterIDs",
				Type:       cadence.NewVariableSizedArrayType(cadence.StringType),
			},
		},
		nil,
	)
}

func newNodeVersionBeaconVersionBeaconEventType() *cadence.EventType {

	// A.01cf0e2f2f715450.NodeVersionBeacon.VersionBeacon

	address, _ := common.HexToAddress("01cf0e2f2f715450")
	location := common.NewAddressLocation(nil, address, "NodeVersionBeacon")

	return cadence.NewEventType(
		location,
		"NodeVersionBeacon.VersionBeacon",
		[]cadence.Field{
			{
				Identifier: "versionBoundaries",
				Type:       cadence.NewVariableSizedArrayType(newNodeVersionBeaconVersionBoundaryStructType()),
			},
			{
				Identifier: "sequence",
				Type:       cadence.UInt64Type,
			},
		},
		nil,
	)
}

func newNodeVersionBeaconVersionBoundaryStructType() *cadence.StructType {

	// A.01cf0e2f2f715450.NodeVersionBeacon.VersionBoundary

	address, _ := common.HexToAddress("01cf0e2f2f715450")
	location := common.NewAddressLocation(nil, address, "NodeVersionBeacon")

	return cadence.NewStructType(
		location,
		"NodeVersionBeacon.VersionBoundary",
		[]cadence.Field{
			{
				Identifier: "blockHeight",
				Type:       cadence.UInt64Type,
			},
			{
				Identifier: "version",
				Type:       newNodeVersionBeaconSemverStructType(),
			},
		},
		nil,
	)
}

func newNodeVersionBeaconSemverStructType() *cadence.StructType {

	// A.01cf0e2f2f715450.NodeVersionBeacon.Semver

	address, _ := common.HexToAddress("01cf0e2f2f715450")
	location := common.NewAddressLocation(nil, address, "NodeVersionBeacon")

	return cadence.NewStructType(
		location,
		"NodeVersionBeacon.Semver",
		[]cadence.Field{
			{
				Identifier: "preRelease",
				Type:       cadence.NewOptionalType(cadence.StringType),
			},
			{
				Identifier: "major",
				Type:       cadence.UInt8Type,
			},
			{
				Identifier: "minor",
				Type:       cadence.UInt8Type,
			},
			{
				Identifier: "patch",
				Type:       cadence.UInt8Type,
			},
		},
		nil,
	)
}

func ufix64FromString(s string) cadence.UFix64 {
	f, err := cadence.NewUFix64(s)
	if err != nil {
		panic(err)
	}
	return f
}
