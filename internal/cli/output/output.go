// Package output provides shared CLI output helpers used by all subcommands.
package output

import (
	"encoding/json"
	"fmt"
	"io"
)

// JSON writes v to w as indented JSON followed by a newline.
func JSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// Println writes args to w followed by a newline, returning any write error.
func Println(w io.Writer, args ...any) error {
	_, err := fmt.Fprintln(w, args...)
	return err
}

// Printf writes a formatted string to w, returning any write error.
func Printf(w io.Writer, format string, args ...any) error {
	_, err := fmt.Fprintf(w, format, args...)
	return err
}
