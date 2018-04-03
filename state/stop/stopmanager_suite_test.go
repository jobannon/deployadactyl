package stop_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestStopmanager(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Stopmanager Suite")
}
