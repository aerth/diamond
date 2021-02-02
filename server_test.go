package diamond

import (
	"fmt"
	"net/http"
	"os"
	"sync"
	"testing"
	"time"
)

var tmpsocket = "tmp-delete-me.socket"

func TestDiamond1(t *testing.T) {
	os.Remove(tmpsocket)
	s, err := New(tmpsocket)
	if err != nil {
		t.Fatal(err)
	}
	webserver := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "it works")
	})
	s.AddHTTPHandler("127.0.0.1:23333", webserver)
	go func(t *testing.T, s *Server) {
		<-time.After(time.Second * 10)
		t.Log("Shutting down tests, we have a timeout")
		if err := s.Runlevel(0); err != nil {
			t.Fatalf("error timeout: %v", err)
		}
		t.Fail()
	}(t, s)

	cycleThruLevels(t, s)
	wg := sync.WaitGroup{}
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			cycleThruLevels(t, s)
			wg.Done()
		}()
	}
	wg.Wait()
	t.Log("test: calling runlvl 4, then 0.")
	s.Runlevel(4)
	s.Runlevel(0)
	if err := s.Wait(); err != nil {
		t.Fatal(err)
	}
}

func BenchLevels(t *testing.B) {
	os.Remove(tmpsocket)
	s, err := New(tmpsocket)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i <= t.N; i++ {
		for lvl := 1; lvl < 10; lvl++ {
			if err := s.Runlevel(lvl); err != nil {
				t.Errorf("switch to level %d: %v", lvl, err)
			}
		}
		for lvl := 9; lvl >= 1; lvl-- {
			if err := s.Runlevel(lvl); err != nil {
				t.Errorf("switch to level %d: %v", lvl, err)
			}
		}
	}
}

func cycleThruLevels(t *testing.T, s *Server) {
	for lvl := 1; lvl <= 4; lvl++ {
		if err := s.Runlevel(lvl); err != nil {
			t.Errorf("test error switch to level %d: %v", lvl, err)
		}
	}

	for lvl := 4; lvl >= 1; lvl-- {
		if err := s.Runlevel(lvl); err != nil {
			t.Errorf("test error switch to level %d: %v", lvl, err)
		}
	}
}
