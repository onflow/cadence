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

package constants

import (
	mapset "github.com/deckarep/golang-set/v2"
)

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

var AllKeywords mapset.Set[string] = mapset.NewSet(
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
	KeywordPriv,
	KeywordPub,
	KeywordAccess,
	KeywordSet,
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
	KeywordTransaction,
	KeywordPrepare,
	KeywordExecute,
	KeywordCase,
	KeywordSwitch,
	KeywordDefault,
	KeywordEnum,
)

var SoftKeywords mapset.Set[string] = mapset.NewSet(
	KeywordFrom,
	KeywordAccount,
	KeywordSet,
	KeywordAll,
)

var HardKeywords mapset.Set[string] = AllKeywords.Difference(SoftKeywords)
