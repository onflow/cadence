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

import (
	mapset "github.com/deckarep/golang-set/v2"
)
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
	KeywordTransaction = "transaction"
	keywordPrepare     = "prepare"
	keywordExecute     = "execute"
	keywordCase        = "case"
	keywordSwitch      = "switch"
	keywordDefault     = "default"
	keywordEnum        = "enum"
)

var Keywords mapset.Set[string] = mapset.NewSet(
	keywordIf,
	keywordElse,
	keywordWhile,
	keywordBreak,
	keywordContinue,
	keywordReturn,
	keywordTrue,
	keywordFalse,
	keywordNil,
	keywordLet,
	keywordVar,
	keywordFun,
	keywordAs,
	keywordCreate,
	keywordDestroy,
	keywordFor,
	keywordIn,
	keywordEmit,
	keywordAuth,
	keywordPriv,
	keywordPub,
	keywordAccess,
	keywordSet,
	keywordAll,
	keywordSelf,
	keywordInit,
	keywordContract,
	keywordAccount,
	keywordImport,
	keywordFrom,
	keywordPre,
	keywordPost,
	keywordEvent,
	keywordStruct,
	keywordResource,
	keywordInterface,
	KeywordTransaction,
	keywordPrepare,
	keywordExecute,
	keywordCase,
	keywordSwitch,
	keywordDefault,
	keywordEnum,
)