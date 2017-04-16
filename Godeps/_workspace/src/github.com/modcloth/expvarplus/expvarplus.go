package expvarplus

import (
	"encoding/json"
	"expvar"
	"fmt"
	"net/http"
)

// HandleExpvars does the same thing as the private expvar.expvarHandler, but
// exposed as public for pluggability into other web frameworks and generates
// json in a maybe slightly kinda more sane way (???).
func HandleExpvars(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	vars := map[string]interface{}{}

	expvar.Do(func(kv expvar.KeyValue) {
		var unStrd interface{}
		json.Unmarshal([]byte(kv.Value.String()), &unStrd)
		vars[kv.Key] = unStrd
	})

	jsonBytes, err := json.MarshalIndent(vars, "", "    ")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error":%q}`, err.Error())
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, string(jsonBytes)+"\n")
}
