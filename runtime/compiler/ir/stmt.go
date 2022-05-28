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

package ir

type Stmt interface {
	isStmt()
	Accept(Visitor) Repr
}

type Sequence struct {
	Stmts []Stmt
}

func (*Sequence) isStmt() {}

func (s *Sequence) Accept(v Visitor) Repr {
	return v.VisitSequence(s)
}

type Block struct {
	Stmts []Stmt
}

func (*Block) isStmt() {}

func (s *Block) Accept(v Visitor) Repr {
	return v.VisitBlock(s)
}

type Loop struct {
	Stmts []Stmt
}

func (*Loop) isStmt() {}

func (s *Loop) Accept(v Visitor) Repr {
	return v.VisitLoop(s)
}

type If struct {
	Test Expr
	Then Stmt
	Else Stmt
}

func (*If) isStmt() {}

func (s *If) Accept(v Visitor) Repr {
	return v.VisitIf(s)
}

type Branch struct {
	Index uint32
}

func (*Branch) isStmt() {}

func (s *Branch) Accept(v Visitor) Repr {
	return v.VisitBranch(s)
}

type BranchIf struct {
	Exp   Expr
	Index uint32
}

func (*BranchIf) isStmt() {}

func (s *BranchIf) Accept(v Visitor) Repr {
	return v.VisitBranchIf(s)
}

type StoreLocal struct {
	LocalIndex uint32
	Exp        Expr
}

func (*StoreLocal) isStmt() {}

func (s *StoreLocal) Accept(v Visitor) Repr {
	return v.VisitStoreLocal(s)
}

type Drop struct {
	Exp Expr
}

func (*Drop) isStmt() {}

func (s *Drop) Accept(v Visitor) Repr {
	return v.VisitDrop(s)
}

type Return struct {
	Exp Expr
}

func (*Return) isStmt() {}

func (s *Return) Accept(v Visitor) Repr {
	return v.VisitReturn(s)
}
