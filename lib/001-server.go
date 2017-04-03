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
// *  0 = halt (os.Exit(0))
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

// Status returns a status report string
func (s *Server) Status() string {
	if s == nil {
		return ""
	}
	str := listnstr(s.level)
	s.levellock.Lock()
	out := fmt.Sprintf("Server Name: %s\nDiamond Version: %s\nCurrent Runlevel: %v\nDebug: %v\n"+
		"Socket: %s\nAddr: %s (%s)\nDefault Level: %v\nUptime: %s\n"+
		"Active Connections: %v\nTotal Connections: %v\nPath: %s\nExecutable: %s",
		s.Config.Name, version, s.level, s.Config.Debug,
		s.Config.Socket,
		s.Config.Addr,
		str,
		s.Config.Level, time.Since(s.since), s.counters.Uint64("active"),
		s.counters.Uint64("total"), os.Getenv("PWD"), exeinfo())
	s.levellock.Unlock()
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
