package cmd

import (
	"fmt"
	"runtime"

	"gopkg.in/urfave/cli.v1"
)

var (
	// Version contains the current version.
	Version = "dev"
	// BuildDate contains a string with the build date.
	BuildDate = "unknown"
)

var versionCmd = cli.Command{
	Name:        "version",
	Usage:       "Print version",
	Description: `Display version and build information about JURA.`,
	ArgsUsage:   " ",
	Action: func(c *cli.Context) error {
		if c.Args().Present() {
			return UsageError("no additional arguments accepted")
		}
		fmt.Printf("last-resort-gateway %s\n", Version)
		fmt.Printf("  Build date: %s\n", BuildDate)
		fmt.Printf("  Built with: %s\n", runtime.Version())
		return nil
	},
}
