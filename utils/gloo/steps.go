package gloo

import (
	"fmt"
	"github.com/solo-io/valet/pkg/step/check"
	"github.com/solo-io/valet/pkg/step/helm"
	"github.com/solo-io/valet/pkg/step/kubectl"
	"github.com/solo-io/valet/pkg/step/script"
	"github.com/solo-io/valet/pkg/workflow"
	"strings"
)

func GatewayProxy() *check.ServiceRef {
	return &check.ServiceRef{
		Namespace: "gloo-system",
		Name:      "gateway-proxy",
	}
}

func CreateAwsSecret() *workflow.Step {
	return &workflow.Step{
		CreateSecret: &kubectl.CreateSecret{
			Namespace: "gloo-system",
			Name:      "aws-creds",
			Type:      "generic",
			Entries: map[string]kubectl.SecretValue{
				"aws_access_key_id":     {EnvVar: "AWS_ACCESS_KEY_ID"},
				"aws_secret_access_key": {EnvVar: "AWS_SECRET_ACCESS_KEY"},
			},
		},
	}
}

func GlooctlCheck() *workflow.Step {
	return &workflow.Step{
		Bash: &script.Bash{
			Inline: "glooctl check",
		},
	}
}


func InstallGloo() *workflow.Step {
	return &workflow.Step{
		InstallHelmChart: &helm.InstallHelmChart{
			ReleaseName: "gloo",
			ReleaseUri:  "https://storage.googleapis.com/solo-public-helm/charts/gloo-1.3.17.tgz",
			Namespace:   "gloo-system",
			WaitForPods: true,
		},
	}
}

func InstallGlooEnterprise() *workflow.Step {
	return &workflow.Step{
		InstallHelmChart: &helm.InstallHelmChart{
			ReleaseName: "gloo",
			ReleaseUri:  "https://storage.googleapis.com/gloo-ee-helm/charts/gloo-ee-1.3.0.tgz",
			Namespace:   "gloo-system",
			WaitForPods: true,
			Set: map[string]string{
				"license_key": "env:LICENSE_KEY",
			},
		},
	}
}

func InstallGlooEnterpriseWithValues(values string) *workflow.Step {
	installStep := InstallGlooEnterprise()
	installStep.InstallHelmChart.ValuesFiles = []string{values}
	return installStep
}

func PatchSettings(path string) *workflow.Step {
	return &workflow.Step{
		Patch: &kubectl.Patch{
			Name:      "default",
			Namespace: "gloo-system",
			KubeType:  "settings",
			PatchType: "merge",
			Path:      path,
		},
	}
}

func DeleteAllVirtualServices() *workflow.Step {
	return &workflow.Step{
		Bash: &script.Bash{
			Inline: "kubectl delete virtualservices.gateway.solo.io -n gloo-system --all",
		},
	}
}

func DeleteNamespaces(ns ...string) *workflow.Step {
	joinedNs := strings.Join(ns, " ")
	return &workflow.Step{
		Bash: &script.Bash{
			Inline: fmt.Sprintf("kubectl delete ns %s --ignore-not-found", joinedNs),
		},
	}
}