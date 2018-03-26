package controller_test

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"

	"io/ioutil"

	"os"

	"github.com/compozed/deployadactyl/config"
	"github.com/compozed/deployadactyl/constants"
	. "github.com/compozed/deployadactyl/controller"
	D "github.com/compozed/deployadactyl/controller/deployer"
	"github.com/compozed/deployadactyl/controller/deployer/bluegreen"
	I "github.com/compozed/deployadactyl/interfaces"
	"github.com/compozed/deployadactyl/logger"
	"github.com/compozed/deployadactyl/mocks"
	"github.com/compozed/deployadactyl/randomizer"
	"github.com/compozed/deployadactyl/structs"
	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/op/go-logging"
	"reflect"
)

var _ = Describe("Controller", func() {

	var (
		deployer             *mocks.Deployer
		silentDeployer       *mocks.Deployer
		pusherCreator        *mocks.PusherCreator
		pusherCreatorFactory *mocks.PusherCreatorFactory
		controller           *Controller
		router               *gin.Engine
		resp                 *httptest.ResponseRecorder
		jsonBuffer           *bytes.Buffer
		deployment           I.Deployment
		eventManager         *mocks.EventManager

		foundationURL string
		appName       string
		environment   string
		org           string
		space         string
		byteBody      []byte
		server        *httptest.Server
		response      *bytes.Buffer
	)

	BeforeEach(func() {
		appName = "appName-" + randomizer.StringRunes(10)
		environment = "environment-" + randomizer.StringRunes(10)
		org = "org-" + randomizer.StringRunes(10)
		space = "non-prod"

		eventManager = &mocks.EventManager{}
		deployer = &mocks.Deployer{}
		silentDeployer = &mocks.Deployer{}
		pusherCreatorFactory = &mocks.PusherCreatorFactory{}
		controller = &Controller{
			Deployer:             deployer,
			SilentDeployer:       silentDeployer,
			Log:                  logger.DefaultLogger(GinkgoWriter, logging.DEBUG, "api_test"),
			PusherCreatorFactory: pusherCreatorFactory,
			EventManager:         eventManager,
			Config:               config.Config{},
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
		pusherCreatorFactory.PusherCreatorCall.Returns.ActionCreator = pusherCreator
	})

	AfterEach(func() {
		server.Close()
	})

	Describe("RunDeploymentViaHttp handler", func() {
		BeforeEach(func() {
			router = gin.New()
			resp = httptest.NewRecorder()
			jsonBuffer = &bytes.Buffer{}

			router.POST("/v2/deploy/:environment/:org/:space/:appName", controller.RunDeploymentViaHttp)

			server = httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
				byteBody, _ = ioutil.ReadAll(req.Body)
				req.Body.Close()
			}))

			silentDeployUrl := server.URL + "/v1/apps/" + os.Getenv("SILENT_DEPLOY_ENVIRONMENT")
			os.Setenv("SILENT_DEPLOY_URL", silentDeployUrl)
		})
		Context("when deployer succeeds", func() {
			It("deploys and returns http.StatusOK", func() {
				foundationURL = fmt.Sprintf("/v2/deploy/%s/%s/%s/%s", environment, org, space, appName)

				req, err := http.NewRequest("POST", foundationURL, jsonBuffer)
				Expect(err).ToNot(HaveOccurred())

				deployer.DeployCall.Returns.Error = nil
				deployer.DeployCall.Returns.StatusCode = http.StatusOK
				deployer.DeployCall.Write.Output = "deploy success"

				router.ServeHTTP(resp, req)

				Eventually(resp.Code).Should(Equal(http.StatusOK))
				Eventually(resp.Body).Should(ContainSubstring("deploy success"))

				Eventually(deployer.DeployCall.Received.Environment).Should(Equal(environment))
				Eventually(deployer.DeployCall.Received.Org).Should(Equal(org))
				Eventually(deployer.DeployCall.Received.Space).Should(Equal(space))
				Eventually(deployer.DeployCall.Received.AppName).Should(Equal(appName))
			})

			It("does not run silent deploy when environment other than non-prop", func() {
				foundationURL = fmt.Sprintf("/v2/deploy/%s/%s/%s/%s", environment, org, "not-non-prod", appName)

				req, err := http.NewRequest("POST", foundationURL, jsonBuffer)
				Expect(err).ToNot(HaveOccurred())

				deployer.DeployCall.Returns.Error = nil
				deployer.DeployCall.Returns.StatusCode = http.StatusOK
				deployer.DeployCall.Write.Output = "deploy success"

				router.ServeHTTP(resp, req)

				Eventually(resp.Code).Should(Equal(http.StatusOK))
				Eventually(resp.Body).Should(ContainSubstring("deploy success"))

				Eventually(len(byteBody)).Should(Equal(0))
			})
		})

		Context("when deployer fails", func() {
			It("doesn't deploy and gives http.StatusInternalServerError", func() {
				foundationURL = fmt.Sprintf("/v2/deploy/%s/%s/%s/%s", environment, org, space, appName)

				req, err := http.NewRequest("POST", foundationURL, jsonBuffer)
				Expect(err).ToNot(HaveOccurred())

				deployer.DeployCall.Returns.Error = errors.New("bork")
				deployer.DeployCall.Returns.StatusCode = http.StatusInternalServerError

				router.ServeHTTP(resp, req)

				Eventually(resp.Code).Should(Equal(http.StatusInternalServerError))
				Eventually(resp.Body).Should(ContainSubstring("bork"))
			})
		})

		Context("when parameters are added to the url", func() {
			It("does not return an error", func() {
				foundationURL = fmt.Sprintf("/v2/deploy/%s/%s/%s/%s?broken=false", environment, org, space, appName)

				req, err := http.NewRequest("POST", foundationURL, jsonBuffer)
				Expect(err).ToNot(HaveOccurred())

				deployer.DeployCall.Write.Output = "deploy success"
				deployer.DeployCall.Returns.StatusCode = http.StatusOK

				router.ServeHTTP(resp, req)

				Eventually(resp.Code).Should(Equal(http.StatusOK))
				Eventually(resp.Body).Should(ContainSubstring("deploy success"))
			})
		})
	})

	Describe("RunDeployment", func() {
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
				deployResponse := controller.RunDeployment(deployment, response)

				Eventually(deployer.DeployCall.Called).Should(Equal(1))
				Eventually(silentDeployer.DeployCall.Called).Should(Equal(0))

				Eventually(deployResponse.StatusCode).Should(Equal(http.StatusOK))

				Eventually(deployer.DeployCall.Received.Authorization).Should(Equal(deployment.Authorization))
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
				deployResponse := controller.RunDeployment(deployment, response)
				receivedBody, _ := ioutil.ReadAll(deployer.DeployCall.Received.Body)
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

				deployer.DeployCall.Returns.Error = nil
				deployer.DeployCall.Returns.StatusCode = http.StatusOK
				deployer.DeployCall.Write.Output = "little-timmy-env.zip"

				deployResponse := controller.RunDeployment(&deployment, response)

				Eventually(deployer.DeployCall.Called).Should(Equal(1))
				Eventually(silentDeployer.DeployCall.Called).Should(Equal(0))

				Eventually(deployResponse.StatusCode).Should(Equal(http.StatusOK))

				Eventually(deployer.DeployCall.Received.Environment).Should(Equal(environment))
				Eventually(deployer.DeployCall.Received.ContentType).Should(Equal(I.DeploymentType{JSON: false}))
				Eventually(deployer.DeployCall.Received.Org).Should(Equal(org))
				Eventually(deployer.DeployCall.Received.Space).Should(Equal(space))
				Eventually(deployer.DeployCall.Received.AppName).Should(Equal(appName))

				ret, _ := ioutil.ReadAll(response)
				Eventually(string(ret)).Should(Equal("little-timmy-env.zip"))
			})

			It("channel resolves when errors occur", func() {
				deployment.CFContext.Environment = environment
				deployment.CFContext.Organization = org
				deployment.CFContext.Space = space
				deployment.CFContext.Application = appName

				deployer.DeployCall.Returns.Error = errors.New("bork")
				deployer.DeployCall.Returns.StatusCode = http.StatusInternalServerError
				deployer.DeployCall.Write.Output = "little-timmy-env.zip"

				deployResponse := controller.RunDeployment(&deployment, response)

				Eventually(deployer.DeployCall.Called).Should(Equal(1))
				Eventually(silentDeployer.DeployCall.Called).Should(Equal(0))

				Eventually(deployResponse.StatusCode).Should(Equal(http.StatusInternalServerError))
				Eventually(deployResponse.Error.Error()).Should(Equal("bork"))

				Eventually(deployer.DeployCall.Received.Environment).Should(Equal(environment))
				Eventually(deployer.DeployCall.Received.ContentType).Should(Equal(I.DeploymentType{JSON: false}))
				Eventually(deployer.DeployCall.Received.Org).Should(Equal(org))
				Eventually(deployer.DeployCall.Received.Space).Should(Equal(space))
				Eventually(deployer.DeployCall.Received.AppName).Should(Equal(appName))

				ret, _ := ioutil.ReadAll(response)
				Eventually(string(ret)).Should(Equal("little-timmy-env.zip"))
			})

			It("does not set the basic auth header if no credentials are passed", func() {
				deployer.DeployCall.Write.Output = "little-timmy-env.zip"

				response := &bytes.Buffer{}

				deployment := &I.Deployment{
					Body: &[]byte{},
					Type: I.DeploymentType{JSON: true},
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

				Eventually(deployer.DeployCall.Received.Authorization.Username).Should(Equal(""))
				Eventually(deployer.DeployCall.Received.Authorization.Password).Should(Equal(""))

			})

			It("sets the basic auth header if credentials are passed", func() {
				deployment.CFContext.Environment = environment
				deployer.DeployCall.Write.Output = "little-timmy-env.zip"

				deployment.Authorization = I.Authorization{
					Username: "TestUsername",
					Password: "TestPassword",
				}

				controller.RunDeployment(&deployment, response)

				Eventually(deployer.DeployCall.Received.Authorization.Username).Should(Equal("TestUsername"))
				Eventually(deployer.DeployCall.Received.Authorization.Password).Should(Equal("TestPassword"))
			})
		})

		Context("when SILENT_DEPLOY_ENVIRONMENT is true", func() {
			It("channel resolves true when no errors occur", func() {
				deployment.CFContext.Environment = environment
				deployment.CFContext.Organization = org
				deployment.CFContext.Space = space
				deployment.CFContext.Application = appName

				os.Setenv("SILENT_DEPLOY_ENVIRONMENT", environment)
				deployer.DeployCall.Returns.Error = nil
				deployer.DeployCall.Returns.StatusCode = http.StatusOK
				deployer.DeployCall.Write.Output = "little-timmy-env.zip"

				deployResponse := controller.RunDeployment(&deployment, response)

				Eventually(deployer.DeployCall.Called).Should(Equal(1))
				Eventually(silentDeployer.DeployCall.Called).Should(Equal(1))

				Eventually(deployResponse.StatusCode).Should(Equal(http.StatusOK))

				Eventually(deployer.DeployCall.Received.Environment).Should(Equal(environment))
				Eventually(deployer.DeployCall.Received.ContentType).Should(Equal(I.DeploymentType{JSON: false}))
				Eventually(deployer.DeployCall.Received.Org).Should(Equal(org))
				Eventually(deployer.DeployCall.Received.Space).Should(Equal(space))
				Eventually(deployer.DeployCall.Received.AppName).Should(Equal(appName))

				ret, _ := ioutil.ReadAll(response)
				Eventually(string(ret)).Should(Equal("little-timmy-env.zip"))
			})
			It("channel resolves when no errors occur", func() {
				deployment.CFContext.Environment = environment
				deployment.CFContext.Organization = org
				deployment.CFContext.Space = space
				deployment.CFContext.Application = appName

				os.Setenv("SILENT_DEPLOY_ENVIRONMENT", environment)
				deployer.DeployCall.Returns.Error = nil
				deployer.DeployCall.Returns.StatusCode = http.StatusOK
				deployer.DeployCall.Write.Output = "little-timmy-env.zip"

				silentDeployer.DeployCall.Returns.Error = errors.New("bork")
				silentDeployer.DeployCall.Returns.StatusCode = http.StatusInternalServerError

				silentDeployUrl := server.URL + "/v1/apps/" + os.Getenv("SILENT_DEPLOY_ENVIRONMENT")
				os.Setenv("SILENT_DEPLOY_URL", silentDeployUrl)

				deployResponse := controller.RunDeployment(&deployment, response)

				Eventually(deployer.DeployCall.Called).Should(Equal(1))
				Eventually(silentDeployer.DeployCall.Called).Should(Equal(1))

				Eventually(deployResponse.StatusCode).Should(Equal(http.StatusOK))

				Eventually(deployer.DeployCall.Received.Environment).Should(Equal(environment))
				Eventually(deployer.DeployCall.Received.ContentType).Should(Equal(I.DeploymentType{JSON: false}))
				Eventually(deployer.DeployCall.Received.Org).Should(Equal(org))
				Eventually(deployer.DeployCall.Received.Space).Should(Equal(space))
				Eventually(deployer.DeployCall.Received.AppName).Should(Equal(appName))

				ret, _ := ioutil.ReadAll(response)
				Eventually(string(ret)).Should(Equal("little-timmy-env.zip"))
			})
		})

		Context("when called", func() {
			It("creates a pusher creator", func() {
				deployment.CFContext.Environment = environment

				environments := map[string]structs.Environment{}
				environments[environment] = structs.Environment{}
				controller.Config.Environments = environments

				controller.RunDeployment(&deployment, response)
				Eventually(pusherCreatorFactory.PusherCreatorCall.Called).Should(Equal(true))

			})
			It("Provides body for pusher creator", func() {
				bodyByte := []byte("body string")
				deployment.CFContext.Environment = environment
				deployment.Body = &bodyByte

				environments := map[string]structs.Environment{}
				environments[environment] = structs.Environment{}
				controller.Config.Environments = environments

				controller.RunDeployment(&deployment, response)
				returnedBody, _ := ioutil.ReadAll(pusherCreatorFactory.PusherCreatorCall.Received.DeployEventData.RequestBody)
				Eventually(returnedBody).Should(Equal(bodyByte))
			})
			It("Provides response for pusher creator", func() {
				deployment.CFContext.Environment = environment

				environments := map[string]structs.Environment{}
				environments[environment] = structs.Environment{}
				controller.Config.Environments = environments
				response = bytes.NewBuffer([]byte("hello"))

				controller.RunDeployment(&deployment, response)
				returnedResponse, _ := ioutil.ReadAll(pusherCreatorFactory.PusherCreatorCall.Received.DeployEventData.Response)
				Eventually(returnedResponse).Should(Equal([]byte("hello")))
			})
			Context("the deployment info", func() {
				Context("when environment does not exist", func() {
					It("returns an error with StatusInternalServerError", func() {
						deployment.CFContext.Environment = "bad env"

						environments := map[string]structs.Environment{}
						controller.Config.Environments = environments

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

								environments := map[string]structs.Environment{}
								environments[environment] = structs.Environment{}
								controller.Config.Environments = environments

								deployment.Authorization.Username = ""
								deployment.Authorization.Password = ""
								controller.Config.Username = "username-" + randomizer.StringRunes(10)
								controller.Config.Password = "password-" + randomizer.StringRunes(10)

								controller.RunDeployment(&deployment, response)

								Eventually(pusherCreatorFactory.PusherCreatorCall.Received.DeployEventData.DeploymentInfo.Username).Should(Equal(controller.Config.Username))
								Eventually(pusherCreatorFactory.PusherCreatorCall.Received.DeployEventData.DeploymentInfo.Password).Should(Equal(controller.Config.Password))
							})
						})
						Context("and authentication is required", func() {
							It("returns an error", func() {
								deployment.CFContext.Environment = environment

								deployment.Authorization.Username = ""
								deployment.Authorization.Password = ""

								environments := map[string]structs.Environment{}
								environments[environment] = structs.Environment{
									Authenticate: true,
								}
								controller.Config.Environments = environments

								deploymentResponse := controller.RunDeployment(&deployment, response)

								Eventually(deploymentResponse.Error).Should(HaveOccurred())
								Eventually(deploymentResponse.Error.Error()).Should(Equal("basic auth header not found"))
							})
						})
					})
					Context("when Authorization has values", func() {
						It("returns username and password from the authorization", func() {
							deployment.CFContext.Environment = environment

							environments := map[string]structs.Environment{}
							environments[environment] = structs.Environment{}
							controller.Config.Environments = environments

							deployment.Authorization.Username = "username-" + randomizer.StringRunes(10)
							deployment.Authorization.Password = "password-" + randomizer.StringRunes(10)

							controller.RunDeployment(&deployment, response)

							Eventually(pusherCreatorFactory.PusherCreatorCall.Received.DeployEventData.DeploymentInfo.Username).Should(Equal(deployment.Authorization.Username))
							Eventually(pusherCreatorFactory.PusherCreatorCall.Received.DeployEventData.DeploymentInfo.Password).Should(Equal(deployment.Authorization.Password))
						})
					})
					It("has the correct org, space ,appname, env, uuid", func() {
						deployment.CFContext.Environment = environment

						environments := map[string]structs.Environment{}
						environments[environment] = structs.Environment{}
						controller.Config.Environments = environments

						deployment.CFContext.Organization = org
						deployment.CFContext.Space = space
						deployment.CFContext.Application = appName
						deployment.CFContext.Environment = environment

						controller.RunDeployment(&deployment, response)

						Eventually(pusherCreatorFactory.PusherCreatorCall.Received.DeployEventData.DeploymentInfo.Org).Should(Equal(org))
						Eventually(pusherCreatorFactory.PusherCreatorCall.Received.DeployEventData.DeploymentInfo.Space).Should(Equal(space))
						Eventually(pusherCreatorFactory.PusherCreatorCall.Received.DeployEventData.DeploymentInfo.AppName).Should(Equal(appName))
						Eventually(pusherCreatorFactory.PusherCreatorCall.Received.DeployEventData.DeploymentInfo.Environment).Should(Equal(environment))

					})

					Context("when uuid is not provided", func() {
						It("creates a new uuid", func() {
							deployment.CFContext.Environment = environment

							environments := map[string]structs.Environment{}
							environments[environment] = structs.Environment{}
							controller.Config.Environments = environments

							controller.RunDeployment(&deployment, response)

							Eventually(pusherCreatorFactory.PusherCreatorCall.Received.DeployEventData.DeploymentInfo.UUID).ShouldNot(BeEmpty())

						})
					})
					Context("when uuid is provided", func() {
						It("uses the provided uuid", func() {
							deployment.CFContext.Environment = environment
							uuid := randomizer.StringRunes(10)
							deployment.CFContext.UUID = uuid

							environments := map[string]structs.Environment{}
							environments[environment] = structs.Environment{}
							controller.Config.Environments = environments

							controller.RunDeployment(&deployment, response)

							Eventually(pusherCreatorFactory.PusherCreatorCall.Received.DeployEventData.DeploymentInfo.UUID).Should(Equal(uuid))

						})
					})
					It("has the correct domain and skipssl", func() {
						deployment.CFContext.Environment = environment
						domain := "domain-" + randomizer.StringRunes(10)
						deployment.Authorization.Username = ""
						deployment.Authorization.Password = ""

						environments := map[string]structs.Environment{}
						environments[environment] = structs.Environment{
							Domain:  domain,
							SkipSSL: true,
						}
						controller.Config.Environments = environments

						controller.RunDeployment(&deployment, response)

						Eventually(pusherCreatorFactory.PusherCreatorCall.Received.DeployEventData.DeploymentInfo.Domain).Should(Equal(domain))
						Eventually(pusherCreatorFactory.PusherCreatorCall.Received.DeployEventData.DeploymentInfo.SkipSSL).Should(BeTrue())
					})
					It("has correct custom parameters", func() {

						customParams := make(map[string]interface{})
						customParams["param1"] = "value1"
						customParams["param2"] = "value2"

						deployment.CFContext.Environment = environment

						environments := map[string]structs.Environment{}
						environments[environment] = structs.Environment{
							CustomParams: customParams,
						}
						controller.Config.Environments = environments

						controller.RunDeployment(&deployment, response)

						Eventually(pusherCreatorFactory.PusherCreatorCall.Received.DeployEventData.DeploymentInfo.CustomParams["param1"]).Should(Equal("value1"))
						Eventually(pusherCreatorFactory.PusherCreatorCall.Received.DeployEventData.DeploymentInfo.CustomParams["param2"]).Should(Equal("value2"))

					})
					It("is passed to the pusher creator", func() {
						deployment.CFContext.Environment = environment

						environments := map[string]structs.Environment{}
						environments[environment] = structs.Environment{}
						controller.Config.Environments = environments

						controller.RunDeployment(&deployment, response)

						Eventually(pusherCreatorFactory.PusherCreatorCall.Received.DeployEventData.DeploymentInfo).ShouldNot(BeNil())
					})

					It("correctly extracts artifact url from body", func() {
						artifactURL := "artifactURL-" + randomizer.StringRunes(10)
						bodyByte := []byte(fmt.Sprintf(`{"artifact_url": "%s"}`, artifactURL))

						deployment.CFContext.Environment = environment
						deployment.Body = &bodyByte
						deployment.Type.JSON = true

						environments := map[string]structs.Environment{}
						environments[environment] = structs.Environment{}
						controller.Config.Environments = environments

						controller.RunDeployment(&deployment, response)

						Eventually(pusherCreatorFactory.PusherCreatorCall.Received.DeployEventData.DeploymentInfo.ArtifactURL).Should(Equal(artifactURL))
					})
					Context("if artifact url isn't provided in body", func() {
						It("returns an error", func() {
							bodyByte := []byte("{}")

							deployment.CFContext.Environment = environment
							deployment.Body = &bodyByte
							deployment.Type.JSON = true
							environments := map[string]structs.Environment{}
							environments[environment] = structs.Environment{}
							controller.Config.Environments = environments

							deploymentResponse := controller.RunDeployment(&deployment, response)

							Eventually(deploymentResponse.Error).ShouldNot(BeNil())
							Eventually(deploymentResponse.Error.Error()).Should(ContainSubstring("The following properties are missing"))
						})
					})
					Context("if body is invalid", func() {
						It("returns an error", func() {
							bodyByte := []byte("")

							deployment.CFContext.Environment = environment
							deployment.Body = &bodyByte
							deployment.Type.JSON = true
							environments := map[string]structs.Environment{}
							environments[environment] = structs.Environment{}
							controller.Config.Environments = environments

							deploymentResponse := controller.RunDeployment(&deployment, response)

							Eventually(deploymentResponse.Error).ShouldNot(BeNil())
							Eventually(deploymentResponse.Error.Error()).Should(ContainSubstring("EOF"))
						})
					})
					It("emits a start event", func() {
						deployment.CFContext.Environment = environment

						environments := map[string]structs.Environment{}
						environments[environment] = structs.Environment{}

						controller.Config.Environments = environments

						controller.RunDeployment(&deployment, response)

						Expect(eventManager.EmitCall.Received.Events[0].Type).Should(Equal(constants.DeployStartEvent))
					})
					Context("when start emit fails", func() {
						It("returns error", func() {
							deployment.CFContext.Environment = environment
							eventManager.EmitCall.Returns.Error = []error{errors.New("a test error")}

							environments := map[string]structs.Environment{}
							environments[environment] = structs.Environment{}

							controller.Config.Environments = environments

							deploymentResponse := controller.RunDeployment(&deployment, response)

							Expect(reflect.TypeOf(deploymentResponse.Error)).Should(Equal(reflect.TypeOf(D.EventError{})))
						})
					})
					It("emits a finished event", func() {
						deployment.CFContext.Environment = environment

						environments := map[string]structs.Environment{}
						environments[environment] = structs.Environment{}

						controller.Config.Environments = environments

						controller.RunDeployment(&deployment, response)
						//
						Expect(eventManager.EmitCall.Received.Events[1].Type).Should(Equal(constants.DeployFinishEvent))
					})
					Context("when finished emit fails", func() {
						It("returns error", func() {
							deployment.CFContext.Environment = environment
							eventManager.EmitCall.Returns.Error = []error{nil, errors.New("a test error")}

							environments := map[string]structs.Environment{}
							environments[environment] = structs.Environment{}

							controller.Config.Environments = environments

							deploymentResponse := controller.RunDeployment(&deployment, response)

							Expect(reflect.TypeOf(deploymentResponse.Error)).Should(Equal(reflect.TypeOf(bluegreen.FinishDeployError{})))
						})
					})
					It("emits a success event", func() {
						deployment.CFContext.Environment = environment

						environments := map[string]structs.Environment{}
						environments[environment] = structs.Environment{}

						controller.Config.Environments = environments

						controller.RunDeployment(&deployment, response)
						//
						Expect(eventManager.EmitCall.Received.Events[1].Type).Should(Equal(constants.DeployFinishEvent))
					})
					//Context("when finished emit fails", func() {
					//	FIt("returns error", func() {
					//		deployment.CFContext.Environment = environment
					//		eventManager.EmitCall.Returns.Error = []error{nil, errors.New("a test error")}
					//
					//		environments := map[string]structs.Environment{}
					//		environments[environment] = structs.Environment{}
					//
					//		controller.Config.Environments = environments
					//
					//		deploymentResponse := controller.RunDeployment(&deployment, response)
					//
					//		Expect(reflect.TypeOf(deploymentResponse.Error)).Should(Equal(reflect.TypeOf(bluegreen.FinishDeployError{})))
					//	})
					//})
				})

				//FIt("has the correct username and password", func() {
				//	//environment = "environment-" + randomizer.StringRunes(10)
				//
				//	//environments                 := map[string]structs.Environment {}
				//	//domain := "domain-" + randomizer.StringRunes(10)
				//	//
				//	//environments[environment] = structs.Environment{
				//	//	Name:           environment,
				//	//	Domain:         domain,
				//	//	//Foundations:    foundations,
				//	//	//Instances:      instances,
				//	//	//CustomParams:   customParams,
				//	//	//EnableRollback: enableRollback,
				//	//}
				//	//c := config.Config{
				//	//	Environments: environments,
				//	//}
				//
				//
				//	//deploymentInfo.Manifest = manifest
				//	//deploymentInfo.AppPath = appPath
				//	//deploymentInfo.Instances = instances
				//
				//
				//	Eventually(pusherCreatorFactory.PusherCreatorCall.Received.DeploymentInfo.Username).Should(Equal(deployment.Authorization.Username))
				//	Eventually(pusherCreatorFactory.PusherCreatorCall.Received.DeploymentInfo.Password).Should(Equal(deployment.Authorization.Password))
				//
				//})
			})

		})

	})
	Describe("StopDeployment", func() {
		Context("when verbose deployer is called", func() {
			It("channel resolves when no errors occur", func() {
				deployment.CFContext.Environment = environment
				deployment.CFContext.Organization = org
				deployment.CFContext.Space = space
				deployment.CFContext.Application = appName

				deployer.DeployCall.Returns.Error = nil
				deployer.DeployCall.Returns.StatusCode = http.StatusOK
				deployer.DeployCall.Write.Output = "little-timmy-env.zip"

				deployResponse := controller.RunDeployment(&deployment, response)

				Eventually(deployer.DeployCall.Called).Should(Equal(1))
				Eventually(silentDeployer.DeployCall.Called).Should(Equal(0))

				Eventually(deployResponse.StatusCode).Should(Equal(http.StatusOK))

				Eventually(deployer.DeployCall.Received.Environment).Should(Equal(environment))
				Eventually(deployer.DeployCall.Received.ContentType).Should(Equal(I.DeploymentType{JSON: false}))
				Eventually(deployer.DeployCall.Received.Org).Should(Equal(deployment.CFContext.Organization))
				Eventually(deployer.DeployCall.Received.Space).Should(Equal(space))
				Eventually(deployer.DeployCall.Received.AppName).Should(Equal(appName))

				ret, _ := ioutil.ReadAll(response)
				Eventually(string(ret)).Should(Equal("little-timmy-env.zip"))
			})

			It("channel resolves when errors occur", func() {
				deployment.CFContext.Environment = environment
				deployment.CFContext.Organization = org
				deployment.CFContext.Space = space
				deployment.CFContext.Application = appName

				deployer.DeployCall.Returns.Error = errors.New("bork")
				deployer.DeployCall.Returns.StatusCode = http.StatusInternalServerError
				deployer.DeployCall.Write.Output = "little-timmy-env.zip"

				deployResponse := controller.RunDeployment(&deployment, response)

				Eventually(deployer.DeployCall.Called).Should(Equal(1))
				Eventually(silentDeployer.DeployCall.Called).Should(Equal(0))

				Eventually(deployResponse.StatusCode).Should(Equal(http.StatusInternalServerError))
				Eventually(deployResponse.Error.Error()).Should(Equal("bork"))

				Eventually(deployer.DeployCall.Received.Environment).Should(Equal(environment))
				Eventually(deployer.DeployCall.Received.ContentType).Should(Equal(I.DeploymentType{JSON: false}))
				Eventually(deployer.DeployCall.Received.Org).Should(Equal(org))
				Eventually(deployer.DeployCall.Received.Space).Should(Equal(space))
				Eventually(deployer.DeployCall.Received.AppName).Should(Equal(appName))

				ret, _ := ioutil.ReadAll(response)
				Eventually(string(ret)).Should(Equal("little-timmy-env.zip"))
			})

			It("does not set the basic auth header if no credentials are passed", func() {
				deployer.DeployCall.Write.Output = "little-timmy-env.zip"

				response := &bytes.Buffer{}

				deployment := &I.Deployment{
					Body: &[]byte{},
					Type: I.DeploymentType{JSON: true},
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

				Eventually(deployer.DeployCall.Received.Authorization.Username).Should(Equal(""))
				Eventually(deployer.DeployCall.Received.Authorization.Password).Should(Equal(""))
			})

			It("sets the basic auth header if credentials are passed", func() {
				deployment.CFContext.Environment = environment

				deployer.DeployCall.Write.Output = "little-timmy-env.zip"

				deployment.Authorization = I.Authorization{
					Username: "TestUsername",
					Password: "TestPassword",
				}
				controller.RunDeployment(&deployment, response)

				Eventually(deployer.DeployCall.Received.Authorization.Username).Should(Equal("TestUsername"))
				Eventually(deployer.DeployCall.Received.Authorization.Password).Should(Equal("TestPassword"))
			})
		})

		Context("when SILENT_DEPLOY_ENVIRONMENT is true", func() {
			It("channel resolves true when no errors occur", func() {
				deployment.CFContext.Environment = environment
				deployment.CFContext.Organization = org
				deployment.CFContext.Space = space
				deployment.CFContext.Application = appName

				os.Setenv("SILENT_DEPLOY_ENVIRONMENT", environment)
				deployer.DeployCall.Returns.Error = nil
				deployer.DeployCall.Returns.StatusCode = http.StatusOK
				deployer.DeployCall.Write.Output = "little-timmy-env.zip"

				deployResponse := controller.RunDeployment(&deployment, response)

				Eventually(deployer.DeployCall.Called).Should(Equal(1))
				Eventually(silentDeployer.DeployCall.Called).Should(Equal(1))

				Eventually(deployResponse.StatusCode).Should(Equal(http.StatusOK))

				Eventually(deployer.DeployCall.Received.Environment).Should(Equal(environment))
				Eventually(deployer.DeployCall.Received.ContentType).Should(Equal(I.DeploymentType{JSON: false}))
				Eventually(deployer.DeployCall.Received.Org).Should(Equal(org))
				Eventually(deployer.DeployCall.Received.Space).Should(Equal(space))
				Eventually(deployer.DeployCall.Received.AppName).Should(Equal(appName))

				ret, _ := ioutil.ReadAll(response)
				Eventually(string(ret)).Should(Equal("little-timmy-env.zip"))
			})
			It("channel resolves when no errors occur", func() {
				deployment.CFContext.Environment = environment
				deployment.CFContext.Space = space
				deployment.CFContext.Application = appName

				os.Setenv("SILENT_DEPLOY_ENVIRONMENT", environment)
				deployer.DeployCall.Returns.Error = nil
				deployer.DeployCall.Returns.StatusCode = http.StatusOK
				deployer.DeployCall.Write.Output = "little-timmy-env.zip"

				silentDeployer.DeployCall.Returns.Error = errors.New("bork")
				silentDeployer.DeployCall.Returns.StatusCode = http.StatusInternalServerError

				silentDeployUrl := server.URL + "/v1/apps/" + os.Getenv("SILENT_DEPLOY_ENVIRONMENT")
				os.Setenv("SILENT_DEPLOY_URL", silentDeployUrl)

				deployResponse := controller.RunDeployment(&deployment, response)

				Eventually(deployer.DeployCall.Called).Should(Equal(1))
				Eventually(silentDeployer.DeployCall.Called).Should(Equal(1))

				Eventually(deployResponse.StatusCode).Should(Equal(http.StatusOK))

				Eventually(deployer.DeployCall.Received.Environment).Should(Equal(environment))
				Eventually(deployer.DeployCall.Received.ContentType).Should(Equal(I.DeploymentType{JSON: false}))
				Eventually(deployer.DeployCall.Received.Org).Should(Equal(deployment.CFContext.Organization))
				Eventually(deployer.DeployCall.Received.Space).Should(Equal(space))
				Eventually(deployer.DeployCall.Received.AppName).Should(Equal(appName))

				ret, _ := ioutil.ReadAll(response)
				Eventually(string(ret)).Should(Equal("little-timmy-env.zip"))
			})
		})
	})
})
