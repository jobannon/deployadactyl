package bluegreen_test

import (
	"bytes"
	"fmt"
	"io"

	"github.com/compozed/deployadactyl/config"
	. "github.com/compozed/deployadactyl/controller/deployer/bluegreen"
	"github.com/compozed/deployadactyl/logger"
	"github.com/compozed/deployadactyl/randomizer"
	S "github.com/compozed/deployadactyl/structs"
	"github.com/compozed/deployadactyl/test/mocks"
	"github.com/go-errors/errors"
	"github.com/op/go-logging"
	"github.com/stretchr/testify/mock"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Bluegreen", func() {

	var (
		environmentName string
		domainName      string
		appName         string
		appPath         string
		org             string
		space           string
		pushOutput      string
		loginOutput     string
		username        string
		password        string
		pusherFactory   *mocks.PusherFactory
		pushers         []*mocks.Pusher
		log             *logging.Logger
		blueGreen       BlueGreen
		environment     config.Environment
		deploymentInfo  S.DeploymentInfo
		buffer          *bytes.Buffer
	)

	BeforeEach(func() {
		environmentName = "environmentName-" + randomizer.StringRunes(10)
		domainName = "domainName-" + randomizer.StringRunes(10)
		appName = "appName-" + randomizer.StringRunes(10)
		appPath = "appPath-" + randomizer.StringRunes(10)
		org = "org-" + randomizer.StringRunes(10)
		space = "space-" + randomizer.StringRunes(10)
		pushOutput = "pushOutput-" + randomizer.StringRunes(10)
		loginOutput = "loginOutput-" + randomizer.StringRunes(10)
		username = "username-" + randomizer.StringRunes(10)
		password = "password-" + randomizer.StringRunes(10)
		buffer = &bytes.Buffer{}

		pusherFactory = &mocks.PusherFactory{}
		pushers = nil

		log = logger.DefaultLogger(GinkgoWriter, logging.DEBUG, "test")

		blueGreen = BlueGreen{pusherFactory, log}

		environment = config.Environment{
			Name:   environmentName,
			Domain: domainName,
		}

		deploymentInfo = S.DeploymentInfo{
			Username: username,
			Password: password,
			Org:      org,
			Space:    space,
			AppName:  appName,
		}
	})

	AfterEach(func() {
		Expect(pusherFactory.AssertExpectations(GinkgoT())).To(BeTrue())
		for _, pusher := range pushers {
			Expect(pusher.AssertExpectations(GinkgoT())).To(BeTrue())
		}
	})

	Context("when any logins fail", func() {
		It("should not start to deploy", func() {
			environment.Foundations = []string{randomizer.StringRunes(10), randomizer.StringRunes(10)}

			for index, foundationURL := range environment.Foundations {
				pusher := &mocks.Pusher{}
				pushers = append(pushers, pusher)
				pusherFactory.On("CreatePusher").Return(pusher, nil).Times(1)

				if index == 0 {
					pusher.On("Login", foundationURL, deploymentInfo, mock.Anything).
						Run(writeToLoginOut(loginOutput)).Return(errors.New("bork"))
				} else {
					pusher.On("Login", foundationURL, deploymentInfo, mock.Anything).
						Run(writeToLoginOut(loginOutput)).Return(nil)
				}

				pusher.On("CleanUp").Return(nil).Times(1)
			}

			Expect(blueGreen.Push(environment, appPath, deploymentInfo, buffer)).ToNot(Succeed())
			Expect(buffer.String()).To(Equal(loginOutput + loginOutput))
		})
	})

	Context("when pushes are successful", func() {
		It("can push an app to a single foundation", func() {
			foundationURL := "foundationURL-" + randomizer.StringRunes(10)
			environment.Foundations = []string{foundationURL}

			pusher := &mocks.Pusher{}
			pushers = append(pushers, pusher)
			pusherFactory.On("CreatePusher").Return(pusher, nil)

			pusher.On("Login", foundationURL, deploymentInfo, mock.Anything).
				Run(writeToLoginOut(loginOutput)).Return(nil)
			pusher.On("Push", appPath, foundationURL, domainName, deploymentInfo, mock.Anything).
				Run(writeToOut(pushOutput)).Return(nil).Times(1)
			pusher.On("FinishPush", foundationURL, deploymentInfo).Return(nil).Times(1)
			pusher.On("CleanUp").Return(nil).Times(1)

			Expect(blueGreen.Push(environment, appPath, deploymentInfo, buffer)).To(Succeed())
			Expect(buffer.String()).To(Equal(loginOutput + pushOutput))
		})

		It("can push an app to multiple foundations", func() {
			environment.Foundations = []string{randomizer.StringRunes(10), randomizer.StringRunes(10)}

			for _, foundationURL := range environment.Foundations {
				pusher := &mocks.Pusher{}
				pushers = append(pushers, pusher)
				pusherFactory.On("CreatePusher").Return(pusher, nil).Times(1)

				pusher.On("Login", foundationURL, deploymentInfo, mock.Anything).
					Run(writeToLoginOut(loginOutput)).Return(nil)
				pusher.On("Push", appPath, foundationURL, domainName, deploymentInfo, mock.Anything).
					Run(writeToOut(pushOutput)).Return(nil).Times(1)
				pusher.On("FinishPush", foundationURL, deploymentInfo).Return(nil).Times(1)
				pusher.On("CleanUp").Return(nil).Times(1)
			}

			Expect(blueGreen.Push(environment, appPath, deploymentInfo, buffer)).To(Succeed())
			Expect(buffer.String()).To(Equal(loginOutput + pushOutput + loginOutput + pushOutput))
		})
	})

	Context("when at least one push is unsuccessful", func() {
		It("should rollback all recent pushes", func() {
			environment.Foundations = []string{randomizer.StringRunes(10), randomizer.StringRunes(10)}

			for index, foundationURL := range environment.Foundations {
				pusher := &mocks.Pusher{}
				pushers = append(pushers, pusher)
				pusherFactory.On("CreatePusher").Return(pusher, nil).Times(1)

				pusher.On("Login", foundationURL, deploymentInfo, mock.Anything).
					Run(writeToLoginOut(loginOutput)).Return(nil)

				if index == 0 {
					pusher.On("Push", appPath, foundationURL, domainName, deploymentInfo, mock.Anything).
						Run(writeToOut(pushOutput)).Return(nil).Times(1)
				} else {
					pusher.On("Push", appPath, foundationURL, domainName, deploymentInfo, mock.Anything).
						Run(writeToOut(pushOutput)).Return(errors.New("bork")).Times(1)
				}

				pusher.On("Unpush", foundationURL, deploymentInfo).Return(nil).Times(1)
				pusher.On("CleanUp").Return(nil).Times(1)
			}

			Expect(blueGreen.Push(environment, appPath, deploymentInfo, buffer)).ToNot(Succeed())
			Expect(buffer.String()).To(Equal(loginOutput + pushOutput + loginOutput + pushOutput))
		})
	})
})

func writeToOut(str string) func(args mock.Arguments) {
	return func(args mock.Arguments) {
		fmt.Fprint(args.Get(4).(io.Writer), str)
	}
}

func writeToLoginOut(str string) func(args mock.Arguments) {
	return func(args mock.Arguments) {
		fmt.Fprint(args.Get(2).(io.Writer), str)
	}
}
