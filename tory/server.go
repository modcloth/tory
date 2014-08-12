package tory

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/codegangsta/negroni"
	"github.com/gorilla/mux"
	"github.com/meatballhat/negroni-logrus"
	"github.com/modcloth/expvarplus"
)

var (
	toryLog = logrus.New()

	mismatchedHostError   = fmt.Errorf("host in body does not match path")
	noHostnameInPathError = fmt.Errorf("no hostname in PATH_INFO")
	noKeyInPathError      = fmt.Errorf("no key in PATH_INFO")
	noValueKeyError       = fmt.Errorf("no value key in payload")
)

func init() {
	port := os.Getenv("PORT")
	if port != "" && os.Getenv("TORY_ADDR") == "" {
		os.Setenv("TORY_ADDR", ":"+port)
	}

	expvarplus.EnvWhitelist = []string{
		"DATABASE_URL",
		"PORT",
		"QUIET",
		"TORY_ADDR",
		"TORY_BRANCH",
		"TORY_GENERATED",
		"TORY_PREFIX",
		"TORY_REVISION",
		"TORY_STATIC_DIR",
		"TORY_VERSION",
		"VERBOSE",
	}
}

// ServerMain is the whole shebang
func ServerMain(opts *ServerOptions) {
	buildServer(opts).Run(opts.Addr)
}

func buildServer(opts *ServerOptions) *server {
	os.Setenv("TORY_ADDR", opts.Addr)
	os.Setenv("TORY_STATIC_DIR", opts.StaticDir)
	os.Setenv("TORY_PREFIX", opts.Prefix)
	os.Setenv("DATABASE_URL", opts.DatabaseURL)

	srv, err := newServer(opts.DatabaseURL)
	if err != nil {
		toryLog.WithFields(logrus.Fields{"err": err}).Fatal("failed to build server")
	}

	srv.Setup(opts)
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

func (srv *server) Setup(opts *ServerOptions) {
	srv.prefix = opts.Prefix

	if opts.Verbose {
		srv.log.Level = logrus.DebugLevel
	}

	if opts.Quiet {
		srv.log.Level = logrus.FatalLevel
	}

	srv.db.Log = srv.log

	srv.r.HandleFunc(srv.prefix, srv.getHostInventory).Methods("GET")
	srv.r.HandleFunc(srv.prefix+`/{hostname}/tags/{key}`, srv.getHostTag).Methods("GET")
	srv.r.HandleFunc(srv.prefix+`/{hostname}/tags/{key}`, srv.updateHostTag).Methods("PUT")
	srv.r.HandleFunc(srv.prefix+`/{hostname}/tags/{key}`, srv.deleteHostTag).Methods("DELETE")
	srv.r.HandleFunc(srv.prefix+`/{hostname}/vars/{key}`, srv.getHostVar).Methods("GET")
	srv.r.HandleFunc(srv.prefix+`/{hostname}/vars/{key}`, srv.updateHostVar).Methods("PUT")
	srv.r.HandleFunc(srv.prefix+`/{hostname}/vars/{key}`, srv.deleteHostVar).Methods("DELETE")
	//	srv.r.HandleFunc(srv.prefix+`/{hostname}/{key}`, srv.getHostKey).Methods("GET")
	//	srv.r.HandleFunc(srv.prefix+`/{hostname}/{key}`, srv.updateHostKey).Methods("PUT")
	//	srv.r.HandleFunc(srv.prefix+`/{hostname}/{key}`, srv.deleteHostKey).Methods("DELETE")
	srv.r.HandleFunc(srv.prefix+`/{hostname}`, srv.getHost).Methods("GET")
	srv.r.HandleFunc(srv.prefix+`/{hostname}`, srv.updateHost).Methods("PUT")
	srv.r.HandleFunc(srv.prefix+`/{hostname}`, srv.deleteHost).Methods("DELETE")

	srv.r.HandleFunc(`/ping`, srv.handlePing).Methods("GET", "HEAD")
	srv.r.HandleFunc(`/debug/vars`, expvarplus.HandleExpvars).Methods("GET")

	srv.n.Use(negroni.NewRecovery())
	srv.n.Use(negroni.NewStatic(http.Dir(opts.StaticDir)))
	srv.n.Use(negronilogrus.NewMiddleware())
	srv.n.Use(newAuthMiddleware(opts.AuthToken))
	srv.n.UseHandler(srv.r)
}

func (srv *server) Run(addr string) {
	srv.n.Run(addr)
}

func (srv *server) sendNotFound(w http.ResponseWriter, msg string) {
	srv.sendJSON(w, map[string]string{"message": msg}, http.StatusNotFound)
}

func (srv *server) sendError(w http.ResponseWriter, err error, status int) {
	srv.log.WithFields(logrus.Fields{"err": err, "status": status}).Error("returning HTTP error")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	fmt.Fprintf(w, `{"error":%q}`, err.Error())
}

func (srv *server) sendUnauthorized(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", "token")
	srv.sendJSON(w, map[string]string{"error": "unauthorized"}, http.StatusUnauthorized)
}

func (srv *server) sendJSON(w http.ResponseWriter, j interface{}, status int) {
	jsonBytes, err := json.MarshalIndent(j, "", "    ")
	if err != nil {
		srv.sendError(w, err, http.StatusInternalServerError)
	}

	jsonString := string(jsonBytes) + "\n"

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(jsonString)))
	w.WriteHeader(status)
	fmt.Fprintf(w, jsonString)
}

