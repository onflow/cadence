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

package checker

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/sema"
	. "github.com/onflow/cadence/runtime/tests/utils"
)

func TestCheckFailableCastingWithResourceAnnotation(t *testing.T) {

	t.Parallel()

	test := func(compositeKind common.CompositeKind) {

		body := "{}"
		switch compositeKind {
		case common.CompositeKindEvent:
			body = "()"
		case common.CompositeKindEnum:
			body = "{ case a }"
		}

		conformances := ""
		if compositeKind == common.CompositeKindEnum {
			conformances = ": Int"
		}

		var baseType string
		if compositeKind == common.CompositeKindAttachment {
			baseType = "for AnyStruct"
		}

		t.Run(compositeKind.Keyword(), func(t *testing.T) {

			t.Parallel()

			_, err := ParseAndCheck(t,
				fmt.Sprintf(
					`
                      %[1]s T %[7]s %[2]s %[3]s

                      let test %[4]s %[5]s T%[6]s as? @T
                    `,
					compositeKind.Keyword(),
					conformances,
					body,
					compositeKind.TransferOperator(),
					compositeKind.ConstructionKeyword(),
					constructorArguments(compositeKind),
					baseType,
				),
			)

			switch compositeKind {
			case common.CompositeKindResource:

				errs := RequireCheckerErrors(t, err, 2)

				assert.IsType(t, &sema.InvalidFailableResourceDowncastOutsideOptionalBindingError{}, errs[0])
				assert.IsType(t, &sema.InvalidNonIdentifierFailableResourceDowncast{}, errs[1])

			case common.CompositeKindAttachment:

				errs := RequireCheckerErrors(t, err, 2)

				assert.IsType(t, &sema.InvalidAttachmentAnnotationError{}, errs[0])
				assert.IsType(t, &sema.InvalidAttachmentUsageError{}, errs[1])

			case common.CompositeKindStructure,
				common.CompositeKindContract,
				common.CompositeKindEnum:

				errs := RequireCheckerErrors(t, err, 1)

				assert.IsType(t, &sema.InvalidResourceAnnotationError{}, errs[0])

			case common.CompositeKindEvent:

				errs := RequireCheckerErrors(t, err, 2)

				assert.IsType(t, &sema.InvalidResourceAnnotationError{}, errs[0])
				assert.IsType(t, &sema.InvalidEventUsageError{}, errs[1])

			default:
				panic(errors.NewUnreachableError())
			}
		})
	}

	for _, compositeKind := range common.AllCompositeKinds {

		test(compositeKind)
	}
}

func TestCheckFunctionDeclarationParameterWithResourceAnnotation(t *testing.T) {

	t.Parallel()

	test := func(kind common.CompositeKind) {

		body := "{}"
		if kind == common.CompositeKindEvent {
			body = "()"
		}

		conformances := ""
		if kind == common.CompositeKindEnum {
			conformances = ": Int"
		}

		var baseType string
		if kind == common.CompositeKindAttachment {
			baseType = "for AnyStruct"
		}

		t.Run(kind.Keyword(), func(t *testing.T) {

			t.Parallel()

			_, err := ParseAndCheck(t,
				fmt.Sprintf(
					`
                      %[1]s T %[5]s %[2]s %[3]s

                      fun test(r: @T) {
                          %[4]s r
                      }
                    `,
					kind.Keyword(),
					conformances,
					body,
					kind.DestructionKeyword(),
					baseType,
				),
			)

			switch kind {
			case common.CompositeKindResource:
				require.NoError(t, err)

			case common.CompositeKindAttachment:
				errs := RequireCheckerErrors(t, err, 1)

				assert.IsType(t, &sema.InvalidAttachmentAnnotationError{}, errs[0])

			case common.CompositeKindStructure,
				common.CompositeKindContract,
				common.CompositeKindEvent,
				common.CompositeKindEnum:

				errs := RequireCheckerErrors(t, err, 1)

				assert.IsType(t, &sema.InvalidResourceAnnotationError{}, errs[0])

			default:
				panic(errors.NewUnreachableError())
			}
		})
	}

	for _, kind := range common.AllCompositeKinds {

		test(kind)
	}
}

func TestCheckFunctionDeclarationParameterWithoutResourceAnnotation(t *testing.T) {

	t.Parallel()

	test := func(kind common.CompositeKind) {

		body := "{}"
		if kind == common.CompositeKindEvent {
			body = "()"
		}

		conformances := ""
		if kind == common.CompositeKindEnum {
			conformances = ": Int"
		}

		var baseType string
		if kind == common.CompositeKindAttachment {
			baseType = "for AnyStruct"
		}

		t.Run(kind.Keyword(), func(t *testing.T) {

			t.Parallel()

			_, err := ParseAndCheck(t,
				fmt.Sprintf(
					`
                      %[1]s T %[5]s %[2]s %[3]s

                      fun test(r: T) {
                          %[4]s r
                      }
                    `,
					kind.Keyword(),
					conformances,
					body,
					kind.DestructionKeyword(),
					baseType,
				),
			)

			switch kind {
			case common.CompositeKindResource:
				errs := RequireCheckerErrors(t, err, 1)

				assert.IsType(t, &sema.MissingResourceAnnotationError{}, errs[0])

			case common.CompositeKindAttachment:
				errs := RequireCheckerErrors(t, err, 1)

				assert.IsType(t, &sema.InvalidAttachmentAnnotationError{}, errs[0])
			case common.CompositeKindStructure,
				common.CompositeKindContract,
				common.CompositeKindEvent,
				common.CompositeKindEnum:

				require.NoError(t, err)

			default:
				panic(errors.NewUnreachableError())
			}
		})
	}

	for _, kind := range common.AllCompositeKinds {

		test(kind)
	}
}

func TestCheckFunctionDeclarationReturnTypeWithResourceAnnotation(t *testing.T) {

	t.Parallel()

	test := func(compositeKind common.CompositeKind) {

		body := "{}"
		switch compositeKind {
		case common.CompositeKindEvent:
			body = "()"
		case common.CompositeKindEnum:
			body = "{ case a }"
		}
		conformances := ""
		if compositeKind == common.CompositeKindEnum {
			conformances = ": Int"
		}

		var baseType string
		if compositeKind == common.CompositeKindAttachment {
			baseType = "for AnyStruct"
		}

		t.Run(compositeKind.Keyword(), func(t *testing.T) {

			t.Parallel()

			_, err := ParseAndCheck(t,
				fmt.Sprintf(
					`
                      %[1]s T %[7]s %[2]s %[3]s

                      fun test(): @T {
                          return %[4]s %[5]s T%[6]s
                      }
                    `,
					compositeKind.Keyword(),
					conformances,
					body,
					compositeKind.MoveOperator(),
					compositeKind.ConstructionKeyword(),
					constructorArguments(compositeKind),
					baseType,
				),
			)

			switch compositeKind {
			case common.CompositeKindResource:

				require.NoError(t, err)

			case common.CompositeKindAttachment:
				errs := RequireCheckerErrors(t, err, 2)

				assert.IsType(t, &sema.InvalidAttachmentAnnotationError{}, errs[0])
				assert.IsType(t, &sema.InvalidAttachmentUsageError{}, errs[1])

			case common.CompositeKindStructure,
				common.CompositeKindEnum:

				errs := RequireCheckerErrors(t, err, 1)

				assert.IsType(t, &sema.InvalidResourceAnnotationError{}, errs[0])

			case common.CompositeKindContract:

				errs := RequireCheckerErrors(t, err, 2)

				assert.IsType(t, &sema.InvalidResourceAnnotationError{}, errs[0])
				assert.IsType(t, &sema.InvalidMoveError{}, errs[1])

			case common.CompositeKindEvent:

				errs := RequireCheckerErrors(t, err, 2)

				assert.IsType(t, &sema.InvalidResourceAnnotationError{}, errs[0])
				assert.IsType(t, &sema.InvalidEventUsageError{}, errs[1])

			default:
				panic(errors.NewUnreachableError())
			}
		})
	}

	for _, compositeKind := range common.AllCompositeKinds {

		test(compositeKind)
	}
}

func TestCheckFunctionDeclarationReturnTypeWithoutResourceAnnotation(t *testing.T) {

	t.Parallel()

	test := func(compositeKind common.CompositeKind) {

		body := "{}"
		switch compositeKind {
		case common.CompositeKindEvent:
			body = "()"
		case common.CompositeKindEnum:
			body = "{ case a }"
		}

		conformances := ""
		if compositeKind == common.CompositeKindEnum {
			conformances = ": Int"
		}

		var baseType string
		if compositeKind == common.CompositeKindAttachment {
			baseType = "for AnyStruct"
		}

		t.Run(compositeKind.Keyword(), func(t *testing.T) {

			t.Parallel()

			_, err := ParseAndCheck(t,
				fmt.Sprintf(
					`
                      %[1]s T %[7]s %[2]s %[3]s

                      fun test(): T {
                          return %[4]s %[5]s T%[6]s
                      }
                    `,
					compositeKind.Keyword(),
					conformances,
					body,
					compositeKind.MoveOperator(),
					compositeKind.ConstructionKeyword(),
					constructorArguments(compositeKind),
					baseType,
				),
			)

			switch compositeKind {
			case common.CompositeKindResource:
				errs := RequireCheckerErrors(t, err, 1)

				assert.IsType(t, &sema.MissingResourceAnnotationError{}, errs[0])

			case common.CompositeKindAttachment:
				errs := RequireCheckerErrors(t, err, 2)

				assert.IsType(t, &sema.InvalidAttachmentAnnotationError{}, errs[0])
				assert.IsType(t, &sema.InvalidAttachmentUsageError{}, errs[1])

			case common.CompositeKindStructure,
				common.CompositeKindContract,
				common.CompositeKindEnum:

				require.NoError(t, err)

			case common.CompositeKindEvent:
				errs := RequireCheckerErrors(t, err, 1)

				assert.IsType(t, &sema.InvalidEventUsageError{}, errs[0])

			default:
				panic(errors.NewUnreachableError())
			}
		})
	}

	for _, compositeKind := range common.AllCompositeKinds {

		if compositeKind == common.CompositeKindContract {
			continue
		}

		test(compositeKind)
	}
}

func TestCheckVariableDeclarationWithResourceAnnotation(t *testing.T) {

	t.Parallel()

	test := func(compositeKind common.CompositeKind) {

		body := "{}"
		switch compositeKind {
		case common.CompositeKindEvent:
			body = "()"
		case common.CompositeKindEnum:
			body = "{ case a }"
		}

		conformances := ""
		if compositeKind == common.CompositeKindEnum {
			conformances = ": Int"
		}

		var baseType string
		if compositeKind == common.CompositeKindAttachment {
			baseType = "for AnyStruct"
		}

		t.Run(compositeKind.Keyword(), func(t *testing.T) {

			t.Parallel()

			_, err := ParseAndCheck(t,
				fmt.Sprintf(
					`
                      %[1]s T %[7]s %[2]s %[3]s

                      let test: @T %[4]s %[5]s T%[6]s
                    `,
					compositeKind.Keyword(),
					conformances,
					body,
					compositeKind.TransferOperator(),
					compositeKind.ConstructionKeyword(),
					constructorArguments(compositeKind),
					baseType,
				),
			)

			switch compositeKind {
			case common.CompositeKindResource:
				require.NoError(t, err)

			case common.CompositeKindAttachment:
				errs := RequireCheckerErrors(t, err, 2)

				assert.IsType(t, &sema.InvalidAttachmentAnnotationError{}, errs[0])
				assert.IsType(t, &sema.InvalidAttachmentUsageError{}, errs[1])

			case common.CompositeKindStructure,
				common.CompositeKindEnum:

				errs := RequireCheckerErrors(t, err, 1)

				assert.IsType(t, &sema.InvalidResourceAnnotationError{}, errs[0])

			case common.CompositeKindContract:
				errs := RequireCheckerErrors(t, err, 2)

				assert.IsType(t, &sema.InvalidResourceAnnotationError{}, errs[0])
				assert.IsType(t, &sema.InvalidMoveError{}, errs[1])

			case common.CompositeKindEvent:
				errs := RequireCheckerErrors(t, err, 2)

				assert.IsType(t, &sema.InvalidResourceAnnotationError{}, errs[0])
				assert.IsType(t, &sema.InvalidEventUsageError{}, errs[1])

			default:
				panic(errors.NewUnreachableError())
			}
		})
	}

	for _, compositeKind := range common.AllCompositeKinds {

		test(compositeKind)
	}
}

func TestCheckVariableDeclarationWithoutResourceAnnotation(t *testing.T) {

	t.Parallel()

	test := func(compositeKind common.CompositeKind) {

		body := "{}"
		switch compositeKind {
		case common.CompositeKindEvent:
			body = "()"
		case common.CompositeKindEnum:
			body = "{ case a }"
		}

		conformances := ""
		if compositeKind == common.CompositeKindEnum {
			conformances = ": Int"
		}

		var baseType string
		if compositeKind == common.CompositeKindAttachment {
			baseType = "for AnyStruct"
		}

		t.Run(compositeKind.Keyword(), func(t *testing.T) {

			t.Parallel()

			_, err := ParseAndCheck(t,
				fmt.Sprintf(
					`
                      %[1]s T %[7]s %[2]s %[3]s

                      let test: T %[4]s %[5]s T%[6]s
                    `,
					compositeKind.Keyword(),
					conformances,
					body,
					compositeKind.TransferOperator(),
					compositeKind.ConstructionKeyword(),
					constructorArguments(compositeKind),
					baseType,
				),
			)

			switch compositeKind {
			case common.CompositeKindResource:
				errs := RequireCheckerErrors(t, err, 1)

				assert.IsType(t, &sema.MissingResourceAnnotationError{}, errs[0])

			case common.CompositeKindAttachment:
				errs := RequireCheckerErrors(t, err, 2)

				assert.IsType(t, &sema.InvalidAttachmentAnnotationError{}, errs[0])
				assert.IsType(t, &sema.InvalidAttachmentUsageError{}, errs[1])

			case common.CompositeKindStructure,
				common.CompositeKindContract,
				common.CompositeKindEnum:

				require.NoError(t, err)

			case common.CompositeKindEvent:
				errs := RequireCheckerErrors(t, err, 1)

				assert.IsType(t, &sema.InvalidEventUsageError{}, errs[0])

			default:
				panic(errors.NewUnreachableError())
			}
		})
	}

	for _, compositeKind := range common.AllCompositeKinds {

		if compositeKind == common.CompositeKindContract {
			continue
		}

		test(compositeKind)
	}
}

func TestCheckFieldDeclarationWithResourceAnnotation(t *testing.T) {

	t.Parallel()

	test := func(kind common.CompositeKind) {

		t.Run(kind.Keyword(), func(t *testing.T) {

			t.Parallel()

			destructor := ""
			if kind == common.CompositeKindResource {
				destructor = `
                  destroy() {
                      destroy self.t
                  }
                `
			}

			_, err := ParseAndCheck(t,
				fmt.Sprintf(
					`
                      %[1]s T {}

                      %[1]s U {
                          let t: @T
                          init(t: @T) {
                              self.t %[2]s t
                          }

                          %[3]s
                      }
                    `,
					kind.Keyword(),
					kind.TransferOperator(),
					destructor,
				),
			)

			switch kind {
			case common.CompositeKindResource:
				require.NoError(t, err)

			case common.CompositeKindStructure:
				errs := RequireCheckerErrors(t, err, 2)

				// NOTE: one invalid resource annotation error for field, one for parameter

				assert.IsType(t, &sema.InvalidResourceAnnotationError{}, errs[0])
				assert.IsType(t, &sema.InvalidResourceAnnotationError{}, errs[1])

			case common.CompositeKindContract:
				errs := RequireCheckerErrors(t, err, 4)

				// NOTE: one invalid resource annotation error for field, one for parameter

				assert.IsType(t, &sema.InvalidResourceAnnotationError{}, errs[0])
				assert.IsType(t, &sema.FieldTypeNotStorableError{}, errs[1])
				assert.IsType(t, &sema.InvalidResourceAnnotationError{}, errs[2])
				assert.IsType(t, &sema.InvalidMoveError{}, errs[3])

			default:
				panic(errors.NewUnreachableError())
			}
		})
	}

	for _, kind := range common.InstantiableCompositeKindsWithFieldsAndFunctions {
		test(kind)
	}
}

func TestCheckFieldDeclarationWithoutResourceAnnotation(t *testing.T) {

	t.Parallel()

	test := func(kind common.CompositeKind) {

		t.Run(kind.Keyword(), func(t *testing.T) {

			t.Parallel()

			destructor := ""
			if kind == common.CompositeKindResource {
				destructor = `
                  destroy() {
                      destroy self.t
                  }
                `
			}

			_, err := ParseAndCheck(t,
				fmt.Sprintf(
					`
                      %[1]s T {}

                      %[1]s U {
                          let t: T
                          init(t: T) {
                              self.t %[2]s t
                          }

                          %[3]s
                      }
                    `,
					kind.Keyword(),
					kind.TransferOperator(),
					destructor,
				),
			)

			switch kind {
			case common.CompositeKindResource:
				// NOTE: one missing resource annotation error for field, one for parameter

				errs := RequireCheckerErrors(t, err, 2)

				assert.IsType(t, &sema.MissingResourceAnnotationError{}, errs[0])
				assert.IsType(t, &sema.MissingResourceAnnotationError{}, errs[1])

			case common.CompositeKindContract:
				errs := RequireCheckerErrors(t, err, 2)

				assert.IsType(t, &sema.FieldTypeNotStorableError{}, errs[0])
				assert.IsType(t, &sema.InvalidMoveError{}, errs[1])

			case common.CompositeKindStructure:
				require.NoError(t, err)

			default:
				panic(errors.NewUnreachableError())
			}
		})
	}

	for _, kind := range common.InstantiableCompositeKindsWithFieldsAndFunctions {
		if kind == common.CompositeKindAttachment {
			continue
		}
		test(kind)
	}
}

func TestCheckFunctionExpressionParameterWithResourceAnnotation(t *testing.T) {

	t.Parallel()

	test := func(kind common.CompositeKind) {

		body := "{}"
		if kind == common.CompositeKindEvent {
			body = "()"
		}

		conformances := ""
		if kind == common.CompositeKindEnum {
			conformances = ": Int"
		}

		var baseType string
		if kind == common.CompositeKindAttachment {
			baseType = "for AnyStruct"
		}

		t.Run(kind.Keyword(), func(t *testing.T) {

			t.Parallel()

			_, err := ParseAndCheck(t,
				fmt.Sprintf(
					`
                      %[1]s T %[5]s %[2]s %[3]s

                      let test = fun (r: @T) {
                          %[4]s r
                      }
                    `,
					kind.Keyword(),
					conformances,
					body,
					kind.DestructionKeyword(),
					baseType,
				),
			)

			switch kind {
			case common.CompositeKindResource:
				require.NoError(t, err)

			case common.CompositeKindAttachment:
				errs := RequireCheckerErrors(t, err, 1)

				assert.IsType(t, &sema.InvalidAttachmentAnnotationError{}, errs[0])

			case common.CompositeKindStructure,
				common.CompositeKindContract,
				common.CompositeKindEvent,
				common.CompositeKindEnum:

				errs := RequireCheckerErrors(t, err, 1)

				assert.IsType(t, &sema.InvalidResourceAnnotationError{}, errs[0])

			default:
				panic(errors.NewUnreachableError())
			}
		})
	}

	for _, kind := range common.AllCompositeKinds {

		test(kind)
	}
}

