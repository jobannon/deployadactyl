package interfaces

type PusherFactory interface {
	CreatePusher() (Pusher, error)
}
