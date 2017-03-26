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

// ConfigFields fields are exported only so that JSON can be read from the config file.
type ConfigFields struct {
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

// Save config to file (JSON)
func (config *ConfigFields) Save(filename string) (n int, err error) {
	b, err := json.Marshal(config)
	if err != nil {
		return 0, err
	}

	file, err := os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0640)
	if err != nil {
		return 0, err
	}
	if err := file.Truncate(0); err != nil {
		return 0, err
	}

	return file.Write(b)
}

func readconf(path string) (*ConfigFields, error) {
	b, e := ioutil.ReadFile(path)
	if e != nil {
		if !strings.Contains(e.Error(), "no such") {
			fmt.Println("⋄ config error", e)
			return nil, nil // return no error, no config
		}
		return &ConfigFields{}, e
	}
	if b == nil {
		return &ConfigFields{}, errors.New("Empty: " + path)
	}
	config := new(ConfigFields)
	err := json.Unmarshal(b, config)
	return config, err
}

func parseconf(c *ConfigFields) error {
	var e1, e2 error
	if c == nil {
		return nil
	}
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

//
// // Transfer the values of ConfigFields to s.Config
// func (s *Server) doconfig(conf *ConfigFields) error {
// 	if s.Config == nil {
// 		s.Config = new(ConfigFields)
// 	}
// 	if conf == nil {
// 		return errors.New("Need config location")
// 	}
// 	if conf.Addr != "" {
// 		s.Config.Addr = conf.Addr
// 	}
// 	if conf.Debug {
// 		s.Config.Debug = conf.Debug
// 	}
// 	if conf.Name != "" {
// 		s.Config.Name = conf.Name
// 	}
// 	if conf.Socket != "" {
// 		s.Config.Socket = conf.Socket
// 	}
//
// 	if s.Config.Debug {
// 		s.ErrorLog.SetFlags(log.Lshortfile)
// 	}
// 	if s.Config.Socket == "" {
// 		tmpfile, er := ioutil.TempFile(os.TempDir(), "/diamond.Socket-")
// 		if er != nil {
// 			return er
// 		}
// 		os.Remove(tmpfile.Name())
// 		s.Config.Socket = tmpfile.Name()
// 	}
// 	if s.Config.Name == "" {
// 		s.Config.Name = "⋄ Diamond"
// 	}
// 	s.Config.Level = conf.Level
// 	if s.Config.Level != 3 && s.Config.Level != 1 {
// 		s.Config.Level = 1
// 	}
// 	s.Config.Kickable = conf.Kickable
// 	s.Config.Kicks = conf.Kicks
// 	s.Config.Log = conf.Log
// 	return nil
// }
