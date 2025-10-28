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
	_ "embed"
	"fmt"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/ast"
)

//go:embed rules.yaml
var subtypeCheckingRules string

// Rule represents a single subtype rule
type Rule struct {
	Super     string   `yaml:"super"`
	Predicate ast.Node `yaml:"predicate"`
}

// RulesFile represents the entire YAML configuration
type RulesFile struct {
	Rules []Rule `yaml:"rules"`
}

type KeyValues = map[string]ast.Node

// ParseRules reads and parses the YAML rules file
func ParseRules() ([]Rule, error) {
	var config RulesFile
	if err := yaml.Unmarshal([]byte(subtypeCheckingRules), &config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	return config.Rules, nil
}

// parseType parses a type with just a name.
func parseType(typePlaceHolder string) Type {
	if !strings.HasSuffix(typePlaceHolder, TypePlaceholderSuffix) {
		panic(fmt.Errorf(
			"type name %#[1]q is not suffixed with %#[2]q."+
				" Replace the type-name with `%[1]s%[2]s`",
			typePlaceHolder,
			TypePlaceholderSuffix,
		))
	}

	typeName := strings.TrimSuffix(typePlaceHolder, TypePlaceholderSuffix)

	switch typeName {
	case TypePlaceholderOptional,
		TypePlaceholderDictionary,
		TypePlaceholderVariableSized,
		TypePlaceholderConstantSized,
		TypePlaceholderReference,
		TypePlaceholderComposite,
		TypePlaceholderInterface,
		TypePlaceholderFunction,
		TypePlaceholderIntersection,
		TypePlaceholderParameterized,
		TypePlaceholderConforming:
		return ComplexType{
			name: typeName,
		}
	default:
		return SimpleType{
			name: typeName,
		}
	}
}

func parseMemberExpression(names []string) Expression {
	size := len(names)
	if size == 1 {
		return IdentifierExpression{
			Name: names[0],
		}
	}

	lastIndex := len(names) - 1

	return MemberExpression{
		Parent:     parseMemberExpression(names[:lastIndex]),
		MemberName: names[lastIndex],
	}
}

// parsePredicate parses a predicate from the YAML
func parsePredicate(predicate ast.Node) (Predicate, error) {

	description := nodeDescription(predicate)

	switch predicate := predicate.(type) {

	case *ast.StringNode:
		switch predicate.Value {
		case "always":
			return AlwaysPredicate{
				description: description,
			}, nil
		case "never":
			return NeverPredicate{
				description: description,
			}, nil
		default:
			return nil, fmt.Errorf("unsupported string predicate: %s", predicate.Value)
		}

	case *ast.MappingNode:
		key, value, err := singleKeyValueFromMap(predicate)
		if err != nil {
			return nil, err
		}

		switch key {

		case "isResource":
			expr, err := parseExpression(value)
			if err != nil {
				return nil, err
			}
			return IsResourcePredicate{
				description: description,
				Expression:  expr,
			}, nil

		case "isAttachment":
			expr, err := parseExpression(value)
			if err != nil {
				return nil, err
			}
			return IsAttachmentPredicate{
				description: description,
				Expression:  expr,
			}, nil

		case "isHashableStruct":
			expr, err := parseExpression(value)
			if err != nil {
				return nil, err
			}
			return IsHashableStructPredicate{
				description: description,
				Expression:  expr,
			}, nil

		case "isStorable":
			expr, err := parseExpression(value)
			if err != nil {
				return nil, err
			}
			return IsStorablePredicate{
				description: description,
				Expression:  expr,
			}, nil

		case "equals":
			sourceExpr, targetExpr, err := parseSourceAndTarget(key, value)
			if err != nil {
				return nil, err
			}

			return EqualsPredicate{
				description: description,
				Source:      sourceExpr,
				Target:      targetExpr,
			}, nil

		case "deepEquals":
			sourceExpr, targetExpr, err := parseSourceAndTarget(key, value)
			if err != nil {
				return nil, err
			}

			return DeepEqualsPredicate{
				description: description,
				Source:      sourceExpr,
				Target:      targetExpr,
			}, nil

		case "subtype":
			superType, subType, err := parseSuperAndSubExpressions(key, value)
			if err != nil {
				return nil, err
			}

			return SubtypePredicate{
				description: description,
				Sub:         subType,
				Super:       superType,
			}, nil

		case "and":
			innerPredicates, err := nodeAsList(value)
			if err != nil {
				return nil, err
			}

			var predicates []Predicate
			for _, cond := range innerPredicates {
				predicate, err := parsePredicate(cond)
				if err != nil {
					return nil, err
				}
				predicates = append(predicates, predicate)
			}

			return AndPredicate{
				description: description,
				Predicates:  predicates,
			}, nil

		case "or":
			innerPredicates, err := nodeAsList(value)
			if err != nil {
				return nil, err
			}

			var predicates []Predicate
			for _, cond := range innerPredicates {
				predicate, err := parsePredicate(cond)
				if err != nil {
					return nil, err
				}
				predicates = append(predicates, predicate)
			}

			return OrPredicate{
				description: description,
				Predicates:  predicates,
			}, nil

		case "not":
			innerPredicate, err := parsePredicate(value)
			if err != nil {
				return nil, err
			}

			return NotPredicate{
				description: description,
				Predicate:   innerPredicate,
			}, nil

		case "permits":
			superType, subType, err := parseSuperAndSubExpressions(key, value)
			if err != nil {
				return nil, err
			}

			return PermitsPredicate{
				description: description,
				Sub:         subType,
				Super:       superType,
			}, nil

		case "mustType":
			keyValues, err := nodeAsKeyValues(value)
			if err != nil {
				return nil, err
			}

			// Get source
			source, ok := keyValues["source"]
			if !ok {
				return nil, fmt.Errorf("cannot find `source` property for `mustType` predicate")
			}

			sourceExpr, err := parseExpression(source)
			if err != nil {
				return nil, err
			}

			// Get target
			typ, ok := keyValues["type"]
			if !ok {
				return nil, fmt.Errorf("cannot find `target` property for `mustType` predicate")
			}

			typeStr, ok := typ.(*ast.StringNode)
			if !ok {
				return nil, fmt.Errorf("type placeholder must be a string, got %s", typ)
			}

			expectedType := parseType(typeStr.Value)

			return TypeAssertionPredicate{
				description: description,
				Source:      sourceExpr,
				Type:        expectedType,
			}, nil

		case "setContains":
			keyValues, err := nodeAsKeyValues(value)
			if err != nil {
				return nil, err
			}

			// Get the set
			set, ok := keyValues["set"]
			if !ok {
				return nil, fmt.Errorf("cannot find `set` property for `setContains` predicate")
			}

			setExpr, err := parseExpression(set)
			if err != nil {
				return nil, err
			}

			// Get element
			element, ok := keyValues["element"]
			if !ok {
				return nil, fmt.Errorf("cannot find `element` property for `setContains` predicate")
			}

			elementExpr, err := parseExpression(element)
			if err != nil {
				return nil, err
			}

			return SetContainsPredicate{
				description: description,
				Set:         setExpr,
				Element:     elementExpr,
			}, nil

		case "isIntersectionSubset":
			superType, subType, err := parseSuperAndSubExpressions(key, value)
			if err != nil {
				return nil, err
			}

			return IsIntersectionSubsetPredicate{
				description: description,
				Sub:         subType,
				Super:       superType,
			}, nil

		case "returnCovariant":
			sourceExpr, targetExpr, err := parseSourceAndTarget(key, value)
			if err != nil {
				return nil, err
			}

			return ReturnCovariantPredicate{
				description: description,
				Source:      sourceExpr,
				Target:      targetExpr,
			}, nil

		case "isParameterizedSubtype":
			superType, subType, err := parseSuperAndSubExpressions(key, value)
			if err != nil {
				return nil, err
			}

			return IsParameterizedSubtypePredicate{
				description: description,
				Sub:         subType,
				Super:       superType,
			}, nil

		case "forAll":
			keyValues, err := nodeAsKeyValues(value)
			if err != nil {
				return nil, err
			}

			// Get source
			source, ok := keyValues["source"]
			if !ok {
				return nil, fmt.Errorf("cannot find `source` property for %#q predicate", key)
			}

			sourceExpr, err := parseExpression(source)
			if err != nil {
				return nil, err
			}

			// Get target
			target, ok := keyValues["target"]
			if !ok {
				return nil, fmt.Errorf("cannot find `target` property for %#q predicate", key)
			}

			targetExpr, err := parseExpression(target)
			if err != nil {
				return nil, err
			}

			// Get inner predicate
			predicate, ok := keyValues["predicate"]
			if !ok {
				return nil, fmt.Errorf("cannot find `predicate` property for %#q predicate", key)
			}

			innerPredicate, err := parsePredicate(predicate)
			if err != nil {
				return nil, err
			}

			return ForAllPredicate{
				description: description,
				Source:      sourceExpr,
				Target:      targetExpr,
				Predicate:   innerPredicate,
			}, nil
		default:
			return nil, fmt.Errorf("unsupported predicate: %s", key)
		}

	default:
		return nil, fmt.Errorf("unsupported predicate type: %T", predicate)
	}
}

func nodeDescription(predicate ast.Node) description {
	comment := predicate.GetComment()
	if comment == nil {
		return ""
	}

	// TODO: improve
	return description(comment.String())
}

func nodeAsKeyValues(node ast.Node) (KeyValues, error) {
	mappingNode, ok := node.(*ast.MappingNode)
	if !ok {
		return nil, fmt.Errorf("expected KeyValues, got %s", node.Type())
	}

	values := mappingNode.Values
	keyValues := make(KeyValues, len(values))

	for _, pair := range values {
		strKey, value, err := stringKeyAndValueFromPair(pair)
		if err != nil {
			return nil, err
		}

		keyValues[strKey] = value
	}

	return keyValues, nil
}

func stringKeyAndValueFromPair(pair *ast.MappingValueNode) (string, ast.Node, error) {
	key := pair.Key
	strKey, ok := key.(*ast.StringNode)
	if !ok {
		return "", nil, fmt.Errorf("expected string-type key, got %s", key.Type())
	}

	value := pair.Value
	return strKey.Value, value, nil
}

func nodeAsList(node ast.Node) ([]ast.Node, error) {
	sequenceNode, ok := node.(*ast.SequenceNode)
	if !ok {
		return nil, fmt.Errorf("expected a list, got %s", node.Type())
	}

	return sequenceNode.Values, nil
}

func parseSourceAndTarget(predicateName string, value ast.Node) (Expression, Expression, error) {
	keyValues, err := nodeAsKeyValues(value)
	if err != nil {
		return nil, nil, err
	}

	// Get source
	source, ok := keyValues["source"]
	if !ok {
		return nil, nil, fmt.Errorf("cannot find `source` property for %#q predicate", predicateName)
	}

	sourceExpr, err := parseExpression(source)
	if err != nil {
		return nil, nil, err
	}

	// Get target
	target, ok := keyValues["target"]
	if !ok {
		return nil, nil, fmt.Errorf("cannot find `target` property for %#q predicate", predicateName)
	}

	targetExpr, err := parseExpression(target)
	if err != nil {
		return nil, nil, err
	}

	return sourceExpr, targetExpr, nil
}

func parseSuperAndSubExpressions(predicateName string, value ast.Node) (Expression, Expression, error) {
	keyValues, err := nodeAsKeyValues(value)
	if err != nil {
		return nil, nil, err
	}

	// Get super type
	super, ok := keyValues["super"]
	if !ok {
		return nil, nil, fmt.Errorf("cannot find `super` property for %#q predicate", predicateName)
	}

	superType, err := parseExpression(super)
	if err != nil {
		return nil, nil, err
	}

	// Get subtype
	sub, ok := keyValues["sub"]
	if !ok {
		return nil, nil, fmt.Errorf("cannot find `sub` property for %#q predicate", predicateName)
	}

	subType, err := parseExpression(sub)
	if err != nil {
		return nil, nil, err
	}

	return superType, subType, nil
}

func singleKeyValueFromMap(mappingNode *ast.MappingNode) (string, ast.Node, error) {
	keyValuePairs := mappingNode.Values
	if len(keyValuePairs) != 1 {
		return "", nil, fmt.Errorf("expected exactly one key value pair")
	}

	return stringKeyAndValueFromPair(keyValuePairs[0])
}

func parseExpression(expr ast.Node) (Expression, error) {
	switch expr := expr.(type) {
	case *ast.StringNode:
		return parseSimpleExpression(expr.Value), nil
	case *ast.MappingNode:
		key, value, err := singleKeyValueFromMap(expr)
		if err != nil {
			return nil, err
		}

		switch key {
		case "oneOf":
			list, ok := value.(*ast.SequenceNode)
			if !ok {
				return nil, fmt.Errorf("expected a list of predicates, got %s", value.Type())
			}

			var expressions []Expression
			for _, item := range list.Values {
				itemExpr, err := parseExpression(item)
				if err != nil {
					return nil, err
				}

				expressions = append(expressions, itemExpr)
			}

			return OneOfExpression{Expressions: expressions}, nil
		default:
			return nil, fmt.Errorf("unsupported key: %s", key)
		}

	default:
		return nil, fmt.Errorf("unsupported expression: %v", expr)
	}
}

// parseSimpleExpression parses an expression that is represented as a string data in YAML.
func parseSimpleExpression(expr string) Expression {
	parts := strings.Split(expr, ".")
	if len(parts) == 1 {
		identifier := parts[0]

		if strings.HasSuffix(identifier, "Type") {
			return TypeExpression{
				Type: parseType(identifier),
			}
		}

		return IdentifierExpression{
			Name: identifier,
		}
	}

	return parseMemberExpression(parts)
}
