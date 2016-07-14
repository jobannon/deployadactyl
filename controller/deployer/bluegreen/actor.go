package bluegreen

import I "github.com/compozed/deployadactyl/interfaces"

func newActor(pusher I.Pusher, foundationURL string) actor {
	commands := make(chan actorCommand)
	errs := make(chan error)

	go func() {
		for command := range commands {
			errs <- command(pusher, foundationURL)
		}
		close(errs)
	}()

	return actor{
		commands: commands,
		errs:     errs,
	}
}

type actor struct {
	commands chan<- actorCommand
	errs     <-chan error
}

type actorCommand func(pusher I.Pusher, foundationURL string) error
