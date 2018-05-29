package push_test

import (
	"bytes"
	"fmt"
	"github.com/compozed/deployadactyl/config"
	"github.com/compozed/deployadactyl/constants"
	D "github.com/compozed/deployadactyl/controller/deployer"
	"github.com/compozed/deployadactyl/controller/deployer/bluegreen"
	"github.com/compozed/deployadactyl/controller/deployer/error_finder"
	I "github.com/compozed/deployadactyl/interfaces"
	"github.com/compozed/deployadactyl/mocks"
	"github.com/compozed/deployadactyl/randomizer"
	"github.com/compozed/deployadactyl/state"
	"github.com/compozed/deployadactyl/state/push"
	"github.com/compozed/deployadactyl/structs"
	"github.com/go-errors/errors"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	"github.com/op/go-logging"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
)

var _ = Describe("RunDeployment", func() {
	var (
		deployer           *mocks.Deployer
		silentDeployer     *mocks.Deployer
		pushManagerFactory *mocks.PushManagerFactory
		eventManager       *mocks.EventManager
		errorFinder        *mocks.ErrorFinder
		controller         *push.PushController
		deployment         I.Deployment
		authResolver       *state.AuthResolver
		logBuffer          *Buffer

		appName     string
		environment string
		org         string
		space       string
		uuid        string

		response *bytes.Buffer
	)

	BeforeEach(func() {
		logBuffer = NewBuffer()
		appName = "appName-" + randomizer.StringRunes(10)
		environment = "environment-" + randomizer.StringRunes(10)
		org = "org-" + randomizer.StringRunes(10)
		space = "non-prod"
		uuid = "uuid-" + randomizer.StringRunes(10)

		eventManager = &mocks.EventManager{}
		deployer = &mocks.Deployer{}
		silentDeployer = &mocks.Deployer{}
		pushManagerFactory = &mocks.PushManagerFactory{}

		authResolver = &state.AuthResolver{Config: config.Config{}}

		errorFinder = &mocks.ErrorFinder{}
		controller = &push.PushController{
			Deployer:           deployer,
			SilentDeployer:     silentDeployer,
			Log:                I.DeploymentLogger{Log: I.DefaultLogger(logBuffer, logging.DEBUG, "api_test"), UUID: uuid},
			PushManagerFactory: pushManagerFactory,
			EventManager:       eventManager,
			Config:             config.Config{},
			ErrorFinder:        errorFinder,
			AuthResolver:       authResolver,
		}

		environments := map[string]structs.Environment{}
		environments[environment] = structs.Environment{}
		controller.Config.Environments = environments
		bodyByte := []byte("{}")
		response = &bytes.Buffer{}

		deployment = I.Deployment{
			Body:          &bodyByte,
			Type:          I.DeploymentType{},
			CFContext:     I.CFContext{},
			Authorization: I.Authorization{},
		}

	})
	Context("when verbose deployer is called", func() {
		It("deployer is provided correct authorization", func() {

			deployer.DeployCall.Returns.Error = nil
			deployer.DeployCall.Returns.StatusCode = http.StatusOK
			deployer.DeployCall.Write.Output = "little-timmy-env.zip"

			response := &bytes.Buffer{}

			deployment := &I.Deployment{
				Body: &[]byte{},
				Authorization: I.Authorization{
					Username: "username",
					Password: "password",
				},
				CFContext: I.CFContext{
					Environment:  environment,
					Organization: org,
					Space:        space,
					Application:  appName,
				},
			}
			deployment.Type.ZIP = true
			deployResponse := controller.RunDeployment(deployment, response)

			Eventually(deployer.DeployCall.Called).Should(Equal(1))
			Eventually(silentDeployer.DeployCall.Called).Should(Equal(0))

			Eventually(deployResponse.StatusCode).Should(Equal(http.StatusOK))

			Eventually(deployer.DeployCall.Received.DeploymentInfo.Username).Should(Equal(deployment.Authorization.Username))
			Eventually(deployer.DeployCall.Received.DeploymentInfo.Password).Should(Equal(deployment.Authorization.Password))
		})

		It("deployer is provided the body", func() {

			deployer.DeployCall.Returns.Error = nil
			deployer.DeployCall.Returns.StatusCode = http.StatusOK
			deployer.DeployCall.Write.Output = "little-timmy-env.zip"

			response := &bytes.Buffer{}

			bodyBytes := []byte("a test body string")

			deployment := &I.Deployment{
				Body: &bodyBytes,
				Authorization: I.Authorization{
					Username: "username",
					Password: "password",
				},
				CFContext: I.CFContext{
					Environment:  environment,
					Organization: org,
					Space:        space,
					Application:  appName,
				},
			}
			deployment.Type.ZIP = true

			deployResponse := controller.RunDeployment(deployment, response)
			receivedBody, _ := ioutil.ReadAll(deployer.DeployCall.Received.DeploymentInfo.Body)
			Eventually(deployer.DeployCall.Called).Should(Equal(1))
			Eventually(silentDeployer.DeployCall.Called).Should(Equal(0))

			Eventually(deployResponse.StatusCode).Should(Equal(http.StatusOK))

			Eventually(receivedBody).Should(Equal(*deployment.Body))
		})

		It("channel resolves when no errors occur", func() {
			deployment.CFContext.Environment = environment
			deployment.CFContext.Organization = org
			deployment.CFContext.Space = space
			deployment.CFContext.Application = appName
			deployment.Type.ZIP = true

			deployer.DeployCall.Returns.Error = nil
			deployer.DeployCall.Returns.StatusCode = http.StatusOK
			deployer.DeployCall.Write.Output = "little-timmy-env.zip"

			deployResponse := controller.RunDeployment(&deployment, response)

			Eventually(deployer.DeployCall.Called).Should(Equal(1))
			Eventually(silentDeployer.DeployCall.Called).Should(Equal(0))

			Eventually(deployResponse.StatusCode).Should(Equal(http.StatusOK))

			Eventually(deployer.DeployCall.Received.DeploymentInfo.ContentType).Should(Equal("ZIP"))
			Eventually(deployer.DeployCall.Received.DeploymentInfo.Environment).Should(Equal(environment))
			Eventually(deployer.DeployCall.Received.DeploymentInfo.Org).Should(Equal(org))
			Eventually(deployer.DeployCall.Received.DeploymentInfo.Space).Should(Equal(space))
			Eventually(deployer.DeployCall.Received.DeploymentInfo.AppName).Should(Equal(appName))

			ret, _ := ioutil.ReadAll(response)
			Eventually(string(ret)).Should(Equal("little-timmy-env.zip"))
		})

		It("channel resolves when errors occur", func() {
			deployment.CFContext.Environment = environment
			deployment.CFContext.Organization = org
			deployment.CFContext.Space = space
			deployment.CFContext.Application = appName
			deployment.Type.ZIP = true

			deployer.DeployCall.Returns.Error = errors.New("bork")
			deployer.DeployCall.Returns.StatusCode = http.StatusInternalServerError
			deployer.DeployCall.Write.Output = "little-timmy-env.zip"

			deployResponse := controller.RunDeployment(&deployment, response)

			Eventually(deployer.DeployCall.Called).Should(Equal(1))
			Eventually(silentDeployer.DeployCall.Called).Should(Equal(0))

			Eventually(deployResponse.StatusCode).Should(Equal(http.StatusInternalServerError))
			Eventually(deployResponse.Error.Error()).Should(Equal("bork"))

			Eventually(deployer.DeployCall.Received.DeploymentInfo.ContentType).Should(Equal("ZIP"))
			Eventually(deployer.DeployCall.Received.DeploymentInfo.Environment).Should(Equal(environment))
			Eventually(deployer.DeployCall.Received.DeploymentInfo.Org).Should(Equal(org))
			Eventually(deployer.DeployCall.Received.DeploymentInfo.Space).Should(Equal(space))
			Eventually(deployer.DeployCall.Received.DeploymentInfo.AppName).Should(Equal(appName))

			ret, _ := ioutil.ReadAll(response)
			Eventually(string(ret)).Should(Equal("little-timmy-env.zip"))
		})

		It("does not set the basic auth header if no credentials are passed", func() {
			deployer.DeployCall.Write.Output = "little-timmy-env.zip"

			response := &bytes.Buffer{}

			deployment := &I.Deployment{
				Body: &[]byte{},
				Type: I.DeploymentType{ZIP: true},
				CFContext: I.CFContext{
					Environment:  environment,
					Organization: org,
					Space:        space,
					Application:  appName,
				},
				Authorization: I.Authorization{
					Username: "",
					Password: "",
				},
			}
			controller.RunDeployment(deployment, response)

			Eventually(deployer.DeployCall.Received.DeploymentInfo.Username).Should(Equal(""))
			Eventually(deployer.DeployCall.Received.DeploymentInfo.Password).Should(Equal(""))
		})

		It("sets the basic auth header if credentials are passed", func() {
			deployment.CFContext.Environment = environment
			deployment.Type.ZIP = true

			deployer.DeployCall.Write.Output = "little-timmy-env.zip"

			deployment.Authorization = I.Authorization{
				Username: "TestUsername",
				Password: "TestPassword",
			}

			controller.RunDeployment(&deployment, response)

			Eventually(deployer.DeployCall.Received.DeploymentInfo.Username).Should(Equal("TestUsername"))
			Eventually(deployer.DeployCall.Received.DeploymentInfo.Password).Should(Equal("TestPassword"))
		})
	})

	Context("when SILENT_DEPLOY_ENVIRONMENT is true", func() {
		It("channel resolves true when no errors occur", func() {
			deployment.CFContext.Environment = environment
			deployment.CFContext.Organization = org
			deployment.CFContext.Space = space
			deployment.CFContext.Application = appName
			deployment.Type.ZIP = true

			os.Setenv("SILENT_DEPLOY_ENVIRONMENT", environment)
			deployer.DeployCall.Returns.Error = nil
			deployer.DeployCall.Returns.StatusCode = http.StatusOK
			deployer.DeployCall.Write.Output = "little-timmy-env.zip"

			deployResponse := controller.RunDeployment(&deployment, response)

			Eventually(deployer.DeployCall.Called).Should(Equal(1))
			Eventually(silentDeployer.DeployCall.Called).Should(Equal(1))

			Eventually(deployResponse.StatusCode).Should(Equal(http.StatusOK))

			Eventually(deployer.DeployCall.Received.DeploymentInfo.ContentType).Should(Equal("ZIP"))
			Eventually(deployer.DeployCall.Received.DeploymentInfo.Environment).Should(Equal(environment))
			Eventually(deployer.DeployCall.Received.DeploymentInfo.Org).Should(Equal(org))
			Eventually(deployer.DeployCall.Received.DeploymentInfo.Space).Should(Equal(space))
			Eventually(deployer.DeployCall.Received.DeploymentInfo.AppName).Should(Equal(appName))

			ret, _ := ioutil.ReadAll(response)
			Eventually(string(ret)).Should(Equal("little-timmy-env.zip"))
		})
		It("channel resolves when no errors occur", func() {
			deployment.CFContext.Environment = environment
			deployment.CFContext.Organization = org
			deployment.CFContext.Space = space
			deployment.CFContext.Application = appName
			deployment.Type.ZIP = true

			os.Setenv("SILENT_DEPLOY_ENVIRONMENT", environment)
			deployer.DeployCall.Returns.Error = nil
			deployer.DeployCall.Returns.StatusCode = http.StatusOK
			deployer.DeployCall.Write.Output = "little-timmy-env.zip"

			silentDeployer.DeployCall.Returns.Error = errors.New("bork")
			silentDeployer.DeployCall.Returns.StatusCode = http.StatusInternalServerError

			server := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
				req.Body.Close()
			}))
			silentDeployUrl := server.URL + "/v1/apps/" + os.Getenv("SILENT_DEPLOY_ENVIRONMENT")
			os.Setenv("SILENT_DEPLOY_URL", silentDeployUrl)

			deployResponse := controller.RunDeployment(&deployment, response)

			Eventually(deployer.DeployCall.Called).Should(Equal(1))
			Eventually(silentDeployer.DeployCall.Called).Should(Equal(1))

			Eventually(deployResponse.StatusCode).Should(Equal(http.StatusOK))

			Eventually(deployer.DeployCall.Received.DeploymentInfo.ContentType).Should(Equal("ZIP"))
			Eventually(deployer.DeployCall.Received.DeploymentInfo.Environment).Should(Equal(environment))
			Eventually(deployer.DeployCall.Received.DeploymentInfo.Org).Should(Equal(org))
			Eventually(deployer.DeployCall.Received.DeploymentInfo.Space).Should(Equal(space))
			Eventually(deployer.DeployCall.Received.DeploymentInfo.AppName).Should(Equal(appName))

			ret, _ := ioutil.ReadAll(response)
			Eventually(string(ret)).Should(Equal("little-timmy-env.zip"))
		})
	})

	Context("when called", func() {
		It("logs building deploymentInfo", func() {
			deployment.CFContext.Environment = environment

			controller.RunDeployment(&deployment, response)
			Eventually(logBuffer).Should(Say("building deploymentInfo"))
		})
		It("creates a pusher creator", func() {
			deployment.CFContext.Environment = environment
			deployment.Type.ZIP = true

			controller.RunDeployment(&deployment, response)
			Eventually(pushManagerFactory.PushManagerCall.Called).Should(Equal(true))

		})
		It("Provides body for pusher creator", func() {
			bodyByte := []byte("body string")
			deployment.CFContext.Environment = environment
			deployment.Body = &bodyByte
			deployment.Type.ZIP = true

			controller.RunDeployment(&deployment, response)
			returnedBody, _ := ioutil.ReadAll(pushManagerFactory.PushManagerCall.Received.DeployEventData.RequestBody)
			Eventually(returnedBody).Should(Equal(bodyByte))
		})
		It("Provides response for pusher creator", func() {
			deployment.CFContext.Environment = environment
			deployment.Type.ZIP = true

			response = bytes.NewBuffer([]byte("hello"))

			controller.RunDeployment(&deployment, response)
			returnedResponse, _ := ioutil.ReadAll(pushManagerFactory.PushManagerCall.Received.DeployEventData.Response)
			Eventually(returnedResponse).Should(Equal([]byte("hello")))
		})
		Context("when type is JSON", func() {
			It("gets the artifact url from the request", func() {
				bodyByte := []byte("{\"artifact_url\": \"the artifact url\"}")
				deployment.Body = &bodyByte
				deployment.CFContext.Environment = environment
				deployment.Type.JSON = true

				controller.RunDeployment(&deployment, response)
				Eventually(pushManagerFactory.PushManagerCall.Received.DeployEventData.DeploymentInfo.ArtifactURL).Should(Equal("the artifact url"))
			})
			It("gets the manifest from the request", func() {
				bodyByte := []byte("{\"artifact_url\": \"the artifact url\", \"manifest\": \"the manifest\"}")
				deployment.Body = &bodyByte
				deployment.CFContext.Environment = environment
				deployment.Type.JSON = true

				controller.RunDeployment(&deployment, response)
				Eventually(pushManagerFactory.PushManagerCall.Received.DeployEventData.DeploymentInfo.Manifest).Should(Equal("the manifest"))
			})
			It("gets the data from the request", func() {
				bodyByte := []byte("{\"artifact_url\": \"the artifact url\", \"data\": {\"avalue\": \"the data\"}}")
				deployment.Body = &bodyByte
				deployment.CFContext.Environment = environment
				deployment.Type.JSON = true

				controller.RunDeployment(&deployment, response)
				Eventually(pushManagerFactory.PushManagerCall.Received.DeployEventData.DeploymentInfo.Data["avalue"]).Should(Equal("the data"))
			})
		})
		Context("the deployment info", func() {
			Context("when environment does not exist", func() {
				It("returns an error with StatusInternalServerError", func() {
					deployment.CFContext.Environment = "bad env"
					deployment.Type.ZIP = true

					deploymentResponse := controller.RunDeployment(&deployment, response)
					Eventually(deploymentResponse.Error).Should(HaveOccurred())
					Eventually(reflect.TypeOf(deploymentResponse.Error)).Should(Equal(reflect.TypeOf(D.EnvironmentNotFoundError{})))
				})
			})
			Context("when environment exists", func() {
				Context("when Authorization doesn't have values", func() {
					Context("and authentication is not required", func() {
						It("returns username and password from the config", func() {
							deployment.CFContext.Environment = environment
							deployment.Type.ZIP = true

							deployment.Authorization.Username = ""
							deployment.Authorization.Password = ""
							authResolver.Config.Username = "username-" + randomizer.StringRunes(10)
							authResolver.Config.Password = "password-" + randomizer.StringRunes(10)

							controller.RunDeployment(&deployment, response)

							Eventually(pushManagerFactory.PushManagerCall.Received.DeployEventData.DeploymentInfo.Username).Should(Equal(authResolver.Config.Username))
							Eventually(pushManagerFactory.PushManagerCall.Received.DeployEventData.DeploymentInfo.Password).Should(Equal(authResolver.Config.Password))
						})
					})
					Context("and authentication is required", func() {
						It("returns an error", func() {
							deployment.CFContext.Environment = environment
							deployment.Type.ZIP = true

							deployment.Authorization.Username = ""
							deployment.Authorization.Password = ""

							controller.Config.Environments[environment] = structs.Environment{
								Authenticate: true,
							}

							deploymentResponse := controller.RunDeployment(&deployment, response)

							Eventually(deploymentResponse.Error).Should(HaveOccurred())
							Eventually(deploymentResponse.Error.Error()).Should(Equal("basic auth header not found"))
						})
					})
				})
				Context("when Authorization has values", func() {
					It("logs checking auth", func() {
						deployment.CFContext.Environment = environment
						deployment.Type.ZIP = true

						controller.RunDeployment(&deployment, response)

						Eventually(logBuffer).Should(Say("checking for basic auth"))
					})
					It("returns username and password from the authorization", func() {
						deployment.CFContext.Environment = environment
						deployment.Type.ZIP = true

						deployment.Authorization.Username = "username-" + randomizer.StringRunes(10)
						deployment.Authorization.Password = "password-" + randomizer.StringRunes(10)

						controller.RunDeployment(&deployment, response)

						Eventually(pushManagerFactory.PushManagerCall.Received.DeployEventData.DeploymentInfo.Username).Should(Equal(deployment.Authorization.Username))
						Eventually(pushManagerFactory.PushManagerCall.Received.DeployEventData.DeploymentInfo.Password).Should(Equal(deployment.Authorization.Password))
					})
				})
				It("has the correct org, space ,appname, env, uuid", func() {
					deployment.CFContext.Environment = environment
					deployment.Type.ZIP = true

					deployment.CFContext.Organization = org
					deployment.CFContext.Space = space
					deployment.CFContext.Application = appName
					deployment.CFContext.Environment = environment

					controller.RunDeployment(&deployment, response)

					Eventually(pushManagerFactory.PushManagerCall.Received.DeployEventData.DeploymentInfo.Org).Should(Equal(org))
					Eventually(pushManagerFactory.PushManagerCall.Received.DeployEventData.DeploymentInfo.Space).Should(Equal(space))
					Eventually(pushManagerFactory.PushManagerCall.Received.DeployEventData.DeploymentInfo.AppName).Should(Equal(appName))
					Eventually(pushManagerFactory.PushManagerCall.Received.DeployEventData.DeploymentInfo.Environment).Should(Equal(environment))
				})
				It("has the correct JSON content type", func() {
					deployment.CFContext.Environment = environment
					deployment.Type.JSON = true
					bodyByte := []byte(`{"artifact_url": "xyz"}`)
					deployment.Body = &bodyByte

					controller.RunDeployment(&deployment, response)

					Eventually(pushManagerFactory.PushManagerCall.Received.DeployEventData.DeploymentInfo.ContentType).Should(Equal("JSON"))
				})
				It("has the correct ZIP content type", func() {
					deployment.CFContext.Environment = environment
					deployment.Type.ZIP = true

					controller.RunDeployment(&deployment, response)

					Eventually(pushManagerFactory.PushManagerCall.Received.DeployEventData.DeploymentInfo.ContentType).Should(Equal("ZIP"))
				})
				It("has the correct body", func() {
					deployment.CFContext.Environment = environment
					deployment.Type.ZIP = true
					bodyByte := []byte(`{"artifact_url": "xyz"}`)
					deployment.Body = &bodyByte

					controller.RunDeployment(&deployment, response)

					returnedBody, _ := ioutil.ReadAll(pushManagerFactory.PushManagerCall.Received.DeployEventData.DeploymentInfo.Body)
					Eventually(string(returnedBody)).Should(Equal(string(bodyByte)))
				})

				Context("when contentType is neither", func() {
					It("returns an error", func() {
						deployment.CFContext.Environment = environment

						deployResponse := controller.RunDeployment(&deployment, response)

						Eventually(reflect.TypeOf(deployResponse.Error)).Should(Equal(reflect.TypeOf(D.InvalidContentTypeError{})))
					})
				})

				Context("when uuid is not provided", func() {
					It("creates a new uuid", func() {
						deployment.CFContext.Environment = environment
						deployment.Type.ZIP = true

						controller.RunDeployment(&deployment, response)

						Eventually(pushManagerFactory.PushManagerCall.Received.DeployEventData.DeploymentInfo.UUID).ShouldNot(BeEmpty())
					})
				})

				It("has the correct domain and skipssl", func() {
					deployment.CFContext.Environment = environment
					domain := "domain-" + randomizer.StringRunes(10)
					deployment.Authorization.Username = ""
					deployment.Authorization.Password = ""
					deployment.Type.ZIP = true

					controller.Config.Environments[environment] = structs.Environment{
						Domain:  domain,
						SkipSSL: true,
					}

					controller.RunDeployment(&deployment, response)

					Eventually(pushManagerFactory.PushManagerCall.Received.DeployEventData.DeploymentInfo.Domain).Should(Equal(domain))
					Eventually(pushManagerFactory.PushManagerCall.Received.DeployEventData.DeploymentInfo.SkipSSL).Should(BeTrue())
				})
				It("has correct custom parameters", func() {

					deployment.CFContext.Environment = environment
					deployment.Type.ZIP = true

					customParams := make(map[string]interface{})
					customParams["param1"] = "value1"
					customParams["param2"] = "value2"

					controller.Config.Environments[environment] = structs.Environment{
						CustomParams: customParams,
					}

					controller.RunDeployment(&deployment, response)

					Eventually(pushManagerFactory.PushManagerCall.Received.DeployEventData.DeploymentInfo.CustomParams["param1"]).Should(Equal("value1"))
					Eventually(pushManagerFactory.PushManagerCall.Received.DeployEventData.DeploymentInfo.CustomParams["param2"]).Should(Equal("value2"))

				})
				It("creates a PushManager", func() {
					deployment.CFContext.Environment = environment
					deployment.Authorization.Username = randomizer.StringRunes(10)
					deployment.Type.ZIP = true

					controller.RunDeployment(&deployment, response)

					Eventually(pushManagerFactory.PushManagerCall.Received.DeployEventData.DeploymentInfo).ShouldNot(BeNil())
					Eventually(pushManagerFactory.PushManagerCall.Received.CFContext.Environment).Should(Equal(environment))
					Eventually(pushManagerFactory.PushManagerCall.Received.Auth.Username).Should(Equal(deployment.Authorization.Username))
					Expect(pushManagerFactory.PushManagerCall.Received.Environment).ToNot(BeNil())
				})
				It("correctly extracts artifact url from body", func() {
					artifactURL := "artifactURL-" + randomizer.StringRunes(10)
					bodyByte := []byte(fmt.Sprintf(`{"artifact_url": "%s"}`, artifactURL))

					deployment.CFContext.Environment = environment
					deployment.Body = &bodyByte
					deployment.Type.JSON = true

					controller.RunDeployment(&deployment, response)

					Eventually(pushManagerFactory.PushManagerCall.Received.DeployEventData.DeploymentInfo.ArtifactURL).Should(Equal(artifactURL))
				})
				Context("if artifact url isn't provided in body", func() {
					It("returns an error", func() {
						bodyByte := []byte("{}")

						deployment.CFContext.Environment = environment
						deployment.Body = &bodyByte
						deployment.Type.JSON = true

						deploymentResponse := controller.RunDeployment(&deployment, response)

						Eventually(deploymentResponse.Error).ShouldNot(BeNil())
						Eventually(deploymentResponse.Error.Error()).Should(ContainSubstring("The following properties are missing: artifact_url"))
					})
				})
				Context("if body is invalid", func() {
					It("returns an error", func() {
						bodyByte := []byte("")

						deployment.CFContext.Environment = environment
						deployment.Body = &bodyByte
						deployment.Type.JSON = true

						deploymentResponse := controller.RunDeployment(&deployment, response)

						Eventually(deploymentResponse.Error).ShouldNot(BeNil())
						Eventually(deploymentResponse.Error.Error()).Should(ContainSubstring("EOF"))
					})
				})
				Context("deploy.start event", func() {
					It("logs a start event", func() {
						deployment.CFContext.Environment = environment
						deployment.Type.ZIP = true

						controller.RunDeployment(&deployment, response)

						Eventually(logBuffer).Should(Say("emitting a deploy.start event"))
					})
					It("calls Emit", func() {
						deployment.CFContext.Environment = environment
						deployment.Type.ZIP = true

						controller.RunDeployment(&deployment, response)

						Expect(eventManager.EmitCall.Received.Events[0].Type).Should(Equal(constants.DeployStartEvent))
					})
					It("calls EmitEvent", func() {
						deployment.CFContext.Environment = environment
						deployment.Type.ZIP = true

						controller.RunDeployment(&deployment, response)

						Expect(eventManager.EmitEventCall.Received.Events[0].Name()).Should(Equal("DeployStartedEvent"))
					})
					Context("when Emit fails", func() {
						It("returns error", func() {
							deployment.CFContext.Environment = environment
							deployment.Type.ZIP = true

							eventManager.EmitCall.Returns.Error = []error{errors.New("a test error")}

							deploymentResponse := controller.RunDeployment(&deployment, response)

							Expect(reflect.TypeOf(deploymentResponse.Error)).Should(Equal(reflect.TypeOf(D.EventError{})))
						})
					})
					Context("when EmitEvent fails", func() {
						It("returns error", func() {
							deployment.CFContext.Environment = environment
							deployment.Type.ZIP = true

							eventManager.EmitEventCall.Returns.Error = []error{errors.New("a test error")}

							deploymentResponse := controller.RunDeployment(&deployment, response)

							Expect(reflect.TypeOf(deploymentResponse.Error)).Should(Equal(reflect.TypeOf(D.EventError{})))
						})
					})
					It("passes populated deploymentInfo to DeployStartEvent event", func() {
						deployment.CFContext.Environment = environment
						deployment.CFContext.Application = appName
						deployment.CFContext.Space = space
						deployment.CFContext.Organization = org
						deployment.Type.ZIP = true

						controller.RunDeployment(&deployment, response)

						deploymentInfo := eventManager.EmitCall.Received.Events[0].Data.(*structs.DeployEventData).DeploymentInfo
						Expect(deploymentInfo.AppName).To(Equal(appName))
						Expect(deploymentInfo.Org).To(Equal(org))
						Expect(deploymentInfo.Space).To(Equal(space))
						Expect(deploymentInfo.UUID).ToNot(BeNil())
					})
					It("passes CFContext to EmitEvent in the event", func() {
						deployment.CFContext.Environment = environment
						deployment.CFContext.Application = appName
						deployment.CFContext.Space = space
						deployment.CFContext.Organization = org

						deployment.Type.ZIP = true

						controller.RunDeployment(&deployment, response)

						event := eventManager.EmitEventCall.Received.Events[0].(push.DeployStartedEvent)
						Expect(event.CFContext.Environment).To(Equal(environment))
						Expect(event.CFContext.Application).To(Equal(appName))
						Expect(event.CFContext.Space).To(Equal(space))
						Expect(event.CFContext.Organization).To(Equal(org))
					})
					It("passes Auth to EmitEvent in the event", func() {
						deployment.CFContext.Environment = environment
						deployment.Authorization = I.Authorization{
							Username: "myuser",
							Password: "mypassword",
						}

						deployment.Type.ZIP = true

						controller.RunDeployment(&deployment, response)

						event := eventManager.EmitEventCall.Received.Events[0].(push.DeployStartedEvent)
						Expect(event.Auth.Username).To(Equal("myuser"))
						Expect(event.Auth.Password).To(Equal("mypassword"))
					})
					It("passes other info to EmitEvent", func() {
						deployment.CFContext.Environment = environment

						controller.Config.Environments[environment] = structs.Environment{
							Name: environment,
						}

						deployment.Type.ZIP = true

						controller.RunDeployment(&deployment, response)

						event := eventManager.EmitEventCall.Received.Events[0].(push.DeployStartedEvent)
						Expect(event.Body).ToNot(BeNil())
						Expect(event.ContentType).To(Equal("ZIP"))
						Expect(event.Environment.Name).To(Equal(environment))
						Expect(event.Response).ToNot(BeNil())
					})
				})
				Context("deploy.finish event", func() {

					It("calls Emit", func() {
						deployment.CFContext.Environment = environment
						deployment.Type.ZIP = true

						controller.RunDeployment(&deployment, response)
						Expect(eventManager.EmitCall.Received.Events[2].Type).Should(Equal(constants.DeployFinishEvent))
					})
					It("calls EmitEvent", func() {
						deployment.CFContext.Environment = environment
						deployment.Type.ZIP = true

						controller.RunDeployment(&deployment, response)

						Expect(eventManager.EmitEventCall.Received.Events[2].Name()).To(Equal(push.DeployFinishedEvent{}.Name()))
					})
					It("passes CFContext to Emit", func() {
						deployment.CFContext.Environment = environment
						deployment.CFContext.Application = appName
						deployment.CFContext.Space = space
						deployment.CFContext.Organization = org
						deployment.Type.ZIP = true

						controller.RunDeployment(&deployment, response)

						deploymentInfo := eventManager.EmitCall.Received.Events[2].Data.(*structs.DeployEventData).DeploymentInfo
						Expect(deploymentInfo.AppName).To(Equal(appName))
						Expect(deploymentInfo.Org).To(Equal(org))
						Expect(deploymentInfo.Space).To(Equal(space))
						Expect(deploymentInfo.UUID).ToNot(BeNil())
					})
					It("passes CFContext to EmitEvent in the event", func() {
						deployment.CFContext.Environment = environment
						deployment.CFContext.Application = appName
						deployment.CFContext.Space = space
						deployment.CFContext.Organization = org

						deployment.Type.ZIP = true

						controller.RunDeployment(&deployment, response)

						event := eventManager.EmitEventCall.Received.Events[2].(push.DeployFinishedEvent)
						Expect(event.CFContext.Environment).To(Equal(environment))
						Expect(event.CFContext.Application).To(Equal(appName))
						Expect(event.CFContext.Space).To(Equal(space))
						Expect(event.CFContext.Organization).To(Equal(org))
					})
					It("passes Auth to EmitEvent in the event", func() {
						deployment.CFContext.Environment = environment
						deployment.Authorization = I.Authorization{
							Username: "myuser",
							Password: "mypassword",
						}

						deployment.Type.ZIP = true

						controller.RunDeployment(&deployment, response)

						event := eventManager.EmitEventCall.Received.Events[2].(push.DeployFinishedEvent)
						Expect(event.Auth.Username).To(Equal("myuser"))
						Expect(event.Auth.Password).To(Equal("mypassword"))
					})
					It("passes other info to EmitEvent", func() {
						deployment.CFContext.Environment = environment

						controller.Config.Environments[environment] = structs.Environment{
							Name: environment,
						}

						deployment.Type.ZIP = true

						controller.RunDeployment(&deployment, response)

						event := eventManager.EmitEventCall.Received.Events[2].(push.DeployFinishedEvent)
						Expect(event.Body).ToNot(BeNil())
						Expect(event.ContentType).To(Equal("ZIP"))
						Expect(event.Environment.Name).To(Equal(environment))
						Expect(event.Response).ToNot(BeNil())
					})
					Context("when Emit fails", func() {
						It("returns error", func() {
							deployment.CFContext.Environment = environment
							deployment.Type.ZIP = true

							eventManager.EmitCall.Returns.Error = []error{nil, nil, errors.New("a test error")}

							deploymentResponse := controller.RunDeployment(&deployment, response)

							Expect(reflect.TypeOf(deploymentResponse.Error)).Should(Equal(reflect.TypeOf(bluegreen.FinishDeployError{})))
						})
					})
					Context("when EmitEvent fails", func() {
						It("returns error", func() {
							deployment.CFContext.Environment = environment
							deployment.Type.ZIP = true

							eventManager.EmitEventCall.Returns.Error = []error{nil, nil, errors.New("a test error")}

							deploymentResponse := controller.RunDeployment(&deployment, response)

							Expect(reflect.TypeOf(deploymentResponse.Error)).Should(Equal(reflect.TypeOf(bluegreen.FinishDeployError{})))
						})
					})
				})
				Context("deploy.success event", func() {
					It("call Emit", func() {
						deployment.CFContext.Environment = environment
						deployment.Type.ZIP = true

						controller.RunDeployment(&deployment, response)
						Expect(eventManager.EmitCall.Received.Events[1].Type).Should(Equal(constants.DeploySuccessEvent))
					})
					It("calls EmitEvent", func() {
						deployment.CFContext.Environment = environment
						deployment.Type.ZIP = true

						controller.RunDeployment(&deployment, response)

						Expect(eventManager.EmitEventCall.Received.Events[1].Name()).To(Equal(push.DeploySuccessEvent{}.Name()))
					})
					It("passes CFContext to Emit", func() {
						deployment.CFContext.Environment = environment
						deployment.CFContext.Application = appName
						deployment.CFContext.Space = space
						deployment.CFContext.Organization = org
						deployment.Type.ZIP = true

						controller.RunDeployment(&deployment, response)

						deploymentInfo := eventManager.EmitCall.Received.Events[1].Data.(*structs.DeployEventData).DeploymentInfo
						Expect(deploymentInfo.AppName).To(Equal(appName))
						Expect(deploymentInfo.Org).To(Equal(org))
						Expect(deploymentInfo.Space).To(Equal(space))
						Expect(deploymentInfo.UUID).ToNot(BeNil())
					})
					It("passes CFContext to EmitEvent in the event", func() {
						deployment.CFContext.Environment = environment
						deployment.CFContext.Application = appName
						deployment.CFContext.Space = space
						deployment.CFContext.Organization = org

						deployment.Type.ZIP = true

						controller.RunDeployment(&deployment, response)

						event := eventManager.EmitEventCall.Received.Events[1].(push.DeploySuccessEvent)
						Expect(event.CFContext.Environment).To(Equal(environment))
						Expect(event.CFContext.Application).To(Equal(appName))
						Expect(event.CFContext.Space).To(Equal(space))
						Expect(event.CFContext.Organization).To(Equal(org))
					})
					It("passes Auth to EmitEvent in the event", func() {
						deployment.CFContext.Environment = environment
						deployment.Authorization = I.Authorization{
							Username: "myuser",
							Password: "mypassword",
						}

						deployment.Type.ZIP = true

						controller.RunDeployment(&deployment, response)

						event := eventManager.EmitEventCall.Received.Events[1].(push.DeploySuccessEvent)
						Expect(event.Auth.Username).To(Equal("myuser"))
						Expect(event.Auth.Password).To(Equal("mypassword"))
					})
					It("passes other info to EmitEvent", func() {
						deployment.CFContext.Environment = environment

						controller.Config.Environments[environment] = structs.Environment{
							Name: environment,
						}

						deployment.Type.ZIP = true

						controller.RunDeployment(&deployment, response)

						event := eventManager.EmitEventCall.Received.Events[1].(push.DeploySuccessEvent)
						Expect(event.Body).ToNot(BeNil())
						Expect(event.ContentType).To(Equal("ZIP"))
						Expect(event.Environment.Name).To(Equal(environment))
						Expect(event.Response).ToNot(BeNil())
					})
					It("logs emitting an event", func() {
						deployment.CFContext.Environment = environment
						deployment.Type.ZIP = true

						controller.RunDeployment(&deployment, response)
						Eventually(logBuffer).Should(Say("emitting a deploy.success event"))
					})
					Context("when Emit fails", func() {
						It("logs an error", func() {
							deployment.CFContext.Environment = environment
							deployment.Type.ZIP = true

							eventManager.EmitCall.Returns.Error = []error{nil, errors.New("a test error"), nil}

							controller.RunDeployment(&deployment, response)
							Eventually(logBuffer).Should(Say("an error occurred when emitting a deploy.success event"))
						})
					})
					Context("when EmitEvent fails", func() {
						It("returns error", func() {
							deployment.CFContext.Environment = environment
							deployment.Type.ZIP = true

							eventManager.EmitEventCall.Returns.Error = []error{nil, errors.New("a test error")}

							controller.RunDeployment(&deployment, response)

							Eventually(logBuffer).Should(Say("an error occurred when emitting a DeploySuccessEvent"))
						})
					})
				})
				Context("Deploy.failure event", func() {
					It("call emit", func() {
						deployment.CFContext.Environment = environment
						deployment.Type.ZIP = true

						eventManager.EmitCall.Returns.Error = []error{errors.New("a test error"), nil, nil}

						controller.RunDeployment(&deployment, response)
						Expect(eventManager.EmitCall.Received.Events[1].Type).Should(Equal(constants.DeployFailureEvent))
					})
					It("calls EmitEvent", func() {
						deployment.CFContext.Environment = environment
						deployment.Type.ZIP = true

						eventManager.EmitEventCall.Returns.Error = []error{errors.New("a test error"), nil, nil}

						controller.RunDeployment(&deployment, response)

						Expect(eventManager.EmitEventCall.Received.Events[1].Name()).To(Equal(push.DeployFailureEvent{}.Name()))
					})
					It("passes CFContext to Emit", func() {
						deployment.CFContext.Environment = environment
						deployment.CFContext.Application = appName
						deployment.CFContext.Space = space
						deployment.CFContext.Organization = org
						deployment.Type.ZIP = true

						controller.RunDeployment(&deployment, response)

						deploymentInfo := eventManager.EmitCall.Received.Events[1].Data.(*structs.DeployEventData).DeploymentInfo
						Expect(deploymentInfo.AppName).To(Equal(appName))
						Expect(deploymentInfo.Org).To(Equal(org))
						Expect(deploymentInfo.Space).To(Equal(space))
						Expect(deploymentInfo.UUID).ToNot(BeNil())
					})
					It("passes CFContext to EmitEvent in the event", func() {
						deployment.CFContext.Environment = environment
						deployment.CFContext.Application = appName
						deployment.CFContext.Space = space
						deployment.CFContext.Organization = org
						deployment.Type.ZIP = true

						eventManager.EmitEventCall.Returns.Error = []error{errors.New("a test error"), nil, nil}

						controller.RunDeployment(&deployment, response)

						event := eventManager.EmitEventCall.Received.Events[1].(push.DeployFailureEvent)
						Expect(event.CFContext.Environment).To(Equal(environment))
						Expect(event.CFContext.Application).To(Equal(appName))
						Expect(event.CFContext.Space).To(Equal(space))
						Expect(event.CFContext.Organization).To(Equal(org))
					})
					It("passes Auth to EmitEvent in the event", func() {
						deployment.CFContext.Environment = environment
						deployment.Authorization = I.Authorization{
							Username: "myuser",
							Password: "mypassword",
						}
						deployment.Type.ZIP = true

						eventManager.EmitEventCall.Returns.Error = []error{errors.New("a test error"), nil, nil}

						controller.RunDeployment(&deployment, response)

						event := eventManager.EmitEventCall.Received.Events[1].(push.DeployFailureEvent)
						Expect(event.Auth.Username).To(Equal("myuser"))
						Expect(event.Auth.Password).To(Equal("mypassword"))
					})
					It("passes other info to EmitEvent", func() {
						deployment.CFContext.Environment = environment
						controller.Config.Environments[environment] = structs.Environment{
							Name: environment,
						}
						deployment.Type.ZIP = true

						eventManager.EmitEventCall.Returns.Error = []error{errors.New("a test error"), nil, nil}

						controller.RunDeployment(&deployment, response)

						event := eventManager.EmitEventCall.Received.Events[1].(push.DeployFailureEvent)
						Expect(event.Body).ToNot(BeNil())
						Expect(event.ContentType).To(Equal("ZIP"))
						Expect(event.Environment.Name).To(Equal(environment))
						Expect(event.Response).ToNot(BeNil())
					})
					It("logs emitting an event", func() {
						deployment.CFContext.Environment = environment
						deployment.Type.ZIP = true

						eventManager.EmitEventCall.Returns.Error = []error{errors.New("a test error"), nil, nil}

						controller.RunDeployment(&deployment, response)
						Eventually(logBuffer).Should(Say("emitting a deploy.failure event"))
					})
					Context("when Emit fails", func() {
						It("logs an error", func() {
							deployment.CFContext.Environment = environment
							deployment.Type.ZIP = true

							eventManager.EmitCall.Returns.Error = []error{errors.New("a test error"), errors.New("a test error"), nil}

							controller.RunDeployment(&deployment, response)
							Eventually(logBuffer).Should(Say("an error occurred when emitting a deploy.failure event"))
						})
					})
					Context("when EmitEvent fails", func() {
						It("returns error", func() {
							deployment.CFContext.Environment = environment
							deployment.Type.ZIP = true

							eventManager.EmitEventCall.Returns.Error = []error{errors.New("a test error"), errors.New("a test error"), nil}

							controller.RunDeployment(&deployment, response)

							Eventually(logBuffer).Should(Say("an error occurred when emitting a DeployFailureEvent"))
						})
					})
				})

				It("prints found errors to the response", func() {
					deployment.CFContext.Environment = environment
					deployment.Type.ZIP = true

					eventManager.EmitCall.Returns.Error = []error{errors.New("a test error"), nil, nil}

					retError := error_finder.CreateLogMatchedError("a description", []string{"some details"}, "a solution", "a code")
					errorFinder.FindErrorsCall.Returns.Errors = []I.LogMatchedError{retError}

					controller.RunDeployment(&deployment, response)
					responseBytes, _ := ioutil.ReadAll(response)
					Eventually(string(responseBytes)).Should(ContainSubstring("The following error was found in the above logs: a description"))
					Eventually(string(responseBytes)).Should(ContainSubstring("Error: some details"))
					Eventually(string(responseBytes)).Should(ContainSubstring("Potential solution: a solution"))
				})
			})
		})

	})

})
