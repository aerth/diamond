package diamond

// ConfigFields as seen in s.Config
type ConfigFields struct {
	Name        string // user friendly name
	Addr        string // :8080 (Short for 0.0.0.0:8080) or 127.0.0.1:8080 (Only localhost)
	Socket      string // path of Socket file to create (/tmp/diamond.sock)
	Level       int
	Debug       bool
	Kicks       bool // will kick to launch
	Kickable    bool // able to be kicked
	DoCycleTest bool // do 1-3-default cycle at launch

	// ssl options
	NoHTTP      bool // dont listen on HTTP
	UseTLS      bool // also listen on TLS
	RedirectTLS bool // open special handler on 80 that only redirects to 443

	RedirectHost string // which host to redirect to
	TLSAddr      string // TLS Addr required for TLS
	TLSCertFile  string // TLS Certificate file location required for TLS
	TLSKeyFile   string // TLS Key file location required for TLS
}
