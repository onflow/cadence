package test

import (
	"github.com/onflow/cadence/tools/doc-generator"
	"io/ioutil"
	"log"
	"os"
	"testing"
)

func TestDocGen(t *testing.T) {
	content, err := ioutil.ReadFile("sample3.cdc")
	if err != nil {
		log.Fatal(err)
	}

	code := string(content)

	err = os.RemoveAll("generated")
	if err != nil {
		log.Fatal(err)
	}
	os.MkdirAll("generated", os.ModePerm)

	docGen := doc_generator.NewDocGenerator()
	docGen.Generate(code, "generated")
}
