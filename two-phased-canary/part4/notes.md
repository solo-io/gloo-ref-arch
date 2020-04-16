# Two-phased canary rollout with Gloo, part 4

In the first part of this series, we tried to come up with a realistic workflow that you could use to perform canary testing and 
progressive delivery, so teams can safely deliver new versions of their services in your production environment. 
In the second part, we looked at how we can scale this to potentially many teams, while maintaining a clean separation between domain and
route-level configuration, to minimize the amount of configuration dev teams need to maintain to facilitate these workflows. 

In the third part, we showed how teams can start to take advantages of other features in Gloo, such as traffic shadowing, 
by enabling options on their routes. 

In this part, we're going to create a helm chart that our development teams can use for deploying applications to 
Gloo on their Kubernetes cluster. This means they can install the application by providing a few helm values, and 
they can invoke the canary upgrade workflow by performing helm upgrades. Doing this provides a number of benefits:
* It lowers the barrier to entry; the workflows are really easy to execute. 
* It drastically reduces the amount of (often) copy-pasted configuration teams need to maintain.
* It provides nice guard rails to minimize the potential for misconfigurations. 
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

* Name ("echo"): this will be provided in the helm command.
* Namespace ("echo"): this will be provided in the helm command. 
* Version ("v1"): this will default to "v1"; in the future, we'll want to start incrementing this. 
* ApiGroup ("example"): this will be the label we use on route tables so it matches our virtual service selector. 

### Updating the resources

We'll update all the resources to use these values (for example, "{{ .Release.Name }}", or "{{ .Values.appVersion }}"). 
And we'll separate the templates into their own file to make it more clear. 

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

We can start by designing some values that might be able to express different points in the workflow.

### Initial state

```yaml
deployment:
  v1: {} 
routes:
  apiGroup: example
  primary:
    version: v1
```

Deploy v2

```yaml
deployment:
  v1: {} 
  v2: {} 
routes:
  apiGroup: example
  primary:
    version: v1
```

### Phase 1

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
    weight: 0 
    headers: 
      stage: canary
```

### Phase 2

Beginning

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

Middle

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

End

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

### Decommission v1

```yaml
deployment:
  v2: {} # for now, we just need the list of versions that should be deployed
routes:
  apiGroup: example
  primary:
    version: v2
```

## Building it into the chart

Now that we know how we want to express our values, we can update the templates. 

### Deployment

We need to update the deployment to create one resource for each entry in the values map. So far, we haven't 
populated the value of this map with anything, so our examples look like this:
```yaml
deployment:
  v1: {}
``` 

It's highly likely we would need to customize the deployment object for each version, for instance to 
pick the desired image. But for now, we'll skip that. 

```yaml

``` 
