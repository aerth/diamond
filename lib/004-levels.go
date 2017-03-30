package diamond

// Runlevel4 can be redefined as well as HookLevel4
func (s *Server) Runlevel4() {
	demo := func() {
		s.ErrorLog.Println("Entering Custom Runlevel")
	}
	demo()
	HookLevel4()
}
