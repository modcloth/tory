package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/codegangsta/cli"
)

var (
	usage = `populate tory inventory from Joyent listmachines json`
)

type joyentHostJSON struct {
	Name string `json:"name"`
	IP   string `json:"primaryIp"`

	Package string `json:"package,omitempty"`
	Image   string `json:"image,omitempty"`
	Type    string `json:"type,omitempty"`

	Disk   int `json:"disk,omitempty"`
	Memory int `json:"memory,omitempty"`

	Tags map[string]interface{} `json:"tags,omitempty"`
	Vars map[string]interface{} `json:"vars,omitempty"`
}

func main() {
	buildApp().Run(os.Args)
}

func buildApp() *cli.App {
	app := cli.NewApp()
	app.Name = "tory-sync-from-joyent"
	app.Usage = usage
	app.Version = "0.1.0"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "s,tory-server",
			Value:  "http://localhost:9462/ansible/hosts",
			Usage:  "tory inventory server",
			EnvVar: "TORY_SERVER",
		},
		cli.StringFlag{
			Name:   "j,sdc-listmachines-json",
			Value:  "-",
			Usage:  "joyent listmachines input json",
			EnvVar: "TORY_SYNC_SDC_LISTMACHINES_JSON",
		},
	}
	app.Action = syncFromJoyent

	return app
}

func syncFromJoyent(c *cli.Context) {
	var (
		fd  io.Reader
		err error
	)

	server := c.String("tory-server")

	sj := c.String("sdc-listmachines-json")
	if sj == "-" {
		fd = os.Stdin
	} else {
		fd, err = os.Open(sj)
		if err != nil {
			log.Fatal(err.Error())
		}
	}

	sjBytes, err := ioutil.ReadAll(fd)
	if err != nil {
		log.Fatal(err.Error())
	}

	sjSlice := []*joyentHostJSON{}
	err = json.Unmarshal(sjBytes, &sjSlice)
	if err != nil {
		log.Fatal(err.Error())
	}

	for i := len(sjSlice) - 1; i >= 0; i-- {
		syncOneMachine(server, sjSlice[i])
	}

	log.Println("Ding!")
}

func syncOneMachine(server string, jhj *joyentHostJSON) {
	if jhj.Vars == nil {
		jhj.Vars = map[string]interface{}{}
	}

	jhj.Vars["disk"] = fmt.Sprintf("%d", jhj.Disk)
	jhj.Vars["memory"] = fmt.Sprintf("%d", jhj.Memory)

	jhjBytes, err := json.Marshal(map[string]*joyentHostJSON{"host": jhj})
	if err != nil {
		log.Fatal(err.Error())
	}

	buf := bytes.NewReader(jhjBytes)
	resp, err := http.Post(server, "application/json", buf)
	if err != nil {
		log.Fatal(err.Error())
	}

	if resp.StatusCode != 201 {
		log.Printf("Failed to create host %v: %#v\n", jhj.Name, resp.Status)
	} else {
		log.Printf("Added host %v\n", jhj.Name)
	}
}
