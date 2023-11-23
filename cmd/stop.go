package cmd

import (
	"fmt"
	"os"
	"strconv"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/mgruener/proxyfleet/pkg/proxyfleet/hetzner"
	"github.com/mitchellh/cli"
)

type StopCommand struct{}

func StopCommandFactory() (cli.Command, error) { return StopCommand{}, nil }

func (cmd StopCommand) Help() string {
	return "stop <count>  # stop <count> proxies; -1 stops all proxies"
}
func (cmd StopCommand) Synopsis() string { return "stop <count>" }
func (cmd StopCommand) Run(args []string) int {
	token := os.Getenv("HETZNER_TOKEN")
	if token == "" {
		fmt.Fprintln(os.Stderr, []any{"must set environment variable HETZNER_TOKEN"}...)
		return 1
	}

	if len(args) < 1 {
		return cli.RunResultHelp
	}

	count, err := strconv.Atoi(args[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, []any{err}...)
		return 1
	}

	pmgr := hetzner.New(hcloud.WithToken(token))
	ips, err := pmgr.DespawnProxies(count)
	if err != nil {
		fmt.Fprintln(os.Stderr, []any{err}...)
		return 1
	}

	for _, ip := range ips {
		fmt.Printf("http://%s:8080\n", ip)
	}

	return 0
}
