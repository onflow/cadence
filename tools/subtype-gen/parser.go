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

// RulePredicate represents different types of predicates in rules
type RulePredicate interface {
	GetType() string
}

// AlwaysPredicate represents an always-true condition
type AlwaysPredicate struct{}

func (a AlwaysPredicate) GetType() string { return "always" }

// IsResourcePredicate represents a resource type check
type IsResourcePredicate struct {
	Type any `yaml:"isResource"`
}

func (i IsResourcePredicate) GetType() string { return "isResource" }

// IsAttachmentPredicate represents an attachment type check
type IsAttachmentPredicate struct {
	Type any `yaml:"isAttachment"`
}

func (i IsAttachmentPredicate) GetType() string { return "isAttachment" }

// IsHashableStructPredicate represents a hashable struct type check
type IsHashableStructPredicate struct {
	Type any `yaml:"isHashableStruct"`
}

func (i IsHashableStructPredicate) GetType() string { return "isHashableStruct" }

// IsStorablePredicate represents a storable type check
type IsStorablePredicate struct {
	Type any `yaml:"isStorable"`
}

func (i IsStorablePredicate) GetType() string { return "isStorable" }

// EqualsPredicate represents an equality check
type EqualsPredicate struct {
	Source any `yaml:"source"`
	Target any `yaml:"target"`
}

func (e EqualsPredicate) GetType() string { return "equals" }

// SubtypePredicate represents a subtype check
type SubtypePredicate struct {
	Sub   any `yaml:"sub"`
	Super any `yaml:"super"`
}

func (s SubtypePredicate) GetType() string { return "subtype" }

// AndPredicate represents a logical AND predicate
type AndPredicate struct {
	Predicates []RulePredicate `yaml:"and"`
}

func (a AndPredicate) GetType() string { return "and" }

// OrPredicate represents a logical OR predicate
type OrPredicate struct {
	Predicates []RulePredicate `yaml:"or"`
}

func (o OrPredicate) GetType() string { return "or" }

// NotPredicate represents a logical NOT predicate
type NotPredicate struct {
	Predicate RulePredicate `yaml:"not"`
}

func (n NotPredicate) GetType() string { return "not" }

// PermitsPredicate represents a permits check
type PermitsPredicate struct {
	Types []any `yaml:"permits"`
}

func (p PermitsPredicate) GetType() string { return "permits" }

// PurityPredicate represents a purity check
type PurityPredicate struct {
	EqualsOrView bool `yaml:"equals_or_view"`
}

func (p PurityPredicate) GetType() string { return "purity" }

// TypeParamsEqualPredicate represents a type parameters equality check
type TypeParamsEqualPredicate struct{}

func (t TypeParamsEqualPredicate) GetType() string { return "typeParamsEqual" }

// ParamsContravariantPredicate represents a params contravariant check
type ParamsContravariantPredicate struct{}

func (p ParamsContravariantPredicate) GetType() string { return "paramsContravariant" }

// ReturnCovariantPredicate represents a return covariant check
type ReturnCovariantPredicate struct{}

func (r ReturnCovariantPredicate) GetType() string { return "returnCovariant" }

// ConstructorEqualPredicate represents a constructor equality check
type ConstructorEqualPredicate struct{}

func (c ConstructorEqualPredicate) GetType() string { return "constructorEqual" }

// ContainsPredicate represents a contains check
type ContainsPredicate struct {
	Types []any `yaml:"contains"`
}

func (c ContainsPredicate) GetType() string { return "contains" }

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

// ParseRules reads and parses the YAML rules file
func ParseRules() ([]Rule, error) {
	var config RulesConfig
	if err := yaml.Unmarshal([]byte(subtypeCheckingRules), &config); err != nil {
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

// parseRulePredicate parses a rule predicate from YAML
func parseRulePredicate(rule any) (RulePredicate, error) {
	switch v := rule.(type) {
	case string:
		switch v {
		case "always":
			return AlwaysPredicate{}, nil
		case "typeParamsEqual":
			return TypeParamsEqualPredicate{}, nil
		case "paramsContravariant":
			return ParamsContravariantPredicate{}, nil
		case "returnCovariant":
			return ReturnCovariantPredicate{}, nil
		case "constructorEqual":
			return ConstructorEqualPredicate{}, nil
		default:
			return nil, fmt.Errorf("unsupported string rule: %s", v)
		}
	case KeyValues:
		// Check for each predicate type
		if _, ok := v["always"]; ok {
			return AlwaysPredicate{}, nil
		}

		if isResource, ok := v["isResource"]; ok {
			return IsResourcePredicate{Type: isResource}, nil
		}

		if isAttachment, ok := v["isAttachment"]; ok {
			return IsAttachmentPredicate{Type: isAttachment}, nil
		}

		if isHashableStruct, ok := v["isHashableStruct"]; ok {
			return IsHashableStructPredicate{Type: isHashableStruct}, nil
		}

		if isStorable, ok := v["isStorable"]; ok {
			return IsStorablePredicate{Type: isStorable}, nil
		}

		if equals, ok := v["equals"].(KeyValues); ok {
			return EqualsPredicate{
				Source: equals["source"],
				Target: equals["target"],
			}, nil
		}

		if subtype, ok := v["subtype"].(KeyValues); ok {
			return SubtypePredicate{
				Sub:   subtype["sub"],
				Super: subtype["super"],
			}, nil
		}

		if and, ok := v["and"].([]any); ok {
			var predicates []RulePredicate
			for _, cond := range and {
				parsed, err := parseRulePredicate(cond)
				if err != nil {
					return nil, err
				}
				predicates = append(predicates, parsed)
			}
			return AndPredicate{Predicates: predicates}, nil
		}

		if or, ok := v["or"].([]any); ok {
			var predicates []RulePredicate
			for _, cond := range or {
				parsed, err := parseRulePredicate(cond)
				if err != nil {
					return nil, err
				}
				predicates = append(predicates, parsed)
			}
			return OrPredicate{Predicates: predicates}, nil
		}

		if not, ok := v["not"]; ok {
			predicate, err := parseRulePredicate(not)
			if err != nil {
				return nil, err
			}
			return NotPredicate{Predicate: predicate}, nil
		}

		if permits, ok := v["permits"].([]any); ok {
			return PermitsPredicate{Types: permits}, nil
		}

		if purity, ok := v["purity"].(KeyValues); ok {
			return PurityPredicate{
				EqualsOrView: purity["equals_or_view"].(bool),
			}, nil
		}

		if typeParamsEqual, ok := v["typeParamsEqual"].(bool); ok && typeParamsEqual {
			return TypeParamsEqualPredicate{}, nil
		}

		if paramsContravariant, ok := v["paramsContravariant"].(bool); ok && paramsContravariant {
			return ParamsContravariantPredicate{}, nil
		}

		if returnCovariant, ok := v["returnCovariant"].(bool); ok && returnCovariant {
			return ReturnCovariantPredicate{}, nil
		}

		if constructorEqual, ok := v["constructorEqual"].(bool); ok && constructorEqual {
			return ConstructorEqualPredicate{}, nil
		}

		if contains, ok := v["contains"].([]any); ok {
			return ContainsPredicate{Types: contains}, nil
		}

		return nil, fmt.Errorf("unsupported rule predicate: %v", v)
	default:
		return nil, fmt.Errorf("unsupported rule type: %T", rule)
	}
}
