package rate_limiting_waf_and_opa

import (
	"context"
	"github.com/solo-io/valet/pkg/step/check"
	"github.com/solo-io/valet/pkg/tests"
	"github.com/solo-io/valet/pkg/workflow"
	"github.com/solo-io/valet/test/e2e/gloo"
)

const (
	// Messenger, 311
	token1 = "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzb2xvLmlvIiwic3ViIjoiMTIzNDU2Nzg5MCIsInR5cGUiOiJNZXNzZW5nZXIiLCJudW1iZXIiOiIzMTEifQ.svbQgUcAUuKHlf6U8in0O3DPGuAIQqgsPv83UIoof1ZnTjOdidqhC-i1p94bLzt67NW5NU_GICZNJU21ZRL3Dmb2ZU8Ee6t708S9rBq3z6hvHt_H-2LuYOfEmj44GqHmwAQm47p4xCaL-3DCZuoFpGUJkB6YCEf5p-r-iWYe76W7WXLqA9LJwmcnZDgasLGlFuf0sTjDzD2-dilFQhY-QFLhQ7iHjmSA6-DHqd021EhsiSrs-pb9Br9e7t39QmUqZM13SMi0VA19oyK6ORNF8zndntPf2KJ2y5M7Pf8tUi2eKTkTA_CpTjFrbsY5KsehA4V1lt-Z4QDukiVtXgSMmg"
	// Whatsapp, 311
	token2 = "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzb2xvLmlvIiwic3ViIjoiMTIzNDU2Nzg5MCIsInR5cGUiOiJXaGF0c2FwcCIsIm51bWJlciI6IjMxMSJ9.HpZKZZ6NG9Zy8R7FD87G7A6bY03xiHSub1NqADC7uCGJZM5k6Rvk4_AcKsHYrGIlSIONoPxv63gvEuesPtqP1KseBrjuNDYJ9hmgAS6E-s8IGcxhL4h5Urm_GWBlAOZbnYRBv26spEqbkpPMttmbne4mq8K8najlMMO2WbLXO0G3XSau--HTyy28rBCNrww1Nz-94Rv4brnka4rGgTb8262Qz-CJZDqhenzT9OSIkUcDTA9EkC1b3sJ_fMB1w06yzW2Ey5SCAaByf6ARtJfApmZwC6dOOlgvBw7NJQFnXOHl22r-_1gRanT2xOzWsAHjSdQjNW1ohIjyiDqrlnCKEg"
	// Whatsapp, 411
	token3 = "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzb2xvLmlvIiwic3ViIjoiMTIzNDU2Nzg5MCIsInR5cGUiOiJXaGF0c2FwcCIsIm51bWJlciI6IjQxMSJ9.nKxJufSAaW7FcM5qhUVXicn55n5tUCwVHElsnE_EfTYjveAbt7VytcrnihFZctUacrK4XguXb3HPbkb4rQ5wuS2BXoJLNJSao_9N9XtTMabGnpBp9M88dUQ7D-H2nAp-ufcbcQntl5B-gVzTcKwuWckiiMS60gdDMJ2MVcqXskeuftGGt8-Qyygi5NV5eHrlVx6I3McsBkwaw1mxgBEDhMPkgM3PTAcwfihJMdO9T25wY4APwuGB2bTyZyJ86L6xRvu-yMVHS5HouEQY--Xp-AMCbJW1Da-tyCJRBUqw8HIGEOp9wIjPNcPvZ5AZkQ1kvseSVBvtRX-QJXlHBHU6Og"
	// SMS, 200
	token4 = "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzb2xvLmlvIiwic3ViIjoiMTIzNDU2Nzg5MCIsInR5cGUiOiJTTVMiLCJudW1iZXIiOiIyMDAifQ.quxs99EylhY2Eod3Ns-NkGRAVbM3riZLQLaCHvPPcpeTn7fEmcATPL82rZvUENLX6nsj_FXetd5dpvAJwPTCTRFhnEmVlK6J9i46nNqlA2JAFwXTww4WlrrpoD6p1fGoq5cGqzqdNBrfK-uee1w5N-c5de3waLAQXK7W6_x-L-0ovAqb0wz4i-fIcTKHGELpReGCh762rrj_iMuwaZMg3SJmIfSbGB7SFfdCcY1kE8fTdwZayoxzG1EzeNFTHd7D-h1Y_odafi_PGn5zwkpU4NkBqTcPx2TbZCS5QPG9VjSgWIi5cWW1tQiPyuv7UOmjgmgZFbXXG-Uf_SBpPZdUhg"
)

