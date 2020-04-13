package part2

import (
	"context"
	"github.com/solo-io/valet/pkg/step/check"
	"github.com/solo-io/valet/pkg/tests"
	"github.com/solo-io/valet/pkg/workflow"
	"github.com/solo-io/valet/test/e2e/gloo"
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

func curlRouteReplacement(path, headerName, headerValue string) *workflow.Step {
	return &workflow.Step{
		Curl: &check.Curl{
			Service:      gloo.GatewayProxy(),
			Headers: map[string]string {
				headerName: headerValue,
			},
			StatusCode:   404,
			Path:         path,
			ResponseBody: "Gloo Gateway has invalid configuration. Administrators should run `glooctl check` to find and fix config errors.",
		},
	}
}

func curlWithHeader(path, responseBody, headerName, headerValue string) *workflow.Step {
	step := curl(path, responseBody)
	step.Curl.Headers = map[string]string{
		headerName: headerValue,
	}
	return step
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
			workflow.Apply("echo.yaml").WithId("deploy-echo"),
			workflow.Apply("foxtrot.yaml").WithId("deploy-foxtrot"),
			workflow.WaitForPods("echo").WithId("wait-echo"),
			workflow.WaitForPods("foxtrot").WithId("wait-foxtrot"),
			workflow.Apply("upstream-echo.yaml").WithId("deploy-upstream-echo"),
			workflow.Apply("upstream-foxtrot.yaml").WithId("deploy-upstream-foxtrot"),
			workflow.Apply("rt-echo-1.yaml").WithId("deploy-rt-echo"),
			workflow.Apply("rt-foxtrot-1.yaml").WithId("deploy-rt-foxtrot"),
			workflow.Apply("vs.yaml").WithId("deploy-vs"),
			curl("/echo", "version:echo-v1"),
			curl("/foxtrot", "version:foxtrot-v1"),

			// Part 2: Start foxtrot v2 rollout phase 1
			workflow.Apply("foxtrot-v2.yaml").WithId("deploy-foxtrot-v2"),
			workflow.WaitForPods("foxtrot").WithId("wait-foxtrot"),
			workflow.Apply("rt-foxtrot-2.yaml").WithId("deploy-rt-foxtrot-2"),
			curl("/echo", "version:echo-v1"),
			curl("/foxtrot", "version:foxtrot-v1"),
			curlWithHeader("/foxtrot", "version:foxtrot-v2", "stage", "canary"),

			// Part 3: Start v2 foxtrot rollout phase 2
			workflow.Apply("rt-foxtrot-3.yaml").WithId("deploy-rt-foxtrot-3"),
			curl("/echo", "version:echo-v1"),
			curl("/foxtrot", "version:foxtrot-v1"),
			curlWithHeader("/foxtrot", "version:foxtrot-v2", "stage", "canary"),

			// Part 4: Start v2 echo rollout phase 1
			workflow.Apply("echo-v2.yaml").WithId("deploy-echo-v2"),
			workflow.Apply("rt-echo-2.yaml").WithId("deploy-rt-echo-2"),
			curl("/echo", "version:echo-v1"),
			curlWithHeader("/echo", "version:echo-v2", "stage", "canary"),
			curl("/foxtrot", "version:foxtrot-v1"),
			curlWithHeader("/foxtrot", "version:foxtrot-v2", "stage", "canary"),

			// Part 5: Invalid echo config, now foxtrot rollout is blocked
			workflow.Apply("rt-echo-3.yaml").WithId("deploy-rt-echo-3"),
			workflow.Apply("rt-foxtrot-4.yaml").WithId("deploy-rt-foxtrot-4"),
			curl("/echo", "version:echo-v1"),
			curlWithHeader("/echo", "version:echo-v2", "stage", "canary"),
			curl("/foxtrot", "version:foxtrot-v1"),
			curlWithHeader("/foxtrot", "version:foxtrot-v2", "stage", "canary"),

			// Part 6: Patch settings to turn on route replacement, now foxtrot rollout is unblocked
			gloo.PatchSettings("settings-patch.yaml").WithId("settings-patch"),
			curl("/foxtrot", "version:foxtrot-v2"),
			curlWithHeader("/foxtrot", "version:foxtrot-v2", "stage", "canary"),
			curl("/echo", "version:echo-v1"),
			curlRouteReplacement("/echo", "stage", "canary"),

			// Part 7: Revert typo, cleanup foxtrot v1
			workflow.Apply("rt-echo-2.yaml").WithId("deploy-rt-echo-2"),
			workflow.Apply("rt-foxtrot-5.yaml").WithId("deploy-rt-foxtrot-5"),
			workflow.Delete("foxtrot-v1.yaml").WithId("delete-foxtrot-v1"),
			curl("/echo", "version:echo-v1"),
			curlWithHeader("/echo", "version:echo-v2", "stage", "canary"),
			curl("/foxtrot", "version:foxtrot-v2"),
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