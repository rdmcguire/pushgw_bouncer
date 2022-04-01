package handlers

import (
	"github.com/docker/docker/client"
	"github.com/sirupsen/logrus"
)

// DockerConn implements the handlers.Handler interface
// Provides functions to restart processes and containers
type DockerConn struct {
	Socket       string
	Log          *logrus.Logger
	dockerClient *client.Client
}

// Connects to the Docker unix socket
func (c *DockerConn) Connect() error {
	var err error
	c.dockerClient, err = client.NewClientWithOpts(client.FromEnv)
	return err
}

// Runs a command inside the container given an array of strings
// Most common will be []string{"bin/systemctl","restart","someservice"}
func (c *DockerConn) RunCommand(container string, command []string) error {
	var err error
	c.Log.Infof("I would run command %+v in %s", command, container)
	return err
}

// Restarts the specified docker container
func (c *DockerConn) RestartContainer(container string) error {
	var err error
	c.Log.Infof("I would restart %s", container)
	return err
}
