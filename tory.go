package main

import (
	"fmt"
	"os"

	"github.com/codegangsta/cli"
	"github.com/modcloth-labs/tory/tory"
)

func main() {
	app := cli.NewApp()
	app.Name = "tory"
	app.Version = fmt.Sprintf("%s revision=%s", tory.VersionString, tory.RevisionString)
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "vv, verbose",
			Usage: "be noisy",
		},
		cli.StringFlag{
			Name:  "a, addr",
			Usage: "server address",
			Value: tory.DefaultServerAddr,
		},
	}
	app.Action = func(ctx *cli.Context) {
		tory.ServerMain(ctx.String("addr"), ctx.Bool("verbose"))
	}

	app.Run(os.Args)
}
