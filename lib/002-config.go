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

// ConfigT fields are exported only so that JSON can be read from the config file.
type ConfigT struct {
	Name        string // user friendly name
	Addr        string // :8080 (Short for 0.0.0.0:8080) or 127.0.0.1:8080 (Only localhost)
	Socket      string // path of Socket file to create (/tmp/diamond.sock)
	Level       int
	Debug       bool
	Kicks       bool   // will kick to launch
	Kickable    bool   // able to be kicked
	DoCycleTest bool   // do 1-3-default cycle at launch
	Log         string // directory to write logs
}

func readconf(path string) (ConfigT, error) {
	b, e := ioutil.ReadFile(path)
	if e != nil {
		if !strings.Contains(e.Error(), "No such") {
			log.Println(e)
			return ConfigT{}, nil // return no error, no config
		}
		return ConfigT{}, e
	}
	if b == nil {
		return ConfigT{}, errors.New("Empty: " + path)
	}
	return readconfigJSON(b)
}
func readconfigJSON(b []byte) (ConfigT, error) {
	var c ConfigT
	e := json.Unmarshal(b, &c)
	if e != nil {
		return ConfigT{}, e
	}

	// All fields are blank
	if c.Addr == "" && !c.Debug && c.Level == 0 && c.Name == "" && c.Socket == "" {
		return ConfigT{}, errors.New("Bad config, need fields: Name, Socket, Addr, Debug")
	}
	//
	// //unexport values
	// c.Addr = bigc.Addr
	// c.debug = bigc.Debug
	// c.kickable = bigc.Kickable
	// c.kicks = bigc.Kicks
	// c.Level = bigc.Level
	// c.log = bigc.Log
	// c.name = bigc.Name
	// c.Socket = bigc.Socket

	// Some blank are OK
	return c, parseconf(c)
}

func parseconf(c ConfigT) error {
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

// Transfer the values of ConfigT to s.Config
func (s *Server) doconfig(conf ConfigT) error {
	if conf.Addr != "" {
		s.Config.Addr = conf.Addr
	}
	if conf.Debug {
		s.Config.Debug = conf.Debug
	}
	if conf.Name != "" {
		s.Config.Name = conf.Name
	}
	if conf.Socket != "" {
		s.Config.Socket = conf.Socket
	}

	if s.Config.Debug {
		s.ErrorLog.SetFlags(log.Lshortfile)
	}
	if s.Config.Socket == "" {
		tmpfile, er := ioutil.TempFile(os.TempDir(), "/diamond.Socket-")
		if er != nil {

			return er
		}
		os.Remove(tmpfile.Name())
		s.Config.Socket = tmpfile.Name()
	}
	if s.Config.Name == "" {
		s.Config.Name = "⋄ Diamond"
	}
	s.Config.Level = conf.Level
	if s.Config.Level != 3 && s.Config.Level != 1 {
		s.Config.Level = 1
	}
	s.Config.Kickable = conf.Kickable
	s.Config.Kicks = conf.Kicks
	s.Config.Log = conf.Log

	s.configured = true // mark server configured
	return nil
}
