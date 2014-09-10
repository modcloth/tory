package main

import (
	"os"

	"github.com/codegangsta/negroni"
	"github.com/gorilla/mux"
	"github.com/modcloth/expvarplus"
)

func main() {
	n := negroni.Classic()
	r := mux.NewRouter()

	r.HandleFunc(`/debug/vars`, expvarplus.HandleExpvars).Methods("GET")
	n.UseHandler(r)

	addr := ":" + os.Getenv("PORT")
	if addr == ":" {
		addr = ":3000"
	}

	n.Run(addr)
}