func TestCheckFunctionExpressionParameterWithoutResourceAnnotation(t *testing.T) {

	t.Parallel()

	test := func(kind common.CompositeKind) {

		body := "{}"
		if kind == common.CompositeKindEvent {
			body = "()"
		}

		conformances := ""
		if kind == common.CompositeKindEnum {
			conformances = ": Int"
		}

		var baseType string
		if kind == common.CompositeKindAttachment {
			baseType = "for AnyStruct"
		}

		t.Run(kind.Keyword(), func(t *testing.T) {

			t.Parallel()

			_, err := ParseAndCheck(t,
				fmt.Sprintf(
					`
                      %[1]s T %[5]s %[2]s %[3]s

                      let test = fun (r: T) {
                          %[4]s r
                      }
                    `,
					kind.Keyword(),
					conformances,
					body,
					kind.DestructionKeyword(),
					baseType,
				),
			)

			switch kind {
			case common.CompositeKindResource:

				errs := RequireCheckerErrors(t, err, 1)

				assert.IsType(t, &sema.MissingResourceAnnotationError{}, errs[0])

			case common.CompositeKindAttachment:
				errs := RequireCheckerErrors(t, err, 1)

				assert.IsType(t, &sema.InvalidAttachmentAnnotationError{}, errs[0])

			case common.CompositeKindStructure,
				common.CompositeKindContract,
				common.CompositeKindEvent,
				common.CompositeKindEnum:

				require.NoError(t, err)

			default:
				panic(errors.NewUnreachableError())
			}
		})
	}

	for _, kind := range common.AllCompositeKinds {

		test(kind)
	}
}

func TestCheckFunctionExpressionReturnTypeWithResourceAnnotation(t *testing.T) {

	t.Parallel()

	test := func(compositeKind common.CompositeKind) {

		body := "{}"
		switch compositeKind {
		case common.CompositeKindEvent:
			body = "()"
		case common.CompositeKindEnum:
			body = "{ case a }"
		}

		conformances := ""
		if compositeKind == common.CompositeKindEnum {
			conformances = ": Int"
		}

		var baseType string
		if compositeKind == common.CompositeKindAttachment {
			baseType = "for AnyStruct"
		}

		t.Run(compositeKind.Keyword(), func(t *testing.T) {

			t.Parallel()

			_, err := ParseAndCheck(t,
				fmt.Sprintf(
					`
                      %[1]s T %[7]s %[2]s %[3]s

                      let test = fun (): @T {
                          return %[4]s %[5]s T%[6]s
                      }
                    `,
					compositeKind.Keyword(),
					conformances,
					body,
					compositeKind.MoveOperator(),
					compositeKind.ConstructionKeyword(),
					constructorArguments(compositeKind),
					baseType,
				),
			)

			switch compositeKind {
			case common.CompositeKindResource:
				require.NoError(t, err)

			case common.CompositeKindAttachment:
				errs := RequireCheckerErrors(t, err, 2)

				assert.IsType(t, &sema.InvalidAttachmentAnnotationError{}, errs[0])
				assert.IsType(t, &sema.InvalidAttachmentUsageError{}, errs[1])

			case common.CompositeKindStructure,
				common.CompositeKindEnum:

				errs := RequireCheckerErrors(t, err, 1)

				assert.IsType(t, &sema.InvalidResourceAnnotationError{}, errs[0])

			case common.CompositeKindContract:
				errs := RequireCheckerErrors(t, err, 2)

				assert.IsType(t, &sema.InvalidResourceAnnotationError{}, errs[0])
				assert.IsType(t, &sema.InvalidMoveError{}, errs[1])

			case common.CompositeKindEvent:
				errs := RequireCheckerErrors(t, err, 2)

				assert.IsType(t, &sema.InvalidResourceAnnotationError{}, errs[0])
				assert.IsType(t, &sema.InvalidEventUsageError{}, errs[1])

			default:
				panic(errors.NewUnreachableError())
			}
		})
	}

	for _, compositeKind := range common.AllCompositeKinds {

		test(compositeKind)
	}
}

func TestCheckFunctionExpressionReturnTypeWithoutResourceAnnotation(t *testing.T) {

	t.Parallel()

	test := func(compositeKind common.CompositeKind) {

		body := "{}"
		switch compositeKind {
		case common.CompositeKindEvent:
			body = "()"
		case common.CompositeKindEnum:
			body = "{ case a }"
		}

		conformances := ""
		if compositeKind == common.CompositeKindEnum {
			conformances = ": Int"
		}

		var baseType string
		if compositeKind == common.CompositeKindAttachment {
			baseType = "for AnyStruct"
		}

		t.Run(compositeKind.Keyword(), func(t *testing.T) {

			t.Parallel()

			_, err := ParseAndCheck(t,
				fmt.Sprintf(
					`
                      %[1]s T %[7]s %[2]s %[3]s

                      let test = fun (): T {
                          return %[4]s %[5]s T%[6]s
                      }
                    `,
					compositeKind.Keyword(),
					conformances,
					body,
					compositeKind.MoveOperator(),
					compositeKind.ConstructionKeyword(),
					constructorArguments(compositeKind),
					baseType,
				),
			)

			switch compositeKind {
			case common.CompositeKindResource:
				errs := RequireCheckerErrors(t, err, 1)

				assert.IsType(t, &sema.MissingResourceAnnotationError{}, errs[0])

			case common.CompositeKindAttachment:
				errs := RequireCheckerErrors(t, err, 2)

				assert.IsType(t, &sema.InvalidAttachmentAnnotationError{}, errs[0])
				assert.IsType(t, &sema.InvalidAttachmentUsageError{}, errs[1])

			case common.CompositeKindStructure,
				common.CompositeKindContract,
				common.CompositeKindEnum:

				require.NoError(t, err)

			case common.CompositeKindEvent:
				errs := RequireCheckerErrors(t, err, 1)

				assert.IsType(t, &sema.InvalidEventUsageError{}, errs[0])

			default:
				panic(errors.NewUnreachableError())
			}
		})
	}

	for _, compositeKind := range common.AllCompositeKinds {

		if compositeKind == common.CompositeKindContract {
			continue
		}

		test(compositeKind)
	}
}

func TestCheckFunctionTypeParameterWithResourceAnnotation(t *testing.T) {

	t.Parallel()

	test := func(kind common.CompositeKind) {

		body := "{}"
		if kind == common.CompositeKindEvent {
			body = "()"
		}

		conformances := ""
		if kind == common.CompositeKindEnum {
			conformances = ": Int"
		}

		var baseType string
		if kind == common.CompositeKindAttachment {
			baseType = "for AnyStruct"
		}

		t.Run(kind.Keyword(), func(t *testing.T) {

			t.Parallel()

			_, err := ParseAndCheck(t,
				fmt.Sprintf(
					`
                      %[1]s T %[5]s %[2]s %[3]s

                      let test: fun(@T): Void = fun (r: @T) {
                          %[4]s r
                      }
                    `,
					kind.Keyword(),
					conformances,
					body,
					kind.DestructionKeyword(),
					baseType,
				),
			)

			switch kind {
			case common.CompositeKindResource:
				require.NoError(t, err)
			case common.CompositeKindAttachment:
				errs := RequireCheckerErrors(t, err, 2)

				assert.IsType(t, &sema.InvalidAttachmentAnnotationError{}, errs[0])

			case common.CompositeKindStructure,
				common.CompositeKindContract,
				common.CompositeKindEvent,
				common.CompositeKindEnum:

				errs := RequireCheckerErrors(t, err, 2)

				assert.IsType(t, &sema.InvalidResourceAnnotationError{}, errs[0])
				assert.IsType(t, &sema.InvalidResourceAnnotationError{}, errs[1])

			default:
				panic(errors.NewUnreachableError())
			}
		})
	}

	for _, kind := range common.AllCompositeKinds {

		test(kind)
	}
}

// NOTE: variable type instead of function parameter
func TestCheckFunctionTypeParameterWithoutResourceAnnotation(t *testing.T) {

	t.Parallel()

	test := func(kind common.CompositeKind) {

		body := "{}"
		if kind == common.CompositeKindEvent {
			body = "()"
		}

		conformances := ""
		if kind == common.CompositeKindEnum {
			conformances = ": Int"
		}

		var baseType string
		if kind == common.CompositeKindAttachment {
			baseType = "for AnyStruct"
		}

		t.Run(kind.Keyword(), func(t *testing.T) {

			t.Parallel()

			_, err := ParseAndCheck(t,
				fmt.Sprintf(
					`
                      %[1]s T %[6]s %[2]s %[3]s

                      let test: fun(T): Void = fun (r: %[4]sT) {
                          %[5]s r
                      }
                    `,
					kind.Keyword(),
					conformances,
					body,
					kind.Annotation(),
					kind.DestructionKeyword(),
					baseType,
				),
			)

			switch kind {
			case common.CompositeKindResource:
				errs := RequireCheckerErrors(t, err, 1)

				assert.IsType(t, &sema.MissingResourceAnnotationError{}, errs[0])

			case common.CompositeKindAttachment:
				errs := RequireCheckerErrors(t, err, 2)

				assert.IsType(t, &sema.InvalidAttachmentAnnotationError{}, errs[0])
				assert.IsType(t, &sema.InvalidAttachmentAnnotationError{}, errs[1])

			case common.CompositeKindStructure,
				common.CompositeKindContract,
				common.CompositeKindEvent,
				common.CompositeKindEnum:

				require.NoError(t, err)

			default:
				panic(errors.NewUnreachableError())
			}
		})
	}

	for _, kind := range common.AllCompositeKinds {

		test(kind)
	}
}

func TestCheckFunctionTypeReturnTypeWithResourceAnnotation(t *testing.T) {

	t.Parallel()

	test := func(compositeKind common.CompositeKind) {

		body := "{}"
		switch compositeKind {
		case common.CompositeKindEvent:
			body = "()"
		case common.CompositeKindEnum:
			body = "{ case a }"
		}

		conformances := ""
		if compositeKind == common.CompositeKindEnum {
			conformances = ": Int"
		}

		var baseType string
		if compositeKind == common.CompositeKindAttachment {
			baseType = "for AnyStruct"
		}

		t.Run(compositeKind.Keyword(), func(t *testing.T) {

			t.Parallel()

			_, err := ParseAndCheck(t,
				fmt.Sprintf(
					`
                      %[1]s T %[7]s %[2]s %[3]s

                      let test: fun(): @T = fun (): @T {
                          return %[4]s %[5]s T%[6]s
                      }
                    `,
					compositeKind.Keyword(),
					conformances,
					body,
					compositeKind.MoveOperator(),
					compositeKind.ConstructionKeyword(),
					constructorArguments(compositeKind),
					baseType,
				),
			)

			switch compositeKind {
			case common.CompositeKindResource:
				require.NoError(t, err)

			case common.CompositeKindAttachment:
				errs := RequireCheckerErrors(t, err, 3)

				assert.IsType(t, &sema.InvalidAttachmentAnnotationError{}, errs[0])
				assert.IsType(t, &sema.InvalidAttachmentAnnotationError{}, errs[1])
				assert.IsType(t, &sema.InvalidAttachmentUsageError{}, errs[2])

			case common.CompositeKindStructure,
				common.CompositeKindContract,
				common.CompositeKindEnum:

				errs := RequireCheckerErrors(t, err, 2)

				assert.IsType(t, &sema.InvalidResourceAnnotationError{}, errs[0])
				assert.IsType(t, &sema.InvalidResourceAnnotationError{}, errs[1])

			case common.CompositeKindEvent:
				errs := RequireCheckerErrors(t, err, 3)

				assert.IsType(t, &sema.InvalidResourceAnnotationError{}, errs[0])
				assert.IsType(t, &sema.InvalidResourceAnnotationError{}, errs[1])
				assert.IsType(t, &sema.InvalidEventUsageError{}, errs[2])

			default:
				panic(errors.NewUnreachableError())
			}
		})
	}

	for _, compositeKind := range common.AllCompositeKinds {

		if compositeKind == common.CompositeKindContract {
			continue
		}

		test(compositeKind)
	}
}

func TestCheckFunctionTypeReturnTypeWithoutResourceAnnotation(t *testing.T) {

	t.Parallel()

	test := func(compositeKind common.CompositeKind) {

		body := "{}"
		switch compositeKind {
		case common.CompositeKindEvent:
			body = "()"
		case common.CompositeKindEnum:
			body = "{ case a }"
		}

		conformances := ""
		if compositeKind == common.CompositeKindEnum {
			conformances = ": Int"
		}

		var baseType string
		if compositeKind == common.CompositeKindAttachment {
			baseType = "for AnyStruct"
		}

		t.Run(compositeKind.Keyword(), func(t *testing.T) {

			t.Parallel()

			_, err := ParseAndCheck(t,
				fmt.Sprintf(
					`
                      %[1]s T %[7]s %[2]s %[3]s

                      let test: fun(): T = fun (): T {
                          return %[4]s %[5]s T%[6]s
                      }
                    `,
					compositeKind.Keyword(),
					conformances,
					body,
					compositeKind.MoveOperator(),
					compositeKind.ConstructionKeyword(),
					constructorArguments(compositeKind),
					baseType,
				),
			)

			switch compositeKind {
			case common.CompositeKindResource:
				errs := RequireCheckerErrors(t, err, 2)

				assert.IsType(t, &sema.MissingResourceAnnotationError{}, errs[0])
				assert.IsType(t, &sema.MissingResourceAnnotationError{}, errs[1])

			case common.CompositeKindAttachment:
				errs := RequireCheckerErrors(t, err, 3)

				assert.IsType(t, &sema.InvalidAttachmentAnnotationError{}, errs[0])
				assert.IsType(t, &sema.InvalidAttachmentAnnotationError{}, errs[1])
				assert.IsType(t, &sema.InvalidAttachmentUsageError{}, errs[2])

			case common.CompositeKindStructure,
				common.CompositeKindContract,
				common.CompositeKindEnum:

				require.NoError(t, err)

			case common.CompositeKindEvent:
				errs := RequireCheckerErrors(t, err, 1)

				assert.IsType(t, &sema.InvalidEventUsageError{}, errs[0])

			default:
				panic(errors.NewUnreachableError())
			}
		})
	}

	for _, compositeKind := range common.AllCompositeKinds {

		if compositeKind == common.CompositeKindContract {
			continue
		}

		test(compositeKind)
	}
}

func TestCheckFailableCastingWithoutResourceAnnotation(t *testing.T) {

	t.Parallel()

	test := func(compositeKind common.CompositeKind) {
		body := "{}"
		if compositeKind == common.CompositeKindEvent {
			body = "()"
		}

		t.Run(compositeKind.Keyword(), func(t *testing.T) {

			t.Parallel()

			_, err := ParseAndCheck(t,
				fmt.Sprintf(
					`
                      %[1]s T %[2]s

                      let test %[3]s %[4]s T%[5]s as? T
                    `,
					compositeKind.Keyword(),
					body,
					compositeKind.TransferOperator(),
					compositeKind.ConstructionKeyword(),
					constructorArguments(compositeKind),
				),
			)

			switch compositeKind {
			case common.CompositeKindResource:
				errs := RequireCheckerErrors(t, err, 3)

				assert.IsType(t, &sema.MissingResourceAnnotationError{}, errs[0])
				assert.IsType(t, &sema.InvalidFailableResourceDowncastOutsideOptionalBindingError{}, errs[1])
				assert.IsType(t, &sema.InvalidNonIdentifierFailableResourceDowncast{}, errs[2])

			case common.CompositeKindStructure,
				common.CompositeKindContract:

				require.NoError(t, err)

			case common.CompositeKindEvent:
				errs := RequireCheckerErrors(t, err, 1)

				assert.IsType(t, &sema.InvalidEventUsageError{}, errs[0])

			default:
				panic(errors.NewUnreachableError())
			}
		})
	}

	for _, compositeKind := range common.AllCompositeKinds {

		if compositeKind == common.CompositeKindEnum || compositeKind == common.CompositeKindAttachment {
			continue
		}

		test(compositeKind)
	}
}

func TestCheckUnaryMove(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource X {}

      fun foo(x: @X): @X {
          return <-x
      }

      fun bar() {
          let x <- foo(x: <-create X())
          destroy x
      }
    `)

	require.NoError(t, err)

}

func TestCheckImmediateDestroy(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource X {}

      fun test() {
          destroy create X()
      }
    `)

	require.NoError(t, err)
}

func TestCheckIndirectDestroy(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource X {}

      fun test() {
          let x <- create X()
          destroy x
      }
    `)

	require.NoError(t, err)
}

func TestCheckInvalidResourceCreationWithoutCreate(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource X {}

      let x <- X()
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.MissingCreateError{}, errs[0])

}

func TestCheckInvalidDestroy(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      struct X {}

      fun test() {
          destroy X()
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.InvalidDestructionError{}, errs[0])
}

func TestCheckUnaryCreateAndDestroy(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource X {}

      fun test() {
          var x <- create X()
          destroy x
      }
    `)

	require.NoError(t, err)
}

func TestCheckUnaryCreateAndDestroyWithInitializer(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource X {
          let x: Int
          init(x: Int) {
              self.x = x
          }
      }

      fun test() {
          var x <- create X(x: 1)
          destroy x
      }
    `)

	require.NoError(t, err)
}

func TestCheckInvalidUnaryCreateAndDestroyWithWrongInitializerArguments(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource X {
          let x: Int
          init(x: Int) {
              self.x = x
          }
      }

      fun test() {
          var x <- create X(y: true)
          destroy x
      }
    `)

	errs := RequireCheckerErrors(t, err, 2)

	assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
	assert.IsType(t, &sema.IncorrectArgumentLabelError{}, errs[1])
}

func TestCheckInvalidUnaryCreateStruct(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      struct X {}

      fun test() {
          create X()
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.InvalidConstructionError{}, errs[0])
}

func TestCheckInvalidCreateImportedResource(t *testing.T) {

	t.Parallel()

	importedChecker, err := ParseAndCheckWithOptions(t,
		`
          access(all) resource R {}
        `,
		ParseAndCheckOptions{
			Location: ImportedLocation,
		},
	)

	require.NoError(t, err)

	_, err = ParseAndCheckWithOptions(t,
		`
          import R from "imported"

          access(all) fun test() {
              destroy create R()
          }
        `,
		ParseAndCheckOptions{
			Config: &sema.Config{
				ImportHandler: func(_ *sema.Checker, _ common.Location, _ ast.Range) (sema.Import, error) {
					return sema.ElaborationImport{
						Elaboration: importedChecker.Elaboration,
					}, nil
				},
			},
		},
	)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.InvalidResourceCreationError{}, errs[0])
}

func TestCheckResourceCreationInContracts(t *testing.T) {

	t.Parallel()

	t.Run("in sibling contract", func(t *testing.T) {

		_, err := ParseAndCheck(t,
			`
              contract A {
                  resource R {}
              }

              contract B {

                  access(all) fun test() {
                      destroy create A.R()
                  }
              }
            `,
		)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidResourceCreationError{}, errs[0])
	})

	t.Run("in same contract", func(t *testing.T) {

		_, err := ParseAndCheck(t,
			`
              contract A {
                  resource R {}

                  access(all) fun test() {
                      destroy create R()
                  }
              }
            `,
		)

		require.NoError(t, err)
	})
}

