package extractor_test

import (
	"io/ioutil"
	"os"
	"path"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/spf13/afero"

	"github.com/op/go-logging"

	. "github.com/compozed/deployadactyl/artifetcher/extractor"
	"github.com/compozed/deployadactyl/logger"
	"github.com/compozed/deployadactyl/randomizer"
)

const deployadactylManifest = `---
applications:
- name: deployadactyl
  memory: 256M
  disk_quota: 256M
`

var _ = Describe("Extracting", func() {
	var (
		af          *afero.Afero
		file        string
		destination string
		extractor   Extractor
	)

	BeforeEach(func() {
		file = "/artifact.jar"
		destination = "../fixtures/deployadactyl-fixture"
		af = &afero.Afero{Fs: afero.NewMemMapFs()}
		extractor = Extractor{logger.DefaultLogger(GinkgoWriter, logging.DEBUG, "extractor_test"), af}

		fileBytes, err := ioutil.ReadFile("../fixtures/deployadactyl-fixture.jar")
		Expect(err).ToNot(HaveOccurred())

		Expect(af.WriteFile(file, fileBytes, 0644)).To(Succeed())
	})

	AfterEach(func() {
		Expect(os.RemoveAll(destination)).To(Succeed())
	})

	It("unzips the artifact", func() {
		Expect(extractor.Unzip(file, destination, "")).To(Succeed())

		extractedFile, err := af.ReadFile(path.Join(destination, "index.html"))
		Expect(err).ToNot(HaveOccurred())
		Expect(extractedFile).To(ContainSubstring("public/assets/images/pterodactyl.png"))
	})

	Context("when manifest is an empty string", func() {
		It("leaves the manifest alone", func() {
			Expect(extractor.Unzip(file, destination, "")).To(Succeed())

			extractedManifest, err := af.ReadFile(path.Join(destination, "manifest.yml"))
			Expect(err).ToNot(HaveOccurred())
			Expect(extractedManifest).To(BeEquivalentTo(deployadactylManifest))
		})
	})

	Context("when manifest is not an empty string", func() {
		It("unzips the artifact and overwrites the manifest", func() {
			manifestContents := "manifestContents-" + randomizer.StringRunes(10)
			Expect(extractor.Unzip(file, destination, manifestContents)).To(Succeed())

			extractedManifest, err := af.ReadFile(path.Join(destination, "manifest.yml"))
			Expect(err).ToNot(HaveOccurred())
			Expect(extractedManifest).To(BeEquivalentTo(manifestContents))
		})
	})

	It("can not unzip an invalid file", func() {
		file := "../fixtures/bad-deployadactyl-fixture.tgz"
		destination = "../fixtures/bad-deployadactyl-fixture"
		af = &afero.Afero{Fs: afero.NewMemMapFs()}
		extractor := Extractor{logger.DefaultLogger(GinkgoWriter, logging.DEBUG, "extractor_test"), af}

		Expect(extractor.Unzip(file, destination, "")).ToNot(Succeed())
	})
})
