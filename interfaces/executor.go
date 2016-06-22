package interfaces

type Executor interface {
	Execute(args ...string) ([]byte, error)
	ExecuteInDirectory(directory string, args ...string) ([]byte, error)
	CleanUp() error
}