func TestCheckInvalidResourceLoss(t *testing.T) {

	t.Parallel()

	t.Run("UnassignedResource", func(t *testing.T) {

		_, err := ParseAndCheck(t, `
            resource X {}

            fun test() {
                create X()
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.ResourceLossError{}, errs[0])
	})

	t.Run("ImmediateMemberAccess", func(t *testing.T) {

		_, err := ParseAndCheck(t, `
            resource Foo {
                fun bar(): Int {
                    return 42
                }
            }

            fun test() {
                let x = (create Foo()).bar()
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.ResourceLossError{}, errs[0])
	})

	t.Run("ImmediateMemberAccessFunctionInvocation", func(t *testing.T) {

		_, err := ParseAndCheck(t, `
            resource Foo {
                fun bar(): Int {
                    return 42
                }
            }

            fun createResource(): @Foo {
                return <-create Foo()
            }

            fun test() {
                let x = createResource().bar()
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.ResourceLossError{}, errs[0])
	})

	t.Run("ImmediateIndexing", func(t *testing.T) {

		_, err := ParseAndCheck(t, `
            resource Foo {}

            fun test() {
                let x <- [<-create Foo(), <-create Foo()][0]
                destroy x
            }
        `)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.ResourceLossError{}, errs[0])
		assert.IsType(t, &sema.InvalidNestedResourceMoveError{}, errs[1])
	})

	t.Run("ImmediateIndexingFunctionInvocation", func(t *testing.T) {

		_, err := ParseAndCheck(t, `
            resource Foo {}

            fun test() {
                let x <- makeFoos()[0]
                destroy x
            }

            fun makeFoos(): @[Foo] {
                return <-[
                    <-create Foo(),
                    <-create Foo()
                ]
            }
        `)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.ResourceLossError{}, errs[0])
		assert.IsType(t, &sema.InvalidNestedResourceMoveError{}, errs[1])
	})

	t.Run("ImmediateComparisonOptionalNil", func(t *testing.T) {

		_, err := ParseAndCheck(t, `
            resource Foo {}

            access(all) fun foo(): @Foo? {
                return <- create Foo()
            }

            access(all) let isNil = foo() == nil
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.ResourceLossError{}, errs[0])
	})

	t.Run("ImmediateComparisonArray", func(t *testing.T) {

		_, err := ParseAndCheck(t, `
            resource Foo {}

            let empty: @[Foo] <- []
            let isEmpty = [<- create Foo()] == empty
        `)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.ResourceLossError{}, errs[1])
	})

	t.Run("Optional chaining", func(t *testing.T) {

		_, err := ParseAndCheck(t, `
            resource R {}

            struct S {
                fun use(_ r: @R) {
                    destroy r
                }
            }

            fun test() {
                let r <- create R()
                let s: S? = S()
                s?.use(<-r)
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.ResourceLossError{}, errs[0])
	})

}

func TestCheckResourceReturn(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource X {}

      fun test(): @X {
          return <-create X()
      }
    `)

	require.NoError(t, err)
}

func TestCheckInvalidResourceReturnMissingMove(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource X {}

      fun test(): @X {
          return create X()
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.MissingMoveOperationError{}, errs[0])
}

func TestCheckInvalidResourceReturnMissingMoveInvalidReturnType(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource X {}

      fun test(): Y {
          return create X()
      }
    `)

	errs := RequireCheckerErrors(t, err, 2)

	assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
	assert.IsType(t, &sema.MissingMoveOperationError{}, errs[1])
}

func TestCheckInvalidNonResourceReturnWithMove(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      struct X {}

      fun test(): X {
          return <-X()
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.InvalidMoveOperationError{}, errs[0])
}

func TestCheckResourceArgument(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource X {}

      fun foo(_ x: @X) {
          destroy x
      }

      fun bar() {
          foo(<-create X())
      }
    `)

	require.NoError(t, err)
}

func TestCheckInvalidResourceArgumentMissingMove(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource X {}

      fun foo(_ x: @X) {
          destroy x
      }

      fun bar() {
          foo(create X())
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.MissingMoveOperationError{}, errs[0])
}

func TestCheckInvalidResourceArgumentMissingMoveInvalidParameterType(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource X {}

      fun foo(_ x: Y) {}

      fun bar() {
          foo(create X())
      }
    `)

	errs := RequireCheckerErrors(t, err, 2)

	assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
	assert.IsType(t, &sema.MissingMoveOperationError{}, errs[1])
}

func TestCheckInvalidNonResourceArgumentWithMove(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      struct X {}

      fun foo(_ x: X) {}

      fun bar() {
          foo(<-X())
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.InvalidMoveOperationError{}, errs[0])
}

func TestCheckResourceVariableDeclarationTransfer(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource X {}

      let x <- create X()
      let y <- x
    `)

	require.NoError(t, err)
}

func TestCheckInvalidResourceVariableDeclarationIncorrectTransfer(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource X {}

      let x = create X()
      let y = x
    `)

	errs := RequireCheckerErrors(t, err, 2)

	assert.IsType(t, &sema.IncorrectTransferOperationError{}, errs[0])
	assert.IsType(t, &sema.IncorrectTransferOperationError{}, errs[1])
}

func TestCheckInvalidNonResourceVariableDeclarationMoveTransfer(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      struct X {}

      let x = X()
      let y <- x
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.IncorrectTransferOperationError{}, errs[0])
}

func TestCheckInvalidResourceAssignmentTransfer(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource X {}

      fun test() {
          let x <- create X()
          var x2 <- create X()
          destroy x2
          x2 <- x
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.InvalidResourceAssignmentError{}, errs[0])
}

func TestCheckInvalidResourceAssignmentIncorrectTransfer(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource X {}

      fun test() {
          let x <- create X()
          var x2 <- create X()
          destroy x2
          x2 = x
      }
    `)

	errs := RequireCheckerErrors(t, err, 2)

	assert.IsType(t, &sema.IncorrectTransferOperationError{}, errs[0])
	assert.IsType(t, &sema.InvalidResourceAssignmentError{}, errs[1])
}

func TestCheckInvalidNonResourceAssignmentMoveTransfer(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      struct X {}

      let x = X()
      fun test() {
        var x2 = X()
        x2 <- x
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.IncorrectTransferOperationError{}, errs[0])
}

func TestCheckResourceAssignmentForceTransfer(t *testing.T) {

	t.Parallel()

	t.Run("new to nil", func(t *testing.T) {

		_, err := ParseAndCheck(t, `
          resource X {}

          fun test() {
              var x: @X? <- nil
              x <-! create X()
              destroy x
          }
        `)

		require.NoError(t, err)
	})

	t.Run("new to non-nil", func(t *testing.T) {

		_, err := ParseAndCheck(t, `
          resource X {}

          fun test() {
              var x: @X? <- create X()
              x <-! create X()
              destroy x
          }
        `)

		require.NoError(t, err)
	})

	t.Run("existing to nil", func(t *testing.T) {

		_, err := ParseAndCheck(t, `
          resource X {}

          fun test() {
              let x <- create X()
              var x2: @X? <- nil
              x2 <-! x
              destroy x2
          }
        `)

		require.NoError(t, err)
	})

	t.Run("existing to non-nil", func(t *testing.T) {

		_, err := ParseAndCheck(t, `
          resource X {}

          fun test() {
              let x <- create X()
              var x2: @X? <- create X()
              x2 <-! x
              destroy x2
          }
        `)

		require.NoError(t, err)
	})

	t.Run("to non-optional", func(t *testing.T) {

		_, err := ParseAndCheck(t, `
          resource X {}

          fun test() {
              let x <- create X()
              var x2 <- create X()
              destroy x2
              x2 <-! x
          }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidResourceAssignmentError{}, errs[0])
	})
}

func TestCheckInvalidResourceLossThroughVariableDeclaration(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource X {}

      fun test() {
        let x <- create X()
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.ResourceLossError{}, errs[0])
}

func TestCheckInvalidResourceLossThroughVariableDeclarationAfterCreation(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource X {}

      fun test() {
          let x <- create X()
          let y <- x
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.ResourceLossError{}, errs[0])
}

func TestCheckInvalidResourceLossThroughAssignment(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource X {}

      fun test() {
          var x <- create X()
          let y <- create X()
          x <- y
      }
    `)

	errs := RequireCheckerErrors(t, err, 2)

	assert.IsType(t, &sema.InvalidResourceAssignmentError{}, errs[0])
	assert.IsType(t, &sema.ResourceLossError{}, errs[1])
}

func TestCheckResourceMoveThroughReturn(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource X {}

      fun test(): @X {
          let x <- create X()
          return <-x
      }
    `)

	require.NoError(t, err)
}

func TestCheckResourceMoveThroughArgumentPassing(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource X {}

      fun test() {
          let x <- create X()
          absorb(<-x)
      }

      fun absorb(_ x: @X) {
          destroy x
      }
    `)

	require.NoError(t, err)
}

func TestCheckInvalidResourceUseAfterMoveToFunction(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource X {}

      fun test() {
          let x <- create X()
          absorb(<-x)
          absorb(<-x)
      }

      fun absorb(_ x: @X) {
          destroy x
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
}

func TestCheckInvalidResourceUseAfterMoveToVariable(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource X {}

      fun test() {
          let x <- create X()
          let y <- x
          let z <- x
      }
    `)

	errs := RequireCheckerErrors(t, err, 3)

	assert.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])

	// NOTE: still two resource losses reported for `y` and `z`

	assert.IsType(t, &sema.ResourceLossError{}, errs[1])

	assert.IsType(t, &sema.ResourceLossError{}, errs[2])
}

func TestCheckInvalidResourceFieldUseAfterMoveToVariable(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource X {
          let id: Int
          init(id: Int) {
              self.id = id
          }
      }

      fun test(): Int {
          let x <- create X(id: 1)
          absorb(<-x)
          return x.id
      }

      fun absorb(_ x: @X) {
          destroy x
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
}

func TestCheckResourceUseAfterMoveInIfStatementThenBranch(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource X {}

      fun test() {
          let x <- create X()
          if 1 > 2 {
              absorb(<-x)
          }
          absorb(<-x)
      }

      fun absorb(_ x: @X) {
          destroy x
      }
    `)

	errs := RequireCheckerErrors(t, err, 2)

	assert.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
	assert.IsType(t, &sema.ResourceLossError{}, errs[1])
}

func TestCheckResourceUseInIfStatement(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource X {}

      fun test() {
          let x <- create X()
          if 1 > 2 {
              absorb(<-x)
          } else {
              absorb(<-x)
          }
      }

      fun absorb(_ x: @X) {
          destroy x
      }
    `)

	require.NoError(t, err)
}

func TestCheckResourceUseInNestedIfStatement(t *testing.T) {

	t.Parallel()

	t.Run("resource loss", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          resource X {}

          fun test() {
              let x <- create X()
              if 1 > 2 {
                  if 2 > 1 {
                      absorb(<-x)
                  }
                  // NOTE: resource is not destroyed in the else path
              } else {
                  absorb(<-x)
              }
          }

          fun absorb(_ x: @X) {
              destroy x
          }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.ResourceLossError{}, errs[0])
	})

	t.Run("no resource loss", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          resource X {}

          fun test() {
              let x <- create X()
              if 1 > 2 {
                  if 2 > 1 {
                      absorb(<-x)
                  } else {
                      absorb(<-x)
                  }
              } else {
                  absorb(<-x)
              }
          }

          fun absorb(_ x: @X) {
              destroy x
          }
        `)

		require.NoError(t, err)
	})
}

////

func TestCheckInvalidResourceUseAfterIfStatement(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource X {}

      fun test(): @X {
          let x <- create X()
          if 1 > 2 {
              absorb(<-x)
          } else {
              absorb(<-x)
          }
          return <-x
      }

      fun absorb(_ x: @X) {
          destroy x
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])

	assert.Equal(t,
		sema.ResourceInvalidation{
			Kind:     sema.ResourceInvalidationKindMoveDefinite,
			StartPos: ast.Position{Offset: 119, Line: 7, Column: 23},
			EndPos:   ast.Position{Offset: 119, Line: 7, Column: 23},
		},
		errs[0].(*sema.ResourceUseAfterInvalidationError).Invalidation,
	)
}

func TestCheckInvalidResourceLossAfterDestroyInIfStatementThenBranch(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource X {}

      fun test() {
          let x <- create X()
          if 1 > 2 {
             destroy x
          }
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.ResourceLossError{}, errs[0])
}

func TestCheckInvalidResourceLossAndUseAfterDestroyInIfStatementThenBranch(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource X {
          let id: Int
          init(id: Int) {
              self.id = id
          }
      }

      fun test() {
          let x <- create X(id: 1)
          if 1 > 2 {
             destroy x
          }
          x.id
      }
    `)

	errs := RequireCheckerErrors(t, err, 2)

	assert.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
	assert.IsType(t, &sema.ResourceLossError{}, errs[1])
}

func TestCheckResourceMoveIntoArray(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource X {}

      let x <- create X()
      let xs <- [<-x]
    `)

	require.NoError(t, err)
}

func TestCheckInvalidResourceMoveIntoArrayMissingMoveOperation(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource X {}

      let x <- create X()
      let xs <- [x]
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.MissingMoveOperationError{}, errs[0])
}

func TestCheckInvalidNonResourceMoveIntoArray(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      struct X {}

      let x = X()
      let xs = [<-x]
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.InvalidMoveOperationError{}, errs[0])
}

func TestCheckInvalidUseAfterResourceMoveIntoArray(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource X {}

      let x <- create X()
      let xs <- [<-x, <-x]
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
}

func TestCheckResourceMoveIntoDictionary(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource X {}

      let x <- create X()
      let xs <- {"x": <-x}
    `)

	require.NoError(t, err)
}

func TestCheckInvalidResourceMoveIntoDictionaryMissingMoveOperation(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource X {}

      let x <- create X()
      let xs <- {"x": x}
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.MissingMoveOperationError{}, errs[0])
}

func TestCheckInvalidNonResourceMoveIntoDictionary(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      struct X {}

      let x = X()
      let xs = {"x": <-x}
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.InvalidMoveOperationError{}, errs[0])
}

func TestCheckInvalidUseAfterResourceMoveIntoDictionary(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource X {}

      let x <- create X()
      let xs <- {
          "x": <-x,
          "x2": <-x
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
}

func TestCheckInvalidUseAfterResourceMoveIntoDictionaryAsKey(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource X {}

      let x <- create X()
      let xs <- {<-x: <-x}
    `)

	errs := RequireCheckerErrors(t, err, 2)

	assert.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
	assert.IsType(t, &sema.InvalidDictionaryKeyTypeError{}, errs[1])
}

func TestCheckInvalidResourceDestroyAfterMoveInWhileStatement(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource X {}

      fun f(_ x: @X) {
          destroy x
      }

      fun test() {
          let x <- create X()
          while true {
              f(<-x)
          }
          destroy x
      }
    `)

	errs := RequireCheckerErrors(t, err, 2)

	assert.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
	assert.IsType(t, &sema.ResourceLossError{}, errs[1])
}

func TestCheckInvalidResourceDestroyAfterDestroyInWhileStatement(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource X {}

      fun test() {
          let x <- create X()
          while true {
              destroy x
          }
          destroy x
      }
    `)

	errs := RequireCheckerErrors(t, err, 2)

	assert.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
	assert.IsType(t, &sema.ResourceLossError{}, errs[1])
}

func TestCheckResourceUseInWhileStatement(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource X {
          let id: Int
          init(id: Int) {
              self.id = id
          }
      }

      fun test() {
          let x <- create X(id: 1)
          while true {
              x.id
          }
          destroy x
      }
    `)

	require.NoError(t, err)
}

func TestCheckInvalidResourceUseInWhileStatementAfterDestroy(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource X {
          let id: Int
          init(id: Int) {
              self.id = id
          }
      }

      fun test() {
          let x <- create X(id: 1)
          while true {
              x.id
              destroy x
          }
          destroy x
      }
    `)

	errs := RequireCheckerErrors(t, err, 2)

	assert.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
	assert.IsType(t, &sema.ResourceLossError{}, errs[1])
}

func TestCheckInvalidResourceUseInWhileStatementAfterDestroyAndLoss(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource X {}

      fun test() {
          let x <- create X()
          while true {
              destroy x
          }
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.ResourceLossError{}, errs[0])
}

func TestCheckInvalidResourceUseInNestedWhileStatementAfterDestroyAndLoss1(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource X {
          let id: Int
          init(id: Int) {
              self.id = id
          }
      }

      fun test() {
          let x <- create X(id: 1)
          while true {
              while true {
                  x.id
                  destroy x
              }
          }
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.ResourceLossError{}, errs[0])
}

func TestCheckInvalidResourceUseInNestedWhileStatementAfterDestroyAndLoss2(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource X {
          let id: Int
          init(id: Int) {
              self.id = id
          }
      }

      fun test() {
          let x <- create X(id: 1)
          while true {
              while true {
                  x.id
              }
              destroy x
          }
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.ResourceLossError{}, errs[0])
}

func TestCheckResourceUseInNestedWhileStatement(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource X {
          let id: Int
          init(id: Int) {
              self.id = id
          }
      }

      fun test() {
          let x <- create X(id: 1)
          while true {
              while true {
                  x.id
              }
          }
          destroy x
      }
    `)

	require.NoError(t, err)
}

func TestCheckInvalidResourceLossThroughReturn(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource X {}

      fun test() {
          let x <- create X()
          return
          destroy x
      }
    `)

	errs := RequireCheckerErrors(t, err, 2)

	assert.IsType(t, &sema.ResourceLossError{}, errs[0])
	assert.IsType(t, &sema.UnreachableStatementError{}, errs[1])
}

func TestCheckInvalidResourceLossThroughReturnInIfStatementThenBranch(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource X {}

      fun test(y: Int) {
          let x <- create X()
          if y == 42 {
              return
          }
          destroy x
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.ResourceLossError{}, errs[0])
}

func TestCheckInvalidResourceLossThroughReturnInIfStatementBranches(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource X {}

      fun test(y: Int) {
          let x <- create X()
          if y == 42 {
              absorb(<-x)
              return
          } else {
              return
          }
          destroy x
      }

      fun absorb(_ x: @X) {
          destroy x
      }
    `)

	errs := RequireCheckerErrors(t, err, 2)

	assert.IsType(t, &sema.ResourceLossError{}, errs[0])
	assert.IsType(t, &sema.UnreachableStatementError{}, errs[1])
}

func TestCheckResourceWithMoveAndReturnInIfStatementThenAndDestroyInElse(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource X {}

      fun test(y: Int) {
          let x <- create X()
          if y == 42 {
              absorb(<-x)
              return
          } else {
              destroy x
          }
      }

      fun absorb(_ x: @X) {
          destroy x
      }
    `)

	require.NoError(t, err)
}

