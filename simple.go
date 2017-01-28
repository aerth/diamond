// The most simple Diamond ⋄ server
package main

import diamond "github.com/aerth/diamond/lib" // ⋄
/*
This Diamond only serves 404 pages!
*/
func main() {
	// Create new diamond.Server
	s := diamond.NewServer()
	s.ErrorLog.Println("Open in browser: http://127.0.0.1:8777/status")
	// Try config.json
	e := s.ConfigPath("config.json")
	if e != nil {
		s.Configure(ccc)
	}
	// Start the server, using a nil handler (404 for every page)
	s.Start()
	// Just keep doing that
	select {}
}

var ccc = []byte(`{
            "Name":"Diamonds! ⋄",
            "Level":3,
            "Addr":":8777",
            "Socket":"/tmp/diamond.socket",
            "Kicks": true,
            "Kickable": true,
    }`)
