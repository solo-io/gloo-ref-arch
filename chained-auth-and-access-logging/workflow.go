package chained_auth_and_access_logging

import (
	"context"
	"github.com/solo-io/valet/pkg/render"
	"github.com/solo-io/valet/pkg/step/check"
	"github.com/solo-io/valet/pkg/step/kubectl"
	"github.com/solo-io/valet/pkg/tests"
	"github.com/solo-io/valet/pkg/workflow"
	"github.com/solo-io/valet/test/e2e/gloo"
)

func initialCurl() *workflow.Step {
	return &workflow.Step{
		Curl: &check.Curl{
			Service:    gloo.GatewayProxy(),
			Path:       "/",
			StatusCode: 200,
			Attempts:   30,
		},
	}
}

func accessLoggingPatch() *workflow.Step {
	return &workflow.Step{
		Patch: &kubectl.Patch{
			Namespace: "gloo-system",
			Name:      "gateway-proxy",
			KubeType:  "gateway",
			PatchType: "merge",
			Path:      "gateway-patch.yaml",
		},
	}
}

func turnOnExtauthDebugLogging() *workflow.Step {
	return &workflow.Step{
		Curl: &check.Curl{
			Path:        "/logging",
			StatusCode:  200,
			Method:      "PUT",
			RequestBody: `{ "level": "debug" }`,
			PortForward: &check.PortForward{
				Namespace:      "gloo-system",
				DeploymentName: "extauth",
				Port:           9091,
			},
		},
	}
}

func GetWorkflow() *workflow.Workflow {
	return &workflow.Workflow{
		Values: render.Values{
			"ClientSecret": "env:GOOGLE_CLIENT_SECRET",
			"ClientId":     "env:GOOGLE_CLIENT_ID",
		},
		SetupSteps: []*workflow.Step{
			gloo.InstallGlooEnterpriseWithValues("values.yaml"),
			gloo.DeleteAllVirtualServices(),
			gloo.GlooctlCheck(),
		},
		Steps: []*workflow.Step{
			// Part 1: Deploy the monolith
			workflow.Apply("petclinic.yaml").WithId("deploy-monolith"),
			workflow.WaitForPods("default").WithId("wait-1"),
			workflow.Apply("vs-1.yaml").WithId("vs-1"),
			initialCurl(),

			// Part 2: Deploy access loggers
			accessLoggingPatch(),

			// Part 3: Deploy auth configs and debug logging for extauth
			workflow.ApplyTemplate("oauth-secret.tmpl"),
			workflow.Apply("allow-jwt.yaml"),
			workflow.ApplyTemplate("auth-config.tmpl"),
			workflow.Apply("vs-2.yaml"),
			turnOnExtauthDebugLogging(),

			// Make sure everything is healthy
			gloo.GlooctlCheck(),
		},
	}
}

func GetTestWorkflow() *tests.TestWorkflow {
	return &tests.TestWorkflow{
		Workflow:          GetWorkflow(),
		Ctx:               workflow.DefaultContext(context.TODO()),
		TestSerialization: true,
	}
}
