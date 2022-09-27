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

// LengthyWriter records how many bytes were written to its wrapped io.Writer.
// Includes the bytes written when an error is returned by the io.Writer, if non-zero.
type LengthyWriter struct {
	w      io.Writer
	length int
}

func NewLengthyWriter(w io.Writer) LengthyWriter {
	return LengthyWriter{w: w}
}

func (l *LengthyWriter) Write(p []byte) (n int, err error) {
	n, err = l.w.Write(p)
	l.length += n
	return
}

func (l *LengthyWriter) Len() int {
	return l.length
}
