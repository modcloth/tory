package main

import (
	"fmt"
	"os"

	"github.com/modcloth/tory/tory/client"
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
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, usage)
		fmt.Fprintf(os.Stderr, "ERROR: Missing <args-file> argument\n")
		os.Exit(1)
	}

	argsFile := os.Args[1]

	if argsFile == "-h" || argsFile == "--help" {
		fmt.Fprintf(os.Stderr, usage)
		os.Exit(0)
	}

	os.Exit(client.AnsibleAddHost(argsFile))
}