func TestCheckResourceWithMoveAndReturnInIfStatementThenBranch(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource X {}

      fun test(y: Int) {
          let x <- create X()
          if y == 42 {
              absorb(<-x)
              return
          }
          destroy x
      }

      fun absorb(_ x: @X) {
          destroy x
      }
    `)

	require.NoError(t, err)
}

func TestCheckResourceWithMoveAndReturnInWhileStatement(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `

          resource X {}

          fun test() {
              let x <- create X()
              while true {
                  absorb(<-x)
                  return
              }
              destroy x
          }

          fun absorb(_ x: @X) {
              destroy x
          }
        `)

	require.NoError(t, err)
}

func TestCheckResourceNesting(t *testing.T) {

	t.Parallel()

	interfacePossibilities := []bool{true, false}

	for _, innerCompositeKind := range common.AllCompositeKinds {

		// Don't test contract fields/parameters: contracts can't be passed by value
		if innerCompositeKind == common.CompositeKindContract {
			continue
		}

		for _, innerIsInterface := range interfacePossibilities {

			if !innerCompositeKind.SupportsInterfaces() && innerIsInterface || innerCompositeKind == common.CompositeKindAttachment {
				continue
			}

			for _, outerCompositeKind := range common.InstantiableCompositeKindsWithFieldsAndFunctions {
				for _, outerIsInterface := range interfacePossibilities {

					if !outerCompositeKind.SupportsInterfaces() && outerIsInterface {
						continue
					}

					testResourceNesting(
						t,
						innerCompositeKind,
						innerIsInterface,
						outerCompositeKind,
						outerIsInterface,
					)
				}
			}
		}
	}
}

func testResourceNesting(
	t *testing.T,
	innerCompositeKind common.CompositeKind,
	innerIsInterface bool,
	outerCompositeKind common.CompositeKind,
	outerIsInterface bool,
) {
	innerInterfaceKeyword := ""
	if innerIsInterface {
		innerInterfaceKeyword = "interface"
	}

	outerInterfaceKeyword := ""
	if outerIsInterface {
		outerInterfaceKeyword = "interface"
	}

	testName := fmt.Sprintf(
		"%s %s in %s %s",
		innerCompositeKind.Keyword(),
		innerInterfaceKeyword,
		outerCompositeKind.Keyword(),
		outerInterfaceKeyword,
	)

	t.Run(testName, func(t *testing.T) {

		innerTypeAnnotation := "T"
		if innerIsInterface {
			innerTypeAnnotation = AsInterfaceType("T", innerCompositeKind)
		}

		// Prepare the initializer, if needed.
		// `outerCompositeKind` is the container composite kind.
		// If it is concrete, i.e. not an interface, it needs an initializer.

		initializer := ""
		if !outerIsInterface {
			initializer = fmt.Sprintf(
				`
                  init(t: %[1]s%[2]s) {
                      self.t %[3]s t
                  }
                `,
				innerCompositeKind.Annotation(),
				innerTypeAnnotation,
				innerCompositeKind.TransferOperator(),
			)
		}

		destructor := ""
		if !outerIsInterface &&
			outerCompositeKind == common.CompositeKindResource &&
			innerCompositeKind == common.CompositeKindResource {

			destructor = `
              destroy() {
                  destroy self.t
              }
            `
		}

		innerBody := "{}"
		if innerCompositeKind == common.CompositeKindEvent {
			innerBody = "()"
		}

		innerConformances := ""
		if innerCompositeKind == common.CompositeKindEnum {
			innerConformances = ": Int"
		}

		// Prepare the full program defining an empty composite,
		// and a second composite which contains the first

		program := fmt.Sprintf(
			`
              %[1]s %[2]s T%[10]s %[3]s

              %[4]s %[5]s U {
                  let t: %[6]s%[7]s
                  %[8]s
                  %[9]s
              }
            `,
			innerCompositeKind.Keyword(),
			innerInterfaceKeyword,
			innerBody,
			outerCompositeKind.Keyword(),
			outerInterfaceKeyword,
			innerCompositeKind.Annotation(),
			innerTypeAnnotation,
			initializer,
			destructor,
			innerConformances,
		)

		_, err := ParseAndCheck(t, program)

		switch outerCompositeKind {
		case common.CompositeKindStructure:

			switch innerCompositeKind {
			case common.CompositeKindStructure,
				common.CompositeKindEvent,
				common.CompositeKindEnum:

				require.NoError(t, err)

			case common.CompositeKindResource:
				errs := RequireCheckerErrors(t, err, 1)

				assert.IsType(t, &sema.InvalidResourceFieldError{}, errs[0])

			default:
				panic(errors.NewUnreachableError())
			}

		case common.CompositeKindResource,
			common.CompositeKindEnum:

			require.NoError(t, err)

		case common.CompositeKindContract:

			if innerCompositeKind == common.CompositeKindEvent {
				errs := RequireCheckerErrors(t, err, 1)

				assert.IsType(t, &sema.FieldTypeNotStorableError{}, errs[0])
			} else {
				require.NoError(t, err)
			}

		default:
			panic(errors.NewUnreachableError())
		}
	})
}

func TestCheckContractResourceField(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource R {}

      contract C {
          let r: @R

          init(r: @R) {
              self.r <- r
          }
      }
    `)

	require.NoError(t, err)
}

func TestCheckInvalidContractResourceFieldMove(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource R {}

      contract C {
          let r: @R

          init(r: @R) {
              self.r <- r
          }
      }

      fun test() {
          let r <- C.r
          destroy r
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.InvalidNestedResourceMoveError{}, errs[0])
}

func TestCheckInvalidEnumResourceField(t *testing.T) {

	t.Parallel()

	t.Run("raw type given", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
	      resource R {}

          enum E: R {}
	    `)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.InvalidEnumRawTypeError{}, errs[0])
	})

	t.Run("raw type given, nested", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
	      resource R {
              enum E: R {}
          }
	    `)

		errs := RequireCheckerErrors(t, err, 2)

		require.IsType(t, &sema.InvalidNestedDeclarationError{}, errs[0])
		require.IsType(t, &sema.InvalidEnumRawTypeError{}, errs[1])
	})

	t.Run("raw type not given", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
	      enum E {}
	    `)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.MissingEnumRawTypeError{}, errs[0])
	})
}

// TestCheckResourceInterfaceConformance tests the check
// of conformance of resources to resource interfaces.
func TestCheckResourceInterfaceConformance(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource interface X {
          fun test()
      }

      resource Y: X {
          fun test() {}
      }
    `)

	require.NoError(t, err)
}

// TestCheckInvalidResourceInterfaceConformance tests the check
// of conformance of resources to resource interfaces.
func TestCheckInvalidResourceInterfaceConformance(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource interface X {
          fun test()
      }

      resource Y: X {}
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.ConformanceError{}, errs[0])
}

// TestCheckInvalidResourceInterfaceUseAsType tests that a resource interface
// can not be used as a type
func TestCheckInvalidResourceInterfaceUseAsType(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource interface I {}

      resource R: I {}

      let r: @I <- create R()
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.InvalidInterfaceTypeError{}, errs[0])
}

// TestCheckResourceInterfaceUseAsType test if a resource
// is a subtype of a restricted AnyResource type.
func TestCheckResourceInterfaceUseAsType(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource interface I {}

      resource R: I {}

      let r: @{I} <- create R()
    `)

	require.NoError(t, err)
}

func TestCheckResourceArrayIndexing(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource Foo {
          var bar: Int
          init(bar: Int) {
              self.bar = bar
          }
      }

      fun test(): Int {
        let foo <- create Foo(bar: 1)
        let foos <- [<-[<-foo]]
        let bar = foos[0][0].bar
        destroy foos
        return bar
      }
    `)

	require.NoError(t, err)
}

func TestCheckInvalidResourceLossReturnResourceAndMemberAccess(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource X {
          let id: Int

          init(id: Int) {
              self.id = id
          }
      }

      fun test(): Int {
          return createX().id
      }

      fun createX(): @X {
          return <-create X(id: 1)
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.ResourceLossError{}, errs[0])
}

func TestCheckInvalidResourceLossAfterMoveThroughArrayIndexing(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource X {}

      fun test() {
          let xs <- [<-create X()]
          foo(x: <-xs[0])
      }

      fun foo(x: @X) {
          destroy x
      }
    `)

	errs := RequireCheckerErrors(t, err, 2)
	assert.IsType(t, &sema.InvalidNestedResourceMoveError{}, errs[0])
	assert.IsType(t, &sema.ResourceLossError{}, errs[1])
}

func TestCheckInvalidResourceLossThroughFunctionResultAccess(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource Foo {
          var bar: Int
          init(bar: Int) {
              self.bar = bar
          }
      }

      fun createFoo(): @Foo {
          return <- create Foo(bar: 1)
      }

      fun test(): Int {
          return createFoo().bar
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.ResourceLossError{}, errs[0])
}

// TestCheckAnyResourceDestruction tests if resources
// can be passed to restricted AnyResources parameters,
// and if the argument can be destroyed.
func TestCheckAnyResourceDestruction(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource interface I {}

      resource R: I {}

      fun foo(_ i: @{I}) {
          destroy i
      }

      fun bar() {
          foo(<-create R())
      }
    `)

	require.NoError(t, err)
}

// TestCheckInvalidResourceFieldMoveThroughVariableDeclaration tests if resources nested
// as a field in another resource cannot be moved out of the containing resource through
// a variable declaration. This would partially invalidate the containing resource
func TestCheckInvalidResourceFieldMoveThroughVariableDeclaration(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource Foo {}

      resource Bar {
          let foo: @Foo

          init(foo: @Foo) {
              self.foo <- foo
          }

          destroy() {
              destroy self.foo
          }
      }

      fun test(): @[Foo] {
          let foo <- create Foo()
          let bar <- create Bar(foo: <-foo)
          let foo2 <- bar.foo
          let foo3 <- bar.foo
          destroy bar
          return <-[<-foo2, <-foo3]
      }
    `)

	errs := RequireCheckerErrors(t, err, 2)

	assert.IsType(t, &sema.InvalidNestedResourceMoveError{}, errs[0])
	assert.IsType(t, &sema.InvalidNestedResourceMoveError{}, errs[1])
}

// TestCheckInvalidResourceFieldMoveThroughParameter tests if resources nested
// as a field in another resource cannot be moved out of the containing resource
// by passing the field as an argument to a function. This would partially invalidate
// the containing resource
func TestCheckInvalidResourceFieldMoveThroughParameter(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource Foo {}

      resource Bar {
          let foo: @Foo

          init(foo: @Foo) {
              self.foo <- foo
          }

          destroy() {
              destroy self.foo
          }
      }

      fun identity(_ foo: @Foo): @Foo {
          return <-foo
      }

      fun test(): @[Foo] {
          let foo <- create Foo()
          let bar <- create Bar(foo: <-foo)
          let foo2 <- identity(<-bar.foo)
          let foo3 <- identity(<-bar.foo)
          destroy bar
          return <-[<-foo2, <-foo3]
      }
    `)

	errs := RequireCheckerErrors(t, err, 2)

	assert.IsType(t, &sema.InvalidNestedResourceMoveError{}, errs[0])
	assert.IsType(t, &sema.InvalidNestedResourceMoveError{}, errs[1])
}

func TestCheckInvalidResourceFieldMoveSelf(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource Y {}

      resource X {

          var y: @Y

          init() {
              self.y <- create Y()
          }

          fun test() {
             absorb(<-self.y)
          }

          destroy() {
              destroy self.y
          }
      }

      fun absorb(_ y: @Y) {
          destroy y
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.InvalidNestedResourceMoveError{}, errs[0])
}

func TestCheckInvalidResourceFieldUseAfterDestroy(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource Y {}

      resource X {

          var y: @Y

          init() {
              self.y <- create Y()
          }

          destroy() {
              destroy self.y
              destroy self.y
          }
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
}

func TestCheckResourceArrayAppend(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource X {}

      fun test() {
          let xs: @[X] <- []
          xs.append(<-create X())
          destroy xs
      }
    `)

	require.NoError(t, err)
}

func TestCheckResourceArrayInsert(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource X {}

      fun test() {
          let xs: @[X] <- []
          xs.insert(at: 0, <-create X())
          destroy xs
      }
    `)

	require.NoError(t, err)
}

func TestCheckResourceArrayRemove(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource X {}

      fun test() {
          let xs: @[X] <- [<-create X()]
          let x <- xs.remove(at: 0)
          destroy x
          destroy xs
      }
    `)

	require.NoError(t, err)
}

func TestCheckInvalidResourceArrayRemoveResourceLoss(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource X {}

      fun test() {
          let xs: @[X] <- [<-create X()]
          xs.remove(at: 0)
          destroy xs
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.ResourceLossError{}, errs[0])
}

func TestCheckResourceArrayRemoveFirst(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource X {}

      fun test() {
          let xs: @[X] <- [<-create X()]
          let x <- xs.removeFirst()
          destroy x
          destroy xs
      }
    `)

	require.NoError(t, err)
}

func TestCheckResourceArrayRemoveLast(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource X {}

      fun test() {
          let xs: @[X] <- [<-create X()]
          let x <- xs.removeLast()
          destroy x
          destroy xs
      }
    `)

	require.NoError(t, err)
}

func TestCheckInvalidResourceArrayContains(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource X {}

      fun test() {
          let xs: @[X] <- [<-create X()]
          xs.contains(<-create X())
          destroy xs
      }
    `)

	errs := RequireCheckerErrors(t, err, 2)

	assert.IsType(t, &sema.InvalidResourceArrayMemberError{}, errs[0])
	assert.IsType(t, &sema.NotEquatableTypeError{}, errs[1])
}

func TestCheckResourceArrayLength(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource X {}

      fun test(): Int {
          let xs: @[X] <- [<-create X()]
          let count = xs.length
          destroy xs
          return count
      }
    `)

	require.NoError(t, err)
}

func TestCheckInvalidResourceArrayConcat(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource X {}

      fun test() {
          let xs: @[X] <- [<-create X()]
          let xs2 <- [<-create X()]
          let xs3 <- xs.concat(<-xs2)
          destroy xs
          destroy xs3
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.InvalidResourceArrayMemberError{}, errs[0])
}

func TestCheckResourceDictionaryRemove(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource X {}

      fun test() {
          let xs: @{String: X} <- {"x1": <-create X()}
          let x <- xs.remove(key: "x1")
          destroy x
          destroy xs
      }
    `)

	require.NoError(t, err)
}

func TestCheckInvalidResourceDictionaryRemoveResourceLoss(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource X {}

      fun test() {
          let xs: @{String: X} <- {"x1": <-create X()}
          xs.remove(key: "x1")
          destroy xs
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.ResourceLossError{}, errs[0])
}

func TestCheckResourceDictionaryInsert(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource X {}

      fun test() {
          let xs: @{String: X} <- {}
          let old <- xs.insert(key: "x1", <-create X())
          destroy old
          destroy xs
      }
    `)

	require.NoError(t, err)
}

func TestCheckInvalidResourceDictionaryInsertResourceLoss(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource X {}

      fun test() {
          let xs: @{String: X} <- {}
          xs.insert(key: "x1", <-create X())
          destroy xs
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.ResourceLossError{}, errs[0])
}

func TestCheckResourceDictionaryLength(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource X {}

      fun test(): Int {
          let xs: @{String: X} <- {"x1": <-create X()}
          let count = xs.length
          destroy xs
          return count
      }
    `)

	require.NoError(t, err)
}

func TestCheckInvalidResourceDictionaryKeys(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource X {}

      fun test() {
          let xs <- {<-create X(): "x1"}
          let keys <- xs.keys
          destroy keys
          destroy xs
      }
    `)

	errs := RequireCheckerErrors(t, err, 3)

	assert.IsType(t, &sema.InvalidDictionaryKeyTypeError{}, errs[0])
	assert.IsType(t, &sema.InvalidResourceDictionaryMemberError{}, errs[1])
	assert.IsType(t, &sema.InvalidNestedResourceMoveError{}, errs[2])
}

func TestCheckInvalidResourceDictionaryValues(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource X {}

      fun test() {
          let xs <- {"x1": <-create X()}
          let values <- xs.values
          destroy values
          destroy xs
      }
    `)

	errs := RequireCheckerErrors(t, err, 2)

	assert.IsType(t, &sema.InvalidResourceDictionaryMemberError{}, errs[0])
	assert.IsType(t, &sema.InvalidNestedResourceMoveError{}, errs[1])
}

func TestCheckInvalidResourceDictionaryKeysForeach(t *testing.T) {
	t.Parallel()

	_, err := ParseAndCheck(t, `
        resource X {}

        fun test() {
            let xs <- {<-create X(): "x1"}

            xs.forEachKey(fun (x: @X): Bool {
                destroy x
                return true
            }) 
            destroy xs
        }
    `)

	errs := RequireCheckerErrors(t, err, 3)

	assert.IsType(t, &sema.InvalidDictionaryKeyTypeError{}, errs[0])
	assert.IsType(t, &sema.InvalidResourceDictionaryMemberError{}, errs[1])
	assert.IsType(t, &sema.ResourceLossError{}, errs[2])
}

func TestCheckInvalidResourceLossAfterMoveThroughDictionaryIndexing(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource X {}

      fun test() {
          let xs <- {"x": <-create X()}
          foo(x: <-xs["x"])
      }

      fun foo(x: @X?) {
          destroy x
      }
    `)

	errs := RequireCheckerErrors(t, err, 2)
	assert.IsType(t, &sema.InvalidNestedResourceMoveError{}, errs[0])
	assert.IsType(t, &sema.ResourceLossError{}, errs[1])
}

func TestCheckInvalidResourceSwap(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource X {}

      fun test() {
         var x <- create X()
         x <-> create X()
         destroy x
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.InvalidSwapExpressionError{}, errs[0])
}

func TestCheckInvalidResourceConstantResourceFieldSwap(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource Foo {}

      resource Bar {
          let foo: @Foo

          init(foo: @Foo) {
              self.foo <- foo
          }

          destroy() {
              destroy self.foo
          }
      }

      fun test() {
          let foo <- create Foo()
          let bar <- create Bar(foo: <-foo)
          var foo2 <- create Foo()
          bar.foo <-> foo2
          destroy bar
          destroy foo2
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.AssignmentToConstantMemberError{}, errs[0])
}

func TestCheckResourceVariableResourceFieldSwap(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource Foo {}

      resource Bar {
          var foo: @Foo

          init(foo: @Foo) {
              self.foo <- foo
          }

          destroy() {
              destroy self.foo
          }
      }

      fun test() {
          let foo <- create Foo()
          let bar <- create Bar(foo: <-foo)
          var foo2 <- create Foo()
          bar.foo <-> foo2
          destroy bar
          destroy foo2
      }
    `)

	require.NoError(t, err)
}

func TestCheckInvalidResourceFieldDestroy(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
     resource Foo {}

     resource Bar {
         var foo: @Foo

         init(foo: @Foo) {
             self.foo <- foo
         }

         destroy() {
             destroy self.foo
         }
     }

     fun test() {
         let foo <- create Foo()
         let bar <- create Bar(foo: <-foo)
         destroy bar.foo
     }
   `)

	errs := RequireCheckerErrors(t, err, 2)

	// TODO: maybe have dedicated error

	assert.IsType(t, &sema.InvalidNestedResourceMoveError{}, errs[0])
	assert.IsType(t, &sema.ResourceLossError{}, errs[1])
}

func TestCheckResourceParameterInInterfaceNoResourceLossError(t *testing.T) {

	t.Parallel()

	declarationKinds := []common.DeclarationKind{
		common.DeclarationKindInitializer,
		common.DeclarationKindFunction,
	}

	for _, compositeKind := range common.InstantiableCompositeKindsWithFieldsAndFunctions {
		for _, declarationKind := range declarationKinds {
			for _, hasCondition := range []bool{true, false} {

				testName := fmt.Sprintf(
					"%s %s/hasCondition=%v",
					compositeKind,
					declarationKind,
					hasCondition,
				)

				innerDeclaration := ""
				switch declarationKind {
				case common.DeclarationKindInitializer:
					innerDeclaration = declarationKind.Keywords()

				case common.DeclarationKindFunction:
					innerDeclaration = fmt.Sprintf("%s test", declarationKind.Keywords())
				}

				functionBlock := ""
				if hasCondition {
					functionBlock = "{ pre { true } }"
				}

				t.Run(testName, func(t *testing.T) {

					_, err := ParseAndCheck(t, fmt.Sprintf(
						`
                          resource X {}

                          %[1]s interface Y {

                              // Should not result in a resource loss error
                              %[2]s(from: @X) %[3]s
                          }
                        `,
						compositeKind.Keyword(),
						innerDeclaration,
						functionBlock,
					))

					require.NoError(t, err)
				})
			}
		}
	}
}

func TestCheckResourceFieldUseAndDestruction(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
     resource interface RI {}

     resource R {
         var ris: @{String: {RI}}

         init(_ ri: @{RI}) {
             self.ris <- {"first": <-ri}
         }

         fun use() {
            let ri <- self.ris.remove(key: "first")
            absorb(<-ri)
         }

         destroy() {
             destroy self.ris
         }
     }

     fun absorb(_ ri: @{RI}?) {
         destroy ri
     }
   `)

	require.NoError(t, err)
}

