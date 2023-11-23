package cmd

import (
	"fmt"
	"os"
	"strconv"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/mgruener/proxyfleet/pkg/proxyfleet/hetzner"
	"github.com/mitchellh/cli"
)

type StartCommand struct{}

func StartCommandFactory() (cli.Command, error) { return StartCommand{}, nil }

func (cmd StartCommand) Help() string     { return "start <count>  # start <count> proxies" }
func (cmd StartCommand) Synopsis() string { return "start <count>" }
func (cmd StartCommand) Run(args []string) int {
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
	if count < 1 {
		return cli.RunResultHelp
	}

	pmgr := hetzner.New(hcloud.WithToken(token))
	ips, err := pmgr.EnsureProxies(uint(count), uint(count))
	if err != nil {
		fmt.Fprintln(os.Stderr, []any{err}...)
		return 1
	}

	for _, ip := range ips {
		fmt.Printf("http://%s:8080\n", ip)
	}

	return 0
}
