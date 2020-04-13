package petclinic

import (
	"context"
	"github.com/solo-io/valet/pkg/step/check"
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

func curlVetsForUpdate() *workflow.Step {
	return &workflow.Step{
		Curl: &check.Curl{
			Service:               gloo.GatewayProxy(),
			Path:                  "/vets.html",
			StatusCode:            200,
			ResponseBodySubstring: "Boston",
			Attempts:              30,
		},
	}
}

func curlContactPageForFix() *workflow.Step {
	return &workflow.Step{
		Curl: &check.Curl{
			Service:               gloo.GatewayProxy(),
			Path:                  "/contact.html",
			StatusCode:            200,
			ResponseBodySubstring: "Enter your email",
			Attempts:              30,
		},
	}
}

func GetWorkflow() *workflow.Workflow {
	return &workflow.Workflow{
		SetupSteps: []*workflow.Step{
			gloo.InstallGloo(),
			gloo.DeleteAllVirtualServices(),
			gloo.GlooctlCheck(),
		},
		Steps: []*workflow.Step{
			// Part 1: Deploy the monolith
			workflow.Apply("petclinic.yaml").WithId("deploy-monolith"),
			workflow.WaitForPods("default").WithId("wait-1"),
			workflow.Apply("vs-1.yaml").WithId("vs-1"),
			initialCurl(),
			// Part 2: Extend with a new microservice
			workflow.Apply("petclinic-vets.yaml").WithId("deploy-vets"),
			workflow.WaitForPods("default").WithId("wait-2"),
			workflow.Apply("vs-2.yaml").WithId("vs-2"),
			curlVetsForUpdate(),
			// Phase 3: AWS
			gloo.CreateAwsSecret().WithId("aws-creds"),
			workflow.Apply("upstream-aws.yaml").WithId("upstream-aws"),
			workflow.Apply("vs-3.yaml").WithId("vs-3"),
			curlContactPageForFix(),
		},
	}
}

func GetTestWorkflow() *tests.TestWorkflow {
	return &tests.TestWorkflow{
		Workflow:          GetWorkflow(),
		Ctx:               workflow.DefaultContext(context.TODO()),
		TestSerialization: true,
		TestDocs:          true,
	}
}
