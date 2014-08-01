package tory

import (
	"os"

	"github.com/Sirupsen/logrus"
	"github.com/codegangsta/negroni"
	"github.com/gorilla/mux"
	"github.com/meatballhat/negroni-logrus"
)

var (
	DefaultServerAddr = ":" + os.Getenv("PORT")
)

func init() {
	if DefaultServerAddr == ":" {
		DefaultServerAddr = ":9462"
	}
}

func ServerMain(addr string, verbose bool) {
	newServer(verbose).Run(addr)
}

type server struct {
	log *logrus.Logger
	n   *negroni.Negroni
	r   *mux.Router
}

func newServer(verbose bool) *server {
	srv := &server{
		log: logrus.New(),
		n:   negroni.New(),
		r:   mux.NewRouter(),
	}

	if verbose {
		srv.log.Level = logrus.DebugLevel
	}

	srv.n.Use(negronilogrus.NewMiddleware())
	srv.n.UseHandler(srv.r)
	return srv
}

func (srv *server) Run(addr string) {
	srv.n.Run(addr)
}
