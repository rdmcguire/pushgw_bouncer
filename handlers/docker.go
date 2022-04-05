package handlers

import (
	"context"
	"io/ioutil"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/sirupsen/logrus"
)

const restartWait = 10
const commandWait = 5

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

	// Configure the command
	config := types.ExecConfig{
		Tty:          false,
		AttachStdout: false,
		Cmd:          command,
	}

	// Create context
	ctx, cancel := context.
		WithTimeout(context.Background(), commandWait*time.Second)
	defer cancel()

	// Prepare the execution
	var r types.IDResponse
	r, err = c.dockerClient.
		ContainerExecCreate(ctx, container, config)

	// Abort now if failed
	if err != nil {
		return err
	}

	// Run the command
	var e types.HijackedResponse
	e, err = c.dockerClient.
		ContainerExecAttach(ctx, r.ID, types.ExecStartCheck{})
	defer e.Close()

	// Log command output at trace level
	o, _ := ioutil.ReadAll(e.Reader)
	c.Log.WithFields(logrus.Fields{
		"container": container,
		"command":   command,
		"output":    string(o),
	}).Trace("Restart command executed")

	return err
}

// Restarts the specified docker container
func (c *DockerConn) RestartContainer(container string) error {
	duration := restartWait * time.Second
	return c.dockerClient.
		ContainerRestart(context.Background(), container, &duration)
}
