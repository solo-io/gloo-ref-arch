# Two-phased canary rollout with Open Source Gloo

As engineering organizations move towards microservices, containers, and Kubernetes, they are leveraging API gateways and
service mesh technology to help with those migrations, and to unlock new capabilities for managing, observing, and securing 
traffic to their application. 

A common operational question to consider when building out your application platform is: how to migrate user traffic to a new
version of a service? This is sometimes referred to as a canary rollout or progressive delivery. In this post, we'll look at 
how Gloo can be used as an API gateway to facilitate canary rollouts of new user-facing services in Kubernetes. 

## Application Versioning in Kubernetes

[TODO: add picture of Kube cluster with echo namespace, gloo-system namespace]

For this scenario, we're going to deploy an application to Kubernetes. The application will be a simple echo server that 
responds to any incoming request with a text response, indicating the server version. In our staged canary rollout, 
we'll be shifting traffic from v1 to v2 of the echo service. This rollout could occur over an extended period of time, 
and the outcome is not guaranteed -- it may be aborted if certain metrics fall below certain thresholds. 

In order to run both versions of the server at the same time, we'll include the version in the Kubernetes Deployment object:

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

Here, we've named the deployment `echo-v1` so that it can be run side-by-side with other versions of the echo server without
a name conflict. We've included a label `app: echo` to know that this pod is part of the `echo` application, and we added a 
label `version: v1` so we can later identify that this is a `v1` pod specifically. Later, we'll introduce a `v2` deployment,
 with an updated name and labels. 

### Exposing inside the cluster with a Kubernetes Service

We want to expose this server via DNS to other services inside the Kubernetes cluster, so we'll create a Service definition:

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

This creates a DNS entry inside the cluster we can use to connect the API gateway to the echo containers. Note that we didn't include any reference 
to the version in the service name or selector. Since the traffic to this service is actually coming from the API 
gateway, we'll keep this service definition stable and use Gloo to determine which version of the application to use as the traffic destination. 

### Exposing outside the cluster with Gloo

We can expose this service with a route in Gloo. First, we'll model the service as a Gloo `Upstream`, which is Gloo's abstraction 
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
    serviceNamespace: default
    servicePort: 8080
```

We can now create a route to this destination in **Gloo** by defining a virtual service:

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

Once we apply these two resources, we can start to send traffic to Gloo:

```bash
➜ curl $(glooctl proxy url)/
version:v1
```

## Canary rollout strategy

For our canary rollout of the `v2` echo service, we're going to follow a two-phased approach. 

In the first phase, we want to initially roll out the new version of the service, and make it available to a small 
fraction of users. To start, we'll route to `v2` when a header is supplied in the incoming request. We can take this 
one step further by integrating with extauth and routing to `v2` when a claim is provided in a JWT. 

When the initial testing is complete, we'll move on to phase 2, where we'll progressively shift all traffic over to v2, 
before eventually decommissioning v1.  

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
