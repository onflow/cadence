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

package parser

// NOTE: ensure to update allKeywords when adding a new keyword
const (
	keywordIf          = "if"
	keywordElse        = "else"
	keywordWhile       = "while"
	keywordBreak       = "break"
	keywordContinue    = "continue"
	keywordReturn      = "return"
	keywordTrue        = "true"
	keywordFalse       = "false"
	keywordNil         = "nil"
	keywordLet         = "let"
	keywordVar         = "var"
	keywordFun         = "fun"
	keywordAs          = "as"
	keywordCreate      = "create"
	keywordDestroy     = "destroy"
	keywordFor         = "for"
	keywordIn          = "in"
	keywordEmit        = "emit"
	keywordAuth        = "auth"
	keywordPriv        = "priv"
	keywordPub         = "pub"
	keywordAccess      = "access"
	keywordSet         = "set"
	keywordAll         = "all"
	keywordSelf        = "self"
	keywordInit        = "init"
	keywordContract    = "contract"
	keywordAccount     = "account"
	keywordImport      = "import"
	keywordFrom        = "from"
	keywordPre         = "pre"
	keywordPost        = "post"
	keywordEvent       = "event"
	keywordStruct      = "struct"
	keywordResource    = "resource"
	keywordInterface   = "interface"
	keywordTransaction = "transaction"
	keywordPrepare     = "prepare"
	keywordExecute     = "execute"
	keywordCase        = "case"
	keywordSwitch      = "switch"
	keywordDefault     = "default"
	keywordEnum        = "enum"
	keywordView        = "view"
	// NOTE: ensure to update allKeywords when adding a new keyword
)

var allKeywords = map[string]struct{}{
	keywordIf:          {},
	keywordElse:        {},
	keywordWhile:       {},
	keywordBreak:       {},
	keywordContinue:    {},
	keywordReturn:      {},
	keywordTrue:        {},
	keywordFalse:       {},
	keywordNil:         {},
	keywordLet:         {},
	keywordVar:         {},
	keywordFun:         {},
	keywordAs:          {},
	keywordCreate:      {},
	keywordDestroy:     {},
	keywordFor:         {},
	keywordIn:          {},
	keywordEmit:        {},
	keywordAuth:        {},
	keywordPriv:        {},
	keywordPub:         {},
	keywordAccess:      {},
	keywordSet:         {},
	keywordAll:         {},
	keywordSelf:        {},
	keywordInit:        {},
	keywordContract:    {},
	keywordAccount:     {},
	keywordImport:      {},
	keywordFrom:        {},
	keywordPre:         {},
	keywordPost:        {},
	keywordEvent:       {},
	keywordStruct:      {},
	keywordResource:    {},
	keywordInterface:   {},
	keywordTransaction: {},
	keywordPrepare:     {},
	keywordExecute:     {},
	keywordCase:        {},
	keywordSwitch:      {},
	keywordDefault:     {},
	keywordEnum:        {},
	keywordView:        {},
}

// Keywords that can be used in identifier position without ambiguity.
var softKeywords = map[string]struct{}{
	keywordFrom:    {},
	keywordAccount: {},
	keywordSet:     {},
	keywordAll:     {},
}

// Keywords that aren't allowed in identifier position.
var hardKeywords = mapDiff(allKeywords, softKeywords)

// take the boolean difference of two maps
func mapDiff[T comparable, U any](minuend map[T]U, subtrahend map[T]U) map[T]U {
	diff := make(map[T]U, len(minuend))
	// iteration order is not important here
	for k, v := range minuend { // nolint:maprangecheck
		if _, exists := subtrahend[k]; !exists {
			diff[k] = v
		}
	}
	return diff
}
