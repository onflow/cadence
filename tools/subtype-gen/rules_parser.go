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
)

//go:embed rules.yaml
var subtypeCheckingRules string

// Rule represents a single subtype rule
type Rule struct {
	Super     string `yaml:"super"`
	Predicate any    `yaml:"predicate"`
}

// RulesConfig represents the entire YAML configuration
type RulesConfig struct {
	Rules []Rule `yaml:"rules"`
}

type KeyValues = map[string]any

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
func parsePredicate(predicate any) (Predicate, error) {
	switch v := predicate.(type) {
	case string:
		switch v {
		case "always":
			return AlwaysPredicate{}, nil
		case "never":
			return NeverPredicate{}, nil
		default:
			return nil, fmt.Errorf("unsupported string predicate: %s", v)
		}
	case KeyValues:
		key, value := singleKeyValueFromMap(v)

		switch key {

		case "isResource":
			expr, err := parseExpression(value)
			if err != nil {
				return nil, err
			}
			return IsResourcePredicate{Expression: expr}, nil

		case "isAttachment":
			expr, err := parseExpression(value)
			if err != nil {
				return nil, err
			}
			return IsAttachmentPredicate{Expression: expr}, nil

		case "isHashableStruct":
			expr, err := parseExpression(value)
			if err != nil {
				return nil, err
			}
			return IsHashableStructPredicate{Expression: expr}, nil

		case "isStorable":
			expr, err := parseExpression(value)
			if err != nil {
				return nil, err
			}
			return IsStorablePredicate{Expression: expr}, nil

		case "equals":
			sourceExpr, targetExpr, err := parseSourceAndTarget(key, value)
			if err != nil {
				return nil, err
			}

			return EqualsPredicate{
				Source: sourceExpr,
				Target: targetExpr,
			}, nil

		case "deepEquals":
			sourceExpr, targetExpr, err := parseSourceAndTarget(key, value)
			if err != nil {
				return nil, err
			}

			return DeepEqualsPredicate{
				Source: sourceExpr,
				Target: targetExpr,
			}, nil

		case "subtype":
			superType, subType, err := parseSuperAndSubExpressions(key, value)
			if err != nil {
				return nil, err
			}

			return SubtypePredicate{
				Sub:   subType,
				Super: superType,
			}, nil

		case "and":
			and, ok := value.([]any)
			if !ok {
				return nil, fmt.Errorf("expected a list of predicates, got %T", value)
			}

			var predicates []Predicate
			for _, cond := range and {
				predicate, err := parsePredicate(cond)
				if err != nil {
					return nil, err
				}
				predicates = append(predicates, predicate)
			}

			return AndPredicate{Predicates: predicates}, nil

		case "or":
			or, ok := value.([]any)
			if !ok {
				return nil, fmt.Errorf("expected a list of predicates, got %T", value)
			}

			var predicates []Predicate
			for _, cond := range or {
				predicate, err := parsePredicate(cond)
				if err != nil {
					return nil, err
				}
				predicates = append(predicates, predicate)
			}

			return OrPredicate{Predicates: predicates}, nil

		case "not":
			innerPredicate, err := parsePredicate(value)
			if err != nil {
				return nil, err
			}

			return NotPredicate{
				Predicate: innerPredicate,
			}, nil

		case "permits":
			superType, subType, err := parseSuperAndSubExpressions(key, value)
			if err != nil {
				return nil, err
			}

			return PermitsPredicate{
				Sub:   subType,
				Super: superType,
			}, nil

		case "mustType":
			keyValues, ok := value.(KeyValues)
			if !ok {
				return nil, fmt.Errorf("expected KeyValues, got %T", value)
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

			expectedType := parseType(typ.(string))

			return TypeAssertionPredicate{
				Source: sourceExpr,
				Type:   expectedType,
			}, nil

		case "setContains":
			keyValues, ok := value.(KeyValues)
			if !ok {
				return nil, fmt.Errorf("expected KeyValues, got %T", value)
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
				Set:     setExpr,
				Element: elementExpr,
			}, nil

		case "isIntersectionSubset":
			superType, subType, err := parseSuperAndSubExpressions(key, value)
			if err != nil {
				return nil, err
			}

			return IsIntersectionSubsetPredicate{
				Sub:   subType,
				Super: superType,
			}, nil

		case "returnCovariant":
			sourceExpr, targetExpr, err := parseSourceAndTarget(key, value)
			if err != nil {
				return nil, err
			}

			return ReturnCovariantPredicate{
				Source: sourceExpr,
				Target: targetExpr,
			}, nil

		case "isParameterizedSubtype":
			superType, subType, err := parseSuperAndSubExpressions(key, value)
			if err != nil {
				return nil, err
			}

			return IsParameterizedSubtypePredicate{
				Sub:   subType,
				Super: superType,
			}, nil

		case "forAll":
			keyValues, ok := value.(KeyValues)
			if !ok {
				return nil, fmt.Errorf("expected KeyValues, got %T", value)
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
				Source:    sourceExpr,
				Target:    targetExpr,
				Predicate: innerPredicate,
			}, nil
		default:
			return nil, fmt.Errorf("unsupported predicate: %s", key)
		}

	default:
		return nil, fmt.Errorf("unsupported predicate type: %T", predicate)
	}
}

func parseSourceAndTarget(predicateName string, value any) (Expression, Expression, error) {
	keyValues, ok := value.(KeyValues)
	if !ok {
		return nil, nil, fmt.Errorf("expected KeyValues, got %T", value)
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

func parseSuperAndSubExpressions(predicateName string, value any) (Expression, Expression, error) {
	keyValues, ok := value.(KeyValues)
	if !ok {
		return nil, nil, fmt.Errorf("expected KeyValues, got %T", value)
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

func singleKeyValueFromMap(v KeyValues) (string, any) {
	if len(v) != 1 {
		panic(fmt.Errorf("expected exactly one key value pair"))
	}

	for key, value := range v { //nolint:maprange
		return key, value
	}

	return "", nil
}

func parseExpression(expr any) (Expression, error) {
	switch expr := expr.(type) {
	case string:
		return parseSimpleExpression(expr), nil
	case KeyValues:
		key, value := singleKeyValueFromMap(expr)

		switch key {
		case "oneOf":
			list, ok := value.([]any)
			if !ok {
				return nil, fmt.Errorf("expected a list of predicates, got %T", value)
			}

			var expressions []Expression
			for _, item := range list {
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
		return nil, fmt.Errorf("unsupported expression: %s", expr)
	}
}

// parseSimpleExpression parses an expression that is represented as a string data in YAML.
func parseSimpleExpression(expr any) Expression {
	switch v := expr.(type) {
	case string:
		parts := strings.Split(v, ".")
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
	default:
		panic(fmt.Errorf("unsupported expression type: %T", expr))
	}
}
