package structs

import "io"

// DeployEventData has a RequestBody and DeploymentInfo.
type DeployEventData struct {
	Writer         io.Writer
	DeploymentInfo *DeploymentInfo
	RequestBody    io.Reader
}
