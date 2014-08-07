package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/mattn/go-shellwords"
)

var (
	usage = `Usage: tory_add_host [-h/--help] <args-file>

Example usage within some ansible yaml, probably in a task list:

    - name: register this host in tory
      delegate_to: 127.0.0.1
      tory_add_host:
        hostname={{ ansible_fqdn }}
        ip={{ ansible_default_ipv4.address }}
        tag_team=hosers
        tag_env={{ env }}
        tag_role={{ primary_role }}
        var_whatever={{ something_from_somewhere }}
        var_this_playbook="{{ lookup('env', 'USER') }} {{ ansible_date_time.iso8601 }}"
        var_special=true

`
	hostnameRegexp = regexp.MustCompile("^(([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\\-]*[a-zA-Z0-9])\\.)*([A-Za-z0-9]|[A-Za-z0-9][A-Za-z0-9\\-]*[A-Za-z0-9])$")
	ipAddrRegexp   = regexp.MustCompile("^((25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)(\\.|$)){4}$")
)

type moduleArgs struct {
	Hostname   string            `json:"hostname"`
	ToryServer string            `json:"tory_server"`
	IP         string            `json:"ip"`
	Tags       map[string]string `json:"tags"`
	Vars       map[string]string `json:"vars"`
}

type hostJSON struct {
	Name string            `json:"name"`
	IP   string            `json:"ip"`
	Tags map[string]string `json:"tags,omitempty"`
	Vars map[string]string `json:"vars,omitempty"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, usage)
		fmt.Fprintf(os.Stderr, "ERROR: Missing <args-file> argument\n")
		os.Exit(1)
	}

	os.Exit(addHost(os.Args[1]))
}

func fail(msg string, status int) int {
	b, _ := json.MarshalIndent(map[string]interface{}{
		"failed": true,
		"msg":    msg,
		"rc":     status,
	}, "", "    ")

	fmt.Println(string(b))
	return status
}

func addHost(argsFile string) int {
	if argsFile == "-h" || argsFile == "--help" {
		fmt.Fprintf(os.Stderr, usage)
		return 0
	}

	args, err := getArgs(argsFile)
	if err != nil {
		return fail(err.Error(), 1)
	}

	st := 400
	err, st = putHost(args)
	if err != nil {
		return fail(err.Error(), 1)
	}

	msg := "registered host"

	failed := st != 201 && st != 200
	if failed {
		msg = "host registration failed"
	}

	out := map[string]interface{}{
		"changed":  st == 201 || st == 200,
		"msg":      msg,
		"failed":   failed,
		"hostname": args.Hostname,
		"ip":       args.IP,
	}

	for key, value := range args.Tags {
		out[fmt.Sprintf("tag_%s", key)] = value
	}

	for key, value := range args.Vars {
		out[fmt.Sprintf("var_%s", key)] = value
	}

	b, err := json.MarshalIndent(out, "", "    ")

	if err != nil {
		return fail(err.Error(), 1)
	}

	fmt.Printf(string(b))
	return 0
}

func getArgs(argsFile string) (*moduleArgs, error) {
	fd, err := os.Open(argsFile)
	if err != nil {
		return nil, err
	}

	argsBytes, err := ioutil.ReadAll(fd)
	if err != nil {
		return nil, err
	}

	rawArgs, err := shellwords.Parse(string(argsBytes))
	if err != nil {
		return nil, err
	}

	argsMap := map[string]string{}
	for _, arg := range rawArgs {
		parts := strings.SplitN(arg, "=", 2)
		if len(parts) < 2 {
			argsMap[parts[0]] = ""
			continue
		}

		argsMap[parts[0]] = parts[1]
	}

	hostname, ok := argsMap["hostname"]
	if !ok {
		return nil, fmt.Errorf("missing \"hostname\" argument")
	}

	if !hostnameRegexp.Match([]byte(hostname)) {
		return nil, fmt.Errorf("invalid \"hostname\" argument")
	}

	ip, ok := argsMap["ip"]
	if !ok {
		return nil, fmt.Errorf("missing \"ip\" argument")
	}

	if !ipAddrRegexp.Match([]byte(ip)) {
		return nil, fmt.Errorf("invalid \"ip\" argument")
	}

	args := &moduleArgs{
		Hostname:   hostname,
		ToryServer: os.Getenv("TORY_SERVER"),
		IP:         ip,
		Tags:       map[string]string{},
		Vars:       map[string]string{},
	}

	if toryServer, ok := argsMap["tory_server"]; ok {
		args.ToryServer = toryServer
	}

	if args.ToryServer == "" {
		return nil, fmt.Errorf("no tory server provided in env " +
			"(as $TORY_SERVER) or module args (as \"tory_server\")")
	}

	for key, value := range argsMap {
		if strings.HasPrefix(key, "tag_") {
			args.Tags[strings.Replace(key, "tag_", "", 1)] = value
		}
		if strings.HasPrefix(key, "var_") {
			args.Vars[strings.Replace(key, "var_", "", 1)] = value
		}
	}

	return args, nil
}

func putHost(args *moduleArgs) (error, int) {
	hj := &hostJSON{
		Name: args.Hostname,
		IP:   args.IP,
		Tags: args.Tags,
		Vars: args.Vars,
	}

	hjBytes, err := json.Marshal(map[string]*hostJSON{"host": hj})
	if err != nil {
		return err, 500
	}

	buf := bytes.NewReader(hjBytes)
	req, err := http.NewRequest("PUT", args.ToryServer+"/"+hj.Name, buf)
	if err != nil {
		return err, 500
	}

	req.Header.Add("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err, 500
	}

	return nil, resp.StatusCode
}
