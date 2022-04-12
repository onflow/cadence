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

type Expr interface {
	isExpr()
	Accept(Visitor) Repr
}

type Const struct {
	Constant Constant
}

func (*Const) isExpr() {}

func (e *Const) Accept(v Visitor) Repr {
	return v.VisitConst(e)
}

type CopyLocal struct {
	LocalIndex uint32
}

func (*CopyLocal) isExpr() {}

func (e *CopyLocal) Accept(v Visitor) Repr {
	return v.VisitCopyLocal(e)
}

type MoveLocal struct {
	LocalIndex uint32
}

func (*MoveLocal) isExpr() {}

func (e *MoveLocal) Accept(v Visitor) Repr {
	return v.VisitMoveLocal(e)
}

type UnOpExpr struct {
	Op   UnOp
	Expr Expr
}

func (*UnOpExpr) isExpr() {}

func (e *UnOpExpr) Accept(v Visitor) Repr {
	return v.VisitUnOpExpr(e)
}

type BinOpExpr struct {
	Op    BinOp
	Left  Expr
	Right Expr
}

func (*BinOpExpr) isExpr() {}

func (e *BinOpExpr) Accept(v Visitor) Repr {
	return v.VisitBinOpExpr(e)
}

type Call struct {
	FunctionIndex uint32
	Arguments     []Expr
}

func (*Call) isExpr() {}

func (e *Call) Accept(v Visitor) Repr {
	return v.VisitCall(e)
}
