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

package stdlib

import (
	"time"
	"unsafe"

	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
)

const getCurrentBlockFunctionName = "getCurrentBlock"

const getCurrentBlockFunctionDocString = `
Returns the current block, i.e. the block which contains the currently executed transaction
`

var getCurrentBlockFunctionType = sema.NewSimpleFunctionType(
	sema.FunctionPurityView,
	nil,
	sema.BlockTypeAnnotation,
)

const getBlockFunctionName = "getBlock"

const getBlockFunctionDocString = `
Returns the block at the given height. If the given block does not exist the function returns nil
`

var getBlockFunctionType = sema.NewSimpleFunctionType(
	sema.FunctionPurityView,
	[]sema.Parameter{
		{
			Label:          "at",
			Identifier:     "height",
			TypeAnnotation: sema.UInt64TypeAnnotation,
		},
	},
	sema.NewTypeAnnotation(
		&sema.OptionalType{
			Type: sema.BlockType,
		},
	),
)

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

func NativeGetBlockFunction(provider BlockAtHeightProvider) interpreter.NativeFunction {
	return func(
		context interpreter.NativeFunctionContext,
		_ interpreter.LocationRange,
		_ interpreter.TypeParameterGetter,
		_ interpreter.Value,
		args ...interpreter.Value,
	) interpreter.Value {
		heightValue := interpreter.AssertValueOfType[interpreter.UInt64Value](args[0])

		block, exists := getBlockAtHeight(provider, uint64(heightValue))
		if !exists {
			return interpreter.Nil
		}

		blockValue := NewBlockValue(context, block)
		return interpreter.NewSomeValueNonCopying(context, blockValue)
	}
}

func NewInterpreterGetBlockFunction(provider BlockAtHeightProvider) StandardLibraryValue {
	return NewNativeStandardLibraryStaticFunction(
		getBlockFunctionName,
		getBlockFunctionType,
		getBlockFunctionDocString,
		NativeGetBlockFunction(provider),
		false,
	)
}

func NewVMGetBlockFunction(provider BlockAtHeightProvider) StandardLibraryValue {
	return NewNativeStandardLibraryStaticFunction(
		getBlockFunctionName,
		getBlockFunctionType,
		getBlockFunctionDocString,
		NativeGetBlockFunction(provider),
		true,
	)
}

var BlockIDStaticType = &interpreter.ConstantSizedStaticType{
	Type: interpreter.PrimitiveStaticTypeUInt8, // unmetered
	Size: 32,
}

var blockIDMemoryUsage = common.NewNumberMemoryUsage(
	8 * int(unsafe.Sizeof(interpreter.UInt8Value(0))),
)

func NewBlockValue(
	context interpreter.ArrayCreationContext,
	block Block,
) interpreter.Value {

	// height
	heightValue := interpreter.NewUInt64Value(
		context,
		func() uint64 {
			return block.Height
		},
	)

	// view
	viewValue := interpreter.NewUInt64Value(
		context,
		func() uint64 {
			return block.View
		},
	)

	// ID
	common.UseMemory(context, blockIDMemoryUsage)
	var values = make([]interpreter.Value, sema.BlockTypeIdFieldType.Size)
	for i, b := range block.Hash {
		values[i] = interpreter.NewUnmeteredUInt8Value(b)
	}

	idValue := interpreter.NewArrayValue(
		context,
		BlockIDStaticType,
		common.ZeroAddress,
		values...,
	)

	// timestamp
	// TODO: verify
	timestampValue := interpreter.NewUFix64ValueWithInteger(
		context,
		func() uint64 {
			return uint64(time.Unix(0, block.Timestamp).Unix())
		},
	)

	return interpreter.NewBlockValue(
		context,
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
	block, exists, err = provider.GetBlockAtHeight(height)
	if err != nil {
		panic(err)
	}
	return
}

type CurrentBlockProvider interface {
	BlockAtHeightProvider
	// GetCurrentBlockHeight returns the current block height.
	GetCurrentBlockHeight() (uint64, error)
}

func NativeGetCurrentBlockFunction(provider CurrentBlockProvider) interpreter.NativeFunction {
	return func(
		context interpreter.NativeFunctionContext,
		locationRange interpreter.LocationRange,
		_ interpreter.TypeParameterGetter,
		_ interpreter.Value,
		_ ...interpreter.Value,
	) interpreter.Value {
		height, err := provider.GetCurrentBlockHeight()
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

		return NewBlockValue(context, block)
	}
}

func NewInterpreterGetCurrentBlockFunction(provider CurrentBlockProvider) StandardLibraryValue {
	return NewNativeStandardLibraryStaticFunction(
		getCurrentBlockFunctionName,
		getCurrentBlockFunctionType,
		getCurrentBlockFunctionDocString,
		NativeGetCurrentBlockFunction(provider),
		false,
	)
}

func NewVMGetCurrentBlockFunction(provider CurrentBlockProvider) StandardLibraryValue {
	return NewNativeStandardLibraryStaticFunction(
		getCurrentBlockFunctionName,
		getCurrentBlockFunctionType,
		getCurrentBlockFunctionDocString,
		NativeGetCurrentBlockFunction(provider),
		true,
	)
}
