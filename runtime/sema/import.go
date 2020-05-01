package sema

import (
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
)

// Import

type Import interface {
	AllValueElements() map[string]ImportElement
	IsImportableValue(name string) bool
	AllTypeElements() map[string]ImportElement
	IsImportableType(name string) bool
}

// ImportElement

type ImportElement struct {
	DeclarationKind common.DeclarationKind
	Access          ast.Access
	Type            Type
	ArgumentLabels  []string
}

// CheckerImport

type CheckerImport struct {
	*Checker
}

func variablesToImportElements(variables map[string]*Variable) map[string]ImportElement {
	elements := make(map[string]ImportElement, len(variables))
	for name, variable := range variables {
		elements[name] = ImportElement{
			DeclarationKind: variable.DeclarationKind,
			Access:          variable.Access,
			Type:            variable.Type,
			ArgumentLabels:  variable.ArgumentLabels,
		}
	}
	return elements
}

func (i CheckerImport) AllValueElements() map[string]ImportElement {
	return variablesToImportElements(i.Checker.GlobalValues)
}

func (i CheckerImport) IsImportableValue(name string) bool {
	_, isBaseValue := BaseValues[name]
	if isBaseValue {
		return false
	}

	_, isPredeclaredValue := i.PredeclaredValues[name]
	return !isPredeclaredValue
}

func (i CheckerImport) AllTypeElements() map[string]ImportElement {
	return variablesToImportElements(i.Checker.GlobalTypes)
}

func (i CheckerImport) IsImportableType(name string) bool {
	_, isPredeclaredType := i.PredeclaredTypes[name]
	return !isPredeclaredType
}

// VirtualImport

type VirtualImport struct {
	ValueElements map[string]ImportElement
	TypeElements  map[string]ImportElement
}

func (i VirtualImport) AllValueElements() map[string]ImportElement {
	return i.ValueElements
}

func (i VirtualImport) IsImportableValue(_ string) bool {
	return true
}

func (i VirtualImport) AllTypeElements() map[string]ImportElement {
	return i.TypeElements
}

func (i VirtualImport) IsImportableType(_ string) bool {
	return true
}
