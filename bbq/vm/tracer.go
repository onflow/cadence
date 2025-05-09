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

package vm

import (
	"time"

	"github.com/onflow/cadence/errors"
)

// Tracer for VM. Currently disabled.
// TODO: Refactor and re-use the tracing from the interpreter.
type Tracer struct{}

func (t Tracer) TracingEnabled() bool {
	return false
}

func (t Tracer) ReportArrayValueDeepRemoveTrace(_ string, _ int, _ time.Duration) {
	panic(errors.NewUnreachableError())
}

func (t Tracer) ReportArrayValueTransferTrace(_ string, _ int, _ time.Duration) {
	panic(errors.NewUnreachableError())
}

func (t Tracer) ReportArrayValueConstructTrace(_ string, _ int, _ time.Duration) {
	panic(errors.NewUnreachableError())
}

func (c *Context) ReportArrayValueDestroyTrace(_ string, _ int, _ time.Duration) {
	panic(errors.NewUnreachableError())
}

func (c *Context) ReportArrayValueConformsToStaticTypeTrace(_ string, _ int, _ time.Duration) {
	panic(errors.NewUnreachableError())
}

func (t Tracer) ReportDictionaryValueTransferTrace(_ string, _ int, _ time.Duration) {
	panic(errors.NewUnreachableError())
}

func (t Tracer) ReportDictionaryValueDeepRemoveTrace(_ string, _ int, _ time.Duration) {
	panic(errors.NewUnreachableError())
}

func (t Tracer) ReportCompositeValueDeepRemoveTrace(_ string, _ string, _ string, _ time.Duration) {
	panic(errors.NewUnreachableError())
}

func (t Tracer) ReportDictionaryValueGetMemberTrace(_ string, _ int, _ string, _ time.Duration) {
	panic(errors.NewUnreachableError())
}

func (c *Context) ReportDictionaryValueConstructTrace(_ string, _ int, _ time.Duration) {
	panic(errors.NewUnreachableError())
}

func (c *Context) ReportDictionaryValueDestroyTrace(_ string, _ int, _ time.Duration) {
	panic(errors.NewUnreachableError())
}

func (c *Context) ReportDictionaryValueConformsToStaticTypeTrace(_ string, _ int, _ time.Duration) {
	panic(errors.NewUnreachableError())
}

func (t Tracer) ReportCompositeValueTransferTrace(_ string, _ string, _ string, _ time.Duration) {
	panic(errors.NewUnreachableError())
}

func (t Tracer) ReportCompositeValueSetMemberTrace(_ string, _ string, _ string, _ string, _ time.Duration) {
	panic(errors.NewUnreachableError())
}

func (t Tracer) ReportCompositeValueGetMemberTrace(_ string, _ string, _ string, _ string, _ time.Duration) {
	panic(errors.NewUnreachableError())
}

func (t Tracer) ReportCompositeValueConstructTrace(_ string, _ string, _ string, _ time.Duration) {
	panic(errors.NewUnreachableError())
}

func (t Tracer) ReportCompositeValueDestroyTrace(_ string, _ string, _ string, _ time.Duration) {
	panic(errors.NewUnreachableError())
}

func (t Tracer) ReportCompositeValueConformsToStaticTypeTrace(_ string, _ string, _ string, _ time.Duration) {
	panic(errors.NewUnreachableError())
}

func (t Tracer) ReportCompositeValueRemoveMemberTrace(_ string, _ string, _ string, _ string, _ time.Duration) {
	panic(errors.NewUnreachableError())
}

func (t Tracer) ReportDomainStorageMapDeepRemoveTrace(_ string, _ int, _ time.Duration) {
	panic(errors.NewUnreachableError())
}
