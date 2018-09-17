package diamond

import (
	"fmt"
	"net"
	"net/rpc"
	"os"
	"strings"
)

type Diamond struct {
	name       string
	socketpath string
	rpc        *rpc.Server
}

func New(name string, socketpath string, modules ...interface{}) *Diamond {
	rpcServer := rpc.NewServer()
	for i := range modules {
		if err := rpcServer.RegisterName(name, modules[i]); err != nil {

			println(err.Error())
			continue
		}
	}
	return &Diamond{
		name:       name,
		socketpath: socketpath,
		rpc:        rpcServer,
	}
}

func (d *Diamond) Listen() error {
	return d.listen()
}

func (d *Diamond) ListenFatal() {
	if err := d.listen(); err != nil {
		println(err.Error())
		os.Exit(111)
	}
}

func (d *Diamond) listen() error {
	defer os.Remove(d.socketpath)
	listener, err := net.Listen("unix", d.socketpath)
	if err != nil {
		return fmt.Errorf("diamond: Could not listen on unix domain socket %q: %v", d.socketpath, err)
	}
	for {
		cnn, err := listener.Accept()
		if err != nil {
			println("diamond:", err.Error())
			continue
		}
		go d.handleconnection(cnn)
	}
}

func (d *Diamond) handleconnection(conn net.Conn) {
	if conn != nil {
		println("diamond conn:", conn.LocalAddr().String())
	}
	d.rpc.ServeConn(conn)
	conn.Close()
	println("diamond conn closed")
}

type Client struct {
	S   string // path to socket
	rpc *rpc.Client
}

func NewClient(path string) (*Client, error) {
	client, err := rpc.Dial("unix", path)
	return &Client{path, client}, err
}

func (c *Client) Send(cmd string, args ...string) error {
	reply := ""
	if err := c.rpc.Call(cmd, strings.Join(args, " "), &reply); err != nil {
		return err
	}
	fmt.Println(reply)
	return nil
}

func (c *Client) Call(s string, i, v interface{}) error {
	return c.rpc.Call(s, i, v)
}

func (c *Client) Close() error {
	return c.rpc.Close()
}
