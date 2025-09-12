package main

import (
	"fmt"
	"os"

	yaml "gopkg.in/yaml.v3"
)

// Rule represents a single subtype rule
type Rule struct {
	Super     any `yaml:"super"`
	Sub       any `yaml:"sub"`
	Condition any `yaml:"condition"` // Support both 'rule' and 'condition' fields
}

// RulesConfig represents the entire YAML configuration
type RulesConfig struct {
	Rules []Rule `yaml:"rules"`
}

// RuleCondition represents different types of conditions in rules
type RuleCondition interface {
	GetType() string
}

// AlwaysCondition represents an always-true condition
type AlwaysCondition struct{}

func (a AlwaysCondition) GetType() string { return "always" }

// IsResourceCondition represents a resource type check
type IsResourceCondition struct {
	Type any `yaml:"isResource"`
}

func (i IsResourceCondition) GetType() string { return "isResource" }

// IsAttachmentCondition represents an attachment type check
type IsAttachmentCondition struct {
	Type any `yaml:"isAttachment"`
}

func (i IsAttachmentCondition) GetType() string { return "isAttachment" }

// IsHashableStructCondition represents a hashable struct type check
type IsHashableStructCondition struct {
	Type any `yaml:"isHashableStruct"`
}

func (i IsHashableStructCondition) GetType() string { return "isHashableStruct" }

// IsStorableCondition represents a storable type check
type IsStorableCondition struct {
	Type any `yaml:"isStorable"`
}

func (i IsStorableCondition) GetType() string { return "isStorable" }

// EqualsCondition represents an equality check
type EqualsCondition struct {
	Source any `yaml:"source"`
	Target any `yaml:"target"`
}

func (e EqualsCondition) GetType() string { return "equals" }

// SubtypeCondition represents a subtype check
type SubtypeCondition struct {
	Sub   any `yaml:"sub"`
	Super any `yaml:"super"`
}

func (s SubtypeCondition) GetType() string { return "subtype" }

// AndCondition represents a logical AND condition
type AndCondition struct {
	Conditions []RuleCondition `yaml:"and"`
}

func (a AndCondition) GetType() string { return "and" }

// OrCondition represents a logical OR condition
type OrCondition struct {
	Conditions []RuleCondition `yaml:"or"`
}

func (o OrCondition) GetType() string { return "or" }

// NotCondition represents a logical NOT condition
type NotCondition struct {
	Condition RuleCondition `yaml:"not"`
}

func (n NotCondition) GetType() string { return "not" }

// PermitsCondition represents a permits check
type PermitsCondition struct {
	Types []any `yaml:"permits"`
}

func (p PermitsCondition) GetType() string { return "permits" }

// PurityCondition represents a purity check
type PurityCondition struct {
	EqualsOrView bool `yaml:"equals_or_view"`
}

func (p PurityCondition) GetType() string { return "purity" }

// TypeParamsEqualCondition represents a type parameters equality check
type TypeParamsEqualCondition struct{}

func (t TypeParamsEqualCondition) GetType() string { return "typeParamsEqual" }

// ParamsContravariantCondition represents a params contravariant check
type ParamsContravariantCondition struct{}

func (p ParamsContravariantCondition) GetType() string { return "paramsContravariant" }

// ReturnCovariantCondition represents a return covariant check
type ReturnCovariantCondition struct{}

func (r ReturnCovariantCondition) GetType() string { return "returnCovariant" }

// ConstructorEqualCondition represents a constructor equality check
type ConstructorEqualCondition struct{}

func (c ConstructorEqualCondition) GetType() string { return "constructorEqual" }

// ContainsCondition represents a contains check
type ContainsCondition struct {
	Types []any `yaml:"contains"`
}

func (c ContainsCondition) GetType() string { return "contains" }

type KeyValues = map[string]any

// TypeInfo represents parsed type information
type TypeInfo struct {
	TypeName       string
	IsGeneric      bool
	GenericArgs    []TypeInfo
	IsOptional     bool
	IsReference    bool
	IsFunction     bool
	IsDictionary   bool
	IsIntersection bool
}

// readYAMLRules reads and parses the YAML rules file
func readYAMLRules(path string) ([]Rule, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", path, err)
	}

	var config RulesConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	return config.Rules, nil
}

