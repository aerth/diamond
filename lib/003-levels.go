package diamond

import (
	"crypto/tls"
	"net"
	"os"
	"strings"
	"time"
)

// HookLevels are called at the end of each runlevel
var (
	HookLevel0  = func() {}
	DoneMessage = "Reached target shutdown"
	HookLevel1  = func() {}
	HookLevel2  = func() {}
	HookLevel3  = func() {}
	HookLevel4  = func() {}
)
var (
	// The ECDHE cipher suites are preferred for performance and forward
	// secrecy.  See https://community.qualys.com/blogs/securitylabs/2013/06/25/ssl-labs-deploying-forward-secrecy.
	preferredCipherSuites = []uint16{
		tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
		tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
		tls.TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA,
		tls.TLS_RSA_WITH_3DES_EDE_CBC_SHA,
		tls.TLS_RSA_WITH_AES_128_CBC_SHA,
		tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
	}
)

func socketExists(path string) bool {
	_, e := os.Open(path)
	if e != nil {
		if strings.Contains(e.Error(), "no such") {
			return false
		}
	}
	return true
}

// tear down and exit
func (s *Server) runlevel0() {
	s.ErrorLog.Printf("Shifted to runlevel 0")
	defer HookLevel0()
	defer func() { s.Done <- DoneMessage }()
	if s.listenerSocket == nil {
		s.ErrorLog.Printf("Socket disappeared")
		return
	}
	e := s.listenerSocket.Close()
	if e != nil {
		s.ErrorLog.Printf("%s", e)
	}

}

// single user mode
func (s *Server) runlevel1() {
	s.lock.Lock()
	s.runlevel6() // stop listeners
	s.level = 1
	<-time.After(1 * time.Second)
	HookLevel1()
	s.lock.Unlock()

}

// multiuser mode
func (s *Server) runlevel3() {
	if s.level == 3 {
		if s.Config.Debug {
			s.ErrorLog.Printf("Already in runlevel 3, switch to runlevel 1 first.")
		}
		return
	}
	s.lock.Lock()

	// not using defer unlock because httpserver will unlock properly.

	if s.Config.Debug {
		s.ErrorLog.Printf("Entering runlevel 3")
	}

	// start listening on s.Config.Addr (config or -http flag)
	l, err := net.Listen("tcp", s.Config.Addr)
	if err != nil {
		s.ErrorLog.Printf("** WARNING **: %s\n", err)
		s.ErrorLog.Printf("Reverting to runlevel: %v\n", s.level)
		s.lock.Unlock()
		return
	}

	s.listenerTCP = l

	if s.Config.UseTLS {
		// start listening on s.Config.TLSAddr (config or -http flag)
		cer, err := tls.LoadX509KeyPair(s.Config.TLSCertFile, s.Config.TLSKeyFile)
		if err != nil {
			s.ErrorLog.Printf("** TLS WARNING **: %s\n", err)
			s.ErrorLog.Printf("Reverting to runlevel: %v\n", s.level)
			s.lock.Unlock()
			s.Runlevel(s.level)
			return
		}
		config := &tls.Config{
			Certificates: []tls.Certificate{cer},
			CipherSuites: preferredCipherSuites,
			PreferServerCipherSuites: true,
	}
		tlsl, err := tls.Listen("tcp", s.Config.TLSAddr, config)
		if err != nil {
			s.ErrorLog.Printf("** TLS WARNING **: %s\n", err)
			s.ErrorLog.Printf("Reverting to runlevel: %v\n", s.level)
			s.lock.Unlock()
			s.Runlevel(s.level)
			return
		}

		s.listenerTLS = tlsl
	}

	//	s.handlerTCP = s.mux

	s.level = 3
	HookLevel3()
	s.serveHTTP()

}

/*

004-levels.go

*/

// restart into single user mode.
// should not be called by anything but other runlevel methods.

func (s *Server) runlevel6() {
	// s.lock is locked
	s.level = 6

	// disallow new multiuser connections

	if s.listenerTCP != nil {
		s.ErrorLog.Printf("Closing TCP listener: %s", s.Config.Addr)
		e := s.listenerTCP.Close()
		if e != nil {
			s.ErrorLog.Println(e)
		}
	}

	if s.listenerTCP != nil {
		s.listenerTCP = nil
	}

	if s.listenerTCP != nil {
		s.ErrorLog.Println("Cant close TLS Listener:", s.listenerTCP.Addr().String())
		s.listenerTLS = nil
	}

	if s.listenerTLS != nil {
		s.ErrorLog.Printf("Closing TLS listener: %s", s.Config.TLSAddr)
		e := s.listenerTLS.Close()
		if e != nil {
			s.ErrorLog.Println(e)
		}
	}

	if s.listenerTLS != nil {
		s.ErrorLog.Println("Cant close TLS Listener:", s.listenerTLS.Addr().String())
		s.listenerTLS = nil
	}

}
