package deployer_test

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"math/rand"
	"net/http"

	"github.com/compozed/deployadactyl/config"
	. "github.com/compozed/deployadactyl/controller/deployer"
	"github.com/compozed/deployadactyl/logger"
	"github.com/compozed/deployadactyl/mocks"
	"github.com/compozed/deployadactyl/randomizer"
	S "github.com/compozed/deployadactyl/structs"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	"github.com/op/go-logging"
	"github.com/spf13/afero"
)

const (
	testManifest = `---
applications:
- name: deployadactyl
  memory: 256M
  disk_quota: 256M
`
	eventManagerNotEnoughCalls = "event manager didn't have the right number of calls"
)

var _ = Describe("Deployer", func() {
	var (
		deployer Deployer

		c              config.Config
		blueGreener    *mocks.BlueGreener
		fetcher        *mocks.Fetcher
		prechecker     *mocks.Prechecker
		eventManager   *mocks.EventManager
		randomizerMock *mocks.Randomizer

		req                  *http.Request
		requestBody          *bytes.Buffer
		appName              string
		appPath              string
		artifactURL          string
		domain               string
		environment          string
		org                  string
		space                string
		username             string
		uuid                 string
		instances            uint16
		password             string
		testManifestLocation string
		buffer               *bytes.Buffer
		logBuffer            = NewBuffer()

		deploymentInfo S.DeploymentInfo
		foundations    []string
		environments   = map[string]config.Environment{}
		log            = logger.DefaultLogger(logBuffer, logging.DEBUG, "deployer tests")
		af             *afero.Afero
	)

	BeforeEach(func() {
		blueGreener = &mocks.BlueGreener{}
		fetcher = &mocks.Fetcher{}
		prechecker = &mocks.Prechecker{}
		eventManager = &mocks.EventManager{}
		randomizerMock = &mocks.Randomizer{}

		appName = "appName-" + randomizer.StringRunes(10)
		appPath = "appPath-" + randomizer.StringRunes(10)
		artifactURL = "artifactURL-" + randomizer.StringRunes(10)
		domain = "domain-" + randomizer.StringRunes(10)
		environment = "environment-" + randomizer.StringRunes(10)
		org = "org-" + randomizer.StringRunes(10)
		password = "password-" + randomizer.StringRunes(10)
		space = "space-" + randomizer.StringRunes(10)
		username = "username-" + randomizer.StringRunes(10)
		uuid = "uuid-" + randomizer.StringRunes(10)
		instances = uint16(rand.Uint32())

		randomizerMock.RandomizeCall.Returns.Runes = uuid
		eventManager.EmitCall.Returns.Error = append(eventManager.EmitCall.Returns.Error, nil)
		eventManager.EmitCall.Returns.Error = append(eventManager.EmitCall.Returns.Error, nil)
		eventManager.EmitCall.Returns.Error = append(eventManager.EmitCall.Returns.Error, nil)
		eventManager.EmitCall.Returns.Error = append(eventManager.EmitCall.Returns.Error, nil)

		requestBody = bytes.NewBufferString(fmt.Sprintf(`{
		  		"artifact_url": "%s"
		  	}`,
			artifactURL,
		))

		req, _ = http.NewRequest("POST", "", requestBody)

		deploymentInfo = S.DeploymentInfo{
			ArtifactURL: artifactURL,
			Username:    username,
			Password:    password,
			Environment: environment,
			Org:         org,
			Space:       space,
			AppName:     appName,
			UUID:        uuid,
			Instances:   instances,
			Manifest:    "",
		}

		foundations = []string{randomizer.StringRunes(10)}
		buffer = &bytes.Buffer{}

		environments = map[string]config.Environment{}
		environments[environment] = config.Environment{
			Name:        environment,
			Domain:      domain,
			Foundations: foundations,
			Instances:   instances,
		}

		c = config.Config{
			Username:     username,
			Password:     password,
			Environments: environments,
		}

		af = &afero.Afero{Fs: afero.NewMemMapFs()}

		testManifestLocation, _ = af.TempDir("", "")

		deployer = Deployer{
			c,
			blueGreener,
			fetcher,
			prechecker,
			eventManager,
			randomizerMock,
			log,
			af,
		}
	})

	Describe("prechecking the environments", func() {
		Context("when Prechecker fails", func() {
			It("rejects the request with a http.StatusInternalServerError", func() {
				prechecker.AssertAllFoundationsUpCall.Returns.Error = errors.New("prechecker failed")

				err, statusCode := deployer.Deploy(req, environment, org, space, appName, "application/json", buffer)
				Expect(err).To(MatchError("prechecker failed"))

				Expect(statusCode).To(Equal(http.StatusInternalServerError))
				Expect(prechecker.AssertAllFoundationsUpCall.Received.Environment).To(Equal(environments[environment]))
			})
		})
	})

	Describe("authentication", func() {
		Context("a username and password are not provided", func() {
			Context("when authenticate in the config is not true", func() {
				It("uses the config username and password and accepts the request with a http.StatusOK", func() {
					By("setting authenticate to false")
					deployer.Config.Environments[environment] = config.Environment{Authenticate: false}

					By("not setting basic auth")

					err, statusCode := deployer.Deploy(req, environment, org, space, appName, "application/json", buffer)
					Expect(err).ToNot(HaveOccurred())
					Expect(statusCode).To(Equal(http.StatusOK))

					Expect(buffer).To(ContainSubstring("deploy was successful"))
					Expect(eventManager.EmitCall.TimesCalled).To(Equal(3), eventManagerNotEnoughCalls)
					Expect(buffer).To(ContainSubstring(username))
				})
			})

			Context("when authenticate in the config is true", func() {
				It("rejects the request with a http.StatusUnauthorized", func() {
					deployer.Config.Environments[environment] = config.Environment{Authenticate: true}

					By("not setting basic auth")

					err, statusCode := deployer.Deploy(req, environment, org, space, appName, "application/json", buffer)
					Expect(err).To(MatchError("basic auth header not found"))

					Expect(statusCode).To(Equal(http.StatusUnauthorized))
					Expect(eventManager.EmitCall.TimesCalled).To(Equal(0), eventManagerNotEnoughCalls)
				})
			})
		})
	})

	Describe("deploying with JSON in the request body", func() {
		Context("with missing properties in the JSON", func() {
			It("returns an error and http.StatusInternalServerError", func() {
				By("sending empty JSON")
				requestBody = bytes.NewBufferString("{}")

				req, _ = http.NewRequest("POST", "", requestBody)

				err, statusCode := deployer.Deploy(req, environment, org, space, appName, "application/json", buffer)
				Expect(err).To(MatchError("The following properties are missing: artifact_url"))

				Expect(statusCode).To(Equal(http.StatusInternalServerError))
			})
		})

		Context("when manifest is given in the request body", func() {
			Context("if the provided manifest is base64 encoded", func() {
				It("decodes the manifest, does not return an error and returns http.StatusOK", func() {
					deploymentInfo.Manifest = "manifest-" + randomizer.StringRunes(10)

					By("base64 encoding the manifest")
					base64Manifest := base64.StdEncoding.EncodeToString([]byte(deploymentInfo.Manifest))

					By("including the manifest in the request body")
					requestBody = bytes.NewBufferString(fmt.Sprintf(`{"artifact_url": "%s", "manifest": "%s"}`,
						artifactURL,
						base64Manifest,
					))

					req, _ = http.NewRequest("POST", "", requestBody)

					err, statusCode := deployer.Deploy(req, environment, org, space, appName, "application/json", buffer)
					Expect(err).ToNot(HaveOccurred())

					Expect(statusCode).To(Equal(http.StatusOK))
				})
			})

			Context("if the provided manifest is not base64 encoded", func() {
				It("returns an error and http.StatusBadRequest", func() {
					deploymentInfo.Manifest = "manifest-" + randomizer.StringRunes(10)

					By("not base64 encoding the manifest")

					By("including the manifest in the JSON")
					requestBody = bytes.NewBufferString(fmt.Sprintf(`{"artifact_url": "%s", "manifest": "%s"}`,
						artifactURL,
						deploymentInfo.Manifest,
					))

					req, _ = http.NewRequest("POST", "", requestBody)

					err, statusCode := deployer.Deploy(req, environment, org, space, appName, "application/json", buffer)
					Expect(err).To(MatchError("cannot open manifest file"))

					Expect(statusCode).To(Equal(http.StatusBadRequest))
				})
			})
		})

		Describe("fetching an artifact from an artifact url", func() {
			Context("when Fetcher fails", func() {
				It("returns an error and http.StatusInternalServerError", func() {
					fetcher.FetchCall.Returns.AppPath = ""
					fetcher.FetchCall.Returns.Error = errors.New("fetcher error")

					err, statusCode := deployer.Deploy(req, environment, org, space, appName, "application/json", buffer)
					Expect(err).To(MatchError("fetcher error"))

					Expect(statusCode).To(Equal(http.StatusInternalServerError))
					Expect(fetcher.FetchCall.Received.ArtifactURL).To(Equal(artifactURL))
					Expect(fetcher.FetchCall.Received.Manifest).To(BeEmpty())
				})
			})
		})
	})

	Describe("deploying with a zip file in the request body", func() {
		Context("when manifest file cannot be found in the extracted zip", func() {
			It("deploys successfully and returns http.StatusOK because manifest is optional", func() {
				err, statusCode := deployer.Deploy(req, environment, org, space, appName, "application/zip", buffer)
				Expect(err).To(BeNil())

				Expect(statusCode).To(Equal(http.StatusOK))
				Expect(buffer).To(ContainSubstring("deploy was successful"))
			})
		})

		Describe("fetching an artifact from the request body", func() {
			Context("when Fetcher fails", func() {
				It("returns an error and http.StatusInternalServerError", func() {
					fetcher.FetchFromZipCall.Returns.AppPath = ""
					fetcher.FetchFromZipCall.Returns.Error = errors.New("fetcher error")

					err, statusCode := deployer.Deploy(req, environment, org, space, appName, "application/zip", buffer)
					Expect(err).To(MatchError("fetcher error"))

					Expect(statusCode).To(Equal(http.StatusInternalServerError))
				})
			})
		})
	})

	Describe("setting the number of instances in the deployment", func() {
		Context("when a manifest with instances is provided", func() {
			It("uses the instances declared in the manifest", func() {
				deploymentInfo.Manifest = `---
applications:
- name: deployadactyl
  instances: 1337
`
				base64Manifest := base64.StdEncoding.EncodeToString([]byte(deploymentInfo.Manifest))

				requestBody = bytes.NewBufferString(fmt.Sprintf(`{
	 					"artifact_url": "%s",
	 					"manifest": "%s"
	 				}`,
					artifactURL,
					base64Manifest,
				))

				req, _ = http.NewRequest("POST", "", requestBody)

				err, _ := deployer.Deploy(req, environment, org, space, appName, "application/json", buffer)
				Expect(err).ToNot(HaveOccurred())

				Expect(blueGreener.PushCall.Received.DeploymentInfo.Instances).To(Equal(uint16(1337)))
			})
		})

		Context("when a manifest is not provided", func() {
			It("uses the instances declared in the deployadactyl config", func() {
				deployer.Config.Environments[environment] = config.Environment{Instances: 303}

				deployer.Deploy(req, environment, org, space, appName, "application/json", buffer)

				Expect(blueGreener.PushCall.Received.DeploymentInfo.Instances).To(Equal(uint16(303)))
			})
		})
	})

	Describe("not finding an environment in the config", func() {
		It("returns an error and an http.StatusInternalServerError", func() {
			deployer = Deployer{
				config.Config{},
				blueGreener,
				fetcher,
				prechecker,
				eventManager,
				randomizerMock,
				log,
				&afero.Afero{Fs: afero.NewMemMapFs()},
			}

			err, statusCode := deployer.Deploy(req, environment, org, space, appName, "application/json", buffer)
			Expect(err).To(MatchError(fmt.Sprintf("environment not found: %s", environment)))

			Expect(statusCode).To(Equal(http.StatusInternalServerError))
			Expect(buffer).To(ContainSubstring(fmt.Sprintf("environment not found: %s", environment)))
		})
	})

	Describe("deployment output", func() {
		It("shows the user deployment info properties", func() {
			_, statusCode := deployer.Deploy(req, environment, org, space, appName, "application/json", buffer)

			Expect(statusCode).To(Equal(http.StatusOK))
			Expect(buffer).To(ContainSubstring(artifactURL))
			Expect(buffer).To(ContainSubstring(username))
			Expect(buffer).To(ContainSubstring(environment))
			Expect(buffer).To(ContainSubstring(org))
			Expect(buffer).To(ContainSubstring(space))
			Expect(buffer).To(ContainSubstring(appName))
		})

		It("shows the user their deploy was successful", func() {
			deployer.Deploy(req, environment, org, space, appName, "application/json", buffer)

			Expect(buffer).To(ContainSubstring("deploy was successful"))
		})
	})

	Describe("emitting events during a deployment", func() {
		BeforeEach(func() {
			eventManager.EmitCall.Returns.Error = nil
		})

		Context("when EventManager fails on deploy.start", func() {
			It("returns an error and an http.StatusInternalServerError", func() {
				eventManager.EmitCall.Returns.Error = append(eventManager.EmitCall.Returns.Error, errors.New("deploy.start error"))
				eventManager.EmitCall.Returns.Error = append(eventManager.EmitCall.Returns.Error, nil)

				err, statusCode := deployer.Deploy(req, environment, org, space, appName, "application/json", buffer)
				Expect(err).To(MatchError("an error occurred in the deploy.start event: deploy.start error"))

				Expect(statusCode).To(Equal(http.StatusInternalServerError))
				Expect(buffer).To(ContainSubstring("deploy.start error"))
				Expect(eventManager.EmitCall.TimesCalled).To(Equal(2), eventManagerNotEnoughCalls)
			})

			Context("when EventManager also fails on deploy.finish", func() {
				It("outputs deploy.finish error", func() {
					eventManager.EmitCall.Returns.Error = append(eventManager.EmitCall.Returns.Error, errors.New("deploy.start error"))
					eventManager.EmitCall.Returns.Error = append(eventManager.EmitCall.Returns.Error, errors.New("deploy.finish error"))

					err, statusCode := deployer.Deploy(req, environment, org, space, appName, "application/json", buffer)
					Expect(err).To(MatchError("an error occurred in the deploy.start event: deploy.start error: an error occurred in the deploy.finish event: deploy.finish error"))

					Expect(statusCode).To(Equal(http.StatusInternalServerError))
					Expect(buffer).To(ContainSubstring("deploy.start error"))
					Expect(buffer).To(ContainSubstring("deploy.finish error"))
					Expect(eventManager.EmitCall.TimesCalled).To(Equal(2), eventManagerNotEnoughCalls)
				})
			})
		})

		Context("when the blue greener fails", func() {
			It("returns an error and outputs deploy.failure", func() {
				eventManager.EmitCall.Returns.Error = append(eventManager.EmitCall.Returns.Error, nil)
				eventManager.EmitCall.Returns.Error = append(eventManager.EmitCall.Returns.Error, nil)
				eventManager.EmitCall.Returns.Error = append(eventManager.EmitCall.Returns.Error, nil)

				blueGreener.PushCall.Returns.Error = errors.New("blue greener failed")

				err, statusCode := deployer.Deploy(req, environment, org, space, appName, "application/json", buffer)
				Expect(err).To(MatchError("blue greener failed"))

				Expect(statusCode).To(Equal(http.StatusInternalServerError))
				Expect(eventManager.EmitCall.Received.Events[1].Type).To(Equal("deploy.failure"))
			})
		})

		Context("when blue greener succeeds", func() {
			It("does not return an error and outputs a deploy.success and http.StatusOK", func() {
				eventManager.EmitCall.Returns.Error = append(eventManager.EmitCall.Returns.Error, nil)
				eventManager.EmitCall.Returns.Error = append(eventManager.EmitCall.Returns.Error, nil)
				eventManager.EmitCall.Returns.Error = append(eventManager.EmitCall.Returns.Error, nil)

				err, statusCode := deployer.Deploy(req, environment, org, space, appName, "application/json", buffer)
				Expect(err).To(BeNil())

				Expect(statusCode).To(Equal(http.StatusOK))
				Expect(eventManager.EmitCall.Received.Events[1].Type).To(Equal("deploy.success"))
			})

			Context("when emitting a deploy.succes event fails", func() {
				It("return an error and outputs a deploy.success and http.StatusOK", func() {
					eventManager.EmitCall.Returns.Error = append(eventManager.EmitCall.Returns.Error, nil)
					eventManager.EmitCall.Returns.Error = append(eventManager.EmitCall.Returns.Error, errors.New("event error"))
					eventManager.EmitCall.Returns.Error = append(eventManager.EmitCall.Returns.Error, nil)

					err, statusCode := deployer.Deploy(req, environment, org, space, appName, "application/json", buffer)
					Expect(err).To(BeNil())

					Expect(statusCode).To(Equal(http.StatusOK))
					Expect(buffer).To(ContainSubstring("event error"))
					Expect(eventManager.EmitCall.Received.Events[1].Type).To(Equal("deploy.success"))
				})
			})
		})
	})

	Describe("BlueGreener.Push", func() {
		Context("when BlueGreener fails with a login failed error", func() {
			It("returns an error and a http.StatusUnauthorized", func() {
				blueGreener.PushCall.Returns.Error = errors.New("login failed")

				err, statusCode := deployer.Deploy(req, environment, org, space, appName, "application/json", buffer)
				Expect(err).To(MatchError("login failed"))

				Expect(statusCode).To(Equal(http.StatusBadRequest))
			})
		})

		Context("when BlueGreener fails during a deploy with a zip file in the request body", func() {
			It("returns an error and a http.StatusInternalServerError", func() {
				Expect(af.WriteFile(testManifestLocation+"/manifest.yml", []byte(testManifest), 0644)).To(Succeed())

				fetcher.FetchFromZipCall.Returns.AppPath = testManifestLocation

				blueGreener.PushCall.Returns.Error = errors.New("blue green error")

				err, statusCode := deployer.Deploy(req, environment, org, space, appName, "application/zip", buffer)
				Expect(err).To(MatchError("blue green error"))

				Expect(statusCode).To(Equal(http.StatusInternalServerError))
				Expect(blueGreener.PushCall.Received.AppPath).To(Equal(testManifestLocation))
				Expect(blueGreener.PushCall.Received.DeploymentInfo.Manifest).To(Equal(fmt.Sprintf("---\napplications:\n- name: deployadactyl\n  memory: 256M\n  disk_quota: 256M\n")))
				Expect(blueGreener.PushCall.Received.DeploymentInfo.ArtifactURL).To(ContainSubstring(testManifestLocation))
			})
		})

		Context("when BlueGreener fails during a deploy with JSON in the request body", func() {
			It("returns an error and a http.StatusInternalServerError", func() {
				fetcher.FetchCall.Returns.AppPath = appPath

				blueGreener.PushCall.Returns.Error = errors.New("blue green error")

				err, statusCode := deployer.Deploy(req, environment, org, space, appName, "application/json", buffer)
				Expect(err).To(MatchError("blue green error"))

				Expect(statusCode).To(Equal(http.StatusInternalServerError))
				Expect(blueGreener.PushCall.Received.AppPath).To(Equal(appPath))
				Expect(blueGreener.PushCall.Received.DeploymentInfo).To(Equal(deploymentInfo))
			})
		})
	})

	Describe("happy path deploying with json in the request body", func() {
		Context("when no errors occur", func() {
			It("accepts the request and returns http.StatusOK", func() {
				fetcher.FetchCall.Returns.AppPath = appPath

				err, statusCode := deployer.Deploy(req, environment, org, space, appName, "application/json", buffer)
				Expect(err).To(BeNil())

				Expect(statusCode).To(Equal(http.StatusOK))

				Expect(buffer).To(ContainSubstring("Deployment Parameters"))
				Expect(buffer).To(ContainSubstring("deploy was successful"))

				Eventually(logBuffer).Should(Say("prechecking the foundations"))
				Eventually(logBuffer).Should(Say("checking for basic auth"))
				Eventually(logBuffer).Should(Say("deploying from json request"))
				Eventually(logBuffer).Should(Say("building deploymentInfo"))
				Eventually(logBuffer).Should(Say("Deployment Parameters"))
				Eventually(logBuffer).Should(Say("emitting a deploy.start event"))
				Eventually(logBuffer).Should(Say("emitting a deploy.success event"))
				Eventually(logBuffer).Should(Say("emitting a deploy.finish event"))

				Expect(prechecker.AssertAllFoundationsUpCall.Received.Environment).To(Equal(environments[environment]))
				Expect(fetcher.FetchCall.Received.ArtifactURL).To(Equal(artifactURL))
				Expect(fetcher.FetchCall.Received.Manifest).To(BeEmpty())
				Expect(eventManager.EmitCall.Received.Events[0].Type).To(Equal("deploy.start"))
				Expect(eventManager.EmitCall.Received.Events[1].Type).To(Equal("deploy.success"))
				Expect(eventManager.EmitCall.Received.Events[2].Type).To(Equal("deploy.finish"))
				Expect(blueGreener.PushCall.Received.Environment).To(Equal(environments[environment]))
				Expect(blueGreener.PushCall.Received.AppPath).To(Equal(appPath))
				Expect(blueGreener.PushCall.Received.DeploymentInfo).To(Equal(deploymentInfo))
			})
		})
	})

	Describe("happy path deploying with a zip file in the request body", func() {
		Context("when no errors occur", func() {
			It("accepts the request and returns http.StatusOK", func() {
				Expect(af.WriteFile(testManifestLocation+"/manifest.yml", []byte(testManifest), 0644)).To(Succeed())

				fetcher.FetchFromZipCall.Returns.AppPath = testManifestLocation

				err, statusCode := deployer.Deploy(req, environment, org, space, appName, "application/zip", buffer)
				Expect(err).To(BeNil())

				Expect(statusCode).To(Equal(http.StatusOK))

				Expect(buffer).To(ContainSubstring("Deployment Parameters"))
				Expect(buffer).To(ContainSubstring("deploy was successful"))

				Eventually(logBuffer).Should(Say("prechecking the foundations"))
				Eventually(logBuffer).Should(Say("checking for basic auth"))
				Eventually(logBuffer).Should(Say("deploying from zip request"))
				Eventually(logBuffer).Should(Say("Deployment Parameters"))
				Eventually(logBuffer).Should(Say("emitting a deploy.start event"))
				Eventually(logBuffer).Should(Say("emitting a deploy.success event"))
				Eventually(logBuffer).Should(Say("emitting a deploy.finish event"))

				Expect(prechecker.AssertAllFoundationsUpCall.Received.Environment).To(Equal(environments[environment]))
				Expect(fetcher.FetchFromZipCall.Received.Request).To(Equal(req))
				Expect(eventManager.EmitCall.Received.Events[0].Type).To(Equal("deploy.start"))
				Expect(eventManager.EmitCall.Received.Events[1].Type).To(Equal("deploy.success"))
				Expect(eventManager.EmitCall.Received.Events[2].Type).To(Equal("deploy.finish"))
				Expect(blueGreener.PushCall.Received.Environment).To(Equal(environments[environment]))
				Expect(blueGreener.PushCall.Received.AppPath).To(Equal(testManifestLocation))
				Expect(blueGreener.PushCall.Received.DeploymentInfo.Manifest).To(Equal(fmt.Sprintf("---\napplications:\n- name: deployadactyl\n  memory: 256M\n  disk_quota: 256M\n")))
				Expect(blueGreener.PushCall.Received.DeploymentInfo.ArtifactURL).To(ContainSubstring(testManifestLocation))
			})
		})
	})
})
