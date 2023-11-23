package cmd

import (
	"fmt"
	"os"
	"strconv"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/mgruener/proxyfleet/pkg/proxyfleet/hetzner"
	"github.com/mitchellh/cli"
)

type ListCommand struct{}

func ListCommandFactory() (cli.Command, error) { return ListCommand{}, nil }

func (cmd ListCommand) Help() string {
	return "list [count]  # list <count> proxies; not specifying a count returns all proxies"
}
func (cmd ListCommand) Synopsis() string { return "list [count]" }
func (cmd ListCommand) Run(args []string) int {
	token := os.Getenv("HETZNER_TOKEN")
	if token == "" {
		fmt.Fprintln(os.Stderr, []any{"must set environment variable HETZNER_TOKEN"}...)
		return 1
	}

	count := -1
	if len(args) > 0 {
		var err error
		count, err = strconv.Atoi(args[0])
		if err != nil {
			fmt.Fprintln(os.Stderr, []any{err}...)
			return 1
		}
	}

	pmgr := hetzner.New(hcloud.WithToken(token))
	ips, err := pmgr.GetProxies(count)
	if err != nil {
		fmt.Fprintln(os.Stderr, []any{err}...)
		return 1
	}

	for _, ip := range ips {
		fmt.Printf("http://%s:8080\n", ip)
	}

	return 0
}
