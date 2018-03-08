package interfaces

type Stopper interface {
	Initially() error
	Execute() error
	Verify() error
	Success() error
	Undo() error
	Finally() error
}
