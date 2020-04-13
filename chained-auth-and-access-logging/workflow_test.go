package chained_auth_and_access_logging_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/solo-io/go-utils/testutils"
	chained_auth_and_access_logging "github.com/solo-io/valet/test/e2e/gloo/chained-auth-and-access-logging"
	"testing"
)

func TestChainedAuthAndAccesslogging(t *testing.T) {
	RegisterFailHandler(Fail)
	testutils.RegisterPreFailHandler(
		func() {
			testutils.PrintTrimmedStack()
		})
	testutils.RegisterCommonFailHandlers()
	RunSpecs(t, "Chained Auth and Access Logging Suite")
}

var _ = Describe("chained auth and access logging suite", func() {
	testWorkflow := chained_auth_and_access_logging.GetTestWorkflow()

	BeforeSuite(func() {
		testWorkflow.Setup(".")
	})

	It("runs", func() {
		testWorkflow.Run(".")
	})
})
