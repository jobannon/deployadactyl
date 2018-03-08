package bluegreen

import (
	I "github.com/compozed/deployadactyl/interfaces"
)

func newStopActor(stopper I.Stopper, foundationURL string) stopActor {
	commands := make(chan stopActorCommand)
	errs := make(chan error)

	go func() {
		for command := range commands {
			errs <- command(stopper, foundationURL)
		}
		close(errs)
	}()

	return stopActor{
		commands: commands,
		errs:     errs,
	}
}

type stopActor struct {
	commands chan<- stopActorCommand
	errs     <-chan error
}

type stopActorCommand func(stopper I.Stopper, foundationURL string) error
