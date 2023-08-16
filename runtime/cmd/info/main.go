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

package main

import (
	"flag"
	"fmt"

	"golang.org/x/exp/slices"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib"
	"github.com/onflow/cadence/runtime/tests/checker"
)

type command struct {
	help    string
	handler func()
}

var includeNested = flag.Bool("nested", false, "include nested")
var includeMembers = flag.Bool("members", false, "include members")

func main() {
	flag.Parse()
	args := flag.Args()

	if len(args) <= 0 {
		printAvailableCommands()
		return
	}

	commandName := args[0]

	command, ok := commands[commandName]
	if !ok {
		fmt.Printf("Unknown command %s\n", commandName)
		printAvailableCommands()
		return
	}

	command.handler()
}

var commands = map[string]command{
	"dump-builtin-types": {
		help:    "Dumps all built-in types",
		handler: dumpBuiltinTypes,
	},
	"dump-builtin-values": {
		help:    "Dumps all built-in values",
		handler: dumpBuiltinValues,
	},
}

func dumpBuiltinTypes() {

	allBaseSemaTypes := checker.AllBaseSemaTypes()

	types := make([]sema.Type, 0, len(allBaseSemaTypes))

	for _, ty := range allBaseSemaTypes {
		types = append(types, ty)
	}

	if *includeNested {
		stack := make([]sema.Type, len(types))
		copy(stack, types)

		for len(stack) > 0 {
			lastIndex := len(stack) - 1
			ty := stack[lastIndex]
			stack[lastIndex] = nil
			stack = stack[:lastIndex]

			containerType, ok := ty.(sema.ContainerType)
			if !ok {
				continue
			}

			nestedTypes := containerType.GetNestedTypes()
			if nestedTypes == nil {
				continue
			}

			nestedTypes.Foreach(func(_ string, nestedType sema.Type) {
				types = append(types, nestedType)
				stack = append(stack, nestedType)
			})
		}
	}

	slices.SortFunc(
		types,
		func(a, b sema.Type) bool {
			return a.QualifiedString() < b.QualifiedString()
		},
	)

	for _, ty := range types {
		id := ty.QualifiedString()
		fmt.Printf("- %s\n", id)

		if *includeMembers {
			dumpTypeMembers(ty)
		}
	}
}

func dumpTypeMembers(ty sema.Type) {
	type namedResolver struct {
		name     string
		resolver sema.MemberResolver
	}

	resolversByName := ty.GetMembers()

	namedResolvers := make([]namedResolver, 0, len(resolversByName))

	for name, resolver := range resolversByName {

		namedResolvers = append(
			namedResolvers,
			namedResolver{
				name:     name,
				resolver: resolver,
			},
		)
	}

	slices.SortFunc(
		namedResolvers,
		func(a, b namedResolver) bool {
			return a.name < b.name
		},
	)

	for _, namedResolver := range namedResolvers {
		name := namedResolver.name
		resolver := namedResolver.resolver

		member := resolver.Resolve(nil, name, ast.EmptyRange, nil)
		if member == nil {
			continue
		}

		declarationKind := resolver.Kind

		switch declarationKind {
		case common.DeclarationKindFunction:
			memberType := member.TypeAnnotation.Type
			functionType, ok := memberType.(*sema.FunctionType)
			if !ok {
				panic(errors.NewUnexpectedError(
					"function declaration with non-function type: %s: %s",
					name,
					memberType,
				))
			}

			fmt.Printf(
				"  - %s\n",
				functionType.NamedQualifiedString(name),
			)

		case common.DeclarationKindField:
			fmt.Printf(
				"  - %s %s: %s\n",
				member.VariableKind.Keyword(),
				name,
				member.TypeAnnotation.QualifiedString(),
			)

		default:
			panic(errors.NewUnexpectedError("unsupported declaration kind: %s", declarationKind.Name()))
		}
	}
}

func dumpBuiltinValues() {

	type valueType struct {
		name string
		ty   sema.Type
	}

	allBaseSemaValueTypes := checker.AllBaseSemaValueTypes()
	standardLibraryValues := stdlib.DefaultScriptStandardLibraryValues(nil)

	valueTypes := make([]valueType, 0, len(allBaseSemaValueTypes)+len(standardLibraryValues))

	for name, ty := range allBaseSemaValueTypes {
		valueTypes = append(
			valueTypes,
			valueType{
				name: name,
				ty:   ty,
			},
		)
	}

	for _, value := range standardLibraryValues {
		valueTypes = append(
			valueTypes,
			valueType{
				name: value.ValueDeclarationName(),
				ty:   value.ValueDeclarationType(),
			},
		)
	}

	slices.SortFunc(
		valueTypes,
		func(a, b valueType) bool {
			return a.name < b.name
		},
	)

	for _, valueType := range valueTypes {

		name := valueType.name
		ty := valueType.ty

		if functionType, ok := ty.(*sema.FunctionType); ok {
			fmt.Printf(
				"- %s\n",
				functionType.NamedQualifiedString(name),
			)
		} else {
			fmt.Printf(
				"- %s: %s\n",
				name,
				sema.NewTypeAnnotation(ty).QualifiedString(),
			)
		}

		if *includeMembers {
			dumpTypeMembers(ty)
		}
	}
}

func printAvailableCommands() {
	type commandHelp struct {
		name string
		help string
	}

	commandHelps := make([]commandHelp, 0, len(commands))

	for name, command := range commands {
		commandHelps = append(
			commandHelps,
			commandHelp{
				name: name,
				help: command.help,
			},
		)
	}

	slices.SortFunc(
		commandHelps,
		func(a, b commandHelp) bool {
			return a.name < b.name
		},
	)

	println("Available commands:")

	for _, commandHelp := range commandHelps {
		fmt.Printf("  %s\t%s\n", commandHelp.name, commandHelp.help)
	}
}