func TestCheckInvalidResourceMethodBinding(t *testing.T) {

	t.Parallel()

	// TODO: replace AnyStruct return type with ([@R]#(@R): Void)
	//   once bound function types are supported

	_, err := ParseAndCheck(t, `
      resource R {}

      fun test(): AnyStruct {
          let rs <- [<-create R()]
          let append = rs.append
          destroy rs
          return append
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.ResourceMethodBindingError{}, errs[0])
}

func TestCheckInvalidResourceMethodCall(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource R {}

      fun test() {
          let rs <- [<-create R()]
          rs.append(<-create R())
          destroy rs
      }
    `)

	require.NoError(t, err)
}

func TestCheckResourceOptionalBinding(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource R {}

      fun test() {
          let maybeR: @R? <- create R()
          if let r <- maybeR {
              destroy r
          }
      }
    `)

	require.NoError(t, err)
}

func TestCheckInvalidResourceOptionalBindingResourceLossInThen(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource R {}

      fun test() {
          let maybeR: @R? <- create R()
          if let r <- maybeR {
              // resource loss of r
          }
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.ResourceLossError{}, errs[0])
}

func TestCheckInvalidResourceOptionalBindingResourceUseAfterInvalidationInThen(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource R {}

      fun test() {
          let maybeR: @R? <- create R()
          if let r <- maybeR {
              destroy r
              destroy maybeR
          }
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
}

func TestCheckInvalidResourceOptionalBindingResourceUseAfterInvalidationAfterBranches(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource R {}

      fun test() {
          let maybeR: @R? <- create R()
          if let r <- maybeR {
              destroy r
          }
          f(<-maybeR)
      }

      fun f(_ r: @R?) {
          destroy r
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
}

func TestCheckResourceOptionalBindingWithSecondValue(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource R {}

      fun test() {
          let r1 <- create R()
          var r2: @R? <- create R()

          if let r3 <- r2 <- r1 {
              // r1 was definitely moved
              // r2 contains r1
              destroy r2
              // only then branch defined r3
              destroy r3
          } else {
              // r1 was definitely moved
              // r2 contains r1
              destroy r2
          }
      }
    `)
	require.NoError(t, err)
}

func TestCheckResourceOptionalBindingResourceInvalidation(t *testing.T) {

	t.Parallel()

	t.Run("separate, without else", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          resource R {}

          fun asOpt(_ r: @R): @R? {
              return <-r
          }

          fun test() {
              let r <- create R()
              let optR <- asOpt(<-r)
              if let r2 <- optR {
                  destroy r2
              }
          }
        `)
		require.NoError(t, err)
	})

	t.Run("separate, with else", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          resource R {}

          fun asOpt(_ r: @R): @R? {
              return <-r
          }

          fun consume(_ r: @R?) {
              destroy <-r
          }

          fun test() {
              let r <- create R()
              let optR <- asOpt(<-r)
              if let r2 <- optR {
                  destroy r2
              } else {
                  consume(<-optR)
              }
          }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
	})

	t.Run("inline, without else", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          resource R {}

          fun asOpt(_ r: @R): @R? {
              return <-r
          }

          fun test() {
              let r <- create R()
              if let r2 <- asOpt(<-r) {
                  destroy r2
              }
          }
        `)

		require.NoError(t, err)
	})

	t.Run("inline, with else, non-optional", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          resource R {}

          fun asOpt(_ r: @R): @R? {
              return <-r
          }

          fun consume(_ r: @R?) {
              destroy <-r
          }

          fun test() {
              let r <- create R()
              if let r2 <- asOpt(<-r) {
                  destroy r2
              } else {
                  consume(<-r)
              }
          }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
	})

	t.Run("inline, with else, optional", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          resource R {}

          fun identity(_ r: @R?): @R? {
              return <-r
          }

          fun consume(_ r: @R?) {
              destroy <-r
          }

          fun test() {
              let r: @R? <- create R()
              if let r2 <- identity(<-r) {
                  destroy r2
              } else {
                  consume(<-r)
              }
          }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
	})
}

func TestCheckResourceOptionalBindingFailableCast(t *testing.T) {

	t.Parallel()

	t.Run("destroy", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          resource interface RI {}

          resource R: RI {}

          fun test() {
              let ri: @{RI} <- create R()
              if let r <- ri as? @R {
                  destroy r
              } else {
                  destroy ri
              }
          }
        `)

		require.NoError(t, err)
	})

	t.Run("return", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          resource interface RI {}

          resource R: RI {}

          fun test(): @R? {
              let ri: @{RI} <- create R()
              if let r <- ri as? @R {
                  return <-r
              } else {
                  destroy ri
                  return nil
              }
          }
        `)

		require.NoError(t, err)
	})

}

func TestCheckInvalidResourceOptionalBindingFailableCastResourceUseAfterInvalidationInThen(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t,
		`
         resource interface RI {}

         resource R: RI {}

         fun test() {
             let ri: @{RI} <- create R()
             if let r <- ri as? @R {
                 destroy r
                 destroy ri
             } else {
                 destroy ri
             }
         }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
}

func TestCheckInvalidResourceOptionalBindingFailableCastResourceUseAfterInvalidationAfterBranches(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t,
		`
         resource interface RI {}

         resource R: RI {}

         fun test() {
             let ri: @{RI} <- create R()
             if let r <- ri as? @R {
                 destroy r
             }
             destroy ri
         }
    `)

	errs := RequireCheckerErrors(t, err, 2)

	assert.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
	assert.IsType(t, &sema.ResourceLossError{}, errs[1])
}

func TestCheckInvalidResourceOptionalBindingFailableCastResourceLossMissingElse(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource interface RI {}

      resource R: RI {}

      fun test() {
          let ri: @{RI} <- create R()
          if let r <- ri as? @R {
              destroy r
          }
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.ResourceLossError{}, errs[0])
}

func TestCheckInvalidResourceOptionalBindingFailableCastResourceUseAfterInvalidationAfterThen(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource interface RI {}

      resource R: RI {}

      fun test() {
          let ri: @{RI} <- create R()
          if let r <- ri as? @R {
              destroy r
          }
          destroy ri
      }
    `)

	errs := RequireCheckerErrors(t, err, 2)

	assert.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
	assert.IsType(t, &sema.ResourceLossError{}, errs[1])
}

func TestCheckInvalidResourceOptionalBindingFailableCastMissingElse(t *testing.T) {

	t.Parallel()

	t.Run("top-level resource interface to resource, missing else", func(t *testing.T) {

		_, err := ParseAndCheck(t, `
          resource interface RI {}

          resource R: RI {}

          fun test(ri: @{RI}) {
              if let r <- ri as? @R {
                  destroy r
              }
          }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.ResourceLossError{}, errs[0])
	})

	t.Run("contract interface resource to contract to resource", func(t *testing.T) {

		_, err := ParseAndCheck(t, `
          contract interface CI {
              resource R {}
          }

          contract C: CI {
              resource R {}
          }

          fun test(r: @CI.R) {
              if let r2 <- r as? @C.R {
                  destroy r2
              }
          }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.ResourceLossError{}, errs[0])
	})
}

func TestCheckInvalidResourceFailableCastOutsideOptionalBinding(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource interface RI {}

      resource R: RI {}

      fun test() {
          let ri: @{RI} <- create R()
          let r <- ri as? @R
          destroy r
      }
    `)

	errs := RequireCheckerErrors(t, err, 2)

	assert.IsType(t, &sema.InvalidFailableResourceDowncastOutsideOptionalBindingError{}, errs[0])
	assert.IsType(t, &sema.ResourceLossError{}, errs[1])
}

func TestCheckInvalidResourceFailableCastNonIdentifier(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource interface RI {}

      resource R: RI {}

      fun createR(): @{RI} {
          return <- create R()
      }

      fun test() {
          if let r <- createR() as? @R {
              destroy r
          }
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.InvalidNonIdentifierFailableResourceDowncast{}, errs[0])
}

func TestCheckInvalidUnaryMoveAndCopyTransfer(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource R {}

      fun test() {
          let r = <- create R()
          destroy r
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.IncorrectTransferOperationError{}, errs[0])
}

func TestCheckInvalidResourceSelfMoveToFunction(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `

      resource X {

          fun test() {
              absorb(<-self)
          }
      }

      fun absorb(_ x: @X) {
          destroy x
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.InvalidSelfInvalidationError{}, errs[0])
}

func TestCheckInvalidResourceSelfMoveInVariableDeclaration(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `

      resource X {

          fun test() {
              let x <- self
              destroy x
          }
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.InvalidSelfInvalidationError{}, errs[0])
}

func TestCheckInvalidResourceSelfDestruction(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `

      resource X {

          fun test() {
              destroy self
          }
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.InvalidSelfInvalidationError{}, errs[0])
}

func TestCheckInvalidResourceSelfMoveReturnFromFunction(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `

      resource X {

          fun test(): @X {
              return <-self
          }
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.InvalidSelfInvalidationError{}, errs[0])
}

func TestCheckInvalidResourceSelfMoveIntoArrayLiteral(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `

      resource X {

          fun test(): @[X] {
              return <-[<-self]
          }
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.InvalidSelfInvalidationError{}, errs[0])
}

func TestCheckInvalidResourceSelfMoveIntoDictionaryLiteral(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `

      resource X {

          fun test(): @{String: X} {
              return <-{"self": <-self}
          }
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.InvalidSelfInvalidationError{}, errs[0])
}

func TestCheckInvalidResourceSelfMoveSwap(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `

      resource X {

          fun test() {
              var x: @X? <- nil
              let oldX <- x <- self
              destroy x
              destroy oldX
          }
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.InvalidSelfInvalidationError{}, errs[0])
}

func TestCheckResourceCreationAndInvalidationInLoop(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `

      resource X {}

      fun loop() {
          var i = 0
          while i < 10 {
              let x <- create X()
              destroy x
              i = i + 1
          }
      }
    `)

	require.NoError(t, err)
}

func TestCheckInvalidResourceCreationAndPotentialInvalidationInLoop(t *testing.T) {

	t.Parallel()

	test := func(loop string, controlFlowStatement string) {
		name := fmt.Sprintf(
			"%s, %s",
			loop,
			controlFlowStatement,
		)

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := ParseAndCheck(t,
				fmt.Sprintf(
					`
                      resource X {}

                      fun loop() {
                          %s {
                              let x <- create X()
                              if false {
                                  %s
                              }
                              destroy x
                          }
                      }
                    `,
					loop,
					controlFlowStatement,
				),
			)

			errs := RequireCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.ResourceLossError{}, errs[0])
		})
	}

	for _, loop := range []string{"while true", "for e in []"} {
		for _, controlFlowStatement := range []string{"continue", "break", "return"} {
			test(loop, controlFlowStatement)
		}
	}
}

func TestCheckResourceCreationAndInvalidationAfterLoopWithJump(t *testing.T) {

	t.Parallel()

	test := func(loop, controlFlowStatement string) {
		t.Run(fmt.Sprintf("%s, %s", loop, controlFlowStatement), func(t *testing.T) {
			t.Parallel()

			_, err := ParseAndCheck(t,
				fmt.Sprintf(
					`
                      resource X {}

                      fun loop() {
                          let x <- create X()
                          %s {
                              if false {
                                  %s
                              }
                          }
                          destroy x
                      }
                    `,
					loop,
					controlFlowStatement,
				),
			)

			require.NoError(t, err)
		})
	}

	for _, loop := range []string{"while true", "for e in []"} {
		for _, controlFlowStatement := range []string{"continue", "break"} {
			test(loop, controlFlowStatement)
		}
	}
}

func TestCheckInvalidResourceCreationAndPotentialInvalidationInLoopWithControlFlow(t *testing.T) {

	t.Parallel()

	test := func(loop, controlFlowStatement, firstAction, secondAction string) {
		name := fmt.Sprintf(
			"%s, %s, %s, %s",
			loop,
			firstAction,
			controlFlowStatement,
			secondAction,
		)

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := ParseAndCheck(t,
				fmt.Sprintf(
					`
                      resource X {}

                      fun drop(_ r: @AnyResource) {
                          destroy r
                      }

                      fun loop() {
                          let x <- create X()
                          %s {
                              if false {
                                  %s
                                  %s
                              }
                          }
                          %s
                      }
                    `,
					loop,
					firstAction,
					controlFlowStatement,
					secondAction,
				),
			)

			errs := RequireCheckerErrors(t, err, 2)

			assert.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
			assert.IsType(t, &sema.ResourceLossError{}, errs[1])
		})
	}

	actions := []string{"destroy x", "drop(<-x)"}

	for _, loop := range []string{"while true", "for e in []"} {
		for _, controlFlowStatement := range []string{"continue", "break"} {
			for _, firstAction := range actions {
				for _, secondAction := range actions {
					test(loop, controlFlowStatement, firstAction, secondAction)
				}
			}
		}
	}
}

func TestCheckInvalidResourceOwnerField(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource Test {
          let owner: PublicAccount

          init(owner: PublicAccount) {
              self.owner = owner
          }
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.InvalidDeclarationError{}, errs[0])
}

func TestCheckInvalidResourceInterfaceOwnerField(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
     resource interface Test {
         let owner: PublicAccount
     }
   `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.InvalidDeclarationError{}, errs[0])
}

func TestCheckInvalidResourceOwnerFunction(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
     resource Test {
         fun owner() {}
     }
   `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.InvalidDeclarationError{}, errs[0])
}

func TestCheckInvalidResourceInterfaceOwnerFunction(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
     resource interface Test {
         fun owner()
     }
   `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.InvalidDeclarationError{}, errs[0])
}

func TestCheckResourceOwnerFieldUse(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
     resource Test {

         fun test(): PublicAccount? {
             return self.owner
         }
     }
   `)

	require.NoError(t, err)
}

func TestCheckResourceInterfaceOwnerFieldUse(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
     resource interface Test {

         fun test() {
             pre { self.owner != nil }
         }
     }
   `)

	require.NoError(t, err)
}

func TestCheckInvalidResourceOwnerFieldInitialization(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
     resource Test {

         init(owner: PublicAccount) {
             self.owner = owner
         }
     }
   `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.AssignmentToConstantMemberError{}, errs[0])
}

func TestCheckInvalidResourceInterfaceType(t *testing.T) {

	t.Parallel()

	t.Run("direct", func(t *testing.T) {
		_, err := ParseAndCheck(t, `
          resource interface RI {}

          resource R: RI {}

          let ri: @RI <- create R()
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidInterfaceTypeError{}, errs[0])
	})

	t.Run("in array", func(t *testing.T) {
		_, err := ParseAndCheck(t, `
          resource interface RI {}

          resource R: RI {}

          let ri: @[RI] <- [<-create R()]
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidInterfaceTypeError{}, errs[0])
	})
}

func TestCheckRestrictedAnyResourceType(t *testing.T) {

	t.Parallel()

	t.Run("direct", func(t *testing.T) {
		_, err := ParseAndCheck(t, `
          resource interface RI {}

          resource R: RI {}

          let ri: @AnyResource{RI} <- create R()
        `)

		require.NoError(t, err)
	})

	t.Run("in array", func(t *testing.T) {
		_, err := ParseAndCheck(t, `
          resource interface RI {}

          resource R: RI {}

          let ri: @[AnyResource{RI}] <- [<-create R()]
        `)

		require.NoError(t, err)
	})
}

func TestCheckInvalidOptionalResourceNilCoalescingResourceLoss(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheckWithPanic(t, `

      resource R {}

      fun returnResourceOpt() {
          let optR: @R? <- create R()
          let r <- optR ?? panic("no R")
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.ResourceLossError{}, errs[0])
}

func TestCheckOptionalResourceCoalescingAndReturn(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheckWithPanic(t, `

      resource R {}

      fun returnResourceOpt(): @R {
          let optR: @R? <- create R()
          return <- (optR ?? panic("no R"))
      }
    `)

	require.NoError(t, err)
}

func TestCheckInvalidOptionalResourceCoalescingRightSide(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `

      resource R {}

      fun returnResourceOpt(): @R {
          let r1: @R? <- create R()
          let r2: @R <- create R()
          return <- (r1 ?? r2)
      }
    `)

	errs := RequireCheckerErrors(t, err, 2)

	assert.IsType(t, &sema.InvalidNilCoalescingRightResourceOperandError{}, errs[0])
	assert.IsType(t, &sema.ResourceLossError{}, errs[1])
}

// https://github.com/dapperlabs/flow-go/issues/3407
//
// Check that an function's return information
// does not influence another function's return information.
func TestCheckInvalidResourceLossInNestedContractResource(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `

      access(all) contract C {

          resource R {

              fun foo(r: @R) {
                  if let r2 <- r as? @R {
                      destroy r2
                  }
              }
          }

          access(all) fun bar() {
              return
          }
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.ResourceLossError{}, errs[0])
}

// https://github.com/onflow/cadence/issues/73
func TestCheckResourceMoveMemberInvocation(t *testing.T) {

	t.Parallel()

	t.Run("invalid use as argument", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `

          resource Test {
              fun use(_ test: @Test) {
                  destroy test
              }
          }

          fun test() {
              let test <- create Test()
              test.use(<-test)
          }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
	})

	t.Run("valid use", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `

          resource Test {
              fun use() {}
          }

          fun test() {
              let test <- create Test()
              test.use()
              destroy test
          }
        `)

		require.NoError(t, err)
	})

	t.Run("valid use, with argument of same type", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `

          resource Test {
              fun use(_ test: @Test) {
                  destroy test
              }
          }

          fun test() {
              let test1 <- create Test()
              let test2 <- create Test()
              test1.use(<-test2)
              destroy test1
          }
        `)

		require.NoError(t, err)
	})

	t.Run("valid use, in argument", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `

          resource Test {
              fun use1(_ x: Int) {}
              fun use2(): Int { return 1 }
          }

          fun test() {
              let test <- create Test()
              test.use1(test.use2())
              destroy test
          }
        `)

		require.NoError(t, err)
	})

	t.Run("invalid loss, invalidation is temporary", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `

          resource Test {
              fun use() {}
          }

          fun test() {
              let test <- create Test()
              test.use()
          }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.ResourceLossError{}, errs[0])
	})

	t.Run("invocation on undeclared variable", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          let x = y.isInstance(Type<Int>())
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
	})
}