func (srv *server) isAuthed(r *http.Request) bool {
	return r.Header.Get("Tory-Authorized") == "yep"
}

func (srv *server) handlePing(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "PONG\n")
}

func (srv *server) getHostInventory(w http.ResponseWriter, r *http.Request) {
	hf := &hostFilter{
		Name: r.FormValue("name"),
		Env:  r.FormValue("env"),
		Team: r.FormValue("team"),
	}
	srv.log.WithFields(logrus.Fields{
		"filter": hf,
	}).Debug("reading hosts with vars and filter")

	hosts, err := srv.db.ReadAllHosts(hf)
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

			inv.AddIPToGroup(fmt.Sprintf("type_%s",
				strings.ToLower(host.Type.String)), host.IP.Addr)
		}

		if r.FormValue("exclude-vars") == "" {
			for key, value := range host.CollapsedVars() {
				inv.Meta.AddHostvar(host.IP.Addr, key, value)
			}
		}

		if host.Tags != nil && host.Tags.Map != nil {
			for key, value := range host.Tags.Map {
				if value.String == "" {
					continue
				}
				invKey := fmt.Sprintf("tag_%s_%s",
					strings.ToLower(key), strings.ToLower(value.String))
				inv.AddIPToGroup(invKey, host.IP.Addr)
			}
		}
	}

	srv.sendJSON(w, inv, http.StatusOK)
}

func (srv *server) addHostToInventory(w http.ResponseWriter, r *http.Request) {
	hj, err := hostJSONFromHTTPBody(r.Body)
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
	srv.sendJSON(w, map[string]*HostJSON{"host": hj}, http.StatusCreated)
}

func (srv *server) getHost(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	hostname, ok := vars["hostname"]
	if !ok {
		srv.sendError(w, noHostnameInPathError, http.StatusBadRequest)
		return
	}

	h, err := srv.db.ReadHost(hostname)
	srv.log.WithField("host", fmt.Sprintf("%#v", h)).Info("got back the host")
	if err != nil {
		srv.sendNotFound(w, "no such host")
		return
	}

	w.Header().Set("Location", srv.prefix+"/"+h.Name)
	srv.log.Info("sending back some json now")

	if r.FormValue("vars-only") != "" {
		srv.sendJSON(w, h.CollapsedVars(), http.StatusOK)
	} else {
		srv.sendJSON(w, map[string]*HostJSON{"host": hostToHostJSON(h)}, http.StatusOK)
	}
}

