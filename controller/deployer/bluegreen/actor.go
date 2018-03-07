package bluegreen

import I "github.com/compozed/deployadactyl/interfaces"

func NewActor(action I.Action, foundationURL string) actor {
	commands := make(chan ActorCommand)
	errs := make(chan error)

	go func() {
		for command := range commands {
			errs <- command(action, foundationURL)
		}
		close(errs)
	}()

	return actor{
		Commands: commands,
		Errs:     errs,
	}
}

type actor struct {
	Commands chan<- ActorCommand
	Errs     <-chan error
}

type ActorCommand func(action I.Action, foundationURL string) error
