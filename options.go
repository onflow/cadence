package format

// QuoteStyle controls string literal quoting. Only double quotes are valid
// in Cadence, so this is a placeholder for potential future expansion.
type QuoteStyle int

const (
	DoubleQuote QuoteStyle = iota
)

// Options controls formatting behavior. All fields have sensible defaults
// via Default().
type Options struct {
	LineWidth       int
	Indent          string
	UseTabs         bool
	SortImports     bool
	QuoteStyle      QuoteStyle
	StripSemicolons bool
	KeepBlankLines  int
	FormatVersion   string
}

// Default returns the canonical default formatting options.
func Default() Options {
	return Options{
		LineWidth:       100,
		Indent:          "    ",
		UseTabs:         false,
		SortImports:     true,
		QuoteStyle:      DoubleQuote,
		StripSemicolons: true,
		KeepBlankLines:  1,
		FormatVersion:   "1",
	}
}
