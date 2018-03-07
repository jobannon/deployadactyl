package interfaces

type Action interface {
	Initially() error
	Execute() error
	Verify() error
	Success() error
	Undo() error
	Finally() error
}
