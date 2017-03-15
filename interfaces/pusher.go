package interfaces

// Pusher interface.
type Pusher interface {
	Login(foundationURL string) error
	Push(appPath, foundationURL string) error
	FinishPush() error
	UndoPush() error
	CleanUp() error
	Exists(appName string)
}
