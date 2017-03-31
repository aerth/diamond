package diamond

import (
	"fmt"
	"strings"
	"time"

	"github.com/aerth/spawn"
)

func (s *Server) respawn() {
	s.ErrorLog.Printf("Respawning %s", time.Now())
	spawn.Spawn()
}

// Kick ! Another Diamond is occupying our socket. Let's kick it!
func (s *Server) Kick() string {
	client := NewClient(s.Config.Socket)
	r, e := client.Send("KICK")
	if e != nil {
		if strings.Contains(e.Error(), "no such file or directory") {
			return ""
		}
		return e.Error()
	}

	return r

}

func exeinfo() string {
	self, _, args := spawn.Exe()
	return fmt.Sprintln(self, args)

}
