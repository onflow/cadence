package execute

import (
	"github.com/logrusorgru/aurora"
	"github.com/onflow/cadence/runtime/interpreter"
)

func colorizeResult(value interpreter.Value) string {
	str := value.String()
	return aurora.Colorize(str, aurora.YellowFg|aurora.BrightFg).String()
}

func formatValue(value interpreter.Value) string {
	if _, isVoid := value.(*interpreter.VoidValue); isVoid || value == nil {
		return ""
	}

	return colorizeResult(value)
}

func colorizeError(message string) string {
	return aurora.Colorize(message, aurora.RedFg|aurora.BrightFg|aurora.BoldFm).String()
}