// parseType parses a type from YAML
func parseType(typeData any) (*TypeInfo, error) {
	switch v := typeData.(type) {
	case string:
		return &TypeInfo{TypeName: v}, nil
	case KeyValues:
		ti := &TypeInfo{}

		// Handle Optional type
		if optional, ok := v["Optional"].(KeyValues); ok {
			ti.IsOptional = true
			inner, err := parseType(optional["inner"])
			if err != nil {
				return nil, err
			}
			ti.GenericArgs = []TypeInfo{*inner}
			return ti, nil
		}

		// Handle Reference type
		if ref, ok := v["Reference"].(KeyValues); ok {
			ti.IsReference = true
			ti.TypeName = "Reference"
			// Parse auth and type parameters
			if auth, ok := ref["auth"]; ok {
				authType, err := parseType(auth)
				if err != nil {
					return nil, err
				}
				ti.GenericArgs = append(ti.GenericArgs, *authType)
			}
			if typeParam, ok := ref["type"]; ok {
				typeType, err := parseType(typeParam)
				if err != nil {
					return nil, err
				}
				ti.GenericArgs = append(ti.GenericArgs, *typeType)
			}
			return ti, nil
		}

		// Handle Function type
		if fn, ok := v["Function"].(KeyValues); ok {
			ti.IsFunction = true
			ti.TypeName = "Function"
			// Parse params and return
			if params, ok := fn["params"]; ok {
				paramsType, err := parseType(params)
				if err != nil {
					return nil, err
				}
				ti.GenericArgs = append(ti.GenericArgs, *paramsType)
			}
			if returnType, ok := fn["return"]; ok {
				returnTypeInfo, err := parseType(returnType)
				if err != nil {
					return nil, err
				}
				ti.GenericArgs = append(ti.GenericArgs, *returnTypeInfo)
			}
			return ti, nil
		}

		// Handle Dictionary type
		if dict, ok := v["Dictionary"].(KeyValues); ok {
			ti.IsDictionary = true
			ti.TypeName = "Dictionary"
			// Parse key and value
			if key, ok := dict["key"]; ok {
				keyType, err := parseType(key)
				if err != nil {
					return nil, err
				}
				ti.GenericArgs = append(ti.GenericArgs, *keyType)
			}
			if value, ok := dict["value"]; ok {
				valueType, err := parseType(value)
				if err != nil {
					return nil, err
				}
				ti.GenericArgs = append(ti.GenericArgs, *valueType)
			}
			return ti, nil
		}

		// Handle Intersection type
		if intersection, ok := v["Intersection"].(KeyValues); ok {
			ti.IsIntersection = true
			ti.TypeName = "Intersection"
			// Parse set
			if set, ok := intersection["set"]; ok {
				setType, err := parseType(set)
				if err != nil {
					return nil, err
				}
				ti.GenericArgs = []TypeInfo{*setType}
			}
			return ti, nil
		}

		return nil, fmt.Errorf("unsupported type pattern: %v", v)
	default:
		return nil, fmt.Errorf("unsupported type: %T", typeData)
	}
}

// parseRuleCondition parses a rule condition from YAML
func parseRuleCondition(rule any) (RuleCondition, error) {
	switch v := rule.(type) {
	case string:
		if v == "always" {
			return AlwaysCondition{}, nil
		}
		return nil, fmt.Errorf("unsupported string rule: %s", v)
	case KeyValues:
		// Check for each condition type
		if _, ok := v["always"]; ok {
			return AlwaysCondition{}, nil
		}

		if isResource, ok := v["isResource"]; ok {
			return IsResourceCondition{Type: isResource}, nil
		}

		if isAttachment, ok := v["isAttachment"]; ok {
			return IsAttachmentCondition{Type: isAttachment}, nil
		}

		if isHashableStruct, ok := v["isHashableStruct"]; ok {
			return IsHashableStructCondition{Type: isHashableStruct}, nil
		}

		if isStorable, ok := v["isStorable"]; ok {
			return IsStorableCondition{Type: isStorable}, nil
		}

		if equals, ok := v["equals"].(KeyValues); ok {
			return EqualsCondition{
				Source: equals["source"],
				Target: equals["target"],
			}, nil
		}

		if subtype, ok := v["subtype"].(KeyValues); ok {
			return SubtypeCondition{
				Sub:   subtype["sub"],
				Super: subtype["super"],
			}, nil
		}

		if and, ok := v["and"].([]any); ok {
			var conditions []RuleCondition
			for _, cond := range and {
				parsed, err := parseRuleCondition(cond)
				if err != nil {
					return nil, err
				}
				conditions = append(conditions, parsed)
			}
			return AndCondition{Conditions: conditions}, nil
		}

		if or, ok := v["or"].([]any); ok {
			var conditions []RuleCondition
			for _, cond := range or {
				parsed, err := parseRuleCondition(cond)
				if err != nil {
					return nil, err
				}
				conditions = append(conditions, parsed)
			}
			return OrCondition{Conditions: conditions}, nil
		}

		if not, ok := v["not"]; ok {
			parsed, err := parseRuleCondition(not)
			if err != nil {
				return nil, err
			}
			return NotCondition{Condition: parsed}, nil
		}

		if permits, ok := v["permits"].([]any); ok {
			return PermitsCondition{Types: permits}, nil
		}

		if purity, ok := v["purity"].(KeyValues); ok {
			return PurityCondition{
				EqualsOrView: purity["equals_or_view"].(bool),
			}, nil
		}

		if typeParamsEqual, ok := v["typeParamsEqual"].(bool); ok && typeParamsEqual {
			return TypeParamsEqualCondition{}, nil
		}

		if paramsContravariant, ok := v["paramsContravariant"].(bool); ok && paramsContravariant {
			return ParamsContravariantCondition{}, nil
		}

		if returnCovariant, ok := v["returnCovariant"].(bool); ok && returnCovariant {
			return ReturnCovariantCondition{}, nil
		}

		if constructorEqual, ok := v["constructorEqual"].(bool); ok && constructorEqual {
			return ConstructorEqualCondition{}, nil
		}

		if contains, ok := v["contains"].([]any); ok {
			return ContainsCondition{Types: contains}, nil
		}

		return nil, fmt.Errorf("unsupported rule condition: %v", v)
	default:
		return nil, fmt.Errorf("unsupported rule type: %T", rule)
	}
}
