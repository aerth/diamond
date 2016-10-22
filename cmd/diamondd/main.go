// command example is an example server using the diamond architecture
package main

import (
	"flag"
	"fmt"
	"net/http"

	diamond "github.com/aerth/diamond/lib"
	"github.com/gorilla/mux"
)

var (
	version    string
	srv        *diamond.Server
	debug      = flag.Bool("v", false, "verbose logs")
	logpath    = flag.String("log", "stdout", "path to log directory")
	configpath = flag.String("conf", "", "path to JSON config")
	httpaddr   = flag.String("http", ":8777", "Listen on IP:port or :port")
	socketpath = flag.String("s", "/tmp/diamond.socket", "Path to UNIX socket")
)

func init() {
	//signal.Notify(quitchan, os.Interrupt, syscall.SIGHUP, syscall.SIGQUIT)
}

func main() {
	flag.Parse()

	srv = diamond.NewServer(route())

	// Config File
	if *configpath != "" {
		err := srv.ConfigPath(*configpath)
		if err != nil {
			panic(err)
		}
	} else {
		// Config JSON bytes
		b := []byte(`{"Socket":"/tmp/dia.sock",
                "Addr":"` + *httpaddr + `",
		"Kicks": true,
		"Kickable": true,
                "Level":1,
                "Log":"` + *logpath + `",
                "Socket":"` + *socketpath + `",
                "Debug":` + fmt.Sprint(*debug) + `}`)
		err := srv.Config(b)
		if err != nil {
			panic(err)
		}
	}
	// Start the server
	e := srv.Start()
	if e != nil {
		panic(e)
	}

	// Run until collapse
	select {}

}

// The rest is up to you
func route() *mux.Router {
	var route = new(mux.Router)
	route.NotFoundHandler = http.HandlerFunc(nilhandler)
	route.Handle("/status", http.HandlerFunc(statusHandler))
	route.Handle("/logo.png", http.HandlerFunc(logohandler))
	route.Handle("/style.css", http.HandlerFunc(stylehandler))
	route.Handle("/jquery.js", http.HandlerFunc(jshandler))
	route.Handle("/", http.HandlerFunc(homehandler))
	route.Handle("/{hello}/world", http.HandlerFunc(hellohandler))
	return route

}

func logohandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "html/logo.png")
}

func homehandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "html/index.html")
}
func hellohandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	w.Write([]byte(vars["hello"] + " world!"))
}
func stylehandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "html/style.css")
}
func jshandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "html/jquery.js")
}

// Return a 503 "Service Unavailable" message everywhere
func nilhandler(w http.ResponseWriter, r *http.Request) {
	http.Error(w, http.StatusText(503), 503)
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(srv.Status()))
}
