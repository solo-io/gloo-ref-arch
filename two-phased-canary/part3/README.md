# Two-phased canary rollout with Gloo, part 3

In the first part of this series, we tried to come up with a robust workflow that you could use to perform canary testing and 
progressive delivery, so teams can safely deliver new versions of their services to users in a production environment. 

In the second part, we looked at how we can scale this to potentially many teams, while maintaining a clean separation between domain and
route-level configuration. This helps minimize the amount of configuration dev teams need to maintain to facilitate these workflows. 

In this part, we're going to create a helm chart that our development teams can use for deploying applications to 
Gloo on their Kubernetes cluster. This means they can install the application by providing a few helm values, and 
they can invoke the canary upgrade workflow by performing helm upgrades. Doing this provides a number of benefits:
* It lowers the barrier to entry; the workflow is really easy to execute. 
* It drastically reduces the amount of (often) copy-pasted configuration different teams need to maintain.
* It provides nice guard rails to minimize the potential for misconfiguration. 
* It becomes trivial to integrate into a GitOps / continuous delivery workflow.

As a disclaimer, a lot of services ultimately require fairly extensive configuration, some of which is unique to the 
particular use case. However, to the extent your teams can follow standard conventions, those conventions can be encoded
in a helm chart like we do here. 

## Breaking down our requirements

In the last part, we deployed each service -- echo and foxtrot -- in a single step by applying a manifest, containing five resources: 
a namespace, deployment, service, gloo upstream, and gloo route table. 

We are going to drop the namespace from the manifest, and rely on Helm or other automation to create it for us. The other 
four resources will be the four templates in our helm chart. 

Let's assume the services that we're deploying with our chart are all single-container pods, using the standard http 
port to listen for traffic. Later, we could improve our chart to support config maps, managing certificates and 
secrets, and setting up things like readiness probes or resource limits. 

## Creating our chart

Helm includes a tool for creating a chart. After running `helm create gloo-app`, you'll get a directory containing 
a `Chart.yaml`, a default `values.yaml`, an empty crds directory, and a templates directory with a number of files. 

We'll update the `Chart.yaml`, delete the empty crds directory, and we'll replace all the files in the templates directory
with the `echo.yaml` manifest from the last part. 

We can make sure this renders by running the following command:

`helm install echo --dry-run --debug ./gloo-app --namespace echo`

If that succeeds, we're ready to start extracting values.  

### Values

We're going to extract the bare minimum values so that the chart can (a) also deploy a foxtrot service, and (b) start
to enable our canary testing workflow. There are four values we need to provide to our application:

* Service name ("echo"): We'll take this from the name of the Helm release, and in our templates access it with 
`{{ .Release.Name }}`
* Namespace ("echo"): this will also be provided by the Helm command, so we'll access it with `{{ .Release.Namespace }}`. 
* App Version ("v1"): this will determine the version label to use on the pods for subset routing. For now, we'll access it
with `{{ .Values.appVersion }}`. 
* Api Group ("example"): this will be the label we use on route tables so it matches our virtual service selector, and for 
now we'll use `{{ .Values.apiGroup }}`

### Updating the resources

Now we can modify our manifest to use these template values instead of hard coded values. We'll also split each 
resource into it's own file: 

`deployment.yaml`:
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ .Release.Name }}-{{ .Values.appVersion }}
  namespace: {{ .Release.Namespace }}
spec:
  replicas: 1
  selector:
    matchLabels:
      app: {{ .Release.Name }}
      version: {{ .Values.appVersion }}
  template:
    metadata:
      labels:
        app: {{ .Release.Name }}
        version: {{ .Values.appVersion }}
    spec:
      containers:
        - image: hashicorp/http-echo
          args:
            - "-text=version:{{ .Release.Name }}-{{ .Values.appVersion }}"
            - -listen=:8080
          imagePullPolicy: Always
          name: {{ .Release.Name }}-{{ .Values.appVersion }}
          ports:
            - containerPort: 8080
```

`service.yaml`:
```yaml
apiVersion: v1
kind: Service
metadata:
  name: {{ .Release.Name }}
  namespace: {{ .Release.Namespace }}
