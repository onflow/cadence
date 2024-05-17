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
	"github.com/onflow/atree"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/format"
	"github.com/onflow/cadence/runtime/interpreter"
)

type LinkValue struct {
	TargetPath PathValue
	Type       StaticType
}

var _ Value = LinkValue{}

func NewLinkValue(targetPath PathValue, staticType StaticType) LinkValue {
	return LinkValue{
		TargetPath: targetPath,
		Type:       staticType,
	}
}

func (LinkValue) isValue() {}

func (v LinkValue) StaticType(gauge common.MemoryGauge) StaticType {
	return interpreter.NewCapabilityStaticType(gauge, v.Type)
}

func (v LinkValue) Transfer(*Config, atree.Address, bool, atree.Storable) Value {
	return v
}

func (v LinkValue) String() string {
	return format.Link(
		v.Type.String(),
		v.TargetPath.String(),
	)
}
