package client

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"

	"github.com/mattn/go-shellwords"
)

var (
	HostnameRegexp = regexp.MustCompile("^(([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\\-]*[a-zA-Z0-9])\\.)*([A-Za-z0-9]|[A-Za-z0-9][A-Za-z0-9\\-]*[A-Za-z0-9])$")
	IPAddrRegexp   = regexp.MustCompile("^((25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)(\\.|$)){4}$")
)

type AnsibleModuleArgs struct {
	Hostname   string            `json:"hostname"`
	ToryServer string            `json:"tory_server"`
	IP         string            `json:"ip"`
	Tags       map[string]string `json:"tags"`
	Vars       map[string]string `json:"vars"`
}

func AnsibleFail(msg string, status int) int {
	b, _ := json.MarshalIndent(map[string]interface{}{
		"failed": true,
		"msg":    msg,
		"rc":     status,
	}, "", "    ")

	fmt.Println(string(b))
	return status
}

func AnsibleAddHost(argsFile string) int {
	args, err := AnsibleGetArgs(argsFile)
	if err != nil {
		return AnsibleFail(err.Error(), 1)
	}

	st := 400
	err, st = PutHost(&RequestJSON{
		Name: args.Hostname,
		IP:   args.IP,
		Tags: args.Tags,
		Vars: args.Vars,
	})

	if err != nil {
		return AnsibleFail(err.Error(), 1)
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
		return AnsibleFail(err.Error(), 1)
	}

	fmt.Printf(string(b))
	return 0
}

func AnsibleGetArgs(argsFile string) (*AnsibleModuleArgs, error) {
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

	if !HostnameRegexp.Match([]byte(hostname)) {
		return nil, fmt.Errorf("invalid \"hostname\" argument")
	}

	ip, ok := argsMap["ip"]
	if !ok {
		return nil, fmt.Errorf("missing \"ip\" argument")
	}

	if !IPAddrRegexp.Match([]byte(ip)) {
		return nil, fmt.Errorf("invalid \"ip\" argument")
	}

	args := &AnsibleModuleArgs{
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