spec:
  ports:
    - port: 80
      targetPort: 8080
      protocol: TCP
  selector:
    app: {{ .Release.Name }}
```

`upstream.yaml`:
```yaml
apiVersion: gloo.solo.io/v1
kind: Upstream
metadata:
  name: {{ .Release.Name }}
  namespace: {{ .Release.Namespace }}
spec:
  kube:
    selector:
      app: {{ .Release.Name }}
    serviceName: {{ .Release.Name }}
    serviceNamespace: {{ .Release.Namespace }}
    servicePort: 8080
    subsetSpec:
      selectors:
        - keys:
            - version
```

`routetable.yaml`:
```yaml
apiVersion: gateway.solo.io/v1
kind: RouteTable
metadata:
  name: {{ .Release.Name }}-routes
  namespace: {{ .Release.Namespace }}
  labels:
    apiGroup: {{ .Values.apiGroup }}
spec:
  routes:
    - matchers:
        - prefix: "/{{ .Release.Name }}"
      routeAction:
        single:
          upstream:
            name: {{ .Release.Name }}
            namespace: {{ .Release.Namespace }}
          subset:
            values:
              version: {{ .Values.appVersion }}
```

Finally, here are our default values:

```yaml
appVersion: v1
apiGroup: example
```

We should now have the pieces we need to deploy echo and foxtrot with the following commands:

```
kubectl create ns echo
helm install echo  ./gloo-app --namespace echo
kubectl create ns foxtrot
helm install foxtrot  ./gloo-app --namespace foxtrot
```

## Modeling our workflow 

Now that we have a base chart, we need to expose values that help execute our phased upgrade workflow. 
Remember we had the following phases in our rollout strategy:
* Phase 1: canary test using a special header that routes a targeted, small amount of traffic to the new version for functional testing
* Phase 2: shift load to the new version while monitoring performance

We can start by designing some values that might be able to express different points in the workflow. In particular, 
if we think about which resources we need to modify during our workflow, we have two main sections in our values:
* Deployments: this section will be used to control which versions of the deployment to install. We may want to deploy a single 
version, or multiple. It may be desirable to deploy a version without routing to it at all, for instance to do shadowing.
It also may be desirable to configure different versions differently, so we'll design with that assumption.  
* Routes: this section will be used to configure which stage of the flow we are in. We'll use `routes.primary` to configure 
our main route, and `routes.canary` to make all the changes necessary for different stages of the canary workflow. 

Given these requirements, we'll make our `deployment` helm value a map, where the key is the version name and the value
is the version-specific configuration. Then, to configure our primary and canary version, we'll use these keys to specify 
a version. 

We'll also use a simple approach to phase 2. We'll add a `routes.canary.weight` parameter so the canary destination 
can be given a weight between 0 and 100. The primary weight will be computed as `100 - routes.canary.weight`. 

### Canary workflow, expressed as Helm values

Now that we've settled on a design, let's actually express this workflow in a series of Helm values. 

First, we have our starting state, where we deploy v1 of the application: 
```yaml
deployment:
  v1: {} 
routes:
  apiGroup: example
  primary:
    version: v1
```

Next, we can deploy v2, but not yet set up any canary routes:
```yaml
deployment:
  v1: {} 
  v2: {} 
routes:
  apiGroup: example
  primary:
    version: v1
```

Now, we can start Phase 1, and provide a canary configuration:
```yaml
deployment:
  v1: {} 
  v2: {} 
routes:
  apiGroup: example
  primary:
    version: v1
  canary:
    version: v2
    headers: 
      stage: canary
```

When we are set to begin phase 2, we can start to adjust the weight of the canary route. Note that the previous values 
are equivalent to this (0 weight) for simplicity:
```yaml
deployment:
  v1: {} # we'll most likely want version-specific deployment configuration in the future;
  v2: {} # for now, we just need the list of versions that should be deployed
routes:
  apiGroup: example
  primary:
    version: v1
  canary:
    version: v2
    weight: 0 # 0 to 100 to facilitate traffic shift
    headers: 
      stage: canary
```

When we are halfway through the workflow, our values may look like this: 
```yaml
deployment:
  v1: {} # we'll most likely want version-specific deployment configuration in the future;
  v2: {} # for now, we just need the list of versions that should be deployed
