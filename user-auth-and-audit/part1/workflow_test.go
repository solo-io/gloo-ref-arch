package part1_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/solo-io/go-utils/testutils"
	"testing"
)

func TestChainedAuthAndAccesslogging(t *testing.T) {
	RegisterFailHandler(Fail)
	testutils.RegisterPreFailHandler(
		func() {
			testutils.PrintTrimmedStack()
		})
	testutils.RegisterCommonFailHandlers()
	RunSpecs(t, "User Auth and Auditing")
}

var _ = Describe("Part 1", func() {
	testWorkflow := part1.GetTestWorkflow()

	BeforeSuite(func() {
		testWorkflow.Setup(".")
	})

	It("runs", func() {
		testWorkflow.Run(".")
	})
})
