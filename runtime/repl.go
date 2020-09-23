/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2020 Dapper Labs, Inc.
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

package runtime

import (
	"fmt"
	"math/rand"
	"sort"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/parser2"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib"
	"github.com/onflow/cadence/runtime/trampoline"
)

type REPL struct {
	checker  *sema.Checker
	inter    *interpreter.Interpreter
	onError  func(error)
	onResult func(interpreter.Value)
}

func NewREPL(onError func(error), onResult func(interpreter.Value), checkerOptions []sema.Option) (*REPL, error) {

	valueDeclarations := append(
		stdlib.FlowBuiltInFunctions(stdlib.FlowBuiltinImpls{
			CreateAccount: func(invocation interpreter.Invocation) trampoline.Trampoline {
				panic(fmt.Errorf("cannot create accounts in the REPL"))
			},
			GetAccount: func(invocation interpreter.Invocation) trampoline.Trampoline {
				panic(fmt.Errorf("cannot get accounts in the REPL"))
			},
			Log: stdlib.LogFunction.Function.Function,
			GetCurrentBlock: func(invocation interpreter.Invocation) trampoline.Trampoline {
				panic(fmt.Errorf("cannot get blocks in the REPL"))
			},
			GetBlock: func(invocation interpreter.Invocation) trampoline.Trampoline {
				panic(fmt.Errorf("cannot get blocks in the REPL"))
			},
			UnsafeRandom: func(invocation interpreter.Invocation) trampoline.Trampoline {
				return trampoline.Done{Result: interpreter.UInt64Value(rand.Uint64())}
			},
		}),
		stdlib.BuiltinFunctions...,
	)

	checkerOptions = append(
		[]sema.Option{
			sema.WithPredeclaredValues(valueDeclarations.ToValueDeclarations()),
			sema.WithPredeclaredTypes(typeDeclarations),
			sema.WithAccessCheckMode(sema.AccessCheckModeNotSpecifiedUnrestricted),
		},
		checkerOptions...,
	)

	checker, err := sema.NewChecker(
		nil,
		REPLLocation{},
		checkerOptions...,
	)
	if err != nil {
		return nil, err
	}

	values := valueDeclarations.ToValues()

	var uuid uint64

	inter, err := interpreter.NewInterpreter(
		checker,
		interpreter.WithPredefinedValues(values),
		interpreter.WithUUIDHandler(func() uint64 {
			defer func() { uuid++ }()
			return uuid
		}),
	)
	if err != nil {
		return nil, err
	}

	repl := &REPL{
		checker:  checker,
		inter:    inter,
		onError:  onError,
		onResult: onResult,
	}
	return repl, nil
}

func (r *REPL) handleCheckerError(code string) bool {
	err := r.checker.CheckerError()
	if err == nil {
		return true
	}
	if r.onError != nil {
		r.onError(err)
	}
	return false
}

func (r *REPL) execute(element ast.Element) {
	result := trampoline.Run(element.Accept(r.inter).(trampoline.Trampoline))
	expStatementRes, ok := result.(interpreter.ExpressionStatementResult)
	if !ok {
		return
	}
	if r.onResult == nil {
		return
	}
	r.onResult(expStatementRes.Value)
}

func (r *REPL) check(element ast.Element, code string) bool {
	element.Accept(r.checker)
	return r.handleCheckerError(code)
}

func (r *REPL) Accept(code string) (inputIsComplete bool) {

	// TODO: detect if the input is complete
	inputIsComplete = true

	var err error
	result, errs := parser2.ParseStatements(code)
	if len(errs) > 0 {
		err = parser2.Error{
			Errors: errs,
		}
	}

	if !inputIsComplete {
		return
	}

	if err != nil {
		r.onError(err)
		return
	}

	r.checker.ResetErrors()
	r.checker.ResetHints()

	for _, element := range result {

		switch typedElement := element.(type) {
		case ast.Declaration:
			program := &ast.Program{
				Declarations: []ast.Declaration{typedElement},
			}

			if !r.check(program, code) {
				return
			}

			r.execute(typedElement)

		case ast.Statement:
			r.checker.Program = nil

			if !r.check(typedElement, code) {
				return
			}

			r.execute(typedElement)

		default:
			panic(errors.NewUnreachableError())
		}
	}

	return
}

type REPLSuggestion struct {
	Name, Description string
}

func (r *REPL) Suggestions() (result []REPLSuggestion) {
	names := map[string]string{}

	for name, variable := range r.checker.GlobalValues {
		if names[name] != "" {
			continue
		}
		names[name] = variable.Type.String()
	}

	for name, description := range names {
		result = append(result, REPLSuggestion{
			Name:        name,
			Description: description,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		a := result[i]
		b := result[j]
		return a.Name < b.Name
	})

	return
}
