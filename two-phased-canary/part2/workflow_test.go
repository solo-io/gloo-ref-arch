package part2_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/solo-io/go-utils/testutils"
	"github.com/solo-io/valet/test/e2e/gloo/two-phased-canary/part2"
	"testing"
)

func TestTwoPhasedCanary(t *testing.T) {
	RegisterFailHandler(Fail)
	testutils.RegisterPreFailHandler(
		func() {
			testutils.PrintTrimmedStack()
		})
	testutils.RegisterCommonFailHandlers()
	RunSpecs(t, "Two Phased Canary Test Suite")
}

var _ = Describe("Two Phased Canary, Part 2", func() {
	testWorkflow := part2.GetTestWorkflow()

	BeforeSuite(func() {
		testWorkflow.Setup(".")
	})

	It("works", func() {
		testWorkflow.Run(".")
	})
})
