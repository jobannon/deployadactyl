package actioncreator_test

import (
	"encoding/base64"
	"github.com/compozed/deployadactyl/controller/deployer/bluegreen/actioncreator"
	"github.com/compozed/deployadactyl/mocks"
	"github.com/compozed/deployadactyl/structs"
	"github.com/go-errors/errors"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Actioncreator", func() {
	Describe("Setup", func() {
		Context("content-type is json", func() {

			manifest := `---
applications:
- instances: 2`
			encodedManifest := base64.StdEncoding.EncodeToString([]byte(manifest))

			It("should extract manifest from the request", func() {

				pusherCreator := &actioncreator.PusherCreator{}
				deploymentInfo := structs.DeploymentInfo{Manifest: encodedManifest, ContentType: "JSON"}
				fetcher := &mocks.Fetcher{}

				pusherCreator.Fetcher = fetcher
				fetcher.FetchCall.Returns.AppPath = "newAppPath"

				_, returnsManifest, _, _ := pusherCreator.SetUp(deploymentInfo)
				Expect(returnsManifest).To(Equal(manifest))
			})
			It("should fetch and return app path", func() {
				pusherCreator := &actioncreator.PusherCreator{}
				deploymentInfo := structs.DeploymentInfo{
					Manifest:    encodedManifest,
					ArtifactURL: "https://artifacturl.com",
					ContentType: "JSON",
				}

				fetcher := &mocks.Fetcher{}
				fetcher.FetchCall.Returns.AppPath = "newAppPath"
				pusherCreator.Fetcher = fetcher

				appPath, _, _, _ := pusherCreator.SetUp(deploymentInfo)
				Expect(appPath).To(Equal("newAppPath"))
				Expect(fetcher.FetchCall.Received.ArtifactURL).To(Equal(deploymentInfo.ArtifactURL))
				Expect(fetcher.FetchCall.Received.Manifest).To(Equal(manifest))

			})
			It("should error when artifact cannot be fetched", func() {
				pusherCreator := &actioncreator.PusherCreator{}
				deploymentInfo := structs.DeploymentInfo{
					Manifest:    encodedManifest,
					ArtifactURL: "https://artifacturl.com",
					ContentType: "JSON",
				}

				fetcher := &mocks.Fetcher{}
				fetcher.FetchCall.Returns.Error = errors.New("fetch error")
				pusherCreator.Fetcher = fetcher

				_, _, _, err := pusherCreator.SetUp(deploymentInfo)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("unzipped app path failed: fetch error"))
			})
			It("should retrieve instances from manifest", func() {
				pusherCreator := &actioncreator.PusherCreator{}
				deploymentInfo := structs.DeploymentInfo{
					Manifest:    encodedManifest,
					ContentType: "JSON",
				}

				fetcher := &mocks.Fetcher{}
				fetcher.FetchCall.Returns.AppPath = "newAppPath"
				pusherCreator.Fetcher = fetcher

				_, _, instances, _ := pusherCreator.SetUp(deploymentInfo)
				Expect(instances).To(Equal(uint16(2)))
			})
		})
		Context("contentType is ZIP", func() {

			It("should extract manifest from the zip file", func() {
				pusherCreator := &actioncreator.PusherCreator{}
				deploymentInfo := structs.DeploymentInfo{ContentType: "ZIP"}

				fetcher := &mocks.Fetcher{}
				fetcher.FetchFromZipCall.Returns.AppPath = "newAppPath"
				pusherCreator.Fetcher = fetcher

				appPath, _, _, _ := pusherCreator.SetUp(deploymentInfo)

				Expect(appPath).To(Equal("newAppPath"))
			})

		})

	})

})
