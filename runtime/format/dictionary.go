package format

import (
	"strings"
)

func Dictionary(pairs []struct{Key string; Value string}) string {
	var builder strings.Builder
	builder.WriteRune('{')
	for i, p := range pairs {
		if i > 0 {
			builder.WriteString(", ")
		}
		builder.WriteString(p.Key)
		builder.WriteString(": ")
		builder.WriteString(p.Value)
	}
	builder.WriteRune('}')
	return builder.String()
}
