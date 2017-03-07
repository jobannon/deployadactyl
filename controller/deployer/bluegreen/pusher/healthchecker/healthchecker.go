package healthchecker

import (
	"fmt"
	"net/http"
	"strings"
)

func Check(endpoint, serverURL string) error {

	endpoint = strings.TrimPrefix(endpoint, "/")
	resp, err := http.Get(fmt.Sprintf("%s/%s", serverURL, endpoint))
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return HealthCheckError{}
	}

	return nil
}
