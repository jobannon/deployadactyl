package interfaces

// PusherFactory interface.
type PusherFactory interface {
	CreatePusher() (Pusher, error)
}
