package utask

type Task interface {
	// Runs the task and waits for it to complete
	Run() error
	// Start the task, but don't wait for it to complete.
	// Can be run in a go-routine if asynchroneous execution is desired.
	Start() error
	// Wait for the task to be completed. Can only be called once.
	Wait() error
}
