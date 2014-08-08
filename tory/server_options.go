package tory

// ServerOptions contains everything needed to build a Server
type ServerOptions struct {
	Addr        string
	AuthToken   string
	DatabaseURL string
	Prefix      string
	Quiet       bool
	StaticDir   string
	Verbose     bool
}
