package healthchecker

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	I "github.com/compozed/deployadactyl/interfaces"
	S "github.com/compozed/deployadactyl/structs"
)

type Client interface {
	Get(url string) (http.Response, error)
}

// HealthChecker will check an endpoint for a http.StatusOK
type HealthChecker struct {
	// OldURL is the prepend on the foundationURL to replace in order to build the
	// newly pushed application URL.
	// Eg: "api.run.pivotal"
	OldURL string

	// NewUrl is what replaces OldURL in the OnEvent function.
	// Eg: "cfapps"
	NewURL string

	Client  Client
	Courier I.Courier
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

	newFoundationURL := strings.Replace(foundationURL, h.OldURL, h.NewURL, 1)
	domain := regexp.MustCompile(fmt.Sprintf("%s.*", h.NewURL)).FindString(newFoundationURL)

	_, err := h.Courier.MapRoute(tempAppWithUUID, domain, tempAppWithUUID)
	if err != nil {
		return MapRouteError{tempAppWithUUID}
	}

	defer func() { h.Courier.UnmapRoute(tempAppWithUUID, domain, tempAppWithUUID) }()

	newFoundationURL = strings.Replace(newFoundationURL, h.NewURL, fmt.Sprintf("%s.%s", tempAppWithUUID, h.NewURL), 1)

	return h.Check(newFoundationURL, deploymentInfo.HealthCheckEndpoint)
}

// Check takes a url and endpoint. It does an http.Get to get the response
// status and returns an error if it is not http.StatusOK.
func (h HealthChecker) Check(url, endpoint string) error {
	trimmedEndpoint := strings.TrimPrefix(endpoint, "/")

	resp, err := h.Client.Get(fmt.Sprintf("%s/%s", url, trimmedEndpoint))
	if err != nil {
		return ClientError{err}
	}

	if resp.StatusCode != http.StatusOK {
		return HealthCheckError{endpoint}
	}

	return nil
}
