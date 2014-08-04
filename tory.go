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
	app.Usage = "ansible inventory server"
	app.Version = fmt.Sprintf("%s revision=%s", tory.VersionString, tory.RevisionString)
	app.Commands = []cli.Command{
		cli.Command{
			Name:      "serve",
			ShortName: "s",
			Usage:     "run the http server",
			Action: func(c *cli.Context) {
				tory.ServerMain(c.String("server-addr"),
					c.String("database-url"), c.String("static-dir"),
					c.String("prefix"), (c.Bool("verbose") || os.Getenv("VERBOSE") != ""))
			},
			Flags: []cli.Flag{
				cli.BoolFlag{
					Name:  "vv, verbose",
					Usage: "be noisy",
				},
				cli.StringFlag{
					Name:  "a, server-addr",
					Usage: "server address",
					Value: tory.DefaultServerAddr,
				},
				cli.StringFlag{
					Name:  "d, database-url",
					Usage: "database connection uri",
					Value: tory.DefaultDatabaseURL,
				},
				cli.StringFlag{
					Name:  "s, static-dir",
					Usage: "static file directory",
					Value: tory.DefaultStaticDir,
				},
				cli.StringFlag{
					Name:  "p, prefix",
					Usage: "public api prefix",
					Value: tory.DefaultPrefix,
				},
			},
		},
		cli.Command{
			Name:      "migrate",
			ShortName: "m",
			Usage:     "run database migrations",
			Action: func(c *cli.Context) {
				tory.MigrateMain(c.String("database-url"))
			},
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "d, database-url",
					Usage: "database connection uri",
					Value: tory.DefaultDatabaseURL,
				},
			},
		},
	}

	app.Run(os.Args)
}
