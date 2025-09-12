package main

import (
	"fmt"
	"os"
	"testing"
)

func TestGen(t *testing.T) {
	// Read and parse YAML rules
	rules, err := readYAMLRules("/Users/supunsetunga/work/cadence-experimental/cadence/tools/subtype-gen/rules.yaml")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading YAML rules: %v\n", err)
		os.Exit(1)
	}

	// Generate code using the comprehensive generator
	gen := NewSubTypeCheckGenerator("github.com/onflow/cadence/sema")
	code, err := gen.generateCheckSubTypeWithoutEqualityFunction(rules)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error generating code: %v\n", err)
		os.Exit(1)
	}

	// Write output
	if err := writeOutput("", []byte(code), true); err != nil {
		fmt.Fprintf(os.Stderr, "error writing output: %v\n", err)
		os.Exit(1)
	}
}
