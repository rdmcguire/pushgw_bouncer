package handlers

import (
	lxd "github.com/lxc/lxd/client"
	"github.com/lxc/lxd/shared/api"
	"github.com/sirupsen/logrus"
)

// LXDConn implements the handlers.Handler interface
// Provides functions to restart processes and containers
type LXDConn struct {
	Socket string
	Client lxd.InstanceServer
	Log    *logrus.Logger
}

// Connect to LXD socket, return error
func (c *LXDConn) Connect() error {
	var err error
	c.Client, err = lxd.ConnectLXDUnix(c.Socket, nil)
	c.Log.Tracef("LXD Client: %+v", c.Client)
	return err
}

// Runs a command inside the container given an array of strings
// Most common will be []string{"bin/systemctl","restart","someservice"}
func (c *LXDConn) RunCommand(container string, command []string) error {
	c.Log.WithFields(logrus.Fields{"container": container, "command": command}).
		Debug("LXD: Running Restart Command")
	req := api.InstanceExecPost{
		Command:     command,
		Interactive: false,
	}
	// Send the command
	restart, err := c.Client.ExecInstance(container, req, nil)
	if err != nil {
		return err
	}
	// Wait for it to complete
	return restart.Wait()
}

// Restarts the entire container
func (c *LXDConn) RestartContainer(container string) error {
	c.Log.WithFields(logrus.Fields{"container": container}).
		Debug("LXD: Restarting Container")
	var err error
	var restart lxd.Operation
	state := api.InstanceStatePut{
		Action:  "restart",
		Timeout: 10,
		Force:   true,
	}
	restart, err = c.Client.UpdateInstanceState(container, state, "")
	if err != nil {
		return err
	}
	// Wait and return
	return restart.Wait()
}
