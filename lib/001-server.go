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

	stoplisten "github.com/hydrogen18/stoppableListener"
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
	level   int
	lock    sync.Mutex // guards only shifting between runlevels
	telinit chan int   // accepts runlevel requests

	// Socket listener that accepts admin commands
	listenerSocket net.Listener
	socketed       bool // true if we have started listening on a socket

	// TCP Listener that can be stopped
	listenerTCP *stoplisten.StoppableListener
	listenerTLS *stoplisten.StoppableListener

	configpath string // path to config file

	numconn, allconn int64      // count connections, used by s.Status()
	counter          sync.Mutex // guards only conn counter writes within connection
	//mux              http.Handler //

	signal bool // handle signals like SIGTERM gracefully

	// old
	//listenlock  sync.Mutex
	//configured  bool         // has been configured
}

// Level returns the current runlevel
func (s *Server) Level() int {
	return s.level
}

func (s *Server) signalcatch() {
	if !s.signal {
		return
	}
	quitchan := make(chan os.Signal, 1)
	signal.Notify(quitchan, os.Interrupt, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM)

	go func() {
		select {
		case sig := <-quitchan:
			println("Diamond got signal:", sig.String()) // stderr
			s.ErrorLog.Println("Diamond got signal:", sig.String())
			s.Runlevel(0)
		}

	}()

}

// LevelString returns the current runlevel as a string
func (s *Server) LevelString() string {
	return strconv.Itoa(s.level)
}

// Status returns a status report string
func (s *Server) Status() string {
	if s == nil {
		return ""
	}
	str := listnstr(s.level)
	s.lock.Lock()
	out := fmt.Sprintf("Server Name: %s\nDiamond Version: %s\nCurrent Runlevel: %v\nDebug: %v\n"+
		"Socket: %s\nAddr: %s (%s)\nDefault Level: %v\nUptime: %s\n"+
		"Active Connections: %v\nTotal Connections: %v\nPath: %s\nExecutable: %s",
		s.Config.Name, version, s.level, s.Config.Debug,
		s.Config.Socket,
		s.Config.Addr,
		str,
		s.Config.Level, time.Since(s.since), s.numconn,
		s.allconn, os.Getenv("PWD"), exeinfo())
	s.lock.Unlock()
	return out
}

// Human readable
func listnstr(i int) string {
	if i >= 3 {
		return "Listening"
	}
	return "Not Listening"
}

// CountConnections returns the total numbers of connections made to the diamond server
func (s *Server) CountConnections() int64 {
	s.lock.Lock()
	num := s.allconn
	s.lock.Unlock()
	return num
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
			s.lock.Lock()
			s.lock.Unlock()

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

// // switch to log file if not stderr
// func (s *Server) dolog() error {
// 	// empty logfile string is stderr
// 	if s.Config.Log == "" {
// 		s.Config.Log = stderr
// 	}
// 	// user didn't chose stderr
// 	if s.Config.Log != stderr {
// 		f, err := os.OpenFile(s.Config.Log, os.O_APPEND|os.O_RDWR|os.O_CREATE, CHMODFILE)
// 		if err == nil {
// 			s.ErrorLog.SetOutput(f)
// 		}
// 		if err != nil {
// 			return err
// 		}
// 		s.ErrorLog.Fprintln(os.Stderr, "Diamond log:", s.Config.Log)
// 	}
// 	return nil
// }
