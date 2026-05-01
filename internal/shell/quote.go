package shell

import "strings"

// QuotePOSIX returns s as a single shell token for POSIX sh.
func QuotePOSIX(s string) string {
	if s == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(s, "'", `'"'"'`) + "'"
}
