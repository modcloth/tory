package tory

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/codegangsta/negroni"
	"github.com/gorilla/mux"
	"github.com/jingweno/negroni-gorelic"
	"github.com/meatballhat/maybestatic"
	"github.com/meatballhat/negroni-logrus"
	"github.com/modcloth/expvarplus"
	"github.com/phyber/negroni-gzip/gzip"
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

	expvarplus.AddToEnvWhitelist(
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
	)
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
	db, err := newDatabase(dbConnStr)
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

	srv.r.HandleFunc(srv.prefix+`/{hostname}`, srv.getHost).Methods("GET")
	srv.r.HandleFunc(srv.prefix+`/{hostname}`, srv.updateHost).Methods("PUT")
	srv.r.HandleFunc(srv.prefix+`/{hostname}`, srv.deleteHost).Methods("DELETE")

	srv.r.HandleFunc(srv.prefix+`/{hostname}/tags/{key}`, srv.getHostTag).Methods("GET")
	srv.r.HandleFunc(srv.prefix+`/{hostname}/tags/{key}`, srv.updateHostTag).Methods("PUT")
	srv.r.HandleFunc(srv.prefix+`/{hostname}/tags/{key}`, srv.deleteHostTag).Methods("DELETE")
	srv.r.HandleFunc(srv.prefix+`/{hostname}/vars/{key}`, srv.getHostVar).Methods("GET")
	srv.r.HandleFunc(srv.prefix+`/{hostname}/vars/{key}`, srv.updateHostVar).Methods("PUT")
	srv.r.HandleFunc(srv.prefix+`/{hostname}/vars/{key}`, srv.deleteHostVar).Methods("DELETE")

	srv.r.HandleFunc(`/ping`, srv.handlePing).Methods("GET", "HEAD")
	srv.r.HandleFunc(`/debug/vars`, expvarplus.HandleExpvars).Methods("GET")
	srv.r.Handle(`/`, http.RedirectHandler(`/index.html`, http.StatusFound))

	srv.n.Use(negroni.NewRecovery())

	if opts.NewRelicOptions.Enabled {
		srv.n.Use(negronigorelic.New(
			opts.NewRelicOptions.LicenseKey,
			opts.NewRelicOptions.AppName,
			opts.NewRelicOptions.Verbose))
	}

	srv.n.Use(gzip.Gzip(gzip.DefaultCompression))
	srv.n.Use(negroni.NewStatic(maybestatic.New(opts.StaticDir, Asset)))
	srv.n.Use(negronilogrus.NewMiddleware())
	srv.n.Use(newAuthMiddleware(opts.AuthToken))
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
	var err error
	sinceTime := zeroTime
	since := r.FormValue("since")
	if since != "" {
		sinceTime, err = time.Parse(time.RFC3339, since)
		if err != nil {
			srv.log.WithField("error", err).Warn("failed to parse \"since\" param")
		}
	}

	beforeTime := zeroTime
	before := r.FormValue("before")
	if before != "" {
		beforeTime, err = time.Parse(time.RFC3339, before)
		if err != nil {
			srv.log.WithField("error", err).Warn("failed to parse \"before\" param")
		}
	}

	hf := &hostFilter{
		Name:   r.FormValue("name"),
		Env:    r.FormValue("env"),
		Team:   r.FormValue("team"),
		Since:  sinceTime,
		Before: beforeTime,
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
		inv.AddHostnameToGroupUnsanitized(host.IP.Addr, host.Name)

		if host.Type.String != "" {
			inv.AddHostnameToGroup(fmt.Sprintf("type_%s",
				strings.ToLower(host.Type.String)), host.Name)
		}

		if host.Tags != nil && host.Tags.Map != nil {
			for key, value := range host.Tags.Map {
				if value.String == "" {
					continue
				}
				invKey := fmt.Sprintf("tag_%s_%s",
					strings.ToLower(key), strings.ToLower(value.String))
				inv.AddHostnameToGroup(invKey, host.Name)
			}
		}

		if r.FormValue("exclude-vars") != "" {
			continue
		}

		for key, value := range host.CollapsedVars() {
			inv.Meta.AddHostvar(host.Name, key, value)
		}
	}

	srv.sendJSON(w, inv, http.StatusOK)
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

	w.Header().Set("Location", path.Join(srv.prefix, h.Name))
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
	hu, err := srv.db.UpdateHost(h)
	if err != nil {
		if err == noHostInDatabaseError {
			srv.log.WithFields(logrus.Fields{
				"host": h.Name,
			}).Info("failed to update, so trying to create instead")
			hu, err = srv.db.CreateHost(h)
			st = http.StatusCreated
		} else {
			err = nil
		}
	}

	if err != nil {
		srv.sendError(w, err, http.StatusInternalServerError)
		return
	}

	huj := hostToHostJSON(hu)
	w.Header().Set("Location", path.Join(srv.prefix, hu.Name))
	srv.sendJSON(w, &HostPayload{Host: huj}, st)
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

	w.Header().Set("Location", path.Join(srv.prefix, hostname))
	srv.sendJSON(w, "", http.StatusNoContent)
}

func (srv *server) getHostKey(keyType string, w http.ResponseWriter, r *http.Request) {
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

	var (
		err   error
		value string
	)
	switch keyType {
	case "vars":
		value, err = srv.db.ReadVar(hostname, key)
	case "tags":
		value, err = srv.db.ReadTag(hostname, key)
	}
	if err != nil {
		if err == noTagError || err == noVarError {
			srv.sendNotFound(w, fmt.Sprintf("could not find %q", key))
			return
		}
		srv.sendError(w, err, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Location", path.Join(srv.prefix, hostname, keyType, key))
	srv.sendJSON(w, map[string]string{"value": value}, http.StatusOK)
}

func (srv *server) updateHostKey(keyType string, w http.ResponseWriter, r *http.Request) {
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
	switch keyType {
	case "vars":
		err = srv.db.UpdateVar(hostname, key, value)
	case "tags":
		err = srv.db.UpdateTag(hostname, key, value)
	}

	if err != nil {
		if err == noHostInDatabaseError {
			srv.sendNotFound(w, "no such host")
			return
		}
		srv.sendError(w, err, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Location", path.Join(srv.prefix, hostname, keyType, key))
	srv.sendJSON(w, map[string]string{"value": value}, st)
}

func (srv *server) deleteHostKey(keyType string, w http.ResponseWriter, r *http.Request) {
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

	var err error
	switch keyType {
	case "vars":
		err = srv.db.DeleteVar(hostname, key)
	case "tags":
		err = srv.db.DeleteTag(hostname, key)
	}
	if err != nil {
		srv.sendError(w, err, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Location", path.Join(srv.prefix, hostname, keyType, key))
	srv.sendJSON(w, "", http.StatusNoContent)
}

func (srv *server) getHostVar(w http.ResponseWriter, r *http.Request) {
	srv.getHostKey("vars", w, r)
}

func (srv *server) updateHostVar(w http.ResponseWriter, r *http.Request) {
	srv.updateHostKey("vars", w, r)
}

func (srv *server) deleteHostVar(w http.ResponseWriter, r *http.Request) {
	srv.deleteHostKey("vars", w, r)
}

func (srv *server) getHostTag(w http.ResponseWriter, r *http.Request) {
	srv.getHostKey("tags", w, r)
}

func (srv *server) updateHostTag(w http.ResponseWriter, r *http.Request) {
	srv.updateHostKey("tags", w, r)
}

func (srv *server) deleteHostTag(w http.ResponseWriter, r *http.Request) {
	srv.deleteHostKey("tags", w, r)
}