routes:
  apiGroup: example
  primary:
    version: v1
  canary:
    version: v2
    weight: 50 # 0 to 100 to facilitate traffic shift
    headers: 
      stage: canary
```

Finally, we have shifted all the traffic to the canary version:
```yaml
deployment:
  v1: {} # we'll most likely want version-specific deployment configuration in the future;
  v2: {} # for now, we just need the list of versions that should be deployed
routes:
  apiGroup: example
  primary:
    version: v1
  canary:
    version: v2
    weight: 100 # 0 to 100 to facilitate traffic shift
    headers: 
      stage: canary
```

And the last step is to decommission the old version:
```yaml
deployment:
  v2: {} # for now, we just need the list of versions that should be deployed
routes:
  apiGroup: example
  primary:
    version: v2
```

Fantastic - we can now express each step of our workflow in declarative helm values. This means that once we update the 
chart, we'll be able to execute our workflow simply through running `helm upgrade`. 

## Building it into the chart

Now that we know how we want to express our values, we can update the templates. 

### Deployment

Our deployment template needs to be updated now that `deployment` is a map of version to configuration. We want to create 
one deployment resource for each version in the map, so we'll wrap our template to range over the deployment value. Here's 
what that looks like:

```yaml
{{- $relname := .Release.Name -}}
{{- $relns := .Release.Namespace -}}
{{- range $version, $config := .Values.deployment }}
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ $relname }}-{{ $version }}
  namespace: {{ $relns }}
spec:
  replicas: 1
  selector:
    matchLabels:
      app: {{ $relname }}
      version: {{ $version }}
  template:
    metadata:
      labels:
        app: {{ $relname }}
        version: {{ $version }}
    spec:
      containers:
        - image: hashicorp/http-echo
          args:
            - "-text=version:{{ $relname }}-{{ $version }}"
            - -listen=:8080
          imagePullPolicy: Always
          name: {{ $relname }}-{{ $version }}
          ports:
            - containerPort: 8080
---
{{- end }}
```

A few notes about this:
* Using range affects the scope, so `.Release.Name` and `.Release.Namespace` aren't available inside the range. Instead, 
we'll save those to variables that we can access. 
* We added a YAML separator since we'll sometimes end up generating multiple objects.

Otherwise, this looks very similar to our template from before. 

### Route table

This template will take a little more work to start using the values we provided, though we'll still end up with a single
route table and won't need to create any variables for ranges. 

```yaml
apiVersion: gateway.solo.io/v1
kind: RouteTable
metadata:
  name: {{ .Release.Name }}-routes
  namespace: {{ .Release.Namespace }}
  labels:
    apiGroup: {{ .Values.routes.apiGroup }}
spec:
  routes:
    {{- if .Values.routes.canary }}
    - matchers:
        - headers:
            {{- range $headerName, $headerValue := .Values.routes.canary.headers }}
            - name: {{ $headerName }}
              value: {{ $headerValue }}
            {{- end }}
          prefix: "/{{ .Release.Name }}"
      routeAction:
        single:
          upstream:
            name: {{ .Release.Name }}
            namespace: {{ .Release.Namespace }}
          subset:
            values:
              version: {{ .Values.routes.canary.version }}
    {{- end }}
    - matchers:
        - prefix: "/{{ .Release.Name }}"
      routeAction:
        {{- if .Values.routes.canary }}
        multi:
          destinations:
            - destination:
                upstream:
                  name: {{ .Release.Name }}
                  namespace: {{ .Release.Namespace }}
                subset:
                  values:
                    version: {{ .Values.routes.primary.version }}
              weight: {{ sub 100 .Values.routes.canary.weight }}
            - destination:
                upstream:
                  name: {{ .Release.Name }}
                  namespace: {{ .Release.Namespace }}
                subset:
                  values:
                    version: {{ .Values.routes.canary.version }}
              weight: {{ add 0 .Values.routes.canary.weight }}
        {{- else }}
        single:
          upstream:
            name: {{ .Release.Name }}
            namespace: {{ .Release.Namespace }}
          subset:
            values:
              version: {{ .Values.routes.primary.version }}
        {{- end }}
