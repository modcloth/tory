package main

import (
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"

	"github.com/codegangsta/cli"
)

func main() {
	buildApp().Run(os.Args)
}

func buildApp() *cli.App {
	app := cli.NewApp()
	app.Name = "tory-ansible-inventory"
	app.Usage = "get ansible inventory from a tory server"
	app.Version = "0.1.0"
	app.Flags = []cli.Flag{
		cli.StringFlag{Name: "host"},
		cli.BoolFlag{Name: "list"},
		cli.StringFlag{
			Name:   "s,tory-server",
			EnvVar: "TORY_SERVER",
			Value:  "http://localhost:9462/ansible/hosts",
			Usage:  "tory inventory server full URI",
		},
	}
	app.Action = getInventory

	return app
}

func getInventory(c *cli.Context) {
	urlStr := c.String("tory-server")
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		log.Fatal(err.Error())
	}

	v := parsedURL.Query()
	if c.String("host") != "" {
		parsedURL.Path = path.Join(parsedURL.Path, c.String("host"))
		v.Set("vars-only", "1")
	} else if !c.Bool("list") {
		v.Set("exclude-vars", "1")
	}

	parsedURL.RawQuery = v.Encode()

	resp, err := http.Get(parsedURL.String())
	if err != nil {
		log.Fatal(err.Error())
	}

	io.Copy(os.Stdout, resp.Body)
}
