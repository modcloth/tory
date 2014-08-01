package tory

import (
	"fmt"
	"net/http"
	"os"

	"github.com/Sirupsen/logrus"
	"github.com/codegangsta/negroni"
	"github.com/gorilla/mux"
	"github.com/meatballhat/negroni-logrus"
	"github.com/modcloth/expvarplus"
)

var (
	// DefaultServerAddr is the default value for the server address
	DefaultServerAddr = ":" + os.Getenv("PORT")

	// DefaultStaticDir is the default value for the static directory
	DefaultStaticDir = os.Getenv("TORY_STATIC_DIR")

	log = logrus.New()
)

func init() {
	if DefaultServerAddr == ":" {
		DefaultServerAddr = os.Getenv("TORY_ADDR")
	}

	if DefaultServerAddr == ":" || DefaultServerAddr == "" {
		DefaultServerAddr = ":9462"
	}

	if DefaultStaticDir == "" {
		DefaultStaticDir = "public"
	}

	expvarplus.EnvWhitelist = []string{
		"TORY_ADDR",
		"TORY_STATIC_DIR",
		"DATABASE_URL",
	}
}

// ServerMain is the whole shebang
func ServerMain(addr, dbConnStr, staticDir string, verbose bool) {
	os.Setenv("TORY_ADDR", addr)
	os.Setenv("TORY_STATIC_DIR", staticDir)
	os.Setenv("DATABASE_URL", dbConnStr)

	srv, err := newServer(dbConnStr)
	if err != nil {
		log.WithFields(logrus.Fields{"err": err}).Fatal("failed to build server")
	}
	srv.Setup(staticDir, verbose)
	srv.Run(addr)
}

type server struct {
	log *logrus.Logger
	d   *db
	n   *negroni.Negroni
	r   *mux.Router
}

func newServer(dbConnStr string) (*server, error) {
	d, err := newDB(dbConnStr)
	if err != nil {
		return nil, err
	}

	srv := &server{
		log: logrus.New(),
		d:   d,
		n:   negroni.New(),
		r:   mux.NewRouter(),
	}

	return srv, nil
}

func (srv *server) Setup(staticDir string, verbose bool) {
	if verbose {
		srv.log.Level = logrus.DebugLevel
	}

	srv.r.HandleFunc(`/ping`, srv.handlePing).Methods("GET", "HEAD")
	srv.r.HandleFunc(`/debug/vars`, expvarplus.HandleExpvars).Methods("GET")

	srv.n.Use(negroni.NewRecovery())
	srv.n.Use(negroni.NewStatic(http.Dir(staticDir)))
	srv.n.Use(negronilogrus.NewMiddleware())
	srv.n.UseHandler(srv.r)
}

func (srv *server) Run(addr string) {
	srv.n.Run(addr)
}

func (srv *server) handlePing(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintf(w, "PONG\n")
}
