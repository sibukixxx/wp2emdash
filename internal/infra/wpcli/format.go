package wpcli

import "fmt"

// sprintfFloat is split out so fs.go can stay free of fmt-specific concerns
// and the helper is trivial to replace if we ever need a faster formatter.
func sprintfFloat(v float64, prec int) string {
	return fmt.Sprintf("%.*f", prec, v)
}
