package cli

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/aquaproj/aqua/pkg/controller"
	"github.com/urfave/cli/v2"
)

var errCommandIsRequired = errors.New("command is required")

func parseExecArgs(args []string) (string, []string, error) {
	if len(args) == 0 {
		return "", nil, errCommandIsRequired
	}
	return filepath.Base(args[0]), args[1:], nil
}

func (runner *Runner) newExecCommand() *cli.Command {
	return &cli.Command{
		Name:  "exec",
		Usage: "Execute tool",
		Description: `Basically you don't have to use this command, because this is used by aqua internally. aqua-proxy invokes this command.
When you execute the command installed by aqua, "aqua exec" is executed internally.

e.g.
$ aqua exec -- gh version
gh version 2.4.0 (2021-12-21)
https://github.com/cli/cli/releases/tag/v2.4.0`,
		Action:    runner.execAction,
		ArgsUsage: `<executed command> [<arg> ...]`,
	}
}

func (runner *Runner) execAction(c *cli.Context) error {
	param := &controller.Param{}
	if err := runner.setCLIArg(c, param); err != nil {
		return fmt.Errorf("parse the command line arguments: %w", err)
	}

	ctrl, err := controller.New(c.Context, param)
	if err != nil {
		return fmt.Errorf("initialize a controller: %w", err)
	}
	exeName, args, err := parseExecArgs(c.Args().Slice())
	if err != nil {
		return err
	}

	return ctrl.Exec(c.Context, param, exeName, args) //nolint:wrapcheck
}
