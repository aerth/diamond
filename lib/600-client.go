package diamond

import (
	"net"
	"net/rpc"
)

// Client to connect to the Server
type Client struct {
	socket     string        // path to socket file
	Server     string        // gets filled in with rpc, optional.
	Name       string        // optional, can be sent to identify the admin
	serveraddr *net.UnixAddr // gets parsed from path in NewClient(path)
}

// NewClient returns a new Client, to connect to the socket at path
// It returns an error only if the socket can not be resolved
func NewClient(socketpath string) (*Client, error) {
	c := new(Client)
	c.socket = socketpath
	addr, err := net.ResolveUnixAddr("unix", c.socket)
	if err != nil {
		return nil, err
	}
	c.serveraddr = addr
	return c, nil

}

// Send a command to the server's unix socket
func (c *Client) Send(cmd string) (reply string, err error) {
	client, err := rpc.Dial("unix", c.serveraddr.String())
	if err != nil {
		return
	}
	err = client.Call("Packet.Command", cmd, &reply)
	if err != nil {
		return
	}

	return
}
