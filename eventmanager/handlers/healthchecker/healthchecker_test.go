package healthchecker_test

import (
	"errors"
	"fmt"
	"net/http"

	. "github.com/compozed/deployadactyl/eventmanager/handlers/healthchecker"
	"github.com/compozed/deployadactyl/mocks"
	"github.com/compozed/deployadactyl/randomizer"
	S "github.com/compozed/deployadactyl/structs"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Healthchecker", func() {

	var (
		randomAppName       string
		randomFoundationURL string
		randomDomain        string
		randomEndpoint      string
		randomUsername      string
		randomPassword      string
		randomOrg           string
		randomSpace         string

		event         S.Event
		healthchecker HealthChecker
		client        *mocks.Client
		courier       *mocks.Courier
	)

	BeforeEach(func() {
		randomAppName = "randomAppName-" + randomizer.StringRunes(10)

		s := "random-" + randomizer.StringRunes(10)

		randomFoundationURL = fmt.Sprintf("https://api.cf.%s.com", s)
		randomDomain = fmt.Sprintf("apps.%s.com", s)

		randomEndpoint = "/" + randomizer.StringRunes(10)

		randomUsername = "randomUsername" + randomizer.StringRunes(10)
		randomPassword = "randomPassword" + randomizer.StringRunes(10)
		randomOrg = "randomOrg" + randomizer.StringRunes(10)
		randomSpace = "randomSpace" + randomizer.StringRunes(10)

		event = S.Event{
			Data: S.PushEventData{
				TempAppWithUUID: randomAppName,
				FoundationURL:   randomFoundationURL,
				DeploymentInfo: &S.DeploymentInfo{
					HealthCheckEndpoint: randomEndpoint,
					Username:            randomUsername,
					Password:            randomPassword,
					Org:                 randomOrg,
					Space:               randomSpace,
				},
			},
		}

		client = &mocks.Client{}
		courier = &mocks.Courier{}

		healthchecker = HealthChecker{
			Client:  client,
			Courier: courier,
			OldURL:  "api.cf",
			NewURL:  "apps",
		}
	})

	Describe("OnEvent", func() {
		Context("the new build application is healthy", func() {
			It("does not return an error", func() {
				client.GetCall.Returns.Response = http.Response{StatusCode: http.StatusOK}

				err := healthchecker.OnEvent(event)
				Expect(err).ToNot(HaveOccurred())
			})

			It("logs into the foundation", func() {
				healthchecker.OnEvent(event)

				Expect(courier.LoginCall.Received.FoundationURL).To(Equal(randomFoundationURL))
				Expect(courier.LoginCall.Received.Username).To(Equal(randomUsername))
				Expect(courier.LoginCall.Received.Password).To(Equal(randomPassword))
				Expect(courier.LoginCall.Received.Org).To(Equal(randomOrg))
				Expect(courier.LoginCall.Received.Space).To(Equal(randomSpace))
			})

			It("maps a new temporary route", func() {
				healthchecker.OnEvent(event)

				Expect(courier.MapRouteCall.Received.AppName).To(Equal(randomAppName))
				Expect(courier.MapRouteCall.Received.Domain).To(Equal(randomDomain))
				Expect(courier.MapRouteCall.Received.Hostname).To(Equal(randomAppName))
			})

			It("formats the foundation url", func() {
				healthchecker.OnEvent(event)

				Expect(client.GetCall.Received.URL).To(Equal(fmt.Sprintf("https://%s.%s%s", randomAppName, randomDomain, randomEndpoint)))
			})

			It("unmaps the temporary route", func() {
				healthchecker.OnEvent(event)

				Expect(courier.UnmapRouteCall.Received.AppName).To(Equal(randomAppName))
				Expect(courier.UnmapRouteCall.Received.Domain).To(Equal(randomDomain))
				Expect(courier.UnmapRouteCall.Received.Hostname).To(Equal(randomAppName))
			})
		})

		Context("the new build application is not healthy", func() {
			It("returns an error", func() {
				client.GetCall.Returns.Response = http.Response{StatusCode: http.StatusBadRequest}

				err := healthchecker.OnEvent(event)
				Expect(err).To(MatchError(HealthCheckError{randomEndpoint}))
			})
		})

		Context("when the login fails", func() {
			It("returns an error", func() {
				courier.LoginCall.Returns.Output = []byte("login output")
				courier.LoginCall.Returns.Error = errors.New("login error")

				err := healthchecker.OnEvent(event)

				Expect(err).To(MatchError(LoginError{[]byte("login output")}))
			})
		})

		Context("when mapping the temporary route fails", func() {
			It("returns an error", func() {
				courier.MapRouteCall.Returns.Error = errors.New("map route error")

				err := healthchecker.OnEvent(event)

				Expect(err).To(MatchError(MapRouteError{randomAppName, randomDomain}))
			})
		})

		Context("when the client fails to send the GET", func() {
			It("returns an error", func() {
				client.GetCall.Returns.Error = errors.New("client GET error")

				err := healthchecker.OnEvent(event)
				Expect(err).To(MatchError(ClientError{errors.New("client GET error")}))
			})
		})

		Context("when a health check endpoint is not provided", func() {
			It("returns nil", func() {
				event = S.Event{
					Data: S.PushEventData{
						TempAppWithUUID: randomAppName,
						FoundationURL:   randomFoundationURL,
						DeploymentInfo: &S.DeploymentInfo{
							HealthCheckEndpoint: "",
						},
					},
				}

				err := healthchecker.OnEvent(event)

				Expect(err).To(BeNil())
			})
		})
	})

	Describe("format of endpoint parameter", func() {
		Context("when the endpoint does not include a '/'", func() {
			It("adds the leading '/'", func() {
				endpoint := "health"

				healthchecker.Check(randomFoundationURL, endpoint)

				Expect(client.GetCall.Received.URL).To(Equal(fmt.Sprintf("%s/%s", randomFoundationURL, endpoint)))
			})
		})

		Context("when the endpoint does include a '/'", func() {
			It("does not add the leading '/'", func() {
				endpoint := "/health"

				healthchecker.Check(randomFoundationURL, endpoint)

				Expect(client.GetCall.Received.URL).To(Equal(fmt.Sprintf("%s%s", randomFoundationURL, endpoint)))
			})
		})
	})
})
