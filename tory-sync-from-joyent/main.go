package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/bitly/go-simplejson"
	"github.com/codegangsta/cli"
	"github.com/modcloth/tory/tory"
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

	sjJSON, err := simplejson.NewFromReader(fd)
	if err != nil {
		log.Fatal(err.Error())
	}

	i := 0
	for {
		j := sjJSON.GetIndex(i)
		if j == nil {
			break
		}

		if _, ok := j.CheckGet("name"); !ok {
			break
		}

		syncOneMachine(server, j)

		i++
	}

	log.Println("Ding!")
}

func syncOneMachine(server string, j *simplejson.Json) {
	hjJSONBytes, err := j.MarshalJSON()
	if err != nil {
		log.Fatal(err.Error())
	}

	jhj := &joyentHostJSON{
		Tags: map[string]interface{}{},
	}
	err = json.Unmarshal(hjJSONBytes, jhj)
	if err != nil {
		log.Fatal(err.Error())
	}

	hj := tory.NewHostJSON()
	hj.Name = jhj.Name
	hj.Package = jhj.Package
	hj.Image = jhj.Image
	hj.Type = jhj.Type
	hj.IP = jhj.IP
	for key, value := range jhj.Tags {
		hj.Tags[key] = fmt.Sprintf("%s", value)
	}
	hj.Vars["disk"] = fmt.Sprintf("%d", jhj.Disk)
	hj.Vars["memory"] = fmt.Sprintf("%d", jhj.Memory)

	hjBytes, err := json.Marshal(map[string]*tory.HostJSON{"host": hj})
	if err != nil {
		log.Fatal(err.Error())
	}

	buf := bytes.NewReader(hjBytes)
	resp, err := http.Post(server, "application/json", buf)
	if err != nil {
		log.Fatal(err.Error())
	}

	if resp.StatusCode != 201 {
		log.Printf("Failed to create host %v: %#v\n", hj.Name, resp.Status)
	} else {
		log.Printf("Added host %v\n", hj.Name)
	}
}
