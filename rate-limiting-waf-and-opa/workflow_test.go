package rate_limiting_waf_and_opa_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/solo-io/go-utils/testutils"
	rate_limiting_waf_and_opa "github.com/solo-io/valet/test/e2e/gloo/rate-limiting-waf-and-opa"
	"testing"
)

func TestRateLimitWafAndOpa(t *testing.T) {
	RegisterFailHandler(Fail)
	testutils.RegisterPreFailHandler(
		func() {
			testutils.PrintTrimmedStack()
		})
	testutils.RegisterCommonFailHandlers()
	RunSpecs(t, "Rate Limit Waf and Opa Suite")
}

var _ = Describe("Rate limit Waf and Opa", func() {
	testWorkflow := rate_limiting_waf_and_opa.GetTestWorkflow()

	BeforeSuite(func() {
		testWorkflow.Setup(".")
	})

	It("works", func() {
		testWorkflow.Run(".")
	})
})
