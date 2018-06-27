package creator

import (
	"os"

	"reflect"
	"runtime"

	"github.com/compozed/deployadactyl/config"
	"github.com/compozed/deployadactyl/controller/deployer"
	"github.com/compozed/deployadactyl/controller/deployer/bluegreen"
	"github.com/compozed/deployadactyl/eventmanager/handlers/healthchecker"
	I "github.com/compozed/deployadactyl/interfaces"
	"github.com/compozed/deployadactyl/mocks"
	"github.com/compozed/deployadactyl/state"
	"github.com/compozed/deployadactyl/state/delete"
	"github.com/compozed/deployadactyl/state/push"
	"github.com/compozed/deployadactyl/state/start"
	"github.com/compozed/deployadactyl/state/stop"
	"github.com/compozed/deployadactyl/structs"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
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
					NewPushController: func(log I.DeploymentLogger, deployer, silentDeployer I.Deployer, eventManager I.EventManager, errorFinder I.ErrorFinder, pushManagerFactory I.PushManagerFactory, authResolver I.AuthResolver, resolver I.EnvResolver) I.PushController {
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
				Expect(concrete.EventManager).ToNot(BeNil())
				Expect(concrete.ErrorFinder).ToNot(BeNil())
				Expect(concrete.PushManagerFactory).ToNot(BeNil())
				Expect(concrete.AuthResolver).ToNot(BeNil())
				Expect(concrete.EnvResolver).ToNot(BeNil())

			})
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

	Describe("CreateStartController", func() {

		Context("when mock constructor is provided ", func() {
			It("should return the mock implementation", func() {
				os.Setenv("CF_USERNAME", "test user")
				os.Setenv("CF_PASSWORD", "test pwd")

				level := "DEBUG"
				configPath := "./testconfig.yml"

				expected := &mocks.StartController{}
				creator, _ := Custom(level, configPath, CreatorModuleProvider{
					NewStartController: func(log I.DeploymentLogger, deployer I.Deployer, eventManager I.EventManager, errorFinder I.ErrorFinder, startmanagerFactory I.StartManagerFactory, authResolver I.AuthResolver, envResolver I.EnvResolver) I.StartController {
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
					NewStopController: func(log I.DeploymentLogger, deployer I.Deployer, eventManager I.EventManager, errorFinder I.ErrorFinder, startManagerFactory I.StartManagerFactory, resolver I.AuthResolver, envResolver I.EnvResolver) I.StopController {
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
				Expect(concrete.EventManager).ToNot(BeNil())
				Expect(concrete.ErrorFinder).ToNot(BeNil())
				Expect(concrete.StopManagerFactory).ToNot(BeNil())
				Expect(concrete.Log).ToNot(BeNil())
				Expect(concrete.AuthResolver).ToNot(BeNil())
			})

		})
	})

	Describe("CreateDeleteController", func() {
		Context("when mock constructor is provided", func() {
			It("should return the mock implementation", func() {
				os.Setenv("CF_USERNAME", "test user")
				os.Setenv("CF_PASSWORD", "test pwd")

				level := "DEBUG"
				configPath := "./testconfig.yml"

				expected := &mocks.DeleteController{}
				creator, _ := Custom(level, configPath, CreatorModuleProvider{
					NewDeleteController: func(log I.DeploymentLogger, deployer I.Deployer, eventManager I.EventManager, errorFinder I.ErrorFinder, stopManagerFactory I.StopManagerFactory, resolver I.AuthResolver, envResolver I.EnvResolver) I.DeleteController {
						return expected
					},
				})
				controller := creator.CreateDeleteController(I.DeploymentLogger{})
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
				controller := creator.CreateDeleteController((I.DeploymentLogger{}))

				Expect(reflect.TypeOf(controller)).To(Equal(reflect.TypeOf(&delete.DeleteController{})))
				concrete := controller.(*delete.DeleteController)
				Expect(concrete.Deployer).ToNot(BeNil())
				Expect(concrete.EventManager).ToNot(BeNil())
				Expect(concrete.ErrorFinder).ToNot(BeNil())
				Expect(concrete.DeleteManagerFactory).ToNot(BeNil())
				Expect(concrete.Log).ToNot(BeNil())
				Expect(concrete.AuthResolver).ToNot(BeNil())
			})

		})
	})

	Describe("CreateDeployer", func() {

		Context("when mock constructor is provided", func() {
			It("should return the mock implementation", func() {
				os.Setenv("CF_USERNAME", "test user")
				os.Setenv("CF_PASSWORD", "test pwd")

				level := "DEBUG"
				configPath := "./testconfig.yml"

				expected := &mocks.Deployer{}
				creator, _ := Custom(level, configPath, CreatorModuleProvider{
					NewDeployer: func(config config.Config, blueGreener I.BlueGreener, prechecker I.Prechecker, eventManager I.EventManager, randomizer I.Randomizer, errorFinder I.ErrorFinder, log I.DeploymentLogger) I.Deployer {
						return expected
					},
				})
				controller := creator.createDeployer(I.DeploymentLogger{})
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
				actual := creator.createDeployer(I.DeploymentLogger{})

				Expect(reflect.TypeOf(actual)).To(Equal(reflect.TypeOf(&deployer.Deployer{})))
				concrete := actual.(*deployer.Deployer)
				Expect(concrete.Config).ToNot(BeNil())
				Expect(concrete.BlueGreener).ToNot(BeNil())
				Expect(concrete.Prechecker).ToNot(BeNil())
				Expect(concrete.EventManager).ToNot(BeNil())
				Expect(concrete.Randomizer).ToNot(BeNil())
				Expect(concrete.ErrorFinder).ToNot(BeNil())
				Expect(concrete.Log).ToNot(BeNil())
			})

		})
	})

	Describe("CreatePushManager", func() {

		Context("when mock constructor is provided", func() {
			It("should return the mock implementation", func() {
				os.Setenv("CF_USERNAME", "test user")
				os.Setenv("CF_PASSWORD", "test pwd")

				level := "DEBUG"
				configPath := "./testconfig.yml"

				expected := &mocks.PushManager{}
				creator, _ := Custom(level, configPath, CreatorModuleProvider{
					NewPushManager: func(courierCreator I.CourierCreator, eventManager I.EventManager, log I.DeploymentLogger, fetcher I.Fetcher, deployEventData structs.DeployEventData, fileSystemCleaner push.FileSystemCleaner, cfContext I.CFContext, auth I.Authorization, environment structs.Environment, envVars map[string]string) I.ActionCreator {
						return expected
					},
				})
				controller := creator.PushManager(I.DeploymentLogger{}, structs.DeployEventData{}, I.CFContext{}, I.Authorization{}, structs.Environment{}, make(map[string]string))
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
				actual := creator.PushManager(I.DeploymentLogger{}, structs.DeployEventData{}, I.CFContext{}, I.Authorization{}, structs.Environment{}, make(map[string]string))

				Expect(reflect.TypeOf(actual)).To(Equal(reflect.TypeOf(&push.PushManager{})))
				concrete := actual.(*push.PushManager)
				Expect(concrete.CourierCreator).ToNot(BeNil())
				Expect(concrete.EventManager).ToNot(BeNil())
				Expect(concrete.Logger).ToNot(BeNil())
				Expect(concrete.Fetcher).ToNot(BeNil())
				Expect(concrete.DeployEventData).ToNot(BeNil())
				Expect(concrete.FileSystemCleaner).ToNot(BeNil())
				Expect(concrete.CFContext).ToNot(BeNil())
				Expect(concrete.Auth).ToNot(BeNil())
				Expect(concrete.Environment).ToNot(BeNil())
				Expect(concrete.EnvironmentVariables).ToNot(BeNil())
			})

		})
	})

	Describe("CreateStopManager", func() {

		Context("when mock constructor is provided", func() {
			It("should return the mock implementation", func() {
				os.Setenv("CF_USERNAME", "test user")
				os.Setenv("CF_PASSWORD", "test pwd")

				level := "DEBUG"
				configPath := "./testconfig.yml"

				expected := &mocks.StopManager{}
				creator, _ := Custom(level, configPath, CreatorModuleProvider{
					NewStopManager: func(courierCreator I.CourierCreator, eventManager I.EventManager, log I.DeploymentLogger, deployEventData structs.DeployEventData) I.ActionCreator {
						return expected
					},
				})
				controller := creator.StopManager(I.DeploymentLogger{}, structs.DeployEventData{})
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
				actual := creator.StopManager(I.DeploymentLogger{}, structs.DeployEventData{})

				Expect(reflect.TypeOf(actual)).To(Equal(reflect.TypeOf(&stop.StopManager{})))
				concrete := actual.(*stop.StopManager)
				Expect(concrete.CourierCreator).ToNot(BeNil())
				Expect(concrete.EventManager).ToNot(BeNil())
				Expect(concrete.Logger).ToNot(BeNil())
				Expect(concrete.DeployEventData).ToNot(BeNil())
			})

		})
	})

	Describe("CreateStartManager", func() {

		Context("when mock constructor is provided", func() {
			It("should return the mock implementation", func() {
				os.Setenv("CF_USERNAME", "test user")
				os.Setenv("CF_PASSWORD", "test pwd")

				level := "DEBUG"
				configPath := "./testconfig.yml"

				expected := &mocks.StartManager{}
				creator, _ := Custom(level, configPath, CreatorModuleProvider{
					NewStartManager: func(courierCreator I.CourierCreator, eventManager I.EventManager, logger I.DeploymentLogger, deployEventData structs.DeployEventData) I.ActionCreator {
						return expected
					},
				})
				actual := creator.StartManager(I.DeploymentLogger{}, structs.DeployEventData{})
				Expect(reflect.TypeOf(actual)).To(Equal(reflect.TypeOf(expected)))
			})
		})

		Context("when mock constructor is not provided", func() {
			It("should return the default implementation", func() {
				os.Setenv("CF_USERNAME", "test user")
				os.Setenv("CF_PASSWORD", "test pwd")

				level := "DEBUG"
				configPath := "./testconfig.yml"

				creator, _ := Custom(level, configPath, CreatorModuleProvider{})
				actual := creator.StartManager(I.DeploymentLogger{}, structs.DeployEventData{})

				Expect(reflect.TypeOf(actual)).To(Equal(reflect.TypeOf(&start.StartManager{})))
				concrete := actual.(*start.StartManager)
				Expect(concrete.CourierCreator).ToNot(BeNil())
				Expect(concrete.EventManager).ToNot(BeNil())
				Expect(concrete.Logger).ToNot(BeNil())
				Expect(concrete.DeployEventData).ToNot(BeNil())
			})
		})
	})

	Describe("CreateBlueGreen", func() {
		Context("when mock constructor is provided", func() {
			It("should return the mock implementation", func() {
				os.Setenv("CF_USERNAME", "test user")
				os.Setenv("CF_PASSWORD", "test pwd")

				level := "DEBUG"
				configPath := "./testconfig.yml"

				expected := &mocks.BlueGreener{}
				creator, _ := Custom(level, configPath, CreatorModuleProvider{
					NewBlueGreen: func(log I.DeploymentLogger) I.BlueGreener {
						return expected
					},
				})
				actual := creator.createBlueGreener(I.DeploymentLogger{})
				Expect(reflect.TypeOf(actual)).To(Equal(reflect.TypeOf(expected)))
			})
		})

		Context("when mock constructor is not provided", func() {
			It("should return the default implementation", func() {
				os.Setenv("CF_USERNAME", "test user")
				os.Setenv("CF_PASSWORD", "test pwd")

				level := "DEBUG"
				configPath := "./testconfig.yml"

				creator, _ := Custom(level, configPath, CreatorModuleProvider{})
				actual := creator.createBlueGreener(I.DeploymentLogger{})

				Expect(reflect.TypeOf(actual)).To(Equal(reflect.TypeOf(bluegreen.BlueGreen{})))

				concrete := actual.(bluegreen.BlueGreen)
				Expect(concrete.Log).ToNot(BeNil())
			})
		})
	})

	Describe("CreateHealthChecker", func() {
		Context("when mock constructor is not provided", func() {
			It("should return the default implementation", func() {
				os.Setenv("CF_USERNAME", "test user")
				os.Setenv("CF_PASSWORD", "test pwd")

				level := "DEBUG"
				configPath := "./testconfig.yml"

				creator, _ := Custom(level, configPath, CreatorModuleProvider{})
				actual := creator.CreateHealthChecker()

				Expect(reflect.TypeOf(actual)).To(Equal(reflect.TypeOf(healthchecker.HealthChecker{})))

				Expect(actual.OldURL).To(Equal("api.cf"))
				Expect(actual.NewURL).To(Equal("apps"))
				Expect(actual.SilentDeployURL).ToNot(BeNil())
				Expect(actual.SilentDeployEnvironment).ToNot(BeNil())
				Expect(actual.Client).ToNot(BeNil())
			})
		})
	})

})
