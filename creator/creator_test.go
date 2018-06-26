package creator

import (
	"os"

	"reflect"
	"runtime"

	"bytes"

	"io/ioutil"

	"github.com/compozed/deployadactyl/config"
	"github.com/compozed/deployadactyl/eventmanager"
	"github.com/compozed/deployadactyl/eventmanager/handlers/healthchecker"
	I "github.com/compozed/deployadactyl/interfaces"
	"github.com/compozed/deployadactyl/mocks"
	"github.com/compozed/deployadactyl/state"
	"github.com/compozed/deployadactyl/state/push"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/op/go-logging"
	"github.com/pkg/errors"
)

var _ = Describe("Creator", func() {

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

	Describe("New", func() {
		Context("if CLI Checker returns error", func() {
			It("returns an error", func() {
				provider := CreatorModuleProvider{CLIChecker: func() error {
					return errors.New("this is a test error")
				}}

				_, err := New(provider)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("this is a test error"))
			})
		})

		Context("when Config constructor is provided", func() {
			It("should return with the provided Config", func() {
				expectedConfig := config.Config{Port: 943}

				creator, err := New(CreatorModuleProvider{
					NewConfig: func() (config.Config, error) {
						return expectedConfig, nil
					},
				})

				Expect(err).ToNot(HaveOccurred())
				Expect(creator.config).To(Equal(expectedConfig))
			})
		})

		Context("when Config constructor is not provided", func() {
			It("should return with the default Config", func() {
				os.Setenv("CF_USERNAME", "myusername")
				os.Setenv("CF_PASSWORD", "mypassword")

				config := `---
environments:
 - name: my-env
   foundations:
    - https://my/foundation
error_matchers:
 - description: a description`

				ioutil.WriteFile("./config.yml", []byte(config), 0777)

				creator, err := New(CreatorModuleProvider{})

				Expect(err).ToNot(HaveOccurred())
				Expect(creator.config.Environments["my-env"].Name).To(Equal("my-env"))
			})

		})

		Context("When config creation fails", func() {
			It("should return an error", func() {
				expectedError := errors.New("a test error")
				_, err := New(CreatorModuleProvider{
					NewConfig: func() (config.Config, error) {
						return config.Config{}, expectedError
					},
				})

				Expect(err).To(HaveOccurred())
				Expect(err).To(Equal(expectedError))
			})
		})

		Context("When Logger creation fails", func() {
			It("should return an error", func() {
				expectedError := errors.New("a test error")
				_, err := New(CreatorModuleProvider{
					NewConfig: func() (config.Config, error) {
						return config.Config{}, nil
					},
					NewLogger: func() (I.Logger, error) {
						return nil, expectedError
					},
				})

				Expect(err).To(HaveOccurred())
				Expect(err).To(Equal(expectedError))
			})
		})

		Context("when Logger constructor is provided", func() {
			It("should return with the provided Logger", func() {
				expectedLog, _ := NewLogger()

				creator, err := New(CreatorModuleProvider{
					NewConfig: func() (config.Config, error) {
						return config.Config{}, nil
					},
					NewLogger: func() (I.Logger, error) {
						return expectedLog, nil
					},
				})

				Expect(err).ToNot(HaveOccurred())
				Expect(creator.logger).To(Equal(expectedLog))
			})
		})

		Context("when logger constructor is not provided", func() {
			It("should return the default logger", func() {
				creator, err := New(CreatorModuleProvider{
					NewConfig: func() (config.Config, error) {
						return config.Config{}, nil
					},
				})

				Expect(err).ToNot(HaveOccurred())
				Expect(reflect.TypeOf(creator.logger)).To(Equal(reflect.TypeOf(&logging.Logger{})))
			})
		})

		Context("when EventManager constructor is provided", func() {
			It("should return with the provided EventManager", func() {
				log, _ := NewLogger()

				expectedEventManager := eventmanager.EventManager{
					Log: log,
				}

				creator, err := New(CreatorModuleProvider{
					NewConfig: func() (config.Config, error) {
						return config.Config{}, nil
					},
					NewEventManager: func(logger I.Logger) I.EventManager {
						return &expectedEventManager
					},
				})

				Expect(err).ToNot(HaveOccurred())
				Expect(creator.eventManager).To(Equal(&expectedEventManager))
			})
		})

		Context("when EventManager constructor is not provided", func() {
			It("should return the default EventManager", func() {
				creator, err := New(CreatorModuleProvider{
					NewConfig: func() (config.Config, error) {
						return config.Config{}, nil
					},
				})

				Expect(err).ToNot(HaveOccurred())
				Expect(reflect.TypeOf(creator.eventManager)).To(Equal(reflect.TypeOf(&eventmanager.EventManager{})))
			})
		})

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
				os.Setenv("CF_USERNAME", "test user")
				os.Setenv("CF_PASSWORD", "test pwd")

				level := "DEBUG"
				configPath := "./testconfig.yml"

				creator, err := Custom(level, configPath, CreatorModuleProvider{})
				Expect(err).ToNot(HaveOccurred())
				resolver := creator.CreateAuthResolver()
				Expect(reflect.TypeOf(resolver)).To(Equal(reflect.TypeOf(state.AuthResolver{})))
				concrete := resolver.(state.AuthResolver)
				Expect(concrete.Config).ToNot(BeNil())
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

	Describe("CreateRequestCreator", func() {
		Context("when the provided request is a PostDeploymentRequest", func() {
			Context("when mock constructor is provided", func() {
				It("should return the mock implementation", func() {
					os.Setenv("CF_USERNAME", "test user")
					os.Setenv("CF_PASSWORD", "test pwd")

					level := "DEBUG"
					configPath := "./testconfig.yml"

					expected := &mocks.RequestCreator{}
					creator, _ := Custom(level, configPath, CreatorModuleProvider{
						NewPushRequestCreator: func(creator Creator, uuid string, request I.PostDeploymentRequest, buffer *bytes.Buffer) I.RequestCreator {
							return expected
						},
					})
					rc, _ := creator.CreateRequestCreator("the uuid", I.PostDeploymentRequest{}, bytes.NewBuffer([]byte{}))
					Expect(rc).To(Equal(expected))
				})
			})

			Context("when mock constructor is not provided", func() {
				It("should return the default implementation", func() {
					os.Setenv("CF_USERNAME", "test user")
					os.Setenv("CF_PASSWORD", "test pwd")

					level := "DEBUG"
					configPath := "./testconfig.yml"

					response := bytes.NewBuffer([]byte("the response"))
					request := I.PostDeploymentRequest{
						Deployment: I.Deployment{
							CFContext: I.CFContext{
								Organization: "the org",
							},
						},
					}

					creator, _ := Custom(level, configPath, CreatorModuleProvider{})
					rc, _ := creator.CreateRequestCreator("the uuid", request, response)

					Expect(reflect.TypeOf(rc)).To(Equal(reflect.TypeOf(&PushRequestCreator{})))
					concrete := rc.(*PushRequestCreator)
					Expect(concrete.Creator.logger).To(Equal(creator.logger))
					Expect(concrete.Creator.fileSystem).To(Equal(creator.fileSystem))
					Expect(concrete.Creator.eventManager).To(Equal(creator.eventManager))
					Expect(concrete.Creator.config).To(Equal(creator.config))
					Expect(concrete.Buffer).To(Equal(response))
					Expect(concrete.Request).To(Equal(request))
					Expect(concrete.Log.UUID).To(Equal("the uuid"))
				})

			})
		})

		Context("when the provided request is a PutDeploymentRequest", func() {
			Context("and requested state is stopped", func() {
				Context("when mock constructor is provided", func() {
					It("should return the mock implementation", func() {
						os.Setenv("CF_USERNAME", "test user")
						os.Setenv("CF_PASSWORD", "test pwd")

						level := "DEBUG"
						configPath := "./testconfig.yml"

						expected := &mocks.RequestCreator{}
						creator, _ := Custom(level, configPath, CreatorModuleProvider{
							NewStopRequestCreator: func(creator Creator, uuid string, request I.PutDeploymentRequest, buffer *bytes.Buffer) I.RequestCreator {
								return expected
							},
						})
						rc, _ := creator.CreateRequestCreator("the uuid", I.PutDeploymentRequest{Request: I.PutRequest{State: "stopped"}}, bytes.NewBuffer([]byte{}))
						Expect(rc).To(Equal(expected))
					})
				})

				Context("when mock constructor is not provided", func() {
					It("should return the default implementation", func() {
						os.Setenv("CF_USERNAME", "test user")
						os.Setenv("CF_PASSWORD", "test pwd")

						level := "DEBUG"
						configPath := "./testconfig.yml"

						response := bytes.NewBuffer([]byte("the response"))
						request := I.PutDeploymentRequest{
							Deployment: I.Deployment{
								CFContext: I.CFContext{
									Organization: "the org",
								},
							},
							Request: I.PutRequest{
								State: "stopped",
							},
						}

						creator, _ := Custom(level, configPath, CreatorModuleProvider{})
						rc, _ := creator.CreateRequestCreator("the uuid", request, response)

						Expect(reflect.TypeOf(rc)).To(Equal(reflect.TypeOf(&StopRequestCreator{})))
						concrete := rc.(*StopRequestCreator)
						Expect(concrete.Creator.logger).To(Equal(creator.logger))
						Expect(concrete.Creator.fileSystem).To(Equal(creator.fileSystem))
						Expect(concrete.Creator.eventManager).To(Equal(creator.eventManager))
						Expect(concrete.Creator.config).To(Equal(creator.config))
						Expect(concrete.Buffer).To(Equal(response))
						Expect(concrete.Request).To(Equal(request))
						Expect(concrete.Log.UUID).To(Equal("the uuid"))
					})

				})
			})

			Context("and requested state is started", func() {
				Context("when mock constructor is provided", func() {
					It("should return the mock implementation", func() {
						os.Setenv("CF_USERNAME", "test user")
						os.Setenv("CF_PASSWORD", "test pwd")

						level := "DEBUG"
						configPath := "./testconfig.yml"

						expected := &mocks.RequestCreator{}
						creator, _ := Custom(level, configPath, CreatorModuleProvider{
							NewStartRequestCreator: func(creator Creator, uuid string, request I.PutDeploymentRequest, buffer *bytes.Buffer) I.RequestCreator {
								return expected
							},
						})
						rc, _ := creator.CreateRequestCreator("the uuid", I.PutDeploymentRequest{Request: I.PutRequest{State: "started"}}, bytes.NewBuffer([]byte{}))
						Expect(rc).To(Equal(expected))
					})
				})

				Context("when mock constructor is not provided", func() {
					It("should return the default implementation", func() {
						os.Setenv("CF_USERNAME", "test user")
						os.Setenv("CF_PASSWORD", "test pwd")

						level := "DEBUG"
						configPath := "./testconfig.yml"

						response := bytes.NewBuffer([]byte("the response"))
						request := I.PutDeploymentRequest{
							Deployment: I.Deployment{
								CFContext: I.CFContext{
									Organization: "the org",
								},
							},
							Request: I.PutRequest{
								State: "started",
							},
						}

						creator, _ := Custom(level, configPath, CreatorModuleProvider{})
						rc, _ := creator.CreateRequestCreator("the uuid", request, response)

						Expect(reflect.TypeOf(rc)).To(Equal(reflect.TypeOf(&StartRequestCreator{})))
						concrete := rc.(*StartRequestCreator)
						Expect(concrete.Creator.logger).To(Equal(creator.logger))
						Expect(concrete.Creator.fileSystem).To(Equal(creator.fileSystem))
						Expect(concrete.Creator.eventManager).To(Equal(creator.eventManager))
						Expect(concrete.Creator.config).To(Equal(creator.config))
						Expect(concrete.Buffer).To(Equal(response))
						Expect(concrete.Request).To(Equal(request))
						Expect(concrete.Log.UUID).To(Equal("the uuid"))
					})

				})
			})
		})

		Context("when the provided request is unknown", func() {
			It("returns an error", func() {
				os.Setenv("CF_USERNAME", "test user")
				os.Setenv("CF_PASSWORD", "test pwd")

				level := "DEBUG"
				configPath := "./testconfig.yml"

				response := bytes.NewBuffer([]byte("the response"))

				creator, _ := Custom(level, configPath, CreatorModuleProvider{})
				_, err := creator.CreateRequestCreator("the uuid", "", response)

				Expect(err).To(HaveOccurred())
				Expect(reflect.TypeOf(err)).To(Equal(reflect.TypeOf(InvalidRequestError{})))
			})
		})
	})

	Describe("CreateRequestProcessor", func() {
		Context("when mock constructor is provided", func() {
			It("should return the mock implementation", func() {
				os.Setenv("CF_USERNAME", "test user")
				os.Setenv("CF_PASSWORD", "test pwd")

				level := "DEBUG"
				configPath := "./testconfig.yml"

				expected := &mocks.RequestProcessor{}
				creator, _ := Custom(level, configPath, CreatorModuleProvider{
					NewPushRequestProcessor: func(log I.DeploymentLogger, controller I.PushController, request I.PostDeploymentRequest, buffer *bytes.Buffer) I.RequestProcessor {
						return expected
					},
				})
				processor := creator.CreateRequestProcessor("the uuid", I.PostDeploymentRequest{}, bytes.NewBuffer([]byte{}))
				Expect(processor).To(Equal(expected))
			})
		})

		Context("when mock constructor is not provided", func() {
			It("should return the default implementation", func() {
				os.Setenv("CF_USERNAME", "test user")
				os.Setenv("CF_PASSWORD", "test pwd")

				level := "DEBUG"
				configPath := "./testconfig.yml"

				response := bytes.NewBuffer([]byte("the response"))
				request := I.PostDeploymentRequest{
					Deployment: I.Deployment{
						CFContext: I.CFContext{
							Organization: "the org",
						},
					},
				}

				creator, _ := Custom(level, configPath, CreatorModuleProvider{})
				processor := creator.CreateRequestProcessor("the uuid", request, response)

				Expect(reflect.TypeOf(processor)).To(Equal(reflect.TypeOf(&push.PushRequestProcessor{})))
				concrete := processor.(*push.PushRequestProcessor)
				Expect(concrete.PushController).ToNot(BeNil())
				Expect(concrete.Response).To(Equal(response))
				Expect(concrete.Request).To(Equal(request))
				Expect(concrete.Log.UUID).To(Equal("the uuid"))
			})

		})

		Context("when an unknown request is provided", func() {
			It("returns an InvalidRequestProcessor", func() {
				os.Setenv("CF_USERNAME", "test user")
				os.Setenv("CF_PASSWORD", "test pwd")

				level := "DEBUG"
				configPath := "./testconfig.yml"

				response := bytes.NewBuffer([]byte("the response"))

				request := ""

				creator, _ := Custom(level, configPath, CreatorModuleProvider{})
				processor := creator.CreateRequestProcessor("the uuid", request, response)

				Expect(reflect.TypeOf(processor)).To(Equal(reflect.TypeOf(InvalidRequestProcessor{})))
			})
		})
	})

})
