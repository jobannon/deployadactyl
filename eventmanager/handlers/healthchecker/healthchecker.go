package healthchecker

import (
	"fmt"
	"net/http"
	"strings"

	S "github.com/compozed/deployadactyl/structs"
)

// HealthChecker will check an endpoint for a http.StatusOK
type HealthChecker struct {
	// OldURL is the prepend on the foundationURL to replace in order to build the
	// newly pushed application URL.
	// Eg: "api.run.pivotal"
	OldURL string

	// NewUrl is what replaces OldURL in the OnEvent function.
	// Eg: "cfapps"
	NewURL string
}

// OnEvent is used for the EventManager to do health checking during deployments.
// It will create the new application URL by combining the tempAppWithUUID to the
// domain URL.
func (h HealthChecker) OnEvent(event S.Event) error {

	var (
		tempAppWithUUID = event.Data.(S.PushEventData).TempAppWithUUID
		foundationURL   = event.Data.(S.PushEventData).FoundationURL
		deploymentInfo  = event.Data.(S.PushEventData).DeploymentInfo
	)

	builtURL := strings.Replace(foundationURL, "api", fmt.Sprintf("%s.api", tempAppWithUUID), 1)
	builtURL = strings.Replace(builtURL, h.OldURL, h.NewURL, 1)

	return h.Check(builtURL, deploymentInfo.HealthCheckEndpoint)
}

// Check takes a url and endpoint. It does an http.Get to get the response
// status and returns an error if it is not http.StatusOK.
func (h HealthChecker) Check(url, endpoint string) error {
	trimmedEndpoint := strings.TrimPrefix(endpoint, "/")

	resp, err := http.Get(fmt.Sprintf("%s/%s", url, trimmedEndpoint))
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return HealthCheckError{endpoint}
	}

	return nil
}