func TestCheckInvalidationInPreCondition(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource R {}

      fun duplicate(_ r: @R): Bool {
          destroy r
          return true
      }

      fun duplicatePre(_ r: @R): @R {
          pre {
              duplicate(<-r)
          }
          return <- r
      }
    `)

	errs := RequireCheckerErrors(t, err, 2)

	assert.IsType(t, &sema.PurityError{}, errs[0])
	assert.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[1])
}

func TestCheckResourceRepeatedInvalidationWithBreak(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `

          resource X {}

          fun test() {
              let x <- create X()
              while true {
                  if true {
                      destroy x
                      break
                  }
              }
              destroy x
          }
        `)

	errs := RequireCheckerErrors(t, err, 2)

	assert.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
	assert.IsType(t, &sema.ResourceLossError{}, errs[1])
}

func TestCheckResourceCreationAndInvalidationAfterControlFlow(t *testing.T) {

	t.Parallel()

	test := func(controlFlowStatement string) {
		t.Run(controlFlowStatement, func(t *testing.T) {

			t.Parallel()

			_, err := ParseAndCheckWithPanic(t,
				fmt.Sprintf(
					`
                      resource X {}

                      fun test(exitEarly: Bool) {
                          if exitEarly {
                              %s
                          }

                          let x <- create X()
                          destroy x
                      }
                    `,
					controlFlowStatement,
				),
			)

			require.NoError(t, err)
		})
	}

	for _, controlFlowStatement := range []string{"return", `panic("")`} {
		test(controlFlowStatement)
	}
}

func TestCheckResourceCreationAndInvalidationAfterControlFlowInIf(t *testing.T) {

	t.Parallel()

	test := func(controlFlowStatement string) {
		t.Run(controlFlowStatement, func(t *testing.T) {

			t.Parallel()

			_, err := ParseAndCheckWithPanic(t,
				fmt.Sprintf(
					`
                      resource X {}

                      fun test(exitEarly: Bool) {
                          if true {
                              if exitEarly {
                                  %s
                              }

                              let x <- create X()
                              destroy x
                          }
                      }
                    `,
					controlFlowStatement,
				),
			)

			require.NoError(t, err)
		})
	}

	for _, controlFlowStatement := range []string{"return", `panic("")`} {
		test(controlFlowStatement)
	}
}

func TestCheckResourceCreationAndInvalidationAfterControlFlowInLoop(t *testing.T) {

	t.Parallel()

	test := func(controlFlowStatement string) {
		t.Run(controlFlowStatement, func(t *testing.T) {

			t.Parallel()

			_, err := ParseAndCheckWithPanic(t,
				fmt.Sprintf(
					`
                      resource X {}

                      fun test(exitEarly: Bool) {
                          while true {
                              if exitEarly {
                                  %s
                              }

                              let x <- create X()
                              destroy x
                          }
                      }
                    `,
					controlFlowStatement,
				),
			)

			require.NoError(t, err)
		})
	}

	for _, controlFlowStatement := range []string{"continue", "break", "return", `panic("")`} {
		test(controlFlowStatement)
	}
}

func TestCheckInvalidationInPostConditionBefore(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource R {}

      fun duplicate(_ r: @R): Bool {
          destroy r
          return true
      }

      fun duplicatePostBefore(_ r: @R): @R {
          post {
              before(duplicate(<-r))
          }
          return <- r
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
}

func TestCheckInvalidationInPostCondition(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource R {}

      fun duplicate(_ r: @R): Bool {
          destroy r
          return true
      }

      fun duplicatePostBefore(_ r: @R): @R {
          post {
              duplicate(<-r)
          }
          return <- r
      }
    `)

	errs := RequireCheckerErrors(t, err, 2)

	assert.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[1])
	assert.IsType(t, &sema.PurityError{}, errs[0])
}

func TestCheckFunctionDefinitelyHaltedNoResourceLoss(t *testing.T) {

	t.Parallel()

	// A function which definitely halts does not lead to a resource loss error

	t.Run("panic statement", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithPanic(t, `
          fun duplicate(_ r: @AnyResource) {
              panic("")
          }
        `)

		require.NoError(t, err)
	})

	t.Run("if statement", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithPanic(t, `
          fun duplicate(_ r: @AnyResource, x: Bool) {
              if x {
                  panic("true")
              } else {
                  panic("false")
              }
          }
        `)

		require.NoError(t, err)
	})
}

func TestCheckOptionalResourceBindingWithSecondValue(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource R {
          let field: Int

          init() {
              self.field = 1
          }
      }

      resource Test {

          var r: @R?

          init() {
              self.r <- create R()
          }

          destroy () {
              destroy self.r
          }

          fun duplicate(): @R? {
              if let r <- self.r <- nil {
                  let r2 <- self.r <- nil
                  self.r <-! r2
                  return <-r
              } else {
                  return nil
              }
          }
      }

      fun test() {
          let test <- create Test()
          let copy <- test.duplicate()

          destroy copy
          destroy test
      }
    `)
	require.NoError(t, err)
}

func TestCheckEmptyResourceCollectionMove(t *testing.T) {

	t.Parallel()

	t.Run("Dictionary", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource R {
                init() {
                }
            }

            fun foo() {
               bar(a: <-{})
            }

            fun bar(a: @{String: R}) {
                destroy a
            }
        `)

		require.NoError(t, err)
	})

	t.Run("Array", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource R {
                init() {
                }
            }

            fun foo() {
               bar(a: <-[])
            }

            fun bar(a: @[R]) {
                destroy a
            }
        `)

		require.NoError(t, err)
	})
}

func TestCheckResourceInvalidationInBranchesAndLoops(t *testing.T) {

	t.Parallel()

	t.Run("if-else: missing else branch", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource R {}

            fun test(r: @R) {
                if true {
                    destroy r
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.ResourceLossError{}, errs[0])
	})

	t.Run("if-else: missing else branch, transaction", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource R {}

            transaction {

                let r: @R

                prepare() {
                    self.r <- create R()
                }

                execute {
                    if true {
                        destroy self.r
                    }
                }
           }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.ResourceFieldNotInvalidatedError{}, errs[0])
	})

	t.Run("if-else: missing else branches", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource R {}

            fun test(r: @R) {
                if true {
                    if true {
                       destroy r
                    }
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.ResourceLossError{}, errs[0])
	})

	t.Run("if-else: missing else branches, transaction", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource R {}

            transaction {

                let r: @R

                prepare() {
                    self.r <- create R()
                }

                execute {
                    if true {
                        if true {
                           destroy self.r
                        }
                    }
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.ResourceFieldNotInvalidatedError{}, errs[0])
	})

	t.Run("if-else: missing else branch in nested if", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource R {}

            fun test(r: @R) {
                if true {
                    if true {
                       destroy r
                    }
                } else {
                    destroy r
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.ResourceLossError{}, errs[0])
	})

	t.Run("if-else: missing else branch in nested if, transaction", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource R {}

            transaction {

                let r: @R

                prepare() {
                    self.r <- create R()
                }

                execute {
                    if true {
                        if true {
                           destroy self.r
                        }
                    } else {
                        destroy self.r
                    }
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.ResourceFieldNotInvalidatedError{}, errs[0])
	})

	t.Run("switch-case: missing destruction in one case", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource R {}

            fun test(n: Int, r: @R) {
                switch n {
                    case 1:
                        destroy r
                    case 2:
                        // Some random statement that has no effect
                        let a = "do nothing"
                    default:
                        destroy r
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.ResourceLossError{}, errs[0])
	})

	t.Run("switch-case: missing destruction in one case, transaction", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource R {}

            transaction(n: Int) {

                let r: @R

                prepare() {
                    self.r <- create R()
                }

                execute {
                    switch n {
                        case 1:
                            destroy self.r
                        case 2:
                            // Some random statement that has no effect
                            let a = "do nothing"
                        default:
                            destroy self.r
                    }
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.ResourceFieldNotInvalidatedError{}, errs[0])
	})

	t.Run("switch-case: missing destruction in default case", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource R {}

            fun test(n: Int, r: @R) {
                switch n {
                    case 1:
                        destroy r
                    case 2:
                        destroy r
                    default:
                        break
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.ResourceLossError{}, errs[0])
	})

	t.Run("switch-case: missing destruction in default case, transaction", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource R {}

            transaction(n: Int) {

                let r: @R

                prepare() {
                    self.r <- create R()
                }

                execute {
                    switch n {
                        case 1:
                            destroy self.r
                        case 2:
                            destroy self.r
                        default:
                            break
                    }
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.ResourceFieldNotInvalidatedError{}, errs[0])
	})

	t.Run("switch-case: resource destruction in all cases", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource R {}

            fun test(n: Int, r: @R) {
                switch n {
                    case 1:
                        destroy r
                    case 2:
                        destroy r
                    default:
                        destroy r
                }
            }
        `)

		require.NoError(t, err)
	})

	t.Run("switch-case: resource destruction in all cases, transaction", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource R {}

            transaction(n: Int) {

                let r: @R

                prepare() {
                    self.r <- create R()
                }

                execute {
                    switch n {
                        case 1:
                            destroy self.r
                        case 2:
                            destroy self.r
                        default:
                            destroy self.r
                    }
                }
            }
        `)

		require.NoError(t, err)
	})

	t.Run("switch-case: no default", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource R {}

            fun test(n: Int, r: @R) {
                switch n {
                    case 1:
                        destroy r
                    case 2:
                        destroy r
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.ResourceLossError{}, errs[0])
	})

	t.Run("switch-case: no default, transaction", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource R {}

            transaction(n: Int) {

                let r: @R

                prepare() {
                    self.r <- create R()
                }

                execute {
                    switch n {
                        case 1:
                            destroy self.r
                        case 2:
                            destroy self.r
                    }
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.ResourceFieldNotInvalidatedError{}, errs[0])
	})

	t.Run("switch-case: missing destruction of one resource in one case", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource R {}

            fun test(n: Int, r1: @R, r2: @R) {
                switch n {
                    case n:
                        destroy r1
                        destroy r2
                    case 2:
                        destroy r1
                    default:
                        destroy r1
                        destroy r2
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.ResourceLossError{}, errs[0])
	})

	t.Run("switch-case: missing destruction of one resource in one case, transaction", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource R {}

            transaction(n: Int) {

                let r1: @R
                let r2: @R

                prepare() {
                    self.r1 <- create R()
                    self.r2 <- create R()
                }

                execute {
                    switch n {
                        case n:
                            destroy self.r1
                            destroy self.r2
                        case 2:
                            destroy self.r1
                        default:
                            destroy self.r1
                            destroy self.r2
                    }
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.ResourceFieldNotInvalidatedError{}, errs[0])
	})

	t.Run("switch-case: missing destruction of one resource in default case", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource R {}

            fun test(n: Int, r1: @R, r2: @R) {
                switch n {
                    case n:
                        destroy r1
                        destroy r2
                    case 2:
                        destroy r1
                        destroy r2
                    default:
                        destroy r1
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.ResourceLossError{}, errs[0])
	})

	t.Run("switch-case: missing destruction of one resource in default case, transaction", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource R {}

            transaction(n: Int) {

                let r1: @R
                let r2: @R

                prepare() {
                    self.r1 <- create R()
                    self.r2 <- create R()
                }

                execute {
                    switch n {
                        case n:
                            destroy self.r1
                            destroy self.r2
                        case 2:
                            destroy self.r1
                            destroy self.r2
                        default:
                            destroy self.r1
                    }
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.ResourceFieldNotInvalidatedError{}, errs[0])
	})

	t.Run("switch-case: loss of all resources in one case", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource R {}

            fun test(n: Int, r1: @R, r2: @R) {
                switch n {
                    case 1:
                        destroy r1
                        destroy r2
                    case 2:
                        break
                    default:
                        destroy r1
                        destroy r2
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.ResourceLossError{}, errs[0])
		assert.IsType(t, &sema.ResourceLossError{}, errs[1])
	})

	t.Run("switch-case: loss of all resources in one case, transaction", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource R {}

            transaction(n: Int) {

                let r1: @R
                let r2: @R

                prepare() {
                    self.r1 <- create R()
                    self.r2 <- create R()
                }

                execute {
                    switch n {
                        case 1:
                            destroy self.r1
                            destroy self.r2
                        case 2:
                            break
                        default:
                            destroy self.r1
                            destroy self.r2
                    }
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.ResourceFieldNotInvalidatedError{}, errs[0])
		assert.IsType(t, &sema.ResourceFieldNotInvalidatedError{}, errs[1])
	})

	t.Run("switch-case: loss of all resources in default case", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource R {}

            fun test(n: Int, r1: @R, r2: @R) {
                switch n {
                    case 1:
                        destroy r1
                        destroy r2
                    case 2:
                        destroy r1
                        destroy r2
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.ResourceLossError{}, errs[0])
		assert.IsType(t, &sema.ResourceLossError{}, errs[1])
	})

	t.Run("switch-case: loss of all resources in default case, transaction", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource R {}

            transaction(n: Int) {

                let r1: @R
                let r2: @R

                prepare() {
                    self.r1 <- create R()
                    self.r2 <- create R()
                }

                execute {
                    switch n {
                        case 1:
                            destroy self.r1
                            destroy self.r2
                        case 2:
                            destroy self.r1
                            destroy self.r2
                    }
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.ResourceFieldNotInvalidatedError{}, errs[0])
		assert.IsType(t, &sema.ResourceFieldNotInvalidatedError{}, errs[1])
	})

	t.Run("switch-case: unreachable destruction due to break", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource R {}

            fun test(n: Int, r: @R) {
                switch n {
                    case 1:
                        break
                        destroy r  // unreachable
                    default:
                        destroy r
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.UnreachableStatementError{}, errs[0])
		assert.IsType(t, &sema.ResourceLossError{}, errs[1])
	})

	t.Run("switch-case: unreachable destruction due to break, transaction", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource R {}

            transaction(n: Int) {

                let r: @R

                prepare() {
                    self.r <- create R()
                }

                execute {
                    switch n {
                        case 1:
                            break
                            destroy self.r  // unreachable
                        default:
                            destroy self.r
                    }
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.UnreachableStatementError{}, errs[0])
		assert.IsType(t, &sema.ResourceFieldNotInvalidatedError{}, errs[1])
	})

	t.Run("switch-case: return in one case", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource R {}

            fun test(n: Int, r: @R) {
                switch n {
                    case 1:
                        destroy r
                        return
                    default:
                        destroy r
                }
            }
        `)

		require.NoError(t, err)
	})

	t.Run("switch-case: return in one case, transaction", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource R {}

            transaction(n: Int) {

                let r: @R

                prepare() {
                    self.r <- create R()
                }

                execute {
                    switch n {
                        case 1:
                            destroy self.r
                            return
                        default:
                            destroy self.r
                    }
                }
            }
        `)

		require.NoError(t, err)
	})

	t.Run("switch-case: break in one case", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource R {}

            fun test(n: Int, r: @R) {
                switch n {
                    case 1:
                        destroy r
                        break
                    default:
                        destroy r
                }
            }
        `)

		require.NoError(t, err)
	})

	t.Run("switch-case: break in one case, transaction", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource R {}

            transaction(n: Int) {

                let r: @R

                prepare() {
                    self.r <- create R()
                }

                execute {
                    switch n {
                        case 1:
                            destroy self.r
                            break
                        default:
                            destroy self.r
                    }
                }
            }
        `)

		require.NoError(t, err)
	})

	t.Run("switch-case: destroy missing in default case", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource R {}

            fun test(n: Int, r: @R) {
                switch n {
                    case 1:
                        destroy r
                        return
                    default:
                        return
                }
                destroy r
            }
        `)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.ResourceLossError{}, errs[0])
		assert.IsType(t, &sema.UnreachableStatementError{}, errs[1])
	})

	t.Run("switch-case: destroy missing in default case, transaction", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource R {}

            transaction(n: Int) {

                let r: @R

                prepare() {
                    self.r <- create R()
                }

                execute {
                    switch n {
                        case 1:
                            destroy self.r
                            return
                        default:
                            return
                    }
                    destroy self.r
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.UnreachableStatementError{}, errs[0])
		assert.IsType(t, &sema.ResourceFieldNotInvalidatedError{}, errs[1])
	})

	t.Run("while loop: unreachable destruction due to break", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource R {}

            fun test(r: @R) {
                while true {
                    break
                    destroy r  // unreachable
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.UnreachableStatementError{}, errs[0])
		assert.IsType(t, &sema.ResourceLossError{}, errs[1])
	})

	t.Run("while loop: unreachable destruction due to break, transaction", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource R {}

            transaction(n: Int) {

                let r: @R

                prepare() {
                    self.r <- create R()
                }

                execute {
                    while true {
                        break
                        destroy self.r  // unreachable
                    }
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.UnreachableStatementError{}, errs[0])
		assert.IsType(t, &sema.ResourceFieldNotInvalidatedError{}, errs[1])
	})

	t.Run("while loop: unreachable destruction due to continue", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource R {}

            fun test(r: @R) {
                while true {
                    continue
                    destroy r  // unreachable
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.UnreachableStatementError{}, errs[0])
		assert.IsType(t, &sema.ResourceLossError{}, errs[1])
	})

	t.Run("while loop: unreachable destruction due to continue, transaction", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource R {}

            transaction(n: Int) {

                let r: @R

                prepare() {
                    self.r <- create R()
                }

                execute {
                    while true {
                        continue
                        destroy self.r  // unreachable
                    }
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.UnreachableStatementError{}, errs[0])
		assert.IsType(t, &sema.ResourceFieldNotInvalidatedError{}, errs[1])
	})

	t.Run("while loop: unreachable destruction due to return", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource R {}

            fun test(r: @R) {
                while true {
                    return
                    destroy r  // unreachable
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 3)

		assert.IsType(t, &sema.ResourceLossError{}, errs[0])
		assert.IsType(t, &sema.UnreachableStatementError{}, errs[1])
		assert.IsType(t, &sema.ResourceLossError{}, errs[2])
	})

	t.Run("while loop: unreachable destruction due to return, transaction", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource R {}

            transaction(n: Int) {

                let r: @R

                prepare() {
                    self.r <- create R()
                }

                execute {
                    while true {
                        return
                        destroy self.r  // unreachable
                    }
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.UnreachableStatementError{}, errs[0])
		assert.IsType(t, &sema.ResourceFieldNotInvalidatedError{}, errs[1])
	})

	t.Run("for loop: unreachable destruction due to break", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource R {}

            fun test(r: @R) {
                for i in [] {
                    break
                    destroy r  // unreachable
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.UnreachableStatementError{}, errs[0])
		assert.IsType(t, &sema.ResourceLossError{}, errs[1])
	})

	t.Run("for loop: unreachable destruction due to break, transaction", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource R {}

            transaction(n: Int) {

                let r: @R

                prepare() {
                    self.r <- create R()
                }

                execute {
                    for i in [] {
                        break
                        destroy self.r  // unreachable
                    }
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.UnreachableStatementError{}, errs[0])
		assert.IsType(t, &sema.ResourceFieldNotInvalidatedError{}, errs[1])
	})

	t.Run("for loop: unreachable destruction due to continue", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource R {}

            fun test(r: @R) {
                for i in [] {
                    continue
                    destroy r  // unreachable
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.UnreachableStatementError{}, errs[0])
		assert.IsType(t, &sema.ResourceLossError{}, errs[1])
	})

	t.Run("for loop: unreachable destruction due to continue, transaction", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource R {}

            transaction(n: Int) {

                let r: @R

                prepare() {
                    self.r <- create R()
                }

                execute {
                    for i in [] {
                        continue
                        destroy self.r  // unreachable
                    }
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.UnreachableStatementError{}, errs[0])
		assert.IsType(t, &sema.ResourceFieldNotInvalidatedError{}, errs[1])
	})

	t.Run("for loop: unreachable destruction due to return", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource R {}

            fun test(r: @R) {
                for i in [] {
                    return
                    destroy r  // unreachable
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 3)
		assert.IsType(t, &sema.ResourceLossError{}, errs[0])
		assert.IsType(t, &sema.UnreachableStatementError{}, errs[1])
		assert.IsType(t, &sema.ResourceLossError{}, errs[2])
	})

	t.Run("for loop: unreachable destruction due to return, transaction", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource R {}

            transaction(n: Int) {

                let r: @R

                prepare() {
                    self.r <- create R()
                }

                execute {
                    for i in [] {
                        return
                        destroy self.r  // unreachable
                    }
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.UnreachableStatementError{}, errs[0])
		assert.IsType(t, &sema.ResourceFieldNotInvalidatedError{}, errs[1])
	})
}

