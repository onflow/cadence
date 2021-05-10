package main

import (
	"fmt"
	"os"

	"io/ioutil"

	"github.com/onflow/cadence/tools/docgen/gen"
)

func main() {
	input := os.Args[1]
	outputDir := os.Args[2]

	content, err := ioutil.ReadFile(input)
	if err != nil {
		panic(err)
	}

	code := string(content)

	docGen := gen.NewDocGenerator()
	err = docGen.Generate(code, outputDir)

	if err != nil {
		panic(err)
	}

	fmt.Println(fmt.Sprintf("Docs generate at: %s", outputDir))
}
