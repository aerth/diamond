/*
* The MIT License (MIT)
*
* Copyright (c) 2016,2017  aerth <aerth@riseup.net>
*
* Permission is hereby granted, free of charge, to any person obtaining a copy
* of this software and associated documentation files (the "Software"), to deal
* in the Software without restriction, including without limitation the rights
* to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
* copies of the Software, and to permit persons to whom the Software is
* furnished to do so, subject to the following conditions:
*
* The above copyright notice and this permission notice shall be included in all
* copies or substantial portions of the Software.
*
* THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
* IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
* FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
* AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
* LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
* OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
* SOFTWARE.
 */

package diamond

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Runlevel switches the current diamond server's runlevel
func (s *Server) Runlevel(i int) {
	if i == 0 {
		s.ErrorLog.Printf("ENTERING RUNLEVEL 0")
		go func() {
			s.telinit <- -1 // kill telinit goroutine
		}()
		s.runlevel0()
		return
	}

	s.ErrorLog.Println("runlevel request:", i)
	s.telinit <- i
	return
}

// CustomCommander allows the admin (via unix socket) to send custom commands
// Example duck function returns hello, world if admin sends "CUSTOM world" command:
//
// 	duck := func(args string, reply *string) error {
// 		*reply = fmt.Sprintf("Hello, %s", args)
// 		return nil
// 	}
//	s.CustomCommander(duck)
func (s *Server) CustomCommander(duck func(args string, reply *string) error) {
	s.customCommander = duck
}

// LevelString returns the current runlevel (string)
func (s *Server) LevelString() string {
	s.levellock.Lock()
	str := strconv.Itoa(s.level)
	s.levellock.Unlock()
	return str
}

// Level returns the current runlevel (int)
func (s *Server) Level() int {
	s.levellock.Lock()
	lvl := s.level
	s.levellock.Unlock()
	return lvl
}

// String returns diamond version
func (s *Server) String() string {
	return version
}

// Status returns a status report string
func (s *Server) Status() string {
	if s == nil {
		return ""
	}
	var out string
	out += fmt.Sprintf("Server Name: %s\n", s.Config.Name)
	out += fmt.Sprintf("Diamond Version: %s\n", version)
	out += fmt.Sprintf("Default Runlevel: %v\n", s.Config.Level)
	s.levellock.Lock()
	out += fmt.Sprintf("Current Runlevel: %v\n", s.level)
	str := listnstr(s.level)
	s.levellock.Unlock()
	out += fmt.Sprintf("Socket: %s\n", s.Config.Socket)
	out += fmt.Sprintf("Addr: %s (%s)\n", s.Config.Addr, str)
	out += fmt.Sprintf("Uptime: %s\n", time.Since(s.since))
	out += fmt.Sprintf("Recent Connections: %v\n", s.counters.Uint64("active"))
	out += fmt.Sprintf("Total Connections: %v\n", s.counters.Uint64("total"))
	if s.Config.Debug {
		out += fmt.Sprintf("Debug: %v\n", s.Config.Debug)
		wd, _ := os.Getwd()
		if wd != "" {
			out += fmt.Sprintf("Working Directory: %s\n", wd)
		}
		exe, _ := os.Executable()
		if exe != "" {
			out += fmt.Sprintf("Executable: %s", exe)
		}
	}
	return out
}

// Uptime returns duration since server creation
func (s *Server) Uptime() time.Duration {
	return time.Now().Sub(s.since)
}

// CountConnectionsActive returns the current active numbers of connections made to the diamond server
func (s *Server) CountConnectionsActive() uint64 {
	return s.counters.Uint64("active")
}

// CountConnectionsTotal returns the total numbers of connections made to the diamond server
func (s *Server) CountConnectionsTotal() uint64 {
	return s.counters.Uint64("total")
}

// Kick ! Another Diamond is occupying our socket. Let's kick it!
func (s *Server) Kick() string {
	client, e := NewClient(s.Config.Socket)
	if e != nil {
		return e.Error()
	}
	reply, e := client.Send("KICK")
	if e != nil {
		return reply + e.Error()
	}
	return reply

}
