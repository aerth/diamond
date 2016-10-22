// The most simple Diamond ⋄
package main

import diamond "github.com/aerth/diamond/lib" // ⋄
/*
This Diamond only serves 404 pages!
*/
func main() {
	// Create new diamond.Server
	s := diamond.NewServer()
	s.ErrorLog.Println("[404 only mode]")
	// Try config.json
	e := s.ConfigPath("config.json")
	if e != nil {
		s.ErrorLog.Println(e)
	}
	// Start the server, using a nil handler (404 for every page)
	s.Start()
	// Just keep doing that
	select {}
}
