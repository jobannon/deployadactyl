package pusher_test

import (
	"errors"
	"fmt"
	"math/rand"

	. "github.com/compozed/deployadactyl/controller/deployer/bluegreen/pusher"
	"github.com/compozed/deployadactyl/logger"
	"github.com/compozed/deployadactyl/mocks"
	"github.com/compozed/deployadactyl/randomizer"
	S "github.com/compozed/deployadactyl/structs"
	"github.com/op/go-logging"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
)

var _ = Describe("Pusher", func() {
	var (
		courier       *mocks.Courier
		pusher        Pusher
		healthchecker *mocks.HealthChecker

		foundationURL  string
		username       string
		password       string
		org            string
		space          string
		skipSSL        bool
		domain         string
		appPath        string
		appName        string
		instances      uint16
		randomUUID     string
		randomEndpoint string
		randomURL      string
		deploymentInfo S.DeploymentInfo
		response       *Buffer
		logBuffer      *Buffer
	)

	BeforeEach(func() {
		courier = &mocks.Courier{}
		healthchecker = &mocks.HealthChecker{}

		foundationURL = "foundationURL-" + randomizer.StringRunes(10)
		username = "username-" + randomizer.StringRunes(10)
		password = "password-" + randomizer.StringRunes(10)
		org = "org-" + randomizer.StringRunes(10)
		space = "space-" + randomizer.StringRunes(10)
		domain = "domain-" + randomizer.StringRunes(10)
		appPath = "appPath-" + randomizer.StringRunes(10)
		appName = "appName-" + randomizer.StringRunes(10)
		randomUUID = randomizer.StringRunes(10)
		instances = uint16(rand.Uint32())
		randomEndpoint = "randomEndpoint-" + randomizer.StringRunes(10)
		randomURL = "randomURL-" + randomizer.StringRunes(10)

		response = NewBuffer()
		logBuffer = NewBuffer()

		pusher = Pusher{
			Courier:       courier,
			HealthChecker: healthchecker,
			Log:           logger.DefaultLogger(logBuffer, logging.DEBUG, "pusher_test"),
		}

		deploymentInfo = S.DeploymentInfo{
			Username:            username,
			Password:            password,
			Org:                 org,
			Space:               space,
			AppName:             appName,
			SkipSSL:             skipSSL,
			Instances:           instances,
			Domain:              domain,
			UUID:                randomUUID,
			HealthCheckEndpoint: randomEndpoint,
		}
	})

	Describe("logging in", func() {
		Context("when login succeeds", func() {
			It("gives the correct info to the courier", func() {

				Expect(pusher.Login(foundationURL, deploymentInfo, response)).To(Succeed())

				Expect(courier.LoginCall.Received.FoundationURL).To(Equal(foundationURL))
				Expect(courier.LoginCall.Received.Username).To(Equal(username))
				Expect(courier.LoginCall.Received.Password).To(Equal(password))
				Expect(courier.LoginCall.Received.Org).To(Equal(org))
				Expect(courier.LoginCall.Received.Space).To(Equal(space))
				Expect(courier.LoginCall.Received.SkipSSL).To(Equal(skipSSL))
			})

			It("writes the output of the courier to the response", func() {
				courier.LoginCall.Returns.Output = []byte("login succeeded")

				Expect(pusher.Login(foundationURL, deploymentInfo, response)).To(Succeed())

				Eventually(response).Should(Say("login succeeded"))
			})
		})

		Context("when login fails", func() {
			It("writes the output of the courier to the writer", func() {
				courier.LoginCall.Returns.Output = []byte("login output")
				courier.LoginCall.Returns.Error = errors.New("login error")

				err := pusher.Login(foundationURL, deploymentInfo, response)
				Expect(err).To(MatchError(LoginError{foundationURL, []byte("login output")}))

				Eventually(response).Should(Say("login output"))
			})
		})
	})

	Describe("pushing an app", func() {
		Context("when an app with the same name does not exist", func() {
			It("reports that the app is new", func() {
				Expect(pusher.Push(appPath, deploymentInfo, response)).To(Succeed())

				Eventually(logBuffer).Should(Say("new app detected"))
			})

			It("pushes the new app", func() {
				courier.PushCall.Returns.Output = []byte("push succeeded")

				Expect(pusher.Push(appPath, deploymentInfo, response)).To(Succeed())

				Expect(courier.PushCall.Received.AppName).To(Equal(appName + TemporaryNameSuffix + randomUUID))
				Expect(courier.PushCall.Received.AppPath).To(Equal(appPath))
				Expect(courier.PushCall.Received.Instances).To(Equal(instances))

				Eventually(response).Should(Say("push succeeded"))

				Eventually(logBuffer).Should(Say(fmt.Sprintf("pushing app %s to %s", appName+TemporaryNameSuffix+randomUUID, domain)))
				Eventually(logBuffer).Should(Say(fmt.Sprintf("tempdir for app %s: %s", appName+TemporaryNameSuffix+randomUUID, appPath)))
				Eventually(logBuffer).Should(Say(fmt.Sprintf("push succeeded")))
			})
		})

		Context("when the push fails", func() {
			It("returns an error", func() {
				courier.PushCall.Returns.Error = errors.New("push error")

				err := pusher.Push(appPath, deploymentInfo, response)

				Expect(err).To(MatchError(PushError{}))
			})
		})

		Context("mapping the load balanced route", func() {
			It("maps the route to the app", func() {
				courier.MapRouteCall.Returns.Output = []byte("mapped route")

				Expect(pusher.Push(appPath, deploymentInfo, response)).To(Succeed())

				Expect(courier.MapRouteCall.Received.AppName).To(Equal(appName + TemporaryNameSuffix + randomUUID))
				Expect(courier.MapRouteCall.Received.Domain).To(Equal(domain))

				Eventually(response).Should(Say("mapped route"))

				Eventually(logBuffer).Should(Say(fmt.Sprintf("mapping route for %s to %s", appName, domain)))
			})

			Context("when a domain is not provided", func() {
				It("does not map the domain", func() {
					courier.MapRouteCall.Returns.Output = []byte("mapped route")

					deploymentInfo.Domain = ""

					Expect(pusher.Push(appPath, deploymentInfo, response)).To(Succeed())

					Expect(courier.MapRouteCall.Received.AppName).To(BeEmpty())
					Expect(courier.MapRouteCall.Received.Domain).To(BeEmpty())

					Eventually(response).ShouldNot(Say("mapped route"))

					Eventually(logBuffer).ShouldNot(Say(fmt.Sprintf("mapping route for %s to ", appName)))
				})
			})

			Context("when MapRoute fails", func() {
				It("returns an error", func() {
					courier.MapRouteCall.Returns.Output = []byte("unable to map route")
					courier.MapRouteCall.Returns.Error = errors.New("map route error")

					err := pusher.Push(appPath, deploymentInfo, response)
					Expect(err).To(MatchError(MapRouteError{}))

					Expect(courier.MapRouteCall.Received.AppName).To(Equal(appName + TemporaryNameSuffix + randomUUID))
					Expect(courier.MapRouteCall.Received.Domain).To(Equal(domain))

					Eventually(response).Should(Say("unable to map route"))

					Eventually(logBuffer).Should(Say(fmt.Sprintf("mapping route for %s to %s", appName, domain)))
				})
			})

			Context("when the courier log call fails", func() {
				It("returns an error", func() {
					courier.MapRouteCall.Returns.Error = errors.New("map route failed")
					courier.LogsCall.Returns.Error = errors.New("logs error")

					err := pusher.Push(appPath, deploymentInfo, response)

					Expect(err).To(MatchError(CloudFoundryGetLogsError{errors.New("map route failed"), errors.New("logs error")}))
				})
			})
		})

		Describe("checking the health of an endpoint ", func() {
			Context("when the endpoint returns a http.StatusOK", func() {
				It("does not return an error", func() {

					err := pusher.Push(appPath, deploymentInfo, response)
					Expect(err).ToNot(HaveOccurred())

					Expect(healthchecker.CheckCall.Received.Endpoint).To(Equal(randomEndpoint))
					Expect(healthchecker.CheckCall.Received.URL).To(Equal(fmt.Sprintf("https://%s.%s/%s", appName+TemporaryNameSuffix+randomUUID, domain, randomEndpoint)))

					Eventually(logBuffer).Should(Say("finished health check successfully"))
				})
			})

			Context("when the endpoint does not return a http.StatusOK", func() {
				It("does not return an error", func() {
					healthchecker.CheckCall.Returns.Error = errors.New("health check error")

					err := pusher.Push(appPath, deploymentInfo, response)

					Expect(err).To(MatchError(errors.New("health check error")))
				})

				It("logs the error", func() {
					healthchecker.CheckCall.Returns.Error = errors.New("health check error")

					pusher.Push(appPath, deploymentInfo, response)

					Eventually(logBuffer).Should(Say(fmt.Sprintf("attempting to health check %s with endpoint %s", appName+TemporaryNameSuffix+randomUUID, randomEndpoint)))
				})
			})

			Context("when a healthcheck endpoint is not provided", func() {
				It("return nil and not perform the health check", func() {
					deploymentInfo.HealthCheckEndpoint = ""
					err := pusher.Push(appPath, deploymentInfo, response)
					Expect(err).ToNot(HaveOccurred())

					Expect(healthchecker.CheckCall.Received.Endpoint).To(BeEmpty())
					Expect(healthchecker.CheckCall.Received.URL).To(BeEmpty())

					Eventually(logBuffer).ShouldNot(Say("finished health check successfully"))
				})
			})
		})
	})

	Describe("when a push fails", func() {
		Context("when the app exists", func() {
			BeforeEach(func() {
				courier.ExistsCall.Returns.Bool = true

				pusher.Exists(appName)
			})

			It("deletes the app that was pushed", func() {
				Expect(pusher.UndoPush(deploymentInfo)).To(Succeed())

				Expect(courier.DeleteCall.Received.AppName).To(Equal(appName + TemporaryNameSuffix + randomUUID))

				Eventually(logBuffer).Should(Say(fmt.Sprintf("rolling back deploy of %s", appName)))
				Eventually(logBuffer).Should(Say(fmt.Sprintf("deleted %s", appName)))
			})

			Context("when deleting fails", func() {
				It("returns an error and writes a message to the info log", func() {
					courier.DeleteCall.Returns.Error = errors.New("delete error")

					err := pusher.UndoPush(deploymentInfo)
					Expect(err).ToNot(HaveOccurred())

					Eventually(logBuffer).Should(Say(fmt.Sprintf("unable to delete %s", appName+TemporaryNameSuffix+randomUUID)))
				})
			})
		})

		Context("when the app does not exist", func() {
			It("renames the newly built app to the intended application name", func() {
				Expect(pusher.UndoPush(deploymentInfo)).To(Succeed())

				Expect(courier.RenameCall.Received.AppName).To(Equal(appName + TemporaryNameSuffix + randomUUID))
				Expect(courier.RenameCall.Received.AppNameVenerable).To(Equal(appName))

				Eventually(logBuffer).Should(Say("renamed app from %s to %s", appName+TemporaryNameSuffix+randomUUID, appName))
			})

			Context("when renaming fails", func() {
				It("returns an error and writes a message to the info log", func() {
					courier.RenameCall.Returns.Error = errors.New("rename error")
					courier.RenameCall.Returns.Output = []byte("rename error")

					err := pusher.UndoPush(deploymentInfo)
					Expect(err).To(MatchError(RenameError{appName + TemporaryNameSuffix + randomUUID, []byte("rename error")}))

					Eventually(logBuffer).Should(Say(fmt.Sprintf("unable to rename venerable app %s: %s", appName+TemporaryNameSuffix+randomUUID, "rename error")))
				})
			})
		})
	})

	Describe("completing a deployment", func() {
		It("renames the newly pushed app to the original name", func() {
			Expect(pusher.FinishPush(deploymentInfo)).To(Succeed())

			Expect(courier.RenameCall.Received.AppName).To(Equal(appName + TemporaryNameSuffix + randomUUID))
			Expect(courier.RenameCall.Received.AppNameVenerable).To(Equal(appName))

			Eventually(logBuffer).Should(Say("renamed %s to %s", appName+TemporaryNameSuffix+randomUUID, appName))
		})

		Context("when rename fail", func() {
			It("returns an error", func() {
				courier.RenameCall.Returns.Output = []byte("rename output")
				courier.RenameCall.Returns.Error = errors.New("rename error")

				err := pusher.FinishPush(deploymentInfo)
				Expect(err).To(MatchError(RenameError{appName + TemporaryNameSuffix + randomUUID, []byte("rename output")}))

				Expect(courier.RenameCall.Received.AppName).To(Equal(appName + TemporaryNameSuffix + randomUUID))
				Expect(courier.RenameCall.Received.AppNameVenerable).To(Equal(appName))

				Eventually(logBuffer).Should(Say("could not rename %s to %s", appName+TemporaryNameSuffix+randomUUID, appName))
			})
		})

		Context("when the app exists", func() {
			It("deletes the original application ", func() {
				courier.ExistsCall.Returns.Bool = true

				pusher.Exists(appName)
				Expect(pusher.FinishPush(deploymentInfo)).To(Succeed())

				Expect(courier.UnmapRouteCall.Received.AppName).To(Equal(appName))
				Expect(courier.DeleteCall.Received.AppName).To(Equal(appName))

				Eventually(logBuffer).Should(Say(fmt.Sprintf("deleted %s", appName)))
			})

			Context("when unmapping the route fails", func() {
				It("only logs an error", func() {
					courier.ExistsCall.Returns.Bool = true
					courier.UnmapRouteCall.Returns.Error = errors.New("Unmap Error")

					pusher.Exists(appName)

					err := pusher.FinishPush(deploymentInfo)
					Expect(err).ToNot(HaveOccurred())

					Eventually(logBuffer).Should(Say(fmt.Sprintf("could not unmap %s", appName)))
				})
			})

			Context("when deleting the original app fails", func() {
				It("returns an error", func() {
					courier.ExistsCall.Returns.Bool = true
					courier.DeleteCall.Returns.Output = []byte("delete output")
					courier.DeleteCall.Returns.Error = errors.New("delete error")

					pusher.Exists(appName)

					err := pusher.FinishPush(deploymentInfo)
					Expect(err).To(MatchError(DeleteApplicationError{appName, []byte("delete output")}))

					Eventually(logBuffer).Should(Say(fmt.Sprintf("could not delete %s", appName)))
				})
			})
		})

		Context("when the application does not exist", func() {
			It("does not delete the non-existant original application", func() {
				courier.ExistsCall.Returns.Bool = false

				pusher.Exists(appName)

				err := pusher.FinishPush(deploymentInfo)
				Expect(err).ToNot(HaveOccurred())

				Expect(courier.DeleteCall.Received.AppName).To(BeEmpty())

				Eventually(logBuffer).ShouldNot(Say("delete"))
			})
		})
	})

	Describe("getting CF logs", func() {
		Context("when a push fails", func() {
			It("gets logs from the courier", func() {
				courier.PushCall.Returns.Error = errors.New("push error")
				courier.LogsCall.Returns.Output = []byte("cf logs")

				Expect(pusher.Push(appPath, deploymentInfo, response)).ToNot(Succeed())

				Eventually(response).Should(Say(("cf logs")))
			})

			Context("when the courier log call fails", func() {
				It("returns an error", func() {
					pushErr := errors.New("push error")
					logsErr := errors.New("logs error")

					courier.PushCall.Returns.Error = pushErr
					courier.LogsCall.Returns.Error = logsErr

					err := pusher.Push(appPath, deploymentInfo, response)

					Expect(err).To(MatchError(CloudFoundryGetLogsError{pushErr, logsErr}))
				})
			})
		})
	})

	Describe("cleaning up temporary directories", func() {
		It("is successful", func() {
			courier.CleanUpCall.Returns.Error = nil

			Expect(pusher.CleanUp()).To(Succeed())
		})
	})

	Describe("checking for an existing application", func() {
		It("it is successful", func() {
			courier.ExistsCall.Returns.Bool = true

			pusher.Exists(appName)

			Expect(courier.ExistsCall.Received.AppName).To(Equal(appName))
		})
	})
})
