package interfaces

type Event struct {
	Type  string
	Data  interface{}
	Error error
}

// EventManager interface.
type EventManager interface {
	AddHandler(handler Handler, eventType string) error
	Emit(event Event) error
	EmitEvent(event interface{}) error
}
