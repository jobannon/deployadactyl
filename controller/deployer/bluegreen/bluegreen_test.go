package bluegreen_test

import (
	"errors"

	"github.com/compozed/deployadactyl/config"
	. "github.com/compozed/deployadactyl/controller/deployer/bluegreen"
	I "github.com/compozed/deployadactyl/interfaces"
	"github.com/compozed/deployadactyl/logger"
	"github.com/compozed/deployadactyl/mocks"
	"github.com/compozed/deployadactyl/randomizer"
	S "github.com/compozed/deployadactyl/structs"
	"github.com/op/go-logging"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
)

var _ = Describe("Bluegreen", func() {

	var (
		environmentName string
		appName         string
		appPath         string
		org             string
		space           string
		pushOutput      string
		loginOutput     string
		username        string
		password        string
		pusherFactory   *mocks.PusherCreator
		pushers         []*mocks.Pusher
		log             I.Logger
		blueGreen       BlueGreen
		environment     config.Environment
		deploymentInfo  S.DeploymentInfo
		response        *Buffer
		logBuffer       *Buffer
	)

	BeforeEach(func() {
		environmentName = "environmentName-" + randomizer.StringRunes(10)
		appName = "appName-" + randomizer.StringRunes(10)
		appPath = "appPath-" + randomizer.StringRunes(10)
		org = "org-" + randomizer.StringRunes(10)
		space = "space-" + randomizer.StringRunes(10)
		pushOutput = "pushOutput-" + randomizer.StringRunes(10)
		loginOutput = "loginOutput-" + randomizer.StringRunes(10)
		username = "username-" + randomizer.StringRunes(10)
		password = "password-" + randomizer.StringRunes(10)
		response = NewBuffer()
		logBuffer = NewBuffer()

		pusherFactory = &mocks.PusherCreator{}
		pushers = nil

		log = logger.DefaultLogger(logBuffer, logging.DEBUG, "test")

		blueGreen = BlueGreen{PusherCreator: pusherFactory, Log: log}

		environment = config.Environment{Name: environmentName}
		environment.Foundations = []string{randomizer.StringRunes(10), randomizer.StringRunes(10)}

		deploymentInfo = S.DeploymentInfo{
			Username: username,
			Password: password,
			Org:      org,
			Space:    space,
			AppName:  appName,
		}
	})

	Context("when pusher factory fails", func() {
		It("returns an error", func() {
			for i := range environment.Foundations {
				if i == 0 {
					pusherFactory.CreatePusherCall.Returns.Pushers = append(pusherFactory.CreatePusherCall.Returns.Pushers, &mocks.Pusher{})
					pusherFactory.CreatePusherCall.Returns.Pushers = append(pusherFactory.CreatePusherCall.Returns.Pushers, nil)
				} else {
					pusherFactory.CreatePusherCall.Returns.Error = append(pusherFactory.CreatePusherCall.Returns.Error, nil)
					pusherFactory.CreatePusherCall.Returns.Error = append(pusherFactory.CreatePusherCall.Returns.Error, errors.New("push creator failed"))
				}
			}

			err := blueGreen.Push(environment, appPath, deploymentInfo, response)

			Expect(err).To(MatchError("push creator failed"))
		})
	})

	Context("when a login command fails", func() {
		It("should not start to deploy", func() {
			for index := range environment.Foundations {
				pusher := &mocks.Pusher{}
				pushers = append(pushers, pusher)
				pusherFactory.CreatePusherCall.Returns.Pushers = append(pusherFactory.CreatePusherCall.Returns.Pushers, pusher)
				pusherFactory.CreatePusherCall.Returns.Error = append(pusherFactory.CreatePusherCall.Returns.Error, nil)

				if index == 0 {
					By("making the first login command fail")
					pusher.LoginCall.Write.Output = loginOutput
					pusher.LoginCall.Returns.Error = errors.New("bork")
				} else {
					pusher.LoginCall.Write.Output = loginOutput
					pusher.LoginCall.Returns.Error = nil
				}

				pusher.CleanUpCall.Returns.Error = nil
			}

			Expect(blueGreen.Push(environment, appPath, deploymentInfo, response)).ToNot(Succeed())

			for i, pusher := range pushers {
				Expect(pusher.LoginCall.Received.FoundationURL).To(Equal(environment.Foundations[i]))
				Expect(pusher.LoginCall.Received.DeploymentInfo).To(Equal(deploymentInfo))
			}

			Expect(response).To(Say(loginOutput))
			Expect(response).To(Say(loginOutput))
		})
	})

	Context("when all push commands are successful", func() {
		It("can push an app to a single foundation", func() {
			By("setting a single foundation")
			foundationURL := "foundationURL-" + randomizer.StringRunes(10)
			environment.Foundations = []string{foundationURL}

			pusher := &mocks.Pusher{}
			pushers = append(pushers, pusher)
			pusherFactory.CreatePusherCall.Returns.Pushers = append(pusherFactory.CreatePusherCall.Returns.Pushers, pusher)
			pusherFactory.CreatePusherCall.Returns.Error = append(pusherFactory.CreatePusherCall.Returns.Error, nil)

			pusher.LoginCall.Write.Output = loginOutput
			pusher.LoginCall.Returns.Error = nil
			pusher.PushCall.Write.Output = pushOutput
			pusher.PushCall.Returns.Error = nil
			pusher.DeleteVenerableCall.Returns.Error = nil
			pusher.CleanUpCall.Returns.Error = nil

			Expect(blueGreen.Push(environment, appPath, deploymentInfo, response)).To(Succeed())

			Expect(pusher.LoginCall.Received.FoundationURL).To(Equal(foundationURL))
			Expect(pusher.LoginCall.Received.DeploymentInfo).To(Equal(deploymentInfo))
			Expect(pusher.ExistsCall.Received.AppName).To(Equal(deploymentInfo.AppName))
			Expect(pusher.PushCall.Received.AppPath).To(Equal(appPath))
			Expect(pusher.PushCall.Received.DeploymentInfo).To(Equal(deploymentInfo))
			Expect(pusher.DeleteVenerableCall.Received.DeploymentInfo).To(Equal(deploymentInfo))

			Expect(response).To(Say(loginOutput))
			Expect(response).To(Say(pushOutput))
		})

		It("can push an app to multiple foundations", func() {
			By("setting up multiple foundations")
			environment.Foundations = []string{randomizer.StringRunes(10), randomizer.StringRunes(10)}

			for range environment.Foundations {
				pusher := &mocks.Pusher{}
				pushers = append(pushers, pusher)
				pusherFactory.CreatePusherCall.Returns.Pushers = append(pusherFactory.CreatePusherCall.Returns.Pushers, pusher)
				pusherFactory.CreatePusherCall.Returns.Error = append(pusherFactory.CreatePusherCall.Returns.Error, nil)

				pusher.LoginCall.Write.Output = loginOutput
				pusher.LoginCall.Returns.Error = nil
				pusher.PushCall.Write.Output = pushOutput
				pusher.PushCall.Returns.Error = nil
				pusher.DeleteVenerableCall.Returns.Error = nil
				pusher.CleanUpCall.Returns.Error = nil
			}

			Expect(blueGreen.Push(environment, appPath, deploymentInfo, response)).To(Succeed())

			for i, pusher := range pushers {
				foundationURL := environment.Foundations[i]

				Expect(pusher.LoginCall.Received.FoundationURL).To(Equal(foundationURL))
				Expect(pusher.LoginCall.Received.DeploymentInfo).To(Equal(deploymentInfo))
				Expect(pusher.ExistsCall.Received.AppName).To(Equal(deploymentInfo.AppName))
				Expect(pusher.PushCall.Received.AppPath).To(Equal(appPath))
				Expect(pusher.PushCall.Received.DeploymentInfo).To(Equal(deploymentInfo))
				Expect(pusher.DeleteVenerableCall.Received.DeploymentInfo).To(Equal(deploymentInfo))
				Expect(pusher.ExistsCall.Received.AppName).To(Equal(deploymentInfo.AppName))
			}

			Expect(response).To(Say(loginOutput))
			Expect(response).To(Say(pushOutput))
			Expect(response).To(Say(loginOutput))
			Expect(response).To(Say(pushOutput))
		})

		Context("when deleting the venerable fails", func() {
			It("logs an error", func() {
				foundationURL := "foundationURL-" + randomizer.StringRunes(10)
				environment.Foundations = []string{foundationURL}

				pusher := &mocks.Pusher{}
				pushers = append(pushers, pusher)
				pusherFactory.CreatePusherCall.Returns.Pushers = append(pusherFactory.CreatePusherCall.Returns.Pushers, pusher)
				pusherFactory.CreatePusherCall.Returns.Error = append(pusherFactory.CreatePusherCall.Returns.Error, nil)

				pusher.DeleteVenerableCall.Returns.Error = errors.New("delete venerable failed")

				Expect(blueGreen.Push(environment, appPath, deploymentInfo, response)).To(Succeed())

				Eventually(logBuffer).Should(Say("delete venerable failed"))
			})
		})
	})

	Context("when pushing to multiple foundations", func() {
		It("checks if the app exists on each foundation", func() {
			environment.Foundations = []string{randomizer.StringRunes(10), randomizer.StringRunes(10), randomizer.StringRunes(10), randomizer.StringRunes(10)}

			for range environment.Foundations {
				pusher := &mocks.Pusher{}
				pushers = append(pushers, pusher)
				pusherFactory.CreatePusherCall.Returns.Pushers = append(pusherFactory.CreatePusherCall.Returns.Pushers, pusher)
				pusherFactory.CreatePusherCall.Returns.Error = append(pusherFactory.CreatePusherCall.Returns.Error, nil)
			}

			Expect(blueGreen.Push(environment, appPath, deploymentInfo, response)).To(Succeed())

			for i := range environment.Foundations {
				Expect(pushers[i].ExistsCall.Received.AppName).To(Equal(appName))
			}
		})
	})

	Context("when app-venerable already exists on Cloud Foundry", func() {
		It("should delete venerable instances before push", func() {
			for range environment.Foundations {
				pusher := &mocks.Pusher{}
				pushers = append(pushers, pusher)
				pusherFactory.CreatePusherCall.Returns.Pushers = append(pusherFactory.CreatePusherCall.Returns.Pushers, pusher)
				pusherFactory.CreatePusherCall.Returns.Error = append(pusherFactory.CreatePusherCall.Returns.Error, nil)

				pusher.DeleteVenerableCall.Returns.Error = nil
			}

			Expect(blueGreen.Push(environment, appPath, deploymentInfo, response)).To(Succeed())

			for _, pusher := range pushers {
				Expect(pusher.DeleteVenerableCall.Received.DeploymentInfo).To(Equal(deploymentInfo))
				Expect(pusher.ExistsCall.Received.AppName).To(Equal(deploymentInfo.AppName))
			}
		})

		Context("when delete venerable fails", func() {
			It("logs an error", func() {
				for range environment.Foundations {
					pusher := &mocks.Pusher{}
					pushers = append(pushers, pusher)
					pusherFactory.CreatePusherCall.Returns.Pushers = append(pusherFactory.CreatePusherCall.Returns.Pushers, pusher)
					pusherFactory.CreatePusherCall.Returns.Error = append(pusherFactory.CreatePusherCall.Returns.Error, nil)

					pusher.DeleteVenerableCall.Returns.Error = errors.New("delete failed")
				}

				Expect(blueGreen.Push(environment, appPath, deploymentInfo, response)).To(Succeed())

				Eventually(logBuffer).Should(Say("delete failed"))
			})
		})
	})

	Context("when at least one push command is unsuccessful", func() {
		It("should rollback all recent pushes and print Cloud Foundry logs", func() {
			for index := range environment.Foundations {
				pusher := &mocks.Pusher{}
				pushers = append(pushers, pusher)
				pusherFactory.CreatePusherCall.Returns.Pushers = append(pusherFactory.CreatePusherCall.Returns.Pushers, pusher)
				pusherFactory.CreatePusherCall.Returns.Error = append(pusherFactory.CreatePusherCall.Returns.Error, nil)

				pusher.LoginCall.Write.Output = loginOutput
				pusher.LoginCall.Returns.Error = nil

				if index == 0 {
					pusher.PushCall.Write.Output = pushOutput
					pusher.PushCall.Returns.Error = nil
				} else {
					By("making a push command fail")
					pusher.PushCall.Write.Output = pushOutput
					pusher.PushCall.Returns.Error = errors.New("bork")
				}

				pusher.RollbackCall.Returns.Error = nil
				pusher.CleanUpCall.Returns.Error = nil
			}

			err := blueGreen.Push(environment, appPath, deploymentInfo, response)

			Expect(err).To(MatchError(PushFailRollbackError{errors.New("")}))

			for i, pusher := range pushers {
				foundationURL := environment.Foundations[i]

				Expect(pusher.LoginCall.Received.FoundationURL).To(Equal(foundationURL))
				Expect(pusher.LoginCall.Received.DeploymentInfo).To(Equal(deploymentInfo))
				Expect(pusher.ExistsCall.Received.AppName).To(Equal(deploymentInfo.AppName))
				Expect(pusher.PushCall.Received.AppPath).To(Equal(appPath))
				Expect(pusher.PushCall.Received.DeploymentInfo).To(Equal(deploymentInfo))
				Expect(pusher.RollbackCall.Received.DeploymentInfo).To(Equal(deploymentInfo))
			}

			Expect(response).To(Say(loginOutput))
			Expect(response).To(Say(pushOutput))
			Expect(response).To(Say(loginOutput))
			Expect(response).To(Say(pushOutput))
		})

		Context("when rollback fails", func() {
			It("logs an error", func() {
				for index := range environment.Foundations {
					pusher := &mocks.Pusher{}
					pushers = append(pushers, pusher)
					pusherFactory.CreatePusherCall.Returns.Pushers = append(pusherFactory.CreatePusherCall.Returns.Pushers, pusher)
					pusherFactory.CreatePusherCall.Returns.Error = append(pusherFactory.CreatePusherCall.Returns.Error, nil)

					if index == 0 {
						pusher.PushCall.Returns.Error = errors.New("bork")
						pusher.RollbackCall.Returns.Error = errors.New("rollback error")
					}
				}

				err := blueGreen.Push(environment, appPath, deploymentInfo, response)
				Expect(err).To(MatchError(PushFailRollbackError{errors.New("rollback error")}))

				Eventually(logBuffer).Should(Say("rollback error"))
			})
		})

		It("should not rollback any pushes on the first deploy when first deploy rollback is disabled", func() {
			environment.DisableFirstDeployRollback = true

			for index := range environment.Foundations {
				pusher := &mocks.Pusher{}
				pushers = append(pushers, pusher)
				pusherFactory.CreatePusherCall.Returns.Pushers = append(pusherFactory.CreatePusherCall.Returns.Pushers, pusher)
				pusherFactory.CreatePusherCall.Returns.Error = append(pusherFactory.CreatePusherCall.Returns.Error, nil)

				pusher.LoginCall.Write.Output = loginOutput
				pusher.LoginCall.Returns.Error = nil

				if index == 0 {
					pusher.PushCall.Write.Output = pushOutput
					pusher.PushCall.Returns.Error = nil
				} else {
					pusher.PushCall.Write.Output = pushOutput
					pusher.PushCall.Returns.Error = errors.New("bork")
				}

				pusher.RollbackCall.Returns.Error = nil
				pusher.CleanUpCall.Returns.Error = nil
			}

			Expect(blueGreen.Push(environment, appPath, deploymentInfo, response)).To(MatchError(PushFailNoRollbackError{}))

			for i, pusher := range pushers {
				foundationURL := environment.Foundations[i]

				Expect(pusher.LoginCall.Received.FoundationURL).To(Equal(foundationURL))
				Expect(pusher.LoginCall.Received.DeploymentInfo).To(Equal(deploymentInfo))
				Expect(pusher.ExistsCall.Received.AppName).To(Equal(deploymentInfo.AppName))
				Expect(pusher.PushCall.Received.AppPath).To(Equal(appPath))
				Expect(pusher.PushCall.Received.DeploymentInfo).To(Equal(deploymentInfo))
				Expect(pusher.RollbackCall.Received.DeploymentInfo).ToNot(Equal(deploymentInfo))
			}

			Expect(response).To(Say(loginOutput))
			Expect(response).To(Say(pushOutput))
			Expect(response).To(Say(loginOutput))
			Expect(response).To(Say(pushOutput))
		})
	})
})
