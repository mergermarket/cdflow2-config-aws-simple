package main

import (
	"os"

	common "github.com/mergermarket/cdflow2-config-common"
	"github.com/mergermarket/cdflow2-config-simple-aws/handler"
)

func main() {
	if len(os.Args) == 2 && os.Args[1] == "forward" {
		common.Forward(os.Stdin, os.Stdout, "")
	} else {
		common.Listen(handler.New(&handler.Opts{}), "", nil)
	}
}
