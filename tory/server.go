package tory

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/Sirupsen/logrus"
	"github.com/bitly/go-simplejson"
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

	// DefaultPrefix is the default value for the public API prefix
	DefaultPrefix = os.Getenv("TORY_PREFIX")

	toryLog = logrus.New()

	missingHostsError = fmt.Errorf("missing \"hosts\" key")
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

	if DefaultPrefix == "" {
		DefaultPrefix = `/ansible/hosts`
	}

	if os.Getenv("QUIET") != "" {
		toryLog.Level = logrus.FatalLevel
	}

	expvarplus.EnvWhitelist = []string{
		"TORY_ADDR",
		"TORY_PREFIX",
		"TORY_STATIC_DIR",
		"DATABASE_URL",
	}
}

// ServerMain is the whole shebang
func ServerMain(addr, dbConnStr, staticDir, prefix string, verbose bool) {
	srv := buildServer(addr, dbConnStr, staticDir, prefix, verbose)
	srv.Run(addr)
}

func buildServer(addr, dbConnStr, staticDir, prefix string, verbose bool) *server {
	os.Setenv("TORY_ADDR", addr)
	os.Setenv("TORY_STATIC_DIR", staticDir)
	os.Setenv("TORY_PREFIX", prefix)
	os.Setenv("DATABASE_URL", dbConnStr)

	srv, err := newServer(dbConnStr)
	if err != nil {
		toryLog.WithFields(logrus.Fields{"err": err}).Fatal("failed to build server")
	}
	srv.Setup(prefix, staticDir, verbose)
	return srv
}

type server struct {
	prefix string

	log *logrus.Logger
	db  *database
	n   *negroni.Negroni
	r   *mux.Router
}

func newServer(dbConnStr string) (*server, error) {
	db, err := newDatabase(dbConnStr, nil)
	if err != nil {
		return nil, err
	}

	srv := &server{
		prefix: `/ansible/hosts`,
		log:    logrus.New(),
		db:     db,
		n:      negroni.New(),
		r:      mux.NewRouter(),
	}

	return srv, nil
}

func (srv *server) Setup(prefix, staticDir string, verbose bool) {
	srv.prefix = prefix

	if verbose {
		srv.log.Level = logrus.DebugLevel
	}

	if os.Getenv("QUIET") != "" {
		srv.log.Level = logrus.FatalLevel
	}

	srv.r.HandleFunc(srv.prefix, srv.getHostInventory).Methods("GET")
	srv.r.HandleFunc(srv.prefix, srv.addHostToInventory).Methods("POST")
	srv.r.HandleFunc(srv.prefix+`/{hostname}`, srv.getHost).Methods("GET")
	srv.r.HandleFunc(srv.prefix+`/{hostname}`, srv.updateHost).Methods("PUT")
	srv.r.HandleFunc(srv.prefix+`/{hostname}`, srv.deleteHost).Methods("DELETE")
	srv.r.HandleFunc(srv.prefix+`/{hostname}/{key:.*}`, srv.getHostKey).Methods("GET")
	srv.r.HandleFunc(srv.prefix+`/{hostname}/{key:.*}`, srv.updateHostKey).Methods("PUT")
	srv.r.HandleFunc(srv.prefix+`/{hostname}/{key:.*}`, srv.deleteHostKey).Methods("DELETE")

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

func (srv *server) sendError(w http.ResponseWriter, err error, status int) {
	srv.log.WithFields(logrus.Fields{"err": err, "status": status}).Error("returning HTTP error")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	fmt.Fprintf(w, `{"error":%q}`, err.Error())
}

func (srv *server) sendJSON(w http.ResponseWriter, j interface{}, status int) {
	jsonBytes, err := json.MarshalIndent(j, "", "    ")
	if err != nil {
		srv.sendError(w, err, http.StatusInternalServerError)
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	fmt.Fprintf(w, string(jsonBytes)+"\n")
}

func (srv *server) handlePing(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "PONG\n")
}

func (srv *server) getHostInventory(w http.ResponseWriter, r *http.Request) {
	hosts, err := srv.db.ReadAllHosts()
	if err != nil {
		srv.sendError(w, err, http.StatusInternalServerError)
		return
	}

	inv := newInventory()
	for _, host := range hosts {
		inv.AddIPToGroupUnsanitized(host.Name, host.IP.Addr)

		if host.Type.String != "" {
			switch host.Type.String {
			case "smartmachine":
				inv.Meta.AddHostvar(host.IP.Addr,
					"ansible_python_interpreter", "/opt/local/bin/python")
			case "virtualmachine":
				inv.Meta.AddHostvar(host.IP.Addr,
					"ansible_python_interpreter", "/usr/bin/python")
			}

			inv.AddIPToGroup(fmt.Sprintf("type_%s", host.Type.String), host.IP.Addr)
		}

		if host.Tags != nil && host.Tags.Map != nil {
			for key, value := range host.Tags.Map {
				if value.String == "" {
					continue
				}
				invKey := fmt.Sprintf("tag_%s_%s", key, value.String)
				inv.AddIPToGroup(invKey, host.IP.Addr)
			}
		}
	}

	srv.sendJSON(w, inv, http.StatusOK)
}

func (srv *server) addHostToInventory(w http.ResponseWriter, r *http.Request) {
	j, err := simplejson.NewFromReader(r.Body)
	if err != nil {
		srv.sendError(w, err, http.StatusBadRequest)
		return
	}

	hjJSON, ok := j.CheckGet("host")
	if !ok {
		srv.sendError(w, missingHostsError, http.StatusBadRequest)
		return
	}

	hostBytes, err := hjJSON.MarshalJSON()
	if err != nil {
		srv.sendError(w, err, http.StatusBadRequest)
		return
	}

	hj := newHostJSON()
	err = json.Unmarshal(hostBytes, hj)
	if err != nil {
		srv.sendError(w, err, http.StatusBadRequest)
		return
	}

	h := hostJSONToHost(hj)
	err = srv.db.CreateHost(h)
	if err != nil {
		srv.sendError(w, err, http.StatusBadRequest)
		return
	}

	hj.ID = h.ID
	w.Header().Set("Location", srv.prefix+"/"+hj.Name)
	srv.sendJSON(w, map[string]*hostJSON{"host": hj}, http.StatusCreated)
}

func (srv *server) getHost(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "NOPE, no host", http.StatusNotImplemented)
}

func (srv *server) updateHost(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "NOPE, cannot update host", http.StatusNotImplemented)
}

func (srv *server) deleteHost(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "NOPE, cannot delete host", http.StatusNotImplemented)
}

func (srv *server) getHostKey(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "NOPE, no host key", http.StatusNotImplemented)
}

func (srv *server) updateHostKey(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "NOPE, no host key", http.StatusNotImplemented)
}

func (srv *server) deleteHostKey(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "NOPE, cannot delete host key", http.StatusNotImplemented)
}
