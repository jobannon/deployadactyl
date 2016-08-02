package interfaces

import S "github.com/compozed/deployadactyl/structs"

// Handler interface.
type Handler interface {
	OnEvent(event S.Event) error
}