``` 

Let's go section by section. The only change we made to the metadata at the top is we moved the `apiGroup` to a new
Helm value. 

Things start to get interesting in the `routes` section. First, we've added a conditional block that will render 
the canary route:

```yaml
    {{- if .Values.routes.canary }}
    - matchers:
        - headers:
            {{- range $headerName, $headerValue := .Values.routes.canary.headers }}
            - name: {{ $headerName }}
              value: {{ $headerValue }}
            {{- end }}
          prefix: "/{{ .Release.Name }}"
      routeAction:
        single:
          upstream:
            name: {{ .Release.Name }}
            namespace: {{ .Release.Namespace }}
          subset:
            values:
              version: {{ .Values.routes.canary.version }}
    {{- end }}
```

This route will be included as long as `canary` isn't nil. It'll use the `version` and `headers` values we 
 introduced to specify how the matching and subset routing should work. 
 
We also need to customize our route action on our other route, because during Phase 2, we change it from a single to a multi 
destination and establish weights:
```yaml 
      routeAction:
        {{- if .Values.routes.canary }}
        multi:
          destinations:
            - destination:
                upstream:
                  name: {{ .Release.Name }}
                  namespace: {{ .Release.Namespace }}
                subset:
                  values:
                    version: {{ .Values.routes.primary.version }}
              weight: {{ sub 100 .Values.routes.canary.weight }}
            - destination:
                upstream:
                  name: {{ .Release.Name }}
                  namespace: {{ .Release.Namespace }}
                subset:
                  values:
                    version: {{ .Values.routes.canary.version }}
              weight: {{ add 0 .Values.routes.canary.weight }}
        {{- else }}
        single:
          upstream:
            name: {{ .Release.Name }}
            namespace: {{ .Release.Namespace }}
          subset:
            values:
              version: {{ .Values.routes.primary.version }}
        {{- end }}
```

To keep it simple, we'll switch to multi-destination as soon as a canary value is provided. Unless a user explicitly
adds a weight to the canary, this will have no effect and all the traffic will continue to be routed to the primary 
version. This just keeps our template and values minimal. 

The other interesting note is that we are using `Helm` arithmetic functions on the weights, so that we always end up
with a total weight of 100, and so that a user only needs to specify a single weight in the values. 

Finally, we'll make sure the other templates are updated to use our new value structure. And with that, we're now ready 
to start testing our chart. 

## Testing helm charts, the abridged version

There are different schools of though on how to test Helm charts. In Gloo, we've created some libraries so we can make sure
that charts render with the expected resources when different sets of values are provided. Helm 3 also has some built in 
starting points for testing. In practice, a chart that is used for production should be well tested, but for this post we'll 
skip unit testing. 

The other technique we may use when developing these templates is regularly running the `helm install` command with the 
`--dry-run` flag, to ensure the template doesn't have a syntax error and the resources render as we expect. 

## Running the workflow with our new chart

We're now ready to execute our entire canary workflow, by performing a helm install and then upgrades and changing the
values. 

I'll assume we have Gloo deployed on a Kubernetes cluster, and don't have any existing virtual services. If you had
`echo` and `foxtrot` installed from before, just run `kubectl delete ns echo foxtrot`. We'll start by deploying our generic 
virtual service for our `example` API group. Then, as we install new services with our helm chart, these routes will 
automatically be picked up. 

```yaml
apiVersion: gateway.solo.io/v1
kind: VirtualService
metadata:
  name: app
  namespace: gloo-system
spec:
  virtualHost:
    domains:
      - '*'
    routes:
      - matchers:
          - prefix: /
        delegateAction:
          selector:
            labels:
              apiGroup: example
            namespaces:
              - "*"
```

We can apply this to the cluster with the following command:

```
kubectl apply -f vs.yaml
```

Now we're ready to start deploying services with our chart. 

### Install echo

Let's create a namespace for echo. 
```
kubectl create ns echo
```

Now we can install with our initial values:
```yaml
deployment:
  v1: {}
routes:
  apiGroup: example
  primary:
    version: v1
```

We'll issue the following commands to install with helm:
```
➜ k create ns echo
namespace/echo created

