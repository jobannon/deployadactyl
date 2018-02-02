package interfaces

type DeploymentError interface {
	Error() string
	Details() []string
	Solution() string
}
