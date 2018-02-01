package interfaces

type ErrorFinder interface {
	FindError(matchTo string) error
	FindErrors(responseString string) []DeploymentError
}
