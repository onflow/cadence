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

func (ctx Tracer) TracingEnabled() bool {
	return false
}

func (ctx Tracer) ReportArrayValueDeepRemoveTrace(_ string, _ int, _ time.Duration) {
	panic(errors.NewUnreachableError())
}

func (ctx Tracer) ReportArrayValueTransferTrace(_ string, _ int, _ time.Duration) {
	panic(errors.NewUnreachableError())
}

func (ctx Tracer) ReportArrayValueConstructTrace(_ string, _ int, _ time.Duration) {
	panic(errors.NewUnreachableError())
}

func (c *Config) ReportArrayValueDestroyTrace(_ string, _ int, _ time.Duration) {
	panic(errors.NewUnreachableError())
}

func (c *Config) ReportArrayValueConformsToStaticTypeTrace(_ string, _ int, _ time.Duration) {
	panic(errors.NewUnreachableError())
}

func (ctx Tracer) ReportDictionaryValueTransferTrace(_ string, _ int, _ time.Duration) {
	panic(errors.NewUnreachableError())
}

func (ctx Tracer) ReportDictionaryValueDeepRemoveTrace(_ string, _ int, _ time.Duration) {
	panic(errors.NewUnreachableError())
}

func (ctx Tracer) ReportCompositeValueDeepRemoveTrace(_ string, _ string, _ string, _ time.Duration) {
	panic(errors.NewUnreachableError())
}

func (ctx Tracer) ReportDictionaryValueGetMemberTrace(_ string, _ int, _ string, _ time.Duration) {
	panic(errors.NewUnreachableError())
}

func (c *Config) ReportDictionaryValueConstructTrace(_ string, _ int, _ time.Duration) {
	panic(errors.NewUnreachableError())
}

func (c *Config) ReportDictionaryValueDestroyTrace(_ string, _ int, _ time.Duration) {
	panic(errors.NewUnreachableError())
}

func (c *Config) ReportDictionaryValueConformsToStaticTypeTrace(_ string, _ int, _ time.Duration) {
	panic(errors.NewUnreachableError())
}

func (ctx Tracer) ReportCompositeValueTransferTrace(_ string, _ string, _ string, _ time.Duration) {
	panic(errors.NewUnreachableError())
}

func (ctx Tracer) ReportCompositeValueSetMemberTrace(_ string, _ string, _ string, _ string, _ time.Duration) {
	panic(errors.NewUnreachableError())
}

func (ctx Tracer) ReportCompositeValueGetMemberTrace(_ string, _ string, _ string, _ string, _ time.Duration) {
	panic(errors.NewUnreachableError())
}

func (ctx Tracer) ReportCompositeValueConstructTrace(_ string, _ string, _ string, _ time.Duration) {
	panic(errors.NewUnreachableError())
}

func (c *Config) ReportCompositeValueDestroyTrace(_ string, _ string, _ string, _ time.Duration) {
	panic(errors.NewUnreachableError())
}

func (c *Config) ReportCompositeValueConformsToStaticTypeTrace(_ string, _ string, _ string, _ time.Duration) {
	panic(errors.NewUnreachableError())
}

func (c *Config) ReportCompositeValueRemoveMemberTrace(_ string, _ string, _ string, _ string, _ time.Duration) {
	panic(errors.NewUnreachableError())
}

func (ctx Tracer) ReportDomainStorageMapDeepRemoveTrace(_ string, _ int, _ time.Duration) {
	panic(errors.NewUnreachableError())
}
