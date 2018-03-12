package actioncreator_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestActioncreator(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Actioncreator Suite")
}
