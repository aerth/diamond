package diamond

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
)

// ConfigFields fields
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

// SaveConfig to file(s) (JSON)
func (s *Server) SaveConfig(filenames ...string) (n int, err error) {
	config := s.Config
	b, err := json.MarshalIndent(config, " ", " ")
	if err != nil {
		return n, err
	}
	if filenames == nil {
		filenames = []string{s.configpath}
	}
	for _, filename := range filenames {
		var n1 int
		file, err := os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, CHMODFILE)
		if err != nil {
			return n, err
		}
		if err = file.Truncate(0); err != nil {
			return n, err
		}
		b = append(b, "\n"...)
		n1, err = file.Write(b)
		n += n1
		if err != nil {
			return n1, err
		}
	}
	return n, nil
}

func readconf(path string) (ConfigFields, error) {
	b, e := ioutil.ReadFile(path)
	if e != nil {
		return ConfigFields{}, e
	}
	if b == nil {
		return ConfigFields{}, errors.New("Empty: " + path)
	}
	config := ConfigFields{}
	err := json.Unmarshal(b, &config)
	return config, err
}

// ConfigPath reads a config file
func (s *Server) ConfigPath(path string) error {
	conf, e := readconf(path)
	if e != nil {
		return e
	}
	s.Config = conf
	s.configpath = path
	return nil
}

// Configure a server using json []byte
func (s *Server) Configure(b []byte) error {
	var config ConfigFields
	err := json.Unmarshal(b, &config)
	if err != nil {
		return err
	}
	s.Config = config
	return nil
}
