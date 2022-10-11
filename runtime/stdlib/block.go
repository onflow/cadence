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

package stdlib

import (
	"time"
	"unsafe"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
)

const getCurrentBlockFunctionDocString = `
Returns the current block, i.e. the block which contains the currently executed transaction
`

var getCurrentBlockFunctionType = &sema.FunctionType{
	ReturnTypeAnnotation: sema.NewTypeAnnotation(
		sema.BlockType,
	),
}

const getBlockFunctionDocString = `
Returns the block at the given height. If the given block does not exist the function returns nil
`

var getBlockFunctionType = &sema.FunctionType{
	Purity: sema.FunctionPurityView,
	Parameters: []*sema.Parameter{
		{
			Label:      "at",
			Identifier: "height",
			TypeAnnotation: sema.NewTypeAnnotation(
				sema.UInt64Type,
			),
		},
	},
	ReturnTypeAnnotation: sema.NewTypeAnnotation(
		&sema.OptionalType{
			Type: sema.BlockType,
		},
	),
}

const BlockHashLength = 32

type BlockHash [BlockHashLength]byte

type Block struct {
	Height    uint64
	View      uint64
	Hash      BlockHash
	Timestamp int64
}

type BlockAtHeightProvider interface {
	// GetBlockAtHeight returns the block at the given height.
	GetBlockAtHeight(height uint64) (block Block, exists bool, err error)
}

func NewGetBlockFunction(provider BlockAtHeightProvider) StandardLibraryValue {
	return NewStandardLibraryFunction(
		"getBlock",
		getBlockFunctionType,
		getBlockFunctionDocString,
		func(invocation interpreter.Invocation) interpreter.Value {
			heightValue, ok := invocation.Arguments[0].(interpreter.UInt64Value)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			memoryGauge := invocation.Interpreter
			locationRange := invocation.LocationRange

			block, exists := getBlockAtHeight(
				provider,
				uint64(heightValue),
			)
			if !exists {
				return interpreter.Nil
			}

			blockValue := NewBlockValue(memoryGauge, locationRange, block)
			return interpreter.NewSomeValueNonCopying(memoryGauge, blockValue)
		},
	)
}

var BlockIDStaticType = interpreter.ConstantSizedStaticType{
	Type: interpreter.PrimitiveStaticTypeUInt8, // unmetered
	Size: 32,
}

var blockIDMemoryUsage = common.NewNumberMemoryUsage(
	8 * int(unsafe.Sizeof(interpreter.UInt8Value(0))),
)

func NewBlockValue(
	inter *interpreter.Interpreter,
	locationRange interpreter.LocationRange,
	block Block,
) interpreter.Value {

	// height
	heightValue := interpreter.NewUInt64Value(
		inter,
		func() uint64 {
			return block.Height
		},
	)

	// view
	viewValue := interpreter.NewUInt64Value(
		inter,
		func() uint64 {
			return block.View
		},
	)

	// ID
	common.UseMemory(inter, blockIDMemoryUsage)
	var values = make([]interpreter.Value, sema.BlockIDSize)
	for i, b := range block.Hash {
		values[i] = interpreter.NewUnmeteredUInt8Value(b)
	}

	idValue := interpreter.NewArrayValue(
		inter,
		locationRange,
		BlockIDStaticType,
		common.Address{},
		values...,
	)

	// timestamp
	// TODO: verify
	timestampValue := interpreter.NewUFix64ValueWithInteger(
		inter,
		func() uint64 {
			return uint64(time.Unix(0, block.Timestamp).Unix())
		},
	)

	return interpreter.NewBlockValue(
		inter,
		heightValue,
		viewValue,
		idValue,
		timestampValue,
	)
}

func getBlockAtHeight(
	provider BlockAtHeightProvider,
	height uint64,
) (
	block Block,
	exists bool,
) {
	var err error
	wrapPanic(func() {
		block, exists, err = provider.GetBlockAtHeight(height)
	})
	if err != nil {
		panic(err)
	}

	return block, exists
}

type CurrentBlockProvider interface {
	BlockAtHeightProvider
	// GetCurrentBlockHeight returns the current block height.
	GetCurrentBlockHeight() (uint64, error)
}

func NewGetCurrentBlockFunction(provider CurrentBlockProvider) StandardLibraryValue {
	return NewStandardLibraryFunction(
		"getCurrentBlock",
		getCurrentBlockFunctionType,
		getCurrentBlockFunctionDocString,
		func(invocation interpreter.Invocation) interpreter.Value {

			var height uint64
			var err error
			wrapPanic(func() {
				height, err = provider.GetCurrentBlockHeight()
			})
			if err != nil {
				panic(err)
			}

			block, exists := getBlockAtHeight(
				provider,
				height,
			)
			if !exists {
				panic(errors.NewUnexpectedError("cannot get current block"))
			}

			memoryGauge := invocation.Interpreter
			locationRange := invocation.LocationRange

			return NewBlockValue(memoryGauge, locationRange, block)
		},
	)
}
