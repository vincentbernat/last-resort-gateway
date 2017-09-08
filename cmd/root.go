// Package cmd handles command-line for Last-Resort Gateway
package cmd

import (
	"gopkg.in/urfave/cli.v1"
)

// RootCmd is the root for all commands.
var RootCmd = cli.NewApp()

// UsageError is the kind of error that should be returned if the
// command failed due to improper usage by user.
type UsageError string

func (e UsageError) Error() string {
	return string(e)
}

// wrapUsageError will transform any command to return a "UsageError"
// in case of usage error.
func wrapUsageError(c cli.Command) cli.Command {
	c.OnUsageError = func(c *cli.Context, err error, isSubcommand bool) error {
		return UsageError(err.Error())
	}
	return c
}

func init() {
	RootCmd.Name = "last-resort-gateway"
	RootCmd.Version = Version
	RootCmd.Usage = "Last-Resort Gateway"
	RootCmd.Description = `Maintain last resort gateways to survive transient
routing daemon failures. Appropriate routes are copied
and kept up-to-date unless they completely disappear.`
	RootCmd.OnUsageError = func(c *cli.Context, err error, isSubcommand bool) error {
		return UsageError(err.Error())
	}
	RootCmd.Commands = []cli.Command{
		wrapUsageError(daemonCmd),
		wrapUsageError(versionCmd),
	}
}