func (srv *server) updateHost(w http.ResponseWriter, r *http.Request) {
	if !srv.isAuthed(r) {
		srv.sendUnauthorized(w)
		return
	}

	vars := mux.Vars(r)
	hostname, ok := vars["hostname"]
	if !ok {
		srv.sendError(w, noHostnameInPathError, http.StatusBadRequest)
		return
	}

	srv.log.WithField("vars", vars).Debug("beginning host update handling")

	hj, err := hostJSONFromHTTPBody(r.Body)
	if err != nil {
		srv.sendError(w, err, http.StatusBadRequest)
		return
	}

	if hj.Name != hostname {
		srv.sendError(w, mismatchedHostError, http.StatusBadRequest)
		return
	}

	h := hostJSONToHost(hj)

	srv.log.WithFields(logrus.Fields{
		"host":     fmt.Sprintf("%#v", h),
		"hostJSON": fmt.Sprintf("%#v", hj),
		"ip":       h.IP,
	}).Debug("attempting to update host")

	st := http.StatusOK
	err = srv.db.UpdateHost(h)
	if err != nil {
		if err == noHostInDatabaseError {
			srv.log.WithFields(logrus.Fields{
				"host": h.Name,
			}).Info("failed to update, so trying to create instead")
			err = srv.db.CreateHost(h)
			st = http.StatusCreated
		} else {
			err = nil
		}
	}

	if err != nil {
		srv.sendError(w, err, http.StatusInternalServerError)
		return
	}

	hj.ID = h.ID

	w.Header().Set("Location", srv.prefix+"/"+hj.Name)
	srv.sendJSON(w, &HostPayload{Host: hj}, st)
}

