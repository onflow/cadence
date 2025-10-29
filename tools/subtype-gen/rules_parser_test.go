/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Flow Foundation
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

package subtype_gen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/goccy/go-yaml/ast"
	"github.com/goccy/go-yaml/parser"
)

func TestParsingComments(t *testing.T) {
	t.Parallel()

	yamlContent := `
      # comment 1
      rules:
        # comment 2
        - super: AnyType
          predicate:

            # comment 3
            and:

              # comment 4
              - equals:
                  # comment 5
                  source: sub
                  target: super

              # comment 6
              - subtype:
                  # comment 7
                  # comment 8
                  sub: sub
                  super: AnyType
        `

	t.Run("in AST", func(t *testing.T) {

		t.Parallel()

		file, err := parser.ParseBytes(
			[]byte(yamlContent),
			parser.ParseComments,
		)
		require.NoError(t, err)

		docs := file.Docs
		require.Len(t, docs, 1)

		doc := docs[0]
		require.IsType(t, &ast.MappingNode{}, doc.Body)
		rules := doc.Body.(*ast.MappingNode)

		_, value, comments, err := singleKeyValueFromMap(rules)
		require.NoError(t, err)
		require.Equal(t, "# comment 1", comments)

		require.IsType(t, &ast.SequenceNode{}, value)
		rulesList := value.(*ast.SequenceNode)

		// Comment on top of the first element of a list, belongs to the list itself.
		rulesListComment := rulesList.Comment
		require.NotEmpty(t, rulesListComment)
		require.Equal(t, "# comment 2", rulesListComment.String())

		require.Len(t, rulesList.Values, 1)
		firstRule := rulesList.Values[0]

		// Get the first rule
		require.IsType(t, &ast.MappingNode{}, firstRule)
		firstRuleFields := firstRule.(*ast.MappingNode)
		require.Len(t, firstRuleFields.Values, 2)

		// Predicate is the second field (at index 1)
		predicateField := firstRuleFields.Values[1]
		_, value, comments, err = stringKeyAndValueFromPair(predicateField)
		require.NoError(t, err)
		// no comments for the 'predicate' field.
		require.Empty(t, comments)

		// 'and' predicate
		require.IsType(t, &ast.MappingNode{}, value)
		andPredicate := value.(*ast.MappingNode)
		_, value, comments, err = singleKeyValueFromMap(andPredicate)
		require.NoError(t, err)
		require.Equal(t, "# comment 3", comments)

		require.IsType(t, &ast.SequenceNode{}, value)
		innerPredicateList := value.(*ast.SequenceNode)

		// Comment on top of the first element of a list, belongs to the list itself.
		innerPredicateListComment := innerPredicateList.Comment
		require.NotEmpty(t, innerPredicateListComment)
		require.Equal(t, "# comment 4", innerPredicateListComment.String())

		// Comments for each item is stored separately
		itemComments := innerPredicateList.ValueHeadComments
		require.Len(t, itemComments, 2)
		// As stated above, first items shouldn't have comments.
		// It gets captured as the comment of the entire list.
		require.Empty(t, itemComments[0])
		require.Equal(t, "# comment 6", itemComments[1].String())

		innerPredicates := innerPredicateList.Values
		require.Len(t, innerPredicates, 2)

		// Equals predicate
		equalsPredicate := innerPredicates[0]
		require.Empty(t, equalsPredicate.GetComment())

		require.IsType(t, &ast.MappingNode{}, equalsPredicate)
		equalsPredicateMap := equalsPredicate.(*ast.MappingNode)
		_, value, comments, err = singleKeyValueFromMap(equalsPredicateMap)
		require.NoError(t, err)
		require.Empty(t, comments)

		equalsPredicateFields := value.(*ast.MappingNode)

		require.Len(t, equalsPredicateFields.Values, 2)
		source := equalsPredicateFields.Values[0]
		require.Equal(t, "# comment 5", source.Comment.String())

		// Subtype predicate
		subtypePredicate := innerPredicates[1]
		require.Empty(t, subtypePredicate.GetComment())

		require.IsType(t, &ast.MappingNode{}, subtypePredicate)
		subtypePredicateMap := subtypePredicate.(*ast.MappingNode)
		_, value, comments, err = singleKeyValueFromMap(subtypePredicateMap)
		require.NoError(t, err)
		require.Empty(t, comments)

		subtypePredicateFields := value.(*ast.MappingNode)

		require.Len(t, subtypePredicateFields.Values, 2)
		subType := subtypePredicateFields.Values[0]
		require.Equal(t, "# comment 7\n# comment 8", subType.Comment.String())
	})

	t.Run("in unmarshalled tree", func(t *testing.T) {

		t.Parallel()

		file, err := parser.ParseBytes(
			[]byte(yamlContent),
			parser.ParseComments,
		)
		require.NoError(t, err)

		docs := file.Docs
		require.Len(t, docs, 1)

		rules, err := parseRulesFromDocument(docs[0])
		require.NoError(t, err)

		require.Equal(
			t,
			RulesFile{
				description: "# comment 1",
				Rules: []Rule{
					{
						description: "# comment 2",
						SuperType: SimpleType{
							name: "Any",
						},
						Predicate: AndPredicate{
							description: "# comment 3",
							Predicates: []Predicate{
								EqualsPredicate{
									description: "# comment 4\n# comment 5",
									Source: IdentifierExpression{
										Name: "sub",
									},
									Target: IdentifierExpression{
										Name: "super",
									},
								},

								SubtypePredicate{
									description: "# comment 6\n# comment 7\n# comment 8",
									Sub: IdentifierExpression{
										Name: "sub",
									},
									Super: TypeExpression{
										Type: SimpleType{
											name: "Any",
										},
									},
								},
							},
						},
					},
				},
			},
			rules,
		)
	})

}
