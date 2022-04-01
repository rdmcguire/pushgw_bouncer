package handlers

type Handler interface {
	Connect() error
	RunCommand(container string, command []string) error
	RestartContainer(container string) error
}