func basicCurl(status int, response string) *workflow.Step {
	return &workflow.Step{
		Curl: &check.Curl{
			Service:      gloo.GatewayProxy(),
			Path:         "/sample-route-1",
			StatusCode:   status,
			ResponseBody: response,
		},
	}
}

func curlWithHeaders(status int, typeHeader, numberHeader string) *workflow.Step {
	return &workflow.Step{
		Curl: &check.Curl{
			Service:    gloo.GatewayProxy(),
			Path:       "/sample-route-1",
			StatusCode: status,
			Headers: map[string]string{
				"x-type":   typeHeader,
				"x-number": numberHeader,
			},
		},
	}
}

func curlWithToken(status int, token string) *workflow.Step {
	return &workflow.Step{
		Curl: &check.Curl{
			Service:    gloo.GatewayProxy(),
			Path:       "/sample-route-1",
			StatusCode: status,
			Headers: map[string]string{
				"x-token": token,
			},
		},
	}
}

func otherCurlWithToken(status int, token string) *workflow.Step {
	step := curlWithToken(status, token)
	step.Curl.Path = "/sample-route-2"
	return step
}

func curlForEventualRateLimit(status int, token string) *workflow.Step {
	step := curlWithToken(status, token)
	step.Curl.Attempts = 100
	step.Curl.Delay = "100ms"
	return step
}

func curlWithModsecurityIntervention() *workflow.Step {
	step := basicCurl(403, "ModSecurity: intervention occured")
	step.Curl.Headers = map[string]string{
		"User-Agent": "scammer",
	}
	return step
}

func GetWorkflow() *workflow.Workflow {
	return &workflow.Workflow{
		SetupSteps: []*workflow.Step{
			gloo.InstallGlooEnterprise(),
			gloo.DeleteAllVirtualServices(),
			gloo.GlooctlCheck(),
		},
		Steps: []*workflow.Step{
			// Part 1: Deploy the app
			workflow.Apply("petstore.yaml"),
			workflow.WaitForPods("default"),
			workflow.Apply("vs-petstore-1.yaml"),
			basicCurl(200, `[{"id":1,"name":"Dog","status":"available"},{"id":2,"name":"Cat","status":"pending"}]`),
			// Part 2: Set up initial RL
			gloo.PatchSettings("settings-patch-1.yaml"),
			workflow.Apply("vs-petstore-2.yaml"),
			basicCurl(429, ""),
			// Part 3: Set up complex rules with priority
			gloo.PatchSettings("settings-patch-2.yaml"),
			workflow.Apply("vs-petstore-3.yaml"),
			curlWithHeaders(429, "Messenger", "311"),
			curlWithHeaders(429, "Whatsapp", "311"),
			curlWithHeaders(200, "Whatsapp", "411"),
			// Part 4: Add JWT filter to set headers from JWT claims
			workflow.Apply("vs-petstore-4.yaml"),
			basicCurl(401, "Jwt is missing"),
			curlWithToken(429, token1),
			curlWithToken(429, token2),
			curlWithToken(200, token3),
			curlForEventualRateLimit(429, token3),
			// Part 5: Now add WAF to block scammers
			workflow.Apply("vs-petstore-5.yaml"),
			curlWithModsecurityIntervention(),
			// Part 6: Now add OPA to block "SMS" type
			curlWithToken(200, token4),
			workflow.Apply("allow-jwt.yaml"),
			workflow.Apply("auth-config.yaml"),
			workflow.Apply("vs-petstore-6.yaml"),
			curlWithToken(403, token4),
			curlWithToken(429, token1),
			// Part 7: Move rate limit to route level and add non-rate-limited route
			workflow.Apply("vs-petstore-7.yaml"),
			curlWithToken(429, token1),
			otherCurlWithToken(200, token1),
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