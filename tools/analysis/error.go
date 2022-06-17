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

package analysis

import (
	"golang.org/x/xerrors"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
)

type ParsingCheckingError struct {
	error
	location common.Location
}

var _ error = ParsingCheckingError{}
var _ xerrors.Wrapper = ParsingCheckingError{}
var _ common.HasLocation = ParsingCheckingError{}
var _ errors.ParentError = ParsingCheckingError{}

func (e ParsingCheckingError) Unwrap() error {
	return e.error
}

func (e ParsingCheckingError) ImportLocation() common.Location {
	return e.location
}

func (e ParsingCheckingError) ChildErrors() []error {
	return []error{e.error}
}
