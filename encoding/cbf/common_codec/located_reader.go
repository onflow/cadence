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

// LocatedReader records how many bytes were read from its wrapped io.Reader.
// Includes the bytes written when an error is returned by the io.Reader, if non-zero.
type LocatedReader struct {
	r        io.Reader
	location int
}

func NewLocatedReader(r io.Reader) LocatedReader {
	return LocatedReader{r: r}
}

func (l *LocatedReader) Read(p []byte) (n int, err error) {
	n, err = l.r.Read(p)
	l.location += n
	return
}

func (l *LocatedReader) Location() int {
	return l.location
}
