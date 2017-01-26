package handlers

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestEnvvarhandler(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Environment Variables Found Event Handler Suite")
}
