// +build !release

package helpers

import (
	"strings"

	"github.com/kylelemons/godebug/pretty"
)

// TrimSpaces will remove any extra space. Preferring new lines to
// spaces.
func TrimSpaces(input string) (output string) {
	next := input
	for output != next {
		output = next
		next = strings.Replace(next, "\t", " ", -1)
		next = strings.Replace(next, "  ", " ", -1)
		next = strings.Replace(next, "\n ", "\n", -1)
		next = strings.Replace(next, " \n", "\n", -1)
	}
	output = strings.TrimSpace(output)
	return
}

var prettyC = pretty.Config{
	Diffable:          true,
	PrintStringers:    true,
	SkipZeroFields:    true,
	IncludeUnexported: false,
}

// Diff return a diff of two objects. If no diff, an empty string is
// returned.
func Diff(a, b interface{}) string {
	return prettyC.Compare(a, b)
}