func TestCheckResourceInvalidationNeverFunctionCall(t *testing.T) {

	t.Parallel()

	t.Run("transaction: if, else", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithPanic(t, `
            resource R {}

            fun f(_ r: @R) { destroy r }

            transaction() {

                let r: @R

                prepare() {
                    self.r <- create R()
                }

                execute {
                    if false {
                        f(<-self.r)
                    } else {
                        panic("")
                    }
                }
            }
        `)

		require.NoError(t, err)
	})

	t.Run("transaction: if-let, else", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithPanic(t, `
            resource R {}

            fun f(_ r: @R) { destroy r }

            transaction() {

                let r: @R

                prepare() {
                    self.r <- create R()
                }

                execute {
                    if let x = nil {
                        f(<-self.r)
                    } else {
                        panic("")
                    }
                }
            }
        `)

		require.NoError(t, err)
	})

	t.Run("transaction: if-let, else if-let, else", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithPanic(t, `
            resource R {}

            fun f(_ r: @R) { destroy r }

            transaction() {

                let r: @R

                prepare() {
                    self.r <- create R()
                }

                execute {
                    if let x = nil {
                        f(<-self.r)
                    } else if let y = nil {
                        f(<-self.r)
                    } else {
                        panic("")
                    }
                }
            }
        `)

		require.NoError(t, err)
	})

	t.Run("function: if, else", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithPanic(t, `
            resource R {}

            fun f(_ r: @R) { destroy r }

            fun test() {
                let r <- create R()
                if false {
                    f(<-r)
                } else {
                    panic("")
                }
            }
        `)

		require.NoError(t, err)
	})

	t.Run("function: if-let, else", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithPanic(t, `
            resource R {}

            fun f(_ r: @R) { destroy r }

            fun test() {
                let r <- create R()
                if let x = nil {
                    f(<-r)
                } else {
                    panic("")
                }
            }
        `)

		require.NoError(t, err)
	})

	t.Run("function: if-let, else if-let, else", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithPanic(t, `
            resource R {}

            fun f(_ r: @R) { destroy r }

            fun test() {
                let r <- create R()
                if let x = nil {
                    f(<-r)
                } else if let y = nil {
                    f(<-r)
                } else {
                    panic("")
                }
            }
        `)

		require.NoError(t, err)
	})

	t.Run("if-else: missing else branch", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithPanic(t, `
            resource R {}

            fun test(r: @R) {
                if true {
                    panic("")
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.ResourceLossError{}, errs[0])
	})

	t.Run("if-else: missing else branch, transaction", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithPanic(t, `
            resource R {}

            transaction {

               let r: @R

               prepare() {
                   self.r <- create R()
               }

               execute {
                   if true {
                       panic("")
                   }
               }
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.ResourceFieldNotInvalidatedError{}, errs[0])
	})

	t.Run("if-else: missing else branches", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithPanic(t, `
            resource R {}

            fun test(r: @R) {
                if true {
                    if true {
                       panic("")
                    }
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.ResourceLossError{}, errs[0])
	})

	t.Run("if-else: missing else branches, transaction", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithPanic(t, `
            resource R {}

            transaction {

                let r: @R

                prepare() {
                    self.r <- create R()
                }

                execute {
                    if true {
                        if true {
                           panic("")
                        }
                    }
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.ResourceFieldNotInvalidatedError{}, errs[0])
	})

	t.Run("if-else: missing else branch in nested if", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithPanic(t, `
            resource R {}

            fun test(r: @R) {
                if true {
                    if true {
                       panic("")
                    }
                } else {
                    panic("")
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.ResourceLossError{}, errs[0])
	})

	t.Run("if-else: missing else branch in nested if, transaction", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithPanic(t, `
            resource R {}

            transaction {

                let r: @R

                prepare() {
                    self.r <- create R()
                }

                execute {
                    if true {
                        if true {
                           panic("")
                        }
                    } else {
                        panic("")
                    }
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.ResourceFieldNotInvalidatedError{}, errs[0])
	})

	t.Run("if-else: invalidation and return in then branch, halt in else branch", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithPanic(t, `
           resource R {}

           fun test() {
               let r <- create R()

               if true {
                   destroy r
                   return
               } else {
                   panic("halt")
               }
           }
        `)

		require.NoError(t, err)
	})

	t.Run("if-else: halt in then branch, invalidation and return in else branch", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithPanic(t, `
           resource R {}

           fun test() {
               let r <- create R()

               if true {
                   panic("halt")
               } else {
                   destroy r
                   return
               }
           }
        `)

		require.NoError(t, err)
	})

	t.Run("switch-case: missing invalidation in one case", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithPanic(t, `
            resource R {}

            fun test(n: Int, r: @R) {
                switch n {
                    case 1:
                        panic("")
                    case 2:
                        // Some random statement that has no effect
                        let a = "do nothing"
                    default:
                        panic("")
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.ResourceLossError{}, errs[0])
	})

	t.Run("switch-case: missing invalidation in one case, transaction", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithPanic(t, `
            resource R {}

            transaction(n: Int) {

                let r: @R

                prepare() {
                    self.r <- create R()
                }

                execute {
                    switch n {
                        case 1:
                            panic("")
                        case 2:
                            // Some random statement that has no effect
                            let a = "do nothing"
                        default:
                            panic("")
                    }
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.ResourceFieldNotInvalidatedError{}, errs[0])
	})

	t.Run("switch-case: missing invalidation in one case, mixed", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithPanic(t, `
            resource R {}

            fun test(n: Int, r: @R) {
                switch n {
                    case 1:
                        panic("")
                    case 2:
                        // Some random statement that has no effect
                        let a = "do nothing"
                    default:
                        destroy r
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.ResourceLossError{}, errs[0])
	})

	t.Run("switch-case: missing invalidation in one case, mixed, transaction", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithPanic(t, `
            resource R {}

            transaction(n: Int) {

                let r: @R

                prepare() {
                    self.r <- create R()
                }

                execute {
                    switch n {
                        case 1:
                            panic("")
                        case 2:
                            // Some random statement that has no effect
                            let a = "do nothing"
                        default:
                            destroy self.r
                    }
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.ResourceFieldNotInvalidatedError{}, errs[0])
	})

	t.Run("switch-case: missing invalidation in default case", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithPanic(t, `
            resource R {}

            fun test(n: Int, r: @R) {
                switch n {
                    case 1:
                        panic("")
                    case 2:
                        panic("")
                    default:
                        break
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.ResourceLossError{}, errs[0])
	})

	t.Run("switch-case: missing invalidation in default case, transaction", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithPanic(t, `
            resource R {}

            transaction(n: Int) {

                let r: @R

                prepare() {
                    self.r <- create R()
                }

                execute {
                    switch n {
                        case 1:
                            panic("")
                        case 2:
                            panic("")
                        default:
                            break
                    }
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.ResourceFieldNotInvalidatedError{}, errs[0])
	})

	t.Run("switch-case: missing invalidation in default case, mixed", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithPanic(t, `
            resource R {}

            fun test(n: Int, r: @R) {
                switch n {
                    case 1:
                        panic("")
                    case 2:
                        destroy r
                    default:
                        break
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.ResourceLossError{}, errs[0])
	})

	t.Run("switch-case: missing invalidation in default case, mixed, transaction", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithPanic(t, `
            resource R {}

            transaction(n: Int) {

                let r: @R

                prepare() {
                    self.r <- create R()
                }

                execute {
                    switch n {
                        case 1:
                            panic("")
                        case 2:
                            destroy self.r
                        default:
                            break
                    }
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.ResourceFieldNotInvalidatedError{}, errs[0])
	})

	t.Run("switch-case: invalidation in all cases", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithPanic(t, `
            resource R {}

            fun test(n: Int, r: @R) {
                switch n {
                    case 1:
                        panic("")
                    case 2:
                        panic("")
                    default:
                        panic("")
                }
            }
        `)

		require.NoError(t, err)
	})

	t.Run("switch-case: invalidation in all cases, transaction", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithPanic(t, `
            resource R {}

            transaction(n: Int) {

                let r: @R

                prepare() {
                    self.r <- create R()
                }

                execute {
                    switch n {
                        case 1:
                            panic("")
                        case 2:
                            panic("")
                        default:
                            panic("")
                    }
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.ResourceFieldNotInvalidatedError{}, errs[0])
	})

	t.Run("switch-case: invalidation in all cases, mixed", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithPanic(t, `
            resource R {}

            fun test(n: Int, r: @R) {
                switch n {
                    case 1:
                        panic("")
                    case 2:
                        destroy r
                    default:
                        panic("")
                }
            }
        `)

		require.NoError(t, err)
	})

	t.Run("switch-case: invalidation in all cases, mixed, transaction", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithPanic(t, `
            resource R {}

            transaction(n: Int) {

                let r: @R

                prepare() {
                    self.r <- create R()
                }

                execute {
                    switch n {
                        case 1:
                            panic("")
                        case 2:
                            destroy self.r
                        default:
                            panic("")
                    }
                }
            }
        `)

		require.NoError(t, err)
	})

	t.Run("switch-case: no default", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithPanic(t, `
            resource R {}

            fun test(n: Int, r: @R) {
                switch n {
                    case 1:
                        panic("")
                    case 2:
                        panic("")
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.ResourceLossError{}, errs[0])
	})

	t.Run("switch-case: no default, transaction", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithPanic(t, `
            resource R {}

            transaction(n: Int) {

                let r: @R

                prepare() {
                    self.r <- create R()
                }

                execute {
                    switch n {
                        case 1:
                            panic("")
                        case 2:
                            panic("")
                    }
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.ResourceFieldNotInvalidatedError{}, errs[0])
	})

	t.Run("switch-case: no default, mixed", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithPanic(t, `
            resource R {}

            fun test(n: Int, r: @R) {
                switch n {
                    case 1:
                        panic("")
                    case 2:
                        destroy r
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.ResourceLossError{}, errs[0])
	})

	t.Run("switch-case: no default, mixed, transaction", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithPanic(t, `
            resource R {}

            transaction(n: Int) {

                let r: @R

                prepare() {
                    self.r <- create R()
                }

                execute {
                    switch n {
                        case 1:
                            panic("")
                        case 2:
                            destroy self.r
                    }
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.ResourceFieldNotInvalidatedError{}, errs[0])
	})

	t.Run("switch-case: missing invalidation of one resource in one case", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithPanic(t, `
            resource R {}

            fun test(n: Int, r1: @R, r2: @R) {
                switch n {
                    case n:
                        panic("")
                    case 2:
                        destroy r1
                    default:
                        panic("")
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.ResourceLossError{}, errs[0])
	})

	t.Run("switch-case: missing invalidation of one resource in one case, transaction", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithPanic(t, `
            resource R {}

            transaction(n: Int) {

                let r1: @R
                let r2: @R

                prepare() {
                    self.r1 <- create R()
                    self.r2 <- create R()
                }

                execute {
                    switch n {
                        case n:
                            panic("")
                        case 2:
                            destroy self.r1
                        default:
                            panic("")
                    }
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.ResourceFieldNotInvalidatedError{}, errs[0])
	})

	t.Run("switch-case: missing invalidation of one resource in one case, mixed", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithPanic(t, `
            resource R {}

            fun test(n: Int, r1: @R, r2: @R) {
                switch n {
                    case n:
                        panic("")
                    case 2:
                        destroy r1
                    default:
                        destroy r1
                        destroy r2
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.ResourceLossError{}, errs[0])
	})

	t.Run("switch-case: missing invalidation of one resource in one case, mixed, transaction", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithPanic(t, `
            resource R {}

            transaction(n: Int) {

                let r1: @R
                let r2: @R

                prepare() {
                    self.r1 <- create R()
                    self.r2 <- create R()
                }

                execute {
                    switch n {
                        case n:
                            panic("")
                        case 2:
                            destroy self.r1
                        default:
                            destroy self.r1
                            destroy self.r2
                    }
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.ResourceFieldNotInvalidatedError{}, errs[0])
	})

	t.Run("switch-case: missing invalidation of one resource in default case", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithPanic(t, `
            resource R {}

            fun test(n: Int, r1: @R, r2: @R) {
                switch n {
                    case n:
                        panic("")
                    case 2:
                        panic("")
                    default:
                        destroy r1
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.ResourceLossError{}, errs[0])
	})

	t.Run("switch-case: missing invalidation of one resource in default case, transaction", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithPanic(t, `
            resource R {}

            transaction(n: Int) {

                let r1: @R
                let r2: @R

                prepare() {
                    self.r1 <- create R()
                    self.r2 <- create R()
                }

                execute {
                    switch n {
                        case n:
                            panic("")
                        case 2:
                            panic("")
                        default:
                            destroy self.r1
                    }
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.ResourceFieldNotInvalidatedError{}, errs[0])
	})

	t.Run("switch-case: missing invalidation of one resource in default case, mixed", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithPanic(t, `
            resource R {}

            fun test(n: Int, r1: @R, r2: @R) {
                switch n {
                    case n:
                        panic("")
                    case 2:
                        destroy r1
                        destroy r2
                    default:
                        destroy r1
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.ResourceLossError{}, errs[0])
	})

	t.Run("switch-case: missing invalidation of one resource in default case, mixed, transaction", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithPanic(t, `
            resource R {}

            transaction(n: Int) {

                let r1: @R
                let r2: @R

                prepare() {
                    self.r1 <- create R()
                    self.r2 <- create R()
                }

                execute {
                    switch n {
                        case n:
                            panic("")
                        case 2:
                            destroy self.r1
                            destroy self.r2
                        default:
                            destroy self.r1
                    }
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.ResourceFieldNotInvalidatedError{}, errs[0])
	})

	t.Run("switch-case: loss of all resources in one case", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithPanic(t, `
            resource R {}

            fun test(n: Int, r1: @R, r2: @R) {
                switch n {
                    case 1:
                        panic("")
                    case 2:
                        break
                    default:
                        panic("")
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.ResourceLossError{}, errs[0])
		assert.IsType(t, &sema.ResourceLossError{}, errs[1])
	})

	t.Run("switch-case: loss of all resources in one case, transaction", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithPanic(t, `
            resource R {}

            transaction(n: Int) {

                let r1: @R
                let r2: @R

                prepare() {
                    self.r1 <- create R()
                    self.r2 <- create R()
                }

                execute {
                    switch n {
                        case 1:
                            panic("")
                        case 2:
                            break
                        default:
                            panic("")
                    }
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.ResourceFieldNotInvalidatedError{}, errs[0])
		assert.IsType(t, &sema.ResourceFieldNotInvalidatedError{}, errs[1])
	})

	t.Run("switch-case: loss of all resources in one case, mixed", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithPanic(t, `
            resource R {}

            fun test(n: Int, r1: @R, r2: @R) {
                switch n {
                    case 1:
                        destroy r1
                        destroy r2
                    case 2:
                        break
                    default:
                        panic("")
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.ResourceLossError{}, errs[0])
		assert.IsType(t, &sema.ResourceLossError{}, errs[1])
	})

	t.Run("switch-case: loss of all resources in one case, mixed, transaction", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithPanic(t, `
            resource R {}

            transaction(n: Int) {

                let r1: @R
                let r2: @R

                prepare() {
                    self.r1 <- create R()
                    self.r2 <- create R()
                }

                execute {
                    switch n {
                        case 1:
                            destroy self.r1
                            destroy self.r2
                        case 2:
                            break
                        default:
                            panic("")
                    }
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.ResourceFieldNotInvalidatedError{}, errs[0])
		assert.IsType(t, &sema.ResourceFieldNotInvalidatedError{}, errs[1])
	})

	t.Run("switch-case: loss of all resources in default case", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithPanic(t, `
            resource R {}

            fun test(n: Int, r1: @R, r2: @R) {
                switch n {
                    case 1:
                        panic("")
                    case 2:
                        panic("")
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.ResourceLossError{}, errs[0])
		assert.IsType(t, &sema.ResourceLossError{}, errs[1])
	})

	t.Run("switch-case: loss of all resources in default case, transaction", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithPanic(t, `
            resource R {}

            transaction(n: Int) {

                let r1: @R
                let r2: @R

                prepare() {
                    self.r1 <- create R()
                    self.r2 <- create R()
                }

                execute {
                    switch n {
                        case 1:
                            panic("")
                        case 2:
                            panic("")
                    }
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.ResourceFieldNotInvalidatedError{}, errs[0])
		assert.IsType(t, &sema.ResourceFieldNotInvalidatedError{}, errs[1])
	})

	t.Run("switch-case: loss of all resources in default case, mixed", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithPanic(t, `
            resource R {}

            fun test(n: Int, r1: @R, r2: @R) {
                switch n {
                    case 1:
                        destroy r1
                        destroy r2
                    case 2:
                        panic("")
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.ResourceLossError{}, errs[0])
		assert.IsType(t, &sema.ResourceLossError{}, errs[1])
	})

	t.Run("switch-case: loss of all resources in default case, mixed, transaction", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithPanic(t, `
            resource R {}

            transaction(n: Int) {

                let r1: @R
                let r2: @R

                prepare() {
                    self.r1 <- create R()
                    self.r2 <- create R()
                }

                execute {
                    switch n {
                        case 1:
                            destroy self.r1
                            destroy self.r2
                        case 2:
                            panic("")
                    }
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.ResourceFieldNotInvalidatedError{}, errs[0])
		assert.IsType(t, &sema.ResourceFieldNotInvalidatedError{}, errs[1])
	})

	t.Run("switch-case: unreachable panic due to break", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithPanic(t, `
            resource R {}

            fun test(n: Int, r: @R) {
                switch n {
                    case 1:
                        break
                        panic("")  // unreachable
                    default:
                        panic("")
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.UnreachableStatementError{}, errs[0])
		assert.IsType(t, &sema.ResourceLossError{}, errs[1])
	})

	t.Run("switch-case: unreachable panic due to break, transaction", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithPanic(t, `
            resource R {}

            transaction(n: Int) {

                let r: @R

                prepare() {
                    self.r <- create R()
                }

                execute {
                    switch n {
                        case 1:
                            break
                            panic("")  // unreachable
                        default:
                            panic("")
                    }
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.UnreachableStatementError{}, errs[0])
		assert.IsType(t, &sema.ResourceFieldNotInvalidatedError{}, errs[1])
	})

	t.Run("switch-case: unreachable panic due to break, mixed", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithPanic(t, `
            resource R {}

            fun test(n: Int, r: @R) {
                switch n {
                    case 1:
                        break
                        panic("")  // unreachable
                    default:
                        destroy r
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.UnreachableStatementError{}, errs[0])
		assert.IsType(t, &sema.ResourceLossError{}, errs[1])
	})

	t.Run("switch-case: unreachable panic due to break, mixed, transaction", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithPanic(t, `
            resource R {}

            transaction(n: Int) {

                let r: @R

                prepare() {
                    self.r <- create R()
                }

                execute {
                    switch n {
                        case 1:
                            break
                            panic("")  // unreachable
                        default:
                            destroy self.r
                    }
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.UnreachableStatementError{}, errs[0])
		assert.IsType(t, &sema.ResourceFieldNotInvalidatedError{}, errs[1])
	})

	t.Run("switch-case: return in one case", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithPanic(t, `
            resource R {}

            fun test(n: Int, r: @R) {
                switch n {
                    case 1:
                        panic("")
                        return
                    default:
                        panic("")
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.UnreachableStatementError{}, errs[0])
	})

	t.Run("switch-case: return in one case, transaction", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithPanic(t, `
            resource R {}

            transaction(n: Int) {

                let r: @R

                prepare() {
                    self.r <- create R()
                }

                execute {
                    switch n {
                        case 1:
                            panic("")
                            return
                        default:
                            panic("")
                    }
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.UnreachableStatementError{}, errs[0])
		assert.IsType(t, &sema.ResourceFieldNotInvalidatedError{}, errs[1])
	})

	t.Run("switch-case: return in one case, mixed", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithPanic(t, `
            resource R {}

            fun test(n: Int, r: @R) {
                switch n {
                    case 1:
                        panic("")
                        return
                    default:
                        destroy r
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.UnreachableStatementError{}, errs[0])
	})

	t.Run("switch-case: return in one case, mixed, transaction", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithPanic(t, `
            resource R {}

            transaction(n: Int) {

                let r: @R

                prepare() {
                    self.r <- create R()
                }

                execute {
                    switch n {
                        case 1:
                            panic("")
                            return
                        default:
                            destroy self.r
                    }
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.UnreachableStatementError{}, errs[0])
	})

	t.Run("switch-case: break in one case", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithPanic(t, `
            resource R {}

            fun test(n: Int, r: @R) {
                switch n {
                    case 1:
                        panic("")
                        break
                    default:
                        panic("")
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.UnreachableStatementError{}, errs[0])
	})

	t.Run("switch-case: break in one case, transaction", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithPanic(t, `
            resource R {}

            transaction(n: Int) {

                let r: @R

                prepare() {
                    self.r <- create R()
                }

                execute {
                    switch n {
                        case 1:
                            panic("")
                            break
                        default:
                            panic("")
                    }
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.UnreachableStatementError{}, errs[0])
		assert.IsType(t, &sema.ResourceFieldNotInvalidatedError{}, errs[1])
	})

	t.Run("switch-case: break in one case, mixed", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithPanic(t, `
            resource R {}

            fun test(n: Int, r: @R) {
                switch n {
                    case 1:
                        panic("")
                        break
                    default:
                        destroy r
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.UnreachableStatementError{}, errs[0])
	})

	t.Run("switch-case: break in one case, mixed, transaction", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithPanic(t, `
            resource R {}

            transaction(n: Int) {

                let r: @R

                prepare() {
                    self.r <- create R()
                }

                execute {
                    switch n {
                        case 1:
                            panic("")
                            break
                        default:
                            destroy self.r
                    }
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.UnreachableStatementError{}, errs[0])
	})

	t.Run("switch-case: invalidation missing in default case", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithPanic(t, `
            resource R {}

            fun test(n: Int, r: @R) {
                switch n {
                    case 1:
                        panic("")
                    default:
                        return
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.ResourceLossError{}, errs[0])
	})

	t.Run("switch-case: invalidation missing in default case, transaction", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithPanic(t, `
            resource R {}

            transaction(n: Int) {

                let r: @R

                prepare() {
                    self.r <- create R()
                }

                execute {
                    switch n {
                        case 1:
                            panic("")
                        default:
                            return
                    }
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.ResourceFieldNotInvalidatedError{}, errs[0])
	})

	t.Run("switch-case: invalidation missing in default case, mixed", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithPanic(t, `
            resource R {}

            fun test(n: Int, r: @R) {
                switch n {
                    case 1:
                        panic("")
                    default:
                        return
                }
                destroy r
            }
        `)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.ResourceLossError{}, errs[0])
		assert.IsType(t, &sema.UnreachableStatementError{}, errs[1])
	})

	t.Run("switch-case: invalidation missing in default case, mixed, transaction", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithPanic(t, `
            resource R {}

            transaction(n: Int) {

                let r: @R

                prepare() {
                    self.r <- create R()
                }

                execute {
                    switch n {
                        case 1:
                            panic("")
                        default:
                            return
                    }
                    destroy self.r
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.UnreachableStatementError{}, errs[0])
		assert.IsType(t, &sema.ResourceFieldNotInvalidatedError{}, errs[1])
	})

	t.Run("while loop: unreachable panic due to break", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithPanic(t, `
            resource R {}

            fun test(r: @R) {
                while true {
                    break
                    panic("")  // unreachable
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.UnreachableStatementError{}, errs[0])
		assert.IsType(t, &sema.ResourceLossError{}, errs[1])
	})

	t.Run("while loop: unreachable panic due to break, transaction", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithPanic(t, `
            resource R {}

            transaction(n: Int) {

                let r: @R

                prepare() {
                    self.r <- create R()
                }

                execute {
                    while true {
                        break
                        panic("")  // unreachable
                    }
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.UnreachableStatementError{}, errs[0])
		assert.IsType(t, &sema.ResourceFieldNotInvalidatedError{}, errs[1])
	})

	t.Run("while loop: unreachable panic due to continue", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithPanic(t, `
            resource R {}

            fun test(r: @R) {
                while true {
                    continue
                    panic("")  // unreachable
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.UnreachableStatementError{}, errs[0])
		assert.IsType(t, &sema.ResourceLossError{}, errs[1])
	})

	t.Run("while loop: unreachable panic due to continue, transaction", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithPanic(t, `
            resource R {}

            transaction(n: Int) {

                let r: @R

                prepare() {
                    self.r <- create R()
                }

                execute {
                    while true {
                        continue
                        panic("")  // unreachable
                    }
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.UnreachableStatementError{}, errs[0])
		assert.IsType(t, &sema.ResourceFieldNotInvalidatedError{}, errs[1])
	})

	t.Run("while loop: unreachable panic due to return", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithPanic(t, `
            resource R {}

            fun test(r: @R) {
                while true {
                    return
                    panic("")  // unreachable
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 3)

		assert.IsType(t, &sema.ResourceLossError{}, errs[0])
		assert.IsType(t, &sema.UnreachableStatementError{}, errs[1])
		assert.IsType(t, &sema.ResourceLossError{}, errs[2])
	})

	t.Run("while loop: unreachable panic due to return, transaction", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithPanic(t, `
            resource R {}

            transaction(n: Int) {

                let r: @R

                prepare() {
                    self.r <- create R()
                }

                execute {
                    while true {
                        return
                        panic("")  // unreachable
                    }
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.UnreachableStatementError{}, errs[0])
		assert.IsType(t, &sema.ResourceFieldNotInvalidatedError{}, errs[1])
	})

	t.Run("for loop: unreachable panic due to break", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithPanic(t, `
            resource R {}

            fun test(r: @R) {
                for i in [] {
                    break
                    panic("")  // unreachable
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.UnreachableStatementError{}, errs[0])
		assert.IsType(t, &sema.ResourceLossError{}, errs[1])
	})

	t.Run("for loop: unreachable panic due to break, transaction", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithPanic(t, `
            resource R {}

            transaction(n: Int) {

                let r: @R

                prepare() {
                    self.r <- create R()
                }

                execute {
                    for i in [] {
                        break
                        panic("")  // unreachable
                    }
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.UnreachableStatementError{}, errs[0])
		assert.IsType(t, &sema.ResourceFieldNotInvalidatedError{}, errs[1])
	})

	t.Run("for loop: unreachable panic due to continue", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithPanic(t, `
            resource R {}

            fun test(r: @R) {
                for i in [] {
                    continue
                    panic("")  // unreachable
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.UnreachableStatementError{}, errs[0])
		assert.IsType(t, &sema.ResourceLossError{}, errs[1])
	})

	t.Run("for loop: unreachable panic due to continue, transaction", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithPanic(t, `
            resource R {}

            transaction(n: Int) {

                let r: @R

                prepare() {
                    self.r <- create R()
                }

                execute {
                    for i in [] {
                        continue
                        panic("")  // unreachable
                    }
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.UnreachableStatementError{}, errs[0])
		assert.IsType(t, &sema.ResourceFieldNotInvalidatedError{}, errs[1])
	})

	t.Run("for loop: unreachable panic due to return", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithPanic(t, `
            resource R {}

            fun test(r: @R) {
                for i in [] {
                    return
                    panic("")  // unreachable
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 3)
		assert.IsType(t, &sema.ResourceLossError{}, errs[0])
		assert.IsType(t, &sema.UnreachableStatementError{}, errs[1])
		assert.IsType(t, &sema.ResourceLossError{}, errs[2])
	})

	t.Run("for loop: unreachable panic due to return, transaction", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithPanic(t, `
            resource R {}

            transaction(n: Int) {

                let r: @R

                prepare() {
                    self.r <- create R()
                }

                execute {
                    for i in [] {
                        return
                        panic("")  // unreachable
                    }
                }
            }
        `)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.UnreachableStatementError{}, errs[0])
		assert.IsType(t, &sema.ResourceFieldNotInvalidatedError{}, errs[1])
	})
}

func TestCheckResourceInvalidationInConditionalExpression(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
            resource R {
                let foo: String

                init() {
                    self.foo = "hello"
                }
            }

            fun test(r1: @R, r2: @R) {
                let r3 <- true ? r1 : r2
                r1.foo
                r2.foo
                destroy r3
                destroy r1
                destroy r2
            }
        `)

	errs := RequireCheckerErrors(t, err, 2)
	assert.IsType(t, &sema.InvalidConditionalResourceOperandError{}, errs[0])
	assert.IsType(t, &sema.InvalidConditionalResourceOperandError{}, errs[1])
}

func TestCheckResourceInvalidationInNilCoalescingExpression(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
            resource R {
                let foo: String

                init() {
                    self.foo = "hello"
                }
            }

            fun test(r1: @R?, r2: @R) {
                let r3 <- r1 ?? r2
                r1?.foo
                r2.foo
                destroy r3
                destroy r1
                destroy r2
            }
        `)

	errs := RequireCheckerErrors(t, err, 3)
	assert.IsType(t, &sema.InvalidNilCoalescingRightResourceOperandError{}, errs[0])
	assert.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[1])
	assert.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[2])
}

func TestCheckResourceInvalidationInForceExpression(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
            resource R {}

            fun test(r: @R) {
                let copy <- r!
                destroy r
                destroy copy
            }
        `)

	errs := RequireCheckerErrors(t, err, 1)
	assert.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
}

func TestCheckResourceInvalidationWithMove(t *testing.T) {

	t.Parallel()

	t.Run("in conditional expression", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource R {}

            fun test(r1: @R, r2: @R) {
                let r3 <- true ? <- r1 : <- r2
                destroy r3
                destroy r1
                destroy r2
            }
        `)

		errs := RequireCheckerErrors(t, err, 6)

		assert.IsType(t, &sema.InvalidConditionalResourceOperandError{}, errs[0])
		assert.IsType(t, &sema.InvalidConditionalResourceOperandError{}, errs[1])
		assert.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[2])
		assert.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[3])
		assert.IsType(t, &sema.ResourceLossError{}, errs[4])
		assert.IsType(t, &sema.ResourceLossError{}, errs[5])
	})

	t.Run("in reference expression", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource R {}

            fun test(r: @R) {
                let ref = &(<- r) as &AnyResource
                destroy r
            }
        `)

		errs := RequireCheckerErrors(t, err, 2)
		assert.IsType(t, &sema.ResourceLossError{}, errs[0])
		assert.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[1])
	})

	t.Run("in casting expression", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource R {}

            fun test(r: @R) {
                let copy <- (<- r) as @R
                destroy r
                destroy copy
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
	})

	t.Run("in member expression", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource R {
                let foo: String

                init() {
                    self.foo = "hello"
                }
            }

            fun test(r: @R) {
                let foo = (<- r).foo
                destroy r
            }
        `)

		errs := RequireCheckerErrors(t, err, 2)
		assert.IsType(t, &sema.ResourceLossError{}, errs[0])
		assert.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[1])
	})

	t.Run("in index expression", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource R {}

            fun test(rs: @[R]) {
                let copy <- (<- rs)[0]
                destroy rs
                destroy copy
            }
        `)

		errs := RequireCheckerErrors(t, err, 3)
		assert.IsType(t, &sema.ResourceLossError{}, errs[0])
		assert.IsType(t, &sema.InvalidNestedResourceMoveError{}, errs[1])
		assert.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[2])
	})

	t.Run("in force expression", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource R {}

            fun test(r: @R) {
                let copy <- (<- r)!
                destroy r
                destroy copy
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
	})

	t.Run("in nil-coalescing expression", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource R {
                let foo: String

                init() {
                    self.foo = "hello"
                }
            }

            fun test(r1: @R?, r2: @R) {
                let r3 <- (<- r1) ?? (<- r2)
                r1?.foo
                r2.foo
                destroy r3
                destroy r1
                destroy r2
            }
        `)

		errs := RequireCheckerErrors(t, err, 6)

		assert.IsType(t, &sema.InvalidNilCoalescingRightResourceOperandError{}, errs[0])
		assert.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[1])
		assert.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[2])
		assert.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[3])
		assert.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[4])
		assert.IsType(t, &sema.ResourceLossError{}, errs[5])
	})

	t.Run("in destroy expression", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource R {}

            fun test(r: @R) {
                destroy (<- r)
                destroy r
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
	})

	t.Run("in function invocation expression", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource R {}

            fun f(_ r: @R) {
                destroy r
            }

            fun test(r: @R) {
                f(<- (<- r))
                destroy r
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
	})

	t.Run("in array expression", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource R {}

            fun test(r: @R) {
                let rs <- [<- (<- r)]
                destroy r
                destroy rs
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
	})

	t.Run("in dictionary expression", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource R {}

            fun test(r: @R) {
                let rs <- {"test": <- (<- r)}
                destroy r
                destroy rs
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
	})
}

func TestCheckResourceInvalidationWithConditionalExprInDestroy(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
        resource R {}

        fun test(r1: @R, r2: @R) {
            destroy true? r1 : r2
            destroy r1
            destroy r2
        }
    `)

	errs := RequireCheckerErrors(t, err, 2)
	assert.IsType(t, &sema.InvalidConditionalResourceOperandError{}, errs[0])
	assert.IsType(t, &sema.InvalidConditionalResourceOperandError{}, errs[1])
}

func TestCheckBadResourceInterface(t *testing.T) {
	t.Parallel()

	t.Run("bad resource interface: shorter", func(t *testing.T) {

		_, err := ParseAndCheck(t, "resource interface foo{struct d:foo{ struct d:foo{ }struct d:foo{ struct d:foo{ }}}}")

		errs := RequireCheckerErrors(t, err, 17)

		assert.IsType(t, &sema.InvalidNestedDeclarationError{}, errs[0])
		assert.IsType(t, &sema.InvalidNestedDeclarationError{}, errs[1])
		assert.IsType(t, &sema.RedeclarationError{}, errs[2])
		assert.IsType(t, &sema.RedeclarationError{}, errs[3])
		assert.IsType(t, &sema.InvalidNestedDeclarationError{}, errs[4])
		assert.IsType(t, &sema.RedeclarationError{}, errs[5])
		assert.IsType(t, &sema.RedeclarationError{}, errs[6])
		assert.IsType(t, &sema.RedeclarationError{}, errs[7])
		assert.IsType(t, &sema.RedeclarationError{}, errs[8])
		assert.IsType(t, &sema.CompositeKindMismatchError{}, errs[9])
		assert.IsType(t, &sema.RedeclarationError{}, errs[10])
		assert.IsType(t, &sema.CompositeKindMismatchError{}, errs[11])
		assert.IsType(t, &sema.ConformanceError{}, errs[12])
		assert.IsType(t, &sema.CompositeKindMismatchError{}, errs[13])
		assert.IsType(t, &sema.ConformanceError{}, errs[14])
		assert.IsType(t, &sema.CompositeKindMismatchError{}, errs[15])
		assert.IsType(t, &sema.ConformanceError{}, errs[16])
	})

	t.Run("bad resource interface: longer", func(t *testing.T) {

		_, err := ParseAndCheck(t, "resource interface foo{struct d:foo{ contract d:foo{ contract x:foo{ struct d{} contract d:foo{ contract d:foo {}}}}}}")

		errs := RequireCheckerErrors(t, err, 22)

		assert.IsType(t, &sema.InvalidNestedDeclarationError{}, errs[0])
		assert.IsType(t, &sema.InvalidNestedDeclarationError{}, errs[1])
		assert.IsType(t, &sema.RedeclarationError{}, errs[2])
		assert.IsType(t, &sema.InvalidNestedDeclarationError{}, errs[3])
		assert.IsType(t, &sema.InvalidNestedDeclarationError{}, errs[4])
		assert.IsType(t, &sema.RedeclarationError{}, errs[5])
		assert.IsType(t, &sema.RedeclarationError{}, errs[6])
		assert.IsType(t, &sema.InvalidNestedDeclarationError{}, errs[7])
		assert.IsType(t, &sema.RedeclarationError{}, errs[8])
		assert.IsType(t, &sema.RedeclarationError{}, errs[9])
		assert.IsType(t, &sema.RedeclarationError{}, errs[10])
		assert.IsType(t, &sema.CompositeKindMismatchError{}, errs[11])
		assert.IsType(t, &sema.ConformanceError{}, errs[12])
		assert.IsType(t, &sema.CompositeKindMismatchError{}, errs[13])
		assert.IsType(t, &sema.ConformanceError{}, errs[14])
		assert.IsType(t, &sema.RedeclarationError{}, errs[15])
		assert.IsType(t, &sema.CompositeKindMismatchError{}, errs[16])
		assert.IsType(t, &sema.RedeclarationError{}, errs[17])
		assert.IsType(t, &sema.CompositeKindMismatchError{}, errs[18])
		assert.IsType(t, &sema.ConformanceError{}, errs[19])
		assert.IsType(t, &sema.CompositeKindMismatchError{}, errs[20])
		assert.IsType(t, &sema.ConformanceError{}, errs[21])
	})
}

func TestCheckUnreachableResourceInvalidation(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheckWithPanic(t, `
        resource R {}

        fun test(_ r : @R): @R {
            if true {
                return <-r
            } else {
                if true {
                    return <-r
                } else {
                    panic("")
                }
            }
        }
    `)

	require.NoError(t, err)
}

func TestCheckConditionalResourceCreationAndReturn(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheckWithPanic(t, `
        resource R {}

        fun mint(id: UInt64): @R {
            if id > 100 {
                return <- create R()
            } else {
                panic("bad id")
            }
        }
    `)

	require.NoError(t, err)
}
