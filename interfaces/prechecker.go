package interfaces

import "github.com/compozed/deployadactyl/config"

type Prechecker interface {
	AssertAllFoundationsUp(environment config.Environment) error
}
