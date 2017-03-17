package healthchecker_test

import (
	"errors"
	"fmt"
	"net/http"

	C "github.com/compozed/deployadactyl/constants"
	. "github.com/compozed/deployadactyl/eventmanager/handlers/healthchecker"
	"github.com/compozed/deployadactyl/logger"
	"github.com/compozed/deployadactyl/mocks"
	"github.com/compozed/deployadactyl/randomizer"
	S "github.com/compozed/deployadactyl/structs"
	logging "github.com/op/go-logging"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
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
		logBuffer     *Buffer
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
			Type: C.PushFinishedEvent,
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

		logBuffer = NewBuffer()

		healthchecker = HealthChecker{
			OldURL:  "api.cf",
			NewURL:  "apps",
			Courier: courier,
			Client:  client,
			Log:     logger.DefaultLogger(logBuffer, logging.DEBUG, "healthchecker_test"),
		}
	})

	Describe("OnEvent", func() {
		Context("the new build application is healthy", func() {
			Context("the endpoint provided is valid", func() {
				It("does not return an error", func() {
					client.GetCall.Returns.Response = http.Response{StatusCode: http.StatusOK}

					err := healthchecker.OnEvent(event)

					Expect(err).ToNot(HaveOccurred())
				})

				It("logs in to the foundation", func() {
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

				It("prints success logs to the console", func() {
					client.GetCall.Returns.Response = http.Response{StatusCode: http.StatusOK}

					healthchecker.OnEvent(event)

					Eventually(logBuffer).Should(Say("starting health check"))
					Eventually(logBuffer).Should(Say("logging in to %s", randomFoundationURL))
					Eventually(logBuffer).Should(Say("logged in to %s", randomFoundationURL))
					Eventually(logBuffer).Should(Say("mapping temporary route %s.%s", randomAppName, randomDomain))
					Eventually(logBuffer).Should(Say("mapped temporary route %s.%s", randomAppName, randomDomain))
					Eventually(logBuffer).Should(Say("checking route https://%s.%s%s", randomAppName, randomDomain, randomEndpoint))
					Eventually(logBuffer).Should(Say("health check successful for https://%s.%s%s", randomAppName, randomDomain, randomEndpoint))
					Eventually(logBuffer).Should(Say("unmapping temporary route %s.%s", randomAppName, randomDomain))
					Eventually(logBuffer).Should(Say("unmapped temporary route %s.%s", randomAppName, randomDomain))
					Eventually(logBuffer).Should(Say("finished health check"))
				})
			})

			Context("the endpoint provided is not valid", func() {
				It("returns an error", func() {
					client.GetCall.Returns.Response = http.Response{StatusCode: http.StatusNotFound}

					err := healthchecker.OnEvent(event)

					Expect(err).To(MatchError(HealthCheckError{randomEndpoint}))
				})

				It("prints the endpoint error to the console", func() {
					client.GetCall.Returns.Response = http.Response{StatusCode: http.StatusNotFound}

					healthchecker.OnEvent(event)

					Eventually(logBuffer).Should(Say("starting health check"))
					Eventually(logBuffer).Should(Say("logging in to %s", randomFoundationURL))
					Eventually(logBuffer).Should(Say("logged in to %s", randomFoundationURL))
					Eventually(logBuffer).Should(Say("mapping temporary route %s.%s", randomAppName, randomDomain))
					Eventually(logBuffer).Should(Say("mapped temporary route %s.%s", randomAppName, randomDomain))
					Eventually(logBuffer).Should(Say("checking route https://%s.%s%s", randomAppName, randomDomain, randomEndpoint))
					Eventually(logBuffer).Should(Say("health check failed for endpoint: %s", randomEndpoint))
				})
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
					Type: C.PushFinishedEvent,
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

		Context("when the healthchecker receives the wrong event type", func() {
			It("returns an error", func() {
				event = S.Event{Type: "wrong.type"}

				err := healthchecker.OnEvent(event)

				Expect(err).To(MatchError(WrongEventTypeError{event.Type}))
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
