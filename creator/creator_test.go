package creator

import (
	"os"

	"github.com/compozed/deployadactyl/config"
	I "github.com/compozed/deployadactyl/interfaces"
	"github.com/compozed/deployadactyl/mocks"
	"github.com/compozed/deployadactyl/state"
	"github.com/compozed/deployadactyl/state/push"
	"github.com/compozed/deployadactyl/state/start"
	"github.com/compozed/deployadactyl/state/stop"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"reflect"
	"runtime"
)

var _ = Describe("Custom creator", func() {

	var path string

	BeforeEach(func() {
		path = os.Getenv("PATH")
		var newpath string
		dir, _ := os.Getwd()
		if runtime.GOOS == "windows" {
			newpath = dir + "\\..\\bin;" + path
		} else {
			newpath = dir + "/../bin:" + path
		}
		os.Setenv("PATH", newpath)
	})

	AfterEach(func() {
		os.Unsetenv("CF_USERNAME")
		os.Unsetenv("CF_PASSWORD")
		os.Setenv("PATH", path)
	})

	It("creates the creator from the provided yaml configuration", func() {

		os.Setenv("CF_USERNAME", "test user")
		os.Setenv("CF_PASSWORD", "test pwd")

		level := "DEBUG"
		configPath := "./testconfig.yml"

		creator, err := Custom(level, configPath, CreatorModuleProvider{})

		Expect(err).ToNot(HaveOccurred())
		Expect(creator.config).ToNot(BeNil())
		Expect(creator.eventManager).ToNot(BeNil())
		Expect(creator.fileSystem).ToNot(BeNil())
		Expect(creator.logger).ToNot(BeNil())
		Expect(creator.writer).ToNot(BeNil())
	})

	It("fails due to lack of required env variables", func() {
		level := "DEBUG"
		configPath := "./testconfig.yml"

		_, err := Custom(level, configPath, CreatorModuleProvider{})

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal("missing environment variables: CF_USERNAME, CF_PASSWORD"))
	})

	Describe("CreatePushController", func() {

		Context("when mock constructor is provided", func() {
			It("should return the mock implementation", func() {
				os.Setenv("CF_USERNAME", "test user")
				os.Setenv("CF_PASSWORD", "test pwd")

				level := "DEBUG"
				configPath := "./testconfig.yml"

				expected := &mocks.PushController{}
				creator, _ := Custom(level, configPath, CreatorModuleProvider{
					NewPushController: func(log I.DeploymentLogger, deployer, silentDeployer I.Deployer, conf config.Config, eventManager I.EventManager, errorFinder I.ErrorFinder, pushManagerFactory I.PushManagerFactory, authResolver I.AuthResolver) I.PushController {
						return expected
					},
				})
				controller := creator.CreatePushController(I.DeploymentLogger{})
				Expect(reflect.TypeOf(controller)).To(Equal(reflect.TypeOf(expected)))
			})
		})

		Context("when mock constructor is not provided", func() {
			It("should return the default implementation", func() {

				os.Setenv("CF_USERNAME", "test user")
				os.Setenv("CF_PASSWORD", "test pwd")

				level := "DEBUG"
				configPath := "./testconfig.yml"

				creator, _ := Custom(level, configPath, CreatorModuleProvider{})
				controller := creator.CreatePushController(I.DeploymentLogger{})
				Expect(reflect.TypeOf(controller)).To(Equal(reflect.TypeOf(&push.PushController{})))
				concrete := controller.(*push.PushController)
				Expect(concrete.Deployer).ToNot(BeNil())
				Expect(concrete.SilentDeployer).ToNot(BeNil())
				Expect(concrete.Log).ToNot(BeNil())
				Expect(concrete.Config).ToNot(BeNil())
				Expect(concrete.EventManager).ToNot(BeNil())
				Expect(concrete.ErrorFinder).ToNot(BeNil())
				Expect(concrete.PushManagerFactory).ToNot(BeNil())
				Expect(concrete.AuthResolver).ToNot(BeNil())

			})
		})

		Describe("CreateAuthResolver", func() {

			Context("when mock constructor is provided", func() {
				It("should return the mock implementation", func() {
					os.Setenv("CF_USERNAME", "test user")
					os.Setenv("CF_PASSWORD", "test pwd")

					level := "DEBUG"
					configPath := "./testconfig.yml"

					expected := &mocks.AuthResolver{}
					creator, _ := Custom(level, configPath, CreatorModuleProvider{
						NewAuthResolver: func(authConfig config.Config) I.AuthResolver {
							return expected
						},
					})
					resolver := creator.CreateAuthResolver()
					Expect(reflect.TypeOf(resolver)).To(Equal(reflect.TypeOf(expected)))
				})
			})

			Context("when mock constructor is not provided", func() {
				It("should return the default implementation", func() {
					os.Setenv("CF_USERNAME", "")
					os.Setenv("CF_PASSWORD", "")

					level := "DEBUG"
					configPath := "./testconfig.yml"

					creator, _ := Custom(level, configPath, CreatorModuleProvider{})
					resolver := creator.CreateAuthResolver()
					Expect(reflect.TypeOf(resolver)).To(Equal(reflect.TypeOf(state.AuthResolver{})))
					concrete := resolver.(state.AuthResolver)
					Expect(concrete.Config).ToNot(BeNil())
				})
			})

		})

	})

	Describe("CreateStartController", func() {

		Context("when mock constructor is provided ", func() {
			It("should return the mock implementation", func() {
				os.Setenv("CF_USERNAME", "test user")
				os.Setenv("CF_PASSWORD", "test pwd")

				level := "DEBUG"
				configPath := "./testconfig.yml"

				expected := &mocks.StartController{}
				creator, _ := Custom(level, configPath, CreatorModuleProvider{
					NewStartController: func(log I.DeploymentLogger, deployer I.Deployer, conf config.Config, eventManager I.EventManager, errorFinder I.ErrorFinder, startmanagerFactory I.StartManagerFactory, authResolver I.AuthResolver) I.StartController {
						return expected
					},
				})
				controller := creator.CreateStartController(I.DeploymentLogger{})
				Expect(reflect.TypeOf(controller)).To(Equal(reflect.TypeOf(expected)))

			})
		})

		Context("when mock constructor is not provided", func() {
			It("should return the default implementation", func() {
				os.Setenv("CF_USERNAME", "test user")
				os.Setenv("CF_PASSWORD", "test pwd")

				level := "DEBUG"
				configPath := "./testconfig.yml"

				creator, _ := Custom(level, configPath, CreatorModuleProvider{})
				controller := creator.CreateStartController(I.DeploymentLogger{})
				Expect(reflect.TypeOf(controller)).To(Equal(reflect.TypeOf(&start.StartController{})))
				concrete := controller.(*start.StartController)
				Expect(concrete.Deployer).ToNot(BeNil())
				Expect(concrete.Config).ToNot(BeNil())
				Expect(concrete.EventManager).ToNot(BeNil())
				Expect(concrete.ErrorFinder).ToNot(BeNil())
				Expect(concrete.StartManagerFactory).ToNot(BeNil())
				Expect(concrete.Log).ToNot(BeNil())
				Expect(concrete.AuthResolver).ToNot(BeNil())
			})
		})
	})

	Describe("CreateStopController", func() {

		Context("when mock constructor is provided", func() {
			It("should return the mock implementation", func() {
				os.Setenv("CF_USERNAME", "test user")
				os.Setenv("CF_PASSWORD", "test pwd")

				level := "DEBUG"
				configPath := "./testconfig.yml"

				expected := &mocks.StopController{}
				creator, _ := Custom(level, configPath, CreatorModuleProvider{
					NewStopController: func(log I.DeploymentLogger, deployer I.Deployer, conf config.Config, eventManager I.EventManager, errorFinder I.ErrorFinder, startManagerFactory I.StartManagerFactory, resolver I.AuthResolver) I.StopController {
						return expected
					},
				})
				controller := creator.CreateStopController(I.DeploymentLogger{})
				Expect(reflect.TypeOf(controller)).To(Equal(reflect.TypeOf(expected)))
			})
		})

		Context("when mock constructor is not provided", func() {
			It("should return the default implementation", func() {
				os.Setenv("CF_USERNAME", "test user")
				os.Setenv("CF_PASSWORD", "test pwd")

				level := "DEBUG"
				configPath := "./testconfig.yml"

				creator, _ := Custom(level, configPath, CreatorModuleProvider{})
				controller := creator.CreateStopController((I.DeploymentLogger{}))

				Expect(reflect.TypeOf(controller)).To(Equal(reflect.TypeOf(&stop.StopController{})))
				concrete := controller.(*stop.StopController)
				Expect(concrete.Deployer).ToNot(BeNil())
				Expect(concrete.Config).ToNot(BeNil())
				Expect(concrete.EventManager).ToNot(BeNil())
				Expect(concrete.ErrorFinder).ToNot(BeNil())
				Expect(concrete.StopManagerFactory).ToNot(BeNil())
				Expect(concrete.Log).ToNot(BeNil())
				Expect(concrete.AuthResolver).ToNot(BeNil())
			})
		})
	})

})
