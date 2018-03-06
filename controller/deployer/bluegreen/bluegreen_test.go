package bluegreen_test

import (
	"errors"

	. "github.com/compozed/deployadactyl/controller/deployer/bluegreen"
	I "github.com/compozed/deployadactyl/interfaces"
	"github.com/compozed/deployadactyl/logger"
	"github.com/compozed/deployadactyl/mocks"
	"github.com/compozed/deployadactyl/randomizer"
	S "github.com/compozed/deployadactyl/structs"
	"github.com/op/go-logging"

	"fmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
)

var _ = Describe("Bluegreen", func() {

	var (
		appName        string
		appPath        string
		pushOutput     string
		loginOutput    string
		pusherFactory  *mocks.PusherCreator
		pushers        []*mocks.Pusher
		log            I.Logger
		blueGreen      BlueGreen
		environment    S.Environment
		deploymentInfo S.DeploymentInfo
		response       *Buffer
		logBuffer      *Buffer
		pushError      = errors.New("push error")
		rollbackError  = errors.New("rollback error")
	)

	BeforeEach(func() {
		appName = "appName-" + randomizer.StringRunes(10)
		appPath = "appPath-" + randomizer.StringRunes(10)
		pushOutput = "pushOutput-" + randomizer.StringRunes(10)
		loginOutput = "loginOutput-" + randomizer.StringRunes(10)
		response = NewBuffer()
		logBuffer = NewBuffer()

		log = logger.DefaultLogger(logBuffer, logging.DEBUG, "test")

		environment = S.Environment{Name: randomizer.StringRunes(10)}
		environment.Foundations = []string{randomizer.StringRunes(10), randomizer.StringRunes(10)}
		environment.EnableRollback = true

		deploymentInfo = S.DeploymentInfo{AppName: appName}

		pusherFactory = &mocks.PusherCreator{}

		pushers = nil
		for range environment.Foundations {
			pusher := &mocks.Pusher{Response: response}
			pushers = append(pushers, pusher)
			pusherFactory.CreatePusherCall.Returns.Pushers = append(pusherFactory.CreatePusherCall.Returns.Pushers, pusher)
			pusherFactory.CreatePusherCall.Returns.Error = append(pusherFactory.CreatePusherCall.Returns.Error, nil)
		}

		blueGreen = BlueGreen{PusherCreator: pusherFactory, Log: log}
	})

	Context("when pusher factory fails", func() {
		It("returns an error", func() {
			pusherFactory = &mocks.PusherCreator{}
			blueGreen = BlueGreen{PusherCreator: pusherFactory, Log: log}

			for i := range environment.Foundations {
				pusherFactory.CreatePusherCall.Returns.Pushers = append(pusherFactory.CreatePusherCall.Returns.Pushers, &mocks.Pusher{})

				if i != 0 {
					pusherFactory.CreatePusherCall.Returns.Error = append(pusherFactory.CreatePusherCall.Returns.Error, errors.New("push creator failed"))
				}
			}

			err := blueGreen.Push(environment, appPath, deploymentInfo, response)

			Expect(err).To(MatchError("push creator failed"))
		})
	})

	Context("when a login command is called", func() {
		It("starts a deployment when successful", func() {
			for i, pusher := range pushers {
				pusher.LoginCall.Write.Output = loginOutput

				if i == 0 {
					pusher.LoginCall.Returns.Error = nil
				}
			}

			err := blueGreen.Push(environment, appPath, deploymentInfo, response)
			Expect(err).ToNot(HaveOccurred())

			for i, pusher := range pushers {
				Expect(pusher.LoginCall.Received.FoundationURL).To(Equal(environment.Foundations[i]))
			}

			for range environment.Foundations {
				Eventually(response).Should(Say(loginOutput))
			}
		})

		It("does not start a deployment when failed", func() {
			for i, pusher := range pushers {
				pusher.LoginCall.Write.Output = loginOutput

				if i == 0 {
					pusher.LoginCall.Returns.Error = errors.New(loginOutput)
				}
			}

			err := blueGreen.Push(environment, appPath, deploymentInfo, response)
			Expect(err).To(MatchError(LoginError{[]error{errors.New(loginOutput)}}))

			for i, pusher := range pushers {
				Expect(pusher.LoginCall.Received.FoundationURL).To(Equal(environment.Foundations[i]))
			}

			for range environment.Foundations {
				Eventually(response).Should(Say(loginOutput))
			}
		})
	})

	Context("when all push commands are successful", func() {
		It("can push an app to a single foundation", func() {
			By("setting a single foundation")
			var (
				foundationURL = "foundationURL-" + randomizer.StringRunes(10)
				pusher        = &mocks.Pusher{Response: response}
				pusherFactory = &mocks.PusherCreator{}
			)

			environment.Foundations = []string{foundationURL}

			pushers = nil
			pushers = append(pushers, pusher)

			pusherFactory.CreatePusherCall.Returns.Pushers = append(pusherFactory.CreatePusherCall.Returns.Pushers, pusher)
			pusherFactory.CreatePusherCall.Returns.Error = append(pusherFactory.CreatePusherCall.Returns.Error, nil)

			pusher.LoginCall.Write.Output = loginOutput
			pusher.PushCall.Write.Output = pushOutput

			blueGreen = BlueGreen{PusherCreator: pusherFactory, Log: log}

			Expect(blueGreen.Push(environment, appPath, deploymentInfo, response)).To(Succeed())

			Expect(pusher.LoginCall.Received.FoundationURL).To(Equal(foundationURL))
			Expect(pusher.PushCall.Received.AppPath).To(Equal(appPath))

			Eventually(response).Should(Say(loginOutput))
			Eventually(response).Should(Say(pushOutput))
		})

		It("can push an app to multiple foundations", func() {
			By("setting up multiple foundations")
			environment.Foundations = []string{randomizer.StringRunes(10), randomizer.StringRunes(10)}

			for _, pusher := range pushers {
				pusher.LoginCall.Write.Output = loginOutput
				pusher.PushCall.Write.Output = pushOutput
			}

			Expect(blueGreen.Push(environment, appPath, deploymentInfo, response)).To(Succeed())

			for i, pusher := range pushers {
				Expect(pusher.LoginCall.Received.FoundationURL).To(Equal(environment.Foundations[i]))
				Expect(pusher.PushCall.Received.AppPath).To(Equal(appPath))
			}

			Eventually(response).Should(Say(loginOutput))
			Eventually(response).Should(Say(loginOutput))
			Eventually(response).Should(Say(pushOutput))
			Eventually(response).Should(Say(pushOutput))
		})

		Context("when enable_rollback is false", func() {
			It("can push an app that does not rollback on fail", func() {
				By("setting a single foundation")
				var (
					foundationURL = "foundationURL-" + randomizer.StringRunes(10)
					pusher        = &mocks.Pusher{Response: response}
					pusherFactory = &mocks.PusherCreator{}
				)

				environment.Foundations = []string{foundationURL}

				pushers = nil
				pushers = append(pushers, pusher)

				pusherFactory.CreatePusherCall.Returns.Pushers = append(pusherFactory.CreatePusherCall.Returns.Pushers, pusher)
				pusherFactory.CreatePusherCall.Returns.Error = append(pusherFactory.CreatePusherCall.Returns.Error, nil)

				pusher.LoginCall.Write.Output = loginOutput
				pusher.PushCall.Write.Output = pushOutput

				blueGreen = BlueGreen{PusherCreator: pusherFactory, Log: log}

				Expect(blueGreen.Push(environment, appPath, deploymentInfo, response)).To(Succeed())

				Expect(pusher.LoginCall.Received.FoundationURL).To(Equal(foundationURL))
				Expect(pusher.PushCall.Received.AppPath).To(Equal(appPath))

				Eventually(response).Should(Say(loginOutput))
				Eventually(response).Should(Say(pushOutput))
			})

		})

		Context("when deleting the venerable fails", func() {
			It("logs an error", func() {
				var (
					foundationURL = "foundationURL-" + randomizer.StringRunes(10)
					pusher        = &mocks.Pusher{Response: response}
					pusherFactory = &mocks.PusherCreator{}
				)

				environment.Foundations = []string{foundationURL}
				pushers = nil
				pushers = append(pushers, pusher)

				pusherFactory.CreatePusherCall.Returns.Pushers = append(pusherFactory.CreatePusherCall.Returns.Pushers, pusher)
				pusherFactory.CreatePusherCall.Returns.Error = append(pusherFactory.CreatePusherCall.Returns.Error, nil)

				pusher.FinishPushCall.Returns.Error = errors.New("finish push error")

				blueGreen = BlueGreen{PusherCreator: pusherFactory, Log: log}

				err := blueGreen.Push(environment, appPath, deploymentInfo, response)

				Expect(err).To(MatchError(FinishPushError{[]error{errors.New("finish push error")}}))
			})
		})
	})

	Context("when at least one push command is unsuccessful and EnableRollback is true", func() {
		It("should rollback all recent pushes and print Cloud Foundry logs", func() {

			for i, pusher := range pushers {
				pusher.LoginCall.Write.Output = loginOutput
				pusher.PushCall.Write.Output = pushOutput

				if i != 0 {
					pusher.PushCall.Returns.Error = pushError
				}
			}

			err := blueGreen.Push(environment, appPath, deploymentInfo, response)
			Expect(err).To(MatchError(PushError{[]error{pushError}}))

			for i, pusher := range pushers {
				Expect(pusher.LoginCall.Received.FoundationURL).To(Equal(environment.Foundations[i]))
				Expect(pusher.PushCall.Received.AppPath).To(Equal(appPath))
			}

			Eventually(response).Should(Say(loginOutput))
			Eventually(response).Should(Say(loginOutput))
			Eventually(response).Should(Say(pushOutput))
			Eventually(response).Should(Say(pushOutput))
		})

		Context("when rollback fails", func() {
			It("return an error", func() {
				pushers[0].PushCall.Returns.Error = pushError
				pushers[0].UndoPushCall.Returns.Error = rollbackError

				err := blueGreen.Push(environment, appPath, deploymentInfo, response)

				Expect(err).To(MatchError(RollbackError{[]error{pushError}, []error{rollbackError}}))
			})
		})

		It("should not rollback any pushes on the first deploy", func() {
			for _, pusher := range pushers {
				pusher.LoginCall.Write.Output = loginOutput
				pusher.PushCall.Write.Output = pushOutput
				pusher.PushCall.Returns.Error = pushError
			}

			err := blueGreen.Push(environment, appPath, deploymentInfo, response)
			Expect(err).To(MatchError(PushError{[]error{pushError, pushError}}))

			for i, pusher := range pushers {
				Expect(pusher.LoginCall.Received.FoundationURL).To(Equal(environment.Foundations[i]))
				Expect(pusher.PushCall.Received.AppPath).To(Equal(appPath))
			}

			Eventually(response).Should(Say(loginOutput))
			Eventually(response).Should(Say(loginOutput))
			Eventually(response).Should(Say(pushOutput))
			Eventually(response).Should(Say(pushOutput))
		})
	})

	Context("when at least one push command is unsuccessful and EnableRollback is false", func() {
		It("app is not rolled back to previous version", func() {
			environment.EnableRollback = false

			for _, pusher := range pushers {
				pusher.PushCall.Returns.Error = pushError
			}

			err := blueGreen.Push(environment, appPath, deploymentInfo, response)

			Expect(err).To(HaveOccurred())
			Expect(pushers[0].UndoPushCall.Received.UndoPushWasCalled).To(Equal(false))
		})
	})
	Describe("Stop", func() {
		Context("when called", func() {
			It("creates a stopper for each foundation", func() {
				stopperFactory := &mocks.StopperCreator{}

				for range environment.Foundations {
					stopperFactory.CreateStopperCall.Returns.Stoppers = append(stopperFactory.CreateStopperCall.Returns.Stoppers, &mocks.StartStopper{})
					stopperFactory.CreateStopperCall.Returns.Error = append(stopperFactory.CreateStopperCall.Returns.Error, nil)
				}

				blueGreen = BlueGreen{StopperCreator: stopperFactory}

				err := blueGreen.Stop(environment, deploymentInfo, NewBuffer())
				Expect(err).ToNot(HaveOccurred())

				for i := range environment.Foundations {
					Expect(stopperFactory.CreateStopperCall.Received[i].DeploymentInfo).To(Equal(deploymentInfo))
				}
			})

			It("returns an error when we fail to create a stopper", func() {
				stopperFactory := &mocks.StopperCreator{}
				stopperFactory.CreateStopperCall.Returns.Stoppers = append(stopperFactory.CreateStopperCall.Returns.Stoppers, &mocks.StartStopper{})
				stopperFactory.CreateStopperCall.Returns.Error = append(stopperFactory.CreateStopperCall.Returns.Error, errors.New("stop creator failed"))

				blueGreen = BlueGreen{StopperCreator: stopperFactory, Log: log}
				err := blueGreen.Stop(environment, deploymentInfo, NewBuffer())

				Expect(err).To(MatchError("stop creator failed"))
			})

			It("logs in to all foundations", func() {
				stopperFactory := &mocks.StopperCreator{}

				var stoppers []*mocks.StartStopper
				for i := range environment.Foundations {
					stoppers = append(stoppers, &mocks.StartStopper{})

					stopperFactory.CreateStopperCall.Returns.Stoppers = append(stopperFactory.CreateStopperCall.Returns.Stoppers, stoppers[i])
					stopperFactory.CreateStopperCall.Returns.Error = append(stopperFactory.CreateStopperCall.Returns.Error, nil)
				}

				blueGreen = BlueGreen{StopperCreator: stopperFactory}

				err := blueGreen.Stop(environment, deploymentInfo, NewBuffer())
				Expect(err).ToNot(HaveOccurred())

				for i, foundationUrl := range environment.Foundations {
					Expect(stoppers[i].LoginCall.Received.FoundationURL).To(Equal(foundationUrl))
				}
			})

			It("does not execute Stop when any login fails", func() {
				stopperFactory := &mocks.StopperCreator{}

				var stoppers []*mocks.StartStopper
				for i := range environment.Foundations {
					stoppers = append(stoppers, &mocks.StartStopper{})

					stopperFactory.CreateStopperCall.Returns.Stoppers = append(stopperFactory.CreateStopperCall.Returns.Stoppers, stoppers[i])
					stopperFactory.CreateStopperCall.Returns.Error = append(stopperFactory.CreateStopperCall.Returns.Error, nil)
				}
				stoppers[0].LoginCall.Returns.Error = errors.New("login to stop failed")
				blueGreen = BlueGreen{StopperCreator: stopperFactory}
				err := blueGreen.Stop(environment, deploymentInfo, NewBuffer())

				Expect(err.Error()).To(Equal("login failed: login to stop failed"))
			})

			It("does not execute Stop when multiple logins fail", func() {
				stopperFactory := &mocks.StopperCreator{}

				var stoppers []*mocks.StartStopper
				for i := range environment.Foundations {
					stoppers = append(stoppers, &mocks.StartStopper{})
					stoppers[i].LoginCall.Returns.Error = errors.New(fmt.Sprintf("login %d to stop failed", i))

					stopperFactory.CreateStopperCall.Returns.Stoppers = append(stopperFactory.CreateStopperCall.Returns.Stoppers, stoppers[i])
					stopperFactory.CreateStopperCall.Returns.Error = append(stopperFactory.CreateStopperCall.Returns.Error, nil)
				}

				blueGreen = BlueGreen{StopperCreator: stopperFactory}
				err := blueGreen.Stop(environment, deploymentInfo, NewBuffer())

				Expect(err.Error()).To(Equal("login failed: login 0 to stop failed: login 1 to stop failed"))
			})

			It("calls Stop for each foundation", func() {
				stopperFactory := &mocks.StopperCreator{}

				var stoppers []*mocks.StartStopper
				for i := range environment.Foundations {
					stoppers = append(stoppers, &mocks.StartStopper{})

					stopperFactory.CreateStopperCall.Returns.Stoppers = append(stopperFactory.CreateStopperCall.Returns.Stoppers, stoppers[i])
					stopperFactory.CreateStopperCall.Returns.Error = append(stopperFactory.CreateStopperCall.Returns.Error, nil)
				}

				blueGreen = BlueGreen{StopperCreator: stopperFactory}

				err := blueGreen.Stop(environment, deploymentInfo, NewBuffer())
				Expect(err).ToNot(HaveOccurred())

				for i, foundationUrl := range environment.Foundations {
					Expect(stoppers[i].StopCall.Received.AppName).To(Equal(appName))
					Expect(stoppers[i].StopCall.Received.FoundationURL).To(Equal(foundationUrl))
				}
			})

			It("returns an error if any Stop fails", func() {
				stopperFactory := &mocks.StopperCreator{}

				var stoppers []*mocks.StartStopper
				for i := range environment.Foundations {
					stoppers = append(stoppers, &mocks.StartStopper{})

					stopperFactory.CreateStopperCall.Returns.Stoppers = append(stopperFactory.CreateStopperCall.Returns.Stoppers, stoppers[i])
					stopperFactory.CreateStopperCall.Returns.Error = append(stopperFactory.CreateStopperCall.Returns.Error, nil)
				}
				stoppers[0].StopCall.Returns.Error = errors.New("stop failed")

				blueGreen = BlueGreen{StopperCreator: stopperFactory}

				err := blueGreen.Stop(environment, deploymentInfo, NewBuffer())
				Expect(err).To(MatchError(StopError{[]error{errors.New("stop failed")}}))
			})

			It("returns all errors when multiple Stops fail", func() {
				stopperFactory := &mocks.StopperCreator{}

				var stoppers []*mocks.StartStopper
				for i := range environment.Foundations {
					stoppers = append(stoppers, &mocks.StartStopper{})
					stoppers[i].StopCall.Returns.Error = errors.New("stop failed")

					stopperFactory.CreateStopperCall.Returns.Stoppers = append(stopperFactory.CreateStopperCall.Returns.Stoppers, stoppers[i])
					stopperFactory.CreateStopperCall.Returns.Error = append(stopperFactory.CreateStopperCall.Returns.Error, nil)
				}

				blueGreen = BlueGreen{StopperCreator: stopperFactory}

				err := blueGreen.Stop(environment, deploymentInfo, NewBuffer())
				Expect(err.Error()).To(Equal("stop failed: stop failed: stop failed"))
			})

			It("rolls back all Stops if any Stop fails", func() {
				stopperFactory := &mocks.StopperCreator{}

				var stoppers []*mocks.StartStopper
				for i := range environment.Foundations {
					stoppers = append(stoppers, &mocks.StartStopper{})

					stopperFactory.CreateStopperCall.Returns.Stoppers = append(stopperFactory.CreateStopperCall.Returns.Stoppers, stoppers[i])
					stopperFactory.CreateStopperCall.Returns.Error = append(stopperFactory.CreateStopperCall.Returns.Error, nil)
				}
				stoppers[0].StopCall.Returns.Error = errors.New("stop failed")

				blueGreen = BlueGreen{StopperCreator: stopperFactory}

				err := blueGreen.Stop(environment, deploymentInfo, NewBuffer())
				Expect(err).To(HaveOccurred())

				for i, foundationUrl := range environment.Foundations {
					Expect(stoppers[i].StartCall.Received.AppName).To(Equal(appName))
					Expect(stoppers[i].StartCall.Received.FoundationURL).To(Equal(foundationUrl))
				}
			})

			It("writes responses to output", func() {
				out := NewBuffer()

				stopperFactory := &mocks.StopperCreator{}

				var stoppers []*mocks.StartStopper
				for i := range environment.Foundations {
					stoppers = append(stoppers, &mocks.StartStopper{})

					stopperFactory.CreateStopperCall.Returns.Stoppers = append(stopperFactory.CreateStopperCall.Returns.Stoppers, stoppers[i])
					stopperFactory.CreateStopperCall.Returns.Error = append(stopperFactory.CreateStopperCall.Returns.Error, nil)
				}

				blueGreen = BlueGreen{StopperCreator: stopperFactory}

				err := blueGreen.Stop(environment, deploymentInfo, out)
				Expect(err).ToNot(HaveOccurred())

				Expect(out).Should(Say("- Cloud Foundry Output -"))
				Expect(out).Should(Say("- End Cloud Foundry Output -"))
			})
		})
	})
})
