//go:build wasmtime
// +build wasmtime

/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Dapper Labs, Inc.
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

package vm

import (
	"fmt"
	"math/big"

	"C"

	"github.com/bytecodealliance/wasmtime-go"

	"github.com/onflow/cadence/runtime/interpreter"
)

type VM interface {
	Invoke(name string, arguments ...interpreter.Value) (interpreter.Value, error)
}

type vm struct {
	instance *wasmtime.Instance
	store    *wasmtime.Store
}

func (m *vm) Invoke(name string, arguments ...interpreter.Value) (interpreter.Value, error) {
	f := m.instance.GetExport(m.store, name).Func()

	rawArguments := make([]any, len(arguments))
	for i, argument := range arguments {
		rawArguments[i] = argument
	}

	res, err := f.Call(m.store, rawArguments...)
	if err != nil {
		return nil, err
	}

	if res == nil {
		return nil, nil
	}

	return res.(interpreter.Value), nil
}

func NewVM(wasm []byte) (VM, error) {

	inter, err := interpreter.NewInterpreter(nil, nil, &interpreter.Config{})
	if err != nil {
		return nil, err
	}

	config := wasmtime.NewConfig()
	config.SetWasmReferenceTypes(true)

	engine := wasmtime.NewEngineWithConfig(config)

	store := wasmtime.NewStore(engine)

	module, err := wasmtime.NewModule(store.Engine, wasm)
	if err != nil {
		return nil, err
	}

	intFunc := wasmtime.WrapFunc(
		store,
		func(caller *wasmtime.Caller, offset int32, length int32) (any, *wasmtime.Trap) {
			if offset < 0 {
				return nil, wasmtime.NewTrap(fmt.Sprintf("Int: invalid offset: %d", offset))
			}

			if length < 2 {
				return nil, wasmtime.NewTrap(fmt.Sprintf("Int: invalid length: %d", length))
			}

			mem := caller.GetExport("mem").Memory()

			bytes := C.GoBytes(mem.Data(store), C.int(length))

			value := new(big.Int).SetBytes(bytes[1:])
			if bytes[0] == 0 {
				value = value.Neg(value)
			}

			return interpreter.NewUnmeteredIntValueFromBigInt(value), nil
		},
	)

	stringFunc := wasmtime.WrapFunc(
		store,
		func(caller *wasmtime.Caller, offset int32, length int32) (any, *wasmtime.Trap) {
			if offset < 0 {
				return nil, wasmtime.NewTrap(fmt.Sprintf("String: invalid offset: %d", offset))
			}

			if length < 0 {
				return nil, wasmtime.NewTrap(fmt.Sprintf("String: invalid length: %d", length))
			}

			mem := caller.GetExport("mem").Memory()

			bytes := C.GoBytes(mem.Data(store), C.int(length))

			return interpreter.NewUnmeteredStringValue(string(bytes)), nil
		},
	)

	addFunc := wasmtime.WrapFunc(
		store,
		func(left, right any) (any, *wasmtime.Trap) {
			leftNumber, ok := left.(interpreter.NumberValue)
			if !ok {
				return nil, wasmtime.NewTrap(fmt.Sprintf("add: invalid left: %#+v", left))
			}

			rightNumber, ok := right.(interpreter.NumberValue)
			if !ok {
				return nil, wasmtime.NewTrap(fmt.Sprintf("add: invalid right: %#+v", right))
			}

			return leftNumber.Plus(inter, rightNumber, interpreter.EmptyLocationRange), nil
		},
	)

	// NOTE: wasmtime currently does not support specifying imports by name,
	// unlike other WebAssembly APIs like wasmer, JavaScript, etc.,
	// i.e. imports are imported in the order they are given.

	instance, err := wasmtime.NewInstance(
		store,
		module,
		[]wasmtime.AsExtern{
			intFunc,
			stringFunc,
			addFunc,
		},
	)
	if err != nil {
		return nil, err
	}

	return &vm{
		instance: instance,
		store:    store,
	}, nil
}
