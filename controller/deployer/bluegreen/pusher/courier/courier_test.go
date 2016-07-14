package courier_test

import (
	. "github.com/compozed/deployadactyl/controller/deployer/bluegreen/pusher/courier"
	"github.com/compozed/deployadactyl/randomizer"
	"github.com/compozed/deployadactyl/test/mocks"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Courier", func() {
	var (
		appName  string
		output   string
		courier  Courier
		executor *mocks.Executor
	)

	BeforeEach(func() {
		appName = "appName-" + randomizer.StringRunes(10)
		output = "output-" + randomizer.StringRunes(10)
		executor = &mocks.Executor{}
		courier = Courier{
			Executor: executor,
		}
	})

	AfterEach(func() {
		Expect(executor.AssertExpectations(GinkgoT())).To(BeTrue())
	})

	Describe("Login", func() {
		It("should get a valid Cloud Foundry login command", func() {
			var (
				api          = "api-" + randomizer.StringRunes(10)
				org          = "org-" + randomizer.StringRunes(10)
				password     = "password-" + randomizer.StringRunes(10)
				space        = "space-" + randomizer.StringRunes(10)
				user         = "user-" + randomizer.StringRunes(10)
				skipSSL      = false
				expectedArgs = []string{"login", "-a", api, "-u", user, "-p", password, "-o", org, "-s", space, ""}
			)

			executor.On("Execute", expectedArgs).Return([]byte(output), nil).Times(1)

			out, err := courier.Login(api, user, password, org, space, skipSSL)
			Expect(err).ToNot(HaveOccurred())

			Expect(string(out)).To(Equal(output))
		})

		It("can skip ssl validation", func() {
			var (
				api          = "api-" + randomizer.StringRunes(10)
				org          = "org-" + randomizer.StringRunes(10)
				password     = "password-" + randomizer.StringRunes(10)
				space        = "space-" + randomizer.StringRunes(10)
				user         = "user-" + randomizer.StringRunes(10)
				skipSSL      = true
				expectedArgs = []string{"login", "-a", api, "-u", user, "-p", password, "-o", org, "-s", space, "--skip-ssl-validation"}
			)

			executor.On("Execute", expectedArgs).Return([]byte(output), nil).Times(1)

			out, err := courier.Login(api, user, password, org, space, skipSSL)
			Expect(err).ToNot(HaveOccurred())

			Expect(string(out)).To(Equal(output))
		})
	})

	Describe("Delete", func() {
		It("should get a valid Cloud Foundry delete command", func() {
			expectedArgs := []string{"delete", appName, "-f"}

			executor.On("Execute", expectedArgs).Return([]byte(output), nil).Times(1)

			out, err := courier.Delete(appName)
			Expect(err).ToNot(HaveOccurred())

			Expect(string(out)).To(Equal(output))
		})
	})

	Describe("Push", func() {
		It("should get a valid Cloud Foundry push command", func() {
			var (
				appLocation  = "appLocation-" + randomizer.StringRunes(10)
				expectedArgs = []string{
					"push", appName,
				}
			)

			executor.On("ExecuteInDirectory", appLocation, expectedArgs).Return([]byte(output), nil).Times(1)

			out, err := courier.Push(appName, appLocation)
			Expect(err).ToNot(HaveOccurred())

			Expect(string(out)).To(Equal(output))
		})
	})

	Describe("Rename", func() {
		It("should get a valid Cloud Foundry rename command", func() {
			var (
				newAppName   = "newAppName-" + randomizer.StringRunes(10)
				expectedArgs = []string{"rename", appName, newAppName}
			)

			executor.On("Execute", expectedArgs).Return([]byte(output), nil).Times(1)

			out, err := courier.Rename(appName, newAppName)
			Expect(err).ToNot(HaveOccurred())

			Expect(string(out)).To(Equal(output))
		})
	})

	Describe("MapRoute", func() {
		It("should get a valid Cloud Foundry map-route command", func() {
			var (
				domain       = "domain-" + randomizer.StringRunes(10)
				expectedArgs = []string{"map-route", appName, domain, "-n", appName}
			)

			executor.On("Execute", expectedArgs).Return([]byte(output), nil).Times(1)

			out, err := courier.MapRoute(appName, domain)
			Expect(err).ToNot(HaveOccurred())

			Expect(string(out)).To(Equal(output))
		})
	})

	Describe("Exists", func() {
		It("should get a valid cloud foundry exists command", func() {
			expectedArgs := []string{"app", appName}

			executor.On("Execute", expectedArgs).Return([]byte(output), nil).Times(1)

			Expect(courier.Exists(appName)).To(BeTrue())
		})
	})

	Describe("CleanUp", func() {
		It("calls CleanUp on the executor", func() {
			executor.On("CleanUp").Return(nil)
			Expect(courier.CleanUp()).To(Succeed())
		})
	})
})
