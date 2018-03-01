package interfaces

import (
	"io"
)

// PushEventData has a RequestBody and DeploymentInfo.
type StopEventData struct {
	FoundationURL string
	Context       CFContext
	Courier       interface{}
	Response      io.ReadWriter
}