➜ helm install echo  ./gloo-app --namespace echo -f values-1.yaml
NAME: echo
LAST DEPLOYED: Thu Apr 16 16:04:29 2020
NAMESPACE: echo
STATUS: deployed
REVISION: 1
TEST SUITE: None
```

And we can verify things are healthy, and test the route:
``` 
➜ glooctl check
Checking deployments... OK
Checking pods... OK
Checking upstreams... OK
Checking upstream groups... OK
Checking auth configs... OK
Checking secrets... OK
Checking virtual services... OK
Checking gateways... OK
Checking proxies... OK
No problems detected.

➜ curl $(glooctl proxy url)/echo
version:echo-v1
```

Great! Our installation is complete. We were able to bring online a new route to a new service in Gloo by installing 
our chart. 

For good measure, let's also install foxtrot:
```
➜ kubectl create ns foxtrot
namespace/foxtrot created

➜ helm install foxtrot  ./gloo-app --namespace foxtrot -f values-1.yaml
NAME: foxtrot
LAST DEPLOYED: Thu Apr 16 16:08:23 2020
NAMESPACE: foxtrot
STATUS: deployed
REVISION: 1
TEST SUITE: None

➜ curl $(glooctl proxy url)/foxtrot
version:foxtrot-v1
```

This is a huge improvement from before, where we were copy pasting a ton of yaml, and needed to do a lot more manual work. 
Let's see how this extends to driving our upgrade workflow. 

### Starting the upgrade to echo-v2

We'll use the following values for our helm upgrade, to initially deploy echo-v2. Note that this doesn't yet create any
canary routes, it simply adds the new deployment:
```yaml
deployment:
  v1: {}
  v2: {}
routes:
  apiGroup: example
  primary:
    version: v1
```
  
With a `helm upgrade`, we can execute this step:
``` 
➜ helm upgrade echo --values values-2.yaml  ./gloo-app --namespace echo
Release "echo" has been upgraded. Happy Helming!
NAME: echo
LAST DEPLOYED: Thu Apr 16 16:12:03 2020
NAMESPACE: echo
STATUS: deployed
REVISION: 4
TEST SUITE: None
```

We can see that the new deployment has been created and a v2 pod is now running:
``` 
➜ kubectl get pods -n echo
NAME                       READY   STATUS    RESTARTS   AGE
echo-v1-8569c78bcf-dtl6m   1/1     Running   0          9m5s
echo-v2-9c4b669cd-ldkdv    1/1     Running   0          90s
```

And our route is still working:
``` 
➜ curl $(glooctl proxy url)/echo
version:echo-v1
```

### Entering phase 1

Now we want to set up a route to match on the header `stage: canary` and route to the new version. All other 
requests should continue to route to the old version. We can deploy that with another `helm upgrade`. We'll 
use these values:
```yaml
deployment:
  v1: {}
  v2: {}
routes:
  apiGroup: example
  primary:
    version: v1
  canary:
    version: v2
    headers:
      stage: canary
```

We'll upgrade with this command:
``` 
➜ helm upgrade echo --values values-3.yaml  ./gloo-app --namespace echo
Release "echo" has been upgraded. Happy Helming!
NAME: echo
LAST DEPLOYED: Thu Apr 16 16:16:29 2020
NAMESPACE: echo
STATUS: deployed
REVISION: 5
TEST SUITE: None
```

Now, in our testing, we'll be able to start using the canary route:
``` 
➜ curl $(glooctl proxy url)/echo
version:echo-v1

➜ curl $(glooctl proxy url)/echo -H "stage: canary"
version:echo-v2
```

### Entering phase 2

As we discussed above, the last set of values will set up our weighted destinations for phase 2, but will set the weight 
to 0 on the canary route. So now, we can do another helm upgrade to change the weights. If we want to change the weights 
so 50% of the traffic goes to the canary upstream, we can use these values:
```yaml
deployment:
  v1: {}
  v2: {}
routes:
  apiGroup: example
  primary:
    version: v1
  canary:
    version: v2
    weight: 50
    headers:
      stage: canary
