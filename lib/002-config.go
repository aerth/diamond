package diamond

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"strings"
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
	NoHTTP      bool   // dont listen on HTTP
	UseTLS      bool   // also listen on TLS
	TLSAddr     string // TLS Addr required for TLS
	TLSCertFile string // TLS Certificate file location required for TLS
	TLSKeyFile  string // TLS Key file location required for TLS
}

// SaveConfig to file (JSON)
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
		file, err := os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0640)
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
		if !strings.Contains(e.Error(), "no such") {
			fmt.Println("⋄ config error", e)
			return ConfigFields{}, nil // return no error, no config
		}
		return ConfigFields{}, e
	}
	if b == nil {
		return ConfigFields{}, errors.New("Empty: " + path)
	}
	config := ConfigFields{}
	err := json.Unmarshal(b, &config)
	return config, err
}

func parseconf(c ConfigFields) error {
	var e1, e2 error
	// Check valid ADDR
	if c.Addr != "" {
		_, e1 = net.ResolveTCPAddr("tcp", c.Addr)
	}

	// Check valid FILENAME
	if c.Socket != "" {
		_, e2 = os.Open(c.Socket)

		if e2 != nil {
			if strings.Contains(e2.Error(), "no such") {
				e2 = nil
			}

		}
	}
	if c.Level != 3 && c.Level != 1 {
		if e1 != nil {
			e1 = errors.New(e1.Error() + " AND incorrect default runLevel")
		}
	}
	if e1 != nil && e2 != nil {
		return errors.New(e1.Error() + " AND " + e2.Error())
	}
	if e2 != nil {
		fmt.Println("returning e2")
		return e2
	}
	return e1
}
