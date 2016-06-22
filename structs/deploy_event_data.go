package structs

import "io"

type DeployEventData struct {
	Writer         io.Writer
	DeploymentInfo *DeploymentInfo
	RequestBody    io.Reader
}
