package diamond

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strings"
)

// configT fields are not exported so that
// This type is used in the Server struct
type configT struct {
	name     string // user friendly name
	addr     string // :8080 (Short for 0.0.0.0:8080) or 127.0.0.1:8080 (Only localhost)
	socket   string // path of socket file to create (/tmp/diamond.sock)
	level    int
	debug    bool
	kicks    bool
	kickable bool
	log      string // directory to write logs
}

// ConfigT fields are exported only so that JSON can be read from the config file.
// This type is not used otherwise.
type cconfigT struct {
	Name     string // user friendly name
	Addr     string // :8080 (Short for 0.0.0.0:8080) or 127.0.0.1:8080 (Only localhost)
	Socket   string // path of socket file to create (/tmp/diamond.sock)
	Level    int
	Debug    bool
	Kicks    bool   // will kick to launch
	Kickable bool   // able to be kicked
	Log      string // directory to write logs
}

func readconf(path string) (configT, error) {
	b, e := ioutil.ReadFile(path)
	if e != nil {
		if !strings.Contains(e.Error(), "No such") {
			log.Println(e)
			return configT{}, nil // return no error, no config
		}
		return configT{}, e
	}
	if b == nil {
		return configT{}, errors.New("Empty: " + path)
	}
	return readconfigJSON(b)
}
func readconfigJSON(b []byte) (configT, error) {
	var c configT
	var bigc cconfigT
	e := json.Unmarshal(b, &bigc)
	if e != nil {
		return configT{}, e
	}

	// All fields are blank
	if bigc.Addr == "" && !bigc.Debug && bigc.Level == 0 && bigc.Name == "" && bigc.Socket == "" {
		return configT{}, errors.New("Bad config, need fields: Name, Socket, Addr, Debug")
	}

	//unexport values
	c.addr = bigc.Addr
	c.debug = bigc.Debug
	c.kickable = bigc.Kickable
	c.kicks = bigc.Kicks
	c.level = bigc.Level
	c.log = bigc.Log
	c.name = bigc.Name
	c.socket = bigc.Socket

	// Some blank are OK
	return c, parseconf(c)
}

func parseconf(c configT) error {
	var e1, e2 error
	// Check valid ADDR
	if c.addr != "" {
		_, e1 = net.ResolveTCPAddr("tcp", c.addr)
	}

	// Check valid FILENAME
	if c.socket != "" {
		_, e2 = os.Open(c.socket)

		if e2 != nil {
			if strings.Contains(e2.Error(), "no such") {
				e2 = nil
			}

		}
	}
	if c.level != 3 && c.level != 1 {
		if e1 != nil {
			e1 = errors.New(e1.Error() + " AND incorrect default runlevel")
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

// Transfer the values of configT to s.config
func (s *Server) doconfig(conf configT) error {
	if conf.addr != "" {
		s.config.addr = conf.addr
	}
	if conf.debug {
		s.config.debug = conf.debug
	}
	if conf.name != "" {
		s.config.name = conf.name
	}
	if conf.socket != "" {
		s.config.socket = conf.socket
	}

	if s.config.debug {
		s.ErrorLog.SetFlags(log.Lshortfile)
	}
	if s.config.socket == "" {
		tmpfile, er := ioutil.TempFile(os.TempDir(), "/diamond.socket-")
		if er != nil {

			return er
		}
		os.Remove(tmpfile.Name())
		s.config.socket = tmpfile.Name()
	}
	if s.config.name == "" {
		s.config.name = "⋄ Diamond"
	}
	s.config.level = conf.level
	if s.config.level != 3 && s.config.level != 1 {
		s.config.level = 1
	}
	s.config.kickable = conf.kickable
	s.config.kicks = conf.kicks
	s.config.log = conf.log

	s.configured = true // mark server configured
	return nil
}
