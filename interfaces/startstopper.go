package interfaces

type StartStopper interface {
	Login(foundationUrl string) error
	Stop(appName, foundationUrl string) error
}
