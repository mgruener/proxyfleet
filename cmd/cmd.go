package cmd

import (
	"fmt"
	"os"

	"github.com/mitchellh/cli"
)

func Cmd() int {
	c := cli.NewCLI("app", "1.0.0")
	c.Args = os.Args[1:]
	c.Commands = map[string]cli.CommandFactory{
		"start": StartCommandFactory,
		"stop":  StopCommandFactory,
		"list":  ListCommandFactory,
	}

	exitStatus, err := c.Run()
	if err != nil {
		fmt.Println(err)
	}

	return exitStatus
}
