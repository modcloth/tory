package expvarplus

import (
	"expvar"
	"os"
	"strings"
)

var (
	// EnvWhitelist is used to expose specific env vars via the "env" expvar
	EnvWhitelist = []string{}
)

func init() {
	expvar.Publish("env", expvar.Func(env))
}

func env() interface{} {
	env := map[string]string{}

	for _, key := range EnvWhitelist {
		if len(key) > 0 {
			env[key] = os.Getenv(key)
		}
	}

	for _, key := range strings.Split(os.Getenv("EXPVARPLUS_WHITELIST"), ",") {
		if len(key) > 0 {
			env[key] = os.Getenv(key)
		}
	}

	return env
}
