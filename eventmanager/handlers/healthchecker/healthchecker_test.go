package healthchecker_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	. "github.com/compozed/deployadactyl/eventmanager/handlers/healthchecker"
	"github.com/compozed/deployadactyl/randomizer"
	S "github.com/compozed/deployadactyl/structs"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Healthchecker", func() {

	var (
		healthchecker HealthChecker
		requestURL    string
	)

	BeforeEach(func() {

	})

	Describe("checking the health of an endpoint", func() {
		Context("when the endpoint returns a http.StatusOK", func() {
			It("does not return an error", func() {
				endpoint := "/health"

				testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					requestURL = r.RequestURI
				}))
				defer testServer.Close()

				err := healthchecker.Check(testServer.URL, endpoint)
				Expect(err).ToNot(HaveOccurred())

				Expect(requestURL).To(Equal(endpoint))
			})
		})

		Context("when the endpoint does not return a http.StatusOK ", func() {
			It("returns an error", func() {
				endpoint := "/health"

				testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusBadRequest)
					requestURL = r.RequestURI
				}))
				defer testServer.Close()

				err := healthchecker.Check(testServer.URL, endpoint)
				Expect(err).To(MatchError(HealthCheckError{endpoint}))

				Expect(requestURL).To(Equal(endpoint))
			})
		})
	})

	Describe("format of endpoint parameter", func() {
		Context("when the endpoint does not include a '/'", func() {
			It("adds the leading '/'", func() {
				endpoint := "health"

				testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					requestURL = fmt.Sprintf("http://%s%s", r.Host, r.RequestURI)
				}))
				defer testServer.Close()

				err := healthchecker.Check(testServer.URL, endpoint)
				Expect(err).ToNot(HaveOccurred())

				Expect(requestURL).To(Equal(fmt.Sprintf("%s/%s", testServer.URL, endpoint)))
			})
		})

		Context("when the endpoint does include a '/'", func() {
			It("adds the leading '/'", func() {
				endpoint := "/health"

				testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					requestURL = fmt.Sprintf("http://%s%s", r.Host, r.RequestURI)
				}))
				defer testServer.Close()

				err := healthchecker.Check(testServer.URL, endpoint)
				Expect(err).ToNot(HaveOccurred())

				Expect(requestURL).To(Equal(fmt.Sprintf("%s%s", testServer.URL, endpoint)))
			})
		})
	})

	Describe("event handling", func() {
		Context("the application is healthy", func() {
			It("does not return an error", func() {
				randomAppName := "randomAppName-" + randomizer.StringRunes(10)

				testserver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				}))

				event := S.Event{
					Data: S.PushEventData{
						TempAppWithUUID: randomAppName,
						FoundationURL:   fmt.Sprintf("http://api.%s", strings.TrimPrefix(testserver.URL, "http://")),
						DeploymentInfo: &S.DeploymentInfo{
							HealthCheckEndpoint: "/health",
						},
					},
				}

				healthchecker = HealthChecker{
					OldURL: randomAppName + ".api.",
					NewURL: "",
				}

				err := healthchecker.OnEvent(event)

				Expect(err).ToNot(HaveOccurred())
			})

			It("does not return an error", func() {
				randomAppName := "randomAppName-" + randomizer.StringRunes(10)

				testserver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				}))

				event := S.Event{
					Data: S.PushEventData{
						TempAppWithUUID: randomAppName,
						FoundationURL:   testserver.URL,
						DeploymentInfo: &S.DeploymentInfo{
							HealthCheckEndpoint: "/health",
						},
					},
				}

				By("not providing OldURL and NewURL")
				healthchecker = HealthChecker{
					OldURL: "",
					NewURL: "",
				}

				err := healthchecker.OnEvent(event)

				Expect(err).ToNot(HaveOccurred())
			})
		})
	})
})