func (srv *server) deleteHost(w http.ResponseWriter, r *http.Request) {
	if !srv.isAuthed(r) {
		srv.sendUnauthorized(w)
		return
	}

	vars := mux.Vars(r)
	hostname, ok := vars["hostname"]
	if !ok {
		srv.sendError(w, noHostnameInPathError, http.StatusBadRequest)
		return
	}

	srv.log.WithField("vars", vars).Debug("beginning host delete handling")

	err := srv.db.DeleteHost(hostname)
	if err != nil {
		if err == noHostInDatabaseError {
			srv.sendNotFound(w, "no such host")
			return
		} else {
			srv.sendError(w, err, http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Location", srv.prefix+"/"+hostname)
	srv.sendJSON(w, "", http.StatusNoContent)
}

func (srv *server) getHostVar(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	hostname, ok := vars["hostname"]
	if !ok {
		srv.sendError(w, noHostnameInPathError, http.StatusBadRequest)
		return
	}

	key, ok := vars["key"]
	if !ok {
		srv.sendError(w, noKeyInPathError, http.StatusBadRequest)
		return
	}

	value, err := srv.db.ReadVar(hostname, key)
	if err != nil {
		if err == noVarError {
			srv.sendNotFound(w, fmt.Sprintf("no var %q", key))
			return
		}
		srv.sendError(w, err, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Location", srv.prefix+"/"+hostname+"/vars/"+key)
	srv.sendJSON(w, map[string]string{"value": value}, http.StatusOK)
}

func (srv *server) updateHostVar(w http.ResponseWriter, r *http.Request) {
	if !srv.isAuthed(r) {
		srv.sendUnauthorized(w)
		return
	}

	vars := mux.Vars(r)

	hostname, ok := vars["hostname"]
	if !ok {
		srv.sendError(w, noHostnameInPathError, http.StatusBadRequest)
		return
	}

	key, ok := vars["key"]
	if !ok {
		srv.sendError(w, noKeyInPathError, http.StatusBadRequest)
		return
	}

	input := map[string]string{}
	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		srv.sendError(w, err, http.StatusInternalServerError)
		return
	}

	value, ok := input["value"]
	if !ok {
		srv.sendError(w, noValueKeyError, http.StatusBadRequest)
		return
	}

	st := http.StatusOK
	err = srv.db.UpdateVar(hostname, key, value)

	if err != nil {
		if err == noHostInDatabaseError {
			srv.sendNotFound(w, "no such host")
			return
		}
		srv.sendError(w, err, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Location", srv.prefix+"/"+hostname+"/vars/"+key)
	srv.sendJSON(w, map[string]string{"value": value}, st)
}

func (srv *server) deleteHostVar(w http.ResponseWriter, r *http.Request) {
	if !srv.isAuthed(r) {
		srv.sendUnauthorized(w)
		return
	}

	vars := mux.Vars(r)

	hostname, ok := vars["hostname"]
	if !ok {
		srv.sendError(w, noHostnameInPathError, http.StatusBadRequest)
		return
	}

	key, ok := vars["key"]
	if !ok {
		srv.sendError(w, noKeyInPathError, http.StatusBadRequest)
		return
	}

	err := srv.db.DeleteVar(hostname, key)
	if err != nil {
		srv.sendError(w, err, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Location", srv.prefix+"/"+hostname+"/vars/"+key)
	srv.sendJSON(w, "", http.StatusNoContent)
}

func (srv *server) getHostTag(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	hostname, ok := vars["hostname"]
	if !ok {
		srv.sendError(w, noHostnameInPathError, http.StatusBadRequest)
		return
	}

	key, ok := vars["key"]
	if !ok {
		srv.sendError(w, noKeyInPathError, http.StatusBadRequest)
		return
	}

	value, err := srv.db.ReadTag(hostname, key)
	if err != nil {
		if err == noTagError {
			srv.sendNotFound(w, fmt.Sprintf("no var %q", key))
			return
		}
		srv.sendError(w, err, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Location", srv.prefix+"/"+hostname+"/tags/"+key)
	srv.sendJSON(w, map[string]string{"value": value}, http.StatusOK)
}

func (srv *server) updateHostTag(w http.ResponseWriter, r *http.Request) {
	if !srv.isAuthed(r) {
		srv.sendUnauthorized(w)
		return
	}

	vars := mux.Vars(r)

	hostname, ok := vars["hostname"]
	if !ok {
		srv.sendError(w, noHostnameInPathError, http.StatusBadRequest)
		return
	}

	key, ok := vars["key"]
	if !ok {
		srv.sendError(w, noKeyInPathError, http.StatusBadRequest)
		return
	}

	input := map[string]string{}
	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		srv.sendError(w, err, http.StatusInternalServerError)
		return
	}

	value, ok := input["value"]
	if !ok {
		srv.sendError(w, noValueKeyError, http.StatusBadRequest)
		return
	}

	st := http.StatusOK
	err = srv.db.UpdateTag(hostname, key, value)

	if err != nil {
		if err == noHostInDatabaseError {
			srv.sendNotFound(w, "no such host")
			return
		}
		srv.sendError(w, err, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Location", srv.prefix+"/"+hostname+"/tags/"+key)
	srv.sendJSON(w, map[string]string{"value": value}, st)
}

func (srv *server) deleteHostTag(w http.ResponseWriter, r *http.Request) {
	if !srv.isAuthed(r) {
		srv.sendUnauthorized(w)
		return
	}

	vars := mux.Vars(r)

	hostname, ok := vars["hostname"]
	if !ok {
		srv.sendError(w, noHostnameInPathError, http.StatusBadRequest)
		return
	}

	key, ok := vars["key"]
	if !ok {
		srv.sendError(w, noKeyInPathError, http.StatusBadRequest)
		return
	}

	err := srv.db.DeleteTag(hostname, key)
	if err != nil {
		srv.sendError(w, err, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Location", srv.prefix+"/"+hostname+"/tags/"+key)
	srv.sendJSON(w, "", http.StatusNoContent)
}
