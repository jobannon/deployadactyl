package artifetcher_test

import (
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/spf13/afero"

	"github.com/op/go-logging"

	. "github.com/compozed/deployadactyl/artifetcher"
	"github.com/compozed/deployadactyl/logger"
	"github.com/compozed/deployadactyl/mocks"
	"github.com/compozed/deployadactyl/randomizer"
)

var _ = Describe("Artifetcher", func() {
	var (
		artifetcher *Artifetcher
		af          *afero.Afero
		extractor   *mocks.Extractor
		testserver  *httptest.Server
		manifest    string
	)

	BeforeEach(func() {
		logger := logger.DefaultLogger(GinkgoWriter, logging.DEBUG, "artifetcher_test")
		af = &afero.Afero{Fs: afero.NewMemMapFs()}
		extractor = &mocks.Extractor{}
		artifetcher = &Artifetcher{af, extractor, logger}
		manifest = "manifest-" + randomizer.StringRunes(10)

		testserver = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, "./fixtures/deployadactyl-fixture.jar")
		}))
	})

	Describe("fetching a zip file", func() {
		It("can fetch a jar file", func() {
			defer testserver.Close()
			extractor.UnzipCall.Returns.Error = nil

			unzippedPath, err := artifetcher.Fetch(testserver.URL, "")
			Expect(err).ToNot(HaveOccurred())

			Expect(af.IsDir(unzippedPath)).To(BeTrue())

			Expect(extractor.UnzipCall.Received.Destination).To(Equal(unzippedPath))
			Expect(extractor.UnzipCall.Received.Manifest).To(BeEmpty())
		})

		It("returns an error when an invalid url is given", func() {
			_, err := artifetcher.Fetch("example://example.example", manifest)
			Expect(err).To(HaveOccurred())
		})

		It("returns an error when the URL returns a 404 not found", func() {
			testserver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "not found", 404)
			}))
			defer testserver.Close()

			_, err := artifetcher.Fetch(testserver.URL, manifest)
			Expect(err).To(HaveOccurred())
		})
	})
})
