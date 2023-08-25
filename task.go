package utask

type Task interface {
	// String() string

	// Control
	Run() error
	Start() error
	Wait() error

	// Get metadata
	// GUID() uuid.UUID
	// Command() string
	// Args() []string
	// WorkingDir() string
	// Status() TaskStatus
	// Env() []string
	// Meta() interface{}

	// Get Status
	// StartedAt() *time.Time
	// FinishedAt() *time.Time
	// ExitCode() int
	// Error() error
	// Executed() bool
	// Output() []TaskOutput
}
