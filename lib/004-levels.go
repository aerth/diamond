package diamond

import "fmt"

/*
This file is ignored by git and should be customized
*/

// custom multiuser mode -- DO NOT RENAME
func (s *Server) runlevel4() {
	demo := func() {
		fmt.Println("Entering Custom Runlevel")
		fmt.Println(s.Config)
	}

	demo()
}
