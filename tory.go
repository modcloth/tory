package main

import (
	"fmt"
	"os"

	"github.com/codegangsta/cli"
	"github.com/modcloth/tory/tory"
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
			Flags: []cli.Flag{
				cli.BoolFlag{
					Name:   "vv, verbose",
					Usage:  "be noisy",
					EnvVar: "VERBOSE",
				},
				cli.BoolFlag{
					Name:   "q, quiet",
					Usage:  "be quiet",
					EnvVar: "QUIET",
				},
				cli.StringFlag{
					Name:   "a, server-addr",
					Value:  ":9462",
					Usage:  "server address (also accepts $PORT)",
					EnvVar: "TORY_ADDR",
				},
				cli.StringFlag{
					Name:   "A, auth-token",
					Value:  "swordfish",
					Usage:  "mutative action auth token",
					EnvVar: "TORY_AUTH",
				},
				cli.StringFlag{
					Name:   "d, database-url",
					Value:  "postgres://localhost/tory?sslmode=disable",
					Usage:  "database connection uri",
					EnvVar: "DATABASE_URL",
				},
				cli.StringFlag{
					Name:   "s, static-dir",
					Value:  "public",
					Usage:  "static file directory",
					EnvVar: "TORY_STATIC_DIR",
				},
				cli.StringFlag{
					Name:   "p, prefix",
					Value:  `/ansible/hosts`,
					Usage:  "public api prefix",
					EnvVar: "TORY_PREFIX",
				},
			},
			Action: func(c *cli.Context) {
				tory.ServerMain(&tory.ServerOptions{
					Addr:        c.String("server-addr"),
					AuthToken:   c.String("auth-token"),
					DatabaseURL: c.String("database-url"),
					Prefix:      c.String("prefix"),
					Quiet:       c.Bool("quiet"),
					StaticDir:   c.String("static-dir"),
					Verbose:     c.Bool("verbose"),
				})
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
					Name:   "d, database-url",
					Value:  "postgres://localhost/tory?sslmode=disable",
					Usage:  "database connection uri",
					EnvVar: "DATABASE_URL",
				},
			},
		},
	}

	app.Run(os.Args)
}
