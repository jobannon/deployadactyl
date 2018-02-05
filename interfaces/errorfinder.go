package interfaces

type ErrorFinder interface {
	FindErrors(responseString string) []DeploymentError
}
