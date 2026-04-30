package format

import "fmt"

// CurrentFormatVersion identifies the formatting algorithm version.
// Bump when rewrite pass order changes or formatting rules change.
const CurrentFormatVersion = "1"

// Options controls formatting behavior. All fields have sensible defaults
// via Default().
type Options struct {
	LineWidth       int
	Indent          string
	UseTabs         bool
	SortImports     bool
	StripSemicolons bool
	KeepBlankLines  int
	FormatVersion   string
	SkipVerify      bool
}

// Validate checks that the Options are valid.
func (o Options) Validate() error {
	if o.FormatVersion != CurrentFormatVersion {
		return fmt.Errorf("unsupported format version %q (current: %s)", o.FormatVersion, CurrentFormatVersion)
	}
	if o.KeepBlankLines < 0 {
		return fmt.Errorf("KeepBlankLines must be >= 0, got %d", o.KeepBlankLines)
	}
	return nil
}

// Default returns the canonical default formatting options.
func Default() Options {
	return Options{
		LineWidth:       100,
		Indent:          "    ",
		UseTabs:         false,
		SortImports:     true,
		StripSemicolons: true,
		KeepBlankLines:  1,
		FormatVersion:   CurrentFormatVersion,
	}
}
