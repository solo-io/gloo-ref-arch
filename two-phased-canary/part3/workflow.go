package part3

import (
	"context"
	"github.com/solo-io/gloo-ref-arch/utils/gloo"
	"github.com/solo-io/valet/pkg/step/check"
	"github.com/solo-io/valet/pkg/tests"
	"github.com/solo-io/valet/pkg/workflow"
)

func curl(path, responseBody string) *workflow.Step {
	return &workflow.Step{
		Curl: &check.Curl{
			Service:      gloo.GatewayProxy(),
			StatusCode:   200,
			Path:         path,
			ResponseBody: responseBody,
		},
	}
}

func GetWorkflow() *workflow.Workflow {
	return &workflow.Workflow{
		SetupSteps: []*workflow.Step{
			gloo.InstallGlooEnterpriseWithValues("values.yaml"),
			gloo.DeleteAllVirtualServices(),
			gloo.GlooctlCheck(),
			gloo.PatchSettings("settings-patch-revert.yaml"),
			gloo.DeleteNamespaces("echo", "foxtrot"),
		},
		Steps: []*workflow.Step{
			// Part 1: Deploy the app
			gloo.PatchSettings("settings-patch.yaml").WithId("patch-settings"),
			workflow.Apply("vs.yaml").WithId("deploy-vs"),
			workflow.Apply("echo.yaml").WithId("deploy-echo"),
			workflow.Apply("foxtrot.yaml").WithId("deploy-foxtrot"),
			workflow.WaitForPods("echo").WithId("wait-echo"),
			workflow.WaitForPods("foxtrot").WithId("wait-foxtrot"),
			curl("/echo", "version:echo-v1"),
			curl("/foxtrot", "version:foxtrot-v1"),

			// Part 2: Setup echo-v2 and shadowing
			workflow.Apply("echo-v2.yaml").WithId("deploy-echo-v2"),
			workflow.Apply("upstream-shadow.yaml").WithId("deploy-shadow-upstream"),
			workflow.WaitForPods("echo").WithId("wait-echo"),
			workflow.Apply("rt-shadow.yaml").WithId("deploy-route-shadowing"),
			curl("/echo", "version:echo-v1"),
		},
	}
}

func GetTestWorkflow() *tests.TestWorkflow{
	return &tests.TestWorkflow{
		Workflow:          GetWorkflow(),
		Ctx:               workflow.DefaultContext(context.TODO()),
		TestSerialization: true,
		TestDocs:          true,
	}
}