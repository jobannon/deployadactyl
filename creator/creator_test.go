package creator

import (
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Custom creator", func() {
	AfterEach(func() {
		os.Unsetenv("CF_USERNAME")
		os.Unsetenv("CF_PASSWORD")
	})

	It("creates the creator from the provided yaml configuration", func() {
		os.Setenv("CF_USERNAME", "test user")
		os.Setenv("CF_PASSWORD", "test pwd")

		level := "DEBUG"
		configPath := "./testconfig.yml"

		creator, err := Custom(level, configPath)

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

		_, err := Custom(level, configPath)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal("missing environment variables: CF_USERNAME, CF_PASSWORD"))
	})
})
