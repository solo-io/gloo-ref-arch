# Two-Phased Canary Rollout with Gloo, part 3

Welcome to part 3 of our series on how to design workflows for canary upgrades with Gloo. 
In [part 1](LINK), we introduced a two-phased workflow that keeps the quality of your upgrades 
high by facilitating correctness and performance testing as you roll out new versions of a service. 

In [part 2](LINK), we looked at how to scale this workflow across many development teams. We leveraged delegation in Gloo
to cleanly separate ownership of the domain from the routes, to model an ops/dev organizational split. We used selectors 
to bind routes to a domain, to facilitate fully self-service onboarding of new services (and dev teams) to the application. 
And finally, we looked at how to configure Gloo settings to minimize the risk of one team's mistake causing another team to be blocked. 

In this part, we're going to quickly onboard a few services to our application using the model described in Part 2, and 
then we're going to look at how a team can take advantage of **traffic shadowing** prior to starting a two-phased upgrade. 

## Initial setup

I'll assume you already have a Kubernetes cluster and you've deployed Gloo (open source or enterprise). Like in the previous
parts, we'll be creating our own `Upstream` objects, so we don't need to have Gloo's discovery component running. 

I'm assuming we don't have any virtual services defined on our cluster. We can verify that by running `kubectl get vs -A`
or `kubectl get proxy -A` to look for Gloo virtual service or proxy objects in any namespace. These should return no results. 

Let's look at the virtual service for our application:

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

This is the generic virtual service from the end of part 2, which will automatically bind any route tables that have
the `apiGroup` label. Let's apply it to the cluster:

```
kubectl apply -f vs.yaml
```

As we saw in part 2, this is how we'll manage the domain itself. Once this is deployed, teams can now be onboarded in a 
self-service fashion if they supply a route table with the expected label. For now, creating this will result in an 
empty proxy, because we haven't created any route tables yet. Let's verify that by checking the proxy CRD, Gloo's intermediate
representation:

```
➜ k get proxy -A -oyaml
apiVersion: v1
items:
- apiVersion: gloo.solo.io/v1
  kind: Proxy
  metadata:
    creationTimestamp: "2020-04-15T16:08:57Z"
    generation: 2
    labels:
      created_by: gateway
    name: gateway-proxy
    namespace: gloo-system
    resourceVersion: "15113724"
    selfLink: /apis/gloo.solo.io/v1/namespaces/gloo-system/proxies/gateway-proxy
    uid: 2b7356e0-e99c-4405-8169-ba2141f29d51
  spec:
    listeners:
    - bindAddress: '::'
      bindPort: 8080
      httpListener:
        virtualHosts:
        - domains:
          - '*'
          metadata:
            sources:
            - kind: '*v1.VirtualService'
              name: app
              namespace: gloo-system
              observedGeneration: 1
          name: gloo-system.app
      metadata:
        sources:
        - kind: '*v1.Gateway'
          name: gateway-proxy
          namespace: gloo-system
          observedGeneration: 5
      name: listener-::-8080
      useProxyProto: false
    - bindAddress: '::'
      bindPort: 8443
      httpListener: {}
      metadata:
        sources:
        - kind: '*v1.Gateway'
          name: gateway-proxy-ssl
          namespace: gloo-system
          observedGeneration: 5
      name: listener-::-8443
      useProxyProto: false
  status:
    reported_by: gloo
    state: 1
kind: List
metadata:
  resourceVersion: ""
  selfLink: ""
```

We can see that a proxy CRD was created, and there is a virtual service bound to the http listener with no routes, as we 
expect. 

## Onboarding our services

Now, let's onboard the echo service to our application. We can now define all the resources we need in a single Kubernetes manifest, 
containing a namespace, deployment, service, upstream, and route table. All of these resources are written to the echo namespace, 
which Gloo watches and immediately binds the routes to the proxy. 

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: echo
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: echo-v1
  namespace: echo
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
        - image: hashicorp/http-echo
          args:
            - "-text=version:echo-v1"
            - -listen=:8080
          imagePullPolicy: Always
          name: echo-v1
          ports:
            - containerPort: 8080
---
apiVersion: v1
kind: Service
metadata:
  name: echo
  namespace: echo
spec:
  ports:
    - port: 80
      targetPort: 8080
      protocol: TCP
  selector:
    app: echo
---
apiVersion: gloo.solo.io/v1
kind: Upstream
metadata:
  name: echo
  namespace: echo
spec:
  kube:
    selector:
      app: echo
    serviceName: echo
    serviceNamespace: echo
    servicePort: 8080
    subsetSpec:
      selectors:
        - keys:
            - version
---
apiVersion: gateway.solo.io/v1
kind: RouteTable
metadata:
  name: echo-routes
  namespace: echo
  labels:
    apiGroup: example
