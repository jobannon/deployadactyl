package pusher_test

import (
	"errors"

	. "github.com/compozed/deployadactyl/deployer/bluegreen/pusher"
	"github.com/compozed/deployadactyl/logger"
	"github.com/compozed/deployadactyl/randomizer"
	S "github.com/compozed/deployadactyl/structs"
	"github.com/compozed/deployadactyl/test/mocks"
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
		deploymentInfo   S.DeploymentInfo
		responseBuffer   *gbytes.Buffer
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
		responseBuffer = gbytes.NewBuffer()

		logBuffer = gbytes.NewBuffer()
		pusher = Pusher{
			courier,
			logger.DefaultLogger(logBuffer, logging.DEBUG, "extractor_test"),
		}

		deploymentInfo = S.DeploymentInfo{
			Username: username,
			Password: password,
			Org:      org,
			Space:    space,
			AppName:  appName,
			SkipSSL:  skipSSL,
		}
	})

	AfterEach(func() {
		Expect(courier.AssertExpectations(GinkgoT())).To(BeTrue())
	})

	Describe("Login", func() {
		Context("when it succeeds", func() {
			It("writes the output of the courier to the Writer", func() {
				courier.On("Login", foundationURL, username, password, org, space, skipSSL).Return([]byte("login succeeded"), nil).Times(1)
				Expect(pusher.Login(foundationURL, deploymentInfo, responseBuffer)).To(Succeed())
				Eventually(responseBuffer).Should(gbytes.Say("login succeeded"))
			})
		})

		Context("when it fails", func() {
			It("writes the output of the courier to the Writer", func() {
				courier.On("Login", foundationURL, username, password, org, space, skipSSL).Return([]byte("login failed"), errors.New("bork")).Times(1)
				Expect(pusher.Login(foundationURL, deploymentInfo, responseBuffer)).ToNot(Succeed())
				Eventually(responseBuffer).Should(gbytes.Say("login failed"))
			})
		})
	})

	Describe("Push", func() {
		It("renames, pushes, and maps route", func() {
			courier.On("Rename", appName, appNameVenerable).Return(nil, nil).Times(1)
			courier.On("Push", appName, appPath).Return([]byte("push succeeded"), nil).Times(1)
			courier.On("MapRoute", appName, domain).Return([]byte("mapped route"), nil).Times(1)

			Expect(pusher.Push(appPath, foundationURL, domain, deploymentInfo, responseBuffer)).To(Succeed())

			Eventually(responseBuffer).Should(gbytes.Say("push succeeded"))
			Eventually(responseBuffer).Should(gbytes.Say("mapped route"))

			Eventually(logBuffer).Should(gbytes.Say("renaming app from " + appName + " to " + appNameVenerable))
			Eventually(logBuffer).Should(gbytes.Say("pushing new app " + appName + " from " + appPath))
			Eventually(logBuffer).Should(gbytes.Say("push succeeded"))
			Eventually(logBuffer).Should(gbytes.Say("mapping route for " + appName + " to " + domain))
		})

		Context("when renaming", func() {
			It("fails when it's not a new app", func() {
				courier.On("Rename", appName, appNameVenerable).Return([]byte("rename failed"), errors.New("bork")).Times(1)
				courier.On("Exists", appName).Return(true).Times(1)

				Expect(pusher.Push(appPath, foundationURL, domain, deploymentInfo, responseBuffer)).ToNot(Succeed())

				Eventually(logBuffer).Should(gbytes.Say("renaming app from " + appName + " to " + appNameVenerable))
				Eventually(logBuffer).Should(gbytes.Say("rename failed"))
			})

			It("doesn't fail when it's a new app", func() {
				courier.On("Rename", appName, appNameVenerable).Return([]byte("rename failed"), errors.New("bork")).Times(1)
				courier.On("Exists", appName).Return(false).Times(1)
				courier.On("Push", appName, appPath).Return([]byte("push succeeded"), nil).Times(1)
				courier.On("MapRoute", appName, domain).Return([]byte("mapped route"), nil).Times(1)

				Expect(pusher.Push(appPath, foundationURL, domain, deploymentInfo, responseBuffer)).To(Succeed())

				Eventually(responseBuffer).Should(gbytes.Say("push succeeded"))
				Eventually(responseBuffer).Should(gbytes.Say("mapped route"))

				Eventually(logBuffer).Should(gbytes.Say("renaming app from " + appName + " to " + appNameVenerable))
				Eventually(logBuffer).Should(gbytes.Say("new app detected"))
				Eventually(logBuffer).Should(gbytes.Say("pushing new app " + appName + " from " + appPath))
				Eventually(logBuffer).Should(gbytes.Say("push succeeded"))
				Eventually(logBuffer).Should(gbytes.Say("mapping route for " + appName + " to " + domain))
			})
		})
	})

	Describe("Unpush", func() {
		It("logs in, deletes, and renames", func() {
			courier.On("Rename", appNameVenerable, appName).Return(nil, nil).Times(1)
			courier.On("Delete", appName).Return(nil, nil).Times(1)

			Expect(pusher.Unpush(foundationURL, deploymentInfo)).To(Succeed())

			Eventually(logBuffer).Should(gbytes.Say("rolling back deploy of " + appName))
			Eventually(logBuffer).Should(gbytes.Say("deleted " + appName))
			Eventually(logBuffer).Should(gbytes.Say("renamed app from " + appNameVenerable + " to " + appName))
		})
	})

	Describe("FinishPush ", func() {
		It("uh logs in, and deletes venerable", func() {
			courier.On("Delete", appNameVenerable).Return(nil, nil).Times(1)

			Expect(pusher.FinishPush(foundationURL, deploymentInfo)).To(Succeed())

			Eventually(logBuffer).Should(gbytes.Say("deleted " + appNameVenerable))
		})
	})

	Describe("CleanUp", func() {
		It("deletes the temporary directory", func() {
			courier.On("CleanUp").Return(nil)
			Expect(pusher.CleanUp()).To(Succeed())
		})
	})
})
