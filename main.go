// Last-Resort Gateway
package main

import (
	"fmt"
	"os"

	"github.com/pkg/errors"

	"lrg/cmd"
)

func main() {
	if err := cmd.RootCmd.Run(os.Args); err != nil {
		switch err := errors.Cause(err).(type) {
		case cmd.UsageError:
			os.Stderr.WriteString(fmt.Sprintf("Usage error: %s\n", err))
			os.Stderr.WriteString("Use --help for usage\n")
		default:
			os.Stderr.WriteString(fmt.Sprintf("Runtime error: %v\n", err))
		}
		os.Exit(1)
	}
}
