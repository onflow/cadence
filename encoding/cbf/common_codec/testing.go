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

package common_codec

import "io"

// MockWriter is for testing.
type MockWriter struct {
	ByteToErrorOn int
	ErrorToReturn error
	CurrentByte   int
}

var _ io.Writer = &MockWriter{}

func (m *MockWriter) Write(p []byte) (n int, err error) {
	currentByte := m.CurrentByte
	m.CurrentByte += len(p)

	if m.ByteToErrorOn < 0 || // erroring disabled
		m.ErrorToReturn == nil || // no erroring
		currentByte > m.ByteToErrorOn || // already errored
		m.CurrentByte <= m.ByteToErrorOn { // not yet erroring
		return len(p), nil
	}

	return 0, m.ErrorToReturn
}

// Flatten turns multiple arrays into one array.
func Flatten[T any](deep ...[]T) []T {
	length := 0
	for _, b := range deep {
		length += len(b)
	}

	flat := make([]T, 0, length)
	for _, b := range deep {
		flat = append(flat, b...)
	}

	return flat
}
