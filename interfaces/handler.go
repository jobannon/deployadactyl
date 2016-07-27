package interfaces

import S "github.com/compozed/deployadactyl/structs"

type Handler interface {
	OnEvent(event S.Event) error
}
