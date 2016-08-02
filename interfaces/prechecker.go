package interfaces

import "github.com/compozed/deployadactyl/config"

// Prechecker interface.
type Prechecker interface {
	AssertAllFoundationsUp(environment config.Environment) error
}