```

And here is our command to upgrade:
``` 
➜ helm upgrade echo --values values-4.yaml  ./gloo-app --namespace echo
Release "echo" has been upgraded. Happy Helming!
NAME: echo
LAST DEPLOYED: Thu Apr 16 16:19:26 2020
NAMESPACE: echo
STATUS: deployed
REVISION: 6
TEST SUITE: None
```

Now the routes are behaving as we expect. The canary route is still there, and we've shifted half the traffic on 
our primary route to the new version:
``` 
➜ curl $(glooctl proxy url)/echo -H "stage: canary"
version:echo-v2

➜ curl $(glooctl proxy url)/echo
version:echo-v2

➜ curl $(glooctl proxy url)/echo
version:echo-v1
```

### Finishing phase 2

Let's update the routes so 100% of the traffic goes to the new version. We'll use these values:
```yaml
deployment:
  v1: {}
  v2: {}
routes:
  apiGroup: example
  primary:
    version: v1
  canary:
    version: v2
    weight: 100
    headers:
      stage: canary
```

And we can deploy with this command:
``` 
➜ helm upgrade echo --values values-5.yaml  ./gloo-app --namespace echo
Release "echo" has been upgraded. Happy Helming!
NAME: echo
LAST DEPLOYED: Thu Apr 16 16:21:37 2020
NAMESPACE: echo
STATUS: deployed
REVISION: 7
TEST SUITE: None
```

When we test the routes, we now see all the traffic going to the new version:
``` 
➜ curl $(glooctl proxy url)/echo
version:echo-v2

➜ curl $(glooctl proxy url)/echo
version:echo-v2

➜ curl $(glooctl proxy url)/echo
version:echo-v2

➜ curl $(glooctl proxy url)/echo  -H "stage: canary"
version:echo-v2
```

### Decommissioning v1

The final step in our workflow is to decommission the old version. Our values reduce to this:
```yaml
deployment:
  v2: {}
routes:
  apiGroup: example
  primary:
    version: v2
```

We can upgrade with this command:
``` 
➜ helm upgrade echo --values values-6.yaml  ./gloo-app --namespace echo
Release "echo" has been upgraded. Happy Helming!
NAME: echo
LAST DEPLOYED: Thu Apr 16 16:24:05 2020
NAMESPACE: echo
STATUS: deployed
REVISION: 8
TEST SUITE: None
```

We can make sure our route is health:
``` 
➜ curl $(glooctl proxy url)/echo
version:echo-v2
```

And we can make sure our old pod was cleaned up:
``` 
➜ kubectl get pod -n echo
NAME                      READY   STATUS    RESTARTS   AGE
echo-v2-9c4b669cd-ldkdv   1/1     Running   0          12m
```

## Closing thoughts

And that's it! We've installed our service using Helm, and then executed our entire two-phased upgrade workflow 
by performing `helm upgrade` and customizing the values. 

From here, we can now look at two different improvements. 

First, we'll need to introduce a lot more values to our chart in order to start supporting more workflows, routing features, 
 and developer use cases. There 
are potentially many things we'd need to customize, such as the deployment image, resource limits, options enabled, and so forth. 
We also may need to support 
multiple routes and alternative paths. Hopefully, at this point you are convinced that we could tackle those
as extensions to our work so far. We'll tackle one of these in the next part.  

Second, we're now ready to integrate fully into a CI/CD process. Many users interact with their production clusters by 
customizing helm values and using automation, such as the Helm operator or the Flux project, to help deliver those 
updates to the cluster when the source of truth (usually a git repo) has changed. Onboarding and upgrading new services 
to our platform has now become trivial with Helm, and can become automated through GitOps. 

## Get Involved in the Gloo Community

Gloo has a large and growing community of open source users, in addition to an enterprise customer base. To learn more about 
Gloo:
* Check out the [repo](https://github.com/solo-io/gloo), where you can see the code and file issues
* Check out the [docs](https://docs.solo.io/gloo/latest), which have an extensive collection of guides and examples
* Join the [slack channel](http://slack.solo.io/) and start chatting with the Solo engineering team and user community

If you'd like to get in touch with me (feedback is always appreciated!), you can find me on Slack or email me at 
**rick.ducott@solo.io**. 