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

	"gopkg.in/yaml.v3"
)

//go:embed rules.yaml
var subtypeCheckingRules string

// Rule represents a single subtype rule
type Rule struct {
	Super     string    `yaml:"super"`
	Predicate yaml.Node `yaml:"predicate"`
}

// RulesConfig represents the entire YAML configuration
type RulesConfig struct {
	Rules []Rule `yaml:"rules"`
}

type KeyValues = map[string]*yaml.Node

// ParseRules reads and parses the YAML rules file
func ParseRules() ([]Rule, error) {
	var config RulesConfig
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
func parsePredicate(predicate *yaml.Node) (Predicate, error) {

	description := description(predicate.HeadComment)

	switch predicate.Kind {
	case yaml.ScalarNode:
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
	case yaml.MappingNode:

		key, value := singleKeyValueFromMap(predicate)

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

			if typ.Kind != yaml.ScalarNode {
				return nil, fmt.Errorf("type placeholder must be a string, got %s", typ.Tag)
			}

			expectedType := parseType(typ.Value)

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

func nodeAsKeyValues(node *yaml.Node) (KeyValues, error) {
	if node.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("expected KeyValues, got %s", node.Tag)
	}

	size := len(node.Content)
	keyValues := make(KeyValues, size/2)

	for i := 0; i < size; i += 2 {
		key := node.Content[i]
		value := node.Content[i+1]

		keyValues[key.Value] = value
	}

	return keyValues, nil
}

func nodeAsList(node *yaml.Node) ([]*yaml.Node, error) {
	if node.Kind != yaml.SequenceNode {
		return nil, fmt.Errorf("expected a list, got %s", node.Tag)
	}

	return node.Content, nil
}

func parseSourceAndTarget(predicateName string, value *yaml.Node) (Expression, Expression, error) {
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

func parseSuperAndSubExpressions(predicateName string, value *yaml.Node) (Expression, Expression, error) {
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

func singleKeyValueFromMap(node *yaml.Node) (string, *yaml.Node) {
	keyValues := node.Content

	if len(keyValues) != 2 {
		panic(fmt.Errorf("expected exactly one key value pair"))
	}

	key := keyValues[0]
	value := keyValues[1]

	return key.Value, value
}

func parseExpression(expr *yaml.Node) (Expression, error) {
	switch expr.Kind {
	case yaml.ScalarNode:
		return parseSimpleExpression(expr.Value), nil
	case yaml.MappingNode:
		key, value := singleKeyValueFromMap(expr)

		switch key {
		case "oneOf":
			if value.Kind != yaml.SequenceNode {
				return nil, fmt.Errorf("expected a list of predicates, got %s", value.Tag)
			}

			var expressions []Expression
			for _, item := range value.Content {
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
