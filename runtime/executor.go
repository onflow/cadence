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

import "github.com/onflow/cadence"

// Executor is a continuation which represents a full unit of transaction/script
// execution.
//
// The full unit of execution is divided into stages:
//  1. Preprocess() initializes the executor in preparation for the actual
//     transaction execution (e.g., parse / type check the input).  Note that
//     the work done by Preprocess() should be embrassingly parallel.
//  2. Execute() performs the actual transaction execution (e.g., run the
//     interpreter to produce the transaction result).
//  3. Result() returns the result of the full unit of execution.
//
// TODO: maybe add Cleanup/Postprocess in the future
type Executor interface {
	// Preprocess prepares the transaction/script for execution.
	//
	// This function returns an error if the program has errors (e.g., syntax
	// errors, type errors).
	//
	// This method may be called multiple times.  Only the first call will
	// trigger meaningful work; subsequent calls will return the cached return
	// value from the original call (i.e., an Executor implementation must
	// guard this method with sync.Once).
	Preprocess() error

	// Execute executes the transaction/script.
	//
	// This function returns an error if Preprocess failed or if the execution
	// fails.
	//
	// This method may be called multiple times.  Only the first call will
	// trigger meaningful work; subsequent calls will return the cached return
	// value from the original call (i.e., an Executor implementation must
	// guard this method with sync.Once).
	//
	// Note: Execute will invoke Preprocess to ensure Preprocess was called at
	// least once.
	Execute() error

	// Result returns the transaction/scipt's execution result.
	//
	// This function returns an error if Preproces or Execute fails.  The
	// cadence.Value is always nil for transaction.
	//
	// This method may be called multiple times.  Only the first call will
	// trigger meaningful work; subsequent calls will return the cached return
	// value from the original call (i.e., an Executor implementation must
	// guard this method with sync.Once).
	//
	// Note: Result will invoke Execute to ensure Execute was called at least
	// once.
	Result() (cadence.Value, error)
}
