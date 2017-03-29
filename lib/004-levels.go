package diamond

import "fmt"

// Runlevel4 can be redefined
func (s *Server) Runlevel4() {
	demo := func() {
		fmt.Println("Entering Custom Runlevel")
	}

	demo()

	HookLevel4()
}
