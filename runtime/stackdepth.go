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

const defaultStackDepthLimit = 2000

type stackDepthLimiter struct {
	depth uint64
	limit uint64
}

func newStackDepthLimiter(stackDepthLimit uint64) *stackDepthLimiter {
	if stackDepthLimit == 0 {
		stackDepthLimit = defaultStackDepthLimit
	}
	return &stackDepthLimiter{
		limit: stackDepthLimit,
	}
}

func (limiter *stackDepthLimiter) OnFunctionInvocation() {
	limiter.depth++

	if limiter.depth <= limiter.limit {
		return
	}

	panic(CallStackLimitExceededError{
		Limit: limiter.limit,
	})
}

func (limiter *stackDepthLimiter) OnInvokedFunctionReturn() {
	limiter.depth--
}
