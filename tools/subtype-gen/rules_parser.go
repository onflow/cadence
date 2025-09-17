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

	yaml "gopkg.in/yaml.v3"
)

//go:embed rules.yaml
var subtypeCheckingRules string

// Rule represents a single subtype rule
type Rule struct {
	Super     any `yaml:"super"`
	Sub       any `yaml:"sub"`
	Predicate any `yaml:"predicate"`
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

// parseType parses type information from YAML data and returns a `Type`.
func parseType(typeData any) Type {
	switch v := typeData.(type) {
	case string:
		return parseSimpleType(v)
	case KeyValues:
		return parseComplexType(v)
	default:
		panic(fmt.Errorf("unsupported type data: %T", typeData))
	}
}

// parseSimpleType parses a type with just a name.
func parseSimpleType(typeName string) Type {
	return &SimpleType{name: typeName}
}

// parseComplexType parses complex types with parameters
func parseComplexType(v KeyValues) Type {
	// TODO:
	panic(fmt.Errorf("complex types are not yet supported"))
}

// parseRulePredicate parses a rule predicate from YAML
func parseRulePredicate(rule any) (Predicate, error) {
	switch v := rule.(type) {
	case string:
		switch v {
		case "always":
			return AlwaysPredicate{}, nil
		default:
			return nil, fmt.Errorf("unsupported string rule: %s", v)
		}
	case KeyValues:
		key, value := singleKeyValueFromMap(v)

		switch key {

		case "isResource":
			typ := parseType(value)
			return IsResourcePredicate{Type: typ}, nil

		case "isAttachment":
			typ := parseType(value)
			return IsAttachmentPredicate{Type: typ}, nil

		case "isHashableStruct":
			typ := parseType(value)
			return IsHashableStructPredicate{Type: typ}, nil

		case "isStorable":
			typ := parseType(value)
			return IsStorablePredicate{Type: typ}, nil

		case "equals":
			equals, ok := value.(KeyValues)
			if !ok {
				return nil, fmt.Errorf("expected KeyValues, got %T", value)
			}

			target, err := parseTypeOrOneOfTypes(equals["target"])
			if err != nil {
				return nil, err
			}

			sourceType := parseType(equals["source"])
			return EqualsPredicate{
				Source: sourceType,
				Target: target,
			}, nil

		case "subtype":
			equals, ok := value.(KeyValues)
			if !ok {
				return nil, fmt.Errorf("expected KeyValues, got %T", value)
			}

			super, err := parseTypeOrOneOfTypes(equals["super"])
			if err != nil {
				return nil, err
			}

			subType := parseType(equals["sub"])
			return SubtypePredicate{
				Sub:   subType,
				Super: super,
			}, nil

		case "and":
			and, ok := value.([]any)
			if !ok {
				return nil, fmt.Errorf("expected []any, got %T", value)
			}

			var predicates []Predicate
			for _, cond := range and {
				predicate, err := parseRulePredicate(cond)
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
				predicate, err := parseRulePredicate(cond)
				if err != nil {
					return nil, err
				}
				predicates = append(predicates, predicate)
			}

			return OrPredicate{Predicates: predicates}, nil

		case "not":
			innerPredicate, err := parseRulePredicate(value)
			if err != nil {
				return nil, err
			}

			return NotPredicate{
				Predicate: innerPredicate,
			}, nil

		case "permits":
			list, ok := value.([]Type)
			if !ok {
				return nil, fmt.Errorf("expected a list of types, got %T", value)
			}
			return PermitsPredicate{Types: list}, nil

		case "contains":
			list, ok := value.([]Type)
			if !ok {
				return nil, fmt.Errorf("expected a list of types, got %T", value)
			}
			return ContainsPredicate{Types: list}, nil

		default:
			return nil, fmt.Errorf("unsupported rule predicate: %v", v)
		}

	default:
		return nil, fmt.Errorf("unsupported rule type: %T", rule)
	}
}

func singleKeyValueFromMap(v KeyValues) (string, any) {
	if len(v) != 1 {
		panic(fmt.Errorf("expected exactly one key value pair"))
	}

	for key, value := range v {
		return key, value
	}

	return "", nil
}

func parseTypeOrOneOfTypes(target any) (any, error) {
	// TODO:
	switch target := target.(type) {
	case string:
		return parseType(target), nil
	case KeyValues:
		key, value := singleKeyValueFromMap(target)

		switch key {
		case "oneOfTypes":
			list, ok := value.([]any)
			if !ok {
				return nil, fmt.Errorf("expected a list of predicates, got %T", value)
			}

			var types []Type
			for _, item := range list {
				types = append(types, parseType(item))
			}

			return OneOfTypes{Types: types}, nil
		default:
			return nil, fmt.Errorf("unsupported predicate: %s", key)
		}

	}

	return target, nil
}
