// +build go1.8

package diamond

// Server runs
import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"
)

const stderr = "stderr"

// Server runlevels
// *  0 = halt (NOT os.Exit(0))
// *  1 = single user mode (default) kills listenerTCP
// *  3 = multiuser mode (public http) boots listenerTCP
type Server struct {
	Server   *http.Server `json:"-"` // s.Server is created immediately before serving in runlevel 3
	ErrorLog *log.Logger  `json:"-"`
	Config   ConfigFields
	Done     chan string `json:"-"`

	// boot time, used for uptime duration
	since time.Time

	// deadline, not implemented
	until time.Time

	// current runlevel
	level     int
	levellock sync.Mutex // guards only shifting between runlevels
	telinit   chan int   // accepts runlevel requests

	// Socket listener that accepts admin commands
	listenerSocket net.Listener
	socketed       bool // true if we have started listening on a socket
	customCommander func(args string, reply *string) error

	// TCP Listener that can be stopped
	listenerTCP net.Listener
	listenerTLS net.Listener

	configpath string // path to config file

	//numconn, allconn int64      // count connections, used by s.Status()
	//counter          sync.Mutex // guards only conn counter writes within connection
	counters mucount
	//mux              http.Handler //

	signal bool // handle signals like SIGTERM gracefully

	// old
	//listenlock  sync.Mutex
	//configured  bool         // has been configured
}

type mucount struct {
	m  map[string]uint64
	mu sync.Mutex // guards map
}

func (m *mucount) Up(t ...string) (current uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, i := range t {
		m.m[i]++
		current = m.m[i]
	}
	return
}
func (m *mucount) Zero(t ...string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, i := range t {
		m.m[i] = 0
	}
	return
}

func (m *mucount) Down(t ...string) (current uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, i := range t {
		if m.m[i] >= 1 {
			m.m[i]--
		}
		current = m.m[i]
	}
	return
}

func (m *mucount) Uint64(t string) (current uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	current = m.m[t]
	return
}

func (s *Server) signalcatch() {
	if !s.signal {
		return
	}
	quitchan := make(chan os.Signal, 1)
	signal.Notify(quitchan, os.Interrupt, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM)
	select {
	case sig := <-quitchan:
		println("Diamond got signal:", sig.String()) // stderr
		s.ErrorLog.Println("Diamond got signal:", sig.String())
		s.Runlevel(0)
	}
}

// LevelString returns the current runlevel as a string
func (s *Server) LevelString() string {
	return strconv.Itoa(s.level)
}

// Level returns the current runlevel
func (s *Server) Level() int {
	return s.level
}

// String returns a status report string
func (s *Server) String() string {
	return s.Status()
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

// Uptime returns duration since boot
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

// Human readable
func listnstr(i int) string {
	if i >= 3 {
		return "Listening"
	}
	return "Not Listening"
}

// Print to log (from net/http)
func (s *Server) logf(format string, args ...interface{}) {
	if s.ErrorLog != nil {
		s.ErrorLog.Printf(format, args...)
	}
}
func (s *Server) log(args ...interface{}) {
	if s.ErrorLog != nil {
		s.ErrorLog.Println(args...)
	}
}

var once sync.Once

func (s *Server) telcom() {

	go s.signalcatch()
	for {

		select {

		case newlevel := <-s.telinit:
			s.levellock.Lock()
			s.levellock.Unlock()

			switch newlevel {
			case -1:
				s.ErrorLog.Println("TELCOM down")
				return
			case 0:
				s.ErrorLog.Printf("Shifting to runlevel 0")
				s.runlevel0()

				for {
					<-time.After(1 * time.Second)
					s.ErrorLog.Print("Can't switch to runlevel 0")
					fmt.Println("Can't switch to runlevel 0")
				}
			case 1:
				s.ErrorLog.Printf("Shifting to runlevel 1")
				s.runlevel1()

			case 3:
				s.ErrorLog.Printf("Shifting to runlevel 3")
				s.runlevel3()
				<-time.After(300 * time.Millisecond)
			case 4:
				s.ErrorLog.Printf("Shifting to runlevel 4")
				s.Runlevel4()
				<-time.After(300 * time.Millisecond)
			default:
				s.ErrorLog.Printf("BAD RUNLEVEL: %v", newlevel)
			}

		}
	}
}
