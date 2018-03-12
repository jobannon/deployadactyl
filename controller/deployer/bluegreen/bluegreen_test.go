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
		pusherCreator  *mocks.PusherCreator
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

		pusherCreator = &mocks.PusherCreator{}

		pushers = nil
		for range environment.Foundations {
			pusher := &mocks.Pusher{Response: response}
			pushers = append(pushers, pusher)
			pusherCreator.CreatePusherCall.Returns.Pushers = append(pusherCreator.CreatePusherCall.Returns.Pushers, pusher)
			pusherCreator.CreatePusherCall.Returns.Error = append(pusherCreator.CreatePusherCall.Returns.Error, nil)
		}

		blueGreen = BlueGreen{Log: log}
	})

	Context("when pusher factory fails", func() {
		It("returns an error", func() {
			pusherCreator = &mocks.PusherCreator{}
			blueGreen = BlueGreen{Log: log}

			for i := range environment.Foundations {
				pusherCreator.CreatePusherCall.Returns.Pushers = append(pusherCreator.CreatePusherCall.Returns.Pushers, &mocks.Pusher{})

				if i != 0 {
					pusherCreator.CreatePusherCall.Returns.Error = append(pusherCreator.CreatePusherCall.Returns.Error, errors.New("push creator failed"))
				}
			}

			err := blueGreen.Execute(pusherCreator, environment, appPath, deploymentInfo, response)

			Expect(err).To(MatchError("push creator failed"))
		})
	})

	Context("when a login command is called", func() {
		It("starts a deployment when successful", func() {
			for i, pusher := range pushers {
				pusher.InitiallyCall.Write.Output = loginOutput

				if i == 0 {
					pusher.InitiallyCall.Returns.Error = nil
				}
			}

			err := blueGreen.Execute(pusherCreator, environment, appPath, deploymentInfo, response)
			Expect(err).ToNot(HaveOccurred())

			for range environment.Foundations {
				Eventually(response).Should(Say(loginOutput))
			}
		})

		It("does not start a deployment when failed", func() {
			for i, pusher := range pushers {
				pusher.InitiallyCall.Write.Output = loginOutput

				if i == 0 {
					pusher.InitiallyCall.Returns.Error = errors.New(loginOutput)
				}
			}

			err := blueGreen.Execute(pusherCreator, environment, appPath, deploymentInfo, response)
			Expect(err).To(MatchError(LoginError{[]error{errors.New(loginOutput)}}))

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
				pusherCreator = &mocks.PusherCreator{}
			)

			environment.Foundations = []string{foundationURL}

			pushers = nil
			pushers = append(pushers, pusher)

			pusherCreator.CreatePusherCall.Returns.Pushers = append(pusherCreator.CreatePusherCall.Returns.Pushers, pusher)
			pusherCreator.CreatePusherCall.Returns.Error = append(pusherCreator.CreatePusherCall.Returns.Error, nil)

			pusher.InitiallyCall.Write.Output = loginOutput
			pusher.ExecuteCall.Write.Output = pushOutput

			blueGreen = BlueGreen{Log: log}

			Expect(blueGreen.Execute(pusherCreator, environment, appPath, deploymentInfo, response)).To(Succeed())

			Eventually(response).Should(Say(loginOutput))
			Eventually(response).Should(Say(pushOutput))
		})

		It("can push an app to multiple foundations", func() {
			By("setting up multiple foundations")
			environment.Foundations = []string{randomizer.StringRunes(10), randomizer.StringRunes(10)}

			for _, pusher := range pushers {
				pusher.InitiallyCall.Write.Output = loginOutput
				pusher.ExecuteCall.Write.Output = pushOutput
			}

			Expect(blueGreen.Execute(pusherCreator, environment, appPath, deploymentInfo, response)).To(Succeed())

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
					pusherCreator = &mocks.PusherCreator{}
				)

				environment.Foundations = []string{foundationURL}

				pushers = nil
				pushers = append(pushers, pusher)

				pusherCreator.CreatePusherCall.Returns.Pushers = append(pusherCreator.CreatePusherCall.Returns.Pushers, pusher)
				pusherCreator.CreatePusherCall.Returns.Error = append(pusherCreator.CreatePusherCall.Returns.Error, nil)

				pusher.InitiallyCall.Write.Output = loginOutput
				pusher.ExecuteCall.Write.Output = pushOutput

				blueGreen = BlueGreen{Log: log}

				Expect(blueGreen.Execute(pusherCreator, environment, appPath, deploymentInfo, response)).To(Succeed())

				Eventually(response).Should(Say(loginOutput))
				Eventually(response).Should(Say(pushOutput))
			})

		})

		Context("when deleting the venerable fails", func() {
			It("logs an error", func() {
				var (
					foundationURL = "foundationURL-" + randomizer.StringRunes(10)
					pusher        = &mocks.Pusher{Response: response}
					pusherCreator = &mocks.PusherCreator{}
				)

				environment.Foundations = []string{foundationURL}
				pushers = nil
				pushers = append(pushers, pusher)

				pusherCreator.CreatePusherCall.Returns.Pushers = append(pusherCreator.CreatePusherCall.Returns.Pushers, pusher)
				pusherCreator.CreatePusherCall.Returns.Error = append(pusherCreator.CreatePusherCall.Returns.Error, nil)

				pusher.SuccessCall.Returns.Error = errors.New("finish push error")

				blueGreen = BlueGreen{Log: log}

				err := blueGreen.Execute(pusherCreator, environment, appPath, deploymentInfo, response)

				Expect(err).To(MatchError(FinishPushError{[]error{errors.New("finish push error")}}))
			})
		})
	})

	Context("when at least one push command is unsuccessful", func() {

		Context("EnableRollback is true", func() {
			It("should rollback all recent pushes and print Cloud Foundry logs", func() {

				for i, pusher := range pushers {
					pusher.InitiallyCall.Write.Output = loginOutput
					pusher.ExecuteCall.Write.Output = pushOutput

					if i != 0 {
						pusher.ExecuteCall.Returns.Error = pushError
					}
				}

				err := blueGreen.Execute(pusherCreator, environment, appPath, deploymentInfo, response)
				Expect(err).To(MatchError(PushError{[]error{pushError}}))

				Eventually(response).Should(Say(loginOutput))
				Eventually(response).Should(Say(loginOutput))
				Eventually(response).Should(Say(pushOutput))
				Eventually(response).Should(Say(pushOutput))
			})

			Context("when rollback fails", func() {
				It("return an error", func() {
					pushers[0].ExecuteCall.Returns.Error = pushError
					pushers[0].UndoCall.Returns.Error = rollbackError

					err := blueGreen.Execute(pusherCreator, environment, appPath, deploymentInfo, response)

					Expect(err).To(MatchError(RollbackError{[]error{pushError}, []error{rollbackError}}))
				})
			})

			It("should not rollback any pushes on the first deploy", func() {
				for _, pusher := range pushers {
					pusher.InitiallyCall.Write.Output = loginOutput
					pusher.ExecuteCall.Write.Output = pushOutput
					pusher.ExecuteCall.Returns.Error = pushError
				}

				err := blueGreen.Execute(pusherCreator, environment, appPath, deploymentInfo, response)
				Expect(err).To(MatchError(PushError{[]error{pushError, pushError}}))

				Eventually(response).Should(Say(loginOutput))
				Eventually(response).Should(Say(loginOutput))
				Eventually(response).Should(Say(pushOutput))
				Eventually(response).Should(Say(pushOutput))
			})
		})

		Context("EnableRollback is false", func() {
			It("app is not rolled back to previous version", func() {
				environment.EnableRollback = false

				for _, pusher := range pushers {
					pusher.ExecuteCall.Returns.Error = pushError
				}

				err := blueGreen.Execute(pusherCreator, environment, appPath, deploymentInfo, response)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("push failed: push error: push error"))
			})

			It("returns a FinishPushError if Success fails", func() {
				environment.EnableRollback = false

				for _, pusher := range pushers {
					pusher.ExecuteCall.Returns.Error = errors.New("a push execute error")
				}
				pushers[0].UndoCall.Returns.Error = errors.New("a push success error")
				err := blueGreen.Execute(pusherCreator, environment, appPath, deploymentInfo, response)

				Expect(err.Error()).To(Equal("push failed: a push execute error: a push execute error: rollback failed: a push success error"))
			})
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

				blueGreen = BlueGreen{}

				err := blueGreen.Stop(stopperFactory, environment, "", deploymentInfo, NewBuffer())
				Expect(err).ToNot(HaveOccurred())

				for i := range environment.Foundations {
					Expect(stopperFactory.CreateStopperCall.Received[i].DeploymentInfo).To(Equal(deploymentInfo))
				}
			})

			It("returns an error when we fail to create a stopper", func() {
				stopperFactory := &mocks.StopperCreator{}
				stopperFactory.CreateStopperCall.Returns.Stoppers = append(stopperFactory.CreateStopperCall.Returns.Stoppers, &mocks.StartStopper{})
				stopperFactory.CreateStopperCall.Returns.Error = append(stopperFactory.CreateStopperCall.Returns.Error, errors.New("stop creator failed"))

				blueGreen = BlueGreen{Log: log}
				err := blueGreen.Stop(stopperFactory, environment, "", deploymentInfo, NewBuffer())

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

				blueGreen = BlueGreen{}

				err := blueGreen.Stop(stopperFactory, environment, "", deploymentInfo, NewBuffer())
				Expect(err).ToNot(HaveOccurred())

			})

			It("does not execute Stop when any login fails", func() {
				stopperFactory := &mocks.StopperCreator{}

				var stoppers []*mocks.StartStopper
				for i := range environment.Foundations {
					stoppers = append(stoppers, &mocks.StartStopper{})

					stopperFactory.CreateStopperCall.Returns.Stoppers = append(stopperFactory.CreateStopperCall.Returns.Stoppers, stoppers[i])
					stopperFactory.CreateStopperCall.Returns.Error = append(stopperFactory.CreateStopperCall.Returns.Error, nil)
				}
				stoppers[0].InitiallyCall.Returns.Error = errors.New("login to stop failed")
				blueGreen = BlueGreen{}
				err := blueGreen.Stop(stopperFactory, environment, "", deploymentInfo, NewBuffer())

				Expect(err.Error()).To(Equal("login failed: login to stop failed"))
			})

			It("does not execute Stop when multiple logins fail", func() {
				stopperFactory := &mocks.StopperCreator{}

				var stoppers []*mocks.StartStopper
				for i := range environment.Foundations {
					stoppers = append(stoppers, &mocks.StartStopper{})
					stoppers[i].InitiallyCall.Returns.Error = errors.New(fmt.Sprintf("login %d to stop failed", i))

					stopperFactory.CreateStopperCall.Returns.Stoppers = append(stopperFactory.CreateStopperCall.Returns.Stoppers, stoppers[i])
					stopperFactory.CreateStopperCall.Returns.Error = append(stopperFactory.CreateStopperCall.Returns.Error, nil)
				}

				blueGreen = BlueGreen{}
				err := blueGreen.Stop(stopperFactory, environment, "", deploymentInfo, NewBuffer())

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

				blueGreen = BlueGreen{}

				err := blueGreen.Stop(stopperFactory, environment, "", deploymentInfo, NewBuffer())
				Expect(err).ToNot(HaveOccurred())

			})

			It("returns an error if any Stop fails", func() {
				stopperFactory := &mocks.StopperCreator{}

				var stoppers []*mocks.StartStopper
				for i := range environment.Foundations {
					stoppers = append(stoppers, &mocks.StartStopper{})

					stopperFactory.CreateStopperCall.Returns.Stoppers = append(stopperFactory.CreateStopperCall.Returns.Stoppers, stoppers[i])
					stopperFactory.CreateStopperCall.Returns.Error = append(stopperFactory.CreateStopperCall.Returns.Error, nil)
				}
				stoppers[0].ExecuteCall.Returns.Error = errors.New("stop failed")

				blueGreen = BlueGreen{}

				err := blueGreen.Stop(stopperFactory, environment, "", deploymentInfo, NewBuffer())
				Expect(err).To(MatchError(StopError{[]error{errors.New("stop failed")}}))
			})

			It("returns all errors when multiple Stops fail", func() {
				stopperFactory := &mocks.StopperCreator{}

				var stoppers []*mocks.StartStopper
				for i := range environment.Foundations {
					stoppers = append(stoppers, &mocks.StartStopper{})
					stoppers[i].ExecuteCall.Returns.Error = errors.New("stop failed")

					stopperFactory.CreateStopperCall.Returns.Stoppers = append(stopperFactory.CreateStopperCall.Returns.Stoppers, stoppers[i])
					stopperFactory.CreateStopperCall.Returns.Error = append(stopperFactory.CreateStopperCall.Returns.Error, nil)
				}

				blueGreen = BlueGreen{}

				err := blueGreen.Stop(stopperFactory, environment, "", deploymentInfo, NewBuffer())
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
				stoppers[0].ExecuteCall.Returns.Error = errors.New("an error occurred")

				blueGreen = BlueGreen{}

				err := blueGreen.Stop(stopperFactory, environment, "", deploymentInfo, NewBuffer())
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("stop failed: an error occurred"))
			})

			It("returns an error if attemped roll back fails", func() {
				stopperFactory := &mocks.StopperCreator{}

				var stoppers []*mocks.StartStopper
				for i := range environment.Foundations {
					stoppers = append(stoppers, &mocks.StartStopper{})

					stopperFactory.CreateStopperCall.Returns.Stoppers = append(stopperFactory.CreateStopperCall.Returns.Stoppers, stoppers[i])
					stopperFactory.CreateStopperCall.Returns.Error = append(stopperFactory.CreateStopperCall.Returns.Error, nil)
				}
				stoppers[0].ExecuteCall.Returns.Error = errors.New("an error occurred")
				stoppers[0].UndoCall.Returns.Error = errors.New("an error occurred while attempting undo")
				blueGreen = BlueGreen{}

				err := blueGreen.Stop(stopperFactory, environment, "", deploymentInfo, NewBuffer())
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("stop failed: an error occurred: rollback failed: an error occurred while attempting undo"))
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

				blueGreen = BlueGreen{}

				err := blueGreen.Stop(stopperFactory, environment, "", deploymentInfo, out)
				Expect(err).ToNot(HaveOccurred())

				Expect(out).Should(Say("- Cloud Foundry Output -"))
				Expect(out).Should(Say("- End Cloud Foundry Output -"))
			})
		})
	})
})
