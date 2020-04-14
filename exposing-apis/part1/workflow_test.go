package part1_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/solo-io/gloo-ref-arch/exposing-apis/part1"
	"github.com/solo-io/go-utils/testutils"
	"testing"
)

func TestExposingApis(t *testing.T) {
	RegisterFailHandler(Fail)
	testutils.RegisterPreFailHandler(
		func() {
			testutils.PrintTrimmedStack()
		})
	testutils.RegisterCommonFailHandlers()
	RunSpecs(t, "Exposing Apis Suite")
}

var _ = Describe("Part 1", func() {
	testWorkflow := part1.GetTestWorkflow()

	BeforeSuite(func() {
		testWorkflow.Setup(".")
	})

	It("works", func() {
		testWorkflow.Run(".")
	})
})
