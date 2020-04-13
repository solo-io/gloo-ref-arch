package part1

import (
	"context"
	"github.com/solo-io/valet/pkg/step/check"
	"github.com/solo-io/valet/pkg/tests"
	"github.com/solo-io/valet/pkg/workflow"
	"github.com/solo-io/valet/test/e2e/gloo"
)

func curl(responseBody string) *workflow.Step {
	return &workflow.Step{
		Curl: &check.Curl{
			Service:      gloo.GatewayProxy(),
			StatusCode:   200,
			Path:         "/",
			ResponseBody: responseBody,
		},
	}
}

func curlWithHeader(responseBody, headerName, headerValue string) *workflow.Step {
	step := curl(responseBody)
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
		},
		Steps: []*workflow.Step{
			// Part 1: Deploy the app
			workflow.Apply("echo.yaml").WithId("deploy-echo"),
			workflow.WaitForPods("echo").WithId("wait-1"),
			workflow.Apply("upstream.yaml").WithId("deploy-upstream"),
			workflow.Apply("vs-1.yaml").WithId("deploy-vs-1"),
			curl("version:v1"),

			// Part 2: Initial subset routing
			workflow.Apply("vs-2.yaml").WithId("deploy-vs-2"),
			curl("version:v1"),

			// Part 3: Deploy v2 with subset route
			workflow.Apply("echo-v2.yaml").WithId("deploy-echo-v2"),
			workflow.WaitForPods("echo").WithId("wait-1"),
			workflow.Apply("vs-3.yaml").WithId("deploy-vs-3"),
			curl("version:v1"),
			curlWithHeader("version:v2", "stage", "canary"),

			// Part 4: Setup weighted destinations, 0% to v2
			workflow.Apply("vs-4.yaml").WithId("deploy-vs-4"),
			curl("version:v1"),
			curlWithHeader("version:v2", "stage", "canary"),

			// Part 5: Start shift, 50% to v1 and 50% to v2
			workflow.Apply("vs-5.yaml").WithId("deploy-vs-5"),
			curl("version:v1"),
			curl("version:v2"),

			// Part 6: Finish shift, 100% to v2
			workflow.Apply("vs-6.yaml").WithId("deploy-vs-6"),
			curl("version:v2"),

			// Part 7: Decommission v1
			workflow.Delete("echo-v1.yaml").WithId("delete-echo-v1"),
			curl("version:v2"),

			// Part 8: Cleanup routes
			workflow.Apply("vs-7.yaml").WithId("deploy-vs-7"),
			curl("version:v2"),
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