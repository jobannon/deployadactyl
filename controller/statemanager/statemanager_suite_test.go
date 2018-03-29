package statemanager_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestStatemanager(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Statemanager Suite")
}
