package push

import (
	"os"

	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

var (
	username string
	password string
)

func TestService(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service Suite")
}

var _ = BeforeSuite(func() {
	gin.SetMode(gin.TestMode)
})

var _ = AfterSuite(func() {
	os.Setenv("CF_USERNAME", username)
	os.Setenv("CF_PASSWORD", password)
})
