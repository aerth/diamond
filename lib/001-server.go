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

const stdout = "stdout"

// Server runlevels
// *  0 = halt (os.Exit(0))
// *  1 = single user mode (default) kills listenerTCP
// *  3 = multiuser mode (public http) boots listenerTCP
type Server struct {
	// boot time, used for uptime duration
	since time.Time

	// deadline, not implemented
	until time.Time

	// runlevel
	level int

	// Log out to where ever, default stdout
	ErrorLog *log.Logger

	// Socket listener that accepts admin commands
	listenerSocket net.Listener

	// TCP Listener that can be stopped
	listenerTCP *stoplisten.StoppableListener
	listenlock  sync.Mutex
	telinit     chan int      // accepts runlevel requests
	lock        sync.Mutex    // guards only shifting between runlevels
	Config      *ConfigFields // parsed config
	configpath  string        // path to config file
	configured  bool          // has been configured

	numconn, allconn int          // count connections, used by s.Status()
	counter          sync.Mutex   // guards only conn counter writes
	mux              http.Handler // given by package main in with s.Start(mux http.Handler)
	Server           *http.Server `json:"-"` // s.Server is created immediately before serving in runlevel 3
	socketed         bool         // true if we have started listening on a socket
	signal           bool         // false if we should not telinit 0 when receive os signal
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
	signal.Notify(quitchan, os.Interrupt, syscall.SIGHUP, syscall.SIGQUIT)

	go func() {
		select {
		case sig := <-quitchan:
			fmt.Println("Diamond got signal:", sig.String())
			s.Runlevel(0)
			os.Exit(0)
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
	return fmt.Sprintf("Server Name: %s\nDiamond Version: %s\nCurrent Runlevel: %v\nDebug: %v\n"+
		"Socket: %s\nAddr: %s (%s)\nDefault Level: %v\nUptime: %s\n"+
		"Active Connections: %v\nTotal Connections: %v\nPath: %s\nExecutable: %s",
		s.Config.Name, version, s.level, s.Config.Debug,
		s.Config.Socket,
		s.Config.Addr,
		str,
		s.Config.Level, time.Since(s.since), s.numconn,
		s.allconn, os.Getenv("PWD"), exeinfo())

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
			s.lock.Lock()
			s.lock.Unlock()

			if s.Config.Debug {
				s.ErrorLog.Printf("runlevel request: %v", newlevel)
			}

			switch newlevel {
			case 0:

				s.runlevel0()
				for {
					time.Sleep(1 * time.Second)
					s.ErrorLog.Print("Can't switch to runlevel 0")
					fmt.Println("Can't switch to runlevel 0")
				}
			case 1:
				s.ErrorLog.Printf("Shifting to runlevel 1")
				s.runlevel1()
				s.ErrorLog.Printf("Shifted to runlevel 1")

			case 3:
				s.ErrorLog.Printf("Shifting to runlevel 3")
				s.runlevel3()
				time.Sleep(300 * time.Millisecond)
				s.ErrorLog.Printf("Shifted to runlevel 3")

			case 4:
				s.ErrorLog.Printf("Shifting to runlevel 4")
				s.Runlevel4()
				time.Sleep(300 * time.Millisecond)
				s.ErrorLog.Printf("Shifted to runlevel 4")
			default:
				s.ErrorLog.Printf("BAD RUNLEVEL: %v", newlevel)
			}

		}
	}
}

// switch to log file if not stdout
func (s *Server) dolog() error {
	// empty logfile string is stdout
	if s.Config.Log == "" {
		s.Config.Log = stdout
	}
	// user didn't chose stdout
	if s.Config.Log != stdout {
		f, err := os.OpenFile(s.Config.Log, os.O_APPEND|os.O_RDWR|os.O_CREATE, CHMODFILE)
		if err == nil {
			s.ErrorLog.SetOutput(f)
		}
		if err != nil {
			return err
		}
		fmt.Println("Diamond log:", s.Config.Log)
	}
	return nil
}
