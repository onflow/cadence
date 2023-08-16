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
		handler: dumpBuiltinTypes,
	},
}

func dumpBuiltinTypes() {
	var types []sema.Type

	for _, ty := range checker.AllBaseSemaTypes() {
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
