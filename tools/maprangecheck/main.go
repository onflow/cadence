package main

import (
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	singlechecker.Main(Analyzer)
}

type analyzerPlugin struct{}

func (*analyzerPlugin) GetAnalyzers() []*analysis.Analyzer {
	return []*analysis.Analyzer{
		Analyzer,
	}
}

// This must be defined and named 'AnalyzerPlugin' for golangci-lint,
// see https://golangci-lint.run/contributing/new-linters/#how-to-write-a-custom-linter
var AnalyzerPlugin analyzerPlugin
