package main

import (
	"os"

	"github.com/mgruener/proxyfleet/cmd"
)

func main() {
	os.Exit(cmd.Cmd())
}
