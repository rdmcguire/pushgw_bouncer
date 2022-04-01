package handlers

import (
	lxd "github.com/lxc/lxd/client"
	"github.com/lxc/lxd/shared/api"
)

// LXDConn implements the handlers.Handler interface
// Provides functions to restart processes and containers
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

// Runs a command inside the container given an array of strings
// Most common will be []string{"bin/systemctl","restart","someservice"}
func (c *LXDConn) RunCommand(container string, command []string) error {
	req := api.InstanceExecPost{
		Command:     command,
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

// Restarts the entire container
func (c *LXDConn) RestartContainer() error {
	var err error
	return err
}
