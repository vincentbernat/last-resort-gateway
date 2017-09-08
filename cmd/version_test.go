package cmd

import (
	"testing"

	"gopkg.in/urfave/cli.v1"
)

func TestVersion(t *testing.T) {
	// We don't really test version but we check if we get an
	// appropriate exit code/error when there is an error.
	var lastCode int
	cli.OsExiter = func(code int) {
		lastCode = code
	}
	cases := []struct {
		args []string
		err  bool
	}{
		{
			args: []string{"version"},
			err:  false,
		}, {
			args: []string{"version", "--help"},
			err:  false,
		}, {
			args: []string{"version", "hop"},
			err:  true,
		}, {
			args: []string{"version", "--hop"},
			err:  true,
		},
	}
	for _, tc := range cases {
		lastCode = -1
		err := RootCmd.Run(tc.args)
		if err != nil && !tc.err {
			t.Errorf("version(%s) error:\n%+v", tc.args, err)
		} else if lastCode != -1 && !tc.err {
			t.Errorf("version(%s) exited with %d", tc.args, lastCode)
		} else if err == nil && tc.err {
			t.Errorf("version(%s) did not trigger an error", tc.args)
		}
	}
}
