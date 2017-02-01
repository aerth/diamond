package diamond

//
// import (
// 	"fmt"
// 	"io/ioutil"
// 	"log"
// 	"math/rand"
// 	"net"
// 	"net/http"
// 	"os"
// 	"strconv"
// 	"strings"
// 	"testing"
// 	"time"
//
// 	"github.com/stretchr/testify/assert"
// )
//
// var s *Server
// var testhome string
//
// func init() {
// 	rand.Seed(time.Now().UnixNano())
// 	doOnce()
// }
//
// func doOnce() {
// 	if s != nil {
// 		return
// 	}
// 	s = NewServer(http.HandlerFunc(s.ServeStatus))
//
// 	tmpf, e := ioutil.TempFile(os.TempDir(), "/diamondtest.socket-")
// 	if e != nil {
// 		panic(e)
// 	}
// 	socketname := tmpf.Name()
// 	tmpf.Close()
// 	os.Remove(socketname)
// 	s.Config.socket = socketname
// 	s.Config.Addr = ":" + strconv.Itoa(30000+rand.Intn(1000))
// 	s.Config.name = "Test Diamond"
// 	s.Config.level = 1
// 	s.Config.debug = true
// 	s.configured = true
// 	if s.Config.debug {
// 		log.SetFlags(log.Llongfile)
// 	}
// 	http.DefaultClient.Timeout = time.Second * 3
// 	testhome = "http://127.0.0.1" + s.Config.Addr
// 	e = s.Start()
// 	if e != nil {
// 		panic(e)
// 	}
// }
//
// func get(t *testing.T, url string) []byte {
// 	resp, er := http.Get(url)
// 	if er != nil {
// 		if t != nil {
// 			log.Println(er)
// 			t.FailNow()
// 		}
// 		return nil
//
// 	}
// 	bod, er := ioutil.ReadAll(resp.Body)
// 	if er != nil {
// 		if t != nil {
// 			log.Println(er)
// 			t.FailNow()
// 		}
// 		return nil
// 	}
// 	e := resp.Body.Close()
// 	if e != nil {
// 		log.Println(e)
// 	}
// 	return bod
// }
//
// func TestAll(t *testing.T) {
// 	for i := 1; i <= 2; i++ {
// 		log.Println("testHTTPRunLevel1")
// 		testHTTPRunLevel1(t)
// 		if t.Failed() {
// 			log.Println("Failed on test A round: ", i)
// 			break
// 		}
// 		log.Println("testShiftRunlevel3")
// 		testShiftRunlevel3(t)
// 		if t.Failed() {
// 			log.Println("Failed on test B round: ", i)
// 			break
// 		}
// 		log.Println("testHTTPRunLevel3")
// 		testHTTPRunLevel3(t)
//
// 		if t.Failed() {
// 			log.Println("Failed on test C round: ", i)
// 			break
// 		}
// 		time.Sleep(time.Second)
// 		log.Println("testShiftRunlevel1")
// 		testShiftRunlevel1(t)
//
// 		if t.Failed() {
// 			log.Println("Failed on test D round: ", i)
// 			break
// 		}
// 		time.Sleep(time.Second)
// 	}
//
// }
//
// func testHTTPRunLevel1(t *testing.T) {
// 	if s.level != 1 {
// 		s.telinit <- 1
// 		time.Sleep(100 * time.Millisecond)
// 	}
//
// 	assert.Equal(t, 1, s.level)
//
// 	// make sure /status is 'connection refused' aka nothing at all (no response)
// 	_, er := http.Get(testhome + "/status")
//
// 	if er != nil {
// 		if !strings.Contains(er.Error(), "refused") {
// 			t.FailNow()
// 		}
// 	} else {
// 		t.FailNow()
// 	}
//
// 	_, er = http.Get(testhome + "/status")
// 	assert.NotNil(t, er)
// 	_, er = http.Get(testhome + "/")
// 	assert.NotNil(t, er)
// 	_, er = http.Get(testhome + "/123")
// 	assert.NotNil(t, er)
// 	_, er = http.Get(testhome + "/index.php")
// 	assert.NotNil(t, er)
// }
// func testShiftRunlevel3(t *testing.T) {
// 	s.telinit <- 3
// 	time.Sleep(1200 * time.Millisecond)
// 	assert.Equal(t, 3, s.level)
//
// }
// func testHTTPRunLevel3(t *testing.T) {
// 	if s.level != 3 {
// 		s.telinit <- 3
// 		time.Sleep(1200 * time.Millisecond)
// 	}
// 	assert.Equal(t, 3, s.level)
// 	bod := get(t, testhome+"/status")
// 	assert.True(t, strings.Contains(string(bod), "Current Runlevel: 3"))
// 	fmt.Println(string(bod))
// }
//
// // TestShiftRunlevel1 fails for a weird reason, maybe related to 'localhost',
// // or could be a diamond bug, forgetting to close a response body
// func testShiftRunlevel1(t *testing.T) {
//
// 	log.Println("Switching to runlevel 1")
// 	s.telinit <- 1
// 	time.Sleep(300 * time.Millisecond)
// 	log.Println("Switching to runlevel 3")
// 	s.telinit <- 3
//
// 	time.Sleep(1 * time.Second)
// 	bod := get(nil, testhome+"/status")
// 	assert.True(t, strings.Contains(string(bod), "Current Runlevel: 3"))
// 	log.Println("Switching to runlevel 1")
// 	s.telinit <- 1
// 	time.Sleep(300 * time.Millisecond)
// 	log.Println("Switching to runlevel 3")
// 	s.telinit <- 3
// 	time.Sleep(1 * time.Second)
// 	log.Println("Switching to runlevel 1")
// 	s.telinit <- 1
//
// 	time.Sleep(300 * time.Millisecond)
// 	assert.Equal(t, 1, s.level)
//
// 	resp, err := http.Get(testhome + "/status")
// 	assert.NotNil(t, err)
// 	if resp != nil {
// 		bod, _ = ioutil.ReadAll(resp.Body)
// 		resp.Body.Close()
// 		log.Println("Got response when we didn't want one:", string(bod))
//
// 	}
//
// }
//
// func testHandlerFail(w http.ResponseWriter, r *http.Request) {
// 	w.Write([]byte("FAIL"))
// }
//
// func testHandlerPass(w http.ResponseWriter, r *http.Request) {
// 	w.Write([]byte("PASS"))
// }
//
// func testTwice(t *testing.T) {
//
// 	// s2 := NewServer(nil) // should be nil
// 	// go admin(s2)
// 	// assert.Nil(t, s2.listenerSocket)
//
// }
//
// // func testB(t *testing.T) {
// //
// // 	s2 := NewServer(nil)
// // 	s2.config.socket = os.TempDir() + "/testsocket" + strconv.Itoa(rand.Intn(24)+100)
// // 	go admin(s2)
// // 	assert.Nil(t, s2.listenerSocket)
// // 	s2.telinit <- 3
// //
// // }
//
// func TestSocket(t *testing.T) {
// 	assert.NotEqual(t, 0, s.level)
//
// 	// check socket does exist
// 	sock := s.listenerSocket.Addr()
// 	assert.NotEmpty(t, sock.String())
// 	unix, _ := net.ResolveUnixAddr(sock.Network(), sock.String())
//
// 	con, er := net.DialUnix("unix", nil, unix)
// 	assert.Nil(t, er)
// 	e := con.Close()
//
// 	assert.Nil(t, e)
//
// 	fmt.Println("Closing Socket")
// 	// telinit 0
// 	s.telinit <- 0
// 	time.Sleep(time.Second)
// 	con, er = net.DialUnix("unix", nil, unix)
// 	if er != nil {
// 		log.Println("1", er)
// 	}
// 	if con != nil {
// 		f, e := con.File()
// 		log.Println("2", e)
// 		log.Println("3?", f.Name())
// 	}
// 	assert.Nil(t, con)
// 	log.Println()
// }
//
// func TestConfigJSONGOOD(t *testing.T) {
// 	tmpfile, e := ioutil.TempFile(os.TempDir(), "/diamondTest")
// 	if e != nil {
// 		fmt.Println(e)
// 		t.FailNow()
// 	}
// 	fmt.Println(tmpfile.Name())
// 	_, e = os.Create(tmpfile.Name())
// 	if e != nil {
// 		fmt.Println("Creating", e)
// 	}
// 	defer os.Remove(tmpfile.Name())
// 	tmpfile.Truncate(0)
// 	goodconfig := `
//
// 	{"Socket":"/tmp/testingZone","Addr":":8008","Name":"Testing Zone!"}
//
// 	`
// 	tmpfile.WriteString(goodconfig)
// 	tmpfile.Close()
// 	config, e := readconf(tmpfile.Name())
// 	if e != nil {
// 		fmt.Println("Er:", e)
// 	}
// 	assert.Nil(t, e)
// 	assert.Equal(t, ":8008", config.Addr)
// 	assert.False(t, Config.Debug)
// 	assert.Equal(t, "Testing Zone!", config.name)
// 	assert.Equal(t, "/tmp/testingZone", config.socket)
// }
// func TestConfigJSONBad(t *testing.T) {
//
// 	badconfigs := []string{
// 		//`{"Socket":"/tmp/testingZone","Addr":":8008","Name":"Testing Zone!"}`, // good json
// 		`{Socket":"/tmp/testingZone","Addr":":8008","Name":"Testing Zone!"}`,        // bad json
// 		`{"Socket":"/tmp/testingZone","Addr":"8008","Name":"Testing Zone!"}`,        // bad addr
// 		`{"Socket":"/root/test/diamond","Addr":":8008","Name":"Testing Zone!"}`,     // bad socket (no permission)
// 		`{"Socket":"/tmp/testingZone","Addr":"five","Name":"Testing Zone!"}`,        // bad addr
// 		`{"Socket":"/tmp/testingZone","Addr":"example.com","Name":"Testing Zone!"}`, //bad addr
// 		`{"Socket":"/tmp/testingZone","Addr":"localhost","Name":"Testing Zone!"}`,   // bad addr
// 	}
// 	for _, badconfig := range badconfigs {
// 		tmpfile, e := ioutil.TempFile(os.TempDir(), "diamondTest")
//
// 		if e != nil {
// 			fmt.Println(e)
// 			t.FailNow()
// 		}
//
// 		tmpfile.WriteString(badconfig)
// 		_, e = readconf(tmpfile.Name())
// 		os.Remove(tmpfile.Name())
// 		if e != nil {
// 			if s.Config.debug {
// 				fmt.Printf("Testing Config:\n%s\nError: %s\n", badconfig, e)
// 			}
// 		}
//
// 		assert.NotNil(t, e)
// 		fmt.Println(e)
// 		time.Sleep(time.Millisecond * 100)
// 		// assert.Equal(t, "", config.Addr)
// 		// assert.False(t, Config.Debug)
// 		// assert.Equal(t, "", config.name)
// 		// assert.Equal(t, "", config.socket)
// 	}
// }
//
// func TestUpgGitPull(t *testing.T) {
// 	fmt.Println("Git Pull")
// 	str, e := upgGitPull()
// 	if e != nil {
// 		fmt.Println("Error:", e)
// 		t.Fail()
// 	}
// 	if !strings.Contains(str, "Already up-to-date") {
// 		fmt.Println(str)
// 	}
// 	// git pull output should contain "FETCH_HEAD"
// 	if !strings.Contains(str, "FETCH_HEAD") {
// 		t.Fail()
// 	}
// }
//
// func TestUpgMake(t *testing.T) {
// 	fmt.Println("Make")
// 	str, e := upgMake()
// 	if e != nil {
// 		fmt.Println("Error:", e)
// 		t.Fail()
// 	}
// 	// make output should contain "Success"
// 	if !strings.Contains(str, "Success") {
// 		fmt.Println(str)
// 		t.Fail()
// 	}
// }
//
// func TestDummy(t *testing.T) {}
