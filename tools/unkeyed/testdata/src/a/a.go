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

package a

import (
	"flag"
	"go/scanner"
	"go/token"
	"sync"
	"unicode"
)

var StringSlice = []string{
	"Name",
	"Usage",
	"DefValue",
}

var Map = map[string]bool{
	"Name":     true,
	"Usage":    true,
	"DefValue": true,
}

var LocalInlineStructBad = struct { // want "unkeyed"
	X string
	Y string
	Z string
}{
	"Name",
	"Usage",
	"DefValue",
}

var LocalInlineStructSlice = []struct {
	A int
	B int
}{
	{1, 2}, // want "unkeyed fields"
	{3, 4}, // want "unkeyed fields"
}

type MyStruct struct {
	X string
	Y string
	Z string
}

var LocalStructOK = MyStruct{
	X: "Name",
	Y: "Usage",
	Z: "DefValue",
}

var LocalStructBad = MyStruct{ // want "unkeyed fields"
	"Name",
	"Usage",
	"DefValue",
}

var LocalStructRefOK = &MyStruct{
	X: "Name",
	Y: "Usage",
	Z: "DefValue",
}

var LocalStructRefBad = &MyStruct{ // want "unkeyed fields"
	"Name",
	"Usage",
	"DefValue",
}

var LocalStructSlice = []MyStruct{
	{X: "foo", Y: "bar", Z: "baz"},
	{X: "aa", Y: "bb", Z: "cc"},
}

var LocalStructSliceBad = []MyStruct{
	{"foo", "bar", "baz"}, // want "unkeyed fields"
	{"aa", "bb", "cc"},    // want "unkeyed fields"
}

var LocalStructPointerSliceOK = []*MyStruct{
	{X: "foo", Y: "bar", Z: "baz"},
	{X: "aa", Y: "bb", Z: "cc"},
}

var LocalStructPointerSliceBad = []*MyStruct{
	{"foo", "bar", "baz"}, // want "unkeyed fields"
	{"aa", "bb", "cc"},    // want "unkeyed fields"
}

var ImportedStructOK = flag.Flag{
	Name:  "Name",
	Usage: "Usage",
}

var ImportedStructBad = flag.Flag{ // want "unkeyed fields"
	"Name",
	"Usage",
	nil, // Value
	"DefValue",
}

var delta [3]rune

// SpecialCase is a named slice of CaseRange to test issue 9171.
var goodNamedSliceLiteral = unicode.SpecialCase{
	{Lo: 1, Hi: 2, Delta: delta},
	unicode.CaseRange{Lo: 1, Hi: 2, Delta: delta},
}
var badNamedSliceLiteral = unicode.SpecialCase{
	{1, 2, delta},                  // want "unkeyed fields"
	unicode.CaseRange{1, 2, delta}, // want "unkeyed fields"
}

// ErrorList is a named slice, so no warnings should be emitted.
var goodScannerErrorList = scanner.ErrorList{
	&scanner.Error{Msg: "foobar"},
}
var badScannerErrorList = scanner.ErrorList{
	&scanner.Error{token.Position{}, "foobar"}, // want "unkeyed fields"
}

// sync.Mutex has unexported fields. We expect a diagnostic but no
// suggested fix.
var mu = sync.Mutex{0, 0} // want "unkeyed fields"

// A named pointer slice of CaseRange to test issue 23539. In
// particular, we're interested in how some slice elements omit their
// type.
var goodNamedPointerSliceLiteral = []*unicode.CaseRange{
	{Lo: 1, Hi: 2},
	&unicode.CaseRange{Lo: 1, Hi: 2},
}
var badNamedPointerSliceLiteral = []*unicode.CaseRange{
	{1, 2, delta},                   // want "unkeyed fields"
	&unicode.CaseRange{1, 2, delta}, // want "unkeyed fields"
}