spec:
  routes:
    - matchers:
        - prefix: /echo
      routeAction:
        single:
          upstream:
            name: echo
            namespace: echo
          subset:
            values:
              version: v1
```

Let's apply that to the cluster:

```
kubectl apply -f echo.yaml
```

Wait until the pods in namespace 'echo' are ready. Use `kubectl get pods -n echo` to check the status.

Now, we can verify the route was picked up by Gloo and bound to the http listener by checking the proxy CRD, as we 
did before. And we can also verify Gloo is working as we expect with a curl request:

``` 
➜ curl $(glooctl proxy url)/echo
version:echo-v1
```

We successfully onboarded echo to our application without changing any resource outside the echo namespace. Cool! 

We can do the same thing for foxtrot:

```
kubectl apply -f foxtrot.yaml
```

Wait until the pods in namespace 'foxtrot' are ready. Use `kubectl get pods -n foxtrot` to check the status.

And we see our foxtrot route working now too:

```
➜ curl $(glooctl proxy url)/foxtrot
version:foxtrot-v1
```

As we mentioned in part two, we can make it even easier for teams to onboard to our platform 
by providing a helm chart that templates out these resources. We'll save that for a future part. 

## Setting up shadowing for v2

Let's say the echo team is preparing a new version of the echo service, but some problems arose during the initial 
upgrade and they had to roll back to v1. They have been struggling to reproduce the problem in their dev environment, 
since it only seems to happen in production and with real user traffic through the proxy. So they'd like to deploy 
v2 to production, and simulate sending some of the real production traffic to it, 
without exposing it on the virtual service with a real route. For this, they can turn on **traffic shadowing**. Let's 
see how this works. 

First, we'll deploy the v2 echo deployment:

```
kubectl apply -f echo-v2.yaml
```

Wait until the pods in namespace 'echo' are ready. Use `kubectl get pods -n echo` to check the status.

Next, we'll deploy an upstream to represent our shadow destination. With our normal routes, we've been using subset 
routing and specifying a version in the subset spec for our routes. This is how we control the specific version that 
is receiving traffic. However, when we create a shadowing configuration for the route, we are not able to specify a 
subset, so we'll instead define a new upstream specifically for shadowing to this version:

```yaml
apiVersion: gloo.solo.io/v1
kind: Upstream
metadata:
  name: shadow
  namespace: echo
spec:
  kube:
    selector:
      app: echo
      version: v2
    serviceName: echo
    serviceNamespace: echo
    servicePort: 8080
```

Let's apply that to the cluster:

```
kubectl apply -f upstream-shadow.yaml
```

Now we can turn on shadowing on our route:

```yaml
apiVersion: gateway.solo.io/v1
kind: RouteTable
metadata:
  name: echo-routes
  namespace: echo
  labels:
    apiGroup: example
spec:
  routes:
    - matchers:
        - prefix: /echo
      routeAction:
        single:
          upstream:
            name: echo
            namespace: echo
          subset:
            values:
              version: v1
      options:
        shadowing:
          percentage: 100
          upstream:
            name: shadow
            namespace: echo

```

In the shadowing configuration, we specify a percentage of traffic (in our case, we'll just send all of it), and the 
upstream destination for the shadowed requests. Let's deploy it to the cluster:

```
kubectl apply -f rt-shadow.yaml
```

## Testing shadowing 

Now that we've enabled traffic shadowing to echo v2, let's see how it works. Let's send a request to `echo`:

``` 
➜ curl $(glooctl proxy url)/echo
version:echo-v1
``` 

This behaves as we expect based on our route definition. We also expect a shadow request to hit v2. To verify that, 
we can look at the logs:

``` 
➜ kubectl logs -n echo deploy/echo-v2 -f
2020/04/15 18:42:52 Server is listening on :8080
2020/04/15 18:43:25 35.227.127.150-shadow 10.52.1.149:59760 "GET /echo HTTP/1.1" 200 16 "curl/7.54.0" 39.609µs
...
```

Each time we send a request to the `echo` route, we'll get a new shadow request logged in v2. This means our 
shadowing configuration is working! The echo team can utilize this for testing v2 with realistic traffic. When satisfied, 
we can remove the shadow option from our route, and clean up the shadow upstream. 

## Get Involved in the Gloo Community

Gloo has a large and growing community of open source users, in addition to an enterprise customer base. To learn more about 
Gloo:
* Check out the [repo](https://github.com/solo-io/gloo), where you can see the code and file issues
* Check out the [docs](https://docs.solo.io/gloo/latest), which have an extensive collection of guides and examples
* Join the [slack channel](http://slack.solo.io/) and start chatting with the Solo engineering team and user community

If you'd like to get in touch with me (feedback is always appreciated!), you can find me on Slack or email me at 
**rick.ducott@solo.io**. 

