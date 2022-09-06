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

const (
	KeywordIf          = "if"
	KeywordElse        = "else"
	KeywordWhile       = "while"
	KeywordBreak       = "break"
	KeywordContinue    = "continue"
	KeywordReturn      = "return"
	KeywordTrue        = "true"
	KeywordFalse       = "false"
	KeywordNil         = "nil"
	KeywordLet         = "let"
	KeywordVar         = "var"
	KeywordFun         = "fun"
	KeywordAs          = "as"
	KeywordCreate      = "create"
	KeywordDestroy     = "destroy"
	KeywordFor         = "for"
	KeywordIn          = "in"
	KeywordEmit        = "emit"
	KeywordAuth        = "auth"
	KeywordPriv        = "priv"
	KeywordPub         = "pub"
	KeywordAccess      = "access"
	KeywordSet         = "set"
	KeywordAll         = "all"
	KeywordSelf        = "self"
	KeywordInit        = "init"
	KeywordContract    = "contract"
	KeywordAccount     = "account"
	KeywordImport      = "import"
	KeywordFrom        = "from"
	KeywordPre         = "pre"
	KeywordPost        = "post"
	KeywordEvent       = "event"
	KeywordStruct      = "struct"
	KeywordResource    = "resource"
	KeywordInterface   = "interface"
	KeywordTransaction = "transaction"
	KeywordPrepare     = "prepare"
	KeywordExecute     = "execute"
	KeywordCase        = "case"
	KeywordSwitch      = "switch"
	KeywordDefault     = "default"
	KeywordEnum        = "enum"
)

var AllKeywords = map[string]struct{}{
	KeywordIf:          {},
	KeywordElse:        {},
	KeywordWhile:       {},
	KeywordBreak:       {},
	KeywordContinue:    {},
	KeywordReturn:      {},
	KeywordTrue:        {},
	KeywordFalse:       {},
	KeywordNil:         {},
	KeywordLet:         {},
	KeywordVar:         {},
	KeywordFun:         {},
	KeywordAs:          {},
	KeywordCreate:      {},
	KeywordDestroy:     {},
	KeywordFor:         {},
	KeywordIn:          {},
	KeywordEmit:        {},
	KeywordAuth:        {},
	KeywordPriv:        {},
	KeywordPub:         {},
	KeywordAccess:      {},
	KeywordSet:         {},
	KeywordAll:         {},
	KeywordSelf:        {},
	KeywordInit:        {},
	KeywordContract:    {},
	KeywordAccount:     {},
	KeywordImport:      {},
	KeywordFrom:        {},
	KeywordPre:         {},
	KeywordPost:        {},
	KeywordEvent:       {},
	KeywordStruct:      {},
	KeywordResource:    {},
	KeywordInterface:   {},
	KeywordTransaction: {},
	KeywordPrepare:     {},
	KeywordExecute:     {},
	KeywordCase:        {},
	KeywordSwitch:      {},
	KeywordDefault:     {},
	KeywordEnum:        {},
}

// Keywords that can be used in identifier position without ambiguity.
var SoftKeywords = map[string]struct{}{
	KeywordFrom:    {},
	KeywordAccount: {},
	KeywordSet:     {},
	KeywordAll:     {},
}

// Keywords that aren't allowed in identifier position.
var HardKeywords map[string]struct{} = mapDiff(AllKeywords, SoftKeywords)

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
