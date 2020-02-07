package main

import (
	"os"

	"github.com/mergermarket/cdflow2-config-simple-aws/command"
	"github.com/mergermarket/cdflow2-config-simple-aws/handler"
)

func main() {
	command.Run(handler.New(), os.Stdin, os.Stdout, os.Stderr)
}
