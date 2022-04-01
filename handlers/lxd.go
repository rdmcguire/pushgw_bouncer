package handlers

import (
	lxd "github.com/lxc/lxd/client"
	"github.com/lxc/lxd/shared/api"
)

type LXDConn struct {
	Socket string
	client lxd.InstanceServer
}

// Connect to LXD socket, return error
func (c *LXDConn) Connect() error {
	var err error
	c.client, err = lxd.ConnectLXDUnix(c.Socket, nil)
	return err
}

// Perform a systemctl restart weewx using the LXD unix socket
func (c *LXDConn) RestartContainerProc(container, proc string) error {
	req := api.InstanceExecPost{
		Command: []string{
			"/bin/systemctl",
			"restart",
			proc,
		},
		Interactive: false,
	}
	// Send the command
	restart, err := c.client.ExecInstance(container, req, nil)
	if err != nil {
		return err
	}
	// Wait for it to complete
	return restart.Wait()
}
