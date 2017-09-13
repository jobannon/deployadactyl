package structs

// Event has a type and event Data.
type Event struct {
	Type  string
	Data  interface{}
	Error error
}
