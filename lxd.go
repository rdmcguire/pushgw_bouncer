package main

import (
	lxd "github.com/lxc/lxd/client"
	"github.com/lxc/lxd/shared/api"
	"github.com/sirupsen/logrus"
)

type lxdConn struct {
	socket string
	client lxd.InstanceServer
}

// Connect to LXD socket, return error
func (c *lxdConn) connect() error {
	var err error
	c.client, err = lxd.ConnectLXDUnix(c.socket, nil)
	return err
}

// Perform a systemctl restart weewx using the LXD unix socket
func (c *lxdConn) restartContainerProc(container, proc string) {
	req := api.InstanceExecPost{
		Command: []string{
			"/bin/systemctl",
			"restart",
			proc,
		},
		Interactive: false,
	}
	logFields := logrus.Fields{
		"command":   req,
		"socket":    c.socket,
		"container": container,
	}
	// Send the command
	restart, err := c.client.ExecInstance(container, req, nil)
	if err != nil {
		log.WithFields(logFields).WithField("error", err).
			Error("Failed to send command to Container Process")
	}
	// Wait for it to complete
	err = restart.Wait()
	if err != nil {
		log.WithFields(logFields).WithField("error", err).
			Error("Failed to restart Container Process")
	}
}
