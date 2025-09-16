package subtype_gen

import (
	"fmt"
	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
	"github.com/dave/dst/decorator/resolver/guess"
	"os"
	"testing"
)

func TestGen(t *testing.T) {
	// Read and parse YAML rules
	rules, err := ParseRules()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading YAML rules: %v\n", err)
		os.Exit(1)
	}

	const pkgPath = "github.com/onflow/cadence/sema"

	// Generate code using the comprehensive generator
	gen := NewSubTypeCheckGenerator(pkgPath)
	decls := gen.GenerateCheckSubTypeWithoutEqualityFunction(rules)

	resolver := guess.New()
	restorer := decorator.NewRestorerWithImports(pkgPath, resolver)

	packageName, err := resolver.ResolvePackage(pkgPath)
	if err != nil {
		panic(err)
	}

	err = restorer.Fprint(
		os.Stdout,
		&dst.File{
			Name:  dst.NewIdent(packageName),
			Decls: decls,
		},
	)
}
