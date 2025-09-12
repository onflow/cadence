package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func main() {
	var (
		yamlPath string
		outPath  string
		pkgName  string
		toStdout bool
	)
	flag.StringVar(&yamlPath, "rules", "rules.yaml", "path to YAML rules")
	flag.StringVar(&outPath, "out", "-", "output file path or '-' for stdout")
	flag.StringVar(&pkgName, "pkg", "sema", "target Go package name")
	flag.BoolVar(&toStdout, "stdout", false, "write to stdout")
	flag.Parse()

	// Read and parse YAML rules
	rules, err := readYAMLRules(yamlPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading YAML rules: %v\n", err)
		os.Exit(1)
	}

	// Generate code using the comprehensive generator
	code, err := generateComprehensiveCode(rules, pkgName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error generating code: %v\n", err)
		os.Exit(1)
	}

	// Write output
	if err := writeOutput(outPath, code, toStdout); err != nil {
		fmt.Fprintf(os.Stderr, "error writing output: %v\n", err)
		os.Exit(1)
	}
}

// writeOutput writes the generated code to the specified output
func writeOutput(dst string, content []byte, stdout bool) error {
	if stdout || dst == "-" {
		_, err := io.Copy(os.Stdout, bytes.NewReader(content))
		return err
	}

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	return os.WriteFile(dst, content, 0o644)
}
