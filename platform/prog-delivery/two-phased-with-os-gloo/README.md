# Two-phased canary rollout with Open Source Gloo

As engineering organizations move towards microservices, containers, and Kubernetes, they are leveraging API gateways and
service mesh technology to help with those migrations, and to unlock new capabilities for managing, observing, and securing 
traffic to their application. 

A common operational question to consider when building out your application platform is: how to migrate user traffic to a new
version of a service? This is sometimes referred to as a canary rollout or progressive delivery. In this post, we'll look at 
how Gloo can be used as an API gateway to facilitate canary rollouts of new user-facing services in Kubernetes. 

## Initial setup

To start, we need a Kubernetes cluster. This scenario doesn't take advantage of any cloud specific 
features, and can be run against a local test cluster such as [minikube](https://kubernetes.io/docs/tasks/tools/install-minikube/). 
This post assumes a basic understanding of Kubernetes and how to interact with it using `kubectl`. 

To start, we'll install the latest [open source Gloo](https://github.com/solo-io/gloo) to the `gloo-system` namespace and deploy 
 version `v1` of an example application to the `echo` namespace. We'll expose this application outside the cluster 
 by creating a route in Gloo, to end up with a picture like this: 

![](1-setup/setup.png)

### Deploying Gloo

To start, we'll deploy open source Gloo. We can get the latest `glooctl` command line tool downloaded and added to the path
by running:

```bash
curl -sL https://run.solo.io/gloo/install | sh
export PATH=$HOME/.gloo/bin:$PATH
```

Now, you should be able to run `glooctl version` to see that it is installed correctly:

```bash
➜ glooctl version
Client: {"version":"1.3.15"}
Server: version undefined, could not find any version of gloo running
```

Now we can install the gateway to our cluster with a simple command:

```bash
glooctl install gateway
```

The console should indicate the install finishes successfully:

```bash
Creating namespace gloo-system... Done.
Starting Gloo installation...

Gloo was successfully installed!

```

### Deploying the application

Our `echo` application is a simple container (thanks to our friends at HashiCorp) that will 
respond with the application version, to help demonstrate our canary workflows as we start testing and 
shifting traffic to a `v2` version of the application. 

Kubernetes gives us a lot of flexibility in terms of modeling this application. We'll adopt the following 
conventions:
* We'll include the version in the deployment name so we can run two versions of the application 
side-by-side and manage their lifecycle differently. 
* We'll label pods with an app label (`app: echo`) and a version label (`version: v1`) to help with our canary rollout. 
* We'll deploy a single Kubernetes `Service` for the application to set up networking. Instead of updating 
this or using multiple services to manage routing to different versions, we'll manage the rollout with Gloo configuration. 

The following is our `v1` echo application: 

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: echo-v1
spec:
  replicas: 1
  selector:
    matchLabels:
      app: echo
      version: v1
  template:
    metadata:
      labels:
        app: echo
        version: v1
    spec:
      containers:
        # Shout out to our friends at Hashi for this useful test server
        - image: hashicorp/http-echo
          args:
            - "-text=version:v1"
            - -listen=:8080
          imagePullPolicy: Always
          name: echo-v1
          ports:
            - containerPort: 8080
```

And here is the `echo` Kubernetes `Service` object:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: echo
spec:
  ports:
    - port: 80
      targetPort: 8080
      protocol: TCP
  selector:
    app: echo
```

For convenience, we've published this yaml in a repo so we can deploy it with the following command:

```bash
kubectl apply -f https://raw.githubusercontent.com/solo-io/gloo-ref-arch/7c8e769bca02af636a783953b897fa79c6154b7c/platform/prog-delivery/two-phased-with-os-gloo/1-setup/echo.yaml
```

We should see the following output:
```bash
namespace/echo created
deployment.apps/echo-v1 created
service/echo created
```

And we should be able to see all the resources healthy in the `echo` namespace:
```bash
➜ k get all -n echo
NAME                           READY   STATUS    RESTARTS   AGE
pod/echo-v1-66dbfffb79-287s5   1/1     Running   0          6s

NAME           TYPE        CLUSTER-IP      EXTERNAL-IP   PORT(S)   AGE
service/echo   ClusterIP   10.55.252.216   <none>        80/TCP    6s

NAME                      READY   UP-TO-DATE   AVAILABLE   AGE
deployment.apps/echo-v1   1/1     1            1           7s

NAME                                 DESIRED   CURRENT   READY   AGE
replicaset.apps/echo-v1-66dbfffb79   1         1         1       7s
```

### Exposing outside the cluster with Gloo

We can now expose this service outside the cluster with Gloo. First, we'll model the application as a Gloo `Upstream`, which is Gloo's abstraction 
for a traffic destination. In this case the `Upstream` object just includes basic information about the Kubernetes service; later, we'll 
add more configuration to support a canary rollout:

```yaml
apiVersion: gloo.solo.io/v1
kind: Upstream
metadata:
  name: echo
  namespace: gloo-system
spec:
  kube:
    selector:
      app: echo
    serviceName: echo
    serviceNamespace: echo
    servicePort: 8080
```

We can now create a route to this upstream in Gloo by defining a **virtual service**:

```yaml
apiVersion: gateway.solo.io/v1
kind: VirtualService
metadata:
  name: echo
  namespace: gloo-system
spec:
  virtualHost:
    domains:
      - '*'
    routes:
      - matchers:
          - prefix: /
        routeAction:
          single:
            upstream:
              name: echo
              namespace: gloo-system
```

We can apply these resources with the following commands:

```bash
kubectl apply -f https://raw.githubusercontent.com/solo-io/gloo-ref-arch/7c8e769bca02af636a783953b897fa79c6154b7c/platform/prog-delivery/two-phased-with-os-gloo/1-setup/upstream.yaml
kubectl apply -f https://raw.githubusercontent.com/solo-io/gloo-ref-arch/7c8e769bca02af636a783953b897fa79c6154b7c/platform/prog-delivery/two-phased-with-os-gloo/1-setup/vs.yaml
```

Once we apply these two resources, we can start to send traffic to the application through Gloo:

```bash
➜ curl $(glooctl proxy url)/
version:v1
```

## Rollout Strategy

Now we have a new version `v2` of the echo application that we wish to roll out. We know that when the  
rollout is complete, we are going to end up with this picture:

![](4-decommissioning-v1/end-state.png)

However, to get there, we may want to perform a few rounds of testing to ensure the new version of the application
meets certain correctness and/or performance acceptance criteria. In this post, we'll introduce a two-phased approach to 
canary rollout with Gloo, that could be used to satisfy the vast majority of acceptance tests. 

In the first phase, we'll perform smoke and correctness tests by routing a small segment of the traffic to the new version 
of the application. In this demo, we'll use a header `stage: canary` to trigger routing to the new service, though in 
practice it may be desirable to make this decision based on another part of the request, such as the claim in a verified JWT. 

In the second phase, we've already established correctness, so we are ready to shift all of the traffic over to the new 
version of the application. We'll configure weighted destinations, and shift the traffic while monitoring certain business 
metrics to ensure the service quality remains at acceptable levels. Once 100% of the traffic is shifted to the new version, 
the old version can be decommissioned. 

In practice, it may be desirable to only use one of the phases for testing, in which case the other phase can be 
skipped. 

## Phase 1: Initial canary rollout of v2

[TODO: picture of v1 and v2 with subset routing]

### Setting up subset routing

First, we'll update the upstream to include a subset definition:

```yaml
apiVersion: gloo.solo.io/v1
kind: Upstream
metadata:
  name: echo
  namespace: gloo-system
spec:
  kube:
    selector:
      app: echo
    serviceName: echo
    serviceNamespace: default
    servicePort: 8080
    subsetSpec:
      selectors:
        - keys:
            - version
```

Now we can create routes that route to specific versions of the echo server. First, we'll update our virtual service 
to specify the `v1` subset to be used for our existing route:

```yaml
apiVersion: gateway.solo.io/v1
kind: VirtualService
metadata:
  name: echo
  namespace: gloo-system
spec:
  virtualHost:
    domains:
      - '*'
    routes:
      - matchers:
          - prefix: /
        routeAction:
          single:
            upstream:
              name: echo
              namespace: gloo-system
            subset:
              values:
                version: v1
```

### Deploying echo server v2

Now we can safely deploy `v2` of the echo server:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: echo-v2
spec:
  replicas: 1
  selector:
    matchLabels:
      app: echo
      version: v2
  template:
    metadata:
      labels:
        app: echo
        version: v2
    spec:
      containers:
        - image: hashicorp/http-echo
          args:
            - "-text=version:v2"
            - -listen=:8080
          imagePullPolicy: Always
          name: echo-v2
          ports:
            - containerPort: 8080
```

Since our gateway is configured to route specifically to the `v1` subset, this should have no effect. However, it does enable 
`v2` to be routable from the gateway if the `v2` subset is configured for a route. 

### Adding a route to v2

To start, we'll route to v2 when the `stage: canary` header is supplied on the request. 

```yaml
apiVersion: gateway.solo.io/v1
kind: VirtualService
metadata:
  name: echo
  namespace: gloo-system
spec:
  virtualHost:
    domains:
      - '*'
    routes:
      - matchers:
          - headers:
              - name: stage
                value: canary
            prefix: /
        routeAction:
          single:
            upstream:
              name: echo
              namespace: gloo-system
            subset:
              values:
                version: v2
      - matchers:
          - prefix: /
        routeAction:
          single:
            upstream:
              name: echo
              namespace: gloo-system
            subset:
              values:
                version: v1
```

### Canary testing

Now that we have this route, we can do some testing. First let's ensure that the existing route is working as expected:

```bash
➜ curl $(glooctl proxy url)/
version:v1
```

```bash
➜ curl $(glooctl proxy url)/ -H "stage: canary"
version:v2
```

### Advanced subsets using JWT claims for routing

Let's say this approach, with a request header, is too open for our initial testing. One option is to add JWT authorization
to the virtual service, and then use the "claims to headers" feature of Gloo to add the stage header based on a claim 
value in the verified JWT. 



## Phase 2: Shifting all traffic to v2 and decommissioning v1



In this option, we want to start shifting a small fraction of the traffic for the echo application to v2, and send the 
rest of the traffic to v1. Over time, we'll increase the weight for v2 until all of the traffic is being routed to that version, while 
monitoring to ensure certain metrics remain above certain thresholds. When this is done, we can decommission v1. 

In order to enable weighted destinations, we need to model each version as a separate upstream destination. First, we'll update 
the existing upstream to add a version to the selector. 

```yaml
apiVersion: gloo.solo.io/v1
kind: Upstream
metadata:
  name: echo
  namespace: gloo-system
spec:
  kube:
    selector:
      app: echo
      version: v1
    serviceName: echo
    serviceNamespace: default
    servicePort: 8080
```

Next, we'll add a new upstream to represent the canary destination:

```yaml
apiVersion: gloo.solo.io/v1
kind: Upstream
metadata:
  name: echo-canary
  namespace: gloo-system
spec:
  kube:
    selector:
      app: echo
      version: v2
    serviceName: echo
    serviceNamespace: default
    servicePort: 8080
```

Now we can change the Gloo route to route to both of these destinations, starting with sending all the traffic to the v1. 

```yaml
apiVersion: gateway.solo.io/v1
kind: VirtualService
metadata:
  name: echo
  namespace: gloo-system
spec:
  virtualHost:
    domains:
      - '*'
    routes:
      - matchers:
          - prefix: /
        routeAction:
          multi:
            destinations:
              - weight: 10
                destination:
                  upstream:
                    name: echo
                    namespace: gloo-system
              - weight: 0
                destination:
                  upstream:
                    name: echo-canary
                    namespace: gloo-system
```

```bash
➜ curl $(glooctl proxy url)/
version:v1
```

### Commence rollout

Now we can change the weights:

```yaml
apiVersion: gateway.solo.io/v1
kind: VirtualService
metadata:
  name: echo
  namespace: gloo-system
spec:
  virtualHost:
    domains:
      - '*'
    routes:
      - matchers:
          - prefix: /
        routeAction:
          multi:
            destinations:
              - weight: 5
                destination:
                  upstream:
                    name: echo
                    namespace: gloo-system
              - weight: 5
                destination:
                  upstream:
                    name: echo-canary
                    namespace: gloo-system
```

### Decommission old upstream
            
```yaml

```
