# Two-phased canary rollout, part 2

In the [last part](https://www.solo.io/blog/two-phased-canary-rollout-with-gloo/), we looked at how you can use Gloo to set up a two-phased approach to canary testing 
and rolling out new versions of your services. 

* In the first phase, you redirect a small slice of your traffic so you can verify the functionality of the new
version. 
* Once satisfied, you move on to the second phase, during which you use weighted destinations to 
gradually shift the load to the new version of the service, until you are complete and the old version 
can be decommissioned. 

Now we're going to look at how we can improve upon that workflow design, so we can scale across many services owned by many teams, 
while taking into account how responsibilities may be separated across different roles in the organization, 
and making sure the platform gracefully handles configuration errors.   

## Scaling across multiple teams

As we saw in the last post, Gloo uses the [VirtualService](https://docs.solo.io/gloo/latest/introduction/architecture/concepts/#virtual-services) to manage the routes for a particular domain. 
We were able to execute our two-phased upgrade workflow in Gloo by modifying the routes on the virtual service object. 

Before, we executed the workflow for a single service called **echo**. Now, we are going to introduce a second 
service **foxtrot**, owned by a different team, and consider how this workflow might scale to multiple teams. 

In particular, we're going to look for an approach that satisfies the following key goals:
  * (A) Avoids bottlenecks on a particular team or person
  * (B) Limits the risk of one team disrupting another team's service health
  * (C) Fits in nicely with an organization's approach to roles, role-based access control, and Kubernetes  

### Option 1: Shared virtual service

The simple approach to scaling would be to have a single VirtualService, and manage all the routes for the **echo** 
and **foxtrot** services with the same resource. 

Right away, we can see a problem. Since all our routes are controlled with a single object, we either
need to grant write permissions to that object to both teams, or we need all the rights to go through a central
admin team. The latter approach would most likely be the lesser of two evils, and you would place an increasingly large 
burden on your central admin team as the number of discrete development teams grows. 

### Option 2: Separating ownership across domains

The first alternative we might consider is to model each service with different domains, so that the routes 
are managed on different objects. For example, if our primary domain was `example.com`, we could have a virtual 
service for each subdomain: `echo.example.com` and `foxtrot.example.com`. 

```yaml
apiVersion: gateway.solo.io/v1
kind: VirtualService
metadata:
  name: echo
  namespace: echo
spec:
  virtualHost:
    domains:
      - 'echo.example.com'
    routes:
      - matchers:
          - prefix: /echo
...
---
apiVersion: gateway.solo.io/v1
kind: VirtualService
metadata:
  name: foxtrot
  namespace: foxtrot
spec:
  virtualHost:
    domains:
      - 'foxtrot.example.com'
...
```

Now, we can more cleanly decentralize the ownership of the routes, across the two services, by giving each 
team ownership over their own virtual services. Gloo can easily watch both namespaces, so you can store 
the virtual service with the other resources owned by each team. 

However, in some organizations (as is the case for the user who reached out to me), 
the ownership of the domains and routes is separated -- with a dev ops team responsible 
for things like DNS and certificate management, and a dev team responsible for specific routes. 
That means there would still be shared ownership of these two virtual services, 
just not shared across development teams.

For such an organization, a better approach would be to have an admin team own the root virtual service, 
and for that team to be able to **delegate** ownership of one or more routes.  

### Option 3: Separating ownership across route tables with **delegation**

To solve these problems, we'll take advantage of a feature in Gloo called **delegation**. With delegation, we can 
define our routes on a different object called a [RouteTable](https://docs.solo.io/gloo/latest/guides/traffic_management/destination_types/delegation/). In the virtual service, we can define a 
**delegate action**, and reference the **route table**. This enables us to
separate our ownership of the domain from the ownership of the specific routes, and also separate the ownership 
of different routes across different development teams. 

This sounds like the most satisfactory approach, so now let's see how this works in Gloo. 

## Setting up Gloo

I'm also assuming you have a decent understanding of the two-phased approach 
detailed in [part 1](https://www.solo.io/blog/two-phased-canary-rollout-with-gloo/).  
I'm also assuming you have a Kubernetes cluster and you've installed Gloo as I explained
in part 1, and are ready to start deploying the services. 

### Deploy the applications

First, we will deploy echo to the echo namespace:

```
kubectl apply -f echo.yaml
```

And we will deploy foxtrot to the foxtrot namespace:

```
kubectl apply -f foxtrot.yaml
```

We should wait for echo and foxtrot to be ready:

Wait until the pods in namespace 'echo' are ready. Use `kubectl get pods -n echo` to check the status.

Wait until the pods in namespace 'foxtrot' are ready. Use `kubectl get pods -n foxtrot` to check the status.

### Model the services as upstreams in Gloo

Let's model echo as an [Upstream](https://docs.solo.io/gloo/latest/introduction/architecture/concepts/#upstreams) destination in Gloo:

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
    subsetSpec:
      selectors:
        - keys:
            - version

```

And deploy it to the cluster:

```
kubectl apply -f upstream-echo.yaml
```

And let's do the same for foxtrot, modeling it as an [Upstream](https://docs.solo.io/gloo/latest/introduction/architecture/concepts/#upstreams):

```yaml
apiVersion: gloo.solo.io/v1
kind: Upstream
metadata:
  name: foxtrot
  namespace: gloo-system
spec:
  kube:
    selector:
      app: foxtrot
    serviceName: foxtrot
    serviceNamespace: foxtrot
    servicePort: 8080
    subsetSpec:
      selectors:
        - keys:
            - version

```

And deploying it to the cluster:

```
kubectl apply -f upstream-foxtrot.yaml
```

### Set up route tables

Now, let's create a route table containing the route to the echo upstream: 

```yaml
apiVersion: gateway.solo.io/v1
kind: RouteTable
metadata:
  name: echo-routes
  namespace: echo
spec:
  routes:
    - matchers:
        - prefix: /echo
      routeAction:
        single:
          upstream:
            name: echo
            namespace: gloo-system
          subset:
            values:
              version: v2
```

And deploy it to the cluster:

```
kubectl apply -f rt-echo-1.yaml
```

And let's create the route table for foxtrot:

```yaml
apiVersion: gateway.solo.io/v1
kind: RouteTable
metadata:
  name: foxtrot-routes
  namespace: foxtrot
spec:
  routes:
    - matchers:
        - prefix: /foxtrot
      routeAction:
        single:
          upstream:
            name: foxtrot
            namespace: gloo-system
          subset:
            values:
              version: v1
```

And deploy it to the cluster:

```
kubectl apply -f rt-foxtrot-1.yaml
```

It is important to note we are storing these route table resources in the echo and foxtrot namespaces 
respectively. This is to simulate having the echo and foxtrot teams own these resources directly. We can 
assume when we make changes to those resources, we are impersonating those teams. 

### Wire it up with a virtual service

Finally, we can wire up both services to a particular domain (in this case, `*`) by creating a virtual 
service. In our scenario, we are assuming a central ops team owns the domain, so we'll keep this config 
in the `gloo-system` namespace. 

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
          - prefix: /echo
        delegateAction:
          selector:
            namespaces:
              - echo
      - matchers:
          - prefix: /foxtrot
        delegateAction:
          selector:
            namespaces:
              - foxtrot
```

And deploy it to the cluster:

```
kubectl apply -f vs.yaml
```

### Test the routes

At this point, we should be able to send requests to both services through Gloo: 

```bash
➜ curl $(glooctl proxy url)/echo
version:echo-v1
```

```bash
➜ curl $(glooctl proxy url)/foxtrot
version:foxtrot-v1
```

## Running the two-phased canary workflow

Now, the echo or foxtrot team can run the two-phased workflow on their service by making edits to their 
corresponding route table. We're going to impersonate the foxtrot team and start rolling out v2. 

### Foxtrot team starts phase 1

For Phase 1, we need to deploy v2 of the foxtrot service, and then update the route table 
to send requests including the header `stage: canary` to v2. 

We can deploy foxtrot-v2 and wait for it to be deployed:

```
kubectl apply -f foxtrot-v2.yaml
```

Wait until the pods in namespace 'foxtrot' are ready. Use `kubectl get pods -n foxtrot` to check the status.

Then we need to update the routes so we can start testing foxtrot v2:

```yaml
apiVersion: gateway.solo.io/v1
kind: RouteTable
metadata:
  name: foxtrot-routes
  namespace: foxtrot
spec:
  routes:
    - matchers:
        - headers:
            - name: stage
              value: canary
          prefix: /foxtrot
      routeAction:
        single:
          upstream:
            name: foxtrot
            namespace: gloo-system
          subset:
            values:
              version: v2
    - matchers:
        - prefix: /foxtrot
      routeAction:
        single:
          upstream:
            name: foxtrot
            namespace: gloo-system
          subset:
            values:
              version: v1
```

And deploy it to the cluster:

```
kubectl apply -f rt-foxtrot-2.yaml
```

### Test the routes 

Now we can test to make sure we can send requests to foxtrot v2. Our other 
routes should continue to behave as before:

```bash
➜ curl $(glooctl proxy url)/echo
version:echo-v1
```

```bash
➜ curl $(glooctl proxy url)/foxtrot
version:foxtrot-v1
```

```bash
➜ curl $(glooctl proxy url)/foxtrot -H "stage: canary"
version:foxtrot-v2
```

The last request is now sent to v2 because the canary header was provided. 

### Foxtrot team starts phase 2

Now we can switch to phase 2 by updating the foxtrot route table with weighted destinations:

```yaml
apiVersion: gateway.solo.io/v1
kind: RouteTable
metadata:
  name: foxtrot-routes
  namespace: foxtrot
spec:
  routes:
    - matchers:
        - headers:
            - name: stage
              value: canary
          prefix: /foxtrot
      routeAction:
        single:
          upstream:
            name: foxtrot
            namespace: gloo-system
          subset:
            values:
              version: v2
    - matchers:
        - prefix: /foxtrot
      routeAction:
        multi:
          destinations:
            - destination:
                upstream:
                  name: foxtrot
                  namespace: gloo-system
                subset:
                  values:
                    version: v1
              weight: 100
            - destination:
                upstream:
                  name: foxtrot
                  namespace: gloo-system
                subset:
                  values:
                    version: v2
              weight: 0
```

And deploy it to the cluster:

```
kubectl apply -f rt-foxtrot-3.yaml
```

We expect the routes to continue to behave as before. 

### Echo team starts v2 rollout 

While the foxtrot team is rolling out v2 of foxtrot, let's say the echo team is ready to start testing
a new version of the echo service. 

We can deploy echo-v2 and wait for it to be deployed:

```
kubectl apply -f echo-v2.yaml
```

Wait until the pods in namespace 'echo' are ready. Use `kubectl get pods -n echo` to check the status.

Then we need to update the routes so we can start testing echo v2:

```yaml
apiVersion: gateway.solo.io/v1
kind: RouteTable
metadata:
  name: echo-routes
  namespace: echo
spec:
  routes:
    - matchers:
        - headers:
            - name: stage
              value: canary
          prefix: /echo
      routeAction:
        single:
          upstream:
            name: echo
            namespace: gloo-system
          subset:
            values:
              version: v2
    - matchers:
        - prefix: /echo
      routeAction:
        single:
          upstream:
            name: echo
            namespace: gloo-system
          subset:
            values:
              version: v1
```

Let's deploy it to the cluster:

```
kubectl apply -f rt-echo-2.yaml
```

### Test the routes

Now we should be able to test v2 of either the echo or foxtrot service, since each team is in the middle 
of a canary rollout. 

```bash
➜ curl $(glooctl proxy url)/echo
version:echo-v1
```

```bash
➜ curl $(glooctl proxy url)/echo -H "stage: canary"
version:echo-v2
```

```bash
➜ curl $(glooctl proxy url)/foxtrot
version:foxtrot-v1
```

```bash
➜ curl $(glooctl proxy url)/foxtrot -H "stage: canary"
version:foxtrot-v2
```

## Handling invalid configuration

As we can now see, using route tables allows us to scale our canary rollout to multiple development teams, 
without needing to give teams ownership over their own domain configuration. Teams can manage their own 
canary rollouts in parallel, without needing to coordinate. **At least, that's true until someone starts authoring
invalid configuration**. We'll see that Gloo's default behavior in response to invalid configuration isn't desirable 
for our use case; however, we can resolve that and change the behavior with a simple settings change. 

### Default Gloo behavior when there is invalid configuration

Typically, when Gloo encounters invalid configuration, it tries to continue serving the last known good configuration
to Envoy. That way, the mistake won't cause working routes to get removed, and as soon as the 
configuration is repaired, Envoy will start receiving updates again. This isn't foolproof - it only works as long as 
Gloo and Envoy have that last known configuration in memory, and won't survive pod restarts. But it preserves decent 
degradation semantics in the event of a problem, and is often the ideal behavior for a particular use case. 

Let's see what that looks like in our use case. Let's simulate the echo team writing configuration that 
is invalid, by referencing an upstream destination that doesn't exist. 

```yaml
apiVersion: gateway.solo.io/v1
kind: RouteTable
metadata:
  name: echo-routes
  namespace: echo
spec:
  routes:
    - matchers:
        - headers:
            - name: stage
              value: canary
          prefix: /echo
      routeAction:
        single:
          upstream:
            name: echo-typo
            namespace: gloo-system
          subset:
            values:
              version: v2
    - matchers:
        - prefix: /echo
      routeAction:
        single:
          upstream:
            name: echo
            namespace: gloo-system
          subset:
            values:
              version: v1
```

Let's deploy it to the cluster:

```
kubectl apply -f rt-echo-3.yaml
```

In parallel, the foxtrot team is trying to finish phase 2 and adjusts the weights to start sending 100% of traffic to the v2 destination:

```yaml
apiVersion: gateway.solo.io/v1
kind: RouteTable
metadata:
  name: foxtrot-routes
  namespace: foxtrot
spec:
  routes:
    - matchers:
        - headers:
            - name: stage
              value: canary
          prefix: /foxtrot
      routeAction:
        single:
          upstream:
            name: foxtrot
            namespace: gloo-system
          subset:
            values:
              version: v2
    - matchers:
        - prefix: /foxtrot
      routeAction:
        multi:
          destinations:
            - destination:
                upstream:
                  name: foxtrot
                  namespace: gloo-system
                subset:
                  values:
                    version: v1
              weight: 0
            - destination:
                upstream:
                  name: foxtrot
                  namespace: gloo-system
                subset:
                  values:
                    version: v2
              weight: 100
```

Let's deploy that to the cluster:

```
kubectl apply -f rt-foxtrot-4.yaml
```

### Test the routes

Now, we can tell there's an error by running `glooctl check`:

```
➜ glooctl check
Checking deployments... OK
Checking pods... OK
Checking upstreams... OK
Checking upstream groups... OK
Checking auth configs... OK
Checking secrets... OK
Checking virtual services... Found virtual service with warnings: gloo-system app
Reason: warning:
  Route Warning: InvalidDestinationWarning. Reason: *v1.Upstream {echo-typo gloo-system} not found
Route Warning: InvalidDestinationWarning. Reason: *v1.Upstream {echo-typo gloo-system} not found
Problems detected!
```

If we test the routes, we'll see the following behavior:

```bash
➜ curl $(glooctl proxy url)/echo
version:echo-v1
```

```bash
➜ curl $(glooctl proxy url)/echo -H "stage: canary"
version:echo-v2
```

```bash
➜ curl $(glooctl proxy url)/foxtrot
version:foxtrot-v1
```

```bash
➜ curl $(glooctl proxy url)/foxtrot -H "stage: canary"
version:foxtrot-v2
```

As we can see, there is an error in the overall proxy config, so Gloo is continuing to serve Envoy the last 
known good configuration. This means the foxtrot team's change to set the weight to v2 did not apply. 
The foxtrot team is blocked, pending the echo team fixing their configuration. 

When considering an API gateway that is used across a large development organization, with many independent teams, 
we would consider it a red flag if one team's mistake could block another team's progress. Fortunately, Gloo 
has a feature called [route replacement](https://docs.solo.io/gloo/latest/guides/traffic_management/configuration_validation/invalid_route_replacement/) that changes the proxy behavior and addresses this concern. 

### Changing the behavior around invalid configuration

Let's create a file called `settings-patch.yaml` that contains the following patch for our Gloo settings:

```yaml
spec:
  gloo:
    invalidConfigPolicy:
      invalidRouteResponseBody: Gloo Gateway has invalid configuration. Administrators
        should run `glooctl check` to find and fix config errors.
      invalidRouteResponseCode: 404
      replaceInvalidRoutes: true
```

This will change the behavior of Gloo, so that in the event of bad configuration, it replaces the routes that 
are affected by the mistake with a 404 and error message as a direct response. In our case, the new echo route table
is invalid, so the route is replaced; however, Gloo will update Envoy with the routes from the 
foxtrot route table, since that object is valid. It will also continue to serve the old `echo` route.  

We can apply the patch with the following command:
```
kubectl patch -n gloo-system settings default --type merge --patch "$(cat settings-patch.yaml)"
```

### Test the routes

Now if we run the same test as above, we'll see the following results:

```bash
➜ curl $(glooctl proxy url)/echo
version:echo-v1
```

```bash
➜ curl $(glooctl proxy url)/echo -H "stage: canary"
Gloo Gateway has invalid configuration. Administrators should run `glooctl check` to find and fix config errors.
```

```bash
➜ curl $(glooctl proxy url)/foxtrot
version:foxtrot-v2
```

```bash
➜ curl $(glooctl proxy url)/foxtrot -H "stage: canary"
version:foxtrot-v2
```

## Finish the foxtrot rollout

Now that we've setup Gloo to replace invalid routes and preserve valid ones, the foxtrot team can finish their 
rollout. 

Since all the traffic has been shifted to v2, we can now clean up the routes:

```yaml
apiVersion: gateway.solo.io/v1
kind: RouteTable
metadata:
  name: foxtrot-routes
  namespace: foxtrot
spec:
  routes:
    - matchers:
        - prefix: /foxtrot
      routeAction:
        single:
          upstream:
            name: foxtrot
            namespace: gloo-system
          subset:
            values:
              version: v2
```

Let's deploy it with the following command:

```
kubectl apply -f rt-foxtrot-5.yaml
```

And we can delete the `v1` foxtrot deployment, which is no longer serving any traffic. 
            
```
kubectl delete -f foxtrot-v1.yaml
```

Finally, let's check to make sure foxtrot is still healthy:

```bash
➜ curl $(glooctl proxy url)/foxtrot
version:foxtrot-v2
```

## Fix the echo configuration

We can also revert the invalid echo config, to unblock the echo team. We'll just revert to the old
route table:

```yaml
apiVersion: gateway.solo.io/v1
kind: RouteTable
metadata:
  name: echo-routes
  namespace: echo
spec:
  routes:
    - matchers:
        - headers:
            - name: stage
              value: canary
          prefix: /echo
      routeAction:
        single:
          upstream:
            name: echo
            namespace: gloo-system
          subset:
            values:
              version: v2
    - matchers:
        - prefix: /echo
      routeAction:
        single:
          upstream:
            name: echo
            namespace: gloo-system
          subset:
            values:
              version: v1
```

Let's deploy it to the cluster:

```
kubectl apply -f rt-echo-2.yaml
```

Now, we'll see the error is cleared up by running `glooctl check`:
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
```

And we'll see the routes working as expected again for echo:

```bash
➜ curl $(glooctl proxy url)/echo
version:echo-v1
```

```bash
➜ curl $(glooctl proxy url)/echo -H "stage: canary"
version:echo-v2
```

## Summary

In this post, we looked at extending our two-phased canary rollout workflow. We scaled it across multiple dev teams
by delegating different route tables to different teams. With delegation, we achieved a cleaner separation of responsibilities
for our dev ops team, who owns the domain, and for our dev teams, who own different routes. Finally, it was easy to customize
the behavior in Gloo to ensure the two teams could operate in parallel without needing to coordinate, and without risk that 
one team will block another by writing invalid configuration. 

Special thanks to Ievgenii Shepeliuk for providing feedback on [part 1](https://www.solo.io/blog/two-phased-canary-rollout-with-gloo/) and sharing how route tables are used 
at his organization. For a deeper look at how Gloo is used at New Age Solutions, check out our 
[case study](https://www.solo.io/blog/end-user-case-study-hrzn-igaming-platform-by-new-age-solutions/).

## Get Involved in the Gloo Community

Gloo has a large and growing community of open source users, in addition to an enterprise customer base. To learn more about 
Gloo:
* Check out the [repo](https://github.com/solo-io/gloo), where you can see the code and file issues
* Check out the [docs](https://docs.solo.io/gloo/latest), which have an extensive collection of guides and examples
* Join the [slack channel](http://slack.solo.io/) and start chatting with the Solo engineering team and user community

If you'd like to get in touch with me (feedback is always appreciated!), you can find me on Slack or email me at 
**rick.ducott@solo.io**. 
