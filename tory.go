package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/codegangsta/cli"
	"github.com/modcloth/tory/tory"
)

func main() {
	whoami := os.Getenv("USER")
	if os.Getenv("DATABASE_URL") == "" && whoami == "" {
		cmd := exec.Command("whoami")
		var out bytes.Buffer
		cmd.Stdout = &out
		err := cmd.Run()
		if err == nil {
			whoami = strings.TrimSpace(out.String())
		}
	}

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
					EnvVar: "TORY_AUTH_TOKEN",
				},
				cli.StringFlag{
					Name:   "d, database-url",
					Value:  fmt.Sprintf("postgres://%s@localhost/tory?sslmode=disable", whoami),
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
				cli.BoolFlag{
					Name:   "E, new-relic-agent-enabled",
					Usage:  "Enable the NewRelic agent",
					EnvVar: "NEW_RELIC_AGENT_ENABLED",
				},
				cli.StringFlag{
					Name:   "l, new-relic-license-key",
					Usage:  "New Relic License Key",
					EnvVar: "NEW_RELIC_LICENSE_KEY",
				},
				cli.StringFlag{
					Name:   "n, new-relic-app-name",
					Value:  "Tory",
					Usage:  "New Relic App Name",
					EnvVar: "NEW_RELIC_APP_NAME",
				},
				cli.BoolFlag{
					Name:   "V, new-relic-verbose",
					Usage:  "Set New Relic agent to report verbosely",
					EnvVar: "NEW_RELIC_VERBOSE",
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
					NewRelicOptions: tory.NewRelicOptions{
						Enabled:    c.Bool("new-relic-agent-enabled"),
						LicenseKey: c.String("new-relic-license-key"),
						AppName:    c.String("new-relic-app-name"),
						Verbose:    c.Bool("new-relic-verbose"),
					},
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
					Value:  fmt.Sprintf("postgres://%s@localhost/tory?sslmode=disable", whoami),
					Usage:  "database connection uri",
					EnvVar: "DATABASE_URL",
				},
			},
		},
	}

	app.Run(os.Args)
}
