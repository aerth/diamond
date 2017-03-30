package diamond

import (
	"fmt"
	"net"
	"net/rpc"
	"os"
)

// Client to connect to the Server
type Client struct {
	socket     string        // path to socket file
	Server     string        // gets filled in with rpc, optional.
	Name       string        // optional, can be sent to identify the admin
	serveraddr *net.UnixAddr // gets parsed from path in NewClient(path)
}

// NewClient returns a new Client, to connect to the socket at path
func NewClient(socketpath string) *Client {
	c := new(Client)
	c.socket = socketpath
	addr, err := net.ResolveUnixAddr("unix", c.socket)
	if err != nil {
		fmt.Println(err)
		os.Exit(2)
		return nil
	}
	c.serveraddr = addr
	return c
}

// Send a command to the server's unix socket
func (c *Client) Send(cmd string) (reply string, err error) {

	client, err := rpc.Dial("unix", c.serveraddr.String())
	if err != nil {
		return "", err
	}

	rep := new(string)
	err = client.Call("Packet.Command", cmd, rep)
	if err != nil {
		return "", err
	}

	return *rep, nil
}
