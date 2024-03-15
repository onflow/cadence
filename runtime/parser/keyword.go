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

package parser

import "github.com/SaveTheRbtz/mph"

// NOTE: ensure to update allKeywords when adding a new keyword
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
	KeywordAccess      = "access"
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
	KeywordEntitlement = "entitlement"
	KeywordMapping     = "mapping"
	KeywordTransaction = "transaction"
	KeywordPrepare     = "prepare"
	KeywordExecute     = "execute"
	KeywordCase        = "case"
	KeywordSwitch      = "switch"
	KeywordDefault     = "default"
	KeywordEnum        = "enum"
	KeywordView        = "view"
	KeywordAttachment  = "attachment"
	KeywordAttach      = "attach"
	KeywordRemove      = "remove"
	KeywordTo          = "to"
	KeywordRequire     = "require"
	KeywordStatic      = "static"
	KeywordNative      = "native"
	KeywordPub         = "pub"
	KeywordPriv        = "priv"
	KeywordInclude     = "include"
	KeywordTry         = "try"
	KeywordCatch       = "catch"
	KeywordFinally     = "finally"
	KeywordGoto        = "goto"
	KeywordConst       = "const"
	KeywordExport      = "export"
	KeywordThrow       = "throw"
	KeywordThrows      = "throws"
	KeywordRequires    = "requires"
	KeywordWhere       = "where"
	KeywordFinal       = "final"
	KeywordInternal    = "internal"
	KeywordTypealias   = "typealias"
	KeywordType        = "type"
	KeywordRepeat      = "repeat"
	KeywordGuard       = "guard"
	KeywordIs          = "is"
	// NOTE: ensure to update allKeywords when adding a new keyword
)

var allKeywords = []string{
	KeywordIf,
	KeywordElse,
	KeywordWhile,
	KeywordBreak,
	KeywordContinue,
	KeywordReturn,
	KeywordTrue,
	KeywordFalse,
	KeywordNil,
	KeywordLet,
	KeywordVar,
	KeywordFun,
	KeywordAs,
	KeywordCreate,
	KeywordDestroy,
	KeywordFor,
	KeywordIn,
	KeywordEmit,
	KeywordAuth,
	KeywordAccess,
	KeywordAll,
	KeywordSelf,
	KeywordInit,
	KeywordContract,
	KeywordAccount,
	KeywordImport,
	KeywordFrom,
	KeywordPre,
	KeywordPost,
	KeywordEvent,
	KeywordStruct,
	KeywordResource,
	KeywordInterface,
	KeywordEntitlement,
	KeywordTransaction,
	KeywordPrepare,
	KeywordExecute,
	KeywordCase,
	KeywordSwitch,
	KeywordDefault,
	KeywordEnum,
	KeywordView,
	KeywordMapping,
	KeywordRequire,
	KeywordAttach,
	KeywordAttachment,
	KeywordTo,
	KeywordRemove,
	KeywordStatic,
	KeywordNative,
	KeywordPub,
	KeywordPriv,
	KeywordInclude,
	KeywordTry,
	KeywordCatch,
	KeywordFinally,
	KeywordGoto,
	KeywordConst,
	KeywordExport,
	KeywordThrow,
	KeywordThrows,
	KeywordRequires,
	KeywordWhere,
	KeywordFinal,
	KeywordInternal,
	KeywordTypealias,
	KeywordType,
	KeywordRepeat,
	KeywordGuard,
	KeywordIs,
}

// softKeywords are keywords that can be used as identifiers anywhere,
// without any restriction or ambiguity.
var softKeywords = []string{
	KeywordFrom,
	KeywordAccount,
	KeywordAll,
	KeywordView,
	KeywordAttach,
	KeywordRemove,
	KeywordTo,
	KeywordType,
}

var softKeywordsTable = mph.Build(softKeywords)

// hardKeywords are restricted from being used as identifiers in certain places.
// i.e: places where ambiguity can exist, such as composite declaration names, function names, etc.
// However, they are not restricted to be used as fields names, and many other places.
var hardKeywords = filter(
	allKeywords,
	func(keyword string) bool {
		_, ok := softKeywordsTable.Lookup(keyword)
		return !ok
	},
)

var hardKeywordsTable = mph.Build(hardKeywords)

func IsHardKeyword(identifier string) bool {
	_, ok := hardKeywordsTable.Lookup(identifier)
	return ok
}

func filter[T comparable](items []T, f func(T) bool) []T {
	result := make([]T, 0, len(items))
	for _, item := range items {
		if f(item) {
			result = append(result, item)
		}
	}
	return result
}
