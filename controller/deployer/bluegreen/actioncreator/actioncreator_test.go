package actioncreator_test

import (
	"bytes"
	"encoding/base64"
	"github.com/compozed/deployadactyl/constants"
	"github.com/compozed/deployadactyl/controller/deployer"
	"github.com/compozed/deployadactyl/controller/deployer/bluegreen/actioncreator"
	"github.com/compozed/deployadactyl/logger"
	"github.com/compozed/deployadactyl/mocks"
	"github.com/compozed/deployadactyl/randomizer"
	"github.com/compozed/deployadactyl/structs"
	"github.com/go-errors/errors"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/op/go-logging"
	"io/ioutil"
	"reflect"
)

var logBuffer = bytes.NewBuffer([]byte{})
var log = logger.DefaultLogger(logBuffer, logging.DEBUG, "deployer tests")

var _ = Describe("Actioncreator", func() {
	Describe("Setup", func() {
		Context("content-type is JSON", func() {

			manifest := `---
applications:
- instances: 2`
			encodedManifest := base64.StdEncoding.EncodeToString([]byte(manifest))

			It("should extract manifest from the request", func() {
				fetcher := &mocks.Fetcher{}
				fetcher.FetchCall.Returns.AppPath = "newAppPath"
				eventManager := &mocks.EventManager{}

				pusherCreator := &actioncreator.PusherCreator{
					Fetcher:      fetcher,
					Logger:       logger.DeploymentLogger{log, randomizer.StringRunes(10)},
					EventManager: eventManager,
				}
				deploymentInfo := structs.DeploymentInfo{Manifest: encodedManifest, ContentType: "JSON"}

				_, returnsManifest, _, _ := pusherCreator.SetUp(deploymentInfo, 0)
				Expect(returnsManifest).To(Equal(manifest))
				logBytes, _ := ioutil.ReadAll(logBuffer)
				Eventually(string(logBytes)).Should(ContainSubstring("deploying from json request"))
			})
			It("should fetch and return app path", func() {
				fetcher := &mocks.Fetcher{}
				fetcher.FetchCall.Returns.AppPath = "newAppPath"
				eventManager := &mocks.EventManager{}

				pusherCreator := &actioncreator.PusherCreator{Fetcher: fetcher,
					Logger:       logger.DeploymentLogger{log, randomizer.StringRunes(10)},
					EventManager: eventManager,
				}
				deploymentInfo := structs.DeploymentInfo{
					Manifest:    encodedManifest,
					ArtifactURL: "https://artifacturl.com",
					ContentType: "JSON",
				}

				appPath, _, _, _ := pusherCreator.SetUp(deploymentInfo, 0)
				Expect(appPath).To(Equal("newAppPath"))
				Expect(fetcher.FetchCall.Received.ArtifactURL).To(Equal(deploymentInfo.ArtifactURL))
				Expect(fetcher.FetchCall.Received.Manifest).To(Equal(manifest))

			})
			It("should error when artifact cannot be fetched", func() {
				fetcher := &mocks.Fetcher{}
				fetcher.FetchCall.Returns.Error = errors.New("fetch error")
				eventManager := &mocks.EventManager{}

				pusherCreator := &actioncreator.PusherCreator{
					Fetcher:      fetcher,
					Logger:       logger.DeploymentLogger{log, randomizer.StringRunes(10)},
					EventManager: eventManager,
				}

				deploymentInfo := structs.DeploymentInfo{
					Manifest:    encodedManifest,
					ArtifactURL: "https://artifacturl.com",
					ContentType: "JSON",
				}

				_, _, _, err := pusherCreator.SetUp(deploymentInfo, 0)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("unzipped app path failed: fetch error"))
			})
			It("should retrieve instances from manifest", func() {
				fetcher := &mocks.Fetcher{}
				fetcher.FetchCall.Returns.AppPath = "newAppPath"
				eventManager := &mocks.EventManager{}
				pusherCreator := &actioncreator.PusherCreator{
					Fetcher:      fetcher,
					Logger:       logger.DeploymentLogger{log, randomizer.StringRunes(10)},
					EventManager: eventManager,
				}

				deploymentInfo := structs.DeploymentInfo{
					Manifest:    encodedManifest,
					ContentType: "JSON",
				}

				_, _, instances, _ := pusherCreator.SetUp(deploymentInfo, 0)
				Expect(instances).To(Equal(uint16(2)))
			})
			It("should emit artifact retrieval events", func() {
				fetcher := &mocks.Fetcher{}
				eventManager := &mocks.EventManager{}
				pusherCreator := &actioncreator.PusherCreator{
					Fetcher:      fetcher,
					Logger:       logger.DeploymentLogger{log, randomizer.StringRunes(10)},
					EventManager: eventManager,
				}
				deploymentInfo := structs.DeploymentInfo{
					Manifest:    encodedManifest,
					ArtifactURL: "https://artifacturl.com",
					ContentType: "JSON",
				}

				pusherCreator.SetUp(deploymentInfo, 0)

				Expect(eventManager.EmitCall.Received.Events[0].Type).Should(Equal(constants.ArtifactRetrievalStart))
				Expect(eventManager.EmitCall.Received.Events[1].Type).Should(Equal(constants.ArtifactRetrievalSuccess))

			})
			It("should return error if start emit fails", func() {
				fetcher := &mocks.Fetcher{}
				eventManager := &mocks.EventManager{}

				eventManager.EmitCall.Returns.Error = []error{errors.New("error")}

				pusherCreator := &actioncreator.PusherCreator{
					Fetcher:      fetcher,
					Logger:       logger.DeploymentLogger{log, randomizer.StringRunes(10)},
					EventManager: eventManager,
				}
				deploymentInfo := structs.DeploymentInfo{
					Manifest:    encodedManifest,
					ArtifactURL: "https://artifacturl.com",
					ContentType: "JSON",
				}

				_, _, _, err := pusherCreator.SetUp(deploymentInfo, 0)

				Expect(reflect.TypeOf(err)).Should(Equal(reflect.TypeOf(deployer.EventError{})))

			})
			It("should return error if emit success fails", func() {
				fetcher := &mocks.Fetcher{}
				eventManager := &mocks.EventManager{}

				eventManager.EmitCall.Returns.Error = []error{nil, errors.New("error")}

				pusherCreator := &actioncreator.PusherCreator{
					Fetcher:      fetcher,
					Logger:       logger.DeploymentLogger{log, randomizer.StringRunes(10)},
					EventManager: eventManager,
				}
				deploymentInfo := structs.DeploymentInfo{
					Manifest:    encodedManifest,
					ArtifactURL: "https://artifacturl.com",
					ContentType: "JSON",
				}

				_, _, _, err := pusherCreator.SetUp(deploymentInfo, 0)

				Expect(reflect.TypeOf(err)).Should(Equal(reflect.TypeOf(deployer.EventError{})))

			})
			It("should emit failure if fetch fails", func() {
				fetcher := &mocks.Fetcher{}
				fetcher.FetchCall.Returns.Error = errors.New("a test error")

				eventManager := &mocks.EventManager{}

				eventManager.EmitCall.Returns.Error = []error{nil, errors.New("error")}

				pusherCreator := &actioncreator.PusherCreator{
					Fetcher:      fetcher,
					Logger:       logger.DeploymentLogger{log, randomizer.StringRunes(10)},
					EventManager: eventManager,
				}
				deploymentInfo := structs.DeploymentInfo{
					Manifest:    encodedManifest,
					ArtifactURL: "https://artifacturl.com",
					ContentType: "JSON",
				}

				pusherCreator.SetUp(deploymentInfo, 0)

				Expect(eventManager.EmitCall.Received.Events[1].Type).Should(Equal(constants.ArtifactRetrievalFailure))
			})
		})

		Context("when instances is nil", func() {
			It("assigns environmental instances as the instance", func() {
				manifest := `---
applications:
- name: long-running-spring-app`
				encodedManifest := base64.StdEncoding.EncodeToString([]byte(manifest))

				fetcher := &mocks.Fetcher{}
				fetcher.FetchCall.Returns.AppPath = "newAppPath"
				eventManager := &mocks.EventManager{}
				pusherCreator := &actioncreator.PusherCreator{
					Fetcher:      fetcher,
					Logger:       logger.DeploymentLogger{log, randomizer.StringRunes(10)},
					EventManager: eventManager,
				}

				deploymentInfo := structs.DeploymentInfo{
					Manifest:    encodedManifest,
					ArtifactURL: "https://artifacturl.com",
					ContentType: "JSON",
				}

				_, _, instances, _ := pusherCreator.SetUp(deploymentInfo, 22)

				Expect(instances).To(Equal(uint16(22)))
			})
		})

		Context("contentType is ZIP", func() {

			It("should extract manifest from the zip file", func() {
				fetcher := &mocks.Fetcher{}
				fetcher.FetchFromZipCall.Returns.AppPath = "newAppPath"
				eventManager := &mocks.EventManager{}
				pusherCreator := &actioncreator.PusherCreator{
					Fetcher:      fetcher,
					Logger:       logger.DeploymentLogger{log, randomizer.StringRunes(10)},
					EventManager: eventManager,
				}
				deploymentInfo := structs.DeploymentInfo{ContentType: "ZIP"}

				appPath, _, _, _ := pusherCreator.SetUp(deploymentInfo, 0)

				Expect(appPath).To(Equal("newAppPath"))
				logBytes, _ := ioutil.ReadAll(logBuffer)
				Eventually(string(logBytes)).Should(ContainSubstring("deploying from zip request"))
			})
			It("should error when artifact cannot be fetched", func() {
				fetcher := &mocks.Fetcher{}
				fetcher.FetchFromZipCall.Returns.Error = errors.New("a test error")
				eventManager := &mocks.EventManager{}
				pusherCreator := &actioncreator.PusherCreator{
					Fetcher:      fetcher,
					Logger:       logger.DeploymentLogger{log, randomizer.StringRunes(10)},
					EventManager: eventManager,
				}
				deploymentInfo := structs.DeploymentInfo{
					ContentType: "ZIP",
				}

				_, _, _, err := pusherCreator.SetUp(deploymentInfo, 0)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("unzipping request body error: a test error"))
			})
		})

	})

})
