package deployadactyl_test

import (
	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestDeployadactyl(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Deployadactyl Suite")
}

var _ = BeforeSuite(func() {
	gin.SetMode(gin.TestMode)
})
