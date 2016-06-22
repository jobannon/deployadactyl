package interfaces

import S "github.com/compozed/deployadactyl/structs"

type EventManager interface {
	AddHandler(handler Handler, eventType string) error
	Emit(event S.Event) error
}
