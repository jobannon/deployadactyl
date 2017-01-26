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
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("Pusher", func() {
	var (
		courier *mocks.Courier
		pusher  Pusher

		foundationURL    string
		username         string
		password         string
		org              string
		space            string
		skipSSL          bool
		domain           string
		appPath          string
		appName          string
		appNameVenerable string
		instances        uint16
		deploymentInfo   S.DeploymentInfo
		response         *gbytes.Buffer
		logBuffer        *gbytes.Buffer
	)

	BeforeEach(func() {
		courier = &mocks.Courier{}

		foundationURL = "foundationURL-" + randomizer.StringRunes(10)
		username = "username-" + randomizer.StringRunes(10)
		password = "password-" + randomizer.StringRunes(10)
		org = "org-" + randomizer.StringRunes(10)
		space = "space-" + randomizer.StringRunes(10)
		domain = "domain-" + randomizer.StringRunes(10)
		appPath = "appPath-" + randomizer.StringRunes(10)
		appName = "appName-" + randomizer.StringRunes(10)
		appNameVenerable = appName + "-venerable"
		instances = uint16(rand.Uint32())

		response = gbytes.NewBuffer()
		logBuffer = gbytes.NewBuffer()

		pusher = Pusher{
			Courier: courier,
			Log:     logger.DefaultLogger(logBuffer, logging.DEBUG, "extractor_test"),
		}

		deploymentInfo = S.DeploymentInfo{
			Username:  username,
			Password:  password,
			Org:       org,
			Space:     space,
			AppName:   appName,
			SkipSSL:   skipSSL,
			Instances: instances,
			Domain:    domain,
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

				Eventually(response).Should(gbytes.Say("login succeeded"))
			})
		})

		Context("when login fails", func() {
			It("writes the output of the courier to the writer", func() {
				courier.LoginCall.Returns.Output = []byte("login failed")
				courier.LoginCall.Returns.Error = errors.New("bork")

				err := pusher.Login(foundationURL, deploymentInfo, response)
				Expect(err).To(MatchError(LoginError{foundationURL, errors.New("bork")}))

				Eventually(response).Should(gbytes.Say("login failed"))
			})
		})
	})

	Describe("pushing an app", func() {
		Context("when an app with the same name already exists", func() {
			It("renames the existing app", func() {
				courier.ExistsCall.Returns.Bool = true
				courier.RenameCall.Returns.Error = nil

				pusher.Exists(appName)

				Expect(pusher.Push(appPath, deploymentInfo, response)).To(Succeed())

				Expect(courier.RenameCall.Received.AppName).To(Equal(appName))
				Expect(courier.RenameCall.Received.AppNameVenerable).To(Equal(appNameVenerable))

				Eventually(logBuffer).Should(gbytes.Say(fmt.Sprintf("renamed app from %s to %s", appName, appNameVenerable)))
			})

			Context("renaming the existing app fails", func() {
				It("returns an error", func() {
					courier.ExistsCall.Returns.Bool = true
					courier.RenameCall.Returns.Error = errors.New("bork")

					pusher.Exists(appName)

					err := pusher.Push(appPath, deploymentInfo, response)
					Expect(err).To(MatchError(RenameFailError{errors.New("bork")}))

					Expect(courier.RenameCall.Received.AppName).To(Equal(appName))
					Expect(courier.RenameCall.Received.AppNameVenerable).To(Equal(appNameVenerable))
				})
			})
		})

		Context("when no app with the same name exists", func() {
			It("reports that the app is new", func() {
				Expect(pusher.Push(appPath, deploymentInfo, response)).To(Succeed())

				Eventually(logBuffer).Should(gbytes.Say("new app detected"))
			})
		})

		It("pushes the new app", func() {
			courier.PushCall.Returns.Output = []byte("push succeeded")

			Expect(pusher.Push(appPath, deploymentInfo, response)).To(Succeed())

			Expect(courier.PushCall.Received.AppName).To(Equal(appName))
			Expect(courier.PushCall.Received.AppPath).To(Equal(appPath))
			Expect(courier.PushCall.Received.Instances).To(Equal(instances))

			Eventually(response).Should(gbytes.Say("push succeeded"))

			Eventually(logBuffer).Should(gbytes.Say(fmt.Sprintf("pushing app %s to %s", appName, domain)))
			Eventually(logBuffer).Should(gbytes.Say(fmt.Sprintf("tempdir for app %s: %s", appName, appPath)))
			Eventually(logBuffer).Should(gbytes.Say(fmt.Sprintf("push succeeded")))
		})

		It("maps the route to the app", func() {
			courier.MapRouteCall.Returns.Output = []byte("mapped route")
			courier.MapRouteCall.Returns.Error = nil

			Expect(pusher.Push(appPath, deploymentInfo, response)).To(Succeed())

			Expect(courier.MapRouteCall.Received.AppName).To(Equal(appName))
			Expect(courier.MapRouteCall.Received.Domain).To(Equal(domain))

			Eventually(response).Should(gbytes.Say("mapped route"))

			Eventually(logBuffer).Should(gbytes.Say(fmt.Sprintf("mapping route for %s to %s", appName, domain)))
		})

		Context("when the push fails", func() {
			It("returns an error", func() {
				courier.PushCall.Returns.Error = errors.New("push error")

				err := pusher.Push(appPath, deploymentInfo, response)

				Expect(err).To(MatchError("push error"))

			})
		})
	})

	Describe("rolling back a deployment", func() {
		It("deletes the app that was pushed", func() {
			Expect(pusher.Rollback(deploymentInfo)).To(Succeed())

			Expect(courier.DeleteCall.Received.AppName).To(Equal(appName))

			Eventually(logBuffer).Should(gbytes.Say(fmt.Sprintf("rolling back deploy of %s", appName)))
			Eventually(logBuffer).Should(gbytes.Say(fmt.Sprintf("deleted %s", appName)))
		})

		Context("when deleting fails", func() {
			It("returns an error and writes a message to the info log", func() {
				courier.DeleteCall.Returns.Error = errors.New("delete error")

				err := pusher.Rollback(deploymentInfo)
				Expect(err).To(MatchError(DeleteApplicationError{appName, errors.New("delete error")}))

				Eventually(logBuffer).Should(gbytes.Say(fmt.Sprintf("unable to delete %s: %s", deploymentInfo.AppName, "delete error")))
			})
		})

		It("renames the venerable app", func() {
			courier.ExistsCall.Returns.Bool = true

			pusher.Exists(appName)

			Expect(pusher.Rollback(deploymentInfo)).To(Succeed())

			Expect(courier.RenameCall.Received.AppName).To(Equal(appNameVenerable))
			Expect(courier.RenameCall.Received.AppNameVenerable).To(Equal(appName))

			Eventually(logBuffer).Should(gbytes.Say("renamed app from %s to %s", appNameVenerable, appName))
		})

		Context("when renaming fails", func() {
			It("returns an error and writes a message to the info log", func() {
				courier.ExistsCall.Returns.Bool = true
				courier.RenameCall.Returns.Error = errors.New("rename error")
				courier.RenameCall.Returns.Output = []byte("rename error")

				pusher.Exists(appName)

				err := pusher.Rollback(deploymentInfo)
				Expect(err).To(MatchError(RenameApplicationError{appName, []byte("rename error")}))

				Eventually(logBuffer).Should(gbytes.Say(fmt.Sprintf("unable to rename venerable app %s: %s", appNameVenerable, "rename error")))
			})
		})
	})

	Describe("completing a deployment", func() {
		It("deletes venerable", func() {
			courier.DeleteCall.Returns.Error = nil
			courier.ExistsCall.Returns.Bool = true

			Expect(pusher.DeleteVenerable(deploymentInfo)).To(Succeed())

			Expect(courier.DeleteCall.Received.AppName).To(Equal(appNameVenerable))

			Eventually(logBuffer).Should(gbytes.Say(fmt.Sprintf("deleted %s", appNameVenerable)))
		})

		Context("when deleting the venerable fails", func() {
			It("returns an error", func() {
				courier.ExistsCall.Returns.Bool = true

				courier.DeleteCall.Returns.Error = errors.New("delete error")

				Expect(pusher.DeleteVenerable(deploymentInfo)).To(MatchError(DeleteApplicationError{appNameVenerable, errors.New("delete error")}))
			})
		})

		Context("when the application does not exist", func() {
			It("returns nil", func() {
				err := pusher.DeleteVenerable(deploymentInfo)

				Expect(err).ToNot(HaveOccurred())
			})
		})
	})

	Describe("getting CF logs", func() {
		Context("when a push fails", func() {
			It("gets logs from the courier", func() {
				courier.PushCall.Returns.Error = errors.New("push error")
				courier.LogsCall.Returns.Output = []byte("cf logs")

				Expect(pusher.Push(appPath, deploymentInfo, response)).ToNot(Succeed())

				Eventually(response).Should(gbytes.Say(("cf logs")))
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

		Context("when MapRoute returns an error", func() {
			It("handles the error", func() {
				courier.MapRouteCall.Returns.Error = errors.New("map route failed")
				courier.LogsCall.Returns.Output = []byte("cf logs")

				err := pusher.Push(appPath, deploymentInfo, response)

				Expect(err).To(MatchError("map route failed"))

				Eventually(response).Should(gbytes.Say(("cf logs")))
			})

			Context("when the courier log call fails", func() {
				It("returns an error", func() {
					mapRouteErr := errors.New("map route failed")
					logsErr := errors.New("logs error")

					courier.MapRouteCall.Returns.Error = mapRouteErr
					courier.LogsCall.Returns.Error = logsErr

					Expect(pusher.Push(appPath, deploymentInfo, response)).To(MatchError(CloudFoundryGetLogsError{mapRouteErr, logsErr}))
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
